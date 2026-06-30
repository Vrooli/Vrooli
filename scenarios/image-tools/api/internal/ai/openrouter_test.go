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

func fakeImageRole(model string) func(context.Context, string) (resolvedImageRole, error) {
	return func(_ context.Context, _ string) (resolvedImageRole, error) {
		r := resolvedImageRole{Model: model, Endpoint: "images"}
		r.RequestDefaults.OutputFormat = "png"
		return r, nil
	}
}

func newTestOpenRouter(t *testing.T, handler http.HandlerFunc) (*openRouterProvider, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	p := &openRouterProvider{
		httpClient:  srv.Client(),
		baseURL:     srv.URL,
		resolveKey:  func(context.Context) string { return fakeKey },
		resolveRole: fakeImageRole("test/image-model"),
	}
	return p, srv.Close
}

// imageResponseBody builds an OpenRouter /api/v1/images success body.
func imageResponseBody(t *testing.T, png []byte) []byte {
	t.Helper()
	resp := map[string]any{
		"data": []any{
			map[string]any{"b64_json": base64.StdEncoding.EncodeToString(png)},
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
		if body["model"] != "test/image-model" {
			t.Errorf("request model = %v, want test/image-model", body["model"])
		}
		if body["prompt"] == nil {
			t.Error("request missing prompt")
		}
		if body["output_format"] != "png" {
			t.Errorf("request output_format = %v, want png (from request_defaults)", body["output_format"])
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
	if res.Meta["model"] != "test/image-model" || res.Meta["role"] != "image.generate.default" {
		t.Errorf("meta = %v", res.Meta)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(got) != string(wantPNG) {
		t.Errorf("written bytes mismatch")
	}
}

func TestOpenRouter_ExplicitRoleParam(t *testing.T) {
	var sawModel string
	p, closeFn := newTestOpenRouter(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		sawModel, _ = body["model"].(string)
		_, _ = w.Write(imageResponseBody(t, []byte("PNG")))
	})
	// the role param drives which role is resolved; the fake echoes a role-specific model
	p.resolveRole = func(_ context.Context, role string) (resolvedImageRole, error) {
		return resolvedImageRole{Model: "model-for/" + role, Endpoint: "images"}, nil
	}
	defer closeFn()

	out := filepath.Join(t.TempDir(), "out.png")
	res, err := p.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "logo", "openrouter_role": "image.generate.logo"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if sawModel != "model-for/image.generate.logo" {
		t.Errorf("explicit role not honored: model = %q", sawModel)
	}
	if res.Meta["role"] != "image.generate.logo" {
		t.Errorf("meta role = %q", res.Meta["role"])
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
			InputReferences []map[string]any `json:"input_references"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, ref := range body.InputReferences {
			if ref["type"] == "image_url" {
				sawImage = true
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
		t.Error("edit request did not include the source image via input_references")
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
		_, _ = w.Write([]byte(`{"data":[]}`))
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

func TestRoleForRequest(t *testing.T) {
	cases := []struct {
		op     string
		params map[string]string
		want   string
	}{
		{"text_to_image", nil, "image.generate.default"},
		{"edit_instruct", nil, "image.edit.default"},
		{"image_to_image", nil, "image.edit.default"},
		{"text_to_image", map[string]string{"openrouter_role": "image.generate.logo"}, "image.generate.logo"},
		{"edit_instruct", map[string]string{"openrouter_role": "image.edit.identity"}, "image.edit.identity"},
	}
	for _, c := range cases {
		got := roleForRequest(backends.Request{Operation: c.op, Params: c.params})
		if got != c.want {
			t.Errorf("roleForRequest(%q,%v) = %q, want %q", c.op, c.params, got, c.want)
		}
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
