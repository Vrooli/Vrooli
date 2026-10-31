"""Shared helpers for the image_tools_sidecar ops: arg parsing, image IO, and a
small ONNX session wrapper. Kept dependency-light (onnxruntime, Pillow, numpy).
"""

from __future__ import annotations

import argparse
import sys
import os
from collections import OrderedDict
from typing import Tuple

# A SAM encoder/decoder pair is the largest cache unit this CPU worker needs.
# Never accumulate every model a long-lived API process has ever used.
_SESSION_CACHE = OrderedDict()
_MAX_SESSIONS = 2
_ONNX_MAX_INTRA_OP_THREADS = 8
_ONNX_INTER_OP_THREADS = 1


def onnx_thread_settings():
    """Return the bounded CPU thread policy used by local ONNX sessions.

    This limits runtime worker fan-out. It does not reserve CPUs, set affinity,
    or impose a CPU quota; those are separate owner-level controls.
    """
    return {
        "intra_op_num_threads": max(1, min(_ONNX_MAX_INTRA_OP_THREADS, os.cpu_count() or 1)),
        "inter_op_num_threads": _ONNX_INTER_OP_THREADS,
        "execution_mode": "sequential",
    }


def parse_device_args(parser):
    """Receive the scheduler's decision before any framework imports/loads."""
    parser.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    args = parser.parse_args()
    os.environ["IMAGE_TOOLS_DEVICE"] = args.device
    return args


def use_cuda(torch):
    """Hardware availability never overrides an explicit CPU fallback."""
    device = os.environ.get("IMAGE_TOOLS_DEVICE", "cpu")
    if device == "cpu":
        return False
    if device != "cuda" or not torch.cuda.is_available():
        raise RuntimeError(f"requested device {device!r} is unavailable")
    return True


def parse_io_args(description: str) -> argparse.Namespace:
    """Standard --model/--image/--out argument set shared by every op."""
    p = argparse.ArgumentParser(description=description)
    p.add_argument("--model", required=True, help="path to the ONNX weight file (or model dir)")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", required=True, help="output image path")
    return p.parse_args()


def fail(msg: str, code: int = 2) -> "None":
    """Print an actionable error to stderr and exit non-zero (the Go runner
    surfaces stderr in the job error), never silently succeed."""
    sys.stderr.write(f"image_tools_sidecar: {msg}\n")
    sys.exit(code)


def require_deps() -> Tuple["object", "object", "object"]:
    """Import the runtime deps with a clear message if the host Python lacks
    them (the documented provisioning step), rather than a raw ImportError."""
    try:
        import numpy as np  # noqa: WPS433
        import onnxruntime as ort  # noqa: WPS433
        from PIL import Image  # noqa: WPS433
    except ImportError as exc:  # pragma: no cover - exercised on un-provisioned hosts
        fail(
            f"missing Python dependency ({exc}); the Python venv is not provisioned. "
            "Ensure `vrooli host install uv`, then restart image-tools to build/repair "
            "the lock-pinned venv (see docs/reference/backends.md).",
            code=3,
        )
        raise
    return np, ort, Image


def resolve_onnx_path(model_arg: str) -> str:
    """Resolve the ONNX file: the arg itself if it is a file, else the single
    .onnx inside a model directory."""
    import os

    if os.path.isfile(model_arg):
        return model_arg
    if os.path.isdir(model_arg):
        onnx = [f for f in sorted(os.listdir(model_arg)) if f.endswith(".onnx")]
        if len(onnx) == 1:
            return os.path.join(model_arg, onnx[0])
        if onnx:
            return os.path.join(model_arg, onnx[0])
    fail(f"no ONNX weight found at {model_arg!r}", code=4)
    raise SystemExit(4)  # unreachable; keeps type-checkers happy


def make_session(onnx_path: str):
    """Create or reuse a CPU onnxruntime InferenceSession."""
    if onnx_path in _SESSION_CACHE:
        _SESSION_CACHE.move_to_end(onnx_path)
        return _SESSION_CACHE[onnx_path]
    _np, ort, _Image = require_deps()
    try:
        # Evict before allocating the next model, rather than exceeding the
        # resident budget during construction.
        while len(_SESSION_CACHE) >= _MAX_SESSIONS:
            _SESSION_CACHE.popitem(last=False)
        settings = onnx_thread_settings()
        options = ort.SessionOptions()
        options.intra_op_num_threads = settings["intra_op_num_threads"]
        options.inter_op_num_threads = settings["inter_op_num_threads"]
        options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
        session = ort.InferenceSession(onnx_path, sess_options=options, providers=["CPUExecutionProvider"])
        _SESSION_CACHE[onnx_path] = session
        return session
    except Exception as exc:  # noqa: BLE001 - surface any load error actionably
        fail(f"failed to load ONNX model {onnx_path!r}: {exc}", code=5)
        raise


def model_input_hw(session) -> Tuple[int, int]:
    """Return the model's (height, width) spatial input, defaulting to 320 when
    the dimensions are dynamic/unknown."""
    try:
        shape = session.get_inputs()[0].shape  # typically [N, C, H, W]
        h = shape[2] if isinstance(shape[2], int) and shape[2] > 0 else 320
        w = shape[3] if isinstance(shape[3], int) and shape[3] > 0 else 320
        return int(h), int(w)
    except Exception:  # noqa: BLE001
        return 320, 320


def load_oriented_image(path: str, max_pixels: int, mode: str = "RGB"):
    """Decode only a bounded raster and retain its exact EXIF coordinate map.

    Coordinates are continuous raster coordinates: a pixel center is (i+.5,
    j+.5). The row-major 3x3 matrix maps original centers into the oriented
    raster, before any model-specific resize. This is separate from learned
    camera/geometry and never implies scene measurements.
    """
    from PIL import Image, ImageOps

    with Image.open(path) as source:
        width, height = source.size
        if width <= 0 or height <= 0 or width * height > max_pixels:
            raise ValueError(f"input image exceeds {max_pixels} pixels")
        orientation = source.getexif().get(274, 1)
        if not isinstance(orientation, int) or orientation not in range(1, 9):
            raise ValueError("EXIF orientation must be an integer in 1..8")
        transforms = {
            1: [[1, 0, 0], [0, 1, 0], [0, 0, 1]],
            2: [[-1, 0, width], [0, 1, 0], [0, 0, 1]],
            3: [[-1, 0, width], [0, -1, height], [0, 0, 1]],
            4: [[1, 0, 0], [0, -1, height], [0, 0, 1]],
            5: [[0, 1, 0], [1, 0, 0], [0, 0, 1]],
            6: [[0, -1, height], [1, 0, 0], [0, 0, 1]],
            7: [[0, -1, height], [-1, 0, width], [0, 0, 1]],
            8: [[0, 1, 0], [-1, 0, width], [0, 0, 1]],
        }
        oriented = ImageOps.exif_transpose(source).convert(mode)
        oriented.load()
    return oriented, {
        "width": oriented.width,
        "height": oriented.height,
        "original_width": width,
        "original_height": height,
        "original_exif_orientation": orientation,
        "exif_orientation_applied": True,
        "original_to_oriented_matrix_row_major": transforms[orientation],
        "pixel_coordinates": "continuous raster coordinates; pixel centers at (i+0.5,j+0.5)",
    }
