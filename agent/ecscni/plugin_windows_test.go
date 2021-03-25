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
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	apieni "github.com/aws/amazon-ecs-agent/agent/api/eni"
	mock_libcni "github.com/aws/amazon-ecs-agent/agent/ecscni/mocks_libcni"
	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	eniID                       = "eni-12345678"
	eniIPV4Address              = "172.31.21.40"
	eniIPV4SecondaryAddress     = "172.31.21.41"
	eniMACAddress               = "02:7b:64:49:b1:40"
	eniSubnetGatewayIPV4Address = "172.31.1.1/20"
	vpcIPv4CIDR                 = "10.0.0.0/16"
)

func getENI() *apieni.ENI {
	return &apieni.ENI{
		ID:                           eniID,
		MacAddress:                   eniMACAddress,
		InterfaceAssociationProtocol: apieni.DefaultInterfaceAssociationProtocol,
		SubnetGatewayIPV4Address:     eniSubnetGatewayIPV4Address,
		IPV4Addresses: []*apieni.ENIIPV4Address{
			{
				Primary: true,
				Address: eniIPV4Address,
			},
			{
				Primary: false,
				Address: eniIPV4SecondaryAddress,
			},
		},
	}
}

// getBaseConfig returns the base configuration which is required to build CNI configurations
func getBaseConfig() *Config {
	_, ipAddr, _ := net.ParseCIDR(vpcIPv4CIDR)
	return &Config{
		ContainerID:          "containerid12",
		ContainerPID:         "pid",
		ContainerNetNS:       "container:1234def",
		NetworkConfigs:       []*NetworkConfig{},
		PrimaryIPV4VPCCIDR:   ipAddr,
		AllIPV4VPCCIDRBlocks: []*net.IPNet{ipAddr},
	}
}

// getNetworkConfig is used to generate a dummy configuration for setting up the task namespace
func getNetworkConfig() *Config {
	config := getBaseConfig()

	eniNetworkConfig, _ := NewBridgeNetworkConfigForTaskNSSetup(getENI(), config)
	taskBridgeConfig, _ := NewBridgeNetworkConfigForTaskBridgeSetup(config)

	config.NetworkConfigs = append(config.NetworkConfigs,
		&NetworkConfig{
			IfName:           TaskENIBridgeNetworkPrefix,
			CNINetworkConfig: eniNetworkConfig,
		},
		&NetworkConfig{
			IfName:           TaskENIBridgeNetworkPrefix,
			CNINetworkConfig: taskBridgeConfig,
		},
	)
	return config
}

// TestSetupNS is used to test if the namespace is setup properly as per the provided configuration
func TestSetupNS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ecscniClient := NewClient("")
	libcniClient := mock_libcni.NewMockCNI(ctrl)
	ecscniClient.(*cniClient).libcni = libcniClient

	gomock.InOrder(
		// vpc-shared-eni plugin called to setup task namespace
		libcniClient.EXPECT().AddNetwork(gomock.Any(), gomock.Any(), gomock.Any()).Return(&current.Result{}, nil).Do(
			func(ctx context.Context, net *libcni.NetworkConfig, rt *libcni.RuntimeConf) {
				assert.Equal(t, ECSVPCSharedENIPluginExecutable, net.Network.Type, "first plugin should be vpc-shared-eni")
			}).Times(2),
	)

	config := getNetworkConfig()
	_, err := ecscniClient.SetupNS(context.TODO(), config, time.Second)
	assert.NoError(t, err)
}

// TestSetupNSTimeout tests the behavior when CNI plugin invocation returns an error
func TestSetupNSTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ecscniClient := NewClient("")
	libcniClient := mock_libcni.NewMockCNI(ctrl)
	ecscniClient.(*cniClient).libcni = libcniClient

	gomock.InOrder(
		// vpc-shared-eni plugin will be called first
		libcniClient.EXPECT().AddNetwork(gomock.Any(), gomock.Any(), gomock.Any()).Return(&current.Result{}, errors.New("timeout")).Do(
			func(ctx context.Context, net *libcni.NetworkConfig, rt *libcni.RuntimeConf) {
			}).MaxTimes(1),
	)

	config := getNetworkConfig()
	_, err := ecscniClient.SetupNS(context.TODO(), config, time.Millisecond)

	assert.Error(t, err)
}

// TestCleanupNS tests the cleanup of the task namespace when CleanupNS is called
func TestCleanupNS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ecscniClient := NewClient("")
	libcniClient := mock_libcni.NewMockCNI(ctrl)
	ecscniClient.(*cniClient).libcni = libcniClient

	libcniClient.EXPECT().DelNetwork(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(2)

	config := getNetworkConfig()
	err := ecscniClient.CleanupNS(context.TODO(), config, time.Second)

	assert.NoError(t, err)
}

