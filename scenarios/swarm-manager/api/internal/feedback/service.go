package feedback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"swarm-manager/internal/proposals"
)

// AgentSpawner is the injected interface the service uses to start or
// continue an agent-manager run for the feedback round. Concrete wiring
// (agentmanager.Service + a SpawnInitiative sibling to SpawnBacklog) lives
// outside this package so feedback stays a leaf.
type AgentSpawner interface {
	// SpawnInitiativeFeedback starts a fresh agent run for a feedback round.
	// Returns the agent-manager RunID, which the service persists on the
	// round so ContinueRun can reuse it.
	SpawnInitiativeFeedback(ctx context.Context, req SpawnRequest) (string, error)

	// ContinueRun appends the user's message to an existing run, reusing
	// the thread/context the agent already has. Mirrors the pattern used
	// by clarification multi-turn.
	ContinueRun(ctx context.Context, runID, message string, attachmentIDs []string) error
}

// SpawnRequest carries everything the spawner needs to start a feedback
// agent run. Fields are a superset of what any one round actually uses —
// the adapter picks the subset relevant to the skill it's invoking.
type SpawnRequest struct {
	InitiativeName string
	RoundNumber    int
	RoundSlug      string
	Purpose        string // "feedback" | "feedback_continue" | "review"
	SubmissionText string
	AttachmentIDs  []string
}

// ItemActivityChecker reports whether any of the initiative's items has an
// active agent run. Used by StartRound to surface blocking state to the
// override dialog. Nil checker degrades the check to "nothing active",
// which is safe for tests.
type ItemActivityChecker interface {
	ActiveRunsForInitiative(initiativeName string) ([]ItemActivity, error)
}

// ItemActivity describes a single in-flight agent run the override dialog
// may surface to the user.
type ItemActivity struct {
	Ref     string // "kind/name"
	RunID   string
	Purpose string
}

// Service orchestrates feedback round lifecycle.
type Service struct {
	store    *Store
	lock     *Lock
	spawner  AgentSpawner
	activity ItemActivityChecker
	apply    *proposals.Applier
	// StateBuilder turns an initiative name into a proposals.CurrentState.
	// Injected so the service doesn't depend on the graph materializer
	// directly — the wiring in main.go closes over the materializer to
	// provide a fresh snapshot per call.
	StateBuilder func(initiativeName string) (proposals.CurrentState, error)
	clock        func() time.Time
}

// Config bundles Service dependencies.
type Config struct {
	Store        *Store
	Lock         *Lock
	Spawner      AgentSpawner
	Activity     ItemActivityChecker
	Apply        *proposals.Applier
	StateBuilder func(initiativeName string) (proposals.CurrentState, error)
	Clock        func() time.Time
}

// NewService constructs a feedback Service.
func NewService(cfg Config) (*Service, error) {
	if cfg.Store == nil {
		return nil, errors.New("feedback: Store is required")
	}
	if cfg.Lock == nil {
		return nil, errors.New("feedback: Lock is required")
	}
	if cfg.Apply == nil {
		return nil, errors.New("feedback: proposals.Applier is required")
	}
	if cfg.StateBuilder == nil {
		return nil, errors.New("feedback: StateBuilder is required")
	}
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	return &Service{
		store:        cfg.Store,
		lock:         cfg.Lock,
		spawner:      cfg.Spawner,
		activity:     cfg.Activity,
		apply:        cfg.Apply,
		StateBuilder: cfg.StateBuilder,
		clock:        clk,
	}, nil
}

// StartRoundRequest is the caller-supplied input for starting a new round.
type StartRoundRequest struct {
	InitiativeName string
	Type           RoundType
	Text           string
	AttachmentIDs  []string
	SlugHint       string // optional; derived from Text if empty
	Override       bool   // preempt active lock if present
	DecidedBy      string // user identifier for audit
}

