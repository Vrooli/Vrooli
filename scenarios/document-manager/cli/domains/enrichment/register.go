package enrichment

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "enrichment"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"EnrichmentService.Enrich": cliapp.ProtoMutation(h.enrichCall, h.enrichReport),
		"EnrichmentService.Embed":  cliapp.ProtoMutation(h.embedCall, h.embedReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("enrichment: load from manifest: %w", err)
	}
	return group, nil
}
