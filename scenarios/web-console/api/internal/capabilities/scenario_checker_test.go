package capabilities

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestScenarioChecker_Healthy(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"success":true,"scenario":{"name":"audio-tools","status":"running","health_status":"healthy"}}`), nil
		},
	}
	status, msg := c.Check(context.Background())
	if status != StatusAvailable {
		t.Fatalf("status = %q, want %q (msg=%q)", status, StatusAvailable, msg)
	}
}

func TestScenarioChecker_Stopped(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"success":true,"scenario":{"name":"audio-tools","status":"stopped","health_status":null}}`), nil
		},
	}
	result := c.CheckResult(context.Background())
	if result.Status != StatusUnavailable {
		t.Fatalf("status = %q, want %q (msg=%q)", result.Status, StatusUnavailable, result.Message)
	}
	if result.ReasonCode != "scenario_stopped" {
		t.Fatalf("reason = %q, want scenario_stopped", result.ReasonCode)
	}
	if result.ActionKind != ActionKindScenarioStart {
		t.Fatalf("action kind = %q, want %q", result.ActionKind, ActionKindScenarioStart)
	}
}

func TestScenarioChecker_CLIMissing(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("exec: vrooli: not found")
		},
	}
	status, msg := c.Check(context.Background())
	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", status, StatusUnavailable)
	}
	if msg == "" {
		t.Errorf("expected non-empty hint message when CLI is missing")
	}
}

func TestScenarioChecker_NoSlug(t *testing.T) {
	c := &ScenarioChecker{}
	status, _ := c.Check(context.Background())
	if status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", status, StatusUnavailable)
	}
}

func TestScenarioChecker_UnknownStatus(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"success":true,"scenario":{"name":"audio-tools","status":"weird"}}`), nil
		},
		Timeout: 100 * time.Millisecond,
	}
	status, _ := c.Check(context.Background())
	if status != StatusUnknown {
		t.Fatalf("status = %q, want %q", status, StatusUnknown)
	}
}

func TestScenarioChecker_LegacyListShape(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"scenarios":[{"name":"audio-tools","status":"running","health_status":"healthy"}]}`), nil
		},
	}
	status, msg := c.Check(context.Background())
	if status != StatusAvailable {
		t.Fatalf("status = %q, want %q (msg=%q)", status, StatusAvailable, msg)
	}
}

func TestScenarioChecker_StartingOperation(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"success":true,"scenario":{"name":"audio-tools","status":"stopped","start_operation":{"status":"running","current_step":"health"}}}`), nil
		},
	}
	result := c.CheckResult(context.Background())
	if result.Status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", result.Status, StatusUnavailable)
	}
	if result.ReasonCode != "scenario_start_in_progress" {
		t.Fatalf("reason = %q, want scenario_start_in_progress", result.ReasonCode)
	}
	if result.OperatorCommand != "vrooli scenario wait audio-tools --json" {
		t.Fatalf("operator command = %q", result.OperatorCommand)
	}
}

func TestScenarioChecker_Degraded(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"success":true,"scenario":{"name":"audio-tools","status":"running","health_status":"degraded","health_error":"tts provider down"}}`), nil
		},
	}
	result := c.CheckResult(context.Background())
	if result.Status != StatusUnavailable {
		t.Fatalf("status = %q, want %q", result.Status, StatusUnavailable)
	}
	if result.ReasonCode != "scenario_degraded" {
		t.Fatalf("reason = %q, want scenario_degraded", result.ReasonCode)
	}
	if result.ActionKind != ActionKindScenarioRestart {
		t.Fatalf("action kind = %q, want %q", result.ActionKind, ActionKindScenarioRestart)
	}
}

func TestScenarioChecker_DegradedWithoutHealthErrorStillExplainsNextStep(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`{"success":true,"scenario":{"name":"audio-tools","status":"running","health_status":"degraded","health_error":""}}`), nil
		},
	}
	result := c.CheckResult(context.Background())
	if result.Status != StatusUnavailable || result.Message == "scenario is degraded" {
		t.Fatalf("result = %+v, want an actionable degraded reason", result)
	}
	if !strings.Contains(result.Message, "reported degraded without a reason") {
		t.Fatalf("message = %q, want factual degraded reason", result.Message)
	}
}

func TestScenarioChecker_MalformedJSON(t *testing.T) {
	c := &ScenarioChecker{
		Slug: "audio-tools",
		Run: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(`not json`), nil
		},
	}
	result := c.CheckResult(context.Background())
	if result.Status != StatusUnknown {
		t.Fatalf("status = %q, want %q", result.Status, StatusUnknown)
	}
	if result.ReasonCode != "scenario_status_malformed_json" {
		t.Fatalf("reason = %q, want scenario_status_malformed_json", result.ReasonCode)
	}
}
