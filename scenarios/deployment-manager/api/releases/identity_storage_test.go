package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalreleases "deployment-manager/internal/releases"
	"github.com/gorilla/mux"

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

func TestSQLCloudStepReceiptIsDurableAndFenced(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-cloud-step-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ('release-cloud-step', 'p1', 'commit-1', '1.0.0', 'stable')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	if err := repo.InsertOperation(ctx, &Operation{ID: "release-op-cloud-step", ReleaseID: "release-cloud-step", ProfileID: "p1", Status: OperationRunning}); err != nil {
		t.Fatal(err)
	}
	base := CloudStep{OperationID: "release-op-cloud-step", RequestKey: "dm-cloud:release-op-cloud-step", RequestDigest: "sha256:fixed", State: "intent", RecoveryIdempotencyKey: CloudRecoveryIdempotencyKey("release-op-cloud-step")}
	if err := repo.RecordCloudStep(WithOperationLease(ctx, base.OperationID, "worker-1", 4), base); err != nil {
		t.Fatalf("record intent: %v", err)
	}
	base.State, base.DeploymentID, base.OwnerOperationID = "operation_admitted", "dep-1", "s2c-op-1"
	if err := repo.RecordCloudStep(WithOperationLease(ctx, base.OperationID, "worker-2", 3), base); err == nil {
		t.Fatal("stale cloud-step fence was accepted")
	}
	if err := repo.RecordCloudStep(WithOperationLease(ctx, base.OperationID, "worker-2", 5), base); err != nil {
		t.Fatalf("record owner receipt: %v", err)
	}
	base.State = "verification_passed"
	base.VerificationReceipt = json.RawMessage(`{"healthy":true,"deployment_id":"dep-1"}`)
	attemptedAt := time.Now().UTC()
	base.VerificationKey = CloudVerificationIdempotencyKey(base.OperationID)
	base.VerificationRequestDigest = "sha256:verification-request"
	base.VerificationState = "passed"
	base.VerificationAttemptedAt = &attemptedAt
	if err := repo.RecordCloudStep(WithOperationLease(ctx, base.OperationID, "worker-2", 6), base); err != nil {
		t.Fatalf("record verification receipt: %v", err)
	}
	stored, err := repo.GetCloudStep(ctx, base.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil || stored.State != "verification_passed" || stored.Fence != 6 || stored.DeploymentID != "dep-1" || stored.OwnerOperationID != "s2c-op-1" || string(stored.VerificationReceipt) != string(base.VerificationReceipt) || stored.VerificationKey != base.VerificationKey || stored.VerificationRequestDigest != base.VerificationRequestDigest || stored.VerificationState != "passed" || stored.VerificationAttemptedAt == nil || !stored.VerificationAttemptedAt.Equal(attemptedAt) {
		t.Fatalf("stored cloud step = %+v", stored)
	}
	base.RecoveryOwnerOperationID = "s2c-recovery-op-1"
	base.RecoveryState, base.RecoveryPreviewRef, base.RecoveryReason = "owner_pending", "preview-owner-1", "human-started rollback is pending"
	if err := repo.RecordCloudStep(WithOperationLease(ctx, base.OperationID, "worker-2", 7), base); err != nil {
		t.Fatalf("record recovery readback: %v", err)
	}
	repo = NewSQLRepository(db) // simulate a DM process restart
	stored, err = repo.GetCloudStep(ctx, base.OperationID)
	if err != nil || stored == nil || stored.RecoveryIdempotencyKey != base.RecoveryIdempotencyKey || stored.RecoveryOwnerOperationID != base.RecoveryOwnerOperationID || stored.RecoveryState != "owner_pending" || stored.RecoveryPreviewRef != "preview-owner-1" || stored.VerificationKey != base.VerificationKey || stored.VerificationRequestDigest != base.VerificationRequestDigest || stored.VerificationState != "passed" || stored.VerificationAttemptedAt == nil || !stored.VerificationAttemptedAt.Equal(attemptedAt) {
		t.Fatalf("restarted recovery standing = %+v err=%v", stored, err)
	}
	conflict := base
	conflict.RequestDigest = "sha256:changed"
	if err := repo.RecordCloudStep(WithOperationLease(ctx, base.OperationID, "worker-2", 6), conflict); err == nil {
		t.Fatal("changed immutable cloud request was accepted")
	}
}

func TestReleaseOperationProtoCarriesCloudStepStanding(t *testing.T) {
	snapshot, err := json.Marshal(durableDeployRequest{CloudAcquireCompute: true, CloudAcquireRegion: "nbg1", CloudAcquireIdempotencyKey: "dm-acquire:op-proto", CloudSpendQuoteIdempotencyKey: "dm-acquire:op-proto"})
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{ID: "op-proto", ReleaseID: "rel-proto", ProfileID: "p1", Status: OperationRunning, RequestSnapshot: snapshot, CloudStep: &CloudStep{
		OperationID: "op-proto", Fence: 7, RequestKey: "dm-cloud:op-proto", RequestDigest: "sha256:request", State: "operation_admitted",
		DeploymentID: "dep-proto", OwnerOperationID: "s2c-op-proto", Receipt: json.RawMessage(`{"receipt":{"outcome":"deployed"}}`), VerificationReceipt: json.RawMessage(`{"healthy":true}`),
		VerificationKey: CloudVerificationIdempotencyKey("op-proto"), VerificationRequestDigest: "sha256:verification", VerificationState: "pending", VerificationAttemptedAt: func() *time.Time { now := time.Now().UTC(); return &now }(),
		RecoveryIdempotencyKey: "dm-recovery:op-proto", RecoveryOwnerOperationID: "s2c-recovery-op-proto", RecoveryState: "owner_pending", RecoveryPreviewRef: "preview-proto", RecoveryReceipt: json.RawMessage(`{"outcome":"rolled_back"}`), RecoveryReason: "pending", Reason: "", UpdatedAt: time.Now().UTC(),
	}}
	wire := operation.Proto()
	if wire.GetCloudStep() == nil || wire.GetCloudStep().GetOwnerOperationId() != "s2c-op-proto" || wire.GetCloudStep().GetFence() != 7 || wire.GetCloudStep().GetVerificationReceiptJson() != `{"healthy":true}` || wire.GetCloudStep().GetVerificationKey() != CloudVerificationIdempotencyKey("op-proto") || wire.GetCloudStep().GetVerificationRequestDigest() != "sha256:verification" || wire.GetCloudStep().GetVerificationState() != "pending" || wire.GetCloudStep().GetVerificationAttemptedAt() == nil || wire.GetCloudStep().GetRecoveryIdempotencyKey() != "dm-recovery:op-proto" || wire.GetCloudStep().GetRecoveryOwnerOperationId() != "s2c-recovery-op-proto" || wire.GetCloudStep().GetRecoveryState() != "owner_pending" || wire.GetCloudStep().GetRecoveryPreviewRef() != "preview-proto" || wire.GetCloudStep().GetRecoveryReceiptJson() != `{"outcome":"rolled_back"}` || wire.GetCloudAcquireIdempotencyKey() != "dm-acquire:op-proto" || wire.GetCloudAcquireRegion() != "nbg1" {
		t.Fatalf("wire operation cloud step = %+v", wire.GetCloudStep())
	}
	decoded := operationFromProto(wire)
	if decoded.CloudStep == nil || decoded.CloudStep.RequestKey != operation.CloudStep.RequestKey || string(decoded.CloudStep.Receipt) != string(operation.CloudStep.Receipt) || string(decoded.CloudStep.VerificationReceipt) != string(operation.CloudStep.VerificationReceipt) || decoded.CloudStep.VerificationKey != operation.CloudStep.VerificationKey || decoded.CloudStep.VerificationRequestDigest != operation.CloudStep.VerificationRequestDigest || decoded.CloudStep.VerificationState != operation.CloudStep.VerificationState || decoded.CloudStep.VerificationAttemptedAt == nil || decoded.CloudStep.RecoveryIdempotencyKey != operation.CloudStep.RecoveryIdempotencyKey || decoded.CloudStep.RecoveryOwnerOperationID != operation.CloudStep.RecoveryOwnerOperationID || decoded.CloudStep.RecoveryState != operation.CloudStep.RecoveryState || decoded.CloudStep.RecoveryPreviewRef != operation.CloudStep.RecoveryPreviewRef || string(decoded.CloudStep.RecoveryReceipt) != string(operation.CloudStep.RecoveryReceipt) || decoded.CloudAcquireIdempotencyKey != "dm-acquire:op-proto" || decoded.CloudAcquireRegion != "nbg1" {
		t.Fatalf("decoded cloud step = %+v", decoded.CloudStep)
	}
}

func TestSQLRestartedCloudOperationCanResumeFromPersistedOwnerIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-cloud-resume-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel, status) VALUES ('release-resume', 'web-console/staging', 'commit-1', '1.0.0', 'stable', 'pending')`); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	operationID := operationIDForRelease("release-resume")
	request := DeployRequest{ProfileID: "web-console/staging", ReleaseID: "release-resume", CandidateID: "candidate-1", DestinationRevisionID: "destination-1", CloudSpendQuoteIdempotencyKey: "quote-1", CloudComputeManagerInstanceID: "cm-1", CloudManifest: json.RawMessage(`{"scenario":{"id":"demo"},"target":{"vps":{"compute_manager_instance_id":"cm-1"}}}`)}
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertOperation(ctx, &Operation{ID: operationID, ReleaseID: request.ReleaseID, ProfileID: request.ProfileID, RequestSnapshot: snapshot, Status: OperationRunning}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordCloudStep(ctx, CloudStep{OperationID: operationID, RequestKey: "dm-cloud:" + operationID, RequestDigest: "sha256:stored-request", State: "operation_admitted", DeploymentID: "dep-1", OwnerOperationID: "s2c-op-1"}); err != nil {
		t.Fatal(err)
	}
	orch := resumeOperationOrchFake{called: make(chan DeployRequest, 1)}
	handler := NewHandler(repo, nil, nil, orch, nil).WithDurableOperations(true).WithExecutionAuthorization(func(context.Context) error { return nil }).WithCloudComputeAdmissionVerifier(&cloudComputeAdmissionFixture{})
	if err := handler.ResumeOperations(ctx); err != nil {
		t.Fatalf("mark interrupted operation: %v", err)
	}
	standing, err := repo.GetOperation(ctx, operationID)
	if err != nil || standing.Status != OperationAmbiguous {
		t.Fatalf("restart standing=%+v err=%v", standing, err)
	}
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/release-operations/"+operationID+"/resume-cloud", nil), map[string]string{"operation_id": operationID})
	response := httptest.NewRecorder()
	handler.ResumeCloudOperation(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("resume status=%d body=%s", response.Code, response.Body.String())
	}
	select {
	case resumed := <-orch.called:
		if resumed.OperationID != operationID || resumed.ReleaseID != request.ReleaseID {
			t.Fatalf("resumed identity=%+v", resumed)
		}
	case <-time.After(time.Second):
		t.Fatal("resumed operation did not re-enter the existing request snapshot")
	}
	storedStep, err := repo.GetCloudStep(ctx, operationID)
	if err != nil || storedStep == nil || storedStep.RequestKey != "dm-cloud:"+operationID || storedStep.OwnerOperationID != "s2c-op-1" {
		t.Fatalf("persisted owner identity=%+v err=%v", storedStep, err)
	}
}
