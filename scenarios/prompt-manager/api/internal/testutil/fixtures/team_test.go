package fixtures

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/internal/teamconfig"
)

func TestWriteTeamMarshalsTeamFixture(t *testing.T) {
	team := IndependentTeam("team-1", "Team One")
	path, err := WriteTeam(t.TempDir(), team)
	if err != nil {
		t.Fatalf("WriteTeam() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var persisted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if persisted.ID != team.ID || filepath.Base(path) != "team.json" {
		t.Fatalf("persisted fixture = %+v at %s", persisted, path)
	}
}

func TestIndependentTeamDefaults(t *testing.T) {
	team := IndependentTeam("team-1", "Team One")

	if team.ID != "team-1" {
		t.Fatalf("expected team ID team-1, got %q", team.ID)
	}
	if team.DisplayName != "Team One" {
		t.Fatalf("expected display name Team One, got %q", team.DisplayName)
	}
	if !team.Enabled || !team.EnabledSet {
		t.Fatalf("expected independent team to be explicitly enabled")
	}
	if team.Runtime.Mode != teamconfig.RuntimeModeMultiProcess {
		t.Fatalf("expected multi-process runtime, got %q", team.Runtime.Mode)
	}
	if team.Coordination.Pattern != teamconfig.CoordinationPatternIndependent {
		t.Fatalf("expected independent coordination, got %q", team.Coordination.Pattern)
	}
	if !team.Coordination.Capabilities.RequireHandoff {
		t.Fatalf("expected independent team to require handoff guidance")
	}
}

func TestIndependentTeamOptions(t *testing.T) {
	team := IndependentTeam(
		"team-1",
		"Team One",
		WithEnabled(false),
		WithContractAgents("agent-a", "agent-b"),
		WithLeadAgent("agent-a"),
	)

	if team.Enabled {
		t.Fatalf("expected WithEnabled(false) to disable team")
	}
	if !team.EnabledSet {
		t.Fatalf("expected WithEnabled to mark enabled as set")
	}
	if team.Coordination.LeadAgentID != "agent-a" {
		t.Fatalf("expected lead agent agent-a, got %q", team.Coordination.LeadAgentID)
	}
	if len(team.OperatingContract.Members) != 2 {
		t.Fatalf("expected operating contract for 2 members, got %d", len(team.OperatingContract.Members))
	}
}

func TestLeaderLedSingleProcessTeamDefaults(t *testing.T) {
	team := LeaderLedSingleProcessTeam("team-1", "Team One", "lead-1")

	if team.Runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Fatalf("expected single-process runtime, got %q", team.Runtime.Mode)
	}
	if team.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		t.Fatalf("expected leader-led coordination, got %q", team.Coordination.Pattern)
	}
	if team.Coordination.LeadAgentID != "lead-1" {
		t.Fatalf("expected lead agent lead-1, got %q", team.Coordination.LeadAgentID)
	}
	if team.Coordination.ReportingMode != teamconfig.ReportingModeLeader {
		t.Fatalf("expected leader reporting mode, got %q", team.Coordination.ReportingMode)
	}
	if team.Coordination.MessagingMode != teamconfig.MessagingModeInSession {
		t.Fatalf("expected in-session messaging, got %q", team.Coordination.MessagingMode)
	}
	if team.Execution.QueuePolicy != teamconfig.QueuePolicySerialized {
		t.Fatalf("expected serialized queue policy, got %q", team.Execution.QueuePolicy)
	}
	if team.Execution.MaxConcurrentRuns != 1 {
		t.Fatalf("expected max concurrent runs 1, got %d", team.Execution.MaxConcurrentRuns)
	}
}
