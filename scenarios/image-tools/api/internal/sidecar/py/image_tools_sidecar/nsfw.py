"""nsfw_classify — classify an image with a ViT-style ONNX safety model.

The sidecar emits a small JSON payload:

  {"score": 0.91, "categories": [{"label": "sfw", ...}, {"label": "nsfw", ...}]}

When --out is supplied the payload is written there; otherwise it is printed to
stdout for the synchronous analysis service. The preprocessing profile matches
common HuggingFace ViT image-classification exports: RGB, ImageNet
normalization, NCHW float32.
"""

from __future__ import annotations

import argparse
import json
import sys

from . import _common


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="ONNX NSFW classification")
    p.add_argument("--model", required=True, help="path to the ONNX weight file (or model dir)")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", help="optional JSON output path; stdout when omitted")
    return p.parse_args()


def _softmax(np, logits):
    arr = np.asarray(logits, dtype=np.float32).reshape(-1)
    if arr.size == 0:
        return arr
    arr = arr - float(arr.max())
    exp = np.exp(arr)
    denom = float(exp.sum())
    if denom <= 0:
        return np.zeros_like(exp)
    return exp / denom


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


def _payload(np, raw_output):
    probs = _softmax(np, raw_output)
    labels = ["sfw", "nsfw"]
    categories = []
    for idx, score in enumerate(probs.tolist()):
        label = labels[idx] if idx < len(labels) else f"class_{idx}"
        categories.append({"label": label, "score": float(score)})
    nsfw_score = 0.0
    for c in categories:
        if c["label"].lower() == "nsfw":
            nsfw_score = float(c["score"])
            break
    return {"score": nsfw_score, "categories": categories}


def main() -> None:
    args = _parse_args()
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
    data = json.dumps(_payload(np, outputs[0]), indent=2).encode("utf-8") + b"\n"

    if args.out:
        try:
            with open(args.out, "wb") as f:
                f.write(data)
        except Exception as exc:  # noqa: BLE001
            _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)
        return
    sys.stdout.buffer.write(data)


if __name__ == "__main__":
    main()
