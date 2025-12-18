package app

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestFindSimilarScenariosQdrant_DirectHTTP(t *testing.T) {
	t.Setenv("USE_RESOURCE_QDRANT_CLI", "")
	t.Setenv("SCENARIO_EMBEDDINGS_COLLECTION", "scenario_embeddings")
	t.Setenv("OLLAMA_EMBEDDING_MODEL", "nomic-embed-text")

	var created atomic.Bool

	qdrantHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/collections/scenario_embeddings":
			w.WriteHeader(http.StatusNotFound)
			return

		case r.Method == http.MethodPut && r.URL.Path == "/collections/scenario_embeddings":
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("qdrant create collection invalid json: %v", err)
			}
			vectors, _ := decoded["vectors"].(map[string]any)
			if vectors == nil {
				t.Fatalf("qdrant create collection missing vectors: %v", decoded)
			}
			if vectors["size"] != float64(3) {
				t.Fatalf("expected vectors.size=3 got %v", vectors["size"])
			}
			if vectors["distance"] != "Cosine" {
				t.Fatalf("expected vectors.distance=Cosine got %v", vectors["distance"])
			}
			created.Store(true)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"result":true}`)
			return

		case r.Method == http.MethodPost && r.URL.Path == "/collections/scenario_embeddings/points/search":
			if !created.Load() {
				t.Fatalf("search called before collection created")
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("qdrant search invalid json: %v", err)
			}
			if decoded["limit"] != float64(5) {
				t.Fatalf("expected limit=5 got %v", decoded["limit"])
			}
			if decoded["with_payload"] != true {
				t.Fatalf("expected with_payload=true got %v", decoded["with_payload"])
			}
			vector, _ := decoded["vector"].([]any)
			if len(vector) != 3 {
				t.Fatalf("expected vector length 3 got %d", len(vector))
			}
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{
			  "result": [
			    {"id":"a","score":0.8,"payload":{"scenario_name":"alpha","description":"A","resources":["qdrant","postgres"]}},
			    {"id":"b","score":0.6,"payload":{"scenario_name":"beta","description":"B","resources":["redis"]}}
			  ]
			}`)
			return
		default:
			t.Fatalf("unexpected qdrant request: %s %s", r.Method, r.URL.Path)
		}
	})
	qdrantListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("network listen not available in test environment: %v", err)
	}
	qdrant := &httptest.Server{Listener: qdrantListener, Config: &http.Server{Handler: qdrantHandler}}
	qdrant.Start()
	defer qdrant.Close()

	ollamaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/embeddings" {
			t.Fatalf("unexpected ollama request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("ollama embeddings invalid json: %v", err)
		}
		if decoded["model"] != "nomic-embed-text" {
			t.Fatalf("expected model nomic-embed-text got %v", decoded["model"])
		}
		if decoded["prompt"] == "" {
			t.Fatalf("expected non-empty prompt")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"embedding":[1,2,3]}`)
	})
	ollamaListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("network listen not available in test environment: %v", err)
	}
	ollama := &httptest.Server{Listener: ollamaListener, Config: &http.Server{Handler: ollamaHandler}}
	ollama.Start()
	defer ollama.Close()

	t.Setenv("QDRANT_URL", qdrant.URL)
	t.Setenv("OLLAMA_URL", ollama.URL)

	matches, err := findSimilarScenariosQdrant("test scenario description", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match (score>0.7) got %d: %#v", len(matches), matches)
	}
	if matches[0]["scenario_name"] != "alpha" {
		t.Fatalf("expected scenario_name alpha got %#v", matches[0]["scenario_name"])
	}
	if matches[0]["description"] != "A" {
		t.Fatalf("expected description A got %#v", matches[0]["description"])
	}
	resources, ok := matches[0]["resources"].([]string)
	if !ok || len(resources) != 2 || resources[0] != "qdrant" || resources[1] != "postgres" {
		t.Fatalf("unexpected resources: %#v", matches[0]["resources"])
	}
}
