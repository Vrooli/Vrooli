package examples_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// EnumParityFixture contains the canonical string mappings for BAS enums.
// This fixture is used by both Go and TypeScript tests to ensure parity.
type EnumParityFixture struct {
	ActionType       map[string]string `json:"actionType"`
	MouseButton      map[string]string `json:"mouseButton"`
	KeyboardModifier map[string]string `json:"keyboardModifier"`
	NavigateWaitEvent map[string]string `json:"navigateWaitEvent"`
	WaitState        map[string]string `json:"waitState"`
	AssertionMode    map[string]string `json:"assertionMode"`
	ScrollBehavior   map[string]string `json:"scrollBehavior"`
	KeyAction        map[string]string `json:"keyAction"`
	ExtractType      map[string]string `json:"extractType"`
	FrameSwitchAction map[string]string `json:"frameSwitchAction"`
	TabSwitchAction  map[string]string `json:"tabSwitchAction"`
	CookieOperation  map[string]string `json:"cookieOperation"`
	StorageType      map[string]string `json:"storageType"`
	CookieSameSite   map[string]string `json:"cookieSameSite"`
	GestureType      map[string]string `json:"gestureType"`
	SwipeDirection   map[string]string `json:"swipeDirection"`
	NetworkMockOperation map[string]string `json:"networkMockOperation"`
	DeviceOrientation map[string]string `json:"deviceOrientation"`
}

// actionTypeToString converts ActionType enum to handler dispatch string.
// These strings MUST match the TypeScript actionTypeToString function.
func actionTypeToString(t basactions.ActionType) string {
	switch t {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basactions.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basactions.ActionType_ACTION_TYPE_INPUT:
		return "input"
	case basactions.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case basactions.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		return "keyboard"
	case basactions.ActionType_ACTION_TYPE_HOVER:
		return "hover"
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		return "focus"
	case basactions.ActionType_ACTION_TYPE_BLUR:
		return "blur"
	case basactions.ActionType_ACTION_TYPE_SUBFLOW:
		return "subflow"
	case basactions.ActionType_ACTION_TYPE_EXTRACT:
		return "extract"
	case basactions.ActionType_ACTION_TYPE_UPLOAD_FILE:
		return "uploadfile"
	case basactions.ActionType_ACTION_TYPE_DOWNLOAD:
		return "download"
	case basactions.ActionType_ACTION_TYPE_FRAME_SWITCH:
		return "frameswitch"
	case basactions.ActionType_ACTION_TYPE_TAB_SWITCH:
		return "tabswitch"
	case basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE:
		return "cookiestorage"
	case basactions.ActionType_ACTION_TYPE_SHORTCUT:
		return "shortcut"
	case basactions.ActionType_ACTION_TYPE_DRAG_DROP:
		return "dragdrop"
	case basactions.ActionType_ACTION_TYPE_GESTURE:
		return "gesture"
	case basactions.ActionType_ACTION_TYPE_NETWORK_MOCK:
		return "networkmock"
	case basactions.ActionType_ACTION_TYPE_ROTATE:
		return "rotate"
	default:
		return "unknown"
	}
}

func mouseButtonToString(b basactions.MouseButton) string {
	switch b {
	case basactions.MouseButton_MOUSE_BUTTON_LEFT:
		return "left"
	case basactions.MouseButton_MOUSE_BUTTON_RIGHT:
		return "right"
	case basactions.MouseButton_MOUSE_BUTTON_MIDDLE:
		return "middle"
	default:
		return ""
	}
}

func keyboardModifierToString(m basactions.KeyboardModifier) string {
	switch m {
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL:
		return "ctrl"
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT:
		return "shift"
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_ALT:
		return "alt"
	case basactions.KeyboardModifier_KEYBOARD_MODIFIER_META:
		return "meta"
	default:
		return ""
	}
}

func navigateWaitEventToString(e basactions.NavigateWaitEvent) string {
	switch e {
	case basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_LOAD:
		return "load"
	case basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED:
		return "domcontentloaded"
	case basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE:
		return "networkidle"
	default:
		return ""
	}
}

func waitStateToString(s basactions.WaitState) string {
	switch s {
	case basactions.WaitState_WAIT_STATE_ATTACHED:
		return "attached"
	case basactions.WaitState_WAIT_STATE_DETACHED:
		return "detached"
	case basactions.WaitState_WAIT_STATE_VISIBLE:
		return "visible"
	case basactions.WaitState_WAIT_STATE_HIDDEN:
		return "hidden"
	default:
		return ""
	}
}

