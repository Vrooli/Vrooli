"""depth_map — estimate a per-pixel depth map from a single image with a
Depth-Anything-family ONNX model (CPU-only, onnxruntime CPUExecutionProvider).

The model emits a single-channel relative-depth prediction. We normalize it to
0..255 (near = bright) and write a grayscale PNG at the original resolution — a
visual depth map usable as a mask, a relight input, or a 3D cue.

Fails loud (non-zero exit, actionable stderr) when deps/weights are absent.
"""

from __future__ import annotations

from . import _common


def _normalize_depth(np, pred):
    """Squeeze to HxW and min-max normalize to a 0..255 uint8 depth image."""
    depth = np.squeeze(pred)
    if depth.ndim == 3:
        # Some exports keep a leading channel; collapse it.
        depth = depth[0] if depth.shape[0] == 1 else depth.mean(axis=0)
    mi, ma = float(depth.min()), float(depth.max())
    if ma - mi > 1e-8:
        depth = (depth - mi) / (ma - mi)
    else:
        depth = np.clip(depth, 0.0, 1.0)
    return (depth * 255).astype(np.uint8)


def _preprocess(np, image, size_hw):
    """Resize and ImageNet-normalize RGB input for Depth-Anything exports."""
    h, w = size_hw
    resized = image.resize((w, h))
    arr = np.array(resized).astype(np.float32) / 255.0
    mean = np.array([0.485, 0.456, 0.406], dtype=np.float32)
    std = np.array([0.229, 0.224, 0.225], dtype=np.float32)
    arr = (arr - mean) / std
    return arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)


def main() -> None:
    args = _common.parse_io_args("ONNX monocular depth estimation")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    h, w = _common.model_input_hw(session)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    inp = _preprocess(np, image, (h, w))

    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    depth = _normalize_depth(np, outputs[0])

    out_img = Image.fromarray(depth, mode="L").resize(image.size)
    try:
        out_img.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
