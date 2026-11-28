package duplex

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
	"sync"
	"time"

	"audio-tools/internal/httpc"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
)

const (
	accountCreditLifecycleBasePath = "/api/v1/account-credits/reservations" // #nosec G101 -- route name is API vocabulary, not credential material.
	accountCreditUSDV1ReservePath  = accountCreditLifecycleBasePath + "/usd-v1"
)

var ErrHumanBearerRequired = errors.New("duplex: verified human bearer required for LPBS account credits")

type AccountCreditTokenSource interface {
	Token(context.Context) (string, error)
}

type AccountCreditCredentialSource interface {
	Resolve(context.Context, string, string) (string, error)
}

// NewAccountCreditServiceTokens consumes the operator-issued Audio Tools
// principal from credential authority and exchanges it for short-lived tokens.
// It never mints or writes credentials. The operator provisions the principal
// declared in .vrooli/service-principals.json through scenario-authenticator.
func NewAccountCreditServiceTokens(credentials AccountCreditCredentialSource, authenticatorURL string, client *http.Client, options ...servicetoken.Option) (*servicetoken.Tokens, error) {
	if credentials == nil || strings.TrimSpace(authenticatorURL) == "" {
		return nil, errors.New("duplex: account-credit credential source and authenticator URL are required")
	}
	return NewAccountCreditServiceTokensWithResolver(credentials, func(context.Context) (string, error) {
		return strings.TrimRight(authenticatorURL, "/"), nil
	}, client, options...)
}

// NewAccountCreditServiceTokensWithResolver defers authenticator discovery
// until token exchange, so temporary startup ordering and port changes do not
// require a process restart.
func NewAccountCreditServiceTokensWithResolver(credentials AccountCreditCredentialSource, authenticatorURL func(context.Context) (string, error), client *http.Client, options ...servicetoken.Option) (*servicetoken.Tokens, error) {
	if credentials == nil || authenticatorURL == nil {
		return nil, errors.New("duplex: account-credit credential source and authenticator resolver are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	clientCopy := *client
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	identity := lpbsidentity.AccountCreditLifecycle.CredentialIdentity("audio-tools")
	role := lpbsidentity.AccountCreditLifecycle
	exchange := servicetoken.AuthenticatorExchanger(&clientCopy, authenticatorURL)
	return servicetoken.New(servicetoken.Config{
		Name: "audio-tools LPBS account-credit",
		Credential: func(ctx context.Context) (string, error) {
			return credentials.Resolve(ctx, identity, "credential")
		},
		Expect:        servicetoken.ExpectIdentity(role.Principal("audio-tools"), role.Audience, role.Scope),
		RefreshMargin: time.Minute,
		Describe:      servicetoken.DescribeWithCause("audio-tools LPBS account-credit"),
	}, append([]servicetoken.Option{servicetoken.WithExchanger(exchange)}, options...)...), nil
}

// WithVerifiedHumanBearer carries only the exact incoming bearer when the
// middleware has verified it as a human consumer access token. It does not
// accept cookies, agent identities, service tokens, or arbitrary strings.
func WithVerifiedHumanBearer(ctx context.Context, authorization string) (context.Context, error) {
	if _, err := verifiedHumanOwnerKey(ctx); err != nil {
		return nil, ErrHumanBearerRequired
	}
	value, found := strings.CutPrefix(strings.TrimSpace(authorization), "Bearer ")
	value = strings.TrimSpace(value)
	if !found || value == "" || strings.ContainsAny(value, " \t\r\n") {
		return nil, ErrHumanBearerRequired
	}
	return context.WithValue(ctx, humanBearerContextKey{}, value), nil
}

// WithVerifiedHumanRequest carries the exact LPBS access token from the
// authenticated request, whether it arrived as a Bearer header or supported
// browser cookie. Call this only after authn middleware has populated ctx.
func WithVerifiedHumanRequest(ctx context.Context, req *http.Request) (context.Context, error) {
	if _, err := verifiedHumanOwnerKey(ctx); err != nil || req == nil {
		return nil, ErrHumanBearerRequired
	}
	token := consumeridentity.RequestAccessToken(req)
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, ErrHumanBearerRequired
	}
	return context.WithValue(ctx, humanBearerContextKey{}, token), nil
}

// verifiedHumanOwnerKey derives a stable, non-reversible journal partition
// from the authenticated subject. It never persists the user's bearer token.
func verifiedHumanOwnerKey(ctx context.Context) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.Verified || principal.Kind != identity.ActorHuman || principal.Source != identity.SourceConsumerAccessToken || strings.TrimSpace(principal.Subject) == "" {
		return "", ErrHumanBearerRequired
	}
	digest := sha256.Sum256([]byte("audio-tools duplex owner v1\n" + strings.TrimSpace(principal.Subject)))
	return hex.EncodeToString(digest[:]), nil
}

