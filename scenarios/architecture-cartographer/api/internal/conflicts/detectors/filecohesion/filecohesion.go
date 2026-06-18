// Package filecohesion detects files that likely carry too many
// responsibilities for a screaming-architecture codebase.
package filecohesion

import (
	"context"
	"fmt"
	"sort"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
)

// Config controls the detector's conservative size signals. A zero threshold
// disables that signal so operators can roll out one dimension at a time.
type Config struct {
	MaxLines   int
	MaxSymbols int
}

// DefaultConfig returns the production defaults.
func DefaultConfig() Config {
	return Config{MaxLines: 400, MaxSymbols: 25}
}

// Detector flags files whose line count or symbol count suggests they should
// be split by responsibility. It intentionally ignores test files and only
// uses metadata already present in the graph snapshot.
type Detector struct {
	cfg Config
}

// New returns the production detector.
func New() *Detector { return NewWithConfig(DefaultConfig()) }

// NewWithConfig returns a detector with explicit thresholds.
func NewWithConfig(cfg Config) *Detector {
	if cfg.MaxLines < 0 {
		cfg.MaxLines = 0
	}
	if cfg.MaxSymbols < 0 {
		cfg.MaxSymbols = 0
	}
	return &Detector{cfg: cfg}
}

func (Detector) Name() string { return "file_cohesion" }

func (Detector) Description() string {
	return "Flags large files whose line or symbol counts suggest mixed responsibilities."
}

func (Detector) EmitsTypes() []string { return []string{"file_cohesion"} }

func (d Detector) Detect(_ context.Context, in conflicts.DetectInput) ([]conflicts.Conflict, error) {
	symbolsByFile := symbolCounts(in.Snapshot.Symbols)
	var out []conflicts.Conflict
	for _, file := range in.Snapshot.Files {
		if file.IsTest {
			continue
		}
		symbols := symbolsByFile[file.ID]
		lineExceeded := d.cfg.MaxLines > 0 && file.Lines > d.cfg.MaxLines
		symbolExceeded := d.cfg.MaxSymbols > 0 && symbols > d.cfg.MaxSymbols
		if !lineExceeded && !symbolExceeded {
			continue
		}
		out = append(out, d.conflictFor(in.Scenario, in.DomainMap.DomainFor(file.Path), file, symbols, lineExceeded, symbolExceeded))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Locations[0] < out[j].Locations[0]
	})
	return out, nil
}

func (d Detector) conflictFor(scenario, domain string, file graph.FileNode, symbols int, lineExceeded, symbolExceeded bool) conflicts.Conflict {
	evidence := make([]conflicts.Evidence, 0, 2)
	if lineExceeded {
		evidence = append(evidence, conflicts.Evidence{
			Kind:    "line_count",
			Summary: fmt.Sprintf("%s has %d lines (max %d)", file.Path, file.Lines, d.cfg.MaxLines),
			Locator: file.Path,
		})
	}
	if symbolExceeded {
		evidence = append(evidence, conflicts.Evidence{
			Kind:    "symbol_count",
			Summary: fmt.Sprintf("%s declares %d symbols (max %d)", file.Path, symbols, d.cfg.MaxSymbols),
			Locator: file.Path,
		})
	}
	return conflicts.Conflict{
		Scenario:  scenario,
		Detector:  d.Name(),
		Type:      "file_cohesion",
		Subtype:   "oversized_file",
		Severity:  conflicts.SeverityWarn,
		Locations: []string{file.Path},
		Domains:   domainList(domain),
		Evidence:  evidence,
		SuggestedFixes: []conflicts.Fix{{
			Summary:    fmt.Sprintf("Split %s into smaller files around one domain responsibility per file.", file.Path),
			Confidence: 0.55,
		}},
	}
}

func symbolCounts(symbols []graph.SymbolNode) map[string]int {
	out := make(map[string]int)
	for _, sym := range symbols {
		if sym.FileID == "" {
			continue
		}
		out[sym.FileID]++
	}
	return out
}

func domainList(domain string) []string {
	if domain == "" {
		return nil
	}
	return []string{domain}
}

var _ conflicts.Detector = (*Detector)(nil)
