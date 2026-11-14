package domains

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
)

// TestSessionManifestCoversService asserts every RPC on SessionService is
// documented in cli/manifest.json. SessionService has no CLI domain package
// (it is the real-time UI/WebSocket bridge), so all of its methods live in
// the manifest's `omitted` list — this test guards that none silently drift
// out of coverage. Placed in the aggregator package because there is no
// domains/session command package to host it.
func TestSessionManifestCoversService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, sessionv1.File_audio_tools_v1_session_session_proto, "SessionService")
}
