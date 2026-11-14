package evaluation_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"web-search/internal/evaluation"
)

func TestCheckedInCorpusValidates(t *testing.T) {
	b, err := os.ReadFile("../../../evals/research-assurance-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var c evaluation.Corpus
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckedInCorpusCoversDeclaredPopulations(t *testing.T) {
	b, err := os.ReadFile("../../../evals/research-assurance-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var c evaluation.Corpus
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, item := range c.Cases {
		counts[item.Split]++
	}
	if len(c.Cases) != 10 || counts["development"] != 5 || counts["holdout"] != 4 || counts["live"] != 1 {
		t.Fatalf("unexpected corpus populations: total=%d counts=%v", len(c.Cases), counts)
	}
	for _, item := range c.Cases {
		if len(item.ExpectedClaims) == 0 {
			t.Fatalf("case %q has no expected disposition", item.ID)
		}
	}
}

func source(id, revision, content string) evaluation.CorpusSource {
	h := sha256.Sum256([]byte(content))
	return evaluation.CorpusSource{ID: id, Revision: revision, Content: content, ContentHash: hex.EncodeToString(h[:])}
}
func validCorpus() evaluation.Corpus {
	c := evaluation.Corpus{Version: "v1", Cases: []evaluation.CorpusCase{{ID: "case-1", Split: "holdout", Provenance: "fixture", Sources: []evaluation.CorpusSource{source("s1", "r1", "The answer is seven.")}, Questions: []evaluation.CorpusQuestion{{ID: "q1", Text: "What is the answer?", Required: true}}, ExpectedClaims: []evaluation.ExpectedClaim{{ClaimID: "q1", Text: "The answer is seven.", Disposition: "supported", SourceID: "s1", StartByte: 0, EndByte: len("The answer is seven.")}}}}}
	c.ContentHash = c.ManifestHash()
	return c
}
func TestCorpusValidationRejectsMissingCoverageAndBadSpan(t *testing.T) {
	c := validCorpus()
	c.Cases[0].ExpectedClaims = nil
	if err := c.Validate(); err == nil {
		t.Fatal("accepted missing expected coverage")
	}
	c = validCorpus()
	c.Cases[0].ExpectedClaims[0].EndByte++
	if err := c.Validate(); err == nil {
		t.Fatal("accepted support span outside bytes")
	}
}
func TestCorpusValidationRejectsDuplicateSplitContentAndSourceChange(t *testing.T) {
	c := validCorpus()
	other := c.Cases[0]
	other.ID = "case-2"
	other.Split = "development"
	c.Cases = append(c.Cases, other)
	if err := c.Validate(); err == nil {
		t.Fatal("accepted duplicate content across splits")
	}
	c = validCorpus()
	c.Cases[0].SourceChange = true
	if err := c.Validate(); err == nil {
		t.Fatal("accepted source change without revisions")
	}
}
func TestAnswerMatchDoesNotUseKeywordPresence(t *testing.T) {
	if evaluation.AnswerMatches("The answer is seven.", "The answer is eight, not seven.") {
		t.Fatal("keyword presence passed as answer")
	}
	if !evaluation.AnswerMatches("The answer is seven.", "The answer is seven.") {
		t.Fatal("exact answer rejected")
	}
}
