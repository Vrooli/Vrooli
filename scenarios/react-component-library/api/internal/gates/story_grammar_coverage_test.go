package gates

import (
	"encoding/json"
	"testing"
)

func TestContainsRawValueMatchesJSONIgnoringWhitespace(t *testing.T) {
	values := []json.RawMessage{json.RawMessage(`{"mode": "compact"}`), json.RawMessage(`7`)}
	if !containsRawValue(values, json.RawMessage(` {"mode": "compact"} `)) {
		t.Fatal("containsRawValue() did not match equivalent raw JSON with surrounding whitespace")
	}
	if containsRawValue(values, json.RawMessage(`"7"`)) {
		t.Fatal("containsRawValue() matched a string value to a numeric value")
	}
}
