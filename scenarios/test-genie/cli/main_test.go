package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"test-genie/cli/execute"
	"test-genie/cli/generate"
)

func TestExecuteAcceptsPositionalPhases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			fmt.Fprintf(w, `{"status":"ok","service":"test-genie"}`)
			return
		case "/api/v1/executions/plan":
			fmt.Fprintf(w, `{"scenarioName":"demo","phases":[{"name":"unit","estimatedDurationSeconds":1,"timeoutSeconds":60},{"name":"integration","estimatedDurationSeconds":2,"timeoutSeconds":120}],"summary":{"phaseCount":2,"estimatedDurationSeconds":3,"timeoutSeconds":180}}`)
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
		case "/health":
			fmt.Fprintf(w, `{"status":"ok","service":"test-genie"}`)
			return
		case "/api/v1/executions/plan":
			fmt.Fprintf(w, `{"scenarioName":"demo","phases":[{"name":"structure","estimatedDurationSeconds":1,"timeoutSeconds":60},{"name":"standards","estimatedDurationSeconds":1,"timeoutSeconds":60}],"summary":{"phaseCount":2,"estimatedDurationSeconds":2,"timeoutSeconds":120}}`)
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
		switch r.URL.Path {
		case "/health":
			fmt.Fprintf(w, `{"status":"ok","service":"test-genie"}`)
			return
		case "/api/v1/suite-requests":
		default:
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
		switch r.URL.Path {
		case "/health":
			fmt.Fprintf(w, `{"status":"ok","service":"test-genie"}`)
			return
		case "/api/v1/scenarios/demo/run-tests":
		default:
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
	if calls != 2 {
		t.Fatalf("expected two health calls, got %d", calls)
	}
}

func TestExecuteCommandHelpDoesNotCallAPI(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"execute", "--help"}); err != nil {
			t.Fatalf("execute help failed: %v", err)
		}
	})
	if !strings.Contains(output, execute.UsageLine) {
		t.Fatalf("expected execute help output, got %s", output)
	}
}

func TestGenerateCommandHelpDoesNotCallAPI(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"generate", "--help"}); err != nil {
			t.Fatalf("generate help failed: %v", err)
		}
	})
	if !strings.Contains(output, generate.UsageLine) {
		t.Fatalf("expected generate help output, got %s", output)
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
