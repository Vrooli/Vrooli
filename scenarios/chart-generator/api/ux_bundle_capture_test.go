package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScenarioBundleCaptureClientValidatesAndFetchesPixels(t *testing.T) {
	pixels := []byte("\x89PNG\r\n\x1a\n")
	digest := sha256.Sum256(pixels)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/ux-capture/scenario":
			var request struct {
				Scenario string `json:"scenario"`
				Route    string `json:"route"`
				Width    int    `json:"width"`
				Height   int    `json:"height"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode capture request: %v", err)
			}
			if request.Scenario != "chart-generator" || request.Route != "/charts/rendered/demo.html" || request.Width != 800 || request.Height != 600 {
				t.Errorf("capture identity = %+v", request)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"manifest":{"schema_version":"ux-capture.v1","id":"123e4567-e89b-12d3-a456-426614174000","source":{"kind":"scenario","scenario":"chart-generator","route":"/charts/rendered/demo.html"},"privacy":{"redaction_status":"complete"},"lenses":{"L0":{"status":"present"}},"artifacts":[{"kind":"pixels","path":"pixels/light-800x600.png","size_bytes":8,"sha256":"%s"}]}}`, hex.EncodeToString(digest[:]))
		case "GET /api/v1/ux-capture/bundles/123e4567-e89b-12d3-a456-426614174000/artifacts/pixels/light-800x600.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pixels)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := scenarioBundleCaptureClient{baseURL: server.URL, httpClient: server.Client()}
	got, err := client.CaptureScenarioPixels(context.Background(), "chart-generator", "/charts/rendered/demo.html", 800, 600)
	if err != nil {
		t.Fatalf("capture pixels: %v", err)
	}
	if string(got) != string(pixels) {
		t.Fatalf("pixels = %q, want %q", got, pixels)
	}
}

func TestScenarioBundleCaptureClientRejectsDigestMismatch(t *testing.T) {
	pixels := []byte("\x89PNG\r\n\x1a\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/ux-capture/scenario":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"manifest":{"schema_version":"ux-capture.v1","id":"123e4567-e89b-12d3-a456-426614174000","source":{"kind":"scenario","scenario":"chart-generator","route":"/charts/rendered/demo.html"},"privacy":{"redaction_status":"complete"},"lenses":{"L0":{"status":"present"}},"artifacts":[{"kind":"pixels","path":"pixels/light-800x600.png","size_bytes":8,"sha256":"0000000000000000000000000000000000000000000000000000000000000000"}]}}`)
		case "GET /api/v1/ux-capture/bundles/123e4567-e89b-12d3-a456-426614174000/artifacts/pixels/light-800x600.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pixels)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := scenarioBundleCaptureClient{baseURL: server.URL, httpClient: server.Client()}
	_, err := client.CaptureScenarioPixels(context.Background(), "chart-generator", "/charts/rendered/demo.html", 800, 600)
	if err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("capture error = %v, want digest mismatch", err)
	}
}
