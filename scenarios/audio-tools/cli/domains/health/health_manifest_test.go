package health

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
)

// TestHealthManifestCoversService asserts every RPC on HealthStatusService
// is bound or omitted in cli/manifest.json.
func TestHealthManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), hsv1.File_audio_tools_v1_health_status_health_status_proto, "HealthStatusService")
}

func TestHealthCommandsDeclarePrimitiveArchitecture(t *testing.T) {
	manifest, err := cliapp.ParseManifest(cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	group := manifest.FindGroup(GroupName)
	if group == nil {
		t.Fatal("health group is missing")
	}
	for _, command := range group.Commands {
		switch command.Name {
		case "show":
			if command.Architecture == nil || command.Architecture.Primitive != string(cliapp.PrimitiveAction) {
				t.Errorf("health show architecture = %#v, want action", command.Architecture)
			}
		case "watch":
			if command.Architecture == nil || command.Architecture.Exception == nil || command.Architecture.Exception.Class != string(cliapp.ExceptionStreaming) {
				t.Errorf("health watch architecture = %#v, want streaming exception", command.Architecture)
			}
		}
	}
}
