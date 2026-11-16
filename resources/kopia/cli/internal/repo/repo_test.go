package repo_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/env"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/invariant"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
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

func TestCreateSFTPUsesAuthorityKeyAndNonSecretProviderArgs(t *testing.T) {
	h := newHarness(t)
	id, err := kopiaregistry.PassphraseIdentity("offsite")
	if err != nil {
		t.Fatal(err)
	}
	key := "-----BEGIN OPENSSH PRIVATE KEY-----\nfixture-secret\n-----END OPENSSH PRIVATE KEY-----\n"
	if err := h.store.Put(id, credentials.SFTPPrivateKeyField, key); err != nil {
		t.Fatal(err)
	}
	h.run.Responder = func(kexec.Call) ([]byte, error) {
		if err := os.MkdirAll(filepath.Dir(h.rt.RepoConfigFile("offsite")), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(h.rt.RepoConfigFile("offsite"), []byte("embedded-key"), 0o600); err != nil {
			return nil, err
		}
		return nil, nil
	}
	args := []string{"--name", "offsite", "--backend", "sftp", "--host", "backup.example", "--username", "vrooli", "--remote-path", "/backup/vrooli", "--port", "2222", "--known-hosts", "/etc/ssh/ssh_known_hosts"}
	if err := h.svc.Create(context.Background(), args); err != nil {
		t.Fatalf("Create: %v", err)
	}
	call, ok := h.run.LastCall()
	if !ok {
		t.Fatal("no Kopia call")
	}
	if call.Env[repoctx.EnvSFTPPrivateKey] != key {
		t.Fatal("authority key not passed to runtime adapter")
	}
	for _, a := range call.Args {
		if strings.Contains(a, "fixture-secret") || a == key {
			t.Fatalf("private key leaked into argv: %q", a)
		}
	}
	if !strings.Contains(strings.Join(call.Args, " "), "--keyfile "+kexec.SFTPKeyFilePlaceholder) {
		t.Fatalf("SFTP keyfile placeholder missing: %v", call.Args)
	}
	e, found, err := h.reg.Get("offsite")
	if err != nil || !found {
		t.Fatalf("registry lookup found=%t err=%v", found, err)
	}
	if e.Backend != registry.BackendSFTP || e.Host != "backup.example" || e.Port != 2222 || e.Username != "vrooli" || e.Path != "/backup/vrooli" || e.KnownHosts != "/etc/ssh/ssh_known_hosts" {
		t.Fatalf("unexpected SFTP registry entry: %+v", e)
	}
	data, err := os.ReadFile(h.rt.RegistryFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "fixture-secret") || strings.Contains(string(data), "PRIVATE KEY") {
		t.Fatal("registry contains private key material")
	}
	if err := h.svc.Connect(context.Background(), []string{"--name", "offsite"}); err != nil {
		t.Fatalf("Connect using registered SFTP backend: %v", err)
	}
	call, ok = h.run.LastCall()
	if !ok || call.Env[repoctx.EnvSFTPPrivateKey] != key {
		t.Fatal("registered SFTP connect did not resolve authority key")
	}
	if !strings.Contains(strings.Join(call.Args, " "), "--embed-credentials") {
		t.Fatalf("SFTP connect omitted credential embedding: %v", call.Args)
	}
}

func TestCreateAmbiguousRunnerFailureRetainsPassphraseAndRefusesBlindRetry(t *testing.T) {
	h := newHarness(t)
	h.run.Err = errors.New("runner lost connection after dispatch")
	dest := filepath.Join(t.TempDir(), "fsrepo")
	args := []string{"--name", "uncertain", "--backend", "filesystem", "--path", dest}

	err := h.svc.Create(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), "outcome is ambiguous") {
		t.Fatalf("Create error = %v, want ambiguous outcome", err)
	}
	identity, _ := kopiaregistry.PassphraseIdentity("uncertain")
	passphrase, resolveErr := h.store.Resolve(identity, kopiaregistry.PassphraseField)
	if resolveErr != nil || passphrase == "" {
		t.Fatalf("passphrase was not retained: %v", resolveErr)
	}

	if err := h.svc.Create(context.Background(), args); err == nil || !strings.Contains(err.Error(), "refusing to rotate") {
		t.Fatalf("retry error = %v, want safe refusal", err)
	}
	if len(h.run.Calls) != 1 {
		t.Fatalf("runner calls = %d, want one; ambiguous retry must not create again", len(h.run.Calls))
	}
	if got, err := h.store.Resolve(identity, kopiaregistry.PassphraseField); err != nil || got != passphrase {
		t.Fatalf("retry changed passphrase: got=%q err=%v", got, err)
	}
}

