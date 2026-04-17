package report

import (
	"time"
)

// ConflictType classifies the kind of cross-skill contradiction.
type ConflictType string

const (
	// ConflictStructural indicates incompatible structural expectations
	// from different skills on the same reference.
	ConflictStructural ConflictType = "structural"
	// ConflictCLI indicates overlapping CLI assertions with different
	// expected values from different skills on the same reference.
	ConflictCLI ConflictType = "cli"
)

// Conflict represents a cross-skill contradiction found on a reference.
type Conflict struct {
	ReferenceID  string       `json:"reference_id"`
	Type         ConflictType `json:"type"`
	Description  string       `json:"description"`
	SkillA       string       `json:"skill_a"`
	ConnectionA  string       `json:"connection_a"`
	SkillB       string       `json:"skill_b"`
	ConnectionB  string       `json:"connection_b"`
	ExpectationA string       `json:"expectation_a,omitempty"`
	ExpectationB string       `json:"expectation_b,omitempty"`
}

// ConflictsReport aggregates all cross-skill contradictions.
type ConflictsReport struct {
	Conflicts   []*Conflict `json:"conflicts"`
	TotalCount  int         `json:"total_count"`
	GeneratedAt time.Time   `json:"generated_at"`
}

// DriftEntry represents a single skill connection that has drifted.
type DriftEntry struct {
	ConnectionID   string `json:"connection_id"`
	ReferenceID    string `json:"reference_id"`
	SkillID        string `json:"skill_id"`
	StoredHash     string `json:"stored_hash"`
	CurrentHash    string `json:"current_hash"`
	StoredVersion  string `json:"stored_version"`
	CurrentVersion string `json:"current_version"`
	VersionChanged bool   `json:"version_changed"`
	ContentChanged bool   `json:"content_changed"`
}

// DriftReport aggregates drift status across all connections.
type DriftReport struct {
	DriftedConnections []*DriftEntry `json:"drifted_connections"`
	TotalConnections   int           `json:"total_connections"`
	DriftedCount       int           `json:"drifted_count"`
	GeneratedAt        time.Time     `json:"generated_at"`
}

// MaturityLevel classifies how mature a skill's expectations are.
type MaturityLevel string

const (
	MaturityLow    MaturityLevel = "low"
	MaturityMedium MaturityLevel = "medium"
	MaturityHigh   MaturityLevel = "high"
)

// SkillMaturity scores a skill's expectation coverage on a specific reference.
type SkillMaturity struct {
	ConnectionID      string        `json:"connection_id"`
	ReferenceID       string        `json:"reference_id"`
	SkillID           string        `json:"skill_id"`
	StructuralCount   int           `json:"structural_count"`
	CLICount          int           `json:"cli_count"`
	TotalExpectations int           `json:"total_expectations"`
	HasStructural     bool          `json:"has_structural"`
	HasCLI            bool          `json:"has_cli"`
	Level             MaturityLevel `json:"level"`
	Score             float64       `json:"score"`
}

// MaturityReport aggregates skill maturity across connections.
type MaturityReport struct {
	Skills       []*SkillMaturity      `json:"skills"`
	Distribution map[MaturityLevel]int `json:"distribution"`
	AverageScore float64               `json:"average_score"`
	GeneratedAt  time.Time             `json:"generated_at"`
}

// ToolBaselineStatus indicates whether a tool's results match stored baselines.
type ToolBaselineStatus string

const (
	BaselinePass       ToolBaselineStatus = "pass"
	BaselineFail       ToolBaselineStatus = "fail"
	BaselineNoBaseline ToolBaselineStatus = "no_baseline"
	BaselineError      ToolBaselineStatus = "error"
)

// ToolBaseline represents the baseline check result for one tool on one reference.
type ToolBaseline struct {
	ReferenceID   string             `json:"reference_id"`
	ToolName      string             `json:"tool_name"`
	Status        ToolBaselineStatus `json:"status"`
	PassCount     int                `json:"pass_count"`
	FailCount     int                `json:"fail_count"`
	ErrorCount    int                `json:"error_count"`
	TotalAsserted int                `json:"total_asserted"`
	Message       string             `json:"message,omitempty"`
}

// ToolBaselinesReport aggregates tool accuracy regression checks.
type ToolBaselinesReport struct {
	Baselines    []*ToolBaseline `json:"baselines"`
	TotalTools   int             `json:"total_tools"`
	PassingTools int             `json:"passing_tools"`
	FailingTools int             `json:"failing_tools"`
	GeneratedAt  time.Time       `json:"generated_at"`
}

// ListOptions controls filtering for report generation.
type ListOptions struct {
	ReferenceID string `json:"reference_id,omitempty"`
	SkillID     string `json:"skill_id,omitempty"`
}
