"""Semantic parser segmentation: a phrase or a point becomes the pixels of a label.

A SegFormer human/clothes parser (``segformer-b2-clothes``, ``segformer-b3-fashion``)
labels every pixel with one of a closed set of classes (hair, upper-clothes,
jacket, pants, ...). Unlike a box-prompted model it never confuses two people or
a group with a part: "hair" is exactly the hair pixels, in one forward pass.
What it cannot do is select a class it was not trained on (a book, a cup); that
is the grounded model's job.

Prompt contract (``--prompts`` JSON), the same shape the other segment adapters
take: ``{"text": "<phrase>"}`` selects the labels the phrase names, and
``{"points": [{"x":..,"y":..}]}`` selects the labels under the positive points.

The phrase-to-label step is pure (``resolve_labels``) so it is unit-tested
without weights. Concepts name what a person says; each lists label keywords in
preference order and the first keyword the *model's own label set* has wins, so
the same word "jacket" selects ``Jacket`` on the fashion parser and falls back to
``Upper-clothes`` on the clothes parser, which has no separate jacket class.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
import time

from . import _common, sam2

MAX_PHRASE_CHARS = 200
# Specks smaller than this fraction of the picture are parser noise, not a
# region a person asked for; holes this small inside a region are filled.
SPECK_FRACTION = 0.0008

STOPWORDS = frozenset({
    "the", "a", "an", "her", "his", "their", "my", "its", "our", "your", "of", "on", "in", "entire", "whole", "all",
    "she", "is", "wearing", "worn",
})

# What a person says -> label keywords to try, in preference order. A keyword
# matches a label synonym exactly (or as the left/right form: "shoe" ~ "Left-shoe"),
# so "hair" never matches "hair accessory".
_UPPER = ("upper-clothes", "top", "shirt")
CONCEPTS: dict[str, tuple[tuple[str, ...], ...]] = {
    "hair": (("hair",),),
    "face": (("face",),),
    "skin": (("face",), ("arm",), ("leg",)),
    "arm": (("arm",),),
    "arms": (("arm",),),
    "hand": (("arm",),),
    "hands": (("arm",),),
    "leg": (("leg",),),
    "legs": (("leg",),),
    "hat": (("hat",),),
    "cap": (("hat",),),
    "glasses": (("glasses", "sunglasses"),),
    "sunglasses": (("sunglasses", "glasses"),),
    "shoe": (("shoe",),),
    "shoes": (("shoe",),),
    "sneakers": (("shoe",),),
    "boots": (("shoe",),),
    "bag": (("bag",),),
    "purse": (("bag",),),
    "scarf": (("scarf",),),
    "belt": (("belt",),),
    "dress": (("dress",),),
    "skirt": (("skirt",),),
    "pants": (("pants",),),
    "jeans": (("pants",),),
    "trousers": (("pants",),),
    "shorts": (("shorts", "pants"),),
    "jacket": (("jacket", "coat") + _UPPER,),
    "blazer": (("jacket", "coat") + _UPPER,),
    "coat": (("coat", "jacket") + _UPPER,),
    "top": (("top", "upper-clothes", "shirt"),),
    "tshirt": (("top", "upper-clothes", "shirt"),),
    "shirt": (("shirt", "top", "upper-clothes"),),
    "blouse": (("shirt", "top", "upper-clothes"),),
    "sweater": (("sweater", "upper-clothes", "top"),),
    "sweatshirt": (("top", "sweater", "upper-clothes"),),
    "cardigan": (("cardigan", "upper-clothes", "top"),),
    "vest": (("vest", "upper-clothes"),),
    "tie": (("tie",),),
}
# Group words select every garment label the model has.
GARMENTS = (
    "upper-clothes", "shirt", "top", "sweater", "cardigan", "jacket", "vest", "pants", "shorts", "skirt",
    "coat", "dress", "jumpsuit", "cape", "scarf", "belt", "tie", "tights", "hood",
)
GROUPS = {
    "outfit": GARMENTS, "clothes": GARMENTS, "clothing": GARMENTS, "garment": GARMENTS, "garments": GARMENTS,
    "attire": GARMENTS, "apparel": GARMENTS,
}

_SIDE = re.compile(r"^(left|right)-")


def synonyms(label: str) -> set[str]:
    """The names one label answers to: 'top, t-shirt, sweatshirt' -> three, and
    'Left-shoe' also answers to 'shoe'."""
    names = set()
    for part in label.lower().split(","):
        part = part.strip()
        if not part:
            continue
        names.add(part)
        names.add(_SIDE.sub("", part))
    return names


def normalise_phrase(phrase: str) -> list[str]:
    words = re.findall(r"[a-z][a-z-]*", phrase.lower().replace("t-shirt", "tshirt"))
    return [w for w in words if w not in STOPWORDS]


def parse_prompt(raw: str):
    """-> ("text", phrase) or ("points", [(x, y), ...]) from the prompt JSON."""
    prompt = json.loads(raw or "{}")
    if not isinstance(prompt, dict):
        raise _common.InputError("segmentation prompt must be an object")
    if "text" in prompt:
        if set(prompt) != {"text"}:
            raise _common.InputError("a text phrase cannot be combined with points or a box")
        text = prompt["text"]
        if not isinstance(text, str) or not text.strip():
            raise _common.InputError("segmentation requires a non-empty text phrase")
        phrase = " ".join(text.split())
        if len(phrase) > MAX_PHRASE_CHARS:
            raise _common.InputError(f"text phrase must be at most {MAX_PHRASE_CHARS} characters")
        return "text", phrase
    if "box" in prompt:
        raise _common.InputError("a semantic parser selects by phrase or point, not by box")
    points = [
        (float(p["x"]), float(p["y"]))
        for p in prompt.get("points", [])
        if isinstance(p, dict) and not p.get("negative", False)
    ]
    if not points:
        raise _common.InputError("segmentation requires a text phrase or a positive point")
    return "points", points


def resolve_labels(phrase: str, id2label: dict[int, str]) -> list[int]:
    """Label ids a phrase names on this parser, or ValueError saying what it knows."""
    by_name: dict[str, set[int]] = {}
    for label_id, label in id2label.items():
        for name in synonyms(label):
            by_name.setdefault(name, set()).add(label_id)

    def ids_for(keywords) -> set[int]:
        out: set[int] = set()
        for keyword in keywords:
            out |= by_name.get(keyword, set())
        return out

    chosen: set[int] = set()
    words = normalise_phrase(phrase)
    for word in words:
        singular = word[:-1] if word.endswith("s") and word[:-1] in CONCEPTS else word
        if word in GROUPS:
            chosen |= ids_for(GROUPS[word])
        elif singular in CONCEPTS:
            for keywords in CONCEPTS[singular]:
                # preference order: the first keyword this parser has wins
                for keyword in keywords:
                    found = ids_for((keyword,))
                    if found:
                        chosen |= found
                        break
        else:
            chosen |= ids_for((word, singular))  # a label's own name ("scarf", "bow")
    if not chosen:
        known = sorted({n for label in id2label.values() for n in synonyms(label) if not _SIDE.match(n)})
        raise _common.InputError(
            f"this parser has no label for {phrase!r}; it can select: {', '.join(known)}. "
            "Use a grounded model for anything else."
        )
    return sorted(chosen)


def labels_under_points(label_map, points) -> list[int]:
    """The distinct parser labels under normalised (x, y) points; background (0) is skipped."""
    height, width = label_map.shape
    ids = set()
    for x, y in points:
        px = min(width - 1, max(0, int(x * width)))
        py = min(height - 1, max(0, int(y * height)))
        label = int(label_map[py, px])
        if label:
            ids.add(label)
    if not ids:
        raise _common.InputError("the point fell on background; click a person, garment or hair")
    return sorted(ids)


def clean_mask(mask, speck_fraction: float = SPECK_FRACTION):
    """Drop specks and fill pinholes smaller than ``speck_fraction`` of the picture."""
    import numpy as np
    from scipy import ndimage

    limit = max(1, int(mask.size * speck_fraction))
    labelled, count = ndimage.label(mask)
    if count:
        sizes = ndimage.sum(mask, labelled, index=range(1, count + 1))
        keep = np.zeros(count + 1, dtype=bool)
        keep[1:] = np.asarray(sizes) >= limit
        mask = keep[labelled]
    holes, hole_count = ndimage.label(~mask)
    if hole_count:
        sizes = ndimage.sum(~mask, holes, index=range(1, hole_count + 1))
        fill = np.zeros(hole_count + 1, dtype=bool)
        fill[1:] = np.asarray(sizes) < limit
        mask = mask | fill[holes]
    return mask


# One resident parser, like sam2's single resident segmenter: a model switch
# replaces it instead of accumulating VRAM.
_RESIDENT_KEY = None
_RESIDENT = None


def _resident_parser(model_dir: str, device):
    global _RESIDENT_KEY, _RESIDENT
    key = (model_dir, str(device))
    if key == _RESIDENT_KEY and _RESIDENT is not None:
        return _RESIDENT, True, 0.0
    import torch
    from transformers import SegformerForSemanticSegmentation, SegformerImageProcessor

    _RESIDENT, _RESIDENT_KEY = None, None
    if device.type == "cuda":
        torch.cuda.empty_cache()
    started = time.monotonic()
    processor = SegformerImageProcessor.from_pretrained(model_dir, local_files_only=True)
    model = SegformerForSemanticSegmentation.from_pretrained(model_dir, local_files_only=True).to(device).eval()
    _RESIDENT, _RESIDENT_KEY = (processor, model), key
    return _RESIDENT, False, (time.monotonic() - started) * 1000


def smoke(model_dir: str, deep: bool = False) -> str:
    from transformers import SegformerForSemanticSegmentation

    model = SegformerForSemanticSegmentation.from_pretrained(model_dir, local_files_only=True)
    count = sum(p.numel() for p in model.parameters())
    if count <= 0 or not model.config.id2label:
        raise RuntimeError("SegFormer checkpoint loaded without parameters or labels")
    summary = f"segformer smoke OK: {count} parameters, {len(model.config.id2label)} labels from {model_dir}"
    if deep:
        import torch
        from PIL import Image
        from transformers import SegformerImageProcessor

        processor = SegformerImageProcessor.from_pretrained(model_dir, local_files_only=True)
        inputs = processor(images=Image.new("RGB", (96, 96), (96, 144, 192)), return_tensors="pt")
        with torch.inference_mode():
            logits = model(**inputs).logits
        if not torch.isfinite(logits).all():
            raise RuntimeError("SegFormer forward pass returned non-finite logits")
        summary += "; synthetic 96x96 forward passed"
    return summary


def run(model_dir: str, image_path: str, out_path: str, prompt_json: str, device_name: str) -> dict:
    import numpy as np
    import torch

    if device_name not in {"cpu", "cuda"}:
        raise ValueError("device must be cpu or cuda")
    if device_name == "cuda" and not torch.cuda.is_available():
        raise RuntimeError("requested CUDA, but CUDA is unavailable")
    kind, prompt = parse_prompt(prompt_json)
    device = torch.device(device_name)
    started = time.monotonic()
    image, _raster = _common.load_oriented_image(image_path, 16_000_000)
    width, height = image.size

    (processor, model), cache_hit, load_ms = _resident_parser(model_dir, device)
    id2label = {int(k): v for k, v in model.config.id2label.items()}
    if device.type == "cuda":
        torch.cuda.reset_peak_memory_stats(device)
    forward_started = time.monotonic()
    inputs = processor(images=image, return_tensors="pt").to(device)
    with torch.inference_mode():
        logits = model(**inputs).logits  # 1 x labels x H/4 x W/4
        upsampled = torch.nn.functional.interpolate(logits, size=(height, width), mode="bilinear", align_corners=False)
        label_map = upsampled.argmax(dim=1)[0].cpu().numpy()
    forward_ms = (time.monotonic() - forward_started) * 1000

    if kind == "text":
        ids = resolve_labels(prompt, id2label)
    else:
        ids = labels_under_points(label_map, prompt)
    mask = clean_mask(np.isin(label_map, ids))
    if not mask.any():
        raise ValueError(f"nothing in the image matched {prompt if kind == 'text' else 'the point'}")
    sam2._write_mask(mask.astype("uint8") * 255, out_path)

    metrics = {
        "device": device_name,
        "labels": ",".join(id2label[i] for i in ids),
        "mask_area_fraction": round(float(mask.mean()), 6),
        "elapsed_ms": round((time.monotonic() - started) * 1000),
        "forward_ms": round(forward_ms),
        "model_load_ms": round(load_ms),
        "model_cache_hit": cache_hit,
        "max_rss_kib": sam2._max_rss_kib(),
        "cuda_peak_allocated_bytes": int(torch.cuda.max_memory_allocated(device)) if device.type == "cuda" else 0,
    }
    print(json.dumps({"semantic_segment_metrics": metrics}), file=sys.stderr, flush=True)
    return metrics


def main() -> dict | None:
    parser = argparse.ArgumentParser(description="SegFormer semantic parser segmentation")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--prompts", default="{}")
    parser.add_argument("--device", choices=("cpu", "cuda"), default="cpu")
    args = parser.parse_args()
    try:
        return run(args.model, args.image, args.out, args.prompts, args.device)
    except Exception as exc:  # noqa: BLE001
        print(f"semantic segmentation failed: {exc}")
        raise SystemExit(_common.exit_status_for(exc)) from exc


if __name__ == "__main__":
    main()
