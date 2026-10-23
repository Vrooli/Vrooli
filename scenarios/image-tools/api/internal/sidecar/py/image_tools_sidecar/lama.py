"""Pinned fixed-512 LaMa ONNX repair with source-size, outside-mask preservation."""
from __future__ import annotations

import argparse
import hashlib
import io
import json
import os
import resource
import time

import numpy as np
from PIL import Image, ImageFilter, PngImagePlugin

from . import _common

MODEL_SHA256 = "1faef5301d78db7dda502fe59966957ec4b79dd64e16f03ed96913c7a4eb68d6"
MODEL_REVISION = "c3c0c9e468934d62e79c329e35d82dd09ff8c444"
# Filled only with the measured graph hash after native export and qualification.
MODEL_1024_SHA256 = "ed44a7c0468b633e106078b3e700ed5e2b0187d87efbdf9e6b03e08fe1b7a2b0"
MODEL_1024_REVISION = "f5fb39a18022c34a71bf9a47a6ec393c804b49ca+AEmotionStudio/353e64100236fd934e1dcbaa1cbff13ea8362151"
MAX_PIXELS = 16_000_000
TILE_EDGE = 1024
TILE_OVERLAP = 256


def _sha256(path):
    digest = hashlib.sha256()
    with open(path, "rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def effective_removal_mask(mask: Image.Image, expansion_pixels=0, removal_regions=()):
    """Return a new mask formed only from explicitly requested additions."""
    if not isinstance(expansion_pixels, int) or isinstance(expansion_pixels, bool) or not 0 <= expansion_pixels <= 64:
        raise ValueError("mask_expansion_pixels must be an integer from 0 to 64")
    if len(removal_regions) > 32:
        raise ValueError("at most 32 additional removal regions are allowed")
    binary = mask.convert("L").point(lambda value: 255 if value >= 128 else 0)
    if expansion_pixels:
        binary = binary.filter(ImageFilter.MaxFilter(expansion_pixels * 2 + 1))
    if removal_regions:
        from PIL import ImageDraw
        draw = ImageDraw.Draw(binary)
        width, height = binary.size
        for region in removal_regions:
            if not isinstance(region, dict):
                raise ValueError("each removal region must be a normalized box object")
            x, y, w, h = (region.get(key) for key in ("x", "y", "width", "height"))
            if any(not isinstance(v, (int, float)) or isinstance(v, bool) or not np.isfinite(v) for v in (x, y, w, h)):
                raise ValueError("removal region coordinates must be finite numbers")
            if x < 0 or y < 0 or w <= 0 or h <= 0 or x + w > 1 or y + h > 1:
                raise ValueError("removal region must have positive area within 0..1")
            x0, y0 = int(np.floor(x * width)), int(np.floor(y * height))
            x1, y1 = min(width, int(np.ceil((x + w) * width))), min(height, int(np.ceil((y + h) * height)))
            draw.rectangle((x0, y0, x1 - 1, y1 - 1), fill=255)
    return binary


def _positions(length):
    if length <= TILE_EDGE:
        return [0]
    step = TILE_EDGE - TILE_OVERLAP
    result = list(range(0, length - TILE_EDGE + 1, step))
    last = length - TILE_EDGE
    if result[-1] != last:
        result.append(last)
    return result


def _model_resolution(session):
    inputs = {value.name: (value.type, value.shape) for value in session.get_inputs()}
    for resolution in (512, 1024):
        expected = {"image": ("tensor(float)", ["batch", 3, resolution, resolution]),
                    "mask": ("tensor(float)", ["batch", 1, resolution, resolution])}
        if inputs == expected:
            return resolution
    raise ValueError(f"LaMa export has an unqualified image/mask tensor signature: {inputs!r}")


def _repair_crop(session, crop, crop_mask, resolution):
    resized, offset, pixels = _letterbox(crop, resolution)
    square_mask = np.zeros((resolution, resolution), dtype=np.float32)
    square_mask[offset[1]:offset[1]+resized[1], offset[0]:offset[0]+resized[0]] = (
        np.asarray(crop_mask.resize(resized, Image.Resampling.NEAREST)) >= 128)
    output = session.run(None, {"image": (pixels.astype(np.float32)/255).transpose(2, 0, 1)[None],
                                "mask": square_mask[None, None]})[0]
    if output.shape != (1, 3, resolution, resolution) or not np.isfinite(output).all():
        raise ValueError(f"LaMa output must be finite float RGB with shape 1×3×{resolution}×{resolution}")
    decoded = Image.fromarray(np.clip(output[0].transpose(1, 2, 0), 0, 255).round().astype(np.uint8))
    decoded = decoded.crop((offset[0], offset[1], offset[0]+resized[0], offset[1]+resized[1]))
    return decoded.resize(crop.size, Image.Resampling.BICUBIC), resized, offset


def _letterbox(crop, resolution=512):
    scale = resolution / max(crop.size)
    resized = (max(1, round(crop.width * scale)), max(1, round(crop.height * scale)))
    offset = ((resolution - resized[0]) // 2, (resolution - resized[1]) // 2)
    pixels = np.asarray(crop.resize(resized, Image.Resampling.LANCZOS))
    pixels = np.pad(pixels, ((offset[1], resolution-resized[1]-offset[1]),
                            (offset[0], resolution-resized[0]-offset[0]), (0, 0)), mode="edge")
    return resized, offset, pixels


def _repair_tiled(session, source, effective_mask, resolution):
    width, height = source.size
    xs, ys = _positions(width), _positions(height)
    rgb = np.asarray(source.convert("RGB"), dtype=np.float32)
    selected = np.asarray(effective_mask) >= 128
    accum = np.zeros_like(rgb, dtype=np.float32)
    weights = np.zeros((height, width), dtype=np.float32)
    records = []

    def axis_weight(coords, start, end, positions, length):
        weight = np.ones_like(coords, dtype=np.float32)
        index = positions.index(start)
        if index > 0:
            overlap_start = start
            overlap_end = min(positions[index-1] + TILE_EDGE, end)
            overlap = max(1, overlap_end - overlap_start)
            weight *= np.clip((coords - overlap_start + 0.5) / overlap, 0, 1)
        if index + 1 < len(positions):
            next_start = positions[index+1]
            overlap = max(1, end - next_start)
            weight *= np.clip((end - coords - 0.5) / overlap, 0, 1)
        return weight

    for y in ys:
        for x in xs:
            right, bottom = min(width, x + TILE_EDGE), min(height, y + TILE_EDGE)
            local_mask = effective_mask.crop((x, y, right, bottom))
            if not np.asarray(local_mask).any():
                continue
            crop = source.convert("RGB").crop((x, y, right, bottom))
            repaired, resized, offset = _repair_crop(session, crop, local_mask, resolution)
            tile = np.asarray(repaired, dtype=np.float32)
            wx = axis_weight(np.arange(x, right, dtype=np.float32), x, right, xs, width)
            wy = axis_weight(np.arange(y, bottom, dtype=np.float32), y, bottom, ys, height)
            weight = wy[:, None] * wx[None, :]
            accum[y:bottom, x:right] += tile * weight[:, :, None]
            weights[y:bottom, x:right] += weight
            records.append({"crop_xyxy_oriented_pixels": [x, y, right, bottom],
                            "crop_resized_pixels": list(resized), "letterbox_offset_pixels": list(offset)})
    result = rgb.copy()
    generated = np.divide(accum, weights[:, :, None], out=np.zeros_like(accum), where=weights[:, :, None] > 0)
    result[selected] = generated[selected]
    return Image.fromarray(np.clip(result, 0, 255).round().astype(np.uint8)), records


def repair(session, source: Image.Image, mask: Image.Image, expansion_pixels=0, removal_regions=()):
    """White pixels are repaired; every unmasked source pixel stays unchanged.

    One contextual crop is letterboxed, never stretched, into the qualified
    export's 512-square raster. Newly exposed scenery is an estimate, not
    reconstruction of hidden ground truth.
    """
    started = time.perf_counter()
    if source.size != mask.size:
        raise ValueError("repair mask must match the oriented source raster")
    resolution = _model_resolution(session)
    original_binary = np.asarray(mask.convert("L")) >= 128
    effective_mask = effective_removal_mask(mask, expansion_pixels, removal_regions)
    binary = np.asarray(effective_mask) >= 128
    ys, xs = np.where(binary)
    if not len(xs):
        raise ValueError("repair mask has no white removal pixels")
    if binary.all():
        raise ValueError("repair mask must retain surrounding scene context")
    width, height = source.size
    margin = max(32, int(max(xs.max() - xs.min() + 1, ys.max() - ys.min() + 1) * .25))
    bounds = (max(0, int(xs.min()) - margin), max(0, int(ys.min()) - margin),
              min(width, int(xs.max()) + margin + 1), min(height, int(ys.max()) + margin + 1))
    crop = source.convert("RGB").crop(bounds)
    crop_mask = effective_mask.crop(bounds)
    tiled = max(crop.size) > TILE_EDGE
    if tiled:
        result, tile_records = _repair_tiled(session, source, effective_mask, resolution)
        resized, offset = (0, 0), (0, 0)
    else:
        decoded, resized, offset = _repair_crop(session, crop, crop_mask, resolution)
        result = source.convert("RGB").copy()
        result.paste(decoded, bounds[:2], crop_mask)
    mask_bytes = io.BytesIO()
    effective_mask.save(mask_bytes, format="PNG")
    effective_png = mask_bytes.getvalue()
    return result, {"crop_xyxy_oriented_pixels": list(bounds), "crop_resized_pixels": list(resized),
                    "letterbox_offset_pixels": list(offset), "model_raster": [resolution, resolution],
                    "tiled_inference": tiled, "tile_crops": tile_records if tiled else [],
                    "removed_pixels": int(binary.sum()), "original_mask_pixels": int(original_binary.sum()),
                    "effective_mask_added_pixels": int(np.count_nonzero(binary & ~original_binary)),
                    "mask_expansion_pixels": expansion_pixels,
                    "removal_regions_normalized": list(removal_regions),
                    "effective_mask_png_sha256": hashlib.sha256(effective_png).hexdigest(),
                    "effective_mask_png_base64": __import__("base64").b64encode(effective_png).decode("ascii"),
                    "outside_effective_mask": "original decoded RGB pixels",
                    "repair_elapsed_ms": round((time.perf_counter() - started) * 1000, 3),
                    "process_peak_rss_bytes": int(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * 1024),
                    "process_peak_rss_scope": "cumulative sidecar process high-water mark",
                    "fill_status": "estimated hidden scenery; requires visual review"}


def main():
    parser = argparse.ArgumentParser(description="Local LaMa ONNX background repair")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--mask", required=True)
    parser.add_argument("--params", default="{}", help="JSON BackgroundRepairParams")
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    model = _common.resolve_onnx_path(args.model)
    if os.path.basename(model) == "lama_1024.onnx":
        if not MODEL_1024_SHA256 or _sha256(model) != MODEL_1024_SHA256:
            raise ValueError("LaMa checkpoint does not match the pinned fixed-1024 export")
        model_revision, model_sha256 = MODEL_1024_REVISION, MODEL_1024_SHA256
    else:
        if _sha256(model) != MODEL_SHA256:
            raise ValueError("LaMa checkpoint does not match the qualified fixed-512 export")
        model_revision, model_sha256 = MODEL_REVISION, MODEL_SHA256
    source, source_raster = _common.load_oriented_image(args.image, MAX_PIXELS)
    mask, mask_raster = _common.load_oriented_image(args.mask, MAX_PIXELS, mode="L")
    params = json.loads(args.params)
    session_started = time.perf_counter()
    session = _common.make_session(model)
    session_load_ms = round((time.perf_counter() - session_started) * 1000, 3)
    result, receipt = repair(session, source, mask,
                             params.get("maskExpansionPixels", 0), params.get("removalRegions", []))
    receipt["model_session_load_ms"] = session_load_ms
    receipt["onnx_thread_settings"] = _common.onnx_thread_settings()
    receipt["cpu_quota_or_affinity"] = "not set by sidecar thread policy"
    receipt.update({"schema": "vrooli.image-tools.background-repair/v1",
                    "model_revision": model_revision, "model_sha256": model_sha256,
                    "source_sha256": _sha256(args.image), "mask_sha256": _sha256(args.mask),
                    "source_raster": source_raster, "mask_raster": mask_raster})
    metadata = PngImagePlugin.PngInfo()
    metadata.add_text("vrooli_repair_provenance", json.dumps(receipt, sort_keys=True, separators=(",", ":")))
    result.save(args.out, format="PNG", pnginfo=metadata)


if __name__ == "__main__":
    main()
