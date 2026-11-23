"""pose_extract — read human pose keypoints from an image with a DWPose ONNX pair.

The installed model directory holds two graphs, told apart by input shape rather
than by filename:

  * the person detector — YOLOX, ``(1,3,640,640)`` BGR in [0, 255];
  * the pose estimator — RTMPose whole-body (``dw-ll_ucoco_384``),
    ``(1,3,384,288)``, producing SimCC x/y heatmaps for the 133 COCO-WholeBody
    keypoints (17 body, 6 feet, 68 face, 21 + 21 hands).

For every person box the estimator runs on a padded, aspect-corrected crop and
the keypoints are mapped back to image pixels. The output is JSON, not a picture:
the body in OpenPose-18 order (what the OpenPose ControlNets are trained on) and
both hands, each point ``[x, y, confidence]`` in pixels.

Person boxes come from, in order: the caller's ``box``, the detector, and — for
drawings, which a detector trained on photographs does not recognise — the
foreground box of the page (``box_source`` says which). Line-art mannequins are
the hard case this fallback exists for: the estimator reads photographs, and on a
bare outline drawing it finds barely half the joints. A drawing is therefore
prepared before it is read (``prepare``: ``auto`` finds line art by itself): the
outline is filled with a shaded skin tone and the original lines are kept faint,
which gave at least nine of the eighteen body joints on all nine mannequins of the
measured chart where the bare drawing gave that on four, and found the limbs the
bare drawing missed.
"""

from __future__ import annotations

import argparse
import json
import os

from . import _common

MAX_INPUT_PIXELS = 64_000_000
DETECTOR_SIZE = 640
DETECTOR_SCORE = 0.3
DETECTOR_NMS_IOU = 0.45
DETECTOR_PAD_VALUE = 114.0
POSE_PADDING = 1.25
SIMCC_SPLIT_RATIO = 2.0
KEYPOINT_FLOOR = 0.3
# Line-art preparation: a page is line art when it is mostly the page colour, nearly
# colourless, and carries ink on only a small share of its pixels.
LINE_ART_PAGE_SHARE = 0.7
LINE_ART_INK_SHARE = 0.3
LINE_ART_SATURATION = 40
LINE_ART_INK_LEVEL = 200
LINE_ART_BLUR = 0.008  # of the longer side
LINE_ART_FILL = (205.0, 160.0, 135.0)
LINE_ART_LINE = (60.0, 40.0, 35.0)
LINE_ART_LINE_WEIGHT = 0.45
LINE_ART_VIEW_MARGIN = 0.3
PREPARE_MODES = ("auto", "none", "line_art")
MEAN = (123.675, 116.28, 103.53)
STD = (58.395, 57.12, 57.375)

# OpenPose-18 body order. Each entry names its source in COCO-WholeBody (index into
# the 133 keypoints) or, for the neck, the two shoulders it sits between.
BODY_NAMES = (
    "nose", "neck", "right_shoulder", "right_elbow", "right_wrist",
    "left_shoulder", "left_elbow", "left_wrist", "right_hip", "right_knee",
    "right_ankle", "left_hip", "left_knee", "left_ankle", "right_eye",
    "left_eye", "right_ear", "left_ear",
)
_COCO_INDEX = {
    "nose": 0, "left_eye": 1, "right_eye": 2, "left_ear": 3, "right_ear": 4,
    "left_shoulder": 5, "right_shoulder": 6, "left_elbow": 7, "right_elbow": 8,
    "left_wrist": 9, "right_wrist": 10, "left_hip": 11, "right_hip": 12,
    "left_knee": 13, "right_knee": 14, "left_ankle": 15, "right_ankle": 16,
}
LEFT_HAND = slice(91, 112)
RIGHT_HAND = slice(112, 133)


class Graphs:
    def __init__(self, detector, estimator):
        self.detector, self.estimator = detector, estimator


def _input_hw(session):
    shape = session.get_inputs()[0].shape
    return int(shape[2]), int(shape[3])


def resolve_graphs(model_dir: str) -> Graphs:
    """Open the ONNX graphs of an installed model directory and assign each a role."""
    if not os.path.isdir(model_dir):
        _common.fail(f"pose_extract needs the installed model directory, got {model_dir!r}", code=4)
    detector = estimator = None
    for name in sorted(os.listdir(model_dir)):
        if not name.endswith(".onnx"):
            continue
        session = _common.make_session(os.path.join(model_dir, name))
        height, width = _input_hw(session)
        if height == width:
            detector = detector or session
        elif height > width:
            estimator = estimator or session
    if estimator is None:
        _common.fail(f"no DWPose estimator ONNX (portrait input) found in {model_dir!r}", code=4)
    return Graphs(detector, estimator)


