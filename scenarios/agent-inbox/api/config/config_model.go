// Package config provides centralized configuration for the Agent Inbox scenario.
// This file resolves the default OpenRouter chat model from a policy role.
package config

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// resolveRoleModelExec is the seam used to resolve an OpenRouter policy role to
// a concrete model slug. It shells out to the resource-openrouter binary by
// default; tests override it to avoid spawning the real process.
var resolveRoleModelExec = func(ctx context.Context, role string) (string, error) {
	cmd := exec.CommandContext(ctx, "resource-openrouter", "policy", "resolve", "--role", role, "--field", "model")
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr != "" {
			return "", fmt.Errorf("%w: %s", err, stderr)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ResolveDefaultChatModel resolves the default OpenRouter model slug for new
// chats directly from the environment (DEFAULT_AI_MODEL explicit override, else
// DEFAULT_AI_ROLE resolved via resource-openrouter). It is a convenience for
// call sites that do not carry a *Config. There is no concrete fallback slug.
func ResolveDefaultChatModel(ctx context.Context) (string, error) {
	ai := AIConfig{
		DefaultRole:  getEnvOrDefault("DEFAULT_AI_ROLE", "chat.default"),
		DefaultModel: getEnvOrDefault("DEFAULT_AI_MODEL", ""),
	}
	return ai.ResolveModel(ctx)
}

// ResolveModel returns the concrete OpenRouter model slug for new chats.
//
// Resolution order:
//  1. An explicit operator override (DefaultModel / DEFAULT_AI_MODEL), if set.
//  2. The DefaultRole resolved to a slug via resource-openrouter.
//
// There is intentionally no hard-coded fallback slug: if the resource is
// unavailable and no explicit override is set, this fails clearly so the
// operator can fix the policy/role configuration rather than silently using a
// stale model.
func (c AIConfig) ResolveModel(ctx context.Context) (string, error) {
	if override := strings.TrimSpace(c.DefaultModel); override != "" {
		return override, nil
	}

	role := strings.TrimSpace(c.DefaultRole)
	if role == "" {
		role = "chat.default"
	}

	rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	model, err := resolveRoleModelExec(rctx, role)
	if err != nil {
		return "", fmt.Errorf("resolve OpenRouter model for role %q via resource-openrouter: %w", role, err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return "", fmt.Errorf("resource-openrouter returned an empty model slug for role %q", role)
	}
	return model, nil
}
