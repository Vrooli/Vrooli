package repo_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/env"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/invariant"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repo"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"

	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"

	kexecmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec/mocks"

	credentialmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials/mocks"
)

type harness struct {
	svc   repo.Service
	run   *kexecmocks.FakeRunner
	store *credentialmocks.FakeStore
	reg   *registry.Registry
	rt    env.Runtime
}

func newHarness(t *testing.T) harness {
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
	run := &kexecmocks.FakeRunner{}
	v := credentialmocks.NewFakeStore()
	reg := registry.New(rt.RegistryFile)
	svc := repo.Service{
		Runner:      run,
		Credentials: v,
		Registry:    reg,
		Resolver:    repoctx.Resolver{Registry: reg, Credentials: v},
		Env:         rt,
		Out:         &strings.Builder{},
	}
	return harness{svc: svc, run: run, store: v, reg: reg, rt: rt}
}

func argsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCreateFilesystemArgv(t *testing.T) {
	h := newHarness(t)
	dest := filepath.Join(t.TempDir(), "fsrepo")

	if err := h.svc.Create(context.Background(), []string{"--name", "nightly", "--backend", "filesystem", "--path", dest}); err != nil {
		t.Fatalf("Create error = %v", err)
	}

	call, ok := h.run.LastCall()
	if !ok {
		t.Fatal("no kopia call recorded")
	}
	want := []string{"--config-file", h.rt.RepoConfigFile("nightly"), "repository", "create", "filesystem", "--path", dest, "--cache-directory", h.rt.RepoCacheDir("nightly")}
	if !argsEqual(call.Args, want) {
		t.Fatalf("argv =\n  %v\nwant\n  %v", call.Args, want)
	}

	// Passphrase injected via env, present in the credential authority, never empty.
	if p := call.Env[repoctx.EnvPassword]; len(p) < 32 {
		t.Fatalf("KOPIA_PASSWORD not injected (len=%d)", len(p))
	}
	identity, err := kopiaregistry.PassphraseIdentity("nightly")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.store.Resolve(identity, kopiaregistry.PassphraseField); err != nil {
		t.Fatalf("passphrase not stored in authority: %v", err)
	}
	if _, hasAWS := call.Env[repoctx.EnvAWSAccessKey]; hasAWS {
		t.Fatal("filesystem repo must not inject AWS creds")
	}

	// Registry reflects the create.
	if _, found, _ := h.reg.Get("nightly"); !found {
		t.Fatal("registry missing nightly after create")
	}

	if tok, found := invariant.FindEncryptionFlag(h.run.AllArgs()); found {
		t.Fatalf("encryption flag leaked into argv: %q", tok)
	}
}

func TestCreateS3ArgvAndCredsInEnvNotArgv(t *testing.T) {
	h := newHarness(t)
	const accessKey = "AKIAEXAMPLE"
	const secretKey = "sUp3rs3cr3t"

	err := h.svc.Create(context.Background(), []string{
		"--name", "offsite", "--backend", "s3",
		"--bucket", "vrooli-backups", "--endpoint", "minio:9000", "--disable-tls",
		"--access-key", accessKey, "--secret-access-key", secretKey,
	})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}

	call, _ := h.run.LastCall()
	want := []string{"--config-file", h.rt.RepoConfigFile("offsite"), "repository", "create", "s3", "--bucket", "vrooli-backups", "--endpoint", "minio:9000", "--disable-tls", "--cache-directory", h.rt.RepoCacheDir("offsite")}
	if !argsEqual(call.Args, want) {
		t.Fatalf("argv =\n  %v\nwant\n  %v", call.Args, want)
	}

	// Creds in env.
	if call.Env[repoctx.EnvAWSAccessKey] != accessKey || call.Env[repoctx.EnvAWSSecretKey] != secretKey {
		t.Fatalf("S3 creds not injected via env: %v", call.Env)
	}
	// Creds NEVER in argv.
	if tok, found := invariant.FindCredentialInArgs(h.run.AllArgs(), accessKey, secretKey); found {
		t.Fatalf("S3 credential leaked into argv: %q", tok)
	}
	// Credentials and passphrase are stored in the authority.
	if got, _, _ := credentials.S3CredentialsFor(h.store, "offsite"); got.AccessKeyID != accessKey {
		t.Fatal("S3 creds not stored in credential authority")
	}
	if tok, found := invariant.FindEncryptionFlag(h.run.AllArgs()); found {
		t.Fatalf("encryption flag leaked into argv: %q", tok)
	}
}

