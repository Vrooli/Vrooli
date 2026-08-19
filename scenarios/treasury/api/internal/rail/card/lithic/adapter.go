// Package lithic implements the scoped-card and settlement contracts against
// Lithic's card-issuing sandbox. Provider names and payloads stay in this
// package; the contracts it satisfies remain vendor-neutral.
package lithic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"treasury/internal/httpc"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
)

const maxResponseBytes = 1 << 20

const cleanupTimeout = 10 * time.Second

var ErrInvalid = errors.New("invalid lithic scoped-card request")

type Adapter struct {
	baseURL string
	client  httpc.Doer
}

func New(baseURL string, client httpc.Doer) (*Adapter, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return nil, fmt.Errorf("%w: base URL must be an absolute origin", ErrInvalid)
	}
	if parsed.Scheme != "https" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" {
		return nil, fmt.Errorf("%w: base URL must use HTTPS except on loopback", ErrInvalid)
	}
	if parsed.Hostname() != "sandbox.lithic.com" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" {
		return nil, fmt.Errorf("%w: this adapter is restricted to the Lithic sandbox", ErrInvalid)
	}
	if client == nil {
		return nil, fmt.Errorf("%w: HTTP client is required", ErrInvalid)
	}
	return &Adapter{baseURL: baseURL, client: client}, nil
}

func (*Adapter) Name() string { return "lithic-sandbox-card" }

type providerCredential struct {
	APIKey       string `json:"api_key"`
	AccountToken string `json:"account_token,omitempty"`
}

type issuedCredential struct {
	APIKey           string    `json:"api_key"`
	CardToken        string    `json:"card_token"`
	PAN              string    `json:"pan"`
	CVV              string    `json:"cvv,omitempty"`
	RuleToken        string    `json:"rule_token"`
	MandateReference string    `json:"mandate_reference"`
	AmountMinor      int64     `json:"amount_minor"`
	Currency         string    `json:"currency"`
	Counterparty     string    `json:"counterparty"`
	MandateExpiresAt time.Time `json:"mandate_expires_at"`
	ProviderExpMonth string    `json:"provider_exp_month"`
	ProviderExpYear  string    `json:"provider_exp_year"`
}

type createCardRequest struct {
	Type               string `json:"type"`
	State              string `json:"state"`
	SpendLimit         int64  `json:"spend_limit"`
	SpendLimitDuration string `json:"spend_limit_duration"`
	ExpMonth           string `json:"exp_month"`
	ExpYear            string `json:"exp_year"`
	Memo               string `json:"memo"`
	AccountToken       string `json:"account_token,omitempty"`
}

type cardResponse struct {
	Token              string `json:"token"`
	PAN                string `json:"pan"`
	CVV                string `json:"cvv"`
	State              string `json:"state"`
	Type               string `json:"type"`
	SpendLimit         int64  `json:"spend_limit"`
	SpendLimitDuration string `json:"spend_limit_duration"`
	ExpMonth           string `json:"exp_month"`
	ExpYear            string `json:"exp_year"`
}

type updateCardRequest struct {
	State string `json:"state"`
}

type condition struct {
	Attribute string   `json:"attribute"`
	Operation string   `json:"operation"`
	Value     []string `json:"value"`
}

type ruleParameters struct {
	Action     string      `json:"action"`
	Conditions []condition `json:"conditions"`
}

type createRuleRequest struct {
	Name        string         `json:"name"`
	CardTokens  []string       `json:"card_tokens"`
	Type        string         `json:"type"`
	EventStream string         `json:"event_stream"`
	Parameters  ruleParameters `json:"parameters"`
}

type ruleVersion struct {
	Parameters ruleParameters `json:"parameters"`
}

type ruleResponse struct {
	Token          string       `json:"token"`
	State          string       `json:"state"`
	CurrentVersion *ruleVersion `json:"current_version"`
	DraftVersion   *ruleVersion `json:"draft_version"`
}

