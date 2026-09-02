// Package transitions loads the inspectable Swarm-to-Agent-Manager transition
// registry. It deliberately selects a mechanism; it never describes how an
// agent workflow executes.
package transitions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SchemaVersion = "swarm-transition/v1"

type Kind string

const (
	SessionMetaOrchestrationKind = "meta_orchestration"
	SessionSwarmOperationsKind   = "swarm_operations"
	SessionWorkflowAuthoringKind = "workflow_authoring"
)

const (
	KindSession       Kind = "session"
	KindWorkflow      Kind = "workflow"
	KindDeterministic Kind = "deterministic"
)

// Definition is one domain transition. Workflow mechanics belong exclusively
// to the Agent Manager declaration identified by Workflow.
type Definition struct {
	SchemaVersion    string              `json:"schemaVersion"`
	Key              string              `json:"key"`
	Subject          string              `json:"subject"`
	Kind             Kind                `json:"kind"`
	Workflow         *Locator            `json:"workflow,omitempty"`
	Requires         []string            `json:"requires,omitempty"`
	InputContract    string              `json:"inputContract"`
	TerminalOutcomes []string            `json:"terminalOutcomes"`
	ApplyAction      string              `json:"applyAction"`
	HumanGates       []HumanGate         `json:"humanGates,omitempty"`
	HumanWait        bool                `json:"humanWait,omitempty"`
	Strategies       []ExecutionStrategy `json:"strategies,omitempty"`
	Session          *SessionConfig      `json:"session,omitempty"`
}

type GateMode string

const (
	GateModeManual  GateMode = "manual"
	GateModeSuggest GateMode = "suggest"
	GateModeAuto    GateMode = "auto"
)

// HumanGate declares one operator decision attached to its transition. The
// declaration is the source of truth for readiness and policy projection.
type HumanGate struct {
	ID          string   `json:"id"`
	Decides     string   `json:"decides"`
	DefaultMode GateMode `json:"defaultMode"`
	Threshold   float64  `json:"threshold"`
	MinSample   int      `json:"minSample"`
}

// EffectiveGateMode resolves the live operator override for a declared gate.
// An absent override deliberately falls back to the versioned declaration.
func EffectiveGateMode(definition Definition, overrides map[string]string, gateID string) (GateMode, bool) {
	for _, gate := range definition.HumanGates {
		if gate.ID != strings.TrimSpace(gateID) {
			continue
		}
		mode := gate.DefaultMode
		if override, ok := overrides[gate.ID]; ok {
			switch GateMode(strings.TrimSpace(override)) {
			case GateModeManual, GateModeSuggest, GateModeAuto:
				mode = GateMode(strings.TrimSpace(override))
			}
		}
		return mode, true
	}
	return "", false
}

