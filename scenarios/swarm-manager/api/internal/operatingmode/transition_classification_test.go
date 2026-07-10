package operatingmode

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
)

const modeSyntheticClassify Mode = "synthetic-classify"

// syntheticClassifyDoc builds the mode-data document for a test-only
// initiative mode whose execute phase routes through a classified transition
// (classification-on-transition): the routing field `outcome` is derived from
// the handoff at the edge, with no dedicated classifier phase in the graph.
// The transitions block is parameterized so the loader-rejection tests can
// exercise malformed declarations through the real loader.
func syntheticClassifyDoc(transitionsJSON string) string {
	const reads = `["OPERATING_MODE", "PHASE", "ROUND_NUMBER", "OPERATOR_NOTE", "PRIOR_ROUNDS_JSON", "INITIATIVE_NAME", "MEMBER_ITEMS_JSON"]`
	return `{
	  "kind": "operating-mode",
	  "id": "synthetic-classify",
	  "label": "Synthetic Classify",
	  "description": "Synthetic classification-on-transition harness; not a production mode.",
	  "best_for": ["Exercising transition classification"],
	  "not_for": ["Production work"],
	  "tradeoffs": ["Test-only"],
	  "when_in_doubt_pick_instead": "item-level",
	  "target": { "kind": "initiative" },
	  "input_contract": ` + testInputContractJSON(string(TargetInitiative), []string{ReadOperatingMode, ReadPhase, ReadRoundNumber, ReadOperatorNote, ReadPriorRoundsJSON, ReadInitiativeName, ReadMemberItemsJSON}) + `,
	  "run_strategy": { "kind": "operator_gated_loop" },
	  "prompt": { "catalog_prefix": "swarm-manager-synthetic-classify" },
	  "artifact": { "root": "modes/synthetic-classify" },
	  "profile": { "default_profile_key": "swarm-manager/deep-work" },
	  "backlog_sync": {
	    "capabilities": ["read_only"],
	    "apply_mode": "operator-gated"
	  },
	  "lock": { "initiative_exclusive": true },
	  "ui": { "workspace_tab_id": "operating-mode" },
	  "phase_graph": {
	    "start_phase": "work",
	    "terminal": ["wrap"],
	    "phases": [
	      {
	        "id": "work",
	        "kind": "execute",
	        "activity_purpose": "synthetic_classify_work",
	        "reads": ` + reads + `,
	        "writes_repo": true,
	        "prompt": { "purpose": "Drain the next slice and emit a handoff." },
	        "declared_output": {
	          "fields": [
	            { "name": "handoff", "type": "object", "required": true, "description": "The execution handoff." }
	          ]
	        },
	        "transitions": ` + transitionsJSON + `
	      },
	      {
	        "id": "wrap",
	        "kind": "review",
	        "activity_purpose": "synthetic_classify_wrap",
	        "reads": ` + reads + `,
	        "prompt": { "purpose": "Wrap up the synthetic classification loop." }
	      }
	    ]
	  }
	}`
}

const syntheticClassifyTransitions = `[
	{
	  "classify": {
	    "field": "outcome",
	    "enum": ["continue", "complete", "blocked"],
	    "from": "handoff",
	    "description": "Whether the drain should continue, has completed, or is blocked."
	  },
	  "routes": {
	    "continue": ["work"],
	    "complete": ["wrap"],
	    "blocked": []
	  }
	}
]`

func loadSyntheticClassifyDefinition(t *testing.T) Definition {
	t.Helper()
	def, err := LoadModeDefinition([]byte(syntheticClassifyDoc(syntheticClassifyTransitions)))
	if err != nil {
		t.Fatalf("load synthetic classify mode: %v", err)
	}
	return def
}

