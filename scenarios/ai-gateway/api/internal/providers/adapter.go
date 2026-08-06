package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	ProviderOllama     = "ollama"
	ProviderOpenRouter = "openrouter"
)

type Adapter struct {
	Provider    string
	CommandName string
	Locality    string
	Runner      CommandRunner
}

type Role struct {
	Provider            string
	Role                string
	Capabilities        []string
	Locality            string
	Status              string
	PolicySchemaVersion string
}

type SmokeResult struct {
	Provider string
	Status   string
	Code     string
	Message  string
	ExitCode int
	Warnings []string
}

type Inventory struct {
	Roles    []Role
	Warnings []string
}

type ExecutionRequest struct {
	Kind            sharedv1.RequestKind
	Role            string
	InputText       string
	MaxOutputTokens int32
	Timeout         time.Duration
	// Temperature requests a specific sampling temperature. It is a pointer so
	// that an explicit 0 (deterministic) is distinguishable from "unset, use
	// the resource default". Only providers whose CLI exposes the control
	// honour it; see Adapter.SupportsTemperature.
	Temperature *float64
}

type ExecutionResult struct {
	OutputText string
	ExitCode   int
}

type ResolvedRole struct {
	Provider       string
	Role           string
	Model          string
	CanonicalModel string
	Capabilities   []string
}

type rolePolicyReport struct {
	Roles []rolePolicyEntry `json:"roles"`
}

type rolePolicyEntry struct {
	SchemaVersion        string   `json:"schema_version"`
	Role                 string   `json:"role"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Capabilities         []string `json:"capabilities"`
}

func NewDefaultAdapters(runner CommandRunner) []Adapter {
	if runner == nil {
		runner = ExecRunner{}
	}
	return []Adapter{
		{Provider: ProviderOllama, CommandName: "resource-ollama", Locality: "local", Runner: runner},
		{Provider: ProviderOpenRouter, CommandName: "resource-openrouter", Locality: "remote", Runner: runner},
	}
}

func (a Adapter) ListRoles(ctx context.Context) (Inventory, error) {
	result, err := a.runPolicyRoles(ctx)
	if err != nil {
		return Inventory{}, mapCommandError(a.Provider, err)
	}

	var report rolePolicyReport
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		return Inventory{}, &CommandError{
			Code:     "malformed_json",
			Command:  a.command().String(),
			ExitCode: result.ExitCode,
			Stderr:   result.Stderr,
			Err:      err,
		}
	}

	inventory := Inventory{Roles: make([]Role, 0, len(report.Roles))}
	for _, entry := range report.Roles {
		roleName := strings.TrimSpace(entry.Role)
		if roleName == "" {
			inventory.Warnings = append(inventory.Warnings, fmt.Sprintf("%s policy returned a role without a name", a.Provider))
			continue
		}
		capabilities := entry.RequiredCapabilities
		if len(capabilities) == 0 {
			capabilities = entry.Capabilities
		}
		if len(capabilities) == 0 {
			inventory.Warnings = append(inventory.Warnings, fmt.Sprintf("%s role %q has no declared capabilities", a.Provider, roleName))
		}
		inventory.Roles = append(inventory.Roles, Role{
			Provider:            a.Provider,
			Role:                roleName,
			Capabilities:        sortedUnique(capabilities),
			Locality:            a.Locality,
			Status:              "available",
			PolicySchemaVersion: strings.TrimSpace(entry.SchemaVersion),
		})
	}
	sort.Slice(inventory.Roles, func(i, j int) bool {
		return inventory.Roles[i].Role < inventory.Roles[j].Role
	})
	return inventory, nil
}

// ResolveRole asks the resource for the concrete model selected by its
// policy. AI Gateway never reads provider model catalogs or credentials.
func (a Adapter) ResolveRole(ctx context.Context, role string) (ResolvedRole, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return ResolvedRole{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("role is required")}
	}
	result, err := a.runner().Run(ctx, Command{Name: a.CommandName, Args: []string{"policy", "resolve", "--role", role, "--json"}, Timeout: DefaultCommandTimeout})
	if err != nil {
		return ResolvedRole{}, mapCommandError(a.Provider, err)
	}
	var response struct {
		Role         string   `json:"role"`
		Model        string   `json:"model"`
		Canonical    string   `json:"canonical_model"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return ResolvedRole{}, &CommandError{Code: "malformed_json", Command: a.command().String(), ExitCode: result.ExitCode, Stderr: result.Stderr, Err: err}
	}
	if strings.TrimSpace(response.Role) != role || strings.TrimSpace(response.Model) == "" {
		return ResolvedRole{}, &CommandError{Code: "invalid_role_resolution", Command: a.command().String(), ExitCode: result.ExitCode, Err: fmt.Errorf("resource returned role %q and model %q for requested role %q", response.Role, response.Model, role)}
	}
	return ResolvedRole{Provider: a.Provider, Role: role, Model: response.Model, CanonicalModel: response.Canonical, Capabilities: sortedUnique(response.Capabilities)}, nil
}

