"""segment — run an ONNX segmentation model and write an alpha-mask PNG.

The sidecar supports two CPU-friendly shapes:

* generic semantic/matting exports that accept one image tensor and emit a mask
  or logits tensor;
* SAM-style directories with separate encoder/decoder ONNX files, using a
  center-point prompt as the automatic mask seed.

It writes a single-channel PNG mask. Product-level smart-selection behavior
still lives in the Go selection domain; this module is the model-backed backend
vertical for the AI operation.
"""

from __future__ import annotations

import os

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
    img = Image.fromarray(mask, mode="L").resize(source_size)
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


def _decoder_feed(np, decoder, embedding, image):
    feed = {}
    for inp in decoder.get_inputs():
        name = inp.name
        lname = name.lower()
        shape = inp.shape
        if "embedding" in lname or "image_embed" in lname:
            feed[name] = embedding
        elif "point_coord" in lname or lname == "point_coords":
            feed[name] = np.array([[[image.width / 2.0, image.height / 2.0]]], dtype=np.float32)
        elif "point_label" in lname or lname == "point_labels":
            feed[name] = np.array([[1]], dtype=np.float32)
        elif "mask_input" in lname or "masks" == lname:
            feed[name] = np.zeros((1, 1, 256, 256), dtype=np.float32)
        elif "has_mask" in lname:
            feed[name] = np.array([0], dtype=np.float32)
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
    embedding = encoder.run(None, {enc_input: _image_tensor(np, image, _common.model_input_hw(encoder))})[0]
    outputs = decoder.run(None, _decoder_feed(np, decoder, embedding, image))
    mask = _to_mask(np, outputs[0])
    _write_mask(Image, mask, image.size, args.out)


def main() -> None:
    args = _common.parse_io_args("ONNX image segmentation")
    np, _ort, Image = _common.require_deps()

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    sam_pair = _find_sam_pair(args.model)
    if sam_pair is not None:
        _run_sam(np, Image, args, image, sam_pair[0], sam_pair[1])
        return
    _run_generic(np, Image, args, image)


if __name__ == "__main__":
    main()
