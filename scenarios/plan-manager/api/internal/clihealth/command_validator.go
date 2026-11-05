package clihealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	scenarioID     = "cli-health"
	defaultTimeout = 3 * time.Second
)

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type Client struct {
	resolver   URLResolver
	httpClient *http.Client
	timeout    time.Duration
}

type Request struct {
	CommandText string
	Qualifiers  []string
}

type Result struct {
	Verdict         string
	ValidationLevel string
	Issues          []Issue
	Suggestions     []string
	Guidance        []string
}

type Issue struct {
	Code    string
	Message string
}

type Adapter[DomainRequest any, DomainResult any] struct {
	client       Client
	toRequest    func(DomainRequest) Request
	fromResponse func(Result) DomainResult
}

func NewClient() Client {
	return Client{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
	}
}

func NewAdapter[DomainRequest any, DomainResult any](
	toRequest func(DomainRequest) Request,
	fromResponse func(Result) DomainResult,
) Adapter[DomainRequest, DomainResult] {
	return Adapter[DomainRequest, DomainResult]{
		client:       NewClient(),
		toRequest:    toRequest,
		fromResponse: fromResponse,
	}
}

func (a Adapter[DomainRequest, DomainResult]) ValidateCommandReference(ctx context.Context, req DomainRequest) (DomainResult, error) {
	result, err := a.client.ValidateCommandReference(ctx, a.toRequest(req))
	if err != nil {
		var zero DomainResult
		return zero, err
	}
	return a.fromResponse(result), nil
}

func (c Client) ValidateCommandReference(ctx context.Context, req Request) (Result, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.effectiveTimeout())
	defer cancel()

	baseURL, err := c.effectiveResolver().ResolveScenarioURLDefault(callCtx, scenarioID)
	if err != nil {
		return Result{}, fmt.Errorf("resolve cli-health URL: %w", err)
	}
	body, err := json.Marshal(map[string]any{
		"commandText": strings.TrimSpace(req.CommandText),
		"policy":      "COMMAND_REFERENCE_POLICY_PLAN",
		"qualifiers":  req.Qualifiers,
	})
	if err != nil {
		return Result{}, fmt.Errorf("marshal request: %w", err)
	}
	call, err := http.NewRequestWithContext(callCtx, http.MethodPost, validationURL(baseURL), bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	call.Header.Set("Content-Type", "application/json")
	call.Header.Set("Accept", "application/json")

	resp, err := c.effectiveHTTPClient().Do(call)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return decodeValidationResponse(respBody)
}

func (c Client) effectiveResolver() URLResolver {
	if c.resolver != nil {
		return c.resolver
	}
	return discovery.NewResolver(discovery.ResolverConfig{})
}

func (c Client) effectiveHTTPClient() *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}
	return http.DefaultClient
}

func (c Client) effectiveTimeout() time.Duration {
	if c.timeout > 0 {
		return c.timeout
	}
	return defaultTimeout
}

func validationURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/vrooli.cli_health.v1.command.CommandReferenceValidationService/ValidateCommandReference"
}

func decodeValidationResponse(body []byte) (Result, error) {
	var decoded struct {
		Result struct {
			Verdict         string `json:"verdict"`
			ValidationLevel string `json:"validationLevel"`
			Issues          []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"issues"`
			Suggestions []struct {
				Command string `json:"command"`
			} `json:"suggestions"`
			Guidance []string `json:"guidance"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Result{}, fmt.Errorf("decode response: %w", err)
	}

	out := Result{
		Verdict:         normalizeEnum(decoded.Result.Verdict, "COMMAND_REFERENCE_VERDICT_"),
		ValidationLevel: normalizeEnum(decoded.Result.ValidationLevel, "COMMAND_REFERENCE_VALIDATION_LEVEL_"),
		Guidance:        append([]string(nil), decoded.Result.Guidance...),
	}
	for _, issue := range decoded.Result.Issues {
		out.Issues = append(out.Issues, Issue{Code: issue.Code, Message: issue.Message})
	}
	for _, suggestion := range decoded.Result.Suggestions {
		if suggestion.Command != "" {
			out.Suggestions = append(out.Suggestions, suggestion.Command)
		}
	}
	return out, nil
}

func normalizeEnum(value, prefix string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), prefix)
	return strings.ToLower(value)
}
