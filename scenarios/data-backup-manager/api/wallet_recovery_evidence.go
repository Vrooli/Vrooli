package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/plans"
	"data-backup-manager/internal/restores"
	"data-backup-manager/internal/runs"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/targets"
	"data-backup-manager/internal/walletrecovery"
)

type walletRecoveryRunReader interface {
	GetRun(context.Context, string) (runs.Run, error)
}
type walletRecoveryRestoreReader interface {
	ListRestores(context.Context, string, int) ([]restores.Restore, error)
}
type walletRecoveryPlanReader interface {
	Get(context.Context, string) (plans.Plan, error)
}
type walletRecoveryTargetReader interface {
	Get(context.Context, string) (targets.Target, error)
}
type walletRecoveryTargetListReader interface {
	List(context.Context, string, int) ([]targets.Target, error)
}
type walletRecoveryDestinationReader interface {
	GetDestination(context.Context, string) (destinations.Destination, error)
	ListTargetSnapshots(context.Context, string, string, string) ([]engine.Snapshot, error)
}
type walletRecoverySnapshotReader interface {
	BrowseSnapshot(context.Context, string, string, string) ([]engine.SnapshotEntry, error)
	SnapshotVerify(context.Context, string, string, int) error
	SnapshotRestore(context.Context, string, string, string) error
}
type walletRecoveryGenerationReader interface {
	Generation(context.Context, string, string) (int64, error)
}

// dbmWalletRecoveryEvidenceReader rebuilds attestation evidence from DBM's
// persisted run, Kopia inventory, target/destination catalog, terminal verify
// lifecycle, and a fresh scratch materialization. It accepts only a manual,
// disabled-plan run and refuses if the snapshot generation changes while it
// reads the evidence. It never reads bundle plaintext or any destination
// credential.
type dbmWalletRecoveryEvidenceReader struct {
	Runs         walletRecoveryRunReader
	Restores     walletRecoveryRestoreReader
	Plans        walletRecoveryPlanReader
	Targets      walletRecoveryTargetReader
	Destinations walletRecoveryDestinationReader
	Engine       walletRecoverySnapshotReader
	Generation   walletRecoveryGenerationReader
}

// dbmWalletRecoverySnapshotCatalog implements the read-lease compare step.
// Catalog reads are bracketed by the same generation counter incremented at
// mutation admission; the reservation store repeats that generation check
// transactionally before publishing the lease.
type dbmWalletRecoverySnapshotCatalog struct {
	Targets      walletRecoveryTargetListReader
	Destinations walletRecoveryDestinationReader
	Generation   walletRecoveryGenerationReader
}

func (r dbmWalletRecoverySnapshotCatalog) NewestRecoverySnapshot(ctx context.Context, targetName, destinationID string) (walletrecovery.SnapshotState, error) {
	if r.Targets == nil || r.Destinations == nil || r.Generation == nil {
		return walletrecovery.SnapshotState{}, errors.New("DBM recovery snapshot catalog is not configured")
	}
	registered, err := r.Targets.List(ctx, "treasury", 1000)
	if err != nil {
		return walletrecovery.SnapshotState{}, err
	}
	var target targets.Target
	for _, candidate := range registered {
		if candidate.Name == targetName {
			if target.ID != "" {
				return walletrecovery.SnapshotState{}, errors.New("DBM recovery target name is ambiguous")
			}
			target = candidate
		}
	}
	if target.ID == "" || target.Owner != "treasury" || target.SourceKind != sources.KindFilesystem || !strings.HasPrefix(target.Name, "treasury-wallet-recovery-") {
		return walletrecovery.SnapshotState{}, errors.New("read lease target is not the registered Treasury wallet recovery target")
	}
	generationBefore, err := r.Generation.Generation(ctx, target.ID, destinationID)
	if err != nil {
		return walletrecovery.SnapshotState{}, err
	}
	destination, err := r.Destinations.GetDestination(ctx, destinationID)
	if err != nil || destination.ID != destinationID || (destination.BackendKind != destinations.BackendS3 && destination.BackendKind != destinations.BackendSFTP) {
		return walletrecovery.SnapshotState{}, errors.New("read lease requires the exact registered off-host recovery destination")
	}
	snapshots, err := r.Destinations.ListTargetSnapshots(ctx, destinationID, target.Owner, target.Name)
	if err != nil {
		return walletrecovery.SnapshotState{}, err
	}
	if len(snapshots) == 0 || strings.TrimSpace(snapshots[0].ID) == "" {
		return walletrecovery.SnapshotState{}, errors.New("off-host recovery target has no identifiable snapshots")
	}
	if len(snapshots) > 1 {
		first, e1 := time.Parse(time.RFC3339Nano, snapshots[0].StartTime)
		second, e2 := time.Parse(time.RFC3339Nano, snapshots[1].StartTime)
		if e1 != nil || e2 != nil || first.Equal(second) {
			return walletrecovery.SnapshotState{}, errors.New("off-host recovery catalog has an ambiguous newest snapshot")
		}
	}
	generationAfter, err := r.Generation.Generation(ctx, target.ID, destinationID)
	if err != nil {
		return walletrecovery.SnapshotState{}, err
	}
	if generationBefore != generationAfter {
		return walletrecovery.SnapshotState{}, errors.New("snapshot mutation overlapped recovery catalog read; retry lease acquisition")
	}
	return walletrecovery.SnapshotState{ID: snapshots[0].ID, TargetID: target.ID, Generation: generationAfter}, nil
}