// Issue creates a paused virtual card with a lifetime provider-side spend
// limit, activates and reads back a card-level rule that declines every
// merchant acceptor ID except the mandate counterparty, and only then opens and
// reads back the card. Any failure after card creation closes the card and
// removes the rule before returning. The returned secret is intended for
// immediate placement in Credential Authority.
func (a *Adapter) Issue(ctx context.Context, command card.IssueCommand) (card.Issued, error) {
	if err := card.ValidateScope(command.Scope); err != nil {
		return card.Issued{}, err
	}
	credential, err := parseProviderCredential(command.Credential)
	if err != nil {
		return card.Issued{}, err
	}
	expires := command.Scope.ExpiresAt.UTC()
	cardRequest := createCardRequest{
		Type: "VIRTUAL", State: "PAUSED", SpendLimit: command.Scope.AmountMinor,
		SpendLimitDuration: "FOREVER", ExpMonth: expires.Format("01"), ExpYear: expires.Format("2006"),
		Memo: boundedMemo(command.Scope.MandateReference, command.Scope.Counterparty), AccountToken: credential.AccountToken,
	}
	var created cardResponse
	if err := a.request(ctx, http.MethodPost, "/v1/cards", credential.APIKey, deterministicUUID(command.IdempotencyKey), cardRequest, &created, http.StatusOK, http.StatusCreated); err != nil {
		return card.Issued{}, fmt.Errorf("issue provider card: %w", err)
	}
	if strings.TrimSpace(created.Token) == "" {
		return card.Issued{}, fmt.Errorf("%w: provider omitted card token", ErrInvalid)
	}
	fail := func(cause error, ruleToken string) (card.Issued, error) {
		return card.Issued{}, a.failIssue(ctx, credential.APIKey, created.Token, ruleToken, cause)
	}
	if strings.TrimSpace(created.PAN) == "" || created.State != "PAUSED" || created.SpendLimit != command.Scope.AmountMinor || created.SpendLimitDuration != "FOREVER" || created.ExpMonth != cardRequest.ExpMonth || created.ExpYear != cardRequest.ExpYear {
		return fail(fmt.Errorf("%w: provider did not create the requested paused card scope", ErrInvalid), "")
	}

	ruleRequest := createRuleRequest{
		Name: "Treasury counterparty " + shortHash(command.InstrumentID), CardTokens: []string{created.Token},
		Type: "CONDITIONAL_ACTION", EventStream: "AUTHORIZATION",
		Parameters: ruleParameters{Action: "DECLINE", Conditions: []condition{{Attribute: "MERCHANT_ID", Operation: "IS_NOT_ONE_OF", Value: []string{strings.ToUpper(command.Scope.Counterparty)}}}},
	}
	var rule ruleResponse
	if err := a.request(ctx, http.MethodPost, "/v2/auth_rules", credential.APIKey, deterministicUUID(command.IdempotencyKey+":merchant-rule"), ruleRequest, &rule, http.StatusCreated); err != nil {
		return fail(fmt.Errorf("create provider merchant rule: %w", err), "")
	}
	if strings.TrimSpace(rule.Token) == "" {
		return fail(fmt.Errorf("%w: provider omitted merchant rule token", ErrInvalid), "")
	}
	if err := a.request(ctx, http.MethodPost, "/v2/auth_rules/"+url.PathEscape(rule.Token)+"/promote", credential.APIKey, "", nil, &rule, http.StatusOK, http.StatusCreated); err != nil {
		return fail(fmt.Errorf("activate provider merchant rule: %w", err), rule.Token)
	}
	var activeRule ruleResponse
	if err := a.request(ctx, http.MethodGet, "/v2/auth_rules/"+url.PathEscape(rule.Token), credential.APIKey, "", nil, &activeRule, http.StatusOK); err != nil {
		return fail(fmt.Errorf("verify provider merchant rule: %w", err), rule.Token)
	}
	if activeRule.Token != rule.Token || activeRule.State != "ACTIVE" || activeRule.CurrentVersion == nil || !permitsOnly(activeRule.CurrentVersion.Parameters, command.Scope.Counterparty) {
		return fail(fmt.Errorf("%w: provider merchant rule is not active with the requested counterparty scope", ErrInvalid), rule.Token)
	}

	var opened cardResponse
	if err := a.request(ctx, http.MethodPatch, "/v1/cards/"+url.PathEscape(created.Token), credential.APIKey, "", updateCardRequest{State: "OPEN"}, &opened, http.StatusOK); err != nil {
		return fail(fmt.Errorf("open provider card: %w", err), rule.Token)
	}
	if opened.Token != "" && (opened.Token != created.Token || opened.State != "OPEN") {
		return fail(fmt.Errorf("%w: provider did not open the scoped card", ErrInvalid), rule.Token)
	}
	var verified cardResponse
	if err := a.request(ctx, http.MethodGet, "/v1/cards/"+url.PathEscape(created.Token), credential.APIKey, "", nil, &verified, http.StatusOK); err != nil {
		return fail(fmt.Errorf("verify provider card: %w", err), rule.Token)
	}
	if !matchesCardScope(verified, created.Token, command.Scope.AmountMinor, cardRequest.ExpMonth, cardRequest.ExpYear) {
		return fail(fmt.Errorf("%w: provider card is not open with the requested mandate scope", ErrInvalid), rule.Token)
	}

	// #nosec G117 -- the API key is intentionally re-encrypted with the issued
	// card envelope by Credential Authority; it is never persisted or logged.
	secret, err := json.Marshal(issuedCredential{
		APIKey: credential.APIKey, CardToken: created.Token, PAN: created.PAN, CVV: created.CVV, RuleToken: rule.Token,
		MandateReference: command.Scope.MandateReference, AmountMinor: command.Scope.AmountMinor,
		Currency: strings.ToUpper(command.Scope.Currency), Counterparty: strings.ToLower(command.Scope.Counterparty),
		MandateExpiresAt: expires, ProviderExpMonth: created.ExpMonth, ProviderExpYear: created.ExpYear,
	})
	if err != nil {
		return fail(fmt.Errorf("encode issued credential: %w", err), rule.Token)
	}
	return card.Issued{ExternalID: created.Token, Credential: string(secret), Scope: command.Scope}, nil
}

