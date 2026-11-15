package researchbudget_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/researchbudget"
)

type tokenSource func(context.Context) (string, error)

func (f tokenSource) Token(ctx context.Context) (string, error) { return f(ctx) }

type serviceCredentialReader struct {
	testingT *testing.T
	value    string
}

func (r serviceCredentialReader) Resolve(_ context.Context, identity, field string) (string, error) {
	r.testingT.Helper()
	require.Equal(r.testingT, "scenario-authenticator/service-principal/treasury-agent-manager/agent-manager", identity)
	require.Equal(r.testingT, "credential", field)
	return r.value, nil
}

type serviceCredentialExchange struct {
	accountsconnect.UnimplementedAccountsServiceHandler
	credential string
	token      string
}

func (h *serviceCredentialExchange) ExchangeServiceCredential(_ context.Context, req *connect.Request[accountsv1.ExchangeServiceCredentialRequest]) (*connect.Response[accountsv1.ExchangeServiceCredentialResponse], error) {
	if req.Msg.GetCredential() != h.credential {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("wrong service credential"))
	}
	return connect.NewResponse(&accountsv1.ExchangeServiceCredentialResponse{
		AccessToken: h.token, AccessTokenExpiresAt: timestamppb.New(time.Now().Add(time.Hour)),
	}), nil
}

func TestNewAuthenticatedClientUsesAgentManagerServicePrincipalAtRunReportRoute(t *testing.T) {
	const serviceToken = "authenticator-issued-agent-manager-service-token"
	var reportPath, reportAuthorization string
	reportServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reportPath, reportAuthorization = r.URL.Path, r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer reportServer.Close()

	exchangeMux := http.NewServeMux()
	path, handler := accountsconnect.NewAccountsServiceHandler(&serviceCredentialExchange{
		credential: "fixture-agent-manager-credential", token: serviceToken,
	})
	exchangeMux.Handle(path, handler)
	exchangeServer := httptest.NewServer(exchangeMux)
	defer exchangeServer.Close()

	client, err := researchbudget.NewAuthenticatedClient(reportServer.URL, reportServer.Client(), serviceCredentialReader{
		testingT: t, value: "fixture-agent-manager-credential",
	}, exchangeServer.URL)
	require.NoError(t, err)
	runID := uuid.New()
	snapshot := researchbudget.Snapshot{
		TreasuryBudgetID: "treasury-budget", MandateID: "mandate", AgentSubject: "run-agent",
		CapMinor: 100, Currency: "USD", UnitCode: "USD_CENTS", PolicyRevision: "revision",
		CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, client.PutSnapshot(context.Background(), runID.String(), snapshot))
	require.Equal(t, "/api/v1/research-budgets/"+runID.String()+"/snapshot", reportPath)
	require.Equal(t, "Bearer "+serviceToken, reportAuthorization)
}

// TestCF7RequestOnlyBridgeClient is invoked by Agent Manager's local handler
// integration test from this module so the real Treasury client can cross the
// Go internal-package boundary without a shared-module dependency.
func TestCF7RequestOnlyBridgeClient(t *testing.T) {
	reportURL := strings.TrimSpace(os.Getenv("CF7_AM_REQUEST_ONLY_URL"))
	if reportURL == "" {
		t.Skip("invoked only by Agent Manager's local CF7 report-route test")
	}
	runID := strings.TrimSpace(os.Getenv("CF7_AM_REQUEST_ONLY_RUN_ID"))
	client, err := researchbudget.NewAuthenticatedClient(reportURL, nil, serviceCredentialReader{
		testingT: t, value: "cf7-agent-manager-test-credential",
	}, reportURL)
	require.NoError(t, err)
	err = client.PutSnapshot(context.Background(), runID, researchbudget.Snapshot{
		TreasuryBudgetID: "cf7-test-budget", MandateID: "cf7-test-mandate", AgentSubject: "cf7-test-agent",
		CapMinor: 100, Currency: "USD", UnitCode: "USD_CENTS", PolicyRevision: "cf7-test-revision",
		CreatedAt: time.Date(2026, 11, 14, 6, 0, 0, 0, time.UTC),
	})
	require.ErrorContains(t, err, "Agent Manager research budget evidence returned 202")
	require.ErrorContains(t, err, "approval_pending")
	err = client.RecordSettlement(context.Background(), runID, researchbudget.Settlement{
		ID: "cf7-unapproved-evidence", SettlementID: "cf7-unapproved-settlement",
		TreasuryReceiptID: "cf7-unapproved-receipt", IntegrationID: "x402:local-test",
		Category: "x402_call", AmountMinor: 1, Currency: "USD", UnitCode: "USD_CENTS",
		ResultRef: "cf7-test-result", BenefitState: "measured", BenefitSummary: "test fixture only",
		SettledAt: time.Date(2026, 11, 14, 6, 1, 0, 0, time.UTC),
	})
	require.ErrorContains(t, err, "Agent Manager research budget evidence returned 202")
	require.ErrorContains(t, err, "approval_pending")
}