func assertionModeToString(m basbase.AssertionMode) string {
	switch m {
	case basbase.AssertionMode_ASSERTION_MODE_EXISTS:
		return "exists"
	case basbase.AssertionMode_ASSERTION_MODE_NOT_EXISTS:
		return "notexists"
	case basbase.AssertionMode_ASSERTION_MODE_VISIBLE:
		return "visible"
	case basbase.AssertionMode_ASSERTION_MODE_HIDDEN:
		return "hidden"
	case basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS:
		return "text_equals"
	case basbase.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS:
		return "text_contains"
	case basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS:
		return "attribute_equals"
	case basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS:
		return "attribute_contains"
	default:
		return "exists"
	}
}

func scrollBehaviorToString(b basactions.ScrollBehavior) string {
	switch b {
	case basactions.ScrollBehavior_SCROLL_BEHAVIOR_AUTO:
		return "auto"
	case basactions.ScrollBehavior_SCROLL_BEHAVIOR_SMOOTH:
		return "smooth"
	default:
		return ""
	}
}

func keyActionToString(a basactions.KeyAction) string {
	switch a {
	case basactions.KeyAction_KEY_ACTION_PRESS:
		return "press"
	case basactions.KeyAction_KEY_ACTION_DOWN:
		return "down"
	case basactions.KeyAction_KEY_ACTION_UP:
		return "up"
	default:
		return ""
	}
}

func extractTypeToString(t basactions.ExtractType) string {
	switch t {
	case basactions.ExtractType_EXTRACT_TYPE_TEXT:
		return "text"
	case basactions.ExtractType_EXTRACT_TYPE_INNER_HTML:
		return "innerHTML"
	case basactions.ExtractType_EXTRACT_TYPE_OUTER_HTML:
		return "outerHTML"
	case basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE:
		return "attribute"
	case basactions.ExtractType_EXTRACT_TYPE_PROPERTY:
		return "property"
	case basactions.ExtractType_EXTRACT_TYPE_VALUE:
		return "value"
	default:
		return "text"
	}
}

func frameSwitchActionToString(a basactions.FrameSwitchAction) string {
	switch a {
	case basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_ENTER:
		return "enter"
	case basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_PARENT:
		return "parent"
	case basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_EXIT:
		return "exit"
	default:
		return "enter"
	}
}

func tabSwitchActionToString(a basactions.TabSwitchAction) string {
	switch a {
	case basactions.TabSwitchAction_TAB_SWITCH_ACTION_OPEN:
		return "open"
	case basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH:
		return "switch"
	case basactions.TabSwitchAction_TAB_SWITCH_ACTION_CLOSE:
		return "close"
	case basactions.TabSwitchAction_TAB_SWITCH_ACTION_LIST:
		return "list"
	default:
		return "switch"
	}
}

func cookieOperationToString(o basactions.CookieOperation) string {
	switch o {
	case basactions.CookieOperation_COOKIE_OPERATION_GET:
		return "get"
	case basactions.CookieOperation_COOKIE_OPERATION_SET:
		return "set"
	case basactions.CookieOperation_COOKIE_OPERATION_DELETE:
		return "delete"
	case basactions.CookieOperation_COOKIE_OPERATION_CLEAR:
		return "clear"
	default:
		return "get"
	}
}

func storageTypeToString(t basactions.StorageType) string {
	switch t {
	case basactions.StorageType_STORAGE_TYPE_COOKIE:
		return "cookie"
	case basactions.StorageType_STORAGE_TYPE_LOCAL_STORAGE:
		return "localStorage"
	case basactions.StorageType_STORAGE_TYPE_SESSION_STORAGE:
		return "sessionStorage"
	default:
		return "cookie"
	}
}

func cookieSameSiteToString(s basactions.CookieSameSite) string {
	switch s {
	case basactions.CookieSameSite_COOKIE_SAME_SITE_STRICT:
		return "Strict"
	case basactions.CookieSameSite_COOKIE_SAME_SITE_LAX:
		return "Lax"
	case basactions.CookieSameSite_COOKIE_SAME_SITE_NONE:
		return "None"
	default:
		return ""
	}
}

func gestureTypeToString(t basactions.GestureType) string {
	switch t {
	case basactions.GestureType_GESTURE_TYPE_SWIPE:
		return "swipe"
	case basactions.GestureType_GESTURE_TYPE_PINCH:
		return "pinch"
	case basactions.GestureType_GESTURE_TYPE_ZOOM:
		return "zoom"
	case basactions.GestureType_GESTURE_TYPE_LONG_PRESS:
		return "longPress"
	case basactions.GestureType_GESTURE_TYPE_DOUBLE_TAP:
		return "doubleTap"
	default:
		return "swipe"
	}
}

