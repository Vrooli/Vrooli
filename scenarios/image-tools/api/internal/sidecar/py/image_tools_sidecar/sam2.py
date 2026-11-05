"""SAM 2.1 image segmentation through the Transformers SAM2 API.

This adapter intentionally uses the versioned Hugging Face Transformers API and
the model directory's local config and weight files. It does not import
the SAM 1 ONNX decoder contract from ``segment``.
"""

from __future__ import annotations

import argparse
import json
import math
import platform
import sys
import time

try:
    import resource
except ImportError:  # Windows does not expose getrusage.
    resource = None

from PIL import Image

from . import _common


def _prompts(raw: str, size: tuple[int, int]) -> dict:
    prompt = json.loads(raw)
    if not isinstance(prompt, dict) or set(prompt) - {"points", "box"}:
        raise ValueError("SAM 2 prompt must contain only points and/or box")
    points = prompt.get("points", [])
    if not isinstance(points, list) or len(points) > 64:
        raise ValueError("SAM 2 requires at most 64 point prompts")
    width, height = size

    def unit(value):
        if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value) or not 0 <= value <= 1:
            raise ValueError("SAM 2 coordinates must be finite numbers in 0..1")
        return float(value)

    coords, labels = [], []
    for point in points:
        if not isinstance(point, dict) or set(point) - {"x", "y", "negative"} or "x" not in point or "y" not in point:
            raise ValueError("SAM 2 point must contain x, y and optional negative")
        negative = point.get("negative", False)
        if not isinstance(negative, bool):
            raise ValueError("SAM 2 negative must be boolean")
        coords.append([unit(point["x"]) * width, unit(point["y"]) * height])
        labels.append(0 if negative else 1)

    box = prompt.get("box")
    result = {}
    if box is not None:
        if not isinstance(box, dict) or set(box) != {"x", "y", "width", "height"}:
            raise ValueError("SAM 2 box must contain x, y, width and height")
        x, y, box_w, box_h = (unit(box[key]) for key in ("x", "y", "width", "height"))
        if box_w <= 0 or box_h <= 0 or x + box_w > 1 or y + box_h > 1:
            raise ValueError("SAM 2 box must have positive area inside the image")
        result["input_boxes"] = [[[x * width, y * height, (x + box_w) * width, (y + box_h) * height]]]

    if coords:
        result["input_points"] = [[coords]]
        result["input_labels"] = [[labels]]
    if not (any(label == 1 for label in labels) or box is not None):
        raise ValueError("SAM 2 requires a positive point or an object box")
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


def run(model_dir: str, image_path: str, out_path: str, prompt_json: str, device_name: str) -> None:
    import torch
    from transformers import Sam2Model, Sam2Processor
    from transformers.models.sam2.image_processing_sam2_fast import Sam2ImageProcessorFast

    if device_name not in {"cpu", "cuda"}:
        raise ValueError("SAM 2 device must be cpu or cuda")
    if device_name == "cuda" and not torch.cuda.is_available():
        raise RuntimeError("SAM 2 requested CUDA, but CUDA is unavailable")
    device = torch.device(device_name)
    started = time.monotonic()
    image, _source_raster = _common.load_oriented_image(image_path, 16_000_000)
    # These preprocessing defaults match the checkpoint's pinned upstream
    # preprocessor_config.json. The small HF metadata files are under Image
    # Tools' binary-artifact size floor, so keep the reviewed contract explicit.
    processor = Sam2Processor(
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
    inputs = processor(
        images=image,
        return_tensors="pt",
        **_prompts(prompt_json, image.size),
    )
    model = Sam2Model.from_pretrained(model_dir, local_files_only=True, low_cpu_mem_usage=True).to(device).eval()
    inputs = inputs.to(device)
    with torch.inference_mode():
        outputs = model(**inputs, multimask_output=True)
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
    print(json.dumps({
        "sam2_metrics": {
            "device": device_name,
            "elapsed_ms": round((time.monotonic() - started) * 1000),
            "max_rss_kib": _max_rss_kib(),
            "cuda_peak_allocated_bytes": int(torch.cuda.max_memory_allocated(device)) if device.type == "cuda" else 0,
        }
    }), file=sys.stderr, flush=True)
    del model, inputs, outputs
    if device.type == "cuda":
        torch.cuda.empty_cache()


def main() -> None:
    parser = argparse.ArgumentParser(description="SAM 2.1 image segmentation")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--prompts", default="{}")
    parser.add_argument("--device", choices=("cpu", "cuda"), default="cpu")
    args = parser.parse_args()
    try:
        run(args.model, args.image, args.out, args.prompts, args.device)
    except Exception as exc:  # noqa: BLE001
        print(f"SAM 2 segmentation failed: {exc}")
        raise SystemExit(1) from exc


if __name__ == "__main__":
    main()