def _iou(np, box, boxes):
    x0 = np.maximum(box[0], boxes[:, 0])
    y0 = np.maximum(box[1], boxes[:, 1])
    x1 = np.minimum(box[2], boxes[:, 2])
    y1 = np.minimum(box[3], boxes[:, 3])
    inter = np.clip(x1 - x0, 0, None) * np.clip(y1 - y0, 0, None)
    area = (box[2] - box[0]) * (box[3] - box[1])
    areas = (boxes[:, 2] - boxes[:, 0]) * (boxes[:, 3] - boxes[:, 1])
    return inter / np.maximum(area + areas - inter, 1e-9)


def nms(np, boxes, scores, threshold):
    """Greedy non-maximum suppression; returns the kept indices, best first."""
    order = list(np.argsort(-scores))
    keep = []
    while order:
        best = order.pop(0)
        keep.append(best)
        if not order:
            break
        overlaps = _iou(np, boxes[best], boxes[order])
        order = [i for i, o in zip(order, overlaps) if o <= threshold]
    return keep


def detect_people(np, Image, session, image):
    """Person boxes ``[x0, y0, x1, y1, score]`` in pixels, best first (YOLOX)."""
    width, height = image.size
    size = DETECTOR_SIZE
    ratio = min(size / height, size / width)
    resized = image.resize((max(1, int(width * ratio)), max(1, int(height * ratio))), Image.BILINEAR)
    canvas = np.full((size, size, 3), DETECTOR_PAD_VALUE, dtype=np.float32)
    canvas[: resized.height, : resized.width] = np.asarray(resized, dtype=np.float32)[:, :, ::-1]
    blob = canvas.transpose(2, 0, 1)[None]
    raw = np.asarray(session.run(None, {session.get_inputs()[0].name: blob})[0])[0]
    grids, strides = [], []
    for stride in (8, 16, 32):
        cells = size // stride
        xv, yv = np.meshgrid(np.arange(cells), np.arange(cells))
        grids.append(np.stack((xv, yv), 2).reshape(-1, 2))
        strides.append(np.full((cells * cells, 1), stride))
    grid, stride = np.concatenate(grids), np.concatenate(strides)
    raw = raw.copy()
    raw[:, :2] = (raw[:, :2] + grid) * stride
    raw[:, 2:4] = np.exp(raw[:, 2:4]) * stride
    person = raw[:, 4] * raw[:, 5]  # objectness x class 0 (person)
    keep = person > DETECTOR_SCORE
    if not keep.any():
        return []
    centre, wh = raw[keep, :2], raw[keep, 2:4]
    boxes = np.concatenate([centre - wh / 2, centre + wh / 2], axis=1) / ratio
    scores = person[keep]
    kept = nms(np, boxes, scores, DETECTOR_NMS_IOU)
    return [[float(v) for v in boxes[i]] + [float(scores[i])] for i in kept]


def foreground_box(np, image, margin=0.0):
    """Bounding box of everything that is not the page: pixels that differ from the
    border's median colour. Returns None for a blank image."""
    arr = np.asarray(image, dtype=np.float32)
    border = np.concatenate([arr[0], arr[-1], arr[:, 0], arr[:, -1]], axis=0)
    ink = np.abs(arr - np.median(border, axis=0)).max(axis=2) > 40.0
    if not ink.any():
        return None
    ys, xs = np.nonzero(ink)
    x0, x1, y0, y1 = xs.min(), xs.max() + 1, ys.min(), ys.max() + 1
    pad_x, pad_y = (x1 - x0) * margin, (y1 - y0) * margin
    return [float(max(0, x0 - pad_x)), float(max(0, y0 - pad_y)),
            float(min(image.width, x1 + pad_x)), float(min(image.height, y1 + pad_y))]


def crop_scale(box, aspect):
    """Centre and size of the padded estimator crop for a person box, corrected to
    the estimator's input aspect (width / height)."""
    cx, cy = (box[0] + box[2]) / 2, (box[1] + box[3]) / 2
    w, h = (box[2] - box[0]) * POSE_PADDING, (box[3] - box[1]) * POSE_PADDING
    if w > h * aspect:
        h = w / aspect
    else:
        w = h * aspect
    return (cx, cy), (w, h)


