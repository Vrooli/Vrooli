// Service is the in-process surface the flows domain exposes to the CLI
// and HTTP handlers. It owns the discover→compile→explain pipeline for
// flows on disk; verification is the verification domain's job.
package flows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flow-verifier/internal/flows/discovery"
	"flow-verifier/internal/flows/kind"
	"flow-verifier/internal/flows/kinds/navigation"
	navigationreconcile "flow-verifier/internal/flows/kinds/navigation/reconcile"
	navigationstudio "flow-verifier/internal/flows/kinds/navigation/studio"
	"flow-verifier/internal/flows/kinds/temporal"
	"flow-verifier/internal/flows/kinds/temporal/codegen"
	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/layout"
	"flow-verifier/internal/flows/kinds/temporal/model"
)

// Summary is the thin row returned by List — enough for the inventory UI
// and `flows list` text output without forcing the full Flow type.
//
// ScenarioID is left empty by List (which works in a single root); the
// scenarios HTTP handler stamps it in when aggregating across roots so
// the UI's global flow list can show which scenario each flow belongs
// to without an extra round-trip.
type Summary struct {
	FlowID       string `json:"flowId"`
	Kind         string `json:"kind"`
	ContractPath string `json:"contractPath"`
	Language     string `json:"language,omitempty"`
	SchemaVer    int    `json:"schemaVersion"`
	ScenarioID   string `json:"scenarioId,omitempty"`
}

// List discovers and summarises every flow rooted at root. The kind
// dispatch happens inside discovery; List projects each loaded Spec to
// a generic Summary, pulling temporal-specific fields (Language) when
// the spec is a temporal flow.
func List(root, flowID, kindFilter string) ([]Summary, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	specs, err := discovery.FindContracts(rootAbs)
	if err != nil {
		return nil, err
	}
	selected := discovery.Filter(specs, flowID)
	if flowID != "" && len(selected) == 0 {
		return nil, fmt.Errorf("unknown flow id %s", flowID)
	}
	if kindFilter != "" {
		filtered := make([]kind.Spec, 0, len(selected))
		for _, s := range selected {
			if s.Kind() == kindFilter {
				filtered = append(filtered, s)
			}
		}
		selected = filtered
	}
	out := make([]Summary, 0, len(selected))
	for _, s := range selected {
		summary := Summary{
			FlowID:       s.FlowID(),
			Kind:         s.Kind(),
			ContractPath: s.ContractPath(),
			SchemaVer:    s.SchemaVersion(),
		}
		if t, ok := s.(*temporal.Spec); ok {
			summary.Language = string(t.Flow().Layout.Language)
		}
		out = append(out, summary)
	}
	return out, nil
}

// Validate compiles every flow under root and returns the first
// compilation/contract error encountered, or nil if all are valid.
// Discovery itself is the validator — FindContracts compiles each flow.
func Validate(root, flowID, kindFilter string) ([]Summary, error) {
	return List(root, flowID, kindFilter)
}

// Explain renders the human-readable report for a single flow.
func Explain(root, flowID string) (string, error) {
	if flowID == "" {
		return "", fmt.Errorf("explain requires a flow id")
	}
	flows, err := discoverFiltered(root, flowID)
	if err != nil {
		return "", err
	}
	return buildExplainReport(root, flows[0]), nil
}

// FlowDetail is the structured projection of a single flow consumed by
// the UI's flow-detail page. The text `report` is kept alongside for the
// CLI's `flows explain` and for any caller that wants a printable view.
//
// Model and Runtime mirror the same-named blocks in flow.json so the
// UI's Overview tab can render module/seed/maxSteps/invariants and the
// runtime package metadata without a second round-trip to the
// scenario's source files.
type FlowDetail struct {
	FlowID        string               `json:"flowId"`
	Kind          string               `json:"kind"`
	Domain        string               `json:"domain,omitempty"`
	Description   string               `json:"description,omitempty"`
	ContractPath  string               `json:"contractPath"`
	Language      string               `json:"language"`
	SchemaVersion int                  `json:"schemaVersion"`
	InitialState  string               `json:"initialState"`
	States        []contract.State     `json:"states"`
	Events        []contract.Event     `json:"events"`
	Transitions   []model.Transition   `json:"transitions"`
	Traces        []contract.Trace     `json:"traces"`
	Invariants    []contract.Invariant `json:"invariants"`
	Model         contract.Model       `json:"model"`
	Runtime       contract.Runtime     `json:"runtime"`
	Report        string               `json:"report"`
}

