package approval_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	"treasury/internal/approval"
	"treasury/internal/authorization"
)

// [REQ:TRS-P0-006] Local queue admission and resolution remain authoritative
// when notification-hub is unavailable; the failed relay is durable and a
// decline releases the authorization hold.
func TestLocalApprovalResolvesWhileRelayIsUnavailable(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(approval.Schema)))
	_, err := handle.ExecContext(ctx, `PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	controller := authorization.NewSQLiteRepository(handle)
	_, err = controller.Create(ctx, authorization.Record{ID: "auth-1", IdempotencyKey: "idem-1", MandateID: "mandate-1", BudgetID: "budget-1", RequestingAgent: "operator:1", AmountMinor: 500, Currency: "USD", Counterparty: "vendor.example", Verdict: authorization.VerdictPending, HoldMinor: 500, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)

	repository := approval.NewSQLiteRepository(handle)
	service := approval.NewService(repository, controller, approval.UnavailableRelay{Cause: errors.New("connection refused")}, schedule.NewFake(now))
	require.NoError(t, service.Admit(ctx, authorization.ApprovalAdmission{ID: "approval-1", AuthorizationID: "auth-1", MandateID: "mandate-1", RequestingAgent: "operator:1", AmountMinor: 500, Currency: "USD", Counterparty: "vendor.example", ExpiresAt: now.Add(time.Hour)}))

	attempts, err := service.RelayAttempts(ctx, "approval-1")
	require.NoError(t, err)
	require.Len(t, attempts, 1)
	require.Equal(t, "failed", attempts[0].Outcome)
	require.Contains(t, attempts[0].Error, "connection refused")

	resolved, err := service.Resolve(ctx, "approval-1", approval.StatusDeclined, "operator:1")
	require.NoError(t, err)
	require.Equal(t, approval.StatusDeclined, resolved.Status)
	require.Equal(t, "operator:1", resolved.ResolverIdentity)
	authorizationRecord, err := controller.Get(ctx, "auth-1")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictReleased, authorizationRecord.Verdict)
	require.Zero(t, authorizationRecord.HoldMinor)
}

func TestExpiredApprovalReleasesHeadroom(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(approval.Schema)))
	_, err := handle.ExecContext(ctx, `PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	controller := authorization.NewSQLiteRepository(handle)
	_, err = controller.Create(ctx, authorization.Record{ID: "auth-expire", IdempotencyKey: "idem-expire", MandateID: "mandate-1", BudgetID: "budget-1", RequestingAgent: "operator:1", AmountMinor: 300, Currency: "USD", Counterparty: "vendor.example", Verdict: authorization.VerdictPending, HoldMinor: 300, CreatedAt: now, ExpiresAt: now.Add(time.Minute)})
	require.NoError(t, err)
	service := approval.NewService(approval.NewSQLiteRepository(handle), controller, nil, clock)
	require.NoError(t, service.Admit(ctx, authorization.ApprovalAdmission{ID: "approval-expire", AuthorizationID: "auth-expire", MandateID: "mandate-1", RequestingAgent: "operator:1", AmountMinor: 300, Currency: "USD", Counterparty: "vendor.example", ExpiresAt: now.Add(time.Minute)}))
	clock.Advance(2 * time.Minute)
	expired, err := service.Expire(ctx, "approval-expire")
	require.NoError(t, err)
	require.Equal(t, approval.StatusExpired, expired.Status)
	authorizationRecord, err := controller.Get(ctx, "auth-expire")
	require.NoError(t, err)
	require.Equal(t, authorization.VerdictReleased, authorizationRecord.Verdict)
	require.Zero(t, authorizationRecord.HoldMinor)
}

func TestApprovalAdmissionIsIdempotentAndRejectsProjectionAliasing(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(approval.Schema)))
	_, err := handle.ExecContext(ctx, `PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repository := approval.NewSQLiteRepository(handle)
	service := approval.NewService(repository, nil, nil, schedule.NewFake(now))
	admission := authorization.ApprovalAdmission{ID: "approval-retry", AuthorizationID: "auth-retry", MandateID: "mandate-1", RequestingAgent: "operator:1", AmountMinor: 100, Currency: "USD", Counterparty: "vendor.example", ExpiresAt: now.Add(time.Hour)}

	require.NoError(t, service.Admit(ctx, admission))
	require.NoError(t, service.Admit(ctx, admission), "a retry after a partial crash must reuse the local gate")
	attempts, err := service.RelayAttempts(ctx, admission.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 1, "the deterministic relay attempt must also be idempotent")

	admission.AmountMinor++
	err = service.Admit(ctx, admission)
	require.ErrorIs(t, err, approval.ErrInvalid)
}

func TestStoppedNotificationHubDoesNotBlockLocalApproval(t *testing.T) {
	baseURL := os.Getenv("TREASURY_TEST_NOTIFICATION_HUB_URL")
	if baseURL == "" {
		t.Skip("set TREASURY_TEST_NOTIFICATION_HUB_URL for the attended stopped-dependency proof")
	}
	relay, err := approval.NewNotificationRelay(baseURL, &http.Client{Timeout: 500 * time.Millisecond})
	require.NoError(t, err)
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle, database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(approval.Schema)))
	_, err = handle.ExecContext(ctx, `PRAGMA foreign_keys = OFF`)
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	controller := authorization.NewSQLiteRepository(handle)
	_, err = controller.Create(ctx, authorization.Record{ID: "auth-live", IdempotencyKey: "idem-live", MandateID: "mandate-1", BudgetID: "budget-1", RequestingAgent: "operator:1", AmountMinor: 200, Currency: "USD", Counterparty: "vendor.example", Verdict: authorization.VerdictPending, HoldMinor: 200, CreatedAt: now, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	service := approval.NewService(approval.NewSQLiteRepository(handle), controller, relay, schedule.NewFake(now))
	require.NoError(t, service.Admit(ctx, authorization.ApprovalAdmission{ID: "approval-live", AuthorizationID: "auth-live", MandateID: "mandate-1", RequestingAgent: "operator:1", AmountMinor: 200, Currency: "USD", Counterparty: "vendor.example", ExpiresAt: now.Add(time.Hour)}))
	attempts, err := service.RelayAttempts(ctx, "approval-live")
	require.NoError(t, err)
	require.Equal(t, "failed", attempts[0].Outcome)
	resolved, err := service.Resolve(ctx, "approval-live", approval.StatusDeclined, "operator:1")
	require.NoError(t, err)
	require.Equal(t, approval.StatusDeclined, resolved.Status)
}
