package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
)

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
	SourceIndex    int            `json:"source_index"`
	NodeID         string         `json:"node_id"`
	Type           StepType       `json:"type"`
	Params         map[string]any `json:"params"`
	OutgoingEdges  []EdgeRef      `json:"outgoing_edges"`
	SourcePosition *Position      `json:"source_position,omitempty"`
	LoopPlan       *ExecutionPlan `json:"loop_plan,omitempty"`
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
	log.Printf("[COMPILER_DEBUG] CompileWorkflow called for workflow: %s (ID: %s)", workflow.Name, workflow.ID)

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
	log.Printf("[COMPILER_DEBUG] Workflow definition marshalled, size: %d bytes", len(data))

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}
	log.Printf("[COMPILER_DEBUG] Workflow definition parsed: %d nodes, %d edges", len(raw.Nodes), len(raw.Edges))

	if len(raw.Nodes) == 0 {
		return &ExecutionPlan{
			WorkflowID:   workflow.ID,
			WorkflowName: workflow.Name,
			Steps:        []ExecutionStep{},
			Metadata:     map[string]any{},
		}, nil
	}

	log.Printf("[COMPILER_DEBUG] Calling compileFlow for workflow: %s", workflow.Name)
	plan, err := compileFlow(flowFragment{definition: raw}, workflow.ID, workflow.Name)
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{}
	if width, height := extractViewportFromSettings(raw.Settings); width > 0 && height > 0 {
		metadata["executionViewport"] = map[string]any{"width": width, "height": height}
	}
	if selector, timeout := extractEntryFromSettings(raw.Settings); selector != "" {
		metadata["entrySelector"] = selector
		if timeout > 0 {
			metadata["entrySelectorTimeoutMs"] = timeout
		}
	}
	if len(metadata) == 0 {
		metadata = nil
	}
	plan.Metadata = metadata

	return plan, nil
}

func compileFlow(fragment flowFragment, workflowID uuid.UUID, workflowName string) (*ExecutionPlan, error) {
	planner := newPlanner(fragment.definition)
	loopFragments, err := planner.extractLoopBodies()
	if err != nil {
		return nil, err
	}

	steps, err := planner.buildSteps()
	if err != nil {
		return nil, err
	}

	// Inline subflow nodes before processing loops
	steps, err = inlineSubflows(steps, workflowID, workflowName)
	if err != nil {
		return nil, err
	}

	for idx := range steps {
		if steps[idx].Type != StepLoop {
			continue
		}
		bodyFragment, ok := loopFragments[steps[idx].NodeID]
		if !ok {
			return nil, fmt.Errorf("loop node %s has no body definition", steps[idx].NodeID)
		}
		childPlan, err := compileFlow(bodyFragment, workflowID, fmt.Sprintf("%s::%s", workflowName, steps[idx].NodeID))
		if err != nil {
			return nil, err
		}
		steps[idx].LoopPlan = childPlan
	}

	plan := &ExecutionPlan{
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		Steps:        steps,
	}

	if len(fragment.specialEdges) > 0 {
		applySpecialEdges(plan, fragment.specialEdges)
	}

	return plan, nil
}

func applySpecialEdges(plan *ExecutionPlan, special map[string][]EdgeRef) {
	if plan == nil || len(special) == 0 {
		return
	}
	index := make(map[string]*ExecutionStep, len(plan.Steps))
	for i := range plan.Steps {
		step := &plan.Steps[i]
		index[step.NodeID] = step
	}
	for nodeID, edges := range special {
		step, ok := index[nodeID]
		if !ok {
			continue
		}
		step.OutgoingEdges = append(step.OutgoingEdges, edges...)
	}
}

