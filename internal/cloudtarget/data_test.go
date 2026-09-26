package cloudtarget

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

const testKeyRef = "fixture/recovery:key"

func dataDeps() DataDeps {
	return DataDeps{
		Keys:      recoverypoint.StaticKeys{testKeyRef: []byte("fixture-recovery-key-material")},
		Providers: recoverypoint.Registry{recoverypoint.ProviderObjectStore: recoverypoint.ObjectStore{}},
		Sealer:    recoverypoint.Sealer{Iterations: 1000},
	}
}

func seedUploads(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"invoice-001.txt": "invoice one\n", "nested/notes-003.txt": "notes three\n"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func uploadsBinding(dir string) recoverypoint.Binding {
	return recoverypoint.Binding{ID: "uploads", Kind: recoverypoint.KindFiles, Provider: recoverypoint.ProviderObjectStore, Locator: dir}
}

// TestDataBackupRestoreVerifyRoundTrip [REQ:STC-P0-029] [REQ:STC-P0-030]
// proves backup is fenced and receipted, restore lands in a clean binding
// with matching inventory and a measured RTO/RPO, and verify checks the
// declared invariants.
func TestDataBackupRestoreVerifyRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := NewStore(filepath.Join(root, "deployments"))
	clock := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	store.Now = func() time.Time { clock = clock.Add(time.Second); return clock }
	source := filepath.Join(root, "host-a", "uploads")
	seedUploads(t, source)
	ctx := context.Background()

	manifest, result, err := store.DataBackup(ctx, DataBackupRequest{
		Effect:   EffectRequest{DeploymentID: "dep", OperationID: "op-1", Step: "data.backup", Fence: 3},
		Bindings: []recoverypoint.Binding{uploadsBinding(source)},
		Refs:     recoverypoint.Refs{SchemaVersion: "schema-v1", ConfigurationDigest: "cfg", CredentialVersionRefs: []string{"fixture/store:password@v1"}},
		KeyRef:   testKeyRef, Provider: "data-backup-manager", ProviderRef: "target:42", RetentionPolicy: "keep-7", MigrationPosture: recoverypoint.PostureGreenfieldWithData,
		Deps: dataDeps(),
	})
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if manifest.ID != "op-1-data.backup" || !manifest.Encrypted || manifest.KeyRef != testKeyRef || result.Receipt.Fence != 3 || result.Receipt.Outcome != OutcomeSucceeded {
		t.Fatalf("manifest=%+v receipt=%+v", manifest, result.Receipt)
	}
	if manifest.Checksums["uploads"].Count != 2 || manifest.MigrationPosture != recoverypoint.PostureGreenfieldWithData {
		t.Fatalf("checksums=%+v posture=%s", manifest.Checksums, manifest.MigrationPosture)
	}
	// Replay returns the same manifest without re-capturing.
	replayManifest, replay, err := store.DataBackup(ctx, DataBackupRequest{
		Effect:   EffectRequest{DeploymentID: "dep", OperationID: "op-1", Step: "data.backup", Fence: 3},
		Bindings: []recoverypoint.Binding{uploadsBinding(source)},
		Refs:     recoverypoint.Refs{SchemaVersion: "schema-v1", ConfigurationDigest: "cfg", CredentialVersionRefs: []string{"fixture/store:password@v1"}},
		KeyRef:   testKeyRef, Provider: "data-backup-manager", ProviderRef: "target:42", RetentionPolicy: "keep-7", MigrationPosture: recoverypoint.PostureGreenfieldWithData,
		Deps: dataDeps(),
	})
	if err != nil || !replay.Replayed || replayManifest.Digest != manifest.Digest {
		t.Fatalf("replay err=%v replayed=%v digest=%s", err, replay.Replayed, replayManifest.Digest)
	}
	// Stale fence is refused before any capture.
	if _, _, err := store.DataBackup(ctx, DataBackupRequest{Effect: EffectRequest{DeploymentID: "dep", OperationID: "op-0", Step: "data.backup", Fence: 2}, Bindings: []recoverypoint.Binding{uploadsBinding(source)}, KeyRef: testKeyRef, Deps: dataDeps()}); AsError(err).Code != CodeFenceStale {
		t.Fatalf("stale fence: %v", err)
	}

	verify, err := store.DataVerify(ctx, DataVerifyRequest{DeploymentID: "dep", RecoveryPointID: manifest.ID, OpenArtifacts: true, Expect: map[string]recoverypoint.Inventory{"uploads": manifest.Checksums["uploads"]}, Deps: dataDeps()})
	if err != nil || !verify.ArtifactsOpened || !verify.InvariantsPassed || len(verify.Invariants) != 2 {
		t.Fatalf("verify err=%v report=%+v", err, verify)
	}
	_, err = store.DataVerify(ctx, DataVerifyRequest{DeploymentID: "dep", RecoveryPointID: manifest.ID, Expect: map[string]recoverypoint.Inventory{"uploads": {Count: 99}}, Deps: dataDeps()})
	if AsError(err).Code != recoverypoint.CodeVerifyFailed {
		t.Fatalf("wrong invariant must fail verify: %v", err)
	}

	target := filepath.Join(root, "host-b", "uploads")
	report, restoreResult, err := store.DataRestore(ctx, DataRestoreRequest{
		Effect:          EffectRequest{DeploymentID: "dep", OperationID: "op-2", Step: "data.restore", Fence: 4},
		RecoveryPointID: manifest.ID, Into: map[string]string{"uploads": target}, Deps: dataDeps(),
	})
	if err != nil || report.Outcome != recoverypoint.OutcomeSucceeded || !report.Bindings[0].Matched {
		t.Fatalf("restore err=%v report=%+v", err, report)
	}
	if restoreResult.Receipt.Details["measured_rto_ms"].(int64) <= 0 || restoreResult.Receipt.Details["recovery_point_age_ms"].(int64) <= 0 {
		t.Fatalf("restore receipt must measure RTO and recovery point age: %v", restoreResult.Receipt.Details)
	}
	restoredInv, _, err := recoverypoint.DirInventory(target)
	if err != nil || restoredInv.Checksum != manifest.Checksums["uploads"].Checksum {
		t.Fatalf("restored inventory %+v err=%v", restoredInv, err)
	}
	// A second restore into the now-occupied target is refused (exit 2) and
	// the receipt records the refusal.
	_, refused, err := store.DataRestore(ctx, DataRestoreRequest{
		Effect:          EffectRequest{DeploymentID: "dep", OperationID: "op-3", Step: "data.restore", Fence: 5},
		RecoveryPointID: manifest.ID, Into: map[string]string{"uploads": target}, Deps: dataDeps(),
	})
	if typed := AsError(err); typed.Code != CodeRestoreTargetNotClean || typed.Exit != ExitRefused || refused.Receipt.Outcome != OutcomeFailed {
		t.Fatalf("occupied target: err=%v receipt=%+v", err, refused.Receipt)
	}
}

