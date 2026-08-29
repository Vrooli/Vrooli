// Package integration holds the real-binary, real-backend round-trip tests for
// resource-kopia (plan §9.2 Tests A–D). They are gated behind KOPIA_INTEGRATION
// and the presence of the kopia binary, so `go test ./...` compiles and skips
// them where the engine/backends are unavailable, and CI runs them with:
//
//	KOPIA_INTEGRATION=1 go test ./... -run Integration -timeout 600s
package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/env"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/maintenance"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repo"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/snapshot"

	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"

	credentialmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials/mocks"
)

// rig is a fully-wired set of services backed by the real kopia binary plus an
// in-memory credential authority and temp storage roots.
type rig struct {
	repo  repo.Service
	snap  snapshot.Service
	pol   policy.Service
	maint maintenance.Service
	store *credentialmocks.FakeStore
	env   env.Runtime
	out   *bytes.Buffer
}

func requireKopia(t *testing.T) {
	t.Helper()
	if os.Getenv("KOPIA_INTEGRATION") == "" {
		t.Skip("set KOPIA_INTEGRATION=1 to run kopia integration tests")
	}
	if _, err := exec.LookPath("kopia"); err != nil {
		t.Skip("kopia binary not found on PATH")
	}
}

func newRig(t *testing.T) *rig {
	t.Helper()
	base := t.TempDir()
	rt := env.Runtime{
		ConfigRoot:   filepath.Join(base, "config"),
		StateRoot:    filepath.Join(base, "state"),
		CacheRoot:    filepath.Join(base, "cache"),
		LogsRoot:     filepath.Join(base, "logs"),
		ReposDir:     filepath.Join(base, "config", "repos"),
		RegistryFile: filepath.Join(base, "state", "registry.json"),
	}
	if err := rt.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}
	runner := kexec.NewBinaryRunner("kopia")
	v := credentialmocks.NewFakeStore()
	reg := registry.New(rt.RegistryFile)
	resolver := repoctx.Resolver{Registry: reg, Credentials: v}
	out := &bytes.Buffer{}
	return &rig{
		repo:  repo.Service{Runner: runner, Credentials: v, Registry: reg, Resolver: resolver, Env: rt, Out: out},
		snap:  snapshot.Service{Runner: runner, Resolver: resolver, Out: out},
		pol:   policy.Service{Runner: runner, Resolver: resolver, Out: out},
		maint: maintenance.Service{Runner: runner, Resolver: resolver, Out: out},
		store: v,
		env:   rt,
		out:   out,
	}
}

func (r *rig) reset() { r.out.Reset() }

// writeFixture writes a known tree and returns a path->sha256 manifest.
func writeFixture(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{
		"a.txt":          "hello kopia",
		"sub/b.txt":      "nested content",
		"sub/deep/c.bin": "\x00\x01\x02binary\xff",
	}
	manifest := map[string]string{}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(content))
		manifest[rel] = hex.EncodeToString(sum[:])
	}
	return manifest
}

