package forecasts

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "forecasts"

func Register(app *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(app)
	bindings := map[string]cliapp.PrimitiveHandler{"ForecastsService.GetForecast": cliapp.ProtoMutation(h.getCall, h.getReport), "ForecastsService.ListForecastSnapshots": cliapp.ProtoList(h.historyCall, h.historyReport)}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil { return cliapp.SubcommandGroup{}, fmt.Errorf("forecasts: load from manifest: %w", err) }
	return group, nil
}
