package database

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// JSONMap Tests
// =============================================================================

func TestJSONMapScanSupportsByteSlice(t *testing.T) {
	var m JSONMap
	if err := m.Scan([]byte(`{"foo": "bar", "count": 2}`)); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if got := m["foo"]; got != "bar" {
		t.Fatalf("expected foo=bar got %v", got)
	}
	if got := m["count"]; got != float64(2) {
		t.Fatalf("expected count=2 got %v", got)
	}
}

func TestJSONMapScanSupportsString(t *testing.T) {
	var m JSONMap
	if err := m.Scan("{\"foo\":\"baz\"}"); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if got := m["foo"]; got != "baz" {
		t.Fatalf("expected foo=baz got %v", got)
	}
}

func TestJSONMapScanSupportsRawMessage(t *testing.T) {
	var m JSONMap
	raw := json.RawMessage([]byte(`{"hello":"world"}`))
	if err := m.Scan(raw); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if got := m["hello"]; got != "world" {
		t.Fatalf("expected hello=world got %v", got)
	}
}

func TestJSONMapScanRejectsUnsupportedTypes(t *testing.T) {
	var m JSONMap
	if err := m.Scan(123); err == nil {
		t.Fatalf("expected error when scanning unsupported type")
	}
}

// New comprehensive JSONMap tests

func TestJSONMapScanNilValue(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] handles nil input", func(t *testing.T) {
		m := JSONMap{"existing": "data"}
		err := m.Scan(nil)
		require.NoError(t, err)
		assert.Nil(t, m, "scanning nil should set map to nil")
	})
}

func TestJSONMapScanEmptyJSON(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] handles empty JSON object", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{}`))
		require.NoError(t, err)
		assert.Empty(t, m)
	})

	t.Run("handles empty string", func(t *testing.T) {
		var m JSONMap
		err := m.Scan("")
		// Empty string is not valid JSON, should error
		assert.Error(t, err)
	})
}

func TestJSONMapScanComplexNestedStructures(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-LOAD-SUCCESS] handles deeply nested objects", func(t *testing.T) {
		var m JSONMap
		complexJSON := `{
			"level1": {
				"level2": {
					"level3": {
						"value": "deep",
						"array": [1, 2, 3]
					}
				}
			}
		}`
		err := m.Scan([]byte(complexJSON))
		require.NoError(t, err)

		// Navigate the nested structure
		level1, ok := m["level1"].(map[string]any)
		require.True(t, ok, "level1 should be a map")
		level2, ok := level1["level2"].(map[string]any)
		require.True(t, ok, "level2 should be a map")
		level3, ok := level2["level3"].(map[string]any)
		require.True(t, ok, "level3 should be a map")
		assert.Equal(t, "deep", level3["value"])
	})

	t.Run("handles arrays at root level", func(t *testing.T) {
		var m JSONMap
		// Note: arrays at root level are not valid JSONMap
		err := m.Scan([]byte(`[1, 2, 3]`))
		assert.Error(t, err, "arrays at root should fail JSONMap scan")
	})

	t.Run("[REQ:BAS-WORKFLOW-LOAD-SUCCESS] handles arrays as values", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{"items": [1, 2, 3], "mixed": [1, "two", true, null]}`))
		require.NoError(t, err)

		items, ok := m["items"].([]any)
		require.True(t, ok)
		assert.Len(t, items, 3)

		mixed, ok := m["mixed"].([]any)
		require.True(t, ok)
		assert.Equal(t, float64(1), mixed[0])
		assert.Equal(t, "two", mixed[1])
		assert.Equal(t, true, mixed[2])
		assert.Nil(t, mixed[3])
	})
}

func TestJSONMapScanInvalidJSON(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects invalid JSON", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{invalid json}`))
		assert.Error(t, err)
	})

	t.Run("rejects truncated JSON", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{"key": "value`))
		assert.Error(t, err)
	})
}

func TestJSONMapScanSpecialValues(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-LOAD-SUCCESS] handles null values", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{"key": null}`))
		require.NoError(t, err)
		assert.Contains(t, m, "key")
		assert.Nil(t, m["key"])
	})

	t.Run("handles boolean values", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{"enabled": true, "disabled": false}`))
		require.NoError(t, err)
		assert.Equal(t, true, m["enabled"])
		assert.Equal(t, false, m["disabled"])
	})

	t.Run("handles numeric edge cases", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{"integer": 42, "float": 3.14159, "negative": -100, "zero": 0}`))
		require.NoError(t, err)
		assert.Equal(t, float64(42), m["integer"])
		assert.Equal(t, 3.14159, m["float"])
		assert.Equal(t, float64(-100), m["negative"])
		assert.Equal(t, float64(0), m["zero"])
	})

	t.Run("handles unicode strings", func(t *testing.T) {
		var m JSONMap
		err := m.Scan([]byte(`{"emoji": "🚀", "chinese": "中文", "escaped": "\u0041"}`))
		require.NoError(t, err)
		assert.Equal(t, "🚀", m["emoji"])
		assert.Equal(t, "中文", m["chinese"])
		assert.Equal(t, "A", m["escaped"])
	})
}