// Detail returns the structured FlowDetail for a single flow id. Mirrors
// Explain's discovery path; the UI fetches this via GET /api/v1/flows/:id.
func Detail(root, flowID string) (FlowDetail, error) {
	if flowID == "" {
		return FlowDetail{}, fmt.Errorf("detail requires a flow id")
	}
	flows, err := discoverFiltered(root, flowID)
	if err != nil {
		return FlowDetail{}, err
	}
	flow := flows[0]
	return FlowDetail{
		FlowID:        flow.FlowID,
		Kind:          temporal.Name,
		Domain:        flow.Domain,
		Description:   flow.Description,
		ContractPath:  flow.ContractPath,
		Language:      string(flow.Layout.Language),
		SchemaVersion: flow.SchemaVersion,
		InitialState:  flow.Initial.ID,
		States:        append([]contract.State(nil), flow.States...),
		Events:        append([]contract.Event(nil), flow.Events...),
		Transitions:   flow.Matrix.Rows(),
		Traces:        append([]contract.Trace(nil), flow.Traces...),
		Invariants:    append([]contract.Invariant(nil), flow.Invariants...),
		Model:         flow.Model,
		Runtime:       flow.Runtime,
		Report:        buildExplainReport(root, flow),
	}, nil
}

// NewOptions configures `flows new`.
type NewOptions struct {
	Root      string
	ParentDir string
	FlowID    string
	// Kind selects which registered Kind to scaffold. Empty defaults to
	// temporal so existing CLI/handler callers keep working unchanged.
	Kind     string
	Language layout.Language
}

// New scaffolds a fresh flow directory and returns the relative path of
// the created flow dir. It dispatches to the registered Kind for the
// requested kind name; it does not run verification — the caller (CLI
// or HTTP handler) decides whether to chain a verify run.
func New(opts NewOptions) (string, error) {
	rootAbs, err := filepath.Abs(opts.Root)
	if err != nil {
		return "", err
	}
	kindName := opts.Kind
	if kindName == "" {
		kindName = temporal.Name
	}
	k, ok := kind.Get(kindName)
	if !ok {
		return "", fmt.Errorf("unknown kind %q (registered: %v)", kindName, kind.Names())
	}
	return k.Scaffold(kind.ScaffoldOptions{
		Root:      rootAbs,
		ParentDir: opts.ParentDir,
		FlowID:    opts.FlowID,
		Language:  kind.Language(opts.Language),
	})
}

// CodegenArtifact is a single file emitted by a kind's codegen pass,
// addressed by its repo-relative destination path.
type CodegenArtifact struct {
	Path    string
	Content []byte
}

// CodegenOptions configures Codegen.
type CodegenOptions struct {
	Root     string
	FlowID   string
	Language string
	// Write controls whether the artifacts are written to disk under
	// the resolved scenario root. Write=false is the default for the
	// HTTP surface; CLI passes Write=true.
	Write bool
}

// CodegenResult is what Codegen returns.
type CodegenResult struct {
	Artifacts []CodegenArtifact
	Written   []string
}

// Codegen runs the navigation kind's codegen pass for a single flow.
// Phase 4 only supports the navigation kind here; the temporal pipeline
// owns its own codegen path under verification.
func Codegen(opts CodegenOptions) (CodegenResult, error) {
	if opts.FlowID == "" {
		return CodegenResult{}, fmt.Errorf("codegen requires a flow id")
	}
	spec, err := loadSpec(opts.Root, opts.FlowID)
	if err != nil {
		return CodegenResult{}, err
	}
	if spec.Kind() != navigation.Name {
		return CodegenResult{}, fmt.Errorf("codegen: flow %s is kind %q; only navigation is supported in Phase 4", opts.FlowID, spec.Kind())
	}
	navKind, ok := kind.Get(navigation.Name)
	if !ok {
		return CodegenResult{}, fmt.Errorf("navigation kind not registered")
	}
	lang := kind.Language(opts.Language)
	if lang == "" {
		lang = kind.LanguageTypeScript
	}
	arts, err := navKind.Codegen(spec, lang)
	if err != nil {
		return CodegenResult{}, err
	}
	scenarioRoot := deriveScenarioRoot(spec.ContractPath())
	result := CodegenResult{}
	for path, content := range arts.Files {
		result.Artifacts = append(result.Artifacts, CodegenArtifact{Path: path, Content: content})
	}
	if opts.Write {
		for _, a := range result.Artifacts {
			dest := filepath.Join(scenarioRoot, a.Path)
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return CodegenResult{}, fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
			}
			if err := os.WriteFile(dest, a.Content, 0o644); err != nil {
				return CodegenResult{}, fmt.Errorf("write %s: %w", dest, err)
			}
			result.Written = append(result.Written, dest)
		}
	}
	return result, nil
}

// ReconcileFinding is one discrepancy reported by Reconcile.
type ReconcileFinding = navigationreconcile.Finding

// ReconcileResult is what Reconcile returns.
type ReconcileResult struct {
	Passed       bool
	Findings     []ReconcileFinding
	FilesScanned int
}

