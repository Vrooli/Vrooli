package walletrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
)

const testRotationTombstone = `{"version":1,"epoch":2,"wallet_id":"old-wallet","network":"eip155:84532","address":"0xold","public_key":"public","successor_id":"new-wallet","revoked_at":"2026-11-15T12:00:00Z","signature":"0xsignature"}`

func testCommitAcknowledgementJSON(t *testing.T, reservationID, targetID, destinationID, snapshotID, tombstoneSHA256, successorID, successorAddress string) string {
	t.Helper()
	value := map[string]any{
		"acknowledgement": receiptsigning.WalletRecoveryCommitAcknowledgement{Version: 1, OperationID: reservationID, ReservationID: reservationID, TargetID: targetID, DestinationID: destinationID, SnapshotID: snapshotID, TombstoneSHA256: tombstoneSHA256, SuccessorID: successorID, SuccessorAddress: successorAddress, SuccessorPublicKey: "fixture-successor-public-key"},
		"signature":       receiptsigning.SignatureEnvelope{Version: receiptsigning.EnvelopeVersionV1, Purpose: receiptsigning.PurposeWalletRecoveryCommitAcknowledgement, Algorithm: receiptsigning.AlgorithmCredentialAuthorityEd25519, KeyID: "treasury-fixture-key", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Signature: "fixture-signature"},
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestRotatePublishesExactTombstoneAndCommitsOnlyAfterNewestRestoreProof(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, "receipt.json")
	digest := sha256.Sum256([]byte(testRotationTombstone))
	hash := hex.EncodeToString(digest[:])
	var calls []string
	var commitSeen bool
	snapshotReads := 0
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		calls = append(calls, tool+" "+strings.Join(args, " "))
		if tool == "data-backup-manager" {
			if len(args) >= 2 && args[0] == "wallet-recovery" && args[1] == "attest" {
				now := time.Now().UTC()
				r := rotationReceipt{Version: 1, WalletID: "old-wallet", Network: "eip155:84532", Address: "0xold", SuccessorID: "new-wallet", DestinationID: "remote-repo", DestinationBackend: "s3", PlanID: "manual-plan", RunID: "run-9", TargetName: "treasury-wallet-recovery-old-wallet", SnapshotID: "snap-9", LatestSnapshotID: "snap-9", TombstoneSHA256: hash, RestoredRevocationSHA256: hash, DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: now.Format(time.RFC3339Nano), ReservationID: rotationReservationID("old-wallet", "new-wallet", "remote-repo"), ReservationExpiresAt: now.Add(5 * time.Minute).Format(time.RFC3339Nano)}
				response, _ := json.Marshal(map[string]any{"receipt": r, "attestation": map[string]string{"version": "vrooli.receipt-signature.v1", "purpose": string(receiptsigning.PurposeWalletRecoveryPublicationReceipt), "algorithm": "credential-authority-ed25519", "keyId": "fixture-key", "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "fixture"}})
				return response, nil
			}
			if len(args) >= 2 && args[0] == "wallet-recovery" && args[1] == "finish" {
				return []byte(`{"state":"ok"}`), nil
			}
			switch strings.Join(args[:2], " ") {
			case "destinations get":
				return []byte(`{"destination":{"id":"remote-repo","backend_kind":"BACKEND_KIND_S3"}}`), nil
			case "targets list":
				return []byte(`{"targets":[{"id":"treasury-target","owner":"treasury","name":"treasury-wallet-recovery-old-wallet","kind":"filesystem","locator":"/private/treasury/recovery"}]}`), nil
			case "plans get":
				return []byte(`{"plan":{"id":"manual-plan","target_ids":["treasury-target"],"destination_ids":["remote-repo"],"schedule":"","enabled":false}}`), nil
			case "runs preflight":
				return []byte(`{"ready":true}`), nil
			case "runs trigger":
				return []byte(`{"run":{"id":"run-9"}}`), nil
			case "runs get":
				return []byte(`{"run":{"id":"run-9","status":"RUN_STATUS_COMPLETED","outcomes":[{"target_id":"treasury-target","destination_id":"remote-repo","status":"TARGET_OUTCOME_STATUS_SUCCEEDED","snapshot_id":"snap-9"}]}}`), nil
			case "destinations snapshots":
				snapshotReads++
				return []byte(`{"snapshots":[{"id":"snap-9","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-old-wallet"},{"id":"snap-8","started_at":"2026-11-15T11:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-old-wallet"}]}`), nil
			case "targets register":
				return []byte(`{"target":{"id":"stage-target"}}`), nil
			case "restores verify":
				return []byte(`{"restore":{"id":"verify-1","target_id":"stage-target","destination_id":"remote-repo","snapshot_id":"snap-9","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFYING"}}`), nil
			case "restores get":
				if args[3] == "verify-1" {
					return []byte(`{"restore":{"id":"verify-1","target_id":"stage-target","destination_id":"remote-repo","snapshot_id":"snap-9","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
				}
				return []byte(`{"restore":{"id":"restore-1","target_id":"stage-target","destination_id":"remote-repo","snapshot_id":"snap-9","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
			case "restores restore":
				if err := os.WriteFile(filepath.Join(stage, "revocation.json"), []byte(testRotationTombstone), 0o600); err != nil {
					t.Fatal(err)
				}
				return []byte(`{"restore":{"id":"restore-1","target_id":"stage-target","destination_id":"remote-repo","snapshot_id":"snap-9","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORING"}}`), nil
			case "targets deregister":
				return []byte(`{"removed":true}`), nil
			}
		}
		if tool == "treasury" && strings.Contains(strings.Join(args, " "), "prepare-rotation") {
			response, _ := json.Marshal(map[string]any{
				"wallet":                    map[string]string{"id": "new-wallet", "network": "eip155:84532", "address": "0xnew"},
				"revocation_tombstone_json": testRotationTombstone, "tombstone_sha256": hash,
			})
			return response, nil
		}
		if tool == "treasury" && strings.Contains(strings.Join(args, " "), "commit-rotation") {
			commitSeen = true
			ack := testCommitAcknowledgementJSON(t, rotationReservationID("old-wallet", "new-wallet", "remote-repo"), "treasury-target", "remote-repo", "snap-9", hash, "new-wallet", "0xnew")
			response, _ := json.Marshal(map[string]any{"wallet": map[string]string{"id": "new-wallet", "network": "eip155:84532", "address": "0xnew"}, "commit_ack_json": ack})
			return response, nil
		}
		return []byte(`{}`), nil
	}
	var output strings.Builder
	commands := Commands{Out: &output, Runner: runner}
	args := []string{"--wallet-id", "old-wallet", "--network", "eip155:84532", "--successor-id", "new-wallet", "--destination-id", "remote-repo", "--plan-id", "manual-plan", "--staging-dir", stage, "--receipt-file", receiptPath}
	if err := commands.Rotate(args); err != nil {
		t.Fatal(err)
	}
	if !commitSeen || snapshotReads != 2 || !strings.Contains(output.String(), "snap-9") {
		t.Fatalf("rotation did not recheck and commit the verified newest snapshot: reads=%d output=%q calls=%v", snapshotReads, output.String(), calls)
	}
	commitPin, treasuryCommit, release := -1, -1, -1
	for i, call := range calls {
		switch {
		case strings.Contains(call, "wallet-recovery finish") && strings.Contains(call, "--state commit"):
			commitPin = i
		case strings.Contains(call, "treasury wallet commit-rotation"):
			treasuryCommit = i
		case strings.Contains(call, "wallet-recovery finish") && strings.Contains(call, "--state release"):
			release = i
		}
	}
	if commitPin < 0 || treasuryCommit <= commitPin || release <= treasuryCommit || !strings.Contains(output.String(), "released the reservation") {
		t.Fatalf("reservation must pin before Treasury and release only after its exact signed acknowledgement: calls=%v output=%q", calls, output.String())
	}
	receiptBytes, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt rotationReceipt
	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.SnapshotID != "snap-9" || receipt.LatestSnapshotID != "snap-9" || receipt.TombstoneSHA256 != hash || receipt.RestoredRevocationSHA256 != hash || receipt.BundlePresent {
		t.Fatalf("receipt does not bind the newest exact tombstone: %+v", receipt)
	}
	if strings.Contains(strings.Join(calls, "\n"), "--snapshot snap-8") {
		t.Fatalf("stale snapshot was used: %v", calls)
	}
}

func TestRotateMissingOffHostDestinationLeavesTreasuryUntouched(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	calledTreasury := false
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		if tool == "treasury" {
			calledTreasury = true
		}
		if tool == "data-backup-manager" && args[0] == "destinations" {
			return []byte(`{"destination":{"id":"local","backend_kind":"BACKEND_KIND_FILESYSTEM"}}`), nil
		}
		return []byte(`{}`), nil
	}
	err := (Commands{Runner: runner}).Rotate([]string{"--wallet-id", "old", "--network", "eip155:84532", "--successor-id", "next", "--destination-id", "local", "--plan-id", "plan", "--staging-dir", stage, "--receipt-file", filepath.Join(root, "receipt.json")})
	if err == nil || !strings.Contains(err.Error(), "S3 or SFTP") || calledTreasury {
		t.Fatalf("missing off-host destination should refuse before Treasury prepare: err=%v treasury_called=%t", err, calledTreasury)
	}
}

func TestRotateResumesPinnedReceiptAfterRestartWithoutPublishingAnotherSnapshot(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	receiptPath := filepath.Join(root, "receipt.json")
	digest := sha256.Sum256([]byte(testRotationTombstone))
	hash := hex.EncodeToString(digest[:])
	reservationID := rotationReservationID("old-wallet", "new-wallet", "remote")
	saved := rotationReceipt{Version: 1, WalletID: "old-wallet", Network: "eip155:84532", Address: "0xold", SuccessorID: "new-wallet", DestinationID: "remote", DestinationBackend: "s3", PlanID: "plan", RunID: "run-9", TargetName: "treasury-wallet-recovery-old-wallet", SnapshotID: "snap-9", LatestSnapshotID: "snap-9", TombstoneSHA256: hash, RestoredRevocationSHA256: hash, DBMVerifyPassed: true, RestoredTombstoneMatches: true, VerifiedAt: time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano), ReservationID: reservationID, ReservationExpiresAt: time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano)}
	savedBytes, _ := json.Marshal(saved)
	if err := os.WriteFile(receiptPath, savedBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	var calls []string
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		call := tool + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if tool == "data-backup-manager" {
			if len(args) >= 2 && args[0] == "wallet-recovery" && args[1] == "attest" {
				now := time.Now().UTC()
				signedReceipt := saved
				signedReceipt.VerifiedAt = now.Format(time.RFC3339Nano)
				signedReceipt.ReservationExpiresAt = now.Add(5 * time.Minute).Format(time.RFC3339Nano)
				response, _ := json.Marshal(map[string]any{"receipt": signedReceipt, "attestation": map[string]string{"version": "vrooli.receipt-signature.v1", "purpose": string(receiptsigning.PurposeWalletRecoveryPublicationReceipt), "algorithm": "credential-authority-ed25519", "keyId": "fixture-key", "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "signature": "fixture"}})
				return response, nil
			}
			if len(args) >= 2 && args[0] == "wallet-recovery" && args[1] == "finish" {
				return []byte(`{"state":"ok"}`), nil
			}
			switch strings.Join(args[:2], " ") {
			case "destinations get":
				return []byte(`{"destination":{"id":"remote","backend_kind":"BACKEND_KIND_S3"}}`), nil
			case "targets list":
				return []byte(`{"targets":[{"id":"treasury-target","owner":"treasury","name":"treasury-wallet-recovery-old-wallet","kind":"filesystem","locator":"/source"}]}`), nil
			}
		}
		if tool == "treasury" && strings.Contains(call, "prepare-rotation") {
			response, _ := json.Marshal(map[string]any{"wallet": map[string]string{"id": "new-wallet", "network": "eip155:84532", "address": "0xnew"}, "revocation_tombstone_json": testRotationTombstone, "tombstone_sha256": hash})
			return response, nil
		}
		if tool == "treasury" && strings.Contains(call, "commit-rotation") {
			ack := testCommitAcknowledgementJSON(t, saved.ReservationID, "treasury-target", "remote", saved.SnapshotID, saved.TombstoneSHA256, "new-wallet", "0xnew")
			response, _ := json.Marshal(map[string]any{"wallet": map[string]string{"id": "new-wallet", "network": "eip155:84532", "address": "0xnew"}, "commit_ack_json": ack})
			return response, nil
		}
		return nil, fmt.Errorf("unexpected call: %s", call)
	}
	args := []string{"--wallet-id", "old-wallet", "--network", "eip155:84532", "--successor-id", "new-wallet", "--destination-id", "remote", "--plan-id", "plan", "--staging-dir", stage, "--receipt-file", receiptPath}
	if err := (Commands{Runner: runner}).Rotate(args); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(calls, "\n"), "runs trigger") || !strings.Contains(strings.Join(calls, "\n"), "--state commit") || !strings.Contains(strings.Join(calls, "\n"), "--state release") {
		t.Fatalf("restart should refresh the same pinned receipt and release only after Treasury acknowledgement: %v", calls)
	}
}

func TestRotateStaleSnapshotReadbackPreventsCommit(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(testRotationTombstone))
	hash := hex.EncodeToString(digest[:])
	calledCommit := false
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		if tool == "treasury" && strings.Contains(strings.Join(args, " "), "commit-rotation") {
			calledCommit = true
		}
		if tool == "data-backup-manager" {
			switch strings.Join(args[:2], " ") {
			case "destinations get":
				return []byte(`{"destination":{"id":"remote","backend_kind":"BACKEND_KIND_SFTP"}}`), nil
			case "targets list":
				return []byte(`{"targets":[{"id":"treasury-target","owner":"treasury","name":"treasury-wallet-recovery-old-wallet","kind":"filesystem","locator":"/source"}]}`), nil
			case "plans get":
				return []byte(`{"plan":{"id":"plan","target_ids":["treasury-target"],"destination_ids":["remote"],"schedule":"","enabled":false}}`), nil
			case "runs preflight":
				return []byte(`{"ready":true}`), nil
			case "runs trigger":
				return []byte(`{"run":{"id":"run-1"}}`), nil
			case "runs get":
				return []byte(`{"run":{"id":"run-1","status":"COMPLETED","outcomes":[{"target_id":"treasury-target","destination_id":"remote","status":"SUCCEEDED","snapshot_id":"snap-stale"}]}}`), nil
			case "destinations snapshots":
				return []byte(`{"snapshots":[{"id":"snap-stale","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-old-wallet"}]}`), nil
			case "targets register":
				return []byte(`{"target":{"id":"stage-target"}}`), nil
			case "restores verify":
				return []byte(`{"restore":{"id":"verify-1","target_id":"stage-target","destination_id":"remote","snapshot_id":"snap-stale","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
			case "restores restore":
				if err := os.WriteFile(filepath.Join(stage, "revocation.json"), []byte(`{"wallet_id":"old-wallet","successor_id":"other"}`), 0o600); err != nil {
					t.Fatal(err)
				}
				return []byte(`{"restore":{"id":"restore-1","target_id":"stage-target","destination_id":"remote","snapshot_id":"snap-stale","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
			case "targets deregister":
				return []byte(`{"removed":true}`), nil
			}
		}
		if tool == "treasury" && strings.Contains(strings.Join(args, " "), "prepare-rotation") {
			response, _ := json.Marshal(map[string]any{"wallet": map[string]string{"id": "new-wallet", "network": "eip155:84532", "address": "0xnew"}, "revocation_tombstone_json": testRotationTombstone, "tombstone_sha256": hash})
			return response, nil
		}
		return []byte(`{}`), nil
	}
	err := (Commands{Runner: runner}).Rotate([]string{"--wallet-id", "old-wallet", "--network", "eip155:84532", "--successor-id", "new-wallet", "--destination-id", "remote", "--plan-id", "plan", "--staging-dir", stage, "--receipt-file", filepath.Join(root, "receipt.json")})
	if err == nil || !strings.Contains(err.Error(), "exact staged Treasury tombstone") || calledCommit {
		t.Fatalf("stale snapshot should refuse commit: err=%v called_commit=%t", err, calledCommit)
	}
}

func TestRotateRejectsWhenANewerSnapshotAppearsAfterItsDBMRun(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(testRotationTombstone))
	hash := hex.EncodeToString(digest[:])
	var calledCommit, registeredScratch bool
	snapshotReads := 0
	runner := func(_ context.Context, tool string, args []string, _ []byte) ([]byte, error) {
		if tool == "treasury" && strings.Contains(strings.Join(args, " "), "commit-rotation") {
			calledCommit = true
		}
		if tool == "data-backup-manager" {
			switch strings.Join(args[:2], " ") {
			case "destinations get":
				return []byte(`{"destination":{"id":"remote","backend_kind":"BACKEND_KIND_S3"}}`), nil
			case "targets list":
				return []byte(`{"targets":[{"id":"treasury-target","owner":"treasury","name":"treasury-wallet-recovery-old-wallet","kind":"filesystem","locator":"/source"}]}`), nil
			case "plans get":
				return []byte(`{"plan":{"id":"plan","target_ids":["treasury-target"],"destination_ids":["remote"],"schedule":"","enabled":false}}`), nil
			case "runs preflight":
				return []byte(`{"ready":true}`), nil
			case "runs trigger":
				return []byte(`{"run":{"id":"run-1"}}`), nil
			case "runs get":
				return []byte(`{"run":{"id":"run-1","status":"RUN_STATUS_COMPLETED","outcomes":[{"target_id":"treasury-target","destination_id":"remote","status":"TARGET_OUTCOME_STATUS_SUCCEEDED","snapshot_id":"snap-from-run"}]}}`), nil
			case "destinations snapshots":
				snapshotReads++
				if snapshotReads == 1 {
					return []byte(`{"snapshots":[{"id":"snap-from-run","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-old-wallet"}]}`), nil
				}
				return []byte(`{"snapshots":[{"id":"snap-newer","started_at":"2026-11-15T12:01:00Z","source":"dbm://treasury/treasury-wallet-recovery-old-wallet"},{"id":"snap-from-run","started_at":"2026-11-15T12:00:00Z","source":"dbm://treasury/treasury-wallet-recovery-old-wallet"}]}`), nil
			case "targets register":
				registeredScratch = true
				return []byte(`{"target":{"id":"stage-target"}}`), nil
			case "restores verify":
				return []byte(`{"restore":{"id":"verify-1","target_id":"stage-target","destination_id":"remote","snapshot_id":"snap-from-run","mode":"RESTORE_MODE_VERIFY","status":"RESTORE_STATUS_VERIFIED"}}`), nil
			case "restores restore":
				if err := os.WriteFile(filepath.Join(stage, "revocation.json"), []byte(testRotationTombstone), 0o600); err != nil {
					t.Fatal(err)
				}
				return []byte(`{"restore":{"id":"restore-1","target_id":"stage-target","destination_id":"remote","snapshot_id":"snap-from-run","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
			case "targets deregister":
				return []byte(`{"removed":true}`), nil
			}
		}
		if tool == "treasury" && strings.Contains(strings.Join(args, " "), "prepare-rotation") {
			response, _ := json.Marshal(map[string]any{"wallet": map[string]string{"id": "new-wallet", "network": "eip155:84532", "address": "0xnew"}, "revocation_tombstone_json": testRotationTombstone, "tombstone_sha256": hash})
			return response, nil
		}
		return []byte(`{}`), nil
	}
	err := (Commands{Runner: runner}).Rotate([]string{"--wallet-id", "old-wallet", "--network", "eip155:84532", "--successor-id", "new-wallet", "--destination-id", "remote", "--plan-id", "plan", "--staging-dir", stage, "--receipt-file", filepath.Join(root, "receipt.json")})
	if err == nil || !strings.Contains(err.Error(), "superseded the verified tombstone") || !registeredScratch || snapshotReads != 2 || calledCommit {
		t.Fatalf("rotation must recheck and stop when a newer snapshot supersedes readback: err=%v scratch_registered=%t snapshot_reads=%d commit_called=%t", err, registeredScratch, snapshotReads, calledCommit)
	}
}

func TestWaitForRestoreRejectsAStatusBoundToDifferentSnapshot(t *testing.T) {
	initial := []byte(`{"restore":{"id":"restore-1","target_id":"stage-target","destination_id":"remote","snapshot_id":"snap-old","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORING"}}`)
	called := false
	runner := func(context.Context, string, []string, []byte) ([]byte, error) {
		called = true
		return []byte(`{"restore":{"id":"restore-1","target_id":"stage-target","destination_id":"remote","snapshot_id":"snap-new","mode":"RESTORE_MODE_RESTORE","status":"RESTORE_STATUS_RESTORED"}}`), nil
	}
	err := waitForRestore(context.Background(), runner, initial, "stage-target", "remote", "snap-old", "RESTORE_MODE_RESTORE", "RESTORE_STATUS_RESTORED")
	if err == nil || !strings.Contains(err.Error(), "changed mode, target, destination, or snapshot") || !called {
		t.Fatalf("DBM status for a different snapshot must be rejected: err=%v called=%t", err, called)
	}
}
