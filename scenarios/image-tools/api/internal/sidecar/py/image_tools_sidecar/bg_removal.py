"""background_removal — produce an alpha matte with a U^2-Net / IS-Net family
ONNX model and composite the subject onto transparency.

CPU-only (onnxruntime CPUExecutionProvider). The preprocessing profile is chosen
from the model's input size, matching the two dominant rembg session families:

  * U^2-Net (u2net, u2netp, silueta): 320x320 input, ImageNet mean/std.
  * IS-Net / DIS / BiRefNet:          >=1024 input, mean 0.5 / std 1.0.

Output is an RGBA PNG: original RGB with the matte as the alpha channel.
"""

from __future__ import annotations

from . import _common


def _preprocess(np, image, size_hw):
    h, w = size_hw
    resized = image.resize((w, h))
    arr = np.array(resized).astype(np.float32)
    if arr.ndim == 2:  # grayscale → RGB
        arr = np.stack([arr] * 3, axis=-1)
    arr = arr[:, :, :3]

    if max(h, w) >= 1024:
        # IS-Net / DIS profile: scale to [0,1], mean 0.5 / std 1.0.
        arr = arr / 255.0
        arr = (arr - 0.5) / 1.0
    else:
        # U^2-Net profile: divide by per-image max, then ImageNet mean/std.
        mx = arr.max()
        if mx > 0:
            arr = arr / mx
        mean = np.array([0.485, 0.456, 0.406], dtype=np.float32)
        std = np.array([0.229, 0.224, 0.225], dtype=np.float32)
        arr = (arr - mean) / std

    arr = arr.transpose(2, 0, 1)  # HWC → CHW
    return np.expand_dims(arr, 0).astype(np.float32)


def _postprocess(np, Image, pred, orig_size):
    pred = np.squeeze(pred)
    mi, ma = float(pred.min()), float(pred.max())
    if ma - mi > 1e-8:
        pred = (pred - mi) / (ma - mi)
    else:
        pred = np.clip(pred, 0.0, 1.0)
    mask = (pred * 255).astype(np.uint8)
    return Image.fromarray(mask, mode="L").resize(orig_size)


def main() -> None:
    args = _common.parse_io_args("ONNX background removal → alpha matte")
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
    matte = _postprocess(np, Image, outputs[0], image.size)

    rgba = image.convert("RGBA")
    rgba.putalpha(matte)
    try:
        rgba.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