type humanBearerContextKey struct{}

func humanBearer(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(humanBearerContextKey{}).(string)
	return strings.TrimSpace(value), ok && strings.TrimSpace(value) != ""
}

type AccountCreditClient struct {
	baseURL string
	tokens  AccountCreditTokenSource
	doer    httpc.Doer
	journal Journal
}

// WithJournal enables crash-recovery correlation for every durable LPBS hold.
// The journal contains no user bearer or other reusable credential.
func (c *AccountCreditClient) WithJournal(j Journal) *AccountCreditClient {
	if c != nil {
		c.journal = j
	}
	return c
}

func NewAccountCreditClient(baseURL string, tokens AccountCreditTokenSource, doer httpc.Doer) (*AccountCreditClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || tokens == nil {
		return nil, errors.New("duplex: valid LPBS base URL and account-credit token source are required")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("duplex: LPBS base URL must not contain user info, query, or fragment")
	}
	if doer == nil {
		doer = &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &AccountCreditClient{baseURL: strings.TrimRight(parsed.String(), "/"), tokens: tokens, doer: doer}, nil
}

type lifecycleReservationResponse struct {
	ReservationID   string    `json:"reservation_id"`
	Created         bool      `json:"created"`
	DispatchAllowed bool      `json:"dispatch_authorized"`
	Phase           string    `json:"phase"`
	LeaseExpiresAt  time.Time `json:"lease_expires_at"`
	ContractVersion int       `json:"contract_version"`
	BudgetMicros    int64     `json:"budget_micros"`
	ReservedAmount  int64     `json:"reserved_amount"`
	CostMultiplier  int64     `json:"cost_multiplier"`
}

type lifecycleRecoveryResponse struct {
	ReservationID     string `json:"reservation_id"`
	Phase             string `json:"phase"`
	ProviderReceiptID string `json:"provider_receipt_id"`
}

// ReconcileJournal queries LPBS for every locally pending operation using the
// current verified human bearer. It never guesses usage: only owner-confirmed
// settled/released/expired phases close local recovery records.
func (c *AccountCreditClient) ReconcileJournal(ctx context.Context, journal RecoveryJournal) error {
	if ctx == nil || journal == nil {
		return errors.New("duplex: recovery context and journal are required")
	}
	if _, ok := humanBearer(ctx); !ok {
		return ErrHumanBearerRequired
	}
	ownerKey, err := verifiedHumanOwnerKey(ctx)
	if err != nil {
		return err
	}
	operations, err := journal.Recoverable(ctx, ownerKey)
	if err != nil {
		return fmt.Errorf("list duplex recovery operations: %w", err)
	}
	for _, op := range operations {
		var response lifecycleRecoveryResponse
		path := accountCreditLifecycleBasePath + "/" + op.OperationID + "/recovery"
		if err := c.call(ctx, http.MethodGet, path, nil, &response); err != nil {
			return fmt.Errorf("recover duplex operation %s: %w", op.OperationID, err)
		}
		if response.ReservationID != op.OperationID {
			return errors.New("duplex: LPBS recovery returned a mismatched operation")
		}
		switch response.Phase {
		case "settled", "released", "expired":
			if err := journal.MarkRecovered(ctx, op, response.Phase, response.ProviderReceiptID); err != nil {
				return err
			}
		case "reserved", "dispatched", "settling", "uncertain":
			// Still owned by LPBS. Keep the durable local record pending. In
			// particular, uncertain is not interchangeable with zero-cost.
		default:
			return errors.New("duplex: LPBS recovery returned an unknown lifecycle phase")
		}
	}
	return nil
}

type lifecycleSettlement struct {
	CostMicros        int64  `json:"cost_micros"`
	ActiveSeconds     int64  `json:"active_seconds"`
	PausedSeconds     int64  `json:"paused_seconds"`
	PromptTokens      int    `json:"prompt_tokens"`
	CompletionTokens  int    `json:"completion_tokens"`
	Model             string `json:"model"`
	Provider          string `json:"provider"`
	ProviderReceiptID string `json:"provider_receipt_id"`
}

// AccountCreditReservation is backed by LPBS's durable account-credit
// lifecycle. USD budgets and measured provider costs stay in micros; LPBS
// resolves and snapshots the authoritative tier/app conversion policy.
type AccountCreditReservation struct {
	client       *AccountCreditClient
	operationID  string
	budgetMicros int64
	model        string
	provider     string
	userBearer   string
	ownerKey     string
	journal      Journal
	journalPhase string
	mu           sync.Mutex
	dispatched   bool
	finalized    bool
}

// AccountCreditAdmission composes the existing privacy/catalog/cap guard with
// LPBS's durable wallet. LPBS owns USD-to-credit conversion and policy snapshots.
type AccountCreditAdmission struct {
	Policy *AdmissionPolicy
	Client *AccountCreditClient
}

func (a *AccountCreditAdmission) Authorize(ctx context.Context, request Request) error {
	if a == nil || a.Policy == nil {
		return ErrNilAdmission
	}
	return a.Policy.Authorize(ctx, request)
}

func (a *AccountCreditAdmission) Reserve(ctx context.Context, request Request) (Reservation, error) {
	if a == nil || a.Policy == nil || a.Client == nil {
		return nil, ErrNilAdmission
	}
	if err := a.Policy.Authorize(ctx, request); err != nil {
		return nil, err
	}
	return a.Client.Reserve(ctx, request.BudgetMicros, request.Model, request.ProviderID)
}

func (c *AccountCreditClient) Reserve(ctx context.Context, budgetMicros uint64, model, provider string) (*AccountCreditReservation, error) {
	request, err := newAccountCreditReserveRequest(ctx, budgetMicros, model, provider)
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	reservation := request.reservation(c, id.String())
	if err := reservation.createJournalIntent(ctx); err != nil {
		return nil, err
	}
	if err := c.createAccountCreditHold(ctx, id.String(), request.budgetMicros); err != nil {
		return nil, err
	}
	if err := reservation.journalTransition(ctx, "reserved", ProviderUsage{}); err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *AccountCreditReservation) BeginDispatch(ctx context.Context) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finalized || r.dispatched {
		return false, nil
	}
	ctx, err := r.ownerContext(ctx)
	if err != nil {
		return false, err
	}
	// Mark the hold dispatch-attempted before the network call. A lost response
	// cannot prove whether LPBS authorized dispatch, so every follow-up must
	// retain/recover the hold rather than attempting release.
	r.dispatched = true
	if err := r.journalTransition(ctx, "dispatch_pending", ProviderUsage{}); err != nil {
		return false, err
	}
	var response lifecycleReservationResponse
	if err := r.client.call(ctx, http.MethodPost, accountCreditLifecycleBasePath+"/"+r.operationID+"/dispatch", map[string]any{}, &response); err != nil {
		_ = r.journalTransition(context.WithoutCancel(ctx), "uncertain", ProviderUsage{})
		return false, err
	}
	if !response.DispatchAllowed || response.Phase != "dispatched" {
		_ = r.journalTransition(ctx, "uncertain", ProviderUsage{})
		return false, nil
	}
	if err := r.journalTransition(ctx, "dispatched", ProviderUsage{}); err != nil {
		return false, err
	}
	return true, nil
}

