package fetch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type acquireDownloader func(context.Context, string, string, func(int64, int64)) error

func (f acquireDownloader) Download(ctx context.Context, url, path string, progress func(int64, int64)) error {
	return f(ctx, url, path, progress)
}

func TestAcquisitionPublishesOnlyVerifiedReplacement(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "model")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(dest, "old.bin")
	if err := os.WriteFile(old, []byte("preserve installed artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte{0x40}, 2048)
	acquirer := Acquirer{Root: root, Download: acquireDownloader(func(_ context.Context, _, path string, _ func(int64, int64)) error {
		if filepath.Dir(path) == dest {
			t.Fatal("download overwrote published model before verification")
		}
		return os.WriteFile(path, payload, 0o644)
	})}
	source := Source{Assets: []Asset{{URL: "https://example.test/model.bin", Filename: "model.bin", SHA256: strings.Repeat("0", 64)}}}
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error = %v", err)
	}
	if got, err := os.ReadFile(old); err != nil || string(got) != "preserve installed artifact" {
		t.Fatalf("failed download destroyed previous model: %q %v", got, err)
	}
	source.Assets[0].SHA256 = ""
	result, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil)
	if err != nil || result.Path != dest || result.SizeBytes != int64(len(payload)) || result.Checksum == "" {
		t.Fatalf("result = %+v, %v", result, err)
	}
	if _, err := os.Stat(old); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("obsolete artifact survived replacement: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "model" {
		t.Fatalf("staging artifacts retained: %v, %v", entries, err)
	}
}

func TestCatalogManifestChecksumOverridesOldLocalPin(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "model")
	oldPayload := bytes.Repeat([]byte{0x40}, 2048)
	newPayload := bytes.Repeat([]byte{0x41}, 2048)
	oldSum, newSum := hashBytes(oldPayload), hashBytes(newPayload)
	acquirer := Acquirer{Root: root, Download: acquireDownloader(func(_ context.Context, _, path string, _ func(int64, int64)) error {
		return os.WriteFile(path, newPayload, 0o644)
	})}
	source := Source{Assets: []Asset{{URL: "https://example.test/model.bin", Filename: "model.bin", SHA256: newSum}}}
	if _, err := acquirer.Acquire(context.Background(), source, dest, oldSum, 1, nil); err != nil {
		t.Fatalf("current catalog checksum should supersede prior pin: %v", err)
	}
	if err := acquirer.ValidateInstalled(source, dest, oldSum); err != nil {
		t.Fatalf("validate current manifest artifact: %v", err)
	}
}

func hashBytes(payload []byte) string {
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum[:])
}

func TestAcquisitionRefusesEscapingAssetAndLeavesLocalReferenceUntouched(t *testing.T) {
	root := t.TempDir()
	acquirer := Acquirer{Root: root, Download: acquireDownloader(func(context.Context, string, string, func(int64, int64)) error {
		t.Fatal("unsafe download began")
		return nil
	})}
	for _, name := range []string{"../escape.bin", "/escape.bin", `..\escape.bin`} {
		_, err := acquirer.Acquire(context.Background(), Source{Assets: []Asset{{URL: "https://example.test/x", Filename: name}}}, filepath.Join(root, "model"), "", 1, nil)
		if err == nil {
			t.Fatalf("accepted unsafe name %q", name)
		}
	}
	path := filepath.Join(root, "local.bin")
	if err := os.WriteFile(path, []byte("local model"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := acquirer.Acquire(context.Background(), Source{LocalPath: path}, filepath.Join(root, "model"), "", 1, nil)
	if err != nil || result.Path != path || result.SizeBytes != 11 || result.Checksum == "" {
		t.Fatalf("local result = %+v, %v", result, err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "local model" {
		t.Fatalf("local reference changed: %q %v", got, err)
	}
}
