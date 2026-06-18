package auth

import (
	"os"
	"path/filepath"
	"testing"

	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAuthManifestCoversAccountsService asserts every RPC on AccountsService
// has a manifest command binding (or is documented as omitted). Adding an RPC
// without binding it here fails.
func TestAuthManifestCoversAccountsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, accountsv1.File_scenario_authenticator_v1_accounts_accounts_proto, "AccountsService")
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
