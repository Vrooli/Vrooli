package vectorstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knowledge-observatory/internal/ports"
)

func TestEnsureCollectionValidatesInputs(t *testing.T) {
	client := &Qdrant{}
	if err := client.EnsureCollection(context.Background(), "", 128); err == nil {
		t.Fatalf("expected error for empty collection")
	}
	if err := client.EnsureCollection(context.Background(), "test", 0); err == nil {
		t.Fatalf("expected error for invalid vector size")
	}
}

func TestEnsureCollectionCreatesWhenMissing(t *testing.T) {
	var createReq createCollectionRequest
	putCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut:
			putCalled = true
			if err := json.NewDecoder(r.Body).Decode(&createReq); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := &Qdrant{BaseURL: server.URL, Client: server.Client()}
	if err := client.EnsureCollection(context.Background(), "demo", 256); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !putCalled {
		t.Fatalf("expected create collection call")
	}
	if createReq.Vectors.Size != 256 {
		t.Fatalf("expected vector size 256, got %d", createReq.Vectors.Size)
	}
	if createReq.Vectors.Distance != "Cosine" {
		t.Fatalf("expected cosine distance, got %q", createReq.Vectors.Distance)
	}
}

func TestEnsureCollectionSkipsWhenExists(t *testing.T) {
	putCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
		case http.MethodPut:
			putCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := &Qdrant{BaseURL: server.URL, Client: server.Client()}
	if err := client.EnsureCollection(context.Background(), "demo", 256); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if putCalled {
		t.Fatalf("did not expect create collection call")
	}
}

func TestBuildFilterAndStringifyID(t *testing.T) {
	filter := buildFilter(&ports.VectorFilter{
		Namespaces: []string{"core"},
		Visibility: []string{"public"},
		Tags:       []string{"tag"},
	})
	if filter == nil || len(filter.Must) != 3 {
		t.Fatalf("expected populated filter")
	}
	if got := stringifyID(42.0); got != "42" {
		t.Fatalf("expected numeric id stringify, got %q", got)
	}
	if got := stringifyID("abc"); got != "abc" {
		t.Fatalf("expected string id stringify, got %q", got)
	}
}

func TestQdrantSearchRejectsMissingCollection(t *testing.T) {
	client := &Qdrant{BaseURL: "http://localhost"}
	_, err := client.Search(context.Background(), "", []float64{0.1}, 5, 0.2, nil)
	if err == nil || !strings.Contains(err.Error(), "collection") {
		t.Fatalf("expected collection error, got %v", err)
	}
}