// TestDataRestoreRefusesCorruptArchiveAndMissingKey [REQ:STC-P0-030] proves
// P12-A03/A04: a tampered sealed artifact is recovery_point_corrupt and an
// unresolvable key reference is recovery_key_unavailable naming the blocker,
// and neither writes into the target.
func TestDataRestoreRefusesCorruptArchiveAndMissingKey(t *testing.T) {
	root := t.TempDir()
	store := NewStore(filepath.Join(root, "deployments"))
	source := filepath.Join(root, "host-a", "uploads")
	seedUploads(t, source)
	ctx := context.Background()
	manifest, _, err := store.DataBackup(ctx, DataBackupRequest{
		Effect:   EffectRequest{DeploymentID: "dep", OperationID: "op-1", Step: "data.backup", Fence: 1},
		Bindings: []recoverypoint.Binding{uploadsBinding(source)}, KeyRef: testKeyRef, Deps: dataDeps(),
	})
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := store.RecoveryPointDir("dep", manifest.ID)
	target := filepath.Join(root, "host-b", "uploads")

	missing := dataDeps()
	missing.Keys = recoverypoint.StaticKeys{}
	_, result, err := store.DataRestore(ctx, DataRestoreRequest{Effect: EffectRequest{DeploymentID: "dep", OperationID: "op-2", Step: "data.restore", Fence: 2}, RecoveryPointID: manifest.ID, Into: map[string]string{"uploads": target}, Deps: missing})
	typed := AsError(err)
	if typed.Code != CodeRecoveryKeyMissing || typed.Exit != ExitRefused || typed.Details["blocker"] != testKeyRef || result.Receipt.Error == nil {
		t.Fatalf("missing key: err=%+v receipt=%+v", typed, result.Receipt)
	}
	if clean, _ := recoverypoint.DirIsClean(target); !clean {
		t.Fatalf("missing key must not write into the target")
	}
	if _, err := store.DataVerify(ctx, DataVerifyRequest{DeploymentID: "dep", RecoveryPointID: manifest.ID, OpenArtifacts: true, Deps: missing}); AsError(err).Code != CodeRecoveryKeyMissing {
		t.Fatalf("verify with missing key: %v", err)
	}

	sealed := filepath.Join(dir, "uploads.sealed")
	raw, err := os.ReadFile(sealed)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 0x5a
	if err := os.WriteFile(sealed, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	_, result, err = store.DataRestore(ctx, DataRestoreRequest{Effect: EffectRequest{DeploymentID: "dep", OperationID: "op-3", Step: "data.restore", Fence: 3}, RecoveryPointID: manifest.ID, Into: map[string]string{"uploads": target}, Deps: dataDeps()})
	typed = AsError(err)
	if typed.Code != CodeRecoveryPointCorrupt || typed.Exit != ExitRefused || result.Receipt.Outcome != OutcomeFailed {
		t.Fatalf("corrupt artifact: err=%+v receipt=%+v", typed, result.Receipt)
	}
	if clean, _ := recoverypoint.DirIsClean(target); !clean {
		t.Fatalf("corrupt artifact must not write into the target")
	}
	if _, err := store.DataVerify(ctx, DataVerifyRequest{DeploymentID: "dep", RecoveryPointID: manifest.ID, Deps: dataDeps()}); AsError(err).Code != CodeRecoveryPointCorrupt {
		t.Fatalf("verify corrupt: %v", err)
	}
	// A missing provider is backup_provider_unavailable, never a generic copy.
	none := dataDeps()
	none.Providers = recoverypoint.Registry{}
	if _, _, err := store.DataBackup(ctx, DataBackupRequest{Effect: EffectRequest{DeploymentID: "dep", OperationID: "op-4", Step: "data.backup", Fence: 4}, Bindings: []recoverypoint.Binding{uploadsBinding(source)}, KeyRef: testKeyRef, Deps: none}); AsError(err).Code != CodeBackupProviderMissing {
		t.Fatalf("missing provider: %v", err)
	}
}

func TestParseRestoreTargetsAndBindings(t *testing.T) {
	into, err := ParseRestoreTargets([]string{"uploads=/srv/uploads", `{"records-db":"fixture_records"}`})
	if err != nil || into["uploads"] != "/srv/uploads" || into["records-db"] != "fixture_records" {
		t.Fatalf("into=%v err=%v", into, err)
	}
	if _, err := ParseRestoreTargets([]string{"uploads"}); AsError(err).Code != CodeInvalidArgument {
		t.Fatalf("malformed target: %v", err)
	}
	bindings, err := ParseDataBindings([]string{`[{"id":"uploads","kind":"files","provider":"object_store","locator":"/srv/uploads"}]`, `{"id":"records-db","kind":"sql","provider":"postgres","locator":"fixture_records"}`})
	if err != nil || len(bindings) != 2 {
		t.Fatalf("bindings=%v err=%v", bindings, err)
	}
	if _, err := ParseDataBindings([]string{`{"id":"x","kind":"tarball","provider":"object_store","locator":"/x"}`}); AsError(err).Code != CodeInvalidArgument {
		t.Fatalf("unknown kind: %v", err)
	}
}
