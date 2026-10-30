// Package upstream is the one System One wire client this resource owns. It
// talks to the local Ollaya daemon (native /api/decide) and to hosted System
// One endpoints (TypeSafe, Eden AI, Cloudflare), and normalizes every answer
// and every failure into the shape ai-gateway's provider adapter reads.
package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
)

// ProviderErrorMarker is the stable line protocol the ai-gateway command
// runner parses from stderr. Keep in sync with
// scenarios/ai-gateway/api/internal/providers.ProviderErrorMarker.
const ProviderErrorMarker = "VROOLI_PROVIDER_ERROR "

// Typed failure codes understood by ai-gateway's runner.
const (
	CodeUnconfigured       = "unconfigured"
	CodeUnreachable        = "unreachable"
	CodeRateLimited        = "rate_limited"
	CodeProviderOverloaded = "provider_overloaded"
	CodeProviderFailed     = "provider_failed"
	CodeContextOverflow    = "context_overflow"
)

const defaultLocalURL = "http://127.0.0.1:11435"

// maxResponseBytes bounds an upstream body. Decision answers are small; a
// larger body is a misrouted endpoint, not a decision.
const maxResponseBytes = 4 << 20

// ProviderError is a classified upstream failure.
type ProviderError struct {
	Code       string
	HTTPStatus int
	RetryAfter string
	Message    string
}

func (e *ProviderError) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("%s (HTTP %d): %s", e.Code, e.HTTPStatus, e.Message)
	}
	return e.Code + ": " + e.Message
}

// MarkerLine renders the stderr line that carries the typed failure across the
// subprocess boundary.
func (e *ProviderError) MarkerLine() string {
	raw, _ := json.Marshal(struct {
		Code       string `json:"code"`
		HTTPStatus int    `json:"http_status,omitempty"`
		RetryAfter string `json:"retry_after,omitempty"`
		Message    string `json:"message,omitempty"`
	}{e.Code, e.HTTPStatus, e.RetryAfter, e.Message})
	return ProviderErrorMarker + string(raw)
}

// Request is the caller's System One body: state plus ordered questions. Both
// stay raw so the caller's question order reaches the upstream unchanged.
type Request struct {
	State     json.RawMessage `json:"state"`
	Questions json.RawMessage `json:"questions"`
}

// Result is the normalized answer the gateway adapter decodes.
type Result struct {
	Upstream  string          `json:"upstream"`
	Locality  string          `json:"locality"`
	Model     string          `json:"model"`
	Answers   json.RawMessage `json:"answers"`
	Usage     Usage           `json:"usage"`
	CostUSD   *float64        `json:"cost_usd"`
	LatencyMS int64           `json:"latency_ms"`
	// EvalMS and LoadMS are the daemon's own timings; nil for hosted upstreams,
	// which report none.
	EvalMS *int64 `json:"eval_ms"`
	LoadMS *int64 `json:"load_ms"`
}

type Usage struct {
	InputTokens  *int64 `json:"input_tokens"`
	OutputTokens *int64 `json:"output_tokens"`
}

type Client struct {
	HTTP   *http.Client
	GetEnv func(string) string
	Now    func() time.Time
}

// LocalURL resolves the daemon base URL from the resource's exported
// environment.
func LocalURL(getenv func(string) string) string {
	if raw := strings.TrimSpace(getenv("OLLAYA_URL")); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	if port := strings.TrimSpace(getenv("OLLAYA_PORT")); port != "" {
		return "http://127.0.0.1:" + port
	}
	return defaultLocalURL
}