// Inspect reads the card and active merchant rule from the provider and only
// reports the scope after both match the credential-bound mandate projection.
func (a *Adapter) Inspect(ctx context.Context, query card.InspectQuery) (card.Issued, error) {
	credential, err := parseIssuedCredential(query.Credential)
	if err != nil {
		return card.Issued{}, err
	}
	if query.ExternalID != credential.CardToken {
		return card.Issued{}, fmt.Errorf("%w: external card reference does not match credential", ErrInvalid)
	}
	var remoteCard cardResponse
	if err := a.request(ctx, http.MethodGet, "/v1/cards/"+url.PathEscape(credential.CardToken), credential.APIKey, "", nil, &remoteCard, http.StatusOK); err != nil {
		return card.Issued{}, fmt.Errorf("inspect provider card: %w", err)
	}
	var remoteRule ruleResponse
	if err := a.request(ctx, http.MethodGet, "/v2/auth_rules/"+url.PathEscape(credential.RuleToken), credential.APIKey, "", nil, &remoteRule, http.StatusOK); err != nil {
		return card.Issued{}, fmt.Errorf("inspect provider merchant rule: %w", err)
	}
	if !matchesCardScope(remoteCard, credential.CardToken, credential.AmountMinor, credential.ProviderExpMonth, credential.ProviderExpYear) || remoteRule.Token != credential.RuleToken || remoteRule.State != "ACTIVE" || remoteRule.CurrentVersion == nil || !permitsOnly(remoteRule.CurrentVersion.Parameters, credential.Counterparty) {
		return card.Issued{}, fmt.Errorf("%w: provider scope differs from mandate-bound credential", ErrInvalid)
	}
	scope := card.Scope{MandateReference: credential.MandateReference, AmountMinor: credential.AmountMinor, Currency: credential.Currency, Counterparty: credential.Counterparty, ExpiresAt: credential.MandateExpiresAt}
	return card.Issued{ExternalID: remoteCard.Token, Scope: scope}, nil
}

func matchesCardScope(remote cardResponse, token string, amountMinor int64, expMonth, expYear string) bool {
	return remote.Token == token && remote.State == "OPEN" && remote.SpendLimit == amountMinor && remote.SpendLimitDuration == "FOREVER" && remote.ExpMonth == expMonth && remote.ExpYear == expYear
}

