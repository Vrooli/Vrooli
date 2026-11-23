"""object_detection — run a YOLOX-like ONNX detector and write boxes as JSON.

Two output shapes are understood:

* the official YOLOX export, whose head is raw (``1 x N x (5+classes)`` with N the
  number of grid cells at strides 8/16/32 for the model's input size): the sidecar
  letterboxes the image with grey padding, feeds BGR pixels in 0-255 (YOLOX does not
  normalise), decodes the grid and strides itself, runs class-wise NMS, and undoes
  the letterbox;
* an already postprocessed ``Nx(4+scores)`` or ``1xNx(4+scores)`` array.

Boxes are emitted in normalized xyxy coordinates with the class id and, for the
80-class COCO head, its label.
"""

from __future__ import annotations

import json

from . import _common


# The 80 COCO class names, in the order the official YOLOX heads emit them.
COCO_LABELS = (
    "person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
    "traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat",
    "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack",
    "umbrella", "handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball",
    "kite", "baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket",
    "bottle", "wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple",
    "sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair",
    "couch", "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse",
    "remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink",
    "refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier",
    "toothbrush",
)
STRIDES = (8, 16, 32)
PAD_VALUE = 114.0  # YOLOX's letterbox grey
NMS_IOU = 0.45
MAX_DETECTIONS = 100


