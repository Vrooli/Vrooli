package readiness

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

type stubResolver struct {
	resolution Resolution
	err        error
}

func (s stubResolver) ResolveReadinessWaits(context.Context, string, string) (Resolution, error) {
	return s.resolution, s.err
}

func navigateNode(id, scenario, path string) *workflowsv1.WorkflowNodeV2 {
	return &workflowsv1.WorkflowNodeV2{
		Id: id,
		Action: &actionsv1.ActionDefinition{
			Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{
				Scenario:        &scenario,
				ScenarioPath:    &path,
				DestinationType: destinationScenario(),
			}},
		},
	}
}

func destinationScenario() *actionsv1.NavigateDestinationType {
	d := actionsv1.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_SCENARIO
	return &d
}

func clickNode(id string) *workflowsv1.WorkflowNodeV2 {
	return &workflowsv1.WorkflowNodeV2{
		Id: id,
		Action: &actionsv1.ActionDefinition{
			Type:   actionsv1.ActionType_ACTION_TYPE_CLICK,
			Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#x"}},
		},
	}
}

func waitNode(id string) *workflowsv1.WorkflowNodeV2 {
	return &workflowsv1.WorkflowNodeV2{
		Id: id,
		Action: &actionsv1.ActionDefinition{
			Type: actionsv1.ActionType_ACTION_TYPE_WAIT,
			Params: &actionsv1.ActionDefinition_Wait{Wait: &actionsv1.WaitParams{
				WaitFor: &actionsv1.WaitParams_Selector{Selector: "#authored"},
			}},
		},
	}
}

// navigateThenClick is the shape every bas case starts with: land on a route,
// then immediately interact with something the route had to finish rendering.
func navigateThenClick() *workflowsv1.WorkflowDefinitionV2 {
	return &workflowsv1.WorkflowDefinitionV2{
		Nodes: []*workflowsv1.WorkflowNodeV2{navigateNode("nav", "browser-automation-studio", "/"), clickNode("click")},
		Edges: []*workflowsv1.WorkflowEdgeV2{{Id: "e1", Source: "nav", Target: "click"}},
	}
}

func declaredWaits() []*actionsv1.WaitParams {
	return []*actionsv1.WaitParams{{WaitFor: &actionsv1.WaitParams_Selector{
		Selector: `[data-testid="projects-grid-surface"][data-experience-state="ready"]`,
	}}}
}

func TestSettleWaitsOnDeclaredSurfaceBeforeTheNextStep(t *testing.T) {
	flow := navigateThenClick()
	out := Settle(context.Background(), stubResolver{resolution: Resolution{
		Waits:              declaredWaits(),
		ProfileVersion:     "experience-readiness-profile/v1",
		RouteMatched:       true,
		RequiredSurfaceIDs: []string{"projects-grid"},
	}}, flow)

	if out.Strategy != StrategyDeclaredSurface {
		t.Fatalf("strategy = %q, want %q (fallback: %s)", out.Strategy, StrategyDeclaredSurface, out.FallbackReason)
	}
	if len(flow.GetNodes()) != 3 {
		t.Fatalf("expected an injected wait node, got %d nodes", len(flow.GetNodes()))
	}
	// The authored click must now run after the wait, not after the navigate.
	// If it still hangs off the navigate the injection achieved nothing.
	for _, edge := range flow.GetEdges() {
		if edge.GetTarget() == "click" && edge.GetSource() == "nav" {
			t.Fatal("click still runs directly after navigate; the wait was not interposed")
		}
	}
}

func TestSettleLeavesAnAuthoredWaitAlone(t *testing.T) {
	flow := &workflowsv1.WorkflowDefinitionV2{
		Nodes: []*workflowsv1.WorkflowNodeV2{navigateNode("nav", "browser-automation-studio", "/"), waitNode("authored")},
		Edges: []*workflowsv1.WorkflowEdgeV2{{Id: "e1", Source: "nav", Target: "authored"}},
	}
	out := Settle(context.Background(), stubResolver{resolution: Resolution{Waits: declaredWaits(), RouteMatched: true}}, flow)

	if out.Strategy != StrategyExplicitWait {
		t.Fatalf("strategy = %q, want %q", out.Strategy, StrategyExplicitWait)
	}
	if len(flow.GetNodes()) != 2 {
		t.Fatalf("an authored wait must not be supplemented; got %d nodes", len(flow.GetNodes()))
	}
}

