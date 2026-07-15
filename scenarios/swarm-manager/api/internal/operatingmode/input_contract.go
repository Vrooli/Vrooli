package operatingmode

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"swarm-manager/internal/operatingmode/promptcatalog"
)

type (
	InputFreshness     string
	InputFailurePolicy string
)

const (
	InputFreshnessStatic    InputFreshness = "static"
	InputFreshnessExecution InputFreshness = "execution"
	InputFreshnessRound     InputFreshness = "round"

	InputFailureRequired InputFailurePolicy = "required"
	InputFailureDegrade  InputFailurePolicy = "degrade"

	maxCallerInputBytes         = 256 * 1024
	maxCallerInputSnapshotBytes = 1024 * 1024
)

// ProviderCapabilityDescriptor is the compiler-visible half of an input
// provider. Retrieval remains code, but compatibility, freshness, and failure
// behavior are data so validation never reverse-engineers a runtime switch.
type ProviderCapabilityDescriptor struct {
	ID            string
	SourceKind    InputSourceKind
	Type          InputValueType
	TargetKinds   []TargetKind
	Freshness     InputFreshness
	FailurePolicy InputFailurePolicy
}

// InputProviderCapabilities returns a copy of the complete typed capability
// registry. A descriptor with no TargetKinds supports every target kind.
func InputProviderCapabilities() map[string]ProviderCapabilityDescriptor {
	descriptors := []ProviderCapabilityDescriptor{
		{ID: "generic.operating_mode", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessStatic, FailurePolicy: InputFailureRequired},
		{ID: "generic.mode_label", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessStatic, FailurePolicy: InputFailureRequired},
		{ID: "generic.phase", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "generic.run_strategy", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessStatic, FailurePolicy: InputFailureRequired},
		{ID: "generic.round_number", SourceKind: InputSourceGenericProvider, Type: InputTypeInteger, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "generic.agent_profile_key", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "generic.operator_note", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		// Structured caller-context providers. Like generic.operator_note, each reads
		// an optional per-run string the caller supplied (RunContext.OperatorInputs),
		// keyed by the operation caller-input name. They give an operation a typed
		// sink for its request context WITHOUT collapsing everything into the operator
		// note and WITHOUT a mode caller-input (the empty-set engine invariant holds).
		// The clarify sinks carry the operator's clarification turn context; the
		// research sinks carry the legacy research request context faithfully.
		{ID: "generic.user_question", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.decision_topic", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.user_message", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.user_prompt", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.context_paths", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.context_targets", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.context_requirements", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.gap_report", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.evidence_request", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessRound, FailurePolicy: InputFailureDegrade},
		{ID: "generic.prior_rounds", SourceKind: InputSourceGenericProvider, Type: InputTypeArray, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "generic.mode_artifacts", SourceKind: InputSourceGenericProvider, Type: InputTypeArray, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "generic.backlog_sync_proposal", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessStatic, FailurePolicy: InputFailureRequired},
		{ID: "generic.elastic_slice", SourceKind: InputSourceGenericProvider, Type: InputTypeString, Freshness: InputFreshnessStatic, FailurePolicy: InputFailureRequired},
		{ID: "target.initiative_name", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetInitiative}, Freshness: InputFreshnessExecution, FailurePolicy: InputFailureRequired},
		{ID: "target.initiative_title", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetInitiative}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.initiative_description", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetInitiative}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.acceptance_criteria", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetInitiative}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.member_items", SourceKind: InputSourceTargetAdapter, Type: InputTypeArray, TargetKinds: []TargetKind{TargetInitiative}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.plan_context", SourceKind: InputSourceTargetAdapter, Type: InputTypeObject, TargetKinds: []TargetKind{TargetInitiative, TargetPlanExecution}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.plan_id", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetPlanExecution}, Freshness: InputFreshnessExecution, FailurePolicy: InputFailureRequired},
		{ID: "target.scenario_name", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetScenario}, Freshness: InputFreshnessExecution, FailurePolicy: InputFailureRequired},
		{ID: "target.item_title", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetBacklogItem}, Freshness: InputFreshnessExecution, FailurePolicy: InputFailureRequired},
		{ID: "target.item_description", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetBacklogItem}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.item_status", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetBacklogItem}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.item_spec", SourceKind: InputSourceTargetAdapter, Type: InputTypeString, TargetKinds: []TargetKind{TargetBacklogItem}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "target.item_plan_ref", SourceKind: InputSourceTargetAdapter, Type: InputTypeObject, TargetKinds: []TargetKind{TargetBacklogItem}, Freshness: InputFreshnessRound, FailurePolicy: InputFailureRequired},
		{ID: "derived.sha256", SourceKind: InputSourceDerived, Type: InputTypeString, Freshness: InputFreshnessExecution, FailurePolicy: InputFailureRequired},
	}
	out := make(map[string]ProviderCapabilityDescriptor, len(descriptors))
	for _, descriptor := range descriptors {
		descriptor.TargetKinds = append([]TargetKind(nil), descriptor.TargetKinds...)
		out[descriptor.ID] = descriptor
	}
	return out
}

