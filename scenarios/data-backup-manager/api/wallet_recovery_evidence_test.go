package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

const evidenceTestTombstone = `{"version":1,"epoch":2,"wallet_id":"wallet","network":"eip155:84532","address":"0xold","public_key":"pub","successor_wallet_id":"successor","revoked_at":"2026-11-16T06:00:00Z","signature":"sig"}`

type walletEvidenceRuns struct{ run runs.Run }

func (f walletEvidenceRuns) GetRun(context.Context, string) (runs.Run, error) { return f.run, nil }

type walletEvidenceRestores struct{ rows []restores.Restore }

func (f walletEvidenceRestores) ListRestores(context.Context, string, int) ([]restores.Restore, error) {
	return f.rows, nil
}

type walletEvidencePlans struct{ plan plans.Plan }

func (f walletEvidencePlans) Get(context.Context, string) (plans.Plan, error) { return f.plan, nil }

type walletEvidenceTargets struct{ target targets.Target }

func (f walletEvidenceTargets) Get(context.Context, string) (targets.Target, error) {
	return f.target, nil
}
func (f walletEvidenceTargets) List(_ context.Context, owner string, _ int) ([]targets.Target, error) {
	if f.target.Owner != owner {
		return nil, nil
	}
	return []targets.Target{f.target}, nil
}

type walletEvidenceDestinations struct {
	destination destinations.Destination
	snapshots   []engine.Snapshot
	second      []engine.Snapshot
	reads       int
}

func (f *walletEvidenceDestinations) GetDestination(context.Context, string) (destinations.Destination, error) {
	return f.destination, nil
}
func (f *walletEvidenceDestinations) ListTargetSnapshots(context.Context, string, string, string) ([]engine.Snapshot, error) {
	f.reads++
	if f.reads > 1 && f.second != nil {
		return f.second, nil
	}
	return f.snapshots, nil
}

func TestDBMWalletRecoverySnapshotCatalogBracketsNewestWithGeneration(t *testing.T) {
	target := walletEvidenceTargets{target: targets.Target{ID: "target", Owner: "treasury", Name: "treasury-wallet-recovery-wallet", SourceKind: sources.KindFilesystem}}
	destination := &walletEvidenceDestinations{
		destination: destinations.Destination{ID: "dest", BackendKind: destinations.BackendS3},
		snapshots: []engine.Snapshot{
			{ID: "newest", Source: "dbm://treasury/treasury-wallet-recovery-wallet", StartTime: "2026-11-16T12:00:00Z"},
			{ID: "older", Source: "dbm://treasury/treasury-wallet-recovery-wallet", StartTime: "2026-11-16T11:00:00Z"},
		},
	}
	generation := &walletEvidenceGeneration{values: []int64{8, 8}}
	catalog := dbmWalletRecoverySnapshotCatalog{Targets: target, Destinations: destination, Generation: generation}
	got, err := catalog.NewestRecoverySnapshot(context.Background(), "treasury-wallet-recovery-wallet", "dest")
	if err != nil || got.ID != "newest" || got.Generation != 8 || generation.reads != 2 {
		t.Fatalf("stable catalog = %+v err=%v generation reads=%d", got, err, generation.reads)
	}

	destination.reads = 0
	generation = &walletEvidenceGeneration{values: []int64{8, 9}}
	catalog.Generation = generation
	if _, err := catalog.NewestRecoverySnapshot(context.Background(), "treasury-wallet-recovery-wallet", "dest"); err == nil || !strings.Contains(err.Error(), "overlapped") {
		t.Fatalf("catalog mutation overlap accepted: %v", err)
	}
}

type walletEvidenceEngine struct {
	calls       int
	browseCalls int
	entries     []engine.SnapshotEntry
}

func (f *walletEvidenceEngine) BrowseSnapshot(context.Context, string, string, string) ([]engine.SnapshotEntry, error) {
	f.browseCalls++
	return f.entries, nil
}

