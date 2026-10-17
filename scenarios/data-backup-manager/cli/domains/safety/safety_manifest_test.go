package safety

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSafetyManifestCoversSafetyService asserts every RPC on SafetyService
// either has a manifest command binding or is documented in the manifest's
// `omitted` array. Adding a new RPC without binding/omitting it fails here —
// the anti-drift guarantee between proto and CLI.
func TestSafetyManifestCoversSafetyService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, safetyv1.File_data_backup_manager_v1_safety_safety_proto, "SafetyService")
}

func TestParseShadowMappingsAcceptsStructuredAndLegacyForms(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want [][2]string
	}{
		{
			name: "json",
			raw:  `[{"target_name":"postgres","location":"shadow-db"},{"target_name":"data","location":"/tmp/shadow"}]`,
			want: [][2]string{{"postgres", "shadow-db"}, {"data", "/tmp/shadow"}},
		},
		{
			name: "legacy",
			raw:  "postgres=shadow-db,data=/tmp/shadow",
			want: [][2]string{{"postgres", "shadow-db"}, {"data", "/tmp/shadow"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShadowMappings(tt.raw)
			if err != nil {
				t.Fatalf("parseShadowMappings() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseShadowMappings() length = %d, want %d", len(got), len(tt.want))
			}
			for i, mapping := range got {
				if actual := [2]string{mapping.GetTargetName(), mapping.GetLocation()}; !reflect.DeepEqual(actual, tt.want[i]) {
					t.Fatalf("mapping[%d] = %v, want %v", i, actual, tt.want[i])
				}
			}
		})
	}
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	// This test lives at cli/domains/safety/; the manifest lives at cli/.
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
