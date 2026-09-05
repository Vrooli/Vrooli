package upstreamcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// AggregateEntry binds a resource display name to the command that emits its
// per-resource `upstream-check --json` Report (e.g.
// ["resource-codex","upstream-check","--json"]). Running the resource CLI
// keeps each resource's source/pin config as the single source of truth — the
// aggregator never re-derives it.
type AggregateEntry struct {
	Name     string
	CheckCmd []string
}

// AggregateReport is the combined verdict across several resources, with the
// behind/unknown names pulled out so callers can act on drift at a glance.
type AggregateReport struct {
	Resources []Report          `json:"resources"`
	Behind    []string          `json:"behind"`
	Unknown   []string          `json:"unknown"`
	Artifacts []ArtifactFinding `json:"artifacts,omitempty"`
}

// ArtifactFinding is a read-only liveness observation for a resource
// acquisition target. It is optional so existing coding-agent consumers keep
// their established report shape.
type ArtifactFinding struct {
	Resource      string            `json:"resource"`
	Target        int               `json:"target"`
	Kind          string            `json:"kind"`
	Reference     string            `json:"reference"`
	Predicate     map[string]string `json:"predicate,omitempty"`
	CheckedAt     string            `json:"checked_at"`
	FirstFailedAt string            `json:"first_failed_at,omitempty"`
	Status        int               `json:"status,omitempty"`
	Reachable     bool              `json:"reachable"`
	Stale         bool              `json:"stale"`
	Note          string            `json:"note,omitempty"`
}

// RunnerFunc runs a resource check command and returns its raw JSON stdout.
// Tests inject a stub; the live wiring uses DefaultAggregateRunner.
type RunnerFunc func(ctx context.Context, args []string) ([]byte, error)

// DefaultAggregateRunner shells out to the resource CLI and captures stdout.
func DefaultAggregateRunner(ctx context.Context, args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("empty check command")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	return cmd.Output()
}

// RunAggregate runs each entry's check command via run and merges the JSON
// Reports. It is agent-safe: a missing binary, non-zero exit, or unparsable
// output degrades that entry to a StatusUnknown Report (with a note) rather
// than failing the whole aggregate.
func RunAggregate(ctx context.Context, run RunnerFunc, entries []AggregateEntry) AggregateReport {
	if run == nil {
		run = DefaultAggregateRunner
	}
	agg := AggregateReport{Resources: make([]Report, 0, len(entries))}
	for _, entry := range entries {
		rep := runOneAggregateEntry(ctx, run, entry)
		agg.Resources = append(agg.Resources, rep)
		switch rep.Status {
		case StatusBehind:
			agg.Behind = append(agg.Behind, rep.Name)
		case StatusUnknown:
			agg.Unknown = append(agg.Unknown, rep.Name)
		}
	}
	sort.Strings(agg.Behind)
	sort.Strings(agg.Unknown)
	return agg
}

func runOneAggregateEntry(ctx context.Context, run RunnerFunc, entry AggregateEntry) Report {
	unknown := func(note string) Report {
		return Report{Name: entry.Name, Status: StatusUnknown, Note: note}
	}
	out, err := run(ctx, entry.CheckCmd)
	if err != nil {
		return unknown(fmt.Sprintf("check command failed: %v", err))
	}
	var rep Report
	if err := json.Unmarshal(out, &rep); err != nil {
		return unknown(fmt.Sprintf("could not parse check output: %v", err))
	}
	if strings.TrimSpace(rep.Name) == "" {
		rep.Name = entry.Name
	}
	if strings.TrimSpace(string(rep.Status)) == "" {
		rep.Status = StatusUnknown
	}
	return rep
}