// ExecutionStrategy is operator-facing metadata for an execution-capable
// transition. The registry remains the declaration authority; consumers only
// select an id from this bounded list.
type ExecutionStrategy struct {
	ID          string `json:"id"`
	WorkflowKey string `json:"workflowKey"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	WhenToUse   string `json:"whenToUse"`
	CostBand    string `json:"costBand"`
}

type SessionConfig struct {
	BriefRef        string   `json:"briefRef"`
	SkillID         string   `json:"skillId"`
	ProfileKey      string   `json:"profileKey"`
	ProposalTargets []string `json:"proposalTargets,omitempty"`
}

type Locator struct {
	Owner string `json:"owner"`
	Key   string `json:"key"`
}

type workflowDeclaration struct {
	SchemaVersion string `json:"schemaVersion"`
	Key           string `json:"key"`
	Nodes         []struct {
		Kind          string `json:"kind"`
		ChildWorkflow *struct {
			WorkflowKey string `json:"workflowKey"`
		} `json:"childWorkflow"`
	} `json:"nodes"`
}

// Registry is immutable after loading. Callers select a definition by its
// stable key and use its requirements for integration preflight.
type Registry struct{ byKey map[string]Definition }

func (r Registry) Get(key string) (Definition, bool) {
	definition, ok := r.byKey[strings.TrimSpace(key)]
	return definition, ok
}

// ResolveWorkflow returns the Agent Manager locator declared for a workflow
// transition. Consumers select the domain transition they intend to perform;
// they do not embed Agent Manager workflow identifiers in their own code.
func (r Registry) ResolveWorkflow(transitionKey string) (Locator, error) {
	definition, ok := r.Get(transitionKey)
	if !ok {
		return Locator{}, fmt.Errorf("transition %q is not registered", strings.TrimSpace(transitionKey))
	}
	if definition.Kind != KindWorkflow || definition.Workflow == nil {
		return Locator{}, fmt.Errorf("transition %q does not declare a workflow", definition.Key)
	}
	return *definition.Workflow, nil
}

func (r Registry) Definitions() []Definition {
	definitions := make([]Definition, 0, len(r.byKey))
	for _, definition := range r.byKey {
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Key < definitions[j].Key })
	return definitions
}

// ValidateWorkflowReachability ensures every declared Agent Manager workflow
// is reachable from a transition root, directly or through child_workflow
// edges. An intentionally dormant workflow must have a non-empty reason.
func ValidateWorkflowReachability(registry Registry, fsys fs.FS, dir string, unboundReasons map[string]string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("read workflow declarations: %w", err)
	}
	declarations := make(map[string]workflowDeclaration)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		contents, readErr := fs.ReadFile(fsys, filepath.Join(dir, entry.Name()))
		if readErr != nil {
			return fmt.Errorf("read workflow declaration %s: %w", entry.Name(), readErr)
		}
		var declaration workflowDeclaration
		if decodeErr := json.Unmarshal(contents, &declaration); decodeErr != nil {
			return fmt.Errorf("decode workflow declaration %s: %w", entry.Name(), decodeErr)
		}
		if declaration.SchemaVersion != "agent-workflow/v1" {
			continue
		}
		key := strings.TrimSpace(declaration.Key)
		if key == "" {
			return fmt.Errorf("workflow declaration %s has no key", entry.Name())
		}
		if _, duplicate := declarations[key]; duplicate {
			return fmt.Errorf("duplicate workflow declaration %q", key)
		}
		declarations[key] = declaration
	}

	queue := make([]string, 0)
	for _, definition := range registry.Definitions() {
		if definition.Kind != KindWorkflow || definition.Workflow == nil {
			continue
		}
		queue = append(queue, strings.TrimSpace(definition.Workflow.Key))
		for _, strategy := range definition.Strategies {
			queue = append(queue, strings.TrimSpace(strategy.WorkflowKey))
		}
	}
	reachable := make(map[string]struct{}, len(declarations))
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if key == "" {
			continue
		}
		if _, seen := reachable[key]; seen {
			continue
		}
		declaration, declared := declarations[key]
		if !declared {
			return fmt.Errorf("reachable workflow %q has no declaration", key)
		}
		reachable[key] = struct{}{}
		for _, node := range declaration.Nodes {
			if node.Kind == "child_workflow" && node.ChildWorkflow != nil {
				queue = append(queue, strings.TrimSpace(node.ChildWorkflow.WorkflowKey))
			}
		}
	}

	unbound := make([]string, 0)
	for key := range declarations {
		if _, ok := reachable[key]; ok {
			continue
		}
		if strings.TrimSpace(unboundReasons[key]) == "" {
			unbound = append(unbound, key)
		}
	}
	if len(unbound) > 0 {
		sort.Strings(unbound)
		return fmt.Errorf("declared workflows have no transition or reachable child binding and no recorded reason: %s", strings.Join(unbound, ", "))
	}
	return nil
}

// IsSessionKind reports whether the registry declares a complete session
// definition for kind. Session consumers use this instead of maintaining a
// second list of extensible session kinds in Go code.
func (r Registry) IsSessionKind(kind string) bool {
	definition, ok := r.Get("session." + strings.TrimSpace(kind))
	return ok && definition.Kind == KindSession && definition.Session != nil
}

// VerifyApplyActions ensures every selected transition kind has a concrete
// domain dispatcher at composition time.
func VerifyApplyActions(registry Registry, actions map[string]struct{}, kinds ...Kind) error {
	selected := map[Kind]struct{}{}
	for _, kind := range kinds {
		selected[kind] = struct{}{}
	}
	missing := map[string]struct{}{}
	for _, definition := range registry.Definitions() {
		if _, ok := selected[definition.Kind]; !ok || definition.ApplyAction == "" {
			continue
		}
		if _, ok := actions[definition.ApplyAction]; !ok {
			missing[definition.ApplyAction] = struct{}{}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	values := make([]string, 0, len(missing))
	for action := range missing {
		values = append(values, action)
	}
	sort.Strings(values)
	return fmt.Errorf("transition dispatch table missing apply actions: %s", strings.Join(values, ", "))
}

// RenderCatalogMarkdown projects the transition registry into the checked-in
// operator catalog. Keep this intentionally mechanical: descriptive prose
// belongs in the registry's contracts, not in a second hand-maintained table.
func RenderCatalogMarkdown(registry Registry) string {
	var out strings.Builder
	out.WriteString("<!-- Code generated by internal/transitions.RenderCatalogMarkdown; DO NOT EDIT. -->\n\n")
	out.WriteString("# Active transition catalog\n\n")
	out.WriteString("Source: [`../../.vrooli/swarm-transitions/registry.json`](../../.vrooli/swarm-transitions/registry.json).\n\n")
	out.WriteString("| Transition | Kind | Subject | Workflow | Input contract | Requires | Terminal outcomes | Apply action |\n")
	out.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, definition := range registry.Definitions() {
		workflow := "—"
		if definition.Workflow != nil {
			workflow = definition.Workflow.Owner + "/" + strings.TrimPrefix(definition.Workflow.Key, definition.Workflow.Owner+"/")
		}
		requires := "—"
		if len(definition.Requires) > 0 {
			requires = strings.Join(definition.Requires, ", ")
		}
		outcomes := strings.Join(definition.TerminalOutcomes, ", ")
		fmt.Fprintf(&out, "| `%s` | %s | %s | `%s` | `%s` | %s | %s | `%s` |\n", definition.Key, definition.Kind, definition.Subject, workflow, definition.InputContract, requires, outcomes, definition.ApplyAction)
	}
	return out.String()
}

func LoadDir(dir string) (Registry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Registry{}, fmt.Errorf("read transition registry: %w", err)
	}
	registry := Registry{byKey: make(map[string]Definition, len(entries))}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		contents, err := os.ReadFile(path)
		if err != nil {
			return Registry{}, fmt.Errorf("read %s: %w", path, err)
		}
		definitions, err := decodeDefinitions(contents)
		if err != nil {
			return Registry{}, fmt.Errorf("decode %s: %w", path, err)
		}
		if err := addDefinitions(registry.byKey, definitions); err != nil {
			return Registry{}, fmt.Errorf("validate %s: %w", path, err)
		}
	}
	if len(registry.byKey) == 0 {
		return Registry{}, fmt.Errorf("transition registry %s contains no JSON definitions", dir)
	}
	return registry, nil
}

func Validate(definition Definition) error {
	if definition.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schemaVersion must be %q", SchemaVersion)
	}
	for field, value := range map[string]string{
		"key": definition.Key, "subject": definition.Subject,
		"inputContract": definition.InputContract, "applyAction": definition.ApplyAction,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if len(definition.TerminalOutcomes) == 0 {
		return fmt.Errorf("terminalOutcomes is required")
	}
	seenOutcome := make(map[string]struct{}, len(definition.TerminalOutcomes))
	for _, outcome := range definition.TerminalOutcomes {
		outcome = strings.TrimSpace(outcome)
		if outcome == "" {
			return fmt.Errorf("terminalOutcomes cannot contain an empty value")
		}
		if _, duplicate := seenOutcome[outcome]; duplicate {
			return fmt.Errorf("terminalOutcomes contains duplicate %q", outcome)
		}
		seenOutcome[outcome] = struct{}{}
	}
	switch definition.Kind {
	case KindWorkflow:
		if definition.Workflow == nil || strings.TrimSpace(definition.Workflow.Owner) == "" || strings.TrimSpace(definition.Workflow.Key) == "" {
			return fmt.Errorf("workflow kind requires workflow.owner and workflow.key")
		}
	case KindSession, KindDeterministic:
		if definition.Workflow != nil {
			return fmt.Errorf("%s kind must not define workflow", definition.Kind)
		}
	default:
		return fmt.Errorf("kind must be session, workflow, or deterministic")
	}
	if definition.Kind == KindSession {
		if definition.Session == nil || strings.TrimSpace(definition.Session.BriefRef) == "" || strings.TrimSpace(definition.Session.SkillID) == "" || strings.TrimSpace(definition.Session.ProfileKey) == "" {
			return fmt.Errorf("session definitions require a complete session block")
		}
	} else if definition.Session != nil {
		return fmt.Errorf("session block is only valid for session definitions")
	}
	seenRequirement := make(map[string]struct{}, len(definition.Requires))
	for _, requirement := range definition.Requires {
		requirement = strings.TrimSpace(requirement)
		if requirement == "" {
			return fmt.Errorf("requires cannot contain an empty value")
		}
		if _, duplicate := seenRequirement[requirement]; duplicate {
			return fmt.Errorf("requires contains duplicate %q", requirement)
		}
		seenRequirement[requirement] = struct{}{}
	}
	seenStrategy := make(map[string]struct{}, len(definition.Strategies))
	for _, strategy := range definition.Strategies {
		strategy.ID = strings.TrimSpace(strategy.ID)
		workflowKey := strings.TrimSpace(strategy.WorkflowKey)
		if workflowKey == "" && definition.Workflow != nil {
			workflowKey = strings.TrimSpace(definition.Workflow.Key)
		}
		if strategy.ID == "" || workflowKey == "" || strings.TrimSpace(strategy.DisplayName) == "" || strings.TrimSpace(strategy.Description) == "" || strings.TrimSpace(strategy.WhenToUse) == "" || strings.TrimSpace(strategy.CostBand) == "" {
			return fmt.Errorf("strategies require id, workflowKey, displayName, description, whenToUse, and costBand")
		}
		if _, duplicate := seenStrategy[strategy.ID]; duplicate {
			return fmt.Errorf("strategies contains duplicate %q", strategy.ID)
		}
		seenStrategy[strategy.ID] = struct{}{}
	}
	seenGate := make(map[string]struct{}, len(definition.HumanGates))
	for _, gate := range definition.HumanGates {
		gate.ID = strings.TrimSpace(gate.ID)
		gate.Decides = strings.TrimSpace(gate.Decides)
		if gate.ID == "" || gate.Decides == "" {
			return fmt.Errorf("humanGates require id and decides")
		}
		if _, duplicate := seenGate[gate.ID]; duplicate {
			return fmt.Errorf("humanGates contains duplicate %q", gate.ID)
		}
		seenGate[gate.ID] = struct{}{}
		switch gate.DefaultMode {
		case GateModeManual, GateModeSuggest, GateModeAuto:
		default:
			return fmt.Errorf("humanGate %q has invalid defaultMode %q", gate.ID, gate.DefaultMode)
		}
		if gate.Threshold < 0 || gate.Threshold > 1 {
			return fmt.Errorf("humanGate %q threshold must be between 0 and 1", gate.ID)
		}
		if gate.MinSample <= 0 {
			return fmt.Errorf("humanGate %q minSample must be positive", gate.ID)
		}
	}
	if definition.HumanWait && len(definition.HumanGates) == 0 {
		return fmt.Errorf("transition %q declares a human wait without a humanGate", definition.Key)
	}
	return nil
}

// LoadFS is the test-friendly form of LoadDir. Keeping validation independent
// of process paths makes malformed registry fixtures deterministic.
func LoadFS(fsys fs.FS, dir string) (Registry, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return Registry{}, err
	}
	registry := Registry{byKey: make(map[string]Definition, len(entries))}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		contents, err := fs.ReadFile(fsys, filepath.Join(dir, entry.Name()))
		if err != nil {
			return Registry{}, err
		}
		definitions, err := decodeDefinitions(contents)
		if err != nil {
			return Registry{}, err
		}
		if err := addDefinitions(registry.byKey, definitions); err != nil {
			return Registry{}, err
		}
	}
	if len(registry.byKey) == 0 {
		return Registry{}, fmt.Errorf("transition registry contains no JSON definitions")
	}
	return registry, nil
}

func decodeDefinitions(contents []byte) ([]Definition, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var definitions []Definition
	if len(bytes.TrimSpace(contents)) > 0 && bytes.TrimSpace(contents)[0] == '[' {
		if err := decoder.Decode(&definitions); err != nil {
			return nil, err
		}
		return definitions, nil
	}
	var definition Definition
	if err := decoder.Decode(&definition); err != nil {
		return nil, err
	}
	return []Definition{definition}, nil
}

func addDefinitions(target map[string]Definition, definitions []Definition) error {
	if len(definitions) == 0 {
		return fmt.Errorf("file contains no transition definitions")
	}
	for _, definition := range definitions {
		for i := range definition.Strategies {
			if strings.TrimSpace(definition.Strategies[i].WorkflowKey) == "" && definition.Workflow != nil {
				definition.Strategies[i].WorkflowKey = definition.Workflow.Key
			}
		}
		if err := Validate(definition); err != nil {
			return err
		}
		if _, exists := target[definition.Key]; exists {
			return fmt.Errorf("duplicate transition key %q", definition.Key)
		}
		target[definition.Key] = definition
	}
	return nil
}