func (r dbmWalletRecoveryEvidenceReader) ReadWalletRecoveryEvidence(ctx context.Context, requestID, targetID, destinationID string) (walletrecovery.Evidence, error) {
	var out walletrecovery.Evidence
	if r.Runs == nil || r.Restores == nil || r.Plans == nil || r.Targets == nil || r.Destinations == nil || r.Engine == nil || r.Generation == nil {
		return out, errors.New("DBM wallet recovery evidence readers are incomplete")
	}
	if strings.TrimSpace(requestID) == "" || strings.TrimSpace(targetID) == "" || strings.TrimSpace(destinationID) == "" {
		return out, errors.New("wallet recovery evidence request, target and destination are required")
	}
	generationBefore, err := r.Generation.Generation(ctx, targetID, destinationID)
	if err != nil {
		return out, fmt.Errorf("read snapshot generation before evidence collection: %w", err)
	}
	target, err := r.Targets.Get(ctx, targetID)
	if err != nil || target.ID != targetID || target.Owner != "treasury" || target.SourceKind != sources.KindFilesystem || !strings.HasPrefix(target.Name, "treasury-wallet-recovery-") || strings.TrimPrefix(target.Name, "treasury-wallet-recovery-") == "" {
		return out, errors.New("target is not the registered Treasury wallet recovery filesystem target")
	}
	destination, err := r.Destinations.GetDestination(ctx, destinationID)
	if err != nil || destination.ID != destinationID || (destination.BackendKind != destinations.BackendS3 && destination.BackendKind != destinations.BackendSFTP) {
		return out, errors.New("wallet recovery attestation requires the selected off-host S3 or SFTP destination")
	}
	snapshots, err := r.Destinations.ListTargetSnapshots(ctx, destinationID, target.Owner, target.Name)
	if err != nil || len(snapshots) == 0 {
		return out, errors.New("DBM has no validated immutable snapshot for the exact Treasury target and destination")
	}
	snapshot := snapshots[0]
	runID := tagValue(snapshot.Tags, "dbm.run_id")
	if runID == "" || tagValue(snapshot.Tags, "dbm.target_id") != targetID || tagValue(snapshot.Tags, "dbm.destination_id") != destinationID {
		return out, errors.New("newest snapshot lacks exact DBM target, destination and run tags")
	}
	planID := ""
	run, err := r.Runs.GetRun(ctx, runID)
	if err != nil || run.Trigger != runs.TriggerManual || run.Status != runs.RunCompleted || run.ID != runID {
		return out, errors.New("newest snapshot is not owned by a successful terminal manual DBM run")
	}
	planID = run.PlanID
	plan, err := r.Plans.Get(ctx, planID)
	if err != nil || plan.ID != planID || plan.Enabled || strings.TrimSpace(plan.Schedule) != "" || !containsString(plan.TargetIDs, targetID) || !containsString(plan.DestinationIDs, destinationID) {
		return out, errors.New("snapshot's run is not from the exact disabled manual plan and target×destination")
	}
	var outcome runs.TargetOutcome
	foundOutcome := false
	for _, candidate := range run.Outcomes {
		if candidate.TargetID == targetID && candidate.DestinationID == destinationID {
			outcome, foundOutcome = candidate, true
			break
		}
	}
	if !foundOutcome || outcome.Status != runs.OutcomeSucceeded || outcome.SnapshotID != snapshot.ID {
		return out, errors.New("manual run outcome does not identify the newest immutable snapshot")
	}
	restoreRecords, err := r.Restores.ListRestores(ctx, targetID, 500)
	if err != nil {
		return out, fmt.Errorf("read terminal DBM restore records: %w", err)
	}
	verifiedRestore := restores.Restore{}
	for _, candidate := range restoreRecords {
		if candidate.TargetID == targetID && candidate.DestinationID == destinationID && candidate.SnapshotID == snapshot.ID && candidate.Mode == restores.ModeVerify && candidate.Status == restores.RestoreVerified && !candidate.FinishedAt.IsZero() {
			verifiedRestore = candidate
			break
		}
	}
	if verifiedRestore.ID == "" || !validEvidenceSHA256(verifiedRestore.Checksum) {
		return out, errors.New("newest snapshot lacks a terminal successful verify lifecycle record")
	}
	if snapshot.StatsTotalSize <= 0 || snapshot.StatsTotalSize > 2<<20 {
		return out, errors.New("wallet recovery snapshot exceeds the bounded evidence materialization size")
	}
	counts := []int64{snapshot.FileCount, snapshot.DirCount, snapshot.SymlinkCount}
	totalEntries := int64(0)
	boundedCounts := true
	for _, count := range counts {
		if count < 0 || count > maxWalletRecoveryEntries-totalEntries {
			boundedCounts = false
			break
		}
		totalEntries += count
	}
	if !boundedCounts {
		return out, errors.New("Kopia snapshot metadata is missing bounded file and directory counts")
	}
	entries, err := r.Engine.BrowseSnapshot(ctx, destination.Name, snapshot.ID, "")
	if err != nil {
		return out, fmt.Errorf("inspect immutable snapshot manifest before materialization: %w", err)
	}
	if err := validateWalletRecoveryManifest(entries); err != nil {
		return out, err
	}
	if err := matchWalletRecoveryManifestCounts(entries, snapshot); err != nil {
		return out, err
	}
	if err := r.Engine.SnapshotVerify(ctx, destination.Name, snapshot.ID, 100); err != nil {
		return out, fmt.Errorf("Kopia rejected the exact newest snapshot during independent verification: %w", err)
	}
	scratch, err := os.MkdirTemp("", "dbm-wallet-recovery-attest-")
	if err != nil {
		return out, fmt.Errorf("create private wallet recovery evidence directory: %w", err)
	}
	defer os.RemoveAll(scratch)
	if err := os.Chmod(scratch, 0o700); err != nil {
		return out, err
	}
	if err := r.Engine.SnapshotRestore(ctx, destination.Name, snapshot.ID, scratch); err != nil {
		return out, fmt.Errorf("materialize newest wallet recovery snapshot for attestation: %w", err)
	}
	materializedChecksum, err := restores.ChecksumDirectory(scratch)
	if err != nil || materializedChecksum != strings.ToLower(verifiedRestore.Checksum) {
		return out, errors.New("fresh snapshot materialization checksum does not match DBM terminal verify record")
	}
	tombstonePath := filepath.Join(scratch, "revocation.json")
	tombstoneInfo, err := os.Lstat(tombstonePath)
	if err != nil || !tombstoneInfo.Mode().IsRegular() || tombstoneInfo.Mode()&os.ModeSymlink != 0 || tombstoneInfo.Size() <= 0 || tombstoneInfo.Size() > 16<<10 {
		return out, errors.New("restored snapshot has no bounded regular Treasury revocation tombstone")
	}
	tombstone, err := os.ReadFile(tombstonePath)
	if err != nil || int64(len(tombstone)) != tombstoneInfo.Size() {
		return out, errors.New("restored Treasury tombstone changed while it was read")
	}
	var revocation struct {
		Version     int    `json:"version"`
		Epoch       uint64 `json:"epoch"`
		WalletID    string `json:"wallet_id"`
		Network     string `json:"network"`
		Address     string `json:"address"`
		SuccessorID string `json:"successor_wallet_id"`
		RevokedAt   string `json:"revoked_at"`
		Signature   string `json:"signature"`
	}
	if err := json.Unmarshal(tombstone, &revocation); err != nil || revocation.Version != 1 || revocation.Epoch < 2 || revocation.WalletID == "" || revocation.Network == "" || revocation.Address == "" || revocation.SuccessorID == "" || revocation.RevokedAt == "" || revocation.Signature == "" {
		return out, errors.New("restored Treasury revocation tombstone is incomplete or malformed")
	}
	if revocation.WalletID != strings.TrimPrefix(target.Name, "treasury-wallet-recovery-") {
		return out, errors.New("restored tombstone wallet identity does not match the Treasury target name")
	}
	bundlePath := filepath.Join(scratch, "wallet.bundle")
	if _, err := os.Lstat(bundlePath); err == nil {
		return out, errors.New("restored revocation snapshot still contains a live wallet bundle")
	} else if !errors.Is(err, os.ErrNotExist) {
		return out, fmt.Errorf("inspect restored wallet bundle: %w", err)
	}
	generationAfter, err := r.Generation.Generation(ctx, targetID, destinationID)
	if err != nil || generationAfter != generationBefore {
		return out, errors.New("target×destination snapshot generation changed during evidence collection")
	}
	latestAfter, err := r.Destinations.ListTargetSnapshots(ctx, destinationID, target.Owner, target.Name)
	if err != nil || len(latestAfter) == 0 || latestAfter[0].ID != snapshot.ID {
		return out, errors.New("newer snapshot superseded wallet recovery evidence during materialization")
	}
	digest := sha256.Sum256(tombstone)
	digestHex := hex.EncodeToString(digest[:])
	out = walletrecovery.Evidence{
		RequestID: requestID, TargetID: targetID, DestinationID: destinationID,
		WalletID: revocation.WalletID, Network: revocation.Network, Address: revocation.Address, SuccessorID: revocation.SuccessorID,
		DestinationBackend: string(destination.BackendKind), PlanID: planID, TargetName: target.Name,
		RunID: run.ID, RunTrigger: string(run.Trigger), RunStatus: string(run.Status),
		OutcomeTargetID: outcome.TargetID, OutcomeDestinationID: outcome.DestinationID, OutcomeStatus: string(outcome.Status), SnapshotID: snapshot.ID,
		SnapshotNewest: true, SnapshotImmutable: true, SnapshotGeneration: generationBefore,
		RestoreID: verifiedRestore.ID, RestoreStatus: "verified", RestoreSnapshotID: verifiedRestore.SnapshotID,
		RestoredTombstoneSHA256: digestHex, RestoredRevocationSHA256: digestHex, DBMVerifyPassed: true, RestoredTombstoneMatches: true,
		WalletBundlePresent: false,
	}
	return out, nil
}