def crop_person(np, Image, image, centre, scale, size_hw):
    """The padded crop, resized to the estimator input; outside the image is the
    page colour so a person at the edge does not meet a black border."""
    height, width = size_hw
    x0, y0 = centre[0] - scale[0] / 2, centre[1] - scale[1] / 2
    arr = np.asarray(image, dtype=np.float32)
    border = np.concatenate([arr[0], arr[-1], arr[:, 0], arr[:, -1]], axis=0)
    fill = tuple(int(v) for v in np.median(border, axis=0))
    pad = int(max(0, -x0, -y0, x0 + scale[0] - image.width, y0 + scale[1] - image.height)) + 2
    if pad > 2:
        canvas = Image.new("RGB", (image.width + 2 * pad, image.height + 2 * pad), fill)
        canvas.paste(image, (pad, pad))
        src, x0, y0 = canvas, x0 + pad, y0 + pad
    else:
        src = image
    return src.resize((width, height), Image.BICUBIC, box=(x0, y0, x0 + scale[0], y0 + scale[1]))


def decode_simcc(np, simcc_x, simcc_y, centre, scale, size_hw):
    """SimCC x/y heatmaps to ``(133, 3)`` image-pixel ``[x, y, confidence]``."""
    height, width = size_hw
    sx, sy = np.asarray(simcc_x)[0], np.asarray(simcc_y)[0]
    x, y = sx.argmax(axis=1), sy.argmax(axis=1)
    conf = np.minimum(sx.max(axis=1), sy.max(axis=1))
    xs = x / SIMCC_SPLIT_RATIO / width * scale[0] + centre[0] - scale[0] / 2
    ys = y / SIMCC_SPLIT_RATIO / height * scale[1] + centre[1] - scale[1] / 2
    return np.stack([xs, ys, np.where(conf > 0, conf, 0.0)], axis=1)


def estimate_pose(np, Image, session, image, box):
    """Whole-body keypoints ``(133, 3)`` for one person box."""
    size_hw = _input_hw(session)
    centre, scale = crop_scale(box, size_hw[1] / size_hw[0])
    crop = crop_person(np, Image, image, centre, scale, size_hw)
    tensor = (np.asarray(crop, dtype=np.float32)[:, :, ::-1] - np.array(MEAN, np.float32)) / np.array(STD, np.float32)
    feed = {session.get_inputs()[0].name: tensor.transpose(2, 0, 1)[None].astype(np.float32)}
    simcc_x, simcc_y = session.run(None, feed)[:2]
    return decode_simcc(np, simcc_x, simcc_y, centre, scale, size_hw)


def is_line_art(np, image):
    """True for a drawing: light page, almost no colour, ink on a small share of it."""
    arr = np.asarray(image, dtype=np.int16)
    grey = arr.mean(axis=2)
    colourful = (arr.max(axis=2) - arr.min(axis=2)) > LINE_ART_SATURATION
    page = (grey > LINE_ART_INK_LEVEL).mean()
    ink = (grey < LINE_ART_INK_LEVEL).mean()
    return page >= LINE_ART_PAGE_SHARE and ink <= LINE_ART_INK_SHARE and colourful.mean() < 0.02


def line_art_to_figure(np, Image, image):
    """A drawing as the estimator can read it: the outline filled with a skin tone
    shaded by distance from the edge (so limbs read as volumes), the original lines
    kept faint, then softened. Anything enclosed by the outline is body; the page
    stays white."""
    from scipy import ndimage  # noqa: WPS433

    grey = np.asarray(image.convert("L"))
    ink = grey < LINE_ART_INK_LEVEL
    body = ndimage.binary_fill_holes(ndimage.binary_closing(ink, structure=np.ones((5, 5)), iterations=2))
    if not body.any():
        return image
    depth = ndimage.distance_transform_edt(body)
    depth = np.clip(depth / max(1.0, np.percentile(depth[body], 95)), 0.0, 1.0)
    shade = 0.62 + 0.38 * np.sqrt(depth)
    out = np.full(body.shape + (3,), 255.0)
    out[body] = np.array(LINE_ART_FILL)[None, :] * shade[body][:, None]
    out[ink] = out[ink] * (1 - LINE_ART_LINE_WEIGHT) + np.array(LINE_ART_LINE) * LINE_ART_LINE_WEIGHT
    out[ink & ~body] = (50, 40, 40)
    sigma = max(2.0, LINE_ART_BLUR * max(body.shape))
    out = ndimage.gaussian_filter(out, (sigma, sigma, 0))
    return Image.fromarray(np.clip(out, 0, 255).astype(np.uint8))


