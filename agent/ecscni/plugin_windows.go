// +build windows

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
	"os"
	"path/filepath"
	"time"

	"github.com/aws/amazon-ecs-agent/agent/utils"
	"github.com/containernetworking/cni/libcni"
)

var (
	// vpcCNIPluginPath is the path of VPC CNI plugin log file
	vpcCNIPluginPath = filepath.Join(utils.DefaultIfBlank(os.Getenv("ProgramData"), `C:\ProgramData`), "Amazon", "ECS", "log", "vpc-shared-eni.log")
)

// ReleaseIPResource marks the ip available in the ipam db
// This method is not required in Windows. HNS takes care of IP management.
func (client *cniClient) ReleaseIPResource(ctx context.Context, cfg *Config, timeout time.Duration) error {
	return nil
}

// Version is the version number of the repository.
// GitShortHash is the short hash of the Git HEAD.
// Built is the build time stamp.
type cniPluginVersion struct {
	Version      string `json:"version"`
	GitShortHash string `json:"gitShortHash"`
	Built        string `json:"built"`
}

// str generates a string version of the CNI plugin version
func (version *cniPluginVersion) str() string {
	return version.GitShortHash + "-" + version.Version
}

// isBridgePluginExecution returns if the cni plugin execution was for creating task bridge
func isBridgePluginExecution(cniNetworkConfig *libcni.NetworkConfig) bool {
	return cniNetworkConfig.Network.Type == ECSVPCSharedENIPluginExecutable && cniNetworkConfig.Network.Name == DefaultECSBridgeNetworkName
}
