package repo_test

import (
	"context"
	"path/filepath"
	"resource-kopia/cli/internal/env"
	"resource-kopia/cli/internal/invariant"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repo"
	"resource-kopia/cli/internal/repoctx"
	"resource-kopia/cli/internal/vault"
	"strings"
	"testing"

	kexecmocks "resource-kopia/cli/internal/kexec/mocks"

	vaultmocks "resource-kopia/cli/internal/vault/mocks"
)

type harness struct {
	svc   repo.Service
	run   *kexecmocks.FakeRunner
	vault *vaultmocks.FakeVault
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
	v := vaultmocks.NewFakeVault()
	reg := registry.New(rt.RegistryFile)
	svc := repo.Service{
		Runner:   run,
		Vault:    v,
		Registry: reg,
		Resolver: repoctx.Resolver{Registry: reg, Vault: v},
		Env:      rt,
		Out:      &strings.Builder{},
	}
	return harness{svc: svc, run: run, vault: v, reg: reg, rt: rt}
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

	// Passphrase injected via env, present in vault, never empty.
	if p := call.Env[repoctx.EnvPassword]; len(p) < 32 {
		t.Fatalf("KOPIA_PASSWORD not injected (len=%d)", len(p))
	}
	if _, ok := h.vault.Value(vault.PassphrasePath("nightly"), "passphrase"); !ok {
		t.Fatal("passphrase not stored in vault")
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
	// Creds + passphrase stored in vault.
	if got, _, _ := vault.S3CredentialsFor(context.Background(), h.vault, "offsite"); got.AccessKeyID != accessKey {
		t.Fatal("S3 creds not stored in vault")
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
	h.vault.Seed(vault.PassphrasePath("nightly"), "passphrase", "passphrase-value-abcdefghijklmnop")
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

	if err := h.svc.Stats(context.Background(), []string{"--name", "nightly", "--json"}); err != nil {
		t.Fatalf("Stats error = %v", err)
	}
	call, _ = h.run.LastCall()
	want = []string{"--config-file", h.rt.RepoConfigFile("nightly"), "content", "stats", "--json"}
	if !argsEqual(call.Args, want) {
		t.Fatalf("stats argv = %v want %v", call.Args, want)
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
