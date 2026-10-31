"""segment — run an ONNX segmentation model and write an alpha-mask PNG.

The sidecar supports two CPU-friendly shapes:

* generic semantic/matting exports that accept one image tensor and emit a mask
  or logits tensor;
* SAM 1/MobileSAM directories with separate encoder/decoder ONNX files and
  explicit normalized positive/negative point or box prompts. SAM 2 exports
  have a different contract and are not interchangeable with these pairs.

It writes a single-channel PNG mask. Product-level smart-selection behavior
still lives in the Go selection domain; this module is the model-backed backend
vertical for the AI operation.
"""

from __future__ import annotations

import os
import argparse
import json
import math

from . import _common


def _image_tensor(np, image, size_hw):
    h, w = size_hw
    resized = image.resize((w, h))
    arr = np.array(resized).astype(np.float32) / 255.0
    if arr.ndim == 2:
        arr = np.stack([arr] * 3, axis=-1)
    arr = arr[:, :, :3]
    return arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)


def _to_mask(np, pred):
    arr = np.asarray(pred, dtype=np.float32)
    arr = np.squeeze(arr)
    if arr.ndim == 3:
        # SAM decoders commonly emit multiple candidate masks. Pick the first;
        # E2E/golden tests can tighten this later when prompt scoring is owned.
        arr = arr[0]
    if arr.ndim != 2:
        _common.fail(f"unsupported segmentation output shape {arr.shape}", code=8)
    if float(arr.min()) < 0.0 or float(arr.max()) > 1.0:
        arr = 1.0 / (1.0 + np.exp(-arr))
    return np.clip(arr * 255.0, 0, 255).astype(np.uint8)


def _write_mask(Image, mask, source_size, out_path):
    img = Image.fromarray(mask, mode="L").resize(source_size, Image.Resampling.NEAREST)
    try:
        img.save(out_path, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)


def _find_sam_pair(model_arg: str):
    if not os.path.isdir(model_arg):
        return None
    files = [os.path.join(model_arg, f) for f in sorted(os.listdir(model_arg)) if f.endswith(".onnx")]
    encoder = next((f for f in files if "encoder" in os.path.basename(f).lower()), "")
    decoder = next((f for f in files if "decoder" in os.path.basename(f).lower()), "")
    if encoder and decoder:
        return encoder, decoder
    return None


def _run_generic(np, Image, args, image):
    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    inp = _image_tensor(np, image, _common.model_input_hw(session))
    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    mask = _to_mask(np, outputs[0])
    _write_mask(Image, mask, image.size, args.out)


def _sam_input(np, Image, image, encoder):
    """Honor the two SAM1 export contracts; never guess a dynamic tensor size."""
    shape = encoder.get_inputs()[0].shape
    if len(shape) == 3 and shape[-1] == 3:
        # samexporter --use-preprocess embeds normalization and padding, but
        # requires a longest-side resize by the caller. Pixel values are 0..255.
        edge = 1024
        embedded_preprocess = True
    elif (len(shape) == 4 and shape[:2] == [1, 3]
          and isinstance(shape[2], int) and shape[2] == shape[3]):
        edge = shape[2]
        embedded_preprocess = False
    else:
        raise ValueError(f"unsupported SAM1 encoder input contract: {shape}")
    scale = edge / max(image.size)
    w, h = (int(value * scale + 0.5) for value in image.size)
    pixels = np.asarray(image.resize((w, h), Image.Resampling.BILINEAR), dtype=np.float32)
    if embedded_preprocess:
        return pixels, (w, h)
    pixels = (pixels - np.array([123.675, 116.28, 103.53], dtype=np.float32)) / np.array(
        [58.395, 57.12, 57.375], dtype=np.float32)
    tensor = np.zeros((1, 3, edge, edge), dtype=np.float32)
    tensor[0, :, :h, :w] = pixels.transpose(2, 0, 1)
    return tensor, (w, h)


