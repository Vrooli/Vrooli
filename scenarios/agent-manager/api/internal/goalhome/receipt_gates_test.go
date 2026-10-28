package goalhome

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestJourneyReceiptsKeepUnrelatedEvidenceAndRejectChangedCases(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*JourneyReceipt)
		want   string
	}{
		{"unchanged dependencies", func(*JourneyReceipt) {}, GateMet},
		{"changed shared dependency", func(r *JourneyReceipt) { r.Cases[0].Current = false; r.Cases[0].Reason = "shared drawer changed" }, GateUnmet},
		{"failed execution", func(r *JourneyReceipt) { r.Cases[0].Passed = false }, GateUnmet},
		{"missing case", func(r *JourneyReceipt) { r.Cases = r.Cases[:1] }, GateUnmet},
		{"wrong scenario", func(r *JourneyReceipt) { r.Scenario = "other" }, GateUnmet},
		{"wrong receipt", func(r *JourneyReceipt) { r.ID = "other" }, GateUnmet},
		{"not terminal", func(r *JourneyReceipt) { r.CompletedAt = time.Time{} }, GateUnmet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := receiptGateReport(t, "workflow-health:owner")
			read := func(id string) (JourneyReceipt, error) {
				v := JourneyReceipt{ID: id, Scenario: "demo", CompletedAt: time.Now(), Cases: []JourneyCaseEvidence{{Path: "bas/cases/one.json", Passed: true, Current: true}, {Path: "bas/cases/two.json", Passed: true, Current: true}}}
				tc.mutate(&v)
				return v, nil
			}
			ResolveJourneyReceipts(r, "demo", read)
			if r.Gates[0].State != tc.want {
				t.Fatalf("%+v", r.Gates)
			}
			if r.Gates[1].State != GateUnmet {
				t.Fatal("journey evidence accepted an independent review gate")
			}
		})
	}
}

func receiptGateReport(t *testing.T, refs string) *EpochReport {
	t.Helper()
	s := fmt.Sprintf("# E1\n- Scenario: demo\n## Exit gate\n- G1 journey: bas/cases/one.json,bas/cases/two.json @ workflow-health\n- G2 review: independent visual review\n## Slice log\n2026-10-25T12:00:00Z | linked owner | exit metric=evidence linked | gate G1=receipt %s | ack=-\n", refs)
	return EvaluateEpoch("E1.md", parseText(t, s), defaultLimits, fixtureNow, nil)
}

func TestJourneyReceiptsUseNewestCaseOutcomeIndependentOfReferenceOrder(t *testing.T) {
	for _, refs := range []string{"workflow-health:old,workflow-health:new", "workflow-health:new,workflow-health:old"} {
		r := receiptGateReport(t, refs)
		ResolveJourneyReceipts(r, "demo", func(id string) (JourneyReceipt, error) {
			at := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
			v := JourneyReceipt{ID: id, Scenario: "demo", CompletedAt: at, Cases: []JourneyCaseEvidence{{Path: "bas/cases/one.json", Passed: true, Current: true}, {Path: "bas/cases/two.json", Passed: true, Current: true}}}
			if id == "new" {
				v.CompletedAt = at.Add(time.Minute)
				v.Cases = v.Cases[:1]
				v.Cases[0].Passed = false
				v.Cases[0].Reason = "latest attempt failed"
			}
			return v, nil
		})
		if r.Gates[0].State != GateUnmet || !strings.Contains(r.Gates[0].Detail, "latest attempt failed") {
			t.Fatalf("new failure hidden by old pass: %+v", r.Gates)
		}
	}
}

func TestJourneyReceiptFailureCannotBeHiddenByAcceptanceOrUnavailableReads(t *testing.T) {
	s := "# E1\n## Exit gate\n- G1 journey: bas/cases/one.json @ workflow-health\n## Slice log\n2026-10-25T12:00:00Z | linked | exit metric=linked | gate G1=receipt workflow-health:owner\nACCEPTED 2026-10-25T12:01:00Z | gates=G1 | prior review\n"
	r := EvaluateEpoch("E1.md", parseText(t, s), defaultLimits, fixtureNow, nil)
	ResolveJourneyReceipts(r, "demo", func(string) (JourneyReceipt, error) { return JourneyReceipt{}, fmt.Errorf("owner offline") })
	if r.Gates[0].State != GateUnmet || !strings.Contains(r.Gates[0].Detail, "owner offline") {
		t.Fatalf("acceptance bypassed current evidence: %+v", r.Gates)
	}
}

func TestReceiptJourneyRejectsManualPassCount(t *testing.T) {
	s := "# E1\n## Exit gate\n- G1 journey: bas/cases/one.json @ workflow-health\n## Slice log\n2026-10-25T12:00:00Z | manually claimed | exit metric=old prose | gate G1=pass 1/1\n"
	r := EvaluateEpoch("E1.md", parseText(t, s), defaultLimits, fixtureNow, nil)
	if r.Gates[0].State != GateUnmet || !strings.Contains(r.Gates[0].Detail, "manual pass/count") {
		t.Fatalf("manual counter bypassed owner: %+v", r.Gates)
	}
}
