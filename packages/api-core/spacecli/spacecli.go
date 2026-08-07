// Package spacecli provides the shared `space` CLI verb that every coverage-space
// projection owner (search-hub / test-genie / prompt-manager) registers. It is
// the read side of the cross-scenario contract: `<owner> space --projection <p>
// --json` emits that owner's space doc as space-definition/v1 JSON (the
// denominator only). meta-optimization-manager joins the numerator live; this
// verb never computes coverage.
//
// Each owner registers exactly one projection (the one it owns); the
// --projection flag exists so the call site is uniform across owners and is
// validated against the registered projection.
package spacecli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"

	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

// Config configures the `space` command for one projection owner.
type Config struct {
	// Owner is the scenario slug that owns the denominator (e.g. "search-hub").
	Owner string
	// Projection is the projection this owner owns.
	Projection spacedoc.Projection
	// DocPath, when set, is read directly instead of resolving the canonical
	// scenario doc path. Primarily for tests.
	DocPath string
	// Out/Err default to os.Stdout/os.Stderr when nil.
	Out io.Writer
	Err io.Writer
}

// CommandGroup builds the flat "space" command group for the given owner. Wire
// it into the owner CLI's domains aggregator CommandGroups(core) return.
func CommandGroup(cfg Config) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Coverage Space",
		Commands: []cliapp.Command{{
			Name:        "space",
			Description: "Emit this scenario's coverage-space denominator as JSON (the cross-scenario read contract)",
			LongDescription: "Reads docs/spaces/<projection>-space.md and emits it as space-definition/v1 JSON " +
				"(the curated denominator only — never live coverage). Consumed by meta-optimization-manager.",
			Run: func(args []string) error { return run(cfg, args) },
		}},
	}
}

func run(cfg Config, args []string) error {
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	fs := flag.NewFlagSet("space", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	projection := fs.String("projection", string(cfg.Projection), "projection (answer|validate|guide|act); must match this owner")
	asJSON := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	requested := spacedoc.Projection(strings.TrimSpace(*projection))
	if requested != cfg.Projection {
		return fmt.Errorf("space: this scenario owns the %q projection, not %q", cfg.Projection, requested)
	}

	docPath, err := resolveDocPath(cfg)
	if err != nil {
		return err
	}
	md, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("space: read %s: %w", docPath, err)
	}
	def, err := spacedoc.Parse(cfg.Projection, md)
	if err != nil {
		return fmt.Errorf("space: parse %s: %w", docPath, err)
	}
	def.Source = relSource(docPath)
	if cfg.Owner == "test-genie" && cfg.Projection == spacedoc.ProjectionValidate {
		if err := enrichValidateAxes(def); err != nil {
			return err
		}
	}

	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(def)
	}
	return renderText(out, def)
}

func enrichValidateAxes(def *spacedoc.SpaceDefinition) error {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return fmt.Errorf("space: locate repo root for target denominator: %w", err)
	}
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return fmt.Errorf("space: load repo contract for target denominator: %w", err)
	}
	targets, err := contract.EnumerateTargets(root)
	if err != nil {
		return fmt.Errorf("space: enumerate target denominator: %w", err)
	}
	counts := map[string]int{}
	for _, target := range targets {
		counts[string(target.Kind)]++
	}
	kinds := make([]string, 0, len(counts))
	for kind := range counts {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	axis := spacedoc.Axes{Concerns: make([]string, 0, len(def.Cells)), TargetKinds: make([]spacedoc.TargetKindAxis, 0, len(kinds))}
	seen := map[string]bool{}
	for _, cell := range def.Cells {
		if !seen[cell.ID] {
			axis.Concerns = append(axis.Concerns, cell.ID)
			seen[cell.ID] = true
		}
	}
	for _, kind := range kinds {
		axis.TargetKinds = append(axis.TargetKinds, spacedoc.TargetKindAxis{Kind: kind, Count: counts[kind]})
	}
	def.Axes = &axis
	def.Rebase = &spacedoc.Rebase{From: "scenario-only", To: "repository-target-kinds", Reason: "Validate coverage now reports concern coverage across the enumerated repository target kinds."}
	return nil
}

func resolveDocPath(cfg Config) (string, error) {
	if strings.TrimSpace(cfg.DocPath) != "" {
		return cfg.DocPath, nil
	}
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("space: locate repo root: %w", err)
	}
	scenarioRoot, err := repocontract.ResolveScenarioPath(root, cfg.Owner)
	if err != nil {
		return "", fmt.Errorf("space: resolve scenario %s: %w", cfg.Owner, err)
	}
	return filepath.Join(scenarioRoot, "docs", "spaces", string(cfg.Projection)+"-space.md"), nil
}

// relSource trims the doc path to a repo-relative locator for provenance.
func relSource(docPath string) string {
	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		if rel, rerr := filepath.Rel(root, docPath); rerr == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return docPath
}

func renderText(out io.Writer, def *spacedoc.SpaceDefinition) error {
	var now, inReach, missing int
	for _, c := range def.Cells {
		switch c.Status {
		case spacedoc.StatusNow:
			now++
		case spacedoc.StatusInReach:
			inReach++
		case spacedoc.StatusMissing:
			missing++
		}
	}
	fmt.Fprintf(out, "Projection:  %s\n", def.Projection)
	fmt.Fprintf(out, "Owner:       %s\n", def.Owner)
	fmt.Fprintf(out, "Confidence:  %s\n", def.DenominatorConfidence)
	fmt.Fprintf(out, "Cells:       %d  (now=%d in_reach=%d missing=%d)\n", len(def.Cells), now, inReach, missing)
	fmt.Fprintf(out, "Source:      %s\n", def.Source)
	return nil
}
