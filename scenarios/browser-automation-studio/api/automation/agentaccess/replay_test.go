package agentaccess

import (
	"context"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/state"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

func TestReplayPromotedFlowUsesSameBrokerSessionAndStopsOnPermissionFailure(t *testing.T) {
	h := newSelectedTabBrokerHarness(t)
	owner := withBridgeContextCaller(context.Background(), selectedTabOwner)
	flow := &workflowsv1.WorkflowDefinitionV2{
		Metadata: &workflowsv1.WorkflowMetadataV2{Labels: map[string]string{"source": "agent-browser-trace"}},
		Nodes: []*workflowsv1.WorkflowNodeV2{
			{Id: "click-1", Action: selectedTabClickRequest().Instruction.Action},
			{Id: "click-2", Action: selectedTabClickRequest().Instruction.Action},
		},
		Edges: []*workflowsv1.WorkflowEdgeV2{{Source: "click-1", Target: "click-2"}},
	}
	close(h.peer.commit)
	result, err := h.broker.ReplayPromotedFlow(owner, h.entry.sessionID, flow, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Completed || result.SessionID != h.entry.sessionID || len(result.Results) != 2 || h.peer.actionCount.Load() != 2 {
		t.Fatalf("replay result=%+v peer-actions=%d", result, h.peer.actionCount.Load())
	}
	failing := proto.Clone(flow).(*workflowsv1.WorkflowDefinitionV2)
	failing.GetNodes()[1].Action = &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: "https://blocked.test/"}}}
	beforeFailure := h.peer.actionCount.Load()
	partial, err := h.broker.ReplayPromotedFlow(owner, h.entry.sessionID, failing, nil)
	if err != nil {
		t.Fatal(err)
	}
	if partial.Completed || len(partial.Results) != 2 || !partial.Results[0].Success || partial.Results[1].FailureClass != "action_denied" || h.peer.actionCount.Load() != beforeFailure+1 {
		t.Fatalf("replay did not stop after denied second action: result=%+v actions=%d before=%d", partial, h.peer.actionCount.Load(), beforeFailure)
	}

	// Revoking control between runs denies the first saved action and prevents
	// the remaining node from reaching the selected-tab bridge.
	observeOnly := h.entry.profile
	observeOnly.Control = false
	observeOnly.Effect = sessionpermission.EffectObserve
	h.broker.profiles = bridgeProfileResolver{profile: observeOnly, authorizer: &bridgePolicyRecorder{profile: observeOnly}}
	before := h.peer.actionCount.Load()
	denied, err := h.broker.ReplayPromotedFlow(owner, h.entry.sessionID, flow, nil)
	if err != nil {
		t.Fatal(err)
	}
	if denied.Completed || len(denied.Results) != 1 || denied.Results[0].FailureClass != "action_denied" || h.peer.actionCount.Load() != before {
		t.Fatalf("revoked replay did not fail-stop before bridge action: result=%+v actions=%d before=%d", denied, h.peer.actionCount.Load(), before)
	}
}

func TestReplayPromotedFlowRejectsActionsOutsideTraceCompilerSubsetBeforeEffects(t *testing.T) {
	for _, tc := range []struct {
		name   string
		action *actionsv1.ActionDefinition
	}{
		{name: "keyboard", action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_KEYBOARD, Params: &actionsv1.ActionDefinition_Keyboard{Keyboard: &actionsv1.KeyboardParams{}}}},
		{name: "screenshot", action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_SCREENSHOT, Params: &actionsv1.ActionDefinition_Screenshot{Screenshot: &actionsv1.ScreenshotParams{}}}},
		{name: "mismatched typed params", action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: "https://example.test"}}}},
		{name: "unstable selector", action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "button"}}}},
		{name: "invalid assertion", action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newSelectedTabBrokerHarness(t)
			flow := &workflowsv1.WorkflowDefinitionV2{
				Metadata: &workflowsv1.WorkflowMetadataV2{Labels: map[string]string{"source": "agent-browser-trace"}},
				Nodes:    []*workflowsv1.WorkflowNodeV2{{Id: "uncompiled", Action: tc.action}},
			}
			before := h.peer.actionCount.Load()
			if _, err := h.broker.ReplayPromotedFlow(withBridgeContextCaller(context.Background(), selectedTabOwner), h.entry.sessionID, flow, nil); err == nil {
				t.Fatal("non-compiler action accepted for deterministic replay")
			}
			if got := h.peer.actionCount.Load(); got != before {
				t.Fatalf("unsupported replay action reached browser owner: before=%d after=%d", before, got)
			}
		})
	}
}

func TestOrderedLinearActionsPreservesWorkflowEdgesAndRejectsBranches(t *testing.T) {
	flow := &workflowsv1.WorkflowDefinitionV2{
		Nodes: []*workflowsv1.WorkflowNodeV2{{Id: "third", Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_ASSERT, Params: &actionsv1.ActionDefinition_Assert{Assert: &actionsv1.AssertParams{Selector: "#go", Mode: basev1.AssertionMode_ASSERTION_MODE_VISIBLE}}}}, {Id: "first", Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_CLICK, Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#go"}}}}, {Id: "second", Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_INPUT, Params: &actionsv1.ActionDefinition_Input{Input: &actionsv1.InputParams{Selector: "#q"}}}}},
		Edges: []*workflowsv1.WorkflowEdgeV2{{Source: "first", Target: "second"}, {Source: "second", Target: "third"}},
	}
	ordered, err := orderedLinearActions(flow)
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 3 || ordered[0].GetId() != "first" || ordered[1].GetId() != "second" || ordered[2].GetId() != "third" {
		t.Fatalf("ordered nodes = %#v", ordered)
	}
	flow.Edges = append(flow.Edges, &workflowsv1.WorkflowEdgeV2{Source: "first", Target: "third"})
	if _, err := orderedLinearActions(flow); err == nil {
		t.Fatal("branching persisted flow was accepted for deterministic replay")
	}
}

func TestReplayParameterValidationRequiresEveryRedactedValue(t *testing.T) {
	nodes := []*workflowsv1.WorkflowNodeV2{{Id: "nav", Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: "${@params/navigation_1}"}}}}, {Id: "input", Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_INPUT, Params: &actionsv1.ActionDefinition_Input{Input: &actionsv1.InputParams{Value: "{{@params/input_2}}"}}}}}
	if err := validateReplayParameters(nodes, map[string]string{"navigation_1": "https://safe.example/", "input_2": "value"}); err != nil {
		t.Fatalf("valid replay params rejected: %v", err)
	}
	if err := validateReplayParameters(nodes, map[string]string{"navigation_1": "https://safe.example/"}); err == nil {
		t.Fatal("missing required flow parameter was accepted")
	}
}

func TestPromotedActionParameterSubstitutionIsTypedAndSinglePass(t *testing.T) {
	value := "a${@params/other}"
	interpolator := state.NewInterpolator(state.New(nil, map[string]any{"navigation_1": "https://example.test/?q=" + value}, nil))
	action := &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE, Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{Url: "${@params/navigation_1}"}}}
	resolved, targetURL, err := interpolatePromotedAction(action, interpolator)
	if err != nil {
		t.Fatal(err)
	}
	want := "https://example.test/?q=" + value
	if targetURL != want || resolved.GetNavigate().GetUrl() != want {
		t.Fatalf("single-pass URL=%q action=%q, want %q", targetURL, resolved.GetNavigate().GetUrl(), want)
	}
	if action.GetNavigate().GetUrl() != "${@params/navigation_1}" {
		t.Fatal("parameter substitution mutated the persisted workflow action")
	}
}
