package goal

import (
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestRunLoadsGoalLoopAndCarriesSentenceWithoutCadence(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/read", readResponse{Skills: []struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}{{ID: "goal-loop", Content: "phase 0"}}})
	stdout, _, err := clitest.Output(t, func() error {
		return run(ctx, []string{"improve program-runtime"})
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(stdout, "Goal: improve program-runtime") || !strings.Contains(stdout, "caller-owned (none supplied)") || !strings.Contains(stdout, "phase 0") {
		t.Fatalf("output=%q", stdout)
	}
	payload, ok := ctx.LastRequest().Payload.(readRequest)
	if !ok || len(payload.Identifiers) != 1 || payload.Identifiers[0] != "goal-loop" {
		t.Fatalf("goal-loop skill was not requested: %+v", ctx.LastRequest())
	}
}

func TestRunPreservesCallerCadence(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("POST", "/skills/read", readResponse{Skills: []struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}{{ID: "goal-loop", Content: "phase 0"}}})
	stdout, _, err := clitest.Output(t, func() error {
		return run(ctx, []string{"--cadence", "15m", "improve"})
	})
	if err != nil || !strings.Contains(stdout, "Cadence: 15m") {
		t.Fatalf("stdout=%q err=%v", stdout, err)
	}
}
