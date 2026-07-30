package secrets

import (
	"os"
	"path/filepath"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestMigrateLegacyFileRequiresExplicitDeletionConsent(t *testing.T) {
	authority, err := credentialauthority.NewAuthority(&nativeTestStore{})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewNativeManagerWithAuthority(&manifest.Manifest{App: manifest.App{Name: "Migration App"}, Secrets: []manifest.Secret{{ID: "API_KEY"}}}, authority)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "secrets.json")
	if err := os.WriteFile(path, []byte(`{"secrets":{"API_KEY":"desktop-test-value","IGNORED":"x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := manager.MigrateLegacyFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Imported) != 1 || report.Imported[0] != "API_KEY" || report.SourceDeleted {
		t.Fatalf("unexpected migration report: %+v", report)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("source was removed without consent: %v", err)
	}
	values, err := manager.Load()
	if err != nil || values["API_KEY"] != "desktop-test-value" {
		t.Fatalf("native import failed: values=%v err=%v", values, err)
	}
}
