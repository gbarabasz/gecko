package hc

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openshift-online/gecko/controllers/hc/manifest"
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
