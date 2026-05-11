package codegen

import (
	"fmt"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/layout"
	"react-vite-temporal-model/internal/model"
)

type File struct {
	Path      string
	Data      []byte
	Generated bool
}

type RenderResult struct {
	Files []File
}

// Render produces the codegen output for a flow. Exactly two files are
// emitted regardless of language: a runtime declarations file and a
// replay helper. Both live under generated/<folderName>/.
func Render(flow model.Flow, built artifact.Artifact) (RenderResult, error) {
	switch flow.Layout.Language {
	case layout.LanguageGo:
		runtime, err := renderGoRuntime(flow, built)
		if err != nil {
			return RenderResult{}, err
		}
		replay, err := renderGoReplayHelper(flow)
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
