package planworkshop

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSubmitResponseIsIdempotentAndRejectsStaleSubject(t *testing.T) {
	version := "subject-v1"
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return version, "plan-1", "hash-1", nil })
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	session, err := svc.Open(Subject{Kind: SubjectBacklog, Ref: "execute/example"}, ReviewPacket{})
	if err != nil {
		t.Fatal(err)
	}
	_, resolution, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: version, IdempotencyKey: "one"})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.State != ResolutionDirectApplied {
		t.Fatalf("state = %s", resolution.State)
	}
	_, repeated, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: version, IdempotencyKey: "one"})
	if err != nil {
		t.Fatal(err)
	}
	if repeated != resolution {
		t.Fatalf("resolution is not idempotent: %+v", repeated)
	}
	version = "subject-v2"
	_, stale, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: "subject-v1", IdempotencyKey: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if stale.State != ResolutionStale {
		t.Fatalf("state = %s", stale.State)
	}
}

func TestAttachFindingIsDurableAndIdempotent(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "", "", nil })
	subject := Subject{Kind: SubjectBacklog, Ref: "research/example"}
	first, err := svc.AttachFinding(subject, Finding{ID: "research-conclusion/execution-1", Severity: "info", Summary: "Evidence is ready", Evidence: "execution/execution-1", Disposition: &Disposition{Kind: "follow_up", Rationale: "Validate the result", Confidence: "medium"}})
	if err != nil || len(first.Packet.Findings) != 1 {
		t.Fatalf("first attach = %+v, err=%v", first, err)
	}
	second, err := svc.AttachFinding(subject, first.Packet.Findings[0])
	if err != nil || len(second.Packet.Findings) != 1 {
		t.Fatalf("repeat attach = %+v, err=%v", second, err)
	}
}

func TestProposalResponseRequiresOneReconciliation(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "plan-1", "hash-1", nil })
	starts := 0
	svc.SetReconciliationStarter(func(_ context.Context, session Session, response Response) (WorkflowProvenance, error) {
		starts++
		return WorkflowProvenance{Transition: "plan.workshop.reconcile", ExecutionID: "execution-" + response.ID, DefinitionDigest: "sha256:workflow", StartedAt: "2026-07-22T00:00:00Z"}, nil
	})
	session, err := svc.Open(Subject{Kind: SubjectInitiative, Ref: "initiative-a"}, ReviewPacket{Proposals: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, resolution, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: "v1", IdempotencyKey: "one", Accepted: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.State != ResolutionReconciliation {
		t.Fatalf("state = %s", resolution.State)
	}
	if resolution.Workflow == nil || starts != 1 {
		t.Fatalf("workflow=%+v starts=%d", resolution.Workflow, starts)
	}
	_, repeated, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: "v1", IdempotencyKey: "one"})
	if err != nil || repeated.Workflow == nil || starts != 1 {
		t.Fatalf("repeated=%+v starts=%d err=%v", repeated, starts, err)
	}
}

func TestDirectProposalUsesCanonicalPacketModeOnce(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "plan-1", "hash-1", nil })
	applied := 0
	svc.SetDirectProposalApplier(func(_ context.Context, proposal ProposalRef, actor string) error {
		applied++
		if proposal.ApplyMode != "direct" || actor != "operator" {
			t.Fatalf("proposal=%+v actor=%q", proposal, actor)
		}
		return nil
	})
	session, err := svc.Open(Subject{Kind: SubjectBacklog, Ref: "fix/example"}, ReviewPacket{Proposals: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a", ApplyMode: "direct"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, resolution, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: "v1", IdempotencyKey: "one", Accepted: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a", ApplyMode: "reconciliation"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.State != ResolutionDirectApplied || applied != 1 {
		t.Fatalf("resolution=%+v applied=%d", resolution, applied)
	}
	_, repeated, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: "v1", IdempotencyKey: "one"})
	if err != nil || repeated.State != ResolutionDirectApplied || applied != 1 {
		t.Fatalf("repeated=%+v applied=%d err=%v", repeated, applied, err)
	}
}

