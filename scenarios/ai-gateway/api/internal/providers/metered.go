package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/lpbsidentity"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

var ErrInsufficientCredits = errors.New("insufficient credits")

const CodeOutcomeUncertain = "outcome_uncertain"

const CreditCallerAuthorizationHeader = lpbsidentity.CreditCallerAuthorizationHeader

type (
	accessTokenContextKey               struct{}
	creditCallerAuthorizationContextKey struct{}
)

// WithAccessToken carries the already-verified consumer bearer token from the
// Connect transport to the outbound metered provider. It is never persisted.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenContextKey{}, strings.TrimSpace(token))
}

func accessToken(ctx context.Context) string {
	token, _ := ctx.Value(accessTokenContextKey{}).(string)
	return strings.TrimSpace(token)
}

// WithCreditCallerAuthorization carries the invoking scenario's separately
// signed service-principal bearer to LPBS. It never replaces the user's
// Authorization token and is only useful when LPBS verifies it independently.
func WithCreditCallerAuthorization(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, creditCallerAuthorizationContextKey{}, strings.TrimSpace(token))
}

func creditCallerAuthorization(ctx context.Context) string {
	token, _ := ctx.Value(creditCallerAuthorizationContextKey{}).(string)
	return strings.TrimSpace(token)
}

type MeteredEgressRecorder interface {
	Record(context.Context, string, string, sharedv1.Profile, int) error
}

type MeteredClientOptions struct {
	BaseURL      string
	ProfileURLs  map[sharedv1.Profile]string
	HTTPClient   *http.Client
	Egress       MeteredEgressRecorder
	FailureLimit int
	Cooldown     time.Duration
	// ResolveAccessToken is used only when the caller did not forward a
	// consumer access token. Implementations must keep refresh credentials in
	// the credential authority and return only a short-lived bearer value.
	// The resolver receives the selected LPBS base URL so issuer switching
	// cannot reuse a token minted by another authority.
	ResolveAccessToken func(context.Context, string) (string, error)
}

type MeteredClient struct {
	baseURL      string
	profileURLs  map[sharedv1.Profile]string
	httpClient   *http.Client
	egress       MeteredEgressRecorder
	failureLimit int
	cooldown     time.Duration
	resolveToken func(context.Context, string) (string, error)
	mu           sync.Mutex
	failures     int
	openedAt     time.Time
}

type MeteredRequest struct {
	Role            string                  `json:"role"`
	RootOperationID string                  `json:"-"`
	Messages        []MeteredMessage        `json:"messages"`
	ConstraintsJSON string                  `json:"constraints_json,omitempty"`
	MaxTokens       int                     `json:"max_tokens,omitempty"`
	Metadata        map[string]string       `json:"metadata,omitempty"`
	Profile         sharedv1.Profile        `json:"-"`
	SpendLimit      *inferencev1.SpendLimit `json:"spend_limit,omitempty"`
	Images          []MeteredImage          `json:"images,omitempty"`
	BrowserSession  *MeteredBrowserSession  `json:"browser_session,omitempty"`
}

type MeteredBrowserSession struct {
	ReservationID string `json:"reservation_id,omitempty"`
	RunID         string `json:"run_id"`
	SessionID     string `json:"session_id"`
	GrantReceipt  string `json:"grant_receipt"`
}

type MeteredImage struct {
	Data      string `json:"data"`
	MediaType string `json:"media_type"`
	SHA256    string `json:"sha256"`
}

type MeteredMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MeteredResponse struct {
	ID                 string                          `json:"id"`
	RootOperationID    string                          `json:"root_operation_id,omitempty"`
	BillingOperationID string                          `json:"billing_operation_id,omitempty"`
	Content            string                          `json:"content"`
	PromptTokens       int64                           `json:"prompt_tokens"`
	CompletionTokens   int64                           `json:"completion_tokens"`
	CreditsCharged     int64                           `json:"credits_charged"`
	FinishReason       string                          `json:"finish_reason"`
	SpendAuthorization *inferencev1.SpendAuthorization `json:"spend_authorization,omitempty"`
	AccountingStatus   string                          `json:"accounting_status,omitempty"`
	ActualCostMicros   *int64                          `json:"actual_cost_micros,omitempty"`
}