func swipeDirectionToString(d basactions.SwipeDirection) string {
	switch d {
	case basactions.SwipeDirection_SWIPE_DIRECTION_UP:
		return "up"
	case basactions.SwipeDirection_SWIPE_DIRECTION_DOWN:
		return "down"
	case basactions.SwipeDirection_SWIPE_DIRECTION_LEFT:
		return "left"
	case basactions.SwipeDirection_SWIPE_DIRECTION_RIGHT:
		return "right"
	default:
		return ""
	}
}

func networkMockOperationToString(o basactions.NetworkMockOperation) string {
	switch o {
	case basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_MOCK:
		return "mock"
	case basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_BLOCK:
		return "block"
	case basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_MODIFY_REQUEST:
		return "modifyRequest"
	case basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_MODIFY_RESPONSE:
		return "modifyResponse"
	case basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_CLEAR:
		return "clear"
	default:
		return "mock"
	}
}

func deviceOrientationToString(o basactions.DeviceOrientation) string {
	switch o {
	case basactions.DeviceOrientation_DEVICE_ORIENTATION_PORTRAIT:
		return "portrait"
	case basactions.DeviceOrientation_DEVICE_ORIENTATION_LANDSCAPE:
		return "landscape"
	default:
		return "portrait"
	}
}

