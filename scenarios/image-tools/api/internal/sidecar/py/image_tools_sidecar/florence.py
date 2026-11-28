"""caption — describe an image with Florence-2 (native Transformers checkpoint).

One pass per image yields the three things the caption operation reports:

  {"backend": "florence-transformers", "model": "florence-2",
   "caption": "A woman sitting in a chair holding a cup of coffee and a book.",
   "detailed": "The image shows a ... (a paragraph)",
   "objects": [{"label": "book", "box": [x1, y1, x2, y2]}]}

``caption`` is the short caption, ``detailed`` the paragraph, ``objects`` the
labelled boxes in source pixels. Florence-2 is steered by task tokens, not free
text, so the caption operation's optional ``prompt`` is accepted and ignored.

The checkpoint is the community Transformers port (``Florence2ForConditionalGeneration``),
which loads with no remote code, so the sidecar never executes code from a model
repository. The model runs in fp16 on CUDA and fp32 on CPU.
"""

from __future__ import annotations

import argparse
import json
import os

from . import _common

TASK_CAPTION = "<CAPTION>"
TASK_DETAILED = "<MORE_DETAILED_CAPTION>"
TASK_OBJECTS = "<OD>"
MAX_NEW_TOKENS = 256
NUM_BEAMS = 3
MAX_PIXELS = 64_000_000

# One resident model, like sam2: a model or device switch replaces it.
_KEY = None
_BUNDLE = None


def _load(model_dir: str, device_name: str):
    global _KEY, _BUNDLE
    key = (model_dir, device_name)
    if key == _KEY and _BUNDLE is not None:
        return _BUNDLE
    import torch
    from transformers import AutoProcessor, Florence2ForConditionalGeneration

    _BUNDLE = None
    _KEY = None
    if device_name == "cuda":
        if not torch.cuda.is_available():
            raise RuntimeError("Florence-2 requested CUDA, but CUDA is unavailable")
        torch.cuda.empty_cache()
    dtype = torch.float16 if device_name == "cuda" else torch.float32
    processor = AutoProcessor.from_pretrained(model_dir, local_files_only=True)
    model = Florence2ForConditionalGeneration.from_pretrained(model_dir, local_files_only=True, dtype=dtype)
    model = model.to(torch.device(device_name)).eval()
    _BUNDLE, _KEY = (processor, model, dtype), key
    return _BUNDLE


def _generate(torch, bundle, device_name: str, image, task: str):
    processor, model, dtype = bundle
    inputs = processor(text=task, images=image, return_tensors="pt").to(torch.device(device_name), dtype)
    with torch.inference_mode():
        generated = model.generate(**inputs, max_new_tokens=MAX_NEW_TOKENS, num_beams=NUM_BEAMS, do_sample=False)
    text = processor.batch_decode(generated, skip_special_tokens=False)[0]
    return processor.post_process_generation(text, task=task, image_size=image.size)[task]


def describe(model_dir: str, image, device_name: str) -> dict:
    import torch

    bundle = _load(model_dir, device_name)
    caption = _generate(torch, bundle, device_name, image, TASK_CAPTION)
    detailed = _generate(torch, bundle, device_name, image, TASK_DETAILED)
    found = _generate(torch, bundle, device_name, image, TASK_OBJECTS)
    objects = [
        {"label": str(label), "box": [round(float(v), 1) for v in box]}
        for box, label in zip(found.get("bboxes", []), found.get("labels", []))
    ]
    return {
        "backend": "florence-transformers",
        "model": "florence-2",
        "caption": str(caption).strip(),
        "detailed": str(detailed).strip(),
        "objects": objects,
    }


def smoke(model_dir: str, deep: bool = False) -> str:
    """Load the pinned checkpoint and tokenizer locally; ``deep`` also describes a
    synthetic image end to end."""
    from PIL import Image

    bundle = _load(model_dir, "cpu")
    _processor, model, _dtype = bundle
    parameters = sum(p.numel() for p in model.parameters())
    if parameters <= 0:
        raise RuntimeError("Florence-2 checkpoint loaded without model parameters")
    summary = f"florence smoke OK: loaded {parameters} parameters from {model_dir}"
    if deep:
        result = describe(model_dir, Image.new("RGB", (128, 128), (96, 144, 192)), "cpu")
        if not result["caption"]:
            raise RuntimeError("Florence-2 produced an empty caption for the synthetic image")
        summary += "; synthetic caption: " + result["caption"]
    return summary


def main() -> None:
    parser = argparse.ArgumentParser(description="Florence-2 image description")
    parser.add_argument("--model", required=True, help="model directory (Transformers checkpoint)")
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    parser.add_argument("--prompt", default="", help="accepted and ignored: Florence-2 takes task tokens, not free text")
    args = parser.parse_args()
    os.environ["IMAGE_TOOLS_DEVICE"] = args.device
    if not os.path.isdir(args.model):
        _common.fail(f"Florence-2 model directory not found at {args.model!r}", code=4)
    try:
        image, _meta = _common.load_oriented_image(args.image, MAX_PIXELS)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)
        raise
    payload = describe(args.model, image, args.device)
    try:
        with open(args.out, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, ensure_ascii=False, indent=2)
            handle.write("\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
