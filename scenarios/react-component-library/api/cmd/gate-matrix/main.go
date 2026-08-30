package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/gates"
)

type matrix struct {
	SchemaVersion string       `json:"schema_version"`
	GeneratedAt   string       `json:"generated_at"`
	AssetCount    int          `json:"asset_count"`
	GateCount     int          `json:"gate_count"`
	Cells         []matrixCell `json:"cells"`
}

type matrixCell struct {
	AssetID                string   `json:"asset_id"`
	Gate                   string   `json:"gate"`
	Verdict                string   `json:"verdict"`
	FindingCodes           []string `json:"finding_codes"`
	FindingCount           int      `json:"finding_count"`
	Inspected              int      `json:"inspected"`
	FindingsWithoutAssetID int      `json:"findings_without_asset_id"`
}

func main() {
	diffPath := flag.String("diff", "", "compare the fresh matrix against this baseline")
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	current, err := buildMatrix(filepath.Clean(*root))
	if err != nil {
		fatal(err)
	}
	if *diffPath != "" {
		if err := printDiff(*diffPath, current); err != nil {
			fatal(err)
		}
		return
	}
	output, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		fatal(err)
	}
	_, _ = os.Stdout.Write(append(output, '\n'))
}

func buildMatrix(root string) (matrix, error) {
	catalogRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	assets, err := catalogcoverage.LoadCatalog(catalogRoot)
	if err != nil {
		return matrix{}, err
	}
	implementations, err := catalogcoverage.LoadImplementations(filepath.Join(catalogRoot, "..", "library"))
	if err != nil {
		return matrix{}, err
	}
	implNames := make(map[string]string, len(implementations))
	for _, implementation := range implementations {
		if implementation.CatalogID != "" {
			implNames[implementation.CatalogID] = implementation.Name
		}
	}
	definitions, err := catalogcoverage.LoadGateDefinitions(filepath.Join(catalogRoot, "config.json"))
	if err != nil {
		return matrix{}, err
	}
	results := make(map[string]gates.Result, len(definitions))
	for _, definition := range definitions {
		runner := gates.GateRunnerFor(definition.ID)
		if runner == nil {
			continue
		}
		result, runErr := runner(gates.Scope{Root: root})
		if runErr != nil {
			return matrix{}, fmt.Errorf("run gate %s: %w", definition.ID, runErr)
		}
		results[definition.ID] = gates.NormalizeResult(root, result)
	}

	var cells []matrixCell
	for _, asset := range assets {
		if _, implemented := implNames[asset.ID]; !implemented {
			continue
		}
		for _, definition := range definitions {
			if definition.Attribution == "corpus" || !hasKind(definition.AppliesTo, asset.Kind) {
				continue
			}
			result, measured := results[definition.ID]
			cell := matrixCell{AssetID: asset.ID, Gate: definition.ID, Verdict: "unmeasured"}
			if measured {
				cell.Inspected = result.Inspected
				cell.FindingsWithoutAssetID = len(result.RunnerError)
				cell.FindingCodes = codesFor(result.Findings, asset.ID, implNames[asset.ID])
				cell.FindingCount = len(cell.FindingCodes)
				cell.Verdict = "pass"
				if len(cell.FindingCodes) > 0 || len(result.RunnerError) > 0 {
					cell.Verdict = "fail"
				}
				if result.Status == "unmeasured" || contains(result.UnmeasuredAssets, asset.ID) {
					cell.Verdict = "unmeasured"
				}
			}
			cells = append(cells, cell)
		}
	}
	for _, definition := range definitions {
		if definition.Attribution != "corpus" {
			continue
		}
		result, measured := results[definition.ID]
		cell := matrixCell{AssetID: "__corpus__", Gate: definition.ID, Verdict: "unmeasured"}
		if measured {
			cell.Inspected = result.Inspected
			cell.FindingsWithoutAssetID = len(result.RunnerError)
			cell.FindingCodes = codesFor(result.Findings, "__corpus__")
			cell.FindingCount = len(cell.FindingCodes)
			cell.Verdict = "pass"
			if cell.FindingCount > 0 || len(result.RunnerError) > 0 {
				cell.Verdict = "fail"
			}
		}
		cells = append(cells, cell)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].AssetID != cells[j].AssetID {
			return cells[i].AssetID < cells[j].AssetID
		}
		return cells[i].Gate < cells[j].Gate
	})
	return matrix{SchemaVersion: "gate-matrix/v1", GeneratedAt: time.Unix(0, 0).UTC().Format(time.RFC3339), AssetCount: len(assets), GateCount: len(definitions), Cells: cells}, nil
}

func hasKind(kinds []string, wanted string) bool {
	for _, kind := range kinds {
		if kind == wanted {
			return true
		}
	}
	return false
}

func codesFor(findings []gates.Finding, ids ...string) []string {
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}
	seen := map[string]bool{}
	for _, finding := range findings {
		if allowed[finding.AssetID] {
			seen[finding.Code] = true
		}
	}
	codes := make([]string, 0, len(seen))
	for code := range seen {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
