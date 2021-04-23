/*
Copyright 2015 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package missionmanager

import (
	"fmt"

	missionv1 "github.com/kubeedge/kubeedge/cloud/pkg/apis/missions/v1"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/edgecore/v1alpha1"

	"k8s.io/klog/v2"
)

const (
	COMMAND_TIMEOUT_SEC = 10
)

var distributionToKubectl = map[string]string {
		"arktos" : "/home/ubuntu/kubectl",
	  }

type Manager struct {
	clusterName string 
	clusterLabels map[string]string
	kubedistrubtion string
	kubeconfigFile string
	kubectlCli string
}

//NewMissionManager creates new mission manager object
func NewMissionManager(edgeClusterConfig *v1alpha1.EdgeCluster) (*Manager, error) {
	if edgeClusterConfig == nil || edgeClusterConfig.Enable == false {
		return nil, fmt.Errorf("edge cluster is not enabled")
	}

	if !FileExists(edgeClusterConfig.ClusterKubeconfig) {
		return nil, fmt.Errorf("Could not open kubeconfig file (%s)", edgeClusterConfig.ClusterKubeconfig)
	}

	if _, exists := distributionToKubectl[edgeClusterConfig.KubeDistribution]; !exists {
		return nil, fmt.Errorf("Invalid kube distribution (%v)", edgeClusterConfig.KubeDistribution)
	}

	if len(edgeClusterConfig.ClusterName) == 0 {
		return nil, fmt.Errorf("cluster name cannot be empty!")
	}

	return &Manager {
		clusterName: edgeClusterConfig.ClusterName,
		clusterLabels: edgeClusterConfig.ClusterLabels,
		kubedistrubtion: edgeClusterConfig.KubeDistribution,
		kubeconfigFile: edgeClusterConfig.ClusterKubeconfig,
		kubectlCli: distributionToKubectl[edgeClusterConfig.KubeDistribution],
	}, nil	
}

func (m *Manager) ApplyMission(mission *missionv1.Mission) error {
	if m.isMatchingMission(mission) == false {
		return nil
	}

	klog.V(4).Infof("Apply mission %#v", *mission)

	deploy_mission_cmd := fmt.Sprintf("printf \"%s\" | %s apply %s -f - ", mission.Spec.Content, m.kubectlCli, m.kubeconfigFile)
	exitcode, output, err := ExecCommandLine(deploy_mission_cmd, COMMAND_TIMEOUT_SEC)
	if exitcode != 0 || err != nil {
		return fmt.Errorf("Command (%s) failed: exit code: v, output: %v, err: %v", exitcode, output, err)
	}
	
	return nil
}


func (m *Manager) DeleteMission(mission *missionv1.Mission) error {
	if m.isMatchingMission(mission) == false {
		return nil
	}

	klog.V(4).Infof("Delete mission %#v", *mission)

	deploy_mission_cmd := fmt.Sprintf("printf \"%s\" | %s delete %s -f - ", mission.Spec.Content, m.kubectlCli, m.kubeconfigFile)
	exitcode, output, err := ExecCommandLine(deploy_mission_cmd, COMMAND_TIMEOUT_SEC)
	if exitcode != 0 || err != nil {
		return fmt.Errorf("Command (%s) failed: exit code: v, output: %v, err: %v", exitcode, output, err)
	}

	return nil
}

func (m *Manager) isMatchingMission(mission *missionv1.Mission) bool {
	// if the placement field is empty, it matches all the edge clusters
	if len(mission.Spec.Placement.Clusters) == 0 && len(mission.Spec.Placement.MatchLabels) == 0 {
		return true
	}

	for _, matchingCluster := range mission.Spec.Placement.Clusters {
		if m.clusterName == matchingCluster.Name {
			return true
		}
	}

	// TODO: use k8s Labels operator to match
	if len(mission.Spec.Placement.MatchLabels) == 0 {
		return false
	}

	for k, v := range mission.Spec.Placement.MatchLabels {
		if val, ok := m.clusterLabels[k]; ok && val == v {
			return true
		}
	}

	return false
}