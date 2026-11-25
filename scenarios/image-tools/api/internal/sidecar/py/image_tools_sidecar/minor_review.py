"""minor_review — the bundled minor-safety reviewer (the non-configurable boundary).

Implements the reviewer contract documented in docs/internal/SECURITY.md. One JSON
object arrives on stdin,

  {"images": [{"media_type": "image/png", "base64": "..."}],
   "profile": { ...declarative review profile... },
   "strict": false, "assume_sexual": false}

and one JSON object leaves on stdout,

  {"action": "allow|flag|refuse", "reason_code": "clean|sexual_adult|...",
   "minor": bool, "sexual": bool, "detail": "...", "images": [...],
   "thresholds": {...}, "models": [...], "reviewer_version": "..."}

The profile is the single source of thresholds and model ids; the shipped
profiles.json is the default. Per image the reviewer runs:

  1. the configured sexual-content classifiers (two in the shipped profiles:
     Falconsai, precise, and AdamCodd, high recall). Either crossing its own
     calibrated bar counts as sexual;
  2. OpenCV-Zoo YuNet finds every face (reusing the decoder face_restore ships);
  3. a FairFace ViT age classifier (ONNX, nine age bins) reads each face crop (and its
     mirror) and answers "how old does this person look".

The face and age models only load when they are needed, so a clearly non-sexual
image costs one ONNX pass.

Decision (per image; the request is refused when ANY image refuses):

  * ``sexual``   - any classifier score >= its threshold (or strict threshold when the
    caller marks the source ``strict``: the producing model can make explicit images);
  * ``minor_face`` - some assessable face has P(age < 20) >= the profile's age
    threshold. The FairFace "10-19" bin includes adults aged 18-19 and the model
    cannot split it, so the whole bin counts as minor mass: the error is deliberately
    one-sided toward refusing;
  * unassessable - a sexual image with no face the age model can read (none found, or
    every face is smaller than the profile's minimum) is refused unless the profile's
    ``on_unassessable`` is ``second_opinion``; the host then asks a face-independent
    judge, which may clear only a confident adult reading.

The reported top-level ``minor`` is the CO-OCCURRENCE verdict (a minor-appearing face in
a sexual image, or an unassessable sexual image), because the host gate refuses on
``action == refuse``: a non-sexual portrait of a child is not refused, and a request
with one such image and one adult sexual image is not either. Per-image findings carry
their own action/reason.

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

# Shipped defaults, kept for the pure helpers and for a request that carries no
# profile (an operator-supplied reviewer may omit it). profiles.json is the
# authoritative source; the Go host sends it on every bundled review.
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
CLASSIFIER_MODELS = (MODEL_NSFW, MODEL_NSFW_RECALL)
ESCALATION_MODELS = (MODEL_FACES, MODEL_AGE)
EXIT_MISSING_WEIGHTS = 4

REASON_CLEAN = "clean"
REASON_SEXUAL_ADULT = "sexual_adult"
REASON_SEXUAL_MINOR_FACE = "sexual_minor_face"
REASON_SEXUAL_UNASSESSABLE = "sexual_unassessable"

ACTION_ALLOW = "allow"
ACTION_FLAG = "flag"
ACTION_REFUSE = "refuse"


class ReviewProfile:
    """The declarative reviewer configuration sent by the host.

    Missing fields fall back to the shipped values, which keeps the contract
    compatible with an operator-supplied reviewer that sends only ``images``.
    """

    def __init__(self, data: Optional[Dict[str, Any]] = None):
        data = dict(data or {})
        self.schema_version = int(data.get("schema_version", 1) or 1)
        self.name = str(data.get("name") or "default")
        self.mode = str(data.get("mode") or "enforce")
        self.on_refusal = str(data.get("on_refusal") or "fail_job")
        self.classifiers = [_classifier(d, i) for i, d in enumerate(data.get("classifiers") or _default_classifiers())]
        self.aggregate = str(data.get("aggregate") or "any")
        face = dict(data.get("face") or {})
        self.face_id = str(face.get("id") or MODEL_FACES)
        self.face_score_threshold = float(face.get("score_threshold", FACE_SCORE_THRESHOLD))
        self.min_face_px = int(face.get("min_px", MIN_FACE_PX))
        self.crop_margin = float(face.get("crop_margin", FACE_CROP_MARGIN))
        self.max_faces = int(face.get("max_faces", MAX_FACES))
        age = dict(data.get("age") or {})
        self.age_id = str(age.get("id") or MODEL_AGE)
        self.age_threshold = float(age.get("threshold", MINOR_THRESHOLD))
        second = dict(data.get("second_opinion") or {})
        self.second_opinion_enabled = bool(second.get("enabled", True))
        self.second_opinion_min_age = int(second.get("min_age", 25))
        self.second_opinion_min_confidence = str(second.get("min_confidence") or "medium")
        self.second_opinion_judge_role = str(second.get("judge_role") or "vision.default")
        self.second_opinion_models = list(second.get("models") or ["qwen3-vl:2b", "llava:7b"])
        self.second_opinion_ollama_url = str(second.get("ollama_url") or "http://127.0.0.1:11434")
        self.on_minor_face = str(data.get("on_minor_face") or "refuse")
        self.on_unassessable = str(data.get("on_unassessable") or "second_opinion")
        self.max_image_pixels = int(data.get("max_image_pixels", MAX_IMAGE_PIXELS))
        self.reviewer_version = str(data.get("reviewer_version") or "unversioned")
        self.validate()

    @property
    def classifier_ids(self) -> List[str]:
        return [c["id"] for c in self.classifiers]

    @property
    def escalation_ids(self) -> List[str]:
        out: List[str] = []
        for model in (self.face_id, self.age_id):
            if model and model not in out:
                out.append(model)
        return out

    @property
    def model_ids(self) -> List[str]:
        return list(dict.fromkeys(self.classifier_ids + self.escalation_ids))

    def threshold_for(self, classifier: Dict[str, Any], strict: bool) -> float:
        if strict and classifier.get("strict_threshold") is not None:
            return float(classifier["strict_threshold"])
        return float(classifier["threshold"])

    def thresholds(self, strict: bool) -> Dict[str, float]:
        out = {c["id"]: self.threshold_for(c, strict) for c in self.classifiers}
        out["age"] = self.age_threshold
        out["face.min_px"] = float(self.min_face_px)
        out["face.score"] = self.face_score_threshold
        return out

    def decide(self, scores: Dict[str, float], strict: bool, assume_sexual: bool = False) -> bool:
        if assume_sexual:
            return True
        for c in self.classifiers:
            if float(scores.get(c["id"], 0.0)) >= self.threshold_for(c, strict):
                return True
        return False

    def validate(self) -> None:
        if self.schema_version != 1:
            raise ValueError(f"profile schema_version = {self.schema_version}, want 1")
        if not self.classifiers:
            raise ValueError("profile has no classifiers")
        if len({c["id"] for c in self.classifiers}) != len(self.classifiers):
            raise ValueError("profile lists a classifier twice")
        for c in self.classifiers:
            if not c["id"]:
                raise ValueError("classifier id is required")
            for field in ("threshold", "strict_threshold"):
                value = c.get(field)
                if value is not None and not 0 <= float(value) <= 1:
                    raise ValueError(f"classifier {c['id']} {field}={value}, want 0..1")
            if c.get("strict_threshold") is not None and float(c["strict_threshold"]) > float(c["threshold"]):
                raise ValueError(f"classifier {c['id']} strict threshold is above its threshold")
        if self.aggregate != "any":
            raise ValueError(f"aggregate={self.aggregate!r}, want any")
        if not self.face_id or not self.age_id:
            raise ValueError("face.id and age.id are required")
        if not 0 <= self.face_score_threshold <= 1 or not 0 <= self.age_threshold <= 1:
            raise ValueError("face/age thresholds must be 0..1")
        if self.min_face_px <= 0 or self.crop_margin < 1 or self.max_faces <= 0:
            raise ValueError("face min_px/max_faces must be positive and crop_margin >= 1")
        if self.on_minor_face != "refuse":
            raise ValueError("on_minor_face must be refuse")
        if self.on_unassessable not in ("refuse", "second_opinion"):
            raise ValueError("on_unassessable must be refuse or second_opinion")
        if self.on_unassessable == "second_opinion" and not self.second_opinion_enabled:
            raise ValueError("on_unassessable=second_opinion but second_opinion.enabled is false")
        if self.second_opinion_enabled and self.second_opinion_min_confidence not in ("medium", "high"):
            raise ValueError("second_opinion.min_confidence must be medium or high (low never clears)")
        if self.max_image_pixels <= 0:
            raise ValueError("max_image_pixels must be positive")

    def as_dict(self) -> Dict[str, Any]:
        return {
            "schema_version": self.schema_version,
            "name": self.name,
            "mode": self.mode,
            "on_refusal": self.on_refusal,
            "classifiers": self.classifiers,
            "aggregate": self.aggregate,
            "face": {
                "id": self.face_id, "score_threshold": self.face_score_threshold,
                "min_px": self.min_face_px, "crop_margin": self.crop_margin, "max_faces": self.max_faces,
            },
            "age": {"id": self.age_id, "threshold": self.age_threshold},
            "second_opinion": {
                "enabled": self.second_opinion_enabled, "min_age": self.second_opinion_min_age,
                "min_confidence": self.second_opinion_min_confidence,
                "judge_role": self.second_opinion_judge_role,
                "models": self.second_opinion_models,
                "ollama_url": self.second_opinion_ollama_url,
            },
            "on_minor_face": self.on_minor_face,
            "on_unassessable": self.on_unassessable,
            "max_image_pixels": self.max_image_pixels,
            "reviewer_version": self.reviewer_version,
        }


def _default_classifiers() -> List[Dict[str, Any]]:
    return [
        {"id": MODEL_NSFW, "threshold": SEXUAL_THRESHOLD, "strict_threshold": STRICT_SEXUAL_THRESHOLD, "input_hw": [224, 224]},
        {"id": MODEL_NSFW_RECALL, "threshold": RECALL_SEXUAL_THRESHOLD, "strict_threshold": STRICT_RECALL_SEXUAL_THRESHOLD, "input_hw": [384, 384]},
    ]


def _classifier(data: Dict[str, Any], index: int) -> Dict[str, Any]:
    item = dict(data or {})
    item["id"] = str(item.get("id") or (CLASSIFIER_MODELS[index] if index < len(CLASSIFIER_MODELS) else f"classifier-{index}"))
    item["threshold"] = float(item.get("threshold", SEXUAL_THRESHOLD))
    if item.get("strict_threshold") is not None:
        item["strict_threshold"] = float(item["strict_threshold"])
    if item.get("input_hw") is not None:
        hw = list(item["input_hw"])
        if len(hw) != 2:
            raise ValueError(f"classifier {item['id']} input_hw must be two integers")
        item["input_hw"] = [int(hw[0]), int(hw[1])]
    return item


def default_profile() -> ReviewProfile:
    return ReviewProfile({})


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
    if not sexual:
        action, reason = ACTION_ALLOW, REASON_CLEAN
    elif minor_face:
        action, reason = ACTION_REFUSE, REASON_SEXUAL_MINOR_FACE
    elif unassessable:
        action, reason = ACTION_REFUSE, REASON_SEXUAL_UNASSESSABLE
    else:
        action, reason = ACTION_FLAG, REASON_SEXUAL_ADULT
    return {
        "sexual": sexual,
        "minor_face": minor_face,
        "unassessable": unassessable,
        "refuse": refuse,
        "assessed_faces": len(assessed),
        "action": action,
        "reason_code": reason,
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
    if refused:
        action = ACTION_REFUSE
        reason_codes = list(dict.fromkeys(
            img.get("reason_code") or (REASON_SEXUAL_MINOR_FACE if img.get("minor_face") else REASON_SEXUAL_UNASSESSABLE)
            for img in per_image if img["refuse"]
        ))
    elif sexual:
        action = ACTION_FLAG
        reason_codes = [REASON_SEXUAL_ADULT]
    else:
        action = ACTION_ALLOW
        reason_codes = [REASON_CLEAN]
    if not notes:
        worst = max((img["max_p_under_20"] for img in per_image), default=0.0)
        notes.append(
            f"passed: {len(per_image)} image(s), max nsfw "
            f"{max((img['nsfw_score'] for img in per_image), default=0.0):.2f} "
            f"(recall {max((img.get('nsfw_score_recall', 0.0) for img in per_image), default=0.0):.2f}), "
            f"max p(age<20) {worst:.2f}"
        )
    return {
        "action": action,
        "reason_code": reason_codes[0],
        "reason_codes": reason_codes,
        "minor": bool(refused),
        "sexual": bool(sexual) or bool(refused),
        "detail": "; ".join(notes),
    }


class _Models:
    """Lazy model holder: the face and age models load only when an image needs them."""

    def __init__(self, models_root: str):
        self.root = models_root
        self._classifiers: Dict[str, Any] = {}
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

    def classifier(self, config: Dict[str, Any]):
        model_id = config["id"]
        if model_id not in self._classifiers:
            session = _common.make_session(_common.resolve_onnx_path(self.path(model_id)))
            default = (224, 224) if model_id == MODEL_NSFW else (384, 384)
            hw = config.get("input_hw") or default
            self._classifiers[model_id] = (session, _common.model_input_hw(session, default_hw=(int(hw[0]), int(hw[1]))))
        return self._classifiers[model_id]

    def nsfw(self):
        return self.classifier(default_profile().classifiers[0])

    def nsfw_recall(self):
        profile = default_profile()
        return self.classifier(profile.classifiers[1] if len(profile.classifiers) > 1 else profile.classifiers[0])

    def detector(self, model_id: Optional[str] = None):
        if self._detector is None:
            self._detector = _common.make_session(_common.resolve_onnx_path(self.path(model_id or default_profile().face_id)))
        return self._detector

    def age(self, model_id: Optional[str] = None):
        """(session, labels): the ONNX age head and its bin names, read from config.json."""
        if self._age is None:
            self._age = load_head(self.path(model_id or default_profile().age_id))
        return self._age


def _classifier_payload(np, Image, model, image) -> Dict[str, Any]:
    from . import nsfw

    session, size_hw = model
    inp = nsfw._preprocess(np, image, size_hw, resample=Image.Resampling.BILINEAR)
    out = session.run(None, {session.get_inputs()[0].name: inp})
    return nsfw._payload(np, out[0])


def _classifier_score(np, Image, model, image) -> float:
    return float(_classifier_payload(np, Image, model, image)["score"])


def _nsfw_score(np, Image, models: _Models, image) -> float:
    """The precise classifier's score (the first configured classifier)."""
    return _classifier_score(np, Image, models.classifier(default_profile().classifiers[0]), image)


