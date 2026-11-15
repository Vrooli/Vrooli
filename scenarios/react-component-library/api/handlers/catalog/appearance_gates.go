package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"react-component-library/internal/catalogcoverage"
	componenttests "react-component-library/internal/componenttests"
	"react-component-library/internal/gates"
)

type (
	assetCheckProfileContextKey struct{}
	assetCheckVersionContextKey struct{}
)

const (
	assetCheckProfilePublish = "publish"
	assetCheckProfileEdit    = "edit"
)

func appearanceGate(gate string) bool {
	// These gates are all backed by immutable, version-pinned browser stories.
	// Accessibility assertions exercise roles, names, focus/keyboard operations,
	// and state; responsive stories assert layout at the story's declared phone
	// geometry. They must enter the browser runner rather than the registry's
	// declaration-only placeholders.
	return gate == "unit" || gate == "interaction" || gate == "accessibility" || gate == "responsive"
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

func (h *handler) runAppearanceGate(ctx context.Context, gate, assetID string, definition catalogcoverage.GateDefinition) (gates.Result, error) {
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
	applyAppearanceStoryEvidence(&result, gate, component.CatalogID, component.LibraryID, report, definition)
	return result, nil
}

func applyAppearanceStoryEvidence(result *gates.Result, gate, catalogID, libraryID string, report componenttests.Report, definition catalogcoverage.GateDefinition) {
	viewportWidths := map[int]bool{}
	axeStories := 0
	for _, item := range report.Results {
		if gate != "visual" && item.Stage != componenttests.StageDeclared {
			continue
		}
		if gate == "visual" && item.Stage != componenttests.StageEvidence {
			continue
		}
		if item.Verdict == componenttests.VerdictPassed {
			// Keep processing the positive result below: accessibility and
			// responsive verdicts require evidence beyond a passing story.
		} else {
			message := item.Message
			if strings.TrimSpace(message) == "" {
				message = fmt.Sprintf("%s runner returned %s", item.Subject, item.Verdict)
			}
			result.Findings = append(result.Findings, gates.Finding{
				Code:           "catalog." + gate + ".render",
				Category:       "rendered-behavior",
				AssetID:        catalogID,
				CatalogID:      catalogID,
				LibraryID:      libraryID,
				Message:        fmt.Sprintf("%s@%s %s: %s", libraryID, item.Version, item.Subject, message),
				Remediation:    item.Remediation,
				RuleSource:     gates.RuleSourceAsset,
				RuleDeclaredIn: "catalog/config.json",
				Owner:          "scenarios/react-component-library/componenttests",
				Severity:       gates.FindingSeverityBlocking,
			})
		}
		if gate == "responsive" && item.Verdict == componenttests.VerdictPassed && item.Geometry != nil {
			if width, ok := widthPixels(item.Geometry.Width); ok {
				viewportWidths[width] = true
			}
		}
		if gate == "accessibility" {
			for _, evidence := range item.Evidence {
				if evidence.Kind != "accessibility" {
					continue
				}
				violations, measured := parseAxeEvidence(evidence.AccessibilityJSON)
				if !measured {
					continue
				}
				axeStories++
				for _, violation := range violations {
					result.Findings = append(result.Findings, gates.Finding{
						Code: "catalog.accessibility.axe." + violation.ID, Category: "axe-accessibility",
						AssetID: catalogID, CatalogID: catalogID, LibraryID: libraryID,
						Message:     fmt.Sprintf("%s@%s %s: axe %s (%s): %s", libraryID, item.Version, item.Subject, violation.ID, violation.Impact, violation.Description),
						Remediation: "Fix the axe-core violation in the rendered story, then rerun the accessibility gate.",
						RuleSource:  gates.RuleSourceAsset, RuleDeclaredIn: "catalog/config.json", Owner: "scenarios/react-component-library/componenttests", Severity: gates.FindingSeverityBlocking,
					})
				}
			}
		}
	}
	if gate == "responsive" {
		for _, required := range definition.RunnerConfig.RequiredViewportWidthsPx {
			if !viewportWidths[required] {
				result.RunnerError = append(result.RunnerError, appearanceRunnerFinding(gate, catalogID, fmt.Sprintf("no passing story was rendered at required width %d px", required), "add a version-pinned story at each required width and assert its responsive layout"))
			}
		}
	}
	if gate == "accessibility" && definition.RunnerConfig.RequireAxeResults && axeStories == 0 {
		result.RunnerError = append(result.RunnerError, appearanceRunnerFinding(gate, catalogID, "no version-pinned story supplied axe-core results", "run axe-core against each rendered story and attach the result to its accessibility evidence"))
	}
	if len(result.RunnerError) > 0 {
		result.Status = "unmeasured"
	}
}

func appearanceRunnerFinding(gate, assetID, message, remediation string) gates.Finding {
	return gates.Finding{Code: "catalog." + gate + ".evidence_missing", AssetID: assetID, CatalogID: assetID, Scope: gates.FindingScopeAsset, Message: message, Remediation: remediation, RuleSource: gates.RuleSourceUniversal, RuleDeclaredIn: "catalog/config.json", Owner: "scenarios/react-component-library/componenttests", Severity: gates.FindingSeverityBlocking}
}

type axeEvidenceDocument struct {
	Schema     string         `json:"schema"`
	Snapshot   string         `json:"snapshot"`
	Status     string         `json:"status"`
	Engine     string         `json:"engine"`
	Version    string         `json:"version"`
	Violations []axeViolation `json:"violations"`
	Axe        struct {
		Engine     string         `json:"engine"`
		Version    string         `json:"version"`
		Violations []axeViolation `json:"violations"`
	} `json:"axe"`
}

type axeViolation struct {
	ID          string   `json:"id"`
	Impact      string   `json:"impact"`
	Description string   `json:"description"`
	Targets     []string `json:"targets,omitempty"`
}

func parseAxeEvidence(raw string) ([]axeViolation, bool) {
	var document axeEvidenceDocument
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &document) != nil {
		return nil, false
	}
	engine, version, violations := document.Engine, document.Version, document.Violations
	if document.Schema == "bas-accessibility-snapshot/v1" && strings.TrimSpace(document.Snapshot) != "" {
		engine, version, violations = document.Axe.Engine, document.Axe.Version, document.Axe.Violations
	} else if document.Schema != "rcl-axe-results/v1" || document.Status != "complete" {
		return nil, false
	}
	if engine != "axe-core" || strings.TrimSpace(version) == "" || violations == nil {
		return nil, false
	}
	for i := range violations {
		if strings.TrimSpace(violations[i].ID) == "" {
			violations[i].ID = "unnamed-violation"
		}
		if strings.TrimSpace(violations[i].Impact) == "" {
			violations[i].Impact = "unspecified impact"
		}
	}
	return violations, true
}

func widthPixels(width string) (int, bool) {
	width = strings.TrimSpace(width)
	if strings.HasSuffix(width, "px") {
		width = strings.TrimSpace(strings.TrimSuffix(width, "px"))
	}
	value, err := strconv.Atoi(width)
	return value, err == nil && value > 0
}
