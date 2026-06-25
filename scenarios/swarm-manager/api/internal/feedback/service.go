package feedback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"swarm-manager/internal/initiativelock"
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

// RunCanceller stops an in-flight agent-manager run. Used by the override
// path: when the user confirms preemption via the warning dialog, the
// service calls StopRun on the current holder and on any blocking item
// runs so "single agent per initiative" is actually enforced, instead of
// just being a lock-file rename while two agents keep running in the
// background. Nil is acceptable (e.g. tests) — the override still takes
// the lock, we just skip the cancel call and rely on the caller to know
// they've left a zombie.
type RunCanceller interface {
	StopRun(ctx context.Context, runID string) error
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
// may surface to the user. Wire format is snake_case so the UI can render
// it without a camel-case translation step.
type ItemActivity struct {
	Ref     string `json:"ref"` // "kind/name"
	RunID   string `json:"run_id,omitempty"`
	Purpose string `json:"purpose,omitempty"`
}

// ErrInitiativeBusy is returned by StartRound when one or more items in the
// initiative have a live agent run and the caller did not pass Override.
// The error wraps the list of blockers so handlers can render them.
var ErrInitiativeBusy = errors.New("initiative has active item-level agent runs")

// ErrProposalValidation signals that a proposal parsed successfully but is
// not valid against the current proposal contract/state, so the user must ask
// the agent for a revision instead of attempting apply.
var ErrProposalValidation = errors.New("proposal is invalid and must be revised")

// BusyError carries the list of blocking item activities for the override
// warning dialog. Caller can errors.As to it to extract the details.
type BusyError struct {
	Activities []ItemActivity
}

func (e *BusyError) Error() string {
	return ErrInitiativeBusy.Error()
}

func (e *BusyError) Unwrap() error { return ErrInitiativeBusy }

// ProposalValidationError carries validation errors for a parseable proposal
// that cannot be reviewed/applied as-is.
type ProposalValidationError struct {
	ProposalID       string
	ValidationErrors []string
}

func (e *ProposalValidationError) Error() string {
	return ErrProposalValidation.Error()
}

func (e *ProposalValidationError) Unwrap() error { return ErrProposalValidation }

// Service orchestrates feedback round lifecycle.
type Service struct {
	store        *Store
	lock         *initiativelock.Lock
	spawner      AgentSpawner
	activity     ItemActivityChecker
	poller       AgentRunPoller
	canceller    RunCanceller
	apply        *proposals.Applier
	StateBuilder func(initiativeName string) (proposals.CurrentState, error)
	clock        func() time.Time
}

// Config bundles Service dependencies.
type Config struct {
	Store        *Store
	Lock         *initiativelock.Lock
	Spawner      AgentSpawner
	Activity     ItemActivityChecker
	Poller       AgentRunPoller
	Canceller    RunCanceller
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
		canceller:    cfg.Canceller,
		apply:        cfg.Apply,
		StateBuilder: cfg.StateBuilder,
		clock:        clk,
	}, nil
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
	round.LastValidationErrors = nil
	round.UpdatedAt = now
	if err := s.store.SaveRound(round); err != nil {
		return Round{}, fmt.Errorf("save round: %w", err)
	}

	// Re-acquire lock (we released it on the last turn).
	holder := initiativelock.Holder{
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
			if relErr := s.lock.Release(req.InitiativeName, holder.RunID); relErr != nil {
				slog.Warn("feedback: release lock failed", "err", relErr, "initiative", req.InitiativeName, "run_id", holder.RunID)
			}
			round.Status = RoundStatusAwaitingUser
			round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
			round.Thread = append(round.Thread, Message{
				Role:      "agent",
				Content:   fmt.Sprintf("continue-run failed: %s", err.Error()),
				CreatedAt: round.UpdatedAt,
			})
			if saveErr := s.store.SaveRound(round); saveErr != nil {
				slog.Warn("feedback: persist round failed", "err", saveErr, "initiative", round.InitiativeName, "round", round.Number)
			}
			return round, fmt.Errorf("continue run: %w", err)
		}
	} else if s.spawner == nil {
		// Degraded mode: flip to awaiting_user immediately so test
		// harnesses can walk the lifecycle without an agent.
		round.Status = RoundStatusAwaitingUser
		round.UpdatedAt = s.clock().UTC().Format(time.RFC3339)
		if saveErr := s.store.SaveRound(round); saveErr != nil {
			slog.Warn("feedback: persist round failed", "err", saveErr, "initiative", round.InitiativeName, "round", round.Number)
		}
		if relErr := s.lock.Release(req.InitiativeName, holder.RunID); relErr != nil {
			slog.Warn("feedback: release lock failed", "err", relErr, "initiative", req.InitiativeName, "run_id", holder.RunID)
		}
	}
	return round, nil
}
