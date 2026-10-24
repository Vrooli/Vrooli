package scenarios

import (
	"context"
	"reflect"
	"strings"
	"testing"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// recordingRunner captures the argv the client invokes and returns a canned
// payload, so we can assert both the wire contract decode and the command shape.
type recordingRunner struct {
	output  string
	err     error
	gotName string
	gotArgs []string
}

func (r *recordingRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	r.gotName = name
	r.gotArgs = append([]string(nil), args...)
	if r.err != nil {
		return nil, r.err
	}
	return []byte(r.output), nil
}

func (r *recordingRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.Run(ctx, name, args...)
}

func withStubClient(t *testing.T, runner vroolicli.Runner) {
	t.Helper()
	prev := cliClient
	cliClient = vroolicli.New(vroolicli.WithRunner(runner))
	t.Cleanup(func() { cliClient = prev })
}

func TestVrooliScenarioListerMapsTypedFields(t *testing.T) {
	runner := &recordingRunner{output: `{
		"success": true,
		"scenarios": [
			{"name": "alpha", "description": "first", "status": "running", "tags": ["a", "b"]},
			{"name": "  ", "description": "blank name skipped"},
			{"name": "beta", "status": "stopped"}
		]
	}`}
	withStubClient(t, runner)

	got, err := NewVrooliScenarioLister().ListScenarios(context.Background())
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}

	want := []ScenarioMetadata{
		{Name: "alpha", Description: "first", Status: "running", Tags: []string{"a", "b"}},
		{Name: "beta", Description: "", Status: "stopped", Tags: nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestVrooliScenarioListerPassesNoGlobalFlags(t *testing.T) {
	runner := &recordingRunner{output: `{"success":true,"scenarios":[]}`}
	withStubClient(t, runner)

	if _, err := NewVrooliScenarioLister().ListScenarios(context.Background()); err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}

	if runner.gotName != "vrooli" {
		t.Fatalf("command = %q, want vrooli", runner.gotName)
	}
	wantArgs := []string{"scenario", "list", "--json"}
	if !reflect.DeepEqual(runner.gotArgs, wantArgs) {
		t.Fatalf("args = %v, want %v", runner.gotArgs, wantArgs)
	}
	if strings.HasPrefix(runner.gotArgs[0], "--") {
		t.Fatalf("args = %v, want no leading global flag", runner.gotArgs)
	}
}

func TestVrooliScenarioListerPropagatesError(t *testing.T) {
	withStubClient(t, &recordingRunner{err: context.DeadlineExceeded})

	if _, err := NewVrooliScenarioLister().ListScenarios(context.Background()); err == nil {
		t.Fatalf("expected error to propagate")
	}
}
