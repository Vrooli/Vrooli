// DOC: docs/concepts/GRAPH.md#node-types
// DOC: docs/concepts/GRAPH.md#edge-types
package graph

import "time"

// NodeType classifies graph nodes.
type NodeType string

const (
	NodeTeam   NodeType = "team"
	NodeAgent  NodeType = "agent"
	NodeSkill  NodeType = "skill"
	NodeAction NodeType = "action"
	NodeCLI    NodeType = "cli"
)

// EdgeKind classifies relationships between nodes.
type EdgeKind string

const (
	EdgeCLIRead       EdgeKind = "cli-read"
	EdgeBoldListed    EdgeKind = "bold-listed"
	EdgeDefaultScope  EdgeKind = "default-scope"
	EdgePathRef       EdgeKind = "path-ref"
	EdgeMembership    EdgeKind = "membership"
	EdgeCodeUsage     EdgeKind = "code-usage"
	EdgeActionUse     EdgeKind = "action-use"
	EdgeActionCommand EdgeKind = "action-command"
)

// Node represents a single entity in the graph.
type Node struct {
	ID          string   `json:"id"`
	Type        NodeType `json:"type"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Edge represents a directed relationship between two nodes.
type Edge struct {
	From       string       `json:"from"`
	To         string       `json:"to"`
	Kind       EdgeKind     `json:"kind"`
	Category   CodeCategory `json:"category,omitempty"`   // Only set for EdgeCodeUsage edges
	Command    string       `json:"command,omitempty"`    // Canonical command token (e.g. "vrooli", "grep")
	Subcommand string       `json:"subcommand,omitempty"` // First non-flag argument, when present
	SourceFile string       `json:"sourceFile,omitempty"`
	LineNumber int          `json:"lineNumber,omitempty"`
}

// HealthMessage explains why a node's health was scored a certain way
// and suggests concrete follow-up actions.
type HealthMessage struct {
	Key            string  `json:"key"`
	Severity       string  `json:"severity"` // info | warning | critical
	Factor         string  `json:"factor,omitempty"`
	Summary        string  `json:"summary"`
	Detail         string  `json:"detail,omitempty"`
	Recommendation string  `json:"recommendation,omitempty"`
	MetricValue    float64 `json:"metricValue,omitempty"`
	Target         string  `json:"target,omitempty"`
}

// HealthScore contains computed health metrics for a node.
type HealthScore struct {
	NodeID string  `json:"nodeId"`
	Score  float64 `json:"score"`
	// Individual factor contributions
	Factors map[string]float64 `json:"factors"`
	// Human-readable diagnostics and recommendations.
	Messages []HealthMessage `json:"messages,omitempty"`
}

// NodeMetricSet stores per-node raw metrics used by health factor evaluators.
type NodeMetricSet map[string]float64

// Graph is the complete in-memory graph structure.
type Graph struct {
	Nodes        []Node        `json:"nodes"`
	Edges        []Edge        `json:"edges"`
	HealthScores []HealthScore `json:"healthScores,omitempty"`
	// NodeMetrics are internal scoring inputs and are not serialized.
	NodeMetrics map[string]NodeMetricSet `json:"-"`
}

// GraphIndex is the persisted graph index.
type GraphIndex struct {
	GeneratedAt string `json:"generatedAt"`
	Graph       Graph  `json:"graph"`
}

// NewGraphIndex creates a new graph index with a current timestamp.
func NewGraphIndex(g Graph) *GraphIndex {
	return &GraphIndex{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Graph:       g,
	}
}

// GraphInvalidator allows other packages to trigger graph index invalidation.
type GraphInvalidator interface {
	Invalidate()
}

// CodeCategory classifies detected code references.
type CodeCategory string

const (
	CodeScenarioCLI  CodeCategory = "scenario-cli"
	CodeExternalTool CodeCategory = "external-tool"
	CodeScript       CodeCategory = "script"
	CodeAPICall      CodeCategory = "api-call"
)

// CodeReference represents a detected code usage in content.
type CodeReference struct {
	Category CodeCategory `json:"category"`
	Value    string       `json:"value"`
	Line     int          `json:"line"`
}
