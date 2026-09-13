package store

import (
	"context"
	"os"
	"testing"

	"prompt-manager/internal/teamconfig"
)

func finiteConfig() *HeartbeatConfig {
	return &HeartbeatConfig{ProfileKey: "qualified", Schedule: "@hourly", FiniteLeader: &teamconfig.FiniteLeader{
		EffortRef: "new:arbitrary-effort", AcceptedRevision: "revision:7", CoordinatorPromptRef: "prompt-manager:coordinator",
		SourceRefs: []string{"owner:assignment"}}}
}

func TestFiniteLeaderConfigPreservesIdentityAndRetirement(t *testing.T) {
	for _, mutation := range []string{"effort", "revision", "prompt", "sources", "profile", "remove", "supervision", "unretire"} {
		t.Run(mutation, func(t *testing.T) {
			s := setupStateTestStore(t)
			ctx := context.Background()
			cfg := finiteConfig()
			cfg.FiniteLeader.Retired = mutation == "unretire"
			if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", cfg); err != nil {
				t.Fatal(err)
			}
			updated, _ := s.GetHeartbeatConfig(ctx, "team-1", "agent-1")
			switch mutation {
			case "effort":
				updated.FiniteLeader.EffortRef = "another-effort"
			case "revision":
				updated.FiniteLeader.AcceptedRevision = "revision:8"
			case "prompt":
				updated.FiniteLeader.CoordinatorPromptRef = "different-prompt"
			case "sources":
				updated.FiniteLeader.SourceRefs = []string{"different-source"}
			case "profile":
				updated.ProfileKey = "different-profile"
			case "remove":
				updated.FiniteLeader = nil
			case "supervision":
				updated.Supervision = &teamconfig.Supervision{}
			case "unretire":
				updated.FiniteLeader.Retired = false
			}
			if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", updated); err == nil {
				t.Fatal("binding mutation bypassed retained identity")
			}
			if err := s.DeleteHeartbeatConfig(ctx, "team-1", "agent-1"); err == nil {
				t.Fatal("deletion discarded finite owner fence")
			}
			cfg.Enabled, cfg.FiniteLeader.Retired = false, true
			if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", cfg); err != nil {
				t.Fatalf("disable/retire refused: %v", err)
			}
		})
	}
}

func TestFiniteLeaderReservationSurvivesStoreRestartAndDuplicateBinding(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()
	for _, agent := range []string{"agent-1", "agent-2"} {
		if err := s.SetHeartbeatConfig(ctx, "team-1", agent, finiteConfig()); err != nil {
			t.Fatal(err)
		}
	}
	err := s.WithFiniteLeader(ctx, "team-1", "agent-1", func(_ *HeartbeatConfig, state *FiniteLeaderState, save func() error) error {
		state.ID, state.TaskID, state.RunID, state.Status = "operation-1", "task-1", "run-1", "parked"
		state.TaskStarted, state.DispatchStarted = true, true
		return save()
	})
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewFileTeamStore(s.configRoot, s.runtimeDataRoot, s.relationStore)
	state, err := restarted.ReadFiniteLeader(ctx, "team-1", "agent-1")
	if err != nil || state.ID != "operation-1" || state.RunID != "run-1" || state.Status != "parked" {
		t.Fatalf("restart lost exact owner identity: %+v %v", state, err)
	}
	if _, err := restarted.ReadFiniteLeader(ctx, "team-1", "agent-2"); err == nil {
		t.Fatal("same effort admitted through another member")
	}
}

func TestFiniteLeaderUncertainAndInvalidStorageNeverMeansIdle(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()
	if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", finiteConfig()); err != nil {
		t.Fatal(err)
	}
	// A read must not create a reservation; unavailable durable state is an error.
	if _, err := s.ReadFiniteLeader(ctx, "team-1", "agent-1"); err != nil {
		t.Fatal(err)
	}
	path := s.finiteLeaderPath(finiteConfig().FiniteLeader.EffortRef)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("read created admission state")
	}
	if err := SaveJSON(path, &FiniteLeaderState{Version: 99}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadFiniteLeader(ctx, "team-1", "agent-1"); err == nil {
		t.Fatal("incompatible durable state became fresh admission")
	}
}

func TestFiniteLeaderMigrationRequiresSettledUnusedHeartbeat(t *testing.T) {
	for _, old := range []*HeartbeatConfig{{Enabled: true}, {LastExecution: &HeartbeatExecResult{RunID: "old-owner", Status: "failed"}}} {
		s := setupStateTestStore(t)
		ctx := context.Background()
		if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", old); err != nil {
			t.Fatal(err)
		}
		if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", finiteConfig()); err == nil {
			t.Fatal("migration discarded ordinary execution uncertainty")
		}
	}
}
