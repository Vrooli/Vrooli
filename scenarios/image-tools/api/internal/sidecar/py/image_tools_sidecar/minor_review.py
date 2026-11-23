"""minor_review — the bundled minor-safety reviewer (the non-configurable boundary).

Implements the reviewer contract documented in docs/internal/SECURITY.md. One JSON
object arrives on stdin,

  {"images": [{"media_type": "image/png", "base64": "..."}], "strict": false, "assume_sexual": false}

and one JSON object leaves on stdout,

  {"minor": bool, "sexual": bool, "detail": "...", "images": [...per-image numbers...]}

Per image the reviewer runs three local models, all CPU-only so it never contends
with a generation for VRAM:

  1. two ViT NSFW classifiers answer "is this sexual": Falconsai (precise, no
     false positive on the measured clothed set) and AdamCodd (high recall on
     small-figure or ambiguous nudity). Either crossing its own calibrated bar
     counts as sexual;
  2. OpenCV-Zoo YuNet finds every face (reusing the decoder face_restore ships);
  3. a FairFace ViT age classifier (ONNX, nine age bins) reads each face crop (and its
     mirror) and answers "how old does this person look".

The face and age models only load when they are needed, so a clearly non-sexual
image costs one ONNX pass.

Decision (per image; the request is refused when ANY image refuses):

  * ``sexual``   - NSFW score >= SEXUAL_THRESHOLD (or STRICT_SEXUAL_THRESHOLD when the
    caller marks the source ``strict``: the producing model can make explicit images,
    and the classifier is weaker on synthetic ones, so it is asked a lower bar);
  * ``minor``    - some assessable face has P(age < 20) >= MINOR_THRESHOLD. The FairFace
    "10-19" bin includes adults aged 18-19 and the model cannot split it, so the whole
    bin counts as minor mass: the error is deliberately one-sided toward refusing;
  * unassessable - a sexual image with no face the age model can read (none found, or
    every face is smaller than MIN_FACE_PX) is refused as well. The reviewer will not
    wave through sexual content whose subject's age it could not establish.

``assume_sexual`` judges the age finding as if every image were sexual (a probe: it shows
what the boundary would do without any sexual content existing). Both flags can only make
the review stricter.

The reported top-level ``minor`` is the CO-OCCURRENCE verdict (a minor-appearing face in
a sexual image, or an unassessable sexual image), because the host gate refuses on
``minor and sexual``: a non-sexual portrait of a child is not refused, and a request
with one such image and one adult sexual image is not either. Per-image ``minor_face``
reports the age finding on its own.

Errors (undecodable image, missing weights, runtime failure) exit non-zero; the host
treats that as "review unavailable" and fails closed.
"""

from __future__ import annotations

import argparse
import base64
import io
import json
import os
import sys
from typing import Any, Dict, List, Optional, Sequence

from . import _common
from .age_estimate import UNDER_20_LABELS, crop_tensor, load_head, mirrored_probs, under_age_mass  # noqa: F401 - re-exported for callers/tests

# Thresholds were chosen on the FairFace validation split (10,954 labelled faces);
# docs/internal/SECURITY.md "Reviewer calibration" has the numbers and
# scripts/minor-review-eval.py reproduces them. All three are deliberately below the
# usual 0.5: a false refusal costs one more generation, a missed minor is the one
# outcome this project does not accept.
#
# Two classifiers run together. Falconsai is the precise one: on the 2026-11-23
# diverse SFW set (six clothed adults across ethnicities, the consenting subject and
# the pose chart) it scored every image at or below 0.13, while the old AdamCodd
# classifier scored three of them 0.42-0.65. AdamCodd is kept as a high-recall
# second signal because Falconsai missed small-figure fine-art nudity (two rated
# "Good" generations at 0.00-0.001, and a full-nudity edit at 0.001) that AdamCodd
# scored 0.97-0.99. The two thresholds are calibrated so neither source's false
# positives cross: AdamCodd's worst SFW score was 0.652 and its lowest true NSFW
# score in that set was 0.884.
SEXUAL_THRESHOLD = 0.30
STRICT_SEXUAL_THRESHOLD = 0.10
RECALL_SEXUAL_THRESHOLD = 0.80
STRICT_RECALL_SEXUAL_THRESHOLD = 0.70
MINOR_THRESHOLD = 0.20
MIN_FACE_PX = 40
FACE_SCORE_THRESHOLD = 0.5
# Crop side = longest face-box edge * this, centred: FairFace crops carry roughly
# this much context around the face box, which is what the age model was trained on.
FACE_CROP_MARGIN = 1.35
MAX_IMAGE_PIXELS = 128_000_000
MAX_FACES = 16

