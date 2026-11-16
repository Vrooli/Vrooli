package desktopoptimize

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

type evaluatorFunc func(context.Context, string, Request, MeasurementPlan) (Matrix, error)

func (f evaluatorFunc) MeasurePaired(c context.Context, operationID string, r Request, p MeasurementPlan) (Matrix, error) {
	return f(c, operationID, r, p)
}

type promotionFake struct {
	promoted, removed, rolled int
	removeErr                 error
	removeOperationID         string
}

func (p *promotionFake) Promote(_ context.Context, operationID string, _ Request) (string, error) {
	p.promoted++
	if operationID != "run-1/promote" {
		return "", fmt.Errorf("unexpected promotion operation id %q", operationID)
	}
	return "promote-1", nil
}

func (p *promotionFake) RemovePredecessor(_ context.Context, operationID string, _ Request, _ string) (string, error) {
	p.removed++
	p.removeOperationID = operationID
	if p.removeErr != nil {
		return "", p.removeErr
	}
	return "remove-1", nil
}

func (p *promotionFake) Rollback(context.Context, string, Request, string) (string, error) {
	p.rolled++
	return "rollback-1", nil
}

func testRequest(now time.Time) Request {
	identity := Identity{Scenario: "hello-desktop", ArtifactDigest: "sha256:old", HostFingerprint: "linux-xvfb-a", Display: "1920x1080", DeploymentMode: "bundled", ProfilerMode: "disabled"}
	candidate := Candidate{ID: "startup-profile-b", PredecessorDigest: identity.ArtifactDigest, ArtifactDigest: "sha256:new", StartupProfile: map[string]string{"main_window": "defer-noncritical"}}
	digest := identityDigest(identity)
	return Request{Schema: RequestSchema, RunID: "run-1", Target: TargetID, Subject: identity, BaselineRef: "performance-baseline.json#cold+warm", Candidate: candidate, Acquisition: AcquisitionProof{ReceiptID: "acquire-1", CandidateID: candidate.ID, ArtifactDigest: candidate.ArtifactDigest, PredecessorDigest: candidate.PredecessorDigest, SubjectDigest: digest}, Rights: GateProof{Decision: "allow", ReceiptID: "rights-1", SubjectDigest: digest, ExpiresAt: now.Add(time.Hour)}, Disk: DiskProof{Decision: "admit", LeaseID: "disk-1", SubjectDigest: digest, RequiredBytes: 100, AvailableBytes: 1000, ExpiresAt: now.Add(time.Hour)}, MinimumPairs: 5, MaxRegressionFraction: .05}
}

func goodMatrix(r Request) Matrix {
	m := Matrix{Subject: r.Subject, CandidateDigest: r.Candidate.ArtifactDigest}
	for _, class := range []string{"cold", "warm"} {
		for _, phase := range requiredPhases {
			for i := 0; i < 5; i++ {
				base := int64(1000)
				if class == "warm" {
					base = 500
				}
				if phase == "splash_to_app_ready" {
					base *= 5
				}
				if phase == "runtime_spawn_to_ready" {
					base *= 4
				}
				m.Pairs = append(m.Pairs, Pair{Class: class, Phase: phase, PairID: fmt.Sprintf("%s-%d", class, i), Control: Sample{RunID: fmt.Sprintf("control-%s-%d", class, i), EvidenceRef: fmt.Sprintf("control/evidence/%s/%d", class, i), DurationMS: base + int64(i)}, Candidate: Sample{RunID: fmt.Sprintf("candidate-%s-%d", class, i), EvidenceRef: fmt.Sprintf("candidate/evidence/%s/%d", class, i), DurationMS: base + int64(i)}})
			}
		}
	}
	for i := range m.Pairs {
		if m.Pairs[i].Phase == "process_to_splash_first_paint" {
			m.Pairs[i].Candidate.DurationMS -= 100
		}
	}
	return m
}

func TestMinimumPlanPinsExactDP1Denominator(t *testing.T) {
	plan := MinimumPlan(testRequest(time.Now()))
	if plan.Target != TargetID || plan.PairsPerClass != 5 || plan.Pairing != "control_then_candidate_same_host_same_artifact_cohort" {
		t.Fatalf("minimum plan header = %+v", plan)
	}
	wantClasses := []string{"cold", "warm"}
	if fmt.Sprint(plan.Classes) != fmt.Sprint(wantClasses) || fmt.Sprint(plan.Phases) != fmt.Sprint(requiredPhases) {
		t.Fatalf("minimum plan cohort/phases = %+v, %+v", plan.Classes, plan.Phases)
	}
	// An explicitly larger denominator is honored; the floor is never reduced.
	r := testRequest(time.Now())
	r.MinimumPairs = 8
	if got := MinimumPlan(r).PairsPerClass; got != 8 {
		t.Fatalf("larger requested denominator = %d, want 8", got)
	}
}

