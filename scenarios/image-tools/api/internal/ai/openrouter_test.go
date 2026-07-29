package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
)

func imageResponseBody(t *testing.T, image []byte) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"data": []any{map[string]any{"b64_json": base64.StdEncoding.EncodeToString(image), "media_type": "image/png"}}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func testOpenRouter(t *testing.T, generate imageGenerator) *openRouterProvider {
	t.Helper()
	return &openRouterProvider{resolveRole: func(context.Context, string) (resolvedImageRole, error) {
		return resolvedImageRole{Model: "resource-resolved-model"}, nil
	}, generateImage: generate}
}

func TestOpenRouter_UsesResourceOwnedTransportAndWritesImage(t *testing.T) {
	want := []byte("resource-image")
	called := false
	p := testOpenRouter(t, func(_ context.Context, role, prompt, inputFile string, count int) ([]byte, error) {
		called = true
		if role != "image.generate.logo" || prompt != "logo" || inputFile != "" || count != 1 {
			t.Fatalf("resource args = %q %q %q %d", role, prompt, inputFile, count)
		}
		return imageResponseBody(t, want), nil
	})
	out := filepath.Join(t.TempDir(), "out.png")
	result, err := p.Execute(context.Background(), backends.Request{Operation: "text_to_image", Params: map[string]string{"prompt": "logo", "openrouter_role": "image.generate.logo"}, Output: storage.OutputTarget{LocalPath: out}})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("resource transport was not called")
	}
	if result.Meta["model"] != "resource-resolved-model" || result.Meta["role"] != "image.generate.logo" {
		t.Fatalf("metadata = %#v", result.Meta)
	}
	got, err := os.ReadFile(out)
	if err != nil || string(got) != string(want) {
		t.Fatalf("output = %q, err = %v", got, err)
	}
}

func TestOpenRouter_RejectsMissingTransportAndPrompt(t *testing.T) {
	p := &openRouterProvider{}
	if p.Available(context.Background()) {
		t.Fatal("missing resource transport must be unavailable")
	}
	_, err := p.Execute(context.Background(), backends.Request{Operation: "text_to_image", Params: map[string]string{"prompt": "x"}, Output: storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "out.png")}})
	if err == nil {
		t.Fatal("expected missing transport error")
	}
	p = testOpenRouter(t, func(context.Context, string, string, string, int) ([]byte, error) {
		t.Fatal("must not dispatch empty prompt")
		return nil, nil
	})
	_, err = p.Execute(context.Background(), backends.Request{Operation: "text_to_image", Output: storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "out.png")}})
	if err == nil {
		t.Fatal("expected missing prompt error")
	}
}

func TestOpenRouter_ForwardsSourceImageForEdits(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := testOpenRouter(t, func(_ context.Context, role, prompt, inputFile string, count int) ([]byte, error) {
		if role != "image.edit.default" || prompt != "make it brighter" || inputFile != source || count != 1 {
			t.Fatalf("resource args = %q %q %q %d", role, prompt, inputFile, count)
		}
		return imageResponseBody(t, []byte("edited")), nil
	})
	out := filepath.Join(t.TempDir(), "out.png")
	_, err := p.Execute(context.Background(), backends.Request{
		Operation: "edit_instruct",
		Params:    map[string]string{"prompt": "make it brighter"},
		InputKeys: []string{source},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRoleForRequest(t *testing.T) {
	if got := roleForRequest(backends.Request{Operation: "text_to_image"}); got != "image.generate.default" {
		t.Fatalf("generate role = %q", got)
	}
	if got := roleForRequest(backends.Request{Operation: "edit_instruct"}); got != "image.edit.default" {
		t.Fatalf("edit role = %q", got)
	}
	if got := roleForRequest(backends.Request{Params: map[string]string{"openrouter_role": "image.generate.logo"}}); got != "image.generate.logo" {
		t.Fatalf("explicit role = %q", got)
	}
}

func TestDecodeOpenRouterImageRejectsEmpty(t *testing.T) {
	if _, _, err := decodeOpenRouterImage([]byte(`{"data":[]}`)); err == nil {
		t.Fatal("expected empty data error")
	}
}
