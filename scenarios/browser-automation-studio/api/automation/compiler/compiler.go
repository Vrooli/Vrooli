package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/paths"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	Index       int    `json:"index"`
	SourceIndex int    `json:"source_index"`
	NodeID      string `json:"node_id"`
	// Action is the compiled V2 execution contract. It is the sole executable
	// representation of a node; the compiler never projects it into a generic
	// type/params map and reconstructs it later.
	Action *basactions.ActionDefinition `json:"-"`
	// Context carries node execution policy into the contracts executor. It is
	// populated from the typed V2 execution settings, never reconstructed from
	// action parameters.
	Context        map[string]any `json:"context,omitempty"`
	OutgoingEdges  []EdgeRef      `json:"outgoing_edges"`
	SourcePosition *Position      `json:"source_position,omitempty"`
	LoopPlan       *ExecutionPlan `json:"loop_plan,omitempty"`
}

// CompileOptions configures compilation behavior for a single workflow.
type CompileOptions struct {
	// SelectorManifestRoot sets the root directory used to resolve selectors.manifest.json.
	// When empty, the compiler falls back to resolving from the current scenario root.
	SelectorManifestRoot string
	// ScenarioRoot identifies the physical root for the scenario under test.
	// It is used only when a navigate destination names that same scenario.
	ScenarioRoot string
	// DeferScenarioURLResolution preserves typed scenario navigation for a
	// target-owned renderer. The target executor will resolve it against the
	// renderer's admitted origin after the target is attached. Normal browser
	// executions must leave this false so missing scenario ports fail at compile
	// time.
	DeferScenarioURLResolution bool
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
// It accepts a proto WorkflowSummary which contains the ID, name, and flow definition.
func CompileWorkflow(workflow *basapi.WorkflowSummary) (*ExecutionPlan, error) {
	return CompileWorkflowWithOptions(workflow, nil)
}

// CompileWorkflowWithOptions converts a stored workflow definition into an execution plan with custom options.
func CompileWorkflowWithOptions(workflow *basapi.WorkflowSummary, opts *CompileOptions) (*ExecutionPlan, error) {
	logrus.WithField("workflow_id", workflow.GetId()).Debug("CompileWorkflow: start")
	if workflow == nil {
		return nil, errors.New("workflow is nil")
	}

	workflowID, err := uuid.Parse(workflow.GetId())
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID: %w", err)
	}
	workflowName := workflow.GetName()

	flowDef := workflow.GetFlowDefinition()
	if flowDef == nil {
		return nil, errors.New("workflow has no flow_definition")
	}
	logrus.WithField("workflow_id", workflow.GetId()).Debug("CompileWorkflow: got flow_definition")

	// Convert proto flow definition to internal flowDefinition struct
	// We use protojson.Marshal to properly serialize proto messages to JSON
	var raw flowDefinition
	logrus.WithField("workflow_id", workflow.GetId()).Debug("CompileWorkflow: about to marshal")
	data, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}).Marshal(flowDef)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow definition: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"workflow_id": workflow.GetId(),
		"data_len":    len(data),
	}).Debug("CompileWorkflow: marshaled")

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}
	attachTypedActions(&raw, flowDef)
	logrus.WithFields(logrus.Fields{
		"workflow_id": workflow.GetId(),
		"node_count":  len(raw.Nodes),
	}).Debug("CompileWorkflow: unmarshaled")

	if len(raw.Nodes) == 0 {
		logrus.WithField("workflow_id", workflow.GetId()).Debug("CompileWorkflow: empty workflow")
		return &ExecutionPlan{
			WorkflowID:   workflowID,
			WorkflowName: workflowName,
			Steps:        []ExecutionStep{},
			Metadata:     map[string]any{},
		}, nil
	}

	logrus.WithField("workflow_id", workflow.GetId()).Debug("CompileWorkflow: about to compileFlow")
	plan, err := compileFlow(flowFragment{definition: raw}, workflowID, workflowName, opts)
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{}
	if width, height := extractViewportFromSettings(raw.Settings); width > 0 && height > 0 {
		metadata["executionViewport"] = map[string]any{"width": width, "height": height}
	}
	fakeMicrophoneWav, err := resolveFakeMediaMicrophone(raw.Settings, opts)
	if err != nil {
		return nil, err
	}
	if fakeMicrophoneWav != "" {
		metadata["fakeMediaMicrophoneWav"] = fakeMicrophoneWav
	}
	if selector, timeout := extractEntryFromSettings(raw.Settings); selector != "" || timeout > 0 {
		if selector != "" {
			metadata["entrySelector"] = selector
		}
		if timeout > 0 {
			metadata["entrySelectorTimeoutMs"] = timeout
		}
	}
	if timeout := extractExecutionTimeoutFromSettings(raw.Settings); timeout > 0 {
		metadata["executionTimeoutMs"] = timeout
	}
	// Workflow labels are the contract-native extension point for execution
	// policy. Keep the external label spelling stable while exposing the
	// executor's typed metadata key. In particular, adhoc validation callers can
	// request a fresh browser without adding a scenario-specific API surface.
	if raw.Metadata != nil {
		if labels, ok := raw.Metadata["labels"].(map[string]any); ok {
			if reuseMode, ok := labels["session_reuse_mode"].(string); ok && strings.TrimSpace(reuseMode) != "" {
				metadata["sessionReuseMode"] = reuseMode
			}
		}
	}
	if len(metadata) == 0 {
		metadata = nil
	}
	plan.Metadata = metadata

	return plan, nil
}

