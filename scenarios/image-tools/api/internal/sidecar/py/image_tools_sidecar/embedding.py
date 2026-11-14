"""embedding — produce an image embedding vector with an ONNX vision encoder.

The sidecar writes JSON shaped for search/indexing:

  {"embedding": [0.1, ...], "dimensions": 768, "norm": 12.3}

It uses the same lightweight runtime as the other ONNX sidecars and intentionally
keeps postprocessing generic: flatten the first output tensor and report its
L2-normalized metadata without making model-specific label assumptions.
"""

from __future__ import annotations

import json

from . import _common


def _preprocess(np, image, size_hw):
    h, w = size_hw
    resized = image.resize((w, h))
    arr = np.array(resized).astype(np.float32) / 255.0
    if arr.ndim == 2:
        arr = np.stack([arr] * 3, axis=-1)
    arr = arr[:, :, :3]
    mean = np.array([0.485, 0.456, 0.406], dtype=np.float32)
    std = np.array([0.229, 0.224, 0.225], dtype=np.float32)
    arr = (arr - mean) / std
    return arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)


def main() -> None:
    args = _common.parse_io_args("ONNX image embedding")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    size_hw = _common.model_input_hw(session)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    inp = _preprocess(np, image, size_hw)
    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    vec = np.asarray(outputs[0], dtype=np.float32).reshape(-1)
    norm = float(np.linalg.norm(vec))
    payload = {
        "embedding": [float(x) for x in vec.tolist()],
        "dimensions": int(vec.size),
        "norm": norm,
    }
    data = json.dumps(payload, indent=2).encode("utf-8") + b"\n"
    try:
        with open(args.out, "wb") as f:
            f.write(data)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
