package feedback

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"swarm-manager/internal/proposals"
)

// AgentSpawner is the injected interface the service uses to start or
// continue an agent-manager run for the feedback round.
type AgentSpawner interface {
	SpawnInitiativeFeedback(ctx context.Context, req SpawnRequest) (string, error)
	ContinueRun(ctx context.Context, req ContinueRequest) error
}

// SpawnRequest carries everything the spawner needs to start a feedback
// agent run.
type SpawnRequest struct {
	InitiativeName string
	RoundNumber    int
	RoundSlug      string
	Purpose        string // "feedback" | "feedback_continue" | "review"
	SubmissionText string
	AttachmentIDs  []string
}

// ContinueRequest carries the same identifying context the spawner used
// at start so the continuation can be tagged into the activity log under
// the same initiative + round.
type ContinueRequest struct {
	InitiativeName string
	RoundNumber    int
	RoundSlug      string
	RunID          string
	Message        string
	AttachmentIDs  []string
}

// AgentRunPoller resolves run lifecycle so the feedback service can pull
// agent output without an inbound webhook. Mirrors the clarification pull
// pattern: when round.Status==agent_thinking, the service asks the poller
// for the run state and, if terminal, records the agent turn.
type AgentRunPoller interface {
	IsEnabled() bool
	GetRunState(ctx context.Context, runID string) (RunState, error)
}

// RunState is the subset of agent-manager run state the feedback service
// needs to drive the pull-based agent-turn flow.
type RunState struct {
	Status   string
	Summary  string
	ErrorMsg string
}

// ItemActivityChecker reports whether any of the initiative's items has an
// active agent run. The service consults this before acquiring the
// initiative lock so callers can surface blocking state to the override
// dialog instead of failing opaquely.
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

// ErrInitiativeBusy is returned by StartRound when one or more items in the
// initiative have a live agent run and the caller did not pass Override.
// The error wraps the list of blockers so handlers can render them.
var ErrInitiativeBusy = errors.New("initiative has active item-level agent runs")

// BusyError carries the list of blocking item activities for the override
// warning dialog. Caller can errors.As to it to extract the details.
type BusyError struct {
	Activities []ItemActivity
}

func (e *BusyError) Error() string {
	return ErrInitiativeBusy.Error()
}

func (e *BusyError) Unwrap() error { return ErrInitiativeBusy }

// Service orchestrates feedback round lifecycle.
type Service struct {
	store        *Store
	lock         *Lock
	spawner      AgentSpawner
	activity     ItemActivityChecker
	poller       AgentRunPoller
	apply        *proposals.Applier
	StateBuilder func(initiativeName string) (proposals.CurrentState, error)
	clock        func() time.Time
}

