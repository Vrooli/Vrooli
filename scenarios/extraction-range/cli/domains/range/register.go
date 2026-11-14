package rangecli

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	rangeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/extraction-range/v1/range/range_v1connect"
)

const GroupName = "range"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	service := rangeconnect.RangeServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		service + "ListLevels": cliapp.ProtoList(h.listCall, h.listReport),
		service + "GetLevel":   cliapp.ProtoList(h.getCall, h.getReport),
		service + "Submit":     cliapp.ProtoMutation(h.submitCall, h.submitReport),
		service + "Scoreboard": cliapp.ProtoList(h.scoreboardCall, h.scoreboardReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("range: load from manifest: %w", err)
	}
	return group, nil
}
