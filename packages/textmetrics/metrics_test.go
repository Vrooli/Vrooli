package textmetrics

import (
	"encoding/json"
	"testing"
)

func TestAnalyzeIsByteStable(t *testing.T) {
	text := "A short sentence. A second, slightly longer sentence!"
	a := Analyze(text, []string{"slightly longer"})
	b := Analyze(text, []string{"slightly longer"})
	left, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != string(right) {
		t.Fatalf("metrics are not byte stable:\n%s\n%s", left, right)
	}
	if len(a.LexiconFlags) != 1 || a.LexiconFlags[0].Start <= 0 {
		t.Fatalf("lexicon span missing: %+v", a.LexiconFlags)
	}
}

func TestAnalyzeSetRetainsPairwiseMatrix(t *testing.T) {
	_, set := AnalyzeSet([]string{"bright river stone", "bright river path", "quiet forest night"}, nil)
	if len(set.PairwiseSimilarity) != 3 || set.PairwiseSimilarity[0][1] == set.PairwiseSimilarity[0][2] {
		t.Fatalf("unexpected matrix: %+v", set.PairwiseSimilarity)
	}
	if set.Diversity <= 0 || set.Basis == "" {
		t.Fatalf("missing diversity basis: %+v", set)
	}
}

func TestComparableIncludesTokenSource(t *testing.T) {
	a := SamplingKey{Strategy: "direct", K: 3, MaxOutputTokens: 512, MaxOutputTokenSource: "profile"}
	b := a
	if err := Comparable(a, b); err != nil {
		t.Fatal(err)
	}
	b.MaxOutputTokenSource = "role-default"
	if err := Comparable(a, b); err == nil {
		t.Fatal("different token provenance was accepted")
	}
}
