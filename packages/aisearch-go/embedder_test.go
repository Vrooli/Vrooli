package aisearch

import (
	"context"
	"testing"
)

func TestTaskPrefixesFor(t *testing.T) {
	cases := []struct {
		model        string
		query, docpx string
	}{
		{"nomic-embed-text", "search_query: ", "search_document: "},
		{"NOMIC-EMBED-TEXT:latest", "search_query: ", "search_document: "},
		{"mxbai-embed-large", "", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		q, d := TaskPrefixesFor(c.model)
		if q != c.query || d != c.docpx {
			t.Errorf("TaskPrefixesFor(%q) = (%q,%q); want (%q,%q)", c.model, q, d, c.query, c.docpx)
		}
	}
}

// capturingRunner records the stdin handed to each embed invocation so a test
// can assert the task prefix the embedder applied.
func capturingRunner(seen *[]string) EmbedRunner {
	return func(_ context.Context, _ []string, stdin []byte) ([]byte, error) {
		*seen = append(*seen, string(stdin))
		return []byte(`{"embedding":[0.1,0.2,0.3]}`), nil
	}
}

func TestCLIEmbedderAppliesTaskPrefixes(t *testing.T) {
	var seen []string
	e := NewEmbedderWithRunnerPrefixed("nomic-embed-text", capturingRunner(&seen))

	te, ok := e.(TaskEmbedder)
	if !ok {
		t.Fatal("nomic embedder must implement TaskEmbedder")
	}
	if _, err := te.EmbedQuery(context.Background(), "list scenarios"); err != nil {
		t.Fatal(err)
	}
	if _, err := te.EmbedDocument(context.Background(), "vrooli scenario list"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Embed(context.Background(), "ping"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"search_query: list scenarios",
		"search_document: vrooli scenario list",
		"ping", // symmetric Embed applies no prefix
	}
	if len(seen) != len(want) {
		t.Fatalf("captured %d inputs, want %d: %q", len(seen), len(want), seen)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Errorf("input[%d] = %q; want %q", i, seen[i], want[i])
		}
	}
}

func TestEmbedderWithEmptyPrefixesIsSymmetric(t *testing.T) {
	var seen []string
	// NewEmbedderWithRunner derives prefixes from the model, so to force symmetric
	// we construct the prefixes explicitly via the unexported path used by
	// NewEmbedderWithPrefixes — mirror it here with the runner seam.
	e := &cliEmbedder{bin: defaultEmbedderBin, model: "nomic-embed-text", role: DefaultEmbedRole, run: capturingRunner(&seen), queryPrefix: "", docPrefix: ""}
	te := TaskEmbedder(e)
	if _, err := te.EmbedQuery(context.Background(), "q"); err != nil {
		t.Fatal(err)
	}
	if _, err := te.EmbedDocument(context.Background(), "d"); err != nil {
		t.Fatal(err)
	}
	if seen[0] != "q" || seen[1] != "d" {
		t.Errorf("empty-prefix embedder must pass text verbatim, got %q", seen)
	}
}

func TestCLIEmbedderUsesRole(t *testing.T) {
	var gotArgs []string
	run := func(_ context.Context, args []string, _ []byte) ([]byte, error) {
		gotArgs = append([]string(nil), args...)
		return []byte(`{"embedding":[0.1,0.2,0.3]}`), nil
	}
	e := newEmbedderWithRoleAndPrefixes(DefaultEmbedModel, "embedding.default", "", "", run)
	if _, err := e.Embed(context.Background(), "ping"); err != nil {
		t.Fatal(err)
	}
	want := []string{defaultEmbedderBin, "gateway", "embed", "--role", "embedding.default", "--json", "--input-stdin"}
	if len(gotArgs) != len(want) {
		t.Fatalf("args = %v, want %v", gotArgs, want)
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Fatalf("args = %v, want %v", gotArgs, want)
		}
	}
}

// nonTaskEmbedder implements only the base Embedder; the role-aware helpers must
// fall back to Embed for it (back-compat with existing fakes / symmetric models).
type nonTaskEmbedder struct{ calls int }

func (e *nonTaskEmbedder) Embed(context.Context, string) ([]float64, error) {
	e.calls++
	return []float64{1}, nil
}
func (e *nonTaskEmbedder) Available(context.Context) bool { return true }

func TestRoleHelpersFallBackToEmbed(t *testing.T) {
	e := &nonTaskEmbedder{}
	if _, err := embedQueryText(context.Background(), e, "q"); err != nil {
		t.Fatal(err)
	}
	if _, err := embedDocumentText(context.Background(), e, "d"); err != nil {
		t.Fatal(err)
	}
	if e.calls != 2 {
		t.Errorf("expected 2 fallback Embed calls, got %d", e.calls)
	}
}
