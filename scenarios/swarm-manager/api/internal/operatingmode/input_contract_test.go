package operatingmode

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func callerInputContractDefinition() Definition {
	def := minimalInputContractDefinition()
	def.InputContract.Specs = []InputSpec{
		{
			ID: "caller.payload", Type: InputTypeObject, Required: true,
			Sensitivity: InputSensitivityInternal, Retention: InputRetentionValue,
			Description: "Caller-supplied execution payload.",
		},
		{
			ID: "caller.limit", Type: InputTypeInteger, Minimum: float64Ptr(1), Maximum: float64Ptr(10),
			Sensitivity: InputSensitivityPublic, Retention: InputRetentionValue,
			Description: "Optional caller-supplied limit.",
		},
	}
	def.InputContract.Sources = []InputSourceBinding{
		{InputID: "caller.payload", Kind: InputSourceCaller},
		{InputID: "caller.limit", Kind: InputSourceCaller},
	}
	def.InputContract.Aliases = []InputAlias{
		{Name: "CALLER_PAYLOAD_JSON", InputID: "caller.payload"},
		{Name: "CALLER_LIMIT", InputID: "caller.limit"},
	}
	phase := def.PhaseGraph.Phases["run"]
	phase.Reads = []string{"CALLER_PAYLOAD_JSON", "CALLER_LIMIT"}
	def.PhaseGraph.Phases["run"] = phase
	return def
}

func float64Ptr(value float64) *float64 { return &value }

func minimalInputContractDefinition() Definition {
	return Definition{
		Mode:        "input-contract-test",
		Target:      TargetPolicy{Kind: TargetPlanExecution},
		RunStrategy: RunStrategyPolicy{Kind: RunStrategySinglePhaseRun},
		InputContract: InputContractDefinition{
			Specs: []InputSpec{
				{ID: "execution.note", Type: InputTypeString, Required: true, Sensitivity: InputSensitivityInternal, Retention: InputRetentionValue, Description: "Current execution note."},
			},
			Sources: []InputSourceBinding{
				{InputID: "execution.note", Kind: InputSourceGenericProvider, Capability: "generic.operator_note"},
			},
			Aliases: []InputAlias{{Name: "OPERATOR_NOTE", InputID: "execution.note"}},
		},
		PhaseGraph: PhaseGraph{
			StartPhase: "run",
			Phases: map[Phase]PhaseDefinition{
				"run": {Phase: "run", Reads: []string{"OPERATOR_NOTE"}},
			},
		},
	}
}

func compileInputTestDefinition(def Definition) error {
	_, err := CompileInputContract(map[Mode]Definition{def.Mode: def}, def)
	return err
}

