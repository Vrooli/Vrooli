package compiler

import (
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

// Parameter builder functions delegate to internal/typeconv for implementation.
// These convert flat parameter maps to typed proto messages.

func buildNavigateParams(data map[string]any) *basactions.NavigateParams {
	return typeconv.BuildNavigateParams(data)
}

func buildClickParams(data map[string]any) *basactions.ClickParams {
	return typeconv.BuildClickParams(data)
}

func buildInputParams(data map[string]any) *basactions.InputParams {
	return typeconv.BuildInputParams(data)
}

func buildWaitParams(data map[string]any) *basactions.WaitParams {
	return typeconv.BuildWaitParams(data)
}

func buildAssertParams(data map[string]any) *basactions.AssertParams {
	return typeconv.BuildAssertParams(data)
}

func buildScrollParams(data map[string]any) *basactions.ScrollParams {
	return typeconv.BuildScrollParams(data)
}

func buildSelectParams(data map[string]any) *basactions.SelectParams {
	return typeconv.BuildSelectParams(data)
}

func buildEvaluateParams(data map[string]any) *basactions.EvaluateParams {
	return typeconv.BuildEvaluateParams(data)
}

func buildKeyboardParams(data map[string]any) *basactions.KeyboardParams {
	return typeconv.BuildKeyboardParams(data)
}

func buildHoverParams(data map[string]any) *basactions.HoverParams {
	return typeconv.BuildHoverParams(data)
}

func buildScreenshotParams(data map[string]any) *basactions.ScreenshotParams {
	return typeconv.BuildScreenshotParams(data)
}

func buildFocusParams(data map[string]any) *basactions.FocusParams {
	return typeconv.BuildFocusParams(data)
}

func buildBlurParams(data map[string]any) *basactions.BlurParams {
	return typeconv.BuildBlurParams(data)
}

func buildActionMetadata(data map[string]any) *basactions.ActionMetadata {
	return typeconv.BuildActionMetadata(data)
}
