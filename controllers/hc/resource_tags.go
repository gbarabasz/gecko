package hc

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/openshift-online/gecko/controllers/hc/manifest"
	privatev1 "github.com/openshift-online/gecko/platform-api/api/private/v1"
)

// LoadResourceTags reads the JSON tag list mounted from the HC configuration.
// Empty lists and null disable tags. Invalid configuration fails startup.
func LoadResourceTags(path string) ([]manifest.ResourceTag, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read resource tags file %s: %w", path, err)
	}
	var tags []manifest.ResourceTag
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
func mergeResourceTags(requested []privatev1.GCPResourceTag, configured []manifest.ResourceTag) ([]manifest.ResourceTag, error) {
	byKey := make(map[string]manifest.ResourceTag, len(requested)+len(configured))
	for _, tag := range requested {
		value := manifest.ResourceTag{Key: tag.Key, Value: tag.Value}
		if tag.ParentID != nil {
			value.ParentID = *tag.ParentID
		}
		byKey[tag.Key] = value
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
	var tags []manifest.ResourceTag
	for _, key := range keys {
		tags = append(tags, byKey[key])
	}
	return tags, nil
}
