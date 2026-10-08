package commitments

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "commitments"

func Register(app *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(app)
	bindings := map[string]cliapp.PrimitiveHandler{"CommitmentsService.ListCommitments": cliapp.ProtoList(h.listCall, h.listReport), "CommitmentsService.CreateCommitment": cliapp.ProtoMutation(h.createCall, h.createReport), "CommitmentsService.UpdateCommitmentState": cliapp.ProtoMutation(h.stateCall, h.stateReport), "CommitmentsService.ReviseCommitment": cliapp.ProtoMutation(h.reviseCall, h.reviseReport)}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("commitments: load from manifest: %w", err)
	}
	return group, nil
}
