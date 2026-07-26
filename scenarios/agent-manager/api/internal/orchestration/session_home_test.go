package orchestration

import (
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestPrepareCodecSessionHome_PersistsCodexRolloutAcrossTurns(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("AM_SQLITE_PATH", filepath.Join(temp, ".vrooli", "data", "agent-manager.db"))
	shared := filepath.Join(temp, ".codex")
	if err := os.MkdirAll(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shared, "auth.json"), []byte(`{"token":"test"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	runID := uuid.New()
	runRoot := filepath.Join(temp, "runs")
	env, err := PrepareCodecSessionHome(runRoot, runID, domain.RunnerTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	home := env["CODEX_HOME"]
	if home == "" {
		t.Fatal("CODEX_HOME was not set")
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); err != nil {
		t.Fatalf("seeded auth missing: %v", err)
	}

	rollout := filepath.Join(home, "sessions", "2026", "07", "25", "rollout-thread.jsonl")
	if err := os.MkdirAll(filepath.Dir(rollout), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rollout, []byte("turn one"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondTurnEnv, err := PrepareCodecSessionHome(runRoot, runID, domain.RunnerTypeCodex)
	if err != nil {
		t.Fatal(err)
	}
	if secondTurnEnv["CODEX_HOME"] != home {
		t.Fatalf("second turn CODEX_HOME = %q, want %q", secondTurnEnv["CODEX_HOME"], home)
	}
	if got, err := os.ReadFile(rollout); err != nil || string(got) != "turn one" {
		t.Fatalf("second turn cannot read first rollout: got %q, err=%v", got, err)
	}

	if err := CleanupCodecSessionHomeCredentials(runRoot, runID, domain.RunnerTypeCodex); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "auth.json")); !os.IsNotExist(err) {
		t.Fatalf("seeded auth retained after cleanup: %v", err)
	}
	if _, err := os.Stat(rollout); err != nil {
		t.Fatalf("rollout removed by credential cleanup: %v", err)
	}
}
