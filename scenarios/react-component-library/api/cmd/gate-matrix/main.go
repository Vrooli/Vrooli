package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
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
	FindingMessages        []string `json:"finding_messages,omitempty"`
	FindingCount           int      `json:"finding_count"`
	Inspected              int      `json:"inspected"`
	FindingsWithoutAssetID int      `json:"findings_without_asset_id"`
	RunnerMessages         []string `json:"runner_messages,omitempty"`
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
	key, keyErr := matrixInputFingerprint(root)
	if keyErr == nil {
		cachePath := filepath.Join(os.TempDir(), "rcl-gate-matrix-"+key+".json")
		if raw, readErr := os.ReadFile(cachePath); readErr == nil {
			var cached struct {
				Key    string `json:"key"`
				Matrix matrix `json:"matrix"`
			}
			if json.Unmarshal(raw, &cached) == nil && cached.Key == key {
				return cached.Matrix, nil
			}
		}
	}
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
	var resultsMu sync.Mutex
	jobs := make(chan catalogcoverage.GateDefinition)
	var firstErr error
	var errMu sync.Mutex
	const workers = 8
	var wait sync.WaitGroup
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for definition := range jobs {
				runner := gates.GateRunnerFor(definition.ID)
				if runner == nil {
					continue
				}
				result, runErr := runner(gates.Scope{Root: root})
				if runErr != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("run gate %s: %w", definition.ID, runErr)
					}
					errMu.Unlock()
					continue
				}
				resultsMu.Lock()
				results[definition.ID] = gates.NormalizeResult(root, result)
				resultsMu.Unlock()
			}
		}()
	}
	for _, definition := range definitions {
		jobs <- definition
	}
	close(jobs)
	wait.Wait()
	if firstErr != nil {
		return matrix{}, firstErr
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
				cell.FindingMessages = messagesFor(result.Findings, asset.ID, implNames[asset.ID])
				cell.FindingCount = len(cell.FindingCodes)
				cell.Verdict = "pass"
				// Runner errors are corpus-level execution evidence. They cannot
				// be attributed to an asset, so never smear them into an asset
				// failure. The asset is unmeasured until the runner can execute.
				if len(cell.FindingCodes) > 0 {
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
			for _, finding := range result.RunnerError {
				if finding.Message != "" && !contains(cell.RunnerMessages, finding.Message) {
					cell.RunnerMessages = append(cell.RunnerMessages, finding.Message)
				}
			}
			cell.FindingCodes = codesFor(result.Findings, "__corpus__")
			cell.FindingMessages = messagesFor(result.Findings, "__corpus__")
			cell.FindingCount = len(cell.FindingCodes)
			cell.Verdict = "pass"
			if cell.FindingCount > 0 {
				cell.Verdict = "fail"
			}
			if len(result.RunnerError) > 0 {
				cell.Verdict = "unmeasured"
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
	current := matrix{SchemaVersion: "gate-matrix/v1", GeneratedAt: time.Unix(0, 0).UTC().Format(time.RFC3339), AssetCount: len(assets), GateCount: len(definitions), Cells: cells}
	if keyErr == nil {
		cachePath := filepath.Join(os.TempDir(), "rcl-gate-matrix-"+key+".json")
		if raw, marshalErr := json.Marshal(struct {
			Key    string `json:"key"`
			Matrix matrix `json:"matrix"`
		}{key, current}); marshalErr == nil {
			_ = os.WriteFile(cachePath, raw, 0o600)
		}
	}
	return current, nil
}

func matrixInputFingerprint(root string) (string, error) {
	hash := sha256.New()
	for _, relative := range []string{
		"scenarios/react-component-library/catalog",
		"scenarios/react-component-library/library",
		"scenarios/react-component-library/ui/src",
		"scenarios/react-component-library/api/internal/gates",
		"scenarios/react-component-library/api/internal/catalogcoverage",
		"scenarios/react-component-library/api/cmd/gate-matrix",
	} {
		base := filepath.Join(root, relative)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == ".retired" {
					return fs.SkipDir
				}
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			_, _ = hash.Write([]byte(filepath.ToSlash(path)))
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write(data)
			_, _ = hash.Write([]byte{0})
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
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

func messagesFor(findings []gates.Finding, ids ...string) []string {
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		allowed[id] = true
	}
	seen := map[string]bool{}
	messages := []string{}
	for _, finding := range findings {
		if !allowed[finding.AssetID] || finding.Message == "" || seen[finding.Message] {
			continue
		}
		seen[finding.Message] = true
		messages = append(messages, finding.Message)
	}
	sort.Strings(messages)
	return messages
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
