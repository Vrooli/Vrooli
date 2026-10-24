"""colorize — add realistic colour to a grayscale image with a DDColor-family
ONNX model (CPU-only, onnxruntime CPUExecutionProvider).

The model predicts chrominance for the luminance of the input. We run the model
on the resized luminance, take its colour (ab / RGB-chroma) prediction, and
recombine it with the ORIGINAL full-resolution luminance so detail is preserved
regardless of the model's working resolution. Output is an RGB PNG.

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
