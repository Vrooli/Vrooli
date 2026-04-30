package operatingmode

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	return &Store{
		InitDir: func(name string) string {
			return filepath.Join(root, "initiatives", name)
		},
		Clock: func() time.Time {
			return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
		},
	}
}

func TestStore_ModeScopedPaths(t *testing.T) {
	store := testStore(t)

	modeDir, err := store.ModeDir("sandboxing", ModeHolisticLoop)
	if err != nil {
		t.Fatalf("ModeDir: %v", err)
	}
	if filepath.ToSlash(modeDir) == "modes/holistic-loop" {
		t.Fatalf("ModeDir returned relative path: %s", modeDir)
	}
	if got := filepath.ToSlash(modeDir); !hasSuffix(got, "initiatives/sandboxing/modes/holistic-loop") {
		t.Fatalf("ModeDir = %s", got)
	}

	roundPath, err := store.RoundPath("sandboxing", ModePhasedPlanDrain, 7)
	if err != nil {
		t.Fatalf("RoundPath: %v", err)
	}
	if got := filepath.ToSlash(roundPath); !hasSuffix(got, "initiatives/sandboxing/modes/phased-plan-drain/rounds/round-007.json") {
		t.Fatalf("RoundPath = %s", got)
	}
}

func TestStore_CreateListLoadRoundPreservesEnvelope(t *testing.T) {
	store := testStore(t)

	created, err := store.CreateRound(RoundEnvelope{
		InitiativeName:  "sandboxing",
		Mode:            string(ModePhasedPlanDrain),
		Phase:           "execute_next",
		RunID:           "run-123",
		Status:          RoundStatusAgentRunning,
		AgentProfileKey: ProfileDeepWork,
		Handoffs: []Handoff{{
			Summary:         "Completed phase 1",
			CompletedPhases: []string{"phase-1"},
			ChangedFiles:    []string{"api/main.go"},
			Tests:           []string{"go test ./..."},
			NextStep:        "Continue with phase 2",
		}},
		ArtifactUpdates: []ArtifactUpdate{{Path: "modes/phased-plan-drain/progress.json", ContentType: "application/json"}},
	})
	if err != nil {
		t.Fatalf("CreateRound: %v", err)
	}
	if created.Round != 1 {
		t.Fatalf("round number = %d, want 1", created.Round)
	}
	if created.RunStrategy != string(RunStrategySequentialHandoff) {
		t.Fatalf("run strategy = %q", created.RunStrategy)
	}

	created.Status = RoundStatusCompleted
	if err := store.SaveRound(created); err != nil {
		t.Fatalf("SaveRound: %v", err)
	}

	second, err := store.CreateRound(RoundEnvelope{
		InitiativeName: "sandboxing",
		Mode:           string(ModePhasedPlanDrain),
		Phase:          "classify_progress",
	})
	if err != nil {
		t.Fatalf("CreateRound second: %v", err)
	}
	if second.Round != 2 {
		t.Fatalf("second round = %d, want 2", second.Round)
	}
	if second.AgentProfileKey != ProfileAnalysis {
		t.Fatalf("defaulted profile = %q, want %q", second.AgentProfileKey, ProfileAnalysis)
	}

	rounds, err := store.ListRounds("sandboxing", ModePhasedPlanDrain)
	if err != nil {
		t.Fatalf("ListRounds: %v", err)
	}
	if len(rounds) != 2 {
		t.Fatalf("round count = %d, want 2", len(rounds))
	}
	if got := rounds[0].Handoffs[0].CompletedPhases[0]; got != "phase-1" {
		t.Fatalf("handoff not preserved: %q", got)
	}
	if rounds[0].AgentProfileKey != ProfileDeepWork {
		t.Fatalf("profile not preserved: %q", rounds[0].AgentProfileKey)
	}

	loaded, err := store.LoadRound("sandboxing", ModePhasedPlanDrain, 1)
	if err != nil {
		t.Fatalf("LoadRound: %v", err)
	}
	if loaded.Status != RoundStatusCompleted {
		t.Fatalf("loaded status = %q", loaded.Status)
	}
}

func TestStore_ListRoundsRejectsMalformedJSON(t *testing.T) {
	store := testStore(t)
	dir, err := store.RoundsDir("sandboxing", ModeHolisticLoop)
	if err != nil {
		t.Fatalf("RoundsDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "round-001.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := store.ListRounds("sandboxing", ModeHolisticLoop); err == nil {
		t.Fatal("ListRounds accepted malformed JSON")
	}
}

func TestStore_LoadRoundNotFound(t *testing.T) {
	store := testStore(t)
	_, err := store.LoadRound("sandboxing", ModeHolisticLoop, 42)
	if !errors.Is(err, ErrRoundNotFound) {
		t.Fatalf("LoadRound err = %v, want ErrRoundNotFound", err)
	}
}

func hasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
