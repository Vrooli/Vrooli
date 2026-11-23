"""Grounded SAM 2: turn a text phrase ("the jacket", "hair", "the book") into a mask.

Two models in one install directory:

* ``detector/`` Grounding DINO: phrase -> scored boxes (open vocabulary);
* ``segmenter/`` SAM 2.1: the best box -> a clean silhouette mask.

The detector decides *what* the phrase means and *where* it is; SAM 2.1 decides
the exact boundary, which is what an edit's mask needs. The module writes the
same single-channel PNG mask as ``sam2`` and prints the same metrics line, so the
warm worker and the AI engine treat it as one more segment technique.

Prompt contract (``--prompts`` JSON): ``{"text": "<phrase>"}``.
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time


from . import _common, sam2

BOX_THRESHOLD = 0.25
TEXT_THRESHOLD = 0.2
MAX_PHRASE_CHARS = 200

# One resident detector, like sam2's single resident segmenter: a model switch
# replaces it instead of accumulating VRAM.
_DETECTOR_KEY = None
_DETECTOR = None


def parse_phrase(raw: str) -> str:
    prompt = json.loads(raw or "{}")
    if not isinstance(prompt, dict) or set(prompt) - {"text"}:
        raise _common.InputError("grounded segmentation prompt must contain only text")
    text = prompt.get("text")
    if not isinstance(text, str) or not text.strip():
        raise _common.InputError("grounded segmentation requires a non-empty text phrase")
    phrase = " ".join(text.split())
    if len(phrase) > MAX_PHRASE_CHARS:
        raise _common.InputError(f"text phrase must be at most {MAX_PHRASE_CHARS} characters")
    return phrase


def detector_query(phrase: str) -> str:
    """Grounding DINO wants one lower-case phrase ending in a full stop."""
    return phrase.lower().strip().rstrip(".").strip() + "."


def best_box(boxes, scores):
    """The highest-scoring box, or None when nothing matched."""
    if len(scores) == 0:
        return None
    index = max(range(len(scores)), key=lambda i: float(scores[i]))
    return [float(v) for v in boxes[index]], float(scores[index])


def layout(model_dir: str):
    detector, segmenter = os.path.join(model_dir, "detector"), os.path.join(model_dir, "segmenter")
    if not (os.path.isdir(detector) and os.path.isdir(segmenter)):
        raise _common.InputError(
            f"model {model_dir!r} has no detector/ and segmenter/ directories; "
            "a text phrase needs a grounded segmentation model (grounded-sam-2)"
        )
    return detector, segmenter


def _resident_detector(detector_dir: str, device):
    global _DETECTOR_KEY, _DETECTOR
    key = (detector_dir, str(device))
    if key == _DETECTOR_KEY and _DETECTOR is not None:
        return _DETECTOR
    import torch
    from transformers import AutoModelForZeroShotObjectDetection, AutoProcessor

    _DETECTOR, _DETECTOR_KEY = None, None
    if device.type == "cuda":
        torch.cuda.empty_cache()
    processor = AutoProcessor.from_pretrained(detector_dir, local_files_only=True)
    model = AutoModelForZeroShotObjectDetection.from_pretrained(detector_dir, local_files_only=True).to(device).eval()
    _DETECTOR, _DETECTOR_KEY = (processor, model), key
    return _DETECTOR


def smoke(model_dir: str, deep: bool = False) -> str:
    detector_dir, segmenter_dir = layout(model_dir)
    sam_summary = sam2.smoke(segmenter_dir, deep=deep)
    from transformers import AutoModelForZeroShotObjectDetection

    model = AutoModelForZeroShotObjectDetection.from_pretrained(detector_dir, local_files_only=True)
    count = sum(p.numel() for p in model.parameters())
    if count <= 0:
        raise RuntimeError("Grounding DINO checkpoint loaded without model parameters")
    return f"grounded-sam-2 smoke OK: detector {count} parameters; {sam_summary}"


def run(model_dir: str, image_path: str, out_path: str, prompt_json: str, device_name: str) -> dict:
    import torch

    if device_name not in {"cpu", "cuda"}:
        raise ValueError("device must be cpu or cuda")
    if device_name == "cuda" and not torch.cuda.is_available():
        raise RuntimeError("requested CUDA, but CUDA is unavailable")
    phrase = parse_phrase(prompt_json)
    detector_dir, segmenter_dir = layout(model_dir)
    device = torch.device(device_name)
    started = time.monotonic()
    image, _raster = _common.load_oriented_image(image_path, 16_000_000)
    width, height = image.size

    det_started = time.monotonic()
    processor, detector = _resident_detector(detector_dir, device)
    inputs = processor(images=image, text=detector_query(phrase), return_tensors="pt").to(device)
    with torch.inference_mode():
        outputs = detector(**inputs)
    found = processor.post_process_grounded_object_detection(
        outputs, inputs.input_ids, threshold=BOX_THRESHOLD, text_threshold=TEXT_THRESHOLD,
        target_sizes=[(height, width)],
    )[0]
    detect_ms = (time.monotonic() - det_started) * 1000
    chosen = best_box(found["boxes"].cpu().tolist(), found["scores"].cpu().tolist())
    if chosen is None:
        raise _common.InputError(f"nothing in the image matched {phrase!r}; try a different or more specific phrase")
    box, score = chosen

    sam_processor = sam2._processor()
    sam_inputs = sam_processor(images=image, input_boxes=[[box]], return_tensors="pt")
    model, cache_hit, model_load_ms = sam2._resident_model(segmenter_dir, device)
    sam_inputs = sam_inputs.to(device)
    with torch.inference_mode():
        outputs = model(**sam_inputs, multimask_output=True)
    scores = outputs.iou_scores[0, 0]
    selected = outputs.pred_masks[0, 0, int(torch.argmax(scores).item())].unsqueeze(0).unsqueeze(0)
    masks = sam_processor.post_process_masks(
        [selected], sam_inputs["original_sizes"], mask_threshold=0.0, binarize=True
    )
    mask = masks[0][0, 0].detach().cpu().numpy().astype("uint8") * 255
    sam2._write_mask(mask, out_path)

    metrics = {
        "device": device_name,
        "phrase": phrase,
        "detector_score": round(score, 4),
        "box_xyxy": [round(v, 1) for v in box],
        "mask_area_fraction": round(float((mask > 0).mean()), 6),
        "elapsed_ms": round((time.monotonic() - started) * 1000),
        "detect_ms": round(detect_ms),
        "model_load_ms": round(model_load_ms),
        "model_cache_hit": cache_hit,
        "max_rss_kib": sam2._max_rss_kib(),
        "cuda_peak_allocated_bytes": int(torch.cuda.max_memory_allocated(device)) if device.type == "cuda" else 0,
    }
    print(json.dumps({"grounded_segment_metrics": metrics}), file=sys.stderr, flush=True)
    return metrics


def main() -> dict | None:
    parser = argparse.ArgumentParser(description="Grounded SAM 2 text-prompted segmentation")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--prompts", default="{}")
    parser.add_argument("--device", choices=("cpu", "cuda"), default="cpu")
    args = parser.parse_args()
    try:
        return run(args.model, args.image, args.out, args.prompts, args.device)
    except Exception as exc:  # noqa: BLE001
        print(f"grounded segmentation failed: {exc}")
        raise SystemExit(_common.exit_status_for(exc)) from exc


if __name__ == "__main__":
    main()