// hashTree returns a relpath->sha256 manifest of every regular file under root.
func hashTree(t *testing.T, root string) map[string]string {
	t.Helper()
	manifest := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		sum := sha256.Sum256(data)
		manifest[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func assertManifestsEqual(t *testing.T, want, got map[string]string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("file count mismatch: want %d got %d (%v vs %v)", len(want), len(got), want, got)
	}
	for rel, sum := range want {
		if got[rel] != sum {
			t.Fatalf("checksum mismatch for %q: want %s got %s", rel, sum, got[rel])
		}
	}
}

// latestSnapshotID lists snapshots as JSON and returns the newest id.
func (r *rig) latestSnapshotID(t *testing.T, repoName, path string) string {
	t.Helper()
	r.reset()
	if err := r.snap.List(context.Background(), []string{"--repo", repoName, "--path", path, "--json"}); err != nil {
		t.Fatalf("snapshot list: %v", err)
	}
	var manifests []struct {
		ID        string `json:"id"`
		StartTime string `json:"startTime"`
	}
	if err := json.Unmarshal(r.out.Bytes(), &manifests); err != nil {
		t.Fatalf("parse snapshot list json: %v\noutput: %s", err, r.out.String())
	}
	if len(manifests) == 0 {
		t.Fatal("no snapshots listed")
	}
	return manifests[len(manifests)-1].ID
}

// TestIntegrationFilesystemRoundTrip is plan §9.2 Test A.
func TestIntegrationFilesystemRoundTrip(t *testing.T) {
	requireKopia(t)
	r := newRig(t)
	ctx := context.Background()
	repoDir := filepath.Join(t.TempDir(), "fsrepo")
	fixtureDir := filepath.Join(t.TempDir(), "fixture")
	want := writeFixture(t, fixtureDir)

	if err := r.repo.Create(ctx, []string{"--name", "local", "--backend", "filesystem", "--path", repoDir}); err != nil {
		t.Fatalf("repo create: %v", err)
	}
	if err := r.snap.Create(ctx, []string{"--repo", "local", "--path", fixtureDir, "--json"}); err != nil {
		t.Fatalf("snapshot create: %v", err)
	}
	id := r.latestSnapshotID(t, "local", fixtureDir)

	restoreDir := filepath.Join(t.TempDir(), "restore")
	if err := r.snap.Restore(ctx, []string{"--repo", "local", "--snapshot", id, "--target", restoreDir}); err != nil {
		t.Fatalf("snapshot restore: %v", err)
	}
	assertManifestsEqual(t, want, hashTree(t, restoreDir))

	r.reset()
	if err := r.snap.Verify(ctx, []string{"--repo", "local", "--verify-files-percent", "100"}); err != nil {
		t.Fatalf("snapshot verify: %v", err)
	}

	// Encryption must actually be applied (algorithm non-null in status JSON).
	r.reset()
	if err := r.repo.Status(ctx, []string{"--name", "local", "--json"}); err != nil {
		t.Fatalf("repo status: %v", err)
	}
	var status map[string]any
	if err := json.Unmarshal(r.out.Bytes(), &status); err != nil {
		t.Fatalf("parse status json: %v\n%s", err, r.out.String())
	}
	if algo, _ := status["encryption"].(string); algo == "" {
		t.Fatalf("expected non-empty encryption algorithm in status, got %v", status["encryption"])
	}

	r.reset()
	if err := r.repo.Stats(ctx, []string{"--name", "local", "--json"}); err != nil {
		t.Fatalf("repo stats: %v", err)
	}
	if r.out.Len() == 0 {
		t.Fatal("expected non-empty repo stats output")
	}
}

// TestIntegrationS3RoundTrip is plan §9.2 Test B. It needs a MinIO/S3 backend;
// configure via KOPIA_TEST_S3_* env vars or it skips.
func TestIntegrationS3RoundTrip(t *testing.T) {
	requireKopia(t)
	endpoint := os.Getenv("KOPIA_TEST_S3_ENDPOINT")
	bucket := os.Getenv("KOPIA_TEST_S3_BUCKET")
	access := os.Getenv("KOPIA_TEST_S3_ACCESS_KEY")
	secret := os.Getenv("KOPIA_TEST_S3_SECRET_KEY")
	if endpoint == "" || bucket == "" || access == "" || secret == "" {
		t.Skip("set KOPIA_TEST_S3_{ENDPOINT,BUCKET,ACCESS_KEY,SECRET_KEY} to run the S3 round-trip")
	}
	r := newRig(t)
	ctx := context.Background()
	fixtureDir := filepath.Join(t.TempDir(), "fixture")
	want := writeFixture(t, fixtureDir)

	createArgs := []string{
		"--name", "offsite", "--backend", "s3",
		"--bucket", bucket, "--endpoint", endpoint,
		"--access-key", access, "--secret-access-key", secret,
	}
	if os.Getenv("KOPIA_TEST_S3_DISABLE_TLS") != "" {
		createArgs = append(createArgs, "--disable-tls")
	}
	if err := r.repo.Create(ctx, createArgs); err != nil {
		t.Fatalf("repo create s3: %v", err)
	}
	if err := r.snap.Create(ctx, []string{"--repo", "offsite", "--path", fixtureDir, "--json"}); err != nil {
		t.Fatalf("snapshot create: %v", err)
	}
	id := r.latestSnapshotID(t, "offsite", fixtureDir)

	restoreDir := filepath.Join(t.TempDir(), "restore")
	if err := r.snap.Restore(ctx, []string{"--repo", "offsite", "--snapshot", id, "--target", restoreDir}); err != nil {
		t.Fatalf("snapshot restore: %v", err)
	}
	assertManifestsEqual(t, want, hashTree(t, restoreDir))

	r.reset()
	if err := r.snap.Verify(ctx, []string{"--repo", "offsite", "--verify-files-percent", "100"}); err != nil {
		t.Fatalf("snapshot verify: %v", err)
	}
}

// TestIntegrationStandaloneDisasterRecovery is plan §9.2 Test C: prove a plain
// kopia binary (no resource-kopia) can connect + restore using only the
// passphrase, with Vrooli not involved. This is the one place a test is allowed
// to call kopia directly.
func TestIntegrationStandaloneDisasterRecovery(t *testing.T) {
	requireKopia(t)
	r := newRig(t)
	ctx := context.Background()
	repoDir := filepath.Join(t.TempDir(), "fsrepo")
	fixtureDir := filepath.Join(t.TempDir(), "fixture")
	want := writeFixture(t, fixtureDir)

	if err := r.repo.Create(ctx, []string{"--name", "dr", "--backend", "filesystem", "--path", repoDir}); err != nil {
		t.Fatalf("repo create: %v", err)
	}
	if err := r.snap.Create(ctx, []string{"--repo", "dr", "--path", fixtureDir, "--json"}); err != nil {
		t.Fatalf("snapshot create: %v", err)
	}
	id := r.latestSnapshotID(t, "dr", fixtureDir)

	// Recover the passphrase the resource generated and stored in the authority.
	identity, err := kopiaregistry.PassphraseIdentity("dr")
	if err != nil {
		t.Fatal(err)
	}
	passphrase, err := r.store.Resolve(identity, kopiaregistry.PassphraseField)
	if err != nil {
		t.Fatalf("recover passphrase: %v", err)
	}

	// Plain kopia, fresh config file, only the passphrase — Vrooli not involved.
	drConfig := filepath.Join(t.TempDir(), "dr.config")
	restoreDir := filepath.Join(t.TempDir(), "dr-restore")
	plainEnv := append(os.Environ(), "KOPIA_PASSWORD="+passphrase)

	connect := exec.CommandContext(ctx, "kopia", "--config-file", drConfig, "repository", "connect", "filesystem", "--path", repoDir)
	connect.Env = plainEnv
	if out, err := connect.CombinedOutput(); err != nil {
		t.Fatalf("plain kopia connect failed: %v\n%s", err, out)
	}
	restore := exec.CommandContext(ctx, "kopia", "--config-file", drConfig, "snapshot", "restore", id, restoreDir)
	restore.Env = plainEnv
	if out, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("plain kopia restore failed: %v\n%s", err, out)
	}
	assertManifestsEqual(t, want, hashTree(t, restoreDir))
}

