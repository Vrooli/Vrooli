package goalhome

import (
	"strings"
	"testing"
)

func gateStates(report GateReport) string {
	var states []string
	for _, gate := range report.Gates {
		states = append(states, gate.ID+"="+gate.State)
	}
	return strings.Join(states, ",")
}

func warningCodes(findings []Finding) string {
	var codes []string
	for _, finding := range findings {
		codes = append(codes, finding.Code)
	}
	return strings.Join(codes, ",")
}

func TestTypedGatesAreJudgedOnlyAgainstTheAdmittedItems(t *testing.T) {
	base := readFixture(t, "epochs", "typed-gates.md")
	for _, tc := range []struct {
		name   string
		edit   func(string) string
		status string
		states string
		detail string
	}{
		{
			// DL-4 Gherkin: the admitted items are met, whatever other evidence
			// (here a pre-write baseline gap) is missing.
			name: "admitted items met", edit: func(s string) string { return s }, status: GateMet,
			states: "G1=met,G2=met,G3=met,G4=met,G5=met,G6=met,G7=dropped",
		},
		{
			name: "inventory compares the measured value with the bound", status: GateUnmet,
			edit:   func(s string) string { return strings.Replace(s, "gate G4=pass 252651", "gate G4=pass 253900", 1) },
			states: "G1=met,G2=met,G3=met,G4=unmet,G5=met,G6=met,G7=dropped", detail: "runtime_lines 253900 is not <= 253452",
		},
		{
			name: "inventory pass without a value is not trusted", status: GateUnmet,
			edit:   func(s string) string { return strings.Replace(s, "gate G4=pass 252651", "gate G4=pass", 1) },
			states: "G1=met,G2=met,G3=met,G4=unmet,G5=met,G6=met,G7=dropped", detail: "pass without a measured value",
		},
		{
			name: "journey count must cover every listed journey", status: GateUnmet,
			edit:   func(s string) string { return strings.Replace(s, "gate G3=pass 12/12", "gate G3=pass 11/11", 1) },
			states: "G1=met,G2=met,G3=unmet,G4=met,G5=met,G6=met,G7=dropped", detail: "covers fewer than the 12 listed journeys",
		},
		{
			name: "unverified needs a dated amendment", status: GateUnmet,
			edit: func(s string) string {
				return strings.Replace(s, "- A2 2026-10-07T14:31:00Z G6 unverified", "- A2 G6 unverified", 1)
			},
			states: "G1=met,G2=met,G3=met,G4=met,G5=met,G6=unmet,G7=dropped",
		},
		{
			name: "a later failure replaces an earlier pass", status: GateUnmet,
			edit: func(s string) string {
				return s + "2026-10-07T03:00:00Z | rerun | exit metric=type-check broke | gate G2=fail | ack=-\n"
			},
			states: "G1=met,G2=unmet,G3=met,G4=met,G5=met,G6=met,G7=dropped", detail: "failed",
		},
		{
			name: "no result recorded", status: GateUnmet,
			edit:   func(s string) string { return strings.Replace(s, " | gate G1=pass", "", 1) },
			states: "G1=unmet,G2=met,G3=met,G4=met,G5=met,G6=met,G7=dropped", detail: "no result recorded",
		},
		{
			name: "ACCEPTED gates= are the orchestrator's own reruns", status: GateMet,
			edit: func(s string) string {
				return strings.Replace(s, " | gate G1=pass", "", 1) + "ACCEPTED 2026-10-07T15:00:00Z | yield=-801 runtime lines | gates=G1,G3 | accepted\n"
			},
			states: "G1=met,G2=met,G3=met,G4=met,G5=met,G6=met,G7=dropped",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := EvaluateGates(parseText(t, tc.edit(base)))
			if report.GateStatus != tc.status || gateStates(report) != tc.states {
				t.Fatalf("status %s gates %s, want %s %s: %+v", report.GateStatus, gateStates(report), tc.status, tc.states, report.Gates)
			}
			if tc.detail != "" {
				found := false
				for _, gate := range report.Gates {
					found = found || strings.Contains(gate.Detail, tc.detail)
				}
				if !found {
					t.Fatalf("no gate detail contains %q: %+v", tc.detail, report.Gates)
				}
			}
		})
	}
}

func TestTypedGateWarningsNameUnreasonedCustomGatesAndAmendments(t *testing.T) {
	report := EvaluateGates(parseText(t, readFixture(t, "epochs", "typed-gates.md")))
	// G7 is custom without a reason; A3 has no ISO time and no reason, so it
	// is not applied and G5 stays an admitted gate; the intro sentence is not
	// a typed item.
	if got := warningCodes(report.Warnings); got != "custom-gate-reason,gate-amendment-reason,untyped-gate" {
		t.Fatalf("warnings = %s: %+v", got, report.Warnings)
	}
}

func TestInventoryGateYieldIsMeasuredAgainstItsBaseline(t *testing.T) {
	epoch := parseText(t, readFixture(t, "epochs", "typed-gates.md"))
	report := EvaluateEpoch("typed-gates.md", epoch, defaultLimits, fixtureNow, nil)
	if report.NetRuntimeLines != -1602 {
		t.Fatalf("summed deltas = %d, want -1602 (the worker logged its running total twice)", report.NetRuntimeLines)
	}
	if report.Yield == nil || report.Yield.Value != -801 || report.YieldSource != "inventory gate G4" {
		t.Fatalf("yield = %+v from %q, want -801 from inventory gate G4", report.Yield, report.YieldSource)
	}
	if report.YieldEstimate == nil || report.YieldEstimate.Value != -600 {
		t.Fatalf("yield estimate = %+v, want -600 (U+2212 minus)", report.YieldEstimate)
	}
}

func TestLegacyGatesWarnAndNeverFail(t *testing.T) {
	for _, name := range []string{"E23.md", "E26.md"} {
		t.Run(name, func(t *testing.T) {
			report := EvaluateGates(parseText(t, readFixture(t, "bas", "epochs", name)))
			if report.GateStatus != GatesLegacy || len(report.Gates) != 0 || warningCodes(report.Warnings) != "legacy-gates" {
				t.Fatalf("legacy exit gate must report legacy with one warning: %+v", report)
			}
		})
	}
}

func TestGateAmendmentsAddAndChangeTypedItems(t *testing.T) {
	text := readFixture(t, "epochs", "typed-gates.md") +
		"2026-10-07T03:00:00Z | journeys | exit metric=J05 | gate G8=pass J05 | gate G2=pass | ack=-\n"
	text = strings.Replace(text, "- A3 2026-10-07 G5 drop", "- A3 2026-10-07 G5 drop — reason: folded into G1\n- A4 2026-10-07T14:40:00Z G8 add: journey: J05 @ bas-goal — reason: E26 touches the J05 export path\n- A5 2026-10-07T14:41:00Z G2 change: test: cd ui && pnpm type-check — reason: vitest export suite moved into G1", 1)
	report := EvaluateGates(parseText(t, text))
	if report.GateStatus != GateMet || gateStates(report) != "G1=met,G2=met,G3=met,G4=met,G5=dropped,G6=met,G7=dropped,G8=met" {
		t.Fatalf("amended gates = %s (%s): %+v", gateStates(report), report.GateStatus, report.Warnings)
	}
	for _, gate := range report.Gates {
		if gate.ID == "G2" && (gate.Spec != "cd ui && pnpm type-check" || strings.Join(gate.AmendedBy, ",") != "A5") {
			t.Fatalf("changed gate not applied: %+v", gate)
		}
	}
}