func (r *AccountCreditReservation) SettleProviderUsage(ctx context.Context, usage ProviderUsage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.validateSettlementEligibility(usage); err != nil {
		return err
	}
	ctx, err := r.ownerContext(ctx)
	if err != nil {
		return err
	}
	if err := r.validateSettlementAmounts(usage); err != nil {
		return err
	}
	if err := r.journalTransition(ctx, "settlement_pending", usage); err != nil {
		return err
	}
	var response lifecycleReservationResponse
	// #nosec G115 -- validateSettlementAmounts proves each uint64 fits int64 before these conversions.
	err = r.client.call(ctx, http.MethodPost, accountCreditLifecycleBasePath+"/"+r.operationID+"/settle-usd-v1", lifecycleSettlement{
		CostMicros: int64(usage.CostMicros), ActiveSeconds: int64(usage.ActiveSeconds), PausedSeconds: int64(usage.PausedSeconds), Model: r.model, Provider: r.provider,
		ProviderReceiptID: strings.TrimSpace(usage.ProviderReceiptID),
	}, &response)
	if err == nil && response.ReservationID == r.operationID && response.Phase == "settled" {
		if transitionErr := r.journalTransition(ctx, "settled", usage); transitionErr != nil {
			return transitionErr
		}
		r.finalized = true
		r.userBearer = ""
	} else if err == nil {
		err = errors.New("duplex: LPBS did not confirm account-credit settlement")
	}
	return err
}

func (r *AccountCreditReservation) validateSettlementEligibility(usage ProviderUsage) error {
	if !r.dispatched || r.finalized || strings.TrimSpace(usage.ProviderReceiptID) == "" || strings.TrimSpace(usage.Model) != r.model || strings.TrimSpace(usage.Provider) != r.provider {
		return errors.New("duplex: measured terminal usage and dispatched hold are required for settlement")
	}
	return nil
}