func TestClassifiedTransitionLoaderExpansion(t *testing.T) {
	def := loadSyntheticClassifyDefinition(t)

	work, err := def.PhaseDefinition("work")
	if err != nil {
		t.Fatalf("work phase: %v", err)
	}
	contract := work.TransitionClassification
	if contract == nil {
		t.Fatalf("work phase has no transition classification contract")
	}
	if contract.Field != "outcome" || contract.From != "handoff" {
		t.Fatalf("contract = %+v, want field=outcome from=handoff", contract)
	}
	if len(contract.Enum) != 3 || contract.Enum[0] != "continue" || contract.Enum[2] != "blocked" {
		t.Fatalf("contract enum = %v, want declared order [continue complete blocked]", contract.Enum)
	}

	guards := def.PhaseGraph.Guards["work"]
	if len(guards) != 3 {
		t.Fatalf("expanded guards = %d, want one eq-guard per enum value", len(guards))
	}
	for i, want := range []struct {
		value string
		to    []Phase
	}{
		{"continue", []Phase{"work"}},
		{"complete", []Phase{"wrap"}},
		{"blocked", nil},
	} {
		g := guards[i]
		if g.When.Op != GuardOpEq || g.When.Field != "outcome" || g.When.Value != want.value {
			t.Fatalf("guard[%d] = %+v, want eq outcome=%s", i, g.When, want.value)
		}
		if len(g.To) != len(want.to) {
			t.Fatalf("guard[%d].To = %v, want %v", i, g.To, want.to)
		}
		for j := range want.to {
			if g.To[j] != want.to[j] {
				t.Fatalf("guard[%d].To = %v, want %v", i, g.To, want.to)
			}
		}
	}
	adjacency := def.PhaseGraph.Transitions["work"]
	if len(adjacency) != 2 || adjacency[0] != "work" || adjacency[1] != "wrap" {
		t.Fatalf("derived adjacency = %v, want [work wrap]", adjacency)
	}
	if err := validateGuardGraph(map[Mode]Definition{def.Mode: def}, def); err != nil {
		t.Fatalf("validateGuardGraph rejected a valid classified transition: %v", err)
	}
}

