"""ocr — read text with the PaddleOCR PP-OCRv5 ONNX pair (detector + recogniser).

The model directory holds the pinned assets the catalog declares:

  det.onnx                PP-OCRv5 server text detector (shared by all languages)
  rec_<lang>.onnx         a PP-OCRv5 recogniser for one script group
  dict_<lang>.txt         its character dictionary, one character per line

The result is JSON the Go service decodes into its OCR result:

  {"full_text": "...", "language": "english",
   "blocks": [{"text": "...", "confidence": 0.98,
               "box": {"x": 10, "y": 20, "width": 100, "height": 30}}]}

Pipeline (the PaddleOCR reference): detector probability map, DB post-process
(threshold, connected regions, minimum-area rectangle, unclip), perspective-crop
each text line, recognise with CTC greedy decoding, then order the lines
top-to-bottom, left-to-right. Geometry uses numpy, Pillow and scikit-image only,
so the isolated environment needs no OpenCV.
"""

from __future__ import annotations

import json
import math
import os

from . import _common

DEFAULT_LANGUAGE = "english"
DET_MIN_SIDE_LEN = 64  # PP-OCRv5 default: never feed a shorter side than this ("min" limit type)
DET_MAX_SIDE_LEN = 4000  # ... and never a longer side than this
DET_THRESH = 0.3
DET_BOX_THRESH = 0.6
DET_UNCLIP_RATIO = 1.5
DET_MIN_SIDE = 3
REC_HEIGHT = 48
REC_MAX_WIDTH = 3200
MIN_TEXT_SCORE = 0.5
# The detector is trained on text a few tens of pixels tall, so scene text in a
# photograph (a book cover in a 12 MP picture) is missed at native size, and small
# text in a modest picture needs enlarging. Every image is therefore read at its own
# size and at each of these long-side targets that differs from it by DET_PYRAMID_STEP,
# and the per-scale lines are merged.
DET_PYRAMID_LONG_SIDES = (3072, 2048, 1280)
DET_PYRAMID_STEP = 1.25
MEAN = (0.485, 0.456, 0.406)
STD = (0.229, 0.224, 0.225)


def _languages(model_dir: str):
    found = []
    for name in sorted(os.listdir(model_dir)):
        if name.startswith("rec_") and name.endswith(".onnx"):
            found.append(name[len("rec_") : -len(".onnx")])
    return found


def _load_dictionary(path: str):
    with open(path, "r", encoding="utf-8") as handle:
        chars = [line.rstrip("\n").rstrip("\r") for line in handle]
    # PaddleOCR's CTC head: index 0 is the blank, then the dictionary, then a
    # trailing space character.
    return ["blank"] + chars + [" "]


def _det_preprocess(np, image):
    width, height = image.size
    scale = 1.0
    if min(height, width) < DET_MIN_SIDE_LEN:
        scale = DET_MIN_SIDE_LEN / float(min(height, width))
    if max(height, width) * scale > DET_MAX_SIDE_LEN:
        scale = DET_MAX_SIDE_LEN / float(max(height, width))
    new_h = max(32, int(round(height * scale / 32.0)) * 32)
    new_w = max(32, int(round(width * scale / 32.0)) * 32)
    resized = image.resize((new_w, new_h))
    bgr = np.asarray(resized, dtype=np.float32)[:, :, ::-1] / 255.0  # PaddleOCR feeds BGR
    arr = (bgr - np.array(MEAN, dtype=np.float32)) / np.array(STD, dtype=np.float32)
    return arr.transpose(2, 0, 1)[np.newaxis].astype(np.float32), (height / float(new_h), width / float(new_w))


def _hull(np, points):
    """Convex hull (Andrew's monotone chain) of an (N,2) float array."""
    pts = np.unique(points, axis=0)
    if len(pts) < 3:
        return pts
    order = np.lexsort((pts[:, 1], pts[:, 0]))
    pts = pts[order]

    def half(seq):
        chain = []
        for p in seq:
            while len(chain) >= 2:
                (x1, y1), (x2, y2) = chain[-2], chain[-1]
                if (x2 - x1) * (p[1] - y1) - (y2 - y1) * (p[0] - x1) <= 0:
                    chain.pop()
                else:
                    break
            chain.append(p)
        return chain

    lower, upper = half(pts), half(pts[::-1])
    return np.array(lower[:-1] + upper[:-1])


