package sidecar

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialize_WritesPackage(t *testing.T) {
	root := t.TempDir()
	got, err := Materialize(root)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if got != root {
		t.Fatalf("expected PYTHONPATH root %q, got %q", root, got)
	}
	for _, f := range []string{
		"__init__.py",
		"_common.py",
		"bg_removal.py",
		"colorize.py",
		"deblur.py",
		"denoise.py",
		"depth.py",
		"detect.py",
		"embedding.py",
		"face_detection.py",
		"nsfw.py",
		"restore.py",
		"sam2.py",
		"segment.py",
		"tagging.py",
		"worker.py",
		"_containment.py",
	} {
		p := filepath.Join(root, PackageName, f)
		info, statErr := os.Stat(p)
		if statErr != nil {
			t.Fatalf("expected %s materialized: %v", f, statErr)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", f)
		}
	}
}

func TestMaterialize_WritesInterpreterStartGuard(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if info, err := os.Stat(filepath.Join(root, "sitecustomize.py")); err != nil || info.Size() == 0 {
		t.Fatalf("the interpreter-start guard must sit on PYTHONPATH beside the package: %v", err)
	}
}

func containmentEnv(t *testing.T, mode string) []string {
	t.Helper()
	env := pythonSidecarEnv(t)
	if mode != "" {
		env = append(env, "IMAGE_TOOLS_SIDECAR_NETWORK="+mode)
	}
	return env
}

// A library that ignores every cache variable and simply opens a socket (urllib,
// requests, torch.hub, insightface, rembg) must be refused during inference,
// without the process ever importing image_tools_sidecar itself: the guard
// arrives through sitecustomize.
func TestPythonSidecar_InferenceProcessesCannotReachTheNetwork(t *testing.T) {
	script := `
import socket, urllib.request, sys
from_libs = []
for attempt in (
    lambda: socket.create_connection(("203.0.113.9", 80), timeout=1),
    lambda: socket.getaddrinfo("huggingface.co", 443),
    lambda: urllib.request.urlopen("https://github.com/deepinsight/insightface/releases/download/v0.7/buffalo_l.zip", timeout=1),
):
    try:
        attempt()
    except OSError as exc:
        text = str(exc) + str(getattr(exc, "reason", ""))
        assert "image-tools blocked a network connection" in text, text
        assert "governed install step" in text, text
    else:
        raise AssertionError("network access was not refused")
`
	cmd := exec.Command("python3", "-c", script)
	cmd.Env = containmentEnv(t, "blocked")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("inference containment: %v\n%s", err, out)
	}
}

func TestPythonSidecar_LoopbackAndAcquisitionAreNotBlocked(t *testing.T) {
	script := `
import socket
from image_tools_sidecar import _containment
listener = socket.socket(); listener.bind(("127.0.0.1", 0)); listener.listen(1)
client = socket.create_connection(listener.getsockname(), timeout=2)
client.close(); listener.close()
import os
if os.environ.get("IMAGE_TOOLS_SIDECAR_NETWORK") == "governed-acquisition":
    assert not _containment._installed, "the governed acquisition step must keep network access"
else:
    assert _containment._installed
`
	for _, mode := range []string{"blocked", "governed-acquisition"} {
		cmd := exec.Command("python3", "-c", script)
		cmd.Env = containmentEnv(t, mode)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("mode %s: %v\n%s", mode, err, out)
		}
	}
}

func TestPythonSidecar_SAM2PromptContracts(t *testing.T) {
	cmd := exec.Command("python3", "-m", "image_tools_sidecar.test_sam2", "-v")
	cmd.Env = pythonSidecarEnv(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("SAM 2 prompt contracts: %v\n%s", err, out)
	}
}

func TestPythonSidecar_SessionCacheReusesModelPath(t *testing.T) {
	script := `
from image_tools_sidecar import _common

class FakeOrt:
    calls = 0
    class ExecutionMode:
        ORT_SEQUENTIAL = "sequential"
    class SessionOptions:
        pass

    @staticmethod
    def InferenceSession(path, sess_options, providers):
        FakeOrt.calls += 1
        assert sess_options.intra_op_num_threads > 0
        assert sess_options.inter_op_num_threads == 1
        assert sess_options.execution_mode == "sequential"
        return {"path": path, "providers": tuple(providers), "ordinal": FakeOrt.calls}

_common.require_deps = lambda: (None, FakeOrt, None)

first = _common.make_session("/models/a/model.onnx")
second = _common.make_session("/models/a/model.onnx")
third = _common.make_session("/models/b/model.onnx")

assert first is second
assert third is not first
assert FakeOrt.calls == 2, FakeOrt.calls
assert first["providers"] == ("CPUExecutionProvider",)
`
	path := filepath.Join(t.TempDir(), "session_cache.py")
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command("python3", path)
	cmd.Env = pythonSidecarEnv(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python sidecar session cache contract failed: %v\n%s", err, out)
	}
}

