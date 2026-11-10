package rehearsals

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/rehearsals/rehearsals_v1connect"
)

const GroupName = "rehearsals"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	prefix := connect.RehearsalServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		prefix + "CreateAndRunRehearsal": cliapp.ProtoMutation(h.createCall, h.createReport),
		prefix + "ListRehearsals":        cliapp.ProtoList(h.listCall, h.listReport),
		prefix + "GetRehearsal":          cliapp.ProtoList(h.getCall, h.getReport),
		prefix + "RecordReview":          cliapp.ProtoMutation(h.reviewCall, h.reviewReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("rehearsals: load from manifest: %w", err)
	}
	return group, nil
}
