package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	internalreleases "deployment-manager/internal/releases"

	_ "modernc.org/sqlite"
)

func TestSQLIdentityRepositoryRoundTripAndTamperDetection(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-identity-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ('release-identity-1', 'p1', 'commit-1', '1.0.0', 'stable')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	candidate := Candidate{
		SourceRevision:       "commit-1",
		ProfileRevision:      "profile-rev-1",
		BuildInputs:          map[string]string{"go": "1.25"},
		DependencyLockDigest: "sha256:lock",
		PolicyDigest:         "sha256:policy",
		Artifacts: []CandidateArtifact{{
			Target:          TargetIdentity{ID: "linux-x64", Platform: "desktop", OS: "linux", Architecture: "amd64", Format: "AppImage"},
			ImmutableRef:    "object://candidate/linux",
			Digest:          "sha256:artifact",
			SizeBytes:       42,
			SignatureDigest: "sha256:signature",
			SignerRef:       "authority://desktop",
		}},
	}
	storedCandidate, err := repo.RegisterCandidate(ctx, candidate)
	if err != nil {
		t.Fatalf("register candidate: %v", err)
	}
	if storedCandidate.ID == "" || storedCandidate.ArtifactManifestDigest == "" {
		t.Fatalf("candidate identity was not stored: %#v", storedCandidate)
	}
	replayed, err := repo.RegisterCandidate(ctx, candidate)
	if err != nil || replayed.ID != storedCandidate.ID {
		t.Fatalf("candidate replay = %#v, %v", replayed, err)
	}

	destination := DestinationRevision{Kind: "object-store", DestinationID: "staging", ConfigurationDigest: "sha256:destination", Channel: "stable", ExpectedChannelRevision: "rev-1"}
	storedDestination, err := repo.RegisterDestinationRevision(ctx, destination)
	if err != nil {
		t.Fatalf("register destination: %v", err)
	}
	if _, err := db.Exec(`UPDATE releases SET candidate_id = ?, destination_revision_id = ? WHERE id = ?`, storedCandidate.ID, storedDestination.ID, "release-identity-1"); err != nil {
		t.Fatalf("bind release identities: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO release_platforms (release_id, platform, status) VALUES (?, ?, ?)`, "release-identity-1", "linux-x64", PlatformStatusPending); err != nil {
		t.Fatalf("bind release target: %v", err)
	}
	binding := ReviewBinding{
		CandidateID:           storedCandidate.ID,
		DestinationRevisionID: storedDestination.ID,
		Targets:               []string{"linux-x64"},
		Channel:               "stable",
		EvidenceSetDigest:     "sha256:evidence",
		PolicyDigest:          "sha256:policy",
		AuthorizationEpoch:    3,
	}
	approvedAt := time.Now().UTC()
	review, err := repo.RegisterReview(ctx, "review-1", binding, "approved", &approvedAt)
	if err != nil {
		t.Fatalf("register review: %v", err)
	}
	if review.Binding.CandidateID != storedCandidate.ID || review.Status != "approved" || review.ApprovedAt == nil {
		t.Fatalf("review binding was not retained: %#v", review)
	}
	if replayedReview, err := repo.RegisterReview(ctx, "review-1", binding, "approved", &approvedAt); err != nil || replayedReview.ID != review.ID {
		t.Fatalf("idempotent review replay = %#v, %v", replayedReview, err)
	}
	conflictingBinding := binding
	conflictingBinding.EvidenceSetDigest = "sha256:other-evidence"
	if _, err := repo.RegisterReview(ctx, "review-1", conflictingBinding, "approved", &approvedAt); err == nil {
		t.Fatal("review identity conflict was silently accepted")
	}
	receipt := PublicationReceipt{CandidateID: storedCandidate.ID, DestinationRevisionID: storedDestination.ID, TargetID: "linux-x64", ArtifactDigest: "sha512:artifact", DestinationObject: "lpbs://staging/stable/linux-x64/1", Producer: "test-producer", ExternalReceipt: "upload-1", Outcome: ReceiptPublished, ObservedAt: approvedAt}
	if err := repo.RecordPublicationReceipt(ctx, "release-identity-1", receipt); err != nil {
		t.Fatalf("record publication receipt: %v", err)
	}
	receipts, err := repo.ListPublicationReceipts(ctx, "release-identity-1")
	if err != nil || len(receipts) != 1 || !receipts[0].TrustedPublication() {
		t.Fatalf("publication receipts = %#v, %v", receipts, err)
	}
	update := ClientUpdateReceipt{
		CandidateID: storedCandidate.ID, PredecessorRef: "desktop-0.9.0", SuccessorDigest: "sha256:artifact",
		TargetID: "linux-x64", VerifiedVersion: "1.0.0", Outcome: ReceiptVerified, Producer: "scenario-to-desktop",
		ExternalReceipt: "update-1", ObservedAt: approvedAt,
	}
	if err := repo.RecordClientUpdateReceipt(ctx, "release-identity-1", update); err != nil {
		t.Fatalf("record client update receipt: %v", err)
	}
	updates, err := repo.ListClientUpdateReceipts(ctx, "release-identity-1")
	if err != nil || len(updates) != 1 || !updates[0].TrustedUpdate() {
		t.Fatalf("client update receipts = %#v, %v", updates, err)
	}
	if err := repo.RecordClientUpdateReceipt(ctx, "release-identity-1", update); err != nil {
		t.Fatalf("idempotent client update receipt: %v", err)
	}
	updates, err = repo.ListClientUpdateReceipts(ctx, "release-identity-1")
	if err != nil || len(updates) != 1 {
		t.Fatalf("duplicate client update receipts = %#v, %v", updates, err)
	}
	conflictingUpdate := update
	conflictingUpdate.SuccessorDigest = "sha256:other-artifact"
	if err := repo.RecordClientUpdateReceipt(ctx, "release-identity-1", conflictingUpdate); err == nil {
		t.Fatal("conflicting client update receipt was silently accepted")
	}
	conflictingPublication := receipt
	conflictingPublication.ArtifactDigest = "sha512:other-artifact"
	if err := repo.RecordPublicationReceipt(ctx, "release-identity-1", conflictingPublication); err == nil {
		t.Fatal("conflicting publication receipt was silently accepted")
	}
	recovery := RecoveryReceipt{
		ReceiptID: "recovery-1", ReleaseID: "release-identity-1", CandidateID: storedCandidate.ID,
		DestinationRevisionID: storedDestination.ID, DeploymentID: "deployment-1", Action: "rollback",
		Outcome: "rolled_back", Health: "channel_restored", BundleSHA256: "sha256:bundle",
		ExternalReceipt: "recovery-owner-1", ObservedAt: approvedAt,
	}
	if err := repo.RecordRecoveryReceipt(ctx, recovery); err != nil {
		t.Fatalf("record recovery receipt: %v", err)
	}
	if err := repo.RecordRecoveryReceipt(ctx, recovery); err != nil {
		t.Fatalf("idempotent recovery receipt: %v", err)
	}
	conflictingRecovery := recovery
	conflictingRecovery.BundleSHA256 = "sha256:other-bundle"
	if err := repo.RecordRecoveryReceipt(ctx, conflictingRecovery); err == nil {
		t.Fatal("conflicting recovery receipt was silently accepted")
	}
	update.Outcome = ReceiptFailed
	if err := repo.RecordClientUpdateReceipt(ctx, "release-identity-1", update); err == nil {
		t.Fatal("failed client update receipt was accepted")
	}
	wrongCandidate := update
	wrongCandidate.CandidateID = "candidate-wrong"
	if err := repo.RecordClientUpdateReceipt(ctx, "release-identity-1", wrongCandidate); err == nil {
		t.Fatal("client update receipt for another candidate was accepted")
	}
	wrongPublication := receipt
	wrongPublication.DestinationRevisionID = "destination-wrong"
	if err := repo.RecordPublicationReceipt(ctx, "release-identity-1", wrongPublication); err == nil {
		t.Fatal("publication receipt for another destination was accepted")
	}
	wrongTarget := receipt
	wrongTarget.TargetID = "windows-x64"
	wrongTarget.ExternalReceipt = "upload-windows"
	if err := repo.RecordPublicationReceipt(ctx, "release-identity-1", wrongTarget); err == nil {
		t.Fatal("publication receipt for an unapproved target was accepted")
	}

	if _, err := db.Exec(`UPDATE release_candidates SET canonical_json = ? WHERE candidate_id = ?`, `{"source_revision":"tampered"}`, storedCandidate.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetCandidate(ctx, storedCandidate.ID); err == nil {
		t.Fatalf("tampered candidate error = %v", err)
	}
}

func TestSQLOperationRequestSnapshotRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-operation-snapshot-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ('r-snapshot', 'p1', 'commit', '1.0.0', 'stable')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	operation := &Operation{ID: "op-snapshot", ReleaseID: "r-snapshot", ProfileID: "p1", RequestSnapshot: json.RawMessage(`{"release_id":"r-snapshot","cloud_bundle_sha256":"sha"}`)}
	if err := repo.InsertOperation(context.Background(), operation); err != nil {
		t.Fatalf("insert operation: %v", err)
	}
	stored, err := repo.GetOperation(context.Background(), operation.ID)
	if err != nil {
		t.Fatalf("get operation: %v", err)
	}
	if string(stored.RequestSnapshot) != string(operation.RequestSnapshot) {
		t.Fatalf("stored request snapshot = %s, want %s", stored.RequestSnapshot, operation.RequestSnapshot)
	}
	active, err := repo.ListActiveOperations(context.Background())
	if err != nil || len(active) != 1 || string(active[0].RequestSnapshot) != string(operation.RequestSnapshot) {
		t.Fatalf("active operation snapshot = %#v, err=%v", active, err)
	}
}

func TestSQLOperationLeaseFencesStaleWorker(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-lease-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ('r1', 'p1', 'commit', '1.0.0', 'stable')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO release_operations (id, release_id, profile_id) VALUES ('op-1', 'r1', 'p1')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	first, ok, err := repo.AcquireOperationLease(ctx, "op-1", "worker-1", time.Minute)
	if err != nil || !ok || first.Fence != 1 {
		t.Fatalf("first lease = %#v, ok=%v, err=%v", first, ok, err)
	}
	if err := repo.UpdateOperationFenced(ctx, first, OperationRunning, "build", ""); err != nil {
		t.Fatalf("first update: %v", err)
	}
	if err := repo.AppendOperationEvent(ctx, first, OperationRunning, "build", "started"); err != nil {
		t.Fatalf("event: %v", err)
	}
	if err := repo.ReleaseOperationLease(ctx, first); err != nil {
		t.Fatalf("release lease: %v", err)
	}
	second, ok, err := repo.AcquireOperationLease(ctx, "op-1", "worker-2", time.Minute)
	if err != nil || !ok || second.Fence != 2 {
		t.Fatalf("second lease = %#v, ok=%v, err=%v", second, ok, err)
	}
	if err := repo.UpdateOperationFenced(ctx, first, OperationFailed, "stale", "must not commit"); err == nil {
		t.Fatal("stale worker updated the operation")
	}
	if err := repo.UpdateOperationFenced(ctx, second, OperationComplete, "complete", ""); err != nil {
		t.Fatalf("current update: %v", err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM release_operations WHERE id = 'op-1'`).Scan(&status); err != nil || status != OperationComplete {
		t.Fatalf("status=%q err=%v", status, err)
	}
}