func (f *walletEvidenceEngine) SnapshotVerify(context.Context, string, string, int) error {
	f.calls++
	return nil
}
func (f *walletEvidenceEngine) SnapshotRestore(_ context.Context, _, _, target string) error {
	f.calls++
	return os.WriteFile(filepath.Join(target, "revocation.json"), []byte(evidenceTestTombstone), 0o600)
}

type walletEvidenceGeneration struct {
	values []int64
	reads  int
}

func (f *walletEvidenceGeneration) Generation(context.Context, string, string) (int64, error) {
	if len(f.values) == 0 {
		return 0, errors.New("missing generation fixture")
	}
	value := f.values[0]
	f.reads++
	if len(f.values) > 1 {
		f.values = f.values[1:]
	}
	return value, nil
}

func validWalletRecoveryEvidenceReader(t *testing.T) (dbmWalletRecoveryEvidenceReader, *walletEvidenceDestinations, *walletEvidenceEngine) {
	const targetID, destinationID, runID, snapshotID, planID = "target", "destination", "run", "snapshot", "manual-plan"
	now := time.Date(2026, 11, 16, 6, 0, 0, 0, time.UTC)
	dest := &walletEvidenceDestinations{
		destination: destinations.Destination{ID: destinationID, Name: "offhost", BackendKind: destinations.BackendS3},
		snapshots:   []engine.Snapshot{{ID: snapshotID, Source: "dbm://treasury/treasury-wallet-recovery-wallet", StartTime: now.Format(time.RFC3339Nano), SizeBytes: 512, StatsTotalSize: int64(len(evidenceTestTombstone)), FileCount: 1, DirCount: 1, SymlinkCount: 0, Tags: []string{"dbm.owner:treasury", "dbm.name:treasury-wallet-recovery-wallet", "dbm.target_id:" + targetID, "dbm.destination_id:" + destinationID, "dbm.run_id:" + runID}}},
	}
	engineReader := &walletEvidenceEngine{entries: []engine.SnapshotEntry{{Path: "revocation.json", SizeBytes: int64(len(evidenceTestTombstone)), Type: "f"}}}
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "revocation.json"), []byte(evidenceTestTombstone), 0o600); err != nil {
		t.Fatal(err)
	}
	checksum, err := restores.ChecksumDirectory(tmp)
	if err != nil {
		t.Fatal(err)
	}
	reader := dbmWalletRecoveryEvidenceReader{
		Runs:         walletEvidenceRuns{run: runs.Run{ID: runID, PlanID: planID, Trigger: runs.TriggerManual, Status: runs.RunCompleted, Outcomes: []runs.TargetOutcome{{TargetID: targetID, DestinationID: destinationID, Status: runs.OutcomeSucceeded, SnapshotID: snapshotID}}}},
		Restores:     walletEvidenceRestores{rows: []restores.Restore{{ID: "restore", TargetID: targetID, DestinationID: destinationID, SnapshotID: snapshotID, Mode: restores.ModeVerify, Status: restores.RestoreVerified, Checksum: checksum, FinishedAt: now}}},
		Plans:        walletEvidencePlans{plan: plans.Plan{ID: planID, TargetIDs: []string{targetID}, DestinationIDs: []string{destinationID}}},
		Targets:      walletEvidenceTargets{target: targets.Target{ID: targetID, Owner: "treasury", Name: "treasury-wallet-recovery-wallet", SourceKind: sources.KindFilesystem}},
		Destinations: dest, Engine: engineReader, Generation: &walletEvidenceGeneration{values: []int64{4, 4}},
	}
	return reader, dest, engineReader
}

