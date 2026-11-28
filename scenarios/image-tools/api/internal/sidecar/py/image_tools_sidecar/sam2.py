"""SAM 2.1 image segmentation through the Transformers SAM2 API.

This adapter intentionally uses the versioned Hugging Face Transformers API and
the model directory's local config and weight files. It does not import
the SAM 1 ONNX decoder contract from ``segment``.
"""

from __future__ import annotations

import argparse
from functools import lru_cache
import json
import math
import os
import platform
import sys
import time

try:
    import resource
except ImportError:  # Windows does not expose getrusage.
    resource = None

from PIL import Image

from . import _common


# A warm worker serves one model residency at a time. Keep exactly one loaded
# checkpoint; model switches replace it instead of accumulating VRAM/RAM.
_MODEL_CACHE_KEY = None
_MODEL_CACHE = None


@lru_cache(maxsize=1)
def _processor():
    from transformers import Sam2Processor
    from transformers.models.sam2.image_processing_sam2_fast import Sam2ImageProcessorFast

    return Sam2Processor(
        image_processor=Sam2ImageProcessorFast(
            size={"height": 1024, "width": 1024},
            mask_size={"height": 256, "width": 256},
            do_resize=True,
            default_to_square=True,
            resample=Image.Resampling.BILINEAR,
            do_rescale=True,
            rescale_factor=1.0 / 255.0,
            do_normalize=True,
            image_mean=[0.485, 0.456, 0.406],
            image_std=[0.229, 0.224, 0.225],
            do_convert_rgb=True,
        ),
        target_size=1024,
    )


def _load_model(model_dir: str, device):
    import torch
    from transformers import Sam2Model

    return Sam2Model.from_pretrained(model_dir, local_files_only=True, low_cpu_mem_usage=True).to(device).eval()


def _resident_model(model_dir: str, device):
    global _MODEL_CACHE_KEY, _MODEL_CACHE
    key = (model_dir, str(device))
    if key == _MODEL_CACHE_KEY and _MODEL_CACHE is not None:
        return _MODEL_CACHE, True, 0.0
    import torch

    previous_device = _MODEL_CACHE_KEY[1] if _MODEL_CACHE_KEY else ""
    _MODEL_CACHE = None
    _MODEL_CACHE_KEY = None
    if previous_device.startswith("cuda"):
        torch.cuda.empty_cache()
    started = time.monotonic()
    model = _load_model(model_dir, device)
    _MODEL_CACHE, _MODEL_CACHE_KEY = model, key
    return model, False, (time.monotonic() - started) * 1000


def smoke(model_dir: str, deep: bool = False) -> str:
    """Load the pinned checkpoint locally, optionally proving a tiny forward pass."""
    model = _load_model(model_dir, "cpu")
    parameter_count = sum(parameter.numel() for parameter in model.parameters())
    if parameter_count <= 0:
        raise RuntimeError("SAM 2 checkpoint loaded without model parameters")
    summary = f"sam2 smoke OK: loaded {parameter_count} parameters from {model_dir}"
    if deep:
        import torch

        image = Image.new("RGB", (96, 96), (96, 144, 192))
        processor = _processor()
        inputs = processor(
            images=image,
            return_tensors="pt",
            **_prompts('{"points":[{"x":0.5,"y":0.5}]}', image.size),
        )
        with torch.inference_mode():
            outputs = model(**inputs, multimask_output=True)
        if outputs.pred_masks.numel() == 0 or outputs.iou_scores.numel() == 0:
            raise RuntimeError("SAM 2 forward pass returned empty mask or score tensors")
        if not torch.isfinite(outputs.pred_masks).all() or not torch.isfinite(outputs.iou_scores).all():
            raise RuntimeError("SAM 2 forward pass returned non-finite outputs")
        summary += "; synthetic 96x96 point-prompt forward passed"
    del model
    return summary


def _prompts(raw: str, size: tuple[int, int]) -> dict:
    prompt = json.loads(raw)
    if not isinstance(prompt, dict) or set(prompt) - {"points", "box"}:
        raise _common.InputError("SAM 2 prompt must contain only points and/or box")
    points = prompt.get("points", [])
    if not isinstance(points, list) or len(points) > 64:
        raise _common.InputError("SAM 2 requires at most 64 point prompts")
    width, height = size

    def unit(value):
        if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or not 0 <= value <= 1:
            raise _common.InputError("SAM 2 coordinates must be finite numbers in 0..1")
        return float(value)

    coords, labels = [], []
    for point in points:
        if not isinstance(point, dict) or set(point) - {"x", "y", "negative"} or "x" not in point or "y" not in point:
            raise _common.InputError("SAM 2 point must contain x, y and optional negative")
        negative = point.get("negative", False)
        if not isinstance(negative, bool):
            raise _common.InputError("SAM 2 negative must be boolean")
        coords.append([unit(point["x"]) * width, unit(point["y"]) * height])
        labels.append(0 if negative else 1)

    box = prompt.get("box")
    result = {}
    if box is not None:
        if not isinstance(box, dict) or set(box) != {"x", "y", "width", "height"}:
            raise _common.InputError("SAM 2 box must contain x, y, width and height")
        x, y, box_w, box_h = (unit(box[key]) for key in ("x", "y", "width", "height"))
        if box_w <= 0 or box_h <= 0 or x + box_w > 1 or y + box_h > 1:
            raise _common.InputError("SAM 2 box must have positive area inside the image")
        result["input_boxes"] = [[[x * width, y * height, (x + box_w) * width, (y + box_h) * height]]]

    if coords:
        result["input_points"] = [[coords]]
        result["input_labels"] = [[labels]]
    if not (any(label == 1 for label in labels) or box is not None):
        raise _common.InputError("SAM 2 requires a positive point or an object box")
    return result