// Config bundles Service dependencies.
type Config struct {
	Store        *Store
	Lock         *Lock
	Spawner      AgentSpawner
	Activity     ItemActivityChecker
	Poller       AgentRunPoller
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
		poller:       cfg.Poller,
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

	// AttachmentLoader, when set, runs inside StartRound after the round
	// directory has been reserved (atomic) but before the round is
	// persisted. It must populate `roundDir` with attachment files and
	// return their relative IDs. Set by handlers serving multipart
	// uploads so attachments land in the dir the service actually
	// claimed — fixes the race where two concurrent submits would
	// otherwise compute the same predicted number and clobber each
	// other's attachments.
	AttachmentLoader func(roundDir string) ([]string, error)
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

	slug := ComputeSlug(req.SlugHint, req.Text)
	number, roundDir, err := s.store.ReserveRound(req.InitiativeName, slug)
	if err != nil {
		return Round{}, fmt.Errorf("reserve round: %w", err)
	}

	// Run the attachment loader against the reserved dir so multipart
	// callers can drop files in the slot the service actually claimed.
	attachmentIDs := append([]string(nil), req.AttachmentIDs...)
	if req.AttachmentLoader != nil {
		ids, loadErr := req.AttachmentLoader(roundDir)
		if loadErr != nil {
			// Best-effort cleanup: remove the reserved dir so the
			// number we claimed gets reused on the next attempt.
			_ = os.RemoveAll(roundDir)
			return Round{}, fmt.Errorf("load attachments: %w", loadErr)
		}
		attachmentIDs = append(attachmentIDs, ids...)
	}

	now := s.clock().UTC().Format(time.RFC3339)
	round := Round{
		InitiativeName: req.InitiativeName,
		Number:         number,
		Slug:           slug,
		Type:           req.Type,
		Status:         RoundStatusSubmitting,
		Submission: Submission{
			Text:          strings.TrimSpace(req.Text),
			AttachmentIDs: attachmentIDs,
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

	// Pre-flight: surface item-level activity so the override dialog can
	// list specific blockers rather than failing opaquely. Override skips
	// the check (the user already saw the warning).
	if !req.Override && s.activity != nil {
		busy, actErr := s.activity.ActiveRunsForInitiative(req.InitiativeName)
		if actErr != nil {
			slog.Warn("feedback: item activity check failed", "err", actErr, "initiative", req.InitiativeName)
		} else if len(busy) > 0 {
			return Round{}, &BusyError{Activities: busy}
		}
	}

	// Feedback rounds acquire the lock before any agent side-effect runs
	// so concurrent submissions can't race us into two spawns. The
	// provisional RunID is swapped for the agent-manager RunID once the
	// spawn succeeds; on spawn failure, Release uses the provisional to
	// clear the lock rather than leaving it wedged.
	holder := Holder{
		Purpose:     "feedback",
		RoundNumber: number,
		AcquiredBy:  req.DecidedBy,
	}
	if s.spawner != nil {
		holder.RunID = fmt.Sprintf("provisional-%d-%d", number, time.Now().UnixNano())
	} else {
		holder.RunID = fmt.Sprintf("no-spawner-%d", number)
	}
	provisionalRunID := holder.RunID
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
		_ = s.lock.Release(req.InitiativeName, provisionalRunID)
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
		_ = s.lock.Release(req.InitiativeName, provisionalRunID)
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
		_ = s.lock.Release(req.InitiativeName, provisionalRunID)
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
	// Swap the provisional holder for the real run id so Inspect surfaces
	// the actual agent-manager run to the UI. If the swap fails — rare,
	// since AcquireOverride is a pure file write — fall back to releasing
	// the provisional so the initiative isn't wedged. We lose the lock
	// guarantee for the duration of this run, but that's a strictly
	// better failure mode than a stuck lock.
	holder.RunID = runID
	if swapErr := s.lock.AcquireOverride(req.InitiativeName, holder); swapErr != nil {
		slog.Warn("feedback: lock run-id swap failed; releasing provisional",
			"err", swapErr,
			"initiative", req.InitiativeName,
			"provisional", provisionalRunID,
			"run_id", runID,
		)
		_ = s.lock.Release(req.InitiativeName, provisionalRunID)
	}
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
	// User asking for a revision clears the structured parse-error signal;
	// the next agent turn either produces a valid proposal (success) or
	// lands back in NeedsRevision (failure), but the current signal is
	// now stale.
	round.NeedsRevision = false
	round.LastParseWarnings = nil
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
		contReq := ContinueRequest{
			InitiativeName: req.InitiativeName,
			RoundNumber:    round.Number,
			RoundSlug:      round.Slug,
			RunID:          round.RunID,
			Message:        strings.TrimSpace(req.Text),
			AttachmentIDs:  req.AttachmentIDs,
		}
		if err := s.spawner.ContinueRun(ctx, contReq); err != nil {
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

// EnsurePolledTurn checks the agent run state for a round in
// RoundStatusAgentThinking and, if the run has reached a terminal state,
// records the agent's output as a turn. Mirrors the clarification pull
// pattern (clarification_state.go:GetClarification) so the UI can advance
// rounds by polling rather than depending on an inbound webhook from the
// agent-manager.
//
// Returns the (possibly updated) round. Callers — typically Handler.Get —
// should invoke this whenever they hand a round to the user. Idempotent
// and safe under poll storms: rounds not in agent_thinking, runs without a
// RunID, missing pollers, and non-terminal run states are no-ops.
func (s *Service) EnsurePolledTurn(ctx context.Context, round Round) (Round, error) {
	if round.Status != RoundStatusAgentThinking {
		return round, nil
	}
	if s.poller == nil || !s.poller.IsEnabled() || strings.TrimSpace(round.RunID) == "" {
		return round, nil
	}
	state, err := s.poller.GetRunState(ctx, round.RunID)
	if err != nil {
		// Polling failures aren't fatal — the round stays in
		// agent_thinking and the next poll will retry. Log so operators
		// can spot wedged runs.
		slog.Warn("feedback: poll run state failed",
			"err", err, "initiative", round.InitiativeName, "round", round.Number, "run_id", round.RunID)
		return round, nil
	}
	if !isTerminalRunStatus(state.Status) {
		return round, nil
	}

	body := strings.TrimSpace(state.Summary)
	if isFailureRunStatus(state.Status) {
		msg := strings.TrimSpace(state.ErrorMsg)
		if msg == "" {
			msg = "agent run failed without an error message"
		}
		body = "agent run failed: " + msg
	}
	if body == "" {
		body = "agent run completed without producing output"
	}
	return s.RecordAgentTurn(round.InitiativeName, round.Number, body)
}

// isTerminalRunStatus matches the clarification_service.isTerminalStatus
// list. Kept private to feedback so the package doesn't import backlog.
func isTerminalRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "complete", "completed", "success", "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
}

func isFailureRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "cancelled", "canceled":
		return true
	}
	return false
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
	msg.ParseWarnings = append([]string(nil), warnings...)
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
		round.NeedsRevision = false
		round.LastParseWarnings = nil
	} else {
		round.AppendThreadMessage(msg)
		// No extractable proposal: the round returns to the user with a
		// structured "revision needed" signal. The UI reads NeedsRevision
		// to render the ask-for-revision CTA; warnings surface why.
		round.NeedsRevision = true
		round.LastParseWarnings = append([]string(nil), warnings...)
		if len(round.LastParseWarnings) == 0 {
			round.LastParseWarnings = []string{"agent output did not contain a parseable proposal JSON block"}
		}
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

// fencedProposalBlockRE finds a fenced code block whose info string is
// case-insensitively "json" (with optional surrounding whitespace). Used
// as the first-pass extractor before falling back to looser strategies.
var fencedProposalBlockRE = regexp.MustCompile("(?si)```\\s*json\\b[^\\n]*\\n(.*?)```")

// genericFencedBlockRE finds any fenced block, including ones with no
// language tag or a non-json language. Used after the json-fenced pass
// fails so prose like ```\n{...}\n``` or ```yaml-like-but-actually-json
// still parses.
var genericFencedBlockRE = regexp.MustCompile("(?s)```[^\\n]*\\n(.*?)```")

// proposalSentinelRE matches `PROPOSAL:` (case-insensitive) followed by an
// optional fence then a JSON object. Lets agents use the explicit
// sentinel pattern documented in the skill prompt.
var proposalSentinelRE = regexp.MustCompile(`(?si)PROPOSAL\s*:[^\{]*?(\{.*\})`)

// extractProposal pulls a JSON proposal envelope out of an agent message
// using a lenient-then-strict strategy:
//  1. ```json fenced blocks (case-insensitive on the language tag)
//  2. any fenced block whose contents start with `{`
//  3. a `PROPOSAL:` sentinel followed by a JSON object
//  4. the first balanced `{...}` substring in the message
//
// All four strategies feed the same parser, so a single message can be
// noisy as long as one extraction succeeds. Returns nil + warnings when
// no extraction parses — the round still records the turn so the user
// can ask for a revision, which is the documented failure mode.
func extractProposal(body string) (*proposals.Proposal, []string) {
	if strings.TrimSpace(body) == "" {
		return nil, nil
	}
	var warnings []string

	tryParse := func(raw string) (*proposals.Proposal, bool) {
		raw = strings.TrimSpace(raw)
		if !strings.HasPrefix(raw, "{") {
			return nil, false
		}
		var p proposals.Proposal
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			warnings = append(warnings, fmt.Sprintf("parse proposal block: %s", err.Error()))
			return nil, false
		}
		return &p, true
	}

	// Strategy 1: ```json fenced blocks.
	for _, m := range fencedProposalBlockRE.FindAllStringSubmatch(body, -1) {
		if p, ok := tryParse(m[1]); ok {
			return p, warnings
		}
	}

	// Strategy 2: any fenced block.
	for _, m := range genericFencedBlockRE.FindAllStringSubmatch(body, -1) {
		if p, ok := tryParse(m[1]); ok {
			return p, warnings
		}
	}

	// Strategy 3: PROPOSAL: sentinel.
	if m := proposalSentinelRE.FindStringSubmatch(body); len(m) == 2 {
		if balanced := extractFirstBalancedJSON(m[1]); balanced != "" {
			if p, ok := tryParse(balanced); ok {
				return p, warnings
			}
		}
	}

	// Strategy 4: any balanced JSON object in the body — last resort,
	// useful when the agent emits raw JSON with no markdown wrapping.
	if balanced := extractFirstBalancedJSON(body); balanced != "" {
		if p, ok := tryParse(balanced); ok {
			return p, warnings
		}
	}

	return nil, warnings
}

// extractFirstBalancedJSON returns the substring starting at the first '{'
// up to (and including) the matching closing '}'. Counts balanced braces
// honoring strings and escapes. Returns "" when no balanced object is
// found. Tolerates leading prose, attribute lists, etc.
func extractFirstBalancedJSON(s string) string {
	start := strings.IndexByte(s, '{')
	if start < 0 {
		return ""
	}
	depth := 0
	inString := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inString {
			escape = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
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
