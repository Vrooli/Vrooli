package codegen

import (
	"fmt"

	"flow-verifier/internal/flows/kinds/temporal/layout"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/verification/artifact"
)

type File struct {
	Path      string
	Data      []byte
	Generated bool
}

type RenderResult struct {
	Files []File
}

// Options carries render-time inputs that do not fit on the Flow or
// Artifact (e.g. the scenario's Go module path, needed by Go replay
// imports). Empty values fall back to the template placeholder
// {{SCENARIO_ID}} so the template itself can still be vendored as-is.
type Options struct {
	GoModulePath string
}

// Render produces the codegen output for a flow. Exactly two files are
// emitted regardless of language: a runtime declarations file and a
// replay helper. Both live under generated/<folderName>/.
func Render(flow model.Flow, built artifact.Artifact, opts Options) (RenderResult, error) {
	switch flow.Layout.Language {
	case layout.LanguageGo:
		runtime, err := renderGoRuntime(flow, built)
		if err != nil {
			return RenderResult{}, err
		}
		replay, err := renderGoReplayHelper(flow, opts)
		if err != nil {
			return RenderResult{}, err
		}
		return RenderResult{Files: []File{
			{Path: flow.Layout.RuntimePath, Data: []byte(runtime), Generated: true},
			{Path: flow.Layout.ReplayHelperPath, Data: []byte(replay), Generated: true},
		}}, nil
	case layout.LanguageTypeScript:
		runtime, err := renderTypeScriptRuntime(flow, built)
		if err != nil {
			return RenderResult{}, err
		}
		helper, err := renderTypeScriptReplayHelper(flow)
		if err != nil {
			return RenderResult{}, err
		}
		return RenderResult{Files: []File{
			{Path: flow.Layout.RuntimePath, Data: []byte(runtime), Generated: true},
			{Path: flow.Layout.ReplayHelperPath, Data: []byte(helper), Generated: true},
		}}, nil
	default:
		return RenderResult{}, fmt.Errorf("unsupported language %q for %s", flow.Layout.Language, flow.FlowID)
	}
}
