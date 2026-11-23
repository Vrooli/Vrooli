"""embedding — produce an image embedding vector with an ONNX vision encoder.

The sidecar writes JSON shaped for search/indexing:

  {"embedding": [0.1, ...], "dimensions": 768, "norm": 12.3}

It uses the same lightweight runtime as the other ONNX sidecars. Preprocessing is
data-driven: when the model directory carries a Hugging Face ``preprocessor_config.json``
its ``image_mean``, ``image_std``, ``size`` and ``resample`` are used (SigLIP2 uses mean
and std 0.5, ImageNet-trained encoders 0.485/0.456/0.406); without one the ImageNet
statistics and the model's fixed input size apply. The vector is the output named
``image_embeds`` or ``pooler_output`` when the graph has one (a vision tower also emits
per-patch ``last_hidden_state``, which is not the image embedding), else the first; a
token grid is reduced to its class token.
"""

from __future__ import annotations

import json
import os

from . import _common

IMAGENET = {"image_mean": [0.485, 0.456, 0.406], "image_std": [0.229, 0.224, 0.225]}
EMBEDDING_OUTPUTS = ("image_embeds", "pooler_output")
PIL_RESAMPLE = {0: "NEAREST", 1: "LANCZOS", 2: "BILINEAR", 3: "BICUBIC", 4: "BOX", 5: "HAMMING"}


def _load_preprocessing(onnx_path: str, size_hw):
    """(mean, std, (h, w), resample name) from preprocessor_config.json beside the weight."""
    config = {}
    path = os.path.join(os.path.dirname(onnx_path), "preprocessor_config.json")
    if os.path.isfile(path):
        with open(path, "r", encoding="utf-8") as handle:
            config = json.load(handle)
    size = config.get("size") or {}
    if isinstance(size, dict) and size.get("height") and size.get("width"):
        size_hw = (int(size["height"]), int(size["width"]))
    resample = PIL_RESAMPLE.get(config.get("resample"), "BICUBIC")
    return (
        config.get("image_mean") or IMAGENET["image_mean"],
        config.get("image_std") or IMAGENET["image_std"],
        size_hw,
        resample,
    )


def _preprocess(np, image, size_hw, mean=IMAGENET["image_mean"], std=IMAGENET["image_std"], resample="BICUBIC"):
    from PIL import Image

    h, w = size_hw
    resized = image.resize((w, h), getattr(Image, resample))
    arr = np.array(resized).astype(np.float32) / 255.0
    if arr.ndim == 2:
        arr = np.stack([arr] * 3, axis=-1)
    arr = arr[:, :, :3]
    arr = (arr - np.array(mean, dtype=np.float32)) / np.array(std, dtype=np.float32)
    return arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)


def _embedding_output(session, outputs):
    """The image-embedding vector among the graph outputs: a pooled output by name,
    else the first output. A (batch, tokens, dim) token grid (a ViT's last hidden state)
    is reduced to its first, class, token: flattening it would make the "embedding"
    197 x 768 numbers per image."""
    chosen = outputs[0]
    for meta, value in zip(session.get_outputs(), outputs):
        if meta.name in EMBEDDING_OUTPUTS:
            chosen = value
            break
    if getattr(chosen, "ndim", 0) == 3:
        return chosen[:, 0, :]
    return chosen


def main() -> None:
    args = _common.parse_io_args("ONNX image embedding")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    mean, std, size_hw, resample = _load_preprocessing(onnx_path, _common.model_input_hw(session, default_hw=(224, 224)))

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    inp = _preprocess(np, image, size_hw, mean, std, resample)
    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    vec = np.asarray(_embedding_output(session, outputs), dtype=np.float32).reshape(-1)
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
