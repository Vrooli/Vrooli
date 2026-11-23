"""Measure the bundled minor-safety reviewer's face + age stages on a labelled manifest.

Run with the scenario's private venv interpreter (the same one the API uses) and the
sidecar on PYTHONPATH, for example:

  LD_LIBRARY_PATH=<venv>/lib/python3.12/site-packages/nvidia/cu13/lib \
  PYTHONPATH=api/internal/sidecar/py CUDA_VISIBLE_DEVICES= \
  <venv>/bin/python -s scripts/minor-review-eval.py --models-root <data root> \
      --manifest <manifest.json> --out <report.json>

The manifest is a JSON list of {"file": path, "age_bin": "0-2|3-9|10-19|20-29|...", ...}.
Every image is run through the face detector and the age model exactly as the reviewer
does (no NSFW gate, because this measures age sensitivity on its own); the report gives,
per labelled bin, the face-detection rate and the share of faces flagged at each
candidate P(age < 20) threshold, so a threshold is chosen from data rather than by taste.
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from collections import defaultdict

THRESHOLDS = (0.10, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.50, 0.60)


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--models-root", required=True)
    p.add_argument("--manifest", required=True)
    p.add_argument("--out", required=True)
    p.add_argument("--limit", type=int, default=0)
    p.add_argument(
        "--nsfw",
        action="store_true",
        help="also record the NSFW classifier score per image (a false-positive measurement on SFW portraits)",
    )
    args = p.parse_args()

    from image_tools_sidecar import _common, face_restore, minor_review as mr  # noqa: WPS433

    np, _ort, Image = _common.require_deps()
    from PIL import ImageOps

    models = mr._Models(args.models_root)
    manifest = json.load(open(args.manifest))
    if args.limit:
        manifest = manifest[: args.limit]

    per_bin = defaultdict(lambda: {"n": 0, "no_face": 0, "tiny": 0, "p": [], "nsfw": []})
    started = time.time()
    for k, row in enumerate(manifest):
        image = ImageOps.exif_transpose(Image.open(row["file"])).convert("RGB")
        bucket = per_bin[row["age_bin"]]
        bucket["n"] += 1
        if args.nsfw:
            bucket["nsfw"].append(mr._nsfw_score(np, Image, models, image))
        found = face_restore.detect_faces(np, Image, models.detector(), image, mr.FACE_SCORE_THRESHOLD)
        if not found:
            bucket["no_face"] += 1
            continue
        # The labelled subject is the dominant face: judge the largest.
        face = max(found, key=lambda f: (f["box"][2] - f["box"][0]) * (f["box"][3] - f["box"][1]))
        size = min(face["box"][2] - face["box"][0], face["box"][3] - face["box"][1])
        if size < mr.MIN_FACE_PX:
            bucket["tiny"] += 1
            continue
        probs = mr._age_probs(np, Image, models, image, [face["box"]])[0]
        labels = models.age()[1]
        bucket["p"].append(mr.under_age_mass(labels, probs))
        if (k + 1) % 500 == 0:
            print(f"{k + 1}/{len(manifest)} {time.time() - started:.0f}s", file=sys.stderr)

    report = {"images": len(manifest), "seconds": round(time.time() - started, 1), "bins": {}}
    for age_bin, b in sorted(per_bin.items()):
        ps = b["p"]
        flagged = {f"{t:.2f}": (sum(1 for v in ps if v >= t) / len(ps) if ps else None) for t in THRESHOLDS}
        report["bins"][age_bin] = {
            "n": b["n"],
            "no_face": b["no_face"],
            "tiny_face": b["tiny"],
            "assessed": len(ps),
            "mean_p_under_20": (sum(ps) / len(ps)) if ps else None,
            "flag_rate_at": flagged,
            "p_values": [round(v, 4) for v in ps],
        }
        if b["nsfw"]:
            scores = b["nsfw"]
            report["bins"][age_bin]["nsfw_flag_rate_at"] = {
                f"{t:.2f}": sum(1 for v in scores if v >= t) / len(scores) for t in (0.10, 0.30, 0.50)
            }
            report["bins"][age_bin]["nsfw_max"] = round(max(scores), 4)
    json.dump(report, open(args.out, "w"), indent=2)
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
