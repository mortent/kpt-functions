package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/setters2"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func init() {
	fieldmeta.SetShortHandRef("$kpt-set")
}

func main() {
	resourceList := &framework.ResourceList{}
	cmd := framework.Command(resourceList, func() error {
		// cmd.Execute() will parse the ResourceList.functionConfig into cmd.Flags from
		// the ResourceList.functionConfig.data field.

		var newItems []*yaml.RNode

		for i := range resourceList.Items {
			r := resourceList.Items[i]
			meta, err := r.GetMeta()
			if err != nil {
				return err
			}
			if meta.Kind == "LocalPackage" {
				replacementNodes, err := handleLocalPackage(r)
				if err != nil {
					return err
				}
				newItems = append(newItems, replacementNodes...)
			} else {
				newItems = append(newItems, r)
			}
		}
		resourceList.Items = newItems
		return nil
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func handleLocalPackage(node *yaml.RNode) ([]*yaml.RNode, error) {
	path, err := node.Pipe(yaml.Lookup("spec", "path"))
	if err != nil {
		return nil, err
	}
	packagePath, err := path.String()
	if err != nil {
		return nil, err
	}

	pkgContent, err := readPackageContent(strings.TrimSpace(packagePath))
	if err != nil {
		return nil, err
	}

	setterInfo, err := extractSetters(node)
	if err != nil {
		return nil, err
	}

	if len(setterInfo) > 0 {
		o, found, err := fetchOpenAPI(strings.TrimSpace(packagePath))
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("no setter found in Kptfile")
		}

		for i := range setterInfo {
			if err := o.PipeE(setters2.SetOpenAPI{
				Name: setterInfo[i].name,
				Value: setterInfo[i].value,
			}); err != nil {
				return nil, err
			}
		}

		err = loadOpenAPI(o)
		if err != nil {
			return nil, err
		}

		for i := range setterInfo {
			_, err = setters2.SetAll(&setters2.Set{
				Name: setterInfo[i].name,
			}).Filter(pkgContent)
			if err != nil {
				return nil, err
			}
		}
	}

	return pkgContent, nil
}

type setterInfo struct {
	name string
	value string
}

func extractSetters(node *yaml.RNode) ([]setterInfo, error) {
	node, err := node.Pipe(yaml.Lookup("spec", "setters"))
	if err != nil {
		return nil, err
	}

	var info []setterInfo
	err = node.VisitFields(func(node *yaml.MapNode) error {
		info = append(info, setterInfo{
			name: node.Key.YNode().Value,
			value: node.Value.YNode().Value,
		})
		return nil
	})
	return info, err
}

func fetchOpenAPI(path string) (*yaml.RNode, bool, error) {
	b, err := ioutil.ReadFile(filepath.Join(path, "Kptfile"))
	if err != nil {
		return nil, false, err
	}

	// parse the yaml file (json is a subset of yaml, so will also parse)
	y, err := yaml.Parse(string(b))
	if err != nil {
		return nil, false, err
	}
	return y, true, nil
}

func loadOpenAPI(node *yaml.RNode) error {
	defs := node.Field(openapi.SupplementaryOpenAPIFieldName)

	oAPI, err := defs.Value.String()
	if err != nil {
		return err
	}

	var o interface{}
	err = yaml.Unmarshal([]byte(oAPI), &o)
	if err != nil {
		return err
	}
	j, err := json.Marshal(o)
	if err != nil {
		return err
	}

	// add the json schema to the global schema
	_, err = openapi.AddSchema(j)
	if err != nil {
		return err
	}
	return nil
}

func readPackageContent(path string) ([]*yaml.RNode, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fullPath := filepath.Join(currentDir, path)
	return kio.LocalPackageReader{
		PackagePath: fullPath,
	}.Read()
}
