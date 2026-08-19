package mandate_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/mandate"
	"treasury/internal/mandate/flow"
)

type memoryRepository struct {
	mu    sync.Mutex
	byID  map[string]mandate.Mandate
	byKey map[string]string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{byID: map[string]mandate.Mandate{}, byKey: map[string]string{}}
}

func (r *memoryRepository) Create(_ context.Context, value mandate.Mandate) (mandate.Mandate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[value.ID] = value
	r.byKey[value.IdempotencyKey] = value.ID
	return value, nil
}

func (r *memoryRepository) Get(_ context.Context, id string) (mandate.Mandate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.byID[id]
	if !ok {
		return mandate.Mandate{}, mandate.ErrNotFound
	}
	return value, nil
}

func (r *memoryRepository) GetByIdempotencyKey(_ context.Context, key string) (mandate.Mandate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.byKey[key]
	if !ok {
		return mandate.Mandate{}, mandate.ErrNotFound
	}
	return r.byID[id], nil
}

type fixedSigner struct {
	signature []byte
	err       error
}

func (s fixedSigner) Sign(_ context.Context, _ []byte) ([]byte, error) {
	return append([]byte(nil), s.signature...), s.err
}

func validIssue(now time.Time) mandate.IssueInput {
	return mandate.IssueInput{
		ID: "mandate-1", IdempotencyKey: "issue-1", BookID: "book-1", BudgetID: "budget-1",
		Authorizer: "operator:1", CapMinor: 5_000, Currency: "USD",
		AllowedCounterparties: []string{"api.example"}, RequiredEvidence: []string{"receipt"},
		ExpiresAt: now.Add(time.Hour),
	}
}

// [REQ:TRS-P0-001] The mandate contract names every load-bearing grant constraint.
func TestIssueRejectsEachMissingGrantConstraint(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		edit func(*mandate.IssueInput)
		want string
	}{
		{"authorizer", func(in *mandate.IssueInput) { in.Authorizer = "" }, "authorizer"},
		{"cap", func(in *mandate.IssueInput) { in.CapMinor = 0 }, "cap_minor"},
		{"counterparty_scope", func(in *mandate.IssueInput) { in.AllowedCounterparties = nil; in.DeniedCounterparties = nil }, "counterparty_scope"},
		{"expiry", func(in *mandate.IssueInput) { in.ExpiresAt = time.Time{} }, "expires_at"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validIssue(now)
			test.edit(&input)
			service := mandate.NewService(newMemoryRepository(), schedule.NewFake(now), fixedSigner{signature: []byte("signed")})
			_, err := service.Issue(context.Background(), input)
			require.ErrorIs(t, err, mandate.ErrInvalid)
			var validationError *mandate.ValidationError
			require.True(t, errors.As(err, &validationError))
			require.Contains(t, validationError.Constraint, test.want)
		})
	}
}

// [REQ:TRS-P0-001] A valid grant is signed, live, and idempotent on caller key.
func TestIssueSignsAndReturnsTheFirstIdempotentGrant(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repository := newMemoryRepository()
	service := mandate.NewService(repository, schedule.NewFake(now), fixedSigner{signature: []byte("signed")})
	first, err := service.Issue(context.Background(), validIssue(now))
	require.NoError(t, err)
	require.Equal(t, flow.MandateLive, first.Status)
	require.Equal(t, []byte("signed"), first.Signature)

	retry := validIssue(now)
	retry.ID = "different-id"
	second, err := service.Issue(context.Background(), retry)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
}

// [REQ:TRS-P0-012] Expiry binds from the stored timestamp without a job or operator action.
func TestGetComputesExpiryFromInjectedClock(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	repository := newMemoryRepository()
	service := mandate.NewService(repository, clock, fixedSigner{signature: []byte("signed")})
	issued, err := service.Issue(context.Background(), validIssue(now))
	require.NoError(t, err)
	require.Equal(t, flow.MandateLive, issued.Status)

	clock.Advance(time.Hour)
	got, err := service.Get(context.Background(), issued.ID)
	require.NoError(t, err)
	require.Equal(t, flow.MandateExpired, got.Status)
	require.Equal(t, flow.MandateLive, repository.byID[issued.ID].Status, "expiry is derived, not a sweep mutation")

	_, err = service.RequireLive(context.Background(), issued.ID)
	require.ErrorIs(t, err, mandate.ErrInactive)
	require.ErrorContains(t, err, string(flow.MandateExpired))
}

func TestIssueSurfacesSignerFailure(t *testing.T) {
	now := time.Now().UTC()
	service := mandate.NewService(newMemoryRepository(), schedule.NewFake(now), fixedSigner{err: errors.New("signer unavailable")})
	_, err := service.Issue(context.Background(), validIssue(now))
	require.ErrorContains(t, err, "sign mandate")
}