def _nsfw_scores(np, Image, models: _Models, image, profile: Optional[ReviewProfile] = None) -> "tuple[float, float]":
    """(primary, high-recall) scores for one image using the supplied profile."""
    profile = profile or default_profile()
    primary = _classifier_score(np, Image, models.classifier(profile.classifiers[0]), image)
    if len(profile.classifiers) < 2:
        return primary, 0.0
    return primary, _classifier_score(np, Image, models.classifier(profile.classifiers[1]), image)


def _profile_scores(np, Image, models: _Models, image, profile: ReviewProfile) -> Dict[str, float]:
    return {c["id"]: _classifier_score(np, Image, models.classifier(c), image) for c in profile.classifiers}


def _age_probs(np, Image, models: _Models, image, boxes, margin: float = FACE_CROP_MARGIN, age_id: Optional[str] = None) -> List[List[float]]:
    """Age-bin probabilities for each face box (mirror-averaged); [] for a degenerate box."""
    session, _labels = models.age(age_id)
    width, height = image.size
    tensors = []
    for box in boxes:
        crop_box = face_crop_box(box, width, height, margin)
        tensors.append(None if crop_box is None else crop_tensor(np, Image, image.crop(crop_box)))
    live = [t for t in tensors if t is not None]
    scored = iter(mirrored_probs(np, session, live))
    return [next(scored) if t is not None else [] for t in tensors]


