package main

import (
	"context"
	"errors"
	"testing"
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

// fakeRunner is a vroolicli.Runner that returns canned bytes and counts calls,
// so List() can be tested without executing the real vrooli CLI.
type fakeRunner struct {
	output []byte
	err    error
	calls  int
}

func (f *fakeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.output, nil
}

// RunCombined satisfies vroolicli.Runner; ScenarioStatuses uses Run, so this
// just mirrors Run for completeness.
func (f *fakeRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return f.Run(ctx, name, args...)
}

func newLocatorWithRunner(r *fakeRunner) *ScenarioLocator {
	return &ScenarioLocator{
		client:   vroolicli.New(vroolicli.WithRunner(r)),
		cacheTTL: time.Minute,
	}
}

func TestScenarioLocatorList_DecodesNamesAndSkipsEmpty(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{
		"success": true,
		"scenarios": [
			{"name": "alpha", "status": "running"},
			{"name": "beta", "status": "stopped"},
			{"name": "", "status": "ghost"}
		]
	}`)}

	got, err := newLocatorWithRunner(runner).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"alpha", "beta"} // empty-name entry skipped
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestScenarioLocatorList_CachesWithinTTL(t *testing.T) {
	runner := &fakeRunner{output: []byte(`{"scenarios":[{"name":"alpha"}]}`)}
	locator := newLocatorWithRunner(runner)

	if _, err := locator.List(context.Background()); err != nil {
		t.Fatalf("first List: %v", err)
	}
	if _, err := locator.List(context.Background()); err != nil {
		t.Fatalf("second List: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected the second List to hit the cache (1 CLI call), got %d", runner.calls)
	}
}

func TestScenarioLocatorList_PropagatesError(t *testing.T) {
	// A CLI failure must surface as an error — never a silently empty list.
	runner := &fakeRunner{err: errors.New("boom")}

	if _, err := newLocatorWithRunner(runner).List(context.Background()); err == nil {
		t.Fatal("expected List to propagate the CLI error, got nil")
	}
}
