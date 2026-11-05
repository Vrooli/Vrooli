package permissionscli

import (
	"fmt"

	"github.com/vrooli/vrooli/resources/grok/cli/internal/permissions"

	"github.com/vrooli/agentharness"
)

var grokPermissionPosture = agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Grok native permission rules remain active; the portable PreToolUse runner requires an installed-version canary before it is considered verified."}}

func (h *Handlers) planDocument(path, scopeRaw string) (agentharness.PermissionPlanResult, func() error, error) {
	adapter, err := h.adapter(scopeRaw)
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	document, data, err := agentharness.LoadPermissionDocumentForRunner(path, h.Stdin, "grok")
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	if document.Scope != "" && document.Scope != string(adapter.Scope) {
		return agentharness.PermissionPlanResult{}, nil, fmt.Errorf("document scope %q does not match --scope %q", document.Scope, adapter.Scope)
	}
	document.Scope = string(adapter.Scope)
	live, err := adapter.Load()
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	allow, ask, deny := agentharness.PermissionPatterns(document)
	desired := permissions.Policy{BashAllow: agentharness.BashToolPatterns(allow), BashAsk: agentharness.BashToolPatterns(ask), BashDeny: agentharness.BashToolPatterns(deny), Hooks: true}
	paths := []string{adapter.SettingsPath, "vrooli-policy-runner (PreToolUse command; installed-version canary required)"}
	result := agentharness.PlanPermissionProjection("grok", document, data, agentharness.PermissionProjection{
		Allow: agentharness.PortableBashPatterns(live.BashAllow), Ask: agentharness.PortableBashPatterns(live.BashAsk), Deny: agentharness.PortableBashPatterns(live.BashDeny),
	}, paths, grokPermissionPosture)
	return result, func() error {
		if err := adapter.Save(desired); err != nil {
			return err
		}
		return adapter.WriteState(desired, h.CLIVersion)
	}, nil
}
