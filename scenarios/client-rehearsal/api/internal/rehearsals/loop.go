package rehearsals

import (
	"fmt"
)

// loopFeedbackEdges recognizes only explicit Split In Batches v3 feedback
// edges: a path must begin at output 1 (loop), reach the source node, and
// return to that same Split In Batches node. All other cycles remain invalid.
func loopFeedbackEdges(workflow Workflow) (map[Connection]bool, error) {
	nodes := workflow.NodesByID()
	feedback := map[Connection]bool{}
	for _, loop := range workflow.Nodes {
		if loop.LoopBatchSize == 0 {
			continue
		}
		var loopOutput *Connection
		for i := range workflow.Connections {
			edge := &workflow.Connections[i]
			if edge.From == loop.ID && edge.OutputPort == 1 {
				if loopOutput != nil {
					return nil, fmt.Errorf("Loop Over Items %s must have one loop-output entry", loop.Name)
				}
				loopOutput = edge
			}
		}
		if loopOutput == nil || nodes[loopOutput.To].ID == "" {
			return nil, fmt.Errorf("Loop Over Items %s must connect output 1 to a body", loop.Name)
		}
		var candidates []Connection
		for _, edge := range workflow.Connections {
			if edge.To == loop.ID && loopPathReaches(workflow, loop.ID, loopOutput.To, edge.From) {
				candidates = append(candidates, edge)
			}
		}
		if len(candidates) != 1 {
			return nil, fmt.Errorf("Loop Over Items %s requires exactly one bounded feedback edge, found %d", loop.Name, len(candidates))
		}
		if candidates[0].OutputPort != 0 || candidates[0].InputPort != 0 {
			return nil, fmt.Errorf("Loop Over Items %s feedback must use main output and input port 0", loop.Name)
		}
		feedback[candidates[0]] = true
	}
	return feedback, nil
}

func loopPathReaches(workflow Workflow, loopID, start, target string) bool {
	if start == target {
		return true
	}
	seen := map[string]bool{}
	queue := []string{start}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		for _, edge := range workflow.Connections {
			if edge.From != id || edge.To == loopID {
				continue
			}
			if edge.To == target {
				return true
			}
			queue = append(queue, edge.To)
		}
	}
	return false
}

func linearLoopBody(workflow Workflow, loop WorkflowNode) (Workflow, string, error) {
	feedbackEdges, err := loopFeedbackEdges(workflow)
	if err != nil {
		return Workflow{}, "", err
	}
	var feedback Connection
	for edge := range feedbackEdges {
		if edge.To == loop.ID {
			feedback = edge
			break
		}
	}
	if feedback.From == "" {
		return Workflow{}, "", fmt.Errorf("Loop Over Items %s has no feedback edge", loop.Name)
	}
	var start string
	for _, edge := range workflow.Connections {
		if edge.From == loop.ID && edge.OutputPort == 1 {
			start = edge.To
			break
		}
	}
	if start == "" {
		return Workflow{}, "", fmt.Errorf("Loop Over Items %s has no loop body", loop.Name)
	}
	nodesByID := workflow.NodesByID()
	selected := []WorkflowNode{}
	selectedIDs := map[string]bool{}
	current := start
	for steps := 0; ; steps++ {
		if steps >= len(workflow.Nodes) || selectedIDs[current] {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s body is not a bounded linear path", loop.Name)
		}
		node, ok := nodesByID[current]
		if !ok {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s body references missing node %s", loop.Name, current)
		}
		selected = append(selected, node)
		selectedIDs[current] = true
		if current == feedback.From {
			break
		}
		var next []Connection
		for _, edge := range workflow.Connections {
			if feedbackEdges[edge] || edge.From != current {
				continue
			}
			next = append(next, edge)
		}
		if len(next) != 1 || next[0].OutputPort != 0 || next[0].InputPort != 0 {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s supports a single linear body path; node %s branches or uses an unmodeled port", loop.Name, node.Name)
		}
		current = next[0].To
	}
	for _, edge := range workflow.Connections {
		if feedbackEdges[edge] {
			continue
		}
		if selectedIDs[edge.To] && edge.To != start && !selectedIDs[edge.From] {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s body node %s has an external input", loop.Name, nodesByID[edge.To].Name)
		}
	}
	body := Workflow{ID: workflow.ID, Name: workflow.Name, Nodes: selected}
	for _, edge := range workflow.Connections {
		if !feedbackEdges[edge] && selectedIDs[edge.From] && selectedIDs[edge.To] {
			body.Connections = append(body.Connections, edge)
		}
	}
	return body, start, nil
}

func loopFeedbackSource(workflow Workflow, loopID string) string {
	feedbackEdges, err := loopFeedbackEdges(workflow)
	if err != nil {
		return ""
	}
	for edge := range feedbackEdges {
		if edge.To == loopID {
			return edge.From
		}
	}
	return ""
}
