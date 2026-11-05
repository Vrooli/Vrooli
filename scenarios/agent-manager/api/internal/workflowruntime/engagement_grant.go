package workflowruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func cloneGrant(grant *domain.WorkflowEngagementGrant) *domain.WorkflowEngagementGrant {
	if grant == nil {
		return nil
	}
	copy := *grant
	copy.AllowedEffects = append([]string(nil), grant.AllowedEffects...)
	return &copy
}

// sameGrant protects idempotent replay from silently changing the authority
// attached to an execution. A retry with the same key must either reproduce
// the original request or be rejected as a conflict.
func sameGrant(left, right *domain.WorkflowEngagementGrant) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.MaxTurns == right.MaxTurns && left.MaxTokens == right.MaxTokens && left.MaxChargeMicroUSD == right.MaxChargeMicroUSD && left.MaxWallTimeSeconds == right.MaxWallTimeSeconds && left.MaxNodeAttempts == right.MaxNodeAttempts && left.MaxChildren == right.MaxChildren && left.MaxConcurrency == right.MaxConcurrency && left.MaxRecursion == right.MaxRecursion && left.MaxRetries == right.MaxRetries && (left.MaxRetries > 0 || left.RetryLimitSet == right.RetryLimitSet) && left.MaxWaitSeconds == right.MaxWaitSeconds && slices.Equal(left.AllowedEffects, right.AllowedEffects)
}

func sameExecutionBinding(existing *domain.WorkflowExecution, binding []ExecutionBinding) bool {
	if existing == nil {
		return false
	}
	if len(binding) == 0 {
		return existing.ApprovalDigest == "" && existing.GrantDigest == ""
	}
	return existing.ApprovalDigest == binding[0].ApprovalDigest && existing.GrantDigest == binding[0].GrantDigest
}

func validateEngagementGrant(grant domain.WorkflowEngagementGrant, declared domain.WorkflowBudgets) error {
	check := func(name string, grantValue, declaredValue int) error {
		if grantValue > 0 && (declaredValue <= 0 || grantValue > declaredValue) {
			return fmt.Errorf("engagement grant %s exceeds workflow declaration", name)
		}
		return nil
	}
	if err := check("max turns", grant.MaxTurns, declared.MaxTurns); err != nil {
		return err
	}
	if err := check("max tokens", grant.MaxTokens, declared.MaxTokens); err != nil {
		return err
	}
	if err := check("max wall time", grant.MaxWallTimeSeconds, declared.WallTimeSeconds); err != nil {
		return err
	}
	if err := check("max node attempts", grant.MaxNodeAttempts, declared.MaxNodeAttempts); err != nil {
		return err
	}
	if err := check("max children", grant.MaxChildren, declared.MaxChildren); err != nil {
		return err
	}
	if err := check("max concurrency", grant.MaxConcurrency, declared.MaxConcurrency); err != nil {
		return err
	}
	if err := check("max recursion", grant.MaxRecursion, declared.MaxRecursion); err != nil {
		return err
	}
	if err := check("max retries", grant.MaxRetries, declared.MaxRetries); err != nil {
		return err
	}
	if err := check("max wait time", grant.MaxWaitSeconds, declared.MaxWaitSeconds); err != nil {
		return err
	}
	if grant.MaxChargeMicroUSD > 0 && (declared.MaxChargeMicroUSD <= 0 || grant.MaxChargeMicroUSD > declared.MaxChargeMicroUSD) {
		return fmt.Errorf("engagement grant max charge exceeds workflow declaration")
	}
	return nil
}

func applyEngagementGrant(revision *domain.WorkflowRevision, grant *domain.WorkflowEngagementGrant) *domain.WorkflowRevision {
	if revision == nil || grant == nil {
		return revision
	}
	copy := *revision
	b := revision.Definition.Budgets
	min := func(current *int, limit int) {
		if revision.Definition.GrantCapacity != nil && limit > 0 {
			*current = limit
			return
		}
		if limit > 0 && (*current <= 0 || limit < *current) {
			*current = limit
		}
	}
	min(&b.MaxTurns, grant.MaxTurns)
	min(&b.MaxTokens, grant.MaxTokens)
	min(&b.WallTimeSeconds, grant.MaxWallTimeSeconds)
	min(&b.MaxNodeAttempts, grant.MaxNodeAttempts)
	min(&b.MaxChildren, grant.MaxChildren)
	min(&b.MaxConcurrency, grant.MaxConcurrency)
	min(&b.MaxRecursion, grant.MaxRecursion)
	min(&b.MaxRetries, grant.MaxRetries)
	if grant.RetryLimitSet && grant.MaxRetries == 0 {
		b.MaxRetries = 0
	}
	min(&b.MaxWaitSeconds, grant.MaxWaitSeconds)
	if grant.MaxChargeMicroUSD > 0 && (revision.Definition.GrantCapacity != nil || b.MaxChargeMicroUSD <= 0 || grant.MaxChargeMicroUSD < b.MaxChargeMicroUSD) {
		b.MaxChargeMicroUSD = grant.MaxChargeMicroUSD
	}
	copy.Definition = revision.Definition
	copy.Definition.Budgets = b
	return &copy
}

