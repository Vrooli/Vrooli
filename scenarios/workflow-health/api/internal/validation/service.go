package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"workflow-health/internal/workflows"
)

type Engine struct {
	Fixers           *FixRegistry
	ReadinessFetcher ReadinessProfileFetcher
}

func NewEngine() *Engine {
	return &Engine{Fixers: NewFixRegistry(), ReadinessFetcher: newExperienceManagerFetcher()}
}

func (e *Engine) ValidateScenario(ctx context.Context, scenario, path string) (Report, error) {
	_ = ctx
	target, err := resolveTarget(scenario, path)
	if err != nil {
		return Report{}, err
	}
	catalog, err := workflows.ScanScenario(target)
	if err != nil {
		return Report{}, err
	}
	report := Report{
		Scenario:   catalog.Scenario,
		TargetPath: target,
		Catalog:    catalog,
	}
	report.Findings = e.evaluate(ctx, target, catalog)
	sortFindings(report.Findings)
	return report, nil
}

func resolveTarget(scenario, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("target path %q: %w", path, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("target path %q is not a directory", path)
		}
		return filepath.Clean(path), nil
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", fmt.Errorf("scenario or path is required")
	}
	target := filepath.Join("..", "..", "..", "scenarios", scenario)
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return filepath.Clean(target), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "scenarios", scenario)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Clean(candidate), nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}
	return "", fmt.Errorf("scenario %q not found", scenario)
}

func (e *Engine) evaluate(ctx context.Context, target string, catalog *workflows.ScenarioWorkflowCatalog) []Finding {
	var findings []Finding
	if len(catalog.Assets) == 0 {
		findings = append(findings, finding(CodeSurfaceAbsent, "", "", "No BAS workflow assets were discovered.", false))
		return findings
	}

	if !catalog.Registry.Exists && len(catalog.Cases) > 0 {
		findings = append(findings, finding(CodeRegistryMissing, "bas/registry.json", "", "BAS case registry is missing.", true))
	}
	registryPaths := make(map[string]struct{}, len(catalog.Registry.Entries))
	for _, entry := range catalog.Registry.Entries {
		registryPaths[entry.File] = struct{}{}
	}
	if catalog.Registry.Exists {
		for _, c := range catalog.Cases {
			if c.ParseError != "" {
				continue
			}
			if _, ok := registryPaths[c.Path]; !ok {
				findings = append(findings, finding(CodeRegistryStale, c.Path, c.ID, "Registry omits a cataloged workflow case.", true))
			}
		}
	}
	for _, path := range catalog.RegistryOnlyPaths {
		findings = append(findings, finding(CodeRegistryStale, path, "", "Registry references a workflow case that no longer exists.", true))
	}

	var profile *readinessProfile
	var profileErr error
	if e.ReadinessFetcher != nil {
		profile, profileErr = e.ReadinessFetcher(ctx, catalog.Scenario)
	}

	assetPaths := make(map[string]struct{}, len(catalog.Assets))
	for _, asset := range catalog.Assets {
		assetPaths[asset.Path] = struct{}{}
	}

	for _, asset := range catalog.Assets {
		if asset.Type == workflows.AssetTypeSeed || asset.Type == workflows.AssetTypeRegistryOnly {
			continue
		}
		findings = append(findings, e.evaluateAsset(target, asset, assetPaths, profile)...)
	}
	// An observer label is a safety classification, not an assertion of intent.
	// Resolve referenced flows/actions as well so an observer case cannot hide a
	// write behind a subflow boundary.
	for _, asset := range catalog.Assets {
		if asset.Type == workflows.AssetTypeSeed || asset.Type == workflows.AssetTypeRegistryOnly || asset.ExecutionMode != "observer" || asset.ParseError != "" {
			continue
		}
		if nodes := observerUnsafeNodes(target, asset, catalog); len(nodes) > 0 {
			findings = append(findings, finding(CodeObserverContentUnsafe, asset.Path, asset.ID, fmt.Sprintf("Observer workflow contains non-read-only action node(s): %s.", strings.Join(nodes, ", ")), true))
		}
	}
	if e.ReadinessFetcher != nil {
		if profileErr != nil {
			if hasExperienceDirectory(target) {
				findings = append(findings, finding(CodeExperienceUnavailable, "experience", "", fmt.Sprintf("Could not resolve the Experience Manager readiness profile: %v", profileErr), false))
			}
		} else {
			findings = append(findings, experienceCoverageFindings(catalog, profile)...)
		}
	}

	return findings
}