func TestRequestValidationRejectsStaleOrMismatchedRightsAndDisk(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	r := testRequest(now)
	r.Rights.ExpiresAt = now
	if err := ValidateRequest(r, now); err == nil {
		t.Fatal("expired rights receipt admitted")
	}
	r = testRequest(now)
	r.Disk.AvailableBytes = 99
	if err := ValidateRequest(r, now); err == nil {
		t.Fatal("insufficient disk admission accepted")
	}
	r = testRequest(now)
	r.Rights.SubjectDigest = "sha256:other"
	if err := ValidateRequest(r, now); err == nil {
		t.Fatal("rights evidence bound to another artifact accepted")
	}
}

func TestExecuteChecksAdmissionBeforeMeasurement(t *testing.T) {
	req := testRequest(time.Now())
	req.Disk = DiskProof{}
	called := false
	got := Execute(context.Background(), Dependencies{Evaluator: evaluatorFunc(func(context.Context, string, Request, MeasurementPlan) (Matrix, error) {
		called = true
		return Matrix{}, nil
	}), Journal: &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}}, req)
	if got.Disposition != "refused" || called {
		t.Fatalf("aggregate API measured without current admission: receipt=%+v owner_called=%v", got, called)
	}
}

func TestMeasureRequiresPreflightAdmissionBeforeOwnerCall(t *testing.T) {
	req := testRequest(time.Now())
	req.PreflightRights, req.PreflightDisk = GateProof{}, DiskProof{}
	req.Rights, req.Disk = GateProof{}, DiskProof{}
	called := false
	got := Measure(context.Background(), Dependencies{Evaluator: evaluatorFunc(func(context.Context, string, Request, MeasurementPlan) (Matrix, error) {
		called = true
		return Matrix{}, nil
	}), Journal: &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}}, req)
	if got.Disposition != "refused" || called {
		t.Fatalf("measurement owner was called without preflight admission: receipt=%+v owner_called=%v", got, called)
	}
}

func TestRequestPreservesUpstreamCanonicalProofBinding(t *testing.T) {
	now := time.Now()
	r := testRequest(now)
	r.ProofBindingDigest = "a1b2-post-acquisition-canonical-digest"
	r.PreflightProofBindingDigest = "c3d4-pre-acquisition-canonical-digest"
	r.Acquisition.SubjectDigest = r.ProofBindingDigest
	r.Rights.SubjectDigest = r.ProofBindingDigest
	r.Disk.SubjectDigest = r.ProofBindingDigest
	r.PreflightRights, r.PreflightDisk = r.Rights, r.Disk
	r.PreflightRights.SubjectDigest = r.PreflightProofBindingDigest
	r.PreflightDisk.SubjectDigest = r.PreflightProofBindingDigest
	if err := ValidateMeasurementRequestAt(r, now); err != nil {
		t.Fatalf("preflight canonical proof digest was not accepted unchanged: %v", err)
	}
	if err := ValidateRequest(r, now); err != nil {
		t.Fatalf("post-acquisition canonical proof digest was not accepted unchanged: %v", err)
	}
	r.Disk.SubjectDigest = identityDigest(r.Subject)
	if err := ValidateRequest(r, now); err == nil {
		t.Fatal("target-specific digest was substituted for the upstream request digest")
	}
}

func TestRequestRejectsAcquisitionDigestDriftBeforeMeasurement(t *testing.T) {
	r := testRequest(time.Now())
	r.Candidate.ArtifactDigest = "sha256:unexpected"
	if err := ValidateRequest(r, time.Now()); err == nil {
		t.Fatal("candidate digest drift from the acquired artifact receipt was accepted")
	}
}