// TestGenerateEnumParityFixture generates the enum parity fixture JSON file.
// This test writes the fixture that TypeScript tests use to verify parity.
func TestGenerateEnumParityFixture(t *testing.T) {
	fixture := EnumParityFixture{
		ActionType: map[string]string{
			"NAVIGATE":       actionTypeToString(basactions.ActionType_ACTION_TYPE_NAVIGATE),
			"CLICK":          actionTypeToString(basactions.ActionType_ACTION_TYPE_CLICK),
			"INPUT":          actionTypeToString(basactions.ActionType_ACTION_TYPE_INPUT),
			"WAIT":           actionTypeToString(basactions.ActionType_ACTION_TYPE_WAIT),
			"ASSERT":         actionTypeToString(basactions.ActionType_ACTION_TYPE_ASSERT),
			"SCROLL":         actionTypeToString(basactions.ActionType_ACTION_TYPE_SCROLL),
			"SELECT":         actionTypeToString(basactions.ActionType_ACTION_TYPE_SELECT),
			"EVALUATE":       actionTypeToString(basactions.ActionType_ACTION_TYPE_EVALUATE),
			"KEYBOARD":       actionTypeToString(basactions.ActionType_ACTION_TYPE_KEYBOARD),
			"HOVER":          actionTypeToString(basactions.ActionType_ACTION_TYPE_HOVER),
			"SCREENSHOT":     actionTypeToString(basactions.ActionType_ACTION_TYPE_SCREENSHOT),
			"FOCUS":          actionTypeToString(basactions.ActionType_ACTION_TYPE_FOCUS),
			"BLUR":           actionTypeToString(basactions.ActionType_ACTION_TYPE_BLUR),
			"SUBFLOW":        actionTypeToString(basactions.ActionType_ACTION_TYPE_SUBFLOW),
			"EXTRACT":        actionTypeToString(basactions.ActionType_ACTION_TYPE_EXTRACT),
			"UPLOAD_FILE":    actionTypeToString(basactions.ActionType_ACTION_TYPE_UPLOAD_FILE),
			"DOWNLOAD":       actionTypeToString(basactions.ActionType_ACTION_TYPE_DOWNLOAD),
			"FRAME_SWITCH":   actionTypeToString(basactions.ActionType_ACTION_TYPE_FRAME_SWITCH),
			"TAB_SWITCH":     actionTypeToString(basactions.ActionType_ACTION_TYPE_TAB_SWITCH),
			"COOKIE_STORAGE": actionTypeToString(basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE),
			"SHORTCUT":       actionTypeToString(basactions.ActionType_ACTION_TYPE_SHORTCUT),
			"DRAG_DROP":      actionTypeToString(basactions.ActionType_ACTION_TYPE_DRAG_DROP),
			"GESTURE":        actionTypeToString(basactions.ActionType_ACTION_TYPE_GESTURE),
			"NETWORK_MOCK":   actionTypeToString(basactions.ActionType_ACTION_TYPE_NETWORK_MOCK),
			"ROTATE":         actionTypeToString(basactions.ActionType_ACTION_TYPE_ROTATE),
		},
		MouseButton: map[string]string{
			"LEFT":   mouseButtonToString(basactions.MouseButton_MOUSE_BUTTON_LEFT),
			"RIGHT":  mouseButtonToString(basactions.MouseButton_MOUSE_BUTTON_RIGHT),
			"MIDDLE": mouseButtonToString(basactions.MouseButton_MOUSE_BUTTON_MIDDLE),
		},
		KeyboardModifier: map[string]string{
			"CTRL":  keyboardModifierToString(basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL),
			"SHIFT": keyboardModifierToString(basactions.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT),
			"ALT":   keyboardModifierToString(basactions.KeyboardModifier_KEYBOARD_MODIFIER_ALT),
			"META":  keyboardModifierToString(basactions.KeyboardModifier_KEYBOARD_MODIFIER_META),
		},
		NavigateWaitEvent: map[string]string{
			"LOAD":             navigateWaitEventToString(basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_LOAD),
			"DOMCONTENTLOADED": navigateWaitEventToString(basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED),
			"NETWORKIDLE":      navigateWaitEventToString(basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE),
		},
		WaitState: map[string]string{
			"ATTACHED": waitStateToString(basactions.WaitState_WAIT_STATE_ATTACHED),
			"DETACHED": waitStateToString(basactions.WaitState_WAIT_STATE_DETACHED),
			"VISIBLE":  waitStateToString(basactions.WaitState_WAIT_STATE_VISIBLE),
			"HIDDEN":   waitStateToString(basactions.WaitState_WAIT_STATE_HIDDEN),
		},
		AssertionMode: map[string]string{
			"EXISTS":             assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_EXISTS),
			"NOT_EXISTS":         assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_NOT_EXISTS),
			"VISIBLE":            assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_VISIBLE),
			"HIDDEN":             assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_HIDDEN),
			"TEXT_EQUALS":        assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS),
			"TEXT_CONTAINS":      assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS),
			"ATTRIBUTE_EQUALS":   assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS),
			"ATTRIBUTE_CONTAINS": assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS),
		},
		ScrollBehavior: map[string]string{
			"AUTO":   scrollBehaviorToString(basactions.ScrollBehavior_SCROLL_BEHAVIOR_AUTO),
			"SMOOTH": scrollBehaviorToString(basactions.ScrollBehavior_SCROLL_BEHAVIOR_SMOOTH),
		},
		KeyAction: map[string]string{
			"PRESS": keyActionToString(basactions.KeyAction_KEY_ACTION_PRESS),
			"DOWN":  keyActionToString(basactions.KeyAction_KEY_ACTION_DOWN),
			"UP":    keyActionToString(basactions.KeyAction_KEY_ACTION_UP),
		},
		ExtractType: map[string]string{
			"TEXT":       extractTypeToString(basactions.ExtractType_EXTRACT_TYPE_TEXT),
			"INNER_HTML": extractTypeToString(basactions.ExtractType_EXTRACT_TYPE_INNER_HTML),
			"OUTER_HTML": extractTypeToString(basactions.ExtractType_EXTRACT_TYPE_OUTER_HTML),
			"ATTRIBUTE":  extractTypeToString(basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE),
			"PROPERTY":   extractTypeToString(basactions.ExtractType_EXTRACT_TYPE_PROPERTY),
			"VALUE":      extractTypeToString(basactions.ExtractType_EXTRACT_TYPE_VALUE),
		},
		FrameSwitchAction: map[string]string{
			"ENTER":  frameSwitchActionToString(basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_ENTER),
			"PARENT": frameSwitchActionToString(basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_PARENT),
			"EXIT":   frameSwitchActionToString(basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_EXIT),
		},
		TabSwitchAction: map[string]string{
			"OPEN":   tabSwitchActionToString(basactions.TabSwitchAction_TAB_SWITCH_ACTION_OPEN),
			"SWITCH": tabSwitchActionToString(basactions.TabSwitchAction_TAB_SWITCH_ACTION_SWITCH),
			"CLOSE":  tabSwitchActionToString(basactions.TabSwitchAction_TAB_SWITCH_ACTION_CLOSE),
			"LIST":   tabSwitchActionToString(basactions.TabSwitchAction_TAB_SWITCH_ACTION_LIST),
		},
		CookieOperation: map[string]string{
			"GET":    cookieOperationToString(basactions.CookieOperation_COOKIE_OPERATION_GET),
			"SET":    cookieOperationToString(basactions.CookieOperation_COOKIE_OPERATION_SET),
			"DELETE": cookieOperationToString(basactions.CookieOperation_COOKIE_OPERATION_DELETE),
			"CLEAR":  cookieOperationToString(basactions.CookieOperation_COOKIE_OPERATION_CLEAR),
		},
		StorageType: map[string]string{
			"COOKIE":          storageTypeToString(basactions.StorageType_STORAGE_TYPE_COOKIE),
			"LOCAL_STORAGE":   storageTypeToString(basactions.StorageType_STORAGE_TYPE_LOCAL_STORAGE),
			"SESSION_STORAGE": storageTypeToString(basactions.StorageType_STORAGE_TYPE_SESSION_STORAGE),
		},
		CookieSameSite: map[string]string{
			"STRICT": cookieSameSiteToString(basactions.CookieSameSite_COOKIE_SAME_SITE_STRICT),
			"LAX":    cookieSameSiteToString(basactions.CookieSameSite_COOKIE_SAME_SITE_LAX),
			"NONE":   cookieSameSiteToString(basactions.CookieSameSite_COOKIE_SAME_SITE_NONE),
		},
		GestureType: map[string]string{
			"SWIPE":      gestureTypeToString(basactions.GestureType_GESTURE_TYPE_SWIPE),
			"PINCH":      gestureTypeToString(basactions.GestureType_GESTURE_TYPE_PINCH),
			"ZOOM":       gestureTypeToString(basactions.GestureType_GESTURE_TYPE_ZOOM),
			"LONG_PRESS": gestureTypeToString(basactions.GestureType_GESTURE_TYPE_LONG_PRESS),
			"DOUBLE_TAP": gestureTypeToString(basactions.GestureType_GESTURE_TYPE_DOUBLE_TAP),
		},
		SwipeDirection: map[string]string{
			"UP":    swipeDirectionToString(basactions.SwipeDirection_SWIPE_DIRECTION_UP),
			"DOWN":  swipeDirectionToString(basactions.SwipeDirection_SWIPE_DIRECTION_DOWN),
			"LEFT":  swipeDirectionToString(basactions.SwipeDirection_SWIPE_DIRECTION_LEFT),
			"RIGHT": swipeDirectionToString(basactions.SwipeDirection_SWIPE_DIRECTION_RIGHT),
		},
		NetworkMockOperation: map[string]string{
			"MOCK":            networkMockOperationToString(basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_MOCK),
			"BLOCK":           networkMockOperationToString(basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_BLOCK),
			"MODIFY_REQUEST":  networkMockOperationToString(basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_MODIFY_REQUEST),
			"MODIFY_RESPONSE": networkMockOperationToString(basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_MODIFY_RESPONSE),
			"CLEAR":           networkMockOperationToString(basactions.NetworkMockOperation_NETWORK_MOCK_OPERATION_CLEAR),
		},
		DeviceOrientation: map[string]string{
			"PORTRAIT":  deviceOrientationToString(basactions.DeviceOrientation_DEVICE_ORIENTATION_PORTRAIT),
			"LANDSCAPE": deviceOrientationToString(basactions.DeviceOrientation_DEVICE_ORIENTATION_LANDSCAPE),
		},
	}

	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal fixture: %v", err)
	}

	// Write to fixture file that TS tests will read
	fixturePath := filepath.Join("testdata", "enum_parity_fixture.json")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("failed to create testdata dir: %v", err)
	}
	if err := os.WriteFile(fixturePath, data, 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	t.Logf("Generated enum parity fixture at %s", fixturePath)
}

