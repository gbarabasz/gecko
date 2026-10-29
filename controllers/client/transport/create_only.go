package transport

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ApplyOptions controls field ownership when persisting desired resources.
type ApplyOptions struct {
	// CreateOnlyFields maps ResourceKey identities to JSON field paths. The first
	// persisted desire owns these fields (including their absence). Subsequent
	// applies preserve them, even across restarts and configuration changes.
	CreateOnlyFields map[string][][]string
}

// CreateOnlyPaths collects the creation-only field paths for one resource.
func CreateOnlyPaths(key string, opts []ApplyOptions) [][]string {
	var paths [][]string
	for _, opt := range opts {
		paths = append(paths, opt.CreateOnlyFields[key]...)
	}
	return paths
}

// PreserveCreateOnlyFields copies creation-only fields from the original desired
// resource, including preserving omission. The caller must atomically read and
// write the persisted desire to prevent concurrent applies from changing them.
func PreserveCreateOnlyFields(desired, original map[string]any, paths [][]string) error {
	for _, path := range paths {
		if len(path) == 0 {
			return fmt.Errorf("creation-only field path must not be empty")
		}
		value, found, err := unstructured.NestedFieldCopy(original, path...)
		if err != nil {
			return fmt.Errorf("read creation-only field %v: %w", path, err)
		}
		if !found {
			unstructured.RemoveNestedField(desired, path...)
			continue
		}
		if err := unstructured.SetNestedField(desired, value, path...); err != nil {
			return fmt.Errorf("preserve creation-only field %v: %w", path, err)
		}
	}
	return nil
}