func TestValidateMatrixRequiresCompletePairedColdWarmEvidence(t *testing.T) {
	now := time.Now()
	req := testRequest(now)
	m := goodMatrix(req)
	if rows, err := ValidateMatrix(req, m); err != nil || len(rows) != len(requiredPhases)*2 {
		t.Fatalf("valid matrix rejected: rows=%d err=%v", len(rows), err)
	}
	m.Pairs = m.Pairs[:len(m.Pairs)-1]
	if _, err := ValidateMatrix(req, m); err == nil {
		t.Fatal("incomplete warm/cold phase matrix accepted")
	}
	m = goodMatrix(req)
	for i := range m.Pairs {
		if m.Pairs[i].Class == "cold" && m.Pairs[i].PairID == "cold-0" && m.Pairs[i].Phase == "runtime_spawn_to_ready" {
			m.Pairs[i].Candidate.RunID = "different-candidate-launch"
		}
	}
	if _, err := ValidateMatrix(req, m); err == nil {
		t.Fatal("phase rows from different launches were accepted as one pair")
	}
}

func TestExecutePersistsPromotionAndRemovalThenReopensByRunID(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	req := testRequest(now)
	journal := &FileJournal{Path: filepath.Join(t.TempDir(), "receipts", "dp1.jsonl")}
	owner := &promotionFake{}
	deps := Dependencies{Evaluator: evaluatorFunc(func(_ context.Context, _ string, got Request, plan MeasurementPlan) (Matrix, error) {
		if got.RunID != req.RunID || plan.PairsPerClass != 5 || len(plan.Phases) != 3 {
			t.Fatalf("unexpected DP1 target plan: req=%+v plan=%+v", got, plan)
		}
		return goodMatrix(req), nil
	}), Promotion: owner, Journal: journal, Now: func() time.Time { return now }}
	got := Execute(context.Background(), deps, req)
	if got.Disposition != "promoted" || got.PromotionReceipt != "promote-1" || got.PredecessorRemovalReceipt != "remove-1" || len(got.Matrix) != 6 {
		t.Fatalf("unexpected terminal receipt: %+v", got)
	}
	reopened := &FileJournal{Path: journal.Path}
	found, ok, err := reopened.Find(req.RunID)
	if err != nil || !ok || found.RequestDigest != got.RequestDigest {
		t.Fatalf("receipt not recoverable: ok=%v err=%v found=%+v", ok, err, found)
	}
	// A process restart/retry returns the durable terminal result without
	// repeating measurement, promotion, or predecessor cleanup.
	replayed := Execute(context.Background(), Dependencies{Evaluator: evaluatorFunc(func(context.Context, string, Request, MeasurementPlan) (Matrix, error) {
		t.Fatal("replayed a completed trial")
		return Matrix{}, nil
	}), Promotion: owner, Journal: reopened, Now: func() time.Time { return now }}, req)
	if replayed.Disposition != "promoted" || owner.promoted != 1 || owner.removed != 1 {
		t.Fatalf("completed run was repeated: receipt=%+v owner=%+v", replayed, owner)
	}
}

func TestExecuteRefusesIncompleteMatrixBeforePromotion(t *testing.T) {
	now := time.Now()
	req := testRequest(now)
	owner := &promotionFake{}
	journal := &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}
	got := Execute(context.Background(), Dependencies{Evaluator: evaluatorFunc(func(context.Context, string, Request, MeasurementPlan) (Matrix, error) {
		m := goodMatrix(req)
		m.Pairs = m.Pairs[1:]
		return m, nil
	}), Promotion: owner, Journal: journal, Now: time.Now}, req)
	if got.Disposition != "refused" || owner.promoted != 0 || owner.removed != 0 {
		t.Fatalf("incomplete evidence advanced to promotion: receipt=%+v owner=%+v", got, owner)
	}
}

func TestExecuteRollsBackIfPredecessorRemovalFails(t *testing.T) {
	now := time.Now()
	req := testRequest(now)
	owner := &promotionFake{removeErr: errors.New("owner removal refused")}
	journal := &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}
	got := Execute(context.Background(), Dependencies{Evaluator: evaluatorFunc(func(context.Context, string, Request, MeasurementPlan) (Matrix, error) { return goodMatrix(req), nil }), Promotion: owner, Journal: journal, Now: time.Now}, req)
	if got.Disposition != "failed" || got.RollbackReceipt != "rollback-1" || owner.rolled != 1 {
		t.Fatalf("cleanup failure did not trigger rollback: receipt=%+v owner=%+v", got, owner)
	}
}

