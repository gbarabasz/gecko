package hc

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openshift-online/gecko/controllers/hc/manifest"
	privatev1 "github.com/openshift-online/gecko/platform-api/api/private/v1"

	"k8s.io/utils/ptr"
)

func TestLoadResourceTags(t *testing.T) {
	for _, tc := range []struct {
		name, data string
		want       []manifest.ResourceTag
		wantErr    string
	}{
		{name: "configured", data: `[{"key":"redhat-managed","value":"true"},{"key":"environment","value":"prod","parentID":"123456789"}]`, want: []manifest.ResourceTag{{Key: "redhat-managed", Value: "true"}, {Key: "environment", Value: "prod", ParentID: "123456789"}}},
		{name: "empty", data: `[]`, want: []manifest.ResourceTag{}},
		{name: "null", data: `null`},
		{name: "invalid JSON", data: `not-json`, wantErr: "parse resource tags"},
		{name: "object instead of list", data: `{"key":"x","value":"y"}`, wantErr: "parse resource tags"},
		{name: "missing key", data: `[{"value":"true"}]`, wantErr: "non-empty key and value"},
		{name: "missing value", data: `[{"key":"redhat-managed"}]`, wantErr: "non-empty key and value"},
		{name: "null entry", data: `[null]`, wantErr: "non-empty key and value"},
		{name: "non-string value", data: `[{"key":"x","value":true}]`, wantErr: "parse resource tags"},
		{name: "duplicate", data: `[{"key":"x","value":"y"},{"key":"x","value":"z"}]`, wantErr: "duplicate resource tag key"},
		{name: "too many", data: `[` + strings.Repeat(`{"key":"x","value":"y"},`, 50) + `{"key":"z","value":"y"}]`, wantErr: "at most 50"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tags, err := LoadResourceTags(writeTemp(t, tc.data))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, tags)
		})
	}
	_, err := LoadResourceTags(filepath.Join(t.TempDir(), "missing.json"))
	require.ErrorContains(t, err, "read resource tags file")
}

func TestMergeResourceTags(t *testing.T) {
	requested := []privatev1.GCPResourceTag{
		{Key: "env", Value: "prod", ParentID: ptr.To("123456789")},
		{Key: "redhat-managed", Value: "false", ParentID: ptr.To("987654321")},
	}
	configured := []manifest.ResourceTag{{Key: "redhat-managed", Value: "true"}, {Key: "app", Value: "platform"}}
	requestedBefore := make([]privatev1.GCPResourceTag, len(requested))
	copy(requestedBefore, requested)
	configuredBefore := append([]manifest.ResourceTag(nil), configured...)
	tags, err := mergeResourceTags(requested, configured)
	require.NoError(t, err)
	require.Equal(t, []manifest.ResourceTag{{Key: "app", Value: "platform"}, {Key: "env", Value: "prod", ParentID: "123456789"}, {Key: "redhat-managed", Value: "true"}}, tags)
	require.Equal(t, requestedBefore, requested)
	require.Equal(t, configuredBefore, configured)
	tags[0].Value = "changed"
	require.Equal(t, configuredBefore, configured)
	require.Equal(t, requestedBefore, requested)
	empty, err := mergeResourceTags(nil, nil)
	require.NoError(t, err)
	require.Empty(t, empty)
	// Each source is valid separately; enforce the limit on the merged union.
	many := make([]privatev1.GCPResourceTag, 50)
	for i := range many {
		many[i] = privatev1.GCPResourceTag{Key: fmt.Sprintf("tag%d", i), Value: "v"}
	}
	_, err = mergeResourceTags(many, []manifest.ResourceTag{{Key: "extra", Value: "v"}})
	require.ErrorContains(t, err, "combined resource tags must contain at most 50")
	merged, err := mergeResourceTags(many, []manifest.ResourceTag{{Key: "tag0", Value: "override"}})
	require.NoError(t, err)
	require.Len(t, merged, 50)
}
