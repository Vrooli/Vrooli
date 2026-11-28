package evaluation_test

import (
	"testing"
	"web-search/internal/evaluation"
)

func obs(task, method string, effort float64, support, coverage bool) evaluation.Observation {
	return evaluation.Observation{TaskID: task, MethodHash: method, Stratum: "cold", Effort: effort, Support: support, Coverage: coverage, Provenance: "test"}
}

func TestCompareAcceptsQualityPreservingEffortReduction(t *testing.T) {
	report, err := evaluation.Compare([]evaluation.Observation{obs("a", "base", 10, true, true), obs("b", "base", 12, true, true)}, []evaluation.Observation{obs("a", "candidate", 8, true, true), obs("b", "candidate", 9, true, true)})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Accepted || report.Reason != "quality_preserved_effort_reduced" {
		t.Fatalf("report = %+v", report)
	}
}

func TestCompareRejectsCheaperWrongCandidate(t *testing.T) {
	report, err := evaluation.Compare([]evaluation.Observation{obs("a", "base", 10, true, true)}, []evaluation.Observation{obs("a", "candidate", 1, false, true)})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted || report.Reason != "quality_regression" {
		t.Fatalf("report = %+v", report)
	}
}

func TestCompareRejectsIncompletePopulation(t *testing.T) {
	report, err := evaluation.Compare([]evaluation.Observation{obs("a", "base", 10, true, true), obs("b", "base", 10, true, true)}, []evaluation.Observation{obs("a", "candidate", 1, true, true)})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted || report.Reason != "incomplete_pairs" {
		t.Fatalf("report = %+v", report)
	}
}

func TestCompareReportsColdAndWarmStrataSeparately(t *testing.T) {
	base := []evaluation.Observation{
		{TaskID: "cold-1", MethodHash: "base", Stratum: "cold", Effort: 10, Support: true, Coverage: true, Provenance: "test"},
		{TaskID: "warm-1", MethodHash: "base", Stratum: "warm", Effort: 10, Support: true, Coverage: true, Provenance: "test"},
	}
	candidate := []evaluation.Observation{
		{TaskID: "cold-1", MethodHash: "candidate", Stratum: "cold", Effort: 12, Support: true, Coverage: true, Provenance: "test"},
		{TaskID: "warm-1", MethodHash: "candidate", Stratum: "warm", Effort: 5, Support: true, Coverage: true, Provenance: "test"},
	}
	report, err := evaluation.Compare(base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if report.Strata["cold"].EffortReduced {
		t.Fatal("cold-only regression was reported as an improvement")
	}
	if !report.Strata["warm"].EffortReduced {
		t.Fatal("warm improvement was not retained")
	}
}

func TestCompareRejectsDifferentCorpusHashes(t *testing.T) {
	base := evaluation.Observation{TaskID: "a", MethodHash: "base", CorpusHash: "corpus-a", SourcePopulation: "sources-a", Stratum: "cold", Effort: 10, Support: true, Coverage: true, Provenance: "test"}
	candidate := base
	candidate.MethodHash = "candidate"
	candidate.CorpusHash = "corpus-b"
	if _, err := evaluation.Compare([]evaluation.Observation{base}, []evaluation.Observation{candidate}); err == nil {
		t.Fatal("accepted observations from different corpus revisions")
	}
}

func TestCompareRejectsChangedSourcePopulation(t *testing.T) {
	base := evaluation.Observation{TaskID: "a", MethodHash: "base", CorpusHash: "corpus", SourcePopulation: "sources-a", Stratum: "cold", Effort: 10, Support: true, Coverage: true, Provenance: "test"}
	candidate := base
	candidate.MethodHash = "candidate"
	candidate.SourcePopulation = "sources-b"
	if _, err := evaluation.Compare([]evaluation.Observation{base}, []evaluation.Observation{candidate}); err == nil {
		t.Fatal("accepted observations from different source populations")
	}
}

func TestAcceptedPairedImprovementReceiptUsesFrozenPopulation(t *testing.T) {
	base := make([]evaluation.Observation, 0, 4)
	candidate := make([]evaluation.Observation, 0, 4)
	for _, item := range []struct {
		task       string
		stratum    string
		baseEffort float64
		newEffort  float64
	}{
		{task: "fact", stratum: "cold", baseEffort: 10, newEffort: 8},
		{task: "compare", stratum: "cold", baseEffort: 12, newEffort: 9},
		{task: "fact", stratum: "warm", baseEffort: 6, newEffort: 5},
		{task: "compare", stratum: "warm", baseEffort: 8, newEffort: 7},
	} {
		base = append(base, evaluation.Observation{TaskID: item.task, MethodHash: "method-base", CorpusHash: "corpus-v1", SourcePopulation: "sources-v1", Stratum: item.stratum, Effort: item.baseEffort, Support: true, Coverage: true, Provenance: "test"})
		candidate = append(candidate, evaluation.Observation{TaskID: item.task, MethodHash: "method-candidate", CorpusHash: "corpus-v1", SourcePopulation: "sources-v1", Stratum: item.stratum, Effort: item.newEffort, Support: true, Coverage: true, Provenance: "test"})
	}
	report, err := evaluation.Compare(base, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Accepted {
		t.Fatalf("report = %+v", report)
	}
	receipt := evaluation.EvaluationReceipt{ID: "receipt-controlled-v1", CandidateHash: evaluation.RevisionHash(evaluation.MethodRevision{ID: "method-candidate", ProgramHash: "program-v1", ConfigHash: "config-v1"}), ReportHash: evaluation.ReportHash(report), Accepted: true}
	t.Logf("evaluation_receipt=%s report_hash=%s population=%d baseline_mean=%.2f candidate_mean=%.2f", receipt.ID, receipt.ReportHash, report.Population, report.MeanBaselineEffort, report.MeanCandidateEffort)
}