// inlineSubflows replaces subflow nodes with their expanded workflow definitions.
// The Python resolver (resolve-workflow.py) converts @fixture/ references into
// workflowDefinition objects. This function flattens those nested definitions
// into the parent execution plan while preserving edge connections.
func inlineSubflows(steps []ExecutionStep, workflowID uuid.UUID, workflowName string) ([]ExecutionStep, error) {
	result := make([]ExecutionStep, 0, len(steps))

	log.Printf("[SUBFLOW_DEBUG] inlineSubflows called with %d steps", len(steps))

	for _, step := range steps {
		if step.Type != StepSubflow {
			result = append(result, step)
			continue
		}

		log.Printf("[SUBFLOW_DEBUG] Found subflow node: %s", step.NodeID)
		log.Printf("[SUBFLOW_DEBUG] step.Params keys: %v", func() []string {
			keys := make([]string, 0, len(step.Params))
			for k := range step.Params {
				keys = append(keys, k)
			}
			return keys
		}())

		// Extract workflowDefinition from params
		workflowDef, ok := step.Params["workflowDefinition"].(map[string]any)
		if !ok {
			log.Printf("[SUBFLOW_DEBUG] NO workflowDefinition in params (or wrong type)! Type: %T", step.Params["workflowDefinition"])
			// Subflow without workflowDefinition - skip it (Python resolver did not inline it)
			// This happens when resolve-workflow.py is not used or when testing pre-resolved workflows
			continue
		}

		log.Printf("[SUBFLOW_DEBUG] workflowDefinition found with %d top-level keys", len(workflowDef))

		// Parse the nested workflow definition
		defData, err := json.Marshal(workflowDef)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal subflow definition for node %s: %w", step.NodeID, err)
		}

		var nestedFlow flowDefinition
		if err := json.Unmarshal(defData, &nestedFlow); err != nil {
			return nil, fmt.Errorf("failed to parse subflow definition for node %s: %w", step.NodeID, err)
		}

		log.Printf("[SUBFLOW_DEBUG] Parsed nestedFlow with %d nodes", len(nestedFlow.Nodes))

		// Compile the nested workflow into steps
		nestedPlan, err := compileFlow(
			flowFragment{definition: nestedFlow},
			workflowID,
			fmt.Sprintf("%s::%s", workflowName, step.NodeID),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to compile subflow %s: %w", step.NodeID, err)
		}

		log.Printf("[SUBFLOW_DEBUG] Compiled nestedPlan with %d steps", len(nestedPlan.Steps))

		// Inline the nested steps, preserving the subflow node's outgoing edges
		// by transferring them to the last step of the nested plan
		if len(nestedPlan.Steps) == 0 {
			// Empty subflow - skip it
			continue
		}

		// Add all nested steps except the last one
		for i := 0; i < len(nestedPlan.Steps)-1; i++ {
			result = append(result, nestedPlan.Steps[i])
		}

		// Add the last nested step and transfer the subflow's outgoing edges to it
		lastStep := nestedPlan.Steps[len(nestedPlan.Steps)-1]
		lastStep.OutgoingEdges = append(lastStep.OutgoingEdges, step.OutgoingEdges...)
		result = append(result, lastStep)
	}

	// Reindex all steps to maintain sequential order
	for i := range result {
		result[i].Index = i
	}

	return result, nil
}

// flowDefinition mirrors the React Flow payload persisted with workflows.
type flowDefinition struct {
	Nodes    []rawNode      `json:"nodes"`
	Edges    []rawEdge      `json:"edges"`
	Settings map[string]any `json:"settings"`
}