func compileFlow(fragment flowFragment, workflowID uuid.UUID, workflowName string, opts *CompileOptions) (*ExecutionPlan, error) {
	logrus.WithFields(logrus.Fields{
		"workflow_id":   workflowID,
		"workflow_name": workflowName,
	}).Debug("compileFlow: start")

	planner := newPlanner(fragment.definition, opts)
	logrus.WithField("workflow_id", workflowID).Debug("compileFlow: planner created")

	loopFragments, err := planner.extractLoopBodies()
	if err != nil {
		return nil, err
	}
	logrus.WithFields(logrus.Fields{
		"workflow_id": workflowID,
		"loop_count":  len(loopFragments),
	}).Debug("compileFlow: loop bodies extracted")

	steps, err := planner.buildSteps()
	if err != nil {
		return nil, err
	}
	logrus.WithFields(logrus.Fields{
		"workflow_id": workflowID,
		"step_count":  len(steps),
	}).Debug("compileFlow: steps built")

	for idx := range steps {
		if steps[idx].Action.GetType() != basactions.ActionType_ACTION_TYPE_LOOP {
			continue
		}
		logrus.WithFields(logrus.Fields{
			"workflow_id": workflowID,
			"loop_idx":    idx,
			"loop_node":   steps[idx].NodeID,
		}).Debug("compileFlow: processing loop step")

		bodyFragment, ok := loopFragments[steps[idx].NodeID]
		if !ok {
			return nil, fmt.Errorf("loop node %s has no body definition", steps[idx].NodeID)
		}
		childPlan, err := compileFlow(bodyFragment, workflowID, fmt.Sprintf("%s::%s", workflowName, steps[idx].NodeID), opts)
		if err != nil {
			return nil, err
		}
		steps[idx].LoopPlan = childPlan
	}
	logrus.WithField("workflow_id", workflowID).Debug("compileFlow: loops processed")

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

// flowDefinition mirrors the React Flow payload persisted with workflows.
type flowDefinition struct {
	Metadata map[string]any `json:"metadata"`
	Nodes    []rawNode      `json:"nodes"`
	Edges    []rawEdge      `json:"edges"`
	Settings map[string]any `json:"settings"`
}

type flowFragment struct {
	definition   flowDefinition
	specialEdges map[string][]EdgeRef
}

// rawNode mirrors protojson WorkflowNodeV2.
type rawNode struct {
	ID                     string                              `json:"id"`
	Position               map[string]any                      `json:"position,omitempty"`
	ExecSettings           map[string]any                      `json:"execution_settings,omitempty"`
	TypedAction            *basactions.ActionDefinition        `json:"-"`
	TypedExecutionSettings *basworkflows.NodeExecutionSettings `json:"-"`
}

// attachTypedActions retains the V2 action that entered the compiler beside
// the graph-layout projection. Execution always works from this typed action.
func attachTypedActions(definition *flowDefinition, source *basworkflows.WorkflowDefinitionV2) {
	if definition == nil || source == nil {
		return
	}
	actionsByID := make(map[string]*basactions.ActionDefinition, len(source.Nodes))
	settingsByID := make(map[string]*basworkflows.NodeExecutionSettings, len(source.Nodes))
	for _, node := range source.Nodes {
		if node == nil {
			continue
		}
		if node.Action != nil {
			actionsByID[node.Id] = node.Action
		}
		settingsByID[node.Id] = node.ExecutionSettings
	}
	for index := range definition.Nodes {
		definition.Nodes[index].TypedAction = actionsByID[definition.Nodes[index].ID]
		definition.Nodes[index].TypedExecutionSettings = settingsByID[definition.Nodes[index].ID]
	}
}

// hasAction returns true if the node has the required V2 action.
func (n rawNode) hasAction() bool {
	return n.TypedAction != nil
}

// actionType returns the canonical action enum for graph control-flow checks.
func (n rawNode) actionType() (basactions.ActionType, error) {
	if !n.hasAction() {
		return basactions.ActionType_ACTION_TYPE_UNSPECIFIED, fmt.Errorf("workflow node %s missing required action field", n.ID)
	}
	if n.TypedAction.GetType() == basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return basactions.ActionType_ACTION_TYPE_UNSPECIFIED, fmt.Errorf("node %s has unspecified action type", n.ID)
	}
	return n.TypedAction.GetType(), nil
}