// TestEnumParityConsistency verifies Go enum converters produce expected strings.
// This is a sanity check that the Go side is consistent.
func TestEnumParityConsistency(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		// ActionType samples
		{"ActionType.NAVIGATE", actionTypeToString(basactions.ActionType_ACTION_TYPE_NAVIGATE), "navigate"},
		{"ActionType.CLICK", actionTypeToString(basactions.ActionType_ACTION_TYPE_CLICK), "click"},
		{"ActionType.UPLOAD_FILE", actionTypeToString(basactions.ActionType_ACTION_TYPE_UPLOAD_FILE), "uploadfile"},
		{"ActionType.COOKIE_STORAGE", actionTypeToString(basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE), "cookiestorage"},

		// MouseButton samples
		{"MouseButton.LEFT", mouseButtonToString(basactions.MouseButton_MOUSE_BUTTON_LEFT), "left"},

		// AssertionMode samples
		{"AssertionMode.TEXT_EQUALS", assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS), "text_equals"},
		{"AssertionMode.NOT_EXISTS", assertionModeToString(basbase.AssertionMode_ASSERTION_MODE_NOT_EXISTS), "notexists"},

		// StorageType samples
		{"StorageType.LOCAL_STORAGE", storageTypeToString(basactions.StorageType_STORAGE_TYPE_LOCAL_STORAGE), "localStorage"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("got %q, expected %q", tc.got, tc.expected)
			}
		})
	}
}
