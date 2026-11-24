package agentaccess

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/state"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/proto"
)

const maxPromotedReplayActions = 100
const maxReplayParameterBytes = 64 << 10

type ReplayActionResult struct {
	NodeID       string `json:"node_id"`
	Success      bool   `json:"success"`
	StepType     string `json:"step_type,omitempty"`
	FailureClass string `json:"failure_class,omitempty"`
}

type ReplayResult struct {
	SessionID string               `json:"session_id"`
	Completed bool                 `json:"completed"`
	Results   []ReplayActionResult `json:"results"`
}

// ReplayPromotedFlow runs a saved trace-derived BAS workflow through the same
// Broker session and its per-action permission checks. It deliberately accepts
// only a linear generated flow; branches, loops, and subflows need an execution
// cursor and are not deterministic under this adapter.
func (b *Broker) ReplayPromotedFlow(ctx context.Context, sessionID string, flow *workflowsv1.WorkflowDefinitionV2, parameters map[string]string) (ReplayResult, error) {
	if flow == nil || flow.GetMetadata() == nil || flow.GetMetadata().GetLabels()["source"] != "agent-browser-trace" {
		return ReplayResult{}, errors.New("workflow is not a trace-promoted BAS flow")
	}
	ordered, err := orderedLinearActions(flow)
	if err != nil {
		return ReplayResult{}, err
	}
	if len(ordered) > maxPromotedReplayActions {
		return ReplayResult{}, fmt.Errorf("trace-promoted flow exceeds the %d-action replay limit", maxPromotedReplayActions)
	}
	if err := validateReplayParameters(ordered, parameters); err != nil {
		return ReplayResult{}, err
	}

	stateValues := make(map[string]any, len(parameters))
	for name, value := range parameters {
		stateValues[name] = value
	}
	interpolator := state.NewInterpolator(state.New(nil, stateValues, nil))
	result := ReplayResult{SessionID: sessionID, Results: make([]ReplayActionResult, 0, len(ordered))}
	for index, node := range ordered {
		action := proto.Clone(node.GetAction()).(*actionsv1.ActionDefinition)
		instruction := contracts.CompiledInstruction{Index: index, NodeID: node.GetId(), Action: action}
		action, targetURL, err := interpolatePromotedAction(action, interpolator)
		if err != nil {
			return result, fmt.Errorf("workflow node %s has an unresolved parameter: %w", node.GetId(), err)
		}
		instruction.Action = action
		acted, actErr := b.Act(ctx, sessionID, ActRequest{Instruction: instruction, TargetURL: targetURL})
		row := ReplayActionResult{NodeID: node.GetId()}
		if actErr != nil {
			row.FailureClass = "action_denied"
			result.Results = append(result.Results, row)
			return result, nil
		}
		row.Success, row.StepType = acted.Outcome.Success, acted.Outcome.StepType
		if !row.Success {
			row.FailureClass = "action_failed"
			if acted.Outcome.Failure != nil && acted.Outcome.Failure.Kind != "" {
				row.FailureClass = string(acted.Outcome.Failure.Kind)
			}
		}
		result.Results = append(result.Results, row)
		if !row.Success {
			return result, nil
		}
	}
	result.Completed = true
	return result, nil
}

