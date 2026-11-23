"""colorize — add realistic colour to a grayscale image with a DDColor-family
ONNX model (CPU-only, onnxruntime CPUExecutionProvider).

The model predicts chrominance for the luminance of the input. Two export
contracts are supported:

  * ab export (the official DDColor pipeline, e.g. the fp16 DDColor-tiny ONNX):
    a (1,3,H,W) grey-RGB input built from the image's CIE Lab L channel, and a
    (1,2,H,W) output of Lab ``a``/``b`` chroma. ab is resized to the source size,
    joined with the ORIGINAL full-resolution L and converted Lab -> sRGB.
  * direct-RGB export: a (1,3,H,W) luminance input and an RGB(-like) output.

Either way the original full-resolution luminance carries the detail, so the
model's working resolution only limits colour resolution. Output is an RGB PNG.

Like every sidecar op this fails loud (non-zero exit, actionable stderr) when its
runtime deps or weights are absent — it never silently returns the input.
"""

from __future__ import annotations

from . import _common


def _to_rgb_prediction(np, pred):
    """Normalize a model output to an HxWx3 uint8 RGB array.

    DDColor exports vary (some emit RGB directly, some emit ab to merge with L).
    We handle the common direct-RGB export: squeeze the batch, move channels last
    if needed, scale to 0..255.
    """
    out = np.squeeze(pred)
    if out.ndim == 3 and out.shape[0] == 3:  # CHW → HWC
        out = out.transpose(1, 2, 0)
    if out.dtype != np.uint8:
        mi, ma = float(out.min()), float(out.max())
        if ma <= 1.0 + 1e-6 and mi >= -1e-6:
            out = out * 255.0
        out = np.clip(out, 0, 255).astype(np.uint8)
    return out


_D65 = (0.95047, 1.0, 1.08883)
_SRGB_TO_XYZ = ((0.4124564, 0.3575761, 0.1804375), (0.2126729, 0.7151522, 0.0721750), (0.0193339, 0.1191920, 0.9503041))
_XYZ_TO_SRGB = ((3.2404542, -1.5371385, -0.4985314), (-0.9692660, 1.8760108, 0.0415560), (0.0556434, -0.2040259, 1.0572252))


def _srgb_to_lab(np, rgb):
    """sRGB floats in [0,1], HxWx3 -> CIE Lab (D65): L in 0..100, a/b about -128..127."""
    linear = np.where(rgb <= 0.04045, rgb / 12.92, ((rgb + 0.055) / 1.055) ** 2.4)
    xyz = linear @ np.array(_SRGB_TO_XYZ, dtype=np.float32).T
    scaled = xyz / np.array(_D65, dtype=np.float32)
    f = np.where(scaled > 216 / 24389, np.cbrt(scaled), (24389 / 27 * scaled + 16) / 116)
    return np.stack([116 * f[..., 1] - 16, 500 * (f[..., 0] - f[..., 1]), 200 * (f[..., 1] - f[..., 2])], axis=-1)


def _lab_to_srgb(np, lab):
    """CIE Lab (D65) -> sRGB floats clipped to [0,1]."""
    fy = (lab[..., 0] + 16) / 116
    fx = fy + lab[..., 1] / 500
    fz = fy - lab[..., 2] / 200
    f = np.stack([fx, fy, fz], axis=-1)
    scaled = np.where(f ** 3 > 216 / 24389, f ** 3, (116 * f - 16) * 27 / 24389)
    xyz = scaled * np.array(_D65, dtype=np.float32)
    linear = np.clip(xyz @ np.array(_XYZ_TO_SRGB, dtype=np.float32).T, 0, 1)
    return np.where(linear <= 0.0031308, linear * 12.92, 1.055 * linear ** (1 / 2.4) - 0.055)


def _colorize_ab(np, Image, session, image, h, w):
    """Official DDColor pipeline for an ab-output ONNX export."""
    source = np.asarray(image, dtype=np.float32) / 255.0
    lab = _srgb_to_lab(np, source)
    grey = _lab_to_srgb(np, np.concatenate([lab[..., :1], np.zeros_like(lab[..., 1:])], axis=-1))
    grey_small = np.asarray(Image.fromarray((grey * 255.0 + 0.5).astype(np.uint8)).resize((w, h), Image.BICUBIC), dtype=np.float32) / 255.0
    inp = grey_small.transpose(2, 0, 1)[np.newaxis].astype(np.float32)
    ab = session.run(None, {session.get_inputs()[0].name: inp})[0]
    ab = np.squeeze(ab).transpose(1, 2, 0)  # HxWx2 in Lab units
    width, height = image.size
    resized = np.stack(
        [np.asarray(Image.fromarray(ab[..., c].astype(np.float32), mode="F").resize((width, height), Image.BILINEAR)) for c in range(2)],
        axis=-1,
    )
    out = _lab_to_srgb(np, np.concatenate([lab[..., :1], resized], axis=-1))
    return Image.fromarray((out * 255.0 + 0.5).astype(np.uint8), mode="RGB")


def main() -> None:
    args = _common.parse_io_args("ONNX colorization (grayscale → colour)")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    h, w = _common.model_input_hw(session)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    out_channels = session.get_outputs()[0].shape[1] if len(session.get_outputs()[0].shape) == 4 else None
    if out_channels == 2:
        colour = _colorize_ab(np, Image, session, image, h, w)
        try:
            colour.save(args.out, format="PNG")
        except Exception as exc:  # noqa: BLE001
            _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)
        return

    # Feed the model the luminance replicated to 3 channels at its input size.
    luminance = image.convert("L").resize((w, h))
    arr = np.array(luminance).astype(np.float32) / 255.0
    inp = np.stack([arr] * 3, axis=0)[np.newaxis, :, :, :].astype(np.float32)

    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    rgb = _to_rgb_prediction(np, outputs[0])

    colour = Image.fromarray(rgb, mode="RGB").resize(image.size)
    try:
        colour.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
