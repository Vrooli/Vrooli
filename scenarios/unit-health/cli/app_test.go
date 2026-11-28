package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapptest"

	"unit-health/cli/internal/testutil"
)

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	output := cliapptest.CaptureStdout(t, func() error {
		if err := app.Run([]string{"--version"}); err != nil {
			return err
		}
		return nil
	})
	testutil.RequireContains(t, output, appVersion)
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	output := cliapptest.CaptureStdout(t, func() error {
		if err := app.Run([]string{"--help"}); err != nil {
			return err
		}
		return nil
	})
	testutil.RequireContains(t, output, appName)
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

// Executed validation delegates its deadline to the server while ordinary
// requests retain their configured client deadline. Exercise the actual
// CLI/Connect transport.
func TestExecutedValidationWaitsForTerminalResponse(t *testing.T) {
	requests := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "ValidateScenario") {
			w.Header().Set("Content-Type", "application/proto")
			body, err := proto.Marshal(&validationv1.ValidateScenarioResponse{RunId: "delayed-validation", Scenario: "demo", Status: "passed"})
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = w.Write(body)
			return
		}
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.HTTPClient = cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Timeout: 30 * time.Second})
	previousTransport := http.DefaultTransport
	transportObservations := make(chan requestDeadlineObservation, 4)
	http.DefaultTransport = recordingRoundTripper{next: previousTransport, observations: transportObservations}
	defer func() { http.DefaultTransport = previousTransport }()
	var runErr error
	output := cliapptest.CaptureStdout(t, func() error {
		runErr = app.Run([]string{"--api-base", server.URL, "validate", "scenario", "demo", "--execution", "--json"})
		return nil
	})
	seen := drainRequests(requests)
	if len(seen) != 2 || seen[0] != "GET /health" || !strings.Contains(seen[1], "ValidateScenario") {
		t.Fatalf("server requests = %v, want health preflight followed by validation", seen)
	}
	deadlines := drainDeadlineObservations(transportObservations)
	if len(deadlines) != 2 || !deadlines[0].hasDeadline || deadlines[1].hasDeadline {
		t.Fatalf("request deadlines = %+v, want bounded health preflight then server-owned validation", deadlines)
	}
	if runErr != nil {
		t.Fatalf("CLI validation returned error: %v", runErr)
	}
	testutil.RequireContains(t, output, "delayed-validation")
}

type requestDeadlineObservation struct {
	path        string
	hasDeadline bool
}

type recordingRoundTripper struct {
	next         http.RoundTripper
	observations chan<- requestDeadlineObservation
}

func (r recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	_, hasDeadline := req.Context().Deadline()
	r.observations <- requestDeadlineObservation{path: req.URL.Path, hasDeadline: hasDeadline}
	return r.next.RoundTrip(req)
}

func drainDeadlineObservations(observations <-chan requestDeadlineObservation) []requestDeadlineObservation {
	var seen []requestDeadlineObservation
	for {
		select {
		case observation := <-observations:
			seen = append(seen, observation)
		default:
			return seen
		}
	}
}

func drainRequests(requests <-chan string) []string {
	var seen []string
	for {
		select {
		case request := <-requests:
			seen = append(seen, request)
		default:
			return seen
		}
	}
}