def sexual_threshold(strict: bool, assume_sexual: bool) -> float:
    """The NSFW score at which an image counts as sexual (default profile)."""
    if assume_sexual:
        return 0.0
    profile = default_profile()
    return profile.threshold_for(profile.classifiers[0], strict)


def recall_sexual_threshold(strict: bool, assume_sexual: bool) -> float:
    """The high-recall classifier's bar (default profile). A probe treats every image as sexual."""
    if assume_sexual:
        return 0.0
    profile = default_profile()
    index = 1 if len(profile.classifiers) > 1 else 0
    return profile.threshold_for(profile.classifiers[index], strict)


def sexual_decision(primary: float, recall: float, *, strict: bool, assume_sexual: bool = False) -> bool:
    """Either default classifier crossing its own calibrated bar counts as sexual."""
    return primary >= sexual_threshold(strict, assume_sexual) or recall >= recall_sexual_threshold(strict, assume_sexual)


def review_image(
    np, Image, models: _Models, image,
    *,
    strict: bool,
    assume_sexual: bool = False,
    profile: Optional[ReviewProfile] = None,
) -> Dict[str, Any]:
    profile = profile or default_profile()
    if len(profile.classifiers) <= 2:
        primary, recall = _nsfw_scores(np, Image, models, image, profile)
        scores = {profile.classifiers[0]["id"]: primary}
        if len(profile.classifiers) > 1:
            scores[profile.classifiers[1]["id"]] = recall
    else:
        scores = _profile_scores(np, Image, models, image, profile)
        primary = float(scores.get(profile.classifiers[0]["id"], 0.0))
        recall = float(scores.get(profile.classifiers[1]["id"], 0.0)) if len(profile.classifiers) > 1 else 0.0
    sexual = profile.decide(scores, strict, assume_sexual)
    faces: List[Dict[str, Any]] = []
    if sexual:
        from . import face_restore

        models.require(profile.escalation_ids)

        found = face_restore.detect_faces(np, Image, models.detector(profile.face_id), image, profile.face_score_threshold)
        found = sorted(found, key=lambda f: (f["box"][2] - f["box"][0]) * (f["box"][3] - f["box"][1]), reverse=True)[: profile.max_faces]
        readable = [f for f in found if min(f["box"][2] - f["box"][0], f["box"][3] - f["box"][1]) >= profile.min_face_px]
        probs = _age_probs(np, Image, models, image, [f["box"] for f in readable], profile.crop_margin, profile.age_id) if readable else []
        labels = models.age(profile.age_id)[1] if readable else []
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
    judged = judge_image(primary, faces, sexual=sexual, minor_threshold=profile.age_threshold, min_face_px=profile.min_face_px)
    return {
        "nsfw_score": round(primary, 4),
        "nsfw_score_recall": round(recall, 4),
        "nsfw_scores": {key: round(value, 4) for key, value in scores.items()},
        "faces_found": len(faces),
        "assessed_faces": judged["assessed_faces"],
        "max_p_under_20": round(max((f.get("p_under_20", 0.0) for f in faces), default=0.0), 4),
        "sexual": judged["sexual"],
        "minor_face": judged["minor_face"],
        "unassessable": judged["unassessable"],
        "refuse": judged["refuse"],
        "action": judged["action"],
        "reason_code": judged["reason_code"],
        "faces": faces,
    }


