package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestManagedComposeManifestBindsStagedLocalBytes(t *testing.T) {
	root, source := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "source"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("actual copied bytes")
	if err := os.WriteFile(filepath.Join(root, "source/server.py"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "server.py"), []byte("later repository edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := binaryfetch.AcquisitionTarget{Compose: []binaryfetch.ComposeStep{{Role: "server", Kind: "local", Source: "server.py", Dest: "source/server.py"}}}
	if err := writeManagedComposeManifest(context.Background(), target, root, "linux-amd64", source); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".vrooli-compose-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Steps []struct {
			SourceDigest string `json:"source_digest"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(body))
	if len(got.Steps) != 1 || got.Steps[0].SourceDigest != expected {
		t.Fatalf("local input not bound to staged bytes: %s", data)
	}
	for _, kind := range []string{"missing", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			out := t.TempDir()
			dest := filepath.Join(out, "bad")
			if kind == "directory" {
				if err := os.Mkdir(dest, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "symlink" {
				if err := os.Symlink(filepath.Join(root, "source/server.py"), dest); err != nil {
					t.Skip(err)
				}
			}
			target.Compose[0].Dest = "bad"
			if err := writeManagedComposeManifest(context.Background(), target, out, "linux-amd64", source); err == nil {
				t.Fatal("published a local digest without regular staged bytes")
			}
			if _, err := os.Stat(filepath.Join(out, ".vrooli-compose-manifest.json")); !os.IsNotExist(err) {
				t.Fatalf("failed closure published manifest: %v", err)
			}
		})
	}
}
