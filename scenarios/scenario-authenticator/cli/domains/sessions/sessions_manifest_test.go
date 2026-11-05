package sessions

import (
	"os"
	"path/filepath"
	"testing"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSessionsManifestCoversSessionsService asserts every RPC on
// SessionsService has a manifest command binding (or is documented as omitted).
func TestSessionsManifestCoversSessionsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, sessionsv1.File_scenario_authenticator_v1_sessions_sessions_proto, "SessionsService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
