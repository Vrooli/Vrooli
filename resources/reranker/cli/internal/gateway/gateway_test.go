package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"resource-reranker/cli/internal/client"
)

type fakeClient struct {
	rerankResults []client.RankResult
	rerankErr     error
	healthErr     error
	info          map[string]any
	infoErr       error

	gotQuery   string
	gotDocs    []string
	gotReturnT bool
}

func (f *fakeClient) Rerank(_ context.Context, query string, documents []string, returnText bool) ([]client.RankResult, error) {
	f.gotQuery = query
	f.gotDocs = documents
	f.gotReturnT = returnText
	return f.rerankResults, f.rerankErr
}

func (f *fakeClient) Health(context.Context) error { return f.healthErr }

func (f *fakeClient) Info(context.Context) (map[string]any, error) { return f.info, f.infoErr }

func newHandlers(f *fakeClient, stdin string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	return &Handlers{
		NewClient: func() Client { return f },
		Stdin:     strings.NewReader(stdin),
		Stdout:    &out,
		Stderr:    &errBuf,
	}, &out, &errBuf
}

func TestRerankJSONOutputTopK(t *testing.T) {
	f := &fakeClient{rerankResults: []client.RankResult{
		{Index: 2, Score: 0.98},
		{Index: 0, Score: 0.42},
		{Index: 1, Score: 0.01},
	}}
	h, out, _ := newHandlers(f, "")
	err := h.Rerank([]string{
		"--query", "restart a scenario",
		"--document", "alpha", "--document", "beta", "--document", "restart via cli",
		"--top-k", "2", "--json",
	})
	if err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if f.gotQuery != "restart a scenario" || len(f.gotDocs) != 3 {
		t.Fatalf("client got query=%q docs=%v", f.gotQuery, f.gotDocs)
	}
	var decoded []client.RankResult
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("output not valid json: %v\n%s", err, out.String())
	}
	if len(decoded) != 2 {
		t.Fatalf("top-k not applied: got %d results", len(decoded))
	}
	if decoded[0].Index != 2 {
		t.Errorf("top result index = %d, want 2", decoded[0].Index)
	}
}

func TestRerankHumanOutputResolvesText(t *testing.T) {
	f := &fakeClient{rerankResults: []client.RankResult{{Index: 1, Score: 0.9}, {Index: 0, Score: 0.1}}}
	h, out, _ := newHandlers(f, "")
	if err := h.Rerank([]string{"--query", "q", "--document", "first", "--document", "second"}); err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "second") || !strings.Contains(got, "0.900000") {
		t.Errorf("human output missing resolved text/score:\n%s", got)
	}
}

func TestRerankStdinDocuments(t *testing.T) {
	f := &fakeClient{rerankResults: []client.RankResult{{Index: 0, Score: 0.5}}}
	h, _, _ := newHandlers(f, "line one\n\nline two\n")
	if err := h.Rerank([]string{"--query", "q", "--documents-stdin"}); err != nil {
		t.Fatalf("Rerank() error = %v", err)
	}
	if len(f.gotDocs) != 2 || f.gotDocs[0] != "line one" || f.gotDocs[1] != "line two" {
		t.Errorf("stdin docs parsed wrong: %v", f.gotDocs)
	}
}

func TestRerankValidation(t *testing.T) {
	f := &fakeClient{}
	h, _, _ := newHandlers(f, "")
	if err := h.Rerank([]string{"--document", "a"}); err == nil {
		t.Error("expected error when --query missing")
	}
	if err := h.Rerank([]string{"--query", "q"}); err == nil {
		t.Error("expected error when no documents provided")
	}
	if err := h.Rerank([]string{"--query", "q", "--document", "a", "--documents-stdin"}); err == nil {
		t.Error("expected error for mutually exclusive doc sources")
	}
}

func TestHealthJSON(t *testing.T) {
	h, out, _ := newHandlers(&fakeClient{}, "")
	if err := h.Health([]string{"--json"}); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if !strings.Contains(out.String(), `"healthy":true`) {
		t.Errorf("health json wrong: %s", out.String())
	}
}

func TestInfoHumanFiltersKeys(t *testing.T) {
	f := &fakeClient{info: map[string]any{"model_id": "BAAI/bge-reranker-v2-m3", "model_type": "reranker", "noise": "x"}}
	h, out, _ := newHandlers(f, "")
	if err := h.Info(nil); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "BAAI/bge-reranker-v2-m3") || strings.Contains(got, "noise") {
		t.Errorf("info output wrong: %s", got)
	}
}

func TestCommandsRegistersSubcommands(t *testing.T) {
	g := Commands(nil)
	if g.Name != "gateway" {
		t.Fatalf("group name = %q", g.Name)
	}
	want := map[string]bool{"rerank": false, "health": false, "info": false}
	for _, c := range g.Subcommands {
		if _, ok := want[c.Name]; ok {
			want[c.Name] = true
		}
		if c.Run == nil {
			t.Errorf("subcommand %q has nil Run", c.Name)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}
