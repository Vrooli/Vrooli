"""tagging — run an ONNX image tagger and write top tags as JSON.

Most tagger exports produce one score vector. When a model-specific labels file
is not installed, this sidecar emits stable anonymous tag ids (`tag_123`) rather
than pretending to know a vocabulary.
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


def _sigmoid(np, x):
    return 1.0 / (1.0 + np.exp(-x))


def main() -> None:
    args = _common.parse_io_args("ONNX image tagging")
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
    scores = np.asarray(outputs[0], dtype=np.float32).reshape(-1)
    if scores.size == 0:
        _common.fail("tagger produced an empty score vector", code=8)
    if float(scores.min()) < 0.0 or float(scores.max()) > 1.0:
        scores = _sigmoid(np, scores)
    order = np.argsort(scores)[::-1][:50]
    tags = [{"tag": f"tag_{int(i)}", "score": float(scores[i])} for i in order if float(scores[i]) >= 0.15]
    payload = {"tags": tags, "count": len(tags)}
    try:
        with open(args.out, "wb") as f:
            f.write(json.dumps(payload, indent=2).encode("utf-8") + b"\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
