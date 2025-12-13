// Package workflow provides V2 workflow storage utilities.
package workflow

import (
	"github.com/vrooli/browser-automation-studio/automation/workflow"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
)

// SchemaVersion identifies the workflow storage format.
type SchemaVersion string

const (
	// SchemaV1 is the legacy React Flow node format.
	SchemaV1 SchemaVersion = "v1"
	// SchemaV2 is the unified proto-based format.
	SchemaV2 SchemaVersion = "v2"
)

// WorkflowFileV2 represents a workflow file with V2 format support.
type WorkflowFileV2 struct {
	// Schema version for the file
	SchemaVersion SchemaVersion `json:"schema_version,omitempty"`

	// Standard workflow metadata
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	FolderPath  string   `json:"folder_path,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Version     int      `json:"version,omitempty"`

	// V1 format (nodes/edges at top level)
	Nodes          []any          `json:"nodes,omitempty"`
	Edges          []any          `json:"edges,omitempty"`
	FlowDefinition map[string]any `json:"flow_definition,omitempty"`

	// V2 format (proto-based definition)
	DefinitionV2 *basv1.WorkflowDefinitionV2 `json:"definition_v2,omitempty"`

	// Additional metadata
	CreatedBy         string `json:"created_by,omitempty"`
	ChangeDescription string `json:"change_description,omitempty"`
	Source            string `json:"source,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
}

// DetectSchemaVersion determines the schema version from a workflow file payload.
func DetectSchemaVersion(payload map[string]any) SchemaVersion {
	// Check for explicit schema version
	if sv, ok := payload["schema_version"].(string); ok {
		if sv == string(SchemaV2) {
			return SchemaV2
		}
		return SchemaV1
	}

	// Check for V2 definition
	if _, hasV2 := payload["definition_v2"]; hasV2 {
		return SchemaV2
	}

	// Default to V1
	return SchemaV1
}

// ParseV2Definition parses a V2 definition from a map payload.
// Returns nil if not present or invalid.
func ParseV2Definition(payload map[string]any) *basv1.WorkflowDefinitionV2 {
	defV2Raw, ok := payload["definition_v2"].(map[string]any)
	if !ok {
		return nil
	}

	def := &basv1.WorkflowDefinitionV2{}

	// Parse metadata
	if metaRaw, ok := defV2Raw["metadata"].(map[string]any); ok {
		def.Metadata = parseV2Metadata(metaRaw)
	}

	// Parse settings
	if settingsRaw, ok := defV2Raw["settings"].(map[string]any); ok {
		def.Settings = parseV2Settings(settingsRaw)
	}

	// Parse nodes
	if nodesRaw, ok := defV2Raw["nodes"].([]any); ok {
		for _, nodeRaw := range nodesRaw {
			if nodeMap, ok := nodeRaw.(map[string]any); ok {
				if node := parseV2Node(nodeMap); node != nil {
					def.Nodes = append(def.Nodes, node)
				}
			}
		}
	}

	// Parse edges
	if edgesRaw, ok := defV2Raw["edges"].([]any); ok {
		for _, edgeRaw := range edgesRaw {
			if edgeMap, ok := edgeRaw.(map[string]any); ok {
				if edge := parseV2Edge(edgeMap); edge != nil {
					def.Edges = append(def.Edges, edge)
				}
			}
		}
	}

	return def
}

// V1ToV2Definition converts V1 nodes/edges to a V2 definition.
func V1ToV2Definition(nodes, edges []any, metadata, settings map[string]any) *basv1.WorkflowDefinitionV2 {
	// Convert using the workflow package utilities
	v1Flow := workflow.V1FlowDefinition{
		Nodes:    make([]workflow.V1Node, 0, len(nodes)),
		Edges:    make([]workflow.V1Edge, 0, len(edges)),
		Metadata: metadata,
		Settings: settings,
	}

	// Convert nodes
	for _, nodeRaw := range nodes {
		if nodeMap, ok := nodeRaw.(map[string]any); ok {
			v1Node := workflow.V1Node{
				ID:   getString(nodeMap, "id"),
				Type: getString(nodeMap, "type"),
				Data: getMap(nodeMap, "data"),
			}
			if posRaw, ok := nodeMap["position"].(map[string]any); ok {
				v1Node.Position = &workflow.V1Position{
					X: getFloat64(posRaw, "x"),
					Y: getFloat64(posRaw, "y"),
				}
			}
			v1Flow.Nodes = append(v1Flow.Nodes, v1Node)
		}
	}

	// Convert edges
	for _, edgeRaw := range edges {
		if edgeMap, ok := edgeRaw.(map[string]any); ok {
			v1Flow.Edges = append(v1Flow.Edges, workflow.V1Edge{
				ID:           getString(edgeMap, "id"),
				Source:       getString(edgeMap, "source"),
				Target:       getString(edgeMap, "target"),
				Type:         getString(edgeMap, "type"),
				Label:        getString(edgeMap, "label"),
				SourceHandle: getString(edgeMap, "sourceHandle"),
				TargetHandle: getString(edgeMap, "targetHandle"),
			})
		}
	}

	v2Def, err := workflow.V1FlowDefinitionToV2(v1Flow)
	if err != nil {
		return nil
	}

	return v2Def
}