MODEL_NSFW = "falconsai-nsfw"
MODEL_NSFW_RECALL = "adamcodd-vit-nsfw"
MODEL_FACES = "yunet"
MODEL_AGE = "fairface-vit-age"
# The classifiers run on every image; the face and age models only on an image a
# classifier calls sexual, so the host acquires them for NSFW-capable producers up
# front and otherwise on demand (exit code 4 + ``missing-weights=`` names them).
CLASSIFIER_MODELS = (MODEL_NSFW, MODEL_NSFW_RECALL)
ESCALATION_MODELS = (MODEL_FACES, MODEL_AGE)
EXIT_MISSING_WEIGHTS = 4


def face_crop_box(box: Sequence[float], width: int, height: int, margin: float = FACE_CROP_MARGIN):
    """Square crop (x1, y1, x2, y2), centred on the face box and clamped to the raster.

    A clamped crop keeps the side the face allows rather than shifting, so a face at
    the border is not stretched into the model's square input.
    """
    x1, y1, x2, y2 = box
    cx, cy = (x1 + x2) / 2.0, (y1 + y2) / 2.0
    half = max(x2 - x1, y2 - y1) * margin / 2.0
    left, top = max(0, int(round(cx - half))), max(0, int(round(cy - half)))
    right, bottom = min(width, int(round(cx + half))), min(height, int(round(cy + half)))
    if right - left < 2 or bottom - top < 2:
        return None
    return left, top, right, bottom


def judge_image(
    nsfw_score: float,
    faces: Sequence[Dict[str, Any]],
    *,
    sexual: Optional[bool] = None,
    sexual_threshold: float = SEXUAL_THRESHOLD,
    minor_threshold: float = MINOR_THRESHOLD,
    min_face_px: int = MIN_FACE_PX,
) -> Dict[str, Any]:
    """Pure per-image decision from the model outputs (no torch, no IO).

    ``faces`` rows carry ``size_px`` (the shorter face-box edge) and, once the age
    model has run, ``p_under_20``. A row without ``p_under_20`` is unassessed.
    ``sexual`` may be supplied by an ensemble decision; otherwise the one score is
    compared with ``sexual_threshold``.
    """
    sexual = nsfw_score >= sexual_threshold if sexual is None else bool(sexual)
    assessed = [f for f in faces if f.get("size_px", 0) >= min_face_px and "p_under_20" in f]
    minor_face = any(f["p_under_20"] >= minor_threshold for f in assessed)
    unassessable = sexual and not assessed
    refuse = sexual and (minor_face or unassessable)
    return {
        "sexual": sexual,
        "minor_face": minor_face,
        "unassessable": unassessable,
        "refuse": refuse,
        "assessed_faces": len(assessed),
    }


def aggregate(per_image: Sequence[Dict[str, Any]]) -> Dict[str, Any]:
    """Fold per-image judgements into the contract object (see module docstring)."""
    refused = [i for i, img in enumerate(per_image) if img["refuse"]]
    sexual = any(img["sexual"] for img in per_image)
    notes: List[str] = []
    for i in refused:
        img = per_image[i]
        if img["minor_face"]:
            notes.append(
                f"image {i}: sexual (nsfw {img['nsfw_score']:.2f}) with a minor-appearing face "
                f"(max p(age<20) {img['max_p_under_20']:.2f})"
            )
        else:
            notes.append(
                f"image {i}: sexual (nsfw {img['nsfw_score']:.2f}) and the subject's age could not be "
                f"established ({img['faces_found']} face(s) found, {img['assessed_faces']} readable)"
            )
    if not notes:
        worst = max((img["max_p_under_20"] for img in per_image), default=0.0)
        notes.append(
            f"passed: {len(per_image)} image(s), max nsfw "
            f"{max((img['nsfw_score'] for img in per_image), default=0.0):.2f} "
            f"(recall {max((img.get('nsfw_score_recall', 0.0) for img in per_image), default=0.0):.2f}), "
            f"max p(age<20) {worst:.2f}"
        )
    return {
        "minor": bool(refused),
        "sexual": bool(sexual) or bool(refused),
        "detail": "; ".join(notes),
    }


