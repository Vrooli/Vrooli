package services

import (
	"context"
	"errors"
	"testing"
	"time"
)

type heavyJobFixture struct {
	snapshot HeavyJobSnapshot
	err      error
}

func (f heavyJobFixture) GetHeavyJobs(context.Context) (HeavyJobSnapshot, error) {
	return f.snapshot, f.err
}

func TestUnattendedHeavyJobRuleReportsOwnerAndDurableJobOnly(t *testing.T) {
	now := time.Date(2026, 11, 15, 20, 0, 0, 0, time.UTC)
	base := HeavyJob{OwnerID: "image-tools", JobID: "job-42", Operation: "text_to_image", State: "running", StartedAt: now.Add(-37 * time.Minute), Heavy: true}
	jobs := []HeavyJob{base}
	waited := base
	waited.JobID, waited.WaiterCount = "job-waited", 1
	jobs = append(jobs, waited)
	light := base
	light.JobID, light.Heavy = "job-light", false
	jobs = append(jobs, light)
	completed := base
	completed.JobID, completed.State = "succeeded", "succeeded"
	jobs = append(jobs, completed)
	rule := NewUnattendedHeavyJobRule(heavyJobFixture{snapshot: HeavyJobSnapshot{ObservedAt: now, Complete: true, Jobs: jobs}})
	eval := rule.Evaluate(context.Background(), AlertSnapshot{Now: now})
	if eval.Status != RuleSupported || len(eval.Conditions) != 1 {
		t.Fatalf("evaluation = %+v, want one supported finding", eval)
	}
	condition := eval.Conditions[0]
	if condition.Key != "image-tools:job-42" || condition.Severity != SeverityWarning || condition.Details["owner_id"] != "image-tools" || condition.Details["job_id"] != "job-42" || condition.Details["report_only"] != true {
		t.Fatalf("condition = %+v, missing stable owner/job report-only identity", condition)
	}
}

func TestUnattendedHeavyJobRuleRequiresFreshCompleteEvidence(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name string
		snap HeavyJobSnapshot
	}{
		{name: "incomplete", snap: HeavyJobSnapshot{ObservedAt: now, Jobs: []HeavyJob{{OwnerID: "image-tools"}}}},
		{name: "stale", snap: HeavyJobSnapshot{ObservedAt: now.Add(-2 * time.Minute), Complete: true}},
		{name: "future", snap: HeavyJobSnapshot{ObservedAt: now.Add(time.Second), Complete: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NewUnattendedHeavyJobRule(heavyJobFixture{snapshot: tc.snap}).Evaluate(context.Background(), AlertSnapshot{Now: now})
			if got.Status != RuleUnsupported {
				t.Fatalf("status = %q, want unsupported: %+v", got.Status, got)
			}
		})
	}
	if got := NewUnattendedHeavyJobRule(heavyJobFixture{err: errors.New("owner unavailable")}).Evaluate(context.Background(), AlertSnapshot{Now: now}); got.Status != RuleFailed {
		t.Fatalf("source error status = %q, want failed", got.Status)
	}
}

func TestUnattendedHeavyJobRuleIgnoresInvalidOrYoungJobs(t *testing.T) {
	now := time.Now().UTC()
	old := HeavyJob{OwnerID: "image-tools", JobID: "old", Operation: "render", State: "running", StartedAt: now.Add(-20 * time.Minute), Heavy: true}
	young := old
	young.JobID, young.StartedAt = "young", now.Add(-5*time.Minute)
	badOwner := old
	badOwner.JobID, badOwner.OwnerID = "bad-owner", ""
	staleAge := old
	staleAge.JobID, staleAge.StartedAt = "bad-clock", now.Add(time.Second)
	fixture := heavyJobFixture{snapshot: HeavyJobSnapshot{ObservedAt: now, Complete: true, Jobs: []HeavyJob{young, badOwner, staleAge}}}
	got := NewUnattendedHeavyJobRule(fixture).Evaluate(context.Background(), AlertSnapshot{Now: now})
	if got.Status != RuleSupported || len(got.Conditions) != 0 {
		t.Fatalf("evaluation = %+v, want clear supported result", got)
	}
}