func TestDBMWalletRecoveryEvidenceRejectsMismatchedCatalogIDsAndChecksum(t *testing.T) {
	t.Run("target id mismatch", func(t *testing.T) {
		reader, _, kopia := validWalletRecoveryEvidenceReader(t)
		reader.Targets = walletEvidenceTargets{target: targets.Target{ID: "other", Owner: "treasury", Name: "treasury-wallet-recovery-wallet", SourceKind: sources.KindFilesystem}}
		if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
			t.Fatal("mismatched target ID must refuse")
		}
		if kopia.calls != 0 {
			t.Fatal("mismatched target must refuse before Kopia calls")
		}
	})
	t.Run("destination id mismatch", func(t *testing.T) {
		reader, _, kopia := validWalletRecoveryEvidenceReader(t)
		reader.Destinations = &walletEvidenceDestinations{destination: destinations.Destination{ID: "other", Name: "offhost", BackendKind: destinations.BackendS3}, snapshots: []engine.Snapshot{{ID: "snapshot"}}}
		if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
			t.Fatal("mismatched destination ID must refuse")
		}
		if kopia.calls != 0 {
			t.Fatal("mismatched destination must refuse before Kopia calls")
		}
	})
	t.Run("materialized checksum mismatch", func(t *testing.T) {
		reader, _, kopia := validWalletRecoveryEvidenceReader(t)
		reader.Restores = walletEvidenceRestores{rows: []restores.Restore{{ID: "restore", TargetID: "target", DestinationID: "destination", SnapshotID: "snapshot", Mode: restores.ModeVerify, Status: restores.RestoreVerified, Checksum: strings.Repeat("0", 64), FinishedAt: time.Now()}}}
		if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
			t.Fatal("different content checksum must refuse")
		}
		if kopia.calls != 2 {
			t.Fatalf("expected verify and restore before comparing digest, got %d calls", kopia.calls)
		}
	})
}

func TestDBMWalletRecoveryEvidenceBoundsManifestBeforeRestore(t *testing.T) {
	reader, destination, kopia := validWalletRecoveryEvidenceReader(t)
	destination.snapshots[0].FileCount = maxWalletRecoveryEntries
	destination.snapshots[0].DirCount = 1
	if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
		t.Fatal("high-file-count snapshot must refuse")
	}
	if kopia.calls != 0 || kopia.browseCalls != 0 {
		t.Fatalf("aggregate counts must refuse before browse, verify or extraction, calls=%d browse=%d", kopia.calls, kopia.browseCalls)
	}
}

func TestDBMWalletRecoveryEvidenceBoundsSymlinksBeforeBrowse(t *testing.T) {
	reader, destination, kopia := validWalletRecoveryEvidenceReader(t)
	destination.snapshots[0].SymlinkCount = maxWalletRecoveryEntries + 1
	if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
		t.Fatal("high symlink-count snapshot must refuse")
	}
	if kopia.browseCalls != 0 {
		t.Fatalf("symlink aggregate count must refuse before browse, calls=%d", kopia.browseCalls)
	}
}

func TestDBMWalletRecoveryEvidenceRequiresExactSummaryCountsAndBytes(t *testing.T) {
	for name, mutate := range map[string]func(*engine.Snapshot){
		"file count":      func(s *engine.Snapshot) { s.FileCount++ },
		"directory count": func(s *engine.Snapshot) { s.DirCount++ },
		"symlink count":   func(s *engine.Snapshot) { s.SymlinkCount++ },
		"total bytes":     func(s *engine.Snapshot) { s.StatsTotalSize++ },
	} {
		t.Run(name, func(t *testing.T) {
			reader, destination, kopia := validWalletRecoveryEvidenceReader(t)
			mutate(&destination.snapshots[0])
			if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
				t.Fatal("summary mismatch must refuse")
			}
			if kopia.calls != 0 {
				t.Fatalf("summary mismatch must refuse before verify and restore, calls=%d", kopia.calls)
			}
		})
	}
}

func TestDBMWalletRecoveryEvidenceRequiresAuthoritativeStatsTotalSize(t *testing.T) {
	reader, destination, kopia := validWalletRecoveryEvidenceReader(t)
	destination.snapshots[0].StatsTotalSize = -1
	// A positive compatibility fallback must not authorize evidence.
	destination.snapshots[0].SizeBytes = 1
	if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
		t.Fatal("missing authoritative stats.totalSize must refuse")
	}
	if kopia.browseCalls != 0 {
		t.Fatalf("missing stats.totalSize must refuse before browse, calls=%d", kopia.browseCalls)
	}
}

