package recoverypoint

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testKeyRef = "vrooli/cloud-recovery:key"

var testKeys = StaticKeys{testKeyRef: []byte("canary-key-material-9f3a")}

var fastSealer = Sealer{Iterations: 1_000}

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func uploadsFixture(t *testing.T) (string, Binding) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "host-a", "uploads")
	writeTree(t, dir, map[string]string{"invoice-001.txt": "invoice one\n", "nested/photo-002.txt": "photo two\n"})
	return dir, Binding{ID: "uploads", Owner: "app", Kind: KindFiles, Provider: ProviderObjectStore, Locator: dir}
}

func captureUploads(t *testing.T, root string, bindings ...Binding) (CaptureRequest, Manifest) {
	t.Helper()
	req := CaptureRequest{
		DeploymentID: "dep-1", RecoveryPointID: "rp-1", Dir: filepath.Join(root, "recovery-points", "rp-1"),
		Bindings: bindings, Refs: Refs{SchemaVersion: "3", ConfigurationDigest: "cfg-1", CredentialVersionRefs: []string{"vrooli/postgres:password@v2"}},
		KeyRef: testKeyRef, Keys: testKeys, Sealer: fastSealer,
		Providers: Registry{ProviderObjectStore: ObjectStore{}}, Provider: "data-backup-manager", ProviderRef: "target:app/uploads",
		RetentionPolicy: "keep-3", MigrationPosture: PostureGreenfieldWithData,
	}
	manifest, err := Capture(context.Background(), req)
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	return req, manifest
}

// TestCaptureRestoreVerifyRoundTrip [REQ:STC-P0-029] [REQ:STC-P0-030]: an
// object-store binding is captured under a consistency record, sealed, and
// restored byte-for-byte into a clean directory; verify proves integrity.
func TestCaptureRestoreVerifyRoundTrip(t *testing.T) {
	root := t.TempDir()
	src, binding := uploadsFixture(t)
	req, manifest := captureUploads(t, root, binding)
	if !manifest.Encrypted || manifest.KeyRef != testKeyRef || manifest.Digest == "" {
		t.Fatalf("manifest = %+v", manifest)
	}
	if string(testKeys[testKeyRef]) == "" {
		t.Fatal("test key missing")
	}
	raw, _ := os.ReadFile(filepath.Join(req.Dir, ManifestFile))
	if strings.Contains(string(raw), string(testKeys[testKeyRef])) {
		t.Fatal("key material leaked into the manifest")
	}
	if strings.Contains(string(raw), "invoice one") {
		t.Fatal("plaintext leaked into the manifest")
	}
	if _, err := os.Stat(filepath.Join(req.Dir, "uploads.tar")); !os.IsNotExist(err) {
		t.Fatal("plaintext artifact left beside the sealed one")
	}
	if manifest.Consistency[0].WriteQuiescence != QuiescenceNotDeclared || manifest.Consistency[0].Mode != ModeObjectInventory {
		t.Fatalf("consistency = %+v", manifest.Consistency)
	}
	if manifest.Checksums["uploads"].Count != 2 || !manifest.Checksums["uploads"].Comparable {
		t.Fatalf("checksums = %+v", manifest.Checksums)
	}

	target := filepath.Join(root, "host-b", "uploads")
	report, err := Restore(context.Background(), RestoreRequest{Dir: req.Dir, Into: map[string]string{"uploads": target}, Keys: testKeys, Sealer: fastSealer, Providers: req.Providers})
	if err != nil || report.Outcome != OutcomeSucceeded || !report.Bindings[0].Matched {
		t.Fatalf("restore: %v report=%+v", err, report)
	}
	srcInv, _, _ := DirInventory(src)
	dstInv, _, _ := DirInventory(target)
	if srcInv != dstInv {
		t.Fatalf("restored inventory %+v != source %+v", dstInv, srcInv)
	}
	verify, err := Verify(context.Background(), VerifyRequest{Dir: req.Dir, Keys: testKeys, Sealer: fastSealer})
	if err != nil || verify.Outcome != OutcomeSucceeded || !verify.ArtifactsOpened || verify.Refs.SchemaVersion != "3" {
		t.Fatalf("verify: %v report=%+v", err, verify)
	}

	// Capture is idempotent for the same (deployment, id).
	again, err := Capture(context.Background(), req)
	if err != nil || again.Digest != manifest.Digest {
		t.Fatalf("second capture: %v digest=%s want %s", err, again.Digest, manifest.Digest)
	}
}

