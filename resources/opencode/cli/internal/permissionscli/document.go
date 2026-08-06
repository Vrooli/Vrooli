package permissionscli

import (
	"encoding/json"
	"errors"
	"fmt"

	"resource-opencode/cli/internal/permissions"

	"github.com/vrooli/cli-core/agentpolicy"
)

var openCodePermissionPosture = agentpolicy.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"OpenCode native permission rules are projected alongside tool.execute.before; plugin firing and refusal require a live canary on the installed version."}}

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
		if err := h.Adapter.Save(desired, h.CLIVersion); err != nil {
			return err
		}
	}
	return h.writePlan(result, *jsonOut)
}

func (h *Handlers) planDocument(path string) (agentpolicy.PermissionPlanResult, permissions.Policy, error) {
	document, data, err := agentpolicy.LoadPermissionDocument(path, h.Stdin)
	if err != nil {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, err
	}
	if document.Scope != "" && document.Scope != "user" {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, fmt.Errorf("OpenCode supports only scope user, got %q", document.Scope)
	}
	live, err := h.Adapter.Load()
	if err != nil {
		return agentpolicy.PermissionPlanResult{}, permissions.Policy{}, err
	}
	allow, ask, deny := agentpolicy.PermissionPatterns(document)
	desired := permissions.Policy{BashAllow: allow, BashAsk: ask, BashDeny: deny}
	return agentpolicy.PlanPermissionProjection("opencode", document, data,
		agentpolicy.PermissionProjection{Allow: live.BashAllow, Ask: live.BashAsk, Deny: live.BashDeny}, []string{h.Adapter.SettingsPath}, openCodePermissionPosture), desired, nil
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
	fmt.Fprintf(h.Stdout, "runner=%s drift=%t changes=%d enforcement=%s\n", result.Runner, result.Drift, len(result.Changes), result.Enforcement.Permissions)
	return nil
}
