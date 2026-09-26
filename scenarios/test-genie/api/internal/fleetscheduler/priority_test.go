package fleetscheduler

import (
	"context"
	"testing"
	"time"
)

func TestCLIPrioritySourceParsesScoreRows(t *testing.T) {
	// Field names mirror the real SCS protojson output, which uses proto names
	// (snake_case): see `scenario-completeness-scoring score list --json`.
	raw := []byte(`{
		"scores": [
			{"scenario":"alpha","importance":0.8,"priority":40.0,"last_run_at":"2026-06-19T10:00:00Z","last_status":"passed"},
			{"scenario":"beta","importance":0.2,"priority":5.0},
			{"scenario":"  ","priority":99}
		],
		"next_page_token": "1"
	}`)
	src := &cliPrioritySource{run: func(context.Context) ([]byte, error) { return raw, nil }}
	got, err := src.Candidates(context.Background())
	if err != nil {
		t.Fatalf("Candidates() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d candidates, want 2 (blank scenario dropped): %+v", len(got), got)
	}
	if got[0].Scenario != "alpha" || got[0].Priority != 40.0 || got[0].LastStatus != "passed" {
		t.Fatalf("alpha = %+v", got[0])
	}
	want := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	if !got[0].LastRunAt.Equal(want) {
		t.Fatalf("alpha LastRunAt = %v, want %v", got[0].LastRunAt, want)
	}
	// beta has no recency -> zero time (never tested), not a parse error.
	if !got[1].LastRunAt.IsZero() || got[1].LastStatus != "" {
		t.Fatalf("beta recency = %v/%q, want zero/empty", got[1].LastRunAt, got[1].LastStatus)
	}
}

func TestCLIPrioritySourcePropagatesRunError(t *testing.T) {
	src := &cliPrioritySource{run: func(context.Context) ([]byte, error) { return nil, context.DeadlineExceeded }}
	if _, err := src.Candidates(context.Background()); err == nil {
		t.Fatal("Candidates() error = nil, want propagated run error")
	}
}
