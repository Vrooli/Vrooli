package codegen

import (
	"fmt"
	"path/filepath"

	"react-vite-temporal-model/internal/artifact"
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

func Render(flow model.Flow, built artifact.Artifact) (RenderResult, error) {
	var result RenderResult
	switch filepath.Ext(flow.Outputs.DeclarationsPath) {
	case ".go":
		declarations, err := renderGoDeclarations(flow, built)
		if err != nil {
			return RenderResult{}, err
		}
		replayTest, err := renderGoReplayTest(flow)
		if err != nil {
			return RenderResult{}, err
		}
		result.Files = append(result.Files,
			File{Path: flow.Outputs.DeclarationsPath, Data: []byte(declarations), Generated: true},
			File{Path: flow.Outputs.ReplayTestPath, Data: []byte(replayTest), Generated: true},
		)
	case ".ts":
		declarations, err := renderTypeScriptDeclarations(flow, built)
		if err != nil {
			return RenderResult{}, err
		}
		replayHelper, err := renderTypeScriptReplayHelper(flow)
		if err != nil {
			return RenderResult{}, err
		}
		replayTest, err := renderTypeScriptReplayTest(flow)
		if err != nil {
			return RenderResult{}, err
		}
		result.Files = append(result.Files,
			File{Path: flow.Outputs.DeclarationsPath, Data: []byte(declarations), Generated: true},
			File{Path: flow.Outputs.ReplayHelperPath, Data: []byte(replayHelper), Generated: true},
			File{Path: flow.Outputs.ReplayTestPath, Data: []byte(replayTest), Generated: true},
		)
	default:
		return RenderResult{}, fmt.Errorf("unsupported declarationsPath extension for %s: %s", flow.FlowID, flow.Outputs.DeclarationsPath)
	}
	return result, nil
}
