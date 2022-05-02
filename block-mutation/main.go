package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	LocalFnPath = "/localconfig/fn-config.yaml"
)

// The block-mutations kpt function checks whether a kpt plan resource
// contains changes in disallowed fields. This allows users to extend
// the kpt plan command with custom rules that can leverage the diff between
// the current live state and the new state.
//
// This function is a POC.
func main() {
	asp := BlockMutationProcessor{}
	cmd := command.Build(&asp, command.StandaloneEnabled, false)

	cmd.Short = ""
	cmd.Long = ""
	cmd.Example = ""
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

type BlockMutationProcessor struct{}

type BlockMutationConfig struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`
	ResourceFields    []ResourceField `json:"resourceFields" yaml:"resourceFields"`
}

type ResourceField struct {
	Group string `json:"group,omitempty" yaml:"group,omitempty"`
	Kind  string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
}

func (bmp *BlockMutationProcessor) Process(resourceList *framework.ResourceList) error {
	err := run(resourceList)
	if err != nil {
		resourceList.Result = &framework.Result{
			Name: "block-mutation",
			Items: []framework.ResultItem{
				{
					Message:  err.Error(),
					Severity: framework.Error,
				},
			},
		}
		return resourceList.Result
	}
	return nil
}

func findConfig(resourceList *framework.ResourceList) (*BlockMutationConfig, error) {
	var bmc BlockMutationConfig
	if !isEmptyFnConfig(resourceList.FunctionConfig) {
		if err := framework.LoadFunctionConfig(resourceList.FunctionConfig, &bmc); err != nil {
			return nil, err
		}
		return &bmc, nil
	}

	if _, err := os.Stat(LocalFnPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &bmc, nil
		}
		return nil, err
	}

	bytes, err := ioutil.ReadFile(LocalFnPath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(bytes, &bmc); err != nil {
		return nil, err
	}
	return &bmc, nil
}

func isEmptyFnConfig(rn *yaml.RNode) bool {
	if rn.GetApiVersion() != "v1" || rn.GetKind() != "ConfigMap" {
		return false
	}
	return len(rn.GetDataMap()) == 0
}

func run(resourceList *framework.ResourceList) error {
	bmc, err := findConfig(resourceList)
	if err != nil {
		return err
	}

	var plan *yaml.RNode
	if plan = findPlan(resourceList); plan == nil {
		return fmt.Errorf("no plan resource found")
	}

	l, err := plan.Pipe(yaml.Lookup("spec", "actions"))
	if err != nil {
		return err
	}

	elems, err := l.Elements()
	if err != nil {
		return err
	}
	for i := range elems {
		action := elems[i]
		apiVersion, err := action.Pipe(yaml.Lookup("apiVersion"))
		if err != nil {
			return err
		}
		kind, err := action.Pipe(yaml.Lookup("kind"))
		if err != nil {
			return err
		}
		a, err := action.Pipe(yaml.Lookup("action"))
		if err != nil {
			return err
		}
		if a.YNode().Value != "Update" {
			continue
		}

		for j := range bmc.ResourceFields {
			rf := bmc.ResourceFields[j]
			if rf.Group == apiVersion.YNode().Value && rf.Kind == kind.YNode().Value {
				if err := checkField(action, rf); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func checkField(action *yaml.RNode, rf ResourceField) error {
	before, err := action.Pipe(yaml.Lookup("original"))
	if err != nil {
		return err
	}
	after, err := action.Pipe(yaml.Lookup("updated"))
	if err != nil {
		return err
	}

	fullPath := strings.Split(rf.Field, ".")

	beforeValNode, err := before.Pipe(yaml.Lookup(fullPath...))
	if err != nil {
		return err
	}
	afterValNode, err := after.Pipe(yaml.Lookup(fullPath...))
	if err != nil {
		return err
	}

	beforeVal, err := beforeValNode.String()
	if err != nil {
		return err
	}
	beforeVal = strings.TrimSpace(beforeVal)
	afterVal, err := afterValNode.String()
	if err != nil {
		return err
	}
	afterVal = strings.TrimSpace(afterVal)

	if beforeVal != afterVal {
		return fmt.Errorf("field %q changed from %q to %q", rf.Field, beforeVal, afterVal)
	}

	return nil
}

func findPlan(resourceList *framework.ResourceList) *yaml.RNode {
	for i := range resourceList.Items {
		item := resourceList.Items[i]

		if item.GetApiVersion() == "kpt.dev/v1alpha1" && item.GetKind() == "Plan" {
			return item
		}
	}
	return nil
}
