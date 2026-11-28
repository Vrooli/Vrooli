"""nsfw_classify — the shared sexual-content classifier.

This entrypoint backs both the standalone `nsfw_classify` analysis op and the
generation auto-scan. It runs the same classifier ensemble and thresholds as the
bundled reviewer, so the two surfaces answer the same question the same way. The
review profile is passed by the host (a JSON object; see minor_review.py); the
legacy single-model form is kept for a direct invocation that names --model.

Output:

  {"score": 0.91, "recall_score": 0.40, "sexual": true,
   "threshold": 0.30, "thresholds": {...}, "models": [...],
   "reason_code": "sexual_adult", "categories": [...]}

`sexual` is the ensemble decision (any classifier crossing its own bar). `score`
is the first (precise) classifier's probability, kept for compatibility with the
analysis UI, and `recall_score` is the high-recall classifier when one ran.
"""

from __future__ import annotations

import argparse
import json
import sys
from typing import Any, Dict, Optional

from . import _common


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="ONNX NSFW classification (profile-driven ensemble)")
    p.add_argument("--models-root", help="image-tools data root that holds models/<id>/")
    p.add_argument("--profile", help="review profile JSON (same shape as minor_review)")
    p.add_argument("--model", help="legacy single-model directory or ONNX path")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", help="optional JSON output path; stdout when omitted")
    return _common.parse_device_args(p)


def _softmax(np, logits):
    arr = np.asarray(logits, dtype=np.float32).reshape(-1)
    if arr.size == 0:
        return arr
    arr = arr - float(arr.max())
    exp = np.exp(arr)
    denom = float(exp.sum())
    if denom <= 0:
        return np.zeros_like(exp)
    return exp / denom


def _preprocess(np, image, size_hw, resample):
    h, w = size_hw
    resized = image.resize((w, h), resample=resample)
    arr = np.array(resized).astype(np.float32) / 255.0
    if arr.ndim == 2:
        arr = np.stack([arr] * 3, axis=-1)
    arr = arr[:, :, :3]
    mean = np.array([0.5, 0.5, 0.5], dtype=np.float32)
    std = np.array([0.5, 0.5, 0.5], dtype=np.float32)
    arr = (arr - mean) / std
    return arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)


def _payload(np, raw_output):
    probs = _softmax(np, raw_output)
    labels = ["sfw", "nsfw"]
    categories = []
    for idx, score in enumerate(probs.tolist()):
        label = labels[idx] if idx < len(labels) else f"class_{idx}"
        categories.append({"label": label, "score": float(score)})
    nsfw_score = 0.0
    for c in categories:
        if c["label"].lower() == "nsfw":
            nsfw_score = float(c["score"])
            break
    return {"score": nsfw_score, "categories": categories}


def _single_model_payload(np, Image, model_path: str, image) -> Dict[str, Any]:
    session = _common.make_session(_common.resolve_onnx_path(model_path))
    size_hw = _common.model_input_hw(session, default_hw=(384, 384))
    inp = _preprocess(np, image, size_hw, resample=Image.Resampling.BILINEAR)
    outputs = session.run(None, {session.get_inputs()[0].name: inp})
    return _payload(np, outputs[0])


def classify(image_path: str, models_root: Optional[str], profile_json: Optional[str], model_path: Optional[str]) -> Dict[str, Any]:
    np, _ort, Image = _common.require_deps()
    from . import minor_review as mr

    profile = mr.ReviewProfile(json.loads(profile_json) if profile_json else None)
    try:
        image = Image.open(image_path).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {image_path!r}: {exc}", code=6)
        raise

    if model_path:
        # Legacy direct call: one named model, one score. The profile still
        # supplies its threshold, so an operator cannot accidentally use a
        # different bar than the reviewer.
        payload = _single_model_payload(np, Image, model_path, image)
        payloads = {profile.classifiers[0]["id"]: payload}
        scores = {profile.classifiers[0]["id"]: float(payload["score"])}
    else:
        if not models_root:
            _common.fail("nsfw: --models-root or --model is required", code=2)
        models = mr._Models(models_root)
        models.require(profile.classifier_ids)
        payloads = {c["id"]: mr._classifier_payload(np, Image, models.classifier(c), image) for c in profile.classifiers}
        scores = {model_id: float(payload["score"]) for model_id, payload in payloads.items()}

    thresholds = profile.thresholds(False)
    chosen = profile.classifiers[0]["id"]
    best_ratio = -1.0
    for c in profile.classifiers:
        threshold = thresholds.get(c["id"], 0.0)
        if threshold > 0:
            ratio = scores.get(c["id"], 0.0) / threshold
            if ratio > best_ratio:
                chosen, best_ratio = c["id"], ratio
    sexual = profile.decide(scores, strict=False)
    return {
        "score": round(float(scores.get(chosen, 0.0)), 6),
        "threshold": thresholds.get(chosen, 0.0),
        "primary_score": round(float(scores.get(profile.classifiers[0]["id"], 0.0)), 6),
        "primary_threshold": thresholds.get(profile.classifiers[0]["id"], 0.0),
        "recall_score": round(float(scores.get(profile.classifiers[1]["id"], 0.0)), 6) if len(profile.classifiers) > 1 else 0.0,
        "scores": {key: round(value, 6) for key, value in scores.items()},
        "sexual": sexual,
        "thresholds": thresholds,
        "models": profile.classifier_ids,
        "reason_code": mr.REASON_SEXUAL_ADULT if sexual else mr.REASON_CLEAN,
        "categories": payloads.get(chosen, {}).get("categories", []),
    }


def main() -> None:
    args = _parse_args()
    data = json.dumps(classify(args.image, args.models_root, args.profile, args.model), indent=2).encode("utf-8") + b"\n"
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