// TestCorruptArtifactIsRefusedBeforeAnyWrite [REQ:STC-P0-030] proves P12-A04:
// a flipped byte in a sealed artifact or a tampered manifest is
// recovery_point_corrupt and the target stays untouched.
func TestCorruptArtifactIsRefusedBeforeAnyWrite(t *testing.T) {
	root := t.TempDir()
	_, binding := uploadsFixture(t)
	req, _ := captureUploads(t, root, binding)
	sealedPath := filepath.Join(req.Dir, "uploads.sealed")
	sealed, _ := os.ReadFile(sealedPath)
	// Flip a byte inside the ciphertext field without changing the length.
	idx := strings.Index(string(sealed), `"ciphertext":"`) + len(`"ciphertext":"`) + 5
	if sealed[idx] == 'A' {
		sealed[idx] = 'B'
	} else {
		sealed[idx] = 'A'
	}
	if err := os.WriteFile(sealedPath, sealed, 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "host-b", "uploads")
	report, err := Restore(context.Background(), RestoreRequest{Dir: req.Dir, Into: map[string]string{"uploads": target}, Keys: testKeys, Sealer: fastSealer, Providers: req.Providers})
	if CodeOf(err) != CodeRecoveryPointCorrupt || report.Outcome != OutcomeFailed {
		t.Fatalf("err=%v report=%+v", err, report)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatal("corrupt restore touched the target")
	}
	verify, err := Verify(context.Background(), VerifyRequest{Dir: req.Dir})
	if CodeOf(err) != CodeRecoveryPointCorrupt || verify.ArtifactsIntact {
		t.Fatalf("verify err=%v report=%+v", err, verify)
	}

	// A manifest edit (for example a forged checksum) breaks the digest.
	manifestPath := filepath.Join(req.Dir, ManifestFile)
	raw, _ := os.ReadFile(manifestPath)
	edited := strings.Replace(string(raw), `"count": 2`, `"count": 3`, 1)
	if err := os.WriteFile(manifestPath, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadManifest(req.Dir); CodeOf(err) != CodeRecoveryPointCorrupt {
		t.Fatalf("tampered manifest err=%v", err)
	}
}

// TestMissingKeyIsATypedBlocker [REQ:STC-P0-030] proves P12-A03: a key
// reference that does not resolve is recovery_key_unavailable naming the
// reference, before any target is written, and never a false completion.
func TestMissingKeyIsATypedBlocker(t *testing.T) {
	root := t.TempDir()
	_, binding := uploadsFixture(t)
	req, _ := captureUploads(t, root, binding)
	target := filepath.Join(root, "host-b", "uploads")
	report, err := Restore(context.Background(), RestoreRequest{Dir: req.Dir, Into: map[string]string{"uploads": target}, Keys: StaticKeys{}, Sealer: fastSealer, Providers: req.Providers})
	typed := AsError(err, "")
	if typed.Code != CodeRecoveryKeyMissing || typed.Blocker != testKeyRef || report.Outcome != OutcomeFailed {
		t.Fatalf("err=%+v report=%+v", typed, report)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatal("restore without a key touched the target")
	}
	// The wrong key authenticates as corruption, never as success.
	wrong := StaticKeys{testKeyRef: []byte("other")}
	_, err = Restore(context.Background(), RestoreRequest{Dir: req.Dir, Into: map[string]string{"uploads": target}, Keys: wrong, Sealer: fastSealer, Providers: req.Providers})
	if CodeOf(err) != CodeRecoveryPointCorrupt {
		t.Fatalf("wrong key err=%v", err)
	}
	// Capture without a resolvable key never produces a recovery point.
	missing := req
	missing.Dir = filepath.Join(root, "recovery-points", "rp-2")
	missing.RecoveryPointID = "rp-2"
	missing.Keys = StaticKeys{}
	if _, err := Capture(context.Background(), missing); CodeOf(err) != CodeRecoveryKeyMissing {
		t.Fatalf("capture err=%v", err)
	}
	if _, statErr := os.Stat(missing.Dir); !os.IsNotExist(statErr) {
		t.Fatal("capture without a key left a directory")
	}
}

// TestRestoreRefusesOperatorOwnedTarget [REQ:STC-P0-030]: a non-empty
// target is restore_target_not_clean and nothing is overwritten.
func TestRestoreRefusesOperatorOwnedTarget(t *testing.T) {
	root := t.TempDir()
	_, binding := uploadsFixture(t)
	req, _ := captureUploads(t, root, binding)
	target := filepath.Join(root, "host-b", "uploads")
	writeTree(t, target, map[string]string{"operator.txt": "operator data\n"})
	_, err := Restore(context.Background(), RestoreRequest{Dir: req.Dir, Into: map[string]string{"uploads": target}, Keys: testKeys, Sealer: fastSealer, Providers: req.Providers})
	if CodeOf(err) != CodeRestoreTargetNotClean {
		t.Fatalf("err=%v", err)
	}
	body, _ := os.ReadFile(filepath.Join(target, "operator.txt"))
	if string(body) != "operator data\n" {
		t.Fatal("operator data was modified")
	}
	entries, _ := os.ReadDir(target)
	if len(entries) != 1 {
		t.Fatalf("target gained entries: %d", len(entries))
	}
}

type fakeRunner struct {
	calls [][]string
	env   [][]string
	fail  map[string]error
	out   map[string][]byte
}

func (f *fakeRunner) Run(_ context.Context, tool string, argv []string, env []string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{tool}, argv...))
	f.env = append(f.env, env)
	if err := f.fail[tool]; err != nil {
		return nil, err
	}
	if tool == "pg_dump" {
		for i, arg := range argv {
			if arg == "--file" {
				_ = os.WriteFile(argv[i+1], []byte("PGDMP fake custom dump"), 0o600)
			}
		}
	}
	return f.out[tool], nil
}

