// +build linux

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

package engine

import (
	"context"

	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	apitask "github.com/aws/amazon-ecs-agent/agent/api/task"
	"github.com/aws/amazon-ecs-agent/agent/ecscni"
	"github.com/containernetworking/cni/pkg/types/current"
)

// invokePluginForContainer is used to invoke the CNI plugin for the given container
// On non-windows platform, we will not invoke CNI plugins for non-pause containers
func (engine *DockerTaskEngine) invokePluginsForContainer(task *apitask.Task, container *apicontainer.Container) error {
	return nil
}

func (engine *DockerTaskEngine) invokeCommandsForTaskBridgeSetup(ctx context.Context, task *apitask.Task,
	config *ecscni.Config, result *current.Result) error {
	return nil
}