// Decide answers one request on the resolved upstream.
func (c Client) Decide(ctx context.Context, resolution policy.Resolution, request Request) (Result, error) {
	if !resolution.Configured() {
		return Result{}, &ProviderError{Code: CodeUnconfigured, Message: "missing credentials: " + strings.Join(resolution.Missing, ", ")}
	}
	if len(bytes.TrimSpace(request.State)) == 0 || len(bytes.TrimSpace(request.Questions)) == 0 {
		return Result{}, errors.New("decision request requires state and questions")
	}
	body := map[string]any{"model": resolution.Model, "state": request.State, "questions": request.Questions}
	endpoint := resolution.Endpoint
	if resolution.Protocol == policy.ProtocolOllaya {
		endpoint = LocalURL(c.GetEnv) + "/api/decide"
	}
	started := c.now()
	raw, err := c.post(ctx, endpoint, resolution.Token, body)
	latency := c.now().Sub(started).Milliseconds()
	if err != nil {
		return Result{}, err
	}
	if resolution.Envelope == policy.EnvelopeCloudflare {
		if raw, err = unwrapCloudflare(raw); err != nil {
			return Result{}, err
		}
	}
	var wire struct {
		Model          string          `json:"model"`
		Answers        json.RawMessage `json:"answers"`
		Usage          Usage           `json:"usage"`
		Cost           *float64        `json:"cost"`
		StateTruncated bool            `json:"state_truncated"`
		EvalDuration   *int64          `json:"eval_duration"`
		LoadDuration   *int64          `json:"load_duration"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return Result{}, &ProviderError{Code: CodeProviderFailed, Message: "decode decision response: " + err.Error()}
	}
	// A decision on part of the state is not a decision on the state. The
	// hosted /v1 shape refuses with STATE_TRUNCATED; the native route reports
	// the truncation, and it is refused the same way here.
	if wire.StateTruncated {
		return Result{}, &ProviderError{Code: CodeContextOverflow, Message: "state does not fit " + firstNonEmpty(wire.Model, resolution.Model) + "'s context"}
	}
	if len(bytes.TrimSpace(wire.Answers)) == 0 || bytes.Equal(bytes.TrimSpace(wire.Answers), []byte("null")) {
		return Result{}, &ProviderError{Code: CodeProviderFailed, Message: "decision response carried no answers"}
	}
	return Result{
		Upstream: resolution.Upstream, Locality: resolution.Locality,
		Model: firstNonEmpty(wire.Model, resolution.Model), Answers: wire.Answers, Usage: wire.Usage,
		CostUSD: wire.Cost, LatencyMS: latency,
		EvalMS: nanosToMillis(wire.EvalDuration), LoadMS: nanosToMillis(wire.LoadDuration),
	}, nil
}

// LocalModels lists the daemon's installed models by canonical name and
// manifest digest.
func (c Client) LocalModels(ctx context.Context) (map[string]string, error) {
	raw, err := c.get(ctx, LocalURL(c.GetEnv)+"/api/tags")
	if err != nil {
		return nil, err
	}
	var tags struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &tags); err != nil {
		return nil, &ProviderError{Code: CodeProviderFailed, Message: "decode /api/tags: " + err.Error()}
	}
	out := make(map[string]string, len(tags.Models))
	for _, model := range tags.Models {
		out[model.Name] = model.Digest
	}
	return out, nil
}

// Pull downloads one model by name (optionally name@sha256:<digest>) and waits
// for the daemon to finish.
func (c Client) Pull(ctx context.Context, name string) error {
	_, err := c.post(ctx, LocalURL(c.GetEnv)+"/api/pull", "", map[string]any{"model": name, "stream": false})
	return err
}

// Version probes the daemon.
func (c Client) Version(ctx context.Context) (string, error) {
	raw, err := c.get(ctx, LocalURL(c.GetEnv)+"/api/version")
	if err != nil {
		return "", err
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &version); err != nil {
		return "", &ProviderError{Code: CodeProviderFailed, Message: "decode /api/version: " + err.Error()}
	}
	return version.Version, nil
}

// InstalledName returns the local name a policy model is listed under; the
// daemon lists an untagged name as name:latest.
func InstalledName(name string) string {
	if strings.Contains(name, ":") {
		return name
	}
	return name + ":latest"
}

func (c Client) post(ctx context.Context, endpoint, token string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return c.do(request)
}

func (c Client) get(ctx context.Context, endpoint string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	return c.do(request)
}

func (c Client) do(request *http.Request) ([]byte, error) {
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, classifyTransport(request.URL.Host, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, &ProviderError{Code: CodeProviderOverloaded, Message: "read response: " + err.Error()}
	}
	if response.StatusCode/100 != 2 {
		return nil, classifyStatus(response, raw)
	}
	return raw, nil
}

// classifyStatus maps an upstream HTTP failure to a gateway code. Ollaya's and
// TypeSafe's error bodies carry a machine code; it is preferred over the
// status where they disagree (a 422 STATE_TRUNCATED is a context overflow, a
// 422 TOO_MANY_OPTIONS is this model's refusal).
func classifyStatus(response *http.Response, raw []byte) error {
	var body struct {
		Error any    `json:"error"`
		Code  string `json:"code"`
	}
	_ = json.Unmarshal(raw, &body)
	message := strings.TrimSpace(fmt.Sprint(body.Error))
	if body.Error == nil || message == "" {
		message = strings.TrimSpace(string(raw))
	}
	if len(message) > 300 {
		message = message[:300]
	}
	failure := &ProviderError{HTTPStatus: response.StatusCode, RetryAfter: strings.TrimSpace(response.Header.Get("Retry-After")), Message: message}
	switch {
	case body.Code == "STATE_TRUNCATED" || body.Code == "INPUT_TOO_LONG":
		failure.Code = CodeContextOverflow
	case body.Code == "MODEL_NOT_FOUND":
		failure.Code = CodeUnconfigured
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		failure.Code = CodeUnconfigured
	case response.StatusCode == http.StatusTooManyRequests:
		failure.Code = CodeRateLimited
	case response.StatusCode == http.StatusServiceUnavailable || response.StatusCode == http.StatusBadGateway || response.StatusCode == http.StatusGatewayTimeout:
		failure.Code = CodeProviderOverloaded
	default:
		failure.Code = CodeProviderFailed
	}
	return failure
}

func classifyTransport(host string, err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &ProviderError{Code: CodeProviderOverloaded, Message: host + ": " + err.Error()}
	}
	return &ProviderError{Code: CodeUnreachable, Message: host + ": " + err.Error()}
}

func unwrapCloudflare(raw []byte) ([]byte, error) {
	var envelope struct {
		Result  json.RawMessage `json:"result"`
		Success *bool           `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, &ProviderError{Code: CodeProviderFailed, Message: "decode Cloudflare envelope: " + err.Error()}
	}
	if envelope.Success != nil && !*envelope.Success {
		messages := make([]string, 0, len(envelope.Errors))
		for _, failure := range envelope.Errors {
			messages = append(messages, failure.Message)
		}
		return nil, &ProviderError{Code: CodeProviderFailed, Message: "Cloudflare reported failure: " + strings.Join(messages, "; ")}
	}
	if len(envelope.Result) == 0 {
		return nil, &ProviderError{Code: CodeProviderFailed, Message: "Cloudflare envelope carried no result"}
	}
	return envelope.Result, nil
}

func (c Client) now() time.Time {
	if c.Now == nil {
		return time.Now()
	}
	return c.Now()
}

func nanosToMillis(value *int64) *int64 {
	if value == nil {
		return nil
	}
	millis := *value / int64(time.Millisecond)
	return &millis
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
