package evals

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"search-hub/cli/domains/evals/publicbench"
)

func TestFormatAggregateShowsFederatedRatesAndUnsetDirectRates(t *testing.T) {
	routing, recall := 0.75, 0.5
	federated := &evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{
		Cases: 4, Met: 2, PassRate: 0.5, RoutingPrecision: &routing, RetrievalRecall: &recall,
	}}
	fixedRecall := 0.25
	federated.Aggregate.FixedDenominatorRecall = &fixedRecall
	federated.Aggregate.RetrievalDenominator = 8
	federated.Aggregate.RetrievalHits = 2
	formatted := formatAggregate(federated)
	for _, want := range []string{"pass_rate=0.500", "routing_precision=0.750", "retrieval_recall=0.500", "fixed_denominator_recall=0.250", "retrieval_hits=2/8"} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("formatted aggregate %q does not contain %q", formatted, want)
		}
	}

	direct := formatAggregate(&evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{PassRate: 1}})
	if !strings.Contains(direct, "routing_precision=unset") || !strings.Contains(direct, "retrieval_recall=unset") || !strings.Contains(direct, "fixed_denominator_recall=unset") {
		t.Fatalf("direct aggregate did not preserve unset rates: %q", direct)
	}
}

func TestRunReportRendersRunSuiteAdequacyWarnings(t *testing.T) {
	report := (&handlers{}).runReport(nil, &evalv1.RunSuiteResponse{
		Run:      &evalv1.EvalRun{RunId: "run-1", SuiteId: "router.routing", Tier: "federated"},
		Adequacy: []*evalv1.AdequacyWarning{{Code: "coverage_gap", Message: "compound coverage is underrepresented"}},
	})
	changes := strings.Join(report.Changes, "\n")
	for _, want := range []string{"adequacy: 1 warning(s)", "coverage_gap: compound coverage is underrepresented"} {
		if !strings.Contains(changes, want) {
			t.Fatalf("run report changes %q do not contain API adequacy warning %q", changes, want)
		}
	}
}

func TestRateDeltaRequiresPresenceOnBothRuns(t *testing.T) {
	if delta, ok := rateDelta(0.4, 0.7, true, true); !ok || math.Abs(delta-0.3) > 1e-9 {
		t.Fatalf("rate delta = %v, %v; want 0.3, true", delta, ok)
	}
	if _, ok := rateDelta(0.4, 0.7, true, false); ok {
		t.Fatal("unset rate pair unexpectedly produced a delta")
	}
}

func TestPublicBenchSuiteBuildsSourceLinkedEvalSuite(t *testing.T) {
	root := t.TempDir()
	claims := `{"id":28,"claim":"A public benchmark claim.","evidence":{"4001":[{"label":"SUPPORT","sentences":[0]}]}}` + "\n"
	corpus := `{"doc_id":4001,"title":"Evidence source","abstract":["Published evidence text."],"structured":false}` + "\n"
	if err := os.WriteFile(filepath.Join(root, "claims_dev.jsonl"), []byte(claims), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "corpus.jsonl"), []byte(corpus), 0o600); err != nil {
		t.Fatal(err)
	}

	dataset, err := publicBenchDataset("scifact-dev", root, 1)
	if err != nil {
		t.Fatalf("publicBenchDataset: %v", err)
	}
	suite, err := dataset.SearchSuite("publicbench.scifact")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(suite.GetSuiteId(), "public.scifact.dev.v1.n1.") || suite.GetProviderId() != "publicbench.scifact" || len(suite.GetCases()) != 1 {
		t.Fatalf("unexpected registered suite shape: %+v", suite)
	}
	caseValue := suite.GetCases()[0]
	if caseValue.GetCaseId() != "scifact/dev/28" || strings.Join(caseValue.GetExpectIds(), ",") != "4001" {
		t.Fatalf("source labels were not carried into EvalSuite: %+v", caseValue)
	}
	for _, want := range []string{"split=dev", "claim_id=28", "evidence", "4001:SUPPORT", "claims_license=CC-BY-4.0", "corpus_license=ODC-By-1.0", "dataset_sha256="} {
		if !strings.Contains(caseValue.GetNote(), want) {
			t.Errorf("case note missing %q: %s", want, caseValue.GetNote())
		}
	}
}

func TestPublicBenchSuiteRejectsUnsupportedSplitAndProvider(t *testing.T) {
	if _, err := publicBenchDataset("scifact-test", "unused", 1); err == nil {
		t.Fatal("test split with withheld labels was accepted")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "claims_dev.jsonl"), []byte(`{"id":1,"claim":"claim","evidence":{"9":[{"label":"SUPPORT","sentences":[0]}]}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "corpus.jsonl"), []byte(`{"doc_id":9,"title":"title","abstract":["abstract"]}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := publicbench.SearchProviderDescriptor("", strings.Repeat("a", 64)); err == nil {
		t.Fatal("suite without provider id was accepted")
	}
}

func TestPublicBenchProviderUsesSearchHubStagedCorpusEndpoint(t *testing.T) {
	descriptor, err := publicbench.SearchProviderDescriptor("publicbench.scifact", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	httpJSON := descriptor.GetEndpoint().GetHttpJson()
	if httpJSON == nil || httpJSON.GetScenarioId() != "search-hub" || httpJSON.GetPath() != "/api/v1/publicbench/scifact/search" {
		t.Fatalf("provider is not wired to the source-backed Search Hub provider route: %+v", descriptor.GetEndpoint())
	}
	if descriptor.GetResultMapping().GetIdField() != "id" || descriptor.GetResultMapping().GetScoreScale().String() != "SCORE_SCALE_RAW" {
		t.Fatalf("provider result mapping does not match the provider response: %+v", descriptor.GetResultMapping())
	}
	if !strings.Contains(httpJSON.GetBodyTemplate(), strings.Repeat("a", 64)) {
		t.Fatalf("provider request does not pin the indexed corpus digest: %s", httpJSON.GetBodyTemplate())
	}
}
