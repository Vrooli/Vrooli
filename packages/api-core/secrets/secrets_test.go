package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestNewUserStoreResolvesCanonicalUserPath(t *testing.T) {
	home := t.TempDir()
	store, err := NewUserStore(Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	want, err := repocontract.UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	if got := store.PlaintextPath(); got != want {
		t.Fatalf("PlaintextPath = %q, want %q", got, want)
	}
	if got := store.HomeDir(); got != home {
		t.Fatalf("HomeDir = %q, want %q", got, home)
	}
}

func TestNewUserStoreUsesUserHomeResolverWhenHomeDirUnset(t *testing.T) {
	home := t.TempDir()
	store, err := NewUserStore(Config{
		UserHomeDir: func() (string, error) { return home, nil },
	})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	want, err := repocontract.UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	if got := store.PlaintextPath(); got != want {
		t.Fatalf("PlaintextPath = %q, want %q", got, want)
	}
}

func TestNewUserStoreReturnsErrorWhenHomeDirResolutionFails(t *testing.T) {
	store, err := NewUserStore(Config{
		UserHomeDir: func() (string, error) { return "", os.ErrNotExist },
	})
	if err == nil {
		t.Fatal("expected home dir resolution error")
	}
	if store != nil {
		t.Fatalf("store = %#v, want nil", store)
	}
}

func TestResolvePrefersEnvThenFile(t *testing.T) {
	home := t.TempDir()
	path, err := repocontract.UserPlaintextSecretsPath(home)
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"API_KEY\":\"file-secret\"}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	store, err := NewUserStore(Config{
		HomeDir: home,
		EnvLookup: func(key string) string {
			if key == "API_KEY" {
				return "env-secret"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}

	resolved, err := store.Resolve("API_KEY")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Source != SourceEnv || resolved.Value != "env-secret" {
		t.Fatalf("Resolve = %#v, want env-secret from env", resolved)
	}

	store, err = NewUserStore(Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	resolved, err = store.Resolve("API_KEY")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Source != SourceFile || resolved.Value != "file-secret" || resolved.SourcePath != path {
		t.Fatalf("Resolve = %#v, want file-secret from file", resolved)
	}
}

func TestResolveReturnsMissingWhenUnavailable(t *testing.T) {
	store, err := NewUserStore(Config{HomeDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewUserStore: %v", err)
	}
	resolved, err := store.Resolve("MISSING")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Source != SourceMissing || resolved.Value != "" {
		t.Fatalf("Resolve = %#v, want missing", resolved)
	}
}

func TestLoadFileDocumentPreservesMetadataAndRejectsBadValues(t *testing.T) {
	path, err := repocontract.UserPlaintextSecretsPath(t.TempDir())
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"_metadata\":{\"environment\":\"development\"},\"API_KEY\":\"secret\"}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	doc, err := LoadFileDocument(path)
	if err != nil {
		t.Fatalf("LoadFileDocument: %v", err)
	}
	if doc.Secrets["API_KEY"] != "secret" {
		t.Fatalf("API_KEY = %q, want secret", doc.Secrets["API_KEY"])
	}
	if _, ok := doc.Metadata["_metadata"]; !ok {
		t.Fatal("expected _metadata to be preserved")
	}

	if err := os.WriteFile(path, []byte("{\"API_KEY\":42}\n"), 0o600); err != nil {
		t.Fatalf("write invalid: %v", err)
	}
	_, err = LoadFileDocument(path)
	if err == nil {
		t.Fatal("expected invalid data error")
	}
}

func TestLoadFileDocumentRejectsUnsafeFiles(t *testing.T) {
	t.Run("broad perms", func(t *testing.T) {
		path, err := repocontract.UserPlaintextSecretsPath(t.TempDir())
		if err != nil {
			t.Fatalf("UserPlaintextSecretsPath: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("{\"API_KEY\":\"secret\"}\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		_, err = LoadFileDocument(path)
		if err == nil {
			t.Fatal("expected insecure permissions error")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.json")
		if err := os.WriteFile(target, []byte("{\"API_KEY\":\"secret\"}\n"), 0o600); err != nil {
			t.Fatalf("write target: %v", err)
		}
		path, err := repocontract.UserPlaintextSecretsPath(dir)
		if err != nil {
			t.Fatalf("UserPlaintextSecretsPath: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		_, err = LoadFileDocument(path)
		if err == nil {
			t.Fatal("expected symlink error")
		}
	})
}

func TestWriteFileDocumentWritesPrivateAtomicFile(t *testing.T) {
	path, err := repocontract.UserPlaintextSecretsPath(t.TempDir())
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	err = WriteFileDocument(path, Document{
		Secrets: map[string]string{"API_KEY": "secret"},
		Metadata: map[string]json.RawMessage{
			"_metadata": json.RawMessage(`{"environment":"development"}`),
		},
	})
	if err != nil {
		t.Fatalf("WriteFileDocument: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	doc, err := LoadFileDocument(path)
	if err != nil {
		t.Fatalf("LoadFileDocument: %v", err)
	}
	if doc.Secrets["API_KEY"] != "secret" {
		t.Fatalf("API_KEY = %q, want secret", doc.Secrets["API_KEY"])
	}
}

func TestNewFileStoreSaveDeleteAndUpdatePreserveMetadata(t *testing.T) {
	path, err := repocontract.UserPlaintextSecretsPath(t.TempDir())
	if err != nil {
		t.Fatalf("UserPlaintextSecretsPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{\"_metadata\":{\"managed_by\":\"test\"},\"API_KEY\":\"old\"}\n"), 0o600); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := store.SaveKey("API_KEY", "updated"); err != nil {
		t.Fatalf("SaveKey: %v", err)
	}
	if err := store.Update(func(doc *Document) error {
		doc.Metadata["_metadata"] = json.RawMessage(`{"managed_by":"test","last_updated":"now"}`)
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	deleted, err := store.DeleteKey("API_KEY")
	if err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	if !deleted {
		t.Fatal("expected API_KEY to be deleted")
	}
	doc, err := store.Document()
	if err != nil {
		t.Fatalf("Document: %v", err)
	}
	if len(doc.Secrets) != 0 {
		t.Fatalf("Secrets = %#v, want empty", doc.Secrets)
	}
	if _, ok := doc.Metadata["_metadata"]; !ok {
		t.Fatal("expected metadata to be preserved")
	}
}