type flowFragment struct {
	definition   flowDefinition
	specialEdges map[string][]EdgeRef
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

func extractEntryFromSettings(settings map[string]any) (string, int) {
	if settings == nil {
		return "", 0
	}
	raw, ok := settings["entrySelector"]
	if !ok {
		return "", 0
	}
	selector, ok := raw.(string)
	if !ok || strings.TrimSpace(selector) == "" {
		return "", 0
	}
	timeout := toPositiveInt(settings["entrySelectorTimeoutMs"])
	if timeout == 0 {
		timeout = toPositiveInt(settings["entryTimeoutMs"])
	}
	return strings.TrimSpace(selector), timeout
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

func (p *planner) extractLoopBodies() (map[string]flowFragment, error) {
	loopFragments := make(map[string]flowFragment)
	assigned := make(map[string]string)

	for _, node := range p.definition.Nodes {
		stepType, err := normalizeStepType(node.Type)
		if err != nil {
			return nil, err
		}
		if stepType != StepLoop {
			continue
		}
		entries := p.loopEntryTargets(node.ID)
		if len(entries) == 0 {
			return nil, fmt.Errorf("loop node %s requires at least one body connection", node.ID)
		}
		bodyNodes, bodyEdges, specialEdges, err := p.collectLoopBody(node.ID, entries, assigned)
		if err != nil {
			return nil, err
		}
		fragment := flowFragment{
			definition: flowDefinition{
				Nodes: rawNodeMapToSlice(bodyNodes),
				Edges: bodyEdges,
			},
			specialEdges: specialEdges,
		}
		loopFragments[node.ID] = fragment
	}

	if len(assigned) == 0 {
		return loopFragments, nil
	}

	if err := p.pruneLoopBodies(assigned); err != nil {
		return nil, err
	}

	return loopFragments, nil
}

func (p *planner) loopEntryTargets(loopNodeID string) []string {
	edges := p.outgoing[loopNodeID]
	targets := make([]string, 0)
	for _, edge := range edges {
		if isLoopBodyEdge(edge) {
			targets = append(targets, edge.Target)
		}
	}
	return uniqueStrings(targets)
}

func (p *planner) collectLoopBody(loopNodeID string, entryTargets []string, assigned map[string]string) (map[string]rawNode, []rawEdge, map[string][]EdgeRef, error) {
	bodyNodes := make(map[string]rawNode)
	specialEdges := make(map[string][]EdgeRef)
	queue := append([]string{}, entryTargets...)

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			continue
		}
		if nodeID == loopNodeID {
			return nil, nil, nil, fmt.Errorf("loop node %s cannot include itself as part of the body", loopNodeID)
		}
		if owner, taken := assigned[nodeID]; taken && owner != loopNodeID {
			return nil, nil, nil, fmt.Errorf("node %s already belongs to loop %s", nodeID, owner)
		}
		if _, exists := bodyNodes[nodeID]; exists {
			continue
		}
		rawNode, ok := p.nodesByID[nodeID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("loop node %s references missing node %s", loopNodeID, nodeID)
		}
		bodyNodes[nodeID] = rawNode
		assigned[nodeID] = loopNodeID

		for _, edge := range p.outgoing[nodeID] {
			if edge.Target == loopNodeID {
				directive, ok := loopDirectiveFromEdge(edge)
				if !ok {
					return nil, nil, nil, fmt.Errorf("loop node %s has invalid return edge from %s; use loopContinue/loopBreak handles", loopNodeID, nodeID)
				}
				specialEdges[nodeID] = append(specialEdges[nodeID], directive)
				continue
			}
			if owner, taken := assigned[edge.Target]; taken && owner != loopNodeID {
				return nil, nil, nil, fmt.Errorf("node %s cannot be shared across loops (already assigned to %s)", edge.Target, owner)
			}
			queue = append(queue, edge.Target)
		}
	}

	for nodeID := range bodyNodes {
		incoming := p.findIncomingEdges(nodeID)
		for _, edge := range incoming {
			if edge.Source == loopNodeID && isLoopBodyEdge(edge) {
				continue
			}
			if _, ok := bodyNodes[edge.Source]; ok {
				continue
			}
			return nil, nil, nil, fmt.Errorf("node %s inside loop %s receives edges from outside the loop body", nodeID, loopNodeID)
		}
	}

	bodyEdges := make([]rawEdge, 0)
	for source := range bodyNodes {
		for _, edge := range p.outgoing[source] {
			if _, ok := bodyNodes[edge.Target]; ok {
				bodyEdges = append(bodyEdges, edge)
			}
		}
	}

	return bodyNodes, bodyEdges, specialEdges, nil
}