def _grid_cells(size_hw):
    h, w = size_hw
    return sum((h // st) * (w // st) for st in STRIDES)


def is_raw_yolox_head(np, arr, size_hw):
    """True for an official-export head: N rows of 5+classes with N the grid cell count."""
    arr = np.squeeze(np.asarray(arr))
    return arr.ndim == 2 and arr.shape[1] >= 6 and arr.shape[0] == _grid_cells(size_hw)


def letterbox(np, image, size_hw):
    """Scale to fit, pad with grey, return (CHW float32 BGR 0-255 batch, scale)."""
    h, w = size_hw
    scale = min(h / image.height, w / image.width)
    nw, nh = max(1, int(image.width * scale)), max(1, int(image.height * scale))
    resized = np.asarray(image.convert("RGB").resize((nw, nh)), dtype=np.float32)[:, :, ::-1]
    canvas = np.full((h, w, 3), PAD_VALUE, dtype=np.float32)
    canvas[:nh, :nw] = resized
    return canvas.transpose(2, 0, 1)[np.newaxis].astype(np.float32), scale


def decode_grid(np, raw, size_hw):
    """Turn the raw head into (cx, cy, w, h) in letterboxed pixels, per YOLOX's demo."""
    h, w = size_hw
    grids, strides = [], []
    for st in STRIDES:
        gy, gx = np.meshgrid(np.arange(h // st), np.arange(w // st), indexing="ij")
        grids.append(np.stack((gx, gy), axis=2).reshape(-1, 2))
        strides.append(np.full((grids[-1].shape[0], 1), st))
    grid, stride = np.concatenate(grids), np.concatenate(strides)
    out = raw.copy()
    out[:, :2] = (raw[:, :2] + grid) * stride
    out[:, 2:4] = np.exp(raw[:, 2:4]) * stride
    return out


def _iou(a, b):
    ix1, iy1 = max(a[0], b[0]), max(a[1], b[1])
    ix2, iy2 = min(a[2], b[2]), min(a[3], b[3])
    inter = max(0.0, ix2 - ix1) * max(0.0, iy2 - iy1)
    union = (a[2] - a[0]) * (a[3] - a[1]) + (b[2] - b[0]) * (b[3] - b[1]) - inter
    return inter / union if union > 0 else 0.0


def nms_by_class(candidates, iou_threshold=NMS_IOU):
    """Keep the best-scoring box of each overlapping group, within one class."""
    kept = []
    for cand in sorted(candidates, key=lambda c: c["score"], reverse=True):
        if all(k["class_id"] != cand["class_id"] or _iou(k["xyxy"], cand["xyxy"]) < iou_threshold for k in kept):
            kept.append(cand)
    return kept


def yolox_detections(np, raw, size_hw, scale, width, height, threshold):
    """Decode, filter, NMS and undo the letterbox; boxes normalised to the source image."""
    decoded = decode_grid(np, np.squeeze(np.asarray(raw, dtype=np.float32)), size_hw)
    scores_all = decoded[:, 4:5] * decoded[:, 5:]
    class_ids = scores_all.argmax(axis=1)
    scores = scores_all[np.arange(len(scores_all)), class_ids]
    keep = scores >= threshold
    candidates = []
    for row, cls, score in zip(decoded[keep], class_ids[keep], scores[keep]):
        cx, cy, bw, bh = (float(v) for v in row[:4])
        xyxy = [(cx - bw / 2) / scale, (cy - bh / 2) / scale, (cx + bw / 2) / scale, (cy + bh / 2) / scale]
        candidates.append({"xyxy": xyxy, "score": float(score), "class_id": int(cls)})
    out = []
    for det in nms_by_class(candidates)[:MAX_DETECTIONS]:
        x1, y1, x2, y2 = det["xyxy"]
        entry = {
            "box": [max(0.0, min(1.0, x1 / width)), max(0.0, min(1.0, y1 / height)), max(0.0, min(1.0, x2 / width)), max(0.0, min(1.0, y2 / height))],
            "score": det["score"],
            "class_id": det["class_id"],
        }
        if det["class_id"] < len(COCO_LABELS):
            entry["label"] = COCO_LABELS[det["class_id"]]
        out.append(entry)
    return out


def _xywh_to_xyxy(row):
    cx, cy, bw, bh = [float(x) for x in row[:4]]
    return [cx - bw / 2.0, cy - bh / 2.0, cx + bw / 2.0, cy + bh / 2.0]


def _normalize_box(box, width, height):
    # Some exports emit pixels, others emit normalized coordinates.
    scale_x = float(width) if max(abs(box[0]), abs(box[2])) > 1.5 else 1.0
    scale_y = float(height) if max(abs(box[1]), abs(box[3])) > 1.5 else 1.0
    x1 = max(0.0, min(1.0, float(box[0]) / scale_x))
    y1 = max(0.0, min(1.0, float(box[1]) / scale_y))
    x2 = max(0.0, min(1.0, float(box[2]) / scale_x))
    y2 = max(0.0, min(1.0, float(box[3]) / scale_y))
    return [x1, y1, x2, y2]


def _detections(np, outputs, width, height, threshold):
    arr = np.asarray(outputs[0], dtype=np.float32)
    arr = np.squeeze(arr)
    if arr.ndim == 1:
        arr = arr.reshape(1, -1)
    if arr.ndim != 2 or arr.shape[1] < 5:
        _common.fail(f"unsupported detector output shape {arr.shape}", code=8)

    boxes = []
    for row in arr:
        if row.shape[0] >= 6:
            objectness = float(row[4])
            class_scores = row[5:]
            class_id = int(np.argmax(class_scores)) if class_scores.size else 0
            score = objectness * float(class_scores[class_id]) if class_scores.size else objectness
        else:
            class_id = 0
            score = float(row[4])
        if score < threshold:
            continue
        box = _normalize_box(_xywh_to_xyxy(row), width, height)
        boxes.append({"box": box, "score": score, "class_id": class_id})
    boxes.sort(key=lambda item: item["score"], reverse=True)
    return boxes[:100]


def main() -> None:
    args = _common.parse_io_args("ONNX object detection")
    np, _ort, Image = _common.require_deps()

    onnx_path = _common.resolve_onnx_path(args.model)
    session = _common.make_session(onnx_path)
    h, w = _common.model_input_hw(session)

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    input_name = session.get_inputs()[0].name
    inp, scale = letterbox(np, image, (h, w))
    outputs = session.run(None, {input_name: inp})
    if is_raw_yolox_head(np, outputs[0], (h, w)):
        boxes = yolox_detections(np, outputs[0], (h, w), scale, image.width, image.height, threshold=0.3)
    else:
        boxes = _detections(np, outputs, image.width, image.height, threshold=0.25)
    payload = {"detections": boxes, "count": len(boxes)}
    try:
        with open(args.out, "wb") as f:
            f.write(json.dumps(payload, indent=2).encode("utf-8") + b"\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
