package repo_test

import (
	"context"
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
