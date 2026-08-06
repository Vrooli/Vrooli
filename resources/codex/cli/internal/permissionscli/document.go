package permissionscli

import (
	"encoding/json"
	"errors"
	"fmt"

	"resource-codex/cli/internal/permissions"

	"github.com/vrooli/cli-core/agentpolicy"
)

var codexPermissionPosture = agentpolicy.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Codex's native permission settings remain intent-only; Vrooli projects a PreToolUse hook, but hook firing requires a live canary on the installed Codex version."}}

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

func (h *Handlers) planDocument(path, scopeRaw string) (agentpolicy.PermissionPlanResult, permissions.Policy, *permissions.Adapter, error) {
	adapter, err := h.adapter(scopeRaw)
	if err != nil {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	document, data, err := agentpolicy.LoadPermissionDocument(path, h.Stdin)
	if err != nil {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	if document.Scope != "" && document.Scope != string(adapter.Scope) {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, nil, fmt.Errorf("document scope %q does not match --scope %q", document.Scope, adapter.Scope)
	}
	document.Scope = string(adapter.Scope)
	live, err := adapter.Load()
	if err != nil {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	allow, ask, deny := agentpolicy.PermissionPatterns(document)
	desired := permissions.Policy{BashAllow: allow, BashAsk: ask, BashDeny: deny}
	return agentpolicy.PlanPermissionProjection("codex", document, data,
		agentpolicy.PermissionProjection{Allow: live.BashAllow, Ask: live.BashAsk, Deny: live.BashDeny}, []string{adapter.SettingsPath}, codexPermissionPosture), desired, adapter, nil
}

func (h *Handlers) writePlan(result agentpolicy.PermissionPlanResult, asJSON bool) error {
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
