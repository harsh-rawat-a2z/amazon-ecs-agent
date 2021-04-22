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
	apicontainer "github.com/aws/amazon-ecs-agent/agent/api/container"
	apitask "github.com/aws/amazon-ecs-agent/agent/api/task"
	"github.com/aws/amazon-ecs-agent/agent/dockerclient"
	"github.com/aws/amazon-ecs-agent/agent/ecscni"
	"github.com/cihub/seelog"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
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

	gateway := result.IPs[0].Gateway.String()
	command1 := strings.Split(fmt.Sprintf(ecscni.DefaultBridgeRouteDeleteCmd, ecscni.RouteExecutable, gateway), " ")
	err := engine.invokeCommand(ctx, task, config, command1)
	if err != nil {
		return err
	}

	if !config.BlockInstanceMetadata{
		imdsRouteAddCmd := strings.Split(fmt.Sprintf(ecscni.IMDSRouteAdditionCmd, ecscni.RouteExecutable, config.TaskPrimaryGateway), " ")
		err := engine.invokeCommand(ctx, task, config, imdsRouteAddCmd)
		if err != nil {
			return err
		}
	} else {
		cmd := fmt.Sprintf("netsh advfirewall firewall add rule name=\"Disable IMDS for %s\" dir=out localip=%s remoteip=169.254.169.254 action=block\n",
			config.TaskPrimaryIP, config.TaskPrimaryIP)
		c := exec.Command("cmd", "/C", cmd)
		c.Run()
	}

	//command2 := strings.Split(fmt.Sprintf(ecscni.DefaultCredEndpointAdd, ecscni.RouteExecutable, gateway), " ")
	//err = engine.invokeCommand(ctx, task, config, command2)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (engine *DockerTaskEngine) invokeCommand(ctx context.Context, task *apitask.Task,
	config *ecscni.Config, command []string) error {

	seelog.Infof("Task [%s]: Executing commands inside pause namespace %v", task.Arn, command)
	execCfg := types.ExecConfig{
		Detach: false,
		Cmd:    command,
		User: "ContainerAdministrator",
	}

	execRes, err := engine.client.CreateContainerExec(ctx, config.ContainerID, execCfg, dockerclient.ContainerExecCreateTimeout)
	if err != nil {
		seelog.Errorf("Failed to execute command in pause namespace [create]: %v", err)
		return errors.Wrapf(err, "failed to execute command in pause namespace")
	}

	err = engine.client.StartContainerExec(ctx, execRes.ID, dockerclient.ContainerExecStartTimeout)
	if err != nil {
		seelog.Errorf("Failed to execute command in pause namespace [pre-start]: %v", err)
		return errors.Wrapf(err, "failed to execute command in pause namespace")
	}

	inspect, err := engine.client.InspectContainerExec(ctx, execRes.ID, dockerclient.ContainerExecInspectTimeout)
	if err != nil {
		seelog.Errorf("Failed to execute command in pause namespace [inspect]: %v", err)
		return errors.Wrapf(err, "failed to execute command in pause namespace")
	}

	if !inspect.Running && inspect.ExitCode != 0 {
		return errors.Errorf("failed to execute command in pause namespace: %v", command)
	}
	seelog.Infof("Information %v --Harsh ", inspect)
	return nil
}
