package upstreamcheck

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestRunAggregate_MergesAndBuckets(t *testing.T) {
	reports := map[string]Report{
		"resource-codex":       {Name: "codex", Installed: "0.141.0", Latest: "0.141.0", Status: StatusUpToDate},
		"resource-opencode":    {Name: "opencode", Installed: "1.15.7", Latest: "1.17.9", Status: StatusBehind},
		"resource-claude-code": {Name: "claude-code", Status: StatusUnknown},
	}
	run := func(ctx context.Context, args []string) ([]byte, error) {
		rep, ok := reports[args[0]]
		if !ok {
			return nil, errors.New("not found")
		}
		return json.Marshal(rep)
	}
	entries := []AggregateEntry{
		{Name: "codex", CheckCmd: []string{"resource-codex", "upstream-check", "--json"}},
		{Name: "opencode", CheckCmd: []string{"resource-opencode", "upstream-check", "--json"}},
		{Name: "claude-code", CheckCmd: []string{"resource-claude-code", "upstream-check", "--json"}},
	}
	agg := RunAggregate(context.Background(), run, entries)

	if len(agg.Resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(agg.Resources))
	}
	if len(agg.Behind) != 1 || agg.Behind[0] != "opencode" {
		t.Fatalf("behind = %v, want [opencode]", agg.Behind)
	}
	if len(agg.Unknown) != 1 || agg.Unknown[0] != "claude-code" {
		t.Fatalf("unknown = %v, want [claude-code]", agg.Unknown)
	}
}

func TestRunAggregate_DegradesOnRunnerError(t *testing.T) {
	run := func(ctx context.Context, args []string) ([]byte, error) {
		return nil, errors.New("binary not found")
	}
	agg := RunAggregate(context.Background(), run, []AggregateEntry{
		{Name: "codex", CheckCmd: []string{"resource-codex", "upstream-check", "--json"}},
	})
	if len(agg.Resources) != 1 || agg.Resources[0].Status != StatusUnknown {
		t.Fatalf("expected one unknown report, got %+v", agg.Resources)
	}
	if agg.Resources[0].Name != "codex" {
		t.Fatalf("degraded report should keep entry name, got %q", agg.Resources[0].Name)
	}
	if len(agg.Unknown) != 1 {
		t.Fatalf("expected codex bucketed unknown, got %v", agg.Unknown)
	}
}

func TestRunAggregate_DegradesOnBadJSON(t *testing.T) {
	run := func(ctx context.Context, args []string) ([]byte, error) {
		return []byte("not json"), nil
	}
	agg := RunAggregate(context.Background(), run, []AggregateEntry{
		{Name: "opencode", CheckCmd: []string{"resource-opencode"}},
	})
	if agg.Resources[0].Status != StatusUnknown {
		t.Fatalf("bad JSON should degrade to unknown, got %+v", agg.Resources[0])
	}
}
