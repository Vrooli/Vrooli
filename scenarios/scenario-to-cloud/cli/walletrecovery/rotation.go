package walletrecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/receiptsigning"
)

type rotationReceipt struct {
	Version                  int                               `json:"version"`
	WalletID                 string                            `json:"wallet_id"`
	Network                  string                            `json:"network"`
	Address                  string                            `json:"address"`
	SuccessorID              string                            `json:"successor_id"`
	DestinationID            string                            `json:"destination_id"`
	DestinationBackend       string                            `json:"destination_backend"`
	PlanID                   string                            `json:"plan_id"`
	RunID                    string                            `json:"run_id"`
	TargetName               string                            `json:"target_name"`
	SnapshotID               string                            `json:"snapshot_id"`
	LatestSnapshotID         string                            `json:"latest_snapshot_id"`
	TombstoneSHA256          string                            `json:"tombstone_sha256"`
	RestoredRevocationSHA256 string                            `json:"restored_revocation_sha256"`
	BundlePresent            bool                              `json:"bundle_present"`
	DBMVerifyPassed          bool                              `json:"dbm_verify_passed"`
	RestoredTombstoneMatches bool                              `json:"restored_tombstone_matches"`
	VerifiedAt               string                            `json:"verified_at"`
	ReservationID            string                            `json:"reservation_id"`
	ReservationExpiresAt     string                            `json:"reservation_expires_at"`
	Attestation              *receiptsigning.SignatureEnvelope `json:"attestation,omitempty"`
}

