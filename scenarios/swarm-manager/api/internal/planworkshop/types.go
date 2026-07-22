// Package planworkshop owns the durable operator-facing Plan Workshop
// aggregate. A session is a projection over Agent Session proposal records; it
// deliberately never copies or becomes a second proposal store.
package planworkshop

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SubjectKind string

const (
	SubjectBacklog SubjectKind = "backlog_item"
)

type Subject struct {
	Kind SubjectKind `json:"kind"`
	Ref  string      `json:"ref"`
}

func (s Subject) Validate() error {
	if s.Kind != SubjectBacklog {
		return fmt.Errorf("workshop subject kind is invalid")
	}
	if strings.TrimSpace(s.Ref) == "" {
		return fmt.Errorf("workshop subject ref is required")
	}
	return nil
}

type Finding struct {
	ID          string       `json:"id"`
	Severity    string       `json:"severity"`
	Summary     string       `json:"summary"`
	Evidence    string       `json:"evidence,omitempty"`
	Disposition *Disposition `json:"disposition,omitempty"`
}

// Disposition is an evidence producer's bounded recommendation. It is not a
// mutation instruction: Plan Workshop remains the place where an operator
// reviews the evidence and chooses whether a proposal should be applied.
type Disposition struct {
	Kind       string `json:"kind"`
	Rationale  string `json:"rationale"`
	Confidence string `json:"confidence"`
	Scope      string `json:"scope,omitempty"`
}

func (d Disposition) Validate() error {
	switch d.Kind {
	case "plan_revision", "plan_authoring", "follow_up", "archive", "supersede", "attention":
	default:
		return fmt.Errorf("finding disposition kind is invalid")
	}
	if strings.TrimSpace(d.Rationale) == "" {
		return fmt.Errorf("finding disposition rationale is required")
	}
	switch d.Confidence {
	case "high", "medium", "low":
	default:
		return fmt.Errorf("finding disposition confidence is invalid")
	}
	return nil
}

type DecisionQuestion struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
}

// ProposalRef points into the one Agent Session proposal store. SessionID and
// ProposalID are both required so a workshop cannot accidentally apply a
// proposal from a different conversation.
type ProposalRef struct {
	SessionID  string `json:"session_id"`
	ProposalID string `json:"proposal_id"`
	// ApplyMode is derived from the stored proposal payload by Swarm. It is
	// display/projection metadata only; response validation always resolves the
	// canonical ref from this packet rather than trusting caller input.
	ApplyMode string `json:"apply_mode,omitempty"`
}

// FollowUpProposal is the typed relationship carried inside an existing Agent
// Session proposal record. It deliberately names the already-declared work
// route instead of inventing a second follow-up workflow or proposal store.
// Target is currently_work for a correction/follow-up execution; distinct
// backlog creation stays a normal mutation-list proposal with SpawnedFrom
// provenance.
type FollowUpProposal struct {
	Route             string `json:"route"`
	Target            string `json:"target"`
	SourceReviewRef   string `json:"source_review_ref"`
	SourceExecutionID string `json:"source_execution_id"`
	Rationale         string `json:"rationale"`
	Confidence        string `json:"confidence"`
	Scope             string `json:"scope,omitempty"`
}

func (p FollowUpProposal) Validate() error {
	if p.Route != "work.follow_up" && p.Route != "work.correct" {
		return fmt.Errorf("follow-up proposal route is invalid")
	}
	if p.Target != "current_work" {
		return fmt.Errorf("follow-up proposal target is invalid")
	}
	if strings.TrimSpace(p.SourceReviewRef) == "" || strings.TrimSpace(p.SourceExecutionID) == "" || strings.TrimSpace(p.Rationale) == "" {
		return fmt.Errorf("follow-up proposal source and rationale are required")
	}
	switch p.Confidence {
	case "high", "medium", "low":
		return nil
	default:
		return fmt.Errorf("follow-up proposal confidence is invalid")
	}
}

type ReviewPacket struct {
	Findings  []Finding          `json:"findings,omitempty"`
	Questions []DecisionQuestion `json:"questions,omitempty"`
	Proposals []ProposalRef      `json:"proposals,omitempty"`
}

// ReviewPacketVersion preserves every review pass. Packet remains the latest
// compatibility projection; history makes prior questions/findings auditable
// without copying proposal records into a second store.
type ReviewPacketVersion struct {
	ID              string       `json:"id"`
	SubjectVersion  string       `json:"subject_version"`
	PlanContentHash string       `json:"plan_content_hash,omitempty"`
	CreatedAt       string       `json:"created_at"`
	Packet          ReviewPacket `json:"packet"`
}

// LegacyHistoryReference keeps pre-cutover workshop material inspectable
// without treating its readiness/finalization fields as live state.
type LegacyHistoryReference struct {
	SourcePath         string `json:"source_path"`
	RoundCount         int    `json:"round_count"`
	ArchivedAt         string `json:"archived_at"`
	BackupPath         string `json:"backup_path,omitempty"`
	ArchivedUnaccepted bool   `json:"archived_unaccepted,omitempty"`
}

