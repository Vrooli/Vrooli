package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestAuthManifestCoversIdentityLifecycle(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	group := manifest.FindGroup(GroupName)
	if group == nil || len(group.Commands) != 2 {
		t.Fatal("auth must expose login and refresh")
	}
	bindings := map[string]bool{}
	for _, command := range group.Commands {
		bindings[command.Binding.BindingKey()] = true
	}
	if !bindings["IdentityService.Login"] || !bindings["IdentityService.Refresh"] {
		t.Fatal("auth login and refresh must bind their IdentityService RPCs")
	}
	for _, omitted := range manifest.Omitted {
		if omitted.Service == "IdentityService" && omitted.Method == "Login" {
			t.Fatal("IdentityService.Login must not be omitted now that auth login exists")
		}
	}
}
