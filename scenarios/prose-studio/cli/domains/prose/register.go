package prose

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "prose"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{}
	for _, method := range []string{"Registry", "CreateStyle", "ResolveProfile", "Generate", "Reroll", "SessionAction", "ReindexDeclarations", "ValidateDeclarations", "CreateDocument", "AssembleDocument", "Conformance"} {
		bindings["ProseStudioService."+method] = cliapp.ProtoList(h.call(method), h.report)
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("prose: load from manifest: %w", err)
	}
	return group, nil
}
