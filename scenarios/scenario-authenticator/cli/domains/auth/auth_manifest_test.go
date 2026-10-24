package auth

import (
	"encoding/json"
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

func TestAuthManifestNeverDeclaresPasswordFlag(t *testing.T) {
	var manifest struct {
		Groups []struct {
			Commands []struct {
				Flags []struct {
					Name string `json:"name"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(readManifest(t), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			for _, flag := range command.Flags {
				if flag.Name == "password" {
					t.Fatalf("command declares insecure --password flag")
				}
			}
		}
	}
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
