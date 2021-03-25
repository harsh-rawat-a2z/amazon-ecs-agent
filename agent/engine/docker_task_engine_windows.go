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

package engine

import (
	"context"
	"fmt"

	"github.com/aws/amazon-ecs-agent/agent/dockerclient"

	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	apitask "github.com/aws/amazon-ecs-agent/agent/api/task"
	"github.com/aws/amazon-ecs-agent/agent/ecscni"
	"github.com/cihub/seelog"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
)

// invokePluginForContainer is used to invoke the CNI plugin for the given container
func (engine *DockerTaskEngine) invokePluginsForContainer(task *apitask.Task, container *apicontainer.Container) error {
	containerInspectOutput, err := engine.inspectContainer(task, container)
	if err != nil {
		return errors.Wrapf(err, "error occurred while inspecting container %v", container.Name)
	}

	cniConfig, err := engine.buildCNIConfigFromTaskContainer(task, containerInspectOutput, false)
	if err != nil {
		return errors.Wrap(err, "unable to build cni configuration")
	}

	// Invoke the cni plugin for the container using libcni
	_, err = engine.cniClient.SetupNS(engine.ctx, cniConfig, cniSetupTimeout)
	if err != nil {
		seelog.Errorf("Task engine [%s]: unable to configure container %v in the pause namespace: %v", task.Arn, container.Name, err)
		return errors.Wrap(err, "failed to connect HNS endpoint to container")
	}

	return nil
}

func (engine *DockerTaskEngine) invokeCommandsForTaskBridgeSetup(ctx context.Context, task *apitask.Task,
	config *ecscni.Config, result *current.Result) error {

	seelog.Info("Task [%s]: Executing commands inside pause namespace for setting up task bridge", task.Arn)
	execCfg := types.ExecConfig{
		User:   "ContainerAdministrator",
		Detach: true,
		Cmd:    []string{fmt.Sprintf(ecscni.CredentialsEndpointRouteAdditionCmd, result.IPs[0].Gateway.String())},
	}

	execRes, err := engine.client.CreateContainerExec(ctx, config.ContainerID, execCfg, dockerclient.ContainerExecCreateTimeout)
	if err != nil {
		seelog.Errorf("Failed to execute commands in pause namespace [create]: %v", err)
		return errors.Wrapf(err, "failed to execute commands in pause namespace")
	}

	err = engine.client.StartContainerExec(ctx, execRes.ID, dockerclient.ContainerExecStartTimeout)
	if err != nil {
		seelog.Errorf("Failed to execute commands in pause namespace [pre-start]: %v", err)
		return errors.Wrapf(err, "failed to execute commands in pause namespace")
	}

	inspect, err := engine.client.InspectContainerExec(ctx, execRes.ID, dockerclient.ContainerExecInspectTimeout)
	if err != nil {
		seelog.Errorf("Failed to execute commands in pause namespace [inspect]: %v", err)
		return errors.Wrapf(err, "failed to execute commands in pause namespace")
	}

	seelog.Infof("Harsh : exit %v \t running: %v", inspect.ExitCode, inspect.Running)
	return nil
}