def run(request: Dict[str, Any], models_root: str) -> Dict[str, Any]:
    np, _ort, Image = _common.require_deps()
    profile = ReviewProfile(request.get("profile"))
    strict = bool(request.get("strict", False))
    assume_sexual = bool(request.get("assume_sexual", False))
    models = _Models(models_root)
    images = request.get("images")
    if not isinstance(images, list) or not images:
        _common.fail("minor_review: request carries no images", code=2)
    per_image = []
    for index, item in enumerate(images):
        try:
            raw = base64.b64decode(item["base64"], validate=True)
            from PIL import ImageOps

            image = Image.open(io.BytesIO(raw))
            if image.width * image.height > profile.max_image_pixels:
                raise ValueError(f"image exceeds {profile.max_image_pixels} pixels")
            image = ImageOps.exif_transpose(image).convert("RGB")
        except Exception as exc:  # noqa: BLE001
            _common.fail(f"minor_review: image {index} is not a decodable raster: {exc}", code=6)
            raise
        per_image.append(review_image(np, Image, models, image, strict=strict, assume_sexual=assume_sexual, profile=profile))
    verdict = aggregate(per_image)
    verdict["images"] = per_image
    verdict["thresholds"] = profile.thresholds(strict)
    verdict["models"] = profile.model_ids
    verdict["strict"] = bool(strict)
    verdict["reviewer_version"] = profile.reviewer_version
    verdict["profile_name"] = profile.name
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
    try:
        profile = ReviewProfile(request.get("profile"))
    except ValueError as exc:
        _common.fail(f"minor_review: invalid review profile: {exc}", code=2)
        raise
    # The classifiers run for every image; fail before decoding anything when they
    # are absent, so the host can acquire exactly the named weights and retry.
    _Models(args.models_root).require(profile.classifier_ids)
    try:
        verdict = run(request, args.models_root)
    except ValueError as exc:
        _common.fail(f"minor_review: invalid review profile: {exc}", code=2)
        raise
    sys.stdout.write(json.dumps(verdict) + "\n")


if __name__ == "__main__":
    main()
