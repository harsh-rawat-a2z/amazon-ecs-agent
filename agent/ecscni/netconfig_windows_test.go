// +build windows,unit

// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package ecscni

import (
	"encoding/json"
	"net"
	"testing"

	apieni "github.com/aws/amazon-ecs-agent/agent/api/eni"
	"github.com/stretchr/testify/assert"
)

const (
	vpcPrimaryIPv4CIDR      = "10.0.0.0/16"
	validVPCGatewayCIDR     = "10.0.0.1/24"
	validVPCGatewayIPv4Addr = "10.0.0.1"
	validDNSServer          = "10.0.0.2"
	ipv4                    = "10.0.0.120"
	ipv4CIDR                = "10.0.0.120/24"
	ipv4Secondary           = "10.0.0.150"
	ipv4SecondaryCIDR       = "10.0.0.150/24"
	mac                     = "02:7b:64:49:b1:40"
	cniMinSupportedVersion  = "1.0.0"
)

func getTaskENI() *apieni.ENI {
	return &apieni.ENI{
		ID:                           "TestENI",
		MacAddress:                   mac,
		InterfaceAssociationProtocol: apieni.DefaultInterfaceAssociationProtocol,
		SubnetGatewayIPV4Address:     validVPCGatewayCIDR,
		IPV4Addresses: []*apieni.ENIIPV4Address{
			{
				Primary: true,
				Address: ipv4,
			},
			{
				Primary: false,
				Address: ipv4Secondary,
			},
		},
	}
}

func getCNIConfig() *Config {
	vpcCIDR := &net.IPNet{
		IP:   net.ParseIP("10.0.0.0"),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	}

	return &Config{
		MinSupportedCNIVersion: cniMinSupportedVersion,
		PrimaryIPV4VPCCIDR:     vpcCIDR,
		AllIPV4VPCCIDRBlocks:   []*net.IPNet{vpcCIDR},
	}
}

// TestNewBridgeNetworkConfigForTaskNSSetup tests the generated configuration when all parameters are valid
func TestNewBridgeNetworkConfigForTaskNSSetup(t *testing.T) {
	taskENI := getTaskENI()
	cniConfig := getCNIConfig()
	config, err := NewBridgeNetworkConfigForTaskNSSetup(taskENI, cniConfig)

	bridgeConfig := &BridgeForTaskENIConfig{}
	json.Unmarshal(config.Bytes, bridgeConfig)

	assert.NoError(t, err)
	assert.EqualValues(t, ECSVPCSharedENIPluginExecutable, config.Network.Type)
	assert.EqualValues(t, TaskENIBridgeNetworkPrefix, config.Network.Name)
	assert.EqualValues(t, cniMinSupportedVersion, config.Network.CNIVersion)
	assert.EqualValues(t, []string{validDNSServer}, bridgeConfig.DNS.Nameservers)
	assert.EqualValues(t, ipv4CIDR, bridgeConfig.ENIIPAddress)
	assert.EqualValues(t, ipv4SecondaryCIDR, bridgeConfig.IPAddress)
	assert.EqualValues(t, mac, bridgeConfig.ENIMACAddress)
	assert.EqualValues(t, validVPCGatewayIPv4Addr, bridgeConfig.GatewayIPAddress)
	assert.False(t, bridgeConfig.TaskENIConfig.NoInfra)
	assert.True(t, bridgeConfig.TaskENIConfig.EnableTaskENI)
	assert.False(t, bridgeConfig.TaskENIConfig.EnableTaskBridge)
}

// TestInvalidNewBridgeNetworkConfigForTaskNSSetup tests the generated configuration when secondary ip of eni is absent
func TestInvalidNewBridgeNetworkConfigForTaskNSSetup(t *testing.T) {
	taskENI := getTaskENI()
	taskENI.IPV4Addresses = []*apieni.ENIIPV4Address{
		{
			Primary: true,
			Address: ipv4,
		},
	}
	cniConfig := getCNIConfig()
	config, err := NewBridgeNetworkConfigForTaskNSSetup(taskENI, cniConfig)

	assert.Error(t, err)
	assert.Nil(t, config)
}

// TestNewBridgeNetworkConfigForTaskBridgeSetup tests the generated configuration when all parameters are valid
func TestNewBridgeNetworkConfigForTaskBridgeSetup(t *testing.T) {
	cniConfig := getCNIConfig()
	config, err := NewBridgeNetworkConfigForTaskBridgeSetup(cniConfig)

	bridgeConfig := &BridgeForTaskENIConfig{}
	json.Unmarshal(config.Bytes, bridgeConfig)

	assert.NoError(t, err)
	assert.EqualValues(t, ECSVPCSharedENIPluginExecutable, config.Network.Type)
	assert.EqualValues(t, config.Network.Name, DefaultECSBridgeNetworkName)
	assert.False(t, bridgeConfig.TaskENIConfig.NoInfra)
	assert.False(t, bridgeConfig.TaskENIConfig.EnableTaskENI)
	assert.True(t, bridgeConfig.TaskENIConfig.EnableTaskBridge)
}

// TestConstructDNSFromVPCGatewaySuccess tests if the dns is constructed properly from the given primary ipv4 VPC CIDR
func TestConstructDNSFromVPCCIDRSuccess(t *testing.T) {
	_, vpcPrimaryCIDR, _ := net.ParseCIDR(vpcPrimaryIPv4CIDR)
	result, err := constructDNSFromVPCCIDR(vpcPrimaryCIDR.IP)

	assert.NoError(t, err)
	assert.EqualValues(t, []string{validDNSServer}, result)
}
