package hc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func renderHCChart(t *testing.T, labels, tags string) ([]unstructured.Unstructured, error) {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm not installed; skipping chart render test")
	}
	values, err := json.Marshal(map[string]any{"image": map[string]any{"tag": "test"}, "customerLabels": labels, "resourceTags": tags})
	require.NoError(t, err)
	cmd := exec.CommandContext(t.Context(), "helm", "template", "test", "../../../helm/charts/gecko-hc", "-f", "-")
	cmd.Stdin = strings.NewReader(string(values))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("helm template: %w: %s", err, out)
	}
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(out)), 4096)
	var objects []unstructured.Unstructured
	for {
		var obj unstructured.Unstructured
		err := decoder.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if obj.Object != nil {
			objects = append(objects, obj)
		}
	}
	return objects, nil
}

func TestHCChartResourceConfig(t *testing.T) {
	labels := `{"partner":"openshift"}`
	tags := `[{"key":"redhat-managed","value":"true","parentID":"123456789"}]`
	for _, tc := range []struct {
		name, labels, tags   string
		wantLabels, wantTags bool
	}{
		{"disabled", "{}", "[]", false, false},
		{"labels only", labels, "[]", true, false},
		{"tags only", "{}", tags, false, true},
		{"both", labels, tags, true, true},
		{"whitespace empty", " { } ", " [ ] ", false, false},
		{"null", "null", "null", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			objects, err := renderHCChart(t, tc.labels, tc.tags)
			require.NoError(t, err)
			var configMaps []unstructured.Unstructured
			var deployment unstructured.Unstructured
			for _, obj := range objects {
				switch obj.GetKind() {
				case "ConfigMap":
					configMaps = append(configMaps, obj)
				case "Deployment":
					deployment = obj
				}
			}
			pod, found, err := unstructured.NestedMap(deployment.Object, "spec", "template", "spec")
			require.NoError(t, err)
			require.True(t, found)
			containers := pod["containers"].([]any)
			container := containers[0].(map[string]any)
			args := container["args"].([]any)
			labelsArg := "--customer-labels-file=/etc/config/resources/customer-labels.json"
			tagsArg := "--resource-tags-file=/etc/config/resources/resource-tags.json"
			if tc.wantLabels {
				require.Contains(t, args, labelsArg)
			} else {
				require.NotContains(t, args, labelsArg)
			}
			if tc.wantTags {
				require.Contains(t, args, tagsArg)
			} else {
				require.NotContains(t, args, tagsArg)
			}
			checksum, _, err := unstructured.NestedString(deployment.Object, "spec", "template", "metadata", "annotations", "checksum/config")
			require.NoError(t, err)
			if !tc.wantLabels && !tc.wantTags {
				require.Empty(t, configMaps)
				require.NotContains(t, pod, "volumes")
				require.NotContains(t, container, "volumeMounts")
				require.Empty(t, checksum)
				return
			}
			require.Len(t, configMaps, 1)
			require.NotEmpty(t, checksum)
			data, _, err := unstructured.NestedStringMap(configMaps[0].Object, "data")
			require.NoError(t, err)
			if tc.wantLabels {
				require.JSONEq(t, tc.labels, data["customer-labels.json"])
			} else {
				require.NotContains(t, data, "customer-labels.json")
			}
			if tc.wantTags {
				require.JSONEq(t, tc.tags, data["resource-tags.json"])
			} else {
				require.NotContains(t, data, "resource-tags.json")
			}
			volumes := pod["volumes"].([]any)
			require.Len(t, volumes, 1)
			volume := volumes[0].(map[string]any)
			require.Equal(t, configMaps[0].GetName(), volume["configMap"].(map[string]any)["name"])
			mounts := container["volumeMounts"].([]any)
			require.Len(t, mounts, 1)
			mount := mounts[0].(map[string]any)
			require.Equal(t, volume["name"], mount["name"])
			require.Equal(t, "/etc/config/resources", mount["mountPath"])
			require.Equal(t, true, mount["readOnly"])
		})
	}
	t.Run("bad JSON", func(t *testing.T) { _, err := renderHCChart(t, "{}", "invalid"); require.Error(t, err) })
	t.Run("wrong shape", func(t *testing.T) { _, err := renderHCChart(t, "{}", `{"key":"x"}`); require.Error(t, err) })
	t.Run("tags trigger rollout", func(t *testing.T) {
		first, err := renderHCChart(t, "{}", tags)
		require.NoError(t, err)
		second, err := renderHCChart(t, "{}", `[{"key":"other","value":"true"}]`)
		require.NoError(t, err)
		checksum := func(objects []unstructured.Unstructured) string {
			for _, obj := range objects {
				if obj.GetKind() == "Deployment" {
					value, _, err := unstructured.NestedString(obj.Object, "spec", "template", "metadata", "annotations", "checksum/config")
					require.NoError(t, err)
					return value
				}
			}
			t.Fatal("no Deployment")
			return ""
		}
		require.NotEqual(t, checksum(first), checksum(second))
	})
}

func TestHCChartRejectsWrongEmptyJSONShapes(t *testing.T) {
	for _, tc := range []struct{ labels, tags string }{{"{}", "{}"}, {"[]", "[]"}, {"{}", `""`}} {
		_, err := renderHCChart(t, tc.labels, tc.tags)
		require.Error(t, err)
	}
}
