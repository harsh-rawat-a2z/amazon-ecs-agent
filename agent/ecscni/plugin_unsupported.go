// +build !linux,!windows

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
	"errors"
	"time"

	"github.com/containernetworking/cni/libcni"
)

var (
	// vpcCNIPluginPath is the path of VPC CNI plugin log file
	vpcCNIPluginPath = "/log/vpc-branch-eni.log"
)

type cniPluginVersion struct{}

// ReleaseIPResource marks the ip available in the ipam db
// On unsupported platforms, we will return an error
func (client *cniClient) ReleaseIPResource(ctx context.Context, cfg *Config, timeout time.Duration) error {
	return errors.New("unsupported platform")
}

// str generates a string version of the CNI plugin version
// On unsupported platforms, we will return an empty string
func (version *cniPluginVersion) str() string {
	return ""
}

// isBridgePluginExecution returns if the cni plugin execution was for creating task bridge
func isBridgePluginExecution(cniNetworkConfig *libcni.NetworkConfig) bool {
	return false
}