def _min_area_rect(np, points):
    """Smallest-area rotated rectangle around points: (center, (w, h), angle) with
    the rectangle's corners returned by _rect_corners."""
    hull = _hull(np, points.astype(np.float64))
    if len(hull) < 3:
        return None
    best = None
    for i in range(len(hull)):
        edge = hull[(i + 1) % len(hull)] - hull[i]
        norm = float(np.hypot(edge[0], edge[1]))
        if norm == 0:
            continue
        ux, uy = edge / norm
        rot = np.array([[ux, uy], [-uy, ux]])
        proj = hull @ rot.T
        lo, hi = proj.min(axis=0), proj.max(axis=0)
        area = float((hi[0] - lo[0]) * (hi[1] - lo[1]))
        if best is None or area < best[0]:
            best = (area, rot, lo, hi)
    _, rot, lo, hi = best
    return rot, lo, hi


def _rect_corners(np, rect, grow=0.0):
    rot, lo, hi = rect
    lo, hi = lo - grow, hi + grow
    local = np.array([[lo[0], lo[1]], [hi[0], lo[1]], [hi[0], hi[1]], [lo[0], hi[1]]])
    return local @ rot  # back to image coordinates


def _rect_size(rect):
    _rot, lo, hi = rect
    return float(hi[0] - lo[0]), float(hi[1] - lo[1])


def _order_points(np, pts):
    pts = np.asarray(pts, dtype=np.float32)
    by_x = pts[np.argsort(pts[:, 0])]
    left, right = by_x[:2], by_x[2:]
    left = left[np.argsort(left[:, 1])]
    right = right[np.argsort(right[:, 1])]
    return np.array([left[0], right[0], right[1], left[1]], dtype=np.float32)


def _region_outline(np, coords):
    """Extreme x per row and extreme y per column of a pixel region: enough points
    to give the same convex hull as every pixel, without a Python loop over them."""
    rows, cols = coords[:, 0], coords[:, 1]
    pts = []
    for axis_vals, other in ((rows, cols), (cols, rows)):
        order = np.lexsort((other, axis_vals))
        sv, so = axis_vals[order], other[order]
        first = np.r_[True, sv[1:] != sv[:-1]]
        last = np.r_[sv[1:] != sv[:-1], True]
        for sel in (first, last):
            if axis_vals is rows:
                pts.append(np.stack([so[sel], sv[sel]], axis=1))  # (x=col, y=row)
            else:
                pts.append(np.stack([sv[sel], so[sel]], axis=1))
    return np.concatenate(pts).astype(np.float64)


def _detect(np, session, image, long_side=None):
    """Text boxes (source pixels) from one detector pass; ``long_side`` runs the
    pass on a copy whose longer side is that many pixels."""
    from skimage.draw import polygon as fill_polygon
    from skimage.measure import label, regionprops

    width, height = image.size
    source = image
    if long_side and max(width, height) != long_side:
        factor = long_side / float(max(width, height))
        source = image.resize((max(1, round(width * factor)), max(1, round(height * factor))))
    inp, (ratio_h, ratio_w) = _det_preprocess(np, source)
    ratio_h, ratio_w = ratio_h * height / source.size[1], ratio_w * width / source.size[0]
    prob = session.run(None, {session.get_inputs()[0].name: inp})[0]
    prob = np.asarray(prob, dtype=np.float32).reshape(prob.shape[-2], prob.shape[-1])
    labelled = label(prob > DET_THRESH, connectivity=2)
    boxes = []
    for region in regionprops(labelled)[:1000]:
        if region.area < 4:
            continue
        rect = _min_area_rect(np, _region_outline(np, region.coords))
        if rect is None:
            continue
        w, h = _rect_size(rect)
        if min(w, h) < DET_MIN_SIDE:
            continue
        corners = _rect_corners(np, rect)
        rr, cc = fill_polygon(corners[:, 1], corners[:, 0], prob.shape)
        if rr.size == 0 or float(prob[rr, cc].mean()) < DET_BOX_THRESH:
            continue
        distance = (w * h) * DET_UNCLIP_RATIO / (2.0 * (w + h))
        grown = _rect_corners(np, rect, grow=distance)
        if min(w, h) + 2 * distance < DET_MIN_SIDE + 2:
            continue
        points = _order_points(np, grown)
        points[:, 0] = np.clip(points[:, 0] * ratio_w, 0, width - 1)
        points[:, 1] = np.clip(points[:, 1] * ratio_h, 0, height - 1)
        boxes.append(points)
    return boxes


