package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const (
	cliHealthScenarioID     = "cli-health"
	defaultCLIHealthTimeout = 3 * time.Second
)

type CLIHealthCommandResolver struct {
	resolver   scenarioURLResolver
	httpClient *http.Client
	timeout    time.Duration
}

type scenarioURLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

func NewCLIHealthCommandResolver() *CLIHealthCommandResolver {
	return &CLIHealthCommandResolver{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
	}
}

func (r *CLIHealthCommandResolver) ResolveCommand(ctx context.Context, argv []string) (CommandResolution, error) {
	if len(argv) == 0 {
		return CommandResolution{Certainty: CertaintyNone, Message: "command argv is empty"}, nil
	}
	commandText := commandTextFromArgv(argv)
	result, err := r.validate(ctx, commandText)
	if err != nil {
		return CommandResolution{
			Certainty: CertaintyNone,
			Target:    argv[0],
			Message:   "CLI Health command validation unavailable: " + err.Error(),
		}, nil
	}
	return commandResolutionFromCLIHealth(argv, result), nil
}

func (r *CLIHealthCommandResolver) validate(ctx context.Context, commandText string) (cliHealthValidationResult, error) {
	resolver := r.resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	client := r.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	timeout := r.timeout
	if timeout <= 0 {
		timeout = defaultCLIHealthTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	baseURL, err := resolver.ResolveScenarioURLDefault(callCtx, cliHealthScenarioID)
	if err != nil {
		return cliHealthValidationResult{}, fmt.Errorf("resolve cli-health URL: %w", err)
	}
	body, err := json.Marshal(map[string]any{
		"commandText": commandText,
		"policy":      "COMMAND_REFERENCE_POLICY_ACTION",
	})
	if err != nil {
		return cliHealthValidationResult{}, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/vrooli.cli_health.v1.command.CommandReferenceValidationService/ValidateCommandReference", bytes.NewReader(body))
	if err != nil {
		return cliHealthValidationResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return cliHealthValidationResult{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return cliHealthValidationResult{}, fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var decoded struct {
		Result cliHealthValidationResult `json:"result"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return cliHealthValidationResult{}, fmt.Errorf("decode response: %w", err)
	}
	return decoded.Result, nil
}

type cliHealthValidationResult struct {
	CommandText      string                          `json:"commandText"`
	Verdict          string                          `json:"verdict"`
	ValidationLevel  string                          `json:"validationLevel"`
	CanonicalCommand string                          `json:"canonicalCommand"`
	Owner            string                          `json:"owner"`
	Issues           []cliHealthValidationIssue      `json:"issues"`
	Suggestions      []cliHealthValidationSuggestion `json:"suggestions"`
	Guidance         []string                        `json:"guidance"`
}

type cliHealthValidationIssue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type cliHealthValidationSuggestion struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

func commandResolutionFromCLIHealth(argv []string, result cliHealthValidationResult) CommandResolution {
	target := ""
	if len(argv) > 0 {
		target = argv[0]
	}
	owner := CommandOwner{ID: result.Owner}
	if result.Owner == "vrooli" {
		owner.Type = "project"
	} else if result.Owner != "" {
		owner.Type = "scenario"
	}
	path := commandPathWithoutTarget(result.CanonicalCommand, target)
	message := cliHealthMessage(result)
	switch normalizeCLIHealthEnum(result.Verdict, "COMMAND_REFERENCE_VERDICT_") {
	case "VALID":
		return CommandResolution{
			Certainty:   CertaintyOwnerOnly,
			Owner:       owner,
			Target:      target,
			CommandPath: path,
			RunSurfaces: []string{"cli", "action"},
			Message:     "CLI Health validated command path and arguments, but action governance is unavailable; " + message,
		}
	case "PARTIAL":
		return CommandResolution{
			Certainty:   CertaintyOwnerOnly,
			Owner:       owner,
			Target:      target,
			CommandPath: path,
			RunSurfaces: []string{"cli", "action"},
			Message:     "CLI Health validated command path with partial argument coverage; " + message,
		}
	case "SKIPPED":
		return CommandResolution{Certainty: CertaintyNone, Owner: owner, Target: target, CommandPath: path, Message: "CLI Health skipped current-command validation; action commands must reference current commands"}
	default:
		return CommandResolution{Certainty: CertaintyNone, Owner: owner, Target: target, CommandPath: path, Message: message}
	}
}

func commandTextFromArgv(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		parts = append(parts, strconv.Quote(arg))
	}
	return strings.Join(parts, " ")
}

func commandPathWithoutTarget(command, target string) []string {
	parts := strings.Fields(command)
	if len(parts) > 0 && parts[0] == target {
		parts = parts[1:]
	}
	return parts
}

func cliHealthMessage(result cliHealthValidationResult) string {
	var parts []string
	verdict := normalizeCLIHealthEnum(result.Verdict, "COMMAND_REFERENCE_VERDICT_")
	level := normalizeCLIHealthEnum(result.ValidationLevel, "COMMAND_REFERENCE_VALIDATION_LEVEL_")
	if verdict != "" {
		if level != "" {
			parts = append(parts, strings.ToLower(verdict)+" at "+strings.ToLower(level))
		} else {
			parts = append(parts, strings.ToLower(verdict))
		}
	}
	for _, issue := range result.Issues {
		if issue.Code != "" && issue.Message != "" {
			parts = append(parts, issue.Code+": "+issue.Message)
		} else if issue.Message != "" {
			parts = append(parts, issue.Message)
		}
	}
	for _, suggestion := range result.Suggestions {
		if suggestion.Command != "" {
			parts = append(parts, "suggestion: "+suggestion.Command)
		}
	}
	parts = append(parts, result.Guidance...)
	if len(parts) == 0 {
		return "CLI Health returned no validation detail"
	}
	return strings.Join(parts, "; ")
}

func normalizeCLIHealthEnum(value, prefix string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, prefix)
	return strings.ToUpper(value)
}
