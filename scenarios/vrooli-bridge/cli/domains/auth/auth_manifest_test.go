package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestAuthManifestCoversIdentityLogin(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	group := manifest.FindGroup(GroupName)
	if group == nil || len(group.Commands) != 1 || group.Commands[0].Binding.BindingKey() != "IdentityService.Login" {
		t.Fatal("auth login must bind IdentityService.Login")
	}
	for _, omitted := range manifest.Omitted {
		if omitted.Service == "IdentityService" && omitted.Method == "Login" {
			t.Fatal("IdentityService.Login must not be omitted now that auth login exists")
		}
	}
}
