// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/onosproject/onos-test/pkg/onit"
	"github.com/spf13/cobra"
)

var (
	createExample = `
		# Create a cluster with a given name that contains one instance of each subsystem (e.g. onos-config, onos-topo)
		onit create cluster onit-cluster-1 

		# Create a cluster that contains two instances of onos-config subsystem and two instances of onos-topo subsystem
		onit-create-cluster onit-cluster-2 --topo-nodes 2 --config-nodes 2

		# Create a cluster that has two 3-node raft partitions
		onit create cluster --partitions 2 --partition-size 3

		# Create a cluster that fetches docker images from a private docker registry
		onit create cluster --docker-registry <host>:<port>
	
		# Create a cluster to deploy topo and config subsystems using the images with custom tags 
        onit create cluster --image-tags topo=test-topo-tag,config=test-config-tag`
)

// getCreateCommand returns a cobra "setup" command for setting up resources
func getCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create {cluster} [args]",
		Short:   "Create a test resource on Kubernetes",
		Example: createExample,
	}
	cmd.AddCommand(getCreateClusterCommand())
	return cmd
}

func initImageTags(imageTags map[string]string) {
	if imageTags["config"] == "" {
		imageTags["config"] = string(onit.Debug)
	}
	if imageTags["topo"] == "" {
		imageTags["topo"] = string(onit.Debug)
	}
	if imageTags["gui"] == "" {
		imageTags["gui"] = string(onit.Latest)
	}
	if imageTags["cli"] == "" {
		imageTags["cli"] = string(onit.Latest)
	}
	if imageTags["atomix"] == "" {
		imageTags["atomix"] = string(onit.Latest)
	}
	if imageTags["raft"] == "" {
		imageTags["raft"] = string(onit.Latest)
	}
	if imageTags["simulator"] == "" {
		imageTags["simulator"] = string(onit.Latest)
	}
	if imageTags["stratum"] == "" {
		imageTags["stratum"] = string(onit.Latest)
	}
	if imageTags["test"] == "" {
		imageTags["test"] = string(onit.Latest)
	}

}

// getCreateClusterCommand returns a cobra command for deploying a test cluster
func getCreateClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster [id]",
		Short: "Setup a test cluster on Kubernetes",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dockerRegistry, _ := cmd.Flags().GetString("docker-registry")
			configNodes, _ := cmd.Flags().GetInt("config-nodes")
			topoNodes, _ := cmd.Flags().GetInt("topo-nodes")
			partitions, _ := cmd.Flags().GetInt("partitions")
			partitionSize, _ := cmd.Flags().GetInt("partition-size")
			configName, _ := cmd.Flags().GetString("config")
			imageTags, _ := cmd.Flags().GetStringToString("image-tags")
			imagePullPolicy, _ := cmd.Flags().GetString("image-pull-policy")
			pullPolicy := corev1.PullPolicy(imagePullPolicy)

			if pullPolicy != corev1.PullAlways && pullPolicy != corev1.PullIfNotPresent && pullPolicy != corev1.PullNever {
				exitError(fmt.Errorf("invalid pull policy; must of one of %s, %s or %s", corev1.PullAlways, corev1.PullIfNotPresent, corev1.PullNever))
			}

			initImageTags(imageTags)

			// Get the onit controller
			controller, err := onit.NewController()
			if err != nil {
				exitError(err)
			}

			// Get or create a cluster ID
			var clusterID string
			if len(args) > 0 {
				clusterID = args[0]
			} else {
				clusterID = fmt.Sprintf("cluster-%s", newUUIDString())
			}

			// Create the cluster configuration
			config := &onit.ClusterConfig{
				Registry:      dockerRegistry,
				Preset:        configName,
				ImageTags:     imageTags,
				PullPolicy:    pullPolicy,
				ConfigNodes:   configNodes,
				TopoNodes:     topoNodes,
				Partitions:    partitions,
				PartitionSize: partitionSize,
			}

			// Create the cluster controller
			cluster, status := controller.NewCluster(clusterID, config)
			if status.Failed() {
				exitStatus(status)
			}

			// Store the cluster before setting it up to ensure other shell sessions can debug setup
			err = setDefaultCluster(clusterID)
			if err != nil {
				exitError(err)
			}

			// Setup the cluster
			if status := cluster.Setup(); status.Failed() {
				exitStatus(status)
			} else {
				fmt.Println(clusterID)
			}
		},
	}

	imageTags := make(map[string]string)
	imageTags["config"] = string(onit.Debug)
	imageTags["topo"] = string(onit.Debug)
	imageTags["simulator"] = string(onit.Latest)
	imageTags["stratum"] = string(onit.Latest)
	imageTags["test"] = string(onit.Latest)
	imageTags["atomix"] = string(onit.Latest)
	imageTags["raft"] = string(onit.Latest)
	imageTags["gui"] = string(onit.Latest)
	imageTags["cli"] = string(onit.Latest)

	cmd.Flags().StringP("config", "c", "default", "test cluster configuration")
	cmd.Flags().String("docker-registry", "", "an optional host:port for a private Docker registry")
	cmd.Flags().Int("config-nodes", 1, "the number of onos-config nodes to deploy")
	cmd.Flags().Int("topo-nodes", 1, "the number of onos-topo nodes to deploy")
	cmd.Flags().IntP("partitions", "p", 1, "the number of Raft partitions to deploy")
	cmd.Flags().IntP("partition-size", "s", 1, "the size of each Raft partition")
	cmd.Flags().StringToString("image-tags", imageTags, "the image docker container tag for each node in the cluster")
	cmd.Flags().String("image-pull-policy", string(corev1.PullIfNotPresent), "the Docker image pull policy")

	return cmd
}
