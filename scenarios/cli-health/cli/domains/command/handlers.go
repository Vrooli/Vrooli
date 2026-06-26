package command

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	commandv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command"
	commandconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command/command_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client commandconnect.CommandReferenceValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: commandconnect.NewCommandReferenceValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	commandText := strings.TrimSpace(ctx.Flag("command"))
	if commandText == "" {
		commandText = strings.TrimSpace(ctx.Positional("command-ref"))
	}
	resp, err := h.client.ValidateCommandReference(context.Background(), connect.NewRequest(&commandv1.ValidateCommandReferenceRequest{
		CommandText: commandText,
		Policy:      parsePolicy(ctx.Flag("policy")),
		Qualifiers:  parseCSV(ctx.Flag("qualifiers")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate command reference", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Result == nil {
		return fmt.Errorf("server returned no command validation result")
	}
	r := resp.Msg.Result
	results := []string{
		fmt.Sprintf("Command: %s", r.CommandText),
		fmt.Sprintf("Verdict: %s", trimEnum(r.Verdict.String(), "COMMAND_REFERENCE_VERDICT_")),
		fmt.Sprintf("Level: %s", trimEnum(r.ValidationLevel.String(), "COMMAND_REFERENCE_VALIDATION_LEVEL_")),
	}
	if r.CanonicalCommand != "" {
		results = append(results, fmt.Sprintf("Canonical: %s", r.CanonicalCommand))
	}
	for _, issue := range r.Issues {
		results = append(results, fmt.Sprintf("%s: %s", issue.Code, issue.Message))
	}
	for _, suggestion := range r.Suggestions {
		results = append(results, fmt.Sprintf("Suggestion: %s (%s)", suggestion.Command, suggestion.Reason))
	}
	for _, guidance := range r.Guidance {
		results = append(results, "Next: "+guidance)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Command reference validation: %s.", strings.ToLower(trimEnum(r.Verdict.String(), "COMMAND_REFERENCE_VERDICT_")))},
		ResultsHeading: "Validation",
		Results:        results,
	})
}

func parsePolicy(s string) commandv1.CommandReferencePolicy {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "docs":
		return commandv1.CommandReferencePolicy_COMMAND_REFERENCE_POLICY_DOCS
	case "skill":
		return commandv1.CommandReferencePolicy_COMMAND_REFERENCE_POLICY_SKILL
	case "plan":
		return commandv1.CommandReferencePolicy_COMMAND_REFERENCE_POLICY_PLAN
	case "action":
		return commandv1.CommandReferencePolicy_COMMAND_REFERENCE_POLICY_ACTION
	default:
		return commandv1.CommandReferencePolicy_COMMAND_REFERENCE_POLICY_UNSPECIFIED
	}
}

func parseCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func trimEnum(s, prefix string) string {
	return strings.TrimPrefix(s, prefix)
}
