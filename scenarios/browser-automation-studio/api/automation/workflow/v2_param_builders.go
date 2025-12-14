package workflow

import (
	"github.com/vrooli/browser-automation-studio/internal/params"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// Parameter builder functions delegate to internal/params for implementation.
// These thin wrappers maintain backward compatibility with existing callers.

func buildNavigateParams(data map[string]any) *basactions.NavigateParams {
	return params.BuildNavigateParams(data)
}

func buildClickParams(data map[string]any) *basactions.ClickParams {
	return params.BuildClickParams(data)
}

func buildInputParams(data map[string]any) *basactions.InputParams {
	return params.BuildInputParams(data)
}

func buildWaitParams(data map[string]any) *basactions.WaitParams {
	return params.BuildWaitParams(data)
}

func buildAssertParams(data map[string]any) *basactions.AssertParams {
	return params.BuildAssertParams(data)
}

func buildScrollParams(data map[string]any) *basactions.ScrollParams {
	return params.BuildScrollParams(data)
}

func buildSelectParams(data map[string]any) *basactions.SelectParams {
	return params.BuildSelectParams(data)
}

func buildEvaluateParams(data map[string]any) *basactions.EvaluateParams {
	return params.BuildEvaluateParams(data)
}

func buildKeyboardParams(data map[string]any) *basactions.KeyboardParams {
	return params.BuildKeyboardParams(data)
}

func buildHoverParams(data map[string]any) *basactions.HoverParams {
	return params.BuildHoverParams(data)
}

func buildScreenshotParams(data map[string]any) *basactions.ScreenshotParams {
	return params.BuildScreenshotParams(data)
}

func buildFocusParams(data map[string]any) *basactions.FocusParams {
	return params.BuildFocusParams(data)
}

func buildBlurParams(data map[string]any) *basactions.BlurParams {
	return params.BuildBlurParams(data)
}

func buildActionMetadata(data map[string]any) *basactions.ActionMetadata {
	return params.BuildActionMetadata(data)
}
