package compiler

// StepType represents a supported workflow action.
type StepType string

const (
	StepNavigate     StepType = "navigate"
	StepClick        StepType = "click"
	StepHover        StepType = "hover"
	StepDragDrop     StepType = "dragDrop"
	StepFocus        StepType = "focus"
	StepBlur         StepType = "blur"
	StepScroll       StepType = "scroll"
	StepSelect       StepType = "select"
	StepRotate       StepType = "rotate"
	StepGesture      StepType = "gesture"
	StepUploadFile   StepType = "uploadFile"
	StepTypeInput    StepType = "type"
	StepShortcut     StepType = "shortcut"
	StepKeyboard     StepType = "keyboard"
	StepWait         StepType = "wait"
	StepScreenshot   StepType = "screenshot"
	StepExtract      StepType = "extract"
	StepEvaluate     StepType = "evaluate"
	StepAssert       StepType = "assert"
	StepCustom       StepType = "custom"
	StepSetVariable  StepType = "setVariable"
	StepUseVariable  StepType = "useVariable"
	StepTabSwitch    StepType = "tabSwitch"
	StepFrameSwitch  StepType = "frameSwitch"
	StepConditional  StepType = "conditional"
	StepLoop         StepType = "loop"
	StepSetCookie    StepType = "setCookie"
	StepGetCookie    StepType = "getCookie"
	StepClearCookie  StepType = "clearCookie"
	StepSetStorage   StepType = "setStorage"
	StepGetStorage   StepType = "getStorage"
	StepClearStorage StepType = "clearStorage"
	StepNetworkMock  StepType = "networkMock"
)

const (
	LoopContinueTarget = "__loop_continue__"
	LoopBreakTarget    = "__loop_break__"
)

const (
	loopHandleBody     = "loopbody"
	loopHandleAfter    = "loopafter"
	loopHandleBreak    = "loopbreak"
	loopHandleContinue = "loopcontinue"

	loopConditionBody     = "loop_body"
	loopConditionAfter    = "loop_next"
	loopConditionBreak    = "loop_break"
	loopConditionContinue = "loop_continue"
)

var supportedStepTypes = map[StepType]struct{}{
	StepNavigate:     {},
	StepClick:        {},
	StepHover:        {},
	StepDragDrop:     {},
	StepFocus:        {},
	StepBlur:         {},
	StepTypeInput:    {},
	StepShortcut:     {},
	StepKeyboard:     {},
	StepScroll:       {},
	StepSelect:       {},
	StepRotate:       {},
	StepGesture:      {},
	StepUploadFile:   {},
	StepWait:         {},
	StepScreenshot:   {},
	StepExtract:      {},
	StepEvaluate:     {},
	StepAssert:       {},
	StepCustom:       {},
	StepSetVariable:  {},
	StepUseVariable:  {},
	StepTabSwitch:    {},
	StepFrameSwitch:  {},
	StepConditional:  {},
	StepLoop:         {},
	StepSetCookie:    {},
	StepGetCookie:    {},
	StepClearCookie:  {},
	StepSetStorage:   {},
	StepGetStorage:   {},
	StepClearStorage: {},
	StepNetworkMock:  {},
}
