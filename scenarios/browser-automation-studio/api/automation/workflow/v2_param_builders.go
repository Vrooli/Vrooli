package workflow

import (
	"github.com/vrooli/browser-automation-studio/internal/params"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
)

// Parameter builder functions delegate to internal/params for implementation.
// These thin wrappers maintain backward compatibility with existing callers.

func buildNavigateParams(data map[string]any) *basv1.NavigateParams {
	return params.BuildNavigateParams(data)
}

func buildClickParams(data map[string]any) *basv1.ClickParams {
	return params.BuildClickParams(data)
}

func buildInputParams(data map[string]any) *basv1.InputParams {
	return params.BuildInputParams(data)
}

func buildWaitParams(data map[string]any) *basv1.WaitParams {
	return params.BuildWaitParams(data)
}

func buildAssertParams(data map[string]any) *basv1.AssertParams {
	return params.BuildAssertParams(data)
}

func buildScrollParams(data map[string]any) *basv1.ScrollParams {
	return params.BuildScrollParams(data)
}

func buildSelectParams(data map[string]any) *basv1.SelectParams {
	return params.BuildSelectParams(data)
}

func buildEvaluateParams(data map[string]any) *basv1.EvaluateParams {
	return params.BuildEvaluateParams(data)
}

func buildKeyboardParams(data map[string]any) *basv1.KeyboardParams {
	return params.BuildKeyboardParams(data)
}

func buildHoverParams(data map[string]any) *basv1.HoverParams {
	return params.BuildHoverParams(data)
}

func buildScreenshotParams(data map[string]any) *basv1.ScreenshotParams {
	return params.BuildScreenshotParams(data)
}

func buildFocusParams(data map[string]any) *basv1.FocusParams {
	return params.BuildFocusParams(data)
}

func buildBlurParams(data map[string]any) *basv1.BlurParams {
	return params.BuildBlurParams(data)
}

func buildActionMetadata(data map[string]any) *basv1.ActionMetadata {
	return params.BuildActionMetadata(data)
}