var observerActionAllowlist = map[string]struct{}{
	"ACTION_TYPE_NAVIGATE":   {},
	"ACTION_TYPE_SCREENSHOT": {},
	"ACTION_TYPE_ASSERT":     {},
	"ACTION_TYPE_EXTRACT":    {},
	"ACTION_TYPE_WAIT":       {},
}

// observerUnsafeNodes returns node IDs whose BAS actions can alter state. It
// follows cataloged dependencies, which is necessary because a case may invoke
// a mutating action indirectly through one or more flows.
func observerUnsafeNodes(root string, asset workflows.WorkflowAsset, catalog *workflows.ScenarioWorkflowCatalog) []string {
	byPath := make(map[string]workflows.WorkflowAsset, len(catalog.Assets))
	for _, candidate := range catalog.Assets {
		byPath[candidate.Path] = candidate
	}
	seen := make(map[string]bool)
	var unsafe []string
	var visit func(workflows.WorkflowAsset)
	visit = func(current workflows.WorkflowAsset) {
		if seen[current.Path] {
			return
		}
		seen[current.Path] = true
		unsafe = append(unsafe, observerUnsafeNodesInFile(root, current)...)
		for _, dependency := range current.Dependencies {
			path := "bas/" + strings.TrimPrefix(filepath.ToSlash(dependency.ToPath), "bas/")
			if child, ok := byPath[path]; ok {
				visit(child)
			}
		}
	}
	visit(asset)
	sort.Strings(unsafe)
	return unsafe
}

func observerUnsafeNodesInFile(root string, asset workflows.WorkflowAsset) []string {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(asset.Path)))
	if err != nil {
		return nil
	}
	var authored map[string]any
	if json.Unmarshal(data, &authored) != nil {
		return nil
	}
	document, err := workflows.ResolveBASDocument(authored)
	if err != nil {
		return nil
	}
	definition, err := json.Marshal(document.Definition)
	if err != nil {
		return nil
	}
	var doc struct {
		Nodes []struct {
			ID     string `json:"id"`
			Action struct {
				Type string `json:"type"`
			} `json:"action"`
		} `json:"nodes"`
	}
	if json.Unmarshal(definition, &doc) != nil {
		return nil
	}
	unsafe := make([]string, 0)
	for _, node := range doc.Nodes {
		if _, ok := observerActionAllowlist[node.Action.Type]; !ok {
			unsafe = append(unsafe, fmt.Sprintf("%s (%s)", firstNonEmptyNodeID(node.ID), node.Action.Type))
		}
	}
	return unsafe
}

func firstNonEmptyNodeID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "<unnamed>"
	}
	return id
}

func hasExperienceDirectory(target string) bool {
	info, err := os.Stat(filepath.Join(target, "experience"))
	return err == nil && info.IsDir()
}

