package mfa

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "mfa"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"MFAService.BeginEnrollment":   h.beginEnrollment,
		"MFAService.ConfirmEnrollment": h.confirmEnrollment,
		"MFAService.RemoveEnrollment":  h.removeEnrollment,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("mfa: load from manifest: %w", err)
	}
	return group, nil
}
