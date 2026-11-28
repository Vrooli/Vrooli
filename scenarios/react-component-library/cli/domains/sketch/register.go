package sketch

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "sketch"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"WorkspaceCandidateService.RenderCandidate": cliapp.ProtoCustom(h.renderCandidateCall, renderCandidateHuman),
		"WorkspaceAssistService.ProposeSketch":      cliapp.ProtoCustom(h.proposeCall, proposeHuman),
		"WorkspaceAssistService.InferSketch":        cliapp.ProtoCustom(h.inferCall, inferenceHuman),
		"WorkspaceAssistService.GetSketchInference": cliapp.ProtoCustom(h.inferenceCall, inferenceHuman),
		"ExperienceService.GetHistory":              cliapp.ProtoCustom(h.historyCall, historyHuman),
		"ExperienceService.RecoverSketch":           cliapp.ProtoCustom(h.recoverCall, recoverHuman),
		"ExperienceService.GetSketch":               cliapp.ProtoCustom(h.getCall, getHuman),
		"ExperienceService.PutSketch":               cliapp.ProtoCustom(h.setCall, setHuman),
		"ExperienceService.VerifySketch":            cliapp.ProtoListOutcome(h.verifyCall, verifyReport, verifyOutcome),
		"ExperienceService.ImportPage":              cliapp.ProtoCustom(h.importPageCall, importPageHuman),
		"ExperienceService.Place":                   cliapp.ProtoCustom(h.placeCall, placeHuman),
		"ExperienceService.Placeholder":             cliapp.ProtoCustom(h.placeholderCall, placeholderHuman),
		"ExperienceService.AddNote":                 cliapp.ProtoCustom(h.noteCall, noteHuman),
		"ExperienceService.Unplace":                 cliapp.ProtoCustom(h.unplaceCall, unplaceHuman),
		"ExperienceService.SetTemplate":             cliapp.ProtoCustom(h.setTemplateCall, setTemplateHuman),
		"ExperienceService.BuildBrief":              cliapp.ProtoCustom(h.briefCall, briefHuman),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("sketch: load from manifest: %w", err)
	}
	return group, nil
}
