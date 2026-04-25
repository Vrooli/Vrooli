package feedback

import (
	"swarm-manager/internal/proposals"
)

// RoundType discriminates between the three surfaces exposed to the user
// from the "Add feedback" entry point. Only `feedback` and `note` are
// active today; `research` is scaffolded (the UI disables the chip) and
// will land in a follow-up initiative.
type RoundType string

const (
	RoundTypeFeedback RoundType = "feedback"
	RoundTypeResearch RoundType = "research"
	RoundTypeNote     RoundType = "note"
)

// RoundStatus tracks where in the round lifecycle we are. A round always
// starts in RoundStatusSubmitting (while the submission is being persisted)
// and ends in exactly one terminal state: applied, rejected, or dismissed.
type RoundStatus string

const (
	// RoundStatusSubmitting — initial write hasn't finished persisting.
	// Only observable inside the service method.
	RoundStatusSubmitting RoundStatus = "submitting"

	// RoundStatusAgentThinking — an agent run is live (either the initial
	// spawn or a ContinueRun). The lock is held during this state.
	RoundStatusAgentThinking RoundStatus = "agent_thinking"

	// RoundStatusAwaitingUser — agent has produced output (possibly a
	// proposal) and the user is reviewing. The lock is released.
	RoundStatusAwaitingUser RoundStatus = "awaiting_user"

	// RoundStatusApplied — the user accepted (or partially accepted) a
	// proposal and the mutations have been run.
	RoundStatusApplied RoundStatus = "applied"

	// RoundStatusRejected — the user rejected the current proposal without
	// requesting a revision. The round ends; a new round is required to
	// try again.
	RoundStatusRejected RoundStatus = "rejected"

	// RoundStatusDismissed — user abandoned the round (note-type rounds
	// land here after submission; feedback-type rounds can be dismissed
	// before or after a proposal is produced).
	RoundStatusDismissed RoundStatus = "dismissed"
)

// DecisionKind enumerates the shapes a terminal user decision can take.
type DecisionKind string

const (
	DecisionAccept        DecisionKind = "accept"
	DecisionPartialAccept DecisionKind = "partial_accept"
	DecisionReject        DecisionKind = "reject"
	DecisionRevise        DecisionKind = "revise"
	DecisionDismiss       DecisionKind = "dismiss"
)

// IsTerminal reports whether a RoundStatus is end-of-life.
func (s RoundStatus) IsTerminal() bool {
	switch s {
	case RoundStatusApplied, RoundStatusRejected, RoundStatusDismissed:
		return true
	}
	return false
}

// Submission is the user's initial input — the text and attachments that
// kick off the round. Additional user messages (during revise) land in
// Thread, not here.
type Submission struct {
	Text          string   `json:"text"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

// Message is a single turn in the agent↔user conversation that follows the
// initial submission. It mirrors workshop.ClarificationMessage by intent
// (role/content/attachments) but is not structurally coupled to it — we
// own our own shape because feedback threads have their own metadata
// (e.g. ProposalID linking the turn to a specific proposal revision).
type Message struct {
	Role          string   `json:"role"` // "user" | "agent"
	Content       string   `json:"content"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
	ProposalID    string   `json:"proposal_id,omitempty"` // set on agent messages that carried a proposal
	RunID         string   `json:"run_id,omitempty"`      // agent-manager run ID, for audit
	// ParseWarnings records the proposal extractor's complaints when an
	// agent turn either produced no extractable proposal or produced one
	// only after falling back. Attached to the message (not the round)
	// so the UI can render per-turn "the agent's JSON was malformed" hints
	// without losing history across subsequent turns.
	ParseWarnings []string `json:"parse_warnings,omitempty"`
	CreatedAt     string   `json:"created_at"`
}

// ProposalRevision pairs an agent-produced proposal with the round + turn
// that produced it. Proposals accumulate across revisions so the user can
// compare revisions if desired.
type ProposalRevision struct {
	ID            string             `json:"id"`
	MessageIndex  int                `json:"message_index"` // index into Thread for the source turn
	Proposal      proposals.Proposal `json:"proposal"`
	Rationale     string             `json:"rationale,omitempty"`
	CreatedAt     string             `json:"created_at"`
	ParseWarnings []string           `json:"parse_warnings,omitempty"`
}

// Decision is the user's terminal verdict on the round. Persisted exactly
// once per round.
type Decision struct {
	Kind                DecisionKind `json:"kind"`
	AcceptedMutationIDs []string     `json:"accepted_mutation_ids,omitempty"`
	RejectedMutationIDs []string     `json:"rejected_mutation_ids,omitempty"`
	Rationale           string       `json:"rationale,omitempty"`
	DecidedAt           string       `json:"decided_at"`
	DecidedBy           string       `json:"decided_by,omitempty"`
}

// Round is the full on-disk record for a single feedback round. Stored at
// `initiatives/{name}/feedback/round-NNN-{slug}/feedback.json`.
type Round struct {
	InitiativeName    string             `json:"initiative_name"`
	Number            int                `json:"number"`
	Slug              string             `json:"slug"`
	Type              RoundType          `json:"type"`
	Status            RoundStatus        `json:"status"`
	Submission        Submission         `json:"submission"`
	Thread            []Message          `json:"thread,omitempty"`
	Proposals         []ProposalRevision `json:"proposals,omitempty"`
	CurrentProposalID string             `json:"current_proposal_id,omitempty"`
	Decision          *Decision          `json:"decision,omitempty"`
	RunID             string             `json:"run_id,omitempty"` // active agent-manager run (cleared on terminal)
	// NeedsRevision is set true when the most recent agent turn produced
	// no extractable proposal. The UI reads this to render the
	// "ask the agent for a revision" CTA instead of a blank proposal
	// panel — a structured replacement for the plan's "parse error" flag.
	// Cleared automatically on the next successful agent turn or when
	// the user continues/decides the round.
	NeedsRevision bool `json:"needs_revision,omitempty"`
	// LastParseWarnings mirrors the ParseWarnings on the most recent
	// agent turn. Surfaced at the round level so the UI's revision CTA
	// can show the reason without scanning the thread.
	LastParseWarnings []string `json:"last_parse_warnings,omitempty"`
	// LastPolledAt records the most recent poll attempt for an
	// agent_thinking round. Lets the stuck-round sweeper find rounds
	// whose polling has wedged without forcing the UI to wait.
	LastPolledAt string `json:"last_polled_at,omitempty"`
	// LastPollError carries the most recent poller error message so the
	// UI can show "agent unreachable: …" instead of a perpetual spinner.
	// Cleared on successful terminal advance.
	LastPollError string `json:"last_poll_error,omitempty"`
	// PollFailureCount counts consecutive poll failures. After a
	// configured threshold, EnsurePolledTurn synthesizes a terminal
	// failure so the round can resolve. Cleared on success.
	PollFailureCount int    `json:"poll_failure_count,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// CurrentProposal returns the ProposalRevision matching CurrentProposalID,
// or nil if none is active (fresh round, or proposal was never parsed).
func (r *Round) CurrentProposal() *ProposalRevision {
	if r.CurrentProposalID == "" {
		return nil
	}
	for i := range r.Proposals {
		if r.Proposals[i].ID == r.CurrentProposalID {
			return &r.Proposals[i]
		}
	}
	return nil
}

// AppendThreadMessage appends a message to the thread and returns its index.
// Callers use the index to cross-reference a proposal back to its source turn.
func (r *Round) AppendThreadMessage(m Message) int {
	r.Thread = append(r.Thread, m)
	return len(r.Thread) - 1
}