// ReconcileOptions configures Reconcile.
type ReconcileOptions struct {
	Root         string
	FlowID       string
	ScenarioRoot string
}

// Reconcile runs the navigation kind's reconciler against a scenario's
// ui/src tree. ScenarioRoot empty derives from the flow's contract path.
func Reconcile(opts ReconcileOptions) (ReconcileResult, error) {
	if opts.FlowID == "" {
		return ReconcileResult{}, fmt.Errorf("reconcile requires a flow id")
	}
	spec, err := loadSpec(opts.Root, opts.FlowID)
	if err != nil {
		return ReconcileResult{}, err
	}
	if spec.Kind() != navigation.Name {
		return ReconcileResult{}, fmt.Errorf("reconcile: flow %s is kind %q; only navigation is supported", opts.FlowID, spec.Kind())
	}
	navSpec, ok := spec.(*navigation.Spec)
	if !ok {
		return ReconcileResult{}, fmt.Errorf("reconcile: spec is %T, want *navigation.Spec", spec)
	}
	scenarioRoot := opts.ScenarioRoot
	if scenarioRoot == "" {
		scenarioRoot = deriveScenarioRoot(spec.ContractPath())
	}
	res, err := navigationreconcile.Run(navSpec.Graph(), scenarioRoot)
	if err != nil {
		return ReconcileResult{}, err
	}
	return ReconcileResult{
		Passed:       res.Passed,
		Findings:     res.Findings,
		FilesScanned: res.FilesScanned,
	}, nil
}

// NavigationStudioOptions configures NavigationStudio.
type NavigationStudioOptions struct {
	Root   string
	FlowID string
}

// NavigationStudio loads a navigation flow, runs the reachability
// verifier, and returns the Flow Studio descriptor (routes, affordances,
// containers, contexts, invariant pass/fail) the UI's navigation
// renderer consumes.
func NavigationStudio(opts NavigationStudioOptions) (navigationstudio.Descriptor, error) {
	if opts.FlowID == "" {
		return navigationstudio.Descriptor{}, fmt.Errorf("navigation studio requires a flow id")
	}
	spec, err := loadSpec(opts.Root, opts.FlowID)
	if err != nil {
		return navigationstudio.Descriptor{}, err
	}
	if spec.Kind() != navigation.Name {
		return navigationstudio.Descriptor{}, fmt.Errorf("navigation studio: flow %s is kind %q; only navigation is supported", opts.FlowID, spec.Kind())
	}
	navSpec, ok := spec.(*navigation.Spec)
	if !ok {
		return navigationstudio.Descriptor{}, fmt.Errorf("navigation studio: spec is %T, want *navigation.Spec", spec)
	}
	return navigationstudio.VerifyAndBuild(context.Background(), navSpec.Graph())
}

// loadSpec discovers and returns the single spec matching flowID under root.
func loadSpec(root, flowID string) (kind.Spec, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	specs, err := discovery.FindContracts(rootAbs)
	if err != nil {
		return nil, err
	}
	selected := discovery.Filter(specs, flowID)
	if len(selected) == 0 {
		return nil, fmt.Errorf("unknown flow id %s", flowID)
	}
	return selected[0], nil
}

// deriveScenarioRoot infers the scenario root from a contract path that
// follows the `<scenario>/<feature>/flow/<name>.json` convention. For
// navigation under `<scenario>/ui/flow/navigation.json` the result is
// `<scenario>`. Falls back to the contract's parent's parent for
// anything else.
func deriveScenarioRoot(contractPath string) string {
	dir := filepath.Dir(contractPath) // .../ui/flow
	if filepath.Base(dir) == "flow" {
		dir = filepath.Dir(dir) // .../ui
		if filepath.Base(dir) == "ui" {
			return filepath.Dir(dir)
		}
		return filepath.Dir(dir)
	}
	return filepath.Dir(dir)
}

func discoverFiltered(root, flowID string) ([]model.Flow, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	specs, err := discovery.FindContracts(rootAbs)
	if err != nil {
		return nil, err
	}
	selected := discovery.Filter(specs, flowID)
	if flowID != "" && len(selected) == 0 {
		return nil, fmt.Errorf("unknown flow id %s", flowID)
	}
	return temporal.FlowsFromSpecs(selected), nil
}

