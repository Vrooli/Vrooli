package photographic

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestSubmitPinsLocalOneImageRequestAndOperationIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(apihttp.TestModeHeader); got != "" {
			t.Errorf("ordinary request gained test-mode header %q", got)
		}
		if r.URL.Path != "/api/v1/ai/text_to_image" || r.Header.Get("Idempotency-Key") != "f7f4f09f-a93a-4f34-93ef-4fa814891f5b" {
			t.Fatalf("request path/header: %s %q", r.URL.Path, r.Header.Get("Idempotency-Key"))
		}
		if err := r.ParseMultipartForm(16 << 10); err != nil {
			t.Fatal(err)
		}
		var params aiv1.AIParams
		if err := protojson.Unmarshal([]byte(r.FormValue("params")), &params); err != nil {
			t.Fatal(err)
		}
		if params.GetPrompt() != "a bright kitchen" || params.GetModelOverride() != "sd-1.5" || params.GetAllowByok() || params.GetFallbackPolicy() != "local_only" || params.GetAllowReclaim() || params.GetVariations() != 1 || params.GetWidth() != 768 || params.GetHeight() != 512 {
			t.Fatalf("unsafe or unexpected params: %+v", params)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"owner-job","modelId":"sd-1.5","tier":"local"}`))
	}))
	defer server.Close()
	c := NewClient(func(_ context.Context, owner string) (string, error) {
		if owner != "image-tools" {
			t.Fatalf("resolved %q", owner)
		}
		return server.URL, nil
	}, server.Client())
	op, status, err := c.Submit(context.Background(), "f7f4f09f-a93a-4f34-93ef-4fa814891f5b", "a bright kitchen", 768, 512)
	if err != nil || status != 202 || op.JobID != "owner-job" || op.State != "queued" {
		t.Fatalf("submit=%+v status=%d err=%v", op, status, err)
	}
}

func TestSubmitPropagatesRoutedTestModeToImageTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(apihttp.TestModeHeader); got != apihttp.TestModeValue {
			t.Errorf("owner test-mode header=%q, want %q", got, apihttp.TestModeValue)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"jobId":"routed-owner-job"}`))
	}))
	defer server.Close()
	c := NewClient(func(context.Context, string) (string, error) { return server.URL, nil }, server.Client())
	if _, _, err := c.Submit(database.WithTestMode(context.Background()), "op-routed", "a scene", 768, 512); err != nil {
		t.Fatal(err)
	}
}

func TestPNGRequiresManagedBoundedExactDimensionResult(t *testing.T) {
	var encoded bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 768, 512))
	img.Set(0, 0, color.RGBA{R: 10, A: 255})
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/blobs/out/result.png" {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(encoded.Bytes())
	}))
	defer s.Close()
	c := NewClient(func(context.Context, string) (string, error) { return s.URL, nil }, s.Client())
	if _, err := c.PNG(context.Background(), "out/result.png", 768, 512); err != nil {
		t.Fatal(err)
	}
	if _, err := c.PNG(context.Background(), "out/result.png", 512, 768); err == nil {
		t.Fatal("wrong dimensions accepted")
	}
	if _, err := c.PNG(context.Background(), "out/../secret.png", 768, 512); err == nil {
		t.Fatal("traversal reference accepted")
	}
}

func TestRequestBounds(t *testing.T) {
	for _, tc := range []struct {
		p    string
		w, h int
		want bool
	}{{"ok", 768, 512, true}, {"ok", 512, 768, true}, {"", 768, 512, false}, {string(bytes.Repeat([]byte("x"), 801)), 768, 512, false}, {"ok", 512, 512, false}} {
		if err := validateRequest(tc.p, tc.w, tc.h); (err == nil) != tc.want {
			t.Fatalf("validate(%d,%d) err=%v", tc.w, tc.h, err)
		}
	}
}

func TestSubmitAndLookupRejectOwnerHTTPAndMalformedReceipts(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"submit rejected", ``, http.StatusBadRequest},
		{"submit malformed", `{`, http.StatusAccepted},
		{"submit missing id", `{}`, http.StatusAccepted},
		{"lookup unavailable", ``, http.StatusServiceUnavailable},
		{"lookup malformed", `{`, http.StatusOK},
		{"lookup missing id", `{}`, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer s.Close()
			c := NewClient(func(context.Context, string) (string, error) { return s.URL, nil }, s.Client())
			if len(tc.name) >= 6 && tc.name[:6] == "submit" {
				_, status, err := c.Submit(context.Background(), "op-1", "scene", 768, 512)
				wantStatus := 0
				if tc.status != http.StatusAccepted {
					wantStatus = tc.status
				}
				if err == nil || status != wantStatus {
					t.Fatalf("submit status=%d want=%d err=%v", status, wantStatus, err)
				}
				return
			}
			_, err := c.Lookup(context.Background(), "op-1")
			if err == nil {
				t.Fatal("invalid or failed owner reply was accepted")
			}
			if tc.status == http.StatusServiceUnavailable {
				ownerErr, ok := err.(OwnerHTTPError)
				if !ok || ownerErr.Status != tc.status {
					t.Fatalf("lookup error=%T %v", err, err)
				}
			}
		})
	}
}

func TestLookupAndPNGRejectInvalidOwnerContent(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		status            int
		body              []byte
		length            int64
	}{
		{"content status", "image/png", http.StatusNotFound, nil, 0},
		{"oversized declared", "image/png", http.StatusOK, nil, MaxPNGBytes + 1},
		{"wrong type", "text/plain", http.StatusOK, []byte("not png"), 0},
		{"empty", "image/png", http.StatusOK, nil, 0},
		{"malformed png", "image/png", http.StatusOK, []byte("not png"), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				if tc.length > 0 {
					w.Header().Set("Content-Length", fmt.Sprint(tc.length))
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.body)
			}))
			defer s.Close()
			c := NewClient(func(context.Context, string) (string, error) { return s.URL, nil }, s.Client())
			if _, err := c.PNG(context.Background(), "out/result.png", 768, 512); err == nil {
				t.Fatal("invalid owner content was accepted")
			}
		})
	}
}

func TestWaitAndCancelPropagateOwnerErrors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer s.Close()
	c := NewClient(func(context.Context, string) (string, error) { return s.URL, nil }, s.Client())
	if _, err := c.Wait(context.Background(), "job-1"); err == nil {
		t.Fatal("wait swallowed owner failure")
	}
	if _, err := c.Cancel(context.Background(), "job-1"); err == nil {
		t.Fatal("cancel swallowed owner failure")
	}
}
