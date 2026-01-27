package typeconv

import (
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

func TestBuildAssertParams_CamelCase(t *testing.T) {
	// Test that camelCase field names work (used by CLI/UI)
	data := map[string]any{
		"selector":       "a",
		"assertMode":     "attribute_contains",
		"attributeName":  "href",
		"expectedValue":  "iana",
		"caseSensitive":  true,
		"failureMessage": "Link should contain 'iana'",
	}

	p := BuildAssertParams(data)

	if p.Selector != "a" {
		t.Errorf("expected selector 'a', got %q", p.Selector)
	}
	if p.Mode != basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS {
		t.Errorf("expected mode ASSERTION_MODE_ATTRIBUTE_CONTAINS, got %v", p.Mode)
	}
	if p.AttributeName == nil || *p.AttributeName != "href" {
		t.Errorf("expected attributeName 'href', got %v", p.AttributeName)
	}
	if p.Expected == nil || p.Expected.GetStringValue() != "iana" {
		t.Errorf("expected stringValue 'iana', got %v", p.Expected)
	}
	if p.CaseSensitive == nil || !*p.CaseSensitive {
		t.Errorf("expected caseSensitive true, got %v", p.CaseSensitive)
	}
	if p.FailureMessage == nil || *p.FailureMessage != "Link should contain 'iana'" {
		t.Errorf("expected failureMessage, got %v", p.FailureMessage)
	}
}

func TestBuildAssertParams_SnakeCase(t *testing.T) {
	// Test that snake_case field names work (used by proto/compiler with UseProtoNames: true)
	data := map[string]any{
		"selector":        "a",
		"assert_mode":     "attribute_contains",
		"attribute_name":  "href",
		"expected_value":  "iana",
		"case_sensitive":  true,
		"failure_message": "Link should contain 'iana'",
	}

	p := BuildAssertParams(data)

	if p.Selector != "a" {
		t.Errorf("expected selector 'a', got %q", p.Selector)
	}
	if p.Mode != basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS {
		t.Errorf("expected mode ASSERTION_MODE_ATTRIBUTE_CONTAINS, got %v", p.Mode)
	}
	if p.AttributeName == nil || *p.AttributeName != "href" {
		t.Errorf("expected attributeName 'href', got %v", p.AttributeName)
	}
	if p.Expected == nil || p.Expected.GetStringValue() != "iana" {
		t.Errorf("expected stringValue 'iana', got %v", p.Expected)
	}
	if p.CaseSensitive == nil || !*p.CaseSensitive {
		t.Errorf("expected caseSensitive true, got %v", p.CaseSensitive)
	}
	if p.FailureMessage == nil || *p.FailureMessage != "Link should contain 'iana'" {
		t.Errorf("expected failureMessage, got %v", p.FailureMessage)
	}
}

func TestBuildAssertParams_ModeAliases(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		expected basbase.AssertionMode
	}{
		{
			name:     "mode field",
			data:     map[string]any{"selector": "a", "mode": "text_contains"},
			expected: basbase.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS,
		},
		{
			name:     "assertMode alias",
			data:     map[string]any{"selector": "a", "assertMode": "text_equals"},
			expected: basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS,
		},
		{
			name:     "assert_mode snake_case",
			data:     map[string]any{"selector": "a", "assert_mode": "attribute_equals"},
			expected: basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildAssertParams(tt.data)
			if p.Mode != tt.expected {
				t.Errorf("expected mode %v, got %v", tt.expected, p.Mode)
			}
		})
	}
}

