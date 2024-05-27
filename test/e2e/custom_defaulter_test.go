//go:build e2e
// +build e2e

/*
Copyright 2021 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

var _ = Describe("Custom Defaulter", func() {
	var (
		ctx                    = context.TODO()
		specName               = "custom-defaulter"
		namespace              *corev1.Namespace
		cancelWatches          context.CancelFunc
		result                 *ApplyClusterTemplateAndWaitResult
		clusterName            string
		clusterctlLogFolder    string
		infrastructureProvider string
	)

	BeforeEach(func() {
		Expect(e2eConfig.Variables).To(HaveKey(KubernetesVersion))

		clusterName = fmt.Sprintf("capik3s-custom-defaulter-%s", util.RandomString(6))
		infrastructureProvider = "docker"

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, bootstrapClusterProxy, artifactFolder)

		result = new(ApplyClusterTemplateAndWaitResult)

		clusterctlLogFolder = filepath.Join(artifactFolder, "clusters", bootstrapClusterProxy.GetName())
	})

	AfterEach(func() {
		cleanInput := cleanupInput{
			SpecName:        specName,
			Cluster:         result.Cluster,
			ClusterProxy:    bootstrapClusterProxy,
			Namespace:       namespace,
			CancelWatches:   cancelWatches,
			IntervalsGetter: e2eConfig.GetIntervals,
			SkipCleanup:     skipCleanup,
			ArtifactFolder:  artifactFolder,
		}

		dumpSpecResourcesAndCleanup(ctx, cleanInput)
	})

	Context("Creating a cluster", func() {
		It("Should create a workload cluster with default values [TESTY-TEST]", func() {
			By("Creating a workload cluster")
			ApplyClusterTemplateAndWait(ctx, ApplyClusterTemplateAndWaitInput{
				ClusterProxy: bootstrapClusterProxy,
				ConfigCluster: clusterctl.ConfigClusterInput{
					LogFolder:                clusterctlLogFolder,
					ClusterctlConfigPath:     clusterctlConfigPath,
					KubeconfigPath:           bootstrapClusterProxy.GetKubeconfigPath(),
					InfrastructureProvider:   infrastructureProvider,
					Namespace:                namespace.Name,
					Flavor:                   "v1beta2",
					ClusterName:              clusterName,
					KubernetesVersion:        e2eConfig.GetVariable(KubernetesVersion),
					ControlPlaneMachineCount: pointer.Int64Ptr(1),
					WorkerMachineCount:       pointer.Int64Ptr(1),
				},
				WaitForClusterIntervals:      e2eConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: e2eConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    e2eConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, result)

			// Expect control plane default values
			Expect(*result.ControlPlane.Spec.KThreesConfigSpec.ServerConfig.DisableCloudController).Should(Equal(true))
			Expect(*result.ControlPlane.Spec.KThreesConfigSpec.ServerConfig.CloudProviderName).Should(Equal("external"))
			// Expect bootstrap default values
			Expect(result.BootstrapConfigs.Items).Should(HaveLen(2))
			Expect(*result.BootstrapConfigs.Items[0].Spec.ServerConfig.DisableCloudController).Should(Equal(true))
			Expect(*result.BootstrapConfigs.Items[0].Spec.ServerConfig.CloudProviderName).Should(Equal("external"))
			Expect(*result.BootstrapConfigs.Items[1].Spec.ServerConfig.DisableCloudController).Should(Equal(true))
			Expect(*result.BootstrapConfigs.Items[1].Spec.ServerConfig.CloudProviderName).Should(Equal("external"))
		})
	})
})
