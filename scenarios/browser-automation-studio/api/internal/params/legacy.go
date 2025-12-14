// Package params provides parameter builder functions for converting
// map-based data to typed proto parameter messages.
//
// # Legacy Field Mappings
//
// This package supports backward compatibility with older workflow formats
// by accepting multiple field names for the same logical parameter. The
// canonical field names are listed first, with legacy aliases following.
//
// ## Action Type Mappings (handled by typeconv.StringToActionType)
//
// Canonical names map to ActionType enum values:
//
//	navigate -> ACTION_TYPE_NAVIGATE (aliases: goto)
//	click    -> ACTION_TYPE_CLICK
//	input    -> ACTION_TYPE_INPUT (aliases: type, fill)
//	wait     -> ACTION_TYPE_WAIT
//	assert   -> ACTION_TYPE_ASSERT
//	scroll   -> ACTION_TYPE_SCROLL
//	select   -> ACTION_TYPE_SELECT (aliases: selectoption)
//	evaluate -> ACTION_TYPE_EVALUATE (aliases: eval)
//	keyboard -> ACTION_TYPE_KEYBOARD (aliases: keypress, press)
//	hover    -> ACTION_TYPE_HOVER
//	screenshot -> ACTION_TYPE_SCREENSHOT
//	focus    -> ACTION_TYPE_FOCUS
//	blur     -> ACTION_TYPE_BLUR
//
// ## Parameter Field Mappings
//
// ### InputParams
//
//	value (canonical) - The text to input
//	text (legacy)     - Alias for value
//
// ### WaitParams
//
//	durationMs (canonical) - Duration to wait in milliseconds
//	duration (legacy)      - Alias for durationMs
//
// ### AssertParams
//
//	mode (canonical)      - Assertion mode (e.g., "visible", "contains")
//	assertMode (legacy)   - Alias for mode
//
// ## V1 Workflow Node Data Field Mappings
//
// When parsing V1 workflow files, the following field names are checked
// in the node's data map:
//
// ### NavigateParams
//
//	url           - Target URL
//	waitForSelector - Selector to wait for after navigation
//	timeoutMs     - Navigation timeout
//	waitUntil     - Page load event to wait for
//
// ### ClickParams
//
//	selector   - Element selector
//	button     - Mouse button (left, right, middle)
//	clickCount - Number of clicks
//	delayMs    - Delay between multiple clicks
//	modifiers  - Array of modifier keys
//	force      - Force click even if element not actionable
//
// ### InputParams
//
//	selector    - Element selector
//	value/text  - Input text (text is legacy alias)
//	isSensitive - Mark as sensitive (for password fields)
//	submit      - Submit form after input
//	clearFirst  - Clear existing value before input
//	delayMs     - Typing delay
//
// ### WaitParams
//
//	durationMs/duration - Wait duration (duration is legacy alias)
//	selector            - Element to wait for
//	state               - Element state to wait for
//	timeoutMs           - Wait timeout
//
// ### AssertParams
//
//	selector       - Element selector
//	mode/assertMode - Assertion mode (assertMode is legacy alias)
//	expected       - Expected value
//	negated        - Negate the assertion
//	caseSensitive  - Case-sensitive string comparison
//	attributeName  - Attribute to assert on
//	failureMessage - Custom failure message
//
// ### ScrollParams
//
//	selector - Element to scroll (optional, defaults to viewport)
//	x, y     - Scroll position
//	deltaX, deltaY - Scroll delta
//	behavior - Scroll behavior (auto, smooth)
//
// ### SelectParams
//
//	selector - Select element
//	value    - Option value to select
//	label    - Option label to select (if value not set)
//	index    - Option index to select (if value and label not set)
//	timeoutMs - Selection timeout
//
// ### EvaluateParams
//
//	expression  - JavaScript expression to evaluate
//	storeResult - Variable name to store result
//
// ### KeyboardParams
//
//	key       - Single key to press
//	keys      - Array of keys for sequences
//	modifiers - Modifier keys (Ctrl, Alt, Shift, Meta)
//	action    - Key action (press, down, up)
//
// ### HoverParams
//
//	selector  - Element to hover
//	timeoutMs - Hover timeout
//
// ### ScreenshotParams
//
//	fullPage - Capture full page
//	selector - Element to capture
//	quality  - Image quality (for JPEG)
//
// ### FocusParams
//
//	selector  - Element to focus
//	scroll    - Scroll element into view
//	timeoutMs - Focus timeout
//
// ### BlurParams
//
//	selector  - Element to blur
//	timeoutMs - Blur timeout
package params
