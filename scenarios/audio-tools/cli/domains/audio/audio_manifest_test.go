package audio

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
)

// TestAudioManifestCoversService asserts every RPC on AudioProcessingService
// is either bound by a manifest command or documented in the manifest's
// `omitted` list. Adding a new RPC without binding or omitting it fails here,
// keeping cli/manifest.json the single source of truth for the CLI surface.
func TestAudioManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), audiov1.File_audio_tools_v1_audio_audio_proto, "AudioProcessingService")
}

func TestAudioCommandsUseDeclaredRendererPrimitives(t *testing.T) {
	group, err := Register(nil, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]cliapp.PrimitiveClass{
		"transcode": cliapp.PrimitiveAction, "trim": cliapp.PrimitiveAction,
		"merge": cliapp.PrimitiveAction, "split": cliapp.PrimitiveProtoMutation,
		"fade": cliapp.PrimitiveAction, "volume": cliapp.PrimitiveAction,
		"normalize": cliapp.PrimitiveAction, "extract-metadata": cliapp.PrimitiveAction,
	}
	for _, command := range group.Subcommands {
		if expected, ok := want[command.Name]; ok {
			cliapptest.AssertDeclaredArchitecture(t, command, cliapp.CommandArchitecture{Primitive: expected})
			cliapptest.AssertPrimitiveEvidence(t, command, expected)
			delete(want, command.Name)
		}
	}
	for name := range want {
		t.Errorf("missing audio command %q", name)
	}
}