class _Models:
    """Lazy model holder: the face and age models load only when an image needs them."""

    def __init__(self, models_root: str):
        self.root = models_root
        self._nsfw = None
        self._nsfw_recall = None
        self._detector = None
        self._age = None

    def path(self, model_id: str, *parts: str) -> str:
        return os.path.join(self.root, "models", model_id, *parts)

    def require(self, model_ids: Sequence[str]) -> None:
        """Exit non-zero naming every model directory that is not installed.

        The host reads ``missing-weights=<ids>`` from stderr, acquires exactly those
        weights through managed acquisition and retries the review once.
        """
        missing = [m for m in model_ids if not os.path.isdir(self.path(m))]
        if missing:
            _common.fail(
                f"minor_review: model weights are not installed: missing-weights={','.join(missing)}",
                code=EXIT_MISSING_WEIGHTS,
            )

    def nsfw(self):
        if self._nsfw is None:
            session = _common.make_session(_common.resolve_onnx_path(self.path(MODEL_NSFW)))
            self._nsfw = (session, _common.model_input_hw(session, default_hw=(224, 224)))
        return self._nsfw

    def nsfw_recall(self):
        if self._nsfw_recall is None:
            session = _common.make_session(_common.resolve_onnx_path(self.path(MODEL_NSFW_RECALL)))
            self._nsfw_recall = (session, _common.model_input_hw(session, default_hw=(384, 384)))
        return self._nsfw_recall

    def detector(self):
        if self._detector is None:
            self._detector = _common.make_session(_common.resolve_onnx_path(self.path(MODEL_FACES)))
        return self._detector

    def age(self):
        """(session, labels): the ONNX age head and its bin names, read from config.json."""
        if self._age is None:
            self._age = load_head(self.path(MODEL_AGE))
        return self._age


def _classifier_score(np, Image, model, image) -> float:
    from . import nsfw

    session, size_hw = model
    inp = nsfw._preprocess(np, image, size_hw, resample=Image.Resampling.BILINEAR)
    out = session.run(None, {session.get_inputs()[0].name: inp})
    return float(nsfw._payload(np, out[0])["score"])


def _nsfw_score(np, Image, models: _Models, image) -> float:
    """The precise classifier's score (Falconsai)."""
    return _classifier_score(np, Image, models.nsfw(), image)


def _nsfw_scores(np, Image, models: _Models, image) -> "tuple[float, float]":
    """(precise, high-recall) classifier scores for one image."""
    return _nsfw_score(np, Image, models, image), _classifier_score(np, Image, models.nsfw_recall(), image)


def _age_probs(np, Image, models: _Models, image, boxes) -> List[List[float]]:
    """Age-bin probabilities for each face box (mirror-averaged); [] for a degenerate box."""
    session, _labels = models.age()
    width, height = image.size
    tensors = []
    for box in boxes:
        crop_box = face_crop_box(box, width, height)
        tensors.append(None if crop_box is None else crop_tensor(np, Image, image.crop(crop_box)))
    live = [t for t in tensors if t is not None]
    scored = iter(mirrored_probs(np, session, live))
    return [next(scored) if t is not None else [] for t in tensors]


def sexual_threshold(strict: bool, assume_sexual: bool) -> float:
    """The NSFW score at which an image counts as sexual. Flags only lower it."""
    if assume_sexual:
        return 0.0
    return STRICT_SEXUAL_THRESHOLD if strict else SEXUAL_THRESHOLD


def recall_sexual_threshold(strict: bool, assume_sexual: bool) -> float:
    """The high-recall classifier's bar. A probe treats every image as sexual."""
    if assume_sexual:
        return 0.0
    return STRICT_RECALL_SEXUAL_THRESHOLD if strict else RECALL_SEXUAL_THRESHOLD


def sexual_decision(primary: float, recall: float, *, strict: bool, assume_sexual: bool = False) -> bool:
    """Either classifier crossing its own calibrated bar counts as sexual.

    Neither classifier is trusted alone: one misses clothed skin, the other misses
    small-figure nudity, and on the measured sets their false-positive ranges do
    not overlap when each uses its own threshold.
    """
    return primary >= sexual_threshold(strict, assume_sexual) or recall >= recall_sexual_threshold(strict, assume_sexual)


