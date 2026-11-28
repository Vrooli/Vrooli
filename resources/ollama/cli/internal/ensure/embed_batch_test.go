package ensure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/runtimeprofile"
)

func TestEmbedBatchTruncatesAndForwardsVariantOptions(t *testing.T) {
	variant, err := runtimeprofile.Current().VariantOptions(runtimeprofile.VariantCPUOnly)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/embed" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var request BatchEmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request.Model != "nomic-embed-text:latest" || !request.Truncate || !reflect.DeepEqual(request.Input, []string{"a", "b"}) {
			t.Errorf("request body = %+v", request)
		}
		if len(request.Options) != len(variant) {
			t.Errorf("options = %#v, want only the cpu-only variant %#v", request.Options, variant)
		}
		_ = json.NewEncoder(w).Encode(EmbedResponse{Model: request.Model, Embeddings: [][]float64{{1, 2}, {3, 4}}, TotalDuration: 900, LoadDuration: 100, PromptEvalCount: 2})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	response, err := client.EmbedBatch(context.Background(), "nomic-embed-text:latest", []string{"a", "b"}, variant)
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}
	if len(response.Embeddings) != 2 || response.TotalDuration != 900 || response.LoadDuration != 100 || response.PromptEvalCount != 2 {
		t.Fatalf("response = %+v", response)
	}
}

func TestEmbedBatchRejectsWrongVectorCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(EmbedResponse{Embeddings: [][]float64{{1, 2}}})
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	if _, err := client.EmbedBatch(context.Background(), "model", []string{"a", "b"}, nil); err == nil {
		t.Fatal("accepted response vector count that differs from input count")
	}
}
