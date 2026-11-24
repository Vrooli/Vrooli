package diagnostics

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
)

// TestDiagnosticsManifestCoversService asserts every RPC on
// DiagnosticsService is bound or omitted in cli/manifest.json.
func TestDiagnosticsManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), diagv1.File_audio_tools_v1_diagnostics_diagnostics_proto, "DiagnosticsService")
}

func TestDiagnosticsCommandsDeclareActionPrimitive(t *testing.T) {
	manifest, err := cliapp.ParseManifest(cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	group := manifest.FindGroup(GroupName)
	if group == nil {
		t.Fatal("diagnostics group is missing")
	}
	for _, name := range []string{"run", "last"} {
		found := false
		for _, command := range group.Commands {
			if command.Name == name {
				found = true
				if command.Architecture == nil || command.Architecture.Primitive != string(cliapp.PrimitiveAction) {
					t.Errorf("diagnostics %s primitive = %#v, want action", name, command.Architecture)
				}
			}
		}
		if !found {
			t.Errorf("diagnostics command %q is missing", name)
		}
	}
}
