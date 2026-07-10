package operatingmode

import (
	"encoding/json"
	"testing"
)

// TestHandoffToProto_CarriesFrontier proves the elastic-slice frontier travels
// over the Connect wire: the declared true-frontier the execute round emits is
// projected onto the OperatingModeHandoff message, so consumers (UI, CLI) can
// read where the next round should continue.
func TestHandoffToProto_CarriesFrontier(t *testing.T) {
	h := &Handoff{
		Summary:  "Landed the durable_run primitive.",
		NextStep: "classify_progress",
		Frontier: "Migrate test-genie execute onto durable run handles.",
	}
	got := handoffToProto(h)
	if got == nil {
		t.Fatal("handoffToProto returned nil")
	}
	if got.Frontier != h.Frontier {
		t.Fatalf("proto frontier = %q, want %q", got.Frontier, h.Frontier)
	}
	if got.NextStep != h.NextStep {
		t.Fatalf("proto next_step = %q, want %q", got.NextStep, h.NextStep)
	}
	if got.Summary != h.Summary {
		t.Fatalf("proto summary = %q, want %q", got.Summary, h.Summary)
	}
}

// TestHandoff_FrontierRoundTripsThroughJSON proves the frontier field parses
// from the agent's operating_mode_result envelope (the inbound direction) via
// the json tag, so a handoff an agent emits is captured, not dropped.
func TestHandoff_FrontierRoundTripsThroughJSON(t *testing.T) {
	const raw = `{"summary":"x","next_step":"review","frontier":"the exact remainder"}`
	var h Handoff
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		t.Fatalf("unmarshal handoff: %v", err)
	}
	if h.Frontier != "the exact remainder" {
		t.Fatalf("frontier = %q, want %q", h.Frontier, "the exact remainder")
	}
}

func TestRoundEnvelopeToProtoCarriesResolution(t *testing.T) {
	round := RoundEnvelope{
		ExecutionID: "execution-1", DefinitionDigest: "sha256:def", Round: 1,
		Status: RoundStatusNeedsAttention,
		Payload: map[string]any{
			resultEnvelopeKey: map[string]any{
				"novel_flag": true,
				"details":    map[string]any{"label": "preserved"},
			},
			payloadResolution: PhaseResolutionRecord{
				Outcome:         ResolutionAbstained,
				Layer:           ResolutionLayerClassifier,
				MessagesScanned: 2,
				Missing:         []string{"verdict"},
				Violations:      []string{"confidence: below minimum"},
				Notes:           []string{"classifier abstained on verdict"},
				SelectedMessage: &SelectedMessageProvenance{
					EventID: "event-final", Sequence: 42,
					ContentDigest:             "sha256:abc123",
					SelectionAlgorithmVersion: finalMessageSelectionVersion,
					FallbackReason:            "earlier_contract_satisfying_assistant_event",
				},
			},
		},
	}
	got := roundEnvelopeToProto(round)
	if got.GetResolution() == nil {
		t.Fatal("resolution projection is nil")
	}
	if got.GetStatus() != string(RoundStatusNeedsAttention) {
		t.Fatalf("status = %q, want needs_attention", got.GetStatus())
	}
	if got.GetExecutionId() != round.ExecutionID || got.GetDefinitionDigest() != round.DefinitionDigest {
		t.Fatalf("execution provenance = %q/%q", got.GetExecutionId(), got.GetDefinitionDigest())
	}
	if got.GetResolution().GetOutcome() != string(ResolutionAbstained) || got.GetResolution().GetLayer() != string(ResolutionLayerClassifier) {
		t.Fatalf("resolution = %+v, want abstained classifier record", got.GetResolution())
	}
	if len(got.GetResolution().GetMissing()) != 1 || got.GetResolution().GetMissing()[0] != "verdict" {
		t.Fatalf("missing = %v, want verdict", got.GetResolution().GetMissing())
	}
	selected := got.GetResolution().GetSelectedMessage()
	if selected.GetEventId() != "event-final" || selected.GetSequence() != 42 || selected.GetContentDigest() != "sha256:abc123" {
		t.Fatalf("selected message = %+v, want stable source provenance", selected)
	}
	if envelope := got.GetResolvedEnvelope().AsMap(); envelope["novel_flag"] != true || envelope["details"].(map[string]any)["label"] != "preserved" {
		t.Fatalf("resolved envelope = %#v, want arbitrary fields", envelope)
	}
}

func TestWorkspaceToProtoCarriesExecutionManifestProvenance(t *testing.T) {
	bundle, digest, err := pinDefinitionBundle(MustDefinition(ModePhasedPlanDrain), DefinitionFor)
	if err != nil {
		t.Fatalf("pinDefinitionBundle: %v", err)
	}
	got := workspaceToProto(Workspace{Executions: []OperatingModeExecution{{
		ExecutionID: "execution-1", ScopeKind: string(TargetPlanManagerPlan), ScopeID: "plan-1",
		Mode: string(ModePhasedPlanDrain), Status: ExecutionStatusActive,
		SchemaVersion: executionManifestSchemaVersion, DefinitionDigest: digest, DefinitionBundle: bundle,
		InputContractDigest: "sha256:inputs", InputSnapshotDigest: "sha256:values",
		ReachablePromptSources: map[string]PinnedPromptSource{"execute": {
			Mode: string(ModePhasedPlanDrain), Phase: "execute", SkillID: "phased-plan-execute-next",
			Revision: "rev-1", ContentHash: "sha256:prompt",
		}},
	}}})
	if len(got.GetExecutions()) != 1 {
		t.Fatalf("executions = %+v", got.GetExecutions())
	}
	execution := got.GetExecutions()[0]
	if execution.GetExecutionId() != "execution-1" || execution.GetDefinitionDigest() != digest {
		t.Fatalf("execution projection = %+v", execution)
	}
	if execution.GetDefinitionBundle() == nil || execution.GetDefinitionBundle().AsMap()["root"] == nil {
		t.Fatalf("definition bundle projection = %+v", execution.GetDefinitionBundle())
	}
	if len(execution.GetReachablePromptSources()) != 1 || execution.GetReachablePromptSources()[0].GetRevision() != "rev-1" {
		t.Fatalf("prompt source projection = %+v", execution.GetReachablePromptSources())
	}
}
