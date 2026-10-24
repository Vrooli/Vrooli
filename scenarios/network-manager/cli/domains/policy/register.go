package policy

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "policy"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"PolicyService.PreviewPolicyChange":        h.preview,
		"PolicyService.ApplyPolicyChange":          h.apply,
		"PolicyService.RollbackPolicyChange":       h.rollback,
		"PolicyService.PauseFiltering":             h.pause,
		"PolicyService.ResumeFiltering":            h.resume,
		"PolicyService.ListPolicyProfiles":         h.profiles,
		"PolicyService.UpsertPolicyProfile":        h.profileSet,
		"PolicyService.EvaluatePolicySchedule":     h.schedule,
		"PolicyService.DiagnoseEncryptedDnsBypass": h.bypassGuidance,
		"PolicyService.GetEndpointDohGuidance":     h.dohGuidance,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("policy: load from manifest: %w", err)
	}
	return group, nil
}
