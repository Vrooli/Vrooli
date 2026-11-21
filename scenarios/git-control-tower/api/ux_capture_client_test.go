package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func TestCaptureScenarioBundleUsesDeclaredVerifiedArtifact(t *testing.T) {
	png := []byte("png-fixture")
	digest := sha256.Sum256(png)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/ux-capture/scenario":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"manifest":{"schema_version":"ux-capture.v1","id":"123e4567-e89b-12d3-a456-426614174000","source":{"kind":"scenario","scenario":"demo","route":"/"},"privacy":{"redaction_status":"complete"},"lenses":{"L0":{"status":"present"}},"artifacts":[{"kind":"full-page-pixels","path":"pixels/full-page-light-1280x720.png","size_bytes":%d,"sha256":"%s"}]}}`, len(png), hex.EncodeToString(digest[:]))
		case "GET /api/v1/ux-capture/bundles/123e4567-e89b-12d3-a456-426614174000/artifacts/pixels/full-page-light-1280x720.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewBrowserAutomationClient(time.Second)
	client.resolver = discovery.NewStaticResolver(server.URL)
	got, err := client.CaptureScenarioBundle(context.Background(), "demo", "/", CapturePreset{Width: 1280, Height: 720, Theme: "light"}, "#root")
	if err != nil {
		t.Fatalf("capture bundle: %v", err)
	}
	if string(got) != string(png) {
		t.Fatalf("artifact = %q, want %q", got, png)
	}
}
