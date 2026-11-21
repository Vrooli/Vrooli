package publicbench

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tiny records exercise the adapter's file/label invariants only. They
// are not exported as benchmark cases; accepted cases always come from an
// externally staged official SciFact release.
func TestLoadSciFactDevUsesDevEvidenceAndBindsCorpus(t *testing.T) {
	root := writeSciFactFiles(t,
		`{"id":9,"claim":"second claim","evidence":{"20":[{"label":"CONTRADICT","sentences":[0]}]}}`+"\n"+
			`{"id":4,"claim":"first claim","evidence":{"10":[{"label":"SUPPORT","sentences":[1]}],"20":[{"label":"SUPPORT","sentences":[2]},{"label":"SUPPORT","sentences":[3]}]}}`+"\n"+
			`{"id":2,"claim":"not enough evidence","evidence":{}}`+"\n",
		`{"doc_id":20,"title":"Second","abstract":["second abstract"],"structured":false}`+"\n"+
			`{"doc_id":10,"title":"First","abstract":["part one","part two"],"structured":false}`+"\n")

	dataset, err := LoadSciFactDev(root, 1)
	if err != nil {
		t.Fatalf("LoadSciFactDev: %v", err)
	}
	if dataset.Manifest.Split != "dev" || dataset.Manifest.SplitFile != "claims_dev.jsonl" {
		t.Fatalf("wrong source split: %+v", dataset.Manifest)
	}
	if dataset.Manifest.ClaimsLicense != "CC-BY-4.0" || dataset.Manifest.AbstractsLicense != "ODC-By-1.0" {
		t.Fatalf("license boundary missing: %+v", dataset.Manifest)
	}
	if dataset.CorpusSHA256 == "" || dataset.LabelSHA256 == "" || dataset.DatasetSHA256 == "" || len(dataset.Cases) != 1 || dataset.SkippedNoGold != 1 {
		t.Fatalf("unexpected subset identity/size: corpus=%q labels=%q dataset=%q cases=%d skipped=%d", dataset.CorpusSHA256, dataset.LabelSHA256, dataset.DatasetSHA256, len(dataset.Cases), dataset.SkippedNoGold)
	}
	got := dataset.Cases[0]
	if got.ClaimID != 4 || got.CaseID != "scifact/dev/4" || got.Query != "first claim" {
		t.Fatalf("unexpected selected case: %+v", got)
	}
	if strings.Join(got.ExpectedIDs, ",") != "10,20" || len(got.EvidenceLabels) != 2 || got.EvidenceLabels[1].Labels[0] != "SUPPORT" {
		t.Fatalf("gold labels were not preserved: %+v", got)
	}
	if !strings.Contains(got.LabelSource, "claims_dev.jsonl record id=4; field evidence") || !strings.Contains(got.LabelSource, "10:SUPPORT") {
		t.Fatalf("expected-label provenance missing: %q", got.LabelSource)
	}
	if len(dataset.Corpus) != 2 || dataset.Corpus[0].ID != "10" || dataset.Corpus[0].Text != "part one part two" {
		t.Fatalf("full corpus was not normalized deterministically: %+v", dataset.Corpus)
	}

	suite, err := dataset.SearchSuite("publicbench.scifact")
	if err != nil {
		t.Fatalf("SearchSuite: %v", err)
	}
	if len(suite.GetCases()) != 1 || suite.GetCases()[0].GetExpectIds()[0] != "10" {
		t.Fatalf("suite did not retain source labels: %+v", suite)
	}
	for _, want := range []string{"claims_license=CC-BY-4.0", "corpus_license=ODC-By-1.0", "claims_dev.jsonl record id=4; field evidence", "10:SUPPORT", "corpus_sha256=", "label_sha256=", "dataset_sha256="} {
		if !strings.Contains(suite.GetCases()[0].GetNote(), want) {
			t.Errorf("suite note lacks %q: %s", want, suite.GetCases()[0].GetNote())
		}
	}
}

func TestLoadSciFactDevRejectsMissingGoldDocument(t *testing.T) {
	root := writeSciFactFiles(t,
		`{"id":7,"claim":"label points outside corpus","evidence":{"404":[{"label":"SUPPORT","sentences":[0]}]}}`+"\n",
		`{"doc_id":10,"title":"Only available document","abstract":["text"],"structured":false}`+"\n")
	if _, err := LoadSciFactDev(root, 1); err == nil || !strings.Contains(err.Error(), `missing corpus document "404"`) {
		t.Fatalf("expected missing public label target to be rejected, got %v", err)
	}
}

func TestLoadSciFactDevRequiresExplicitBoundedSubset(t *testing.T) {
	for _, n := range []int{0, -1, maxSciFactCases + 1} {
		if _, err := LoadSciFactDev("unused", n); err == nil {
			t.Errorf("maxCases=%d should fail", n)
		}
	}
}

func writeSciFactFiles(t *testing.T, claims, corpus string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "claims_dev.jsonl"), []byte(claims), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "corpus.jsonl"), []byte(corpus), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
