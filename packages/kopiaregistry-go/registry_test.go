package kopiaregistry

import (
	"os"
	"strings"
	"testing"
)

func TestPassphraseIdentity(t *testing.T) {
	identity, err := PassphraseIdentity("nightly")
	if err != nil {
		t.Fatal(err)
	}
	if identity != "vrooli/kopia/nightly" {
		t.Fatalf("identity = %q", identity)
	}
}

func TestPassphraseIdentityRejectsAmbiguousNames(t *testing.T) {
	for _, name := range []string{"", "nightly/other", `nightly\\other`, ".", ".."} {
		if _, err := PassphraseIdentity(name); err == nil {
			t.Errorf("PassphraseIdentity(%q) accepted an ambiguous name", name)
		}
	}
}

func TestRegistryRoundTripContainsNoPassphrase(t *testing.T) {
	path := t.TempDir() + "/registry.json"
	registry := New(path)
	if err := registry.Upsert(Entry{Name: "nightly", Backend: BackendFilesystem, ConfigFile: "/cfg"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "repository-passphrase") {
		t.Fatal("registry contains the credential field")
	}
}
