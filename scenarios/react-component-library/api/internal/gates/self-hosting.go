package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateSelfHosting(scope Scope) (Result, error) {
	root := scope.Root
	uiRoot := filepath.Join(root, "scenarios", "react-component-library", "ui", "src")
	policyPath := filepath.Join(root, "scenarios", "react-component-library", "catalog", "self-hosting-policy.json")
	policyData, err := os.ReadFile(policyPath)
	if err != nil {
		return Result{RunnerError: []Finding{{
			Code: "catalog.self_hosting_policy_unavailable", AssetID: "__corpus__.self-hosting",
			Message:     "self-hosting policy is unavailable: " + err.Error(),
			Remediation: "Restore catalog/self-hosting-policy.json with the reviewed coverage floor and explicit exemptions.",
			DocsRef:     "docs/reference/composition-validation.md",
		}}}, nil
	}
	var policy struct {
		MinimumCovered int `json:"minimumCovered"`
		Exemptions     []struct {
			Pattern string `json:"pattern"`
			Reason  string `json:"reason"`
		} `json:"exemptions"`
	}
	if err := json.Unmarshal(policyData, &policy); err != nil {
		return Result{}, fmt.Errorf("decode self-hosting policy: %w", err)
	}
	assets, err := loadLibraryAssets(scope)
	if err != nil {
		return Result{}, err
	}
	consumed := map[string]bool{}
	known := map[string]string{}
	for _, asset := range assets {
		if asset.Asset.ID != "" {
			known[asset.Asset.Name] = asset.Asset.ID
		}
	}
	files := 0
	err = filepath.WalkDir(uiRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		files++
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range libraryImportRE.FindAllStringSubmatch(string(data), -1) {
			if assetID := known[match[1]]; assetID != "" {
				consumed[assetID] = true
			}
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{Inspected: 1}
	if files == 0 {
		result.RunnerError = append(result.RunnerError, Finding{
			Code: "catalog.self_hosting_no_sources", AssetID: "__corpus__.self-hosting",
			Message:     "catalog application source tree contains no inspectable files",
			Remediation: "Restore the catalog application source tree before measuring self-hosting.",
			DocsRef:     "docs/internal/TESTING.md",
		})
		return result, nil
	}
	result.InformationalFindings = append(result.InformationalFindings, Finding{
		Code: "catalog.self_hosting_measurement", AssetID: "__corpus__.self-hosting",
		Message:     fmt.Sprintf("catalog application imports %d of %d implemented catalog assets across %d source files; floor is %d; exemptions: %d", len(consumed), len(known), files, policy.MinimumCovered, len(policy.Exemptions)),
		Remediation: "Increase the reviewed coverage floor when a new asset is consumed. Add an exemption only for a documented surface outside the catalog application's remit.",
		DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
	})
	if len(consumed) < policy.MinimumCovered {
		result.Findings = append(result.Findings, Finding{
			Code: "catalog.self_hosting_uncovered", AssetID: "__corpus__.self-hosting",
			Message:     fmt.Sprintf("catalog application consumes %d implemented assets, below the required floor of %d", len(consumed), policy.MinimumCovered),
			Remediation: "Add real catalog application usage paths or lower the floor only through a reviewed policy change with a documented reason; keep this gate blocking.",
			DocsRef:     "docs/concepts/ARCHITECTURE.md#catalog-graph-projection",
		})
	}
	return nonEmpty(result, "self-hosting"), nil
}

// ValidateBASGenericity keeps browser workflows capability-driven. Component
// names and version query parameters belong in story/runner data, not in a
// workflow file that must be copied for every asset.