type rawEdge struct {
	ID             string         `json:"id"`
	Source         string         `json:"source"`
	Target         string         `json:"target"`
	SourceHandle   string         `json:"sourceHandle,omitempty"`  // camelCase (json_name)
	TargetHandle   string         `json:"targetHandle,omitempty"`  // camelCase (json_name)
	SourceHandleV2 string         `json:"source_handle,omitempty"` // snake_case (proto_name)
	TargetHandleV2 string         `json:"target_handle,omitempty"` // snake_case (proto_name)
	Data           map[string]any `json:"data,omitempty"`
	Type           string         `json:"type,omitempty"`
	Label          string         `json:"label,omitempty"`
}

// getSourceHandle returns the source handle, preferring camelCase over snake_case.
func (e rawEdge) getSourceHandle() string {
	if e.SourceHandle != "" {
		return e.SourceHandle
	}
	return e.SourceHandleV2
}

// getTargetHandle returns the target handle, preferring camelCase over snake_case.
func (e rawEdge) getTargetHandle() string {
	if e.TargetHandle != "" {
		return e.TargetHandle
	}
	return e.TargetHandleV2
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

// resolveFakeMediaMicrophone extracts settings.fake_media.microphone_wav and
// resolves it to an absolute WAV path. Relative paths resolve against the
// execution's project root (and, mirroring the selector-manifest contract,
// its parent when the root is a scenario's bas/ folder) and must stay within
// that root so committed workflows can only reference repo fixtures.
func resolveFakeMediaMicrophone(settings map[string]any, opts *CompileOptions) (string, error) {
	if settings == nil {
		return "", nil
	}
	fm, ok := settings["fake_media"].(map[string]any)
	if !ok {
		fm, ok = settings["fakeMedia"].(map[string]any)
	}
	if !ok {
		return "", nil
	}
	wav, _ := fm["microphone_wav"].(string)
	if strings.TrimSpace(wav) == "" {
		wav, _ = fm["microphoneWav"].(string)
	}
	wav = strings.TrimSpace(wav)
	if wav == "" {
		return "", nil
	}

	if filepath.IsAbs(wav) {
		if _, err := os.Stat(wav); err != nil {
			return "", fmt.Errorf("fake_media.microphone_wav %q not readable: %w", wav, err)
		}
		return filepath.Clean(wav), nil
	}

	projectRoot := ""
	if opts != nil {
		projectRoot = strings.TrimSpace(opts.SelectorManifestRoot)
	}
	if projectRoot == "" {
		return "", fmt.Errorf("fake_media.microphone_wav %q is relative but no project_root was provided to resolve it against", wav)
	}

	roots := []string{filepath.Clean(projectRoot)}
	if filepath.Base(roots[0]) == "bas" {
		roots = append(roots, filepath.Dir(roots[0]))
	}
	tried := make([]string, 0, len(roots))
	for _, root := range roots {
		candidate := filepath.Clean(filepath.Join(root, wav))
		if !strings.HasPrefix(candidate, root+string(filepath.Separator)) {
			return "", fmt.Errorf("fake_media.microphone_wav %q escapes project root %q", wav, root)
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		tried = append(tried, candidate)
	}
	return "", fmt.Errorf("fake_media.microphone_wav %q not found under project root (tried: %v)", wav, tried)
}

func extractViewportFromSettings(settings map[string]any) (int, int) {
	if settings == nil {
		return 0, 0
	}

	// Try snake_case (UseProtoNames: true).
	width := toPositiveInt(settings["viewport_width"])
	height := toPositiveInt(settings["viewport_height"])
	if width > 0 && height > 0 {
		return width, height
	}

	// Try camelCase (default protojson).
	width = toPositiveInt(settings["viewportWidth"])
	height = toPositiveInt(settings["viewportHeight"])
	if width > 0 && height > 0 {
		return width, height
	}

	// Fall back to nested executionViewport object.
	viewportValue, ok := settings["executionViewport"]
	if !ok {
		return 0, 0
	}
	viewportMap, ok := viewportValue.(map[string]any)
	if !ok {
		return 0, 0
	}
	width = toPositiveInt(viewportMap["width"])
	height = toPositiveInt(viewportMap["height"])
	if width <= 0 || height <= 0 {
		return 0, 0
	}
	return width, height
}

func extractEntryFromSettings(settings map[string]any) (string, int) {
	if settings == nil {
		return "", 0
	}

	// Try snake_case first.
	if raw, ok := settings["entry_selector"]; ok {
		if selector, ok := raw.(string); ok && strings.TrimSpace(selector) != "" {
			timeout := toPositiveInt(settings["entry_selector_timeout_ms"])
			return strings.TrimSpace(selector), timeout
		}
	}
	if timeout := toPositiveInt(settings["entry_selector_timeout_ms"]); timeout > 0 {
		return "", timeout
	}

	// Fall back to camelCase.
	raw, ok := settings["entrySelector"]
	if !ok {
		timeout := toPositiveInt(settings["entrySelectorTimeoutMs"])
		if timeout == 0 {
			timeout = toPositiveInt(settings["entryTimeoutMs"])
		}
		return "", timeout
	}
	selector, ok := raw.(string)
	if !ok || strings.TrimSpace(selector) == "" {
		timeout := toPositiveInt(settings["entrySelectorTimeoutMs"])
		if timeout == 0 {
			timeout = toPositiveInt(settings["entryTimeoutMs"])
		}
		return "", timeout
	}
	timeout := toPositiveInt(settings["entrySelectorTimeoutMs"])
	if timeout == 0 {
		timeout = toPositiveInt(settings["entryTimeoutMs"])
	}
	return strings.TrimSpace(selector), timeout
}

// extractExecutionTimeoutFromSettings reads the typed workflow-level timeout.
// The compiler accepts both proto-name JSON and default protojson casing so
// persisted and API-created workflows receive the same execution policy.
func extractExecutionTimeoutFromSettings(settings map[string]any) int {
	if settings == nil {
		return 0
	}
	if timeout := toPositiveInt(settings["timeout_ms"]); timeout > 0 {
		return timeout
	}
	return toPositiveInt(settings["timeoutMs"])
}

type planner struct {
	definition           flowDefinition
	nodesByID            map[string]rawNode
	order                map[string]int
	outgoing             map[string][]rawEdge
	incomingCount        map[string]int
	selectorManifestRoot string
	scenarioRoot         string
	deferScenarioURLs    bool
}

func newPlanner(def flowDefinition, opts *CompileOptions) *planner {
	manifestRoot := ""
	if opts != nil {
		manifestRoot = strings.TrimSpace(opts.SelectorManifestRoot)
	}
	scenarioRoot := ""
	if opts != nil {
		scenarioRoot = strings.TrimSpace(opts.ScenarioRoot)
	}
	deferScenarioURLs := opts != nil && opts.DeferScenarioURLResolution

	p := &planner{
		definition:           def,
		nodesByID:            make(map[string]rawNode, len(def.Nodes)),
		order:                make(map[string]int, len(def.Nodes)),
		outgoing:             make(map[string][]rawEdge),
		incomingCount:        make(map[string]int),
		selectorManifestRoot: manifestRoot,
		scenarioRoot:         scenarioRoot,
		deferScenarioURLs:    deferScenarioURLs,
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
		actionType, err := node.actionType()
		if err != nil {
			return nil, err
		}
		if actionType != basactions.ActionType_ACTION_TYPE_LOOP {
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
	logrus.Debug("buildSteps: about to topologicalOrder")
	order := p.topologicalOrder()
	logrus.WithField("order_len", len(order)).Debug("buildSteps: topologicalOrder done")
	if len(order) != len(p.definition.Nodes) {
		return nil, errors.New("workflow contains a cycle or disconnected nodes")
	}

	steps := make([]ExecutionStep, 0, len(order))
	for idx, nodeID := range order {
		logrus.WithFields(logrus.Fields{
			"idx":     idx,
			"node_id": nodeID,
		}).Debug("buildSteps: processing node")

		node := p.nodesByID[nodeID]
		if _, err := node.actionType(); err != nil {
			return nil, err
		}
		if node.TypedAction == nil {
			return nil, fmt.Errorf("workflow node %s missing required action field", nodeID)
		}
		action, ok := proto.Clone(node.TypedAction).(*basactions.ActionDefinition)
		if !ok {
			return nil, fmt.Errorf("clone typed action for node %s", nodeID)
		}
		step := ExecutionStep{
			Index:       idx,
			SourceIndex: p.order[nodeID],
			NodeID:      node.ID,
			Action:      action,
		}
		if node.TypedExecutionSettings != nil {
			step.Context = executionSettingsToContext(node.TypedExecutionSettings)
		}

		if pos := toPosition(node.Position); pos != nil {
			step.SourcePosition = pos
		}

		// Resolve navigate node URLs from destinationType: "scenario" format
		if action.GetType() == basactions.ActionType_ACTION_TYPE_NAVIGATE {
			logrus.WithField("node_id", nodeID).Debug("buildSteps: resolving navigate URL")
			deferResolution := p.deferScenarioURLs && hasTypedScenarioDestination(action)
			if !deferResolution {
				if err := resolveNavigateURL(&step, p.scenarioRoot); err != nil {
					return nil, fmt.Errorf("failed to resolve navigate URL for node %s: %w", nodeID, err)
				}
			} else {
				logrus.WithField("node_id", nodeID).Debug("buildSteps: deferring target-owned scenario URL resolution")
			}
			logrus.WithField("node_id", nodeID).Debug("buildSteps: navigate URL resolved")
		}

		// Resolve @selector/ references in all steps that have selector parameters
		logrus.WithField("node_id", nodeID).Debug("buildSteps: about to resolveSelectors")
		if err := resolveSelectors(&step, p.selectorManifestRoot); err != nil {
			return nil, fmt.Errorf("failed to resolve selectors for node %s: %w", nodeID, err)
		}
		logrus.WithField("node_id", nodeID).Debug("buildSteps: selectors resolved")

		for _, edge := range p.outgoing[nodeID] {
			if isLoopBodyEdge(edge) {
				continue
			}
			step.OutgoingEdges = append(step.OutgoingEdges, EdgeRef{
				ID:         edge.ID,
				TargetNode: edge.Target,
				Condition:  strings.TrimSpace(edgeCondition(edge)),
				SourcePort: strings.TrimSpace(edge.getSourceHandle()),
				TargetPort: strings.TrimSpace(edge.getTargetHandle()),
			})
		}
		steps = append(steps, step)
	}

	return steps, nil
}

func hasTypedScenarioDestination(action *basactions.ActionDefinition) bool {
	if action == nil || action.GetType() != basactions.ActionType_ACTION_TYPE_NAVIGATE {
		return false
	}
	navigate := action.GetNavigate()
	if navigate == nil || strings.TrimSpace(navigate.GetUrl()) != "" {
		return false
	}
	return navigate.GetDestinationType() == basactions.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_SCENARIO ||
		strings.TrimSpace(navigate.GetScenario()) != ""
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
	return strings.EqualFold(strings.TrimSpace(edge.getSourceHandle()), loopHandleBody) ||
		strings.EqualFold(strings.TrimSpace(edge.getTargetHandle()), loopConditionBody)
}

func loopDirectiveFromEdge(edge rawEdge) (EdgeRef, bool) {
	handle := strings.ToLower(strings.TrimSpace(edge.getTargetHandle()))
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

// resolveNavigateURL resolves a typed navigate action whose destination is a scenario.
func resolveNavigateURL(step *ExecutionStep, scenarioRoot string) error {
	if step == nil || step.Action == nil || step.Action.GetType() != basactions.ActionType_ACTION_TYPE_NAVIGATE {
		return nil
	}
	navigate := step.Action.GetNavigate()
	if navigate == nil {
		return fmt.Errorf("navigate action missing navigate parameters")
	}

	// If URL is already set, no resolution needed
	if strings.TrimSpace(navigate.GetUrl()) != "" {
		return nil
	}

	scenarioName := strings.TrimSpace(navigate.GetScenario())
	scenarioPath := strings.TrimSpace(navigate.GetScenarioPath())
	isScenario := navigate.GetDestinationType() == basactions.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_SCENARIO || scenarioName != ""

	if !isScenario {
		// Not a scenario destination, no resolution needed
		return nil
	}

	if scenarioName == "" {
		return fmt.Errorf("navigate node with destinationType 'scenario' missing scenario name")
	}

	// Resolve URL via scenarioport package with a timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var (
		resolvedURL string
		err         error
	)
	if strings.TrimSpace(scenarioRoot) != "" && filepath.Base(filepath.Clean(scenarioRoot)) == scenarioName {
		resolvedURL, _, err = scenarioport.ResolveURLAtPath(ctx, scenarioName, scenarioRoot, scenarioPath)
	} else {
		resolvedURL, _, err = scenarioport.ResolveURL(ctx, scenarioName, scenarioPath)
	}
	if err != nil {
		return fmt.Errorf("failed to resolve URL for scenario %s: %w", scenarioName, err)
	}

	// The resolved target remains in the canonical typed action.
	navigate.Url = resolvedURL

	return nil
}

// loadSelectorManifest loads the selector manifest from ui/src/consts/selectors.manifest.json,
// scoped by the provided manifestRoot (typically a scenario root or bas/ folder).
// It returns the parsed manifest and the path it was read from so callers can
// surface which manifest actually served a resolution.
func loadSelectorManifest(manifestRoot string) (map[string]interface{}, string, error) {
	// Project files may be resynchronized while BAS stays running. Load the
	// small manifest per compilation so a new canonical selector is usable in
	// the next workflow without a server restart.
	return readSelectorManifest(manifestRoot)
}

func readSelectorManifest(manifestRoot string) (map[string]interface{}, string, error) {
	logrus.WithField("manifest_root", manifestRoot).Debug("loadSelectorManifest: called")
	scenarioDir := paths.ResolveScenarioDir(nil)
	logrus.WithField("scenario_dir", scenarioDir).Debug("loadSelectorManifest: resolved scenario dir")

	searchRoots := make([]string, 0, 4)
	addRoot := func(root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		for _, existing := range searchRoots {
			if existing == root {
				return
			}
		}
		searchRoots = append(searchRoots, root)
	}

	addRoot(manifestRoot)
	if strings.TrimSpace(manifestRoot) != "" && filepath.Base(manifestRoot) == "bas" {
		addRoot(filepath.Dir(manifestRoot))
	}
	addRoot(scenarioDir)

	manifestPaths := make([]string, 0, len(searchRoots)*2+4)
	for _, root := range searchRoots {
		manifestPaths = append(manifestPaths,
			filepath.Join(root, "ui", "src", "consts", "selectors.manifest.json"),
			filepath.Join(root, "ui", "src", "constants", "selectors.manifest.json"),
		)
	}
	manifestPaths = append(manifestPaths,
		"ui/src/consts/selectors.manifest.json",
		"ui/src/constants/selectors.manifest.json",
		"../ui/src/consts/selectors.manifest.json",
		"../ui/src/constants/selectors.manifest.json",
	)

	var data []byte
	var err error
	manifestPath := ""
	for _, path := range manifestPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			manifestPath = path
			break
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to read selector manifest (tried: %v): %w", manifestPaths, err)
	}

	logrus.WithFields(logrus.Fields{
		"manifest_path": manifestPath,
		"manifest_root": manifestRoot,
	}).Info("loadSelectorManifest: manifest found")

	// Parse JSON
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, "", fmt.Errorf("failed to parse selector manifest %s: %w", manifestPath, err)
	}

	return manifest, manifestPath, nil
}

// resolveSelectors resolves @selector/ references in selector-bearing typed action
// fields. Expressions are included because evaluate actions may safely embed a
// symbolic selector inside document.querySelector* calls; leaving such a token
// literal reaches the browser as invalid CSS.
func resolveSelectors(step *ExecutionStep, manifestRoot string) error {
	if step == nil || step.Action == nil {
		return nil
	}
	logrus.WithField("node_id", step.NodeID).Debug("resolveSelectors: start")

	if !messageHasSelectorReference(step.Action.ProtoReflect()) {
		logrus.WithField("node_id", step.NodeID).Debug("resolveSelectors: no @selector/ refs, returning early")
		return nil
	}
	// A selector namespace belongs to the target project, not BAS itself. If
	// the caller did not supply project_root, retain the symbolic reference for
	// the execution boundary instead of resolving it against this repository's
	// unrelated manifest.
	if strings.TrimSpace(manifestRoot) == "" {
		logrus.WithField("node_id", step.NodeID).Debug("resolveSelectors: no project root, preserving selector references")
		return nil
	}

	logrus.WithField("node_id", step.NodeID).Debug("resolveSelectors: has @selector/ refs, loading manifest")
	manifest, manifestPath, err := loadSelectorManifest(manifestRoot)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"node_id":       step.NodeID,
			"manifest_root": manifestRoot,
			"error":         err.Error(),
		}).Debug("resolveSelectors: manifest load failed")
		return err
	}
	logrus.WithField("node_id", step.NodeID).Debug("resolveSelectors: manifest loaded")

	return resolveMessageSelectors(step.Action.ProtoReflect(), manifest, manifestRoot, manifestPath)
}

