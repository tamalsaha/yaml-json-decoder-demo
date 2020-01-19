package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

/*

func (c *Client) namespace() string {
	if ns, _, err := c.Factory.ToRawKubeConfigLoader().Namespace(); err == nil {
		return ns
	}
	return v1.NamespaceDefault
}

// newBuilder returns a new resource builder for structured api objects.
func (c *Client) newBuilder() *resource.Builder {
	return c.Factory.NewBuilder().
		ContinueOnError().
		NamespaceParam(c.namespace()).
		DefaultNamespace().
		Flatten()
}

// Build validates for Kubernetes objects and returns unstructured infos.
func (c *Client) Build(reader io.Reader, validate bool) (ResourceList, error) {
	schema, err := c.Factory.Validator(validate)
	if err != nil {
		return nil, err
	}
	result, err := c.newBuilder().
		Unstructured().
		Schema(schema).
		Stream(reader, "").
		Do().Infos()
	return result, scrubValidationError(err)
}

*/
func main() {
	var resources []*unstructured.Unstructured

	err := filepath.Walk("testdata", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		reader := yaml.NewYAMLOrJSONDecoder(f, 2048)
		for {
			var obj unstructured.Unstructured
			err = reader.Decode(&obj)
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
			if obj.IsList() {
				err := obj.EachListItem(func(item runtime.Object) error {
					castItem := item.(*unstructured.Unstructured)
					if castItem.GetNamespace() == "" {
						castItem.SetNamespace(core.NamespaceDefault)
					}
					resources = append(resources, castItem)
					return nil
				})
				if err != nil {
					panic(err)
				}
			} else {
				if obj.GetNamespace() == "" {
					obj.SetNamespace(core.NamespaceDefault)
				}
				resources = append(resources, &obj)
			}
		}
		fmt.Printf("visited file or dir: %q\n", path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	w := new(tabwriter.Writer)

	// Format in tab-separated columns with a tab stop of 8.
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "APIVersion\tKind\tNamespace\tName\t")
	for _, u := range resources {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", u.GetAPIVersion(), u.GetKind(), u.GetNamespace(), u.GetName())
	}
	fmt.Fprintln(w)
	w.Flush()
}
