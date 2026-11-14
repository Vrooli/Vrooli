package investigation

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "investigate"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"InvestigationPolicyService.GetPolicy":       h.getPolicy,
		"InvestigationPolicyService.PutPolicy":       h.putPolicy,
		"InvestigationPolicyService.PreviewTrigger":  h.preview,
		"InvestigationPolicyService.RecordTrigger":   h.record,
		"InvestigationPolicyService.LinkTrigger":     h.link,
		"InvestigationPolicyService.ListIncidents":   h.listIncidents,
		"InvestigationPolicyService.GetIncident":     h.getIncident,
		"InvestigationPolicyService.ListOccurrences": h.listOccurrences,
		"InvestigationPolicyService.GetBrief":        h.getBrief,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("investigation: load group: %w", err)
	}
	return group, nil
}