func (e *Engine) evaluateAsset(target string, asset workflows.WorkflowAsset, assetPaths map[string]struct{}, profile *readinessProfile) []Finding {
	var findings []Finding
	if asset.ParseError != "" {
		findings = append(findings, finding(CodeParseError, asset.Path, asset.ID, asset.ParseError, false))
		return findings
	}
	if err := validateBASSchema(target, asset); err != nil {
		findings = append(findings, finding(CodeSchemaInvalid, asset.Path, asset.ID, err.Error(), false))
		return findings
	}
	if len(asset.EnvelopeUnknownFields) > 0 {
		findings = append(findings, finding(CodeEnvelopeUnknownField, asset.Path, asset.ID, fmt.Sprintf("BAS persistence envelope contains fields Workflow Health does not interpret: %s. They are retained for BAS compatibility and do not affect validation of flow_definition.", strings.Join(asset.EnvelopeUnknownFields, ", ")), false))
	}
	if asset.Name == "" || asset.Description == "" {
		findings = append(findings, finding(CodeMetadataIncomplete, asset.Path, asset.ID, "Workflow metadata must include name and description.", true))
	}
	if asset.Role == workflows.AssetRoleValidationCase && len(asset.Requirements) == 0 {
		findings = append(findings, finding(CodeRequirementUnlinked, asset.Path, asset.ID, "Validation case is not linked to a requirement.", false))
	}
	for _, ref := range asset.Selectors {
		if ref.Key == "" && strings.TrimSpace(ref.Raw) != "" && !isAuthoredExperienceBinding(ref.Raw, profile) {
			findings = append(findings, finding(CodeSelectorUnregistered, asset.Path, asset.ID, fmt.Sprintf("Selector %q is not an @selector registry reference.", ref.Raw), false))
		}
	}
	for _, edge := range asset.Dependencies {
		toPath := "bas/" + strings.TrimPrefix(edge.ToPath, "bas/")
		if _, ok := assetPaths[toPath]; !ok {
			findings = append(findings, finding(CodeSubflowUnresolved, asset.Path, asset.ID, fmt.Sprintf("Dependency %q does not resolve to a cataloged workflow asset.", edge.ToPath), false))
		}
	}
	if !validExecutionMode(asset.ExecutionMode) {
		findings = append(findings, finding(CodeExecutionModeInvalid, asset.Path, asset.ID, "Workflow execution_mode must be observer, mutating, or destructive.", true))
	}
	if hasLegacyReset(target, asset.Path) {
		findings = append(findings, finding(CodeResetLegacy, asset.Path, asset.ID, "Legacy reset value database must be normalized to full.", true))
	}
	if asset.Safety.Mutating && !hasExplicitConfirmation(target, asset.Path) {
		findings = append(findings, finding(CodeMutatingSafety, asset.Path, asset.ID, "Mutating workflows must declare requires_confirmation=true.", false))
	}
	if asset.Safety.Mutating && !hasRoutedIsolation(target, asset.Path) {
		findings = append(findings, finding(CodeMutatingSafety, asset.Path, asset.ID, "Mutating workflows must declare routed_isolation=true before execution.", false))
	}
	if asset.Safety.Mutating && asset.Reset == "full" && !hasSeedDependency(asset) {
		findings = append(findings, finding(CodeSeedMissing, asset.Path, asset.ID, "Full-reset mutating workflows must declare a seed or fixture dependency.", false))
	}
	return findings
}

func validateBASSchema(target string, asset workflows.WorkflowAsset) error {
	data, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(asset.Path)))
	if err != nil {
		return fmt.Errorf("read authored workflow for BAS schema validation: %w", err)
	}
	if _, err := workflows.DecodeBASDefinitionJSON(data); err != nil {
		return fmt.Errorf("BAS WorkflowDefinitionV2 schema is invalid: %w", err)
	}
	return nil
}

// isAuthoredExperienceBinding permits the exact runtime binding declared by an
// adopted Experience Manager profile. Those bindings are the scenario's
// selector source of truth, so requiring a duplicate registry alias would make
// a workflow choose between two validators rather than add useful evidence.
func isAuthoredExperienceBinding(raw string, profile *readinessProfile) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || profile == nil {
		return false
	}
	for _, page := range profile.Pages {
		for _, region := range page.Regions {
			if binding := strings.TrimSpace(region.Binding.Selector); binding != "" {
				if raw == binding || (strings.Contains(raw, binding) && strings.Contains(raw, "data-experience-state")) {
					return true
				}
			}
			if testID := strings.TrimSpace(region.Binding.TestID); testID != "" && (raw == testID || raw == fmt.Sprintf("[data-testid=%q]", testID)) {
				return true
			}
		}
	}
	return false
}

func validExecutionMode(value string) bool {
	return value == "observer" || value == "mutating" || value == "destructive"
}

func hasSeedDependency(asset workflows.WorkflowAsset) bool {
	for _, edge := range asset.Dependencies {
		if edge.Kind == "fixture" {
			return true
		}
	}
	return false
}

func finding(code, path, assetID, description string, autofix bool) Finding {
	return Finding{
		Code:             code,
		Severity:         codeSeverity[code],
		Title:            titleForCode(code),
		Description:      description,
		FilePath:         path,
		AssetID:          assetID,
		AutofixAvailable: autofix,
		Remediation:      remediationForCode(code),
	}
}

