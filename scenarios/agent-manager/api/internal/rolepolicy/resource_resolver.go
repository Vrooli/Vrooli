// Package rolepolicy resolves portable coding roles through resource-owned
// policy CLIs. Agent Manager never reads resource model catalogs or native
// agent configuration files directly.
package rolepolicy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"agent-manager/internal/domain"
)

const resourcePolicySchemaVersion = "v1"

var (
	// ErrResourceUnavailable means the resource CLI could not answer a role
	// query. Callers can continue to a later fallback candidate.
	ErrResourceUnavailable = errors.New("resource role policy unavailable")
	// ErrUnknownResourceRole means the resource is reachable but does not
	// declare the requested portable role.
	ErrUnknownResourceRole = errors.New("unknown resource role")
	// ErrInvalidResourceResponse means a resource CLI returned a response that
	// cannot safely become immutable execution evidence.
	ErrInvalidResourceResponse = errors.New("invalid resource role response")
)

// CommandExecutor is the subprocess seam. Production uses CombinedOutput so
// resource diagnostics remain available for error classification; tests use a
// deterministic fake and never need a resource binary.
type CommandExecutor interface {
	Run(ctx context.Context, command string, args ...string) ([]byte, error)
}

type commandExecutor struct{}

func (commandExecutor) Run(ctx context.Context, command string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, command, args...).CombinedOutput()
}

// ResourceRoleResolver resolves one role from the resource that owns the
// concrete model vocabulary for the requested runner.
type ResourceRoleResolver struct {
	executor CommandExecutor
}

// Resolver is the narrow role-resolution seam used by the Agent Manager role
// catalog. Implementations must return resource-owned concrete evidence and
// never consult an Agent Manager model inventory.
type Resolver interface {
	Resolve(ctx context.Context, runner domain.RunnerType, role string) (ResolvedRole, error)
}

func NewResourceRoleResolver(executor CommandExecutor) *ResourceRoleResolver {
	if executor == nil {
		executor = commandExecutor{}
	}
	return &ResourceRoleResolver{executor: executor}
}

// ResolvedRole is the validated resource response. It intentionally carries
// concrete models only as resolved evidence, never as Agent Manager input.
type ResolvedRole struct {
	Runner         domain.RunnerType
	Role           string
	Model          string
	CanonicalModel string
	Fallbacks      []string
	Capabilities   []string
	Provenance     ResourceProvenance
	Enforcement    EnforcementPosture
	PolicyPath     string
	PolicyDigest   string
	Billing        domain.BillingSnapshot
	Challenger     *domain.ChallengerConfig
}

type ResourceProvenance struct {
	Source     string
	ObservedAt string
}

type EnforcementPosture struct {
	Permissions string
	Caveats     []string
}

type resourceRoleResponse struct {
	SchemaVersion  string   `json:"schema_version"`
	Runner         string   `json:"runner"`
	Role           string   `json:"role"`
	Model          string   `json:"model"`
	CanonicalModel string   `json:"canonical_model,omitempty"`
	Fallbacks      []string `json:"fallbacks"`
	Description    string   `json:"description"`
	Capabilities   []string `json:"capabilities"`
	Provenance     struct {
		Source     string `json:"source"`
		ObservedAt string `json:"observed_at"`
	} `json:"provenance"`
	Enforcement struct {
		Permissions string   `json:"permissions"`
		Caveats     []string `json:"caveats"`
	} `json:"enforcement"`
	PolicyPath   string                   `json:"policy_path"`
	PolicyDigest string                   `json:"policy_digest"`
	Billing      domain.BillingSnapshot   `json:"billing,omitempty"`
	Challenger   *domain.ChallengerConfig `json:"challenger,omitempty"`
}