func TestDBMWalletRecoveryEvidenceRejectsSpecialManifestEntries(t *testing.T) {
	reader, _, kopia := validWalletRecoveryEvidenceReader(t)
	kopia.entries = []engine.SnapshotEntry{{Path: "pipe", Type: "p"}}
	if _, err := reader.ReadWalletRecoveryEvidence(context.Background(), "req", "target", "destination"); err == nil {
		t.Fatal("special entries must refuse before extraction")
	}
	if kopia.calls != 0 {
		t.Fatalf("unsupported entry must refuse before verify/restore, calls=%d", kopia.calls)
	}
}

func TestDBMWalletRecoveryEvidenceRebuildsFromNewestManualRunAndMaterialization(t *testing.T) {
	reader, destination, engineReader := validWalletRecoveryEvidenceReader(t)
	requestID := walletrecovery.ReservationID("wallet", "successor", "destination")
	e, err := reader.ReadWalletRecoveryEvidence(context.Background(), requestID, "target", "destination")
	if err != nil {
		t.Fatal(err)
	}
	if e.RequestID != requestID || e.SnapshotID != "snapshot" || e.RunID != "run" || e.RestoreID != "restore" || e.WalletID != "wallet" || e.SuccessorID != "successor" || e.RestoredTombstoneSHA256 == "" || !e.SnapshotNewest || !e.SnapshotImmutable || !e.DBMVerifyPassed || !e.RestoredTombstoneMatches || e.WalletBundlePresent {
		t.Fatalf("evidence did not bind expected owner state: %+v", e)
	}
	if destination.reads != 2 || engineReader.calls != 2 {
		t.Fatalf("expected newest-snapshot recheck and independent verify+restore; reads=%d engine_calls=%d", destination.reads, engineReader.calls)
	}
}

func TestDBMWalletRecoveryEvidenceRefusesConcurrentSnapshotAndScheduledPlan(t *testing.T) {
	t.Run("generation changes during materialization", func(t *testing.T) {
		reader, _, _ := validWalletRecoveryEvidenceReader(t)
		reader.Generation = &walletEvidenceGeneration{values: []int64{4, 5}}
		_, err := reader.ReadWalletRecoveryEvidence(context.Background(), walletrecovery.ReservationID("wallet", "successor", "destination"), "target", "destination")
		if err == nil {
			t.Fatal("generation change should refuse the receipt")
		}
	})
	t.Run("scheduled plan", func(t *testing.T) {
		reader, _, _ := validWalletRecoveryEvidenceReader(t)
		reader.Plans = walletEvidencePlans{plan: plans.Plan{ID: "manual-plan", TargetIDs: []string{"target"}, DestinationIDs: []string{"destination"}, Schedule: "0 * * * *"}}
		_, err := reader.ReadWalletRecoveryEvidence(context.Background(), walletrecovery.ReservationID("wallet", "successor", "destination"), "target", "destination")
		if err == nil {
			t.Fatal("scheduled plan must not produce rotation attestation")
		}
	})
	t.Run("newer snapshot appears during restore", func(t *testing.T) {
		reader, destination, _ := validWalletRecoveryEvidenceReader(t)
		destination.second = []engine.Snapshot{{ID: "newer", Source: "dbm://treasury/treasury-wallet-recovery-wallet", StartTime: "2026-11-16T06:01:00Z", SizeBytes: 512, StatsTotalSize: int64(len(evidenceTestTombstone)), FileCount: 1, DirCount: 1, SymlinkCount: 0, Tags: []string{"dbm.owner:treasury", "dbm.name:treasury-wallet-recovery-wallet", "dbm.target_id:target", "dbm.destination_id:destination", "dbm.run_id:another-run"}}}
		_, err := reader.ReadWalletRecoveryEvidence(context.Background(), walletrecovery.ReservationID("wallet", "successor", "destination"), "target", "destination")
		if err == nil {
			t.Fatal("newer snapshot must prevent signing stale evidence")
		}
	})
}
