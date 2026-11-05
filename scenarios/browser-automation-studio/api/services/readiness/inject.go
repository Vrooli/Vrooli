package readiness

import (
	"context"
	"strings"

	"github.com/google/uuid"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

// InjectPostNavigationWaits rewires a flow so the resolved readiness waits run
// immediately after its first navigate node, before anything that navigate was
// feeding into.
//
// It is graph-level rather than a navigate timeout on purpose: the distinction
// between "the document loaded" and "the page's declared surfaces settled" is
// the whole point, and collapsing them into one timeout throws it away.
//
// Returns true when the flow was rewritten. A flow with no navigate node, or no
// waits to add, is left untouched.
func InjectPostNavigationWaits(flow *workflowsv1.WorkflowDefinitionV2, waits []*actionsv1.WaitParams) bool {
	if flow == nil || len(waits) == 0 {
		return false
	}
	navID := ""
	for _, node := range flow.GetNodes() {
		if node.GetAction().GetNavigate() != nil {
			navID = node.GetId()
			break
		}
	}
	if navID == "" {
		return false
	}

	var waitIDs []string
	for _, wait := range waits {
		if wait == nil {
			continue
		}
		waitID := uuid.NewString()
		waitIDs = append(waitIDs, waitID)
		flow.Nodes = append(flow.Nodes, &workflowsv1.WorkflowNodeV2{
			Id: waitID,
			Action: &actionsv1.ActionDefinition{
				Type:   actionsv1.ActionType_ACTION_TYPE_WAIT,
				Params: &actionsv1.ActionDefinition_Wait{Wait: wait},
			},
			// A readiness wait must never turn a passing case into a failing
			// one. It is a settle hint derived from a contract the case did not
			// author, so a route where the declared region does not render — a
			// tab-scoped surface reached by a later click, for instance — has to
			// lapse and proceed, exactly as the case behaved before injection.
			// Without this the contract trades one timeout for another and
			// makes the suite worse.
			ExecutionSettings: &workflowsv1.NodeExecutionSettings{
				ContinueOnError: proto.Bool(true),
			},
		})
	}
	if len(waitIDs) == 0 {
		return false
	}

	// Everything the navigate fed now hangs off the last wait, so the declared
	// surfaces are settled before the next authored step runs.
	var edges []*workflowsv1.WorkflowEdgeV2
	for _, edge := range flow.GetEdges() {
		if edge.GetSource() == navID {
			edges = append(edges, &workflowsv1.WorkflowEdgeV2{
				Id:     edge.GetId(),
				Source: waitIDs[len(waitIDs)-1],
				Target: edge.GetTarget(),
			})
			continue
		}
		edges = append(edges, edge)
	}
	edges = append(edges, &workflowsv1.WorkflowEdgeV2{Id: uuid.NewString(), Source: navID, Target: waitIDs[0]})
	for index := 1; index < len(waitIDs); index++ {
		edges = append(edges, &workflowsv1.WorkflowEdgeV2{
			Id:     uuid.NewString(),
			Source: waitIDs[index-1],
			Target: waitIDs[index],
		})
	}
	flow.Edges = edges
	return true
}

// NavigateTarget reports the scenario and route of a flow's first navigate node
// when that navigate addresses a known local scenario. A raw-URL navigate, an
// external site, or a flow with no navigate at all returns ok=false, which is
// what keeps undeclared and third-party targets on generic behavior.
func NavigateTarget(flow *workflowsv1.WorkflowDefinitionV2) (scenario, route string, ok bool) {
	if flow == nil {
		return "", "", false
	}
	for _, node := range flow.GetNodes() {
		navigate := node.GetAction().GetNavigate()
		if navigate == nil {
			continue
		}
		scenario = strings.TrimSpace(navigate.GetScenario())
		if scenario == "" {
			return "", "", false
		}
		route = strings.TrimSpace(navigate.GetScenarioPath())
		if route == "" {
			route = "/"
		}
		return scenario, route, true
	}
	return "", "", false
}

// HasExplicitWait reports whether the flow already waits immediately after its
// first navigate. An author who wrote their own wait has said what readiness
// means for this flow, and that always outranks the declared profile.
func HasExplicitWait(flow *workflowsv1.WorkflowDefinitionV2) bool {
	if flow == nil {
		return false
	}
	navID := ""
	for _, node := range flow.GetNodes() {
		if node.GetAction().GetNavigate() != nil {
			navID = node.GetId()
			break
		}
	}
	if navID == "" {
		return false
	}
	byID := make(map[string]*workflowsv1.WorkflowNodeV2, len(flow.GetNodes()))
	for _, node := range flow.GetNodes() {
		byID[node.GetId()] = node
	}
	for _, edge := range flow.GetEdges() {
		if edge.GetSource() != navID {
			continue
		}
		if target := byID[edge.GetTarget()]; target.GetAction().GetWait() != nil {
			return true
		}
	}
	return false
}

// Outcome records which rung of the settle ladder a flow ended up on, so a run
// report can state it rather than leaving a silent fallback looking like a fast
// pass.
type Outcome struct {
	Strategy           string
	ProfileVersion     string
	Route              string
	RequiredSurfaceIDs []string
	FallbackReason     string
}

// Settle applies the readiness ladder to a flow in place and reports what it
// did. The order is deliberate:
//
//  1. an explicit authored wait wins outright;
//  2. otherwise a known local scenario's declared required surfaces are waited
//     on by terminal lifecycle state;
//  3. otherwise the flow is untouched and keeps whatever its navigate already
//     did — which is exactly today's behavior.
//
// Settle never returns an error. Every failure path degrades to rung 3 with a
// stated reason, because a scenario that has not adopted the contract, or an
// Experience Manager that is down, must not fail somebody's workflow run.
func Settle(ctx context.Context, resolver Resolver, flow *workflowsv1.WorkflowDefinitionV2) Outcome {
	if flow == nil {
		return Outcome{Strategy: StrategyGenericNavigation, FallbackReason: "no flow definition"}
	}
	if HasExplicitWait(flow) {
		return Outcome{Strategy: StrategyExplicitWait}
	}
	if resolver == nil {
		return Outcome{Strategy: StrategyGenericNavigation, FallbackReason: "no readiness resolver configured"}
	}
	scenario, route, ok := NavigateTarget(flow)
	if !ok {
		return Outcome{Strategy: StrategyGenericNavigation, FallbackReason: "flow does not navigate to a known local scenario"}
	}
	resolution, err := resolver.ResolveReadinessWaits(ctx, scenario, route)
	if err != nil {
		return Outcome{
			Strategy:       StrategyGenericNavigation,
			Route:          route,
			FallbackReason: "declared readiness profile unavailable: " + err.Error(),
		}
	}
	if !InjectPostNavigationWaits(flow, resolution.Waits) {
		return Outcome{
			Strategy:       StrategyGenericNavigation,
			ProfileVersion: resolution.ProfileVersion,
			Route:          route,
			FallbackReason: resolution.FallbackReason(),
		}
	}
	return Outcome{
		Strategy:           StrategyDeclaredSurface,
		ProfileVersion:     resolution.ProfileVersion,
		Route:              route,
		RequiredSurfaceIDs: resolution.RequiredSurfaceIDs,
	}
}