func (a *Adapter) failIssue(ctx context.Context, apiKey, cardToken, ruleToken string, cause error) error {
	cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	var cleanupErrors []error
	if err := a.request(cleanupContext, http.MethodPatch, "/v1/cards/"+url.PathEscape(cardToken), apiKey, "", updateCardRequest{State: "CLOSED"}, nil, http.StatusOK); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("close provider card: %w", err))
	}
	if ruleToken != "" {
		if err := a.request(cleanupContext, http.MethodDelete, "/v2/auth_rules/"+url.PathEscape(ruleToken), apiKey, "", nil, nil, http.StatusOK, http.StatusNoContent); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("delete provider merchant rule: %w", err))
		}
	}
	if cleanupErr := errors.Join(cleanupErrors...); cleanupErr != nil {
		return fmt.Errorf("%w; provider cleanup failed: %v", cause, cleanupErr)
	}
	return cause
}

type authorizeRequest struct {
	Amount             int64  `json:"amount"`
	Descriptor         string `json:"descriptor"`
	PAN                string `json:"pan"`
	MerchantAcceptorID string `json:"merchant_acceptor_id"`
	MerchantCurrency   string `json:"merchant_currency"`
}

type transactionResponse struct {
	Token  string `json:"token"`
	Status string `json:"status"`
	Result string `json:"result"`
}

func (a *Adapter) Settle(ctx context.Context, command rail.SettleCommand) (rail.Result, error) {
	if err := rail.ValidateSettle(command); err != nil {
		return rail.Result{}, err
	}
	credential, err := parseIssuedCredential(command.Credential)
	if err != nil {
		return rail.Result{}, err
	}
	if command.MandateReference != credential.MandateReference {
		return rail.Result{}, fmt.Errorf("%w: card credential belongs to a different mandate", ErrInvalid)
	}
	attempt := authorizeRequest{Amount: command.AmountMinor, Descriptor: boundedDescriptor(command.Counterparty), PAN: credential.PAN, MerchantAcceptorID: strings.ToUpper(command.Counterparty), MerchantCurrency: strings.ToUpper(command.Currency)}
	var authorized transactionResponse
	status, err := a.requestAny(ctx, http.MethodPost, "/v1/simulate/authorize", credential.APIKey, deterministicUUID(command.IdempotencyKey), attempt, &authorized)
	if err != nil {
		return rail.Result{}, fmt.Errorf("simulate provider authorization: %w", err)
	}
	if status == http.StatusUnprocessableEntity {
		return rail.Result{Outcome: rail.OutcomeFailed, Basis: "provider_card_decline", OccurredAt: time.Now().UTC(), Detail: "provider declined card authorization"}, nil
	}
	if status != http.StatusCreated || strings.TrimSpace(authorized.Token) == "" {
		return rail.Result{}, fmt.Errorf("%w: provider authorization returned status %d without a transaction token", ErrInvalid, status)
	}
	clear := map[string]any{"amount": 0, "token": authorized.Token}
	if err := a.request(ctx, http.MethodPost, "/v1/simulate/clearing", credential.APIKey, deterministicUUID(command.IdempotencyKey+":clear"), clear, nil, http.StatusOK, http.StatusCreated); err != nil {
		return rail.Result{}, fmt.Errorf("clear provider authorization: %w", err)
	}
	now := time.Now().UTC()
	return rail.Result{Outcome: rail.OutcomeSettled, ExternalID: authorized.Token, ReceiptReference: "lithic-sandbox:" + authorized.Token, Basis: "provider_card", OccurredAt: now, Detail: "provider authorized and cleared sandbox card transaction"}, nil
}

func (a *Adapter) QueryOutcome(ctx context.Context, query rail.Query) (rail.Result, error) {
	credential, err := parseIssuedCredential(query.Credential)
	if err != nil {
		return rail.Result{}, err
	}
	if strings.TrimSpace(query.ExternalID) == "" {
		return rail.Result{}, fmt.Errorf("%w: external transaction reference is required", ErrInvalid)
	}
	var transaction transactionResponse
	if err := a.request(ctx, http.MethodGet, "/v1/transactions/"+url.PathEscape(query.ExternalID), credential.APIKey, "", nil, &transaction, http.StatusOK); err != nil {
		return rail.Result{}, err
	}
	outcome := rail.OutcomeUnknown
	switch strings.ToUpper(transaction.Status) {
	case "SETTLED":
		outcome = rail.OutcomeSettled
	case "DECLINED", "VOIDED", "REVERSED":
		outcome = rail.OutcomeFailed
	}
	return rail.Result{Outcome: outcome, ExternalID: transaction.Token, ReceiptReference: "lithic-sandbox:" + transaction.Token, Basis: "provider_card_query", OccurredAt: time.Now().UTC(), Detail: strings.ToLower(transaction.Status)}, nil
}