type CompiledInputContract struct {
	SchemaVersion string                      `json:"schema_version"`
	RootMode      Mode                        `json:"root_mode"`
	Modes         []CompiledModeInputContract `json:"modes"`
}

type CompiledModeInputContract struct {
	Mode   Mode                         `json:"mode"`
	Target TargetKind                   `json:"target"`
	Inputs []CompiledInput              `json:"inputs"`
	Phases []CompiledPhaseInputContract `json:"phases"`
}

type CompiledInput struct {
	Spec    InputSpec          `json:"spec"`
	Source  InputSourceBinding `json:"source"`
	Aliases []string           `json:"aliases"`
}

type CompiledPhaseInputContract struct {
	Phase    Phase                       `json:"phase"`
	Bindings []CompiledPhaseInputBinding `json:"bindings"`
}

type CompiledPhaseInputBinding struct {
	Variable string `json:"variable"`
	InputID  string `json:"input_id"`
}

// ValidateCallerInputSnapshot validates execution-scoped caller values against
// the complete pinned parent + delegated input contract. The returned JSON is
// normalized once and is therefore safe to hash and persist as the runtime
// snapshot used by later rounds.
//
// Caller values currently require value retention and non-sensitive
// classification. Digest/omit retention cannot satisfy replay after a process
// restart, while sensitive value retention is prohibited by the compiler; the
// honest behavior is to reject that combination until a secure runtime value
// store exists instead of persisting a secret or dispatching an unreplayable
// execution.
func ValidateCallerInputSnapshot(compiled CompiledInputContract, supplied map[string]any) (json.RawMessage, string, map[string]any, error) {
	specs := map[string]InputSpec{}
	for _, mode := range compiled.Modes {
		for _, input := range mode.Inputs {
			if input.Source.Kind != InputSourceCaller {
				continue
			}
			if existing, ok := specs[input.Spec.ID]; ok {
				existingJSON, _ := json.Marshal(existing)
				currentJSON, _ := json.Marshal(input.Spec)
				if string(existingJSON) != string(currentJSON) {
					return nil, "", nil, fmt.Errorf("caller input %q has conflicting transitive specifications", input.Spec.ID)
				}
				continue
			}
			specs[input.Spec.ID] = input.Spec
		}
	}

	unknown := make([]string, 0)
	for id := range supplied {
		if _, ok := specs[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, "", nil, fmt.Errorf("unknown caller inputs: %s", strings.Join(unknown, ", "))
	}

	ids := make([]string, 0, len(specs))
	for id := range specs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	normalized := make(map[string]any, len(supplied))
	retention := make(map[string]any, len(specs))
	for _, id := range ids {
		spec := specs[id]
		value, present := supplied[id]
		if !present {
			if spec.Required {
				return nil, "", nil, fmt.Errorf("required caller input %q is missing", id)
			}
			retention[id] = map[string]any{
				"present": false, "sensitivity": spec.Sensitivity, "retention": spec.Retention,
			}
			continue
		}
		if spec.Sensitivity == InputSensitivitySensitive {
			return nil, "", nil, fmt.Errorf("caller input %q is sensitive and cannot be retained for replay", id)
		}
		if spec.Retention != InputRetentionValue {
			return nil, "", nil, fmt.Errorf("caller input %q uses %q retention; replayable caller inputs require value retention", id, spec.Retention)
		}
		if err := validateInputValue(spec, value); err != nil {
			return nil, "", nil, fmt.Errorf("caller input %q: %w", id, err)
		}
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return nil, "", nil, fmt.Errorf("caller input %q is not JSON: %w", id, err)
		}
		if len(valueJSON) > maxCallerInputBytes {
			return nil, "", nil, fmt.Errorf("caller input %q encoded size %d exceeds maximum %d bytes", id, len(valueJSON), maxCallerInputBytes)
		}
		var canonical any
		if err := json.Unmarshal(valueJSON, &canonical); err != nil {
			return nil, "", nil, fmt.Errorf("normalize caller input %q: %w", id, err)
		}
		normalized[id] = canonical
		valueDigest := sha256.Sum256(valueJSON)
		retention[id] = map[string]any{
			"present": true, "sensitivity": spec.Sensitivity, "retention": spec.Retention,
			"value_digest": fmt.Sprintf("sha256:%x", valueDigest[:]),
		}
	}

	snapshot, err := json.Marshal(normalized)
	if err != nil {
		return nil, "", nil, fmt.Errorf("marshal caller input snapshot: %w", err)
	}
	if len(snapshot) > maxCallerInputSnapshotBytes {
		return nil, "", nil, fmt.Errorf("caller input snapshot encoded size %d exceeds maximum %d bytes", len(snapshot), maxCallerInputSnapshotBytes)
	}
	digest, err := canonicalJSONDigest(snapshot)
	if err != nil {
		return nil, "", nil, fmt.Errorf("digest caller input snapshot: %w", err)
	}
	return snapshot, digest, retention, nil
}

