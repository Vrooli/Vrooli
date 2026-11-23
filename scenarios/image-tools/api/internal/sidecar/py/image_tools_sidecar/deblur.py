"""deblur — restore a blurry image with an image-to-image ONNX model.

NAFNet-style exports take an RGB tensor and emit an RGB tensor. A dynamic-shape
export (the registry's NAFNet: height and width are free) runs at the image's own
resolution: reflect-padded to the network's stride, in overlapping tiles when the
image is large, and cropped back to the source size, so no detail is lost to a
resize. A fixed-shape export keeps the old contract: the image is resized to the
model's input for the call and the output is resized back.
"""

from __future__ import annotations

from . import _common, _tiling

TILE = 1024  # interior edge of one tile, in pixels
MARGIN = 64  # context pixels kept on every side of a tile and then discarded
MULTIPLE = 32  # NAFNet downsamples four times; pad to a safe multiple
MIN_SIDE = 384  # the deepest stage needs a real spatial extent: 320 px fails in ONNX Runtime


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


def _is_dynamic(session) -> bool:
    """True when the model's height and width are free (symbolic or -1)."""
    shape = session.get_inputs()[0].shape
    return len(shape) == 4 and not all(isinstance(d, int) and d > 0 for d in shape[2:])


def deblur_array(np, session, rgb, tile=TILE, margin=MARGIN):
    """Deblur an HxWx3 float32 array in [0, 1] at its own resolution."""
    out = _tiling.tiled_apply(np, session, rgb, tile=tile, margin=margin, multiple=MULTIPLE, minimum=MIN_SIDE)
    return np.clip(out, 0.0, 1.0)


def main() -> None:
    args = _common.parse_io_args("ONNX image deblur")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    if _is_dynamic(session):
        source = np.asarray(image, dtype=np.float32) / 255.0
        result = deblur_array(np, session, source)
        out_img = Image.fromarray((result * 255.0 + 0.5).astype(np.uint8), mode="RGB")
    else:
        h, w = _common.model_input_hw(session)
        resized = image.resize((w, h))
        arr = np.array(resized).astype(np.float32) / 255.0
        inp = arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)
        outputs = session.run(None, {session.get_inputs()[0].name: inp})
        out_img = Image.fromarray(_to_rgb(np, outputs[0]), mode="RGB").resize(image.size)
    try:
        out_img.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