func TestBuildAssertParams_ExpectedAliases(t *testing.T) {
	tests := []struct {
		name          string
		data          map[string]any
		expectedValue string
	}{
		{
			name:          "expected field",
			data:          map[string]any{"selector": "a", "expected": "hello"},
			expectedValue: "hello",
		},
		{
			name:          "expectedText alias",
			data:          map[string]any{"selector": "a", "expectedText": "world"},
			expectedValue: "world",
		},
		{
			name:          "expected_text snake_case",
			data:          map[string]any{"selector": "a", "expected_text": "foo"},
			expectedValue: "foo",
		},
		{
			name:          "expectedValue alias",
			data:          map[string]any{"selector": "a", "expectedValue": "bar"},
			expectedValue: "bar",
		},
		{
			name:          "expected_value snake_case",
			data:          map[string]any{"selector": "a", "expected_value": "baz"},
			expectedValue: "baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildAssertParams(tt.data)
			if p.Expected == nil {
				t.Fatal("expected value was nil")
			}
			if p.Expected.GetStringValue() != tt.expectedValue {
				t.Errorf("expected string value %q, got %q", tt.expectedValue, p.Expected.GetStringValue())
			}
		})
	}
}

func TestBuildAssertParams_AttributeNameAliases(t *testing.T) {
	tests := []struct {
		name       string
		data       map[string]any
		wantAttr   string
		wantExists bool
	}{
		{
			name:       "attributeName camelCase",
			data:       map[string]any{"selector": "a", "attributeName": "href"},
			wantAttr:   "href",
			wantExists: true,
		},
		{
			name:       "attribute_name snake_case",
			data:       map[string]any{"selector": "a", "attribute_name": "data-testid"},
			wantAttr:   "data-testid",
			wantExists: true,
		},
		{
			name:       "no attribute name",
			data:       map[string]any{"selector": "a"},
			wantAttr:   "",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildAssertParams(tt.data)
			if tt.wantExists {
				if p.AttributeName == nil {
					t.Fatal("expected attributeName to be set")
				}
				if *p.AttributeName != tt.wantAttr {
					t.Errorf("expected attributeName %q, got %q", tt.wantAttr, *p.AttributeName)
				}
			} else {
				if p.AttributeName != nil {
					t.Errorf("expected attributeName to be nil, got %q", *p.AttributeName)
				}
			}
		})
	}
}

func TestBuildNavigateParams_SnakeCaseAliases(t *testing.T) {
	// Test that snake_case aliases work for navigate params
	data := map[string]any{
		"url":              "https://example.com",
		"wait_for_selector": "#loaded",
		"timeout_ms":       5000,
	}

	p := BuildNavigateParams(data)

	if p.Url != "https://example.com" {
		t.Errorf("expected url 'https://example.com', got %q", p.Url)
	}
	// wait_for_selector should be supported (protojson snake_case)
	if p.WaitForSelector != nil && *p.WaitForSelector != "#loaded" {
		t.Errorf("expected waitForSelector '#loaded', got %v", p.WaitForSelector)
	}
}

func TestBuildClickParams_Basic(t *testing.T) {
	data := map[string]any{
		"selector":   "#button",
		"button":     "left",
		"clickCount": 2,
	}

	p := BuildClickParams(data)

	if p.Selector != "#button" {
		t.Errorf("expected selector '#button', got %q", p.Selector)
	}
	if p.Button == nil || *p.Button != basactions.MouseButton_MOUSE_BUTTON_LEFT {
		t.Errorf("expected button LEFT, got %v", p.Button)
	}
	if p.ClickCount == nil || *p.ClickCount != 2 {
		t.Errorf("expected clickCount 2, got %v", p.ClickCount)
	}
}

func TestBuildInputParams_TextAlias(t *testing.T) {
	// Test that "text" is aliased to "value"
	data := map[string]any{
		"selector": "#email",
		"text":     "user@example.com",
	}

	p := BuildInputParams(data)

	if p.Selector != "#email" {
		t.Errorf("expected selector '#email', got %q", p.Selector)
	}
	if p.Value != "user@example.com" {
		t.Errorf("expected value 'user@example.com', got %q", p.Value)
	}
}

