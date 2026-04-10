// DOC: docs/concepts/ARCHITECTURE.md#data-model
package main

import "time"

// Scheme is a named capture workspace that groups related information items
// and thoughts. It is the top-level organizational unit — analogous to a
// project or notebook.
type Scheme struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Information is a raw capture item placed on the spatial canvas.
// Each item has a type (currently "text"; future: "voice", "url", "image")
// and canvas coordinates for spatial layout. Information items are the
// unprocessed inputs; Thoughts are the refined, graph-connected counterparts.
type Information struct {
	ID        string    `json:"id"`
	SchemeID  string    `json:"scheme_id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	CanvasX   float64   `json:"canvas_x"`
	CanvasY   float64   `json:"canvas_y"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Thought is a refined idea node in the relationship graph.
// Unlike Information (raw captures), Thoughts are titled concepts that can be
// connected via directional edges to form a knowledge graph.
// SchemeID is nullable to support cross-scheme thoughts that bridge workspaces.
type Thought struct {
	ID        string    `json:"id"`
	SchemeID  *string   `json:"scheme_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CanvasX   float64   `json:"canvas_x"`
	CanvasY   float64   `json:"canvas_y"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ThoughtEdge is a labeled, directional connection between two thoughts
// (e.g., "causes", "supports", "contradicts"). The DB enforces a unique
// constraint on (source_id, target_id) to prevent duplicate edges.
type ThoughtEdge struct {
	ID        string    `json:"id"`
	SourceID  string    `json:"source_id"`
	TargetID  string    `json:"target_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateSchemeInput is the input for creating a scheme
type CreateSchemeInput struct {
	Name string `json:"name"`
}

// UpdateSchemeInput is the input for updating a scheme
type UpdateSchemeInput struct {
	Name string `json:"name"`
}

// CreateInformationInput is the input for creating an information item
type CreateInformationInput struct {
	Type    string  `json:"type"`
	Content string  `json:"content"`
	CanvasX float64 `json:"canvas_x"`
	CanvasY float64 `json:"canvas_y"`
}

// UpdateInformationInput is the input for updating an information item
type UpdateInformationInput struct {
	Type    *string  `json:"type,omitempty"`
	Content *string  `json:"content,omitempty"`
	CanvasX *float64 `json:"canvas_x,omitempty"`
	CanvasY *float64 `json:"canvas_y,omitempty"`
}

// CreateThoughtInput is the input for creating a thought
type CreateThoughtInput struct {
	SchemeID *string `json:"scheme_id"`
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	CanvasX  float64 `json:"canvas_x"`
	CanvasY  float64 `json:"canvas_y"`
}

// UpdateThoughtInput is the input for updating a thought
type UpdateThoughtInput struct {
	Title   *string  `json:"title,omitempty"`
	Body    *string  `json:"body,omitempty"`
	CanvasX *float64 `json:"canvas_x,omitempty"`
	CanvasY *float64 `json:"canvas_y,omitempty"`
}

// CreateEdgeInput is the input for creating a thought edge
type CreateEdgeInput struct {
	TargetID string `json:"target_id"`
	Label    string `json:"label"`
}
