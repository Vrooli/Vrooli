package runtimeapp

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

func TestRunHelpUsesRuntimeContract(t *testing.T) {
	var output bytes.Buffer
	app := &App{Version: "test"}
	ctx := &CommandContext{Stdout: &output}

	if err := app.Run(ctx, []string{"--help"}); err != nil {
		t.Fatalf("Run(--help): %v", err)
	}
	if output.String() != HelpText {
		t.Fatalf("help output differs from runtime contract")
	}
}

func TestSupervisorStatusJSONUsesTypedProtoShape(t *testing.T) {
	pid := 4242
	report := runtimesupervisor.StatusReport{
		SupervisorID:                  "sup-1",
		Status:                        "running",
		StatusReason:                  "healthy",
		PID:                           &pid,
		LastHeartbeatAt:               time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
		EffectiveRenewInterval:        10 * time.Second,
		EffectiveMaxHealthConcurrency: 16,
	}

	var output bytes.Buffer
	if err := WriteSupervisorStatusJSON(&output, report); err != nil {
		t.Fatalf("WriteSupervisorStatusJSON: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("typed output is not JSON: %v", err)
	}
	if got["supervisor_id"] != "sup-1" || got["status"] != "running" {
		t.Fatalf("unexpected supervisor identity: %v", got)
	}
	if got["last_heartbeat_at"] != "2026-06-11T12:00:00Z" {
		t.Fatalf("unexpected heartbeat timestamp: %v", got["last_heartbeat_at"])
	}
	if got["effective_renew_interval"] != "10000000000" {
		t.Fatalf("int64 duration must use protojson string form: %v", got["effective_renew_interval"])
	}
}

func TestRunRejectsUnknownRuntimeCommand(t *testing.T) {
	ctx := &CommandContext{Stdout: &bytes.Buffer{}}
	if err := (&App{}).Run(ctx, []string{"unknown"}); err == nil {
		t.Fatal("Run should reject an unknown runtime command")
	}
}