// StartRound creates a new round, acquires the lock, and (for active round
// types) spawns the feedback agent. For note-type rounds the agent is
// skipped — the round lands in RoundStatusDismissed right away because
// notes are signal-only entries, not conversations.
func (s *Service) StartRound(ctx context.Context, req StartRoundRequest) (Round, error) {
	if strings.TrimSpace(req.InitiativeName) == "" {
		return Round{}, errors.New("initiative name is required")
	}
	if req.Type == "" {
		req.Type = RoundTypeFeedback
	}
	if !isValidRoundType(req.Type) {
		return Round{}, fmt.Errorf("unsupported round type %q", req.Type)
	}
	if req.Type == RoundTypeResearch {
		return Round{}, errors.New("research-type rounds are not implemented in this release")
	}
	if strings.TrimSpace(req.Text) == "" {
		return Round{}, errors.New("text is required")
	}

	number, err := s.store.NextRoundNumber(req.InitiativeName)
	if err != nil {
		return Round{}, fmt.Errorf("assign round number: %w", err)
	}
	slug := ComputeSlug(req.SlugHint, req.Text)

	now := s.clock().UTC().Format(time.RFC3339)
	round := Round{
		InitiativeName: req.InitiativeName,
		Number:         number,
		Slug:           slug,
		Type:           req.Type,
		Status:         RoundStatusSubmitting,
		Submission: Submission{
			Text:          strings.TrimSpace(req.Text),
			AttachmentIDs: append([]string(nil), req.AttachmentIDs...),
			CreatedAt:     now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Note-type rounds skip the agent path: record the submission and the
	// user's authored message, then mark dismissed. They still count as a
	// feedback event for the future meta-optimizer, which is the whole
	// point of the note type.
	if req.Type == RoundTypeNote {
		round.Thread = []Message{{
			Role:          "user",
			Content:       round.Submission.Text,
			AttachmentIDs: round.Submission.AttachmentIDs,
			CreatedAt:     now,
		}}
		round.Status = RoundStatusDismissed
		round.Decision = &Decision{
			Kind:      DecisionDismiss,
			DecidedAt: now,
			DecidedBy: req.DecidedBy,
			Rationale: "note",
		}
		if err := s.store.SaveRound(round); err != nil {
			return Round{}, fmt.Errorf("save note round: %w", err)
		}
		return round, nil
	}

	// Feedback rounds acquire the lock before any agent side-effect runs
	// so concurrent submissions can't race us into two spawns.
	holder := Holder{
		Purpose:     "feedback",
		RoundNumber: number,
		AcquiredBy:  req.DecidedBy,
	}
	if s.spawner != nil {
		// RunID assigned post-spawn; acquire with a provisional id so
		// Inspect still returns something meaningful.
		holder.RunID = fmt.Sprintf("provisional-%d-%d", number, time.Now().UnixNano())
	} else {
		holder.RunID = fmt.Sprintf("no-spawner-%d", number)
	}
	if req.Override {
		if err := s.lock.AcquireOverride(req.InitiativeName, holder); err != nil {
			return Round{}, fmt.Errorf("override lock: %w", err)
		}
	} else {
		if err := s.lock.Acquire(req.InitiativeName, holder); err != nil {
			return Round{}, err
		}
	}

	// Record the submission to disk before spawning so a spawn failure
	// leaves evidence of the user's attempt.
	round.Status = RoundStatusAgentThinking
	round.Thread = []Message{{
		Role:          "user",
		Content:       round.Submission.Text,
		AttachmentIDs: round.Submission.AttachmentIDs,
		CreatedAt:     now,
	}}
	if err := s.store.SaveRound(round); err != nil {
		_ = s.lock.Release(req.InitiativeName, holder.RunID)
		return Round{}, fmt.Errorf("save round: %w", err)
	}

	if s.spawner == nil {
		// Test / degraded mode: no agent spawn, just persist and flip to
		// awaiting_user so the caller can walk the rest of the lifecycle.
		round.Status = RoundStatusAwaitingUser
		round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
		if err := s.store.SaveRound(round); err != nil {
			return Round{}, err
		}
		_ = s.lock.Release(req.InitiativeName, holder.RunID)
		return round, nil
	}

	runID, err := s.spawner.SpawnInitiativeFeedback(ctx, SpawnRequest{
		InitiativeName: req.InitiativeName,
		RoundNumber:    number,
		RoundSlug:      slug,
		Purpose:        "feedback",
		SubmissionText: round.Submission.Text,
		AttachmentIDs:  round.Submission.AttachmentIDs,
	})
	if err != nil {
		_ = s.lock.Release(req.InitiativeName, holder.RunID)
		round.Status = RoundStatusAwaitingUser
		round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
		round.Thread = append(round.Thread, Message{
			Role:      "agent",
			Content:   fmt.Sprintf("agent spawn failed: %s", err.Error()),
			CreatedAt: round.UpdatedAt,
		})
		_ = s.store.SaveRound(round)
		return round, fmt.Errorf("spawn agent: %w", err)
	}
	round.RunID = runID
	round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
	if err := s.store.SaveRound(round); err != nil {
		slog.Warn("feedback: persist run id after spawn failed", "err", err, "run_id", runID)
	}
	// Swap the provisional holder for the real one so Inspect shows the
	// actual agent-manager run ID to the UI.
	holder.RunID = runID
	_ = s.lock.AcquireOverride(req.InitiativeName, holder)
	return round, nil
}

// ContinueRoundRequest carries a follow-up user message for an existing
// round. The round must be in RoundStatusAwaitingUser.
type ContinueRoundRequest struct {
	InitiativeName string
	RoundNumber    int
	Text           string
	AttachmentIDs  []string
	DecidedBy      string
}

// ContinueRound appends the user's message to the round's thread and
// asks the spawner to continue the agent run with the same RunID. The
// round flips back to RoundStatusAgentThinking until RecordAgentTurn is
// called with the agent's response.
func (s *Service) ContinueRound(ctx context.Context, req ContinueRoundRequest) (Round, error) {
	round, err := s.store.LoadRound(req.InitiativeName, req.RoundNumber)
	if err != nil {
		return Round{}, err
	}
	if round.Status != RoundStatusAwaitingUser {
		return Round{}, fmt.Errorf("round is in status %q; continue requires %q", round.Status, RoundStatusAwaitingUser)
	}
	if strings.TrimSpace(req.Text) == "" {
		return Round{}, errors.New("text is required")
	}

	now := s.clock().UTC().Format(time.RFC3339)
	round.Thread = append(round.Thread, Message{
		Role:          "user",
		Content:       strings.TrimSpace(req.Text),
		AttachmentIDs: append([]string(nil), req.AttachmentIDs...),
		CreatedAt:     now,
	})
	round.Status = RoundStatusAgentThinking
	round.UpdatedAt = now
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, fmt.Errorf("save round: %w", err)
	}

	// Re-acquire lock (we released it on the last turn).
	holder := Holder{
		RunID:       round.RunID,
		Purpose:     "feedback_continue",
		RoundNumber: round.Number,
		AcquiredBy:  req.DecidedBy,
	}
	if holder.RunID == "" {
		holder.RunID = fmt.Sprintf("continue-%d-%d", round.Number, time.Now().UnixNano())
	}
	if err := s.lock.AcquireOverride(req.InitiativeName, holder); err != nil {
		return Round{}, fmt.Errorf("acquire continue lock: %w", err)
	}

	if s.spawner != nil && round.RunID != "" {
		if err := s.spawner.ContinueRun(ctx, round.RunID, strings.TrimSpace(req.Text), req.AttachmentIDs); err != nil {
			_ = s.lock.Release(req.InitiativeName, holder.RunID)
			round.Status = RoundStatusAwaitingUser
			round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
			round.Thread = append(round.Thread, Message{
				Role:      "agent",
				Content:   fmt.Sprintf("continue-run failed: %s", err.Error()),
				CreatedAt: round.UpdatedAt,
			})
			_ = s.store.SaveRound(round)
			return round, fmt.Errorf("continue run: %w", err)
		}
	} else if s.spawner == nil {
		// Degraded mode: flip to awaiting_user immediately so test
		// harnesses can walk the lifecycle without an agent.
		round.Status = RoundStatusAwaitingUser
		round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
		_ = s.store.SaveRound(round)
		_ = s.lock.Release(req.InitiativeName, holder.RunID)
	}
	return round, nil
}

// RecordAgentTurn persists an agent-generated message into the round's
// thread and, when the message body carries a structured proposal JSON
// block, attaches it as a new ProposalRevision and makes it current.
//
// This is the inbound hook for the agent-manager listener: when the agent
// emits output on its run, the listener calls here with the raw text. The
// hook releases the lock and flips the round to awaiting_user — the user
// now owns the next move.
func (s *Service) RecordAgentTurn(initiativeName string, roundNumber int, body string) (Round, error) {
	round, err := s.store.LoadRound(initiativeName, roundNumber)
	if err != nil {
		return Round{}, err
	}
	if round.Status != RoundStatusAgentThinking {
		return Round{}, fmt.Errorf("round is in status %q; agent turn requires %q", round.Status, RoundStatusAgentThinking)
	}
	now := s.clock().UTC().Format(time.RFC3339)
	msg := Message{
		Role:      "agent",
		Content:   body,
		RunID:     round.RunID,
		CreatedAt: now,
	}

	extracted, warnings := extractProposal(body)
	if extracted != nil {
		revision := ProposalRevision{
			ID:            fmt.Sprintf("p%d", len(round.Proposals)+1),
			Proposal:      *extracted,
			CreatedAt:     now,
			ParseWarnings: warnings,
		}
		msg.ProposalID = revision.ID
		revision.MessageIndex = round.AppendThreadMessage(msg)
		round.Proposals = append(round.Proposals, revision)
		round.CurrentProposalID = revision.ID
	} else {
		round.AppendThreadMessage(msg)
	}
	round.Status = RoundStatusAwaitingUser
	round.UpdatedAt = now
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, err
	}
	_ = s.lock.Release(initiativeName, round.RunID)
	return round, nil
}