func TestCreateRegistryFailureRetainsPassphraseAndRefusesBlindRetry(t *testing.T) {
	h := newHarness(t)
	// Replace the registry filename with a directory from the runner callback,
	// after Create's preflight read and just before the post-create upsert.
	registryPath := filepath.Join(t.TempDir(), "registry.json")
	h.svc.Registry = registry.New(registryPath)
	h.run.Responder = func(kexec.Call) ([]byte, error) {
		if err := os.MkdirAll(registryPath, 0o700); err != nil {
			return nil, err
		}
		return nil, nil
	}
	dest := filepath.Join(t.TempDir(), "fsrepo")
	args := []string{"--name", "catalog-failure", "--backend", "filesystem", "--path", dest}

	err := h.svc.Create(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), "registry update failed") {
		t.Fatalf("Create error = %v, want registry failure", err)
	}
	identity, _ := kopiaregistry.PassphraseIdentity("catalog-failure")
	passphrase, resolveErr := h.store.Resolve(identity, kopiaregistry.PassphraseField)
	if resolveErr != nil || passphrase == "" {
		t.Fatalf("passphrase was not retained: %v", resolveErr)
	}
	if err := h.svc.Create(context.Background(), args); err == nil || !strings.Contains(err.Error(), "read repository registry") {
		t.Fatalf("retry error = %v, want safe refusal", err)
	}
	if len(h.run.Calls) != 1 {
		t.Fatalf("runner calls = %d, want one; registry retry must not create again", len(h.run.Calls))
	}
	if got, err := h.store.Resolve(identity, kopiaregistry.PassphraseField); err != nil || got != passphrase {
		t.Fatalf("retry changed passphrase: got=%q err=%v", got, err)
	}
}

func TestConnectRebindsExistingFilesystemRepositoryWithoutChangingCredentialIdentity(t *testing.T) {
	h := newHarness(t)
	const repoName = "free-disk-root"
	const oldPath = "/media/old-mount/vrooli-backups/free-disk-root/repositories/free-disk-root.kopia"
	const newPath = "/media/new-mount/vrooli-backups/free-disk-root/repositories/free-disk-root.kopia"
	const passphrase = "existing-repository-passphrase"
	h.store.SeedPassphrase(repoName, passphrase)
	identity, err := kopiaregistry.PassphraseIdentity(repoName)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: repoName, Backend: registry.BackendFilesystem, ConfigFile: h.rt.RepoConfigFile(repoName), CacheDir: h.rt.RepoCacheDir(repoName), Path: oldPath}); err != nil {
		t.Fatal(err)
	}

	if err := h.svc.Connect(context.Background(), []string{"--name", repoName, "--backend", "filesystem", "--path", newPath}); err != nil {
		t.Fatalf("Connect rebind: %v", err)
	}
	entry, found, err := h.reg.Get(repoName)
	if err != nil || !found {
		t.Fatalf("registered repository missing: found=%v err=%v", found, err)
	}
	if entry.Path != newPath || entry.Name != repoName || entry.Backend != registry.BackendFilesystem {
		t.Fatalf("registry entry = %+v, want same repo at %q", entry, newPath)
	}
	if got, err := h.store.Resolve(identity, kopiaregistry.PassphraseField); err != nil || got != passphrase {
		t.Fatalf("passphrase identity changed: value=%q err=%v", got, err)
	}
	if len(h.run.Calls) != 1 {
		t.Fatalf("Kopia calls = %d, want one connect and no create", len(h.run.Calls))
	}
	call := h.run.Calls[0]
	want := []string{"--config-file", h.rt.RepoConfigFile(repoName), "repository", "connect", "filesystem", "--path", newPath, "--cache-directory", h.rt.RepoCacheDir(repoName)}
	if !argsEqual(call.Args, want) {
		t.Fatalf("argv = %v, want %v", call.Args, want)
	}
	if call.Env[repoctx.EnvPassword] != passphrase {
		t.Fatal("connect must reuse the repository's existing passphrase")
	}
}

func TestConnectFailureDoesNotCommitCandidateRegistryPath(t *testing.T) {
	h := newHarness(t)
	const repoName = "connect-failure"
	const oldPath = "/media/known/repositories/connect-failure.kopia"
	const candidatePath = "/media/candidate/repositories/connect-failure.kopia"
	h.store.SeedPassphrase(repoName, "existing-repository-passphrase")
	if err := h.reg.Upsert(registry.Entry{Name: repoName, Backend: registry.BackendFilesystem, ConfigFile: h.rt.RepoConfigFile(repoName), CacheDir: h.rt.RepoCacheDir(repoName), Path: oldPath}); err != nil {
		t.Fatal(err)
	}
	h.run.Err = errors.New("injected Kopia connect failure")
	err := h.svc.Connect(context.Background(), []string{"--name", repoName, "--backend", "filesystem", "--path", candidatePath})
	if err == nil || !strings.Contains(err.Error(), "was not committed to the registry") {
		t.Fatalf("Connect error = %v, want explicit uncommitted candidate failure", err)
	}
	entry, found, err := h.reg.Get(repoName)
	if err != nil || !found {
		t.Fatalf("original registry entry missing after failed connect: found=%v err=%v", found, err)
	}
	if entry.Path != oldPath {
		t.Fatalf("failed connect changed registered path to %q, want original %q", entry.Path, oldPath)
	}
	if len(h.run.Calls) != 1 || !strings.Contains(strings.Join(h.run.Calls[0].Args, " "), candidatePath) {
		t.Fatalf("Connect did not attempt the requested candidate exactly once: %+v", h.run.Calls)
	}
}

func TestConnectRequiresRegistryBeforeBackendSideEffects(t *testing.T) {
	h := newHarness(t)
	const repoName = "no-registry"
	h.store.SeedPassphrase(repoName, "existing-repository-passphrase")
	h.svc.Registry = nil
	err := h.svc.Connect(context.Background(), []string{"--name", repoName, "--backend", "filesystem", "--path", "/media/candidate/repository"})
	if err == nil || !strings.Contains(err.Error(), "registry is unavailable") {
		t.Fatalf("Connect error = %v, want registry precondition failure", err)
	}
	if len(h.run.Calls) != 0 {
		t.Fatalf("Kopia ran without a registry commit target: %+v", h.run.Calls)
	}
}

