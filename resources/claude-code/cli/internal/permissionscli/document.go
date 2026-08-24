package permissionscli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"resource-claude-code/cli/internal/permissions"

	"github.com/vrooli/agentharness"
)

var claudePermissionPosture = agentharness.EnforcementPosture{Permissions: "hook_verified", Caveats: []string{"Claude native permission denials remain active; the source-controlled PreToolUse matcher is verified by data-only replay and a non-mutating live probe."}}

func (h *Handlers) Plan(args []string) error {
	fs := h.flagSet("permissions plan")
	documentPath := fs.String("document", "", "Path to the whole desired permission document, or - for stdin")
	jsonOut := fs.Bool("json", false, "Emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *documentPath == "" {
		return errors.New("--document is required")
	}
	result, _, err := h.planDocument(*documentPath)
	if err != nil {
		return err
	}
	return h.writePlan(result, *jsonOut)
}

func (h *Handlers) Reconcile(args []string) error {
	fs := h.flagSet("permissions reconcile")
	documentPath := fs.String("document", "", "Path to the whole desired permission document, or - for stdin")
	jsonOut := fs.Bool("json", false, "Emit machine-readable JSON")
	authorized := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate (only when a human explicitly authorized this call)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *documentPath == "" {
		return errors.New("--document is required")
	}
	result, desired, err := h.planDocument(*documentPath)
	if err != nil {
		return err
	}
	if err := h.gate("permissions reconcile", true, *authorized); err != nil {
		return err
	}
	if result.Drift {
		if err := h.Adapter.Save(desired); err != nil {
			return err
		}
		if err := h.Adapter.WriteState(desired, h.CLIVersion); err != nil {
			return err
		}
	}
	return h.writePlan(result, *jsonOut)
}

func (h *Handlers) planDocument(path string) (agentharness.PermissionPlanResult, permissions.Policy, error) {
	document, data, err := agentharness.LoadPermissionDocument(path, h.Stdin)
	if err != nil {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, err
	}
	if document.Scope != "" && document.Scope != "user" {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, fmt.Errorf("Claude Code supports only scope user, got %q", document.Scope)
	}
	live, err := h.Adapter.Load()
	if err != nil {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, err
	}
	allow, ask, deny := agentharness.PermissionPatterns(document)
	allow = claudeBashPatterns(allow)
	ask = claudeBashPatterns(ask)
	deny = claudeBashPatterns(deny)
	desired := permissions.Policy{BashAllow: allow, BashAsk: ask, BashDeny: deny, Hooks: true}
	paths := []string{h.Adapter.SettingsPath}
	return agentharness.PlanPermissionProjection("claude-code", document, data,
		agentharness.PermissionProjection{Allow: claudePortablePatterns(live.BashAllow), Ask: claudePortablePatterns(live.BashAsk), Deny: claudePortablePatterns(live.BashDeny)}, paths, claudePermissionPosture), desired, nil
}

func claudeBashPatterns(patterns []string) []string {
	formatted := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		formatted = append(formatted, "Bash("+pattern+")")
	}
	return formatted
}

func claudePortablePatterns(patterns []string) []string {
	portable := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.HasPrefix(pattern, "Bash(") && strings.HasSuffix(pattern, ")") {
			portable = append(portable, strings.TrimSuffix(strings.TrimPrefix(pattern, "Bash("), ")"))
			continue
		}
		portable = append(portable, pattern)
	}
	return portable
}

func (h *Handlers) writePlan(result agentharness.PermissionPlanResult, asJSON bool) error {
	if asJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(h.Stdout, string(data))
		return err
	}
	fmt.Fprintf(h.Stdout, "runner=%s drift=%t changes=%d enforcement=%s\n", result.Runner, result.Drift, len(result.Changes), result.Enforcement.Permissions)
	return nil
}