def _prompt_arrays(np, prompt, resized_size):
    if not isinstance(prompt, dict) or set(prompt) - {"points", "box"}:
        raise ValueError("SAM prompt must contain only points and/or box")
    points = prompt.get("points", [])
    if not isinstance(points, list) or len(points) > 64:
        raise ValueError("SAM requires at most 64 point prompts")
    coords, labels = [], []
    w, h = resized_size

    def unit(value):
        if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or not 0 <= value <= 1:
            raise ValueError("SAM coordinates must be finite numbers in 0..1")
        return float(value)

    for point in points:
        if not isinstance(point, dict) or set(point) - {"x", "y", "negative"}:
            raise ValueError("SAM point must contain x, y and optional negative")
        negative = point.get("negative", False)
        if not isinstance(negative, bool):
            raise ValueError("SAM negative must be boolean")
        coords.append([unit(point["x"]) * w, unit(point["y"]) * h])
        labels.append(0 if negative else 1)
    box = prompt.get("box")
    if box is not None:
        if not isinstance(box, dict) or set(box) != {"x", "y", "width", "height"}:
            raise ValueError("SAM box must contain x, y, width and height")
        x, y, bw, bh = (unit(box[key]) for key in ("x", "y", "width", "height"))
        if bw <= 0 or bh <= 0 or x + bw > 1 or y + bh > 1:
            raise ValueError("SAM box must have positive area inside the image")
        coords.extend([[x * w, y * h], [(x + bw) * w, (y + bh) * h]])
        labels.extend([2, 3])
    elif coords:
        coords.append([0, 0])
        labels.append(-1)  # SAM's explicit no-box padding prompt
    if not coords or not any(label > 0 for label in labels):
        raise ValueError("SAM requires a positive point or an object box; auto-subject is not supported")
    return np.array([coords], dtype=np.float32), np.array([labels], dtype=np.float32)


def _decoder_feed(np, decoder, embedding, image, coords, labels):
    feed = {}
    for inp in decoder.get_inputs():
        name = inp.name
        lname = name.lower()
        shape = inp.shape
        if "embedding" in lname or "image_embed" in lname:
            feed[name] = embedding
        elif "point_coord" in lname or lname == "point_coords":
            feed[name] = coords
        elif "point_label" in lname or lname == "point_labels":
            feed[name] = labels
        elif "has_mask" in lname:
            feed[name] = np.array([0], dtype=np.float32)
        elif "mask_input" in lname or "masks" == lname:
            feed[name] = np.zeros((1, 1, 256, 256), dtype=np.float32)
        elif "orig_im_size" in lname or "orig_size" in lname:
            value = np.array([image.height, image.width], dtype=np.float32)
            feed[name] = value.reshape(1, 2) if len(shape) == 2 else value
        else:
            _common.fail(f"unsupported SAM decoder input {name!r}", code=9)
    return feed


def _run_sam(np, Image, args, image, encoder_path, decoder_path):
    encoder = _common.make_session(encoder_path)
    decoder = _common.make_session(decoder_path)
    enc_input = encoder.get_inputs()[0].name
    tensor, size = _sam_input(np, Image, image, encoder)
    coords, labels = _prompt_arrays(np, json.loads(args.prompts), size)
    embedding = encoder.run(None, {enc_input: tensor})[0]
    outputs = decoder.run(None, _decoder_feed(np, decoder, embedding, image, coords, labels))
    named = dict(zip((o.name for o in decoder.get_outputs()), outputs))
    if "masks" not in named or "iou_predictions" not in named:
        raise ValueError("SAM1 decoder must return source-size masks and iou_predictions")
    masks = np.asarray(named["masks"])
    scores = np.asarray(named["iou_predictions"]).reshape(-1)
    if (masks.ndim != 4 or masks.shape[0] != 1 or masks.shape[1] != scores.size
            or masks.shape[2:] != (image.height, image.width)
            or not np.isfinite(masks).all() or not np.isfinite(scores).all()):
        raise ValueError("invalid SAM mask/score dimensions or non-finite predictions")
    # These decoder masks are logits. Do not infer a sigmoid contract from the
    # observed min/max or choose the first of several ambiguous candidates.
    mask = (masks[0, int(np.argmax(scores))] > 0).astype(np.uint8) * 255
    _write_mask(Image, mask, image.size, args.out)


def main() -> None:
    parser = argparse.ArgumentParser(description="ONNX image segmentation")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--prompts", default="{}", help="JSON normalized points and/or box for SAM1/MobileSAM")
    args = parser.parse_args()
    np, _ort, Image = _common.require_deps()

    try:
        from PIL import ImageOps
        with Image.open(args.image) as source:
            if source.width * source.height > 16_000_000:
                raise ValueError("segmentation input exceeds 16 million pixels")
            image = ImageOps.exif_transpose(source).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    sam_pair = _find_sam_pair(args.model)
    if sam_pair is not None:
        _run_sam(np, Image, args, image, sam_pair[0], sam_pair[1])
        return
    if args.prompts != "{}":
        raise ValueError("prompted segmentation requires a compatible SAM1/MobileSAM encoder-decoder pair")
    _run_generic(np, Image, args, image)


if __name__ == "__main__":
    main()