def review_image(np, Image, models: _Models, image, *, strict: bool, assume_sexual: bool = False) -> Dict[str, Any]:
    nsfw_score, nsfw_recall = _nsfw_scores(np, Image, models, image)
    sexual = sexual_decision(nsfw_score, nsfw_recall, strict=strict, assume_sexual=assume_sexual)
    faces: List[Dict[str, Any]] = []
    if sexual:
        from . import face_restore

        models.require(ESCALATION_MODELS)

        found = face_restore.detect_faces(np, Image, models.detector(), image, FACE_SCORE_THRESHOLD)
        found = sorted(found, key=lambda f: (f["box"][2] - f["box"][0]) * (f["box"][3] - f["box"][1]), reverse=True)[:MAX_FACES]
        readable = [f for f in found if min(f["box"][2] - f["box"][0], f["box"][3] - f["box"][1]) >= MIN_FACE_PX]
        probs = _age_probs(np, Image, models, image, [f["box"] for f in readable]) if readable else []
        labels = models.age()[1] if readable else []
        by_box = {id(f): p for f, p in zip(readable, probs)}
        for f in found:
            x1, y1, x2, y2 = f["box"]
            row: Dict[str, Any] = {
                "box": [round(x1, 1), round(y1, 1), round(x2, 1), round(y2, 1)],
                "detector_score": round(float(f["score"]), 3),
                "size_px": round(min(x2 - x1, y2 - y1), 1),
            }
            p = by_box.get(id(f))
            if p:
                row["p_under_20"] = round(under_age_mass(labels, p), 4)
                row["age_probs"] = {label: round(v, 4) for label, v in zip(labels, p)}
            faces.append(row)
    judged = judge_image(nsfw_score, faces, sexual=sexual)
    return {
        "nsfw_score": round(nsfw_score, 4),
        "nsfw_score_recall": round(nsfw_recall, 4),
        "faces_found": len(faces),
        "assessed_faces": judged["assessed_faces"],
        "max_p_under_20": round(max((f.get("p_under_20", 0.0) for f in faces), default=0.0), 4),
        "sexual": judged["sexual"],
        "minor_face": judged["minor_face"],
        "unassessable": judged["unassessable"],
        "refuse": judged["refuse"],
        "faces": faces,
    }


def run(request: Dict[str, Any], models_root: str) -> Dict[str, Any]:
    np, _ort, Image = _common.require_deps()
    images = request.get("images")
    if not isinstance(images, list) or not images:
        _common.fail("minor_review: request carries no images", code=2)
    strict = bool(request.get("strict", False))
    assume_sexual = bool(request.get("assume_sexual", False))
    models = _Models(models_root)
    per_image = []
    for index, item in enumerate(images):
        try:
            raw = base64.b64decode(item["base64"], validate=True)
            from PIL import ImageOps

            image = Image.open(io.BytesIO(raw))
            if image.width * image.height > MAX_IMAGE_PIXELS:
                raise ValueError(f"image exceeds {MAX_IMAGE_PIXELS} pixels")
            image = ImageOps.exif_transpose(image).convert("RGB")
        except Exception as exc:  # noqa: BLE001
            _common.fail(f"minor_review: image {index} is not a decodable raster: {exc}", code=6)
            raise
        per_image.append(review_image(np, Image, models, image, strict=strict, assume_sexual=assume_sexual))
    verdict = aggregate(per_image)
    verdict["images"] = per_image
    verdict["thresholds"] = {
        "sexual": sexual_threshold(strict, assume_sexual),
        "sexual_recall": recall_sexual_threshold(strict, assume_sexual),
        "minor": MINOR_THRESHOLD,
        "min_face_px": MIN_FACE_PX,
        "strict": strict,
        "assume_sexual": assume_sexual,
    }
    return verdict


def main(argv: Optional[Sequence[str]] = None) -> None:
    p = argparse.ArgumentParser(description="bundled minor-safety reviewer (stdin JSON -> stdout JSON)")
    p.add_argument("--models-root", required=True, help="image-tools data root that holds models/<id>/")
    args = _common.parse_device_args(p) if argv is None else p.parse_args(argv)
    try:
        request = json.load(sys.stdin)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"minor_review: stdin is not a JSON object: {exc}", code=2)
        raise
    _Models(args.models_root).require(CLASSIFIER_MODELS)
    sys.stdout.write(json.dumps(run(request, args.models_root)) + "\n")


if __name__ == "__main__":
    main()
