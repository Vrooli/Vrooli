package services

import (
	"encoding/json"
	"testing"
	"time"
)

// =============================================================================
// Time and Duration Utilities Tests
// =============================================================================

func TestHumanizeDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "negative duration",
			duration: -5 * time.Second,
			expected: "5s",
		},
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "one minute",
			duration: 1 * time.Minute,
			expected: "1m",
		},
		{
			name:     "minutes and seconds (shows only minutes)",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m",
		},
		{
			name:     "one hour",
			duration: 1 * time.Hour,
			expected: "1h",
		},
		{
			name:     "hours and minutes",
			duration: 2*time.Hour + 15*time.Minute,
			expected: "2h 15m",
		},
		{
			name:     "hours, minutes, and seconds (shows only h and m)",
			duration: 3*time.Hour + 25*time.Minute + 45*time.Second,
			expected: "3h 25m",
		},
		{
			name:     "large duration (days worth)",
			duration: 48*time.Hour + 30*time.Minute,
			expected: "48h 30m",
		},
		{
			name:     "59 seconds",
			duration: 59 * time.Second,
			expected: "59s",
		},
		{
			name:     "60 seconds",
			duration: 60 * time.Second,
			expected: "1m",
		},
		{
			name:     "59 minutes 59 seconds",
			duration: 59*time.Minute + 59*time.Second,
			expected: "59m",
		},
		{
			name:     "fractional seconds (rounds down)",
			duration: 500 * time.Millisecond,
			expected: "0s",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := humanizeDuration(tc.duration)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFormatMillisTimestamp(t *testing.T) {
	testCases := []struct {
		name     string
		millis   int64
		expected string
	}{
		{
			name:     "zero timestamp",
			millis:   0,
			expected: "",
		},
		{
			name:     "negative timestamp",
			millis:   -1000,
			expected: "",
		},
		{
			name:     "valid timestamp",
			millis:   1609459200000, // 2021-01-01 00:00:00 UTC
			expected: "00:00:00.000",
		},
		{
			name:     "timestamp with milliseconds",
			millis:   1609459200123, // 2021-01-01 00:00:00.123 UTC
			expected: "00:00:00.123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatMillisTimestamp(tc.millis)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// =============================================================================
// String Utilities Tests
// =============================================================================

func TestNormalizeIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "already normalized",
			input:    "test-app",
			expected: "test-app",
		},
		{
			name:     "uppercase to lowercase",
			input:    "TEST-APP",
			expected: "test-app",
		},
		{
			name:     "mixed case",
			input:    "MyTestApp",
			expected: "mytestapp",
		},
		{
			name:     "with leading whitespace",
			input:    "  test-app",
			expected: "test-app",
		},
		{
			name:     "with trailing whitespace",
			input:    "test-app  ",
			expected: "test-app",
		},
		{
			name:     "with both leading and trailing whitespace",
			input:    "  test-app  ",
			expected: "test-app",
		},
		{
			name:     "with internal spaces",
			input:    "test app",
			expected: "test app",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeIdentifier(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestStringValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "nil pointer",
			input:    nil,
			expected: "",
		},
		{
			name:     "empty string",
			input:    stringPtr(""),
			expected: "",
		},
		{
			name:     "non-empty string",
			input:    stringPtr("test"),
			expected: "test",
		},
		{
			name:     "whitespace string",
			input:    stringPtr("  test  "),
			expected: "  test  ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stringValue(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTrimmedString(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string with whitespace",
			input:    "  test  ",
			expected: "test",
		},
		{
			name:     "integer",
			input:    42,
			expected: "42",
		},
		{
			name:     "nil",
			input:    nil,
			expected: "",
		},
		{
			name:     "byte slice",
			input:    []byte("  data  "),
			expected: "data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := trimmedString(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "all empty",
			values:   []string{"", "", ""},
			expected: "",
		},
		{
			name:     "first non-empty",
			values:   []string{"first", "second", "third"},
			expected: "first",
		},
		{
			name:     "skip empty strings",
			values:   []string{"", "second", "third"},
			expected: "second",
		},
		{
			name:     "skip whitespace-only strings",
			values:   []string{"  ", "  \t  ", "third"},
			expected: "third",
		},
		{
			name:     "no values",
			values:   []string{},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := firstNonEmpty(tc.values...)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		limit    int
		expected string
	}{
		{
			name:     "shorter than limit",
			input:    "short",
			limit:    10,
			expected: "short",
		},
		{
			name:     "exactly at limit",
			input:    "exactly",
			limit:    7,
			expected: "exactly",
		},
		{
			name:     "longer than limit",
			input:    "this is a very long string",
			limit:    10,
			expected: "this is...",
		},
		{
			name:     "limit too small for ellipsis",
			input:    "test",
			limit:    2,
			expected: "te",
		},
		{
			name:     "empty string",
			input:    "",
			limit:    10,
			expected: "",
		},
		{
			name:     "zero limit",
			input:    "test",
			limit:    0,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateString(tc.input, tc.limit)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestSanitizeCommandIdentifier(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "clean alphanumeric",
			input:    "test123",
			expected: "test123",
		},
		{
			name:     "with hyphens",
			input:    "test-app-name",
			expected: "testappname",
		},
		{
			name:     "with spaces",
			input:    "test app name",
			expected: "test_app_name",
		},
		{
			name:     "with special characters",
			input:    "test$app@name",
			expected: "testappname",
		},
		{
			name:     "with slashes",
			input:    "test/app/name",
			expected: "testappname",
		},
		{
			name:     "with dots",
			input:    "test.app.name",
			expected: "testappname",
		},
		{
			name:     "multiple spaces",
			input:    "test   app   name",
			expected: "test_app_name",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  test app  ",
			expected: "test_app",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeCommandIdentifier(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestParseOrEchoTimestamp(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid RFC3339 timestamp",
			input:    "2024-01-01T12:00:00Z",
			expected: "2024-01-01T12:00:00Z",
		},
		{
			name:     "invalid timestamp",
			input:    "not a timestamp",
			expected: "not a timestamp",
		},
		{
			name:     "partial timestamp",
			input:    "2024-01-01",
			expected: "2024-01-01",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseOrEchoTimestamp(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// =============================================================================
// Slice Utilities Tests
// =============================================================================

func TestUniqueStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with empty strings",
			input:    []string{"a", "", "b", "", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with whitespace",
			input:    []string{"a", "  ", "b", "  a  ", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all empty",
			input:    []string{"", "", ""},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := uniqueStrings(tc.input)
			if len(result) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(result))
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("At index %d: expected %q, got %q", i, tc.expected[i], v)
				}
			}
		})
	}
}

func TestDedupeStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "preserves order of first occurrence",
			input:    []string{"c", "a", "b", "a", "c"},
			expected: []string{"c", "a", "b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := dedupeStrings(tc.input)
			if (result == nil) != (tc.expected == nil) {
				t.Fatalf("Expected nil=%v, got nil=%v", tc.expected == nil, result == nil)
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(result))
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("At index %d: expected %q, got %q", i, tc.expected[i], v)
				}
			}
		})
	}
}