func titleForCode(code string) string {
	switch code {
	case CodeSurfaceAbsent:
		return "Workflow surface absent"
	case CodeRegistryMissing:
		return "Workflow registry missing"
	case CodeRegistryStale:
		return "Workflow registry stale"
	case CodeParseError:
		return "Workflow JSON parse error"
	case CodeSchemaInvalid:
		return "Workflow does not match the BAS schema"
	case CodeEnvelopeUnknownField:
		return "BAS envelope field not interpreted"
	case CodeMetadataIncomplete:
		return "Workflow metadata incomplete"
	case CodeRequirementUnlinked:
		return "Validation case missing requirement link"
	case CodeSelectorUnregistered:
		return "Selector bypasses registry"
	case CodeSubflowUnresolved:
		return "Workflow dependency unresolved"
	case CodeExecutionModeInvalid:
		return "Execution mode invalid"
	case CodeObserverContentUnsafe:
		return "Observer workflow contains a mutating action"
	case CodeResetLegacy:
		return "Legacy reset metadata"
	case CodeMutatingSafety:
		return "Mutating workflow safety missing"
	case CodeSeedMissing:
		return "Mutating workflow seed missing"
	case CodeExecutionRefused:
		return "Workflow execution refused"
	case CodeExecutionFailed:
		return "Workflow execution failed"
	case CodeExperienceUnavailable:
		return "Experience profile unavailable"
	case CodeExperienceRouteMissing:
		return "Workflow route missing from experience contract"
	case CodeExperienceBindingMissing:
		return "Required experience region is not covered"
	case CodeExperienceStateMissing:
		return "Experience lifecycle is not covered"
	default:
		return code
	}
}

func remediationForCode(code string) string {
	switch code {
	case CodeRegistryMissing, CodeRegistryStale:
		return "Regenerate bas/registry.json from cataloged validation cases."
	case CodeSchemaInvalid:
		return "For a persisted BAS workflow, keep catalog metadata in the outer envelope and make flow_definition a valid WorkflowDefinitionV2. For a bare workflow, make the document itself a valid WorkflowDefinitionV2."
	case CodeEnvelopeUnknownField:
		return "No change is required unless Workflow Health needs to enforce one of these envelope fields; the nested flow_definition remains the executable BAS workflow."
	case CodeMetadataIncomplete:
		return "Add workflow metadata name and description."
	case CodeExecutionModeInvalid:
		return "Set metadata.execution_mode to observer, mutating, or destructive."
	case CodeObserverContentUnsafe:
		return "Relabel this workflow mutating, add confirmation and routed-isolation labels, or remove the mutating action."
	case CodeResetLegacy:
		return "Replace metadata.labels.reset=database with full."
	case CodeRequirementUnlinked:
		return "Link the case from requirements validation refs or workflow metadata."
	case CodeMutatingSafety:
		return "Declare confirmation and routed isolation metadata before execution."
	case CodeSeedMissing:
		return "Declare deterministic seed or fixture setup for full-reset mutation."
	case CodeExecutionRefused:
		return "Confirm mutating execution only after routed isolation proof is present."
	case CodeExecutionFailed:
		return "Inspect BAS validation, execution output, and workflow artifacts."
	case CodeExperienceUnavailable:
		return "Start Experience Manager or repair the adopted experience contract before enforcing workflow coverage."
	case CodeExperienceRouteMissing:
		return "Declare the tested route in the scenario experience page contract, or remove the stale workflow target."
	case CodeExperienceBindingMissing:
		return "Assert the required region using its authored testid or selector binding; do not create a separate selector registry."
	case CodeExperienceStateMissing:
		return "Add workflow assertions for data-experience-state=loading and a declared terminal state on the required async region."
	default:
		return "Inspect and repair the workflow asset."
	}
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].FilePath != findings[j].FilePath {
			return findings[i].FilePath < findings[j].FilePath
		}
		return findings[i].Code < findings[j].Code
	})
}

// SortFindings applies the package's stable finding ordering.
func SortFindings(findings []Finding) {
	sortFindings(findings)
}
