package onboarding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPHandoffClientIsInertWhenEndpointUnset(t *testing.T) {
	selection, err := (HTTPHandoffClient{}).Resolve(context.Background(), HandoffRequest{NodeID: "node-1"})
	require.NoError(t, err)
	require.False(t, selection.Apply)
}

func TestHTTPHandoffClientCarriesIdentityAndDecodesSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		var request HandoffRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, HandoffRequest{MachineID: "machine-1", NodeID: "node-1", NodeKind: "agent"}, request)
		_, _ = w.Write([]byte(`{"scenarios":["demo"],"apply":true}`))
	}))
	defer server.Close()

	selection, err := (HTTPHandoffClient{Endpoint: server.URL}).Resolve(context.Background(), HandoffRequest{MachineID: "machine-1", NodeID: "node-1", NodeKind: "agent"})
	require.NoError(t, err)
	require.True(t, selection.Apply)
	require.Equal(t, []string{"demo"}, selection.Scenarios)
}

type fakeRunner struct {
	command string
	result  Result
}

func (f *fakeRunner) Run(_ context.Context, _ Target, command string) (Result, error) {
	f.command = command
	return f.result, nil
}

func TestApplySendsCapabilityDocumentWithoutArgvJSON(t *testing.T) {
	selection := Selection{Scenarios: []string{"alpha", "beta"}, OptionalResources: []string{"ollama"}, Apply: true}
	runner := &fakeRunner{result: Result{ExitCode: 0, Stdout: `{"status":"applied"}`}}
	result, err := Apply(context.Background(), runner, Target{Host: "node"}, selection)
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if result.ExitCode != 0 || !strings.Contains(runner.command, "vrooli-onboarding wizard apply") {
		t.Fatalf("unexpected remote result/command: %+v %q", result, runner.command)
	}
	if strings.Contains(runner.command, `"alpha"`) || strings.Contains(runner.command, "ollama") {
		t.Fatalf("selection JSON leaked into command argv: %q", runner.command)
	}
	if !strings.Contains(runner.command, "mktemp") || !strings.Contains(runner.command, "base64") {
		t.Fatalf("command does not use a private stdin/file handoff: %q", runner.command)
	}
}

func TestFromSetupProfile(t *testing.T) {
	selection, ok := FromSetupProfile("alpha,beta", "ollama", false)
	if !ok || len(selection.Scenarios) != 2 || len(selection.OptionalResources) != 1 {
		t.Fatalf("selection=%+v, ok=%t", selection, ok)
	}
	if _, ok := FromSetupProfile("none", "none", false); ok {
		t.Fatal("empty profile should not trigger remote onboarding")
	}
}

func TestReadinessExitCode(t *testing.T) {
	if code, err := ReadinessExitCode(Result{ExitCode: 0}, nil); err != nil || code != 0 {
		t.Fatalf("green result = %d, %v", code, err)
	}
	if code, err := ReadinessExitCode(Result{ExitCode: 2}, nil); err != nil || code != 2 {
		t.Fatalf("remote readiness result = %d, %v", code, err)
	}
	if code, err := ReadinessExitCode(Result{}, context.DeadlineExceeded); err == nil || code != 75 {
		t.Fatalf("transport failure = %d, %v", code, err)
	}
}

func TestApplyAndReadinessReturnsAuthoritativeRemoteReport(t *testing.T) {
	runner := &sequenceRunner{results: []Result{
		{ExitCode: 0, Stdout: `{"status":"applied"}`},
		{ExitCode: 0, Stdout: `{"status":"ready"}`},
	}}
	result, err := ApplyAndReadiness(context.Background(), runner, Target{Host: "example.test"}, Selection{Scenarios: []string{"demo"}, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.Stdout != `{"status":"ready"}` {
		t.Fatalf("readiness result = %+v", result)
	}
	if len(runner.commands) != 2 || !strings.Contains(runner.commands[1], "vrooli-onboarding readiness --json") {
		t.Fatalf("commands = %#v", runner.commands)
	}
}

type sequenceRunner struct {
	results  []Result
	commands []string
}

func (r *sequenceRunner) Run(_ context.Context, _ Target, command string) (Result, error) {
	r.commands = append(r.commands, command)
	result := r.results[0]
	r.results = r.results[1:]
	return result, nil
}