func (p *planner) pruneLoopBodies(assigned map[string]string) error {
	if len(assigned) == 0 {
		return nil
	}

	for nodeID := range assigned {
		delete(p.nodesByID, nodeID)
		delete(p.outgoing, nodeID)
		delete(p.incomingCount, nodeID)
	}

	filteredOutgoing := make(map[string][]rawEdge, len(p.outgoing))
	for source, edges := range p.outgoing {
		if _, removed := assigned[source]; removed {
			continue
		}
		filtered := make([]rawEdge, 0, len(edges))
		for _, edge := range edges {
			if _, removed := assigned[edge.Target]; removed {
				continue
			}
			if isLoopBodyEdge(edge) {
				continue
			}
			filtered = append(filtered, edge)
		}
		filteredOutgoing[source] = filtered
	}
	p.outgoing = filteredOutgoing

	newIncoming := make(map[string]int, len(p.nodesByID))
	for nodeID := range p.nodesByID {
		incoming := p.findIncomingEdges(nodeID)
		count := 0
		for _, edge := range incoming {
			if isLoopBodyEdge(edge) {
				continue
			}
			count++
		}
		newIncoming[nodeID] = count
	}
	p.incomingCount = newIncoming
	p.definition.Nodes = rawNodeMapToSlice(p.nodesByID)
	rebuiltEdges := make([]rawEdge, 0)
	for _, edges := range p.outgoing {
		rebuiltEdges = append(rebuiltEdges, edges...)
	}
	p.definition.Edges = rebuiltEdges
	return nil
}

func (p *planner) findIncomingEdges(nodeID string) []rawEdge {
	incoming := make([]rawEdge, 0)
	for _, edges := range p.outgoing {
		for _, edge := range edges {
			if edge.Target == nodeID {
				incoming = append(incoming, edge)
			}
		}
	}
	return incoming
}

func (p *planner) buildSteps() ([]ExecutionStep, error) {
	order := p.topologicalOrder()
	if len(order) != len(p.definition.Nodes) {
		return nil, errors.New("workflow contains a cycle or disconnected nodes")
	}

	steps := make([]ExecutionStep, 0, len(order))
	for idx, nodeID := range order {
		node := p.nodesByID[nodeID]
		stepType, err := normalizeStepType(node.Type)
		if err != nil {
			return nil, err
		}
		step := ExecutionStep{
			Index:       idx,
			SourceIndex: p.order[nodeID],
			NodeID:      node.ID,
			Type:        stepType,
			Params:      copyMap(node.Data),
		}
		if pos := toPosition(node.Position); pos != nil {
			step.SourcePosition = pos
		}

		// Resolve navigate node URLs from destinationType: "scenario" format
		if stepType == StepNavigate {
			if err := resolveNavigateURL(&step); err != nil {
				return nil, fmt.Errorf("failed to resolve navigate URL for node %s: %w", nodeID, err)
			}
		}

		// Normalize assert node params (assertMode -> mode)
		if stepType == StepAssert {
			if err := normalizeAssertParams(&step); err != nil {
				return nil, fmt.Errorf("failed to normalize assert params for node %s: %w", nodeID, err)
			}
		}

		// Resolve @selector/ references in all steps that have selector parameters
		if err := resolveSelectors(&step); err != nil {
			return nil, fmt.Errorf("failed to resolve selectors for node %s: %w", nodeID, err)
		}

		for _, edge := range p.outgoing[nodeID] {
			if isLoopBodyEdge(edge) {
				continue
			}
			step.OutgoingEdges = append(step.OutgoingEdges, EdgeRef{
				ID:         edge.ID,
				TargetNode: edge.Target,
				Condition:  strings.TrimSpace(edgeCondition(edge)),
				SourcePort: strings.TrimSpace(edge.SourceHandle),
				TargetPort: strings.TrimSpace(edge.TargetHandle),
			})
		}
		steps = append(steps, step)
	}

	return steps, nil
}

