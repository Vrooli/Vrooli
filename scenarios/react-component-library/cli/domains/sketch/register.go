package sketch

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "sketch"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"SketchService.RenderCandidate":          {Run: h.renderCandidate},
		"SketchService.AcceptCandidate":          {Run: h.accept},
		"SketchService.GetAcceptance":            {Run: h.acceptance},
		"SketchService.CheckCandidateAcceptance": {Run: h.acceptanceCheck},
		"SketchService.ProposeSketch":            {Run: h.propose},
		"SketchService.InferSketch":              {Run: h.infer},
		"SketchService.GetSketchInference":       {Run: h.inference},
		"SketchService.GetHistory":               {Run: h.history},
		"SketchService.RecoverSketch":            {Run: h.recover},
		"SketchService.GetSketch":                {Run: h.get},
		"SketchService.PutSketch":                {Run: h.set},
		"SketchService.VerifySketch":             {Run: h.verify},
		"SketchService.ImportPage":               {Run: h.importPage},
		"SketchService.Place":                    {Run: h.place},
		"SketchService.Placeholder":              {Run: h.placeholder},
		"SketchService.AddNote":                  {Run: h.note},
		"SketchService.Unplace":                  {Run: h.unplace},
		"SketchService.SetTemplate":              {Run: h.setTemplate},
		"SketchService.BuildBrief":               {Run: h.brief},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("sketch: load from manifest: %w", err)
	}
	return group, nil
}
