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
	start := ""
	for _, edge := range workflow.Connections {
		if edge.From == loop.ID && edge.OutputPort == 1 {
			if start != "" {
				return Workflow{}, "", fmt.Errorf("Loop Over Items %s has multiple loop-body entries", loop.Name)
			}
			start = edge.To
		}
	}
	if start == "" {
		return Workflow{}, "", fmt.Errorf("Loop Over Items %s has no loop body", loop.Name)
	}
	nodes := workflow.NodesByID()
	if nodes[start].ID == "" || nodes[feedback.From].ID == "" {
		return Workflow{}, "", fmt.Errorf("Loop Over Items %s references a missing body node", loop.Name)
	}
	seenEdges := map[Connection]bool{}
	reachable := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if len(reachable) > len(workflow.Nodes) {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s body exceeds graph bounds", loop.Name)
		}
		out := 0
		for _, edge := range workflow.Connections {
			if feedbackEdges[edge] || edge.From != current {
				continue
			}
			if seenEdges[edge] {
				return Workflow{}, "", fmt.Errorf("Loop Over Items %s body contains a duplicate edge", loop.Name)
			}
			seenEdges[edge] = true
			if current == feedback.From {
				return Workflow{}, "", fmt.Errorf("Loop Over Items %s feedback target has an unmodeled outgoing edge", loop.Name)
			}
			if edge.To == loop.ID {
				return Workflow{}, "", fmt.Errorf("Loop Over Items %s body contains an unowned cycle", loop.Name)
			}
			out++
			if !reachable[edge.To] {
				reachable[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
		if out == 0 && current != feedback.From {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s has a body branch that does not reach End Loop (%s)", loop.Name, nodes[current].Name)
		}
	}
	if !reachable[feedback.From] {
		return Workflow{}, "", fmt.Errorf("Loop Over Items %s body does not reach its feedback source", loop.Name)
	}
	reaches := map[string]bool{feedback.From: true}
	changed := true
	for changed {
		changed = false
		for _, edge := range workflow.Connections {
			if !feedbackEdges[edge] && reaches[edge.To] && reachable[edge.From] && !reaches[edge.From] {
				reaches[edge.From] = true
				changed = true
			}
		}
	}
	for id := range reachable {
		if !reaches[id] {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s branch from %s does not converge on End Loop", loop.Name, nodes[id].Name)
		}
	}
	for _, edge := range workflow.Connections {
		if feedbackEdges[edge] {
			continue
		}
		if reachable[edge.To] && !reachable[edge.From] && !(edge.From == loop.ID && edge.To == start) {
			return Workflow{}, "", fmt.Errorf("Loop Over Items %s body node %s has an external input", loop.Name, nodes[edge.To].Name)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, edge := range workflow.AIConnections {
			if reachable[edge.To] && !reachable[edge.From] {
				reachable[edge.From] = true
				changed = true
			}
		}
	}
	selected := make([]WorkflowNode, 0, len(reachable))
	for _, node := range workflow.Nodes {
		if reachable[node.ID] {
			selected = append(selected, node)
		}
	}
	body := Workflow{ID: workflow.ID, Name: workflow.Name, Nodes: selected}
	for _, edge := range workflow.Connections {
		if !feedbackEdges[edge] && reachable[edge.From] && reachable[edge.To] {
			body.Connections = append(body.Connections, edge)
		}
	}
	for _, edge := range workflow.AIConnections {
		if reachable[edge.From] && reachable[edge.To] {
			body.AIConnections = append(body.AIConnections, edge)
		}
	}
	if _, err := topologicalOrder(body); err != nil {
		return Workflow{}, "", fmt.Errorf("Loop Over Items %s body is not a bounded DAG: %w", loop.Name, err)
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