def _crop(np, image, points):
    from PIL import Image

    width = int(max(np.linalg.norm(points[0] - points[1]), np.linalg.norm(points[2] - points[3])))
    height = int(max(np.linalg.norm(points[0] - points[3]), np.linalg.norm(points[1] - points[2])))
    if width < 1 or height < 1:
        return None
    # PIL's QUAD maps the output rectangle from a source quadrilateral given as
    # top-left, bottom-left, bottom-right, top-right.
    quad = (
        points[0][0], points[0][1], points[3][0], points[3][1],
        points[2][0], points[2][1], points[1][0], points[1][1],
    )
    crop = image.transform((width, height), Image.QUAD, quad, resample=Image.BICUBIC)
    if height / float(width) >= 1.5:
        crop = crop.rotate(90, expand=True)  # a tall crop is a vertically set line
    return crop


def _rec_preprocess(np, crop):
    width, height = crop.size
    new_w = min(REC_MAX_WIDTH, max(8, int(math.ceil(REC_HEIGHT * width / float(height)))))
    resized = np.asarray(crop.resize((new_w, REC_HEIGHT)), dtype=np.float32)[:, :, ::-1] / 255.0  # BGR
    resized = (resized - 0.5) / 0.5
    return resized.transpose(2, 0, 1)[np.newaxis].astype(np.float32)


def _recognise(np, session, chars, crop):
    inp = _rec_preprocess(np, crop)
    logits = np.asarray(session.run(None, {session.get_inputs()[0].name: inp})[0], dtype=np.float32)
    logits = logits.reshape(-1, logits.shape[-1])
    if logits.shape[1] != len(chars):
        _common.fail(
            f"recogniser has {logits.shape[1]} classes but its dictionary gives {len(chars)}; "
            "the dictionary does not belong to this recogniser",
            code=9,
        )
    best = logits.argmax(axis=1)
    scores = logits.max(axis=1)
    text, kept, previous = [], [], 0
    for index, score in zip(best.tolist(), scores.tolist()):
        if index != 0 and index != previous:
            text.append(chars[index])
            kept.append(score)
        previous = index
    confidence = float(np.mean(kept)) if kept else 0.0
    return "".join(text), confidence


def _pyramid(image):
    """Long-side targets for one image, descending: its own size (capped by the
    detector limit) plus each pyramid size at least DET_PYRAMID_STEP away from every
    size already chosen."""
    sizes = [min(max(image.size), DET_MAX_SIDE_LEN)]
    for target in DET_PYRAMID_LONG_SIDES:
        if all(max(target, kept) / float(min(target, kept)) >= DET_PYRAMID_STEP for kept in sizes):
            sizes.append(target)
    return sorted(sizes, reverse=True)


def _inside(point, quad):
    """Whether point lies in the convex quadrilateral quad (corners in order)."""
    sign = 0
    for i in range(4):
        (x1, y1), (x2, y2) = quad[i], quad[(i + 1) % 4]
        cross = (x2 - x1) * (point[1] - y1) - (y2 - y1) * (point[0] - x1)
        if cross == 0:
            continue
        if sign == 0:
            sign = 1 if cross > 0 else -1
        elif (cross > 0) != (sign > 0):
            return False
    return True


def _centre(quad):
    return (sum(p[0] for p in quad) / 4.0, sum(p[1] for p in quad) / 4.0)