func TestPythonSidecar_SessionCacheIsBounded(t *testing.T) {
	script := `
from image_tools_sidecar import _common
class FakeOrt:
    class ExecutionMode:
        ORT_SEQUENTIAL = "sequential"
    class SessionOptions:
        pass
    @staticmethod
    def InferenceSession(path, sess_options, providers):
        return object()
_common.require_deps = lambda: (None, FakeOrt, None)
for i in range(10):
    _common.make_session(str(i))
assert _common._MAX_SESSIONS >= 4, 'face_swap keeps four graphs (swapper, embedder, detector, occluder) resident'
assert len(_common._SESSION_CACHE) <= _common._MAX_SESSIONS, len(_common._SESSION_CACHE)
assert any(key.endswith(":9") for key in _common._SESSION_CACHE)
`
	cmd := exec.Command("python3", "-c", script)
	cmd.Env = pythonSidecarEnv(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bounded model residency: %v\n%s", err, out)
	}
}

func TestPythonSidecar_SAMExportContracts(t *testing.T) {
	cmd := exec.Command("python3", "-m", "image_tools_sidecar.test_segment", "-v")
	cmd.Env = pythonSidecarEnv(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("SAM export contracts: %v\n%s", err, out)
	}
}

func TestPythonSidecar_CPUFallbackDoesNotProbeOrUseCUDA(t *testing.T) {
	script := `
import os
from image_tools_sidecar import _common
class CUDA:
    @staticmethod
    def is_available():
        raise AssertionError("CPU fallback probed CUDA")
class Torch:
    cuda = CUDA
os.environ["IMAGE_TOOLS_DEVICE"] = "cpu"
assert not _common.use_cuda(Torch)
CUDA.is_available = staticmethod(lambda: False)
os.environ["IMAGE_TOOLS_DEVICE"] = "cuda"
try:
    _common.use_cuda(Torch)
except RuntimeError:
    pass
else:
    raise AssertionError("unavailable GPU silently accepted")
`
	cmd := exec.Command("python3", "-c", script)
	cmd.Env = pythonSidecarEnv(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("device fallback: %v\n%s", err, out)
	}
}

func TestMaterialize_Idempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root); err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	if _, err := Materialize(root); err != nil {
		t.Fatalf("second materialize should be idempotent: %v", err)
	}
}

func TestMaterialize_EmptyRootRejected(t *testing.T) {
	if _, err := Materialize(""); err == nil {
		t.Fatalf("expected error for empty root")
	}
}

func TestEnsureOnPath_PrependsAndDedupes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PYTHONPATH", "/some/existing")
	path, err := EnsureOnPath(root)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got := os.Getenv("PYTHONPATH")
	if !strings.HasPrefix(got, path+string(os.PathListSeparator)) {
		t.Fatalf("expected sidecar prepended, got %q", got)
	}
	if !strings.Contains(got, "/some/existing") {
		t.Fatalf("expected existing PYTHONPATH preserved, got %q", got)
	}
	// A second call must not duplicate the entry.
	if _, err := EnsureOnPath(root); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if c := strings.Count(os.Getenv("PYTHONPATH"), path); c != 1 {
		t.Fatalf("sidecar path should appear exactly once, got %d in %q", c, os.Getenv("PYTHONPATH"))
	}
}