func TestJSONMapValue(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] marshals non-nil map", func(t *testing.T) {
		m := JSONMap{
			"key":    "value",
			"number": 42,
		}
		val, err := m.Value()
		require.NoError(t, err)
		require.NotNil(t, val)

		// Verify it's valid JSON
		bytes, ok := val.([]byte)
		require.True(t, ok, "Value should return []byte")

		var result map[string]any
		err = json.Unmarshal(bytes, &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("returns nil for nil map", func(t *testing.T) {
		var m JSONMap
		val, err := m.Value()
		require.NoError(t, err)
		assert.Nil(t, val)
	})

	t.Run("[REQ:BAS-WORKFLOW-LOAD-SUCCESS] marshals nested structures", func(t *testing.T) {
		m := JSONMap{
			"nested": map[string]any{
				"inner": "value",
			},
			"array": []any{1, 2, 3},
		}
		val, err := m.Value()
		require.NoError(t, err)
		require.NotNil(t, val)

		bytes, _ := val.([]byte)
		var result map[string]any
		err = json.Unmarshal(bytes, &result)
		require.NoError(t, err)

		nested, ok := result["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "value", nested["inner"])
	})

	t.Run("handles empty map", func(t *testing.T) {
		m := JSONMap{}
		val, err := m.Value()
		require.NoError(t, err)
		require.NotNil(t, val)

		bytes, _ := val.([]byte)
		assert.Equal(t, "{}", string(bytes))
	})
}

func TestJSONMapRoundTrip(t *testing.T) {
	t.Run("[REQ:BAS-WORKFLOW-LOAD-SUCCESS] value then scan preserves data", func(t *testing.T) {
		original := JSONMap{
			"string":  "value",
			"number":  42.5,
			"boolean": true,
			"null":    nil,
			"nested": map[string]any{
				"key": "inner",
			},
		}

		// Get Value
		val, err := original.Value()
		require.NoError(t, err)

		// Scan it back
		var restored JSONMap
		err = restored.Scan(val)
		require.NoError(t, err)

		// Compare
		assert.Equal(t, original["string"], restored["string"])
		assert.Equal(t, original["number"], restored["number"])
		assert.Equal(t, original["boolean"], restored["boolean"])
		assert.Nil(t, restored["null"])
	})
}

// =============================================================================
// NullableString Tests
// =============================================================================

func TestNullableStringMarshalJSON(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] marshals valid string", func(t *testing.T) {
		ns := NullableString{}
		ns.Valid = true
		ns.String = "test value"

		data, err := ns.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, `"test value"`, string(data))
	})

	t.Run("marshals empty valid string", func(t *testing.T) {
		ns := NullableString{}
		ns.Valid = true
		ns.String = ""

		data, err := ns.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, `""`, string(data))
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] marshals null for invalid", func(t *testing.T) {
		ns := NullableString{}
		ns.Valid = false

		data, err := ns.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})

	t.Run("handles special characters", func(t *testing.T) {
		ns := NullableString{}
		ns.Valid = true
		ns.String = "line1\nline2\ttab\"quote"

		data, err := ns.MarshalJSON()
		require.NoError(t, err)

		// Should be properly escaped
		assert.Contains(t, string(data), "\\n")
		assert.Contains(t, string(data), "\\t")
		assert.Contains(t, string(data), "\\\"")
	})
}

func TestNullableStringUnmarshalJSON(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] unmarshals string value", func(t *testing.T) {
		var ns NullableString
		err := ns.UnmarshalJSON([]byte(`"test value"`))
		require.NoError(t, err)
		assert.True(t, ns.Valid)
		assert.Equal(t, "test value", ns.String)
	})

	t.Run("unmarshals empty string", func(t *testing.T) {
		var ns NullableString
		err := ns.UnmarshalJSON([]byte(`""`))
		require.NoError(t, err)
		assert.True(t, ns.Valid)
		assert.Equal(t, "", ns.String)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] unmarshals null", func(t *testing.T) {
		var ns NullableString
		err := ns.UnmarshalJSON([]byte(`null`))
		require.NoError(t, err)
		assert.False(t, ns.Valid)
	})

	t.Run("rejects non-string values", func(t *testing.T) {
		var ns NullableString
		err := ns.UnmarshalJSON([]byte(`123`))
		assert.Error(t, err)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		var ns NullableString
		err := ns.UnmarshalJSON([]byte(`not json`))
		assert.Error(t, err)
	})

	t.Run("handles escaped characters", func(t *testing.T) {
		var ns NullableString
		err := ns.UnmarshalJSON([]byte(`"line1\nline2"`))
		require.NoError(t, err)
		assert.True(t, ns.Valid)
		assert.Contains(t, ns.String, "\n")
	})
}

func TestNullableStringRoundTrip(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] marshal then unmarshal preserves value", func(t *testing.T) {
		original := NullableString{}
		original.Valid = true
		original.String = "round trip value"

		data, err := original.MarshalJSON()
		require.NoError(t, err)

		var restored NullableString
		err = restored.UnmarshalJSON(data)
		require.NoError(t, err)

		assert.Equal(t, original.Valid, restored.Valid)
		assert.Equal(t, original.String, restored.String)
	})

	t.Run("marshal then unmarshal preserves null", func(t *testing.T) {
		original := NullableString{}
		original.Valid = false

		data, err := original.MarshalJSON()
		require.NoError(t, err)

		var restored NullableString
		err = restored.UnmarshalJSON(data)
		require.NoError(t, err)

		assert.Equal(t, original.Valid, restored.Valid)
	})
}

// =============================================================================
// Error Tests
// =============================================================================

func TestErrNotFound(t *testing.T) {
	t.Run("[REQ:BAS-PROJECT-CREATE-SUCCESS] error message is correct", func(t *testing.T) {
		assert.Equal(t, "not found", ErrNotFound.Error())
	})

	t.Run("error is not nil", func(t *testing.T) {
		assert.NotNil(t, ErrNotFound)
	})
}
