package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBNativeManifestSchemaPreservesSourceContract(t *testing.T) {
	schema := compileRepoSchema(t, "resource.schema.json")
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		valid  bool
	}{
		{"native release", func(map[string]any) {}, true},
		{"missing adoption version", func(m map[string]any) { delete(m["managed_service"].(map[string]any), "host_tool_version") }, false},
		{"path instead of command", func(m map[string]any) { m["managed_service"].(map[string]any)["host_tool_executable"] = "../openrgb" }, false},
		{"unknown SDK kind", func(m map[string]any) { m["health_checks"].([]any)[0].(map[string]any)["type"] = "openrgb-sdk-v99" }, false},
		{"unknown service field", func(m map[string]any) { m["managed_service"].(map[string]any)["unowned_launch_fallback"] = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			tc.mutate(m)
			err := schema.Validate(m)
			if (err == nil) != tc.valid {
				t.Fatalf("schema valid=%t want %t: %v", err == nil, tc.valid, err)
			}
		})
	}
}