type Response struct {
	ID             string            `json:"id"`
	Actor          string            `json:"actor"`
	SubjectVersion string            `json:"subject_version"`
	Answers        map[string]string `json:"answers,omitempty"`
	Accepted       []ProposalRef     `json:"accepted_proposals,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	SubmittedAt    string            `json:"submitted_at"`
}

func (p ReviewPacket) Validate() error {
	seenFindings := map[string]bool{}
	for _, finding := range p.Findings {
		if strings.TrimSpace(finding.ID) == "" || strings.TrimSpace(finding.Summary) == "" || seenFindings[finding.ID] {
			return fmt.Errorf("review packet contains an invalid finding")
		}
		if finding.Disposition != nil {
			if err := finding.Disposition.Validate(); err != nil {
				return err
			}
		}
		seenFindings[finding.ID] = true
	}
	seenQuestions := map[string]bool{}
	for _, question := range p.Questions {
		id := strings.TrimSpace(question.ID)
		if id == "" || strings.TrimSpace(question.Question) == "" || seenQuestions[id] {
			return fmt.Errorf("review packet contains an invalid decision question")
		}
		seenQuestions[id] = true
	}
	seenProposals := map[string]bool{}
	for _, proposal := range p.Proposals {
		key := strings.TrimSpace(proposal.SessionID) + "/" + strings.TrimSpace(proposal.ProposalID)
		if key == "/" || seenProposals[key] {
			return fmt.Errorf("review packet contains an invalid proposal reference")
		}
		seenProposals[key] = true
	}
	return nil
}

func (r Response) ValidateAgainst(packet ReviewPacket) error {
	questions := map[string]bool{}
	for _, question := range packet.Questions {
		questions[question.ID] = true
	}
	for id := range r.Answers {
		if !questions[id] {
			return fmt.Errorf("response answers an unknown decision question %q", id)
		}
	}
	proposals := map[string]bool{}
	for _, proposal := range packet.Proposals {
		proposals[proposal.SessionID+"/"+proposal.ProposalID] = true
	}
	for _, proposal := range r.Accepted {
		key := strings.TrimSpace(proposal.SessionID) + "/" + strings.TrimSpace(proposal.ProposalID)
		if !proposals[key] {
			return fmt.Errorf("response accepts a proposal outside the review packet")
		}
	}
	return nil
}

type ResolutionState string

const (
	ResolutionPending            ResolutionState = "pending"
	ResolutionDirectApplied      ResolutionState = "direct_applied"
	ResolutionReconciliation     ResolutionState = "reconciliation_required"
	ResolutionCandidateReady     ResolutionState = "candidate_ready"
	ResolutionCandidateApplied   ResolutionState = "candidate_applied"
	ResolutionCandidateDiscarded ResolutionState = "candidate_discarded"
	ResolutionNeedsAttention     ResolutionState = "needs_attention"
	ResolutionStale              ResolutionState = "stale"
	ResolutionUnavailable        ResolutionState = "integration_unavailable"
)

// WorkflowProvenance is the durable correlation to the one bounded internal
// reconciliation engagement. It is displayed as session history; it is not an
// operator-facing workflow choice.
type WorkflowProvenance struct {
	Transition       string `json:"transition"`
	ExecutionID      string `json:"execution_id"`
	DefinitionDigest string `json:"definition_digest"`
	RunID            string `json:"run_id,omitempty"`
	StartedAt        string `json:"started_at"`
}

type ReviewState string

const (
	ReviewPending ReviewState = "pending"
	ReviewRunning ReviewState = "running"
	ReviewApplied ReviewState = "applied"
	ReviewStale   ReviewState = "stale"
	ReviewFailed  ReviewState = "failed"
)

// ReviewRun is the durable review workflow correlation. It stores no agent
// prose: authoritative review content becomes typed findings/questions and
// proposal references only after Swarm validates and projects the result.
type ReviewRun struct {
	State          ReviewState        `json:"state"`
	Workflow       WorkflowProvenance `json:"workflow"`
	AgentSessionID string             `json:"agent_session_id,omitempty"`
	Error          string             `json:"error,omitempty"`
	AppliedAt      string             `json:"applied_at,omitempty"`
}

// ProposalDraft is the bounded workflow result used to create one record in
// the existing Agent Session proposal store. Payload is JSON rather than an
// in-memory parallel proposal type, so existing proposal validation/apply is
// the only mutation authority.
type ProposalDraft struct {
	Summary   string          `json:"summary"`
	Payload   json.RawMessage `json:"payload"`
	ApplyMode string          `json:"apply_mode"`
}

type ReviewResult struct {
	Outcome   string             `json:"outcome"`
	Findings  []Finding          `json:"findings,omitempty"`
	Questions []DecisionQuestion `json:"questions,omitempty"`
	Proposals []ProposalDraft    `json:"proposals,omitempty"`
	Reason    string             `json:"reason,omitempty"`
}

func (r ReviewResult) Validate() error {
	switch r.Outcome {
	case "packet":
		if strings.TrimSpace(r.Reason) != "" {
			return fmt.Errorf("packet review result must not include an attention reason")
		}
	case "needs_attention", "abstained":
		if strings.TrimSpace(r.Reason) == "" {
			return fmt.Errorf("%s review result requires a reason", r.Outcome)
		}
		if len(r.Proposals) > 0 || len(r.Questions) > 0 {
			return fmt.Errorf("%s review result must not include proposals or decision questions", r.Outcome)
		}
	default:
		return fmt.Errorf("unsupported review outcome %q", r.Outcome)
	}
	if err := (ReviewPacket{Findings: r.Findings, Questions: r.Questions}).Validate(); err != nil {
		return err
	}
	for _, proposal := range r.Proposals {
		if strings.TrimSpace(proposal.Summary) == "" || len(proposal.Payload) == 0 {
			return fmt.Errorf("review result contains an invalid proposal")
		}
		if proposal.ApplyMode != "direct" && proposal.ApplyMode != "reconciliation" && proposal.ApplyMode != "attention" {
			return fmt.Errorf("review proposal apply_mode is invalid")
		}
		var payload any
		if json.Unmarshal(proposal.Payload, &payload) != nil {
			return fmt.Errorf("review proposal payload must be JSON")
		}
	}
	return nil
}

type Resolution struct {
	ResponseID       string              `json:"response_id"`
	State            ResolutionState     `json:"state"`
	ReconciliationID string              `json:"reconciliation_id,omitempty"`
	Workflow         *WorkflowProvenance `json:"workflow,omitempty"`
	Candidate        *CandidateReference `json:"candidate,omitempty"`
	Error            string              `json:"error,omitempty"`
	AppliedAt        string              `json:"applied_at,omitempty"`
}

// CandidateReference is a Plan Manager-owned candidate revision. It is never
// a replacement plan reference: a separate guarded, operator-authorized apply
// operation is required before the canonical plan changes.
type CandidateReference struct {
	ID                      string                 `json:"id"`
	PlanID                  string                 `json:"plan_id"`
	ExpectedBaseContentHash string                 `json:"expected_base_content_hash"`
	QualityStatus           string                 `json:"quality_status,omitempty"`
	QualityFindings         []string               `json:"quality_findings,omitempty"`
	Diff                    []CandidateFieldChange `json:"diff,omitempty"`
	Diagnostics             []CandidateDiagnostic  `json:"diagnostics,omitempty"`
	Impact                  CandidateImpact        `json:"impact,omitempty"`
}

// CandidateFieldChange is Plan Manager's structured authored-field diff,
// retained with the workshop so the operator sees the exact reviewed change.
type CandidateFieldChange struct {
	Field      string `json:"field"`
	BeforeJSON string `json:"before_json"`
	AfterJSON  string `json:"after_json"`
}

type CandidateDiagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Location string `json:"location,omitempty"`
	Message  string `json:"message"`
	Guidance string `json:"guidance,omitempty"`
}

type CandidateImpact struct {
	BeforeGrade              string   `json:"before_grade,omitempty"`
	AfterGrade               string   `json:"after_grade,omitempty"`
	AddedIssueCodes          []string `json:"added_issue_codes,omitempty"`
	ClearedIssueCodes        []string `json:"cleared_issue_codes,omitempty"`
	ExecutionGradeRegression bool     `json:"execution_grade_regression,omitempty"`
}

type ReconciliationResult struct {
	Outcome       string          `json:"outcome"`
	CandidatePlan json.RawMessage `json:"candidate,omitempty"`
	Reason        string          `json:"reason,omitempty"`
}

func (r ReconciliationResult) Validate() error {
	switch r.Outcome {
	case "candidate":
		if len(r.CandidatePlan) == 0 || !json.Valid(r.CandidatePlan) {
			return fmt.Errorf("candidate reconciliation result requires a JSON candidate")
		}
	case "needs_attention", "abstained":
		if strings.TrimSpace(r.Reason) == "" {
			return fmt.Errorf("%s reconciliation result requires a reason", r.Outcome)
		}
	default:
		return fmt.Errorf("unsupported reconciliation outcome %q", r.Outcome)
	}
	return nil
}

type Session struct {
	ID              string                  `json:"id"`
	Subject         Subject                 `json:"subject"`
	SubjectVersion  string                  `json:"subject_version"`
	PlanID          string                  `json:"plan_id,omitempty"`
	PlanContentHash string                  `json:"plan_content_hash,omitempty"`
	Packet          ReviewPacket            `json:"packet"`
	PacketHistory   []ReviewPacketVersion   `json:"packet_history,omitempty"`
	LegacyHistory   *LegacyHistoryReference `json:"legacy_history,omitempty"`
	Review          *ReviewRun              `json:"review,omitempty"`
	Responses       []Response              `json:"responses,omitempty"`
	Resolutions     []Resolution            `json:"resolutions,omitempty"`
	CreatedAt       string                  `json:"created_at"`
	UpdatedAt       string                  `json:"updated_at"`
}

func (s Session) Validate() error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("workshop id is required")
	}
	if err := s.Subject.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.SubjectVersion) == "" {
		return fmt.Errorf("workshop subject_version is required")
	}
	return nil
}
