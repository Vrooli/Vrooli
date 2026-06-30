package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Role-based OpenRouter model selection.
//
// The concrete model slug is owned exclusively by the OpenRouter resource policy.
// There is intentionally NO hard-coded model slug anywhere in this package: when
// no explicit model override is supplied, the effective model is resolved at call
// time via `resource-openrouter policy resolve --role <role> --field model`.
const (
	// openRouterRoleEnv selects the policy role used to resolve the default model.
	openRouterRoleEnv = "BAS_OPENROUTER_ROLE"

	// defaultOpenRouterRole is the role consulted when none is configured.
	// This is a ROLE name, not a model slug — the slug it maps to is decided by
	// the OpenRouter resource policy, never by this code.
	defaultOpenRouterRole = "chat.default"
)

// resolveRoleModelFunc is a seam so tests can stub policy resolution without
// shelling out to the resource-openrouter binary.
var resolveRoleModelFunc = execResolveRoleModel

// openRouterRole returns the configured policy role, falling back to
// defaultOpenRouterRole when BAS_OPENROUTER_ROLE is unset or blank.
func openRouterRole() string {
	role := strings.TrimSpace(os.Getenv(openRouterRoleEnv))
	if role == "" {
		return defaultOpenRouterRole
	}
	return role
}

// resolveRoleModel resolves a concrete model slug for the given role through the
// OpenRouter resource policy. It returns an error (never a hard-coded fallback
// slug) when resolution fails.
func resolveRoleModel(ctx context.Context, role string) (string, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		role = defaultOpenRouterRole
	}
	return resolveRoleModelFunc(ctx, role)
}

// execResolveRoleModel shells out to resource-openrouter to resolve the role.
func execResolveRoleModel(ctx context.Context, role string) (string, error) {
	cmd := exec.CommandContext(ctx, openRouterCommand, "policy", "resolve", "--role", role, "--field", "model")
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return "", fmt.Errorf("resolve openrouter model for role %q via %s policy resolve: %w: %s", role, openRouterCommand, err, stderr)
			}
		}
		return "", fmt.Errorf("resolve openrouter model for role %q via %s policy resolve: %w", role, openRouterCommand, err)
	}

	model := strings.TrimSpace(string(out))
	if model == "" {
		return "", fmt.Errorf("%s policy resolve returned an empty model for role %q", openRouterCommand, role)
	}
	return model, nil
}