func messageHasSelectorReference(message protoreflect.Message) bool {
	found := false
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() {
			list := value.List()
			for i := 0; i < list.Len(); i++ {
				if field.Kind() == protoreflect.MessageKind && messageHasSelectorReference(list.Get(i).Message()) {
					found = true
					return false
				}
			}
			return !found
		}
		if field.IsMap() {
			return true
		}
		if field.Kind() == protoreflect.MessageKind {
			found = messageHasSelectorReference(value.Message())
			return !found
		}
		if isSelectorField(field) && strings.Contains(value.String(), "@selector/") {
			found = true
			return false
		}
		return true
	})
	return found
}

func resolveMessageSelectors(message protoreflect.Message, manifest map[string]interface{}, manifestRoot, manifestPath string) error {
	var resolveErr error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() {
			list := value.List()
			for i := 0; i < list.Len(); i++ {
				if field.Kind() == protoreflect.MessageKind {
					if err := resolveMessageSelectors(list.Get(i).Message(), manifest, manifestRoot, manifestPath); err != nil {
						resolveErr = err
						return false
					}
				}
			}
			return true
		}
		if field.IsMap() {
			return true
		}
		if field.Kind() == protoreflect.MessageKind {
			if err := resolveMessageSelectors(value.Message(), manifest, manifestRoot, manifestPath); err != nil {
				resolveErr = err
				return false
			}
			return true
		}
		if !isSelectorField(field) || field.Kind() != protoreflect.StringKind {
			return true
		}
		original := value.String()
		cleaned := strings.Split(original, " /*dup-")[0]
		resolved, err := resolveSelectorTokens(cleaned, manifest, manifestRoot, manifestPath)
		if err != nil {
			resolveErr = fmt.Errorf("field %s: %w", field.FullName(), err)
			return false
		}
		if resolved != cleaned {
			message.Set(field, protoreflect.ValueOfString(resolved))
		} else if cleaned != original {
			message.Set(field, protoreflect.ValueOfString(cleaned))
		}
		return true
	})
	return resolveErr
}