def estimate_person(np, Image, session, image, box, prepare):
    """Whole-body keypoints ``(133, 3)`` for one person box, reading a prepared view
    of the surroundings when the image is a drawing."""
    if prepare == "none" or (prepare == "auto" and not is_line_art(np, image)):
        return estimate_pose(np, Image, session, image, box)
    grow_x, grow_y = (box[2] - box[0]) * LINE_ART_VIEW_MARGIN, (box[3] - box[1]) * LINE_ART_VIEW_MARGIN
    view_box = (max(0, int(box[0] - grow_x)), max(0, int(box[1] - grow_y)),
                min(image.width, int(box[2] + grow_x) + 1), min(image.height, int(box[3] + grow_y) + 1))
    view = line_art_to_figure(np, Image, image.crop(view_box))
    local = [box[0] - view_box[0], box[1] - view_box[1], box[2] - view_box[0], box[3] - view_box[1]]
    whole = estimate_pose(np, Image, session, view, local)
    whole[:, 0] += view_box[0]
    whole[:, 1] += view_box[1]
    return whole


def body18(np, whole):
    """OpenPose-18 ``[x, y, confidence]`` rows from the 133 whole-body keypoints."""
    rows = []
    for name in BODY_NAMES:
        if name == "neck":
            left, right = whole[_COCO_INDEX["left_shoulder"]], whole[_COCO_INDEX["right_shoulder"]]
            visible = left[2] > KEYPOINT_FLOOR and right[2] > KEYPOINT_FLOOR
            rows.append([(left[0] + right[0]) / 2, (left[1] + right[1]) / 2, min(left[2], right[2]) if visible else 0.0])
        else:
            rows.append(list(whole[_COCO_INDEX[name]]))
    return rows


def _round(rows):
    return [[round(float(v), 2) for v in row] for row in rows]


def person_record(np, whole, box, source, score):
    return {
        "box": [round(float(v), 1) for v in box],
        "box_source": source,
        "score": round(float(score), 4),
        "body": _round(body18(np, whole)),
        "left_hand": _round(whole[LEFT_HAND]),
        "right_hand": _round(whole[RIGHT_HAND]),
    }


def extract(np, Image, graphs, image, options):
    """Pose of every person in ``image`` as the JSON-ready result document."""
    box = options.get("box")
    max_people = max(1, int(options.get("max_people", 4)))
    prepare = options.get("prepare", "auto")
    if prepare not in PREPARE_MODES:
        raise _common.InputError(f"prepare must be one of {', '.join(PREPARE_MODES)}, got {prepare!r}")
    if box:
        if len(box) != 4 or box[2] <= box[0] or box[3] <= box[1]:
            raise _common.InputError(f"box must be [x0, y0, x1, y1] with positive size, got {box!r}")
        boxes = [(list(map(float, box)), "given", 1.0)]
    else:
        found = detect_people(np, Image, graphs.detector, image) if graphs.detector is not None else []
        boxes = [(d[:4], "detector", d[4]) for d in found[:max_people]]
        if not boxes:
            page = foreground_box(np, image)
            if page is None:
                raise _common.InputError("the image is blank: no person or drawing to read a pose from")
            boxes = [(page, "foreground", 0.0)]
    people = [person_record(np, estimate_person(np, Image, graphs.estimator, image, b, prepare), b, source, score) for b, source, score in boxes]
    return {
        "schema": "pose-v1",
        "image": {"width": image.width, "height": image.height},
        "body_keypoints": list(BODY_NAMES),
        "keypoint_floor": KEYPOINT_FLOOR,
        "people": people,
    }


def main():
    p = argparse.ArgumentParser(description="DWPose ONNX pose keypoints (JSON)")
    p.add_argument("--model", required=True, help="installed model directory (person detector + pose estimator)")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", required=True, help="output JSON path")
    p.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    p.add_argument("--params", default="", help="JSON operation parameters (pose_extract params_schema)")
    args = p.parse_args()
    os.environ["IMAGE_TOOLS_DEVICE"] = args.device
    try:
        options = json.loads(args.params) if args.params else {}
    except json.JSONDecodeError as exc:
        _common.fail(f"--params is not valid JSON: {exc}", code=2)
    if not isinstance(options, dict):
        _common.fail("--params must be a JSON object", code=2)
    np, _ort, Image = _common.require_deps()
    graphs = resolve_graphs(args.model)
    try:
        image = _common.load_oriented_image(args.image, MAX_INPUT_PIXELS)[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)
    document = extract(np, Image, graphs, image, options)
    try:
        with open(args.out, "wb") as f:
            f.write(json.dumps(document).encode("utf-8") + b"\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)
    return {"people": len(document["people"]), "box_source": document["people"][0]["box_source"]}


if __name__ == "__main__":
    main()
