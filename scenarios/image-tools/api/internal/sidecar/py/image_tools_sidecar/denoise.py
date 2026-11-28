"""denoise — blind real-world denoising with a SCUNet-family ONNX model.

The model takes a ``[1, 3, H, W]`` float32 RGB tensor in ``[0, 1]`` (H and W
multiples of 8) and returns the denoised tensor of the same shape. Big images
run as overlapping tiles: every tile is processed with a margin of context and
only its interior is kept, so tile seams do not show. The reflect-padded border
is cropped away, so the output has exactly the input's size.

Fails loud (non-zero exit, actionable stderr) when the runtime deps or weights
are absent. There is no classical fallback: the registry's denoise model is the
weights, so a missing or non-ONNX model is an error, never a silent smoothing.
"""

from __future__ import annotations

from . import _common, _tiling

TILE = 512  # interior edge of one tile, in pixels
MARGIN = 32  # context pixels kept on every side of a tile and then discarded
MULTIPLE = 64  # the Swin-UNet downsamples three times; pad to a safe multiple


def denoise_array(np, session, rgb, tile=TILE, margin=MARGIN):
    """Denoise an HxWx3 float32 array in [0, 1]; returns the same shape."""
    out = _tiling.tiled_apply(np, session, rgb, tile=tile, margin=margin, multiple=MULTIPLE, pass_multiple=8)
    return np.clip(out, 0.0, 1.0)


def main() -> None:
    args = _common.parse_io_args("ONNX blind denoising (SCUNet family)")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    source = np.asarray(image, dtype=np.float32) / 255.0
    result = denoise_array(np, session, source)
    out = Image.fromarray((result * 255.0 + 0.5).astype(np.uint8), mode="RGB")
    try:
        out.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