func (p *planner) topologicalOrder() []string {
	incoming := make(map[string]int, len(p.incomingCount))
	for k, v := range p.incomingCount {
		incoming[k] = v
	}

	queue := make([]string, 0)
	for nodeID, count := range incoming {
		if count == 0 {
			queue = append(queue, nodeID)
		}
	}
	sort.Slice(queue, func(i, j int) bool {
		return p.order[queue[i]] < p.order[queue[j]]
	})

	order := make([]string, 0, len(p.definition.Nodes))
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		order = append(order, nodeID)

		for _, edge := range p.outgoing[nodeID] {
			if isLoopBodyEdge(edge) {
				continue
			}
			incoming[edge.Target]--
			if incoming[edge.Target] == 0 {
				queue = append(queue, edge.Target)
			}
		}
		sort.Slice(queue, func(i, j int) bool {
			return p.order[queue[i]] < p.order[queue[j]]
		})
	}

	return order
}

func normalizeStepType(raw string) (StepType, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("step type cannot be empty")
	}
	stepType := StepType(trimmed)
	if _, ok := supportedStepTypes[stepType]; !ok {
		return "", fmt.Errorf("unsupported step type: %s", stepType)
	}
	return stepType, nil
}

func copyMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func toPosition(pos map[string]any) *Position {
	if pos == nil {
		return nil
	}
	x := toPositiveFloat(pos["x"])
	y := toPositiveFloat(pos["y"])
	if x == 0 && y == 0 {
		return nil
	}
	return &Position{X: x, Y: y}
}

func toPositiveFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return v
		}
	case float32:
		if v > 0 {
			return float64(v)
		}
	case int:
		if v > 0 {
			return float64(v)
		}
	case int64:
		if v > 0 {
			return float64(v)
		}
	case json.Number:
		if fVal, err := v.Float64(); err == nil && fVal > 0 {
			return fVal
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0
		}
		if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func rawNodeMapToSlice(m map[string]rawNode) []rawNode {
	if len(m) == 0 {
		return nil
	}
	result := make([]rawNode, 0, len(m))
	for _, node := range m {
		result = append(result, node)
	}
	return result
}

func isLoopBodyEdge(edge rawEdge) bool {
	return strings.EqualFold(strings.TrimSpace(edge.SourceHandle), loopHandleBody) ||
		strings.EqualFold(strings.TrimSpace(edge.TargetHandle), loopConditionBody)
}

func loopDirectiveFromEdge(edge rawEdge) (EdgeRef, bool) {
	handle := strings.ToLower(strings.TrimSpace(edge.TargetHandle))
	switch handle {
	case loopHandleContinue, loopConditionContinue:
		return EdgeRef{ID: edge.ID, TargetNode: LoopContinueTarget, Condition: loopConditionContinue}, true
	case loopHandleBreak, loopConditionBreak:
		return EdgeRef{ID: edge.ID, TargetNode: LoopBreakTarget, Condition: loopConditionBreak}, true
	case loopHandleAfter, loopConditionAfter:
		return EdgeRef{ID: edge.ID, TargetNode: loopHandleAfter, Condition: loopConditionAfter}, true
	default:
		return EdgeRef{}, false
	}
}

func edgeCondition(edge rawEdge) string {
	if edge.Data == nil {
		return ""
	}
	if cond, ok := edge.Data["condition"]; ok {
		if s, ok := cond.(string); ok {
			return s
		}
	}
	return ""
}

// resolveNavigateURL resolves destination URLs for navigate nodes with destinationType: "scenario"
func resolveNavigateURL(step *ExecutionStep) error {
	if step == nil || step.Type != StepNavigate || step.Params == nil {
		return nil
	}

	// If URL is already set, no resolution needed
	if url, ok := step.Params["url"].(string); ok && strings.TrimSpace(url) != "" {
		return nil
	}

	// Check if this is a scenario destination
	destinationType, _ := step.Params["destinationType"].(string)
	if strings.TrimSpace(destinationType) != "scenario" {
		// Not a scenario destination, no resolution needed
		return nil
	}

	// Extract scenario name and path
	scenarioName, _ := step.Params["scenario"].(string)
	scenarioPath, _ := step.Params["scenarioPath"].(string)

	scenarioName = strings.TrimSpace(scenarioName)
	if scenarioName == "" {
		return fmt.Errorf("navigate node with destinationType 'scenario' missing scenario name")
	}

	// Resolve URL via scenarioport package
	ctx := context.Background()
	resolvedURL, _, err := scenarioport.ResolveURL(ctx, scenarioName, scenarioPath)
	if err != nil {
		return fmt.Errorf("failed to resolve URL for scenario %s: %w", scenarioName, err)
	}

	// Set the resolved URL in params
	step.Params["url"] = resolvedURL

	return nil
}

