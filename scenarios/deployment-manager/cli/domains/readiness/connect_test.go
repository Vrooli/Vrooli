package readiness

import "testing"

func TestParseFacts(t *testing.T) {
	facts, err := parseFacts([]string{"commercial_release=true", "schema_changed=false"})
	if err != nil {
		t.Fatalf("parseFacts returned error: %v", err)
	}
	if facts["commercial_release"] != "true" || facts["schema_changed"] != "false" {
		t.Fatalf("facts = %#v", facts)
	}

	for _, raw := range []string{"missing-equals", "=true", "paid_release="} {
		if _, err := parseFacts([]string{raw}); err == nil {
			t.Errorf("parseFacts(%q) accepted malformed fact", raw)
		}
	}
	if _, err := parseFacts([]string{"commercial_release=true", "commercial_release=false"}); err == nil {
		t.Error("parseFacts accepted duplicate fact")
	}
}