// CompileInputContract compiles the root and every reachable delegated mode
// into one deterministic contract. It is deliberately pure: callers may use
// it for load validation, simulation, UI schemas, or execution preflight and
// receive byte-stable JSON for the same definitions.
func CompileInputContract(defs map[Mode]Definition, root Definition) (CompiledInputContract, error) {
	modes, err := reachableInputModes(defs, root)
	if err != nil {
		return CompiledInputContract{}, err
	}
	compiled := CompiledInputContract{SchemaVersion: "1", RootMode: root.Mode}
	for _, def := range modes {
		modeContract, err := compileModeInputContract(def)
		if err != nil {
			return CompiledInputContract{}, fmt.Errorf("mode %q input contract: %w", def.Mode, err)
		}
		compiled.Modes = append(compiled.Modes, modeContract)
	}
	return compiled, nil
}

func reachableInputModes(defs map[Mode]Definition, root Definition) ([]Definition, error) {
	seen := map[Mode]bool{}
	var out []Definition
	var visit func(Definition) error
	visit = func(def Definition) error {
		if seen[def.Mode] {
			return nil
		}
		seen[def.Mode] = true
		out = append(out, def)
		phases := make([]string, 0, len(def.PhaseGraph.Phases))
		for phase := range def.PhaseGraph.Phases {
			phases = append(phases, string(phase))
		}
		sort.Strings(phases)
		for _, phase := range phases {
			phaseDef := def.PhaseGraph.Phases[Phase(phase)]
			if !phaseDef.Delegated() {
				continue
			}
			sub, ok := defs[phaseDef.ExecutedBy]
			if !ok {
				return fmt.Errorf("mode %q phase %q delegates to unknown mode %q", def.Mode, phaseDef.Phase, phaseDef.ExecutedBy)
			}
			if err := visit(sub); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	return out, nil
}

func compileModeInputContract(def Definition) (CompiledModeInputContract, error) {
	if !def.RunsModeRounds() {
		return CompiledModeInputContract{Mode: def.Mode, Target: def.Target.Kind}, nil
	}
	hasRegularPhase := false
	for _, phase := range def.PhaseGraph.Phases {
		if !phase.Delegated() {
			hasRegularPhase = true
			break
		}
	}
	if !hasRegularPhase {
		if len(def.InputContract.Specs) > 0 || len(def.InputContract.Sources) > 0 || len(def.InputContract.Aliases) > 0 {
			return CompiledModeInputContract{}, fmt.Errorf("a mode with only delegated phases must not declare local inputs")
		}
		return CompiledModeInputContract{Mode: def.Mode, Target: def.Target.Kind}, nil
	}
	contract := def.InputContract
	if len(contract.Specs) == 0 {
		return CompiledModeInputContract{}, fmt.Errorf("at least one input spec is required for a mode with regular phases")
	}

	specs := make(map[string]InputSpec, len(contract.Specs))
	for _, spec := range contract.Specs {
		if err := validateInputSpec(spec); err != nil {
			return CompiledModeInputContract{}, err
		}
		if _, exists := specs[spec.ID]; exists {
			return CompiledModeInputContract{}, fmt.Errorf("duplicate input spec %q", spec.ID)
		}
		specs[spec.ID] = spec
	}

	sources := make(map[string]InputSourceBinding, len(contract.Sources))
	for _, source := range contract.Sources {
		if _, ok := specs[source.InputID]; !ok {
			return CompiledModeInputContract{}, fmt.Errorf("source references unknown input %q", source.InputID)
		}
		if _, exists := sources[source.InputID]; exists {
			return CompiledModeInputContract{}, fmt.Errorf("input %q has competing sources", source.InputID)
		}
		if err := validateInputSource(def.Target.Kind, specs[source.InputID], source); err != nil {
			return CompiledModeInputContract{}, err
		}
		sources[source.InputID] = source
	}
	for id := range specs {
		if _, ok := sources[id]; !ok {
			return CompiledModeInputContract{}, fmt.Errorf("input %q has no source", id)
		}
	}
	if err := validateDerivedInputCycles(sources); err != nil {
		return CompiledModeInputContract{}, err
	}

	aliases := make(map[string]string, len(contract.Aliases))
	aliasesByInput := make(map[string][]string, len(contract.Aliases))
	for _, alias := range contract.Aliases {
		if !templateSlotRE.MatchString("{{" + alias.Name + "}}") {
			return CompiledModeInputContract{}, fmt.Errorf("input alias %q must be SCREAMING_SNAKE", alias.Name)
		}
		if _, ok := specs[alias.InputID]; !ok {
			return CompiledModeInputContract{}, fmt.Errorf("input alias %q references unknown input %q", alias.Name, alias.InputID)
		}
		if _, exists := aliases[alias.Name]; exists {
			return CompiledModeInputContract{}, fmt.Errorf("duplicate input alias %q", alias.Name)
		}
		aliases[alias.Name] = alias.InputID
		aliasesByInput[alias.InputID] = append(aliasesByInput[alias.InputID], alias.Name)
	}

	out := CompiledModeInputContract{Mode: def.Mode, Target: def.Target.Kind}
	usedAliases := map[string]bool{}
	phaseNames := make([]string, 0, len(def.PhaseGraph.Phases))
	for phase := range def.PhaseGraph.Phases {
		phaseNames = append(phaseNames, string(phase))
	}
	sort.Strings(phaseNames)
	for _, phaseName := range phaseNames {
		phaseDef := def.PhaseGraph.Phases[Phase(phaseName)]
		if phaseDef.Delegated() {
			continue
		}
		if len(phaseDef.Reads) == 0 {
			return CompiledModeInputContract{}, fmt.Errorf("phase %q declares no input aliases", phaseDef.Phase)
		}
		phase := CompiledPhaseInputContract{Phase: phaseDef.Phase}
		seen := map[string]bool{}
		for _, variable := range phaseDef.Reads {
			if seen[variable] {
				return CompiledModeInputContract{}, fmt.Errorf("phase %q declares duplicate input alias %q", phaseDef.Phase, variable)
			}
			seen[variable] = true
			inputID, ok := aliases[variable]
			if !ok {
				return CompiledModeInputContract{}, fmt.Errorf("phase %q references undeclared input alias %q", phaseDef.Phase, variable)
			}
			usedAliases[variable] = true
			phase.Bindings = append(phase.Bindings, CompiledPhaseInputBinding{Variable: variable, InputID: inputID})
		}
		out.Phases = append(out.Phases, phase)
	}
	for alias := range aliases {
		if !usedAliases[alias] {
			return CompiledModeInputContract{}, fmt.Errorf("input alias %q is declared but unused by every phase", alias)
		}
	}

	ids := make([]string, 0, len(specs))
	for id := range specs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if len(aliasesByInput[id]) == 0 {
			return CompiledModeInputContract{}, fmt.Errorf("input %q has no prompt alias or explicit non-template consumer", id)
		}
		sort.Strings(aliasesByInput[id])
		out.Inputs = append(out.Inputs, CompiledInput{Spec: specs[id], Source: sources[id], Aliases: aliasesByInput[id]})
	}
	return out, nil
}

func validateInputSpec(spec InputSpec) error {
	id := strings.TrimSpace(spec.ID)
	if id == "" || !strings.Contains(id, ".") || strings.HasPrefix(id, ".") || strings.HasSuffix(id, ".") {
		return fmt.Errorf("input spec id %q must be a dotted, namespaced identity", spec.ID)
	}
	for _, segment := range strings.Split(id, ".") {
		if segment == "" || !isInputIDSegment(segment) {
			return fmt.Errorf("input spec id %q has invalid segment %q", spec.ID, segment)
		}
	}
	switch spec.Type {
	case InputTypeString, InputTypeInteger, InputTypeNumber, InputTypeBoolean, InputTypeObject, InputTypeArray:
	default:
		return fmt.Errorf("input %q has unsupported type %q", spec.ID, spec.Type)
	}
	if strings.TrimSpace(spec.Description) == "" {
		return fmt.Errorf("input %q requires an operator-facing description", spec.ID)
	}
	switch spec.Sensitivity {
	case InputSensitivityPublic, InputSensitivityInternal, InputSensitivitySensitive:
	default:
		return fmt.Errorf("input %q has invalid sensitivity %q", spec.ID, spec.Sensitivity)
	}
	switch spec.Retention {
	case InputRetentionValue, InputRetentionDigest, InputRetentionOmit:
	default:
		return fmt.Errorf("input %q has invalid retention %q", spec.ID, spec.Retention)
	}
	if spec.Sensitivity == InputSensitivitySensitive && spec.Retention == InputRetentionValue {
		return fmt.Errorf("input %q is sensitive and must retain digest or omit, not value", spec.ID)
	}
	if spec.Minimum != nil || spec.Maximum != nil {
		if spec.Type != InputTypeInteger && spec.Type != InputTypeNumber {
			return fmt.Errorf("input %q declares numeric bounds for type %q", spec.ID, spec.Type)
		}
		if spec.Minimum != nil && spec.Maximum != nil && *spec.Minimum > *spec.Maximum {
			return fmt.Errorf("input %q minimum exceeds maximum", spec.ID)
		}
	}
	if spec.MinLength != nil || spec.MaxLength != nil {
		if spec.Type != InputTypeString {
			return fmt.Errorf("input %q declares string bounds for type %q", spec.ID, spec.Type)
		}
		if err := validateIntBounds(spec.ID, "length", spec.MinLength, spec.MaxLength); err != nil {
			return err
		}
	}
	if spec.MinItems != nil || spec.MaxItems != nil {
		if spec.Type != InputTypeArray {
			return fmt.Errorf("input %q declares collection bounds for type %q", spec.ID, spec.Type)
		}
		if err := validateIntBounds(spec.ID, "items", spec.MinItems, spec.MaxItems); err != nil {
			return err
		}
	}
	return nil
}

func validateInputSource(target TargetKind, spec InputSpec, source InputSourceBinding) error {
	switch source.Kind {
	case InputSourceGenericProvider, InputSourceTargetAdapter, InputSourceDerived:
		if strings.TrimSpace(source.Capability) == "" {
			return fmt.Errorf("input %q source %q requires a capability", spec.ID, source.Kind)
		}
		descriptor, ok := InputProviderCapabilities()[source.Capability]
		if !ok {
			return fmt.Errorf("input %q references unavailable capability %q", spec.ID, source.Capability)
		}
		if descriptor.SourceKind != source.Kind {
			return fmt.Errorf("input %q source kind %q conflicts with capability %q kind %q", spec.ID, source.Kind, source.Capability, descriptor.SourceKind)
		}
		if descriptor.Type != spec.Type {
			return fmt.Errorf("input %q type %q conflicts with capability %q type %q", spec.ID, spec.Type, source.Capability, descriptor.Type)
		}
		if len(descriptor.TargetKinds) > 0 && !containsTargetKind(descriptor.TargetKinds, target) {
			return fmt.Errorf("input %q capability %q does not support target %q", spec.ID, source.Capability, target)
		}
		if source.Kind == InputSourceDerived && len(source.DependsOn) == 0 {
			return fmt.Errorf("derived input %q requires depends_on", spec.ID)
		}
	case InputSourceCaller:
		if source.Capability != "" || len(source.DependsOn) > 0 || source.Default != nil {
			return fmt.Errorf("caller input %q must not declare capability, dependencies, or default", spec.ID)
		}
	case InputSourceDefault:
		if source.Capability != "" || len(source.DependsOn) > 0 {
			return fmt.Errorf("default input %q must not declare capability or dependencies", spec.ID)
		}
		if source.Default == nil {
			return fmt.Errorf("default input %q requires a non-null default", spec.ID)
		}
		if err := validateInputValue(spec, source.Default); err != nil {
			return fmt.Errorf("default input %q: %w", spec.ID, err)
		}
	default:
		return fmt.Errorf("input %q has unsupported source kind %q", spec.ID, source.Kind)
	}
	if source.Kind != InputSourceDerived && len(source.DependsOn) > 0 {
		return fmt.Errorf("input %q source %q must not declare depends_on", spec.ID, source.Kind)
	}
	return nil
}

func validateDerivedInputCycles(sources map[string]InputSourceBinding) error {
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("derived input cycle includes %q", id)
		case 2:
			return nil
		}
		state[id] = 1
		source := sources[id]
		for _, dependency := range source.DependsOn {
			if _, ok := sources[dependency]; !ok {
				return fmt.Errorf("derived input %q depends on unknown input %q", id, dependency)
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range sources {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func validateInputValue(spec InputSpec, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("value is not JSON: %w", err)
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return fmt.Errorf("normalize JSON value: %w", err)
	}
	switch spec.Type {
	case InputTypeString:
		value, ok := normalized.(string)
		if !ok {
			return fmt.Errorf("want string, got %T", normalized)
		}
		if spec.MinLength != nil && len(value) < *spec.MinLength {
			return fmt.Errorf("length %d below minimum %d", len(value), *spec.MinLength)
		}
		if spec.MaxLength != nil && len(value) > *spec.MaxLength {
			return fmt.Errorf("length %d exceeds maximum %d", len(value), *spec.MaxLength)
		}
	case InputTypeInteger:
		value, ok := normalized.(float64)
		if !ok || math.Trunc(value) != value {
			return fmt.Errorf("want integer, got %v", normalized)
		}
		if err := validateNumericValue(spec, value); err != nil {
			return err
		}
	case InputTypeNumber:
		value, ok := normalized.(float64)
		if !ok {
			return fmt.Errorf("want number, got %T", normalized)
		}
		if err := validateNumericValue(spec, value); err != nil {
			return err
		}
	case InputTypeBoolean:
		if _, ok := normalized.(bool); !ok {
			return fmt.Errorf("want boolean, got %T", normalized)
		}
	case InputTypeObject:
		if _, ok := normalized.(map[string]any); !ok {
			return fmt.Errorf("want object, got %T", normalized)
		}
	case InputTypeArray:
		value, ok := normalized.([]any)
		if !ok {
			return fmt.Errorf("want array, got %T", normalized)
		}
		if spec.MinItems != nil && len(value) < *spec.MinItems {
			return fmt.Errorf("item count %d below minimum %d", len(value), *spec.MinItems)
		}
		if spec.MaxItems != nil && len(value) > *spec.MaxItems {
			return fmt.Errorf("item count %d exceeds maximum %d", len(value), *spec.MaxItems)
		}
	default:
		return fmt.Errorf("unsupported type %q", spec.Type)
	}
	return nil
}

func validateNumericValue(spec InputSpec, value float64) error {
	if spec.Minimum != nil && value < *spec.Minimum {
		return fmt.Errorf("value %v below minimum %v", value, *spec.Minimum)
	}
	if spec.Maximum != nil && value > *spec.Maximum {
		return fmt.Errorf("value %v exceeds maximum %v", value, *spec.Maximum)
	}
	return nil
}

func validateIntBounds(id, label string, minimum, maximum *int) error {
	if minimum != nil && *minimum < 0 || maximum != nil && *maximum < 0 {
		return fmt.Errorf("input %q %s bounds must be non-negative", id, label)
	}
	if minimum != nil && maximum != nil && *minimum > *maximum {
		return fmt.Errorf("input %q minimum %s exceeds maximum", id, label)
	}
	return nil
}

func containsTargetKind(kinds []TargetKind, target TargetKind) bool {
	for _, kind := range kinds {
		if kind == target {
			return true
		}
	}
	return false
}

func isInputIDSegment(segment string) bool {
	for i, r := range segment {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' && i > 0 || r == '_' && i > 0 || r == '-' && i > 0 {
			continue
		}
		return false
	}
	return true
}

func (rc RunContext) compiledInputContract() (CompiledInputContract, error) {
	if rc.Execution != nil && len(rc.Execution.CompiledInputContract) > 0 {
		var compiled CompiledInputContract
		if err := json.Unmarshal(rc.Execution.CompiledInputContract, &compiled); err != nil {
			return CompiledInputContract{}, fmt.Errorf("decode execution input contract: %w", err)
		}
		return compiled, nil
	}
	bundle, _, err := pinDefinitionBundle(rc.Def, DefinitionFor)
	if err != nil {
		return CompiledInputContract{}, err
	}
	root, err := bundle.RootDefinition()
	if err != nil {
		return CompiledInputContract{}, err
	}
	return CompileInputContract(bundle.Definitions, root)
}

func (rc RunContext) compiledModeAndPhaseInputs() (CompiledModeInputContract, CompiledPhaseInputContract, error) {
	compiled, err := rc.compiledInputContract()
	if err != nil {
		return CompiledModeInputContract{}, CompiledPhaseInputContract{}, err
	}
	for _, mode := range compiled.Modes {
		if mode.Mode != rc.Def.Mode {
			continue
		}
		for _, phase := range mode.Phases {
			if phase.Phase == rc.PhaseDef.Phase {
				return mode, phase, nil
			}
		}
		return CompiledModeInputContract{}, CompiledPhaseInputContract{}, fmt.Errorf("compiled input contract for mode %q has no phase %q", rc.Def.Mode, rc.PhaseDef.Phase)
	}
	return CompiledModeInputContract{}, CompiledPhaseInputContract{}, fmt.Errorf("compiled input contract has no mode %q", rc.Def.Mode)
}

func (rc RunContext) resolveCompiledInputs(round RoundEnvelope, note string, phaseOnly bool) (map[string]string, error) {
	mode, phase, err := rc.compiledModeAndPhaseInputs()
	if err != nil {
		return nil, err
	}
	inputByID := make(map[string]CompiledInput, len(mode.Inputs))
	for _, input := range mode.Inputs {
		inputByID[input.Spec.ID] = input
	}
	bindings := phase.Bindings
	if !phaseOnly {
		bindings = nil
		for _, input := range mode.Inputs {
			for _, alias := range input.Aliases {
				bindings = append(bindings, CompiledPhaseInputBinding{Variable: alias, InputID: input.Spec.ID})
			}
		}
		sort.Slice(bindings, func(i, j int) bool { return bindings[i].Variable < bindings[j].Variable })
	}
	resolved := map[string]any{}
	resolving := map[string]bool{}
	var resolve func(string) (any, error)
	resolve = func(id string) (any, error) {
		if value, ok := resolved[id]; ok {
			return value, nil
		}
		if resolving[id] {
			return nil, fmt.Errorf("runtime derived input cycle includes %q", id)
		}
		input, ok := inputByID[id]
		if !ok {
			return nil, fmt.Errorf("compiled phase references unknown input %q", id)
		}
		resolving[id] = true
		var value any
		switch input.Source.Kind {
		case InputSourceGenericProvider, InputSourceTargetAdapter:
			value, err = rc.resolveProviderCapability(input.Source.Capability, round, note)
		case InputSourceCaller:
			value, err = rc.executionInputSnapshotValue(id, input.Spec.Required)
		case InputSourceDefault:
			value = input.Source.Default
		case InputSourceDerived:
			dependencies := make([]any, 0, len(input.Source.DependsOn))
			for _, dependency := range input.Source.DependsOn {
				dependencyValue, dependencyErr := resolve(dependency)
				if dependencyErr != nil {
					err = dependencyErr
					break
				}
				dependencies = append(dependencies, dependencyValue)
			}
			if err == nil {
				value, err = resolveDerivedCapability(input.Source.Capability, dependencies)
			}
		default:
			err = fmt.Errorf("unsupported compiled source kind %q", input.Source.Kind)
		}
		delete(resolving, id)
		if err != nil {
			return nil, fmt.Errorf("resolve input %q: %w", id, err)
		}
		if value == nil {
			if input.Spec.Required {
				return nil, fmt.Errorf("required input %q resolved null", id)
			}
			resolved[id] = nil
			return nil, nil
		}
		if err := validateInputValue(input.Spec, value); err != nil {
			return nil, fmt.Errorf("input %q violates compiled contract: %w", id, err)
		}
		resolved[id] = value
		return value, nil
	}

	out := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		value, err := resolve(binding.InputID)
		if err != nil {
			return nil, err
		}
		out[binding.Variable] = renderInputValue(value)
	}
	return out, nil
}

func (rc RunContext) resolveProviderCapability(capability string, round RoundEnvelope, note string) (any, error) {
	switch capability {
	case "generic.operating_mode":
		return string(rc.Def.Mode), nil
	case "generic.mode_label":
		return rc.Def.Label, nil
	case "generic.phase":
		return string(rc.PhaseDef.Phase), nil
	case "generic.run_strategy":
		return string(rc.Def.RunStrategy.Kind), nil
	case "generic.round_number":
		return round.Round, nil
	case "generic.agent_profile_key":
		return rc.PhaseDef.ProfileKey, nil
	case "generic.operator_note":
		return strings.TrimSpace(note), nil
	case "generic.user_question":
		return rc.operatorInput("USER_QUESTION"), nil
	case "generic.decision_topic":
		return rc.operatorInput("DECISION_TOPIC"), nil
	case "generic.user_message":
		return rc.operatorInput("USER_MESSAGE"), nil
	case "generic.user_prompt":
		return rc.operatorInput("USER_PROMPT"), nil
	case "generic.context_paths":
		return rc.operatorInput("CONTEXT_PATHS"), nil
	case "generic.context_targets":
		return rc.operatorInput("CONTEXT_TARGETS"), nil
	case "generic.context_requirements":
		return rc.operatorInput("CONTEXT_REQUIREMENTS"), nil
	case "generic.gap_report":
		return rc.operatorInput("GAP_REPORT"), nil
	case "generic.evidence_request":
		return rc.operatorInput("EVIDENCE_REQUEST"), nil
	case "generic.prior_rounds":
		return append([]RoundEnvelope{}, rc.Rounds...), nil
	case "generic.mode_artifacts":
		return append([]ArtifactSnapshot{}, rc.Artifacts...), nil
	case "generic.backlog_sync_proposal":
		return promptcatalog.BacklogSyncProposalSnippet(), nil
	case "generic.elastic_slice":
		return promptcatalog.ElasticSliceSnippet(), nil
	}
	adapter, err := rc.Adapter()
	if err != nil {
		return nil, err
	}
	value, ok := adapter.Values(rc.Target)[capability]
	if !ok {
		return nil, fmt.Errorf("provider capability %q has no runtime implementation for target %q", capability, rc.Target.Kind)
	}
	if value == nil {
		return nil, nil
	}
	return value, nil
}

func (rc RunContext) executionInputSnapshotValue(id string, required bool) (any, error) {
	if rc.Execution == nil || len(rc.Execution.ValidatedInputSnapshot) == 0 {
		if !required {
			return nil, nil
		}
		return nil, fmt.Errorf("caller input snapshot is unavailable")
	}
	var snapshot map[string]any
	if err := json.Unmarshal(rc.Execution.ValidatedInputSnapshot, &snapshot); err != nil {
		return nil, fmt.Errorf("decode caller input snapshot: %w", err)
	}
	value, ok := snapshot[id]
	if !ok {
		if !required {
			return nil, nil
		}
		return nil, fmt.Errorf("caller input is absent from the validated snapshot")
	}
	return value, nil
}

func resolveDerivedCapability(capability string, dependencies []any) (any, error) {
	switch capability {
	case "derived.sha256":
		data, err := json.Marshal(dependencies)
		if err != nil {
			return nil, fmt.Errorf("marshal derived digest inputs: %w", err)
		}
		digest := sha256.Sum256(data)
		return fmt.Sprintf("sha256:%x", digest[:]), nil
	default:
		return nil, fmt.Errorf("derived capability %q has no runtime implementation", capability)
	}
}

func renderInputValue(value any) string {
	if value == nil {
		return "null"
	}
	if text, ok := value.(string); ok {
		return text
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "null"
	}
	return string(data)
}