// TestPostgresProviderUsesDatabaseNativeArgv [REQ:STC-P0-029]: the SQL
// binding is captured through pg_dump --format=custom with credentials in
// the environment, never in argv, and its inventory counts TOC entries.
func TestPostgresProviderUsesDatabaseNativeArgv(t *testing.T) {
	runner := &fakeRunner{out: map[string][]byte{"pg_restore": []byte(";\n; Archive created\n;\n1; 100 TABLE public records\n2; 101 TABLE DATA public records\n"), "psql": []byte("0\n")}}
	provider := Postgres{Runner: runner, CredentialEnv: func(context.Context, Binding) ([]string, error) {
		return []string{"PGPASSWORD=canary-secret-9f3a", "PGHOST=127.0.0.1"}, nil
	}}
	binding := Binding{ID: "records-db", Kind: KindSQL, Provider: ProviderPostgres, Locator: "fixture_records"}
	stage := t.TempDir()
	result, err := provider.Capture(context.Background(), binding, stage)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inventory.Count != 2 || result.Inventory.Comparable {
		t.Fatalf("inventory = %+v", result.Inventory)
	}
	dump := runner.calls[0]
	if strings.Join(dump, " ") != "pg_dump --format=custom --no-password --dbname fixture_records --file "+filepath.Join(stage, "records-db.pgdump") {
		t.Fatalf("dump argv = %q", dump)
	}
	for _, call := range runner.calls {
		if strings.Contains(strings.Join(call, " "), "canary-secret") {
			t.Fatalf("secret in argv: %q", call)
		}
	}
	if err := provider.Restore(context.Background(), binding, result.ArtifactPath, "fixture_records_b"); err != nil {
		t.Fatal(err)
	}
	restore := runner.calls[len(runner.calls)-1]
	if strings.Join(restore, " ") != "pg_restore --no-password --exit-on-error --dbname fixture_records_b "+result.ArtifactPath {
		t.Fatalf("restore argv = %q", restore)
	}
	runner.out["psql"] = []byte("4\n")
	if err := provider.EnsureClean(context.Background(), binding, "live_db"); CodeOf(err) != CodeRestoreTargetNotClean {
		t.Fatalf("non-empty database err=%v", err)
	}
}