func TestPythonSidecar_GoldenPostprocessingContracts(t *testing.T) {
	script := `
import numpy as np
from PIL import Image

from image_tools_sidecar import bg_removal, colorize, deblur, depth, detect, embedding, nsfw, segment, tagging

def close(got, want, tol=1e-5):
    if abs(float(got) - float(want)) > tol:
        raise AssertionError(f"got {got}, want {want}")

img = Image.new("RGB", (4, 2), (128, 64, 32))
imagenet = [
    (128.0 / 255.0 - 0.485) / 0.229,
    (64.0 / 255.0 - 0.456) / 0.224,
    (32.0 / 255.0 - 0.406) / 0.225,
]

u2 = bg_removal._preprocess(np, img, (320, 320))
assert u2.shape == (1, 3, 320, 320)
assert u2.dtype == np.float32
close(u2[0, 0, 0, 0], (1.0 - 0.485) / 0.229)
close(u2[0, 1, 0, 0], (0.5 - 0.456) / 0.224)
close(u2[0, 2, 0, 0], (0.25 - 0.406) / 0.225)

isnet = bg_removal._preprocess(np, img, (1024, 1024))
close(isnet[0, 0, 0, 0], 128.0 / 255.0 - 0.5)
close(isnet[0, 1, 0, 0], 64.0 / 255.0 - 0.5)
close(isnet[0, 2, 0, 0], 32.0 / 255.0 - 0.5)

depth_in = depth._preprocess(np, img, (2, 4))
assert depth_in.shape == (1, 3, 2, 4)
assert depth_in.dtype == np.float32
for channel, want in enumerate(imagenet):
    close(depth_in[0, channel, 0, 0], want, 1e-6)

depth_img = depth._normalize_depth(np, np.array([[0.0, 1.0], [2.0, 3.0]], dtype=np.float32))
assert depth_img.dtype == np.uint8
assert depth_img.tolist() == [[0, 85], [170, 255]]

seg_in = segment._image_tensor(np, img, (2, 4))
assert seg_in.shape == (1, 3, 2, 4)
assert seg_in.dtype == np.float32
close(seg_in[0, 0, 0, 0], 128.0 / 255.0)
close(seg_in[0, 1, 0, 0], 64.0 / 255.0)
close(seg_in[0, 2, 0, 0], 32.0 / 255.0)

mask = segment._to_mask(np, np.array([[-8.0, 0.0], [8.0, 0.0]], dtype=np.float32))
assert mask.dtype == np.uint8
assert mask[0, 0] < 1
assert 126 <= mask[0, 1] <= 128
assert mask[1, 0] >= 254

boxes = detect._detections(
    np,
    [np.array([[50.0, 50.0, 20.0, 20.0, 0.9, 0.1, 0.8]], dtype=np.float32)],
    100,
    100,
    0.25,
)
assert len(boxes) == 1
assert boxes[0]["class_id"] == 1
close(boxes[0]["score"], 0.72, 1e-6)
for got, want in zip(boxes[0]["box"], [0.4, 0.4, 0.6, 0.6]):
    close(got, want, 1e-6)

# YOLOX takes BGR pixels in 0-255, letterboxed with grey padding: the 4x2 test
# image fills a 4x4 model input at scale 1 and leaves the bottom half grey.
det_in, det_scale = detect.letterbox(np, img, (4, 4))
assert det_in.shape == (1, 3, 4, 4)
assert det_in.dtype == np.float32
close(det_scale, 1.0)
close(det_in[0, 0, 0, 0], 32.0)
close(det_in[0, 1, 0, 0], 64.0)
close(det_in[0, 2, 0, 0], 128.0)
close(det_in[0, 0, 3, 0], detect.PAD_VALUE)

rgb = colorize._to_rgb_prediction(np, np.ones((3, 2, 2), dtype=np.float32))
assert rgb.shape == (2, 2, 3)
assert rgb.dtype == np.uint8
assert int(rgb[0, 0, 0]) == 255

restored = deblur._to_rgb(np, np.array([[0.0, 0.5], [1.0, 0.25]], dtype=np.float32))
assert restored.shape == (2, 2, 3)
assert restored.dtype == np.uint8
assert restored[1, 0].tolist() == [255, 255, 255]

scores = tagging._sigmoid(np, np.array([-2.0, 0.0, 2.0], dtype=np.float32))
assert scores[0] < 0.12
assert 0.49 < scores[1] < 0.51
assert scores[2] > 0.88

for module in (embedding, tagging):
    inp = module._preprocess(np, img, (2, 4))
    assert inp.shape == (1, 3, 2, 4)
    assert inp.dtype == np.float32
    for channel, want in enumerate(imagenet):
        close(inp[0, channel, 0, 0], want, 1e-6)

nsfw_in = nsfw._preprocess(np, img, (2, 4), resample=Image.Resampling.BILINEAR)
assert nsfw_in.shape == (1, 3, 2, 4)
assert nsfw_in.dtype == np.float32
for channel, value in enumerate([128.0 / 255.0, 64.0 / 255.0, 32.0 / 255.0]):
    close(nsfw_in[0, channel, 0, 0], (value - 0.5) / 0.5, 1e-6)

payload = nsfw._payload(np, np.array([1.0, 3.0], dtype=np.float32))
assert payload["categories"][0]["label"] == "sfw"
assert payload["categories"][1]["label"] == "nsfw"
assert payload["score"] > 0.88
`
	path := filepath.Join(t.TempDir(), "sidecar_contract.py")
	if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cmd := exec.Command("python3", path)
	cmd.Env = pythonSidecarEnv(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No module named 'numpy'") || strings.Contains(string(out), "No module named 'PIL'") {
			t.Skipf("python sidecar golden contract requires numpy and Pillow: %s", out)
		}
		t.Fatalf("python sidecar golden contract failed: %v\n%s", err, out)
	}
}

func pythonSidecarEnv(t *testing.T) []string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get sidecar test working directory: %v", err)
	}
	for current := filepath.Clean(cwd); ; current = filepath.Dir(current) {
		candidate := filepath.Join(current, "py", "image_tools_sidecar", "__init__.py")
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			path := filepath.Dir(filepath.Dir(candidate))
			env := make([]string, 0, len(os.Environ())+1)
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "PYTHONPATH=") {
					env = append(env, entry)
				}
			}
			return append(env, "PYTHONPATH="+path)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	t.Fatal("locate Python sidecar source directory")
	return nil
}