func orderedLinearActions(flow *workflowsv1.WorkflowDefinitionV2) ([]*workflowsv1.WorkflowNodeV2, error) {
	nodes := flow.GetNodes()
	edges := flow.GetEdges()
	if len(nodes) == 0 || len(nodes) > maxPromotedReplayActions || len(edges) != len(nodes)-1 {
		return nil, errors.New("trace-promoted workflow must be a non-empty linear action sequence")
	}
	byID := make(map[string]*workflowsv1.WorkflowNodeV2, len(nodes))
	indegree := make(map[string]int, len(nodes))
	next := make(map[string]string, len(edges))
	for _, node := range nodes {
		if node == nil || strings.TrimSpace(node.GetId()) == "" || node.GetAction() == nil {
			return nil, errors.New("trace-promoted workflow contains an invalid action node")
		}
		if !isTraceReplayAction(node.GetAction()) {
			return nil, fmt.Errorf("trace-promoted workflow node %s uses an action outside the deterministic trace subset", node.GetId())
		}
		if _, exists := byID[node.GetId()]; exists {
			return nil, errors.New("trace-promoted workflow contains duplicate node IDs")
		}
		byID[node.GetId()] = node
	}
	for _, edge := range edges {
		if edge == nil || byID[edge.GetSource()] == nil || byID[edge.GetTarget()] == nil || edge.GetSource() == edge.GetTarget() || next[edge.GetSource()] != "" {
			return nil, errors.New("trace-promoted workflow contains an invalid or branching edge")
		}
		next[edge.GetSource()] = edge.GetTarget()
		indegree[edge.GetTarget()]++
		if indegree[edge.GetTarget()] > 1 {
			return nil, errors.New("trace-promoted workflow contains a joining edge")
		}
	}
	var start string
	for id := range byID {
		if indegree[id] == 0 {
			if start != "" {
				return nil, errors.New("trace-promoted workflow contains multiple entry nodes")
			}
			start = id
		}
	}
	ordered := make([]*workflowsv1.WorkflowNodeV2, 0, len(nodes))
	seen := make(map[string]bool, len(nodes))
	for id := start; id != ""; id = next[id] {
		if seen[id] {
			return nil, errors.New("trace-promoted workflow contains a cycle")
		}
		seen[id] = true
		ordered = append(ordered, byID[id])
	}
	if len(ordered) != len(nodes) {
		return nil, errors.New("trace-promoted workflow has disconnected or cyclic nodes")
	}
	return ordered, nil
}

// isTraceReplayAction mirrors the only action forms emitted by compileTraceDraft:
// navigation/click/input followed by a typed post-action assertion. A source
// label alone is not enough to make a broader workflow deterministic.
func isTraceReplayAction(action *actionsv1.ActionDefinition) bool {
	if action == nil {
		return false
	}
	switch action.GetType() {
	case actionsv1.ActionType_ACTION_TYPE_NAVIGATE, actionsv1.ActionType_ACTION_TYPE_CLICK, actionsv1.ActionType_ACTION_TYPE_INPUT:
		_, _, err := promotableTraceInstruction(contracts.CompiledInstruction{Action: action}, 1)
		return err == nil
	case actionsv1.ActionType_ACTION_TYPE_ASSERT:
		return validatePostActionAssertion(action) == nil
	default:
		return false
	}
}

func validateReplayParameters(nodes []*workflowsv1.WorkflowNodeV2, parameters map[string]string) error {
	if len(parameters) > maxPromotedReplayActions {
		return fmt.Errorf("workflow replay accepts at most %d parameters", maxPromotedReplayActions)
	}
	required := map[string]bool{}
	for _, node := range nodes {
		var value string
		if nav := node.GetAction().GetNavigate(); nav != nil {
			value = nav.GetUrl()
		} else if input := node.GetAction().GetInput(); input != nil {
			value = input.GetValue()
		}
		for cursor := 0; ; {
			start := strings.Index(value[cursor:], "@params/")
			if start < 0 {
				break
			}
			start += cursor + len("@params/")
			end := start
			for end < len(value) && value[end] != '}' {
				end++
			}
			if end == start || end == len(value) {
				return fmt.Errorf("workflow node %s has a malformed parameter reference", node.GetId())
			}
			required[value[start:end]] = true
			cursor = end + 1
		}
	}
	for name := range required {
		if _, ok := parameters[name]; !ok {
			return fmt.Errorf("required workflow parameter %q is missing", name)
		}
	}
	for name := range parameters {
		if strings.TrimSpace(name) == "" {
			return errors.New("workflow parameter names cannot be empty")
		}
		if len(parameters[name]) > maxReplayParameterBytes {
			return fmt.Errorf("workflow parameter %q exceeds the %d-byte limit", name, maxReplayParameterBytes)
		}
	}
	return nil
}

func interpolatePromotedAction(action *actionsv1.ActionDefinition, interpolator *state.Interpolator) (*actionsv1.ActionDefinition, string, error) {
	if action == nil || interpolator == nil {
		return nil, "", errors.New("typed action and parameter interpolator are required")
	}
	copy := proto.Clone(action).(*actionsv1.ActionDefinition)
	if nav := copy.GetNavigate(); nav != nil {
		value, err := interpolator.InterpolateStringStrict(nav.GetUrl())
		if err != nil {
			return nil, "", err
		}
		nav.Url = value
		return copy, value, nil
	}
	if input := copy.GetInput(); input != nil {
		value, err := interpolator.InterpolateStringStrict(input.GetValue())
		if err != nil {
			return nil, "", err
		}
		input.Value = value
	}
	return copy, "", nil
}