func (r *AccountCreditReservation) validateSettlementAmounts(usage ProviderUsage) error {
	maxInt64 := uint64(^uint64(0) >> 1)
	if usage.CostMicros > maxInt64 || usage.ActiveSeconds > maxInt64 || usage.PausedSeconds > maxInt64 {
		return errors.New("duplex: measured provider cost overflows LPBS settlement")
	}
	if int64(usage.CostMicros) > r.budgetMicros {
		return errors.New("duplex: measured provider cost exceeds the reserved USD budget")
	}
	return nil
}

func (r *AccountCreditReservation) RetainUncertain(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.dispatched || r.finalized {
		return errors.New("duplex: only a dispatched nonterminal account-credit hold can be retained as uncertain")
	}
	ctx, err := r.ownerContext(ctx)
	if err != nil {
		return err
	}
	var response lifecycleReservationResponse
	if err := r.client.call(ctx, http.MethodPost, accountCreditLifecycleBasePath+"/"+r.operationID+"/retain-uncertain", map[string]any{}, &response); err != nil {
		return err
	}
	if response.ReservationID != r.operationID || response.Phase != "uncertain" {
		return errors.New("duplex: LPBS did not confirm the uncertain account-credit state")
	}
	if r.journalPhase != "" && r.journalPhase != "uncertain" {
		if err := r.journalTransition(ctx, "uncertain", ProviderUsage{}); err != nil {
			return err
		}
	}
	r.finalized = true
	r.userBearer = ""
	return nil
}

// Release is intentionally available only before dispatch authorization.
// Once dispatch might have reached LPBS, the caller must use RetainUncertain.
func (r *AccountCreditReservation) ReleaseBeforeDispatch(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.dispatched || r.finalized {
		return errors.New("duplex: dispatched or finalized account-credit hold cannot be released")
	}
	ctx, err := r.ownerContext(ctx)
	if err != nil {
		return err
	}
	var response lifecycleReservationResponse
	if err := r.client.call(ctx, http.MethodPost, accountCreditLifecycleBasePath+"/"+r.operationID+"/release", map[string]any{}, &response); err != nil {
		return err
	}
	if response.ReservationID != r.operationID || response.Phase != "released" {
		return errors.New("duplex: LPBS did not confirm account-credit release")
	}
	if r.journalPhase != "" {
		if err := r.journalTransition(ctx, "released", ProviderUsage{}); err != nil {
			return err
		}
	}
	r.finalized = true
	r.userBearer = ""
	return nil
}

func (r *AccountCreditReservation) journalTransition(ctx context.Context, next string, usage ProviderUsage) error {
	if r.journal == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("duplex: nil account-credit journal context")
	}
	if err := r.journal.Transition(ctx, r.operationID, r.journalPhase, next, usage); err != nil {
		return err
	}
	r.journalPhase = next
	return nil
}

func (r *AccountCreditReservation) Settle(actualMicros uint64) error {
	return errors.New("duplex: account-credit settlement requires provider receipt context")
}

func (r *AccountCreditReservation) Release() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = r.ReleaseBeforeDispatch(ctx)
}

// ownerContext always uses the human bearer captured when this exact hold was
// created. A caller context may be canceled or unrelated to the hold owner.
func (r *AccountCreditReservation) ownerContext(ctx context.Context) (context.Context, error) {
	if ctx == nil {
		return nil, errors.New("duplex: nil account-credit request context")
	}
	if r.userBearer == "" {
		return nil, ErrHumanBearerRequired
	}
	return context.WithValue(ctx, humanBearerContextKey{}, r.userBearer), nil
}

func (c *AccountCreditClient) call(ctx context.Context, method, path string, body, out any) error {
	if ctx == nil {
		return errors.New("duplex: nil account-credit request context")
	}
	userToken, ok := humanBearer(ctx)
	if !ok {
		return ErrHumanBearerRequired
	}
	serviceToken, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(serviceToken) == "" {
		return errors.New("duplex: LPBS account-credit service identity unavailable")
	}
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(encoded)
	}
	// #nosec G704 -- baseURL is operator-configured and validated at construction; callers pass fixed lifecycle paths, not request URLs.
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+userToken)
	req.Header.Set(lpbsidentity.AccountCreditLifecycleAuthorizationHeader, "Bearer "+strings.TrimSpace(serviceToken))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := c.doer.Do(req)
	if err != nil {
		return errors.New("duplex: LPBS account-credit request failed")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return errors.New("duplex: LPBS account-credit response could not be read")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("duplex: LPBS account-credit request refused (HTTP %d)", response.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return errors.New("duplex: invalid LPBS account-credit response")
		}
	}
	return nil
}