// [REQ:REQ-P0-011-INPUT-CONTRACT]
func TestCompileInputContractShippedGraphIsDeterministic(t *testing.T) {
	defs := cloneRegistryForTest()
	root := defs[ModeHolisticLoop]
	first, err := CompileInputContract(defs, root)
	if err != nil {
		t.Fatalf("CompileInputContract(first): %v", err)
	}
	second, err := CompileInputContract(defs, root)
	if err != nil {
		t.Fatalf("CompileInputContract(second): %v", err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("compiled contract is not deterministic\nfirst:  %s\nsecond: %s", firstJSON, secondJSON)
	}
	if len(first.Modes) != 2 || first.Modes[0].Mode != ModeHolisticLoop || first.Modes[1].Mode != ModePhasedPlanDrain {
		t.Fatalf("compiled reachable modes = %+v, want holistic-loop then phased-plan-drain", first.Modes)
	}
}

func TestCompileInputContractRejectsInvalidContracts(t *testing.T) {
	tests := []struct {
		name string
		want string
		edit func(*Definition)
	}{
		{
			name: "unnamespaced id", want: "namespaced identity",
			edit: func(def *Definition) { def.InputContract.Specs[0].ID = "note" },
		},
		{
			name: "duplicate spec", want: "duplicate input spec",
			edit: func(def *Definition) {
				def.InputContract.Specs = append(def.InputContract.Specs, def.InputContract.Specs[0])
			},
		},
		{
			name: "competing sources", want: "competing sources",
			edit: func(def *Definition) {
				def.InputContract.Sources = append(def.InputContract.Sources, def.InputContract.Sources[0])
			},
		},
		{
			name: "missing source", want: "has no source",
			edit: func(def *Definition) { def.InputContract.Sources = nil },
		},
		{
			name: "unavailable capability", want: "unavailable capability",
			edit: func(def *Definition) { def.InputContract.Sources[0].Capability = "generic.missing" },
		},
		{
			name: "capability type mismatch", want: "conflicts with capability",
			edit: func(def *Definition) { def.InputContract.Specs[0].Type = InputTypeArray },
		},
		{
			name: "target mismatch", want: "does not support target",
			edit: func(def *Definition) {
				def.InputContract.Sources[0] = InputSourceBinding{InputID: "execution.note", Kind: InputSourceTargetAdapter, Capability: "target.initiative_name"}
			},
		},
		{
			name: "sensitive value retention", want: "must retain digest or omit",
			edit: func(def *Definition) { def.InputContract.Specs[0].Sensitivity = InputSensitivitySensitive },
		},
		{
			name: "unknown phase alias", want: "undeclared input alias",
			edit: func(def *Definition) {
				phase := def.PhaseGraph.Phases["run"]
				phase.Reads = []string{"UNKNOWN_INPUT"}
				def.PhaseGraph.Phases["run"] = phase
			},
		},
		{
			name: "unused alias", want: "declared but unused",
			edit: func(def *Definition) {
				def.InputContract.Aliases = append(def.InputContract.Aliases, InputAlias{Name: "OTHER_NOTE", InputID: "execution.note"})
			},
		},
		{
			name: "wrong default type", want: "want string",
			edit: func(def *Definition) {
				def.InputContract.Sources[0] = InputSourceBinding{InputID: "execution.note", Kind: InputSourceDefault, Default: true}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := minimalInputContractDefinition()
			tt.edit(&def)
			err := compileInputTestDefinition(def)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("CompileInputContract error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestCompileInputContractAcceptsCallerDefaultAndRejectsDerivedCycle(t *testing.T) {
	def := minimalInputContractDefinition()
	def.InputContract.Specs = append(def.InputContract.Specs,
		InputSpec{ID: "caller.payload", Type: InputTypeObject, Required: true, Sensitivity: InputSensitivitySensitive, Retention: InputRetentionDigest, Description: "Caller-supplied structured payload."},
		InputSpec{ID: "execution.label", Type: InputTypeString, Sensitivity: InputSensitivityPublic, Retention: InputRetentionValue, Description: "Default display label."},
	)
	def.InputContract.Sources = append(def.InputContract.Sources,
		InputSourceBinding{InputID: "caller.payload", Kind: InputSourceCaller},
		InputSourceBinding{InputID: "execution.label", Kind: InputSourceDefault, Default: "default label"},
	)
	def.InputContract.Aliases = append(def.InputContract.Aliases,
		InputAlias{Name: "CALLER_PAYLOAD_JSON", InputID: "caller.payload"},
		InputAlias{Name: "DEFAULT_LABEL", InputID: "execution.label"},
	)
	phase := def.PhaseGraph.Phases["run"]
	phase.Reads = append(phase.Reads, "CALLER_PAYLOAD_JSON", "DEFAULT_LABEL")
	def.PhaseGraph.Phases["run"] = phase
	if err := compileInputTestDefinition(def); err != nil {
		t.Fatalf("caller/default contract rejected: %v", err)
	}

	def = minimalInputContractDefinition()
	def.InputContract.Specs = []InputSpec{
		{ID: "derived.first", Type: InputTypeString, Sensitivity: InputSensitivityInternal, Retention: InputRetentionDigest, Description: "First derived digest."},
		{ID: "derived.second", Type: InputTypeString, Sensitivity: InputSensitivityInternal, Retention: InputRetentionDigest, Description: "Second derived digest."},
	}
	def.InputContract.Sources = []InputSourceBinding{
		{InputID: "derived.first", Kind: InputSourceDerived, Capability: "derived.sha256", DependsOn: []string{"derived.second"}},
		{InputID: "derived.second", Kind: InputSourceDerived, Capability: "derived.sha256", DependsOn: []string{"derived.first"}},
	}
	def.InputContract.Aliases = []InputAlias{
		{Name: "FIRST_DIGEST", InputID: "derived.first"},
		{Name: "SECOND_DIGEST", InputID: "derived.second"},
	}
	phase = def.PhaseGraph.Phases["run"]
	phase.Reads = []string{"FIRST_DIGEST", "SECOND_DIGEST"}
	def.PhaseGraph.Phases["run"] = phase
	if err := compileInputTestDefinition(def); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("derived cycle error = %v, want cycle rejection", err)
	}
}

func TestValidateCallerInputSnapshotNormalizesAndHashesDeterministically(t *testing.T) {
	def := callerInputContractDefinition()
	compiled, err := CompileInputContract(map[Mode]Definition{def.Mode: def}, def)
	if err != nil {
		t.Fatalf("CompileInputContract: %v", err)
	}
	first, firstDigest, retention, err := ValidateCallerInputSnapshot(compiled, map[string]any{
		"caller.limit":   7,
		"caller.payload": map[string]any{"enabled": true, "labels": []string{"one", "two"}},
	})
	if err != nil {
		t.Fatalf("ValidateCallerInputSnapshot: %v", err)
	}
	second, secondDigest, _, err := ValidateCallerInputSnapshot(compiled, map[string]any{
		"caller.payload": map[string]any{"labels": []any{"one", "two"}, "enabled": true},
		"caller.limit":   float64(7),
	})
	if err != nil {
		t.Fatalf("ValidateCallerInputSnapshot replay: %v", err)
	}
	if string(first) != string(second) || firstDigest != secondDigest || !strings.HasPrefix(firstDigest, "sha256:") {
		t.Fatalf("snapshot replay mismatch\nfirst=%s %s\nsecond=%s %s", first, firstDigest, second, secondDigest)
	}
	limitMeta, ok := retention["caller.limit"].(map[string]any)
	if !ok || limitMeta["present"] != true || limitMeta["retention"] != InputRetentionValue {
		t.Fatalf("retention metadata = %#v", retention)
	}
}

// [REQ:REQ-P0-011-INPUT-CONTRACT]
func TestValidateCallerInputSnapshotRejectsInvalidValues(t *testing.T) {
	def := callerInputContractDefinition()
	compiled, err := CompileInputContract(map[Mode]Definition{def.Mode: def}, def)
	if err != nil {
		t.Fatalf("CompileInputContract: %v", err)
	}
	tests := []struct {
		name     string
		supplied map[string]any
		want     string
	}{
		{name: "missing required", supplied: map[string]any{}, want: "required caller input"},
		{name: "unknown", supplied: map[string]any{"caller.payload": map[string]any{}, "caller.extra": true}, want: "unknown caller inputs"},
		{name: "mistyped", supplied: map[string]any{"caller.payload": "not-an-object"}, want: "want object"},
		{name: "non integer", supplied: map[string]any{"caller.payload": map[string]any{}, "caller.limit": 1.5}, want: "want integer"},
		{name: "below bound", supplied: map[string]any{"caller.payload": map[string]any{}, "caller.limit": 0}, want: "below minimum"},
		{name: "above bound", supplied: map[string]any{"caller.payload": map[string]any{}, "caller.limit": 11}, want: "exceeds maximum"},
		{name: "oversized", supplied: map[string]any{"caller.payload": map[string]any{"value": strings.Repeat("x", maxCallerInputBytes)}}, want: "encoded size"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ValidateCallerInputSnapshot(compiled, tt.supplied)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}

	for _, retention := range []InputRetention{InputRetentionDigest, InputRetentionOmit} {
		t.Run(fmt.Sprintf("unreplayable retention %s", retention), func(t *testing.T) {
			mutated := callerInputContractDefinition()
			mutated.InputContract.Specs[0].Retention = retention
			compiled, err := CompileInputContract(map[Mode]Definition{mutated.Mode: mutated}, mutated)
			if err != nil {
				t.Fatalf("CompileInputContract: %v", err)
			}
			_, _, _, err = ValidateCallerInputSnapshot(compiled, map[string]any{"caller.payload": map[string]any{}})
			if err == nil || !strings.Contains(err.Error(), "replayable caller inputs require value retention") {
				t.Fatalf("error = %v", err)
			}
		})
	}

	mutated := callerInputContractDefinition()
	mutated.InputContract.Specs[0].Sensitivity = InputSensitivitySensitive
	mutated.InputContract.Specs[0].Retention = InputRetentionDigest
	compiled, err = CompileInputContract(map[Mode]Definition{mutated.Mode: mutated}, mutated)
	if err != nil {
		t.Fatalf("CompileInputContract sensitive: %v", err)
	}
	_, _, _, err = ValidateCallerInputSnapshot(compiled, map[string]any{"caller.payload": map[string]any{}})
	if err == nil || !strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("sensitive caller error = %v", err)
	}
}

func TestOptionalCallerInputRendersNullWhenAbsentFromSnapshot(t *testing.T) {
	def := callerInputContractDefinition()
	def.InputContract.Specs[0].Required = false
	compiled, err := CompileInputContract(map[Mode]Definition{def.Mode: def}, def)
	if err != nil {
		t.Fatalf("CompileInputContract: %v", err)
	}
	snapshot, digest, retention, err := ValidateCallerInputSnapshot(compiled, nil)
	if err != nil {
		t.Fatalf("ValidateCallerInputSnapshot: %v", err)
	}
	compiledJSON, _ := json.Marshal(compiled)
	rc := RunContext{
		Def: def, PhaseDef: def.PhaseGraph.Phases["run"],
		Execution: &OperatingModeExecution{
			CompiledInputContract: compiledJSON, ValidatedInputSnapshot: snapshot,
			InputSnapshotDigest: digest, InputRetentionMetadata: retention,
		},
	}
	reads, err := rc.DeclaredReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("DeclaredReads: %v", err)
	}
	if reads["CALLER_PAYLOAD_JSON"] != "null" || reads["CALLER_LIMIT"] != "null" {
		t.Fatalf("optional caller reads = %#v, want null values", reads)
	}
}

// TestGenericCallerContextProviderRendersOperatorInput proves the structured
// caller-context provider seam end to end at the render layer: a mode that binds
// generic.user_prompt to a phase read renders the per-run OperatorInputs value the
// operation runner routes in, WITHOUT the mode declaring a caller-source input. An
// absent operator input degrades to an empty read rather than erroring, matching
// the optional-input contract every structured caller-context provider honors.
func TestGenericCallerContextProviderRendersOperatorInput(t *testing.T) {
	def := minimalInputContractDefinition()
	def.InputContract.Specs = []InputSpec{
		{ID: "context.user_prompt", Type: InputTypeString, Sensitivity: InputSensitivityInternal, Retention: InputRetentionValue, Description: "Operator research prompt."},
	}
	def.InputContract.Sources = []InputSourceBinding{
		{InputID: "context.user_prompt", Kind: InputSourceGenericProvider, Capability: "generic.user_prompt"},
	}
	def.InputContract.Aliases = []InputAlias{{Name: "USER_PROMPT", InputID: "context.user_prompt"}}
	phase := def.PhaseGraph.Phases["run"]
	phase.Reads = []string{"USER_PROMPT"}
	def.PhaseGraph.Phases["run"] = phase

	compiled, err := CompileInputContract(map[Mode]Definition{def.Mode: def}, def)
	if err != nil {
		t.Fatalf("CompileInputContract: %v", err)
	}
	compiledJSON, _ := json.Marshal(compiled)
	rc := RunContext{
		Def: def, PhaseDef: def.PhaseGraph.Phases["run"],
		Execution:      &OperatingModeExecution{CompiledInputContract: compiledJSON},
		OperatorInputs: map[string]string{"USER_PROMPT": "focus on the auth boundary"},
	}
	reads, err := rc.DeclaredReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("DeclaredReads: %v", err)
	}
	if reads["USER_PROMPT"] != "focus on the auth boundary" {
		t.Fatalf("USER_PROMPT read = %q, want the routed operator input", reads["USER_PROMPT"])
	}

	rc.OperatorInputs = nil
	reads, err = rc.DeclaredReads(RoundEnvelope{Round: 1}, "")
	if err != nil {
		t.Fatalf("DeclaredReads with no operator input: %v", err)
	}
	if reads["USER_PROMPT"] != "" {
		t.Fatalf("absent operator input should degrade to empty, got %q", reads["USER_PROMPT"])
	}
}

// [REQ:REQ-P0-011-INPUT-CONTRACT]
func TestDeclaredReadsUsesPinnedExecutionContractAfterLiveSourceMutation(t *testing.T) {
	store := testStore(t)
	store.ExecutionID = func() string { return "execution-input-pinned" }
	live, err := clonePinnedDefinition(MustDefinition(ModePhasedPlanDrain))
	if err != nil {
		t.Fatalf("clone live definition: %v", err)
	}
	execution, err := store.CreateExecution("plan-123", live)
	if err != nil {
		t.Fatalf("CreateExecution: %v", err)
	}
	for i := range live.InputContract.Sources {
		if live.InputContract.Sources[i].InputID == "execution.operator_note" {
			live.InputContract.Sources[i].Capability = "generic.mutated-after-start"
		}
	}
	rc := RunContext{
		Def: live, PhaseDef: live.PhaseGraph.Phases["execute"], Execution: &execution,
		Target: TargetInstance{
			Kind: TargetPlanExecution, ID: "plan-123",
			Plan: &PlanExecutionContext{ExecutionID: "plan-123"},
		},
	}
	reads, err := rc.DeclaredReads(RoundEnvelope{Round: 1}, "pinned note")
	if err != nil {
		t.Fatalf("DeclaredReads through pinned contract: %v", err)
	}
	if reads[ReadOperatorNote] != "pinned note" {
		t.Fatalf("operator note = %q, want pinned note", reads[ReadOperatorNote])
	}
	if _, err := store.CreateExecution("plan-new", live); err == nil || !strings.Contains(err.Error(), "unavailable capability") {
		t.Fatalf("new execution error = %v, want mutated capability rejection", err)
	}
}
