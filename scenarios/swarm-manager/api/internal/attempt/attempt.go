// Package attempt defines the durable, domain-neutral record of one
// transition attempt. Lifecycle state belongs exclusively to transitionrun.
package attempt

import "swarm-manager/internal/transitionrun"

// Attempt stores the operator-facing output of a transition. It deliberately
// excludes execution id, apply state, outcome, and retry state: Projection
// joins those facts from transitionrun.Correlation at read time.
type Attempt struct {
	SubjectKind   string     `json:"subject_kind"`
	SubjectRef    string     `json:"subject_ref"`
	TransitionKey string     `json:"transition_key"`
	RoundNum      int        `json:"round_num"`
	Status        string     `json:"status"`
	GeneratedAt   string     `json:"generated_at"`
	Assessment    string     `json:"assessment,omitempty"`
	Verdict       string     `json:"verdict,omitempty"`
	Evidence      []Evidence `json:"evidence,omitempty"`
	Proposals     []Proposal `json:"proposals,omitempty"`
	Disposition   *Decision  `json:"disposition,omitempty"`
}

// RoundNumber lets Attempt use the shared durable attempt store without
// introducing a second file payload type for goal and milestone workflows.
func (a Attempt) RoundNumber() int { return a.RoundNum }

type Evidence struct {
	ID                string       `json:"id"`
	CriterionID       string       `json:"criterion_id,omitempty"`
	Settlement        string       `json:"settlement"`
	Type              string       `json:"type"`
	Title             string       `json:"title"`
	Description       string       `json:"description,omitempty"`
	Producer          string       `json:"producer"`
	Trust             string       `json:"trust"`
	UnavailableReason string       `json:"unavailable_reason,omitempty"`
	AttemptedProducer string       `json:"attempted_producer,omitempty"`
	Artifact          *Artifact    `json:"artifact,omitempty"`
	TestResults       []TestResult `json:"test_results,omitempty"`
}

// Artifact follows the portable evidence descriptor emitted by Test Genie.
type Artifact struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	RelativePath string `json:"relative_path"`
	SizeBytes    int64  `json:"size_bytes"`
	SHA256       string `json:"sha256"`
	ContentType  string `json:"content_type"`
}

type TestResult struct {
	Name          string `json:"name"`
	Passed        bool   `json:"passed"`
	OutputSummary string `json:"output_summary,omitempty"`
}

type Proposal struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
}

type Decision struct {
	Kind      string `json:"kind"`
	Rationale string `json:"rationale,omitempty"`
}

// Projection is the single read model that combines a domain attempt with
// lifecycle facts from the shared correlation journal.
type Projection struct {
	Attempt
	ExecutionID  string `json:"execution_id"`
	ApplyState   string `json:"apply_state"`
	Outcome      string `json:"outcome"`
	TerminalCode string `json:"terminal_code,omitempty"`
}

func Project(value Attempt, correlation transitionrun.Correlation) Projection {
	return Projection{Attempt: value, ExecutionID: correlation.ExecutionID, ApplyState: correlation.ApplyState, Outcome: correlation.Outcome, TerminalCode: correlation.TerminalCode}
}
