// DOC: docs/concepts/ARCHITECTURE.md#review-evidence
// DOC: docs/internal/SEAMS.md#review-evidence
//
// Package review provides types and file I/O for the post-execution evidence
// gathering system. A review agent produces typed evidence (screenshots, API
// test results, CLI output captures, etc.) that helps humans decide whether to
// archive or follow up on a completed backlog item.
package review

// EvidenceType enumerates the kinds of evidence a review agent can produce.
type EvidenceType string

const (
	EvidenceTypeScreenshot        EvidenceType = "screenshot"
	EvidenceTypeAPITest           EvidenceType = "api_test"
	EvidenceTypeCLIOutput         EvidenceType = "cli_output"
	EvidenceTypeConfigDiff        EvidenceType = "config_diff"
	EvidenceTypeWorkflowRecording EvidenceType = "workflow_recording"
	EvidenceTypeCustom            EvidenceType = "custom"
)

// RoundStatus tracks the review round lifecycle.
type RoundStatus string

const (
	RoundStatusPending   RoundStatus = "pending"
	RoundStatusGathering RoundStatus = "gathering"
	RoundStatusComplete  RoundStatus = "complete"
	RoundStatusFailed    RoundStatus = "failed"
)

// Round captures one pass of evidence gathering by the review agent.
type Round struct {
	RoundNum        int             `json:"round"`
	GeneratedAt     string          `json:"generated_at"`
	ExecutionID     string          `json:"execution_id"`
	Status          RoundStatus     `json:"status"`
	AgentAssessment string          `json:"agent_assessment,omitempty"`
	Classification  string          `json:"classification,omitempty"`
	Notes           []string        `json:"notes,omitempty"`
	Evidence        []EvidenceItem  `json:"evidence"`
	RequestThreads  []RequestThread `json:"request_threads,omitempty"`
	// RunID is the agent-manager run ID for the review agent session.
	RunID string `json:"run_id,omitempty"`
}

// EvidenceItem is a single piece of proof that work was done correctly.
type EvidenceItem struct {
	ID                string               `json:"id"`
	Type              EvidenceType         `json:"type"`
	Title             string               `json:"title"`
	Description       string               `json:"description"`
	CapturePath       string               `json:"capture_path,omitempty"`
	Verified          bool                 `json:"verified"`
	VerifiedAt        string               `json:"verified_at,omitempty"`
	BeforeCapturePath string               `json:"before_capture_path,omitempty"`
	TestResults       []EvidenceTestResult `json:"test_results,omitempty"`
}

// EvidenceTestResult is a structured test outcome within an evidence item.
type EvidenceTestResult struct {
	Name          string `json:"name"`
	Passed        bool   `json:"passed"`
	OutputSummary string `json:"output_summary,omitempty"`
}

// RequestThread is a multi-turn conversation about needing more evidence.
type RequestThread struct {
	ID         string           `json:"id"`
	EvidenceID string           `json:"evidence_id,omitempty"`
	Status     string           `json:"status"` // pending, fulfilled, dismissed
	Messages   []RequestMessage `json:"messages"`
	CreatedAt  string           `json:"created_at"`
	// RunID is the agent-manager run ID for the targeted evidence request.
	RunID string `json:"run_id,omitempty"`
}

// RequestMessage is a single turn in a request thread.
type RequestMessage struct {
	Role             string   `json:"role"` // user, assistant
	Content          string   `json:"content"`
	Timestamp        string   `json:"timestamp"`
	AddedEvidenceIDs []string `json:"added_evidence_ids,omitempty"`
}
