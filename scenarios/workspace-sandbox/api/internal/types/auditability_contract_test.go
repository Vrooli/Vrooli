// Tests for the auditability-contract types added by
// execute/agent-manager-sandbox-auto-apply-defaults Phase 2:
//   - ApprovalSource enum (incl. system-only SourceWorkspaceSandboxGC)
//   - JSON round-trip for ApprovalRequest.Source and ApplyAtRunEndRequest
//   - ProvenanceFileState IsValid
//
// See scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md and
// scenarios/swarm-manager/execute/agent-manager-sandbox-auto-apply-defaults/plan.md
// (Decisions D6, D8) for the contract these tests pin.

package types

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestApprovalSource_IsValid(t *testing.T) {
	cases := []struct {
		s     ApprovalSource
		valid bool
	}{
		{SourceUnspecified, true},
		{SourceAgentManagerAutoApply, true},
		{SourceGitControlTower, true},
		{SourceAgentManagerUI, true},
		{SourceWorkspaceSandboxUI, true},
		{SourceCLI, true},
		{SourceWorkspaceSandboxGC, true},
		{ApprovalSource("totally-bogus"), false},
	}
	for _, c := range cases {
		if got := c.s.IsValid(); got != c.valid {
			t.Errorf("ApprovalSource(%q).IsValid() = %v, want %v", c.s, got, c.valid)
		}
	}
}

func TestApprovalSource_IsValidInbound(t *testing.T) {
	cases := []struct {
		s            ApprovalSource
		inboundOK    bool
		whyOnFailure string
	}{
		{SourceAgentManagerAutoApply, true, ""},
		{SourceGitControlTower, true, ""},
		{SourceAgentManagerUI, true, ""},
		{SourceWorkspaceSandboxUI, true, ""},
		{SourceCLI, true, ""},
		// System-only — never accepted on inbound requests.
		{SourceWorkspaceSandboxGC, false, "system-initiated; agents/operators must not claim system identity"},
		{SourceUnspecified, false, "must specify a source explicitly"},
		{ApprovalSource("invented"), false, "unknown enum value"},
	}
	for _, c := range cases {
		if got := c.s.IsValidInbound(); got != c.inboundOK {
			t.Errorf("ApprovalSource(%q).IsValidInbound() = %v, want %v (%s)",
				c.s, got, c.inboundOK, c.whyOnFailure)
		}
	}
}

func TestApprovalRequest_SourceRoundTrip(t *testing.T) {
	req := ApprovalRequest{
		SandboxID: uuid.New(),
		Mode:      "all",
		Source:    SourceGitControlTower,
		Actor:     "operator-1",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ApprovalRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Source != SourceGitControlTower {
		t.Errorf("decoded Source = %q, want %q", decoded.Source, SourceGitControlTower)
	}
}

func TestApplyAtRunEndRequest_RoundTrip(t *testing.T) {
	req := ApplyAtRunEndRequest{
		SandboxID:         uuid.New(),
		AgentManagerRunID: "run-abc-123",
		ConversationID:    "conv-xyz-456",
		Cost:              0.42,
		RunOutcome:        "success",
		Source:            SourceAgentManagerAutoApply,
		Actor:             "auto-apply",
		CreateCommit:      true,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ApplyAtRunEndRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.AgentManagerRunID != req.AgentManagerRunID {
		t.Errorf("AgentManagerRunID round-trip lost: got %q, want %q", decoded.AgentManagerRunID, req.AgentManagerRunID)
	}
	if decoded.ConversationID != req.ConversationID {
		t.Errorf("ConversationID round-trip lost: got %q, want %q", decoded.ConversationID, req.ConversationID)
	}
	if decoded.Cost != req.Cost {
		t.Errorf("Cost round-trip lost: got %v, want %v", decoded.Cost, req.Cost)
	}
	if decoded.RunOutcome != req.RunOutcome {
		t.Errorf("RunOutcome round-trip lost: got %q, want %q", decoded.RunOutcome, req.RunOutcome)
	}
	if decoded.Source != req.Source {
		t.Errorf("Source round-trip lost: got %q, want %q", decoded.Source, req.Source)
	}
}

func TestProvenanceFileState_IsValid(t *testing.T) {
	cases := []struct {
		s     ProvenanceFileState
		valid bool
	}{
		{"", true}, // empty allowed for legacy rows
		{ProvenanceFileStateApplied, true},
		{ProvenanceFileStatePendingReview, true},
		{ProvenanceFileStateDenied, true},
		{ProvenanceFileState("inflight"), false},
	}
	for _, c := range cases {
		if got := c.s.IsValid(); got != c.valid {
			t.Errorf("ProvenanceFileState(%q).IsValid() = %v, want %v", c.s, got, c.valid)
		}
	}
}

func TestAuditEvent_SourceRoundTrip(t *testing.T) {
	id := uuid.New()
	sandboxID := uuid.New()
	evt := AuditEvent{
		ID:        id,
		SandboxID: &sandboxID,
		EventType: "approved",
		Actor:     "operator-2",
		ActorType: "user",
		Source:    SourceWorkspaceSandboxUI,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded AuditEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Source != SourceWorkspaceSandboxUI {
		t.Errorf("AuditEvent.Source round-trip lost: got %q, want %q",
			decoded.Source, SourceWorkspaceSandboxUI)
	}
}
