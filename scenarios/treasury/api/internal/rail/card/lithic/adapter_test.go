package lithic_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"treasury/internal/rail"
	"treasury/internal/rail/card"
	"treasury/internal/rail/card/lithic"

	"github.com/stretchr/testify/require"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

type sandboxState struct {
	mu               sync.Mutex
	capMinor         int64
	counterparty     string
	expMonth         string
	expYear          string
	cardState        string
	cardCurrency     string
	ruleActive       bool
	ruleDeleted      bool
	declines         int
	events           []string
	cardCreations    int
	cardRequestByKey map[string]string
	cardKeys         []string
}

func newSandbox(t *testing.T) (*httptest.Server, *sandboxState) {
	t.Helper()
	state := &sandboxState{cardRequestByKey: make(map[string]string), cardCurrency: "USD"}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "sandbox-api-key", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/cards":
			var body struct {
				State              string `json:"state"`
				SpendLimit         int64  `json:"spend_limit"`
				SpendLimitDuration string `json:"spend_limit_duration"`
				ExpMonth           string `json:"exp_month"`
				ExpYear            string `json:"exp_year"`
				Memo               string `json:"memo"`
			}
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			require.Equal(t, "FOREVER", body.SpendLimitDuration)
			require.Equal(t, "PAUSED", body.State)
			idempotencyKey := request.Header.Get("Idempotency-Key")
			require.NotEmpty(t, idempotencyKey)
			fingerprintBytes, err := json.Marshal(body)
			require.NoError(t, err)
			if previous, exists := state.cardRequestByKey[idempotencyKey]; exists {
				if previous != string(fingerprintBytes) {
					writer.WriteHeader(http.StatusUnprocessableEntity)
					return
				}
			} else {
				state.cardRequestByKey[idempotencyKey] = string(fingerprintBytes)
				state.cardKeys = append(state.cardKeys, idempotencyKey)
				state.cardCreations++
				state.capMinor, state.expMonth, state.expYear, state.cardState = body.SpendLimit, body.ExpMonth, body.ExpYear, body.State
				state.events = append(state.events, "create-paused-card")
			}
			writer.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(writer, `{"token":"card-token-1","pan":"4111111111111111","cvv":"123","state":%q,"type":"VIRTUAL","cardholder_currency":%q,"spend_limit":%d,"spend_limit_duration":"FOREVER","exp_month":%q,"exp_year":%q}`, body.State, state.cardCurrency, body.SpendLimit, body.ExpMonth, body.ExpYear)
		case request.Method == http.MethodPost && request.URL.Path == "/v2/auth_rules":
			var body struct {
				CardTokens []string `json:"card_tokens"`
				Parameters struct {
					Conditions []struct {
						Attribute string   `json:"attribute"`
						Operation string   `json:"operation"`
						Value     []string `json:"value"`
					} `json:"conditions"`
				} `json:"parameters"`
			}
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			require.Equal(t, []string{"card-token-1"}, body.CardTokens)
			require.Equal(t, "MERCHANT_ID", body.Parameters.Conditions[0].Attribute)
			require.Equal(t, "IS_NOT_ONE_OF", body.Parameters.Conditions[0].Operation)
			state.counterparty = body.Parameters.Conditions[0].Value[0]
			state.events = append(state.events, "create-rule")
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"token":"rule-token-1","state":"INACTIVE"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2/auth_rules/rule-token-1/promote":
			state.ruleActive = true
			state.events = append(state.events, "promote-rule")
			_, _ = writer.Write([]byte(`{"token":"rule-token-1","state":"ACTIVE"}`))
		case request.Method == http.MethodPatch && request.URL.Path == "/v1/cards/card-token-1":
			var body struct {
				State string `json:"state"`
			}
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			state.cardState = body.State
			state.events = append(state.events, "card-"+strings.ToLower(body.State))
			_, _ = fmt.Fprintf(writer, `{"token":"card-token-1","state":%q}`, body.State)
		case request.Method == http.MethodDelete && request.URL.Path == "/v2/auth_rules/rule-token-1":
			state.ruleDeleted = true
			state.ruleActive = false
			state.events = append(state.events, "delete-rule")
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/cards/card-token-1":
			state.events = append(state.events, "get-card")
			_, _ = fmt.Fprintf(writer, `{"token":"card-token-1","state":%q,"type":"VIRTUAL","cardholder_currency":%q,"spend_limit":%d,"spend_limit_duration":"FOREVER","exp_month":%q,"exp_year":%q}`, state.cardState, state.cardCurrency, state.capMinor, state.expMonth, state.expYear)
		case request.Method == http.MethodGet && request.URL.Path == "/v2/auth_rules/rule-token-1":
			require.True(t, state.ruleActive)
			state.events = append(state.events, "get-rule")
			_, _ = fmt.Fprintf(writer, `{"token":"rule-token-1","state":"ACTIVE","current_version":{"parameters":{"action":"DECLINE","conditions":[{"attribute":"MERCHANT_ID","operation":"IS_NOT_ONE_OF","value":[%q]}]}}}`, state.counterparty)
		case request.Method == http.MethodPost && request.URL.Path == "/v1/simulate/authorize":
			var body struct {
				Amount             int64  `json:"amount"`
				MerchantAcceptorID string `json:"merchant_acceptor_id"`
			}
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			if body.Amount > state.capMinor || body.MerchantAcceptorID != state.counterparty {
				state.declines++
				writer.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = writer.Write([]byte(`{"message":"provider scope declined authorization"}`))
				return
			}
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"token":"transaction-1","status":"PENDING"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v1/simulate/clearing":
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"debugging_request_id":"clear-1"}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, state
}

