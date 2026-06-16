package phases

import "testing"

func TestTranslateUIHealthReport_ErrorSeverityFailsPhase(t *testing.T) {
	rep := &uiHealthReport{
		Scenario: "demo",
		Passed:   false,
		Findings: []uiHealthFinding{
			{Severity: "SEVERITY_ERROR", Code: "overlay_unknown_slot", Message: "unknown slot", Location: "manifest.json"},
		},
	}
	rep.Summary.Errors = 1
	out := translateUIHealthReport(rep, 1)
	if out.Success {
		t.Fatal("expected Success=false on ERROR finding")
	}
	if out.FailureClass == "" {
		t.Fatal("expected failure class set")
	}
}

func TestTranslateUIHealthReport_PreservesLocalMaturitySummary(t *testing.T) {
	rep := &uiHealthReport{
		Scenario:   "demo",
		Passed:     true,
		Assessment: testProviderAssessment("demo", "ui-health", "ui-health", "L3", "L4"),
	}
	out := translateUIHealthReport(rep, 0)
	if out.Summary.LocalCurrentLevel != "L3" || out.Summary.LocalNextLevel != "L4" {
		t.Fatalf("local summary = current %q next %q, want L3/L4", out.Summary.LocalCurrentLevel, out.Summary.LocalNextLevel)
	}
	if got := out.Summary.String(); got != "demo passed=true errors=0 warnings=0 infos=0 local=L3 next=L4" {
		t.Fatalf("summary string = %q", got)
	}
}

func TestParseUIHealthOutput_Empty(t *testing.T) {
	if _, err := parseUIHealthOutput([]byte("  ")); err == nil {
		t.Fatal("expected error on empty output")
	}
}

func TestParseUIHealthOutput_RejectsMalformedAssessment(t *testing.T) {
	raw := []byte(`{"scenario":"demo","passed":true,"findings":[],"summary":{},"assessment":{"provider":"ui-health","phase":"ui-health","local":{}}}`)
	if _, err := parseUIHealthOutput(raw); err == nil {
		t.Fatal("expected malformed assessment error")
	}
}

func TestParseUIHealthOutput_RejectsMissingAssessment(t *testing.T) {
	raw := []byte(`{"scenario":"demo","passed":true,"findings":[],"summary":{}}`)
	if _, err := parseUIHealthOutput(raw); err == nil {
		t.Fatal("expected missing assessment error")
	}
}
