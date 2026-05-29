// Package mislocatedfile is the mislocated-file detector. For each
// file in the snapshot, it asks the aggregator (via
// DetectInput.VerdictProvider) for the verdict, and emits a conflict
// when the verdict's auto_place domain differs from the domain the
// derived map assigns to the file's path.
//
// The verdict's evidence travels into the conflict's evidence so the
// operator + analytics see the exact basis the aggregator used.
package mislocatedfile

import (
	"context"
	"fmt"

	"architecture-cartographer/internal/conflicts"
)

// Detector is the production mislocated-file detector.
type Detector struct{}

// New returns the production detector.
func New() *Detector { return &Detector{} }

func (Detector) Name() string { return "mislocated_file" }
func (Detector) Description() string {
	return "Flags files whose verdict-recommended domain differs from the domain the derived map assigns to them."
}

func (Detector) EmitsTypes() []string {
	return []string{"mislocated_file"}
}

func (d Detector) Detect(ctx context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	if in.VerdictProvider == nil {
		return nil, nil
	}
	chunks := in.Snapshot.Chunks()
	if len(chunks) == 0 {
		return nil, nil
	}
	// One batched verdict call: snapshot + domain map + GraphContext
	// are built once and the aggregator runs concurrently across chunks.
	// This replaces the previous per-chunk loop that made detect
	// O(F²×D×S) on large scenarios.
	verdicts, err := in.VerdictProvider.VerdictsFor(ctx, in.Scenario, chunks)
	if err != nil {
		return nil, err
	}
	var out []conflicts.Conflict
	for i, chunk := range chunks {
		v := verdicts[i]
		current := in.DomainMap.DomainFor(chunk.Path)
		if v.Tier != "auto_place" || v.TopDomain == "" {
			continue
		}
		if v.TopDomain == current {
			continue
		}
		payload := []byte(fmt.Sprintf(`{"from_domain":%q,"to_domain":%q,"path":%q}`, current, v.TopDomain, chunk.Path))
		conflict := conflicts.Conflict{
			Scenario:  in.Scenario,
			Detector:  d.Name(),
			Type:      "mislocated_file",
			Subtype:   classifyMove(current, v.TopDomain),
			Severity:  conflicts.SeverityWarn,
			Locations: []string{chunk.Path},
			Domains:   filterEmpty(current, v.TopDomain),
			Evidence: []conflicts.Evidence{
				{
					Kind:    "verdict_top_domain",
					Summary: fmt.Sprintf("aggregator suggests %s (value=%.3f)", v.TopDomain, v.TopValue),
					Locator: chunk.Path,
				},
				{
					Kind:    "derived_domain_location",
					Summary: fmt.Sprintf("derived map places file in %q", current),
					Locator: chunk.Path,
				},
			},
			SuggestedFixes: []conflicts.Fix{{
				ID:         "fix:move-file",
				Kind:       conflicts.FixKindMoveFile,
				Resolver:   "mislocated_file",
				Summary:    fmt.Sprintf("move %s from %q to %q", chunk.Path, current, v.TopDomain),
				Payload:    payload,
				Confidence: v.TopValue,
			}},
		}
		out = append(out, conflict)
	}
	return out, nil
}

func classifyMove(from, to string) string {
	switch {
	case from == "":
		return "unassigned-to-domain"
	case to == "":
		return "to-unassigned"
	default:
		return fmt.Sprintf("%s-to-%s", from, to)
	}
}

func filterEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
