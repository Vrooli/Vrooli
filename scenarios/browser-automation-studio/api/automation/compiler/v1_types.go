// Package compiler provides workflow compilation and format conversion utilities.
package compiler

// WorkflowSchemaVersion identifies the workflow storage format.
type WorkflowSchemaVersion string

const (
	// SchemaVersionV1 is the legacy React Flow node format.
	SchemaVersionV1 WorkflowSchemaVersion = "v1"
	// SchemaVersionV2 is the unified proto-based format.
	SchemaVersionV2 WorkflowSchemaVersion = "v2"
)

// V1Node represents a legacy React Flow workflow node.
type V1Node struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Data     map[string]any `json:"data"`
	Position *V1Position    `json:"position,omitempty"`
}

// V1Position represents node position in canvas.
type V1Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// V1Edge represents a legacy React Flow workflow edge.
type V1Edge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	Type         string `json:"type,omitempty"`
	Label        string `json:"label,omitempty"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

// V1FlowDefinition represents the legacy workflow storage format.
type V1FlowDefinition struct {
	Nodes    []V1Node       `json:"nodes"`
	Edges    []V1Edge       `json:"edges"`
	Settings map[string]any `json:"settings,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}
