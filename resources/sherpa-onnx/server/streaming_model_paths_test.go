package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStreamingModelFile(t *testing.T) {
	t.Run("keeps the declared dated model file when present", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"encoder-epoch.onnx", "encoder.onnx"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("model"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		got := resolveStreamingModelFile(dir, "", "encoder-epoch.onnx", "encoder.onnx")
		if want := filepath.Join(dir, "encoder-epoch.onnx"); got != want {
			t.Fatalf("resolved path = %q, want preferred model file %q", got, want)
		}
	})

	t.Run("selects generic files from compact model archives", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "encoder.onnx"), []byte("model"), 0o600); err != nil {
			t.Fatal(err)
		}
		got := resolveStreamingModelFile(dir, "", "encoder-epoch.onnx", "encoder.onnx")
		if want := filepath.Join(dir, "encoder.onnx"); got != want {
			t.Fatalf("resolved path = %q, want generic archive file %q", got, want)
		}
	})

	t.Run("selects the 20M Zipformer int8 encoder and joiner with fp32 decoder", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{
			"encoder-epoch-99-avg-1.int8.onnx",
			"decoder-epoch-99-avg-1.onnx",
			"joiner-epoch-99-avg-1.int8.onnx",
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("model"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		for _, tc := range []struct{ role, preferred, want string }{
			{"encoder", "encoder-epoch-99-avg-1-chunk-16-left-128.onnx", "encoder-epoch-99-avg-1.int8.onnx"},
			{"decoder", "decoder-epoch-99-avg-1-chunk-16-left-128.onnx", "decoder-epoch-99-avg-1.onnx"},
			{"joiner", "joiner-epoch-99-avg-1-chunk-16-left-128.onnx", "joiner-epoch-99-avg-1.int8.onnx"},
		} {
			got := resolveStreamingModelFile(dir, "", tc.preferred, tc.want, tc.role+".onnx")
			if want := filepath.Join(dir, tc.want); got != want {
				t.Errorf("%s resolved path = %q, want %q", tc.role, got, want)
			}
		}
	})

	t.Run("explicit path wins", func(t *testing.T) {
		dir := t.TempDir()
		override := filepath.Join(dir, "selected.onnx")
		got := resolveStreamingModelFile(dir, override, "encoder-epoch.onnx", "encoder.onnx")
		if got != override {
			t.Fatalf("resolved path = %q, want explicit override %q", got, override)
		}
	})

	t.Run("returns the preferred path for the caller's normal missing-file error", func(t *testing.T) {
		dir := t.TempDir()
		got := resolveStreamingModelFile(dir, "", "encoder-epoch.onnx", "encoder.onnx")
		if want := filepath.Join(dir, "encoder-epoch.onnx"); got != want {
			t.Fatalf("resolved path = %q, want %q", got, want)
		}
	})
}
