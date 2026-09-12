package guidance

import "testing"

func TestParseOrientationOutputReturnsNextGate(t *testing.T) {
	out := []byte(`{
		"success": false,
		"orientation": {
			"scenario": "demo",
			"finalized": false,
			"completed": 2,
			"required": 9,
			"finalize_required": false,
			"message": "Next incomplete orientation step: Domain map.",
			"next_step": {
				"id": "domain-map",
				"title": "Domain map",
				"description": "Record real bounded contexts.",
				"docs": ["docs/START-HERE.md", "docs/concepts/DOMAINS.md"],
				"required": true,
				"complete": false,
				"checks": [
					{"kind":"file_exists","label":"docs/concepts/DOMAINS.md","passed":false,"skipped":false,"message":"missing file","optional":false}
				]
			}
		}
	}`)

	got, err := ParseOrientationOutput(out)
	if err != nil {
		t.Fatalf("ParseOrientationOutput() error = %v", err)
	}
	if got.Scenario != "demo" || got.Complete {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Gate == nil {
		t.Fatal("expected gate")
	}
	if got.Gate.ID != "domain-map" || len(got.Gate.Checks) != 1 {
		t.Fatalf("unexpected gate: %+v", got.Gate)
	}
	if got.Gate.Checks[0].Passed {
		t.Fatalf("expected failing check: %+v", got.Gate.Checks[0])
	}
	if len(got.Gate.Remediation) < 2 {
		t.Fatalf("expected remediation pointers: %+v", got.Gate.Remediation)
	}
}

func TestParseOrientationOutputReportsFinalizeRequired(t *testing.T) {
	out := []byte(`{
		"success": true,
		"orientation": {
			"scenario": "template-manager",
			"finalized": false,
			"completed": 9,
			"required": 9,
			"finalize_required": true,
			"message": "All required orientation steps are complete.",
			"next_step": null
		}
	}`)

	got, err := ParseOrientationOutput(out)
	if err != nil {
		t.Fatalf("ParseOrientationOutput() error = %v", err)
	}
	if !got.Complete || !got.FinalizeRequired {
		t.Fatalf("unexpected completion state: %+v", got)
	}
	if got.Gate != nil {
		t.Fatalf("expected no gate, got %+v", got.Gate)
	}
}
