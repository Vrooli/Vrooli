"""deblur — restore a blurry image with an image-to-image ONNX model.

The sidecar handles NAFNet-style exports that accept an RGB tensor and emit an
RGB tensor. It preserves the original image dimensions by resizing only for the
model call, then resizing the model output back to the source size.
"""

from __future__ import annotations

from . import _common


def _to_rgb(np, pred):
    out = np.squeeze(pred)
    if out.ndim == 4:
        out = np.squeeze(out, axis=0)
    if out.ndim == 3 and out.shape[0] in (1, 3):
        out = out.transpose(1, 2, 0)
    if out.ndim == 2:
        out = np.stack([out] * 3, axis=-1)
    if out.shape[-1] == 1:
        out = np.repeat(out, 3, axis=-1)
    if out.dtype != np.uint8:
        if float(out.max()) <= 1.0 + 1e-6 and float(out.min()) >= -1e-6:
            out = out * 255.0
        out = np.clip(out, 0, 255).astype(np.uint8)
    return out[:, :, :3]


def main() -> None:
    args = _common.parse_io_args("ONNX image deblur")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    h, w = _common.model_input_hw(session)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    resized = image.resize((w, h))
    arr = np.array(resized).astype(np.float32) / 255.0
    inp = arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)

    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    rgb = _to_rgb(np, outputs[0])
    out_img = Image.fromarray(rgb, mode="RGB").resize(image.size)
    try:
        out_img.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
