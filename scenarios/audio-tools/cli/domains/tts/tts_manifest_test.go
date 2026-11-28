package tts

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// TestTTSManifestCoversService asserts every RPC on TTSService is bound or
// omitted in cli/manifest.json.
func TestTTSManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), ttsv1.File_audio_tools_v1_tts_tts_proto, "TTSService")
}

func TestTTSCommandsUseExpectedRendererPrimitives(t *testing.T) {
	manifest := cliapptest.ReadManifest(t)
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		architecture cliapp.CommandArchitecture
		evidence     cliapp.PrimitiveClass
	}{
		"synthesize":        {cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}, cliapp.PrimitiveAction},
		"synthesize-stream": {cliapp.CommandArchitecture{Exception: "streaming", ExceptionReason: "Consumes a long-lived server stream and writes ordered audio frames as they arrive, so output depends on stream lifecycle rather than a single response renderer."}, cliapp.PrimitiveStreaming},
		"voices":            {cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}, cliapp.PrimitiveProtoList},
		"formats":           {cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}, cliapp.PrimitiveAction},
	}
	for _, command := range group.Subcommands {
		if expected, ok := want[command.Name]; ok {
			cliapptest.AssertDeclaredArchitecture(t, command, expected.architecture)
			cliapptest.AssertPrimitiveEvidence(t, command, expected.evidence)
			delete(want, command.Name)
		}
	}
	for name := range want {
		t.Errorf("missing TTS command %q", name)
	}
}
