package catalogcoverage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"

	"react-component-library/internal/gates"
)

func RecomputeEvidence(root string) ([]GateEvidence, error) {
	return recomputeEvidence(root, nil)
}

func recomputeEvidence(root string, runtimeDB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
},
) ([]GateEvidence, error) {
	revisions, err := BuildRevisionIndex(root)
	if err != nil {
		return nil, err
	}
	return recomputeEvidenceWithSkip(root, runtimeDB, nil, revisions)
}

// recomputeEvidenceWithSkip runs the gate corpus. When stale is nil every
// runner executes, which is the cold path and the path RecomputeEvidence takes.
// When stale is present a runner is skipped only if no applicable asset needs
// it; the runners themselves are whole-corpus by construction, so the saving is
// the runner's entire cost or nothing.
func recomputeEvidenceWithSkip(root string, runtimeDB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, stale map[string]map[string]bool, revisions map[string]string,
) ([]GateEvidence, error) {
	// needed reports whether a runner must execute this pass. A gate the
	// freshness pass knows nothing about — every corpus gate, and any gate a
	// stale map does not mention — always runs.
	needed := func(gate string) bool {
		if stale == nil {
			return true
		}
		assets, known := stale[gate]
		if !known {
			return true
		}
		return len(assets) > 0
	}
	catalogRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	assets, err := LoadCatalog(catalogRoot)
	if err != nil {
		return nil, err
	}
	impls, err := LoadImplementations(filepath.Join(root, "scenarios", "react-component-library", "library"))
	if err != nil {
		return nil, err
	}
	implByAsset := map[string]Implementation{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			implByAsset[impl.CatalogID] = impl
		}
	}
	definitions, err := LoadGateDefinitions(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	runners := map[string]gates.Result{}
	bindingsByAsset := make(map[string]map[string]RuleBinding, len(assets))
	for _, asset := range assets {
		if _, ok := implByAsset[asset.ID]; !ok {
			continue
		}
		bindings, resolveErr := ResolveRuleSetFor(catalogRoot, asset, definitions)
		if resolveErr != nil {
			return nil, resolveErr
		}
		bindingsByAsset[asset.ID] = make(map[string]RuleBinding, len(bindings))
		for _, binding := range bindings {
			bindingsByAsset[asset.ID][binding.GateID] = binding
		}
	}
	for _, definition := range definitions {
		if !needed(definition.ID) && definition.Attribution != "corpus" {
			continue
		}
		var selected []string
		if definition.Attribution != "corpus" && stale != nil {
			for assetID := range stale[definition.ID] {
				selected = append(selected, assetID)
			}
			sort.Strings(selected)
		}
		// Name the gate on the way out. A runner that fails on a filesystem
		// read used to surface as a bare "open <path>: no such file" with no
		// indication of which of the sixty-one gates produced it.
		result, available, runErr := gates.Run(definition.ID, gates.Scope{Root: root, Assets: selected, DB: runtimeDB})
		if runErr != nil {
			return nil, fmt.Errorf("gate %s: %w", definition.ID, runErr)
		}
		if available {
			result = gates.NormalizeResult(root, result)
			if err := AnnotateFindings(root, definition.ID, &result); err != nil {
				return nil, fmt.Errorf("gate %s: annotate findings: %w", definition.ID, err)
			}
			runners[definition.ID] = result
		}
	}
	for name, runner := range runners {
		runners[name] = gates.NormalizeResult(root, runner)
	}
	quarantined := map[string]bool{}
	for _, definition := range definitions {
		if !definition.Blocking {
			continue
		}
		runner := gates.GateRunnerFor(definition.ID)
		calibration, calibrationErr := gates.Calibrate(root, definition.ID, runner)
		if calibrationErr != nil {
			return nil, calibrationErr
		}
		if calibration.NonDiscriminating {
			quarantined[definition.ID] = true
		}
	}
	var out []GateEvidence
	// Gates without a declared runner are explicitly unmeasured. They are
	// still represented for every applicable built asset (and for corpus gates)
	// so the score cannot improve when a runner disappears.
	for _, asset := range assets {
		if _, ok := implByAsset[asset.ID]; !ok {
			continue
		}
		target := "react-vite"
		if len(asset.Targets) > 0 {
			target = asset.Targets[0]
		}
		revision, known := revisions[asset.ID]
		if !known {
			return nil, fmt.Errorf("catalog asset %q has no computable revision", asset.ID)
		}
		for _, definition := range definitions {
			binding, applicable := bindingsByAsset[asset.ID][definition.ID]
			if !applicable || binding.Source == RuleSourceCorpus {
				continue
			}
			gateName := definition.ID
			// An asset the freshness pass found current keeps the verdict
			// already in the evidence store. Emitting nothing here is what
			// makes that verdict survive; emitting "unmeasured" would quietly
			// lower the asset's rung on every warm pass.
			if stale != nil {
				if assets, known := stale[gateName]; known && !assets[asset.ID] {
					continue
				}
			}
			runner, ok := runners[gateName]
			if !ok || quarantined[gateName] {
				out = append(out, GateEvidence{AssetID: asset.ID, Target: target, Version: implByAsset[asset.ID].Latest, Gate: gateName, Result: "unmeasured", SourceRevision: revision})
				continue
			}
			result := "pass"
			if contains(runner.UnmeasuredAssets, asset.ID) || (runner.Status == "unmeasured" && len(runner.UnmeasuredAssets) == 0) {
				result = "unmeasured"
			} else if hasFinding(runner.Findings, asset.ID, implByAsset[asset.ID].Name, gateName) || len(runner.RunnerError) > 0 {
				result = "fail"
			} else if runner.Inspected == 0 {
				result = "skipped"
			}
			out = append(out, GateEvidence{AssetID: asset.ID, Target: target, Version: implByAsset[asset.ID].Latest, Gate: gateName, Result: result, SourceRevision: revision})
		}
	}
	for _, definition := range definitions {
		if definition.Attribution != "corpus" || (len(definition.Runner) > 0 && !quarantined[definition.ID]) {
			continue
		}
		out = append(out, GateEvidence{AssetID: "__corpus__", Target: "corpus", Version: "", Gate: definition.ID, Result: "unmeasured", SourceRevision: "corpus"})
	}
	for i := range out {
		if out[i].AssetID == "__corpus__" {
			continue
		}
		digest, err := RuleSetDigest(root, out[i].AssetID)
		if err != nil {
			return nil, err
		}
		out[i].RuleSetDigest = digest
	}
	return out, nil
}

func hasFinding(findings []gates.Finding, assetID, implementation, gate string) bool {
	for _, finding := range findings {
		if finding.AssetID == assetID || finding.AssetID == implementation {
			return true
		}
	}
	return false
}
