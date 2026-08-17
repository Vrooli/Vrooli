package prose

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/prose-studio/v1/prose"
)

const GroupName = "prose"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"ProseStudioService.Registry":             cliapp.ProtoList(h.registryCall, protoReport[*v1.RegistryResponse]("Prose Studio registry")),
		"ProseStudioService.CreateStyle":          cliapp.ProtoList(h.createStyleCall, protoReport[*v1.CreateStyleResponse]("Prose Studio style")),
		"ProseStudioService.ResolveProfile":       cliapp.ProtoList(h.resolveProfileCall, protoReport[*v1.ResolveProfileResponse]("Prose Studio profile")),
		"ProseStudioService.Generate":             cliapp.ProtoList(h.generateCall, protoReport[*v1.GenerateResponse]("Prose Studio generation")),
		"ProseStudioService.Reroll":               cliapp.ProtoList(h.rerollCall, protoReport[*v1.RerollResponse]("Prose Studio reroll")),
		"ProseStudioService.SessionAction":        cliapp.ProtoList(h.sessionActionCall, protoReport[*v1.SessionActionResponse]("Prose Studio session")),
		"ProseStudioService.ReindexDeclarations":  cliapp.ProtoList(h.reindexCall, protoReport[*v1.ReindexDeclarationsResponse]("Prose Studio declarations")),
		"ProseStudioService.ValidateDeclarations": cliapp.ProtoList(h.validateCall, protoReport[*v1.ValidateDeclarationsResponse]("Prose Studio declaration validation")),
		"ProseStudioService.CreateDocument":       cliapp.ProtoList(h.createDocumentCall, protoReport[*v1.CreateDocumentResponse]("Prose Studio document")),
		"ProseStudioService.AssembleDocument":     cliapp.ProtoList(h.assembleDocumentCall, protoReport[*v1.AssembleDocumentResponse]("Prose Studio assembled document")),
		"ProseStudioService.Conformance":          cliapp.ProtoList(h.conformanceCall, protoReport[*v1.ConformanceResponse]("Prose Studio conformance")),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("prose: load from manifest: %w", err)
	}
	return group, nil
}