func NewMeteredClient(opts MeteredClientOptions) *MeteredClient {
	limit := opts.FailureLimit
	if limit <= 0 {
		limit = 3
	}
	cooldown := opts.Cooldown
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	return &MeteredClient{
		baseURL:      strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		profileURLs:  opts.ProfileURLs,
		httpClient:   client,
		egress:       opts.Egress,
		failureLimit: limit,
		cooldown:     cooldown,
		resolveToken: opts.ResolveAccessToken,
	}
}

func (c *MeteredClient) Run(ctx context.Context, request MeteredRequest) (MeteredResponse, error) {
	if err := validateMeteredRequest(&request); err != nil {
		return MeteredResponse{}, err
	}
	baseURL, token, err := c.endpointAndToken(ctx, request)
	if err != nil {
		return MeteredResponse{}, err
	}
	if !c.allow() {
		return MeteredResponse{}, errors.New("metered inference circuit breaker is open")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return MeteredResponse{}, fmt.Errorf("marshal metered inference request: %w", err)
	}
	httpRequest, err := c.newInferenceRequest(ctx, baseURL, token, body)
	if err != nil {
		return MeteredResponse{}, err
	}
	if caller := creditCallerAuthorization(ctx); caller != "" {
		httpRequest.Header.Set(CreditCallerAuthorizationHeader, "Bearer "+caller)
	}
	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		c.failed()
		return MeteredResponse{}, &CommandError{Code: CodeOutcomeUncertain, Command: ProviderMetered, ExitCode: -1, Err: fmt.Errorf("metered inference outcome is uncertain for root_operation_id %q: %w", request.RootOperationID, err)}
	}
	return c.handleResponse(ctx, request, response)
}

func validateMeteredRequest(request *MeteredRequest) error {
	if strings.TrimSpace(request.Role) == "" || len(request.Messages) == 0 {
		return errors.New("metered inference role and messages are required")
	}
	if err := validateMeteredBrowserRequest(*request); err != nil {
		return err
	}
	return addMeteredOperationMetadata(request)
}

func validateMeteredBrowserRequest(request MeteredRequest) error {
	if !strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") {
		return nil
	}
	if !validMeteredRootOperationID(request.RootOperationID) {
		return &CommandError{Code: CodeUnconfigured, Command: ProviderMetered, ExitCode: -1, Err: errors.New("browser metered inference requires a caller-generated root_operation_id")}
	}
	if err := validateMeteredBrowserBinding(request.BrowserSession); err != nil {
		return err
	}
	return nil
}

func validateMeteredBrowserBinding(binding *MeteredBrowserSession) error {
	if binding == nil || strings.TrimSpace(binding.RunID) == "" || strings.TrimSpace(binding.SessionID) == "" || strings.TrimSpace(binding.GrantReceipt) == "" || len(binding.RunID) > 64 || len(binding.SessionID) > 128 || len(binding.GrantReceipt) > 512 || strings.ContainsAny(binding.RunID+binding.SessionID+binding.GrantReceipt, "\r\n") {
		return &CommandError{Code: "invalid_request", Command: ProviderMetered, ExitCode: -1, Err: errors.New("browser metered inference requires its authenticated run, grant and BAS session binding")}
	}
	return nil
}

func addMeteredOperationMetadata(request *MeteredRequest) error {
	if request.RootOperationID != "" {
		if request.Metadata == nil {
			request.Metadata = make(map[string]string)
		}
		if existing := strings.TrimSpace(request.Metadata["gateway_root_operation_id"]); existing != "" && existing != request.RootOperationID {
			return &CommandError{Code: "invalid_request", Command: ProviderMetered, ExitCode: -1, Err: errors.New("metered metadata operation id conflicts with the Gateway root operation id")}
		}
		request.Metadata["gateway_root_operation_id"] = request.RootOperationID
	}
	return nil
}

func (c *MeteredClient) endpointAndToken(ctx context.Context, request MeteredRequest) (string, string, error) {
	baseURL := c.baseURL
	if profileURL := strings.TrimSpace(c.profileURLs[request.Profile]); profileURL != "" {
		baseURL = strings.TrimRight(profileURL, "/")
	}
	if baseURL == "" {
		return "", "", errors.New("metered inference endpoint is not configured")
	}
	token := accessToken(ctx)
	if token == "" && c.resolveToken != nil {
		resolved, err := c.resolveToken(ctx, baseURL)
		if err != nil {
			return "", "", fmt.Errorf("resolve metered inference access token: %w", err)
		}
		token = strings.TrimSpace(resolved)
	}
	if token == "" {
		return "", "", errors.New("metered inference access token is required")
	}
	return baseURL, token, nil
}

