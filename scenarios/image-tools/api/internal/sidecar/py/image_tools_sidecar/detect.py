"""object_detection — run a YOLOX-like ONNX detector and write boxes as JSON.

The parser accepts common postprocessed detector outputs shaped Nx(4+scores) or
1xNx(4+scores). Boxes are emitted in normalized xyxy coordinates with anonymous
class ids so the result remains useful even when a model does not ship a labels
asset.
"""

from __future__ import annotations

import json

from . import _common


def _preprocess(np, image, size_hw):
    h, w = size_hw
    resized = image.resize((w, h))
    arr = np.array(resized).astype(np.float32) / 255.0
    return arr.transpose(2, 0, 1)[np.newaxis, :, :, :].astype(np.float32)


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

    inp = _preprocess(np, image, (h, w))
    input_name = session.get_inputs()[0].name
    outputs = session.run(None, {input_name: inp})
    boxes = _detections(np, outputs, image.width, image.height, threshold=0.25)
    payload = {"detections": boxes, "count": len(boxes)}
    try:
        with open(args.out, "wb") as f:
            f.write(json.dumps(payload, indent=2).encode("utf-8") + b"\n")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
