package credits

import (
	"context"
	"sync"
	"time"
)

// MockService is a test double for CreditService.
// It allows controlling credit availability and tracking charges for assertions.
type MockService struct {
	mu               sync.Mutex
	creditsRemaining int
	creditsLimit     int
	enabled          bool
	charges          []ChargeRequest
	failedOps        []ChargeRequest
	costs            OperationCosts
	canChargeFn      func(OperationType) bool
}

// MockServiceOption configures the mock service.
type MockServiceOption func(*MockService)

// WithCredits sets the initial credit balance.
func WithCredits(credits int) MockServiceOption {
	return func(m *MockService) {
		m.creditsRemaining = credits
		m.creditsLimit = credits
	}
}

// WithUnlimited configures the mock for unlimited credits.
func WithUnlimited() MockServiceOption {
	return func(m *MockService) {
		m.creditsRemaining = -1
		m.creditsLimit = -1
	}
}

// WithEnabled sets whether credits are enabled.
func WithEnabled(enabled bool) MockServiceOption {
	return func(m *MockService) {
		m.enabled = enabled
	}
}

// WithCanChargeFn sets a custom can-charge predicate.
func WithCanChargeFn(fn func(OperationType) bool) MockServiceOption {
	return func(m *MockService) {
		m.canChargeFn = fn
	}
}

// Note: WithCosts was intentionally removed. Operation costs are hard-coded
// to prevent bypassing charges. The mock always uses DefaultOperationCosts().

// NewMockService creates a new mock credit service.
func NewMockService(opts ...MockServiceOption) *MockService {
	m := &MockService{
		creditsRemaining: 100, // Default
		creditsLimit:     100,
		enabled:          true,
		costs:            DefaultOperationCosts(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// CanCharge implements CreditService.
func (m *MockService) CanCharge(ctx context.Context, userIdentity string, op OperationType) (bool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return true, -1, nil
	}

	if m.canChargeFn != nil {
		return m.canChargeFn(op), m.creditsRemaining, nil
	}

	if m.creditsLimit < 0 {
		return true, -1, nil // Unlimited
	}

	cost := m.costs.GetCost(op)
	return m.creditsRemaining >= cost, m.creditsRemaining, nil
}

// Charge implements CreditService.
func (m *MockService) Charge(ctx context.Context, req ChargeRequest) (*ChargeResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return &ChargeResult{Charged: 0, RemainingCredits: -1, WasCharged: false}, nil
	}

	cost := m.costs.GetCost(req.Operation)

	if m.creditsLimit >= 0 {
		m.creditsRemaining -= cost
	}

	m.charges = append(m.charges, req)

	remaining := m.creditsRemaining
	if m.creditsLimit < 0 {
		remaining = -1
	}

	return &ChargeResult{
		Charged:          cost,
		RemainingCredits: remaining,
		WasCharged:       cost > 0,
	}, nil
}

// ChargeIfAllowed implements CreditService.
func (m *MockService) ChargeIfAllowed(ctx context.Context, req ChargeRequest) (*ChargeResult, error) {
	can, remaining, err := m.CanCharge(ctx, req.UserIdentity, req.Operation)
	if err != nil {
		return nil, err
	}
	if !can {
		cost := m.costs.GetCost(req.Operation)
		return nil, &insufficientCreditsError{needed: cost, have: remaining}
	}
	return m.Charge(ctx, req)
}

type insufficientCreditsError struct {
	needed int
	have   int
}

func (e *insufficientCreditsError) Error() string {
	return "insufficient credits"
}

func (e *insufficientCreditsError) Is(target error) bool {
	return target == ErrInsufficientCredits
}

// GetUsage implements CreditService.
func (m *MockService) GetUsage(ctx context.Context, userIdentity string) (*UsageSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	totalUsed := 0
	if m.creditsLimit >= 0 {
		totalUsed = m.creditsLimit - m.creditsRemaining
	}

	byOp := make(map[OperationType]int)
	opCounts := make(map[OperationType]int)
	for _, req := range m.charges {
		cost := m.costs.GetCost(req.Operation)
		byOp[req.Operation] += cost
		opCounts[req.Operation]++
	}

	now := time.Now()
	return &UsageSummary{
		UserIdentity:     userIdentity,
		BillingMonth:     now.Format("2006-01"),
		TotalCreditsUsed: totalUsed,
		TotalOperations:  len(m.charges),
		ByOperation:      byOp,
		OperationCounts:  opCounts,
		CreditsLimit:     m.creditsLimit,
		CreditsRemaining: m.creditsRemaining,
		PeriodStart:      firstDayOfMonth(now),
		PeriodEnd:        lastDayOfMonth(now),
		ResetDate:        firstDayOfMonth(now).AddDate(0, 1, 0),
	}, nil
}

// GetOperationCost implements CreditService.
func (m *MockService) GetOperationCost(op OperationType) int {
	return m.costs.GetCost(op)
}

// LogFailedOperation implements CreditService.
func (m *MockService) LogFailedOperation(ctx context.Context, req ChargeRequest, opErr error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedOps = append(m.failedOps, req)
	return nil
}

// IsEnabled implements CreditService.
func (m *MockService) IsEnabled() bool {
	return m.enabled
}

// GetUsageHistory implements CreditService.
// Returns current usage as the only period for the mock.
func (m *MockService) GetUsageHistory(ctx context.Context, userIdentity string, months, offset int) ([]UsageSummary, bool, error) {
	// Get usage without holding lock to avoid deadlock
	usage, _ := m.GetUsage(ctx, userIdentity)
	return []UsageSummary{*usage}, false, nil
}

// GetOperationLog implements CreditService.
// Returns charges as operation log entries.
func (m *MockService) GetOperationLog(ctx context.Context, userIdentity, month, category string, limit, offset int) (*OperationLogPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}

	operations := make([]OperationLogEntry, 0, len(m.charges))
	for i, charge := range m.charges {
		operations = append(operations, OperationLogEntry{
			ID:             string(rune(i + 1)),
			OperationType:  charge.Operation,
			CreditsCharged: m.costs.GetCost(charge.Operation),
			Success:        true,
			CreatedAt:      time.Now(),
		})
	}

	// Apply pagination
	total := len(operations)
	if offset >= len(operations) {
		operations = nil
	} else if offset+limit < len(operations) {
		operations = operations[offset : offset+limit]
	} else {
		operations = operations[offset:]
	}

	return &OperationLogPage{
		UserIdentity: userIdentity,
		BillingMonth: month,
		Operations:   operations,
		Total:        total,
		Limit:        limit,
		Offset:       offset,
		HasMore:      offset+len(operations) < total,
	}, nil
}

// Test helpers

// GetCharges returns all charges made (for test assertions).
func (m *MockService) GetCharges() []ChargeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ChargeRequest(nil), m.charges...)
}

// GetFailedOps returns all failed operations logged (for test assertions).
func (m *MockService) GetFailedOps() []ChargeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ChargeRequest(nil), m.failedOps...)
}

// Reset clears all recorded charges and failed operations.
func (m *MockService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.charges = nil
	m.failedOps = nil
}

// SetCredits sets the remaining credit balance.
func (m *MockService) SetCredits(credits int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.creditsRemaining = credits
}

// Compile-time interface check
var _ CreditService = (*MockService)(nil)
