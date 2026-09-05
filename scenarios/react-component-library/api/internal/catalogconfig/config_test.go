package catalogconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeclaredMaturityFloorReadsCanonicalKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"adoptionMaturityFloor":"scaffolded"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := DeclaredMaturityFloor(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "scaffolded" {
		t.Fatalf("floor = %q, want scaffolded", got)
	}
}

func TestDeclaredMaturityFloorRejectsMissingOrEmptyKey(t *testing.T) {
	for _, body := range []string{`{}`, `{"adoptionMaturityFloor":"  "}`} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := DeclaredMaturityFloor(path); err == nil {
			t.Fatalf("DeclaredMaturityFloor(%s) succeeded, want error", body)
		}
	}
}