def _write_mask(mask, out_path: str) -> None:
    try:
        Image.fromarray(mask).save(out_path, format="PNG")
    except Exception as exc:  # noqa: BLE001
        raise RuntimeError(f"failed to write output {out_path!r}: {exc}") from exc


def _max_rss_kib():
    if resource is None:
        return None
    peak = int(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss)
    # Linux reports KiB; macOS and the BSDs report bytes.
    if sys.platform == "darwin" or "bsd" in platform.system().lower():
        peak //= 1024
    return peak


def segmenter_dir(model_dir: str) -> str:
    """The SAM 2.1 checkpoint directory of a model install.

    A plain SAM 2.1 install keeps its config and weights at the top level; a
    grounded install (detector + segmenter) keeps them under ``segmenter/``.
    """
    nested = os.path.join(model_dir, "segmenter")
    return nested if os.path.isdir(nested) else model_dir


def run(model_dir: str, image_path: str, out_path: str, prompt_json: str, device_name: str) -> dict:
    import torch

    model_dir = segmenter_dir(model_dir)

    if device_name not in {"cpu", "cuda"}:
        raise ValueError("SAM 2 device must be cpu or cuda")
    if device_name == "cuda" and not torch.cuda.is_available():
        raise RuntimeError("SAM 2 requested CUDA, but CUDA is unavailable")
    device = torch.device(device_name)
    started = time.monotonic()
    preprocess_started = started
    image, _source_raster = _common.load_oriented_image(image_path, 16_000_000)
    # These preprocessing defaults match the checkpoint's pinned upstream
    # preprocessor_config.json. The small HF metadata files are under Image
    # Tools' binary-artifact size floor, so keep the reviewed contract explicit.
    processor = _processor()
    inputs = processor(
        images=image,
        return_tensors="pt",
        **_prompts(prompt_json, image.size),
    )
    preprocess_ms = (time.monotonic() - preprocess_started) * 1000
    model, cache_hit, model_load_ms = _resident_model(model_dir, device)
    inputs = inputs.to(device)
    if device.type == "cuda":
        torch.cuda.reset_peak_memory_stats(device)
        # CUDA kernels are asynchronous; synchronize before and after the
        # forward pass so the phase metric represents actual device time.
        torch.cuda.synchronize(device)
    forward_started = time.monotonic()
    with torch.inference_mode():
        outputs = model(**inputs, multimask_output=True)
    if device.type == "cuda":
        torch.cuda.synchronize(device)
    forward_ms = (time.monotonic() - forward_started) * 1000
    postprocess_started = time.monotonic()
    scores = outputs.iou_scores[0, 0]
    selected = outputs.pred_masks[0, 0, int(torch.argmax(scores).item())].unsqueeze(0).unsqueeze(0)
    masks = processor.post_process_masks(
        [selected],
        inputs["original_sizes"],
        mask_threshold=0.0,
        binarize=True,
    )
    mask = (masks[0][0, 0].detach().cpu().numpy().astype("uint8") * 255)
    _write_mask(mask, out_path)
    postprocess_ms = (time.monotonic() - postprocess_started) * 1000
    metrics = {
            "device": device_name,
            "elapsed_ms": round((time.monotonic() - started) * 1000),
            "preprocess_ms": round(preprocess_ms),
            "model_load_ms": round(model_load_ms),
            "model_cache_hit": cache_hit,
            "forward_ms": round(forward_ms),
            "postprocess_ms": round(postprocess_ms),
            "max_rss_kib": _max_rss_kib(),
            "cuda_peak_allocated_bytes": int(torch.cuda.max_memory_allocated(device)) if device.type == "cuda" else 0,
        }
    print(json.dumps({"sam2_metrics": metrics}), file=sys.stderr, flush=True)
    return metrics


def main() -> dict | None:
    parser = argparse.ArgumentParser(description="SAM 2.1 image segmentation")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--prompts", default="{}")
    parser.add_argument("--device", choices=("cpu", "cuda"), default="cpu")
    args = parser.parse_args()
    try:
        return run(args.model, args.image, args.out, args.prompts, args.device)
    except Exception as exc:  # noqa: BLE001
        print(f"SAM 2 segmentation failed: {exc}")
        raise SystemExit(_common.exit_status_for(exc)) from exc


if __name__ == "__main__":
    main()
