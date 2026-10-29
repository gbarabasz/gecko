package transport

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreserveCreateOnlyFields(t *testing.T) {
	for _, tc := range []struct{ name, original, desired, want string }{
		{"keep tags on change", `{"spec":{"platform":{"gcp":{"resourceTags":[{"key":"old","value":"true"}]}}}}`, `{"spec":{"release":"new","platform":{"gcp":{"resourceTags":[{"key":"new","value":"true"}]}}}}`, `{"spec":{"release":"new","platform":{"gcp":{"resourceTags":[{"key":"old","value":"true"}]}}}}`},
		{"keep absence", `{"spec":{"platform":{"gcp":{}}}}`, `{"spec":{"platform":{"gcp":{"resourceTags":[{"key":"new","value":"true"}]}}}}`, `{"spec":{"platform":{"gcp":{}}}}`},
		{"keep tags on config removal", `{"spec":{"platform":{"gcp":{"resourceTags":[{"key":"old","value":"true","parentID":"123"}]}}}}`, `{"spec":{"platform":{"gcp":{}}}}`, `{"spec":{"platform":{"gcp":{"resourceTags":[{"key":"old","value":"true","parentID":"123"}]}}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var original, desired map[string]any
			require.NoError(t, json.Unmarshal([]byte(tc.original), &original))
			require.NoError(t, json.Unmarshal([]byte(tc.desired), &desired))
			require.NoError(t, PreserveCreateOnlyFields(desired, original, [][]string{{"spec", "platform", "gcp", "resourceTags"}}))
			raw, err := json.Marshal(desired)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(raw))
			raw, err = json.Marshal(original)
			require.NoError(t, err)
			require.JSONEq(t, tc.original, string(raw), "original must not change")
		})
	}
}

func TestCreateOnlyPaths(t *testing.T) {
	opts := []ApplyOptions{{CreateOnlyFields: map[string][][]string{"hc": {{"spec", "tags"}}}}, {CreateOnlyFields: map[string][][]string{"hc": {{"spec", "id"}}}}}
	require.Equal(t, [][]string{{"spec", "tags"}, {"spec", "id"}}, CreateOnlyPaths("hc", opts))
	require.Empty(t, CreateOnlyPaths("other", opts))
}

func TestPreserveCreateOnlyFields_FailsOnMalformedContent(t *testing.T) {
	require.ErrorContains(t, PreserveCreateOnlyFields(map[string]any{}, map[string]any{}, [][]string{nil}), "path must not be empty")
	require.ErrorContains(t, PreserveCreateOnlyFields(map[string]any{}, map[string]any{"spec": "invalid"}, [][]string{{"spec", "tags"}}), "read creation-only field")
	require.ErrorContains(t, PreserveCreateOnlyFields(map[string]any{"spec": "invalid"}, map[string]any{"spec": map[string]any{"tags": []any{"tag"}}}, [][]string{{"spec", "tags"}}), "preserve creation-only field")
}
