package hc

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openshift-online/gecko/controllers/hc/manifest"
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
