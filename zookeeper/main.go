package main

import (
	"fmt"
	"os"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"strconv"
	"strings"
)

func main() {
	resourceList := &framework.ResourceList{}
	cmd := framework.Command(resourceList, func() error {
		for i := range resourceList.Items {
			r := resourceList.Items[i]
			err := processResource(r)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func processResource(node *yaml.RNode) error {
	n, err := node.Pipe(yaml.PathGetter{
		Path: []string{"spec", "template", "spec", "containers", "[name=zookeeper]", "env", "[name=ZOO_SERVERS]", "value"},
	})
	if err != nil {
		return err
	}
	if n == nil {
		return nil
	}

	nameNode, err := node.Pipe(yaml.PathGetter{
		Path: []string{"metadata", "name"},
	})
	if err != nil {
		return err
	}
	name := nameNode.YNode().Value

	nsNode, err := node.Pipe(yaml.PathGetter{
		Path: []string{"metadata", "namespace"},
	})
	if err != nil {
		return err
	}
	ns := nsNode.YNode().Value

	serviceNameNode, err := node.Pipe(yaml.PathGetter{
		Path: []string{"spec", "serviceName"},
	})
	if err != nil {
		return err
	}
	serviceName := serviceNameNode.YNode().Value

	replicaNode, err := node.Pipe(yaml.PathGetter{
		Path: []string{"spec", "replicas"},
	})
	if err != nil {
		return err
	}
	replicas := replicaNode.YNode().Value
	reps, err := strconv.Atoi(replicas)
	if err != nil {
		return err
	}

	var builder strings.Builder
	for i := 0; i < reps; i++ {
		builder.WriteString(
			fmt.Sprintf("%s-%d.%s.%s.svc.cluster.local:2888:3888 ",
				name, i, serviceName, ns))
	}
	n.YNode().Value = strings.TrimSpace(builder.String())
	return nil
}