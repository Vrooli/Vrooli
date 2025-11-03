package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// StepType represents a supported workflow action.
type StepType string

const (
	StepNavigate     StepType = "navigate"
	StepClick        StepType = "click"
	StepTypeInput    StepType = "type"
	StepShortcut     StepType = "shortcut"
	StepWait         StepType = "wait"
	StepScreenshot   StepType = "screenshot"
	StepExtract      StepType = "extract"
	StepAssert       StepType = "assert"
	StepWorkflowCall StepType = "workflowcall"
	StepCustom       StepType = "custom"
)

var supportedStepTypes = map[StepType]struct{}{
	StepNavigate:     {},
	StepClick:        {},
	StepTypeInput:    {},
	StepShortcut:     {},
	StepWait:         {},
	StepScreenshot:   {},
	StepExtract:      {},
	StepAssert:       {},
	StepWorkflowCall: {},
	StepCustom:       {},
}

// ExecutionPlan represents a validated sequence of steps derived from a workflow definition.
type ExecutionPlan struct {
	WorkflowID   uuid.UUID       `json:"workflow_id"`
	WorkflowName string          `json:"workflow_name"`
	Steps        []ExecutionStep `json:"steps"`
	Metadata     map[string]any  `json:"metadata,omitempty"`
}

// ExecutionStep captures the information required to execute one workflow node.
type ExecutionStep struct {
	Index          int            `json:"index"`
	NodeID         string         `json:"node_id"`
	Type           StepType       `json:"type"`
	Params         map[string]any `json:"params"`
	OutgoingEdges  []EdgeRef      `json:"outgoing_edges"`
	SourcePosition *Position      `json:"source_position,omitempty"`
}

// EdgeRef references an outgoing connection from a node.
type EdgeRef struct {
	ID         string `json:"id"`
	TargetNode string `json:"target_node"`
	Condition  string `json:"condition,omitempty"`
	SourcePort string `json:"source_port,omitempty"`
	TargetPort string `json:"target_port,omitempty"`
}

// Position captures a node's canvas coordinates (if present) to enable deterministic ordering ties.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// CompileWorkflow converts a stored workflow definition into an execution plan.
func CompileWorkflow(workflow *database.Workflow) (*ExecutionPlan, error) {
	if workflow == nil {
		return nil, errors.New("workflow is nil")
	}

	if workflow.FlowDefinition == nil {
		return nil, errors.New("workflow has no flow_definition")
	}

	var raw flowDefinition
	data, err := json.Marshal(workflow.FlowDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow definition: %w", err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}

	if len(raw.Nodes) == 0 {
		return &ExecutionPlan{
			WorkflowID:   workflow.ID,
			WorkflowName: workflow.Name,
			Steps:        []ExecutionStep{},
			Metadata:     map[string]any{},
		}, nil
	}

	compiler := newPlanner(raw)
	steps, err := compiler.buildSteps()
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{}
	if width, height := extractViewportFromSettings(raw.Settings); width > 0 && height > 0 {
		metadata["executionViewport"] = map[string]int{"width": width, "height": height}
	}
	if len(metadata) == 0 {
		metadata = nil
	}

	return &ExecutionPlan{
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		Steps:        steps,
		Metadata:     metadata,
	}, nil
}

// flowDefinition mirrors the React Flow payload persisted with workflows.
type flowDefinition struct {
	Nodes    []rawNode      `json:"nodes"`
	Edges    []rawEdge      `json:"edges"`
	Settings map[string]any `json:"settings"`
}

type rawNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Data     map[string]any `json:"data"`
	Position map[string]any `json:"position"`
}

type rawEdge struct {
	ID           string         `json:"id"`
	Source       string         `json:"source"`
	Target       string         `json:"target"`
	SourceHandle string         `json:"sourceHandle"`
	TargetHandle string         `json:"targetHandle"`
	Data         map[string]any `json:"data"`
}