// selectorManifest holds the loaded selector mappings from selectors.manifest.json
var selectorManifest map[string]interface{}
var selectorManifestOnce sync.Once
var selectorManifestErr error

// loadSelectorManifest loads the selector manifest from ui/src/consts/selectors.manifest.json
func loadSelectorManifest() error {
	selectorManifestOnce.Do(func() {
		// Try multiple paths to find the manifest, depending on working directory
		// (working directory could be scenario root or api/ subdirectory)
		manifestPaths := []string{
			"ui/src/consts/selectors.manifest.json",      // When running from scenario root
			"../ui/src/consts/selectors.manifest.json",   // When running from api/ subdirectory
		}

		var data []byte
		var err error

		for _, path := range manifestPaths {
			data, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}

		if err != nil {
			selectorManifestErr = fmt.Errorf("failed to read selector manifest (tried: %v): %w", manifestPaths, err)
			return
		}

		// Parse JSON
		var manifest map[string]interface{}
		if err := json.Unmarshal(data, &manifest); err != nil {
			selectorManifestErr = fmt.Errorf("failed to parse selector manifest: %w", err)
			return
		}

		selectorManifest = manifest
	})

	return selectorManifestErr
}

// normalizeAssertParams normalizes assert node parameters from workflow format to playwright-driver format
// Specifically: assertMode -> mode
func normalizeAssertParams(step *ExecutionStep) error {
	if step == nil || step.Type != StepAssert || step.Params == nil {
		return nil
	}

	// Map assertMode to mode (playwright-driver expects "mode")
	if assertMode, ok := step.Params["assertMode"].(string); ok && assertMode != "" {
		step.Params["mode"] = assertMode
		// Keep assertMode for backward compatibility, but mode takes precedence
	}

	return nil
}