func TestExecuteRefusesAmbiguousNonterminalReceiptAfterRestart(t *testing.T) {
	now := time.Now().UTC()
	req := testRequest(now)
	journal := &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}
	intent := Receipt{Schema: ReceiptSchema, RunID: req.RunID, RequestDigest: requestDigest(req), Disposition: "running", Stage: "predecessor-removal-intent", StartedAt: now, PromotionReceipt: "promote-1", Subject: req.Subject, Candidate: req.Candidate}
	if err := journal.Append(intent); err != nil {
		t.Fatal(err)
	}
	owner := &promotionFake{}
	evaluated := false
	got := Execute(context.Background(), Dependencies{Evaluator: evaluatorFunc(func(context.Context, string, Request, MeasurementPlan) (Matrix, error) {
		evaluated = true
		return goodMatrix(req), nil
	}), Promotion: owner, Journal: journal}, req)
	if got.Disposition != "promoted" || got.Stage != "complete" || evaluated || owner.promoted != 0 || owner.removed != 1 || owner.removeOperationID != "run-1/remove-predecessor" {
		t.Fatalf("cleanup stage did not resume with stable owner identity: receipt=%+v evaluated=%v owner=%+v", got, evaluated, owner)
	}
}

func TestStagedOperationsDoNotRepeatMeasurementOrCleanUpEarly(t *testing.T) {
	now := time.Now()
	req := testRequest(now)
	preflight := testRequest(now)
	req.PreflightRights, req.PreflightDisk = preflight.Rights, preflight.Disk
	req.Rights = GateProof{}
	req.Disk = DiskProof{}
	journal := &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}
	owner := &promotionFake{}
	measurements := 0
	deps := Dependencies{Evaluator: evaluatorFunc(func(_ context.Context, operationID string, got Request, _ MeasurementPlan) (Matrix, error) {
		measurements++
		if operationID != "run-1/measure" {
			t.Fatalf("measurement operation id = %q", operationID)
		}
		return goodMatrix(got), nil
	}), Promotion: owner, Journal: journal}
	measured := Measure(context.Background(), deps, req)
	if measured.Stage != "measurement-complete" || measurements != 1 || owner.promoted != 0 || owner.removed != 0 {
		t.Fatalf("measure stage crossed effect boundary: receipt=%+v measurements=%d owner=%+v", measured, measurements, owner)
	}
	if again := Measure(context.Background(), deps, req); again.Stage != "measurement-complete" || measurements != 1 {
		t.Fatalf("measurement receipt was not idempotent: receipt=%+v measurements=%d", again, measurements)
	}
	// Program Runtime issues these proofs after its evaluation stage. Gate
	// evidence may arrive later without changing the pinned target request.
	req.Rights = testRequest(now).Rights
	req.Disk = testRequest(now).Disk
	promoted := Promote(context.Background(), deps, req)
	if promoted.Stage != "promotion-complete" || owner.promoted != 1 || owner.removed != 0 {
		t.Fatalf("promotion stage removed predecessor early: receipt=%+v owner=%+v", promoted, owner)
	}
	if again := Promote(context.Background(), deps, req); again.Stage != "promotion-complete" || owner.promoted != 1 || owner.removed != 0 {
		t.Fatalf("promotion receipt was not idempotent: receipt=%+v owner=%+v", again, owner)
	}
	cleaned := Cleanup(context.Background(), deps, req)
	if cleaned.Disposition != "promoted" || cleaned.Stage != "complete" || measurements != 1 || owner.promoted != 1 || owner.removed != 1 {
		t.Fatalf("cleanup stage did not complete exactly once: receipt=%+v measurements=%d owner=%+v", cleaned, measurements, owner)
	}
}

func TestMeasureRetriesAnInterruptedIntentWithSameOwnerOperationID(t *testing.T) {
	now := time.Now()
	req := testRequest(now)
	journal := &FileJournal{Path: filepath.Join(t.TempDir(), "receipts.jsonl")}
	ownerCalls := 0
	operationIDs := []string{}
	deps := Dependencies{Evaluator: evaluatorFunc(func(_ context.Context, operationID string, got Request, _ MeasurementPlan) (Matrix, error) {
		ownerCalls++
		operationIDs = append(operationIDs, operationID)
		if ownerCalls == 1 {
			return Matrix{}, errors.New("temporary evaluator outage")
		}
		return goodMatrix(got), nil
	}), Journal: journal}
	first := Measure(context.Background(), deps, req)
	if first.Stage != "measurement-intent" || first.Disposition != "running" {
		t.Fatalf("failed owner call did not leave resumable intent: %+v", first)
	}
	second := Measure(context.Background(), deps, req)
	if second.Stage != "measurement-complete" || second.Disposition != "running" || ownerCalls != 2 || fmt.Sprint(operationIDs) != "[run-1/measure run-1/measure]" {
		t.Fatalf("retry did not reuse stable owner operation identity: receipt=%+v calls=%d ids=%v", second, ownerCalls, operationIDs)
	}
}
