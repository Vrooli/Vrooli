package permissionscli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"resource-grok/cli/internal/permissions"

	"github.com/vrooli/agentharness"
)

var grokPermissionPosture = agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Grok native permission rules remain active; the portable PreToolUse runner requires an installed-version canary before it is considered verified."}}

func (h *Handlers) Plan(args []string) error {
	fs, scopeRaw := h.flagSet("permissions plan")
	documentPath := fs.String("document", "", "Path to the whole desired permission document, or - for stdin")
	jsonOut := fs.Bool("json", false, "Emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *documentPath == "" {
		return errors.New("--document is required")
	}
	result, _, _, err := h.planDocument(*documentPath, *scopeRaw)
	if err != nil {
		return err
	}
	return h.writePlan(result, *jsonOut)
}

func (h *Handlers) Reconcile(args []string) error {
	fs, scopeRaw := h.flagSet("permissions reconcile")
	documentPath := fs.String("document", "", "Path to the whole desired permission document, or - for stdin")
	jsonOut := fs.Bool("json", false, "Emit machine-readable JSON")
	authorized := fs.Bool("i-was-explicitly-authorized", false, "Override the agent gate (only when a human explicitly authorized this call)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *documentPath == "" {
		return errors.New("--document is required")
	}
	result, desired, adapter, err := h.planDocument(*documentPath, *scopeRaw)
	if err != nil {
		return err
	}
	if err := h.gate("permissions reconcile", true, *authorized); err != nil {
		return err
	}
	if result.Drift {
		if err := adapter.Save(desired); err != nil {
			return err
		}
		if err := adapter.WriteState(desired, h.CLIVersion); err != nil {
			return err
		}
	}
	return h.writePlan(result, *jsonOut)
}

func (h *Handlers) planDocument(path, scopeRaw string) (agentharness.PermissionPlanResult, permissions.Policy, *permissions.Adapter, error) {
	adapter, err := h.adapter(scopeRaw)
	if err != nil {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	document, data, err := agentharness.LoadPermissionDocument(path, h.Stdin)
	if err != nil {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	if document.Scope != "" && document.Scope != string(adapter.Scope) {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, nil, fmt.Errorf("document scope %q does not match --scope %q", document.Scope, adapter.Scope)
	}
	document.Scope = string(adapter.Scope)
	live, err := adapter.Load()
	if err != nil {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	allow, ask, deny := agentharness.PermissionPatterns(document)
	allow = grokBashPatterns(allow)
	ask = grokBashPatterns(ask)
	deny = grokBashPatterns(deny)
	desired := permissions.Policy{BashAllow: allow, BashAsk: ask, BashDeny: deny, Hooks: true}
	paths := []string{adapter.SettingsPath, "vrooli-policy-runner (PreToolUse command; installed-version canary required)"}
	return agentharness.PlanPermissionProjection("grok", document, data,
		agentharness.PermissionProjection{Allow: grokPortablePatterns(live.BashAllow), Ask: grokPortablePatterns(live.BashAsk), Deny: grokPortablePatterns(live.BashDeny)}, paths, grokPermissionPosture), desired, adapter, nil
}

func grokBashPatterns(patterns []string) []string {
	formatted := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		formatted = append(formatted, "Bash("+pattern+")")
	}
	return formatted
}

func grokPortablePatterns(patterns []string) []string {
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
	fmt.Fprintf(h.Stdout, "runner=%s scope=%s drift=%t changes=%d enforcement=%s\n", result.Runner, result.Scope, result.Drift, len(result.Changes), result.Enforcement.Permissions)
	return nil
}