// resolveSelectors resolves @selector/ references in step parameters to actual CSS selectors
func resolveSelectors(step *ExecutionStep) error {
	log.Printf("[SELECTOR_RESOLVE] resolveSelectors called for step type=%s nodeID=%s", step.Type, step.NodeID)
	if step == nil || step.Params == nil {
		return nil
	}

	// Check if there are any @selector/ references before loading manifest
	// This avoids unnecessary file I/O and allows workflows with pre-resolved selectors to work
	hasSelectorsRefs := false
	selectorParams := []string{"selector", "successSelector", "failureSelector", "sourceSelector", "targetSelector"}

	for _, paramName := range selectorParams {
		if selectorRef, ok := step.Params[paramName].(string); ok {
			log.Printf("[SELECTOR_RESOLVE] checking param=%s value='%s' hasPrefix=%v", paramName, selectorRef, strings.HasPrefix(selectorRef, "@selector/"))
			if strings.HasPrefix(selectorRef, "@selector/") {
				hasSelectorsRefs = true
				break
			}
		}
	}
	log.Printf("[SELECTOR_RESOLVE] hasSelectorsRefs=%v (will %s manifest loading)", hasSelectorsRefs, map[bool]string{true: "proceed with", false: "skip"}[hasSelectorsRefs])

	if !hasSelectorsRefs {
		if resilience, ok := step.Params["resilience"].(map[string]interface{}); ok {
			if successSelector, ok := resilience["successSelector"].(string); ok && strings.HasPrefix(successSelector, "@selector/") {
				hasSelectorsRefs = true
			}
			if !hasSelectorsRefs {
				if failureSelector, ok := resilience["failureSelector"].(string); ok && strings.HasPrefix(failureSelector, "@selector/") {
					hasSelectorsRefs = true
				}
			}
		}
	}

	// Only load manifest if we found @selector/ references
	if !hasSelectorsRefs {
		return nil
	}

	// Load manifest if not already loaded
	if err := loadSelectorManifest(); err != nil {
		return err
	}

	// Check for selector-related parameters and resolve @selector/ references
	for _, paramName := range selectorParams {
		if selectorRef, ok := step.Params[paramName].(string); ok {
			// Strip /*dup-N*/ suffix first (used to make selector IDs unique in workflows)
			cleanedRef := selectorRef
			if idx := strings.Index(cleanedRef, " /*dup-"); idx != -1 {
				cleanedRef = cleanedRef[:idx]
			}

			// Debug logging - ALWAYS log selector processing
			log.Printf("[SELECTOR_RESOLVE] param=%s original='%s' cleaned='%s'", paramName, selectorRef, cleanedRef)

			// Try to resolve @selector/ references
			if resolved := resolveSelectorReference(cleanedRef); resolved != "" {
				log.Printf("[SELECTOR_RESOLVE] param=%s resolved='%s'", paramName, resolved)
				step.Params[paramName] = resolved
			} else if cleanedRef != selectorRef {
				// No @selector/ reference, but we cleaned the /*dup-N*/ suffix
				log.Printf("[SELECTOR_RESOLVE] param=%s no_manifest using_cleaned='%s'", paramName, cleanedRef)
				step.Params[paramName] = cleanedRef
			} else {
				log.Printf("[SELECTOR_RESOLVE] param=%s unchanged='%s'", paramName, selectorRef)
			}
		}
	}

	// Handle resilience.successSelector and resilience.failureSelector
	if resilience, ok := step.Params["resilience"].(map[string]interface{}); ok {
		if successSelector, ok := resilience["successSelector"].(string); ok {
			cleanedRef := successSelector
			if idx := strings.Index(cleanedRef, " /*dup-"); idx != -1 {
				cleanedRef = cleanedRef[:idx]
			}
			if resolved := resolveSelectorReference(cleanedRef); resolved != "" {
				resilience["successSelector"] = resolved
			} else if cleanedRef != successSelector {
				resilience["successSelector"] = cleanedRef
			}
		}
		if failureSelector, ok := resilience["failureSelector"].(string); ok {
			cleanedRef := failureSelector
			if idx := strings.Index(cleanedRef, " /*dup-"); idx != -1 {
				cleanedRef = cleanedRef[:idx]
			}
			if resolved := resolveSelectorReference(cleanedRef); resolved != "" {
				resilience["failureSelector"] = resolved
			} else if cleanedRef != failureSelector {
				resilience["failureSelector"] = cleanedRef
			}
		}
	}

	return nil
}

// resolveSelectorReference resolves a single @selector/ reference to an actual CSS selector
func resolveSelectorReference(selectorRef string) string {
	// Check if this is a @selector/ reference
	if !strings.HasPrefix(selectorRef, "@selector/") {
		return "" // Not a reference, leave as-is
	}

	// Extract the path (e.g., "dashboard.newProjectButton" from "@selector/dashboard.newProjectButton")
	path := strings.TrimPrefix(selectorRef, "@selector/")

	// Strip /*dup-N*/ suffix if present (used to make selectors unique in workflows)
	// Example: "dialogs.project.root /*dup-1*/" -> "dialogs.project.root"
	if idx := strings.Index(path, " /*dup-"); idx != -1 {
		path = path[:idx]
	}

	// Look up in manifest
	selectors, ok := selectorManifest["selectors"].(map[string]interface{})
	if !ok {
		return ""
	}

	entry, ok := selectors[path].(map[string]interface{})
	if !ok {
		return ""
	}

	selector, ok := entry["selector"].(string)
	if !ok {
		return ""
	}

	return selector
}
