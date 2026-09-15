"""Shared helpers for the image_tools_sidecar ops: arg parsing, image IO, and a
small ONNX session wrapper. Kept dependency-light (onnxruntime, Pillow, numpy).
"""

from __future__ import annotations

import argparse
import sys
from typing import Tuple

_SESSION_CACHE = {}


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
        return _SESSION_CACHE[onnx_path]
    _np, ort, _Image = require_deps()
    try:
        session = ort.InferenceSession(onnx_path, providers=["CPUExecutionProvider"])
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
