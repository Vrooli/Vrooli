package summarize

import (
	"testing"

	testutil "audio-tools/cli/internal/testutil"
	"github.com/vrooli/cli-core/cliapp"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

// TestSummarizeManifestCoversService asserts every RPC on SummarizeService
// is bound or omitted in cli/manifest.json.
func TestSummarizeManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), summv1.File_audio_tools_v1_summarize_summarize_proto, "SummarizeService")
}
