package walletrecovery

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRecoveryElapsedReportsMeasuredRestoreToProofInterval(t *testing.T) {
	elapsed, err := recoveryElapsed("2026-11-15T11:59:00Z", "2026-11-15T12:00:42Z")
	if err != nil {
		t.Fatal(err)
	}
	if elapsed != 102*time.Second {
		t.Fatalf("elapsed = %s, want 1m42s", elapsed)
	}
	if _, err = recoveryElapsed("2026-11-15T12:01:00Z", "2026-11-15T12:00:00Z"); err == nil {
		t.Fatal("a backwards wall clock must not produce a negative recovery duration")
	}
}

func bridgeTestRunner(owner Runner) Runner {
	return func(ctx context.Context, tool string, args []string, stdin []byte) ([]byte, error) {
		if tool == "vrooli-bridge" {
			if len(args) >= 3 && args[0] == "nodes" && args[1] == "get" {
				return []byte(`{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`), nil
			}
			if len(args) >= 3 && args[0] == "machines" && args[1] == "get" {
				return []byte(`{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"node-1","current":true}]}}`), nil
			}
			if len(args) >= 3 && args[0] == "nodes" && args[1] == "doctor" {
				return []byte(`{}`), nil
			}
			return nil, errors.New("unexpected Bridge command")
		}
		if tool == "data-backup-manager" && len(args) >= 2 && args[0] == "wallet-recovery" {
			flag := func(name string) string {
				for i := 0; i+1 < len(args); i++ {
					if args[i] == name {
						return args[i+1]
					}
				}
				return ""
			}
			switch args[1] {
			case "read-lease-acquire":
				return json.Marshal(map[string]any{"lease_id": flag("--lease-id"), "target_id": "source-target-1", "destination_id": flag("--destination-id"), "snapshot_id": flag("--snapshot-id"), "generation": 4, "expires_at": time.Now().UTC().Add(30 * time.Minute)})
			case "read-lease-renew":
				return json.Marshal(map[string]any{"lease_id": flag("--lease-id"), "target_id": flag("--target-id"), "expires_at": time.Now().UTC().Add(30 * time.Minute)})
			case "read-lease-release":
				return []byte(`{"state":"released"}`), nil
			}
		}
		return owner(ctx, tool, args, stdin)
	}
}

func TestBridgeRecoveryTargetRequiresExactIdentityScopeAndFreshDoctor(t *testing.T) {
	for name, tc := range map[string]struct {
		get       string
		machine   string
		doctorErr error
		wantErr   string
	}{
		"approved live target":  {get: `{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`, machine: `{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"node-1","current":true}]}}`},
		"mismatched identity":   {get: `{"node":{"id":"other","scopes":["scenario-to-cloud:read"]}}`, wantErr: "different durable node identity"},
		"missing scope":         {get: `{"node":{"id":"node-1","scopes":["scenario:read"]}}`, wantErr: "not approved"},
		"wrong machine lineage": {get: `{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`, machine: `{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"old-node","current":true}]}}`, wantErr: "not the current node"},
		"stale or disconnected": {get: `{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`, machine: `{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"node-1","current":true}]}}`, doctorErr: errors.New("heartbeat_fresh age=302623s"), wantErr: "stale, disconnected"},
	} {
		t.Run(name, func(t *testing.T) {
			doctorCalled := false
			runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
				if tool != "vrooli-bridge" {
					t.Fatalf("unexpected executable %s", tool)
				}
				if strings.HasPrefix(strings.Join(args, " "), "nodes get") {
					return []byte(tc.get), nil
				}
				if strings.HasPrefix(strings.Join(args, " "), "machines get") {
					return []byte(tc.machine), nil
				}
				doctorCalled = true
				return nil, tc.doctorErr
			}
			err := verifyBridgeRecoveryTarget(context.Background(), runner, "node-1", "machine-1", "scenario-to-cloud:read")
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				if tc.wantErr == "different durable node identity" && doctorCalled {
					t.Fatal("doctor queried after node identity mismatch")
				}
				return
			}
			if err != nil || !doctorCalled {
				t.Fatalf("live target preflight = %v; doctor called=%t", err, doctorCalled)
			}
		})
	}
	called := false
	err := verifyBridgeRecoveryTarget(context.Background(), func(context.Context, string, []string, []byte) ([]byte, error) {
		called = true
		return nil, nil
	}, "node-1", "machine-1", "vrooli-bridge:write")
	if err == nil || !strings.Contains(err.Error(), "exact Bridge scope scenario-to-cloud:read") || called {
		t.Fatalf("overbroad Bridge scope accepted: err=%v called=%t", err, called)
	}
}