func TestConnectS3RequiresAuthorityCredentialsAndPreservesActivePair(t *testing.T) {
	t.Run("does not stage credentials when no complete active pair exists", func(t *testing.T) {
		for _, partial := range []bool{false, true} {
			t.Run(map[bool]string{false: "absent", true: "partial"}[partial], func(t *testing.T) {
				h := newHarness(t)
				const repoName = "s3-connect-unprovisioned"
				h.store.SeedPassphrase(repoName, "existing-passphrase")
				if partial {
					identity, err := kopiaregistry.PassphraseIdentity(repoName)
					if err != nil {
						t.Fatal(err)
					}
					h.store.Seed(identity, credentials.S3AccessKeyIDField, "preserve-me")
				}
				if err := h.reg.Upsert(registry.Entry{Name: repoName, Backend: registry.BackendS3, ConfigFile: h.rt.RepoConfigFile(repoName), CacheDir: h.rt.RepoCacheDir(repoName), Bucket: "known-bucket"}); err != nil {
					t.Fatal(err)
				}
				err := h.svc.Connect(context.Background(), []string{"--name", repoName, "--backend", "s3", "--bucket", "candidate-bucket"})
				if err == nil || !strings.Contains(err.Error(), "not provisioned in the credential authority") {
					t.Fatalf("Connect error = %v, want fail-closed credential staging error", err)
				}
				if len(h.run.Calls) != 0 {
					t.Fatalf("Kopia ran with unprovisioned candidate credentials: %+v", h.run.Calls)
				}
				if partial {
					identity, err := kopiaregistry.PassphraseIdentity(repoName)
					if err != nil {
						t.Fatal(err)
					}
					got, err := h.store.Resolve(identity, credentials.S3AccessKeyIDField)
					if err != nil || got != "preserve-me" {
						t.Fatalf("partial credential changed: got=%q err=%v", got, err)
					}
					if _, err := h.store.Resolve(identity, credentials.S3SecretAccessKeyField); err == nil {
						t.Fatal("Connect unexpectedly staged a secret for a partial credential identity")
					}
				}
				entry, found, err := h.reg.Get(repoName)
				if err != nil || !found || entry.Bucket != "known-bucket" {
					t.Fatalf("registry changed during rejected candidate: %+v found=%v err=%v", entry, found, err)
				}
			})
		}
	})

	t.Run("rejects credential replacement before connect", func(t *testing.T) {
		h := newHarness(t)
		const repoName = "s3-connect-failure"
		const oldBucket = "known-bucket"
		oldCreds := credentials.S3Credentials{AccessKeyID: "old-access", SecretAccessKey: "old-secret"}
		h.store.SeedPassphrase(repoName, "existing-passphrase")
		h.store.SeedS3(repoName, oldCreds)
		if err := h.reg.Upsert(registry.Entry{Name: repoName, Backend: registry.BackendS3, ConfigFile: h.rt.RepoConfigFile(repoName), CacheDir: h.rt.RepoCacheDir(repoName), Bucket: oldBucket}); err != nil {
			t.Fatal(err)
		}
		err := h.svc.Connect(context.Background(), []string{"--name", repoName, "--backend", "s3", "--bucket", "candidate-bucket", "--access-key", "new-access", "--secret-access-key", "new-secret"})
		if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
			t.Fatalf("Connect error = %v, want rejection of secret-bearing flags", err)
		}
		if len(h.run.Calls) != 0 {
			t.Fatalf("Kopia ran before refusing credential replacement: %+v", h.run.Calls)
		}
		got, found, err := credentials.S3CredentialsFor(h.store, repoName)
		if err != nil || !found || got != oldCreds {
			t.Fatalf("active S3 credentials changed: got=%+v found=%v err=%v", got, found, err)
		}
		entry, found, err := h.reg.Get(repoName)
		if err != nil || !found || entry.Bucket != oldBucket {
			t.Fatalf("old S3 registry entry changed: %+v found=%v err=%v", entry, found, err)
		}
	})

	t.Run("failed connect leaves same active credential and registry", func(t *testing.T) {
		h := newHarness(t)
		const repoName = "s3-connect-runner-failure"
		const bucket = "known-bucket"
		activeCreds := credentials.S3Credentials{AccessKeyID: "active-access", SecretAccessKey: "active-secret"}
		h.store.SeedPassphrase(repoName, "existing-passphrase")
		h.store.SeedS3(repoName, activeCreds)
		if err := h.reg.Upsert(registry.Entry{Name: repoName, Backend: registry.BackendS3, ConfigFile: h.rt.RepoConfigFile(repoName), CacheDir: h.rt.RepoCacheDir(repoName), Bucket: bucket}); err != nil {
			t.Fatal(err)
		}
		h.run.Err = errors.New("injected S3 connect failure")
		err := h.svc.Connect(context.Background(), []string{"--name", repoName, "--backend", "s3", "--bucket", bucket})
		if err == nil {
			t.Fatal("expected injected S3 connect failure")
		}
		got, found, err := credentials.S3CredentialsFor(h.store, repoName)
		if err != nil || !found || got != activeCreds {
			t.Fatalf("failed connect changed active S3 credentials: got=%+v found=%v err=%v", got, found, err)
		}
		entry, found, err := h.reg.Get(repoName)
		if err != nil || !found || entry.Bucket != bucket {
			t.Fatalf("failed connect changed S3 registry entry: %+v found=%v err=%v", entry, found, err)
		}
	})
}

