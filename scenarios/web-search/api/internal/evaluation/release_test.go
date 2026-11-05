package evaluation_test

import (
	"testing"
	"web-search/internal/evaluation"
)

func TestPromotionRequiresMatchingPassingReceipt(t *testing.T) {
	revision := evaluation.MethodRevision{ID: "candidate", ProgramHash: "p1", ConfigHash: "c1"}
	base := obs("a", "base", 10, true, true)
	candidate := obs("a", "candidate", 8, true, true)
	report, err := evaluation.Compare([]evaluation.Observation{base}, []evaluation.Observation{candidate})
	if err != nil {
		t.Fatal(err)
	}
	registry := &evaluation.MethodRegistry{}
	bad := evaluation.EvaluationReceipt{ID: "r", CandidateHash: "wrong", ReportHash: evaluation.ReportHash(report), Accepted: true}
	if err := registry.Promote("", revision, bad, report, "grant-1"); err == nil {
		t.Fatal("expected mismatched receipt to be rejected")
	}
	good := evaluation.EvaluationReceipt{ID: "r", CandidateHash: evaluation.RevisionHash(revision), ReportHash: evaluation.ReportHash(report), Accepted: true}
	if err := registry.Promote("", revision, good, report, "grant-1"); err != nil {
		t.Fatal(err)
	}
}

func TestRollbackRequiresKnownTargetAndCurrentHash(t *testing.T) {
	registry := &evaluation.MethodRegistry{}
	revision := evaluation.MethodRevision{ID: "one", ProgramHash: "p1", ConfigHash: "c1"}
	report := evaluation.Report{Accepted: true}
	receipt := evaluation.EvaluationReceipt{ID: "r", CandidateHash: evaluation.RevisionHash(revision), ReportHash: evaluation.ReportHash(report), Accepted: true}
	if err := registry.Promote("", revision, receipt, report, "grant-1"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Rollback(evaluation.RevisionHash(revision), "missing"); err == nil {
		t.Fatal("expected unknown rollback target to fail")
	}
}

func TestPromotionRequiresGrantAndSuspension(t *testing.T) {
	registry := &evaluation.MethodRegistry{}
	revision := evaluation.MethodRevision{ID: "candidate", ProgramHash: "p1", ConfigHash: "c1"}
	report := evaluation.Report{Accepted: true}
	receipt := evaluation.EvaluationReceipt{ID: "r", CandidateHash: evaluation.RevisionHash(revision), ReportHash: evaluation.ReportHash(report), Accepted: true}
	if err := registry.Promote("", revision, receipt, report, ""); err == nil {
		t.Fatal("missing grant bypassed promotion")
	}
	if err := registry.Suspend(evaluation.RevisionHash(revision), "source correction"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Promote("", revision, receipt, report, "grant-1"); err == nil {
		t.Fatal("suspended revision was promoted")
	}
}
