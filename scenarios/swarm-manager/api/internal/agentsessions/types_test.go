package agentsessions

import (
	"encoding/json"
	"errors"
	"testing"

	"swarm-manager/internal/identity"

	"google.golang.org/protobuf/encoding/protojson"
)

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
		Type:        identity.TypeAgent,
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
		ProposalTarget: &ProposalTarget{Type: ContextInitiative, Ref: "api-quality-gates", Name: "API Quality Gates"},
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
			Summary:     "Create the API quality gate initiative.",
			PayloadJSON: `{"items":[]}`,
			CreatedAt:   testTimestamp,
			UpdatedAt:   testTimestamp,
			Attribution: attr,
		}},
		Artifacts: []Artifact{{
			ID:             "art-1",
			SessionID:      "sess_test",
			ArtifactType:   ArtifactInitiative,
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