func TestResponseRejectsForeignProposalAndOpenPreservesPacketHistory(t *testing.T) {
	version := "v1"
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return version, "plan-1", "hash-1", nil })
	session, err := svc.Open(Subject{Kind: SubjectBacklog, Ref: "fix/example"}, ReviewPacket{Questions: []DecisionQuestion{{ID: "choice", Question: "Choose"}}, Proposals: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: version, IdempotencyKey: "foreign", Accepted: []ProposalRef{{SessionID: "other", ProposalID: "proposal-a"}}}); err == nil {
		t.Fatal("foreign proposal was accepted")
	}
	version = "v2"
	updated, err := svc.Open(session.Subject, ReviewPacket{Findings: []Finding{{ID: "new", Severity: "info", Summary: "new evidence"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.PacketHistory) != 2 || updated.PacketHistory[0].SubjectVersion != "v1" || updated.PacketHistory[1].SubjectVersion != "v2" {
		t.Fatalf("packet history = %+v", updated.PacketHistory)
	}
}

func TestReviewRunProjectsTypedResultOnce(t *testing.T) {
	version := "v1"
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return version, "plan-1", "hash-1", nil })
	now := time.Date(2026, 7, 22, 3, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	starts, records := 0, 0
	svc.SetReviewStarter(func(_ context.Context, session Session) (WorkflowProvenance, error) {
		starts++
		return WorkflowProvenance{Transition: "plan.workshop.review", ExecutionID: "review-" + session.ID, DefinitionDigest: "sha256:review", RunID: "run-1", StartedAt: now.Format(time.RFC3339Nano)}, nil
	})
	svc.SetReviewCollector(func(_ context.Context, _ Session, provenance WorkflowProvenance) (ReviewResult, error) {
		if provenance.ExecutionID == "" {
			t.Fatal("collector did not receive workflow provenance")
		}
		return ReviewResult{Outcome: "packet", Findings: []Finding{{ID: "evidence", Severity: "info", Summary: "reviewed"}}, Questions: []DecisionQuestion{{ID: "choice", Question: "Proceed?"}}, Proposals: []ProposalDraft{{Summary: "Apply safe graph mutation", Payload: json.RawMessage(`{"form":"mutation_list","mutations":[]}`), ApplyMode: "direct"}}}, nil
	})
	svc.SetProposalRecorder(func(_ context.Context, _ Session, _ WorkflowProvenance, drafts []ProposalDraft) (string, []ProposalRef, error) {
		records++
		if len(drafts) != 1 || drafts[0].ApplyMode != "direct" {
			t.Fatalf("drafts = %+v", drafts)
		}
		return "sess-review", []ProposalRef{{SessionID: "sess-review", ProposalID: "proposal-review", ApplyMode: "direct"}}, nil
	})
	session, err := svc.Open(Subject{Kind: SubjectInitiative, Ref: "initiative-a"}, ReviewPacket{})
	if err != nil {
		t.Fatal(err)
	}
	started, review, err := svc.StartReview(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if starts != 1 || review.State != ReviewRunning || started.Review == nil {
		t.Fatalf("started=%+v review=%+v starts=%d", started, review, starts)
	}
	updated, applied, err := svc.ApplyReview(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != ReviewApplied || records != 1 || len(updated.Packet.Proposals) != 1 || updated.Packet.Proposals[0].ApplyMode != "direct" {
		t.Fatalf("updated=%+v applied=%+v records=%d", updated, applied, records)
	}
	_, repeated, err := svc.ApplyReview(context.Background(), session.ID)
	if err != nil || repeated.State != ReviewApplied || records != 1 {
		t.Fatalf("repeated=%+v records=%d err=%v", repeated, records, err)
	}
}

func TestApplyReviewMarksChangedSubjectStale(t *testing.T) {
	version := "v1"
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return version, "plan-1", "hash-1", nil })
	svc.SetReviewStarter(func(_ context.Context, _ Session) (WorkflowProvenance, error) {
		return WorkflowProvenance{ExecutionID: "review-1", DefinitionDigest: "sha256:review"}, nil
	})
	svc.SetReviewCollector(func(_ context.Context, _ Session, _ WorkflowProvenance) (ReviewResult, error) {
		t.Fatal("stale run must not be collected")
		return ReviewResult{}, nil
	})
	svc.SetProposalRecorder(func(_ context.Context, _ Session, _ WorkflowProvenance, _ []ProposalDraft) (string, []ProposalRef, error) {
		t.Fatal("stale run must not write proposals")
		return "", nil, nil
	})
	session, err := svc.Open(Subject{Kind: SubjectBacklog, Ref: "fix/example"}, ReviewPacket{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.StartReview(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	version = "v2"
	updated, review, err := svc.ApplyReview(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if review.State != ReviewStale || updated.Review == nil || updated.Review.State != ReviewStale {
		t.Fatalf("updated=%+v review=%+v", updated, review)
	}
}

func TestApplyReconciliationCreatesCandidateWithoutCanonicalApply(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "plan-1", "hash-1", nil })
	svc.SetReconciliationStarter(func(_ context.Context, _ Session, response Response) (WorkflowProvenance, error) {
		return WorkflowProvenance{Transition: "plan.workshop.reconcile", ExecutionID: "reconcile-" + response.ID, DefinitionDigest: "sha256:reconcile"}, nil
	})
	svc.SetReconciliationCollector(func(_ context.Context, _ Session, _ Response, _ WorkflowProvenance) (ReconciliationResult, error) {
		return ReconciliationResult{Outcome: "candidate", CandidatePlan: json.RawMessage(`{"title":"Candidate plan","purpose":"Reconciled change"}`)}, nil
	})
	created := 0
	svc.SetCandidateCreator(func(_ context.Context, session Session, response Response, _ WorkflowProvenance, raw json.RawMessage) (CandidateReference, error) {
		created++
		if session.PlanID != "plan-1" || response.ID == "" || !json.Valid(raw) {
			t.Fatalf("session=%+v response=%+v candidate=%s", session, response, raw)
		}
		return CandidateReference{ID: "candidate-1", PlanID: "plan-1", ExpectedBaseContentHash: "hash-1", QualityStatus: "pass"}, nil
	})
	appliedCandidate := 0
	svc.SetCandidateApplier(func(_ context.Context, session Session, candidate CandidateReference, acknowledged bool) error {
		appliedCandidate++
		if !acknowledged || session.ID == "" || candidate.ID != "candidate-1" {
			t.Fatalf("session=%+v candidate=%+v acknowledged=%v", session, candidate, acknowledged)
		}
		return nil
	})
	session, err := svc.Open(Subject{Kind: SubjectInitiative, Ref: "initiative-a"}, ReviewPacket{Proposals: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a", ApplyMode: "reconciliation"}}})
	if err != nil {
		t.Fatal(err)
	}
	_, resolution, err := svc.SubmitResponse(context.Background(), session.ID, Response{Actor: "operator", SubjectVersion: "v1", IdempotencyKey: "one", Accepted: []ProposalRef{{SessionID: "session-a", ProposalID: "proposal-a"}}})
	if err != nil || resolution.Workflow == nil {
		t.Fatalf("resolution=%+v err=%v", resolution, err)
	}
	updated, applied, err := svc.ApplyReconciliation(context.Background(), session.ID, resolution.ResponseID)
	if err != nil {
		t.Fatal(err)
	}
	if applied.State != ResolutionCandidateReady || applied.Candidate == nil || applied.Candidate.ID != "candidate-1" || created != 1 {
		t.Fatalf("updated=%+v applied=%+v created=%d", updated, applied, created)
	}
	_, repeated, err := svc.ApplyReconciliation(context.Background(), session.ID, resolution.ResponseID)
	if err != nil || repeated.State != ResolutionCandidateReady || created != 1 {
		t.Fatalf("repeated=%+v created=%d err=%v", repeated, created, err)
	}
	if _, _, err := svc.ApplyCandidate(context.Background(), session.ID, resolution.ResponseID, false); err == nil {
		t.Fatal("candidate application did not require quality acknowledgement")
	}
	_, candidateApplied, err := svc.ApplyCandidate(context.Background(), session.ID, resolution.ResponseID, true)
	if err != nil || candidateApplied.State != ResolutionCandidateApplied || appliedCandidate != 1 {
		t.Fatalf("candidateApplied=%+v appliedCandidate=%d err=%v", candidateApplied, appliedCandidate, err)
	}
	_, candidateRepeated, err := svc.ApplyCandidate(context.Background(), session.ID, resolution.ResponseID, true)
	if err != nil || candidateRepeated.State != ResolutionCandidateApplied || appliedCandidate != 1 {
		t.Fatalf("candidateRepeated=%+v appliedCandidate=%d err=%v", candidateRepeated, appliedCandidate, err)
	}
}

func TestDiscardCandidateLeavesCanonicalPlanUntouchedAndIsIdempotent(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "plan-1", "hash-1", nil })
	session, err := svc.Open(Subject{Kind: SubjectInitiative, Ref: "initiative-a"}, ReviewPacket{})
	if err != nil {
		t.Fatal(err)
	}
	session.Responses = []Response{{ID: "response-1", Actor: "operator", SubjectVersion: "v1", IdempotencyKey: "response-1"}}
	session.Resolutions = []Resolution{{ResponseID: "response-1", State: ResolutionCandidateReady, Candidate: &CandidateReference{ID: "candidate-1", PlanID: "plan-1", ExpectedBaseContentHash: "hash-1"}}}
	if err := svc.store.Save(session); err != nil {
		t.Fatal(err)
	}
	discarded := 0
	svc.SetCandidateDiscarder(func(_ context.Context, got Session, candidate CandidateReference, reason string) error {
		discarded++
		if got.ID != session.ID || candidate.ID != "candidate-1" || reason != "not suitable" {
			t.Fatalf("session=%+v candidate=%+v reason=%q", got, candidate, reason)
		}
		return nil
	})
	_, resolution, err := svc.DiscardCandidate(context.Background(), session.ID, "response-1", "not suitable")
	if err != nil || resolution.State != ResolutionCandidateDiscarded || discarded != 1 {
		t.Fatalf("resolution=%+v discarded=%d err=%v", resolution, discarded, err)
	}
	_, repeated, err := svc.DiscardCandidate(context.Background(), session.ID, "response-1", "not suitable")
	if err != nil || repeated.State != ResolutionCandidateDiscarded || discarded != 1 {
		t.Fatalf("repeated=%+v discarded=%d err=%v", repeated, discarded, err)
	}
}

func TestAttachProposalIsIdempotentAndKeepsFindingHistory(t *testing.T) {
	svc := NewService(NewStore(t.TempDir()), func(Subject) (string, string, string, error) { return "v1", "plan-1", "hash-1", nil })
	subject := Subject{Kind: SubjectBacklog, Ref: "fix/example"}
	if _, err := svc.AttachFinding(subject, Finding{ID: "review-1", Severity: "info", Summary: "remaining work", Disposition: &Disposition{Kind: "follow_up", Rationale: "fix the verified regression", Confidence: "high"}}); err != nil {
		t.Fatal(err)
	}
	ref := ProposalRef{SessionID: "session-review", ProposalID: "proposal-follow-up", ApplyMode: "direct"}
	first, err := svc.AttachProposal(subject, ref)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.AttachProposal(subject, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Packet.Findings) != 1 || len(second.Packet.Proposals) != 1 || second.Packet.Proposals[0] != ref {
		t.Fatalf("first=%+v second=%+v", first.Packet, second.Packet)
	}
}
