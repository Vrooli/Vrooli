package knowledgeobservatory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testResolver struct{ base string }

func (r testResolver) ResolveScenarioURLDefault(_ context.Context, scenario string) (string, error) {
	if scenario != "knowledge-observatory" {
		panic("unexpected scenario: " + scenario)
	}
	return r.base, nil
}

func TestClientAndAdapterValidateMarkdownDiagrams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != procedure {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"findings":[{"code":"mermaid_invalid","message":"Parse error","line":3}]}`))
	}))
	defer server.Close()
	client := Client{resolver: testResolver{base: server.URL}, httpClient: server.Client(), timeout: defaultTimeout}
	result, err := client.ValidateMarkdownDiagrams(context.Background(), Request{Content: "```mermaid\ninvalid\n```", Source: "fixture.md"})
	if err != nil || len(result.Findings) != 1 || result.Findings[0].Line != 3 {
		t.Fatalf("unexpected client result: %+v err=%v", result, err)
	}
	adapter := Adapter[string, bool]{client: client, toRequest: func(string) Request { return Request{} }, fromResult: func(result Result) bool { return len(result.Findings) == 1 }}
	ok, err := adapter.ValidateMarkdownDiagrams(context.Background(), "ignored")
	if err != nil || !ok {
		t.Fatalf("unexpected adapter result: ok=%t err=%v", ok, err)
	}
}