func TestClassifiedTransitionLoaderRejections(t *testing.T) {
	cases := []struct {
		name        string
		transitions string
		wantErr     string
	}{
		{
			name: "routes missing an enum value",
			transitions: `[{ "classify": { "field": "outcome", "enum": ["continue", "complete"], "from": "handoff" },
				"routes": { "continue": ["work"] } }]`,
			wantErr: `missing enum value "complete"`,
		},
		{
			name: "routes key outside the enum",
			transitions: `[{ "classify": { "field": "outcome", "enum": ["continue"], "from": "handoff" },
				"routes": { "continue": ["work"], "bogus": ["wrap"] } }]`,
			wantErr: `routes declares "bogus"`,
		},
		{
			name: "duplicate enum value rejected by schema",
			transitions: `[{ "classify": { "field": "outcome", "enum": ["continue", "continue"], "from": "handoff" },
				"routes": { "continue": ["work"] } }]`,
			wantErr: "schema validation failed",
		},
		{
			name: "two classified transitions on one phase",
			transitions: `[
				{ "classify": { "field": "outcome", "enum": ["continue"], "from": "handoff" }, "routes": { "continue": ["work"] } },
				{ "classify": { "field": "other", "enum": ["done"], "from": "handoff" }, "routes": { "done": ["wrap"] } }
			]`,
			wantErr: "more than one classified transition",
		},
		{
			name: "invalid field path rejected by schema",
			transitions: `[{ "classify": { "field": "Outcome", "enum": ["continue"], "from": "handoff" },
				"routes": { "continue": ["work"] } }]`,
			wantErr: "schema validation failed",
		},
		{
			name:        "empty enum rejected by schema",
			transitions: `[{ "classify": { "field": "outcome", "enum": [], "from": "handoff" }, "routes": { "continue": ["work"] } }]`,
			wantErr:     "schema validation failed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadModeDefinition([]byte(syntheticClassifyDoc(tc.transitions)))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("LoadModeDefinition error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}

	// The loader enforces the same structural rules in Go (belt and braces for
	// documents that bypass JSON-Schema validation).
	t.Run("expansion rejects duplicate enum value", func(t *testing.T) {
		_, _, err := expandClassifiedTransition(classificationContractDoc{Field: "outcome", Enum: []string{"continue", "continue"}}, map[string][]string{"continue": {"work"}})
		if err == nil || !strings.Contains(err.Error(), `duplicate value "continue"`) {
			t.Fatalf("expand = %v, want duplicate-enum rejection", err)
		}
	})
	t.Run("expansion rejects invalid field path", func(t *testing.T) {
		_, _, err := expandClassifiedTransition(classificationContractDoc{Field: "Outcome", Enum: []string{"continue"}}, map[string][]string{"continue": {"work"}})
		if err == nil || !strings.Contains(err.Error(), "must be a dotted lowercase field-path") {
			t.Fatalf("expand = %v, want field-path rejection", err)
		}
	})
	t.Run("expansion rejects empty enum", func(t *testing.T) {
		_, _, err := expandClassifiedTransition(classificationContractDoc{Field: "outcome"}, map[string][]string{"continue": {"work"}})
		if err == nil || !strings.Contains(err.Error(), "at least one routing value") {
			t.Fatalf("expand = %v, want empty-enum rejection", err)
		}
	})
}

func TestClassifiedTransitionSemanticValidation(t *testing.T) {
	t.Run("from must be a declared output field", func(t *testing.T) {
		doc := strings.Replace(syntheticClassifyDoc(syntheticClassifyTransitions), `"from": "handoff"`, `"from": "report"`, 1)
		def, err := LoadModeDefinition([]byte(doc))
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		err = validateGuardGraph(map[Mode]Definition{def.Mode: def}, def)
		if err == nil || !strings.Contains(err.Error(), `classify.from "report" is not a declared output field`) {
			t.Fatalf("validateGuardGraph = %v, want undeclared classify.from rejection", err)
		}
	})
	t.Run("field declared required in declared_output is a dead declaration", func(t *testing.T) {
		doc := strings.Replace(syntheticClassifyDoc(syntheticClassifyTransitions),
			`{ "name": "handoff", "type": "object", "required": true, "description": "The execution handoff." }`,
			`{ "name": "handoff", "type": "object", "required": true, "description": "The execution handoff." },
			 { "name": "outcome", "type": "string", "required": true, "enum": ["continue", "complete", "blocked"], "description": "Routing outcome." }`, 1)
		def, err := LoadModeDefinition([]byte(doc))
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		err = validateGuardGraph(map[Mode]Definition{def.Mode: def}, def)
		if err == nil || !strings.Contains(err.Error(), "classification would always short-circuit") {
			t.Fatalf("validateGuardGraph = %v, want required-field conflict rejection", err)
		}
	})
	t.Run("optional declared field is allowed", func(t *testing.T) {
		doc := strings.Replace(syntheticClassifyDoc(syntheticClassifyTransitions),
			`{ "name": "handoff", "type": "object", "required": true, "description": "The execution handoff." }`,
			`{ "name": "handoff", "type": "object", "required": true, "description": "The execution handoff." },
			 { "name": "outcome", "type": "string", "enum": ["continue", "complete", "blocked"], "description": "Routing outcome." }`, 1)
		def, err := LoadModeDefinition([]byte(doc))
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if err := validateGuardGraph(map[Mode]Definition{def.Mode: def}, def); err != nil {
			t.Fatalf("validateGuardGraph rejected an optional declared routing field: %v", err)
		}
	})
}

func TestClassifiedModeValidatesInFullRegistry(t *testing.T) {
	def := loadSyntheticClassifyDefinition(t)
	withSyntheticModeRegistry(t, def)
	if err := ValidateLoadedModes(cloneRegistryForTest()); err != nil {
		t.Fatalf("ValidateLoadedModes with classified mode returned error: %v", err)
	}
}

// newSyntheticClassifyService registers the classified mode and builds a test
// service around one initiative bound to it.
func newSyntheticClassifyService(t *testing.T, classifier FieldClassifier) (*Service, *fakeAgent) {
	t.Helper()
	def := loadSyntheticClassifyDefinition(t)
	withSyntheticModeRegistry(t, def)
	agent := &fakeAgent{states: map[string]agentmanager.RunState{}}
	svc := newTestServiceWithOptions(t, t.TempDir(), serviceOptions{
		agent:      agent,
		classifier: classifier,
		initiatives: fakeInitiatives{items: map[string]InitiativeSnapshot{
			"init-classify": {
				Name:               "init-classify",
				Title:              "Classify Initiative",
				Description:        "Exercises classification-on-transition.",
				Mode:               string(modeSyntheticClassify),
				Items:              []string{"execute/classify"},
				AcceptanceCriteria: []string{"routes without a classifier phase"},
			},
		}},
	})
	return svc, agent
}

func runClassifyRound(t *testing.T, svc *Service, agent *fakeAgent, phase Phase, summary string) RoundEnvelope {
	t.Helper()
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-classify",
		Phase:          string(phase),
		RequestedBy:    "classify-test",
	})
	if err != nil {
		t.Fatalf("StartPhase(%s): %v", phase, err)
	}
	agent.states[round.RunID] = agentmanager.RunState{
		RunID:     round.RunID,
		Status:    "complete",
		Summary:   summary,
		StartedAt: "2026-04-30T12:00:00Z",
	}
	refreshed, err := svc.RefreshRound(context.Background(), "init-classify", modeSyntheticClassify, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound(%03d): %v", round.Round, err)
	}
	return refreshed
}

