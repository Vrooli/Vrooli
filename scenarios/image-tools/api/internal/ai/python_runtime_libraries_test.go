package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferenceCommandAddsPrivateNvidiaLibraryDirectories(t *testing.T) {
	t.Setenv("LD_LIBRARY_PATH", "/host/cuda/lib")
	venv := t.TempDir()
	lib := filepath.Join(venv, "lib", "python3.12", "site-packages", "nvidia", "cu13", "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	python := filepath.Join(venv, "bin", "python")
	cmd := inferenceCommand(withPythonRuntimeLibraries(context.Background(), python), "/bin/true")
	for _, value := range cmd.Env {
		if strings.HasPrefix(value, "LD_LIBRARY_PATH=") {
			got := strings.TrimPrefix(value, "LD_LIBRARY_PATH=")
			if !strings.HasPrefix(got, lib+string(os.PathListSeparator)) || !strings.HasSuffix(got, "/host/cuda/lib") {
				t.Fatalf("LD_LIBRARY_PATH = %q", got)
			}
			return
		}
	}
	t.Fatal("inference command omitted LD_LIBRARY_PATH")
}

func TestProbeONNXCUDAUsesPrivateNvidiaLibraryDirectories(t *testing.T) {
	t.Setenv("CUDA_VISIBLE_DEVICES", "GPU-probe-test")
	venv := t.TempDir()
	lib := filepath.Join(venv, "lib", "python3.12", "site-packages", "nvidia", "cu13", "lib")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	python := filepath.Join(venv, "bin", "python")
	if err := os.MkdirAll(filepath.Dir(python), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ncase \":$LD_LIBRARY_PATH:\" in *\"" + lib + "\"*) if [ \"$CUDA_VISIBLE_DEVICES\" = GPU-probe-test ]; then echo cuda-ready; else echo cpu-only; fi;; *) echo cpu-only;; esac\n"
	if err := os.WriteFile(python, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if !probeONNXCUDA(context.Background(), python) {
		t.Fatal("ONNX CUDA probe did not receive the venv's private CUDA library path")
	}
}
