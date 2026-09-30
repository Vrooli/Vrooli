package fetch

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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
