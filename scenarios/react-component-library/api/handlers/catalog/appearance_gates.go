package catalog

import (
	"context"
	"fmt"
	"strings"

	componenttests "react-component-library/internal/componenttests"
	"react-component-library/internal/gates"
)

type assetCheckProfileContextKey struct{}
type assetCheckVersionContextKey struct{}

const (
	assetCheckProfilePublish = "publish"
	assetCheckProfileEdit    = "edit"
)

func appearanceGate(gate string) bool {
	return gate == "unit" || gate == "interaction" || gate == "visual"
}

func profileFor(ctx context.Context) string {
	if profile, ok := ctx.Value(assetCheckProfileContextKey{}).(string); ok && profile == assetCheckProfileEdit {
		return assetCheckProfileEdit
	}
	return assetCheckProfilePublish
}

func gateIsBlocking(ctx context.Context, gate string, configured bool) bool {
	// Edit validation is deliberately appearance-first: registry drift is
	// reported, but it cannot hide a broken rendered story while an agent is
	// iterating on one asset. Publish keeps the catalog's authored blocking
	// policy unchanged.
	return configured && (profileFor(ctx) == assetCheckProfilePublish || appearanceGate(gate))
}

func (h *handler) runAppearanceGate(ctx context.Context, gate, assetID string) (gates.Result, error) {
	result := gates.Result{Status: "measured"}
	if strings.TrimSpace(assetID) == "" {
		result.Status = "unmeasured"
		result.RunnerError = []gates.Finding{{
			Code:           "catalog." + gate + ".asset_required",
			AssetID:        "__corpus__",
			Message:        "appearance gate requires an asset id; use the one-asset check or pass --asset-id",
			Remediation:    "run the gate for a catalog asset so its version-pinned browser result can be attributed",
			RuleSource:     gates.RuleSourceUniversal,
			RuleDeclaredIn: "catalog/config.json",
		}}
		return result, nil
	}
	if h.testService == nil || h.assets == nil {
		result.Status = "unmeasured"
		result.RunnerError = []gates.Finding{{
			Code:           "catalog." + gate + ".runner_unavailable",
			AssetID:        assetID,
			Message:        "browser-backed component-test service is not configured",
			Remediation:    "start the component-test service and its BAS executor before trusting the appearance gate",
			RuleSource:     gates.RuleSourceUniversal,
			RuleDeclaredIn: "catalog/config.json",
		}}
		return result, nil
	}
	// RunGate --all fans out declared gates concurrently. The component-test
	// report store is SQLite-backed, so serialize the shared browser/report
	// operation and let the later gates reuse the immutable report.
	h.appearanceMu.Lock()
	defer h.appearanceMu.Unlock()

	component, err := resolveAsset(ctx, h.assets, assetID)
	if err != nil {
		return result, fmt.Errorf("resolve appearance-gate asset %q: %w", assetID, err)
	}
	version := strings.TrimSpace(component.LatestVersion)
	if version == "" {
		result.Status = "unmeasured"
		result.RunnerError = []gates.Finding{{
			Code:           "catalog." + gate + ".version_unavailable",
			AssetID:        assetID,
			Message:        "asset has no latest immutable version to render",
			Remediation:    "publish or index an immutable asset version before running the appearance gate",
			RuleSource:     gates.RuleSourceAsset,
			RuleDeclaredIn: "catalog/config.json",
		}}
		return result, nil
	}

	report, _, err := h.testService.RunWithReuse(ctx, componenttests.Request{ComponentID: component.ID, Version: version, IncludeClosure: false})
	if err != nil {
		result.Status = "unmeasured"
		result.RunnerError = []gates.Finding{{
			Code:           "catalog." + gate + ".runner_error",
			AssetID:        assetID,
			Message:        err.Error(),
			Remediation:    "repair the version-pinned component-test runner before trusting the appearance gate",
			RuleSource:     gates.RuleSourceAsset,
			RuleDeclaredIn: "catalog/config.json",
		}}
		return result, nil
	}

	result.Inspected = 1
	result.InspectedVersions = 1
	result.InspectedAssets = []string{component.CatalogID}
	for _, item := range report.Results {
		if gate != "visual" && item.Stage != componenttests.StageDeclared {
			continue
		}
		if gate == "visual" && item.Stage != componenttests.StageEvidence {
			continue
		}
		if item.Verdict == componenttests.VerdictPassed {
			continue
		}
		message := item.Message
		if strings.TrimSpace(message) == "" {
			message = fmt.Sprintf("%s runner returned %s", item.Subject, item.Verdict)
		}
		result.Findings = append(result.Findings, gates.Finding{
			Code:           "catalog." + gate + ".render",
			Category:       "rendered-behavior",
			AssetID:        component.CatalogID,
			CatalogID:      component.CatalogID,
			LibraryID:      component.LibraryID,
			Message:        fmt.Sprintf("%s@%s %s: %s", component.LibraryID, item.Version, item.Subject, message),
			Remediation:    item.Remediation,
			RuleSource:     gates.RuleSourceAsset,
			RuleDeclaredIn: "catalog/config.json",
			Owner:          "scenarios/react-component-library/componenttests",
			Severity:       gates.FindingSeverityBlocking,
		})
	}
	return result, nil
}
