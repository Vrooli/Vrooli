package fetch

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
)

// writeTemp writes content to a temp file and returns its path.
func writeTemp(t *testing.T, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// onnxBytes returns a buffer that opens like an ONNX protobuf (ir_version tag
// 0x08) and is large enough to clear the size floor.
func onnxBytes() []byte {
	b := append([]byte{0x08, 0x06, 0x12, 0x07}, []byte("pytorch")...)
	return append(b, bytes.Repeat([]byte{0x00}, 2048)...)
}

func TestValidateArtifact_RejectsHTMLPage(t *testing.T) {
	// The exact install-stub symptom: a GitHub repo/release PAGE served as HTML.
	html := append([]byte("<!DOCTYPE html>\n<html><head><title>rembg</title></head><body>"), bytes.Repeat([]byte("x"), 2048)...)
	p := writeTemp(t, "page.onnx", html)
	err := ValidateArtifact(p, Asset{Filename: "page.onnx", Kind: ArtifactONNX})
	if !errors.Is(err, binaryfetch.ErrLooksLikeHTML) {
		t.Fatalf("expected ErrLooksLikeHTML for an HTML page, got %v", err)
	}
}

func TestValidateArtifact_RejectsHTMLEvenWhenGenericKind(t *testing.T) {
	html := append([]byte("<html><body>not a model</body>"), bytes.Repeat([]byte("x"), 2048)...)
	p := writeTemp(t, "page.bin", html)
	if err := ValidateArtifact(p, Asset{Filename: "page.bin", Kind: ArtifactGeneric}); !errors.Is(err, binaryfetch.ErrLooksLikeHTML) {
		t.Fatalf("HTML must be rejected for generic kind too, got %v", err)
	}
}

func TestValidateArtifact_RejectsTooSmall(t *testing.T) {
	p := writeTemp(t, "tiny.onnx", []byte("nope"))
	if err := ValidateArtifact(p, Asset{Filename: "tiny.onnx", Kind: ArtifactONNX}); !errors.Is(err, binaryfetch.ErrTooSmall) {
		t.Fatalf("expected ErrTooSmall, got %v", err)
	}
}

func TestValidateArtifact_RejectsBelowDeclaredMinBytes(t *testing.T) {
	// 4 KiB of real ONNX, but the asset declares it should be at least 1 MiB.
	p := writeTemp(t, "short.onnx", append(onnxBytes(), bytes.Repeat([]byte{0}, 4096)...))
	err := ValidateArtifact(p, Asset{Filename: "short.onnx", Kind: ArtifactONNX, MinBytes: 1 << 20})
	if !errors.Is(err, binaryfetch.ErrTooSmall) {
		t.Fatalf("expected ErrTooSmall vs declared min_bytes, got %v", err)
	}
}

func TestValidateSmallPinnedYAMLConfigBelowGenericWeightFloor(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	body := []byte("cond_image_size: 512\nrenderer:\n  radius: 0.87\n" + strings.Repeat("# pinned model architecture\n", 24))
	if err := os.WriteFile(p, body, 0600); err != nil {
		t.Fatal(err)
	}
	if len(body) >= int(binaryfetch.DefaultSizeFloor) {
		t.Fatalf("fixture must exercise config-specific size floor: %d", len(body))
	}
	if err := ValidateArtifact(p, Asset{Filename: "config.yaml", Kind: ArtifactConfig, MinBytes: int64(len(body))}); err != nil {
		t.Fatalf("valid small YAML config rejected: %v", err)
	}
	if err := os.WriteFile(p, []byte("<!doctype html><html>"+strings.Repeat("x", 1100)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifact(p, Asset{Filename: "config.yaml", Kind: ArtifactConfig, MinBytes: 128}); err == nil {
		t.Fatal("HTML was accepted as model config")
	}
}

func TestValidateArtifact_RejectsWrongONNXMagic(t *testing.T) {
	// Large, not HTML, but not an ONNX protobuf (first byte is not 0x08).
	junk := append([]byte("RIFF....WEBP"), bytes.Repeat([]byte{0x42}, 2048)...)
	p := writeTemp(t, "wrong.onnx", junk)
	if err := ValidateArtifact(p, Asset{Filename: "wrong.onnx", Kind: ArtifactONNX}); !errors.Is(err, ErrArtifactNotWeight) {
		t.Fatalf("expected ErrArtifactNotWeight for non-ONNX bytes, got %v", err)
	}
}

func TestValidateArtifact_AcceptsRealONNXShape(t *testing.T) {
	p := writeTemp(t, "u2netp.onnx", onnxBytes())
	if err := ValidateArtifact(p, Asset{Filename: "u2netp.onnx", Kind: ArtifactONNX}); err != nil {
		t.Fatalf("a real ONNX-shaped artifact should validate, got %v", err)
	}
}

func TestValidateArtifact_AcceptsGGUFAndSafetensors(t *testing.T) {
	gguf := append([]byte("GGUF"), bytes.Repeat([]byte{0}, 2048)...)
	if err := ValidateArtifact(writeTemp(t, "m.gguf", gguf), Asset{Filename: "m.gguf", Kind: ArtifactGGUF}); err != nil {
		t.Fatalf("GGUF should validate, got %v", err)
	}
	// safetensors: 8-byte header-len then '{' at offset 8.
	st := append([]byte{0x10, 0, 0, 0, 0, 0, 0, 0, '{'}, bytes.Repeat([]byte{' '}, 2048)...)
	if err := ValidateArtifact(writeTemp(t, "m.safetensors", st), Asset{Filename: "m.safetensors", Kind: ArtifactSafetensors}); err != nil {
		t.Fatalf("safetensors should validate, got %v", err)
	}
}

func TestValidateSmallPinnedJSONConfigAndRejectHubErrors(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	body := []byte(`{"architectures":["ViTForImageClassification"],"id2label":{"0":"0-2","1":"3-9","2":"10-19","3":"20-29"},"image_size":224,"model_type":"vit"` + strings.Repeat(" ", 200) + "}")
	if err := os.WriteFile(p, body, 0600); err != nil {
		t.Fatal(err)
	}
	if len(body) >= int(binaryfetch.DefaultSizeFloor) {
		t.Fatalf("fixture must stay below the generic weight floor: %d", len(body))
	}
	if err := ValidateArtifact(p, Asset{Filename: "config.json", Kind: ArtifactConfig, MinBytes: 128}); err != nil {
		t.Fatalf("valid small JSON config rejected: %v", err)
	}
	for name, bad := range map[string]string{
		"hub error": `{"error":"Access to model is restricted. You must have access to it and be authenticated."` + strings.Repeat(" ", 200) + "}",
		"truncated": `{"architectures":["ViT"],"image_size":224,` + strings.Repeat(`"k":1,`, 40),
	} {
		if err := os.WriteFile(p, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if err := ValidateArtifact(p, Asset{Filename: "config.json", Kind: ArtifactConfig, MinBytes: 128}); !errors.Is(err, ErrArtifactNotWeight) {
			t.Fatalf("%s: expected ErrArtifactNotWeight, got %v", name, err)
		}
	}
}

func TestValidateTextConfigThatOpensWithABracketIsNotJSON(t *testing.T) {
	vocab := []byte("[PAD]\n[unused1]\n[UNK]\n" + strings.Repeat("word\n", 400))
	p := filepath.Join(t.TempDir(), "vocab.txt")
	if err := os.WriteFile(p, vocab, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifact(p, Asset{Filename: "detector/vocab.txt", Kind: ArtifactConfig, MinBytes: 128}); err != nil {
		t.Fatalf("a bracket-led vocabulary file was treated as JSON: %v", err)
	}
	// The same bytes under a .json name are still refused.
	if err := os.WriteFile(filepath.Join(filepath.Dir(p), "vocab.json"), vocab, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateArtifact(filepath.Join(filepath.Dir(p), "vocab.json"), Asset{Filename: "vocab.json", Kind: ArtifactConfig, MinBytes: 128}); !errors.Is(err, ErrArtifactNotWeight) {
		t.Fatalf("invalid JSON under a .json name accepted: %v", err)
	}
}