func isSelectorField(field protoreflect.FieldDescriptor) bool {
	name := string(field.Name())
	return name == "selector" || strings.HasSuffix(name, "_selector") || name == "expression"
}

// resolveSelectorReference resolves a single @selector/ reference to an actual CSS selector
func resolveSelectorReference(selectorRef string, manifest map[string]interface{}) string {
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

	// Split a dynamic selector invocation from an optional CSS suffix. Dynamic
	// registry entries are deliberately parameterized (for example
	// projects.cardById(id="${@params/projectId}")), so treating the whole call
	// as a manifest key makes valid reusable BAS subflows uncompilable.
	basePath := path
	suffix := ""
	arguments := map[string]string(nil)
	callIdx := strings.Index(basePath, "(")
	cssIdx := strings.IndexAny(basePath, ":[")
	if callIdx != -1 && (cssIdx == -1 || callIdx < cssIdx) {
		idx := callIdx
		closeIdx := strings.LastIndex(basePath, ")")
		if closeIdx <= idx {
			return ""
		}
		parsed, ok := parseSelectorArguments(basePath[idx+1 : closeIdx])
		if !ok {
			return ""
		}
		arguments = parsed
		suffix = basePath[closeIdx+1:]
		basePath = basePath[:idx]
	} else if cssIdx != -1 {
		idx := cssIdx
		suffix = basePath[idx:]
		basePath = basePath[:idx]
	}

	// Look up in manifest
	if manifest == nil {
		return ""
	}
	if selectors, ok := manifest["selectors"].(map[string]interface{}); ok {
		if entry, ok := selectors[basePath].(map[string]interface{}); ok {
			if selector, ok := entry["selector"].(string); ok {
				return selector + suffix
			}
		}
	}

	// Zero-argument dynamic selectors are stable selector aliases. They belong in
	// the same runtime namespace as literal selectors; only parameterized
	// definitions need call-time interpolation that BAS does not yet support.
	if dynamicSelectors, ok := manifest["dynamicSelectors"].(map[string]interface{}); ok {
		if entry, ok := dynamicSelectors[basePath].(map[string]interface{}); ok {
			params, hasParams := entry["params"].([]interface{})
			if hasParams && len(params) > 0 && arguments == nil {
				return ""
			}
			if selector, ok := entry["selectorPattern"].(string); ok {
				for _, rawParam := range params {
					param, ok := rawParam.(map[string]interface{})
					if !ok {
						return ""
					}
					name, ok := param["name"].(string)
					if !ok {
						return ""
					}
					value, ok := arguments[name]
					if !ok {
						return ""
					}
					selector = strings.ReplaceAll(selector, "${"+name+"}", value)
				}
				return selector + suffix
			}
		}
	}

	return ""
}