// DecideRequest is the user's terminal choice on the current proposal.
// Mutation IDs in AcceptedMutationIDs are applied (in order); anything not
// listed is dropped. For DecisionReject the applier is not invoked.
type DecideRequest struct {
	InitiativeName      string
	RoundNumber         int
	Kind                DecisionKind
	AcceptedMutationIDs []string
	Rationale           string
	DecidedBy           string
}

// Decide resolves the round to a terminal state. For DecisionAccept /
// DecisionPartialAccept the apply layer is invoked against the current
// proposal; the applied/failed counts are persisted on the decision's
// rationale for audit.
func (s *Service) Decide(ctx context.Context, req DecideRequest) (Round, *proposals.ApplyResult, error) {
	if req.Kind == "" {
		return Round{}, nil, errors.New("decision kind is required")
	}
	round, err := s.store.LoadRound(req.InitiativeName, req.RoundNumber)
	if err != nil {
		return Round{}, nil, err
	}
	if round.Status != RoundStatusAwaitingUser {
		return Round{}, nil, fmt.Errorf("round is in status %q; decide requires %q", round.Status, RoundStatusAwaitingUser)
	}

	now := s.clock().UTC().Format(time.RFC3339)
	decision := &Decision{
		Kind:                req.Kind,
		AcceptedMutationIDs: append([]string(nil), req.AcceptedMutationIDs...),
		Rationale:           req.Rationale,
		DecidedAt:           now,
		DecidedBy:           req.DecidedBy,
	}

	var applyResult *proposals.ApplyResult
	switch req.Kind {
	case DecisionAccept, DecisionPartialAccept:
		current := round.CurrentProposal()
		if current == nil {
			return Round{}, nil, errors.New("round has no current proposal to accept")
		}
		var err error
		applyResult, err = s.applyCurrentProposal(ctx, round, req.AcceptedMutationIDs, req.DecidedBy, now)
		if err != nil {
			return Round{}, nil, fmt.Errorf("apply proposal: %w", err)
		}
		decision.RejectedMutationIDs = computeRejected(current.Proposal, req.AcceptedMutationIDs)
		round.Status = RoundStatusApplied
	case DecisionReject, DecisionDismiss:
		round.Status = decisionToStatus(req.Kind)
	case DecisionRevise:
		// Revise isn't a terminal decision — it's handled via ContinueRound.
		// Reject here to keep the state machine legible.
		return Round{}, nil, errors.New("use ContinueRound for revisions; Decide requires a terminal decision")
	default:
		return Round{}, nil, fmt.Errorf("unknown decision kind %q", req.Kind)
	}

	round.Decision = decision
	round.UpdatedAt = now
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, applyResult, fmt.Errorf("save round: %w", err)
	}
	_ = s.lock.Release(req.InitiativeName, round.RunID)
	return round, applyResult, nil
}

