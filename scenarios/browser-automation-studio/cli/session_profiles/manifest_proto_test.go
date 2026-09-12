package session_profiles

import (
	"os"
	"path/filepath"
	"testing"

	sessionprofilesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSessionProfilesManifestCoversSessionProfilesService asserts the BAS
// CLI manifest binds every RPC declared on SessionProfilesService. Part of
// the per-domain parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 9).
func TestSessionProfilesManifestCoversSessionProfilesService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, sessionprofilesv1.File_browser_automation_studio_v1_session_profiles_session_profiles_proto, "SessionProfilesService")
}
