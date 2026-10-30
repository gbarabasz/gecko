package hc

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	privatev1 "github.com/openshift-online/gecko/platform-api/api/private/v1"
)

// LoadResourceTags reads the JSON tag list mounted from the HC configuration.
// Empty lists and null disable tags. Invalid configuration fails startup.
func LoadResourceTags(path string) ([]privatev1.GCPResourceTag, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read resource tags file %s: %w", path, err)
	}
	var tags []privatev1.GCPResourceTag
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, fmt.Errorf("parse resource tags file %s: %w", path, err)
	}
	if len(tags) > 50 {
		return nil, fmt.Errorf("resource tags must contain at most 50 entries")
	}
	seen := make(map[string]bool, len(tags))
	for _, tag := range tags {
		if tag.Key == "" || tag.Value == "" {
			return nil, fmt.Errorf("resource tags require non-empty key and value")
		}
		if seen[tag.Key] {
			return nil, fmt.Errorf("duplicate resource tag key %q", tag.Key)
		}
		seen[tag.Key] = true
	}
	return tags, nil
}

// mergeResourceTags gives platform-configured tags precedence over per-cluster
// tags by short key, including their parentID. A fresh sorted list avoids
// mutating either input and prevents spurious changes to an atomic tag list.
func mergeResourceTags(requested, configured []privatev1.GCPResourceTag) ([]privatev1.GCPResourceTag, error) {
	byKey := make(map[string]privatev1.GCPResourceTag, len(requested)+len(configured))
	for _, tag := range requested {
		byKey[tag.Key] = tag
	}
	for _, tag := range configured {
		byKey[tag.Key] = tag
	}
	if len(byKey) > 50 {
		return nil, fmt.Errorf("combined resource tags must contain at most 50 entries")
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var tags []privatev1.GCPResourceTag
	for _, key := range keys {
		tag := byKey[key]
		tags = append(tags, *tag.DeepCopy())
	}
	return tags, nil
}