// TestIntegrationPolicyRetention is plan §9.2 Test D.
func TestIntegrationPolicyRetention(t *testing.T) {
	requireKopia(t)
	r := newRig(t)
	ctx := context.Background()
	repoDir := filepath.Join(t.TempDir(), "fsrepo")
	fixtureDir := filepath.Join(t.TempDir(), "fixture")
	writeFixture(t, fixtureDir)

	if err := r.repo.Create(ctx, []string{"--name", "ret", "--backend", "filesystem", "--path", repoDir}); err != nil {
		t.Fatalf("repo create: %v", err)
	}
	// Keep only the single most-recent snapshot; zero all GFS buckets so
	// maintenance prunes down to exactly one.
	policyArgs := []string{
		"--repo", "ret", "--path", fixtureDir,
		"--keep-latest", "1", "--keep-hourly", "0", "--keep-daily", "0",
		"--keep-weekly", "0", "--keep-monthly", "0", "--keep-annual", "0",
		"--compression", "zstd",
	}
	if err := r.pol.Set(ctx, policyArgs); err != nil {
		t.Fatalf("policy set: %v", err)
	}
	r.reset()
	if err := r.pol.Show(ctx, []string{"--repo", "ret", "--path", fixtureDir, "--json"}); err != nil {
		t.Fatalf("policy show: %v", err)
	}
	if !bytes.Contains(r.out.Bytes(), []byte("zstd")) {
		t.Fatalf("policy show does not reflect compression: %s", r.out.String())
	}

	for i := 0; i < 3; i++ {
		if err := r.snap.Create(ctx, []string{"--repo", "ret", "--path", fixtureDir, "--json"}); err != nil {
			t.Fatalf("snapshot create %d: %v", i, err)
		}
	}
	if err := r.maint.Run(ctx, []string{"--repo", "ret", "--full"}); err != nil {
		t.Fatalf("maintenance run: %v", err)
	}

	r.reset()
	if err := r.snap.List(ctx, []string{"--repo", "ret", "--path", fixtureDir, "--json"}); err != nil {
		t.Fatalf("snapshot list: %v", err)
	}
	var manifests []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(r.out.Bytes(), &manifests); err != nil {
		t.Fatalf("parse snapshot list: %v\n%s", err, r.out.String())
	}
	if len(manifests) != 1 {
		t.Fatalf("expected retention to keep 1 snapshot, got %d", len(manifests))
	}
}
