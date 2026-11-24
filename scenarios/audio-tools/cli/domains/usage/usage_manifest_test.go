package usage

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
)

// TestUsageManifestCoversService asserts every RPC on UsageService is bound
// or omitted in cli/manifest.json.
func TestUsageManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), usagev1.File_audio_tools_v1_usage_usage_proto, "UsageService")
}

func TestUsageCommandsDeclareActionPrimitive(t *testing.T) {
	manifest, err := cliapp.ParseManifest(cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	group := manifest.FindGroup(GroupName)
	if group == nil {
		t.Fatal("usage group is missing")
	}
	for _, name := range []string{"list", "summary"} {
		found := false
		for _, command := range group.Commands {
			if command.Name == name {
				found = true
				if command.Architecture == nil || command.Architecture.Primitive != string(cliapp.PrimitiveAction) {
					t.Errorf("usage %s primitive = %#v, want action", name, command.Architecture)
				}
			}
		}
		if !found {
			t.Errorf("usage command %q is missing", name)
		}
	}
}
