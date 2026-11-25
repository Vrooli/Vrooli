"""face_count — how many people does each image show (the consent gate's question).

Same stdin contract as the bundled reviewer, one JSON object,

  {"images": [{"media_type": "image/png", "base64": "..."}]}

and one JSON object on stdout,

  {"counts": [<faces in image 0>, <faces in image 1>, ...]}

It runs the OpenCV-Zoo YuNet detector the reviewer and face_restore already ship, on
the CPU, so asking costs a fraction of a second and never takes VRAM from a job.
Every face counts at the reviewer's detector threshold, whatever its size: the
question is "is anyone shown", not "can the age model read them".

Errors (undecodable image, missing weights, runtime failure) exit non-zero. The host
treats that as "could not check" and keeps the consent requirement (fail closed).
Missing weights exit 4 with ``missing-weights=yunet`` on stderr, like the reviewer.
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
from .minor_review import (
    EXIT_MISSING_WEIGHTS,
    FACE_SCORE_THRESHOLD,
    MAX_IMAGE_PIXELS,
    MODEL_FACES,
    ReviewProfile,
)


def _decode(np, Image, item: Dict[str, Any], index: int, max_pixels: int = MAX_IMAGE_PIXELS):
    from PIL import ImageOps

    try:
        raw = base64.b64decode(item["base64"], validate=True)
        image = Image.open(io.BytesIO(raw))
        if image.width * image.height > max_pixels:
            raise ValueError(f"image exceeds {max_pixels} pixels")
        return ImageOps.exif_transpose(image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"face_count: image {index} is not a decodable raster: {exc}", code=6)
        raise


def count_faces(request: Dict[str, Any], models_root: str, detector=None) -> Dict[str, List[int]]:
    """Count the faces of every image in the request; ``detector`` is injectable for tests."""
    np, _ort, Image = _common.require_deps()
    profile = ReviewProfile(request.get("profile"))
    images = request.get("images")
    if not isinstance(images, list) or not images:
        _common.fail("face_count: request carries no images", code=2)
    model_id = profile.face_id or MODEL_FACES
    model_dir = os.path.join(models_root, "models", model_id)
    if detector is None:
        if not os.path.isdir(model_dir):
            _common.fail(
                f"face_count: model weights are not installed: missing-weights={model_id}",
                code=EXIT_MISSING_WEIGHTS,
            )
        detector = _common.make_session(_common.resolve_onnx_path(model_dir))
    from . import face_restore

    counts = []
    for index, item in enumerate(images):
        image = _decode(np, Image, item, index, profile.max_image_pixels)
        counts.append(len(face_restore.detect_faces(np, Image, detector, image, profile.face_score_threshold or FACE_SCORE_THRESHOLD)))
    return {"counts": counts}


def main(argv: Optional[Sequence[str]] = None) -> None:
    p = argparse.ArgumentParser(description="face counter for the consent gate (stdin JSON -> stdout JSON)")
    p.add_argument("--models-root", required=True, help="image-tools data root that holds models/<id>/")
    args = _common.parse_device_args(p) if argv is None else p.parse_args(argv)
    try:
        request = json.load(sys.stdin)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"face_count: stdin is not a JSON object: {exc}", code=2)
        raise
    sys.stdout.write(json.dumps(count_faces(request, args.models_root)) + "\n")


if __name__ == "__main__":
    main()
