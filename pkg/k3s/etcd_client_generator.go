/*
Copyright 2020 The Kubernetes Authors.

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

package k3s

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/k3s-io/cluster-api-k3s/pkg/etcd"
	"github.com/k3s-io/cluster-api-k3s/pkg/etcd/util"
	"github.com/k3s-io/cluster-api-k3s/pkg/proxy"
)

// EtcdClientGenerator generates etcd clients that connect to specific etcd members on particular control plane nodes.
type EtcdClientGenerator struct {
	restConfig   *rest.Config
	tlsConfig    *tls.Config
	createClient clientCreator
	etcdPodMap   *map[string]string
}

type clientCreator func(ctx context.Context, endpoint string) (*etcd.Client, error)

var errEtcdNodeConnection = errors.New("failed to connect to etcd node")

// NewEtcdClientGenerator returns a new etcdClientGenerator instance.
func NewEtcdClientGenerator(restConfig *rest.Config, tlsConfig *tls.Config, etcdDialTimeout, etcdCallTimeout time.Duration) *EtcdClientGenerator {
	ecg := &EtcdClientGenerator{restConfig: restConfig, tlsConfig: tlsConfig}
	ecg.createClient = func(ctx context.Context, endpoint string) (*etcd.Client, error) {
		p := proxy.Proxy{
			Kind:       "pods",
			Namespace:  metav1.NamespaceSystem,
			KubeConfig: ecg.restConfig,
			Port:       2379,
		}
		return etcd.NewClient(ctx, etcd.ClientConfiguration{
			Endpoint:    endpoint,
			Proxy:       p,
			TLSConfig:   tlsConfig,
			DialTimeout: etcdDialTimeout,
			CallTimeout: etcdCallTimeout,
		})
	}

	return ecg
}

func (c *EtcdClientGenerator) findEtcdProxyPod(ctx context.Context, nodeName string) (string, error) {
	if c.etcdPodMap == nil {
		clientset, err := kubernetes.NewForConfig(c.restConfig)
		if err != nil {
			return "", errors.Wrap(err, "unable to create client to target cluster")
		}

		pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{LabelSelector: "app=etcd-proxy"})
		if err != nil {
			return "", errors.Wrap(err, "unable to list etcd-proxy pods in target cluster")
		}

		if len(pods.Items) == 0 {
			return "", errors.New("there isn't any etcd-proxy pods in target cluster")
		}

		etcdmap := make(map[string]string)
		for _, pod := range pods.Items {
			etcdmap[pod.Spec.NodeName] = pod.Name
		}

		c.etcdPodMap = &etcdmap
	}

	podName, exists := (*c.etcdPodMap)[nodeName]
	if !exists {
		return "", errors.New(fmt.Sprintf("unable to find etcd proxy pod for node %s", nodeName))
	}

	return podName, nil
}

// forFirstAvailableNode takes a list of nodes and returns a client for the first one that connects.
func (c *EtcdClientGenerator) forFirstAvailableNode(ctx context.Context, nodeNames []string) (*etcd.Client, error) {
	// This is an additional safeguard for avoiding this func to return nil, nil.
	if len(nodeNames) == 0 {
		return nil, errors.New("invalid argument: forLeader can't be called with an empty list of nodes")
	}

	clientset, err := kubernetes.NewForConfig(c.restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create client to target cluster")
	}

	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{LabelSelector: "app=etcd-proxy"})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list etcd-proxy pods in target cluster")
	}

	etcdmap := make(map[string]string)
	for _, pod := range pods.Items {
		etcdmap[pod.Spec.NodeName] = pod.Name
	}

	// Loop through the existing control plane nodes.
	var errs []error
	for _, name := range nodeNames {
		podName, err := c.findEtcdProxyPod(ctx, name)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		client, err := c.createClient(ctx, podName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return client, nil
	}
	return nil, errors.Wrap(kerrors.NewAggregate(errs), "could not establish a connection to any etcd node")
}

// forLeader takes a list of nodes and returns a client to the leader node.
func (c *EtcdClientGenerator) forLeader(ctx context.Context, nodeNames []string) (*etcd.Client, error) {
	// This is an additional safeguard for avoiding this func to return nil, nil.
	if len(nodeNames) == 0 {
		return nil, errors.New("invalid argument: forLeader can't be called with an empty list of nodes")
	}

	nodes := sets.Set[string]{}
	for _, n := range nodeNames {
		nodes.Insert(n)
	}

	// Loop through the existing control plane nodes.
	var errs []error
	for _, nodeName := range nodeNames {
		cl, err := c.getLeaderClient(ctx, nodeName, nodes)
		if err != nil {
			if errors.Is(err, errEtcdNodeConnection) {
				errs = append(errs, err)
				continue
			}
			return nil, err
		}

		return cl, nil
	}
	return nil, errors.Wrap(kerrors.NewAggregate(errs), "could not establish a connection to the etcd leader")
}

// getLeaderClient provides an etcd client connected to the leader. It returns an
// errEtcdNodeConnection if there was a connection problem with the given etcd
// node, which should be considered non-fatal by the caller.
func (c *EtcdClientGenerator) getLeaderClient(ctx context.Context, nodeName string, allNodes sets.Set[string]) (*etcd.Client, error) {
	// Get a temporary client to the etcd instance hosted on the node.
	client, err := c.forFirstAvailableNode(ctx, []string{nodeName})
	if err != nil {
		return nil, kerrors.NewAggregate([]error{err, errEtcdNodeConnection})
	}
	defer client.Close()

	// Get the list of members.
	members, err := client.Members(ctx)
	if err != nil {
		return nil, kerrors.NewAggregate([]error{err, errEtcdNodeConnection})
	}

	// Get the leader member.
	var leaderMember *etcd.Member
	for _, member := range members {
		if member.ID == client.LeaderID {
			leaderMember = member
			break
		}
	}

	// If we found the leader, and it is one of the nodes,
	// get a connection to the etcd leader via the node hosting it.
	if leaderMember != nil {
		nodeName := util.NodeNameFromMember(leaderMember)
		if !allNodes.Has(nodeName) {
			return nil, errors.Errorf("etcd leader is reported as %x with node name %q, but we couldn't find a corresponding Node in the cluster", leaderMember.ID, nodeName)
		}
		client, err = c.forFirstAvailableNode(ctx, []string{nodeName})
		return client, err
	}

	// If it is not possible to get a connection to the leader via existing nodes,
	// it means that the control plane is an invalid state, with an etcd member - the current leader -
	// without a corresponding node.
	// TODO: In future we can eventually try to automatically remediate this condition by moving the leader
	//  to another member with a corresponding node.
	return nil, errors.Errorf("etcd leader is reported as %x, but we couldn't find any matching member", client.LeaderID)
}