func classificationRecord(t *testing.T, round RoundEnvelope) PhaseResolutionRecord {
	t.Helper()
	record, ok := RoundPayload(round.Payload).TransitionClassification()
	if !ok {
		t.Fatalf("round %03d has no transition_classification record (payload=%v)", round.Round, round.Payload)
	}
	return record
}

func TestClassifyTransitionL1DerivesFromStructuredHandoff(t *testing.T) {
	svc, agent := newSyntheticClassifyService(t, nil)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "drained slice 1", "next_step": "slice 2", "outcome": "continue"}
		}
	}`)
	if round.Status != RoundStatusCompleted {
		t.Fatalf("round status = %q error=%q, want completed", round.Status, round.Error)
	}
	record := classificationRecord(t, round)
	if record.Outcome != ResolutionResolved || record.Layer != ResolutionLayerExtract {
		t.Fatalf("record = %+v, want resolved via L1 deterministic extraction", record)
	}
	if record.ClassifiedField != "outcome" || record.ClassifiedValue != "continue" {
		t.Fatalf("record classified %s=%s, want outcome=continue", record.ClassifiedField, record.ClassifiedValue)
	}
	if got, _ := RoundPayload(round.Payload).String("outcome"); got != "continue" {
		t.Fatalf("payload outcome = %q, want hoisted derived value", got)
	}
	// The derived value feeds the ordinary guard evaluation: continue loops work.
	actions := workspacePhaseActionsFor(t, svc, "init-classify")
	if !actions["work"].Startable || actions["wrap"].Startable {
		t.Fatalf("actions after continue = work:%+v wrap:%+v, want work startable only", actions["work"], actions["wrap"])
	}
}

func TestClassifyTransitionShortCircuitsWhenFieldEmitted(t *testing.T) {
	svc, agent := newSyntheticClassifyService(t, nil)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "all slices drained"},
			"outcome": "complete"
		}
	}`)
	if round.Status != RoundStatusCompleted {
		t.Fatalf("round status = %q error=%q, want completed", round.Status, round.Error)
	}
	record := classificationRecord(t, round)
	if record.Outcome != ResolutionNotRequired || record.Layer != ResolutionLayerNone {
		t.Fatalf("record = %+v, want not_required short-circuit", record)
	}
	if record.ClassifiedValue != "complete" {
		t.Fatalf("record value = %q, want complete", record.ClassifiedValue)
	}
	actions := workspacePhaseActionsFor(t, svc, "init-classify")
	if !actions["wrap"].Startable || actions["work"].Startable {
		t.Fatalf("actions after complete = work:%+v wrap:%+v, want wrap startable only", actions["work"], actions["wrap"])
	}
}