func TestCreateS3ReadsCredentialAuthorityAndNeverAcceptsSecretFlags(t *testing.T) {
	h := newHarness(t)
	const accessKey = "AKIAEXAMPLE"
	const secretKey = "sUp3rs3cr3t"
	h.store.SeedS3("offsite", credentials.S3Credentials{AccessKeyID: accessKey, SecretAccessKey: secretKey})

	err := h.svc.Create(context.Background(), []string{
		"--name", "offsite", "--backend", "s3",
		"--bucket", "vrooli-backups", "--endpoint", "minio:9000", "--disable-tls",
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
	// Credentials were sourced from the authority and cannot be supplied as process args.
	if got, _, _ := credentials.S3CredentialsFor(h.store, "offsite"); got.AccessKeyID != accessKey {
		t.Fatal("S3 creds not stored in credential authority")
	}
	if err := h.svc.Create(context.Background(), []string{"--name", "another", "--backend", "s3", "--bucket", "b", "--access-key", accessKey, "--secret-access-key", secretKey}); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("secret flags accepted: %v", err)
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

func TestCacheProbeUsesEphemeralConfigAndCacheWithoutLeakingSecret(t *testing.T) {
	h := newHarness(t)
	const secret = "probe-passphrase-never-in-argv"
	h.store.SeedPassphrase("elements", secret)
	entry := registry.Entry{
		Name: "elements", Backend: registry.BackendFilesystem,
		ConfigFile: h.rt.RepoConfigFile("elements"), CacheDir: h.rt.RepoCacheDir("elements"),
		Path: "/media/operator/Elements/repositories/elements.kopia",
	}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	h.run.Responder = func(call kexec.Call) ([]byte, error) {
		if strings.Contains(strings.Join(call.Args, " "), "repository connect") {
			return []byte("connected\n"), nil
		}
		return []byte("[]\n"), nil
	}
	if err := h.svc.CacheProbe(context.Background(), []string{"--name", "elements", "--json"}); err != nil {
		t.Fatalf("CacheProbe error = %v", err)
	}
	if len(h.run.Calls) != 2 {
		t.Fatalf("Kopia calls = %d, want ephemeral connect and snapshot list", len(h.run.Calls))
	}
	connect, list := h.run.Calls[0], h.run.Calls[1]
	if !argsEqual(connect.Args[2:6], []string{"repository", "connect", "filesystem", "--path"}) || connect.Args[6] != entry.Path {
		t.Fatalf("unexpected isolated connect argv: %v", connect.Args)
	}
	if connect.Args[0] != "--config-file" || list.Args[0] != "--config-file" || connect.Args[1] != list.Args[1] {
		t.Fatalf("isolated config must be shared by the probe steps: connect=%v list=%v", connect.Args, list.Args)
	}
	probeDir := filepath.Dir(connect.Args[1])
	if !strings.HasPrefix(probeDir, filepath.Join(h.rt.CacheRoot, "diagnostics")+string(os.PathSeparator)) {
		t.Fatalf("temporary config escaped diagnostic cache root: %s", probeDir)
	}
	if !strings.Contains(strings.Join(connect.Args, " "), "--cache-directory "+filepath.Join(probeDir, "cache")) {
		t.Fatalf("connect did not receive its isolated cache path: %v", connect.Args)
	}
	if !argsEqual(list.Args[2:], []string{"--log-level=info", "--no-progress", "snapshot", "list", "--json"}) {
		t.Fatalf("unexpected snapshot list argv: %v", list.Args)
	}
	for _, call := range h.run.Calls {
		if call.Env[repoctx.EnvPassword] != secret {
			t.Fatal("probe must resolve the passphrase through credential authority")
		}
		for _, arg := range call.Args {
			if strings.Contains(arg, secret) {
				t.Fatalf("repository passphrase leaked into argv: %q", arg)
			}
		}
	}
	if _, err := os.Stat(probeDir); !os.IsNotExist(err) {
		t.Fatalf("temporary probe directory should be removed, stat err=%v", err)
	}
	var result struct {
		Repository         string `json:"repository"`
		Path               string `json:"path"`
		CacheIsolated      bool   `json:"cache_isolated"`
		RepositoryModified bool   `json:"repository_modified"`
		Status             string `json:"status"`
		SnapshotListSHA    string `json:"snapshot_list_output_sha256"`
	}
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &result); err != nil {
		t.Fatalf("decode cache probe result: %v", err)
	}
	if result.Repository != entry.Name || result.Path != entry.Path || !result.CacheIsolated || result.RepositoryModified || result.Status != "snapshot_list_passed" || result.SnapshotListSHA == "" {
		t.Fatalf("unexpected cache probe result: %+v", result)
	}
}

func TestCacheProbeRejectsUnknownAndNonFilesystemRepositories(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.CacheProbe(context.Background(), []string{"--name", "missing"}); err == nil {
		t.Fatal("expected unknown repository error")
	}
	if err := h.reg.Upsert(registry.Entry{Name: "remote", Backend: registry.BackendS3, ConfigFile: h.rt.RepoConfigFile("remote"), Bucket: "bucket"}); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.CacheProbe(context.Background(), []string{"--name", "remote"}); err == nil || !strings.Contains(err.Error(), "not a registered filesystem repository") {
		t.Fatalf("expected filesystem-only refusal, got %v", err)
	}
	if len(h.run.Calls) != 0 {
		t.Fatal("invalid repository must not reach Kopia")
	}
}

func TestCacheProbeSummarizesMissingBlobFailure(t *testing.T) {
	h := newHarness(t)
	h.store.SeedPassphrase("elements", "probe-passphrase")
	entry := registry.Entry{Name: "elements", Backend: registry.BackendFilesystem, ConfigFile: h.rt.RepoConfigFile("elements"), Path: "/repo/elements.kopia"}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	const missingBlob = "q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144"
	h.run.Responder = func(call kexec.Call) ([]byte, error) {
		if strings.Contains(strings.Join(call.Args, " "), "repository connect") {
			return []byte("connected"), nil
		}
		return nil, errors.New("snapshot list failed: error getting cached content from blob \"" + missingBlob + "\": BLOB not found\nlarge verbose stack")
	}
	if err := h.svc.CacheProbe(context.Background(), []string{"--name", "elements", "--json"}); err != nil {
		t.Fatalf("CacheProbe should emit a structured diagnostic result, got %v", err)
	}
	var result struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &result); err != nil {
		t.Fatalf("decode cache probe result: %v", err)
	}
	if result.Status != "snapshot_list_failed" || !strings.Contains(result.Error, missingBlob) || strings.Contains(result.Error, "verbose stack") {
		t.Fatalf("missing-blob failure should be concise and actionable: %+v", result)
	}
}

func TestCacheProbeReturnsTypedMissingBlobAndSuffixlessOrphanTemp(t *testing.T) {
	h := newHarness(t)
	h.store.SeedPassphrase("elements", "probe-passphrase")
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".shards"), []byte(`{"default":[1,1]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := registry.Entry{Name: "elements", Backend: registry.BackendFilesystem, ConfigFile: h.rt.RepoConfigFile("elements"), Path: root}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	const missingBlob = "q6390189f3d3250c965c737fd82cc103e"
	parts := []string{missingBlob[:1], missingBlob[1:2], missingBlob[2:] + ".f"}
	shard := filepath.Join(root, parts[0], parts[1])
	if err := os.MkdirAll(shard, 0o700); err != nil {
		t.Fatal(err)
	}
	tempPath := filepath.Join(append([]string{root}, parts...)...) + ".tmp"
	if err := os.WriteFile(tempPath, []byte("preserved candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	h.run.Responder = func(call kexec.Call) ([]byte, error) {
		args := strings.Join(call.Args, " ")
		if strings.Contains(args, "repository connect") {
			return []byte("connected"), nil
		}
		return nil, errors.New(`error getting cached content from blob "` + missingBlob + `": BLOB not found`)
	}
	if err := h.svc.CacheProbe(context.Background(), []string{"--name", "elements", "--json"}); err != nil {
		t.Fatalf("CacheProbe should return a typed diagnostic, got %v", err)
	}
	var result struct {
		Blob *struct {
			BlobID        string   `json:"blob_id"`
			RelativePath  string   `json:"relative_path"`
			State         string   `json:"state"`
			OrphanTemp    bool     `json:"orphan_temp"`
			AdjacentTemps []string `json:"adjacent_temps"`
		} `json:"blob"`
		Candidate *struct {
			FinalBlobAbsent bool   `json:"final_blob_absent"`
			OverlayStatus   string `json:"overlay_snapshot_status"`
		} `json:"orphan_temp_blob_candidate"`
	}
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &result); err != nil {
		t.Fatalf("decode cache probe result: %v", err)
	}
	if result.Blob == nil || result.Blob.BlobID != missingBlob || result.Blob.State != "missing" || !result.Blob.OrphanTemp || len(result.Blob.AdjacentTemps) != 1 || result.Blob.RelativePath == tempPath {
		t.Fatalf("missing blob should carry repository-relative orphan classification: %+v", result.Blob)
	}
	if result.Candidate == nil || !result.Candidate.FinalBlobAbsent {
		t.Fatalf("suffixless temp should be exposed as the missing blob candidate: %+v", result.Candidate)
	}
	if _, err := os.Stat(tempPath); err != nil {
		t.Fatalf("read-only probe must preserve candidate bytes: %v", err)
	}
}

func TestRepairFilesystemFormatPlanIsRegisteredAndNonMutating(t *testing.T) {
	h := newHarness(t)
	entry := registry.Entry{
		Name: "elements", Backend: registry.BackendFilesystem,
		ConfigFile: h.rt.RepoConfigFile("elements"), CacheDir: h.rt.RepoCacheDir("elements"),
		Path: "/media/operator/Elements/repositories/elements.kopia",
	}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	h.run.Output = []byte("Would recover repository format from a redundant copy.\n")

	if err := h.svc.RepairFilesystemFormatPlan(context.Background(), []string{"--name", "elements", "--json"}); err != nil {
		t.Fatalf("RepairFilesystemFormatPlan error = %v", err)
	}
	call, ok := h.run.LastCall()
	if !ok {
		t.Fatal("no kopia call recorded")
	}
	want := []string{"--config-file", entry.ConfigFile, "--dangerous-commands=enabled", "repository", "repair", "filesystem", "--path", entry.Path, "--dry-run"}
	if !argsEqual(call.Args, want) {
		t.Fatalf("argv =\n  %v\nwant\n  %v", call.Args, want)
	}
	for _, arg := range call.Args {
		if strings.Contains(arg, "passphrase-value") {
			t.Fatalf("repository passphrase leaked into argv: %q", arg)
		}
	}
	var plan struct {
		Repository  string `json:"repository"`
		Backend     string `json:"backend"`
		Path        string `json:"path"`
		DryRun      bool   `json:"dry_run"`
		PlanSHA256  string `json:"plan_sha256"`
		PlanOutput  string `json:"plan_output"`
		ApplyAction string `json:"apply_action"`
	}
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &plan); err != nil {
		t.Fatalf("decode repair plan: %v", err)
	}
	if plan.Repository != "elements" || plan.Backend != registry.BackendFilesystem || plan.Path != entry.Path || !plan.DryRun || plan.PlanOutput == "" || plan.PlanSHA256 == "" || plan.ApplyAction != "not_supported_by_this_command" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestRepairFilesystemFormatPlanRejectsUnknownAndNonFilesystemRepositories(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.RepairFilesystemFormatPlan(context.Background(), []string{"--name", "missing"}); err == nil {
		t.Fatal("expected unknown repository error")
	}
	if len(h.run.Calls) != 0 {
		t.Fatal("unknown repository must not reach Kopia")
	}
	if err := h.reg.Upsert(registry.Entry{Name: "remote", Backend: registry.BackendS3, ConfigFile: h.rt.RepoConfigFile("remote"), Bucket: "bucket"}); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.RepairFilesystemFormatPlan(context.Background(), []string{"--name", "remote"}); err == nil || !strings.Contains(err.Error(), "not a registered filesystem repository") {
		t.Fatalf("expected filesystem-only refusal, got %v", err)
	}
	if len(h.run.Calls) != 0 {
		t.Fatal("non-filesystem repository must not reach Kopia repair")
	}
}

func TestRepairFilesystemFormatPlanRejectsEmptyDryRunOutput(t *testing.T) {
	h := newHarness(t)
	entry := registry.Entry{
		Name: "elements", Backend: registry.BackendFilesystem,
		ConfigFile: h.rt.RepoConfigFile("elements"), CacheDir: h.rt.RepoCacheDir("elements"),
		Path: "/media/operator/Elements/repositories/elements.kopia",
	}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}

	err := h.svc.RepairFilesystemFormatPlan(context.Background(), []string{"--name", "elements", "--json"})
	if err == nil || !strings.Contains(err.Error(), "returned no reviewable plan") {
		t.Fatalf("expected empty dry-run plan refusal, got %v", err)
	}
	if got := h.svc.Out.(*strings.Builder).String(); got != "" {
		t.Fatalf("empty dry-run output must not emit a success-shaped response: %q", got)
	}
}

func TestIndexRecoveryScanIsNonCommittingAndReviewable(t *testing.T) {
	h := newHarness(t)
	h.store.SeedPassphrase("elements", "test-passphrase")
	entry := registry.Entry{
		Name: "elements", Backend: registry.BackendFilesystem,
		ConfigFile: h.rt.RepoConfigFile("elements"), CacheDir: h.rt.RepoCacheDir("elements"),
		Path: "/media/operator/Elements/repositories/elements.kopia",
	}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	h.run.Output = []byte("Scanned 4 packs; 1 index record could be reconstructed.\n")
	if err := h.svc.IndexRecoveryScan(context.Background(), []string{"--name", "elements", "--json"}); err != nil {
		t.Fatalf("IndexRecoveryScan error = %v", err)
	}
	call, ok := h.run.LastCall()
	if !ok {
		t.Fatal("no kopia call recorded")
	}
	want := []string{"--config-file", entry.ConfigFile, "--log-level=info", "--no-progress", "--dangerous-commands=enabled", "index", "recover", "--no-commit"}
	if !argsEqual(call.Args, want) {
		t.Fatalf("argv =\n  %v\nwant\n  %v", call.Args, want)
	}
	if call.Env[repoctx.EnvPassword] == "" {
		t.Fatal("index recovery scan must resolve the repository secret through the credential authority")
	}
	var result struct {
		Repository string `json:"repository"`
		Backend    string `json:"backend"`
		Path       string `json:"path"`
		Committed  bool   `json:"committed"`
		Deleted    bool   `json:"deleted"`
		OutputSHA  string `json:"output_sha256"`
		Output     string `json:"output"`
	}
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &result); err != nil {
		t.Fatalf("decode scan result: %v", err)
	}
	if result.Repository != "elements" || result.Backend != registry.BackendFilesystem || result.Path != entry.Path || result.Committed || result.Deleted || result.Output == "" || result.OutputSHA == "" {
		t.Fatalf("unexpected scan result: %+v", result)
	}
}

func TestIndexRecoveryScanRejectsUnknownAndNonFilesystemRepositories(t *testing.T) {
	h := newHarness(t)
	if err := h.svc.IndexRecoveryScan(context.Background(), []string{"--name", "missing"}); err == nil {
		t.Fatal("expected unknown repository error")
	}
	if err := h.reg.Upsert(registry.Entry{Name: "remote", Backend: registry.BackendS3, ConfigFile: h.rt.RepoConfigFile("remote"), Bucket: "bucket"}); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.IndexRecoveryScan(context.Background(), []string{"--name", "remote"}); err == nil || !strings.Contains(err.Error(), "not a registered filesystem repository") {
		t.Fatalf("expected filesystem-only refusal, got %v", err)
	}
	if len(h.run.Calls) != 0 {
		t.Fatal("invalid repository must not reach Kopia")
	}
}

func TestIndexRecoveryScanRejectsEmptyOutput(t *testing.T) {
	h := newHarness(t)
	h.store.SeedPassphrase("elements", "test-passphrase")
	entry := registry.Entry{Name: "elements", Backend: registry.BackendFilesystem, ConfigFile: h.rt.RepoConfigFile("elements"), Path: "/repo"}
	if err := h.reg.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.IndexRecoveryScan(context.Background(), []string{"--name", "elements", "--json"}); err == nil || !strings.Contains(err.Error(), "no reviewable output") {
		t.Fatalf("expected empty output refusal, got %v", err)
	}
	if got := h.svc.Out.(*strings.Builder).String(); got != "" {
		t.Fatalf("empty scan must not emit success-shaped output: %q", got)
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

func TestDamageScanCommandRequiresRegisteredFilesystemAndEmitsTypedJSON(t *testing.T) {
	h := newHarness(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "p"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "p", "abcdef.f"), []byte("blob"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: "fixture", Backend: registry.BackendFilesystem, Path: root}); err != nil {
		t.Fatal(err)
	}
	out := &strings.Builder{}
	h.svc.Out = out
	if err := h.svc.DamageScan(context.Background(), []string{"--name", "fixture", "--json"}); err != nil {
		t.Fatal(err)
	}
	var got repo.DamageScan
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatalf("damage scan JSON: %v: %s", err, out.String())
	}
	if got.Files != 1 || got.Bytes != 4 || got.Root != root {
		t.Fatalf("damage scan = %+v, want one 4-byte file under registered root", got)
	}

	if err := h.reg.Upsert(registry.Entry{Name: "remote", Backend: registry.BackendS3}); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DamageScan(context.Background(), []string{"--name", "remote", "--json"}); err == nil {
		t.Fatal("damage scan must refuse a backend the filesystem scanner cannot inspect")
	}

	secondRoot := filepath.Join(t.TempDir(), "second-host-repo")
	if err := os.MkdirAll(filepath.Join(secondRoot, "p"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "kopia.repository.f"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondRoot, "p", "xyz.f"), []byte("blob"), 0o600); err != nil {
		t.Fatal(err)
	}
	pathOut := &strings.Builder{}
	h.svc.Out = pathOut
	if err := h.svc.DamageScan(context.Background(), []string{"--path", secondRoot, "--json"}); err != nil {
		t.Fatalf("explicit read-only path scan: %v", err)
	}
	var pathResult repo.DamageScan
	if err := json.Unmarshal([]byte(pathOut.String()), &pathResult); err != nil || pathResult.Root != secondRoot || pathResult.Files != 2 {
		t.Fatalf("explicit path result=%+v err=%v", pathResult, err)
	}
	if err := h.svc.DamageScan(context.Background(), []string{"--name", "fixture", "--path", secondRoot}); err == nil {
		t.Fatal("scan command must reject ambiguous name and path inputs")
	}
	if err := h.svc.DamageScan(context.Background(), []string{"--path", t.TempDir()}); err == nil {
		t.Fatal("explicit path mode must reject a directory without the Kopia root marker")
	}
}

func TestQuarantineOrphanTempBlobReplaysOneStableSameVolumeMove(t *testing.T) {
	h := newHarness(t)
	root := filepath.Join(t.TempDir(), "repo")
	rel := "s/abc/blob.f.tmp.deadbeef"
	source := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("preserve this unreadable temp")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: "elements", Backend: registry.BackendFilesystem, Path: root, ConfigFile: h.rt.RepoConfigFile("elements"), CacheDir: h.rt.RepoCacheDir("elements")}); err != nil {
		t.Fatal(err)
	}
	h.store.SeedPassphrase("elements", "a sufficiently long test passphrase")
	args := []string{"--name", "elements", "--relative-temp-path", rel, "--confirm-quarantine", "quarantine-orphan-temp:elements:" + rel, "--json"}
	readResult := func() map[string]any {
		t.Helper()
		var got map[string]any
		if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &got); err != nil {
			t.Fatalf("quarantine JSON: %v", err)
		}
		return got
	}
	if err := h.svc.QuarantineOrphanTempBlob(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	first := readResult()
	quarantine, ok := first["quarantine_path"].(string)
	if !ok || first["committed"] != true || first["source_data_deleted"] != false || first["final_blob_written"] != false {
		t.Fatalf("first result = %v", first)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("original remains or lookup failed: %v", err)
	}
	if got, err := os.ReadFile(quarantine); err != nil || string(got) != string(payload) {
		t.Fatalf("quarantine bytes = %q err=%v", got, err)
	}
	h.svc.Out = &strings.Builder{}
	if err := h.svc.QuarantineOrphanTempBlob(context.Background(), args); err != nil {
		t.Fatalf("same approved operation should return its prior result on retry: %v", err)
	}
	second := readResult()
	if second["quarantine_path"] != quarantine || second["committed"] != true {
		t.Fatalf("replayed result = %v; want stable prior quarantine %q", second, quarantine)
	}
	if got, err := os.ReadFile(quarantine); err != nil || string(got) != string(payload) {
		t.Fatalf("replayed quarantine bytes = %q err=%v", got, err)
	}
}

func TestQuarantineOrphanTempBlobAcceptsExactSuffixlessKopiaName(t *testing.T) {
	h := newHarness(t)
	root := filepath.Join(t.TempDir(), "repo")
	rel := "s/abc/session.f.tmp"
	source := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("retain suffixless temp bytes")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: "elements", Backend: registry.BackendFilesystem, Path: root, ConfigFile: h.rt.RepoConfigFile("elements"), CacheDir: h.rt.RepoCacheDir("elements")}); err != nil {
		t.Fatal(err)
	}
	h.store.SeedPassphrase("elements", "a sufficiently long test passphrase")
	args := []string{"--name", "elements", "--relative-temp-path", rel, "--confirm-quarantine", "quarantine-orphan-temp:elements:" + rel, "--json"}
	if err := h.svc.QuarantineOrphanTempBlob(context.Background(), args); err != nil {
		t.Fatalf("quarantine suffixless temp: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &got); err != nil {
		t.Fatal(err)
	}
	quarantine, ok := got["quarantine_path"].(string)
	if !ok || got["source_data_deleted"] != false || got["final_blob_written"] != false {
		t.Fatalf("result = %v", got)
	}
	if _, err := os.Lstat(source); !os.IsNotExist(err) {
		t.Fatalf("original source remains or lookup failed: %v", err)
	}
	if actual, err := os.ReadFile(quarantine); err != nil || !bytes.Equal(actual, payload) {
		t.Fatalf("quarantine bytes = %q err=%v", actual, err)
	}
}

func TestRestoreFromTempPublishesExactHashBoundObjectAndReplays(t *testing.T) {
	h := newHarness(t)
	root := filepath.Join(t.TempDir(), "repo")
	rel := "q/abc/object.f.tmp.deadbeef"
	source := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("verified temp object")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: "fixture", Backend: registry.BackendFilesystem, Path: root, ConfigFile: h.rt.RepoConfigFile("fixture"), CacheDir: h.rt.RepoCacheDir("fixture")}); err != nil {
		t.Fatal(err)
	}
	h.store.SeedPassphrase("fixture", "a sufficiently long test passphrase")
	digest := sha256.Sum256(payload)
	sha := hex.EncodeToString(digest[:])
	args := []string{"--name", "fixture", "--repository-path", root, "--relative-temp-path", rel, "--temp-sha256", sha, "--confirm-restore", "restore-temp:fixture:" + root + ":" + rel + ":" + sha, "--json"}
	if err := h.svc.RestoreFromTemp(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(filepath.Dir(source), "object.f")
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("target bytes = %q err=%v", got, err)
	}
	if got, err := os.ReadFile(source); err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("source bytes = %q err=%v", got, err)
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &first); err != nil {
		t.Fatal(err)
	}
	if first["committed"] != true || first["source_preserved"] != true || first["target_replaced"] != false || first["replay"] != false {
		t.Fatalf("first result = %v", first)
	}
	h.svc.Out = &strings.Builder{}
	if err := h.svc.RestoreFromTemp(context.Background(), args); err != nil {
		t.Fatalf("exact replay: %v", err)
	}
	var replay map[string]any
	if err := json.Unmarshal([]byte(h.svc.Out.(*strings.Builder).String()), &replay); err != nil {
		t.Fatal(err)
	}
	if replay["replay"] != true || replay["target_replaced"] != false {
		t.Fatalf("replay result = %v", replay)
	}
}

func TestRestoreFromTempRejectsHashMismatchAndConflictingTarget(t *testing.T) {
	h := newHarness(t)
	root := filepath.Join(t.TempDir(), "repo")
	rel := "s/abc/object.f.tmp.deadbeef"
	source := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(source), "object.f"), []byte("different"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.reg.Upsert(registry.Entry{Name: "fixture", Backend: registry.BackendFilesystem, Path: root, ConfigFile: h.rt.RepoConfigFile("fixture"), CacheDir: h.rt.RepoCacheDir("fixture")}); err != nil {
		t.Fatal(err)
	}
	h.store.SeedPassphrase("fixture", "a sufficiently long test passphrase")
	digest := sha256.Sum256([]byte("source"))
	sha := hex.EncodeToString(digest[:])
	args := []string{"--name", "fixture", "--repository-path", root, "--relative-temp-path", rel, "--temp-sha256", strings.Repeat("0", 64), "--confirm-restore", "restore-temp:fixture:" + root + ":" + rel + ":" + strings.Repeat("0", 64)}
	if err := h.svc.RestoreFromTemp(context.Background(), args); err == nil || !strings.Contains(err.Error(), "hash changed") {
		t.Fatalf("hash mismatch error = %v", err)
	}
	args[7] = sha
	args[9] = "restore-temp:fixture:" + root + ":" + rel + ":" + sha
	if err := h.svc.RestoreFromTemp(context.Background(), args); err == nil || !strings.Contains(err.Error(), "different") {
		t.Fatalf("conflicting target error = %v", err)
	}
}