const (
	maxWalletRecoveryEntries   = 128
	maxWalletRecoveryPathBytes = 128 << 10
	maxWalletRecoveryPathDepth = 32
	maxWalletRecoveryFileBytes = 2 << 20
)

func validEvidenceSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func matchWalletRecoveryManifestCounts(entries []engine.SnapshotEntry, snapshot engine.Snapshot) error {
	files, dirs, symlinks, bytes := int64(0), int64(0), int64(0), int64(0)
	for _, entry := range entries {
		switch entry.Type {
		case "f":
			files++
			bytes += entry.SizeBytes
		case "d":
			dirs++
		case "l":
			symlinks++
		default:
			return errors.New("wallet recovery snapshot contains an unsupported entry type")
		}
	}
	// Kopia's aggregate directory count includes the snapshot root itself;
	// Browse returns descendants only.
	if files != snapshot.FileCount || dirs+1 != snapshot.DirCount || symlinks != snapshot.SymlinkCount || bytes != snapshot.StatsTotalSize {
		return errors.New("complete Browse manifest counts or bytes disagree with the exact snapshot summary")
	}
	return nil
}

// The snapshot byte total alone does not bound inode or path overhead. Inspect
// Kopia's immutable manifest before extraction and cap all three dimensions.
func validateWalletRecoveryManifest(entries []engine.SnapshotEntry) error {
	if len(entries) == 0 || len(entries) > maxWalletRecoveryEntries {
		return errors.New("wallet recovery snapshot manifest exceeds the bounded entry count")
	}
	pathBytes, fileBytes := 0, int64(0)
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		name := strings.TrimPrefix(entry.Path, "./")
		clean := path.Clean(name)
		if name == "" || clean == "." || path.IsAbs(name) || clean != name || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsRune(name, '\\') {
			return errors.New("wallet recovery snapshot manifest contains an unsafe path")
		}
		if _, ok := seen[name]; ok {
			return errors.New("wallet recovery snapshot manifest contains duplicate paths")
		}
		seen[name] = struct{}{}
		pathBytes += len(name)
		if pathBytes > maxWalletRecoveryPathBytes || strings.Count(name, "/")+1 > maxWalletRecoveryPathDepth {
			return errors.New("wallet recovery snapshot manifest exceeds bounded path overhead")
		}
		if entry.SizeBytes < 0 {
			return errors.New("wallet recovery snapshot manifest has a negative entry size")
		}
		if entry.Type != "d" && entry.Type != "f" {
			return errors.New("wallet recovery snapshot manifest contains an unsupported non-regular entry")
		}
		if !entry.IsDir {
			fileBytes += entry.SizeBytes
			if fileBytes > maxWalletRecoveryFileBytes {
				return errors.New("wallet recovery snapshot manifest exceeds bounded logical file size")
			}
		}
	}
	return nil
}

func tagValue(tags []string, key string) string {
	prefix := key + ":"
	for _, tag := range tags {
		if strings.HasPrefix(tag, prefix) {
			return strings.TrimPrefix(tag, prefix)
		}
	}
	return ""
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
