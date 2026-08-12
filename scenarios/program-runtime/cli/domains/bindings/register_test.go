package bindings

import (
	"strings"
	"testing"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

func TestGroupName(t *testing.T) {
	if GroupName != "bindings" {
		t.Fatal(GroupName)
	}
}

func TestConditionReportPrintsInstrumentationTriple(t *testing.T) {
	report := (&handlers{}).conditionReport(nil, &bindingsv1.GetBindingConditionResponse{
		TotalBindings:        4,
		InstrumentedBindings: 1,
		WindowSeconds:        3600,
	})
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "instrumented=1/4") {
		t.Fatalf("condition summary=%v, want instrumentation triple", report.Summary)
	}
}
