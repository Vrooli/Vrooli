package usage

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

const runtimeFrictionFixture = `{"window":{"since":"2026-11-09T00:00:00Z","until":"2026-11-16T00:00:00Z"},"coverage":{"state":"partial","reason":"imported and native runs included","runsObserved":4,"importedRuns":2,"importedRunsIncluded":true,"unknownRuns":1,"unclassifiedBlockedRuns":1},"totals":{"totalDurationMs":3600000,"knownDurationMs":2400000,"unknownDurationMs":1200000},"categories":[{"category":"waiting_on_validation","durationMs":60000,"runs":1,"evidenceCount":1}],"blockedChecks":[{"checkId":"","check":"go test ./...","runs":1,"occurrences":1,"responses":{"stopped":0,"workaround":0,"asked":0,"succeeded":0,"unknown":1},"runIds":["run-1"]},{"checkId":"git_verb_blocked","check":"git_verb_blocked","runs":2,"occurrences":3,"outcomes":{"blocked":3,"allowed":0,"unknown":0},"enforcementDurationMs":27,"enforcementDurationKnown":2,"enforcementDurationUnknown":1,"responses":{"stopped":1,"workaround":1,"asked":1,"succeeded":0,"unknown":0},"runIds":["run-2","run-3"]}]}`

func runRuntimeFriction(t *testing.T, base string, args ...string) (string, error) {
	t.Helper()
	cmd := runtimeFrictionCommand(func(*cliapp.ScenarioApp) (paceHTTPDoer, string) {
		return http.DefaultClient, base
	})
	var stdout, stderr bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &stderr)
	if err != nil {
		return "", err
	}
	if err := cmd.RunCtx(ctx); err != nil {
		return stdout.String(), err
	}
	return stdout.String(), nil
}

func TestRuntimeFrictionCommandUsesBoundedWindowAndReturnsOwnerJSON(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, runtimeFrictionFixture)
	}))
	defer server.Close()

	out, err := runRuntimeFriction(t, server.URL, "--since", "2026-11-09T00:00:00Z", "--until", "2026-11-16T00:00:00Z", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != runtimeFrictionAPIPath+"?since=2026-11-09T00%3A00%3A00Z&until=2026-11-16T00%3A00%3A00Z" {
		t.Fatalf("request = %s %s", method, path)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("--json is not the owner API response: %v\n%s", err, out)
	}
	coverage := decoded["coverage"].(map[string]any)
	if coverage["importedRunsIncluded"] != true || coverage["importedRuns"] != float64(2) {
		t.Fatalf("imported-run coverage was lost: %+v", coverage)
	}
}

func TestRuntimeFrictionCommandExplainsUnknownCoverageAndIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, runtimeFrictionFixture)
	}))
	defer server.Close()

	out, err := runRuntimeFriction(t, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"2 imported included", "40m0s known, 20m0s unknown", "waiting_on_validation", "no evaluator ID recorded", "unknown 1", "checkId=git_verb_blocked", "2 runs, 3 occurrences", "enforcement blocked 3", "enforcement request 27ms measured across 2 occurrences, 1 unknown"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q:\n%s", want, out)
		}
	}
}

func TestRuntimeFrictionRejectsOversizedResponseBeforeEmittingPartialJSON(t *testing.T) {
	const maxBytes = 2 << 20
	const first = `{"coverage":{"state":"complete"}}`
	body := first + strings.Repeat(" ", maxBytes-len(first)) + `{"coverage":{"state":"corrupt-tail"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	out, err := runRuntimeFriction(t, server.URL, "--json")
	if err == nil || !strings.Contains(err.Error(), "2 MiB") {
		t.Fatalf("oversized response error = %v, want 2 MiB limit", err)
	}
	if out != "" {
		t.Fatalf("partial JSON was written before rejecting oversized response (%d bytes)", len(out))
	}
}

func TestRuntimeFrictionTruncatesOversizedHTTPErrorBodyButKeepsStatus(t *testing.T) {
	const maxBytes = 2 << 20
	const first = `{"error":"unavailable"}`
	body := first + strings.Repeat(" ", maxBytes-len(first)+128) + "hidden tail"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	_, err := runRuntimeFriction(t, server.URL)
	if err == nil || err.Error() != "runtime friction request failed (502): "+first {
		t.Fatalf("error = %q, want bounded body and original HTTP status", err)
	}
}
