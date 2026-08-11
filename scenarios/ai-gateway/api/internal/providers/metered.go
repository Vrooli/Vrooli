package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type accessTokenContextKey struct{}

// WithAccessToken carries the already-verified consumer bearer token from the
// Connect transport to the outbound metered provider. It is never persisted.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenContextKey{}, strings.TrimSpace(token))
}

func accessToken(ctx context.Context) string {
	token, _ := ctx.Value(accessTokenContextKey{}).(string)
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
	Role            string            `json:"role"`
	Messages        []MeteredMessage  `json:"messages"`
	ConstraintsJSON string            `json:"constraints_json,omitempty"`
	MaxTokens       int               `json:"max_tokens,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Profile         sharedv1.Profile  `json:"-"`
}

type MeteredMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MeteredResponse struct {
	ID               string `json:"id"`
	Content          string `json:"content"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	CreditsCharged   int64  `json:"credits_charged"`
	FinishReason     string `json:"finish_reason"`
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
	if strings.TrimSpace(request.Role) == "" || len(request.Messages) == 0 {
		return MeteredResponse{}, errors.New("metered inference role and messages are required")
	}
	baseURL := c.baseURL
	if profileURL := strings.TrimSpace(c.profileURLs[request.Profile]); profileURL != "" {
		baseURL = strings.TrimRight(profileURL, "/")
	}
	if baseURL == "" {
		return MeteredResponse{}, errors.New("metered inference endpoint is not configured")
	}
	token := accessToken(ctx)
	if token == "" && c.resolveToken != nil {
		resolved, err := c.resolveToken(ctx, baseURL)
		if err != nil {
			return MeteredResponse{}, fmt.Errorf("resolve metered inference access token: %w", err)
		}
		token = strings.TrimSpace(resolved)
	}
	if token == "" {
		return MeteredResponse{}, errors.New("metered inference access token is required")
	}
	if !c.allow() {
		return MeteredResponse{}, errors.New("metered inference circuit breaker is open")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return MeteredResponse{}, fmt.Errorf("marshal metered inference request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/ai/inference", strings.NewReader(string(body)))
	if err != nil {
		return MeteredResponse{}, fmt.Errorf("create metered inference request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", token)
	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		c.failed()
		return MeteredResponse{}, fmt.Errorf("metered inference request failed: %w", err)
	}
	defer response.Body.Close()
	var result MeteredResponse
	if decodeErr := json.NewDecoder(response.Body).Decode(&result); decodeErr != nil {
		c.failed()
		return MeteredResponse{}, fmt.Errorf("decode metered inference response: %w", decodeErr)
	}
	if c.egress != nil {
		_ = c.egress.Record(ctx, ProviderMetered, request.Role, request.Profile, response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		c.failed()
		return MeteredResponse{}, fmt.Errorf("metered inference returned HTTP %d", response.StatusCode)
	}
	c.succeeded()
	return result, nil
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