func TestRestoreStopsAtBridgeReadinessBeforeDBMOrTreasury(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	var ownerCalls []string
	runner := func(ctx context.Context, tool string, args []string, stdin []byte) ([]byte, error) {
		if tool == "vrooli-bridge" && args[0] == "nodes" && args[1] == "get" {
			return []byte(`{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`), nil
		}
		if tool == "vrooli-bridge" && args[0] == "machines" {
			return []byte(`{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"node-1","current":true}]}}`), nil
		}
		if tool == "vrooli-bridge" && args[0] == "nodes" && args[1] == "doctor" {
			return nil, errors.New("heartbeat_fresh age=302623s")
		}
		ownerCalls = append(ownerCalls, tool+" "+strings.Join(args, " "))
		if len(stdin) != 0 {
			t.Fatal("operator secret reached an owner before Bridge readiness passed")
		}
		return nil, nil
	}
	err := (Commands{In: strings.NewReader(strings.Repeat("s", 40)), Runner: runner}).Run([]string{
		"restore", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read",
		"--destination-id", "d", "--destination-name", "repo", "--backend", "s3", "--location", "s3://repo", "--created-at", "2026-11-14T12:00:00Z",
		"--staging-dir", stage, "--evidence-file", filepath.Join(root, "evidence.json"), "--secret-stdin",
	})
	if err == nil || !strings.Contains(err.Error(), "Bridge recovery target preflight") {
		t.Fatalf("restore result = %v, want Bridge readiness refusal", err)
	}
	if len(ownerCalls) != 0 {
		t.Fatalf("DBM/Treasury called before Bridge readiness: %v", ownerCalls)
	}
}