func endOfUTCMonth(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
}

type loseFirstCardResponse struct {
	base http.RoundTripper
	once sync.Once
}

func (transport *loseFirstCardResponse) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := transport.base.RoundTrip(request)
	if err != nil || request.Method != http.MethodPost || request.URL.Path != "/v1/cards" {
		return response, err
	}
	drop := false
	transport.once.Do(func() { drop = true })
	if !drop {
		return response, nil
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	return nil, errors.New("simulated lost response after provider created the card")
}

// [REQ:TRS-P1-003] The provider receives the mandate cap, counterparty rule,
// and expiry; inspection reads those controls back from provider endpoints.
func TestIssueAndInspectProviderEnforcedScope(t *testing.T) {
	server, state := newSandbox(t)
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	expires := time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)
	scope := card.Scope{MandateReference: "mandate-1", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: expires}
	issued, err := adapter.Issue(context.Background(), card.IssueCommand{InstrumentID: "instrument-1", IdempotencyKey: "issue-key-1", Credential: `{"api_key":"sandbox-api-key"}`, Scope: scope})
	require.NoError(t, err)
	require.Equal(t, "card-token-1", issued.ExternalID)
	require.NotContains(t, issued.Credential, "instrument-1")
	require.True(t, card.EqualScope(scope, issued.Scope))
	state.mu.Lock()
	require.Equal(t, []string{"create-paused-card", "create-rule", "promote-rule", "get-rule", "card-open", "get-card"}, state.events)
	state.mu.Unlock()

	inspected, err := adapter.Inspect(context.Background(), card.InspectQuery{ExternalID: issued.ExternalID, Credential: issued.Credential})
	require.NoError(t, err)
	require.True(t, card.EqualScope(scope, inspected.Scope))
}

