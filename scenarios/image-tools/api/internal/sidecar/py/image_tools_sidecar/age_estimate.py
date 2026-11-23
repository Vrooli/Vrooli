"""age_estimate — apparent-age bins for a face with the FairFace ViT head (ONNX).

The model is the age stage of the bundled minor-safety reviewer (``minor_review``) and
is also a standalone op: give it an image of ONE face (a tight crop) and it returns the
nine FairFace age bins as probabilities,

  {"labels": ["0-2", ...], "probs": {"0-2": 0.01, ...}, "p_under_20": 0.03}

``p_under_20`` is the mass on the bins strictly below 20. The 10-19 bin cannot separate
17 from 19, so the whole bin counts. The probabilities are averaged with the mirrored
crop (a cheap test-time augmentation that steadies borderline faces). This is an
appearance estimate of an age RANGE, not identity, and not proof of anyone's age. To
read faces inside a whole photo, use the reviewer (it detects faces first).
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from typing import Any, List, Sequence, Tuple

from . import _common

# FairFace bins strictly below 20 years old (names as the model's config.json spells them).
UNDER_20_LABELS = ("0-2", "3-9", "10-19")
INPUT_SIDE = 224


def under_age_mass(labels: Sequence[str], probs: Sequence[float], under: Sequence[str] = UNDER_20_LABELS) -> float:
    """Probability mass the age head puts on the ``under`` bins."""
    wanted = set(under)
    return float(sum(p for label, p in zip(labels, probs) if label in wanted))


def read_labels(model_dir: str) -> List[str]:
    """The age bin names in head order, from the model's config.json (``id2label``)."""
    with open(os.path.join(model_dir, "config.json"), encoding="utf-8") as fh:
        id2label = json.load(fh)["id2label"]
    labels = [str(id2label[str(i)]) for i in range(len(id2label))]
    missing = [label for label in UNDER_20_LABELS if label not in labels]
    if missing:
        raise ValueError(f"age model labels {labels} lack the expected bins {missing}")
    return labels


def load_head(model_dir: str) -> Tuple[Any, List[str]]:
    """(onnx session, labels) for an installed age model directory."""
    try:
        labels = read_labels(model_dir)
    except Exception as exc:  # noqa: BLE001 - name the directory, not a bare KeyError
        _common.fail(f"age_estimate: cannot read the age model labels in {model_dir!r}: {exc}", code=5)
        raise
    return _common.make_session(_common.resolve_onnx_path(model_dir)), labels


def crop_tensor(np, Image, crop):
    """One face crop as the model's CHW float tensor (224 px, mean/std 0.5)."""
    resized = crop.convert("RGB").resize((INPUT_SIDE, INPUT_SIDE), Image.Resampling.BILINEAR)
    arr = (np.asarray(resized, dtype=np.float32) / 255.0 - 0.5) / 0.5
    return arr.transpose(2, 0, 1)


def mirrored_probs(np, session, tensors: Sequence[Any]) -> List[List[float]]:
    """Softmax bin probabilities per tensor, each averaged with its mirror image."""
    if not tensors:
        return []
    batch = np.stack(list(tensors) + [t[:, :, ::-1].copy() for t in tensors]).astype(np.float32)
    logits = session.run(None, {session.get_inputs()[0].name: batch})[0]
    shifted = logits - logits.max(axis=-1, keepdims=True)
    probs = np.exp(shifted) / np.exp(shifted).sum(axis=-1, keepdims=True)
    half = len(tensors)
    return ((probs[:half] + probs[half:]) / 2.0).tolist()


def payload(labels: Sequence[str], probs: Sequence[float]) -> dict:
    return {
        "labels": list(labels),
        "probs": {label: round(float(p), 5) for label, p in zip(labels, probs)},
        "p_under_20": round(under_age_mass(labels, probs), 5),
    }


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="FairFace ViT apparent-age bins for one face crop")
    p.add_argument("--model", required=True, help="path to the installed age model directory (or its model.onnx)")
    p.add_argument("--image", required=True, help="input image: ONE face, cropped")
    p.add_argument("--out", help="optional JSON output path; stdout when omitted")
    return _common.parse_device_args(p)


def main() -> None:
    args = _parse_args()
    np, _ort, Image = _common.require_deps()
    model_dir = args.model if os.path.isdir(args.model) else os.path.dirname(os.path.abspath(args.model))
    session, labels = load_head(model_dir)
    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)
        raise
    probs = mirrored_probs(np, session, [crop_tensor(np, Image, image)])[0]
    data = json.dumps(payload(labels, probs), indent=2).encode("utf-8") + b"\n"
    if args.out:
        try:
            with open(args.out, "wb") as fh:
                fh.write(data)
        except Exception as exc:  # noqa: BLE001
            _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)
        return
    sys.stdout.buffer.write(data)


if __name__ == "__main__":
    main()