// V2ToV1Definition converts a V2 definition to V1 nodes/edges.
func V2ToV1Definition(def *basv1.WorkflowDefinitionV2) (nodes []any, edges []any, err error) {
	if def == nil {
		return []any{}, []any{}, nil
	}

	v1Flow, err := workflow.WorkflowDefinitionV2ToV1(def)
	if err != nil {
		return nil, nil, err
	}

	// Convert nodes to []any
	nodes = make([]any, 0, len(v1Flow.Nodes))
	for _, n := range v1Flow.Nodes {
		nodeMap := map[string]any{
			"id":   n.ID,
			"type": n.Type,
			"data": n.Data,
		}
		if n.Position != nil {
			nodeMap["position"] = map[string]any{
				"x": n.Position.X,
				"y": n.Position.Y,
			}
		}
		nodes = append(nodes, nodeMap)
	}

	// Convert edges to []any
	edges = make([]any, 0, len(v1Flow.Edges))
	for _, e := range v1Flow.Edges {
		edgeMap := map[string]any{
			"id":     e.ID,
			"source": e.Source,
			"target": e.Target,
		}
		if e.Type != "" {
			edgeMap["type"] = e.Type
		}
		if e.Label != "" {
			edgeMap["label"] = e.Label
		}
		if e.SourceHandle != "" {
			edgeMap["sourceHandle"] = e.SourceHandle
		}
		if e.TargetHandle != "" {
			edgeMap["targetHandle"] = e.TargetHandle
		}
		edges = append(edges, edgeMap)
	}

	return nodes, edges, nil
}

// Helper functions for parsing V2 format

func parseV2Metadata(raw map[string]any) *basv1.WorkflowMetadataV2 {
	if raw == nil {
		return nil
	}
	meta := &basv1.WorkflowMetadataV2{
		Labels: make(map[string]string),
	}
	if name := getString(raw, "name"); name != "" {
		meta.Name = &name
	}
	if desc := getString(raw, "description"); desc != "" {
		meta.Description = &desc
	}
	if version := getString(raw, "version"); version != "" {
		meta.Version = &version
	}
	if labels, ok := raw["labels"].(map[string]any); ok {
		for k, v := range labels {
			if s, ok := v.(string); ok {
				meta.Labels[k] = s
			}
		}
	}
	return meta
}

func parseV2Settings(raw map[string]any) *basv1.WorkflowSettingsV2 {
	if raw == nil {
		return nil
	}
	settings := &basv1.WorkflowSettingsV2{}
	if vw := getInt32(raw, "viewport_width"); vw != 0 {
		settings.ViewportWidth = &vw
	}
	if vh := getInt32(raw, "viewport_height"); vh != 0 {
		settings.ViewportHeight = &vh
	}
	if ua := getString(raw, "user_agent"); ua != "" {
		settings.UserAgent = &ua
	}
	if locale := getString(raw, "locale"); locale != "" {
		settings.Locale = &locale
	}
	if ts := getInt32(raw, "timeout_seconds"); ts != 0 {
		settings.TimeoutSeconds = &ts
	}
	if headless, ok := raw["headless"].(bool); ok {
		settings.Headless = &headless
	}
	return settings
}

func parseV2Node(raw map[string]any) *basv1.WorkflowNodeV2 {
	if raw == nil {
		return nil
	}
	node := &basv1.WorkflowNodeV2{
		Id: getString(raw, "id"),
	}

	// Parse action
	if actionRaw, ok := raw["action"].(map[string]any); ok {
		node.Action = parseV2Action(actionRaw)
	}

	// Parse position
	if posRaw, ok := raw["position"].(map[string]any); ok {
		node.Position = &basv1.NodePosition{
			X: getFloat64(posRaw, "x"),
			Y: getFloat64(posRaw, "y"),
		}
	}

	return node
}

func parseV2Action(raw map[string]any) *basv1.ActionDefinition {
	if raw == nil {
		return nil
	}

	actionTypeStr := getString(raw, "type")
	// Use the existing conversion function from workflow package
	v1Data := getMap(raw, "params")
	if v1Data == nil {
		v1Data = make(map[string]any)
	}

	// Build using v1DataToActionDefinition logic
	action, _ := workflow.V1NodeToWorkflowNodeV2(workflow.V1Node{
		ID:   "temp",
		Type: actionTypeStr,
		Data: v1Data,
	})
	if action != nil {
		return action.Action
	}

	return &basv1.ActionDefinition{}
}

func parseV2Edge(raw map[string]any) *basv1.WorkflowEdgeV2 {
	if raw == nil {
		return nil
	}
	edge := &basv1.WorkflowEdgeV2{
		Id:     getString(raw, "id"),
		Source: getString(raw, "source"),
		Target: getString(raw, "target"),
	}
	if t := getString(raw, "type"); t != "" {
		edge.Type = &t
	}
	if l := getString(raw, "label"); l != "" {
		edge.Label = &l
	}
	if sh := getString(raw, "source_handle"); sh != "" {
		edge.SourceHandle = &sh
	}
	if th := getString(raw, "target_handle"); th != "" {
		edge.TargetHandle = &th
	}
	return edge
}

// Utility functions

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat64(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getInt32(m map[string]any, key string) int32 {
	switch v := m[key].(type) {
	case float64:
		return int32(v)
	case float32:
		return int32(v)
	case int:
		return int32(v)
	case int32:
		return v
	case int64:
		return int32(v)
	}
	return 0
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}