// TestApplicationHooksRecordQuiescence [REQ:STC-P0-029]: a declared quiesce
// hook runs through argv and is released after capture; an undeclared hook is
// recorded as not_declared for files and snapshot_safe for a database.
func TestApplicationHooksRecordQuiescence(t *testing.T) {
	runner := &fakeRunner{}
	hooks := ApplicationHooks{Runner: runner}
	withHook := Binding{ID: "uploads", Kind: KindFiles, Provider: ProviderObjectStore, Locator: "/x", Quiesce: &Hook{Tool: "app-cli", Argv: []string{"maintenance", "enter"}}, Release: &Hook{Tool: "app-cli", Argv: []string{"maintenance", "exit"}}}
	record, release, err := hooks.Enter(context.Background(), withHook, ModeObjectInventory)
	if err != nil || record.WriteQuiescence != QuiescenceQuiesced || record.Token == "" {
		t.Fatalf("record=%+v err=%v", record, err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 2 || runner.calls[1][2] != "exit" {
		t.Fatalf("calls = %q", runner.calls)
	}
	record, _, _ = hooks.Enter(context.Background(), Binding{ID: "u", Kind: KindFiles, Provider: ProviderObjectStore, Locator: "/x"}, ModeObjectInventory)
	if record.WriteQuiescence != QuiescenceNotDeclared {
		t.Fatalf("files without hook = %s", record.WriteQuiescence)
	}
	record, _, _ = hooks.Enter(context.Background(), Binding{ID: "d", Kind: KindSQL, Provider: ProviderPostgres, Locator: "db"}, ModeDatabaseNative)
	if record.WriteQuiescence != QuiescenceSnapshotSafe {
		t.Fatalf("database without hook = %s", record.WriteQuiescence)
	}
	failing := &fakeRunner{fail: map[string]error{"app-cli": errors.New("boom")}}
	if _, _, err := (ApplicationHooks{Runner: failing}).Enter(context.Background(), withHook, ModeObjectInventory); CodeOf(err) != CodeCaptureFailed {
		t.Fatalf("failing hook err=%v", err)
	}
}

// TestCaptureFailureLeavesNoRecoveryPoint [REQ:STC-P0-029]: a provider
// failure on the second binding removes the staging directory entirely.
func TestCaptureFailureLeavesNoRecoveryPoint(t *testing.T) {
	root := t.TempDir()
	_, uploads := uploadsFixture(t)
	missing := Binding{ID: "missing", Kind: KindFiles, Provider: ProviderObjectStore, Locator: filepath.Join(root, "does-not-exist")}
	req := CaptureRequest{DeploymentID: "dep-1", RecoveryPointID: "rp-x", Dir: filepath.Join(root, "rp-x"), Bindings: []Binding{uploads, missing}, KeyRef: testKeyRef, Keys: testKeys, Sealer: fastSealer, Providers: Registry{ProviderObjectStore: ObjectStore{}}}
	if _, err := Capture(context.Background(), req); CodeOf(err) != CodeCaptureFailed {
		t.Fatalf("err=%v", err)
	}
	for _, dir := range []string{req.Dir, req.Dir + stagingSuffix} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("%s left behind", dir)
		}
	}
	unknown := req
	unknown.Bindings = []Binding{{ID: "q", Kind: KindSQL, Provider: "nope", Locator: "db"}}
	if _, err := Capture(context.Background(), unknown); CodeOf(err) != CodeProviderUnavailable {
		t.Fatalf("unknown provider err=%v", err)
	}
}