func TestBuildWaitParams_DurationAlias(t *testing.T) {
	// Test that "duration" is aliased to "durationMs"
	data := map[string]any{
		"duration": 1000,
	}

	p := BuildWaitParams(data)

	waitFor := p.GetDurationMs()
	if waitFor != 1000 {
		t.Errorf("expected durationMs 1000, got %d", waitFor)
	}
}

func TestBuildExtractParams_Basic(t *testing.T) {
	data := map[string]any{
		"selector": "#price",
	}

	p := BuildExtractParams(data)

	if p.Selector != "#price" {
		t.Errorf("expected selector '#price', got %q", p.Selector)
	}
}

func TestBuildExtractParams_WithAttribute(t *testing.T) {
	data := map[string]any{
		"selector":  "a.download-link",
		"attribute": "href",
		"outputKey": "downloadUrl",
	}

	p := BuildExtractParams(data)

	if p.Selector != "a.download-link" {
		t.Errorf("expected selector 'a.download-link', got %q", p.Selector)
	}
	if p.AttributeName == nil || *p.AttributeName != "href" {
		t.Errorf("expected attributeName 'href', got %v", p.AttributeName)
	}
	if p.StoreAs == nil || *p.StoreAs != "downloadUrl" {
		t.Errorf("expected storeAs 'downloadUrl', got %v", p.StoreAs)
	}
	if p.ExtractType == nil || *p.ExtractType != basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE {
		t.Errorf("expected extractType ATTRIBUTE, got %v", p.ExtractType)
	}
}

func TestBuildExtractParams_AttributeNameAlias(t *testing.T) {
	// Test that attributeName (camelCase) works
	data := map[string]any{
		"selector":      "#link",
		"attributeName": "data-id",
	}

	p := BuildExtractParams(data)

	if p.AttributeName == nil || *p.AttributeName != "data-id" {
		t.Errorf("expected attributeName 'data-id', got %v", p.AttributeName)
	}
	if p.ExtractType == nil || *p.ExtractType != basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE {
		t.Errorf("expected extractType ATTRIBUTE, got %v", p.ExtractType)
	}
}

func TestBuildExtractParams_StoreAsAliases(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		expected string
	}{
		{
			name:     "outputKey alias",
			data:     map[string]any{"selector": "#el", "outputKey": "value1"},
			expected: "value1",
		},
		{
			name:     "storeAs direct",
			data:     map[string]any{"selector": "#el", "storeAs": "value2"},
			expected: "value2",
		},
		{
			name:     "store_as snake_case",
			data:     map[string]any{"selector": "#el", "store_as": "value3"},
			expected: "value3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := BuildExtractParams(tt.data)
			if p.StoreAs == nil || *p.StoreAs != tt.expected {
				t.Errorf("expected storeAs %q, got %v", tt.expected, p.StoreAs)
			}
		})
	}
}

func TestBuildShortcutParams_Basic(t *testing.T) {
	data := map[string]any{
		"keys": "Control+a",
	}

	p := BuildShortcutParams(data)

	if p.Shortcut != "Control+a" {
		t.Errorf("expected shortcut 'Control+a', got %q", p.Shortcut)
	}
}

func TestBuildShortcutParams_ShortcutAlias(t *testing.T) {
	// Test that "shortcut" key works directly
	data := map[string]any{
		"shortcut": "Meta+Shift+s",
	}

	p := BuildShortcutParams(data)

	if p.Shortcut != "Meta+Shift+s" {
		t.Errorf("expected shortcut 'Meta+Shift+s', got %q", p.Shortcut)
	}
}

func TestBuildShortcutParams_WithSelector(t *testing.T) {
	data := map[string]any{
		"keys":     "Control+c",
		"selector": "#editor",
	}

	p := BuildShortcutParams(data)

	if p.Shortcut != "Control+c" {
		t.Errorf("expected shortcut 'Control+c', got %q", p.Shortcut)
	}
	if p.Selector == nil || *p.Selector != "#editor" {
		t.Errorf("expected selector '#editor', got %v", p.Selector)
	}
}