func TestCreateRejectsUnknownBackend(t *testing.T) {
	h := newHarness(t)
	err := h.svc.Create(context.Background(), []string{"--name", "x", "--backend", "ftp"})
	if err == nil || !strings.Contains(err.Error(), "unsupported --backend") {
		t.Fatalf("expected unsupported backend error, got %v", err)
	}
}

func TestCreateRequiresName(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.Create(context.Background(), []string{"--backend", "filesystem", "--path", "/tmp/x"}); err == nil {
		t.Fatal("expected --name required error")
	}
}

func TestStatusAndStatsArgv(t *testing.T) {
	h := newHarness(t)
	// Seed a connected repo.
	h.store.SeedPassphrase("nightly", "passphrase-value-abcdefghijklmnop")
	if err := h.reg.Upsert(registry.Entry{Name: "nightly", Backend: registry.BackendFilesystem, ConfigFile: h.rt.RepoConfigFile("nightly"), Path: "/p"}); err != nil {
		t.Fatal(err)
	}

	if err := h.svc.Status(context.Background(), []string{"--name", "nightly", "--json"}); err != nil {
		t.Fatalf("Status error = %v", err)
	}
	call, _ := h.run.LastCall()
	want := []string{"--config-file", h.rt.RepoConfigFile("nightly"), "repository", "status", "--json"}
	if !argsEqual(call.Args, want) {
		t.Fatalf("status argv = %v want %v", call.Args, want)
	}
	if call.Env[repoctx.EnvPassword] == "" {
		t.Fatal("status must inject passphrase env")
	}

	// `repo stats` reports the physical on-disk footprint via `blob stats`.
	// kopia has no `--json` for stats subcommands, so the wrapper runs the
	// `--raw` text form and constructs JSON itself.
	h.run.Output = []byte("Count: 1332\nTotal: 11097041777\nAverage: 8331112\nHistogram:\n        0 between 0 and 10 (total 0)\n")
	out := h.svc.Out.(*strings.Builder)
	out.Reset()
	if err := h.svc.Stats(context.Background(), []string{"--name", "nightly", "--json"}); err != nil {
		t.Fatalf("Stats error = %v", err)
	}
	call, _ = h.run.LastCall()
	want = []string{"--config-file", h.rt.RepoConfigFile("nightly"), "blob", "stats", "--raw"}
	if !argsEqual(call.Args, want) {
		t.Fatalf("stats argv = %v want %v", call.Args, want)
	}
	if !strings.Contains(out.String(), `"physicalBytes": 11097041777`) {
		t.Fatalf("stats --json must emit physicalBytes; got %q", out.String())
	}
	if !strings.Contains(out.String(), `"blobCount": 1332`) {
		t.Fatalf("stats --json must emit blobCount; got %q", out.String())
	}
}

func TestStatusUnknownRepoFailsClosed(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.Status(context.Background(), []string{"--name", "ghost"}); err == nil {
		t.Fatal("expected error for unknown repo")
	}
	if len(h.run.Calls) != 0 {
		t.Fatal("must not exec kopia for an unknown repo")
	}
}

func TestDeleteRemovesRegistryLocalFilesAndAuthoritySecrets(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	cfg := h.rt.RepoConfigFile("nightly")
	cache := h.rt.RepoCacheDir("nightly")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: "nightly", Backend: registry.BackendFilesystem, ConfigFile: cfg, CacheDir: cache, Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	h.store.SeedPassphrase("nightly", "passphrase-value-abcdefghijklmnop")

	if err := h.svc.Delete(ctx, []string{"--name", "nightly"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, found, err := h.reg.Get("nightly"); err != nil || found {
		t.Fatalf("registry found=%v err=%v, want removed", found, err)
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Fatalf("config file still exists or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("cache dir still exists or unexpected stat error: %v", err)
	}
	identity, err := kopiaregistry.PassphraseIdentity("nightly")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.store.Resolve(identity, kopiaregistry.PassphraseField); err == nil {
		t.Fatal("passphrase credential still present")
	}
	if len(h.run.Calls) != 0 {
		t.Fatalf("delete metadata should not shell out to kopia, got %v", h.run.Calls)
	}
}