func TestRestoreStopsWhenDBMReadLeaseIsUnavailable(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	var ownerCalls []string
	runner := func(ctx context.Context, tool string, args []string, stdin []byte) ([]byte, error) {
		if tool == "vrooli-bridge" {
			switch args[0] {
			case "nodes":
				if args[1] == "get" {
					return []byte(`{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`), nil
				}
				return []byte(`{}`), nil
			case "machines":
				return []byte(`{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"node-1","current":true}]}}`), nil
			}
		}
		ownerCalls = append(ownerCalls, tool+" "+strings.Join(args, " "))
		if len(stdin) != 0 {
			t.Fatal("operator secret sent before DBM read lease was acquired")
		}
		if tool == "data-backup-manager" && args[0] == "destinations" && args[1] == "snapshots" {
			return []byte(`{"snapshots":[{"id":"snap-current","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"}]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" && args[1] == "list" {
			return []byte(`{"targets":[]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "wallet-recovery" && args[1] == "read-lease-acquire" {
			return nil, errors.New("HTTP 503 wallet recovery read lease is not configured")
		}
		return []byte(`{}`), nil
	}
	err := (Commands{In: strings.NewReader(strings.Repeat("s", 40)), Runner: runner}).Run([]string{
		"restore", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read",
		"--destination-id", "dest", "--destination-name", "repo", "--backend", "s3", "--location", "s3://repo", "--created-at", "2026-11-14T12:00:00Z",
		"--staging-dir", stage, "--evidence-file", filepath.Join(root, "evidence.json"), "--secret-stdin",
	})
	if err == nil || !strings.Contains(err.Error(), "reserve exact newest DBM snapshot") {
		t.Fatalf("restore result = %v, want DBM lease refusal", err)
	}
	for _, call := range ownerCalls {
		if strings.Contains(call, "restores verify") || strings.Contains(call, "restores restore") || strings.HasPrefix(call, "treasury ") {
			t.Fatalf("consumer reached artifact restore/Treasury without a DBM read lease: %v", ownerCalls)
		}
	}
}

func TestProofRefusesSnapshotSupersededAfterRestoreBeforeTreasury(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(stage, "wallet.bundle")
	witness := filepath.Join(stage, "witness.json")
	if err := os.WriteFile(bundle, []byte("encrypted-bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(witness, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bundleHash, _ := sha256RegularFile(bundle, maxBundleBytes)
	witnessHash, _ := sha256RegularFile(witness, 16<<10)
	evidencePath := filepath.Join(root, "evidence.json")
	evidence := RecoveryEvidence{Version: 1, WalletID: "w", Network: "net", Address: "0xabcd", BridgeNodeID: "node-1", BridgeMachineID: "machine-1", BridgeScope: "scenario-to-cloud:read", DBMDestinationID: "repo", DBMTargetName: "treasury-wallet-recovery-w", SnapshotID: "restored-snapshot", StagingDir: stage, BundleSHA256: bundleHash, WitnessSHA256: witnessHash, RecoveryStartedAt: "2026-11-15T11:59:00Z", RestoredAt: "2026-11-15T12:00:00Z"}
	if err := writeNewEvidence(evidencePath, evidence); err != nil {
		t.Fatal(err)
	}
	treasuryCalled := false
	runner := bridgeTestRunner(func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		if tool == "data-backup-manager" {
			return []byte(`{"snapshots":[{"id":"newer-tombstone","started_at":"2026-11-15T12:01:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"},{"id":"restored-snapshot","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"}]}`), nil
		}
		if tool == "treasury" {
			treasuryCalled = true
		}
		return []byte(`{}`), nil
	})
	err := (Commands{Runner: runner}).Run([]string{"prove-after-restart", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--expected-address", "0xabcd", "--nonce", "fresh-retry-nonce", "--evidence-file", evidencePath})
	if err == nil || !strings.Contains(err.Error(), "snapshot changed after restore") {
		t.Fatalf("post-restart result = %v, want supersession refusal", err)
	}
	if treasuryCalled {
		t.Fatal("Treasury signed control proof after a newer DBM snapshot superseded the restored bundle")
	}
}

func TestProofStopsOnDisconnectedBridgeBeforeDBMCatalogOrTreasury(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "wallet.bundle"), []byte("encrypted-bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "witness.json"), []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bundleHash, _ := sha256RegularFile(filepath.Join(stage, "wallet.bundle"), maxBundleBytes)
	witnessHash, _ := sha256RegularFile(filepath.Join(stage, "witness.json"), 16<<10)
	evidencePath := filepath.Join(root, "evidence.json")
	evidence := RecoveryEvidence{Version: 1, WalletID: "w", Network: "net", Address: "0xabcd", BridgeNodeID: "node-1", BridgeMachineID: "machine-1", BridgeScope: "scenario-to-cloud:read", DBMDestinationID: "repo", DBMTargetName: "treasury-wallet-recovery-w", SnapshotID: "snap", StagingDir: stage, BundleSHA256: bundleHash, WitnessSHA256: witnessHash, RecoveryStartedAt: "2026-11-15T11:59:00Z", RestoredAt: "2026-11-15T12:00:00Z"}
	if err := writeNewEvidence(evidencePath, evidence); err != nil {
		t.Fatal(err)
	}
	var ownerCalls []string
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		if tool == "vrooli-bridge" && args[0] == "nodes" && args[1] == "get" {
			return []byte(`{"node":{"id":"node-1","scopes":["scenario-to-cloud:read"]}}`), nil
		}
		if tool == "vrooli-bridge" && args[0] == "machines" {
			return []byte(`{"machine":{"id":"machine-1","nodeLineage":[{"nodeId":"node-1","current":true}]}}`), nil
		}
		if tool == "vrooli-bridge" && args[0] == "nodes" && args[1] == "doctor" {
			return nil, errors.New("channel_held=false")
		}
		ownerCalls = append(ownerCalls, tool+" "+strings.Join(args, " "))
		return []byte(`{}`), nil
	}
	err := (Commands{Runner: runner}).Run([]string{"prove-after-restart", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--expected-address", "0xabcd", "--nonce", "fresh-nonce", "--evidence-file", evidencePath})
	if err == nil || !strings.Contains(err.Error(), "Bridge recovery target preflight") {
		t.Fatalf("proof result = %v, want stale Bridge refusal", err)
	}
	if len(ownerCalls) != 0 {
		t.Fatalf("DBM/Treasury called after Bridge disconnected: %v", ownerCalls)
	}
}

func TestDBMToTreasuryPortableRecoveryUsesSameBundleAndSecretOnlyOnStdin(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, "recovery-evidence.json")
	if err := os.Chmod(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	const secret = "operator-held recovery secret used only by this test"
	var calls []string
	var seenSecret []byte
	verifyReads, restoreReads := 0, 0
	runner := func(ctx context.Context, tool string, args []string, stdin []byte) ([]byte, error) {
		calls = append(calls, tool+" "+strings.Join(args, " "))
		for _, arg := range args {
			if strings.Contains(arg, secret) {
				t.Fatalf("secret exposed in argv: %q", arg)
			}
		}
		if tool == "treasury" && args[1] == "restore-portable-recovery" {
			seenSecret = append([]byte(nil), stdin...)
			return []byte(`{"wallet":{"id":"cf3-wallet","network":"eip155:84532","address":"0xAbCd"}}`), nil
		}
		if tool == "treasury" && args[1] == "prove" {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 10*time.Minute || time.Until(deadline) <= 0 {
				t.Fatalf("Treasury proof lacks the bounded context needed to stay within the DBM lease: deadline=%v present=%t", deadline, ok)
			}
			return []byte(`{"proof":{"wallet":{"id":"cf3-wallet","address":"0xAbCd"},"message":"Vrooli research wallet address-control proof v1\nNo spending authorization.\nwallet:cf3-wallet\nbook:test\noperator:test\nnetwork:eip155:84532\naddress:0xAbCd\nnonce:after-restart-unique-001","signature":"0xdeadbeef","verified":true}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "destinations" && args[1] == "snapshots" {
			return []byte(`{"snapshots":[{"id":"snap-42","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-cf3-wallet"},{"id":"snap-41","started_at":"2026-11-14T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-cf3-wallet"}]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" && args[1] == "list" {
			return []byte(`{"targets":[]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" {
			return []byte(`{"target":{"id":"target-7"}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "restores" && args[1] == "verify" {
			return []byte(`{"restore":{"id":"verify-7","target_id":"target-7","destination_id":"repo-7","snapshot_id":"snap-42","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFYING"}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "restores" && args[1] == "get" {
			if args[3] == "verify-7" {
				verifyReads++
				return []byte(`{"restore":{"id":"verify-7","target_id":"target-7","destination_id":"repo-7","snapshot_id":"snap-42","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
			}
			restoreReads++
			return []byte(`{"restore":{"id":"restore-7","target_id":"target-7","destination_id":"repo-7","snapshot_id":"snap-42","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "restores" && args[1] == "restore" {
			path := args[len(args)-1]
			if err := os.WriteFile(filepath.Join(path, "wallet.bundle"), []byte("opaque encrypted bundle"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "witness.json"), []byte(`{"version":1,"mac":"test"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			return []byte(`{"restore":{"id":"restore-7","target_id":"target-7","destination_id":"repo-7","snapshot_id":"snap-42","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORING"}}`), nil
		}
		return []byte(`{}`), nil
	}
	var output strings.Builder
	cmd := Commands{In: strings.NewReader(secret + "\n"), Out: &output, Runner: bridgeTestRunner(runner)}
	restoreArgs := []string{"restore", "--wallet-id", "cf3-wallet", "--network", "eip155:84532", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--destination-id", "repo-7", "--destination-name", "recovery-repo", "--backend", "s3", "--location", "s3://recovery/repository", "--created-at", "2026-11-14T12:00:00Z", "--staging-dir", stage, "--evidence-file", evidence, "--secret-stdin"}
	if err := cmd.Run(restoreArgs); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seenSecret, []byte(secret)) {
		t.Fatalf("Treasury stdin = %q", seenSecret)
	}
	want := []string{
		"data-backup-manager destinations import-existing --id repo-7 --name recovery-repo --backend s3 --location s3://recovery/repository --created-at 2026-11-14T12:00:00Z",
		"data-backup-manager targets list --owner treasury --json",
		"data-backup-manager destinations snapshots --id repo-7 --owner treasury --name treasury-wallet-recovery-cf3-wallet --json",
		"data-backup-manager targets register --owner treasury --name treasury-wallet-recovery-cf3-wallet --kind filesystem --locator " + stage + " --json",
		"data-backup-manager restores verify --target target-7 --destination repo-7 --snapshot snap-42",
		"data-backup-manager restores get --id verify-7 --json",
		"data-backup-manager restores restore --target target-7 --destination repo-7 --snapshot snap-42 --location " + stage,
		"data-backup-manager restores get --id restore-7 --json",
		"data-backup-manager destinations snapshots --id repo-7 --owner treasury --name treasury-wallet-recovery-cf3-wallet --json",
		"treasury wallet restore-portable-recovery --id cf3-wallet --network eip155:84532 --bundle-file " + filepath.Join(stage, "wallet.bundle") + " --witness-file " + filepath.Join(stage, "witness.json") + " --secret-stdin --json",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
	if verifyReads != 1 || restoreReads != 1 {
		t.Fatalf("restore lifecycle reads verify=%d restore=%d, want one each", verifyReads, restoreReads)
	}
	if strings.Contains(output.String(), secret) || !strings.Contains(output.String(), "Restart Treasury") {
		t.Fatalf("unsafe or incomplete operator output: %s", output.String())
	}

	proofOutput := new(strings.Builder)
	if err := (Commands{Out: proofOutput, Runner: bridgeTestRunner(runner)}).Run([]string{"prove-after-restart", "--wallet-id", "cf3-wallet", "--network", "eip155:84532", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--expected-address", "0xabcd", "--nonce", "after-restart-unique-001", "--evidence-file", evidence}); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(evidence)
	if err != nil || !strings.Contains(string(stored), `"snapshot_id": "snap-42"`) || !strings.Contains(string(stored), `"signature": "0xdeadbeef"`) {
		t.Fatalf("provider recovery evidence missing exact snapshot/proof: %s, err=%v", stored, err)
	}
	if !strings.Contains(proofOutput.String(), "after restart") || !strings.Contains(proofOutput.String(), "Recovery elapsed:") {
		t.Fatalf("proof output = %q", proofOutput.String())
	}
	if calls[len(calls)-1] != "treasury wallet prove --id cf3-wallet --network eip155:84532 --nonce after-restart-unique-001 --json" {
		t.Fatalf("proof call = %q", calls[len(calls)-1])
	}
}

func TestRestoreRefusesSnapshotSupersededDuringRestoreBeforeTreasury(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	args := []string{"restore", "--wallet-id", "w", "--network", "eip155:84532", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--destination-id", "repo", "--destination-name", "recovery", "--backend", "s3", "--location", "s3://recovery", "--created-at", "2026-11-14T12:00:00Z", "--staging-dir", stage, "--evidence-file", filepath.Join(root, "evidence.json"), "--secret-stdin"}
	snapshotReads, treasuryCalled := 0, false
	runner := func(_ context.Context, tool string, a []string, _ []byte) ([]byte, error) {
		if tool == "treasury" {
			treasuryCalled = true
		}
		if tool == "data-backup-manager" && a[0] == "destinations" && a[1] == "snapshots" {
			snapshotReads++
			if snapshotReads == 1 {
				return []byte(`{"snapshots":[{"id":"snap-1","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"}]}`), nil
			}
			return []byte(`{"snapshots":[{"id":"snap-tombstone","started_at":"2026-11-15T12:01:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"},{"id":"snap-1","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"}]}`), nil
		}
		if tool == "data-backup-manager" && a[0] == "targets" && a[1] == "list" {
			return []byte(`{"targets":[]}`), nil
		}
		if tool == "data-backup-manager" && a[0] == "targets" {
			return []byte(`{"target":{"id":"target"}}`), nil
		}
		if tool == "data-backup-manager" && a[0] == "restores" && a[1] == "verify" {
			return []byte(`{"restore":{"id":"verify","target_id":"target","destination_id":"repo","snapshot_id":"snap-1","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
		}
		if tool == "data-backup-manager" && a[0] == "restores" && a[1] == "restore" {
			if err := os.WriteFile(filepath.Join(stage, "wallet.bundle"), []byte("cipher"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(stage, "witness.json"), []byte(`{"version":1}`), 0o600); err != nil {
				t.Fatal(err)
			}
			return []byte(`{"restore":{"id":"restore","target_id":"target","destination_id":"repo","snapshot_id":"snap-1","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
		}
		if tool == "data-backup-manager" && a[0] == "restores" && a[1] == "get" {
			return []byte(`{"restore":{"id":"restore","target_id":"target","destination_id":"repo","snapshot_id":"snap-1","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
		}
		return []byte(`{}`), nil
	}
	err := (Commands{In: strings.NewReader(strings.Repeat("s", 40)), Runner: bridgeTestRunner(runner)}).Run(args)
	if err == nil || !strings.Contains(err.Error(), "snapshot changed during restore") {
		t.Fatalf("restore error = %v", err)
	}
	if treasuryCalled {
		t.Fatal("Treasury received bundle/secret after a newer snapshot superseded the restored snapshot")
	}
	if snapshotReads != 2 {
		t.Fatalf("snapshot list reads = %d, want initial and post-restore recheck", snapshotReads)
	}
}

func TestProveAfterRestartRefusesArtifactTamperingBeforeTreasuryCall(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(stage, "wallet.bundle")
	witness := filepath.Join(stage, "witness.json")
	if err := os.WriteFile(bundle, []byte("ciphertext"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(witness, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bundleHash, _ := sha256RegularFile(bundle, maxBundleBytes)
	witnessHash, _ := sha256RegularFile(witness, 16<<10)
	evidencePath := filepath.Join(root, "evidence.json")
	evidence := RecoveryEvidence{Version: 1, WalletID: "w", Network: "net", Address: "0xabcd", BridgeNodeID: "node-1", BridgeMachineID: "machine-1", BridgeScope: "scenario-to-cloud:read", DBMDestinationID: "repo", DBMTargetName: "target", SnapshotID: "snap", StagingDir: stage, BundleSHA256: bundleHash, WitnessSHA256: witnessHash, RecoveryStartedAt: "2026-11-15T11:59:00Z", RestoredAt: "2026-11-15T12:00:00Z"}
	if err := writeNewEvidence(evidencePath, evidence); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundle, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	err := (Commands{Runner: bridgeTestRunner(func(_ context.Context, tool string, _ []string, _ []byte) ([]byte, error) {
		if tool == "treasury" {
			called = true
		}
		return nil, nil
	})}).Run([]string{"prove-after-restart", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--expected-address", "0xabcd", "--nonce", "fresh-nonce", "--evidence-file", evidencePath})
	if err == nil || called {
		t.Fatalf("tampered recovery evidence accepted: err=%v called=%t", err, called)
	}
}

func TestRestoreRefusesWeakSecretUnsafeStageAndUnsupportedBackend(t *testing.T) {
	base := []string{"restore", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--destination-id", "d", "--destination-name", "repo", "--backend", "s3", "--location", "s3://repo", "--created-at", "2026-11-14T12:00:00Z", "--snapshot-id", "s", "--staging-dir", t.TempDir(), "--secret-stdin"}
	for name, scenario := range map[string]struct {
		input  string
		adjust func([]string) []string
	}{
		"short secret": {input: "short"},
		"local repository": {input: strings.Repeat("x", 40), adjust: func(args []string) []string {
			for i := range args {
				if args[i] == "--backend" {
					args[i+1] = "filesystem"
				}
			}
			return args
		}},
	} {
		t.Run(name, func(t *testing.T) {
			args := append([]string(nil), base...)
			if scenario.adjust != nil {
				args = scenario.adjust(args)
			}
			err := (Commands{In: strings.NewReader(scenario.input), Runner: func(context.Context, string, []string, []byte) ([]byte, error) {
				t.Fatal("owner command called before validation")
				return nil, nil
			}}).Run(args)
			if err == nil {
				t.Fatal("invalid recovery input accepted")
			}
		})
	}
	occupied := t.TempDir()
	if err := os.Chmod(occupied, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(occupied, "other"), []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := append([]string(nil), base...)
	for i := range args {
		if args[i] == "--staging-dir" {
			args[i+1] = occupied
		}
	}
	err := (Commands{In: strings.NewReader(strings.Repeat("s", 40)), Runner: func(context.Context, string, []string, []byte) ([]byte, error) {
		t.Fatal("owner command called for non-empty staging")
		return nil, nil
	}}).Run(args)
	if err == nil {
		t.Fatal("non-empty staging directory accepted")
	}
	if got, _ := os.ReadFile(filepath.Join(occupied, "other")); string(got) != "preserve" {
		t.Fatalf("existing staging data changed: %q", got)
	}
}

func TestRecoveryTargetIdentityCannotBeRepointed(t *testing.T) {
	stage := t.TempDir()
	if err := verifyTargetRegistrationPath([]byte(`{"targets":[{"owner":"treasury","name":"treasury-wallet-recovery-w","locator":"/old/source/path"}]}`), "treasury-wallet-recovery-w", stage); err == nil {
		t.Fatal("existing stable recovery target with another locator was accepted")
	}
	if err := verifyTargetRegistrationPath([]byte(`{"targets":[]}`), "treasury-wallet-recovery-w", stage); err != nil {
		t.Fatalf("empty fresh-node target catalog rejected: %v", err)
	}
}

func TestRestoreRefusesLifecycleSnapshotMismatchBeforeFetchingArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	args := []string{"restore", "--wallet-id", "w", "--network", "eip155:84532", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--destination-id", "repo", "--destination-name", "recovery", "--backend", "s3", "--location", "s3://recovery", "--created-at", "2026-11-14T12:00:00Z", "--staging-dir", stage, "--evidence-file", filepath.Join(root, "evidence.json"), "--secret-stdin"}
	var restoreOrTreasuryCalled bool
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		if tool == "data-backup-manager" && args[0] == "destinations" && args[1] == "snapshots" {
			return []byte(`{"snapshots":[{"id":"snap-current","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"}]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" && args[1] == "list" {
			return []byte(`{"targets":[]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" {
			return []byte(`{"target":{"id":"target"}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "restores" && args[1] == "verify" {
			return []byte(`{"restore":{"id":"verify","target_id":"target","destination_id":"repo","snapshot_id":"snap-stale","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "restores" && args[1] == "restore" || tool == "treasury" {
			restoreOrTreasuryCalled = true
		}
		return []byte(`{}`), nil
	}
	err := (Commands{In: strings.NewReader(strings.Repeat("s", 40)), Runner: bridgeTestRunner(runner)}).Run(args)
	if err == nil || !strings.Contains(err.Error(), "different mode, target, destination, or snapshot") {
		t.Fatalf("restore error = %v, want lifecycle binding refusal", err)
	}
	if restoreOrTreasuryCalled {
		t.Fatal("restore continued to artifact fetch or Treasury after lifecycle snapshot mismatch")
	}
}

func TestRestoreSelectsLatestSnapshotAndRefusesPostRotationTombstoneWithoutFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, "evidence.json")
	selected := ""
	var calls []string
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		calls = append(calls, tool+" "+strings.Join(args, " "))
		if tool == "data-backup-manager" && args[0] == "destinations" && args[1] == "snapshots" {
			return []byte(`{"snapshots":[{"id":"post-rotation","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"},{"id":"pre-rotation","started_at":"2026-11-14T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-w"}]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" && args[1] == "list" {
			return []byte(`{"targets":[]}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "targets" {
			return []byte(`{"target":{"id":"target"}}`), nil
		}
		if tool == "data-backup-manager" && args[0] == "restores" {
			for i, arg := range args {
				if arg == "--snapshot" {
					selected = args[i+1]
				}
			}
			if args[1] == "verify" {
				return []byte(`{"restore":{"id":"verify-1","target_id":"target","destination_id":"repo","snapshot_id":"post-rotation","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
			}
			if args[1] == "restore" {
				if err := os.WriteFile(filepath.Join(stage, "wallet.bundle"), []byte("test ciphertext"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(stage, "witness.json"), []byte(`{"version":1,"mac":"test"}`), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(stage, "revocation.json"), []byte(`{"state":"revoked","epoch":2}`), 0o600); err != nil {
					t.Fatal(err)
				}
				return []byte(`{"restore":{"id":"restore-1","target_id":"target","destination_id":"repo","snapshot_id":"post-rotation","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
			}
		}
		return []byte(`{}`), nil
	}
	args := []string{"restore", "--wallet-id", "w", "--network", "eip155:84532", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--destination-id", "repo", "--destination-name", "recovery", "--backend", "s3", "--location", "s3://recovery", "--created-at", "2026-11-14T12:00:00Z", "--staging-dir", stage, "--evidence-file", evidence, "--secret-stdin"}
	err := (Commands{In: strings.NewReader(strings.Repeat("s", 40)), Runner: bridgeTestRunner(runner)}).Run(args)
	if err == nil || !strings.Contains(err.Error(), "revocation tombstone") {
		t.Fatalf("restore error = %v", err)
	}
	if selected != "post-rotation" {
		t.Fatalf("selected snapshot = %q; want latest post-rotation tombstone", selected)
	}
	for _, call := range calls {
		if strings.Contains(call, "--snapshot pre-rotation") {
			t.Fatalf("fell back to stale snapshot: %s", call)
		}
		if strings.Contains(call, "treasury wallet restore-portable-recovery") {
			t.Fatalf("called Treasury despite revocation tombstone: %s", call)
		}
	}
}

func TestProofRequiresVerifiedMatchingAddressAndSignature(t *testing.T) {
	for name, response := range map[string]string{
		"unverified":        `{"proof":{"wallet":{"id":"w","address":"0xAbCd"},"message":"wallet:w\nnonce:fresh-nonce","signature":"sig","verified":false}}`,
		"wrong address":     `{"proof":{"wallet":{"id":"w","address":"0x0000"},"message":"wallet:w\nnonce:fresh-nonce","signature":"sig","verified":true}}`,
		"missing signature": `{"proof":{"wallet":{"id":"w","address":"0xAbCd"},"message":"wallet:w\nnonce:fresh-nonce","verified":true}}`,
		"wrong nonce":       `{"proof":{"wallet":{"id":"w","address":"0xAbCd"},"message":"wallet:w\nnonce:old-nonce","signature":"sig","verified":true}}`,
		"wrong wallet":      `{"proof":{"wallet":{"id":"other","address":"0xAbCd"},"message":"wallet:other\nnonce:fresh-nonce","signature":"sig","verified":true}}`,
	} {
		t.Run(name, func(t *testing.T) {
			err := (Commands{Runner: bridgeTestRunner(func(context.Context, string, []string, []byte) ([]byte, error) { return []byte(response), nil })}).Run([]string{"prove-after-restart", "--wallet-id", "w", "--network", "net", "--bridge-node-id", "node-1", "--bridge-machine-id", "machine-1", "--bridge-scope", "scenario-to-cloud:read", "--expected-address", "0xabcd", "--nonce", "fresh-nonce"})
			if err == nil {
				t.Fatal("unqualified control proof accepted")
			}
		})
	}
}

func TestProductionRunnerDoesNotExposeStdinSecretInOwnerDiagnostics(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "owner-cli")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\ncat >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	const secret = "test-only recovery secret that must stay on stdin"
	_, err := (Commands{}).runner()(context.Background(), tool, []string{"wallet", "restore"}, []byte(secret))
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("runner error leaked secret or succeeded: %v", err)
	}
}
