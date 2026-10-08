package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/storage"
)

type fakeMediaGateway struct {
	request mediaGenerationRequest
	result  mediaGenerationResult
	err     error
}

func (f *fakeMediaGateway) Generate(_ context.Context, req mediaGenerationRequest) (mediaGenerationResult, error) {
	f.request = req
	return f.result, f.err
}

func TestAIGatewayProvider_DelegatesIntentAndPreservesGatewayOutput(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.svg")
	if err := os.WriteFile(out, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 8 8"></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/svg+xml", Model: "recraft/recraft-v4.1-vector"}}
	provider := newAIGatewayProvider(media)

	result, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "a precise mark", "openrouter_role": "image.generate.logo"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatal(err)
	}
	if media.request.Operation != "text_to_image" || media.request.Role != "image.generate.logo" || media.request.Prompt != "a precise mark" {
		t.Fatalf("gateway intent = %#v", media.request)
	}
	if media.request.OutputPath != out || media.request.InputFile != "" {
		t.Fatalf("gateway paths = %#v", media.request)
	}
	if result.Tier != backends.TierBYOK || result.Meta["gateway"] != "ai-gateway" || result.Meta["media_type"] != "image/svg+xml" {
		t.Fatalf("result = %#v", result)
	}
	if result.Meta["model"] != "recraft/recraft-v4.1-vector" {
		t.Fatalf("resolved model metadata = %#v", result.Meta)
	}
}

func TestAIGatewayProvider_ForwardsEditInput(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(source, []byte("\x89PNG\r\n\x1a\nsource"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.png")
	if err := os.WriteFile(out, []byte("output"), 0o600); err != nil {
		t.Fatal(err)
	}
	media := &fakeMediaGateway{result: mediaGenerationResult{MediaType: "image/png"}}
	provider := newAIGatewayProvider(media)

	_, err := provider.Execute(context.Background(), backends.Request{
		Operation: "edit_instruct",
		InputKeys: []string{source},
		Params:    map[string]string{"prompt": "make it brighter"},
		Output:    storage.OutputTarget{LocalPath: out},
	})
	if err != nil {
		t.Fatal(err)
	}
	if media.request.Role != "image.edit.default" || media.request.InputFile != source {
		t.Fatalf("gateway edit intent = %#v", media.request)
	}
}

func TestAIGatewayProvider_RejectsMissingOutputAndEmptyGatewayOutput(t *testing.T) {
	provider := newAIGatewayProvider(&fakeMediaGateway{})
	if _, err := provider.Execute(context.Background(), backends.Request{Operation: "text_to_image", Params: map[string]string{"prompt": "x"}}); err == nil {
		t.Fatal("expected missing output path error")
	}

	out := filepath.Join(t.TempDir(), "empty.png")
	if err := os.WriteFile(out, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Execute(context.Background(), backends.Request{
		Operation: "text_to_image",
		Params:    map[string]string{"prompt": "x"},
		Output:    storage.OutputTarget{LocalPath: out},
	}); err == nil {
		t.Fatal("expected empty gateway output error")
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