func TestClientPostsRunBoundImmutableEvidenceWithTreasuryServiceToken(t *testing.T) {
	runID := uuid.New()
	created, settled := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC), time.Date(2026, 11, 13, 12, 1, 0, 0, time.UTC)
	var requests []struct {
		path, authorization string
		body                map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(data, &body))
		requests = append(requests, struct {
			path, authorization string
			body                map[string]any
		}{r.URL.Path, r.Header.Get("Authorization"), body})
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := researchbudget.NewClient(server.URL, server.Client(), tokenSource(func(context.Context) (string, error) { return "issuer-signed-service-token", nil }))
	require.NoError(t, err)
	snapshot := researchbudget.Snapshot{
		TreasuryBudgetID: "budget-treasury-8", MandateID: "mandate-4", AgentSubject: "run-agent",
		CapMinor: 1000, Currency: "USD", UnitCode: "USD_CENTS", PolicyRevision: "rev-7", CreatedAt: created,
	}
	require.NoError(t, client.PutSnapshot(context.Background(), runID.String(), snapshot))
	require.NoError(t, client.PutSnapshot(context.Background(), runID.String(), snapshot))
	settlement := researchbudget.Settlement{
		ID: "evidence-1", SettlementID: "settlement-1", TreasuryReceiptID: "treasury-receipt-1",
		IntegrationID: "x402:local-test", Category: "x402_call", AmountMinor: 25, Currency: "USD",
		UnitCode: "USD_CENTS", ResultRef: "result:call-1", BenefitState: "measured",
		BenefitSummary: "one local fixture result", SettledAt: settled,
	}
	// Agent Manager's ledger makes identical settlement retries idempotent. The
	// Treasury transport retries the same evidence identity and exact body.
	require.NoError(t, client.RecordSettlement(context.Background(), runID.String(), settlement))
	require.NoError(t, client.RecordSettlement(context.Background(), runID.String(), settlement))
	require.Len(t, requests, 4)
	require.Equal(t, "/api/v1/research-budgets/"+runID.String()+"/snapshot", requests[0].path)
	require.Equal(t, requests[0], requests[1])
	require.Equal(t, "/api/v1/research-budgets/"+runID.String()+"/settlements", requests[2].path)
	require.Equal(t, requests[2], requests[3])
	for _, request := range requests {
		require.Equal(t, "Bearer issuer-signed-service-token", request.authorization)
	}
	require.Equal(t, "treasury-receipt-1", requests[2].body["treasury_receipt_id"])
	require.Equal(t, "settlement-1", requests[2].body["settlement_id"])
	require.NotContains(t, requests[2].body, "agent_token")
}

func TestClientRejectsInvalidRunAndMissingServiceIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { t.Fatal("unexpected unauthenticated request") }))
	defer server.Close()
	client, err := researchbudget.NewClient(server.URL, server.Client(), tokenSource(func(context.Context) (string, error) { return "", nil }))
	require.NoError(t, err)
	snapshot := researchbudget.Snapshot{TreasuryBudgetID: "b", MandateID: "m", AgentSubject: "a", CapMinor: 1, Currency: "USD", UnitCode: "USD_CENTS", PolicyRevision: "r", CreatedAt: time.Now().UTC()}
	require.ErrorContains(t, client.PutSnapshot(context.Background(), "not-a-run", snapshot), "valid Agent Manager run UUID")
	require.ErrorContains(t, client.PutSnapshot(context.Background(), uuid.NewString(), snapshot), "service identity is unavailable")
}

func TestClientRejectsNonNoContentAndNeverAcceptsCallerSelectedPath(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		http.Error(w, "operation not admitted", http.StatusForbidden)
	}))
	defer server.Close()
	client, err := researchbudget.NewClient(server.URL, server.Client(), tokenSource(func(context.Context) (string, error) { return "service-token", nil }))
	require.NoError(t, err)
	runID := uuid.NewString()
	settlement := researchbudget.Settlement{ID: "i", SettlementID: "s", TreasuryReceiptID: "r", IntegrationID: "x402:test", Category: "x402_call", AmountMinor: 0, Currency: "USD", UnitCode: "USD_CENTS", SettledAt: time.Now().UTC()}
	err = client.RecordSettlement(context.Background(), runID, settlement)
	require.ErrorContains(t, err, "returned 403")
	require.True(t, strings.HasPrefix(path, "/api/v1/research-budgets/"+runID+"/"))
}

func TestClientDoesNotFollowEvidenceRedirectsOrForwardTreasuryToken(t *testing.T) {
	var redirectedRequest bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirectedRequest = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer destination.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer origin.Close()

	client, err := researchbudget.NewClient(origin.URL, origin.Client(), tokenSource(func(context.Context) (string, error) { return "service-token", nil }))
	require.NoError(t, err)
	runID := uuid.New()
	snapshot := researchbudget.Snapshot{
		TreasuryBudgetID: "budget", MandateID: "mandate", AgentSubject: "agent",
		CapMinor: 1, Currency: "USD", UnitCode: "USD_MINOR", PolicyRevision: "rev", CreatedAt: time.Now().UTC(),
	}
	err = client.PutSnapshot(context.Background(), strings.ToUpper(runID.String()), snapshot)
	require.ErrorContains(t, err, "returned 307")
	require.False(t, redirectedRequest, "service token must not be sent to a redirect destination")
}

func TestNewClientRequiresServiceTokenSource(t *testing.T) {
	_, err := researchbudget.NewClient("http://localhost:1", nil, nil)
	require.ErrorContains(t, err, "service token source")
}