func (c *MeteredClient) newInferenceRequest(ctx context.Context, baseURL, token string, body []byte) (*http.Request, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/ai/inference", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create metered inference request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", token)
	return httpRequest, nil
}

func (c *MeteredClient) handleResponse(ctx context.Context, request MeteredRequest, response *http.Response) (MeteredResponse, error) {
	defer response.Body.Close()
	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		c.failed()
		return MeteredResponse{}, &CommandError{Code: CodeOutcomeUncertain, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, Err: fmt.Errorf("metered inference outcome is uncertain for root_operation_id %q: read response: %w", request.RootOperationID, readErr)}
	}
	if c.egress != nil {
		_ = c.egress.Record(ctx, ProviderMetered, request.Role, request.Profile, response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return MeteredResponse{}, c.httpFailure(request, response, bodyBytes)
	}
	var result MeteredResponse
	if decodeErr := json.Unmarshal(bodyBytes, &result); decodeErr != nil {
		c.failed()
		return MeteredResponse{}, &CommandError{Code: CodeOutcomeUncertain, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, Err: fmt.Errorf("metered inference outcome is uncertain for root_operation_id %q: decode response: %w", request.RootOperationID, decodeErr)}
	}
	if request.RootOperationID != "" && result.RootOperationID != request.RootOperationID {
		c.failed()
		return MeteredResponse{}, &CommandError{Code: CodeOutcomeUncertain, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, Err: fmt.Errorf("metered inference outcome is uncertain for root_operation_id %q: response operation identity did not match", request.RootOperationID)}
	}
	if strings.HasPrefix(strings.TrimSpace(request.Role), "browser.") && strings.TrimSpace(result.BillingOperationID) == "" {
		c.failed()
		return MeteredResponse{}, &CommandError{Code: CodeOutcomeUncertain, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, Err: errors.New("metered response omitted its LPBS session reservation identity")}
	}
	c.succeeded()
	return result, nil
}

func (c *MeteredClient) httpFailure(request MeteredRequest, response *http.Response, body []byte) error {
	retryAfter := strings.TrimSpace(response.Header.Get("Retry-After"))
	detail := strings.TrimSpace(string(body))
	if response.StatusCode == http.StatusPaymentRequired {
		return &CommandError{
			Code: CodeInsufficientCredits, Command: ProviderMetered, ExitCode: -1,
			HTTPStatus: response.StatusCode, RetryAfter: retryAfter, Err: fmt.Errorf("%w: %s", ErrInsufficientCredits, detail),
		}
	}
	if response.StatusCode >= http.StatusInternalServerError && request.RootOperationID != "" {
		c.failed()
		return &CommandError{
			Code: CodeOutcomeUncertain, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, RetryAfter: retryAfter,
			Err: fmt.Errorf("metered inference outcome is uncertain for root_operation_id %q: HTTP %d: %s", request.RootOperationID, response.StatusCode, detail),
		}
	}
	c.failed()
	if response.StatusCode == http.StatusTooManyRequests {
		return &CommandError{
			Code: CodeRateLimited, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, RetryAfter: retryAfter,
			Err: fmt.Errorf("metered inference rate limited: HTTP %d: %s", response.StatusCode, detail),
		}
	}
	if response.StatusCode == http.StatusServiceUnavailable || response.StatusCode == 502 || response.StatusCode == 504 {
		return &CommandError{
			Code: CodeProviderOverloaded, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, RetryAfter: retryAfter,
			Err: fmt.Errorf("metered inference provider overloaded: HTTP %d: %s", response.StatusCode, detail),
		}
	}
	return &CommandError{
		Code: CodeProviderFailed, Command: ProviderMetered, ExitCode: -1, HTTPStatus: response.StatusCode, RetryAfter: retryAfter,
		Err: fmt.Errorf("metered inference returned HTTP %d: %s", response.StatusCode, detail),
	}
}

func validMeteredRootOperationID(value string) bool {
	return len(value) >= 32 && len(value) <= 128 && strings.TrimSpace(value) == value && !strings.ContainsAny(value, " \t\r\n")
}

func (c *MeteredClient) allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.openedAt.IsZero() || time.Since(c.openedAt) >= c.cooldown {
		return true
	}
	return false
}

func (c *MeteredClient) failed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	if c.failures >= c.failureLimit {
		c.openedAt = time.Now()
	}
}

func (c *MeteredClient) succeeded() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.openedAt = time.Time{}
}
