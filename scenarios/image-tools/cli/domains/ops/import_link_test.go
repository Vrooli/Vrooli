package ops

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImportImageLinkUsesSecureImportAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/import/link" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["url"] != "https://images.example/photo.png" {
			t.Errorf("url=%q", body["url"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"input_ref":"input/abc.png","format":"png","mime_type":"image/png","width":32,"height":24}`))
	}))
	defer server.Close()
	got, err := importImageLinkWithClient(server.Client(), server.URL, "https://images.example/photo.png")
	if err != nil {
		t.Fatal(err)
	}
	if got.InputRef != "input/abc.png" || got.Width != 32 || got.Height != 24 || got.Format != "png" {
		t.Fatalf("result=%+v", got)
	}
}

func TestImportImageLinkSurfacesAPIRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unsafe URL", http.StatusBadRequest)
	}))
	defer server.Close()
	if _, err := importImageLinkWithClient(server.Client(), server.URL, "http://127.0.0.1/image.png"); err == nil {
		t.Fatal("expected API error")
	}
}