def _merge_scales(blocks):
    """Keep the best reading of each piece of text across scales. Readings are ranked
    by characters read times confidence, so a whole line read once beats a fragment
    of it read with higher confidence at another scale. A reading whose centre lies
    inside the rotated outline of an already kept one is the same text and dropped."""
    kept = []
    for block in sorted(blocks, key=lambda b: -b["confidence"] * len(b["text"].strip())):
        centre = _centre(block["quad"])
        if not any(_inside(centre, other["quad"]) for other in kept):
            kept.append(block)
    return kept


def _reading_order(blocks):
    """Top-to-bottom, left-to-right, treating boxes that overlap vertically by
    more than half the shorter height as one row."""
    blocks = sorted(blocks, key=lambda b: (b["box"]["y"], b["box"]["x"]))
    ordered = []
    for block in blocks:
        placed = False
        for row in ordered:
            top = max(row[0]["box"]["y"], block["box"]["y"])
            bottom = min(row[0]["box"]["y"] + row[0]["box"]["height"], block["box"]["y"] + block["box"]["height"])
            shorter = min(row[0]["box"]["height"], block["box"]["height"])
            if shorter > 0 and bottom - top > 0.5 * shorter:
                row.append(block)
                placed = True
                break
        if not placed:
            ordered.append([block])
    return [b for row in ordered for b in sorted(row, key=lambda item: item["box"]["x"])], ordered


def main() -> None:
    import argparse

    parser = argparse.ArgumentParser(description="PaddleOCR ONNX text recognition")
    parser.add_argument("--model", required=True, help="model directory (det.onnx, rec_<lang>.onnx, dict_<lang>.txt)")
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--language", default=DEFAULT_LANGUAGE)
    parser.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    args = parser.parse_args()
    os.environ["IMAGE_TOOLS_DEVICE"] = args.device

    np, _ort, Image = _common.require_deps()
    model_dir = args.model
    if not os.path.isdir(model_dir):
        _common.fail(f"OCR model directory not found at {model_dir!r}", code=4)
    languages = _languages(model_dir)
    if args.language not in languages:
        _common.fail(f"language {args.language!r} is not installed; installed: {', '.join(languages) or 'none'}", code=4)
    det_path = os.path.join(model_dir, "det.onnx")
    rec_path = os.path.join(model_dir, f"rec_{args.language}.onnx")
    dict_path = os.path.join(model_dir, f"dict_{args.language}.txt")
    for path in (det_path, rec_path, dict_path):
        if not os.path.isfile(path):
            _common.fail(f"missing OCR asset {path!r}", code=4)

    try:
        image, _meta = _common.load_oriented_image(args.image, max_pixels=100_000_000)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)
        raise
    det = _common.make_session(det_path)
    rec = _common.make_session(rec_path)
    chars = _load_dictionary(dict_path)

    blocks = []
    for long_side in _pyramid(image):
        for points in _detect(np, det, image, long_side):
            crop = _crop(np, image, points)
            if crop is None:
                continue
            text, confidence = _recognise(np, rec, chars, crop)
            if not text.strip() or confidence < MIN_TEXT_SCORE:
                continue
            x0, y0 = points[:, 0].min(), points[:, 1].min()
            x1, y1 = points[:, 0].max(), points[:, 1].max()
            blocks.append(
                {
                    "quad": points.tolist(),
                    "text": text,
                    "confidence": round(confidence, 4),
                    "box": {"x": int(x0), "y": int(y0), "width": int(x1 - x0), "height": int(y1 - y0)},
                }
            )
    blocks = [{k: v for k, v in b.items() if k != "quad"} for b in _merge_scales(blocks)]
    ordered, rows = _reading_order(blocks)
    full_text = "\n".join(" ".join(b["text"] for b in sorted(row, key=lambda item: item["box"]["x"])) for row in rows)
    payload = {"full_text": full_text, "language": args.language, "blocks": ordered}
    try:
        with open(args.out, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, ensure_ascii=False, indent=2)
            handle.write("\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
