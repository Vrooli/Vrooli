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
	items, set := AnalyzeSet([]string{"bright river stone", "bright river path", "quiet forest night"}, nil)
	if len(set.PairwiseSimilarity) != 3 || set.PairwiseSimilarity[0][1] == set.PairwiseSimilarity[0][2] {
		t.Fatalf("unexpected matrix: %+v", set.PairwiseSimilarity)
	}
	if set.Diversity <= 0 || set.Basis == "" {
		t.Fatalf("missing diversity basis: %+v", set)
	}
	if items[0].RougeLHomogenize <= 0 || items[0].RougeLHomogenize >= 1 {
		t.Fatalf("ROUGE-L homogenization was not computed: %+v", items[0])
	}
}

func TestComparableIncludesTokenSource(t *testing.T) {
	a := SamplingKey{K: 3, MaxOutputTokens: 512, MaxOutputTokenSource: "profile"}
	b := a
	if err := Comparable(a, b); err != nil {
		t.Fatal(err)
	}
	b.MaxOutputTokenSource = "role-default"
	if err := Comparable(a, b); err == nil {
		t.Fatal("different token provenance was accepted")
	}
}

// The strategy is the variable an experiment varies, so two sets drawn under
// identical conditions must compare no matter which strategy produced them.
// The key holds no strategy field at all; this guards the conditions that do
// remain, so a later addition cannot reintroduce a strategy proxy unnoticed.
func TestComparableIgnoresStrategyButHoldsConditions(t *testing.T) {
	base := SamplingKey{K: 5, TemperatureStance: "role_default", MaxOutputTokens: 2048, MaxOutputTokenSource: "request"}
	if err := Comparable(base, base); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*SamplingKey){
		"candidate count": func(k *SamplingKey) { k.K = 3 },
		"sampling stance": func(k *SamplingKey) { k.TemperatureStance = "0.7" },
		"output ceiling":  func(k *SamplingKey) { k.MaxOutputTokens = 1024 },
		"ceiling source":  func(k *SamplingKey) { k.MaxOutputTokenSource = "role-default" },
	} {
		other := base
		mutate(&other)
		if err := Comparable(base, other); err == nil {
			t.Fatalf("a differing %s was accepted as comparable", name)
		}
	}
}

func TestAnalyzeSetSemanticUsesEmbeddingGeometryAndNamesDimension(t *testing.T) {
	items, set, err := AnalyzeSetSemantic([]string{"same", "near", "different"}, [][]float64{{1, 0}, {0.9, 0.1}, {-1, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || set.PairwiseSimilarity[0][1] <= set.PairwiseSimilarity[0][2] {
		t.Fatalf("semantic distances did not preserve geometry: %#v", set.PairwiseSimilarity)
	}
	if set.Basis != "semantic cosine similarity over gateway embeddings (dimension 2)" {
		t.Fatalf("unexpected semantic basis: %q", set.Basis)
	}
}

func TestAnalyzeSetSemanticRejectsMalformedVectors(t *testing.T) {
	if _, _, err := AnalyzeSetSemantic([]string{"a"}, nil); err == nil || err.Error() != "semantic_measurement_vector_count_mismatch" {
		t.Fatalf("expected named vector count error, got %v", err)
	}
	if _, _, err := AnalyzeSetSemantic([]string{"a", "b"}, [][]float64{{1}, {1, 0}}); err == nil || err.Error() != "semantic_measurement_dimension_mismatch" {
		t.Fatalf("expected named dimension error, got %v", err)
	}
}

func TestBasisNamesTheRealEmbeddingDimension(t *testing.T) {
	// The basis string is what a reader consults to decide whether a similarity
	// number can be trusted. A stub formatter returned "3" for every dimension
	// above two, so a 768-dimension measurement described itself as
	// three-dimensional and looked degenerate.
	vectors := make([][]float64, 2)
	for i := range vectors {
		vectors[i] = make([]float64, 768)
		vectors[i][i] = 1
	}
	_, set, err := AnalyzeSetSemantic([]string{"a", "b"}, vectors)
	if err != nil {
		t.Fatal(err)
	}
	if set.Basis != "semantic cosine similarity over gateway embeddings (dimension 768)" {
		t.Fatalf("basis misreports the embedding dimension: %q", set.Basis)
	}
}

func TestSectionNoveltySeparatesRestatementFromNewMaterial(t *testing.T) {
	first := "The runner owns the execution after the command returns. Logs stream to durable storage and the identifier stays addressable."
	restated := "The execution is owned by the runner once the command has returned. Durable storage receives the streamed logs and the identifier remains addressable."
	advanced := "Operators reconnect hours later from another machine. A colleague inherits the release decision without reconstructing missing telemetry."

	if got := SectionNovelty(first, first); got != 0 {
		t.Fatalf("a section repeated verbatim must introduce nothing, got %v", got)
	}
	// Restatement in fresh words is the failure this measure exists to catch:
	// it scores low here while sharing few tokens with what it repeats.
	low := SectionNovelty(restated, first)
	high := SectionNovelty(advanced, first)
	if low >= high {
		t.Fatalf("paraphrased restatement (%v) must score below advancing material (%v)", low, high)
	}
	if high <= 0.5 {
		t.Fatalf("a section on new material should clear 0.5, got %v", high)
	}
}

func TestMinSectionNoveltyReportsWeakestTransition(t *testing.T) {
	sections := []string{
		"Durable runs keep their identity after the caller exits.",
		"Durable runs retain identity once the caller has exited.",
		"Scheduling policy decides which suites run overnight on shared hardware.",
	}
	got := MinSectionNovelty(sections)
	if got >= 0.5 {
		t.Fatalf("the restating transition should set the floor, got %v", got)
	}
	if only := MinSectionNovelty([]string{"one section"}); only != 1 {
		t.Fatalf("a document with no transition has nothing to measure, got %v", only)
	}
}

func TestArtifactsPresentCountsDistinctLiteralsNotOccurrences(t *testing.T) {
	artifacts := []string{
		"test-genie runs wait --json",
		"vrooli scenario test",
		"docs/TESTING.md",
		"vrooli scenario test abort",
	}
	text := "Start with vrooli scenario test, then block once on test-genie runs wait --json. " +
		"Run test-genie runs wait --json again and it returns the same record. See docs/TESTING.md."

	got := ArtifactsPresent(text, artifacts)
	// Repeating one command does not make the prose more concrete, and the
	// abort command never appears at all.
	want := []string{"test-genie runs wait --json", "vrooli scenario test", "docs/TESTING.md"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("declaration order not preserved: expected %v, got %v", want, got)
		}
	}
	if none := ArtifactsPresent("prose with no artifacts in it", artifacts); len(none) != 0 {
		t.Fatalf("expected no artifacts, got %v", none)
	}
}
