#!/usr/bin/env python3
"""Build a normalized visual review sheet for the eight-screen reference.

The source screenshot is a 1200x480 contact sheet, not eight native captures.
The crop rectangles below isolate its eight phone panels. Each panel is resized
to the curated workspace capture size before a simple pixel-difference summary
is computed. The metrics are review aids only: JPEG compression, crop alignment,
resampling, and device-frame differences make them unsuitable as a quality gate.

Example:
  python3 tools/compare_eight_screen_reference.py \
    --reference docs/mockups/assets/eight-screens-reference.jpeg \
    --captures /path/to/eight-screens-live-v41 \
    --capture-version v41 \
    --output /path/to/eight-screens-reference-vs-local-v41.png
"""

from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw


SCREENS = (
    ("money-night", (25, 78, 170, 412)),
    ("money-day", (176, 78, 319, 412)),
    ("run-before", (320, 78, 460, 412)),
    ("wind-down", (460, 78, 604, 412)),
    ("plant-day", (607, 78, 743, 412)),
    ("phrase-practice", (746, 78, 883, 412)),
    ("run-after", (887, 78, 1026, 412)),
    ("plant-night", (1028, 78, 1174, 412)),
)
CAPTURE_SIZE = (390, 844)
REVIEW_SIZE = (155, 335)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--reference", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument(
        "--captures",
        type=Path,
        default=Path(__file__).parents[1] / "docs/mockups/assets",
    )
    parser.add_argument(
        "--capture-version",
        default="v28",
        help="Version shown in the review labels and metadata (default: v28).",
    )
    parser.add_argument(
        "--capture-suffix",
        default="-v28",
        help="Filename suffix before .png for each screen capture (default: -v28; use an empty string for name.png).",
    )
    args = parser.parse_args()

    reference = Image.open(args.reference).convert("RGB")
    if reference.size != (1200, 480):
        raise SystemExit(f"Expected a 1200x480 reference contact sheet; got {reference.size}")

    sheet = Image.new("RGB", (len(SCREENS) * REVIEW_SIZE[0], 2 * (REVIEW_SIZE[1] + 26)), "#f7f6f3")
    draw = ImageDraw.Draw(sheet)
    metrics: list[dict[str, object]] = []
    for index, (name, crop_box) in enumerate(SCREENS):
        baseline = reference.crop(crop_box).resize(CAPTURE_SIZE, Image.Resampling.LANCZOS)
        candidate_path = args.captures / f"{name}{args.capture_suffix}.png"
        candidate = Image.open(candidate_path).convert("RGB")
        if candidate.size != CAPTURE_SIZE:
            # Browser captures may use a device scale factor greater than 1.
            # Normalize them to the same logical dimensions as the reference.
            candidate = candidate.resize(CAPTURE_SIZE, Image.Resampling.LANCZOS)

        # Descriptive RGB MAE and changed-pixel share after a fixed threshold.
        baseline_pixels = list(baseline.getdata())
        candidate_pixels = list(candidate.getdata())
        differences = [
            tuple(abs(a - b) for a, b in zip(left, right, strict=True))
            for left, right in zip(baseline_pixels, candidate_pixels, strict=True)
        ]
        mae = sum(sum(pixel) for pixel in differences) / (len(differences) * 3)
        changed = sum(max(pixel) > 20 for pixel in differences)
        metrics.append(
            {
                "screen": name,
                "candidate": candidate_path.name,
                "crop_box_px": crop_box,
                "rgb_mean_absolute_error_0_255": round(mae, 3),
                "pixels_over_20_rgb_levels_percent": round(100 * changed / len(differences), 2),
            }
        )

        x = index * REVIEW_SIZE[0]
        sheet.paste(baseline.resize(REVIEW_SIZE, Image.Resampling.LANCZOS), (x, 0))
        draw.text((x + 4, REVIEW_SIZE[1] + 5), f"{name} · reference", fill="#111111")
        sheet.paste(candidate.resize(REVIEW_SIZE, Image.Resampling.LANCZOS), (x, REVIEW_SIZE[1] + 26))
        draw.text((x + 4, 2 * REVIEW_SIZE[1] + 31), f"{name} · {args.capture_version}", fill="#111111")

    args.output.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(args.output)
    metrics_path = args.output.with_suffix(".json")
    metrics_path.write_text(
        json.dumps(
            {
                "reference": str(args.reference),
                "captures": os.path.relpath(args.captures, Path(__file__).parents[1]),
                "capture_version": args.capture_version,
                "capture_suffix": args.capture_suffix,
                "normalization": "manual phone-panel crops resized to 390x844 with Lanczos",
                "metric_limit": "Descriptive only; not a visual-quality score or acceptance threshold.",
                "screens": metrics,
            },
            indent=2,
        )
        + "\n",
        encoding="utf-8",
    )
    print(f"Review sheet: {args.output}")
    print(f"Metrics: {metrics_path}")
    for item in metrics:
        print(
            f"{item['screen']}: MAE={item['rgb_mean_absolute_error_0_255']} "
            f"changed={item['pixels_over_20_rgb_levels_percent']}%"
        )


if __name__ == "__main__":
    main()
