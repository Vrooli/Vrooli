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
		TotalBindings:   4,
		WindowSeconds:   3600,
		LedgerExercise:  &bindingsv1.ExerciseBasisSummary{Basis: "local_invocation_ledger", InstrumentedBindings: 3, TotalBindings: 4, Invocations: 7},
		ReceiptExercise: &bindingsv1.ExerciseBasisSummary{Basis: "fleet_receipt_aggregate", InstrumentedBindings: 1, TotalBindings: 4, Invocations: 2},
	})
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "ledger instrumented=3/4") || !strings.Contains(report.Summary[0], "receipts instrumented=1/4") {
		t.Fatalf("condition summary=%v, want instrumentation triple", report.Summary)
	}
}
