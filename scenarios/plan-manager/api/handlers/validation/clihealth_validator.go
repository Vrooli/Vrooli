package validation

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

	internalvalidation "plan-manager/internal/validation"
)

const (
	cliHealthScenarioID     = "cli-health"
	defaultCLIHealthTimeout = 3 * time.Second
)

type cliHealthCommandValidator struct {
	resolver   scenarioURLResolver
	httpClient *http.Client
	timeout    time.Duration
}

func newCLIHealthCommandValidator() cliHealthCommandValidator {
	return cliHealthCommandValidator{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
	}
}

func (v cliHealthCommandValidator) ValidateCommandReference(ctx context.Context, req internalvalidation.CommandReferenceRequest) (internalvalidation.CommandReferenceResult, error) {
	resolver := v.resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	timeout := v.timeout
	if timeout <= 0 {
		timeout = defaultCLIHealthTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	baseURL, err := resolver.ResolveScenarioURLDefault(callCtx, cliHealthScenarioID)
	if err != nil {
		return internalvalidation.CommandReferenceResult{}, fmt.Errorf("resolve cli-health URL: %w", err)
	}
	body, err := json.Marshal(map[string]any{
		"commandText": strings.TrimSpace(req.CommandText),
		"policy":      "COMMAND_REFERENCE_POLICY_PLAN",
		"qualifiers":  req.Qualifiers,
	})
	if err != nil {
		return internalvalidation.CommandReferenceResult{}, fmt.Errorf("marshal request: %w", err)
	}
	call, err := http.NewRequestWithContext(callCtx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/vrooli.cli_health.v1.command.CommandReferenceValidationService/ValidateCommandReference", bytes.NewReader(body))
	if err != nil {
		return internalvalidation.CommandReferenceResult{}, fmt.Errorf("create request: %w", err)
	}
	call.Header.Set("Content-Type", "application/json")
	call.Header.Set("Accept", "application/json")
	resp, err := client.Do(call)
	if err != nil {
		return internalvalidation.CommandReferenceResult{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return internalvalidation.CommandReferenceResult{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
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
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return internalvalidation.CommandReferenceResult{}, fmt.Errorf("decode response: %w", err)
	}
	out := internalvalidation.CommandReferenceResult{
		Verdict:         normalizeCLIHealthEnum(decoded.Result.Verdict, "COMMAND_REFERENCE_VERDICT_"),
		ValidationLevel: normalizeCLIHealthEnum(decoded.Result.ValidationLevel, "COMMAND_REFERENCE_VALIDATION_LEVEL_"),
		Guidance:        append([]string(nil), decoded.Result.Guidance...),
	}
	for _, issue := range decoded.Result.Issues {
		out.Issues = append(out.Issues, internalvalidation.CommandIssue{Code: issue.Code, Message: issue.Message})
	}
	for _, suggestion := range decoded.Result.Suggestions {
		if suggestion.Command != "" {
			out.Suggestions = append(out.Suggestions, suggestion.Command)
		}
	}
	return out, nil
}

func normalizeCLIHealthEnum(value, prefix string) string {
	value = strings.TrimPrefix(strings.TrimSpace(value), prefix)
	return strings.ToLower(value)
}