// TestCleanupNSTimeout tests the behavior of CleanupNS when we get an error from CNI invocation
func TestCleanupNSTimeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ecscniClient := NewClient("")
	libcniClient := mock_libcni.NewMockCNI(ctrl)
	ecscniClient.(*cniClient).libcni = libcniClient

	// This will be called for both bridge and eni plugin
	libcniClient.EXPECT().DelNetwork(gomock.Any(), gomock.Any(), gomock.Any()).Do(
		func(x interface{}, y interface{}, z interface{}) {
		}).Return(errors.New("timeout")).MaxTimes(1)

	ctx, cancel := context.WithTimeout(context.TODO(), 1*time.Millisecond)
	defer cancel()

	config := getNetworkConfig()
	err := ecscniClient.CleanupNS(ctx, config, time.Millisecond)

	assert.Error(t, err)
}

// TestConstructNetworkConfig tests if we create an appropriate config from NewBridgeNetworkConfigForTaskNSSetup
func TestConstructNetworkConfig(t *testing.T) {

	config := getBaseConfig()

	taskENI := getENI()

	taskENIBridgeNetworkConfig, err := NewBridgeNetworkConfigForTaskNSSetup(taskENI, config)
	require.NoError(t, err, "failed to construct configuration for task ENI bridge")
	assert.Equal(t, TaskENIBridgeNetworkPrefix, taskENIBridgeNetworkConfig.Network.Name)

	taskENIBridgeConfig := &BridgeForTaskENIConfig{}
	err = json.Unmarshal(taskENIBridgeNetworkConfig.Bytes, taskENIBridgeConfig)
	require.NoError(t, err, "unmarshal bridge config from bytes failed")

	assert.Equal(t, ECSVPCSharedENIPluginName, taskENIBridgeConfig.Type)
	assert.Equal(t, "", taskENIBridgeConfig.ENIName)

	subnet := strings.Split(eniSubnetGatewayIPV4Address, "/")
	ipv4Addr := fmt.Sprintf("%s/%s", taskENI.GetPrimaryIPv4Address(), subnet[1])
	ipv4SecondaryAddr := fmt.Sprintf("%s/%s", eniIPV4SecondaryAddress, subnet[1])

	assert.EqualValues(t, ipv4Addr, taskENIBridgeConfig.ENIIPAddress)
	assert.EqualValues(t, ipv4SecondaryAddr, taskENIBridgeConfig.IPAddress)
	assert.EqualValues(t, taskENI.MacAddress, taskENIBridgeConfig.ENIMACAddress)
	assert.EqualValues(t, subnet[0], taskENIBridgeConfig.GatewayIPAddress)

	assert.False(t, taskENIBridgeConfig.TaskENIConfig.EnableTaskBridge)
	assert.True(t, taskENIBridgeConfig.TaskENIConfig.EnableTaskENI)
	assert.False(t, taskENIBridgeConfig.TaskENIConfig.NoInfra)

	taskBridgeNetworkConfig, err := NewBridgeNetworkConfigForTaskBridgeSetup(config)
	require.NoError(t, err, "failed to construct configuration for task bridge")
	assert.EqualValues(t, DefaultECSBridgeNetworkName, taskBridgeNetworkConfig.Network.Name)

	taskBridgeConfig := &BridgeForTaskENIConfig{}
	err = json.Unmarshal(taskBridgeNetworkConfig.Bytes, taskBridgeConfig)
	require.NoError(t, err, "unmarshal bridge config from bytes failed")

	assert.Equal(t, ECSVPCSharedENIPluginName, taskBridgeConfig.Type)
	assert.Equal(t, TaskENIBridgeNetworkPrefix, taskBridgeConfig.ENIName)

	assert.True(t, taskBridgeConfig.TaskENIConfig.EnableTaskBridge)
	assert.False(t, taskBridgeConfig.TaskENIConfig.EnableTaskENI)
	assert.False(t, taskBridgeConfig.TaskENIConfig.NoInfra)
}

// TestCNIPluginVersion tests if the string generated by version is correct
func TestCNIPluginVersion(t *testing.T) {
	testCases := []struct {
		version *cniPluginVersion
		str     string
	}{
		{
			version: &cniPluginVersion{
				Version:      "1",
				GitShortHash: "abcd",
				Built:        "July",
			},
			str: "abcd-1",
		},
		{
			version: &cniPluginVersion{
				Version:      "1",
				GitShortHash: "abcdef",
				Built:        "June",
			},
			str: "abcdef-1",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("version string %s", tc.str), func(t *testing.T) {
			assert.Equal(t, tc.str, tc.version.str())
		})
	}
}