func (c Commands) rotate(args []string) error {
	fs := flag.NewFlagSet("wallet-recovery rotate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("wallet-id", "", "current Treasury root wallet id")
	network := fs.String("network", "", "Treasury wallet network")
	successorID := fs.String("successor-id", "", "stable id for the successor wallet; reuse it on retry")
	destinationID := fs.String("destination-id", "", "qualified existing off-host DBM destination id")
	planID := fs.String("plan-id", "", "manual DBM plan for this wallet target and destination")
	stage := fs.String("staging-dir", "", "new private empty directory for verified tombstone readback")
	receiptFile := fs.String("receipt-file", "", "new private output path for the DBM publication receipt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for name, value := range map[string]string{"wallet-id": *id, "network": *network, "successor-id": *successorID, "destination-id": *destinationID, "plan-id": *planID, "staging-dir": *stage, "receipt-file": *receiptFile} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("--%s is required", name)
		}
	}
	if !walletIDPattern.MatchString(*id) || !walletIDPattern.MatchString(*successorID) || *id == *successorID {
		return errors.New("wallet ids must use letters, digits, underscore or hyphen, and the successor must differ from the source")
	}
	stageDir, err := prepareStage(*stage)
	if err != nil {
		return err
	}
	receiptPath, resumeReceipt, err := prepareRotationReceiptPath(*receiptFile, stageDir)
	if err != nil {
		return err
	}
	targetName := "treasury-wallet-recovery-" + *id
	runner := c.runner()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Read-only qualification happens before Treasury stages the successor or
	// writes a tombstone. A local, unresolved, scheduled, or misbound plan is a
	// refusal with the old root still active.
	destination, err := runner(ctx, "data-backup-manager", []string{"destinations", "get", *destinationID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("read selected DBM destination: %w", err)
	}
	backend, err := offHostBackend(destination, *destinationID)
	if err != nil {
		return err
	}
	targetsOut, err := runner(ctx, "data-backup-manager", []string{"targets", "list", "--owner", "treasury", "--json"}, nil)
	if err != nil {
		return fmt.Errorf("read Treasury recovery target: %w", err)
	}
	targetID, err := stableTreasuryTargetID(targetsOut, targetName)
	if err != nil {
		return err
	}
	if resumeReceipt != nil {
		return c.resumePinnedRotation(ctx, runner, receiptPath, targetID, *id, *network, *successorID, *destinationID, backend, targetName, *planID, *resumeReceipt)
	}
	planOut, err := runner(ctx, "data-backup-manager", []string{"plans", "get", *planID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("read selected DBM plan: %w", err)
	}
	if err := verifyRotationPlan(planOut, *planID, targetID, *destinationID); err != nil {
		return err
	}
	preflight, err := runner(ctx, "data-backup-manager", []string{"runs", "preflight", "--plan", *planID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("DBM refused the rotation plan preflight: %w", err)
	}
	if err := readyPreflight(preflight); err != nil {
		return err
	}

	preparedOut, err := runner(ctx, "treasury", []string{"wallet", "prepare-rotation", "--id", *id, "--network", *network, "--successor-id", *successorID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("Treasury could not prepare the rotation; no scalar was deleted: %w", err)
	}
	prepared, err := parseRotationPreparation(preparedOut)
	if err != nil {
		return err
	}
	if prepared.Wallet.ID != *successorID || prepared.Wallet.Network != *network || prepared.Wallet.Address == "" {
		return errors.New("Treasury preparation returned a different successor wallet")
	}
	sha := sha256.Sum256([]byte(prepared.Tombstone))
	tombstoneHash := hex.EncodeToString(sha[:])
	if prepared.TombstoneSHA256 != tombstoneHash {
		return errors.New("Treasury preparation tombstone digest did not match its exact bytes")
	}
	var tombstone struct {
		WalletID    string `json:"wallet_id"`
		Network     string `json:"network"`
		Address     string `json:"address"`
		SuccessorID string `json:"successor_id"`
	}
	if err := json.Unmarshal([]byte(prepared.Tombstone), &tombstone); err != nil || tombstone.WalletID != *id || tombstone.Network != *network || tombstone.Address == "" || tombstone.SuccessorID != *successorID {
		return errors.New("Treasury preparation returned a tombstone with a different wallet lineage")
	}

	triggerOut, err := runner(ctx, "data-backup-manager", []string{"runs", "trigger", "--plan", *planID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("start DBM publication of the exact staged tombstone: %w", err)
	}
	runID, err := triggeredRunID(triggerOut)
	if err != nil {
		return err
	}
	run, err := waitForRun(ctx, runner, runID)
	if err != nil {
		return err
	}
	snapshotID, err := successfulSnapshot(run, targetID, *destinationID)
	if err != nil {
		return err
	}
	snapshots, err := runner(ctx, "data-backup-manager", []string{"destinations", "snapshots", "--id", *destinationID, "--owner", "treasury", "--name", targetName, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("read newest off-host Treasury snapshot: %w", err)
	}
	newestID, err := latestSnapshotID(snapshots, targetName)
	if err != nil {
		return err
	}
	if snapshotID != newestID {
		return errors.New("the newest DBM snapshot is not the snapshot produced by this rotation attempt; Treasury remains unrotated")
	}

	stageName := fmt.Sprintf("wallet-rotation-verify-%s-%d", *id, time.Now().UnixNano())
	registered := false
	defer func() {
		if registered {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_, _ = runner(cleanupCtx, "data-backup-manager", []string{"targets", "deregister", "--owner", "scenario-to-cloud", "--name", stageName, "--json"}, nil)
		}
	}()
	stageOut, err := runner(ctx, "data-backup-manager", []string{"targets", "register", "--owner", "scenario-to-cloud", "--name", stageName, "--kind", "filesystem", "--locator", stageDir, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("register private tombstone verification stage: %w", err)
	}
	stageTargetID, err := parseID(stageOut)
	if err != nil {
		return err
	}
	registered = true
	verifiedOut, verifyErr := runner(ctx, "data-backup-manager", []string{"restores", "verify", "--target", stageTargetID, "--destination", *destinationID, "--snapshot", newestID, "--json"}, nil)
	if verifyErr != nil {
		return fmt.Errorf("DBM could not verify the newest off-host snapshot: %w", verifyErr)
	}
	if err = waitForRestore(ctx, runner, verifiedOut, stageTargetID, *destinationID, newestID, "RESTORE_MODE_VERIFY", "RESTORE_STATUS_VERIFIED"); err != nil {
		return fmt.Errorf("DBM verification did not reach VERIFIED: %w", err)
	}
	restoredOut, restoreErr := runner(ctx, "data-backup-manager", []string{"restores", "restore", "--target", stageTargetID, "--destination", *destinationID, "--snapshot", newestID, "--location", stageDir, "--json"}, nil)
	if restoreErr != nil {
		return fmt.Errorf("restore newest tombstone snapshot to private verification stage: %w", restoreErr)
	}
	if err = waitForRestore(ctx, runner, restoredOut, stageTargetID, *destinationID, newestID, "RESTORE_MODE_RESTORE", "RESTORE_STATUS_RESTORED"); err != nil {
		return fmt.Errorf("DBM snapshot restore did not reach RESTORED: %w", err)
	}
	revocationPath := filepath.Join(stageDir, "revocation.json")
	revocationBytes, err := readPrivateFile(revocationPath, 16<<10)
	if err != nil || string(revocationBytes) != prepared.Tombstone {
		return errors.New("newest verified DBM snapshot does not contain the exact staged Treasury tombstone")
	}
	if _, err := os.Lstat(filepath.Join(stageDir, "wallet.bundle")); err == nil {
		return errors.New("newest DBM snapshot still contains a live wallet bundle; rotation commit refused")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect newest snapshot for stale wallet bundle: %w", err)
	}
	if _, err := runner(ctx, "data-backup-manager", []string{"targets", "deregister", "--owner", "scenario-to-cloud", "--name", stageName, "--json"}, nil); err != nil {
		return fmt.Errorf("verified snapshot is valid but the temporary DBM target could not be retired; Treasury was not committed: %w", err)
	}
	registered = false
	finalSnapshots, err := runner(ctx, "data-backup-manager", []string{"destinations", "snapshots", "--id", *destinationID, "--owner", "treasury", "--name", targetName, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("recheck newest off-host Treasury snapshot immediately before commit: %w", err)
	}
	finalNewestID, err := latestSnapshotID(finalSnapshots, targetName)
	if err != nil {
		return err
	}
	if finalNewestID != newestID {
		return errors.New("a newer DBM snapshot superseded the verified tombstone before commit; Treasury remains unrotated")
	}
	reservationID := rotationReservationID(*id, *successorID, *destinationID)
	attestedOut, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "attest", "--reservation-id", reservationID, "--target-id", targetID, "--destination-id", *destinationID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("DBM could not independently attest and reserve the restored newest snapshot: %w", err)
	}
	var signed struct {
		Receipt     rotationReceipt                  `json:"receipt"`
		Attestation receiptsigning.SignatureEnvelope `json:"attestation"`
	}
	if err := json.Unmarshal(attestedOut, &signed); err != nil {
		return fmt.Errorf("parse DBM-signed wallet recovery receipt: %w", err)
	}
	receipt := signed.Receipt
	receipt.Attestation = &signed.Attestation
	if err := validateAttestedRotationReceipt(receipt, *id, *network, tombstone.Address, *successorID, *destinationID, backend, *planID, runID, targetName, newestID, tombstoneHash, reservationID); err != nil {
		return err
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := writeNewJSON(receiptPath, receiptBytes); err != nil {
		return fmt.Errorf("persist private DBM rotation receipt: %w", err)
	}
	// Pin this exact target×destination snapshot before Treasury consumes the
	// signature. A commit_pending reservation survives process restart and
	// blocks newer snapshot mutations until Treasury confirms scalar deletion.
	if _, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "finish", "--reservation-id", reservationID, "--target-id", targetID, "--state", "commit", "--json"}, nil); err != nil {
		return fmt.Errorf("DBM could not durably pin the attested snapshot; Treasury remains uncommitted: %w", err)
	}
	committed, err := finishAttestedRotation(ctx, runner, receipt, targetID)
	if err != nil {
		return err
	}
	return c.write(fmt.Sprintf("Treasury confirmed rotation of root wallet %s to %s using DBM snapshot %s. DBM remains pinned pending a verifiable Treasury commit acknowledgement; private receipt: %s", *id, committed.Address, newestID, receiptPath))
}

func (c Commands) resumePinnedRotation(ctx context.Context, runner Runner, receiptPath, targetID, walletID, network, successorID, destinationID, backend, targetName, planID string, saved rotationReceipt) error {
	if saved.WalletID != walletID || saved.Network != network || saved.SuccessorID != successorID || saved.DestinationID != destinationID || saved.DestinationBackend != backend || saved.TargetName != targetName || saved.PlanID != planID || saved.ReservationID != rotationReservationID(walletID, successorID, destinationID) {
		return errors.New("existing rotation receipt belongs to a different wallet, successor, destination, plan, or reservation")
	}
	preparedOut, err := runner(ctx, "treasury", []string{"wallet", "prepare-rotation", "--id", walletID, "--network", network, "--successor-id", successorID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("Treasury could not resume the prepared rotation: %w", err)
	}
	prepared, err := parseRotationPreparation(preparedOut)
	if err != nil {
		return err
	}
	if prepared.Wallet.ID != successorID || prepared.Wallet.Network != network || prepared.Wallet.Address == "" || prepared.TombstoneSHA256 != saved.TombstoneSHA256 {
		return errors.New("Treasury prepared rotation does not match the saved DBM attestation")
	}
	reservationID := rotationReservationID(walletID, successorID, destinationID)
	attestedOut, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "attest", "--reservation-id", reservationID, "--target-id", targetID, "--destination-id", destinationID, "--json"}, nil)
	if err != nil {
		return fmt.Errorf("DBM could not refresh the same pinned snapshot attestation: %w", err)
	}
	var signed struct {
		Receipt     rotationReceipt                  `json:"receipt"`
		Attestation receiptsigning.SignatureEnvelope `json:"attestation"`
	}
	if err := json.Unmarshal(attestedOut, &signed); err != nil {
		return fmt.Errorf("parse refreshed DBM receipt: %w", err)
	}
	receipt := signed.Receipt
	receipt.Attestation = &signed.Attestation
	if err := validateAttestedRotationReceipt(receipt, walletID, network, saved.Address, successorID, destinationID, backend, planID, saved.RunID, targetName, saved.SnapshotID, saved.TombstoneSHA256, reservationID); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err := replacePrivateRotationReceipt(receiptPath, bytes); err != nil {
		return fmt.Errorf("persist refreshed signed rotation receipt: %w", err)
	}
	committed, err := finishAttestedRotation(ctx, runner, receipt, targetID)
	if err != nil {
		return err
	}
	return c.write(fmt.Sprintf("Treasury confirmed rotation of root wallet %s to %s using the same DBM snapshot %s. DBM remains pinned pending a verifiable Treasury commit acknowledgement; private receipt: %s", walletID, committed.Address, receipt.SnapshotID, receiptPath))
}

func finishAttestedRotation(ctx context.Context, runner Runner, receipt rotationReceipt, targetID string) (recoveredWallet, error) {
	reservationID := receipt.ReservationID
	if _, err := runner(ctx, "data-backup-manager", []string{"wallet-recovery", "finish", "--reservation-id", reservationID, "--target-id", targetID, "--state", "commit", "--json"}, nil); err != nil {
		return recoveredWallet{}, fmt.Errorf("DBM could not durably pin the attested snapshot; Treasury remains uncommitted: %w", err)
	}
	envelopeJSON, err := json.Marshal(receipt.Attestation)
	if err != nil {
		return recoveredWallet{}, err
	}
	commitArgs := []string{"wallet", "commit-rotation", "--id", receipt.WalletID, "--network", receipt.Network, "--successor-id", receipt.SuccessorID,
		"--address", receipt.Address, "--destination-id", receipt.DestinationID, "--destination-backend", receipt.DestinationBackend, "--plan-id", receipt.PlanID,
		"--run-id", receipt.RunID, "--target-name", receipt.TargetName, "--snapshot-id", receipt.SnapshotID,
		"--latest-snapshot-id", receipt.LatestSnapshotID, "--tombstone-sha256", receipt.TombstoneSHA256,
		"--restored-revocation-sha256", receipt.RestoredRevocationSHA256, "--verified-at", receipt.VerifiedAt,
		"--reservation-id", receipt.ReservationID, "--reservation-expires-at", receipt.ReservationExpiresAt,
		"--dbm-attestation-json", string(envelopeJSON),
		"--bundle-present=false", "--dbm-verify-passed=true", "--restored-tombstone-matches=true", "--json"}
	committedOut, err := runner(ctx, "treasury", commitArgs, nil)
	if err != nil {
		return recoveredWallet{}, fmt.Errorf("Treasury did not commit the rotation; retain the DBM pin and retry the same receipt: %w", err)
	}
	committed, err := restoredWallet(committedOut)
	if err != nil || committed.ID != receipt.SuccessorID || committed.Network != receipt.Network || committed.Address == "" {
		return recoveredWallet{}, errors.New("Treasury commit did not confirm the expected successor wallet; retain the DBM pin for reconciliation")
	}
	// Do not release commit_pending here. The current owner contract has no
	// authenticated Treasury-success acknowledgement that DBM can verify; an
	// operator CLI response is not proof. DBM remains pinned until that contract
	// is implemented and authorized.
	return committed, nil
}

func waitForRestore(ctx context.Context, runner Runner, initial []byte, targetID, destinationID, snapshotID, mode, wanted string) error {
	var response struct {
		Restore struct {
			ID            string `json:"id"`
			TargetID      string `json:"target_id"`
			DestinationID string `json:"destination_id"`
			Mode          string `json:"mode"`
			Status        string `json:"status"`
			SnapshotID    string `json:"snapshot_id"`
		} `json:"restore"`
	}
	if err := json.Unmarshal(initial, &response); err != nil || response.Restore.ID == "" {
		return errors.New("DBM restore response omitted its lifecycle record")
	}
	if response.Restore.TargetID != targetID || response.Restore.DestinationID != destinationID || response.Restore.SnapshotID != snapshotID || response.Restore.Mode != mode {
		return errors.New("DBM restore lifecycle is bound to a different mode, target, destination, or snapshot")
	}
	status := response.Restore.Status
	for {
		switch status {
		case wanted:
			return nil
		case "RESTORE_STATUS_FAILED":
			return errors.New("DBM restore lifecycle failed")
		case "", "RESTORE_STATUS_UNSPECIFIED":
			return errors.New("DBM restore lifecycle returned an unknown status")
		}
		if wanted == "RESTORE_STATUS_VERIFIED" && status == "RESTORE_STATUS_RESTORED" {
			return errors.New("DBM restore reached RESTORED when verification was required")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		data, err := runner(ctx, "data-backup-manager", []string{"restores", "get", "--id", response.Restore.ID, "--json"}, nil)
		if err != nil {
			return fmt.Errorf("read DBM restore lifecycle: %w", err)
		}
		if err := json.Unmarshal(data, &response); err != nil || response.Restore.ID == "" {
			return errors.New("DBM restore status response was malformed")
		}
		if response.Restore.TargetID != targetID || response.Restore.DestinationID != destinationID || response.Restore.SnapshotID != snapshotID || response.Restore.Mode != mode {
			return errors.New("DBM restore status changed mode, target, destination, or snapshot binding")
		}
		status = response.Restore.Status
	}
}

func rotationReservationID(walletID, successorID, destinationID string) string {
	digest := sha256.Sum256([]byte(walletID + "\x00" + successorID + "\x00" + destinationID))
	return "wallet-recovery-" + hex.EncodeToString(digest[:])
}

func prepareRotationReceiptPath(path, stage string) (string, *rotationReceipt, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	parent := filepath.Dir(abs)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return "", nil, errors.New("rotation receipt parent must be a real private directory with mode 0700 or stricter")
	}
	rel, err := filepath.Rel(stage, abs)
	if err != nil || rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", nil, errors.New("rotation receipt must be stored outside the DBM staging target")
	}
	fileInfo, err := os.Lstat(abs)
	if errors.Is(err, os.ErrNotExist) {
		return abs, nil, nil
	}
	if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Mode()&os.ModeSymlink != 0 || fileInfo.Mode().Perm()&0o077 != 0 || fileInfo.Size() <= 0 || fileInfo.Size() > 64<<10 {
		return "", nil, errors.New("existing rotation receipt must be a private regular file within the size limit")
	}
	data, err := os.ReadFile(abs)
	if err != nil || len(data) != int(fileInfo.Size()) {
		return "", nil, errors.New("existing rotation receipt changed while it was read")
	}
	var receipt rotationReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return "", nil, errors.New("existing rotation receipt is not valid rotation evidence")
	}
	return abs, &receipt, nil
}

func replacePrivateRotationReceipt(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".rotation-receipt-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func validateAttestedRotationReceipt(r rotationReceipt, walletID, network, address, successorID, destinationID, backend, planID, runID, targetName, snapshotID, tombstoneHash, reservationID string) error {
	if r.Version != 1 || r.WalletID != walletID || r.Network != network || r.Address != address || r.SuccessorID != successorID ||
		r.DestinationID != destinationID || r.DestinationBackend != backend || r.PlanID != planID || r.RunID != runID ||
		r.TargetName != targetName || r.SnapshotID != snapshotID || r.LatestSnapshotID != snapshotID ||
		r.TombstoneSHA256 != tombstoneHash || r.RestoredRevocationSHA256 != tombstoneHash || r.BundlePresent || !r.DBMVerifyPassed || !r.RestoredTombstoneMatches ||
		r.ReservationID != reservationID || r.Attestation == nil || r.Attestation.Purpose != receiptsigning.PurposeWalletRecoveryPublicationReceipt {
		return errors.New("DBM attestation does not bind the exact Treasury lineage, destination, run, newest snapshot, tombstone and reservation")
	}
	verifiedAt, err := time.Parse(time.RFC3339Nano, r.VerifiedAt)
	if err != nil {
		return errors.New("DBM attestation has an invalid verification time")
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, r.ReservationExpiresAt)
	if err != nil || !expiresAt.After(time.Now().UTC()) || expiresAt.Sub(verifiedAt) <= 0 || expiresAt.Sub(verifiedAt) > 10*time.Minute {
		return errors.New("DBM attestation reservation is expired or outside the bounded lease")
	}
	return nil
}

type preparedRotation struct {
	Wallet          recoveredWallet
	Tombstone       string
	TombstoneSHA256 string
}

func parseRotationPreparation(data []byte) (preparedRotation, error) {
	var response struct {
		Wallet                  recoveredWallet `json:"wallet"`
		RevocationTombstoneJSON string          `json:"revocation_tombstone_json"`
		TombstoneSHA256         string          `json:"tombstone_sha256"`
	}
	if err := json.Unmarshal(data, &response); err != nil || response.RevocationTombstoneJSON == "" {
		return preparedRotation{}, errors.New("Treasury rotation preparation omitted the signed tombstone")
	}
	return preparedRotation{Wallet: response.Wallet, Tombstone: response.RevocationTombstoneJSON, TombstoneSHA256: response.TombstoneSHA256}, nil
}

func offHostBackend(data []byte, expectedID string) (string, error) {
	var response struct {
		Destination struct {
			ID          string `json:"id"`
			BackendKind string `json:"backend_kind"`
		} `json:"destination"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", errors.New("DBM destination metadata is malformed")
	}
	if response.Destination.ID != expectedID {
		return "", errors.New("DBM destination response does not match the selected destination id")
	}
	kind := strings.ToLower(response.Destination.BackendKind)
	switch {
	case strings.Contains(kind, "sftp"):
		return "sftp", nil
	case strings.Contains(kind, "s3"):
		return "s3", nil
	default:
		return "", errors.New("wallet rotation requires an existing S3 or SFTP DBM destination")
	}
}

func stableTreasuryTargetID(data []byte, name string) (string, error) {
	var response struct {
		Targets []struct {
			ID      string `json:"id"`
			Owner   string `json:"owner"`
			Name    string `json:"name"`
			Kind    string `json:"kind"`
			Locator string `json:"locator"`
		} `json:"targets"`
	}
	if err := json.Unmarshal(data, &response); err != nil || response.Targets == nil {
		return "", errors.New("DBM target list is missing or malformed")
	}
	for _, target := range response.Targets {
		if target.Owner != "treasury" || target.Name != name {
			continue
		}
		if target.ID == "" || target.Kind != "filesystem" || strings.TrimSpace(target.Locator) == "" {
			return "", errors.New("Treasury recovery target has an invalid stable identity")
		}
		return target.ID, nil
	}
	return "", errors.New("the stable Treasury recovery target is not registered in DBM")
}

func verifyRotationPlan(data []byte, planID, targetID, destinationID string) error {
	var response struct {
		Plan struct {
			ID             string   `json:"id"`
			TargetIDs      []string `json:"target_ids"`
			DestinationIDs []string `json:"destination_ids"`
			Schedule       string   `json:"schedule"`
			Enabled        bool     `json:"enabled"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(data, &response); err != nil || response.Plan.ID == "" || response.Plan.TargetIDs == nil || response.Plan.DestinationIDs == nil {
		return errors.New("DBM rotation plan metadata is malformed")
	}
	if response.Plan.ID != planID || len(response.Plan.TargetIDs) != 1 || response.Plan.TargetIDs[0] != targetID || len(response.Plan.DestinationIDs) != 1 || response.Plan.DestinationIDs[0] != destinationID || response.Plan.Schedule != "" || response.Plan.Enabled {
		return errors.New("rotation plan must be manual-only, disabled, and bound only to this wallet target and selected destination")
	}
	return nil
}

func readyPreflight(data []byte) error {
	var response struct {
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return errors.New("DBM plan preflight response is malformed")
	}
	if !response.Ready {
		return errors.New("DBM plan is not ready; Treasury remains unrotated")
	}
	return nil
}

func triggeredRunID(data []byte) (string, error) {
	var response struct {
		Run struct {
			ID string `json:"id"`
		} `json:"run"`
	}
	if err := json.Unmarshal(data, &response); err != nil || strings.TrimSpace(response.Run.ID) == "" {
		return "", errors.New("DBM trigger response omitted its durable run id")
	}
	return response.Run.ID, nil
}

type runSnapshot struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Outcomes []struct {
		TargetID      string `json:"target_id"`
		DestinationID string `json:"destination_id"`
		Status        string `json:"status"`
		SnapshotID    string `json:"snapshot_id"`
	} `json:"outcomes"`
}

func waitForRun(ctx context.Context, runner Runner, id string) (runSnapshot, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		data, err := runner(ctx, "data-backup-manager", []string{"runs", "get", id, "--json"}, nil)
		if err != nil {
			return runSnapshot{}, fmt.Errorf("inspect DBM publication run %q: %w", id, err)
		}
		var response struct {
			Run runSnapshot `json:"run"`
		}
		if err := json.Unmarshal(data, &response); err != nil || response.Run.ID != id {
			return runSnapshot{}, errors.New("DBM returned a malformed publication run")
		}
		status := strings.ToUpper(response.Run.Status)
		if strings.Contains(status, "COMPLETED") || strings.Contains(status, "FAILED") || strings.Contains(status, "PARTIAL_FAILED") {
			return response.Run, nil
		}
		select {
		case <-ctx.Done():
			return runSnapshot{}, fmt.Errorf("wait for DBM publication run %q: %w", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func successfulSnapshot(run runSnapshot, targetID, destinationID string) (string, error) {
	for _, outcome := range run.Outcomes {
		if outcome.TargetID != targetID || outcome.DestinationID != destinationID {
			continue
		}
		if !strings.Contains(strings.ToUpper(outcome.Status), "SUCCEEDED") || strings.TrimSpace(outcome.SnapshotID) == "" {
			return "", errors.New("DBM did not create a successful snapshot for the selected Treasury target and off-host destination")
		}
		return outcome.SnapshotID, nil
	}
	return "", errors.New("DBM run omitted the selected Treasury target and destination outcome")
}

func readPrivateFile(path string, maxSize int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() <= 0 || info.Size() > maxSize {
		return nil, errors.New("restored tombstone is not a private regular file within its size limit")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) != int(info.Size()) {
		return nil, errors.New("restored tombstone changed while it was read")
	}
	return data, nil
}

func writeNewJSON(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	return closeErr
}
