package treasuryadmin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	bookv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/book"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	mandatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/handlers/treasuryadmin"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/mandate"
	"treasury/internal/operatorauth"
)

// [REQ:TRS-P0-004] Every TreasuryAdmin method rejects the agent realm before
// its domain implementation is reached, while an operator credential crosses
// the realm boundary and reaches the owning domain (or its unavailable-dependency boundary).
func TestEveryMethodSeparatesAgentAndOperatorRealms(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	_, transport := authorizationconnect.NewTreasuryAdminHandler(treasuryadmin.NewConnectHandler(authorizer))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)

	methods := []struct {
		name         string
		operatorCode connect.Code
		call         func(headers func(http.Header)) error
	}{
		{name: "CreateBook", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.CreateBookRequest{})
			set(request.Header())
			_, err := client.CreateBook(context.Background(), request)
			return err
		}},
		{name: "GetBook", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.GetBookRequest{})
			set(request.Header())
			_, err := client.GetBook(context.Background(), request)
			return err
		}},
		{name: "CreateMandate", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.CreateMandateRequest{})
			set(request.Header())
			_, err := client.CreateMandate(context.Background(), request)
			return err
		}},
		{name: "RevokeMandate", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.RevokeMandateRequest{})
			set(request.Header())
			_, err := client.RevokeMandate(context.Background(), request)
			return err
		}},
		{name: "SetBudgetCaps", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.SetBudgetCapsRequest{})
			set(request.Header())
			_, err := client.SetBudgetCaps(context.Background(), request)
			return err
		}},
		{name: "SetGating", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.SetGatingRequest{})
			set(request.Header())
			_, err := client.SetGating(context.Background(), request)
			return err
		}},
		{name: "ListApprovals", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.ListApprovalsRequest{})
			set(request.Header())
			_, err := client.ListApprovals(context.Background(), request)
			return err
		}},
		{name: "ResolveApproval", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.ResolveApprovalRequest{})
			set(request.Header())
			_, err := client.ResolveApproval(context.Background(), request)
			return err
		}},
		{name: "FreezeBudget", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.FreezeBudgetRequest{})
			set(request.Header())
			_, err := client.FreezeBudget(context.Background(), request)
			return err
		}},
		{name: "UnfreezeBudget", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.UnfreezeBudgetRequest{})
			set(request.Header())
			_, err := client.UnfreezeBudget(context.Background(), request)
			return err
		}},
		{name: "RegisterInstrument", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.RegisterInstrumentRequest{})
			set(request.Header())
			_, err := client.RegisterInstrument(context.Background(), request)
			return err
		}},
		{name: "ReportManualOutcome", operatorCode: connect.CodeFailedPrecondition, call: func(set func(http.Header)) error {
			request := connect.NewRequest(&authorizationv1.ReportManualOutcomeRequest{})
			set(request.Header())
			_, err := client.ReportManualOutcome(context.Background(), request)
			return err
		}},
	}

	for _, method := range methods {
		t.Run(method.name+" rejects agent", func(t *testing.T) {
			err := method.call(func(header http.Header) { header.Set(operatorauth.HeaderAgentToken, "valid-agent-token") })
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
		})
		t.Run(method.name+" accepts operator realm", func(t *testing.T) {
			err := method.call(func(header http.Header) { header.Set(operatorauth.HeaderOperatorToken, "operator-secret") })
			require.Equal(t, method.operatorCode, connect.CodeOf(err), "operator must cross the realm boundary and reach the handler")
		})
	}
}

