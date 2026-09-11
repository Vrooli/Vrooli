package execution

import (
	"context"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
)

const (
	continuationOperation     = "continue"
	continuationStartedBy     = "swarm-manager-sweeper"
	continuationMaxNoProgress = 3
)

// continueExhaustedLocked creates at most one continuation child per sweep
// for each eligible budget-exhausted execution. The caller does not hold the
// service mutex; the whole selection and append is serialized here so two
// sweeper ticks cannot create duplicate children.
func (s *Service) continueExhaustedLocked(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return
	}
	changed := false
	for _, parent := range records {
		if parent.Status != StatusBudgetExhausted {
			continue
		}
		item, itemErr := s.loadBacklogItemByRecord(&parent)
		if itemErr != nil || strings.ToLower(strings.TrimSpace(item.Continuation)) != "until-allowance" {
			continue
		}
		if strings.TrimSpace(item.ContinuationHaltedAt) != "" {
			continue
		}
		if hasActiveContinuationRecord(records, parent) {
			continue
		}
		if hasContinuationChild(records, parent.ExecutionID) {
			continue
		}

		if continuationChainDepth(records, parent) >= continuationMaxNoProgress {
			now := nowRFC3339()
			if s.updateContinuationState(parent.BacklogKind, parent.BacklogName, now, continuationStartedBy, "no_progress") == nil {
				changed = true
			}
			continue
		}

		remaining, reason := remainingContinuationAllowance(records, parent)
		if reason != "" {
			if s.updateContinuationState(parent.BacklogKind, parent.BacklogName, "", "", reason) == nil {
				changed = true
			}
			continue
		}

		child := parent
		child.ExecutionID = idgen.Generate()
		child.Status = StatusPending
		child.PreviousStatus = string(parent.Status)
		child.RunID = ""
		child.TaskID = ""
		child.StartedAt = ""
		child.FinishedAt = ""
		child.FailureReason = ""
		child.WorkflowGrant = nil
		child.SettledUsage = nil
		child.ContinuationOf = parent.ExecutionID
		child.ParentExecutionID = parent.ExecutionID
		child.Operation = continuationOperation
		child.StartedBy = continuationStartedBy
		child.MaxSlices = minPositive(parent.MaxSlices, remaining)
		child.QueuedAt = nowRFC3339()
		child.CreatedAt = child.QueuedAt
		child.UpdatedAt = child.QueuedAt
		child.ExecutionPreferences = cloneExecutionPreferences(parent.ExecutionPreferences)
		for i := range records {
			if records[i].ExecutionID == parent.ExecutionID {
				records[i].ContinuationChildIDs = append(records[i].ContinuationChildIDs, child.ExecutionID)
				break
			}
		}
		records = append(records, child)
		changed = true
	}
	if changed {
		_ = s.store.Save(records)
	}
	_ = ctx // reserved for future plan-status/circuit evidence reads
}

func hasActiveContinuationRecord(records []Record, parent Record) bool {
	for _, record := range records {
		if record.ExecutionID == parent.ExecutionID || record.BacklogKind != parent.BacklogKind || record.BacklogName != parent.BacklogName {
			continue
		}
		switch record.Status {
		case StatusPending, StatusStarting, StatusRunning, StatusNeedsReview, StatusValidating, StatusCancelling:
			return true
		}
	}
	return false
}

func hasContinuationChild(records []Record, parentID string) bool {
	for _, record := range records {
		if record.ContinuationOf == parentID {
			return true
		}
	}
	return false
}

func continuationChainDepth(records []Record, record Record) int {
	depth := 0
	current := record
	for current.ContinuationOf != "" && depth < continuationMaxNoProgress+1 {
		depth++
		found := false
		for _, candidate := range records {
			if candidate.ExecutionID == current.ContinuationOf {
				current = candidate
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return depth
}

// remainingContinuationAllowance mirrors the aggregate subtraction in
// prepareExecutionGrantLocked without reserving or mutating a new record.
// Unknown terminal accounting is fail-closed: continuation must not treat it
// as free capacity.
func remainingContinuationAllowance(records []Record, parent Record) (int, string) {
	limits := parent.ExecutionLimits
	if limits == nil {
		return 0, "allowance"
	}
	tokens := limits.MaxTokens
	turns := int64(limits.MaxTurns)
	wall := limits.MaxWallSeconds
	charge := limits.MaxChargeMicroUSD
	children := int64(limits.MaxChildren)
	attempts := int64(limits.MaxNodeAttempts)
	retries := int64(limits.MaxRetries)
	slices := int64(limits.MaxSlices)
	reservations := int64(0)
	for _, prior := range records {
		if prior.BacklogKind != parent.BacklogKind || prior.BacklogName != parent.BacklogName || prior.ApprovalDigest != parent.ApprovalDigest || prior.WorkflowGrant == nil || prior.ExecutionID == parent.ExecutionID {
			continue
		}
		reservations++
		if prior.SettledUsage == nil || !prior.SettledUsage.TokensKnown || !prior.SettledUsage.ChargeMeasured || prior.SettledUsage.WallSeconds <= 0 {
			return 0, "usage_unknown"
		}
		usage := prior.SettledUsage
		tokens -= usage.Tokens
		turns -= usage.Turns
		wall -= usage.WallSeconds
		charge -= usage.ChargeMicroUSD
		children -= usage.Children
		attempts -= usage.NodeAttempts
		retries -= usage.Retries
		slices -= usage.Slices
	}
	if parent.WorkflowGrant != nil {
		reservations++
	}
	if parent.SettledUsage != nil {
		usage := parent.SettledUsage
		tokens -= usage.Tokens
		turns -= usage.Turns
		wall -= usage.WallSeconds
		charge -= usage.ChargeMicroUSD
		children -= usage.Children
		attempts -= usage.NodeAttempts
		retries -= usage.Retries
		slices -= usage.Slices
	}
	retries -= reservations
	for _, dimension := range []struct {
		value  int64
		reason string
	}{
		{tokens, "tokens"}, {turns, "turns"}, {wall, "wall"}, {charge, "charge"},
		{children, "children"}, {attempts, "node_attempts"}, {retries, "retries"}, {slices, "slices"},
	} {
		if dimension.value <= 0 {
			return 0, dimension.reason
		}
	}
	return int(slices), ""
}

func minPositive(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}

func splitContinuationItem(itemKey string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(itemKey), "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

// HaltContinuation prevents new sweeper-created children. It does not cancel
// an execution that is already running.
func (s *Service) HaltContinuation(itemKey, reason string) error {
	kind, name, ok := splitContinuationItem(itemKey)
	if !ok {
		return apierr.BadRequest("item must be KIND/NAME")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.loadBacklogItem(kind, name); err != nil {
		return apierr.NotFound("backlog item not found: %s/%s", kind, name)
	}
	if strings.TrimSpace(reason) == "" {
		reason = "operator"
	}
	return s.updateContinuationState(kind, name, nowRFC3339(), reason, "halted")
}

// ResumeContinuation clears the durable halt and prior stop reason.
func (s *Service) ResumeContinuation(itemKey string) error {
	kind, name, ok := splitContinuationItem(itemKey)
	if !ok {
		return apierr.BadRequest("item must be KIND/NAME")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.loadBacklogItem(kind, name); err != nil {
		return apierr.NotFound("backlog item not found: %s/%s", kind, name)
	}
	return s.updateContinuationState(kind, name, "", "", "")
}