func TestSettleFallsBackAndSaysWhy(t *testing.T) {
	cases := []struct {
		name     string
		flow     *workflowsv1.WorkflowDefinitionV2
		resolver Resolver
		reason   string
	}{
		{
			name:     "experience manager unavailable",
			flow:     navigateThenClick(),
			resolver: stubResolver{err: errors.New("dial tcp: connection refused")},
			reason:   "declared readiness profile unavailable: dial tcp: connection refused",
		},
		{
			name:     "route not declared",
			flow:     navigateThenClick(),
			resolver: stubResolver{resolution: Resolution{ProfileVersion: "v1"}},
			reason:   "declared readiness profile does not include the requested route",
		},
		{
			name:     "route declared but nothing bound",
			flow:     navigateThenClick(),
			resolver: stubResolver{resolution: Resolution{ProfileVersion: "v1", RouteMatched: true}},
			reason:   "declared readiness route has no bound required surfaces",
		},
		{
			name: "external url is not a known scenario",
			flow: &workflowsv1.WorkflowDefinitionV2{
				Nodes: []*workflowsv1.WorkflowNodeV2{navigateNode("nav", "", "")},
			},
			resolver: stubResolver{resolution: Resolution{Waits: declaredWaits(), RouteMatched: true}},
			reason:   "flow does not navigate to a known local scenario",
		},
		{
			name:     "no resolver configured",
			flow:     navigateThenClick(),
			resolver: nil,
			reason:   "no readiness resolver configured",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := len(tc.flow.GetNodes())
			out := Settle(context.Background(), tc.resolver, tc.flow)
			if out.Strategy != StrategyGenericNavigation {
				t.Fatalf("strategy = %q, want %q", out.Strategy, StrategyGenericNavigation)
			}
			if out.FallbackReason != tc.reason {
				t.Fatalf("reason = %q, want %q", out.FallbackReason, tc.reason)
			}
			if len(tc.flow.GetNodes()) != before {
				t.Fatalf("fallback must leave the flow untouched; nodes %d -> %d", before, len(tc.flow.GetNodes()))
			}
		})
	}
}

func TestTerminalSelectorExcludesLoadingButKeepsError(t *testing.T) {
	got := TerminalSelector(`[data-testid="grid"]`, "async", []string{"loading", "ready", "empty", "error", "static"})
	want := `[data-testid="grid"][data-experience-state="ready"], ` +
		`[data-testid="grid"][data-experience-state="empty"], ` +
		`[data-testid="grid"][data-experience-state="error"]`
	if got != want {
		t.Fatalf("selector = %q, want %q", got, want)
	}
}

func TestNavigateTargetDefaultsRootRoute(t *testing.T) {
	flow := &workflowsv1.WorkflowDefinitionV2{
		Nodes: []*workflowsv1.WorkflowNodeV2{navigateNode("nav", "browser-automation-studio", "")},
	}
	scenario, route, ok := NavigateTarget(flow)
	if !ok || scenario != "browser-automation-studio" || route != "/" {
		t.Fatalf("got (%q, %q, %v)", scenario, route, ok)
	}
}

// TestInjectedWaitCannotFailACase locks the invariant that makes this contract
// safe to turn on fleet-wide: a readiness wait is an optimization, never a new
// failure mode.
//
// The regression it guards is real. BAS declares `projects-grid` on the
// dashboard page whose route is "/", but that region only renders under
// ?tab=projects — the cases reach it by clicking a tab after navigating. Route
// matching discards the query string, so navigating to bare "/" resolved the
// region and injected a wait for a surface that cannot exist yet. With a
// visible-state 30s wait and no continue_on_error, that turned three cases
// which had been PASSING into 30-second failures.
func TestInjectedWaitCannotFailACase(t *testing.T) {
	flow := &workflowsv1.WorkflowDefinitionV2{
		Nodes: []*workflowsv1.WorkflowNodeV2{{
			Id: "nav",
			Action: &actionsv1.ActionDefinition{
				Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE,
				Params: &actionsv1.ActionDefinition_Navigate{Navigate: &actionsv1.NavigateParams{
					Scenario: proto.String("browser-automation-studio"), ScenarioPath: proto.String("/"),
				}},
			},
		}},
	}
	wait := &actionsv1.WaitParams{
		WaitFor:   &actionsv1.WaitParams_Selector{Selector: `[data-testid="x"][data-experience-state="ready"]`},
		State:     actionsv1.WaitState_WAIT_STATE_ATTACHED.Enum(),
		TimeoutMs: proto.Int32(int32(settleTimeout.Milliseconds())),
	}

	require.True(t, InjectPostNavigationWaits(flow, []*actionsv1.WaitParams{wait}))

	var injected *workflowsv1.WorkflowNodeV2
	for _, node := range flow.GetNodes() {
		if node.GetAction().GetWait() != nil {
			injected = node
			break
		}
	}
	require.NotNil(t, injected, "expected an injected wait node")

	require.True(t, injected.GetExecutionSettings().GetContinueOnError(),
		"an injected readiness wait must lapse rather than fail the case")
	require.Equal(t, actionsv1.WaitState_WAIT_STATE_ATTACHED, injected.GetAction().GetWait().GetState(),
		"a lifecycle probe must not require visibility: empty and error states may render nothing")
	require.Less(t, injected.GetAction().GetWait().GetTimeoutMs(), int32(30000),
		"the settle must be bounded well below the driver default it replaces")
}