// [REQ:TRS-P0-001] [REQ:TRS-P0-003] The generated operator client can
// establish and mutate the complete grant spine without bypassing the API.
func TestOperatorCanCreateAndControlGrantSpine(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle,
		database.SchemaProviderFunc(book.Schema),
		database.SchemaProviderFunc(budget.Schema),
		database.SchemaProviderFunc(mandate.Schema),
	))
	now := time.Date(2026, 8, 19, 4, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	signer, err := mandate.NewHMACSigner([]byte("operator-handler-test-signing-key"))
	require.NoError(t, err)
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	handler := treasuryadmin.NewConnectHandler(authorizer,
		book.NewService(book.NewSQLiteRepository(handle), clock),
		budget.NewService(budget.NewSQLiteRepository(handle), clock),
		mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer),
	)
	_, transport := authorizationconnect.NewTreasuryAdminHandler(handler)
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewTreasuryAdminClient(server.Client(), server.URL)
	operatorRequest := func(header http.Header) { header.Set(operatorauth.HeaderOperatorToken, "operator-secret") }

	bookRequest := connect.NewRequest(&authorizationv1.CreateBookRequest{Book: &bookv1.Book{Id: "book-1", Name: "Operator", BeneficiaryIdentity: "caller-spoof"}})
	operatorRequest(bookRequest.Header())
	bookResponse, err := client.CreateBook(ctx, bookRequest)
	require.NoError(t, err)
	require.Equal(t, "local-operator", bookResponse.Msg.GetBook().GetBeneficiaryIdentity(), "book ownership is bound to the verified operator")
	getBookRequest := connect.NewRequest(&authorizationv1.GetBookRequest{BookId: "book-1"})
	operatorRequest(getBookRequest.Header())
	getBookResponse, err := client.GetBook(ctx, getBookRequest)
	require.NoError(t, err)
	require.Equal(t, bookResponse.Msg.GetBook().GetId(), getBookResponse.Msg.GetBook().GetId())

	budgetRequest := connect.NewRequest(&authorizationv1.SetBudgetCapsRequest{Budget: &budgetv1.Budget{Id: "budget-1", BookId: "book-1", Currency: "USD", TotalCapMinor: 1000, PeriodicCapMinor: 500, PerTransactionCapMinor: 100, PeriodSeconds: 3600, AllowedCounterparties: []string{"vendor.example"}, RequiresApproval: false, Frozen: true}})
	operatorRequest(budgetRequest.Header())
	budgetResponse, err := client.SetBudgetCaps(ctx, budgetRequest)
	require.NoError(t, err)
	require.False(t, budgetResponse.Msg.GetBudget().GetFrozen(), "cap creation cannot smuggle in a freeze-state mutation")

	gatingRequest := connect.NewRequest(&authorizationv1.SetGatingRequest{BudgetId: "budget-1", RequiresApproval: true})
	operatorRequest(gatingRequest.Header())
	gatingResponse, err := client.SetGating(ctx, gatingRequest)
	require.NoError(t, err)
	require.True(t, gatingResponse.Msg.GetBudget().GetRequiresApproval())

	freezeRequest := connect.NewRequest(&authorizationv1.FreezeBudgetRequest{BudgetId: "budget-1"})
	operatorRequest(freezeRequest.Header())
	freezeResponse, err := client.FreezeBudget(ctx, freezeRequest)
	require.NoError(t, err)
	require.True(t, freezeResponse.Msg.GetBudget().GetFrozen())
	unfreezeRequest := connect.NewRequest(&authorizationv1.UnfreezeBudgetRequest{BudgetId: "budget-1"})
	operatorRequest(unfreezeRequest.Header())
	unfreezeResponse, err := client.UnfreezeBudget(ctx, unfreezeRequest)
	require.NoError(t, err)
	require.False(t, unfreezeResponse.Msg.GetBudget().GetFrozen())

	mandateRequest := connect.NewRequest(&authorizationv1.CreateMandateRequest{Mandate: &mandatev1.Mandate{Id: "mandate-1", IdempotencyKey: "mandate-key-1", BookId: "book-1", BudgetId: "budget-1", Authorizer: "caller-spoof", CapMinor: 100, Currency: "USD", AllowedCounterparties: []string{"vendor.example"}, RequiredEvidence: []string{"receipt"}, ExpiresAt: timestamppb.New(now.Add(24 * time.Hour))}})
	operatorRequest(mandateRequest.Header())
	mandateResponse, err := client.CreateMandate(ctx, mandateRequest)
	require.NoError(t, err)
	require.Equal(t, "local-operator", mandateResponse.Msg.GetMandate().GetAuthorizer(), "authorizer is bound to the verified operator")
	require.NotEmpty(t, mandateResponse.Msg.GetMandate().GetSignature())

	revokeRequest := connect.NewRequest(&authorizationv1.RevokeMandateRequest{MandateId: "mandate-1"})
	operatorRequest(revokeRequest.Header())
	revokeResponse, err := client.RevokeMandate(ctx, revokeRequest)
	require.NoError(t, err)
	require.Equal(t, mandatev1.MandateStatus_MANDATE_STATUS_REVOKED, revokeResponse.Msg.GetMandate().GetStatus())
}