func parseProviderCredential(raw string) (providerCredential, error) {
	var credential providerCredential
	if err := strictJSON(raw, &credential); err != nil {
		return providerCredential{}, fmt.Errorf("%w: provider credential must be strict JSON: %v", ErrInvalid, err)
	}
	credential.APIKey = strings.TrimSpace(credential.APIKey)
	credential.AccountToken = strings.TrimSpace(credential.AccountToken)
	if credential.APIKey == "" {
		return providerCredential{}, fmt.Errorf("%w: api_key is required", ErrInvalid)
	}
	return credential, nil
}

func parseIssuedCredential(raw string) (issuedCredential, error) {
	var credential issuedCredential
	if err := strictJSON(raw, &credential); err != nil {
		return issuedCredential{}, fmt.Errorf("%w: issued credential must be strict JSON: %v", ErrInvalid, err)
	}
	if credential.APIKey == "" || credential.CardToken == "" || credential.PAN == "" || credential.RuleToken == "" || credential.MandateReference == "" || credential.AmountMinor <= 0 || credential.Currency == "" || credential.Counterparty == "" || credential.MandateExpiresAt.IsZero() {
		return issuedCredential{}, fmt.Errorf("%w: issued credential is incomplete", ErrInvalid)
	}
	return credential, nil
}

func strictJSON(raw string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("credential contains trailing data")
	}
	return nil
}

func permitsOnly(parameters ruleParameters, counterparty string) bool {
	if parameters.Action != "DECLINE" {
		return false
	}
	for _, item := range parameters.Conditions {
		if item.Attribute == "MERCHANT_ID" && item.Operation == "IS_NOT_ONE_OF" && len(item.Value) == 1 && strings.EqualFold(item.Value[0], counterparty) {
			return true
		}
	}
	return false
}

func (a *Adapter) request(ctx context.Context, method, path, apiKey, idempotencyKey string, body, target any, accepted ...int) error {
	status, err := a.requestAny(ctx, method, path, apiKey, idempotencyKey, body, target)
	if err != nil {
		return err
	}
	for _, candidate := range accepted {
		if status == candidate {
			return nil
		}
	}
	return fmt.Errorf("provider returned HTTP %d", status)
}

func (a *Adapter) requestAny(ctx context.Context, method, path, apiKey, idempotencyKey string, body, target any) (int, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reader)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", apiKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response, err := a.client.Do(request)
	if err != nil {
		return 0, err
	}
	if response == nil || response.Body == nil {
		return 0, errors.New("provider returned an empty response")
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return 0, err
	}
	if len(payload) > maxResponseBytes {
		return 0, errors.New("provider response exceeds limit")
	}
	if target != nil && len(bytes.TrimSpace(payload)) > 0 {
		if err := json.Unmarshal(payload, target); err != nil {
			return 0, fmt.Errorf("decode provider response: %w", err)
		}
	}
	return response.StatusCode, nil
}

func deterministicUUID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	bytes := sum[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x50
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	hexValue := hex.EncodeToString(bytes)
	return hexValue[0:8] + "-" + hexValue[8:12] + "-" + hexValue[12:16] + "-" + hexValue[16:20] + "-" + hexValue[20:32]
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:6])
}

func boundedMemo(mandate, counterparty string) string {
	value := "treasury:" + shortHash(mandate) + ":" + strings.TrimSpace(counterparty)
	if len(value) > 50 {
		return value[:50]
	}
	return value
}

func boundedDescriptor(counterparty string) string {
	value := strings.ToUpper(strings.TrimSpace(counterparty))
	if len(value) > 25 {
		return value[:25]
	}
	return value
}

var (
	_ card.Issuer  = (*Adapter)(nil)
	_ rail.Adapter = (*Adapter)(nil)
)