type inheritedBudgetExhausted struct{ dimension string }

func (err *inheritedBudgetExhausted) Error() string {
	return "parent workflow allowance exhausted: " + err.dimension
}

// inheritedChildGrant derives a reservation from the persisted parent's
// remaining allowance. It cannot widen the child declaration, and the parent
// remains the approval authority. Serial dispatch reserves all of that bounded
// remainder until the child's terminal receipt is added back to parent usage.
func (e *Engine) inheritedChildGrant(ctx context.Context, parent *domain.WorkflowExecution, child *domain.WorkflowRevision) (*domain.WorkflowEngagementGrant, error) {
	if parent.EngagementGrant == nil {
		return nil, nil
	}
	if parent.Status.Terminal() || parent.Status == domain.WorkflowExecutionCancelling {
		return nil, fmt.Errorf("parent workflow is no longer authorized to dispatch")
	}
	if !parent.BudgetUsage.AccountingComplete {
		return nil, fmt.Errorf("parent workflow terminal usage is unresolved; descendant dispatch remains paused")
	}
	if e.Catalog == nil {
		return nil, fmt.Errorf("parent workflow catalog unavailable")
	}
	revision, err := e.Catalog.GetByDigest(ctx, parent.DefinitionDigest)
	if err != nil || revision == nil {
		return nil, fmt.Errorf("parent workflow revision unavailable: %v", err)
	}
	budgets := applyEngagementGrant(revision, parent.EngagementGrant).Definition.Budgets
	// A shared remaining grant is safe only for the serial lane. Supporting
	// parallel descendants requires durable sibling reservations first.
	if budgets.MaxConcurrency != 1 {
		return nil, fmt.Errorf("explicit parent allowance requires serial descendant admission")
	}
	journal, err := e.Store.ListJournal(ctx, parent.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	now := e.now()
	wall := int((time.Duration(budgets.WallTimeSeconds)*time.Second - (now.Sub(parent.CreatedAt) - waitedDuration(journal, now))) / time.Second)
	usage := parent.BudgetUsage
	remaining := domain.WorkflowBudgets{
		MaxTurns: budgets.MaxTurns - usage.Turns, MaxTokens: budgets.MaxTokens - usage.Tokens,
		MaxChargeMicroUSD: budgets.MaxChargeMicroUSD - usage.ChargeMicroUSD, WallTimeSeconds: wall,
		MaxNodeAttempts: budgets.MaxNodeAttempts - usage.NodeAttempts,
		MaxChildren:     budgets.MaxChildren - usage.Children - 1, // This child workflow consumes one parent slot.
		MaxRetries:      budgets.MaxRetries - usage.Retries,
	}
	for _, dimension := range []struct {
		name  string
		value int64
	}{
		{"turns", int64(remaining.MaxTurns)}, {"tokens", int64(remaining.MaxTokens)}, {"charge", remaining.MaxChargeMicroUSD},
		{"wall_time", int64(remaining.WallTimeSeconds)}, {"node_attempts", int64(remaining.MaxNodeAttempts)}, {"children", int64(remaining.MaxChildren)},
	} {
		if dimension.value <= 0 {
			return nil, &inheritedBudgetExhausted{dimension.name}
		}
	}
	if remaining.MaxRetries < 0 {
		return nil, &inheritedBudgetExhausted{"retries"}
	}
	declared := child.Definition.Budgets
	grant := &domain.WorkflowEngagementGrant{
		MaxTurns: min(remaining.MaxTurns, declared.MaxTurns), MaxTokens: min(remaining.MaxTokens, declared.MaxTokens),
		MaxChargeMicroUSD:  min(remaining.MaxChargeMicroUSD, declared.MaxChargeMicroUSD),
		MaxWallTimeSeconds: min(remaining.WallTimeSeconds, declared.WallTimeSeconds),
		MaxNodeAttempts:    min(remaining.MaxNodeAttempts, declared.MaxNodeAttempts), MaxChildren: min(remaining.MaxChildren, declared.MaxChildren),
		MaxConcurrency: 1, MaxRecursion: min(budgets.MaxRecursion, declared.MaxRecursion),
		MaxRetries: min(remaining.MaxRetries, declared.MaxRetries), RetryLimitSet: true,
		MaxWaitSeconds: min(budgets.MaxWaitSeconds, declared.MaxWaitSeconds),
		AllowedEffects: append([]string(nil), parent.EngagementGrant.AllowedEffects...),
	}
	return grant, nil
}

func inheritedGrantDigest(parentID, attemptID uuid.UUID, grant *domain.WorkflowEngagementGrant) string {
	data, _ := json.Marshal(struct {
		ParentID, AttemptID uuid.UUID
		Grant               *domain.WorkflowEngagementGrant
	}{parentID, attemptID, grant})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
