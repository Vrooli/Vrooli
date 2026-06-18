"""denoise — classical CPU image denoise (no model weights required).

This is a genuine, dependency-light denoise (a median pre-pass to knock out
salt-and-pepper noise, then an edge-preserving smoothing pass via Pillow). It
keeps the `denoise` op runnable on any host that has Pillow, independent of the
ONNX model-zoo provisioning. The `--model` argument is accepted for interface
parity with the ONNX ops and ignored here; an ONNX denoise upgrade can replace
this module without changing the Go provider contract.
"""

from __future__ import annotations

from . import _common


def main() -> None:
    args = _common.parse_io_args("classical CPU denoise")
    try:
        from PIL import Image, ImageFilter
    except ImportError as exc:  # pragma: no cover
        _common.fail(f"missing Pillow ({exc}); pip install pillow", code=3)
        raise

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    # Median removes impulse noise; SMOOTH_MORE does a light low-pass while
    # Pillow's median already preserves edges better than a plain blur.
    out = image.filter(ImageFilter.MedianFilter(size=3))
    out = out.filter(ImageFilter.SMOOTH)

    try:
        out.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
