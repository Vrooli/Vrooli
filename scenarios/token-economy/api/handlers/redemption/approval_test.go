package redemption_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	accessH "token-economy/handlers/access"
	redemptionH "token-economy/handlers/redemption"
	internalaccess "token-economy/internal/access"
	domain "token-economy/internal/redemption"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

type approvalValidator struct{}

func (approvalValidator) Validate(_ context.Context, token string) (internalaccess.Identity, error) {
	switch token {
	case "holder":
		return internalaccess.Identity{Subject: "auth:sam", Scopes: []string{internalaccess.ScopeHolder}}, nil
	case "minter":
		return internalaccess.Identity{Subject: "parent:alex", Scopes: []string{internalaccess.ScopeMinter}}, nil
	default:
		return internalaccess.Identity{}, internalaccess.ErrUnauthenticated
	}
}

type redemptionSpy struct {
	requestedBy string
	decidedBy   string
	value       domain.Redemption
}

func (s *redemptionSpy) Request(_ context.Context, input domain.RequestInput) (domain.Redemption, error) {
	s.requestedBy = input.AuthenticatedSubject
	s.value = domain.Redemption{ID: "redemption-1", HolderID: "holder:sam", CatalogEntryID: input.CatalogEntryID, GrantID: input.GrantID, TokenTypeID: "chores", Amount: 7, IdempotencyKey: input.IdempotencyKey, State: domain.StatePendingApproval, RequestedAt: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)}
	return s.value, nil
}

func (s *redemptionSpy) ListPending(context.Context) ([]domain.Redemption, error) {
	return []domain.Redemption{s.value}, nil
}

func (s *redemptionSpy) ListForSubject(context.Context, string) ([]domain.Redemption, error) {
	return []domain.Redemption{s.value}, nil
}

func (s *redemptionSpy) Approve(_ context.Context, input domain.DecisionInput) (domain.Redemption, error) {
	s.decidedBy = input.DeciderSubject
	s.value.State = domain.StateSettled
	s.value.DeciderSubject = input.DeciderSubject
	s.value.DecisionReason = input.Reason
	return s.value, nil
}

func (s *redemptionSpy) Deny(_ context.Context, input domain.DecisionInput) (domain.Redemption, error) {
	s.decidedBy = input.DeciderSubject
	s.value.State = domain.StateDenied
	return s.value, nil
}

// [REQ:TKE-P0-013] The generated holder and minter surfaces complete a gated
// request and approval using authenticated identities, with no relay mounted.
func TestApprovalFlowUsesDurableQueueWithoutNotificationHub(t *testing.T) {
	spy := &redemptionSpy{}
	module := accessH.Module(nil, nil, nil, nil, nil, redemptionH.NewConnectHandler(spy, nil), nil, approvalValidator{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	holder := accessconnect.NewHolderServiceClient(server.Client(), server.URL)
	request := connect.NewRequest(&accessv1.RequestRedemptionRequest{
		Redemption:     &accessv1.Redemption{CatalogEntryId: "day-trip", GrantId: "grant-1"},
		IdempotencyKey: "request-1",
	})
	request.Header().Set("Authorization", "Bearer holder")
	requested, err := holder.RequestRedemption(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, accessv1.RedemptionState_REDEMPTION_STATE_PENDING_APPROVAL, requested.Msg.Redemption.State)
	require.Equal(t, "auth:sam", spy.requestedBy)

	minter := accessconnect.NewMinterServiceClient(server.Client(), server.URL)
	list := connect.NewRequest(&accessv1.ListPendingRedemptionsRequest{})
	list.Header().Set("Authorization", "Bearer minter")
	pending, err := minter.ListPendingRedemptions(context.Background(), list)
	require.NoError(t, err)
	require.Len(t, pending.Msg.Redemptions, 1)

	approve := connect.NewRequest(&accessv1.ApproveRedemptionRequest{RedemptionId: "redemption-1", Reason: "earned it", IdempotencyKey: "approve-1"})
	approve.Header().Set("Authorization", "Bearer minter")
	approved, err := minter.ApproveRedemption(context.Background(), approve)
	require.NoError(t, err)
	require.Equal(t, accessv1.RedemptionState_REDEMPTION_STATE_SETTLED, approved.Msg.Redemption.State)
	require.Equal(t, "parent:alex", spy.decidedBy)
}
