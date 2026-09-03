package gates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
)

type persistedStoryReport struct {
	Results []persistedStoryResult `json:"results"`
}

type persistedStoryResult struct {
	Stage          string            `json:"stage"`
	AssetLibraryID string            `json:"assetLibraryId"`
	Subject        string            `json:"subject"`
	Evidence       []json.RawMessage `json:"evidence"`
}

type persistedEvidence struct {
	Kind    string `json:"kind"`
	Console struct {
		ConsoleErrors  []string `json:"consoleErrors"`
		PageErrors     []string `json:"pageErrors"`
		FailedRequests []string `json:"failedRequests"`
	} `json:"console"`
	Performance struct {
		MountMS float64 `json:"mountMs"`
	} `json:"performance"`
}

func loadStoryEvidence(scope Scope, kinds ...string) (Result, error) {
	root := scope.Root
	allowedKinds := map[string]bool{}
	for _, kind := range kinds {
		allowedKinds[kind] = true
	}
	// Use the same routed evidence database as the freshness gate. The
	// scenario's source tree may not contain a local database in development,
	// while the control plane keeps the live store under ~/.vrooli/data.
	db := scope.DB
	if db == nil {
		return Result{}, nil
	}
	rows, err := db.QueryContext(context.Background(), `SELECT results_json FROM component_test_reports ORDER BY created_at DESC`)
	if err != nil {
		return Result{}, err
	}
	defer rows.Close()
	assetIDs, err := libraryAssetIDs(root)
	if err != nil {
		return Result{}, err
	}
	assets, err := loadAssets(scope)
	if err != nil {
		return Result{}, err
	}
	byID := map[string]assetDoc{}
	for _, asset := range assets {
		byID[asset.Asset.ID] = asset
	}
	storyPaths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", "*", "*", "versions", "*", "story.json"))
	if err != nil {
		return Result{}, err
	}
	storyPathByAsset := map[string]string{}
	result := Result{}
	for _, storyPath := range storyPaths {
		storyPathByAsset[implementationName(storyPath)] = storyPath
		if len(scope.Assets) == 0 || scopeReportsAsset(scope, implementationName(storyPath)) {
			result.Inspected++
		}
	}
	seen := map[string]bool{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return Result{}, err
		}
		report, err := decodePersistedStoryReport([]byte(raw))
		if err != nil {
			return Result{}, err
		}
		for _, story := range report.Results {
			if story.Stage != "declared_behavior" || len(story.Evidence) == 0 {
				continue
			}
			assetID := assetIDs[story.AssetLibraryID]
			if assetID == "" {
				continue
			}
			if len(scope.Assets) > 0 && !scopeReportsAsset(scope, assetID) {
				continue
			}
			key := assetID + "\x00" + story.Subject
			if seen[key] {
				continue
			}
			seen[key] = true
			result.InspectedAssets = appendUnique(result.InspectedAssets, assetID)
			for _, rawEvidence := range story.Evidence {
				var evidence persistedEvidence
				if json.Unmarshal(rawEvidence, &evidence) != nil {
					continue
				}
				if !allowedKinds[evidence.Kind] {
					continue
				}
				switch evidence.Kind {
				case "console":
					if len(evidence.Console.ConsoleErrors)+len(evidence.Console.PageErrors)+len(evidence.Console.FailedRequests) > 0 {
						result.Findings = append(result.Findings, Finding{Code: "catalog.console_clean", AssetID: assetID, Message: fmt.Sprintf("story %q emitted console/page/request errors", story.Subject), Remediation: "Fix the React warning, uncaught page error, or failed request before treating the story as clean."})
					}
				case "performance":
					budget := defaultMountBudget(byID[assetID].Asset.Kind)
					if byID[assetID].Budgets.MountMS > 0 {
						budget = byID[assetID].Budgets.MountMS
					}
					if evidence.Performance.MountMS > budget {
						result.Findings = append(result.Findings, Finding{Code: "catalog.performance_budget", AssetID: assetID, File: repoRel(root, storyPathByAsset[assetID]), Message: fmt.Sprintf("story %q mountMs %.2f exceeds budget %.2f", story.Subject, evidence.Performance.MountMS, budget), Remediation: "Reduce mount work or raise the explicit budget with evidence."})
					}
				}
			}
		}
	}
	return result, rows.Err()
}

func decodePersistedStoryReport(raw []byte) (persistedStoryReport, error) {
	var report persistedStoryReport
	if err := json.Unmarshal(raw, &report); err == nil {
		return report, nil
	}
	var results []persistedStoryResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return persistedStoryReport{}, err
	}
	return persistedStoryReport{Results: results}, nil
}

func unmeasuredStoryGate(scope Scope) Result {
	result, _ := UnmeasuredGate(scope.Root)
	if len(scope.Assets) > 0 {
		assets, err := loadLibraryAssets(scope)
		if err == nil {
			result.Inspected = 0
			for _, asset := range assets {
				if scopeReportsAsset(scope, asset.Asset.ID) {
					result.Inspected++
				}
			}
		}
	}
	return result
}

func defaultMountBudget(kind string) float64 {
	switch kind {
	case "primitive":
		return 8
	case "page-template":
		return 50
	default:
		return 16
	}
}

func libraryAssetIDs(root string) (map[string]string, error) {
	result := map[string]string{}
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := librarywalk.Glob(filepath.Join(root, "scenarios", "react-component-library", "library", kind, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			var manifest struct {
				LibraryID string `json:"libraryId"`
				CatalogID string `json:"catalogId"`
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, err
			}
			if manifest.LibraryID != "" && manifest.CatalogID != "" {
				result[manifest.LibraryID] = manifest.CatalogID
			}
		}
	}
	return result, nil
}
