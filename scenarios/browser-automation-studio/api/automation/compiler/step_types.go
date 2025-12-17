// Package compiler provides workflow compilation utilities.
// This file re-exports action types from the unified automation/actions package
// for backwards compatibility with existing code.
package compiler

import "github.com/vrooli/browser-automation-studio/automation/actions"

// StepType is an alias to the unified ActionType for backwards compatibility.
// New code should import from automation/actions directly.
type StepType = actions.ActionType

// Re-export action type constants for backwards compatibility.
// New code should import from automation/actions directly.
const (
	StepNavigate     = actions.Navigate
	StepClick        = actions.Click
	StepHover        = actions.Hover
	StepDragDrop     = actions.DragDrop
	StepFocus        = actions.Focus
	StepBlur         = actions.Blur
	StepScroll       = actions.Scroll
	StepSelect       = actions.Select
	StepRotate       = actions.Rotate
	StepGesture      = actions.Gesture
	StepUploadFile   = actions.UploadFile
	StepTypeInput    = actions.TypeInput
	StepShortcut     = actions.Shortcut
	StepKeyboard     = actions.Keyboard
	StepWait         = actions.Wait
	StepScreenshot   = actions.Screenshot
	StepExtract      = actions.Extract
	StepEvaluate     = actions.Evaluate
	StepAssert       = actions.Assert
	StepCustom       = actions.Custom
	StepSetVariable  = actions.SetVariable
	StepUseVariable  = actions.UseVariable
	StepTabSwitch    = actions.TabSwitch
	StepFrameSwitch  = actions.FrameSwitch
	StepConditional  = actions.Conditional
	StepLoop         = actions.Loop
	StepSetCookie    = actions.SetCookie
	StepGetCookie    = actions.GetCookie
	StepClearCookie  = actions.ClearCookie
	StepSetStorage   = actions.SetStorage
	StepGetStorage   = actions.GetStorage
	StepClearStorage = actions.ClearStorage
	StepNetworkMock  = actions.NetworkMock
	StepSubflow      = actions.Subflow
)

// Re-export loop control targets for backwards compatibility.
const (
	LoopContinueTarget = actions.LoopContinueTarget
	LoopBreakTarget    = actions.LoopBreakTarget
)

// Re-export loop handles for backwards compatibility.
const (
	loopHandleBody     = actions.LoopHandleBody
	loopHandleAfter    = actions.LoopHandleAfter
	loopHandleBreak    = actions.LoopHandleBreak
	loopHandleContinue = actions.LoopHandleContinue

	loopConditionBody     = actions.LoopConditionBody
	loopConditionAfter    = actions.LoopConditionAfter
	loopConditionBreak    = actions.LoopConditionBreak
	loopConditionContinue = actions.LoopConditionContinue
)

// supportedStepTypes is derived from the unified registry for backwards compatibility.
// Use actions.Registry directly for new code.
var supportedStepTypes = func() map[StepType]struct{} {
	m := make(map[StepType]struct{}, len(actions.Registry))
	for t := range actions.Registry {
		m[t] = struct{}{}
	}
	return m
}()
