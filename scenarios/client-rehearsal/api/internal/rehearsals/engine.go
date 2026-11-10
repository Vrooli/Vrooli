package rehearsals

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxSubworkflowDepth = 16

// SyntheticEngine evaluates a validated acyclic workflow graph. All HTTP
// operations go through the injected simulator; the interpreter has no HTTP
// client, runtime endpoint, or credential surface.
type SyntheticEngine struct{}

func (SyntheticEngine) Execute(ctx context.Context, workflow Workflow, triggerNodeID string, simulator Simulator, injectFault bool, now time.Time) ([]TraceEvent, error) {
	trace := make([]TraceEvent, 0, len(workflow.Nodes)*2)
	if len(workflow.EntryNodeIDs) == 0 {
		return trace, fmt.Errorf("workflow %s has no externally invokable trigger", workflow.ID)
	}
	if triggerNodeID == "" && len(workflow.EntryNodeIDs) == 1 {
		triggerNodeID = workflow.EntryNodeIDs[0]
	}
	entry := ""
	for _, candidate := range workflow.EntryNodeIDs {
		if candidate == triggerNodeID {
			entry = candidate
			break
		}
	}
	if entry == "" {
		return trace, ErrInvalid{Field: "trigger_node_id", Reason: "must identify one of the imported workflow's trigger nodes"}
	}
	if err := ctx.Err(); err != nil {
		return trace, err
	}
	input := syntheticTriggerInput(workflow.NodesByID()[entry].Trigger)
	if _, err := executeWorkflowEntry(ctx, workflow, entry, input, simulator, injectFault, now, &trace, map[string]bool{}, 0); err != nil {
		return trace, err
	}
	return trace, nil
}

func executeWorkflowEntry(ctx context.Context, workflow Workflow, entryID string, input map[string]any, simulator Simulator, injectFault bool, now time.Time, trace *[]TraceEvent, stack map[string]bool, depth int) (map[string]any, error) {
	output, runErr := executeGraph(ctx, workflow, entryID, input, simulator, injectFault, now, trace, stack, depth)
	if runErr == nil || workflow.ErrorWorkflowID == "" {
		return output, runErr
	}
	handler := workflow.Subworkflows[workflow.ErrorWorkflowID]
	if handler == nil || handler.ErrorNodeID == "" {
		return output, fmt.Errorf("workflow %s failed: %w; configured error workflow %s is not executable", workflow.ID, runErr, workflow.ErrorWorkflowID)
	}
	errorInput := map[string]any{"error": map[string]any{"message": runErr.Error(), "workflowId": workflow.ID}, "execution": map[string]any{"status": "failed"}}
	handlerStack := cloneSet(stack)
	handlerStack[workflow.ID] = true
	if _, handlerErr := executeGraph(ctx, *handler, handler.ErrorNodeID, errorInput, simulator, false, now, trace, handlerStack, depth+1); handlerErr != nil {
		return output, fmt.Errorf("workflow %s failed: %w; error workflow %s failed: %v", workflow.ID, runErr, handler.ID, handlerErr)
	}
	return output, runErr
}

func executeGraph(ctx context.Context, workflow Workflow, entryID string, initial map[string]any, simulator Simulator, injectFault bool, now time.Time, trace *[]TraceEvent, stack map[string]bool, depth int) (map[string]any, error) {
	if depth > maxSubworkflowDepth {
		return nil, fmt.Errorf("subworkflow nesting exceeds %d", maxSubworkflowDepth)
	}
	if workflow.ID != "" && stack[workflow.ID] {
		return nil, fmt.Errorf("recursive workflow reference detected at %s", workflow.ID)
	}
	stack = cloneSet(stack)
	stack[workflow.ID] = true
	nodes := workflow.NodesByID()
	if nodes[entryID].ID == "" {
		return nil, fmt.Errorf("workflow %s entry node %s is missing", workflow.ID, entryID)
	}
	order, err := topologicalOrder(workflow)
	if err != nil {
		return nil, err
	}
	inputs := map[string][]map[string]any{entryID: {initial}}
	var last map[string]any
	for _, id := range order {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		node := nodes[id]
		var input map[string]any
		if node.Trigger != "" {
			if node.ID != entryID {
				continue
			}
			input = cloneMap(initial)
		} else {
			incoming := inputs[id]
			if len(incoming) == 0 {
				continue
			}
			input = mergeMaps(incoming)
		}
		outputPort := 0
		output := cloneMap(input)
		operation := node.Name
		switch {
		case node.Request != nil:
			response, err := invokeWithRetry(ctx, simulator, *node.Request, injectFault, now, trace, workflow.ID, node.Name)
			if err != nil {
				return last, err
			}
			output = mergeMaps([]map[string]any{input, decodeObject(response)})
			operation = node.Request.Operation
		case node.Condition != nil:
			matches := evaluateCondition(*node.Condition, input)
			if !matches {
				outputPort = 1
			}
			operation = "If " + node.Condition.Path + " " + node.Condition.Operator
		case node.OutputPath != "":
			output = projectedValue(input, node.OutputPath)
			operation = "Set JSON projection"
		case len(node.SwitchRules) > 0:
			outputPort = -1
			for _, rule := range node.SwitchRules {
				if evaluateCondition(rule.Condition, input) {
					outputPort = rule.OutputPort
					break
				}
			}
			operation = "Switch"
		case node.SubworkflowID != "":
			child := workflow.Subworkflows[node.SubworkflowID]
			if child == nil {
				return last, fmt.Errorf("subworkflow %s is not in the import bundle", node.SubworkflowID)
			}
			entry := child.SubworkflowEntryID
			if entry == "" {
				return last, fmt.Errorf("subworkflow %s has no Execute Workflow Trigger", child.ID)
			}
			nested, err := executeWorkflowEntry(ctx, *child, entry, input, simulator, injectFault, now, trace, stack, depth+1)
			if err != nil {
				return last, err
			}
			output = mergeMaps([]map[string]any{input, nested})
			operation = "Execute Workflow " + child.ID
		case node.Trigger == TriggerError:
			operation = "Error Trigger"
		case node.Trigger == TriggerSubworkflow:
			operation = "Execute Workflow Trigger"
		case node.Type == "n8n-nodes-base.respondToWebhook":
			operation = "Respond to Webhook"
		case node.Type == "n8n-nodes-base.stickyNote":
			operation = "Non-executable Note"
		case node.Trigger != "":
			operation = string(node.Trigger) + " Trigger"
		case node.Type == "n8n-nodes-base.noOp":
			operation = "No Operation"
		}
		*trace = append(*trace, TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflow.ID + "/" + operation, RequestSummary: node.Name, ResponseSummary: "synthetic dataflow", Status: "succeeded", OccurredAt: now.UTC()})
		last = output
		for _, edge := range workflow.Connections {
			if edge.From == id && edge.OutputPort == outputPort {
				inputs[edge.To] = append(inputs[edge.To], cloneMap(output))
			}
		}
	}
	return last, nil
}

