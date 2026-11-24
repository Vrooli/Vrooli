package summarize

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

// TestSummarizeManifestCoversService asserts every RPC on SummarizeService
// is bound or omitted in cli/manifest.json.
func TestSummarizeManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, cliapptest.ReadManifest(t), summv1.File_audio_tools_v1_summarize_summarize_proto, "SummarizeService")
}

func TestSummarizeTextUsesActionPrimitive(t *testing.T) {
	group, err := Register(nil, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(group.Subcommands) != 1 || group.Subcommands[0].Name != "text" {
		t.Fatalf("unexpected summarize commands: %+v", group.Subcommands)
	}
	command := group.Subcommands[0]
	cliapptest.AssertDeclaredArchitecture(t, command, cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction})
	cliapptest.AssertPrimitiveEvidence(t, command, cliapp.PrimitiveAction)
}