func decisionToStatus(k DecisionKind) RoundStatus {
	switch k {
	case DecisionAccept, DecisionPartialAccept:
		return RoundStatusApplied
	case DecisionReject:
		return RoundStatusRejected
	case DecisionDismiss:
		return RoundStatusDismissed
	}
	return RoundStatusDismissed
}

func computeRejected(p proposals.Proposal, accepted []string) []string {
	set := make(map[string]struct{}, len(accepted))
	for _, id := range accepted {
		set[id] = struct{}{}
	}
	out := make([]string, 0)
	for _, m := range p.Mutations {
		if _, ok := set[m.ID]; ok {
			continue
		}
		out = append(out, m.ID)
	}
	return out
}

func isValidRoundType(t RoundType) bool {
	switch t {
	case RoundTypeFeedback, RoundTypeResearch, RoundTypeNote:
		return true
	}
	return false
}

// proposalBlockRE finds the first ```json ... ``` code block that parses as
// a Proposal envelope. Lenient enough to tolerate leading prose, strict on
// the parsed JSON shape.
var proposalBlockRE = regexp.MustCompile("(?s)```json\\s*(.*?)\\s*```")

// extractProposal pulls a JSON proposal block out of an agent message, if
// present. Returns nil + warnings when the block is missing or doesn't
// parse — the round still records the turn so the user can ask for a
// revision, which is the documented failure mode.
func extractProposal(body string) (*proposals.Proposal, []string) {
	matches := proposalBlockRE.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil, nil
	}
	var warnings []string
	for _, m := range matches {
		raw := strings.TrimSpace(m[1])
		if !strings.HasPrefix(raw, "{") {
			continue
		}
		var p proposals.Proposal
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			warnings = append(warnings, fmt.Sprintf("parse proposal block: %s", err.Error()))
			continue
		}
		return &p, warnings
	}
	return nil, warnings
}

// deriveSlugFromText produces a slug from a free-form submission. Falls
// back to "round" when the text contains no word characters.
func deriveSlugFromText(text string) string {
	s := Sanitize(text)
	if s == "" {
		return "round"
	}
	// Trim to the first few dash-separated tokens for a readable folder name.
	tokens := strings.Split(s, "-")
	if len(tokens) > 5 {
		tokens = tokens[:5]
	}
	return strings.Join(tokens, "-")
}

// ComputeSlug derives the canonical slug a round will get, given a slug
// hint and submission text. Exposed so callers that need to place files
// on disk *before* StartRound runs (HTTP multipart handlers) can agree on
// the same folder name the service will pick.
func ComputeSlug(slugHint, text string) string {
	if s := Sanitize(slugHint); s != "" {
		return s
	}
	return deriveSlugFromText(text)
}
