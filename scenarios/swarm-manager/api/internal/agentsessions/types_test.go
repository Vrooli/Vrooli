package agentsessions

import (
	"encoding/json"
	"errors"
	"testing"

	"swarm-manager/internal/identity"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestProposalDecisionPublicationTransitionStateRejectsInvalidEdges(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{name: "published cannot be reopened", from: ProposalDecisionPublicationPublished, to: ProposalDecisionPublicationPending},
		{name: "legacy marker cannot be overwritten in place", from: ProposalDecisionPublicationLegacyUnresolved, to: ProposalDecisionPublicationPending},
		{name: "unknown current state", from: "mystery", to: ProposalDecisionPublicationPublished},
		{name: "unknown next state", from: ProposalDecisionPublicationPending, to: "mystery"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publication := ProposalDecisionPublication{State: tt.from}
			if err := publication.TransitionState(tt.to); err == nil {
				t.Fatalf("TransitionState(%q) unexpectedly succeeded from %q", tt.to, tt.from)
			}
			if publication.State != tt.from {
				t.Fatalf("failed transition mutated state to %q", publication.State)
			}
		})
	}
}

func TestProposalDecisionPublicationTransitionStateAllowsPublishAndIdempotence(t *testing.T) {
	publication := ProposalDecisionPublication{State: ProposalDecisionPublicationPending}
	if err := publication.TransitionState(ProposalDecisionPublicationPublished); err != nil {
		t.Fatalf("pending -> published: %v", err)
	}
	if err := publication.TransitionState(ProposalDecisionPublicationPublished); err != nil {
		t.Fatalf("published idempotence: %v", err)
	}
}

const testTimestamp = "2026-05-01T12:00:00Z"

func TestSessionValidateAcceptsCompleteContract(t *testing.T) {
	session := validSession()
	if err := session.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSessionValidateRejectsInvalidStatusAndKind(t *testing.T) {
	session := validSession()
	session.Kind = "custom"
	if err := session.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("Validate() error = %v, want ErrValidation", err)
	}

	session = validSession()
	session.Status = "done"
	if err := session.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("Validate() error = %v, want ErrValidation", err)
	}
}

func TestSessionTransitionStatusEnforcesLifecycleEdges(t *testing.T) {
	for from, allowed := range sessionStatusTransitions {
		for to := range allowed {
			session := Session{Status: from}
			if err := session.TransitionStatus(to); err != nil {
				t.Errorf("TransitionStatus(%s -> %s) error = %v", from, to, err)
			}
			if session.Status != to {
				t.Errorf("TransitionStatus(%s -> %s) left status %s", from, to, session.Status)
			}
		}
	}

	for _, edge := range [][2]Status{
		{StatusDraft, StatusComplete},
		{StatusApplying, StatusRunning},
		{StatusCanceled, StatusRunning},
	} {
		session := Session{Status: edge[0]}
		if err := session.TransitionStatus(edge[1]); !errors.Is(err, ErrValidation) {
			t.Errorf("TransitionStatus(%s -> %s) error = %v, want ErrValidation", edge[0], edge[1], err)
		}
		if session.Status != edge[0] {
			t.Errorf("rejected TransitionStatus(%s -> %s) mutated status to %s", edge[0], edge[1], session.Status)
		}
	}
}

func TestProposalTransitionStatusEnforcesReviewEdges(t *testing.T) {
	for from, allowed := range proposalStatusTransitions {
		for to := range allowed {
			proposal := Proposal{Status: from}
			if err := proposal.TransitionStatus(to); err != nil {
				t.Errorf("TransitionStatus(%s -> %s) error = %v", from, to, err)
			}
			if proposal.Status != to {
				t.Errorf("TransitionStatus(%s -> %s) left status %s", from, to, proposal.Status)
			}
		}
	}
	for _, edge := range [][2]ProposalStatus{
		{ProposalStatusReady, ProposalStatusDraft},
		{ProposalStatusApplied, ProposalStatusReady},
		{ProposalStatusRejected, ProposalStatusApplied},
	} {
		proposal := Proposal{Status: edge[0]}
		if err := proposal.TransitionStatus(edge[1]); !errors.Is(err, ErrValidation) {
			t.Errorf("TransitionStatus(%s -> %s) error = %v, want ErrValidation", edge[0], edge[1], err)
		}
		if proposal.Status != edge[0] {
			t.Errorf("rejected TransitionStatus(%s -> %s) mutated status to %s", edge[0], edge[1], proposal.Status)
		}
	}
}

