package dochealth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	commandv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command"
	commandconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command/command_v1connect"
)

const (
	cliHealthScenarioID     = "cli-health"
	defaultCLIHealthTimeout = 3 * time.Second
)

type CommandReferenceRequest struct {
	CommandText string
	Qualifiers  []string
}

type CommandReferenceIssue struct {
	Code     string
	Message  string
	Severity string
}

type CommandReferenceSuggestion struct {
	Command string
	Reason  string
}

type CommandReferenceResult struct {
	CommandText      string
	Verdict          string
	ValidationLevel  string
	CanonicalCommand string
	Issues           []CommandReferenceIssue
	Suggestions      []CommandReferenceSuggestion
	Guidance         []string
}

type CommandReferenceValidator interface {
	ValidateCommandReference(context.Context, CommandReferenceRequest) (CommandReferenceResult, error)
}

type scenarioURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type cliHealthCommandReferenceValidator struct {
	resolver   scenarioURLResolver
	httpClient connect.HTTPClient
	timeout    time.Duration
}

func NewCLIHealthCommandReferenceValidator() CommandReferenceValidator {
	return cliHealthCommandReferenceValidator{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
	}
}

func (v cliHealthCommandReferenceValidator) ValidateCommandReference(ctx context.Context, req CommandReferenceRequest) (CommandReferenceResult, error) {
	resolver := v.resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := v.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	timeout := v.timeout
	if timeout <= 0 {
		timeout = defaultCLIHealthTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseURL, err := resolver.ResolveScenarioURLDefault(callCtx, cliHealthScenarioID)
	if err != nil {
		return CommandReferenceResult{}, fmt.Errorf("resolve cli-health URL: %w", err)
	}
	resp, err := commandconnect.NewCommandReferenceValidationServiceClient(httpClient, baseURL).ValidateCommandReference(callCtx, connect.NewRequest(&commandv1.ValidateCommandReferenceRequest{
		CommandText: strings.TrimSpace(req.CommandText),
		Policy:      commandv1.CommandReferencePolicy_COMMAND_REFERENCE_POLICY_DOCS,
		Qualifiers:  append([]string(nil), req.Qualifiers...),
	}))
	if err != nil {
		return CommandReferenceResult{}, fmt.Errorf("validate command reference: %w", err)
	}
	return commandReferenceResultFromProto(resp.Msg.GetResult()), nil
}

func commandReferenceResultFromProto(r *commandv1.CommandReferenceValidationResult) CommandReferenceResult {
	if r == nil {
		return CommandReferenceResult{Verdict: "unknown", ValidationLevel: "unspecified"}
	}
	out := CommandReferenceResult{
		CommandText:      r.GetCommandText(),
		Verdict:          commandEnumSuffix(r.GetVerdict().String(), "COMMAND_REFERENCE_VERDICT_"),
		ValidationLevel:  commandEnumSuffix(r.GetValidationLevel().String(), "COMMAND_REFERENCE_VALIDATION_LEVEL_"),
		CanonicalCommand: r.GetCanonicalCommand(),
		Guidance:         append([]string(nil), r.GetGuidance()...),
	}
	for _, issue := range r.GetIssues() {
		out.Issues = append(out.Issues, CommandReferenceIssue{
			Code:     issue.GetCode(),
			Message:  issue.GetMessage(),
			Severity: issue.GetSeverity(),
		})
	}
	for _, suggestion := range r.GetSuggestions() {
		out.Suggestions = append(out.Suggestions, CommandReferenceSuggestion{
			Command: suggestion.GetCommand(),
			Reason:  suggestion.GetReason(),
		})
	}
	return out
}

func commandEnumSuffix(value, prefix string) string {
	value = strings.TrimPrefix(value, prefix)
	value = strings.ToLower(value)
	if value == "unspecified" {
		return ""
	}
	return value
}
