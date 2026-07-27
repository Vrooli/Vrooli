package support

import "testing"

func TestParseBodyValidatesJSON(t *testing.T) {
	body, err := ParseBody(`{"enabled":true}`)
	if err != nil {
		t.Fatalf("ParseBody returned an error for valid JSON: %v", err)
	}
	if string(body) != `{"enabled":true}` {
		t.Fatalf("ParseBody returned %q, want original JSON", body)
	}
	if _, err := ParseBody(`{`); err == nil {
		t.Fatal("ParseBody accepted malformed JSON")
	}
}