func (a Adapter) Smoke(ctx context.Context) SmokeResult {
	inventory, err := a.ListRoles(ctx)
	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			return SmokeResult{
				Provider: a.Provider,
				Status:   "unavailable",
				Code:     cmdErr.Code,
				Message:  cmdErr.Error(),
				ExitCode: cmdErr.ExitCode,
			}
		}
		return SmokeResult{Provider: a.Provider, Status: "unavailable", Code: "unknown", Message: err.Error(), ExitCode: -1}
	}
	warnings := append([]string{}, inventory.Warnings...)
	if len(inventory.Roles) == 0 {
		warnings = append(warnings, "provider policy returned zero roles")
		return SmokeResult{Provider: a.Provider, Status: "degraded", Code: "empty_inventory", Message: "provider command responded but returned no roles", Warnings: warnings}
	}
	return SmokeResult{Provider: a.Provider, Status: "available", Code: "ok", Message: fmt.Sprintf("provider policy returned %d role(s)", len(inventory.Roles)), Warnings: warnings}
}

func (a Adapter) Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {
	command, err := a.executionCommand(req)
	if err != nil {
		return ExecutionResult{}, err
	}
	result, err := a.runner().Run(ctx, command)
	if err != nil {
		return ExecutionResult{ExitCode: result.ExitCode}, mapCommandError(a.Provider, err)
	}
	return ExecutionResult{
		OutputText: strings.TrimSpace(result.Stdout),
		ExitCode:   result.ExitCode,
	}, nil
}

func (a Adapter) runPolicyRoles(ctx context.Context) (Result, error) {
	return a.runner().Run(ctx, a.command())
}

func (a Adapter) command() Command {
	return Command{Name: a.CommandName, Args: []string{"policy", "roles", "--json"}, Timeout: DefaultCommandTimeout}
}

func (a Adapter) runner() CommandRunner {
	if a.Runner == nil {
		return ExecRunner{}
	}
	return a.Runner
}

func (a Adapter) executionCommand(req ExecutionRequest) (Command, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultCommandTimeout
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		return Command{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("role is required")}
	}
	input := req.InputText
	if strings.TrimSpace(input) == "" {
		return Command{}, &CommandError{Code: "invalid_request", Command: a.Provider, ExitCode: -1, Err: errors.New("input_text is required")}
	}
	switch a.Provider {
	case ProviderOllama:
		return a.ollamaExecutionCommand(req, role, input, timeout)
	case ProviderOpenRouter:
		return a.openRouterExecutionCommand(req, role, input, timeout)
	default:
		return Command{}, &CommandError{Code: "unsupported_provider", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("provider %q does not expose a gateway execution command", a.Provider)}
	}
}

func (a Adapter) ollamaExecutionCommand(req ExecutionRequest, role string, input string, timeout time.Duration) (Command, error) {
	switch req.Kind {
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING:
		return Command{
			Name:    a.CommandName,
			Args:    []string{"gateway", "embed", "--role", role, "--json", "--input-stdin"},
			Stdin:   input,
			Timeout: timeout,
		}, nil
	case sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION:
		args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
		if req.MaxOutputTokens > 0 {
			args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxOutputTokens))
		}
		if req.Temperature != nil {
			args = append(args, "--temperature", strconv.FormatFloat(*req.Temperature, 'g', -1, 64))
		}
		return Command{Name: a.CommandName, Args: args, Stdin: input, Timeout: timeout}, nil
	default:
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable", req.Kind.String())}
	}
}

func (a Adapter) openRouterExecutionCommand(req ExecutionRequest, role string, input string, timeout time.Duration) (Command, error) {
	if req.Kind != sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION &&
		req.Kind != sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION {
		return Command{}, &CommandError{Code: "unsupported_kind", Command: a.Provider, ExitCode: -1, Err: fmt.Errorf("request kind %s is not executable by openrouter", req.Kind.String())}
	}
	args := []string{"generate", "--role", role, "--json"}
	if req.MaxOutputTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", req.MaxOutputTokens))
	}
	return Command{Name: a.CommandName, Args: args, Stdin: input, Timeout: timeout}, nil
}

func mapCommandError(provider string, err error) error {
	var cmdErr *CommandError
	if errors.As(err, &cmdErr) {
		if cmdErr.Code == "" {
			cmdErr.Code = "command_failed"
		}
		return cmdErr
	}
	return &CommandError{Code: "command_failed", Command: provider, ExitCode: -1, Err: err}
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