func TestProposalValidateRejectsInvalidPayloadJSON(t *testing.T) {
	proposal := validSession().Proposals[0]
	proposal.PayloadJSON = "{not-json"
	if err := proposal.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("Validate() error = %v, want ErrValidation", err)
	}
}

func TestArtifactValidateRequiresReference(t *testing.T) {
	artifact := validSession().Artifacts[0]
	artifact.EntityRef = ""
	if err := artifact.Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("Validate() error = %v, want ErrValidation", err)
	}
}

func TestSessionJSONRoundTrip(t *testing.T) {
	session := validSession()
	payload, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded Session
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded Validate() error = %v", err)
	}
	if decoded.ID != session.ID || decoded.Proposals[0].Kind != ProposalBacklogBatchImport {
		t.Fatalf("decoded session mismatch: %+v", decoded)
	}
}

func TestSessionProtoJSONRoundTrip(t *testing.T) {
	session := validSession()
	msg := SessionToProto(session)
	payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson marshal error = %v", err)
	}
	decoded := SessionToProto(Session{})
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, decoded); err != nil {
		t.Fatalf("protojson unmarshal error = %v", err)
	}
	if decoded.GetCreatedBy().GetSessionId() != "sess_test" {
		t.Fatalf("created_by.session_id = %q, want sess_test", decoded.GetCreatedBy().GetSessionId())
	}
	if decoded.GetProposals()[0].GetPayloadJson() != `{"items":[]}` {
		t.Fatalf("payload_json = %q", decoded.GetProposals()[0].GetPayloadJson())
	}
	if decoded.GetProposalTarget().GetRef() != "api-quality-gates" {
		t.Fatalf("proposal_target.ref = %q, want api-quality-gates", decoded.GetProposalTarget().GetRef())
	}
}

func TestAttributionFromProvenance(t *testing.T) {
	attr := AttributionFromProvenance(identity.Provenance{
		Actor:       identity.TypeAgent,
		RunID:       "run-1",
		TaskID:      "task-1",
		ProfileKey:  "swarm-manager/default",
		SessionID:   "sess_test",
		SessionKind: string(KindMetaOrchestration),
		Source:      "session/sess_test",
	})
	if err := attr.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if attr.Type != AttributionAgent || attr.RunID != "run-1" {
		t.Fatalf("unexpected attribution: %+v", attr)
	}
	if attr.SessionID != "sess_test" || attr.SessionKind != KindMetaOrchestration || attr.Source != "session/sess_test" {
		t.Fatalf("session attribution was not copied: %+v", attr)
	}
}

func validSession() Session {
	attr := &Attribution{
		Type:        AttributionAgent,
		RunID:       "run-1",
		TaskID:      "task-1",
		ProfileKey:  "swarm-manager/default",
		SessionID:   "sess_test",
		SessionKind: KindMetaOrchestration,
		Source:      "session/sess_test",
	}
	return Session{
		ID:             "sess_test",
		Title:          "Plan API quality gates",
		Kind:           KindMetaOrchestration,
		Status:         StatusProposalReady,
		SkillID:        "swarm-manager-meta-orchestrator",
		TaskID:         "task-1",
		RunID:          "run-1",
		CreatedAt:      testTimestamp,
		UpdatedAt:      testTimestamp,
		CreatedBy:      attr,
		ProposalTarget: &ProposalTarget{Type: ContextGoal, Ref: "api-quality-gates", Name: "API Quality Gates"},
		Messages: []Message{{
			ID:        "msg-1",
			Role:      MessageRoleUser,
			Content:   "Plan this work.",
			CreatedAt: testTimestamp,
		}},
		Proposals: []Proposal{{
			ID:          "prop-1",
			Kind:        ProposalBacklogBatchImport,
			Status:      ProposalStatusReady,
			Summary:     "Create the API quality gate milestone.",
			PayloadJSON: `{"items":[]}`,
			CreatedAt:   testTimestamp,
			UpdatedAt:   testTimestamp,
			Attribution: attr,
		}},
		Artifacts: []Artifact{{
			ID:             "art-1",
			SessionID:      "sess_test",
			ArtifactType:   ArtifactMilestone,
			Action:         ArtifactActionProposed,
			EntityRef:      "api-quality-gates",
			Title:          "API Quality Gates",
			ProposalID:     "prop-1",
			MutationSource: "agent_sessions.proposal",
			Attribution:    attr,
			CreatedAt:      testTimestamp,
		}},
	}
}
