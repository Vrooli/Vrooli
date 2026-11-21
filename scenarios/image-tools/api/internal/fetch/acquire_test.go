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

type repoFetcherFunc func(context.Context, RepoSpec, string, func(int64, int64)) error

func (f repoFetcherFunc) Snapshot(ctx context.Context, repo RepoSpec, dir string, progress func(int64, int64)) error {
	return f(ctx, repo, dir, progress)
}

func TestCombinedRepoAndAuxiliaryAssetsPublishOnePinnedTree(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "adapter")
	license := []byte(strings.Repeat("CreativeML Open RAIL-M pinned license text\n", 5))
	licenseHash := hashBytes(license)
	acquirer := Acquirer{
		Root: root,
		RepoFetch: repoFetcherFunc(func(_ context.Context, repo RepoSpec, dir string, _ func(int64, int64)) error {
			if repo.Revision != "immutable-commit" || len(repo.AllowPatterns) != 2 {
				t.Fatalf("repo spec = %+v", repo)
			}
			return os.WriteFile(filepath.Join(dir, "diffusion_pytorch_model.safetensors"), append(make([]byte, 8), '{', '}', '\n'), 0o600)
		}),
		Download: acquireDownloader(func(_ context.Context, url, path string, _ func(int64, int64)) error {
			switch url {
			case "https://models.example/license.txt":
				return os.WriteFile(path, license, 0o600)
			case "https://models.example/pose.task":
				return os.WriteFile(path, bytes.Repeat([]byte{0x2a}, 2048), 0o600)
			default:
				t.Fatalf("unexpected asset URL %q", url)
				return nil
			}
		}),
	}
	source := Source{
		Repo: RepoSpec{RepoID: "org/model", Revision: "immutable-commit", AllowPatterns: []string{"config.json", "diffusion_pytorch_model.safetensors"}},
		Assets: []Asset{
			{URL: "https://models.example/license.txt", Filename: "LICENSE.OpenRAIL-M.txt", Kind: ArtifactConfig, SHA256: licenseHash},
			{URL: "https://models.example/pose.task", Filename: "pose/pose.task", MinBytes: 1024},
		},
	}
	result, err := acquirer.Acquire(context.Background(), source, dest, "", 1400, nil)
	if err != nil {
		t.Fatalf("combined acquisition: %v", err)
	}
	if result.Path != dest || result.Checksum == "" || result.SizeBytes == 0 {
		t.Fatalf("combined receipt = %+v", result)
	}
	if err := acquirer.ValidateInstalled(source, dest, result.Checksum); err != nil {
		t.Fatalf("validate combined installed tree: %v", err)
	}
	for _, name := range []string{"diffusion_pytorch_model.safetensors", "LICENSE.OpenRAIL-M.txt", "pose/pose.task"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Fatalf("combined install omitted %s: %v", name, err)
		}
	}
}

func TestRepoFileSHA256PinsAreCheckedBeforePublicationAndOnReadback(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "adapter")
	weight := []byte("pinned-model-weights")
	config := []byte("{\"conditioning\":\"openpose\"}\n")
	fileSHA := map[string]string{
		"diffusion_pytorch_model.safetensors": hashBytes(weight),
		"config.json":                         hashBytes(config),
	}
	fetcher := repoFetcherFunc(func(_ context.Context, _ RepoSpec, dir string, _ func(int64, int64)) error {
		if err := os.WriteFile(filepath.Join(dir, "diffusion_pytorch_model.safetensors"), weight, 0o600); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "config.json"), config, 0o600)
	})
	source := Source{Repo: RepoSpec{RepoID: "org/model", Revision: "immutable", FileSHA256: fileSHA}}
	acquirer := Acquirer{Root: root, RepoFetch: fetcher}
	result, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil)
	if err != nil {
		t.Fatalf("acquire pinned repo files: %v", err)
	}
	if err := acquirer.ValidateInstalled(source, dest, result.Checksum); err != nil {
		t.Fatalf("validate pinned repo files: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "config.json"), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := acquirer.ValidateInstalled(source, dest, result.Checksum); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("tampered pinned config accepted: %v", err)
	}

	fileSHA["diffusion_pytorch_model.safetensors"] = strings.Repeat("0", 64)
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("mismatched repo file pin accepted: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dest, "diffusion_pytorch_model.safetensors")); err != nil || string(got) != string(weight) {
		t.Fatalf("failed replacement damaged installed model: %q %v", got, err)
	}
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

func TestAcquisitionMarksPreviousArtifactCleanupFailureUnverified(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "model")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "old.bin"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	cleanupFailure := errors.New("simulated cleanup failure")
	var previousPath string
	acquirer := Acquirer{
		Root: root,
		Download: acquireDownloader(func(_ context.Context, _, path string, _ func(int64, int64)) error {
			return os.WriteFile(path, bytes.Repeat([]byte("n"), 2048), 0o644)
		}),
		removeAll: func(path string) error {
			if strings.HasSuffix(path, ".previous") {
				previousPath = path
				return cleanupFailure
			}
			return os.RemoveAll(path)
		},
	}
	result, err := acquirer.Acquire(context.Background(), Source{Assets: []Asset{{URL: "https://example.test/model.bin", Filename: "model.bin"}}}, dest, "", 1, nil)
	if !errors.Is(err, ErrCleanupUnverified) || !errors.Is(err, cleanupFailure) {
		t.Fatalf("previous cleanup error=%v, want ErrCleanupUnverified wrapping injected failure", err)
	}
	if result.Path != dest {
		t.Fatalf("verified replacement was not published: %+v", result)
	}
	if _, err := os.Stat(previousPath); err != nil {
		t.Fatalf("old artifact missing despite unverified cleanup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "model.bin")); err != nil {
		t.Fatalf("new verified artifact missing: %v", err)
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
