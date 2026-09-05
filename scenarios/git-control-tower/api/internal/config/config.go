// Package config loads operator-tunable policy from
// `<scenarioDir>/.vrooli/config.json` under the top-level `policy` key.
//
// Today this is the agent-access gate (allow|warn|confirm|deny) for
// mutating Connect-RPC methods. The gate is enforced server-side via a
// Connect interceptor (see api/internal/policygate package) and mirrored
// client-side by the CLI for friction at the point of intent.
//
// Distinct from service.json (lifecycle/dependencies) — see
// docs/concepts/ARCHITECTURE.md "Policy Gate" for the full rationale.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AgentAccess is the policy applied to mutating commands when the caller
// is classified as an agent.
type AgentAccess string

const (
	// AgentAccessAllow means agents may run mutating commands without
	// friction. Off by default for safety.
	AgentAccessAllow AgentAccess = "allow"
	// AgentAccessWarn runs the command but prints the agent-deny message
	// first. Useful while migrating consumers.
	AgentAccessWarn AgentAccess = "warn"
	// AgentAccessConfirm requires the override flag (agentOverrideFlag)
	// to be passed (CLI) or X-Vrooli-Authorized: true header (RPC).
	// Without it, the command is refused with the agent-deny message.
	// Default policy.
	AgentAccessConfirm AgentAccess = "confirm"
	// AgentAccessDeny refuses with the agent-deny message; no override
	// flag bypasses.
	AgentAccessDeny AgentAccess = "deny"
)

// CallerDetection chooses between the strict and broad detectors in
// packages/cli-core/cliutil/agent_context.go.
type CallerDetection string

const (
	// CallerDetectionStrict matches only Vrooli-spawned agent contexts
	// (IsAgentControlledContext).
	CallerDetectionStrict CallerDetection = "strict"
	// CallerDetectionBroad matches Vrooli-spawned AND known external
	// agent runtimes (IsLikelyAgentContext). Default.
	CallerDetectionBroad CallerDetection = "broad"
)

// PolicyConfig governs the agent-access gate.
type PolicyConfig struct {
	// AgentAccess is the gate applied to mutating commands. Default
	// confirm.
	AgentAccess AgentAccess `json:"agentAccess,omitempty"`

	// AgentOverrideFlag is the CLI flag that satisfies the `confirm`
	// policy. Default --i-was-explicitly-authorized.
	AgentOverrideFlag string `json:"agentOverrideFlag,omitempty"`

	// CallerDetection chooses which cliutil detector to use. Default
	// broad.
	CallerDetection CallerDetection `json:"callerDetection,omitempty"`

	// AgentDenyMessageTemplate is rendered when an agent caller is
	// refused by the gate. Supports {command} (full command identifier,
	// e.g. "WorktreeService.CreateWorktree" or "repo commit") and
	// {policy} placeholders. Empty falls back to the hardcoded default.
	AgentDenyMessageTemplate string `json:"agentDenyMessageTemplate,omitempty"`
}

// Config is the loader's top-level shape.
type Config struct {
	Policy PolicyConfig `json:"policy"`
}

// DefaultConfig returns the locked defaults: confirm policy, broad
// detection, standard override flag, empty message template (the
// runtime uses its hardcoded strong default).
func DefaultConfig() Config {
	return Config{
		Policy: PolicyConfig{
			AgentAccess:              AgentAccessConfirm,
			AgentOverrideFlag:        "--i-was-explicitly-authorized",
			CallerDetection:          CallerDetectionBroad,
			AgentDenyMessageTemplate: "",
		},
	}
}

// Load reads `<scenarioDir>/.vrooli/config.json`, merges its `policy` key
// onto DefaultConfig(), and returns the result. Missing file or missing
// `policy` key resolves to DefaultConfig() (greenfield). Only malformed
// JSON returns a non-nil error.
func Load(scenarioDir string) (Config, error) {
	defaults := DefaultConfig()
	if scenarioDir == "" {
		return defaults, nil
	}
	path := filepath.Join(scenarioDir, ".vrooli", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return defaults, nil
	}
	var raw struct {
		Policy *PolicyConfig `json:"policy,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return defaults, fmt.Errorf("parse %s: %w", path, err)
	}
	if raw.Policy == nil {
		return defaults, nil
	}
	merged := defaults
	if v := strings.TrimSpace(string(raw.Policy.AgentAccess)); v != "" {
		merged.Policy.AgentAccess = AgentAccess(v)
	}
	if v := strings.TrimSpace(raw.Policy.AgentOverrideFlag); v != "" {
		merged.Policy.AgentOverrideFlag = v
	}
	if v := strings.TrimSpace(string(raw.Policy.CallerDetection)); v != "" {
		merged.Policy.CallerDetection = CallerDetection(v)
	}
	if v := raw.Policy.AgentDenyMessageTemplate; v != "" {
		merged.Policy.AgentDenyMessageTemplate = v
	}
	if err := merged.Policy.Validate(); err != nil {
		return defaults, fmt.Errorf("validate %s: %w", path, err)
	}
	return merged, nil
}

// Validate enforces the enum constraints on PolicyConfig. Unknown values
// abort startup so an operator typo doesn't silently relax the gate.
func (p PolicyConfig) Validate() error {
	switch p.AgentAccess {
	case AgentAccessAllow, AgentAccessWarn, AgentAccessConfirm, AgentAccessDeny:
	default:
		return fmt.Errorf("policy.agentAccess: unknown value %q (allow|warn|confirm|deny)", p.AgentAccess)
	}
	switch p.CallerDetection {
	case CallerDetectionStrict, CallerDetectionBroad:
	default:
		return fmt.Errorf("policy.callerDetection: unknown value %q (strict|broad)", p.CallerDetection)
	}
	if p.AgentOverrideFlag == "" {
		return errors.New("policy.agentOverrideFlag: must not be empty")
	}
	return nil
}
