package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
)

const fakeKey = "sk-or-v1-0000000000000000000000000000000000000000"

func newTestOpenRouter(t *testing.T, handler http.HandlerFunc) (*openRouterProvider, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	p := &openRouterProvider{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
		model:      "test/image-model",
		resolveKey: func(context.Context) string { return fakeKey },
	}
	return p, srv.Close
}

func imageResponseBody(t *testing.T, png []byte) []byte {
	t.Helper()
	resp := map[string]any{
		"choices": []any{
			map[string]any{
				"message": map[string]any{
					"images": []any{
						map[string]any{"image_url": map[string]any{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)}},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestOpenRouter_AvailableGatesOnKey(t *testing.T) {
	p := &openRouterProvider{resolveKey: func(context.Context) string { return "" }}
	if p.Available(context.Background()) {
		t.Fatal("expected unavailable with no key")
	}
	p.resolveKey = func(context.Context) string { return "not-a-real-key" }
	if p.Available(context.Background()) {
		t.Fatal("expected unavailable with a non-sk-or key")
	}
	p.resolveKey = func(context.Context) string { return fakeKey }
	if !p.Available(context.Background()) {
		t.Fatal("expected available with a usable key")
	}
}

func TestOpenRouter_IsCloudLastTier(t *testing.T) {
	p := newOpenRouterProvider()
	if !p.IsCloud() {
		t.Fatal("openrouter must report IsCloud=true")
	}
	if !p.Standalone() {
		t.Fatal("openrouter does not need ComfyUI")
	}
}

func TestOpenRouter_GenerateWritesImage(t *testing.T) {
	wantPNG := []byte("\x89PNG\r\n\x1a\nFAKE")
	p, closeFn := newTestOpenRouter(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+fakeKey {
			t.Errorf("authorization header = %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["modalities"]; !ok {
			t.Error("request missing image modality")
		}
		_, _ = w.Write(imageResponseBody(t, wantPNG))
	})
	defer closeFn()

	out := filepath.Join(t.TempDir(), "out.png")
	res, err := p.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "a minimal logo"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Tier != backends.TierBYOK {
		t.Errorf("tier = %v, want byok-cloud", res.Tier)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(got) != string(wantPNG) {
		t.Errorf("written bytes mismatch")
	}
}

func TestOpenRouter_EditSendsInputImage(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.png")
	if err := os.WriteFile(src, []byte("\x89PNGsource"), 0o644); err != nil {
		t.Fatal(err)
	}
	sawImage := false
	p, closeFn := newTestOpenRouter(t, func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Content []map[string]any `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, m := range body.Messages {
			for _, c := range m.Content {
				if c["type"] == "image_url" {
					sawImage = true
				}
			}
		}
		_, _ = w.Write(imageResponseBody(t, []byte("EDITED")))
	})
	defer closeFn()

	out := filepath.Join(t.TempDir(), "out.png")
	_, err := p.Execute(context.Background(), backends.Request{
		Operation: "edit_instruct",
		InputKeys: []string{src},
		Params:    map[string]string{"prompt": "make it navy"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !sawImage {
		t.Error("edit request did not include the source image")
	}
}

func TestOpenRouter_ErrorStatusSurfaces(t *testing.T) {
	p, closeFn := newTestOpenRouter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"message":"insufficient credits"}}`))
	})
	defer closeFn()

	_, err := p.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "x"},
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "o.png")},
	})
	if err == nil {
		t.Fatal("expected error on non-200")
	}
}

func TestOpenRouter_NoImageInResponse(t *testing.T) {
	p, closeFn := newTestOpenRouter(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"images":[]}}]}`))
	})
	defer closeFn()

	_, err := p.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "x"},
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "o.png")},
	})
	if err == nil {
		t.Fatal("expected error when the model returns no image")
	}
}

func TestOpenRouter_RequiresPrompt(t *testing.T) {
	p, closeFn := newTestOpenRouter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach the server without a prompt")
	})
	defer closeFn()

	_, err := p.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "o.png")},
	})
	if err == nil {
		t.Fatal("expected error with no prompt")
	}
}

func TestSanitizeModelSlug(t *testing.T) {
	cases := map[string]string{
		"":                          "",
		"  ":                        "",
		"${OPENROUTER_IMAGE_MODEL}": "", // unsubstituted lifecycle placeholder
		" google/gemini-2.5-flash ": "google/gemini-2.5-flash",
		"openai/some-image-model":   "openai/some-image-model",
	}
	for in, want := range cases {
		if got := sanitizeModelSlug(in); got != want {
			t.Errorf("sanitizeModelSlug(%q) = %q, want %q", in, got, want)
		}
	}
	// An unset placeholder resolves to the default model, not a bogus slug.
	t.Setenv("OPENROUTER_IMAGE_MODEL", "${OPENROUTER_IMAGE_MODEL}")
	if p := newOpenRouterProvider(); p.model != defaultOpenRouterImageModel {
		t.Errorf("placeholder model = %q, want default %q", p.model, defaultOpenRouterImageModel)
	}
}

func TestParseExportedKey(t *testing.T) {
	out := "export OPENROUTER_API_KEY=\"sk-or-v1-abc\"\nexport OTHER=1\n"
	if got := parseExportedKey(out, "OPENROUTER_API_KEY"); got != "sk-or-v1-abc" {
		t.Errorf("parseExportedKey = %q", got)
	}
}

// TestOpenRouter_LiveSmoke is a real BYOK round-trip, gated on a configured key
// AND an explicit opt-in (it spends money). Run with:
//
//	IMAGE_TOOLS_OPENROUTER_LIVE=1 go test ./internal/ai -run LiveSmoke
func TestOpenRouter_LiveSmoke(t *testing.T) {
	if os.Getenv("IMAGE_TOOLS_OPENROUTER_LIVE") == "" {
		t.Skip("set IMAGE_TOOLS_OPENROUTER_LIVE=1 to run the metered live OpenRouter smoke test")
	}
	p := newOpenRouterProvider()
	if !p.Available(context.Background()) {
		t.Skip("no usable OPENROUTER_API_KEY configured")
	}
	out := filepath.Join(t.TempDir(), "live.png")
	if _, err := p.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "a minimal flat vector logo of a blue circle on white"},
		Output:    storage.OutputTarget{LocalPath: out},
	}); err != nil {
		t.Fatalf("live generate: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() == 0 {
		t.Fatalf("live generate produced no image: %v", err)
	}
}