func projectedValue(input map[string]any, path string) map[string]any {
	var value any = input
	for _, component := range strings.Split(path, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return map[string]any{}
		}
		value, ok = object[component]
		if !ok {
			return map[string]any{}
		}
	}
	if object, ok := value.(map[string]any); ok {
		return cloneMap(object)
	}
	return map[string]any{"value": value}
}

func invokeWithRetry(ctx context.Context, simulator Simulator, step Step, injectFault bool, now time.Time, trace *[]TraceEvent, workflowID, nodeName string) (string, error) {
	request := step.Method + " " + step.Host + step.URL
	for attempt := uint32(1); attempt <= 2; attempt++ {
		response, callErr := simulator.Invoke(ctx, step, attempt, injectFault)
		event := TraceEvent{Sequence: uint32(len(*trace) + 1), Operation: workflowID + "/" + nodeName + ": " + step.Operation, RequestSummary: request, ResponseSummary: response, OccurredAt: now.UTC()}
		if callErr != nil {
			event.Status = "failed"
			event.ResponseSummary = callErr.Error()
			*trace = append(*trace, event)
			if attempt == 2 {
				return "", fmt.Errorf("operation %s failed after retry: %w", step.Operation, callErr)
			}
			continue
		}
		event.Status = "succeeded"
		*trace = append(*trace, event)
		return response, nil
	}
	return "", fmt.Errorf("operation %s did not produce a result", step.Operation)
}

func topologicalOrder(workflow Workflow) ([]string, error) {
	indegree := make(map[string]int, len(workflow.Nodes))
	outgoing := make(map[string][]string, len(workflow.Nodes))
	for _, edge := range workflow.Connections {
		indegree[edge.To]++
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
	}
	orderIndex := make(map[string]int, len(workflow.Nodes))
	ready := make([]string, 0, len(workflow.Nodes))
	for index, node := range workflow.Nodes {
		orderIndex[node.ID] = index
		if indegree[node.ID] == 0 {
			ready = append(ready, node.ID)
		}
	}
	less := func(ids []string) {
		sort.Slice(ids, func(i, j int) bool { return orderIndex[ids[i]] < orderIndex[ids[j]] })
	}
	less(ready)
	order := make([]string, 0, len(workflow.Nodes))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, next := range outgoing[id] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
				less(ready)
			}
		}
	}
	if len(order) != len(workflow.Nodes) {
		return nil, fmt.Errorf("workflow %s contains a graph cycle", workflow.ID)
	}
	return order, nil
}

func (workflow Workflow) NodesByID() map[string]WorkflowNode {
	result := make(map[string]WorkflowNode, len(workflow.Nodes))
	for _, node := range workflow.Nodes {
		result[node.ID] = node
	}
	return result
}

func syntheticTriggerInput(kind TriggerKind) map[string]any {
	return map[string]any{"id": "synthetic-001", "status": "active", "source": string(kind), "email": "avery@example.invalid", "body": map[string]any{"action": "edited", "id": "synthetic-001"}}
}

func evaluateCondition(condition Condition, input map[string]any) bool {
	var value any = input
	for _, component := range strings.Split(condition.Path, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		value = object[component]
	}
	left := fmt.Sprint(value)
	right := fmt.Sprint(condition.Value)
	switch condition.Operator {
	case "equals":
		return left == right
	case "notEquals":
		return left != right
	case "contains":
		return strings.Contains(left, right)
	case "gt", "gte", "lt", "lte":
		l, lErr := strconv.ParseFloat(left, 64)
		r, rErr := strconv.ParseFloat(right, 64)
		if lErr != nil || rErr != nil {
			return false
		}
		switch condition.Operator {
		case "gt":
			return l > r
		case "gte":
			return l >= r
		case "lt":
			return l < r
		default:
			return l <= r
		}
	default:
		return false
	}
}

func mergeMaps(items []map[string]any) map[string]any {
	result := map[string]any{}
	for _, item := range items {
		for key, value := range item {
			result[key] = value
		}
	}
	return result
}

func cloneMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func cloneSet(input map[string]bool) map[string]bool {
	result := make(map[string]bool, len(input)+1)
	for key, value := range input {
		result[key] = value
	}
	return result
}

func decodeObject(raw string) map[string]any {
	result := map[string]any{}
	if json.Unmarshal([]byte(raw), &result) != nil {
		return result
	}
	return result
}