func TestClassifyTransitionL2FallbackViaStubbedClassifier(t *testing.T) {
	classifier := &stubClassifier{answers: map[string]string{"outcome": "complete"}}
	svc, agent := newSyntheticClassifyService(t, classifier)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "prose only: everything on the plan is finished and validated"}
		}
	}`)
	if round.Status != RoundStatusCompleted {
		t.Fatalf("round status = %q error=%q, want completed via L2", round.Status, round.Error)
	}
	record := classificationRecord(t, round)
	if record.Outcome != ResolutionRecovered || record.Layer != ResolutionLayerClassifier {
		t.Fatalf("record = %+v, want recovered via L2 classifier", record)
	}
	if record.ClassifiedValue != "complete" {
		t.Fatalf("record value = %q, want complete", record.ClassifiedValue)
	}
	actions := workspacePhaseActionsFor(t, svc, "init-classify")
	if !actions["wrap"].Startable {
		t.Fatalf("wrap action = %+v, want startable after L2-classified complete", actions["wrap"])
	}
}

func TestClassifyTransitionAbstainParksRoundInNeedsAttention(t *testing.T) {
	// Classifier abstains (no answer for the field) — the round must land in
	// needs_attention and must NOT route.
	classifier := &stubClassifier{answers: map[string]string{}}
	svc, agent := newSyntheticClassifyService(t, classifier)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "ambiguous state"}
		}
	}`)
	if round.Status != RoundStatusNeedsAttention {
		t.Fatalf("round status = %q, want needs_attention on classification abstain", round.Status)
	}
	if !strings.Contains(round.Error, "transition classification abstained") || !strings.Contains(round.Error, `"outcome"`) {
		t.Fatalf("round error = %q, want abstain reason naming the routing field", round.Error)
	}
	record := classificationRecord(t, round)
	if record.Outcome != ResolutionAbstained {
		t.Fatalf("record = %+v, want abstained", record)
	}
	if _, present := round.Payload["outcome"]; present {
		t.Fatalf("payload outcome = %v, want no fabricated routing value", round.Payload["outcome"])
	}
	// No route: the round did not complete, so its guards routed nowhere (a
	// fresh start of the start phase remains available to the operator, but the
	// abstained round never routed to wrap).
	actions := workspacePhaseActionsFor(t, svc, "init-classify")
	if actions["wrap"].Startable {
		t.Fatalf("wrap action after abstain = %+v, want not startable (no route on abstain)", actions["wrap"])
	}
}

func TestClassifyTransitionEmittedOutOfEnumValueAbstains(t *testing.T) {
	// An explicit out-of-enum routing value is a contract violation the
	// classifier never overrides.
	classifier := &stubClassifier{answers: map[string]string{"outcome": "complete"}}
	svc, agent := newSyntheticClassifyService(t, classifier)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "done-ish"},
			"outcome": "finished"
		}
	}`)
	if round.Status != RoundStatusNeedsAttention {
		t.Fatalf("round status = %q error=%q, want needs_attention on out-of-enum emitted value", round.Status, round.Error)
	}
	record := classificationRecord(t, round)
	if record.Outcome != ResolutionAbstained || len(record.Violations) == 0 {
		t.Fatalf("record = %+v, want abstained with enum violation", record)
	}
}

func TestClassifyTransitionGuardedStopOnBlocked(t *testing.T) {
	svc, agent := newSyntheticClassifyService(t, nil)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "cannot proceed", "outcome": "blocked"}
		}
	}`)
	if round.Status != RoundStatusCompleted {
		t.Fatalf("round status = %q error=%q, want completed (blocked is a guarded stop, not an abstain)", round.Status, round.Error)
	}
	actions := workspacePhaseActionsFor(t, svc, "init-classify")
	if actions["work"].Startable || actions["wrap"].Startable {
		t.Fatalf("actions after blocked = work:%+v wrap:%+v, want guarded stop (nothing startable)", actions["work"], actions["wrap"])
	}
}

func workspacePhaseActionsFor(t *testing.T, svc *Service, initiative string) map[Phase]WorkspacePhase {
	t.Helper()
	workspace, err := svc.Workspace(context.Background(), initiative)
	if err != nil {
		t.Fatalf("Workspace(%s): %v", initiative, err)
	}
	return workspacePhaseActions(workspace)
}

func classifyExampleRun(stepOutput string) []byte {
	return []byte(fmt.Sprintf(`{
	  "kind": "operating-mode-example-run",
	  "id": "happy-path",
	  "mode": "synthetic-classify",
	  "steps": [
	    { "phase": "work", "output": %s }
	  ],
	  "expected_path": ["work", "wrap"]
	}`, stepOutput))
}

func TestWalkExampleRunDerivesClassifiedRoutingField(t *testing.T) {
	def := loadSyntheticClassifyDefinition(t)
	run, err := LoadExampleRun(classifyExampleRun(`{"handoff": {"summary": "drained", "outcome": "complete"}}`))
	if err != nil {
		t.Fatalf("load example-run: %v", err)
	}
	walked, err := WalkExampleRun(map[Mode]Definition{def.Mode: def}, def, run)
	if err != nil {
		t.Fatalf("WalkExampleRun: %v", err)
	}
	if len(walked) != 2 || walked[0] != "work" || walked[1] != "wrap" {
		t.Fatalf("walked = %v, want [work wrap]", walked)
	}
	if _, present := run.Steps[0].Output["outcome"]; present {
		t.Fatalf("walk mutated the fixture's seeded output: %v", run.Steps[0].Output)
	}
}