func TestFilterNonEmptyStrings(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "all non-empty",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "mixed empty and non-empty",
			input:    []string{"a", "", "b", "  ", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all empty",
			input:    []string{"", "  ", "\t"},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterNonEmptyStrings(tc.input)
			if (result == nil) != (tc.expected == nil) {
				t.Fatalf("Expected nil=%v, got nil=%v", tc.expected == nil, result == nil)
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(result))
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("At index %d: expected %q, got %q", i, tc.expected[i], v)
				}
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	testCases := []struct {
		name     string
		values   []string
		target   string
		expected bool
	}{
		{
			name:     "empty slice",
			values:   []string{},
			target:   "test",
			expected: false,
		},
		{
			name:     "found at beginning",
			values:   []string{"test", "other", "values"},
			target:   "test",
			expected: true,
		},
		{
			name:     "found in middle",
			values:   []string{"a", "test", "b"},
			target:   "test",
			expected: true,
		},
		{
			name:     "found at end",
			values:   []string{"a", "b", "test"},
			target:   "test",
			expected: true,
		},
		{
			name:     "not found",
			values:   []string{"a", "b", "c"},
			target:   "test",
			expected: false,
		},
		{
			name:     "case sensitive",
			values:   []string{"Test", "TEST", "TeSt"},
			target:   "test",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsString(tc.values, tc.target)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSanitizeCaptureClasses(t *testing.T) {
	testCases := []struct {
		name       string
		classes    []string
		maxEntries int
		maxLength  int
		expected   []string
	}{
		{
			name:       "empty slice",
			classes:    []string{},
			maxEntries: 5,
			maxLength:  50,
			expected:   nil,
		},
		{
			name:       "within limits",
			classes:    []string{"btn", "primary", "large"},
			maxEntries: 5,
			maxLength:  50,
			expected:   []string{"btn", "primary", "large"},
		},
		{
			name:       "exceeds max entries",
			classes:    []string{"a", "b", "c", "d", "e", "f"},
			maxEntries: 3,
			maxLength:  50,
			expected:   []string{"a", "b", "c"},
		},
		{
			name:       "exceeds max length",
			classes:    []string{"very-long-class-name-that-exceeds-limit"},
			maxEntries: 5,
			maxLength:  10,
			expected:   []string{"very-long-"},
		},
		{
			name:       "trims whitespace",
			classes:    []string{"  btn  ", "  primary  "},
			maxEntries: 5,
			maxLength:  50,
			expected:   []string{"btn", "primary"},
		},
		{
			name:       "filters empty after trim",
			classes:    []string{"btn", "  ", "primary"},
			maxEntries: 5,
			maxLength:  50,
			expected:   []string{"btn", "primary"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeCaptureClasses(tc.classes, tc.maxEntries, tc.maxLength)
			if (result == nil) != (tc.expected == nil) {
				t.Fatalf("Expected nil=%v, got nil=%v", tc.expected == nil, result == nil)
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(result))
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("At index %d: expected %q, got %q", i, tc.expected[i], v)
				}
			}
		})
	}
}

// =============================================================================
// Number Utilities Tests
// =============================================================================

func TestValueOrDefault(t *testing.T) {
	testCases := []struct {
		name     string
		value    *int
		fallback int
		expected int
	}{
		{
			name:     "nil value uses fallback",
			value:    nil,
			fallback: 42,
			expected: 42,
		},
		{
			name:     "non-nil value used",
			value:    intPtr(100),
			fallback: 42,
			expected: 100,
		},
		{
			name:     "zero value used (not fallback)",
			value:    intPtr(0),
			fallback: 42,
			expected: 0,
		},
		{
			name:     "negative value used",
			value:    intPtr(-10),
			fallback: 42,
			expected: -10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := valueOrDefault(tc.value, tc.fallback)
			if result != tc.expected {
				t.Errorf("Expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestAnyNumber(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected float64
	}{
		{
			name:     "float64",
			value:    3.14,
			expected: 3.14,
		},
		{
			name:     "float32",
			value:    float32(2.5),
			expected: 2.5,
		},
		{
			name:     "int",
			value:    42,
			expected: 42.0,
		},
		{
			name:     "int64",
			value:    int64(100),
			expected: 100.0,
		},
		{
			name:     "string number",
			value:    "123.45",
			expected: 123.45,
		},
		{
			name:     "json.Number",
			value:    json.Number("678.9"),
			expected: 678.9,
		},
		{
			name:     "nil",
			value:    nil,
			expected: 0,
		},
		{
			name:     "invalid string",
			value:    "not a number",
			expected: 0,
		},
		{
			name:     "boolean",
			value:    true,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := anyNumber(tc.value)
			if result != tc.expected {
				t.Errorf("Expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestPlural(t *testing.T) {
	testCases := []struct {
		name     string
		count    int
		expected string
	}{
		{
			name:     "zero",
			count:    0,
			expected: "s",
		},
		{
			name:     "one (singular)",
			count:    1,
			expected: "",
		},
		{
			name:     "two",
			count:    2,
			expected: "s",
		},
		{
			name:     "negative",
			count:    -1,
			expected: "s",
		},
		{
			name:     "large number",
			count:    1000,
			expected: "s",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := plural(tc.count)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// =============================================================================
// Port Resolution Utilities Tests
// =============================================================================

func TestParsePortValue(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected int
		ok       bool
	}{
		{
			name:     "valid int",
			value:    8080,
			expected: 8080,
			ok:       true,
		},
		{
			name:     "valid int32",
			value:    int32(3000),
			expected: 3000,
			ok:       true,
		},
		{
			name:     "valid int64",
			value:    int64(5432),
			expected: 5432,
			ok:       true,
		},
		{
			name:     "valid float64",
			value:    float64(8000),
			expected: 8000,
			ok:       true,
		},
		{
			name:     "valid string",
			value:    "9000",
			expected: 9000,
			ok:       true,
		},
		{
			name:     "json.Number",
			value:    json.Number("7000"),
			expected: 7000,
			ok:       true,
		},
		{
			name:     "zero int",
			value:    0,
			expected: 0,
			ok:       false,
		},
		{
			name:     "negative int",
			value:    -1,
			expected: 0,
			ok:       false,
		},
		{
			name:     "empty string",
			value:    "",
			expected: 0,
			ok:       false,
		},
		{
			name:     "whitespace string",
			value:    "  ",
			expected: 0,
			ok:       false,
		},
		{
			name:     "invalid string",
			value:    "not-a-port",
			expected: 0,
			ok:       false,
		},
		{
			name:     "nil",
			value:    nil,
			expected: 0,
			ok:       false,
		},
		{
			name:     "string with whitespace",
			value:    "  8080  ",
			expected: 8080,
			ok:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			port, ok := parsePortValue(tc.value)
			if ok != tc.ok {
				t.Errorf("Expected ok=%v, got ok=%v", tc.ok, ok)
			}
			if port != tc.expected {
				t.Errorf("Expected port=%d, got port=%d", tc.expected, port)
			}
		})
	}
}

func TestParsePortValueFromString(t *testing.T) {
	testCases := []struct {
		name      string
		value     string
		expected  int
		expectErr bool
	}{
		{
			name:      "valid port",
			value:     "8080",
			expected:  8080,
			expectErr: false,
		},
		{
			name:      "valid port with whitespace",
			value:     "  3000  ",
			expected:  3000,
			expectErr: false,
		},
		{
			name:      "empty string",
			value:     "",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "whitespace only",
			value:     "   ",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "invalid characters",
			value:     "abc",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "zero port",
			value:     "0",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "negative port",
			value:     "-1",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "port too large",
			value:     "70000",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "max valid port",
			value:     "65535",
			expected:  65535,
			expectErr: false,
		},
		{
			name:      "min valid port",
			value:     "1",
			expected:  1,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			port, err := parsePortValueFromString(tc.value)
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if port != tc.expected {
					t.Errorf("Expected port=%d, got port=%d", tc.expected, port)
				}
			}
		})
	}
}

func TestResolvePort(t *testing.T) {
	testCases := []struct {
		name          string
		portMappings  map[string]interface{}
		preferredKeys []string
		expected      int
	}{
		{
			name:          "empty mappings",
			portMappings:  map[string]interface{}{},
			preferredKeys: []string{"api", "ui"},
			expected:      0,
		},
		{
			name: "finds preferred key (case insensitive)",
			portMappings: map[string]interface{}{
				"API_PORT": 8080,
				"UI_PORT":  3000,
			},
			preferredKeys: []string{"api_port"},
			expected:      8080,
		},
		{
			name: "uses first preferred key match",
			portMappings: map[string]interface{}{
				"API_PORT": 8080,
				"UI_PORT":  3000,
			},
			preferredKeys: []string{"ui_port", "api_port"},
			expected:      3000,
		},
		{
			name: "fallback to any port if no preferred match",
			portMappings: map[string]interface{}{
				"OTHER_PORT": 5000,
			},
			preferredKeys: []string{"api", "ui"},
			expected:      5000,
		},
		{
			name: "skips invalid port values",
			portMappings: map[string]interface{}{
				"INVALID": "not-a-port",
				"VALID":   8080,
			},
			preferredKeys: []string{"invalid"},
			expected:      8080,
		},
		{
			name: "handles various port value types",
			portMappings: map[string]interface{}{
				"int_port":    8080,
				"string_port": "9000",
				"float_port":  float64(7000),
			},
			preferredKeys: []string{"string_port"},
			expected:      9000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolvePort(tc.portMappings, tc.preferredKeys)
			if result != tc.expected {
				t.Errorf("Expected port=%d, got port=%d", tc.expected, result)
			}
		})
	}
}

func TestNormalizePortValue(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "int unchanged",
			value:    8080,
			expected: 8080,
		},
		{
			name:     "float64 to int",
			value:    float64(3000),
			expected: 3000,
		},
		{
			name:     "float32 to int",
			value:    float32(5000),
			expected: 5000,
		},
		{
			name:     "int32 to int",
			value:    int32(7000),
			expected: 7000,
		},
		{
			name:     "int64 to int",
			value:    int64(9000),
			expected: 9000,
		},
		{
			name:     "valid string to int",
			value:    "4000",
			expected: 4000,
		},
		{
			name:     "invalid string unchanged",
			value:    "not-a-port",
			expected: "not-a-port",
		},
		{
			name:     "empty string unchanged",
			value:    "",
			expected: "",
		},
		{
			name:     "string with whitespace to int",
			value:    "  6000  ",
			expected: 6000,
		},
		{
			name:     "json.Number to int",
			value:    json.Number("2000"),
			expected: 2000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizePortValue(tc.value)
			if result != tc.expected {
				t.Errorf("Expected %v (type %T), got %v (type %T)", tc.expected, tc.expected, result, result)
			}
		})
	}
}

func TestConvertPortsToMap(t *testing.T) {
	testCases := []struct {
		name     string
		entries  []scenarioPort
		expected map[string]interface{}
	}{
		{
			name:     "empty entries",
			entries:  []scenarioPort{},
			expected: nil,
		},
		{
			name: "valid entries",
			entries: []scenarioPort{
				{Key: "API_PORT", Port: 8080},
				{Key: "UI_PORT", Port: 3000},
			},
			expected: map[string]interface{}{
				"API_PORT": 8080,
				"UI_PORT":  3000,
			},
		},
		{
			name: "entries with empty keys",
			entries: []scenarioPort{
				{Key: "", Port: 8080},
				{Key: "UI_PORT", Port: 3000},
			},
			expected: map[string]interface{}{
				"UI_PORT": 3000,
			},
		},
		{
			name: "entries with nil port",
			entries: []scenarioPort{
				{Key: "API_PORT", Port: nil},
				{Key: "UI_PORT", Port: 3000},
			},
			expected: map[string]interface{}{
				"UI_PORT": 3000,
			},
		},
		{
			name: "entries with whitespace keys",
			entries: []scenarioPort{
				{Key: "  API_PORT  ", Port: 8080},
			},
			expected: map[string]interface{}{
				"API_PORT": 8080,
			},
		},
		{
			name: "entries with various port types",
			entries: []scenarioPort{
				{Key: "INT_PORT", Port: 8080},
				{Key: "STRING_PORT", Port: "9000"},
				{Key: "FLOAT_PORT", Port: float64(7000)},
			},
			expected: map[string]interface{}{
				"INT_PORT":    8080,
				"STRING_PORT": 9000,
				"FLOAT_PORT":  7000,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertPortsToMap(tc.entries)
			if (result == nil) != (tc.expected == nil) {
				t.Fatalf("Expected nil=%v, got nil=%v", tc.expected == nil, result == nil)
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(result))
			}
			for key, expectedVal := range tc.expected {
				actualVal, ok := result[key]
				if !ok {
					t.Errorf("Expected key %q not found in result", key)
					continue
				}
				if actualVal != expectedVal {
					t.Errorf("Key %q: expected %v, got %v", key, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestExtractInterfaceMap(t *testing.T) {
	testCases := []struct {
		name     string
		value    interface{}
		expected map[string]interface{}
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: nil,
		},
		{
			name: "map[string]interface{}",
			value: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
		{
			name: "map[string]int",
			value: map[string]int{
				"port1": 8080,
				"port2": 3000,
			},
			expected: map[string]interface{}{
				"port1": 8080,
				"port2": 3000,
			},
		},
		{
			name: "map[string]float64",
			value: map[string]float64{
				"metric1": 1.5,
				"metric2": 2.7,
			},
			expected: map[string]interface{}{
				"metric1": 1.5,
				"metric2": 2.7,
			},
		},
		{
			name: "map[string]string",
			value: map[string]string{
				"env1": "value1",
				"env2": "value2",
			},
			expected: map[string]interface{}{
				"env1": "value1",
				"env2": "value2",
			},
		},
		{
			name:     "unsupported type",
			value:    []string{"a", "b"},
			expected: nil,
		},
		{
			name:     "primitive type",
			value:    42,
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractInterfaceMap(tc.value)
			if (result == nil) != (tc.expected == nil) {
				t.Fatalf("Expected nil=%v, got nil=%v", tc.expected == nil, result == nil)
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(result))
			}
			for key, expectedVal := range tc.expected {
				actualVal, ok := result[key]
				if !ok {
					t.Errorf("Expected key %q not found in result", key)
					continue
				}
				if actualVal != expectedVal {
					t.Errorf("Key %q: expected %v, got %v", key, expectedVal, actualVal)
				}
			}
		})
	}
}

func TestAnyStringFromMap(t *testing.T) {
	testCases := []struct {
		name     string
		data     map[string]interface{}
		keys     []string
		expected string
	}{
		{
			name: "simple key",
			data: map[string]interface{}{
				"message": "hello",
			},
			keys:     []string{"message"},
			expected: "hello",
		},
		{
			name: "nested key",
			data: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "error occurred",
				},
			},
			keys:     []string{"error", "message"},
			expected: "error occurred",
		},
		{
			name: "key not found",
			data: map[string]interface{}{
				"other": "value",
			},
			keys:     []string{"missing"},
			expected: "",
		},
		{
			name:     "nil data",
			data:     nil,
			keys:     []string{"key"},
			expected: "",
		},
		{
			name: "empty keys",
			data: map[string]interface{}{
				"key": "value",
			},
			keys:     []string{},
			expected: "",
		},
		{
			name: "non-string value",
			data: map[string]interface{}{
				"count": 42,
			},
			keys:     []string{"count"},
			expected: "42",
		},
		{
			name: "nested value is not map",
			data: map[string]interface{}{
				"error": "simple string",
			},
			keys:     []string{"error", "message"},
			expected: "simple string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := anyStringFromMap(tc.data, tc.keys...)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func stringPtr(s string) *string {
	return &s
}