func TestIssueClosesCardAndDeletesRuleOnConfigurationFailure(t *testing.T) {
	tests := []struct {
		name       string
		failAt     string
		wantEvents []string
	}{
		{name: "rule creation", failAt: "create-rule", wantEvents: []string{"create-paused-card", "create-rule", "card-closed"}},
		{name: "rule promotion", failAt: "promote-rule", wantEvents: []string{"create-paused-card", "create-rule", "promote-rule", "card-closed", "delete-rule"}},
		{name: "rule verification", failAt: "verify-rule", wantEvents: []string{"create-paused-card", "create-rule", "promote-rule", "get-rule", "card-closed", "delete-rule"}},
		{name: "card opening", failAt: "open-card", wantEvents: []string{"create-paused-card", "create-rule", "promote-rule", "get-rule", "card-open", "card-closed", "delete-rule"}},
		{name: "card verification", failAt: "verify-card", wantEvents: []string{"create-paused-card", "create-rule", "promote-rule", "get-rule", "card-open", "get-card", "card-closed", "delete-rule"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, events := newIssuanceFailureSandbox(t, test.failAt, false)
			adapter, err := lithic.New(server.URL, server.Client())
			require.NoError(t, err)
			_, err = adapter.Issue(context.Background(), card.IssueCommand{
				InstrumentID: "instrument-1", IdempotencyKey: "issue-key-1", Credential: `{"api_key":"sandbox-api-key"}`,
				Scope: card.Scope{MandateReference: "mandate-1", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)},
			})
			require.Error(t, err)
			require.Equal(t, test.wantEvents, events.snapshot())
		})
	}
}

func TestIssueReportsCleanupFailure(t *testing.T) {
	server, events := newIssuanceFailureSandbox(t, "promote-rule", true)
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	_, err = adapter.Issue(context.Background(), card.IssueCommand{
		InstrumentID: "instrument-1", IdempotencyKey: "issue-key-1", Credential: `{"api_key":"sandbox-api-key"}`,
		Scope: card.Scope{MandateReference: "mandate-1", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)},
	})
	require.ErrorContains(t, err, "activate provider merchant rule")
	require.ErrorContains(t, err, "provider cleanup failed")
	require.ErrorContains(t, err, "close provider card")
	require.ErrorContains(t, err, "delete provider merchant rule")
	require.Equal(t, []string{"create-paused-card", "create-rule", "promote-rule", "card-closed", "delete-rule"}, events.snapshot())
}

func TestInspectRejectsPausedProviderCard(t *testing.T) {
	server, state := newSandbox(t)
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	scope := card.Scope{MandateReference: "mandate-1", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)}
	issued, err := adapter.Issue(context.Background(), card.IssueCommand{InstrumentID: "instrument-1", IdempotencyKey: "issue-key-1", Credential: `{"api_key":"sandbox-api-key"}`, Scope: scope})
	require.NoError(t, err)
	state.mu.Lock()
	state.cardState = "PAUSED"
	state.mu.Unlock()
	_, err = adapter.Inspect(context.Background(), card.InspectQuery{ExternalID: issued.ExternalID, Credential: issued.Credential})
	require.ErrorContains(t, err, "provider scope differs")
}

type eventLog struct {
	mu     sync.Mutex
	events []string
}

func (log *eventLog) add(event string) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.events = append(log.events, event)
}

func (log *eventLog) snapshot() []string {
	log.mu.Lock()
	defer log.mu.Unlock()
	return append([]string(nil), log.events...)
}

