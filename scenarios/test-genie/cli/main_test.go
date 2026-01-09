package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestExecuteAcceptsPositionalPhases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/phases":
			fmt.Fprintf(w, `{"items":[{"name":"unit","description":"Unit tests"},{"name":"integration","description":"Integration tests"}]}`)
			return
		case "/api/v1/executions":
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			if !bytes.Contains(body, []byte(`"phases":["unit","integration"]`)) {
				t.Fatalf("expected phases in payload, got: %s", string(body))
			}
			fmt.Fprintf(w, `{"success":true,"phases":[{"name":"unit","status":"passed","durationSeconds":1}],"executionId":"abc"}`)
			return
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"execute", "demo", "unit", "integration"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}

func TestExecuteAllPhaseSkipsExplicitList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/phases":
			fmt.Fprintf(w, `{"items":[]}`)
			return
		case "/api/v1/executions":
			body, _ := io.ReadAll(r.Body)
			defer r.Body.Close()
			if bytes.Contains(body, []byte(`"phases"`)) {
				t.Fatalf("expected phases to be omitted when 'all' requested, got: %s", string(body))
			}
			fmt.Fprintf(w, `{"success":true,"phases":[],"executionId":"abc"}`)
			return
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"execute", "demo", "all"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}

func TestConfigureSetsValues(t *testing.T) {
	app := newTestApp(t)
	if err := app.Run([]string{"configure", "api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure api_base: %v", err)
	}
	if app.core.Config.APIBase != "http://example.com" {
		t.Fatalf("expected api base saved")
	}
	if err := app.Run([]string{"configure", "token", "secret"}); err != nil {
		t.Fatalf("configure token: %v", err)
	}
	if app.core.Config.Token != "secret" {
		t.Fatalf("expected token saved")
	}
}

func TestBuildAPIBaseOptionsUsesPortEnv(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("TEST_GENIE_API_PORT", "4567")
	base := cliutil.DetermineAPIBase(app.core.APIBaseOptions())
	if base != "http://localhost:4567" {
		t.Fatalf("expected base from port env, got %s", base)
	}
}

func TestGenerateCommandSendsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/suite-requests" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if !bytes.Contains(body, []byte(`"coverageTarget":80`)) {
			t.Fatalf("expected coverage in payload, got %s", string(body))
		}
		fmt.Fprintf(w, `{"id":"1","scenarioName":"demo","status":"queued","requestedTypes":["unit"]}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"generate", "demo", "--coverage", "80", "--types", "unit"}); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func TestRunTestsCommandSendsType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scenarios/demo/run-tests" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if !bytes.Contains(body, []byte(`"type":"phased"`)) {
			t.Fatalf("expected type in payload, got %s", string(body))
		}
		if !bytes.Contains(body, []byte(`"paths":["api/foo.go"]`)) {
			t.Fatalf("expected paths in payload, got %s", string(body))
		}
		if !bytes.Contains(body, []byte(`"playbooks":["bas/cases/run.json"]`)) {
			t.Fatalf("expected playbooks in payload, got %s", string(body))
		}
		if !bytes.Contains(body, []byte(`"filter":"UserTest"`)) {
			t.Fatalf("expected filter in payload, got %s", string(body))
		}
		fmt.Fprintf(w, `{"type":"phased","status":"ok","command":{"command":["echo"],"workingDir":"."}}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"run-tests", "demo", "--type", "phased", "--path", "api/foo.go", "--playbook", "bas/cases/run.json", "--filter", "UserTest"}); err != nil {
		t.Fatalf("run-tests failed: %v", err)
	}
}

func TestStatusCommandRequestsHealth(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"status":"ok","service":"test-genie","version":"1.0","dependencies":{"db":"up"},"operations":{"queue":{"pending":0,"queued":1,"delegated":0,"running":0,"failed":0,"oldestQueuedAgeSeconds":0}}}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"status"}); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one health call, got %d", calls)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("TEST_GENIE_CONFIG_DIR", temp)
	t.Setenv("TEST_GENIE_API_TOKEN", "test-token")
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}
