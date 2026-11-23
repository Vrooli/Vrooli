"""masked_edit — edit only the selected area with an instruction-edit family.

An instruction-edit model (FLUX.2 [klein], FLUX.1 Kontext, Qwen-Image-Edit, ...)
edits the whole picture from a sentence. A selected-area edit needs the opposite
guarantee: everything outside the mask stays byte-identical. This module gives
an instruction family that guarantee by composing three steps:

1. crop the source to the mask bounds plus context and resize the crop to the
   model's comfortable size (small selections are upscaled so the model has
   detail to work with; large ones are downscaled to the size cap);
2. run the family edit on the crop with the user's instruction;
3. paste the result back through the grown, feathered mask, so pixels outside
   the grown mask are the untouched source and the seam fades over
   ``feather_px`` instead of showing a cut line.

Mask semantics match inpaint everywhere else: white = edit, black = keep.

The geometry (``plan_crop``, ``blend_alpha``, ``composite_crop``) is pure
numpy/PIL so it is unit-tested without any model weights.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import time
from typing import NamedTuple, Optional, Tuple

from PIL import Image
from PIL.PngImagePlugin import PngInfo

from . import _common, _diffusers

# Crop sizing. The family pipelines are tuned around ~1MP; snapping to 16 keeps
# every flow-transformer patch grid valid.
SNAP = 16
MIN_LONG_EDGE = 768
DEFAULT_MAX_EDGE = 1024
MAX_EDGE_CEILING = 1536
DEFAULT_GROW_PX = 6
DEFAULT_FEATHER_PX = 14
MAX_GROW_PX = 128
MAX_FEATHER_PX = 128
MIN_CONTEXT_PX = 48
MAX_CONTEXT_PX = 384


class CropPlan(NamedTuple):
    box: Tuple[int, int, int, int]  # source-space crop box (x0, y0, x1, y1)
    infer_size: Tuple[int, int]  # model-input size, multiples of SNAP
    context_px: int
    scale: float


def _snap(value: float) -> int:
    return max(SNAP, int(round(value / SNAP)) * SNAP)


def mask_bounds(mask_binary) -> Tuple[int, int, int, int]:
    """Bounding box (x0, y0, x1, y1) of a boolean mask array, or fail."""
    import numpy as np

    rows = np.flatnonzero(mask_binary.any(axis=1))
    cols = np.flatnonzero(mask_binary.any(axis=0))
    if rows.size == 0 or cols.size == 0:
        raise _common.InputError("masked edit requires a non-empty white mask")
    return int(cols[0]), int(rows[0]), int(cols[-1]) + 1, int(rows[-1]) + 1


def grow_mask(mask_binary, grow_px: int):
    """Dilate a boolean mask by ``grow_px`` (euclidean). 0 returns it unchanged."""
    if grow_px <= 0:
        return mask_binary
    from scipy.ndimage import distance_transform_edt

    return distance_transform_edt(~mask_binary) <= grow_px


def plan_crop(
    image_size: Tuple[int, int],
    bounds: Tuple[int, int, int, int],
    *,
    context_px: int = 0,
    max_edge: int = DEFAULT_MAX_EDGE,
    blend_px: int = 0,
) -> CropPlan:
    """Choose the source crop and the model-input size for a mask's bounds.

    ``bounds`` are the grown mask's. ``blend_px`` is the feather width: the crop
    always keeps at least that much context so the seam fades inside the crop."""
    if not MIN_LONG_EDGE // 2 <= max_edge <= MAX_EDGE_CEILING:
        raise _common.InputError(f"crop max edge must be between {MIN_LONG_EDGE // 2} and {MAX_EDGE_CEILING}")
    if context_px < 0 or context_px > 512:
        raise _common.InputError("crop context must be between 0 and 512 pixels")
    width, height = image_size
    x0, y0, x1, y1 = bounds
    if context_px == 0:
        context_px = max(MIN_CONTEXT_PX, min(MAX_CONTEXT_PX, round(0.5 * max(x1 - x0, y1 - y0))))
    context_px = max(context_px, blend_px)
    box = (
        max(0, x0 - context_px),
        max(0, y0 - context_px),
        min(width, x1 + context_px),
        min(height, y1 + context_px),
    )
    crop_w, crop_h = box[2] - box[0], box[3] - box[1]
    long_edge = max(crop_w, crop_h)
    # Small selections are upscaled to a workable size; large ones capped.
    target = min(max_edge, max(MIN_LONG_EDGE, long_edge))
    scale = target / long_edge
    return CropPlan(box, (_snap(crop_w * scale), _snap(crop_h * scale)), context_px, scale)


def blend_alpha(mask_crop_binary, feather_px: int):
    """Float alpha 0..1: 1 inside the (already grown) mask, fading to exactly 0
    at ``feather_px`` outside it (smoothstep over the euclidean distance), so
    the untouched area starts at a known distance from the mask."""
    import numpy as np

    if feather_px <= 0:
        return mask_crop_binary.astype("float32")
    from scipy.ndimage import distance_transform_edt

    t = np.clip(1.0 - distance_transform_edt(~mask_crop_binary) / float(feather_px), 0.0, 1.0)
    return (t * t * (3.0 - 2.0 * t)).astype("float32")


def composite_crop(original: Image.Image, generated: Image.Image, alpha, box) -> Image.Image:
    """Blend ``generated`` over ``original``'s ``box`` through ``alpha``.

    ``generated`` is resized to the box; pixels where alpha is 0 and everything
    outside the box are the original's bytes.
    """
    import numpy as np

    x0, y0, x1, y1 = box
    out = np.asarray(original.convert("RGB")).copy()
    region = out[y0:y1, x0:x1].astype("float32")
    gen = np.asarray(generated.convert("RGB").resize((x1 - x0, y1 - y0), Image.Resampling.LANCZOS)).astype("float32")
    a = alpha[..., None]
    blended = np.clip(np.rint(gen * a + region * (1.0 - a)), 0, 255).astype("uint8")
    keep = alpha <= 0.0
    blended[keep] = out[y0:y1, x0:x1][keep]
    out[y0:y1, x0:x1] = blended
    return Image.fromarray(out, mode="RGB")


def masked_edit(
    *,
    family: str,
    model_dir: str,
    image_path: str,
    mask_path: str,
    out_path: str,
    params: "_diffusers.Params",
    grow_px: int = DEFAULT_GROW_PX,
    feather_px: int = DEFAULT_FEATHER_PX,
    context_px: int = 0,
    max_edge: int = DEFAULT_MAX_EDGE,
    full_frame: bool = False,
    provenance: Optional[dict] = None,
) -> dict:
    import numpy as np

    started = time.perf_counter()
    if not 0 <= grow_px <= MAX_GROW_PX or not 0 <= feather_px <= MAX_FEATHER_PX:
        raise _common.InputError(f"grow_px and feather_px must be between 0 and {MAX_GROW_PX}")
    try:
        image, source_raster = _common.load_oriented_image(image_path, 16_000_000, mode="RGB")
        mask_image, mask_raster = _common.load_oriented_image(mask_path, 16_000_000, mode="L")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image/mask: {exc}", code=6)
        raise
    if mask_image.size != image.size:
        raise _common.InputError("masked edit mask must match the oriented source image raster")
    binary = np.asarray(mask_image) >= 128
    grown = grow_mask(binary, grow_px)
    bounds = mask_bounds(grown)

    if full_frame:
        box = (0, 0, image.width, image.height)
        plan = None
        crop = image
    else:
        plan = plan_crop(image.size, bounds, context_px=context_px, max_edge=max_edge, blend_px=feather_px)
        box = plan.box
        crop = image.crop(box)
        if crop.size != plan.infer_size:
            crop = crop.resize(plan.infer_size, Image.Resampling.LANCZOS)
        params = params._replace(width=plan.infer_size[0], height=plan.infer_size[1])

    generated, receipt = _diffusers.run_family_images(
        family=family, model_dir=model_dir, images=[crop], params=params, op="masked_edit", started=started
    )
    x0, y0, x1, y1 = box
    alpha = blend_alpha(grown[y0:y1, x0:x1], feather_px)
    result = composite_crop(image, generated, alpha, box)

    changed_outside = int(np.count_nonzero(
        (np.asarray(result) != np.asarray(image)).any(axis=2) & ~grow_mask(binary, grow_px + feather_px)
    ))
    receipt.update(dict(provenance or {}))
    receipt.update({
        "schema": "vrooli.image-tools.masked-edit/v1",
        "mask_semantics": "white pixels edited; pixels outside the grown, feathered mask are the untouched source",
        "source_raster": source_raster,
        "mask_raster": mask_raster,
        "grow_px": grow_px,
        "feather_px": feather_px,
        "crop_box_xyxy": list(box),
        "crop_plan": None if plan is None else {"context_px": plan.context_px, "scale": round(plan.scale, 4), "inference_size": list(plan.infer_size)},
        "full_frame": full_frame,
        "mask_area_fraction": round(float(binary.mean()), 6),
        "changed_pixels_outside_blend_zone": changed_outside,
        "elapsed_ms": round((time.perf_counter() - started) * 1000, 3),
    })
    try:
        info = PngInfo()
        info.add_text("vrooli_masked_edit_provenance", json.dumps(receipt, sort_keys=True, separators=(",", ":")))
        result.save(out_path, format="PNG", pnginfo=info)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)
    return receipt


def _sha256(path: str) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="instruction-edit family restricted to a mask (crop, edit, feathered paste-back)")
    p.add_argument("--model", required=True, help="path to the diffusers model dir")
    p.add_argument("--model-id", default="", help="catalog model identifier")
    p.add_argument("--family", required=True, help="registered diffusers family adapter (runtime.family)")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--mask", required=True, help="mask image path (white = edit, black = keep)")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--prompt", required=True, help="the natural-language edit instruction")
    p.add_argument("--negative-prompt", dest="negative_prompt", default="", help="optional negative prompt")
    p.add_argument("--steps", type=int, default=0, help="sampler steps (0 = family default)")
    p.add_argument("--guidance", type=float, default=None, help="text guidance scale")
    p.add_argument("--image-guidance", dest="image_guidance", type=float, default=None, help="family-specific identity/edit-strength knob")
    p.add_argument("--seed", type=int, default=0, help="RNG seed (0 = nondeterministic)")
    p.add_argument("--grow-px", type=int, default=DEFAULT_GROW_PX, help="grow the mask by this many pixels before blending")
    p.add_argument("--feather-px", type=int, default=DEFAULT_FEATHER_PX, help="soften the paste-back seam over this many pixels")
    p.add_argument("--crop-context-pixels", type=int, default=0, help="source pixels of context around the mask (0 = automatic)")
    p.add_argument("--crop-max-edge-px", type=int, default=DEFAULT_MAX_EDGE, help="largest edge of the model input")
    p.add_argument("--full-frame", action="store_true", help="edit the whole frame and paste back through the mask (no crop)")
    return _common.parse_device_args(p)


def main() -> None:
    args = _parse_args()
    if not args.prompt.strip():
        _common.fail("masked edit requires a non-empty --prompt (the edit instruction)", code=2)
    params = _diffusers.Params(
        prompt=args.prompt,
        steps=args.steps,
        guidance=args.guidance,
        image_guidance=args.image_guidance,
        negative_prompt=args.negative_prompt or None,
        seed=args.seed,
    )
    provenance = {
        "model_id": args.model_id,
        "source_sha256": _sha256(args.image),
        "mask_sha256": _sha256(args.mask),
        "prompt": args.prompt,
    }
    try:
        masked_edit(
            family=args.family,
            model_dir=args.model,
            image_path=args.image,
            mask_path=args.mask,
            out_path=args.out,
            params=params,
            grow_px=args.grow_px,
            feather_px=args.feather_px,
            context_px=args.crop_context_pixels,
            max_edge=args.crop_max_edge_px,
            full_frame=args.full_frame,
            provenance=provenance,
        )
    except ValueError as exc:
        _common.fail(str(exc), code=2)


if __name__ == "__main__":
    main()