func buildExplainReport(root string, flow model.Flow) string {
	coverage := model.NamedTraceCoverage(flow)
	var b strings.Builder
	handAuthored := handAuthoredFiles(flow)
	writeSection(&b, "",
		pair("flow", flow.FlowID),
		pair("contract", flow.ContractPath),
		pair("layout", fmt.Sprintf("hand-authored: %d file(s) at %s/; generated: %s/", len(handAuthored), flow.Layout.BaseDir, flow.Layout.BaseDir+"/"+layout.GeneratedDirName)),
	)
	writeSection(&b, "Hand-authored (edit these)", handAuthored...)
	writeSection(&b, "Generated (regenerated; do not edit)",
		pair("model", flow.Layout.ModelPath),
		pair("artifact", flow.Layout.ArtifactPath),
		pair("runtime", flow.Layout.RuntimePath),
		pair("replay helper", flow.Layout.ReplayHelperPath),
	)
	writeSection(&b, "Runtime",
		pair("language", string(flow.Layout.Language)),
		pair("status type", runtimeStatusType(flow)),
		pair("event type", runtimeEventType(flow)),
		pair("generated runtime unions", yesNo(hasGeneratedRuntimeUnions(flow))),
		pair("fixture contract", yesNo(flow.Layout.Language == layout.LanguageTypeScript)),
		pair("subpackage import", layout.SubpackageImportPath(flow.Layout)),
	)
	writeSection(&b, "Topology",
		pair("states", fmt.Sprintf("%d (initial %s; terminal %s)", len(flow.States), flow.Initial.ID, terminalSummary(flow))),
		pair("events", fmt.Sprint(len(flow.Events))),
		pair("expanded transitions", fmt.Sprint(flow.Matrix.Len())),
		pair("invalid transitions", fmt.Sprint(invalidTransitionCount(flow))),
	)
	writeSection(&b, "Replay",
		pair("transition", flow.Replay.Transition.Function),
		pair("fixture", fixtureSummary(flow)),
	)
	writeSection(&b, "Coverage requirements",
		pair("named traces", fmt.Sprint(len(flow.Traces))),
		pair("named trace states", coverageSummary(coverage.CoveredStates, coverage.MissingStates)),
		pair("named trace events", coverageSummary(coverage.CoveredEvents, coverage.MissingEvents)),
	)
	writeSection(&b, "Commands",
		pair("regenerate", fmt.Sprintf("flow-verifier verify run --root %s --flow %s", filepath.ToSlash(root), flow.FlowID)),
		pair("check", "flow-verifier verify check"),
	)
	return b.String()
}

func writeSection(b *strings.Builder, title string, rows ...string) {
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	if title != "" {
		fmt.Fprintf(b, "%s:\n", title)
	}
	for _, row := range rows {
		if title == "" {
			fmt.Fprintln(b, row)
		} else {
			fmt.Fprintf(b, "  %s\n", row)
		}
	}
}

func pair(label, value string) string { return label + ": " + value }

func handAuthoredFiles(flow model.Flow) []string {
	out := []string{
		"contract: " + flow.ContractPath,
		"wrapper: " + flow.Layout.TransitionPath,
	}
	if flow.Layout.FixturesPath != "" {
		out = append(out, "fixtures: "+flow.Layout.FixturesPath)
	}
	out = append(out, "thin replay test: "+flow.Layout.TestPath)
	return out
}

func fixtureSummary(flow model.Flow) string {
	if flow.Layout.FixturesPath == "" {
		return "n/a (go-test replay)"
	}
	return fmt.Sprintf("%s (export %s)", flow.Layout.FixturesPath, codegen.TypeScriptFixturesExportName(flow))
}

func runtimeStatusType(flow model.Flow) string {
	if flow.Runtime.Go != nil {
		return flow.Runtime.Go.StatusType
	}
	if flow.Runtime.TypeScript != nil {
		return flow.Runtime.TypeScript.StatusType
	}
	return ""
}

func runtimeEventType(flow model.Flow) string {
	if flow.Runtime.Go != nil {
		return flow.Runtime.Go.EventType
	}
	if flow.Runtime.TypeScript != nil {
		return flow.Runtime.TypeScript.EventType
	}
	return ""
}

func hasGeneratedRuntimeUnions(flow model.Flow) bool {
	return flow.Runtime.TypeScript != nil && flow.Runtime.TypeScript.StateUnionType != "" && flow.Runtime.TypeScript.EventUnionType != ""
}

func terminalSummary(flow model.Flow) string {
	var terminal []string
	for _, state := range flow.States {
		if state.Terminal {
			terminal = append(terminal, state.ID)
		}
	}
	if len(terminal) == 0 {
		return "none"
	}
	return strings.Join(terminal, ", ")
}

func invalidTransitionCount(flow model.Flow) int {
	total := 0
	for _, transition := range flow.Matrix.Rows() {
		if transition.WantError {
			total++
		}
	}
	return total
}

func coverageSummary(covered, missing []string) string {
	if len(missing) == 0 {
		return "all covered"
	}
	return fmt.Sprintf("covered %s; missing %s", strings.Join(covered, ", "), strings.Join(missing, ", "))
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