func toPositiveInt(value any) int {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return int(v)
		}
	case float32:
		if v > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case json.Number:
		if intVal, err := v.Int64(); err == nil && intVal > 0 {
			return int(intVal)
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if parsed, err := strconv.Atoi(trimmed); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func extractViewportFromSettings(settings map[string]any) (int, int) {
	if settings == nil {
		return 0, 0
	}
	viewportValue, ok := settings["executionViewport"]
	if !ok {
		return 0, 0
	}
	viewportMap, ok := viewportValue.(map[string]any)
	if !ok {
		return 0, 0
	}
	width := toPositiveInt(viewportMap["width"])
	height := toPositiveInt(viewportMap["height"])
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	return width, height
}

type planner struct {
	definition    flowDefinition
	nodesByID     map[string]rawNode
	order         map[string]int
	outgoing      map[string][]rawEdge
	incomingCount map[string]int
}

func newPlanner(def flowDefinition) *planner {
	p := &planner{
		definition:    def,
		nodesByID:     make(map[string]rawNode, len(def.Nodes)),
		order:         make(map[string]int, len(def.Nodes)),
		outgoing:      make(map[string][]rawEdge),
		incomingCount: make(map[string]int),
	}

	for idx, node := range def.Nodes {
		p.nodesByID[node.ID] = node
		p.order[node.ID] = idx
	}

	for _, edge := range def.Edges {
		if edge.Source == "" || edge.Target == "" {
			continue
		}
		p.outgoing[edge.Source] = append(p.outgoing[edge.Source], edge)
		p.incomingCount[edge.Target]++
		if _, ok := p.incomingCount[edge.Source]; !ok {
			p.incomingCount[edge.Source] = 0
		}
	}

	for _, node := range def.Nodes {
		if _, ok := p.incomingCount[node.ID]; !ok {
			p.incomingCount[node.ID] = 0
		}
	}

	return p
}

func (p *planner) buildSteps() ([]ExecutionStep, error) {
	type queueItem struct {
		nodeID string
	}

	var queue []queueItem
	for nodeID, indegree := range p.incomingCount {
		if indegree == 0 {
			queue = append(queue, queueItem{nodeID: nodeID})
		}
	}

	sort.Slice(queue, func(i, j int) bool {
		return p.less(queue[i].nodeID, queue[j].nodeID)
	})

	var order []string
	visitedCount := 0

	incoming := make(map[string]int, len(p.incomingCount))
	for k, v := range p.incomingCount {
		incoming[k] = v
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		order = append(order, item.nodeID)
		visitedCount++

		for _, edge := range p.outgoing[item.nodeID] {
			incoming[edge.Target]--
			if incoming[edge.Target] == 0 {
				queue = append(queue, queueItem{nodeID: edge.Target})
			}
		}

		sort.Slice(queue, func(i, j int) bool {
			return p.less(queue[i].nodeID, queue[j].nodeID)
		})
	}

	if visitedCount != len(p.nodesByID) {
		return nil, errors.New("workflow graph contains cycles or disconnected nodes")
	}

	steps := make([]ExecutionStep, 0, len(order))
	for idx, nodeID := range order {
		node := p.nodesByID[nodeID]
		stepType, err := normalizeStepType(node.Type)
		if err != nil {
			return nil, err
		}

		params := make(map[string]any)
		for k, v := range node.Data {
			params[k] = v
		}

		edges := p.outgoing[nodeID]
		edgeRefs := make([]EdgeRef, 0, len(edges))
		for _, edge := range edges {
			edgeRefs = append(edgeRefs, EdgeRef{
				ID:         edge.ID,
				TargetNode: edge.Target,
				Condition:  extractString(edge.Data, "condition", "label"),
				SourcePort: edge.SourceHandle,
				TargetPort: edge.TargetHandle,
			})
		}

		executionStep := ExecutionStep{
			Index:          idx,
			NodeID:         nodeID,
			Type:           stepType,
			Params:         params,
			OutgoingEdges:  edgeRefs,
			SourcePosition: extractPosition(node.Position),
		}
		steps = append(steps, executionStep)
	}

	return steps, nil
}

func (p *planner) less(a, b string) bool {
	oa, ob := p.order[a], p.order[b]
	if oa != ob {
		return oa < ob
	}

	posA := extractPosition(p.nodesByID[a].Position)
	posB := extractPosition(p.nodesByID[b].Position)

	if posA != nil && posB != nil {
		if posA.X != posB.X {
			return posA.X < posB.X
		}
		if posA.Y != posB.Y {
			return posA.Y < posB.Y
		}
	}

	return strings.Compare(a, b) < 0
}

func normalizeStepType(raw string) (StepType, error) {
	normalized := StepType(strings.ToLower(strings.TrimSpace(raw)))
	if normalized == "" {
		return "", fmt.Errorf("node type is required")
	}

	if _, ok := supportedStepTypes[normalized]; !ok {
		return "", fmt.Errorf("unsupported node type: %s", raw)
	}

	return normalized, nil
}

func extractString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			continue
		}
		if val, ok := m[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

func extractPosition(raw map[string]any) *Position {
	if raw == nil {
		return nil
	}

	var xVal, yVal float64
	var xOK, yOK bool

	if v, ok := raw["x"]; ok {
		switch t := v.(type) {
		case float64:
			xVal = t
			xOK = true
		case json.Number:
			if f, err := t.Float64(); err == nil {
				xVal = f
				xOK = true
			}
		}
	}

	if v, ok := raw["y"]; ok {
		switch t := v.(type) {
		case float64:
			yVal = t
			yOK = true
		case json.Number:
			if f, err := t.Float64(); err == nil {
				yVal = f
				yOK = true
			}
		}
	}

	if !xOK || !yOK {
		return nil
	}

	return &Position{X: xVal, Y: yVal}
}
