package permissionscli

import (
	"fmt"

	"github.com/vrooli/vrooli/resources/opencode/cli/internal/permissions"

	"github.com/vrooli/agentharness"
)

var openCodePermissionPosture = agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"OpenCode native permission rules are projected alongside tool.execute.before; plugin firing and refusal require a live canary on the installed version."}}

func (h *Handlers) planDocument(path, _ string) (agentharness.PermissionPlanResult, func() error, error) {
	document, data, err := agentharness.LoadPermissionDocumentForRunner(path, h.Stdin, "opencode")
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	if document.Scope != "" && document.Scope != "user" {
		return agentharness.PermissionPlanResult{}, nil, fmt.Errorf("OpenCode supports only scope user, got %q", document.Scope)
	}
	live, err := h.Adapter.Load()
	if err != nil {
		return agentharness.PermissionPlanResult{}, nil, err
	}
	allow, ask, deny := agentharness.PermissionPatterns(document)
	desired := permissions.Policy{BashAllow: allow, BashAsk: ask, BashDeny: deny}
	result := agentharness.PlanPermissionProjection("opencode", document, data,
		agentharness.PermissionProjection{Allow: live.BashAllow, Ask: live.BashAsk, Deny: live.BashDeny}, []string{h.Adapter.SettingsPath}, openCodePermissionPosture)
	return result, func() error { return h.Adapter.Save(desired, h.CLIVersion) }, nil
}
