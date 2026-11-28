"""face_detection — run OpenCV YuNet and write anonymous face boxes as JSON."""

from __future__ import annotations

import argparse
import json

from . import _common


def _load_deps():
    try:
        import cv2  # noqa: WPS433
        import numpy as np  # noqa: WPS433,F401
    except ImportError as exc:  # pragma: no cover - exercised on unprovisioned hosts
        _common.fail(
            f"missing Python dependency ({exc}). Provision OpenCV for YuNet: "
            "install opencv-python-headless and numpy (see docs/reference/backends.md)",
            code=3,
        )
        raise
    return cv2


def _detector_factory(cv2):
    if hasattr(cv2, "FaceDetectorYN"):
        return cv2.FaceDetectorYN.create
    if hasattr(cv2, "FaceDetectorYN_create"):
        return cv2.FaceDetectorYN_create
    _common.fail("OpenCV build lacks FaceDetectorYN YuNet support", code=4)
    raise SystemExit(4)


def _face_row(row):
    vals = [float(v) for v in row]
    face = {
        "bbox": {
            "x": vals[0],
            "y": vals[1],
            "width": vals[2],
            "height": vals[3],
        },
        "score": vals[14] if len(vals) > 14 else None,
    }
    if len(vals) >= 14:
        face["landmarks"] = [
            {"x": vals[i], "y": vals[i + 1]}
            for i in range(4, 14, 2)
        ]
    return face


def main() -> None:
    p = argparse.ArgumentParser(description="OpenCV YuNet face detection")
    p.add_argument("--model", required=True, help="path to YuNet ONNX weight")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", required=True, help="output JSON path")
    p.add_argument("--score-threshold", type=float, default=0.6)
    p.add_argument("--nms-threshold", type=float, default=0.3)
    p.add_argument("--top-k", type=int, default=5000)
    args = p.parse_args()

    cv2 = _load_deps()
    image = cv2.imread(args.image)
    if image is None:
        _common.fail(f"failed to open input image {args.image!r}", code=5)
    height, width = image.shape[:2]

    try:
        create_detector = _detector_factory(cv2)
        detector = create_detector(
            args.model,
            "",
            (width, height),
            args.score_threshold,
            args.nms_threshold,
            args.top_k,
        )
        _retval, faces = detector.detect(image)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"YuNet detection failed: {exc}", code=6)

    rows = [] if faces is None else [_face_row(row) for row in faces]
    payload = {
        "backend": "library-cgo",
        "model": "yunet",
        "count": len(rows),
        "image": {"width": width, "height": height},
        "faces": rows,
    }
    try:
        with open(args.out, "w", encoding="utf-8") as fh:
            json.dump(payload, fh, indent=2)
            fh.write("\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
