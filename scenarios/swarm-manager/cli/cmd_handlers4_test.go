package main

import (
	"net/http"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdPromptsSkills_FiltersByContains(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/prompts/skills" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"items":[{"id":"react-skill","name":"React","usage_type":"steer"},{"id":"go-skill","name":"Go","usage_type":"steer"}]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdPromptsSkills([]string{"--contains", "react"})
	})
	if !strings.Contains(out, "Found 1 prompt skill(s)") || !strings.Contains(out, "react-skill") {
		t.Errorf("filtered output = %q", out)
	}
	if strings.Contains(out, "go-skill") {
		t.Errorf("filter should exclude go-skill: %q", out)
	}
}

func TestCmdPromptsSkills_EmptyAfterFilter(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"go-skill","name":"Go","usage_type":"steer"}]}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdPromptsSkills([]string{"--contains", "nomatch"})
	})
	if !strings.Contains(out, "No prompt skills found.") {
		t.Errorf("empty output = %q", out)
	}
}

func TestCmdAgentManagerRunGet_RequiresID(t *testing.T) {
	app := newAppT(t)
	if err := app.cmdAgentManagerRunGet([]string{}); err == nil || !strings.Contains(err.Error(), "--id is required") {
		t.Fatalf("expected --id required, got %v", err)
	}
}

func TestCmdAgentManagerRunGet_RendersActiveRun(t *testing.T) {
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/agent-manager/runs/run-1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"run_id":"run-1","task_id":"task-1","status":"running","active":true,"started_at":"t0","duration_seconds":12.3}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentManagerRunGet([]string{"--id", "run-1"})
	})
	if !strings.Contains(out, "Run run-1 (running)") {
		t.Errorf("summary missing: %q", out)
	}
	if !strings.Contains(out, "Duration: 12.3s") || !strings.Contains(out, "Task ID: task-1") {
		t.Errorf("details missing: %q", out)
	}
	// active run shows run-stop next step.
	if !strings.Contains(out, "run-stop") {
		t.Errorf("active run should show run-stop hint: %q", out)
	}
}

func TestCmdAgentManagerRunStop_PostsAndRenders(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"run_id":"run-9","stopped":true}`))
	}))
	app := newAppT(t)
	out := clitest.CaptureStdout(t, func() error {
		return app.cmdAgentManagerRunStop([]string{"--id", "run-9"})
	})
	if method != http.MethodPost || path != "/api/v1/agent-manager/runs/run-9/stop" {
		t.Errorf("request = %s %s", method, path)
	}
	if !strings.Contains(out, "Stop requested successfully.") {
		t.Errorf("output = %q", out)
	}
}
