package permissionscli

import (
	"fmt"

	"github.com/vrooli/vrooli/resources/claude-code/cli/internal/permissions"

	"github.com/vrooli/agentharness"
)

var claudePermissionPosture = agentharness.EnforcementPosture{Permissions: "hook_verified", Caveats: []string{"Claude native permission denials remain active; the source-controlled PreToolUse matcher is verified by data-only replay and a non-mutating live probe."}}

func (h *Handlers) planDocument(path, _ string) (agentharness.PermissionPlanResult, func() error, error) {
	document, data, err := agentharness.LoadPermissionDocumentForRunner(path, h.Stdin, "claude-code")
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	if document.Scope != "" && document.Scope != "user" {
		return agentharness.PermissionPlanResult{}, nil, fmt.Errorf("Claude Code supports only scope user, got %q", document.Scope)
	}
	live, err := h.Adapter.Load()
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	allow, ask, deny := agentharness.PermissionPatterns(document)
	desired := permissions.Policy{BashAllow: agentharness.BashToolPatterns(allow), BashAsk: agentharness.BashToolPatterns(ask), BashDeny: agentharness.BashToolPatterns(deny), Hooks: true}
	result := agentharness.PlanPermissionProjection("claude-code", document, data, agentharness.PermissionProjection{
		Allow: agentharness.PortableBashPatterns(live.BashAllow), Ask: agentharness.PortableBashPatterns(live.BashAsk), Deny: agentharness.PortableBashPatterns(live.BashDeny),
	}, []string{h.Adapter.SettingsPath}, claudePermissionPosture)
	return result, func() error {
		if err := h.Adapter.Save(desired); err != nil {
			return err
		}
		return h.Adapter.WriteState(desired, h.CLIVersion)
	}, nil
}
