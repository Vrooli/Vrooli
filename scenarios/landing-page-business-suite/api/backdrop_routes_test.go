package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestBackdropRouteResolvesProducerAtRequestTimeAndNormalizesAssetURL(t *testing.T) {
	producer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/vrooli.backdrop_studio.v1.release.ReleaseService/GetReference" {
			t.Fatalf("producer request = %s %s", r.Method, r.URL.Path)
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["id"] != "hero" {
			t.Fatalf("producer body = %+v, err = %v", request, err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "hero", "uri": "/assets/hero.png"})
	}))
	defer producer.Close()

	server := &Server{
		router: mux.NewRouter(),
		backdropResolver: func(context.Context) (string, error) {
			return producer.URL, nil
		},
		backdropHTTPClient: producer.Client(),
	}
	registerBackdropRoutes(server)

	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/backdrops/hero", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var metadata map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["url"] != producer.URL+"/assets/hero.png" {
		t.Fatalf("url = %v, want producer-relative URL", metadata["url"])
	}
}