// Resolve executes `<resource> policy resolve --role <role> --json` and
// validates that the result corresponds to the exact requested runner/role.
func (r *ResourceRoleResolver) Resolve(ctx context.Context, runner domain.RunnerType, role string) (ResolvedRole, error) {
	if !runner.IsValid() {
		return ResolvedRole{}, fmt.Errorf("%w: runner %q is unsupported", ErrInvalidResourceResponse, runner)
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return ResolvedRole{}, fmt.Errorf("%w: role is required", ErrUnknownResourceRole)
	}
	if r == nil || r.executor == nil {
		return ResolvedRole{}, fmt.Errorf("%w: resolver is not configured", ErrResourceUnavailable)
	}

	output, err := r.executor.Run(ctx, resourceCommand(runner), "policy", "resolve", "--role", role, "--json")
	if err != nil {
		if strings.Contains(strings.ToLower(string(output)+" "+err.Error()), "unknown coding role") {
			return ResolvedRole{}, fmt.Errorf("%w: runner %q does not declare %q: %v", ErrUnknownResourceRole, runner, role, err)
		}
		return ResolvedRole{}, fmt.Errorf("%w: resolve %q through %s: %v", ErrResourceUnavailable, role, resourceCommand(runner), err)
	}

	response, err := parseResourceRoleResponse(output)
	if err != nil {
		return ResolvedRole{}, fmt.Errorf("%w: runner %q role %q: %v", ErrInvalidResourceResponse, runner, role, err)
	}
	if response.Runner != string(runner) {
		return ResolvedRole{}, fmt.Errorf("%w: response runner %q does not match requested runner %q", ErrInvalidResourceResponse, response.Runner, runner)
	}
	if response.Role != role {
		return ResolvedRole{}, fmt.Errorf("%w: response role %q does not match requested role %q", ErrInvalidResourceResponse, response.Role, role)
	}
	return response.toResolvedRole(), nil
}

func resourceCommand(runner domain.RunnerType) string {
	return "resource-" + string(runner)
}

func parseResourceRoleResponse(data []byte) (resourceRoleResponse, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var response resourceRoleResponse
	if err := decoder.Decode(&response); err != nil {
		return resourceRoleResponse{}, fmt.Errorf("parse JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if !errors.Is(err, io.EOF) {
			return resourceRoleResponse{}, fmt.Errorf("parse trailing JSON: %w", err)
		}
	} else {
		return resourceRoleResponse{}, errors.New("multiple JSON values")
	}
	if err := response.validate(); err != nil {
		return resourceRoleResponse{}, err
	}
	return response, nil
}

func (r resourceRoleResponse) validate() error {
	if r.SchemaVersion != resourcePolicySchemaVersion {
		return fmt.Errorf("schema_version must be %q", resourcePolicySchemaVersion)
	}
	if !domain.RunnerType(r.Runner).IsValid() {
		return fmt.Errorf("runner %q is unsupported", r.Runner)
	}
	for field, value := range map[string]string{
		"role": r.Role, "model": r.Model, "description": r.Description,
		"provenance.source": r.Provenance.Source, "provenance.observed_at": r.Provenance.ObservedAt,
		"policy_path": r.PolicyPath, "policy_digest": r.PolicyDigest,
	} {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return fmt.Errorf("%s is required", field)
		}
	}
	if len(r.Capabilities) == 0 {
		return errors.New("capabilities must not be empty")
	}
	if r.Billing.Mode != "" {
		switch r.Billing.Mode {
		case domain.BillingModeMetered, domain.BillingModeSubscription, domain.BillingModeLocal, domain.BillingModeUnknown:
		default:
			return fmt.Errorf("billing.mode %q is unsupported", r.Billing.Mode)
		}
	}
	for _, capability := range r.Capabilities {
		if strings.TrimSpace(capability) == "" || strings.TrimSpace(capability) != capability {
			return errors.New("capabilities must contain trimmed values")
		}
	}
	for _, fallback := range r.Fallbacks {
		if strings.TrimSpace(fallback) == "" || strings.TrimSpace(fallback) != fallback {
			return errors.New("fallbacks must contain trimmed values")
		}
	}
	if r.Challenger != nil && (strings.TrimSpace(r.Challenger.Model) == "" || r.Challenger.SampleRate < 0 || r.Challenger.SampleRate > 1) {
		return errors.New("challenger requires a model and sample_rate between 0 and 1")
	}
	switch r.Enforcement.Permissions {
	case "native", "hook_backed", "intent_only":
	default:
		return fmt.Errorf("enforcement.permissions %q is unsupported", r.Enforcement.Permissions)
	}
	return nil
}

func (r resourceRoleResponse) toResolvedRole() ResolvedRole {
	return ResolvedRole{
		Runner:         domain.RunnerType(r.Runner),
		Role:           r.Role,
		Model:          r.Model,
		CanonicalModel: r.CanonicalModel,
		Fallbacks:      append([]string(nil), r.Fallbacks...),
		Capabilities:   append([]string(nil), r.Capabilities...),
		Provenance:     ResourceProvenance{Source: r.Provenance.Source, ObservedAt: r.Provenance.ObservedAt},
		Enforcement:    EnforcementPosture{Permissions: r.Enforcement.Permissions, Caveats: append([]string(nil), r.Enforcement.Caveats...)},
		PolicyPath:     r.PolicyPath,
		PolicyDigest:   r.PolicyDigest,
		Billing:        r.Billing,
		Challenger:     r.Challenger,
	}
}
