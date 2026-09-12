package development

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	"swarm-manager/internal/identity"
)

// WorkValidator resolves an existing work item and rejects an incompatible plan
// engagement. Admission never creates a backlog item as an incidental effect.
type (
	WorkValidator          func(context.Context, string) error
	PlanReferenceValidator func(context.Context, string, PlanReference) error
)

type Service struct {
	repo         Repository
	reviewer     Reviewer
	validateWork WorkValidator
	validatePlan PlanReferenceValidator
	evidence     EvidenceResolver
	launchBlocks func(context.Context) []string
	now          func() time.Time
}

// SetPlanReferenceValidator binds approval to the backlog item's canonical
// Plan Manager reference. Keeping this composition-owned avoids making the
// development domain read backlog storage directly.
func (s *Service) SetPlanReferenceValidator(validate PlanReferenceValidator) {
	if s != nil {
		s.validatePlan = validate
	}
}

func NewService(repo Repository, reviewer Reviewer, validate WorkValidator, evidence EvidenceResolver) *Service {
	return &Service{repo: repo, reviewer: reviewer, validateWork: validate, evidence: evidence, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Get(ctx context.Context, item string) (Engagement, error) {
	return s.repo.Get(ctx, item)
}

func (s *Service) Snapshot(ctx context.Context, digest string) (Snapshot, error) {
	return s.repo.Snapshot(ctx, digest)
}

// SetLaunchBlockerProjection installs the composition-owned readiness read.
// The development domain retains the evidence resolver blocker, while the
// control plane supplies live transition and integration standing.
func (s *Service) SetLaunchBlockerProjection(projection func(context.Context) []string) {
	s.launchBlocks = projection
}

func (s *Service) LaunchBlockers(ctx context.Context) []string {
	if s != nil && s.launchBlocks != nil {
		return append([]string(nil), s.launchBlocks(ctx)...)
	}
	return RuntimeBlockers()
}

// Approve captures the very bytes checked against reviewedDigest. On amendment
// it preserves incurred/reserved usage and history. Approval never dispatches.
func (s *Service) Approve(ctx context.Context, p Proposal, reviewedDigest string, expected int64, reason string) (Engagement, error) {
	var state Engagement
	p.BudgetPolicy = defaultBudgetPolicy(p.BudgetPolicy)
	actor, err := operator(ctx)
	if err != nil {
		return state, err
	}
	if !validItem(p.WorkItem) || expected < 0 || len(reason) > 4096 || strings.TrimSpace(reason) == "" {
		return state, fmt.Errorf("canonical item, expected version and approval rationale required: %w", ErrInvalid)
	}
	p.PlanRef = normalizePlanReference(p.PlanRef)
	if err := validatePlanReference(p.PlanRef); err != nil {
		return state, fmt.Errorf("adaptive development requires a canonical plan: %w", ErrInvalid)
	}
	if strings.TrimSpace(p.ExecutionStrategy) == "" {
		p.ExecutionStrategy = AdaptiveStrategy
	}
	if p.ExecutionStrategy != AdaptiveStrategy {
		return state, fmt.Errorf("development approval requires the adaptive-improvement strategy: %w", ErrInvalid)
	}
	if s.validateWork == nil {
		return state, fmt.Errorf("work owner is unavailable: %w", ErrDenied)
	}
	if err := s.validateWork(ctx, p.WorkItem); err != nil {
		return state, err
	}
	if s.validatePlan != nil {
		if err := s.validatePlan(ctx, p.WorkItem, *p.PlanRef); err != nil {
			return state, err
		}
	}
	review, err := s.reviewer.Preview(p)
	if err != nil {
		return state, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if !review.ReviewComplete {
		return state, fmt.Errorf("proposal has unresolved review findings: %w", ErrInvalid)
	}
	if reviewedDigest == "" || review.ProposalDigest != reviewedDigest {
		return state, ErrConflict
	}
	state, err = s.repo.Get(ctx, p.WorkItem)
	if errors.Is(err, ErrNotFound) && expected == 0 {
		state = Engagement{WorkItem: p.WorkItem, WorkShape: "contract-development"}
	} else if err != nil {
		return state, err
	}
	if state.Version != expected {
		return state, ErrConflict
	}
	if state.PlanRef != nil && !planReferencesEqual(state.PlanRef, p.PlanRef) {
		return state, fmt.Errorf("approval cannot move an engagement to a different plan: %w", ErrConflict)
	}
	if len(state.Approvals) >= 128 {
		return state, fmt.Errorf("approval history limit reached; operator archival design required: %w", ErrDenied)
	}
	if state.Status == "accepted" || activeAttempt(state) != nil {
		return state, fmt.Errorf("settle the active attempt before amending: %w", ErrDenied)
	}
	limit := Usage{p.MaxTokens, p.MaxWallSeconds}
	if !state.Used.fits(limit) {
		return state, fmt.Errorf("amendment cannot erase incurred usage: %w", ErrInvalid)
	}
	state.Version++
	state.PlanRef = p.PlanRef
	state.Digest, state.Status, state.StopReason = reviewedDigest, "approved", ""
	state.Evidence = nil
	state.Approvals = append(state.Approvals, Approval{Digest: reviewedDigest, Actor: actor, Reason: reason, At: s.now()})
	state.Campaign = NewCampaignCheckpoint(reviewedDigest, p.Outcomes, "")
	snapshot := Snapshot{Digest: reviewedDigest, Proposal: p, Artifacts: review.Artifacts, Contents: review.Contents, GoalMessage: review.GoalMessage}
	if err := s.repo.Commit(ctx, expected, state, &snapshot); err != nil {
		return Engagement{}, err
	}
	return state, nil
}

// Reserve is an internal dispatch-adapter operation, not a public mutation.
// The adapter must reserve the workflow owner's actual upper bounds, not accept
// a budget claimed by the agent. Native and fallback starts share this ledger.
func (s *Service) Reserve(ctx context.Context, item, digest, key, mode string, upper Usage) (Engagement, error) {
	state, _, err := s.reserve(ctx, item, digest, key, mode, upper)
	return state, err
}

// ReserveOwned reports whether this caller created the durable reservation.
// The ownership bit comes from the repository compare-and-swap result; a
// caller must never infer ownership from versions observed before Reserve.
func (s *Service) ReserveOwned(ctx context.Context, item, digest, key, mode string, upper Usage) (Engagement, bool, error) {
	return s.reserve(ctx, item, digest, key, mode, upper)
}

func (s *Service) reserve(ctx context.Context, item, digest, key, mode string, upper Usage) (Engagement, bool, error) {
	state, err := s.repo.Get(ctx, item)
	if err != nil {
		return state, false, err
	}
	if digest != state.Digest {
		return state, false, ErrConflict
	}
	if !slugPattern.MatchString(key) || len(key) > 128 || (mode != "native-goal" && mode != "workflow-fallback") || upper.Tokens <= 0 || upper.WallSeconds <= 0 {
		return state, false, ErrInvalid
	}
	for _, attempt := range state.Attempts {
		if attempt.Key == key {
			if attempt.Digest != digest || attempt.Mode != mode || attempt.Reserved != upper {
				return state, false, ErrConflict
			}
			// A retry can reconcile an existing reservation, never start another.
			return state, false, nil
		}
	}
	if state.Status != "approved" && state.Status != "paused" {
		return state, false, fmt.Errorf("engagement status %q cannot start: %w", state.Status, ErrDenied)
	}
	if activeAttempt(state) != nil {
		return state, false, ErrConflict
	}
	if len(state.Attempts) >= 1000 {
		return state, false, fmt.Errorf("attempt history limit reached: %w", ErrDenied)
	}
	snapshot, err := s.repo.Snapshot(ctx, digest)
	if err != nil {
		return state, false, err
	}
	remaining := Usage{snapshot.Proposal.MaxTokens - state.Used.Tokens, snapshot.Proposal.MaxWallSeconds - state.Used.WallSeconds}
	if !upper.fits(remaining) {
		return state, false, fmt.Errorf("aggregate budget exhausted: %w", ErrDenied)
	}
	if state.Campaign.ApprovalDigest != digest || len(state.Campaign.RequiredOutcomeIDs) == 0 {
		state.Campaign = NewCampaignCheckpoint(digest, snapshot.Proposal.Outcomes, key)
	} else {
		state.Campaign.AttemptKey = key
		state.Campaign.OwnerExecutionID = ""
		state.Campaign.Pending = true
		state.Campaign.Digest = checkpointDigest(state.Campaign)
	}
	expected := state.Version
	state.Version++
	state.Status, state.Reserved = "running", upper
	state.Attempts = append(state.Attempts, Attempt{Key: key, Digest: digest, Mode: mode, Reserved: upper, StartedAt: s.now()})
	committed, err := s.commit(ctx, expected, state)
	return committed, err == nil, err
}

// BindExecution may be retried after a lost response. A different owner execution
// for the same reservation is refused, not accounted as a free continuation.
func (s *Service) BindExecution(ctx context.Context, item, key, executionID string, workflowDigest ...string) (Engagement, error) {
	state, err := s.repo.Get(ctx, item)
	if err != nil {
		return state, err
	}
	if strings.TrimSpace(executionID) == "" || len(executionID) > 200 {
		return state, ErrInvalid
	}
	attempt := activeAttempt(state)
	if attempt == nil || attempt.Key != key {
		return state, ErrConflict
	}
	if attempt.ExecutionID != "" {
		if attempt.ExecutionID != executionID {
			return state, ErrConflict
		}
		return state, nil
	}
	expected := state.Version
	state.Version++
	for i := range state.Attempts {
		if state.Attempts[i].Key == key {
			if len(workflowDigest) > 0 {
				state.Attempts[i].WorkflowDigest = strings.TrimSpace(workflowDigest[0])
			}
			if len(workflowDigest) > 1 {
				state.Attempts[i].GrantDigest = strings.TrimSpace(workflowDigest[1])
			}
			if len(workflowDigest) > 2 {
				state.Attempts[i].CapabilityRevision = strings.TrimSpace(workflowDigest[2])
			}
			if len(workflowDigest) > 3 {
				state.Attempts[i].SelectionReason = strings.TrimSpace(workflowDigest[3])
			}
			state.Attempts[i].ExecutionID = executionID
		}
	}
	state.Campaign = state.Campaign.withOwner(executionID)
	return s.commit(ctx, expected, state)
}

// Settle consumes terminal usage fetched by the adapter from Agent Manager.
// Unknown usage must not call this method: the reservation remains held until
// reconciliation succeeds. Even a cancelled or over-budget attempt is charged.
func (s *Service) Settle(ctx context.Context, item, key, executionID string, measured *Usage, checkpoint string) (Engagement, error) {
	state, err := s.repo.Get(ctx, item)
	if err != nil {
		return state, err
	}
	if measured == nil {
		return state, fmt.Errorf("owner usage is unknown; reservation remains held: %w", ErrDenied)
	}
	used := *measured
	if !used.valid() || len(checkpoint) > 32768 {
		return state, ErrInvalid
	}
	index := -1
	for i, a := range state.Attempts {
		if a.Key == key {
			index = i
			break
		}
	}
	if index < 0 {
		return state, ErrNotFound
	}
	a := &state.Attempts[index]
	if executionID == "" || a.ExecutionID != executionID {
		return state, ErrConflict
	}
	if a.SettledAt != nil {
		if a.Used != used || a.Checkpoint != checkpoint {
			return state, ErrConflict
		}
		return state, nil
	}
	if state.Used.Tokens > math.MaxInt64-used.Tokens || state.Used.WallSeconds > math.MaxInt64-used.WallSeconds {
		return state, ErrInvalid
	}
	expected := state.Version
	state.Version++
	now := s.now()
	a.SettledAt, a.Used, a.Checkpoint = &now, used, checkpoint
	state.Campaign = state.Campaign.settled(executionID, checkpoint)
	state.Used.Tokens += used.Tokens
	state.Used.WallSeconds += used.WallSeconds
	state.Reserved, state.Checkpoint = Usage{}, checkpoint
	if state.Status != "cancelled" {
		state.Status = "paused"
	}
	if !used.fits(a.Reserved) {
		snapshot, err := s.repo.Snapshot(ctx, a.Digest)
		if err != nil {
			return Engagement{}, err
		}
		if snapshot.Proposal.BudgetPolicy == BudgetMetered && used.WallSeconds <= a.Reserved.WallSeconds && state.Status != "cancelled" {
			state.StopReason = "Metered cancellation overshoot retained in incurred usage; only the remaining aggregate allowance may fund another attempt."
			if state.Used.Tokens >= snapshot.Proposal.MaxTokens {
				state.Status, state.StopReason = "budget_exhausted", "Metered cancellation overshoot retained in incurred usage; no further dispatch is authorized."
			}
		} else {
			state.Status, state.StopReason = "cancelled", "Owner usage exceeded its reservation; operator review is required."
		}
	}
	return s.commit(ctx, expected, state)
}

// Revoke immediately closes new dispatch. It does not claim the owner stopped:
// an active reservation remains until owner cancellation and usage reconciliation.
func (s *Service) Revoke(ctx context.Context, item string, expected int64, reason string) (Engagement, error) {
	if _, err := operator(ctx); err != nil {
		return Engagement{}, err
	}
	state, err := s.repo.Get(ctx, item)
	if err != nil {
		return state, err
	}
	if state.Version != expected {
		return state, ErrConflict
	}
	if state.Status == "accepted" || strings.TrimSpace(reason) == "" || len(reason) > 4096 {
		return state, ErrInvalid
	}
	if state.Status == "cancelled" && state.Cancellation != nil && state.Cancellation.Reason == reason {
		return state, nil
	}
	state.Version++
	state.Status, state.StopReason = "cancelled", reason
	state.Cancellation = &CancellationIntent{OperationID: fmt.Sprintf("cancel/%s/%d", item, state.Version), Reason: reason, State: "requested", RequestedAt: s.now()}
	return s.commit(ctx, expected, state)
}

// AcknowledgeCancellation records the owner's acceptance of the stable stop
// operation. Terminal execution and final usage still require reconciliation.
func (s *Service) AcknowledgeCancellation(ctx context.Context, item string, expected int64, operationID string) (Engagement, error) {
	state, err := s.repo.Get(ctx, item)
	if err != nil {
		return state, err
	}
	if state.Version != expected || state.Cancellation == nil || state.Cancellation.OperationID != operationID {
		return state, ErrConflict
	}
	if state.Cancellation.State == "acknowledged" {
		return state, nil
	}
	state.Version++
	now := s.now()
	state.Cancellation.State, state.Cancellation.AckAt = "acknowledged", &now
	return s.commit(ctx, expected, state)
}

// Accept independently resolves each protected outcome, including revision and
// execution binding. It never interprets a successful agent exit as acceptance.
// Only references are accepted from callers; passed/status are owner facts.
func (s *Service) Accept(ctx context.Context, item string, expected int64, refs map[string]string) (Engagement, error) {
	actor, err := operator(ctx)
	if err != nil {
		return Engagement{}, err
	}
	state, err := s.repo.Get(ctx, item)
	if err != nil {
		return state, err
	}
	if state.Version != expected {
		return state, ErrConflict
	}
	if (state.Status != "paused" && state.Status != "budget_exhausted") || activeAttempt(state) != nil {
		return state, ErrDenied
	}
	if s.evidence == nil {
		return state, fmt.Errorf("authoritative outcome resolver unavailable: %w", ErrDenied)
	}
	snapshot, err := s.repo.Snapshot(ctx, state.Digest)
	if err != nil {
		return state, err
	}
	if state.Used.WallSeconds > snapshot.Proposal.MaxWallSeconds || (snapshot.Proposal.BudgetPolicy != BudgetMetered && state.Used.Tokens > snapshot.Proposal.MaxTokens) {
		return state, ErrDenied
	}
	if len(refs) != len(snapshot.Proposal.Outcomes) {
		return state, fmt.Errorf("every protected outcome requires evidence: %w", ErrInvalid)
	}
	revision, err := s.evidence.CurrentRevision(ctx, snapshot.Proposal.Scenario)
	if err != nil || revision == "" {
		return state, fmt.Errorf("current product revision cannot be verified: %w", ErrDenied)
	}
	var accepted []Evidence
	for _, outcome := range snapshot.Proposal.Outcomes {
		ref := refs[outcome.ID]
		if ref == "" || len(ref) > 2048 {
			return state, fmt.Errorf("missing evidence for %s: %w", outcome.ID, ErrInvalid)
		}
		evidence, err := s.evidence.Resolve(ctx, outcome.EvidenceSource, ref)
		if err != nil {
			return state, fmt.Errorf("resolve %s evidence: %w", outcome.ID, err)
		}
		contract, contractErr := ParseEvidenceContract(outcome.EvidenceSource)
		now := s.now()
		if contractErr != nil || !evidence.Passed || evidence.Status != "passed" || evidence.OutcomeID != outcome.ID || evidence.Source != outcome.EvidenceSource || evidence.ResolverID != contract.ResolverID || evidence.ReceiptSchema != contract.Schema || evidence.Cohort != contract.Cohort || evidence.ReceiptID != ref || evidence.Digest != state.Digest || evidence.SubjectRevision != revision || evidence.ObservedAt.IsZero() || evidence.ObservedAt.After(now) || evidence.FreshUntil.IsZero() || now.After(evidence.FreshUntil) {
			return state, fmt.Errorf("missing, stale, failed or mismatched evidence for %s: %w", outcome.ID, ErrDenied)
		}
		bound := false
		for _, attempt := range state.Attempts {
			if attempt.ExecutionID == evidence.ExecutionID && attempt.Digest == state.Digest && attempt.SettledAt != nil && !evidence.ObservedAt.Before(attempt.StartedAt) {
				bound = true
				break
			}
		}
		if !bound {
			return state, fmt.Errorf("evidence is not bound to a settled engagement attempt: %w", ErrDenied)
		}
		accepted = append(accepted, evidence)
	}
	// Detect edits while owner receipts were being resolved.
	after, err := s.evidence.CurrentRevision(ctx, snapshot.Proposal.Scenario)
	if err != nil || after != revision {
		return state, ErrConflict
	}
	now := s.now()
	state.Version++
	state.Status, state.Evidence, state.AcceptedBy, state.AcceptedAt = "accepted", accepted, actor, &now
	return s.commit(ctx, expected, state)
}

func (s *Service) commit(ctx context.Context, expected int64, value Engagement) (Engagement, error) {
	if err := s.repo.Commit(ctx, expected, value, nil); err != nil {
		return Engagement{}, err
	}
	return value, nil
}

func activeAttempt(state Engagement) *Attempt {
	for i := range state.Attempts {
		if state.Attempts[i].SettledAt == nil {
			return &state.Attempts[i]
		}
	}
	return nil
}

func validItem(item string) bool {
	kind, name, ok := strings.Cut(item, "/")
	if !ok || len(name) > 200 || !slugPattern.MatchString(name) {
		return false
	}
	switch kind {
	case "idea", "research", "fix", "execute", "chore":
		return true
	}
	return false
}

func operator(ctx context.Context) (string, error) {
	p := identity.FromContext(ctx)
	// Attribution is not authentication. In particular, absent agent headers
	// must never become authority to approve an adaptive plan grant.
	if p.IsAgent() || (p.VerificationStatus != "" && p.VerificationStatus != provenance.VerificationAbsent) {
		return "", ErrDenied
	}
	if p.Actor != "" && p.Actor != identity.TypeOperator {
		return "", ErrDenied
	}
	principal, err := authn.RequireHuman(ctx)
	if err != nil {
		return "", fmt.Errorf("verified human approval required: %w", ErrDenied)
	}
	if _, err := authn.RequireCapability(ctx, "swarm-manager:write"); err != nil {
		return "", fmt.Errorf("development decision capability required: %w", ErrDenied)
	}
	return principal.Subject, nil
}