func newIssuanceFailureSandbox(t *testing.T, failAt string, cleanupFailure bool) (*httptest.Server, *eventLog) {
	t.Helper()
	events := &eventLog{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "sandbox-api-key", request.Header.Get("Authorization"))
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/cards":
			events.add("create-paused-card")
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"token":"card-token-1","pan":"4111111111111111","cvv":"123","state":"PAUSED","type":"VIRTUAL","cardholder_currency":"USD","spend_limit":750,"spend_limit_duration":"FOREVER","exp_month":"02","exp_year":"2027"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2/auth_rules":
			events.add("create-rule")
			if failAt == "create-rule" {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"token":"rule-token-1","state":"INACTIVE"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2/auth_rules/rule-token-1/promote":
			events.add("promote-rule")
			if failAt == "promote-rule" {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = writer.Write([]byte(`{"token":"rule-token-1","state":"ACTIVE"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v2/auth_rules/rule-token-1":
			events.add("get-rule")
			state := "ACTIVE"
			if failAt == "verify-rule" {
				state = "INACTIVE"
			}
			_, _ = fmt.Fprintf(writer, `{"token":"rule-token-1","state":%q,"current_version":{"parameters":{"action":"DECLINE","conditions":[{"attribute":"MERCHANT_ID","operation":"IS_NOT_ONE_OF","value":["MERCHANT123"]}]}}}`, state)
		case request.Method == http.MethodPatch && request.URL.Path == "/v1/cards/card-token-1":
			var body struct {
				State string `json:"state"`
			}
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			events.add("card-" + strings.ToLower(body.State))
			if (body.State == "OPEN" && failAt == "open-card") || (body.State == "CLOSED" && cleanupFailure) {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = fmt.Fprintf(writer, `{"token":"card-token-1","state":%q}`, body.State)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/cards/card-token-1":
			events.add("get-card")
			state := "OPEN"
			if failAt == "verify-card" {
				state = "PAUSED"
			}
			_, _ = fmt.Fprintf(writer, `{"token":"card-token-1","state":%q,"type":"VIRTUAL","cardholder_currency":"USD","spend_limit":750,"spend_limit_duration":"FOREVER","exp_month":"02","exp_year":"2027"}`, state)
		case request.Method == http.MethodDelete && request.URL.Path == "/v2/auth_rules/rule-token-1":
			events.add("delete-rule")
			if cleanupFailure {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, events
}

// [REQ:TRS-P1-003] These attempts are deliberately sent to the provider; the
// adapter does not pre-decline them, and the provider returns HTTP 422.
func TestProviderDeclinesAmountAndCounterpartyOutsideMandate(t *testing.T) {
	server, state := newSandbox(t)
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	scope := card.Scope{MandateReference: "mandate-1", AmountMinor: 500, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: endOfUTCMonth(time.Now().UTC().AddDate(0, 1, 0))}
	issued, err := adapter.Issue(context.Background(), card.IssueCommand{InstrumentID: "instrument-1", IdempotencyKey: "issue-key-1", Credential: `{"api_key":"sandbox-api-key"}`, Scope: scope})
	require.NoError(t, err)

	base := rail.SettleCommand{SettlementID: "settlement-1", AuthorizationID: "authorization-1", MandateReference: scope.MandateReference, InstrumentID: "instrument-1", IdempotencyKey: "settle-key-1", AmountMinor: scope.AmountMinor + 1, Currency: scope.Currency, Counterparty: scope.Counterparty, Credential: issued.Credential}
	result, err := adapter.Settle(context.Background(), base)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)
	require.Equal(t, "provider_card_decline", result.Basis)

	base.AmountMinor = scope.AmountMinor
	base.Counterparty = "MERCHANT999"
	base.IdempotencyKey = "settle-key-2"
	result, err = adapter.Settle(context.Background(), base)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)

	state.mu.Lock()
	defer state.mu.Unlock()
	require.Equal(t, 2, state.declines)
}

func TestAdapterRejectsCredentialWithUnknownFields(t *testing.T) {
	adapter, err := lithic.New("https://sandbox.lithic.com", http.DefaultClient)
	require.NoError(t, err)
	_, err = adapter.Issue(context.Background(), card.IssueCommand{InstrumentID: "instrument-1", IdempotencyKey: "issue-key", Credential: `{"api_key":"secret","unexpected":"leak"}`, Scope: card.Scope{MandateReference: "mandate-1", AmountMinor: 1, Currency: "USD", Counterparty: "merchant", ExpiresAt: time.Now().UTC().Add(time.Hour)}})
	require.Error(t, err)
	require.NotContains(t, strings.ToLower(err.Error()), "secret")
}

// [REQ:TRS-P1-003] An unrepresentable or unsupported expiry is refused before
// the provider can issue a card with weaker expiry than the mandate.
func TestIssueRejectsExpiredOrBeyondProviderCardHorizonBeforeRequest(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	for _, tc := range []struct {
		name      string
		expiresAt time.Time
	}{
		{name: "already expired", expiresAt: time.Now().UTC().Add(-time.Minute)},
		{name: "mid-month cannot be represented by provider expiry", expiresAt: time.Now().UTC().AddDate(0, 1, 0)},
		{name: "beyond provider five-year maximum", expiresAt: time.Now().UTC().AddDate(6, 0, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := adapter.Issue(context.Background(), card.IssueCommand{
				InstrumentID: "instrument-1", IdempotencyKey: "issue-key", Credential: `{"api_key":"sandbox-api-key"}`,
				Scope: card.Scope{MandateReference: "mandate-1", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: tc.expiresAt},
			})
			require.ErrorIs(t, err, card.ErrInvalid)
		})
	}
	require.Zero(t, requests.Load(), "invalid expiration must be refused before any vendor request")
}

// [REQ:TRS-P1-003] Lithic spend limits are USD cents; a non-USD mandate must
// be refused before issuing a card with a different monetary meaning.
func TestIssueRejectsNonUSDSpendScopeBeforeRequest(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	_, err = adapter.Issue(context.Background(), card.IssueCommand{
		InstrumentID: "instrument-eur", IdempotencyKey: "issue-eur", Credential: `{"api_key":"sandbox-api-key"}`,
		Scope: card.Scope{MandateReference: "mandate-eur", AmountMinor: 750, Currency: "EUR", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)},
	})
	require.ErrorIs(t, err, card.ErrInvalid)
	require.Zero(t, requests.Load(), "non-USD mandate must be refused before any provider request")
}

// [REQ:TRS-P1-003] Provider readback must confirm the card's currency before
// Treasury accepts the provider spend-limit amount as mandate-bound.
func TestIssueRefusesProviderCardWithMismatchedCardholderCurrency(t *testing.T) {
	server, state := newSandbox(t)
	state.cardCurrency = "EUR"
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	_, err = adapter.Issue(context.Background(), card.IssueCommand{
		InstrumentID: "instrument-currency-mismatch", IdempotencyKey: "issue-currency-mismatch", Credential: `{"api_key":"sandbox-api-key"}`,
		Scope: card.Scope{MandateReference: "mandate-usd", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)},
	})
	require.ErrorIs(t, err, lithic.ErrInvalid)
	state.mu.Lock()
	defer state.mu.Unlock()
	require.Equal(t, []string{"create-paused-card", "card-closed"}, state.events, "mismatched currency must be closed before rules or activation")
}

// [REQ:TRS-P1-003] An expired card mandate cannot dispatch a provider
// authorization.
func TestSettleAfterMandateExpiryRefusesBeforeProviderRequest(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	credential, err := json.Marshal(map[string]any{
		"api_key": "sandbox-api-key", "card_token": "card-token-1", "pan": "4111111111111111", "rule_token": "rule-token-1",
		"mandate_reference": "mandate-1", "amount_minor": 500, "currency": "USD", "counterparty": "merchant-1",
		"mandate_expires_at": time.Now().UTC().Add(-time.Minute), "provider_exp_month": "11", "provider_exp_year": "2026",
	})
	require.NoError(t, err)
	result, err := adapter.Settle(context.Background(), rail.SettleCommand{
		SettlementID: "settlement-1", AuthorizationID: "authorization-1", MandateReference: "mandate-1", InstrumentID: "instrument-1",
		IdempotencyKey: "settle-1", AmountMinor: 1, Currency: "USD", Counterparty: "merchant-1", Credential: string(credential),
	})
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)
	require.Equal(t, "mandate_expired", result.Basis)
	require.Zero(t, requests.Load(), "expired mandate must never dispatch a vendor authorization")
}

// [REQ:TRS-P1-003] Ambiguous clearing remains queryable and never causes a
// second authorization or clear dispatch.
func TestSettleAmbiguousClearReturnsTransactionHandleForQueryReconciliation(t *testing.T) {
	var authCalls, clearCalls, queryCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v1/simulate/authorize":
			authCalls.Add(1)
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"token":"transaction-1","status":"PENDING"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v1/simulate/clearing":
			clearCalls.Add(1)
			writer.WriteHeader(http.StatusInternalServerError)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/transactions/transaction-1":
			if queryCalls.Add(1) == 1 {
				_, _ = writer.Write([]byte(`{"token":"transaction-1","status":"PENDING"}`))
			} else {
				_, _ = writer.Write([]byte(`{"token":"transaction-1","status":"SETTLED"}`))
			}
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	command := rail.SettleCommand{
		SettlementID: "settlement-1", AuthorizationID: "authorization-1", MandateReference: "mandate-1", InstrumentID: "instrument-1",
		IdempotencyKey: "settlement-idempotency-1", AmountMinor: 100, Currency: "USD", Counterparty: "merchant-1", Credential: issuedCredentialJSON(t, time.Now().UTC().Add(time.Hour)),
	}
	first, err := adapter.Settle(context.Background(), command)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	require.Equal(t, "transaction-1", first.ExternalID)
	require.Equal(t, "lithic-sandbox:transaction-1", first.ReceiptReference)

	query := rail.Query{ExternalID: first.ExternalID, Credential: command.Credential}
	firstRead, err := adapter.QueryOutcome(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeUnknown, firstRead.Outcome)
	secondRead, err := adapter.QueryOutcome(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, secondRead.Outcome)
	require.Equal(t, first.ExternalID, secondRead.ExternalID)
	require.EqualValues(t, 1, authCalls.Load(), "reconciliation reads the known transaction and never redispatches authorization")
	require.EqualValues(t, 1, clearCalls.Load())
}

// [REQ:TRS-P1-003] Retries preserve one provider card identity without
// allowing a different instrument to alias that idempotency key.
func TestIssueIdempotencyKeyIsStablePerInstrumentAndIsolatesDifferentInstruments(t *testing.T) {
	var mu sync.Mutex
	var keys []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		keys = append(keys, request.Header.Get("Idempotency-Key"))
		mu.Unlock()
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	scope := card.Scope{MandateReference: "mandate-1", AmountMinor: 500, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)}
	for _, instrumentID := range []string{"instrument-1", "instrument-1", "instrument-2"} {
		_, err := adapter.Issue(context.Background(), card.IssueCommand{InstrumentID: instrumentID, IdempotencyKey: "same-client-key", Credential: `{"api_key":"sandbox-api-key"}`, Scope: scope})
		require.Error(t, err)
	}
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, keys, 3)
	require.NotEmpty(t, keys[0])
	require.Equal(t, keys[0], keys[1], "same instrument retries must reuse the provider key")
	require.NotEqual(t, keys[0], keys[2], "different instruments must not alias one provider card")
}

// [REQ:TRS-P1-003] A provider-accepted create with a lost response recovers
// the original card when retried with the same request identity.
func TestIssueRetryAfterLostCreateResponseReusesOneProviderCard(t *testing.T) {
	server, state := newSandbox(t)
	client := server.Client()
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = &loseFirstCardResponse{base: base}
	adapter, err := lithic.New(server.URL, client)
	require.NoError(t, err)
	scope := card.Scope{MandateReference: "mandate-retry", AmountMinor: 750, Currency: "USD", Counterparty: "MERCHANT123", ExpiresAt: time.Date(2027, 2, 28, 23, 59, 59, 999999999, time.UTC)}
	command := card.IssueCommand{InstrumentID: "instrument-retry", IdempotencyKey: "retry-after-lost-create", Credential: `{"api_key":"sandbox-api-key"}`, Scope: scope}
	_, err = adapter.Issue(context.Background(), command)
	require.ErrorContains(t, err, "issue provider card")
	issued, err := adapter.Issue(context.Background(), command)
	require.NoError(t, err)
	require.Equal(t, "card-token-1", issued.ExternalID)

	state.mu.Lock()
	defer state.mu.Unlock()
	require.Equal(t, 1, state.cardCreations, "the provider idempotency replay must return the original card")
	require.Len(t, state.cardKeys, 1)
	require.Equal(t, []string{"create-paused-card", "create-rule", "promote-rule", "get-rule", "card-open", "get-card"}, state.events)
}

// [REQ:TRS-P1-003] Query reconciliation cannot replace the requested provider
// transaction identity with a different transaction response.
func TestQueryOutcomeRejectsMismatchedTransactionAndKeepsRequestedHandle(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v1/transactions/transaction-expected", request.URL.Path)
		if calls.Add(1) == 1 {
			_, _ = writer.Write([]byte(`{"token":"transaction-other","status":"SETTLED"}`))
			return
		}
		_, _ = writer.Write([]byte(`{"token":"transaction-expected","status":"SETTLED"}`))
	}))
	defer server.Close()
	adapter, err := lithic.New(server.URL, server.Client())
	require.NoError(t, err)
	query := rail.Query{ExternalID: "transaction-expected", Credential: issuedCredentialJSON(t, time.Now().UTC().Add(time.Hour))}
	first, err := adapter.QueryOutcome(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeUnknown, first.Outcome)
	require.Equal(t, query.ExternalID, first.ExternalID)
	require.Equal(t, "lithic-sandbox:"+query.ExternalID, first.ReceiptReference)

	second, err := adapter.QueryOutcome(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeSettled, second.Outcome)
	require.Equal(t, query.ExternalID, second.ExternalID)
	require.EqualValues(t, 2, calls.Load())
}

func issuedCredentialJSON(t *testing.T, expiresAt time.Time) string {
	t.Helper()
	value, err := json.Marshal(map[string]any{
		"api_key": "sandbox-api-key", "card_token": "card-token-1", "pan": "4111111111111111", "rule_token": "rule-token-1",
		"mandate_reference": "mandate-1", "amount_minor": 500, "currency": "USD", "counterparty": "merchant-1",
		"mandate_expires_at": expiresAt, "provider_exp_month": expiresAt.Format("01"), "provider_exp_year": expiresAt.Format("2006"),
	})
	require.NoError(t, err)
	return string(value)
}

// [REQ:TRS-P1-003] This operator-gated integration proves the real issuing
// sandbox, not Treasury, declines both an excessive amount and another
// merchant acceptor. It intentionally resolves the API key through Credential
// Authority at use time and never accepts it through a test flag or log.
func TestLithicSandboxProviderEnforcesMandateScope(t *testing.T) {
	if os.Getenv("TREASURY_LITHIC_SANDBOX_INTEGRATION") != "1" {
		t.Skip("set TREASURY_LITHIC_SANDBOX_INTEGRATION=1 after provisioning the operator-created sandbox credential")
	}
	authority, err := credentialauthority.Default()
	require.NoError(t, err)
	client, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: authority})
	require.NoError(t, err)
	providerCredential, err := client.Resolve(context.Background(), "vrooli/treasury/lithic", "value")
	require.NoError(t, err)
	adapter, err := lithic.New("https://sandbox.lithic.com", &http.Client{Timeout: 30 * time.Second})
	require.NoError(t, err)

	unique := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	scope := card.Scope{MandateReference: "sandbox-mandate-" + unique, AmountMinor: 500, Currency: "USD", Counterparty: "VROOLI1", ExpiresAt: endOfUTCMonth(time.Now().UTC().AddDate(0, 1, 0))}
	issued, err := adapter.Issue(context.Background(), card.IssueCommand{InstrumentID: "sandbox-instrument-" + unique, IdempotencyKey: "sandbox-issue-" + unique, Credential: providerCredential, Scope: scope})
	require.NoError(t, err)
	inspected, err := adapter.Inspect(context.Background(), card.InspectQuery{ExternalID: issued.ExternalID, Credential: issued.Credential})
	require.NoError(t, err)
	require.True(t, card.EqualScope(scope, inspected.Scope))

	attempt := rail.SettleCommand{SettlementID: "sandbox-settlement-cap-" + unique, AuthorizationID: "sandbox-auth-cap-" + unique, MandateReference: scope.MandateReference, InstrumentID: "sandbox-instrument-" + unique, IdempotencyKey: "sandbox-cap-" + unique, AmountMinor: scope.AmountMinor + 1, Currency: scope.Currency, Counterparty: scope.Counterparty, Credential: issued.Credential}
	result, err := adapter.Settle(context.Background(), attempt)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)

	attempt.SettlementID = "sandbox-settlement-merchant-" + unique
	attempt.AuthorizationID = "sandbox-auth-merchant-" + unique
	attempt.IdempotencyKey = "sandbox-merchant-" + unique
	attempt.AmountMinor = scope.AmountMinor
	attempt.Counterparty = "VROOLI2"
	result, err = adapter.Settle(context.Background(), attempt)
	require.NoError(t, err)
	require.Equal(t, rail.OutcomeFailed, result.Outcome)
}
