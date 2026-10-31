package cliutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var installIdentityForwardingTransport sync.Once

// APIError represents a structured error from the API with rich recovery information.
type APIError struct {
	StatusCode   int
	Message      string
	Code         string
	Category     string
	Details      map[string]interface{}
	Recovery     string
	RecoveryHint string
	AutoFix      *AutoFixInfo
	ManualSteps  []string
	RawResponse  []byte
}

// AutoFixInfo describes an automatic fix command if available.
type AutoFixInfo struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Safe        bool   `json:"safe"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
}

// IsStructured returns true if the error has rich recovery information.
func (e *APIError) IsStructured() bool {
	return e.Code != "" || e.RecoveryHint != "" || e.AutoFix != nil || len(e.ManualSteps) > 0
}

// FormatConcise returns a human-readable error with recovery information.
func (e *APIError) FormatConcise() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Error: %s\n", e.Message))
	if e.Code != "" {
		b.WriteString(fmt.Sprintf("Code: %s\n", e.Code))
	}
	if e.RecoveryHint != "" {
		b.WriteString(fmt.Sprintf("\nRecovery: %s\n", e.RecoveryHint))
	}
	if e.AutoFix != nil && e.AutoFix.Command != "" {
		safe := ""
		if e.AutoFix.Safe {
			safe = " (safe)"
		}
		b.WriteString(fmt.Sprintf("\nAuto-fix%s:\n  %s\n", safe, e.AutoFix.Command))
	}
	if len(e.ManualSteps) > 0 {
		b.WriteString("\nManual steps:\n")
		for i, step := range e.ManualSteps {
			b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
		}
	}
	return b.String()
}

// ParseAPIError parses a structured API error response.
func ParseAPIError(statusCode int, data []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode, RawResponse: data}

	var parsed struct {
		Error        string                 `json:"error"`
		Message      string                 `json:"message"`
		Code         string                 `json:"code"`
		Category     string                 `json:"category"`
		Details      map[string]interface{} `json:"details"`
		Recovery     string                 `json:"recovery"`
		RecoveryHint string                 `json:"recovery_hint"`
		AutoFix      *AutoFixInfo           `json:"auto_fix"`
		ManualSteps  []string               `json:"manual_steps"`
	}

	if err := json.Unmarshal(data, &parsed); err == nil {
		apiErr.Message = parsed.Error
		if apiErr.Message == "" {
			apiErr.Message = parsed.Message
		}
		apiErr.Code = parsed.Code
		apiErr.Category = parsed.Category
		apiErr.Details = parsed.Details
		apiErr.Recovery = parsed.Recovery
		apiErr.RecoveryHint = parsed.RecoveryHint
		apiErr.AutoFix = parsed.AutoFix
		apiErr.ManualSteps = parsed.ManualSteps
	} else {
		apiErr.Message = strings.TrimSpace(string(data))
	}

	if apiErr.Message == "" {
		apiErr.Message = fmt.Sprintf("HTTP %d", statusCode)
	}
	return apiErr
}

// HTTPClient wraps an http.Client with base URL resolution, token injection,
// and JSON request helpers.
type agentCredentialPin struct {
	token          string
	operationToken string
	resolve        func(context.Context) (string, error)
	resolvePair    func(context.Context) (AgentCredentials, error)
}
type (
	agentCredentialPinKey          struct{}
	identityInjectionBypassKey     struct{}
	resolvedAgentCredentialPairKey struct{}
	resolvedAgentCredentialPair    struct {
		baselineToken  string
		operationToken string
	}
	HTTPClient struct {
		// Enabled only on an explicitly protected request-local copy.
		rejectRedirectResponses bool
		identityPin             *agentCredentialPin
		client                  *http.Client
		baseOptions             APIBaseOptions
		token                   string
		dryRun                  bool
		headerSource            func() map[string]string
		// invocationHeaderSource is owned by cliapp's per-command preflight. It
		// must remain distinct from an application header source: scenarios may
		// require their own transport contract (for example, runtime attribution)
		// in addition to common invocation provenance.
		invocationHeaderSource func() map[string]string
	}
)

type HTTPClientOptions struct {
	Client      *http.Client
	BaseOptions APIBaseOptions
	Token       string
	Timeout     time.Duration
}

func NewHTTPClient(opts HTTPClientOptions) *HTTPClient {
	installIdentityForwardingTransport.Do(func() {
		http.DefaultTransport = identityForwardingTransport{next: http.DefaultTransport}
	})
	client := opts.Client
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	} else if opts.Timeout > 0 {
		client.Timeout = opts.Timeout
	}
	return &HTTPClient{
		client:      client,
		baseOptions: opts.BaseOptions,
		token:       opts.Token,
	}
}

// identityForwardingTransport keeps verified Agent Manager identity attached to
// raw HTTP and Connect clients that use http.DefaultTransport. Scenario CLIs
// normally use HTTPClient.ApplyRequestHeaders, but durable streaming clients
// intentionally create their own no-timeout http.Client. Reading the token at
// request time makes both paths carry the same provenance without each
// scenario needing a special case.
type identityForwardingTransport struct{ next http.RoundTripper }

func (t identityForwardingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil {
		return t.roundTripper().RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	resolvedPair, err := applyAgentIdentity(clone)
	if err != nil {
		return nil, err
	}
	if resolvedPair {
		clone = clone.WithContext(context.WithValue(clone.Context(), resolvedAgentCredentialPairKey{}, resolvedAgentCredentialPair{
			baselineToken:  clone.Header.Get(HeaderAgentIdentityToken),
			operationToken: clone.Header.Get(HeaderAgentOperationToken),
		}))
	}
	return t.roundTripper().RoundTrip(clone)
}

// ApplyAgentIdentity resolves current authority for a direct transport, such
// as a WebSocket upgrade. It also strips inherited identity on remote hops.
func ApplyAgentIdentity(req *http.Request) error {
	_, err := applyAgentIdentity(req)
	return err
}

// HasResolvedAgentCredentialPair reports whether this exact trusted-local
// request carries the pair resolved by the CLI forwarding transport. Header
// bytes are checked against the private request-local record so forged or
// subsequently mutated identity cannot use the marker.
func HasResolvedAgentCredentialPair(req *http.Request) bool {
	if req == nil || !TrustedIdentityDestination(req) || req.Header == nil {
		return false
	}
	pair, ok := req.Context().Value(resolvedAgentCredentialPairKey{}).(resolvedAgentCredentialPair)
	if !ok || pair.baselineToken == "" || pair.operationToken == "" {
		return false
	}
	baselineValues := req.Header.Values(HeaderAgentIdentityToken)
	operationValues := req.Header.Values(HeaderAgentOperationToken)
	return len(baselineValues) == 1 && baselineValues[0] == pair.baselineToken &&
		len(operationValues) == 1 && operationValues[0] == pair.operationToken
}

// applyAgentIdentity returns true only when it successfully resolved and
// applied the current baseline+operation pair, including an exact pinned pair.
func applyAgentIdentity(req *http.Request) (bool, error) {
	if req != nil {
		if bypass, _ := req.Context().Value(identityInjectionBypassKey{}).(bool); bypass {
			req.Header.Del(HeaderAgentIdentityToken)
			req.Header.Del(HeaderAgentOperationToken)
			return false, nil
		}
	}
	if req != nil {
		if pin, ok := req.Context().Value(agentCredentialPinKey{}).(*agentCredentialPin); ok {
			if !TrustedIdentityDestination(req) {
				return false, errors.New("pinned native credential requires trusted destination")
			}
			current, err := pin.resolveCredentials(req.Context())
			if err != nil {
				return false, err
			}
			if current.BaselineToken != pin.token || current.OperationToken != pin.operationToken || req.Header.Get(HeaderAgentIdentityToken) != pin.token || req.Header.Get(HeaderAgentOperationToken) != pin.operationToken {
				return false, errors.New("native credential changed during exact request")
			}
			return pin.token != "" && pin.operationToken != "", nil
		}
	}
	if !TrustedIdentityDestination(req) {
		if req != nil {
			req.Header.Del(HeaderAgentIdentityToken)
			req.Header.Del(HeaderAgentOperationToken)
		}
		return false, nil
	}
	if supplied := strings.TrimSpace(req.Header.Get(HeaderAgentIdentityToken)); supplied != "" && supplied != strings.TrimSpace(os.Getenv(EnvIdentityToken)) {
		req.Header.Del(HeaderAgentOperationToken)
		return false, nil
	}
	credentials, err := ResolveAgentCredentials(req.Context())
	if err != nil {
		return false, err
	}
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	token := credentials.BaselineToken
	if token == "" {
		token = credentials.LegacyToken
	}
	if token == "" {
		req.Header.Del(HeaderAgentIdentityToken)
	} else {
		req.Header.Set(HeaderAgentIdentityToken, token)
	}
	if credentials.OperationToken == "" {
		req.Header.Del(HeaderAgentOperationToken)
	} else {
		req.Header.Set(HeaderAgentOperationToken, credentials.OperationToken)
	}
	return credentials.BaselineToken != "" && credentials.OperationToken != "", nil
}

func (pin *agentCredentialPin) resolveCredentials(ctx context.Context) (AgentCredentials, error) {
	if pin.resolvePair != nil {
		return pin.resolvePair(ctx)
	}
	if pin.resolve == nil {
		return AgentCredentials{}, errors.New("pinned native credential resolver unavailable")
	}
	token, err := pin.resolve(ctx)
	return AgentCredentials{BaselineToken: token}, err
}

func (t identityForwardingTransport) roundTripper() http.RoundTripper {
	if t.next == nil {
		return http.DefaultTransport
	}
	return t.next
}

func (h *HTTPClient) SetToken(token string) {
	h.token = token
}

// SetDryRun enables or disables dry-run mode. When enabled, requests include
// the X-Dry-Run header so APIs can skip mutations after validation.
func (h *HTTPClient) SetDryRun(enabled bool) {
	h.dryRun = enabled
}

// SetHeaderSource installs a callback invoked once per request to produce
// extra HTTP headers added to every outgoing call. Use this for headers
// whose value is computed lazily (e.g. read from environment variables
// that may be set after client construction). Returning nil or an empty
// map skips extra-header injection. Empty values in the returned map are
// dropped at request time so callers can express "skip this header" via
// an empty string without conditional wrapping.
//
// Repeated calls replace the previous source. Pass nil to clear.
func (h *HTTPClient) SetHeaderSource(fn func() map[string]string) {
	h.headerSource = fn
}

// SetInvocationHeaderSource installs the per-command provenance source used by
// cliapp. It composes with SetHeaderSource instead of replacing application
// headers that are required by a scenario API contract. Repeated calls replace
// only the prior invocation source.
func (h *HTTPClient) SetInvocationHeaderSource(fn func() map[string]string) {
	h.invocationHeaderSource = fn
}

func (h *HTTPClient) SetBaseOptions(opts APIBaseOptions) {
	h.baseOptions = opts
}

func (h *HTTPClient) BaseURL() string {
	return DetermineAPIBase(h.baseOptions)
}

// Timeout returns the configured HTTP timeout, or zero when no client exists.
func (h *HTTPClient) Timeout() time.Duration {
	if h == nil || h.client == nil {
		return 0
	}
	return h.client.Timeout
}

// CloneWithTimeout returns a copy of the client that uses timeout instead of the
// default. Every other field is carried over — base options, token, dry-run, and
// both header sources — so a long-running maintenance call keeps the same
// transport contract and provenance headers as ordinary commands. The
// underlying *http.Client is replaced rather than mutated, because it is shared
// with any other client built from the same options.
func (h *HTTPClient) CloneWithTimeout(timeout time.Duration) *HTTPClient {
	if h == nil || timeout <= 0 {
		return h
	}
	return h.cloneWithTimeout(timeout)
}

// CloneWithoutTimeout explicitly delegates the response deadline to the
// server (or request context), as required for durable blocking waits.
// Ordinary CloneWithTimeout callers retain their existing default behavior.
func (h *HTTPClient) CloneWithoutTimeout() *HTTPClient {
	if h == nil {
		return nil
	}
	return h.cloneWithTimeout(0)
}

func (h *HTTPClient) cloneWithTimeout(timeout time.Duration) *HTTPClient {
	clone := *h
	clone.client = &http.Client{Timeout: timeout}
	if h.client != nil {
		transport := *h.client
		transport.Timeout = timeout
		clone.client = &transport
	}
	return &clone
}

// Do performs an HTTP request with JSON encoding and standard error handling.
func (h *HTTPClient) Do(method, path string, query url.Values, body interface{}) ([]byte, error) {
	return h.DoWithContext(context.Background(), method, path, query, body)
}

// DoWithContext performs an HTTP request with a provided context.
func (h *HTTPClient) DoWithContext(ctx context.Context, method, path string, query url.Values, body interface{}) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := strings.TrimSpace(h.BaseURL())
	if base == "" {
		return nil, fmt.Errorf("api base URL is empty; configure an API base or set an API port")
	}
	if parsed, err := url.Parse(base); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid api base URL %q", base)
	}
	endpoint := strings.TrimRight(base, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		switch payload := body.(type) {
		case json.RawMessage:
			reader = bytes.NewReader(payload)
		case []byte:
			reader = bytes.NewReader(payload)
		case string:
			reader = strings.NewReader(payload)
		default:
			encoded, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("encode payload: %w", err)
			}
			reader = bytes.NewReader(encoded)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	h.ApplyRequestHeaders(req)
	if h.identityPin != nil {
		// Pin after all application/invocation headers and retain the marker
		// through the ordinary identity-forwarding transport. Recheck the live
		// authority; a responding refusal never falls back to the retained leaf.
		req = req.WithContext(context.WithValue(req.Context(), agentCredentialPinKey{}, h.identityPin))
		req.Header.Del("Authorization")
		req.Header.Set(HeaderAgentIdentityToken, h.identityPin.token)
		if h.identityPin.operationToken == "" {
			req.Header.Del(HeaderAgentOperationToken)
		} else {
			req.Header.Set(HeaderAgentOperationToken, h.identityPin.operationToken)
		}
		if err := ApplyAgentIdentity(req); err != nil {
			return nil, err
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if h.rejectRedirectResponses && resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: "protected request redirect response refused"}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, ParseAPIError(resp.StatusCode, data)
	}
	return data, nil
}

// ApplyRequestHeaders applies the configured authentication, dry-run, and
// custom headers to req. It lets non-JSON transports reuse the same request
// decoration as DoWithContext without inheriting JSON body handling.
func (h *HTTPClient) ApplyRequestHeaders(req *http.Request) {
	if h == nil || req == nil {
		return
	}
	if h.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.token))
	}
	if h.dryRun {
		req.Header.Set("X-Dry-Run", "true")
	}
	applyHeaderSource(req, h.headerSource)
	applyHeaderSource(req, h.invocationHeaderSource)
}

func applyHeaderSource(req *http.Request, source func() map[string]string) {
	if source == nil {
		return
	}
	for k, v := range source() {
		if k == HeaderAgentIdentityToken && !TrustedIdentityDestination(req) {
			continue
		}
		if v == "" {
			continue
		}
		req.Header.Set(k, v)
	}
}

// ExtractErrorMessage pulls a human-readable error string from a JSON error
// response; falls back to the raw body when parsing fails.
func ExtractErrorMessage(data []byte) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err == nil {
		if errObj, ok := parsed["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok {
				return msg
			}
		}
	}
	return strings.TrimSpace(string(data))
}

// TrustedIdentityDestination bounds ambient identity forwarding to the local
// scenario transport. Remote hops require an explicit owner-configured client.
func TrustedIdentityDestination(req *http.Request) bool {
	if req == nil || req.URL == nil || (req.URL.Scheme != "http" && req.URL.Scheme != "https" && req.URL.Scheme != "ws" && req.URL.Scheme != "wss") {
		return false
	}
	host := req.URL.Hostname()
	ip := net.ParseIP(host)
	return host == "localhost" || (ip != nil && ip.IsLoopback())
}