func TestWalkExampleRunFailsWhenClassificationNotDerivable(t *testing.T) {
	def := loadSyntheticClassifyDefinition(t)
	run, err := LoadExampleRun(classifyExampleRun(`{"handoff": {"summary": "prose only"}}`))
	if err != nil {
		t.Fatalf("load example-run: %v", err)
	}
	_, err = WalkExampleRun(map[Mode]Definition{def.Mode: def}, def, run)
	if err == nil || !strings.Contains(err.Error(), `could not derive routing field "outcome" deterministically`) {
		t.Fatalf("WalkExampleRun = %v, want deterministic-derivation authoring error", err)
	}
}

func TestClassifiedRoundEnvelopeSurfacesClassificationOverConnect(t *testing.T) {
	svc, agent := newSyntheticClassifyService(t, nil)
	round := runClassifyRound(t, svc, agent, "work", `{
		"operating_mode_result": {
			"handoff": {"summary": "drained slice 1", "outcome": "continue"}
		}
	}`)
	pb := roundEnvelopeToProto(round)
	if pb.TransitionClassification == nil {
		t.Fatalf("proto round has no transition_classification")
	}
	if pb.TransitionClassification.ClassifiedField != "outcome" || pb.TransitionClassification.ClassifiedValue != "continue" {
		t.Fatalf("proto classification = %+v, want outcome=continue", pb.TransitionClassification)
	}
	if pb.TransitionClassification.Outcome != string(ResolutionResolved) {
		t.Fatalf("proto classification outcome = %q, want resolved", pb.TransitionClassification.Outcome)
	}
	if pb.Resolution == nil || pb.Resolution.ClassifiedField != "" {
		t.Fatalf("proto phase resolution = %+v, want unmarked phase-output record", pb.Resolution)
	}
}

func TestDeriveTransitionValuePrecedence(t *testing.T) {
	contract := TransitionClassification{Field: "outcome", Enum: []string{"continue", "complete"}, From: "handoff"}

	// Emitted field wins over the inline source value.
	both := NewMapFieldLookup(map[string]any{
		"outcome": "complete",
		"handoff": map[string]any{"outcome": "continue"},
	})
	value, layer, _, ok := deriveTransitionValue(contract, both)
	if !ok || value != "complete" || layer != ResolutionLayerNone {
		t.Fatalf("derive = (%q, %q, %v), want emitted complete short-circuit", value, layer, ok)
	}

	// Inline on the source object is L1.
	inline := NewMapFieldLookup(map[string]any{"handoff": map[string]any{"outcome": "continue"}})
	value, layer, _, ok = deriveTransitionValue(contract, inline)
	if !ok || value != "continue" || layer != ResolutionLayerExtract {
		t.Fatalf("derive = (%q, %q, %v), want L1 continue", value, layer, ok)
	}

	// Absent everywhere: not derivable, no violations.
	value, _, violations, ok := deriveTransitionValue(contract, NewMapFieldLookup(map[string]any{"handoff": map[string]any{}}))
	if ok || value != "" || len(violations) != 0 {
		t.Fatalf("derive = (%q, %v, %v), want plain miss", value, violations, ok)
	}
}

func TestSetPayloadFieldMergesIntoTypedIntermediate(t *testing.T) {
	payload := map[string]any{"progress": ProgressState{Decision: ProgressContinue, Rationale: "keep going"}}
	setPayloadField(payload, "progress.decision", "complete")
	decision, present := NewMapFieldLookup(payload).Lookup("progress.decision")
	if !present || decision != "complete" {
		t.Fatalf("progress.decision = %v/%v, want merged write", decision, present)
	}
	rationale, present := NewMapFieldLookup(payload).Lookup("progress.rationale")
	if !present || rationale != "keep going" {
		t.Fatalf("progress.rationale = %v/%v, want sibling preserved through struct coercion", rationale, present)
	}
}