// parseSelectorArguments accepts the intentionally small named-argument
// grammar used by selector registry references: name=value pairs separated by
// commas, with values optionally quoted. Quoted values may contain commas and
// preserve workflow placeholders such as ${@params/projectId} verbatim for the
// normal execution-parameter interpolation phase.
func parseSelectorArguments(input string) (map[string]string, bool) {
	args := make(map[string]string)
	for len(strings.TrimSpace(input)) > 0 {
		input = strings.TrimSpace(input)
		eq := strings.IndexByte(input, '=')
		if eq <= 0 {
			return nil, false
		}
		name := strings.TrimSpace(input[:eq])
		if name == "" {
			return nil, false
		}
		input = strings.TrimSpace(input[eq+1:])
		value := ""
		if len(input) > 0 && (input[0] == '\'' || input[0] == '"') {
			quote := input[0]
			end := 1
			for end < len(input) && input[end] != quote {
				if input[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(input) {
				return nil, false
			}
			quoted := input[:end+1]
			if quote == '"' {
				unquoted, err := strconv.Unquote(quoted)
				if err != nil {
					return nil, false
				}
				value = unquoted
			} else {
				value = quoted[1 : len(quoted)-1]
			}
			input = strings.TrimSpace(input[end+1:])
		} else {
			end := strings.IndexByte(input, ',')
			if end == -1 {
				value, input = strings.TrimSpace(input), ""
			} else {
				value, input = strings.TrimSpace(input[:end]), input[end:]
			}
		}
		if value == "" || args[name] != "" {
			return nil, false
		}
		args[name] = value
		if input == "" {
			break
		}
		if input[0] != ',' {
			return nil, false
		}
		input = input[1:]
	}
	return args, true
}

// resolveSelectorTokens replaces any @selector/ references embedded in a selector string.
// An unresolved reference is a hard error: forwarding the literal token would only
// surface later as an opaque runtime selector failure inside the browser driver.
func resolveSelectorTokens(selectorRef string, manifest map[string]interface{}, manifestRoot, manifestPath string) (string, error) {
	resolved := selectorRef
	searchStart := 0
	for {
		idx := strings.Index(resolved[searchStart:], "@selector/")
		if idx == -1 {
			return resolved, nil
		}
		idx += searchStart // Adjust to absolute position
		end := idx + len("@selector/")
		parenDepth := 0
		var quote byte
		for end < len(resolved) {
			ch := resolved[end]
			if quote != 0 {
				if ch == '\\' && end+1 < len(resolved) {
					end += 2
					continue
				}
				if ch == quote {
					quote = 0
				}
				end++
				continue
			}
			switch ch {
			case '\'', '"':
				if parenDepth == 0 {
					goto tokenComplete
				}
				quote = ch
			case '(':
				parenDepth++
			case ')':
				if parenDepth == 0 {
					goto tokenComplete
				}
				parenDepth--
			case ' ', ',':
				if parenDepth == 0 {
					goto tokenComplete
				}
			}
			if parenDepth == 0 && (ch == ';' || ch == '!') {
				break
			}
			end++
		}
	tokenComplete:
		token := resolved[idx:end]
		replacement := resolveSelectorReference(token, manifest)
		if replacement == "" {
			selectorCount := 0
			if selectors, ok := manifest["selectors"].(map[string]interface{}); ok {
				selectorCount = len(selectors)
			}
			logrus.WithFields(logrus.Fields{
				"token":          token,
				"manifest_root":  manifestRoot,
				"manifest_path":  manifestPath,
				"selector_count": selectorCount,
			}).Warn("resolveSelectorTokens: unresolved @selector/ reference")
			return "", fmt.Errorf(
				"unresolved selector reference %q: not present in manifest %s (%d selectors, manifest root %q); "+
					"check that the selector key exists in the target scenario's ui/src/consts/selectors.manifest.json "+
					"and that execution parameter project_root is an absolute path to that scenario",
				token, manifestPath, selectorCount, manifestRoot)
		}
		resolved = resolved[:idx] + replacement + resolved[end:]
		// Adjust search position to account for replacement length difference
		searchStart = idx + len(replacement)
	}
}
