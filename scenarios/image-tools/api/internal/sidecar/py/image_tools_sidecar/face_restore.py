"""face_restore — detect, align, restore and paste back every face in a photo
with a 512px blind face-restoration ONNX model (CodeFormer, RestoreFormer++ or
any export with the same contract).

The installed model directory holds two ONNX graphs, told apart by signature
rather than by filename:

  * the face detector — OpenCV Zoo YuNet (``cls_8`` … ``kps_32`` outputs,
    fixed 640x640 BGR input, 0..255, no normalization);
  * the restorer — a ``(1,3,512,512)`` RGB input in [-1, 1] whose first output is
    the restored face in the same range. An optional scalar ``weight`` input is
    CodeFormer's fidelity knob (0 = strongest restoration, 1 = closest to the
    input face).

Each detected face is warped with a similarity transform from its five YuNet
landmarks onto the FFHQ 512 template, restored, warped back and feathered into
the original raster. Pixels outside faces are untouched, so the output keeps
the source size. A photo with no detectable face fails loud (exit 9) instead of
returning the input unchanged.
"""

from __future__ import annotations

import argparse
import os

from . import _common

# FFHQ-aligned five-point template at 512x512 (eye centres, nose tip, mouth
# corners; image-left first), the alignment the restorers were trained on.
FFHQ_TEMPLATE_512 = (
    (192.98138, 239.94708),
    (318.90277, 240.19360),
    (256.63416, 314.01935),
    (201.26117, 371.41043),
    (313.08905, 371.15118),
)
FACE_SIZE = 512
DETECTOR_SIZE = 640
DETECTOR_STRIDES = (8, 16, 32)
DEFAULT_FIDELITY = 0.5
DEFAULT_SCORE_THRESHOLD = 0.6
NMS_THRESHOLD = 0.3
MAX_INPUT_PIXELS = 64_000_000
# Mean |R-G|+|G-B| (0..510 scale) at or below which a photo counts as monochrome.
GRAY_CHROMA_TOLERANCE = 3.0


def _is_detector(session) -> bool:
    return {"cls_8", "obj_8", "bbox_8", "kps_8"} <= {o.name for o in session.get_outputs()}


def _resolve_sessions(model_arg: str):
    """Return (detector, restorer) sessions from the installed model directory."""
    if not os.path.isdir(model_arg):
        _common.fail(f"face_restore needs the installed model directory, got {model_arg!r}", code=4)
    paths = [os.path.join(model_arg, f) for f in sorted(os.listdir(model_arg)) if f.endswith(".onnx")]
    detector = restorer = None
    for path in paths:
        session = _common.make_session(path)
        if _is_detector(session):
            detector = detector or session
        else:
            restorer = restorer or session
    if detector is None:
        _common.fail(f"no YuNet face detector ONNX found in {model_arg!r}", code=4)
    if restorer is None:
        _common.fail(f"no 512px face restorer ONNX found in {model_arg!r}", code=4)
    return detector, restorer


def _nms(np, boxes, scores, threshold):
    """Greedy IoU suppression over x1,y1,x2,y2 boxes; returns kept indices."""
    order = scores.argsort()[::-1]
    keep = []
    while order.size:
        i = int(order[0])
        keep.append(i)
        rest = order[1:]
        xx1 = np.maximum(boxes[i, 0], boxes[rest, 0])
        yy1 = np.maximum(boxes[i, 1], boxes[rest, 1])
        xx2 = np.minimum(boxes[i, 2], boxes[rest, 2])
        yy2 = np.minimum(boxes[i, 3], boxes[rest, 3])
        inter = np.clip(xx2 - xx1, 0, None) * np.clip(yy2 - yy1, 0, None)
        area_i = (boxes[i, 2] - boxes[i, 0]) * (boxes[i, 3] - boxes[i, 1])
        area_r = (boxes[rest, 2] - boxes[rest, 0]) * (boxes[rest, 3] - boxes[rest, 1])
        iou = inter / np.maximum(area_i + area_r - inter, 1e-9)
        order = rest[iou <= threshold]
    return keep


def decode_yunet(np, outputs, scale, score_threshold=DEFAULT_SCORE_THRESHOLD):
    """Decode YuNet's per-stride heads into faces in source-raster coordinates.

    ``outputs`` maps output name -> array. ``scale`` is source px per detector px.
    Returns a list of {"box": (x1,y1,x2,y2), "score": s, "landmarks": 5x2}.
    """
    boxes, scores, kps = [], [], []
    for stride in DETECTOR_STRIDES:
        cols = rows = DETECTOR_SIZE // stride
        cls = np.clip(outputs[f"cls_{stride}"].reshape(-1), 0, 1)
        obj = np.clip(outputs[f"obj_{stride}"].reshape(-1), 0, 1)
        bbox = outputs[f"bbox_{stride}"].reshape(-1, 4)
        kp = outputs[f"kps_{stride}"].reshape(-1, 10)
        score = np.sqrt(cls * obj)
        idx = np.nonzero(score >= score_threshold)[0]
        if idx.size == 0:
            continue
        c = (idx % cols).astype(np.float32)
        r = (idx // cols).astype(np.float32)
        cx = (c + bbox[idx, 0]) * stride
        cy = (r + bbox[idx, 1]) * stride
        w = np.exp(bbox[idx, 2]) * stride
        h = np.exp(bbox[idx, 3]) * stride
        boxes.append(np.stack([cx - w / 2, cy - h / 2, cx + w / 2, cy + h / 2], axis=1))
        scores.append(score[idx])
        pts = kp[idx].reshape(-1, 5, 2)
        pts[:, :, 0] = (pts[:, :, 0] + c[:, None]) * stride
        pts[:, :, 1] = (pts[:, :, 1] + r[:, None]) * stride
        kps.append(pts)
    if not boxes:
        return []
    boxes = np.concatenate(boxes) * scale
    scores = np.concatenate(scores)
    kps = np.concatenate(kps) * scale
    del rows
    return [
        {"box": tuple(float(v) for v in boxes[i]), "score": float(scores[i]), "landmarks": kps[i]}
        for i in _nms(np, boxes, scores, NMS_THRESHOLD)
    ]


def detect_faces(np, Image, detector, image, score_threshold=DEFAULT_SCORE_THRESHOLD):
    """Letterbox the RGB image into YuNet's fixed 640px input and decode faces."""
    width, height = image.size
    scale = max(width, height) / DETECTOR_SIZE
    resized = image.resize((max(1, round(width / scale)), max(1, round(height / scale))), Image.BILINEAR)
    canvas = np.zeros((DETECTOR_SIZE, DETECTOR_SIZE, 3), dtype=np.float32)
    arr = np.asarray(resized, dtype=np.float32)[:, :, ::-1]  # RGB -> BGR
    canvas[: arr.shape[0], : arr.shape[1]] = arr
    feed = {detector.get_inputs()[0].name: canvas.transpose(2, 0, 1)[np.newaxis]}
    names = [o.name for o in detector.get_outputs()]
    outputs = dict(zip(names, detector.run(None, feed)))
    return decode_yunet(np, outputs, scale, score_threshold)


def similarity_transform(np, src, dst):
    """Least-squares similarity (Umeyama, no reflection) mapping src -> dst.

    Returns a 3x3 homogeneous matrix.
    """
    src = np.asarray(src, dtype=np.float64)
    dst = np.asarray(dst, dtype=np.float64)
    src_mean, dst_mean = src.mean(axis=0), dst.mean(axis=0)
    src_c, dst_c = src - src_mean, dst - dst_mean
    cov = dst_c.T @ src_c / len(src)
    u, s, vt = np.linalg.svd(cov)
    d = np.ones(2)
    if np.linalg.det(u) * np.linalg.det(vt) < 0:
        d[-1] = -1
    rot = u @ np.diag(d) @ vt
    var = (src_c ** 2).sum() / len(src)
    scale = (s * d).sum() / var
    m = np.eye(3)
    m[:2, :2] = scale * rot
    m[:2, 2] = dst_mean - scale * rot @ src_mean
    return m


def _affine_data(m):
    """PIL AFFINE coefficients for an output->input 3x3 matrix."""
    return (m[0, 0], m[0, 1], m[0, 2], m[1, 0], m[1, 1], m[1, 2])


def feather_mask(np, Image, ImageFilter, size=FACE_SIZE):
    """A face-space blend mask: opaque centre, soft edge, zero at the border."""
    margin = size // 16
    mask = Image.new("L", (size, size), 0)
    mask.paste(255, (margin, margin, size - margin, size - margin))
    return mask.filter(ImageFilter.GaussianBlur(radius=margin * 0.75))


def restore_face(np, restorer, face_rgb, fidelity):
    """Run the restorer on one aligned 512x512 RGB uint8 face."""
    x = (face_rgb.astype(np.float32) / 255.0 - 0.5) / 0.5
    feed = {}
    for inp in restorer.get_inputs():
        if inp.name == "weight":
            feed[inp.name] = np.array(fidelity, dtype=np.float64)
        else:
            feed[inp.name] = x.transpose(2, 0, 1)[np.newaxis]
    out = restorer.run([restorer.get_outputs()[0].name], feed)[0]
    out = np.clip(np.squeeze(out, 0).transpose(1, 2, 0), -1, 1)
    return ((out + 1) * 127.5 + 0.5).astype(np.uint8)


def paste_face(np, Image, base, restored, mask, to_face):
    """Warp a restored face back through ``to_face`` (source->face) and blend.

    ``base`` is a float32 HxWx3 array updated in place.
    """
    height, width = base.shape[:2]
    corners = np.array([[0, 0, 1], [FACE_SIZE, 0, 1], [0, FACE_SIZE, 1], [FACE_SIZE, FACE_SIZE, 1]], dtype=np.float64)
    quad = (np.linalg.inv(to_face) @ corners.T).T[:, :2]
    x0, y0 = (max(0, int(np.floor(v))) for v in quad.min(axis=0))
    x1 = min(width, int(np.ceil(quad[:, 0].max())))
    y1 = min(height, int(np.ceil(quad[:, 1].max())))
    if x1 <= x0 or y1 <= y0:
        return
    shift = np.array([[1, 0, x0], [0, 1, y0], [0, 0, 1]], dtype=np.float64)
    data = _affine_data(to_face @ shift)
    region = (x1 - x0, y1 - y0)
    face = Image.fromarray(restored).transform(region, Image.AFFINE, data, resample=Image.BICUBIC)
    alpha = mask.transform(region, Image.AFFINE, data, resample=Image.BILINEAR)
    a = np.asarray(alpha, dtype=np.float32)[:, :, None] / 255.0
    patch = base[y0:y1, x0:x1]
    patch[:] = a * np.asarray(face, dtype=np.float32) + (1 - a) * patch


def is_monochrome(np, image, tolerance=GRAY_CHROMA_TOLERANCE):
    """True when the RGB raster carries no colour (a grey or sepia-free scan)."""
    arr = np.asarray(image, dtype=np.int16)
    chroma = np.abs(arr[..., 0] - arr[..., 1]) + np.abs(arr[..., 1] - arr[..., 2])
    return float(chroma.mean()) <= tolerance


def to_monochrome(np, face_rgb):
    """Rec.601 luminance replicated to three channels."""
    luma = face_rgb.astype(np.float32) @ np.array([0.299, 0.587, 0.114], dtype=np.float32)
    return np.repeat(np.clip(luma + 0.5, 0, 255).astype(np.uint8)[..., None], 3, axis=2)


def restore_image(np, Image, ImageFilter, detector, restorer, image, fidelity=DEFAULT_FIDELITY, score_threshold=DEFAULT_SCORE_THRESHOLD):
    """Restore every detected face; returns (RGB image, face count).

    Restorers trained on colour FFHQ invent skin and eye colour on black-and-white
    photos (CodeFormer paints blue irises and pink lips); a monochrome source
    therefore keeps only the restored luminance.
    """
    faces = detect_faces(np, Image, detector, image, score_threshold)
    if not faces:
        return image, 0
    monochrome = is_monochrome(np, image)
    template = np.array(FFHQ_TEMPLATE_512, dtype=np.float64)
    mask = feather_mask(np, Image, ImageFilter)
    base = np.asarray(image, dtype=np.float32).copy()
    # Smaller faces first so a large foreground face is blended last and wins
    # any overlap.
    for face in sorted(faces, key=lambda f: (f["box"][2] - f["box"][0]) * (f["box"][3] - f["box"][1])):
        to_face = similarity_transform(np, face["landmarks"], template)
        aligned = image.transform((FACE_SIZE, FACE_SIZE), Image.AFFINE, _affine_data(np.linalg.inv(to_face)), resample=Image.BICUBIC)
        restored = restore_face(np, restorer, np.asarray(aligned), fidelity)
        if monochrome:
            restored = to_monochrome(np, restored)
        paste_face(np, Image, base, restored, mask, to_face)
    return Image.fromarray(np.clip(base + 0.5, 0, 255).astype(np.uint8)), len(faces)


def main():
    p = argparse.ArgumentParser(description="ONNX blind face restoration (detect, align, restore, paste back)")
    p.add_argument("--model", required=True, help="installed model directory (restorer + YuNet ONNX)")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    p.add_argument("--fidelity", type=float, default=DEFAULT_FIDELITY, help="CodeFormer fidelity weight 0..1 (ignored by restorers without it)")
    p.add_argument("--score-threshold", type=float, default=DEFAULT_SCORE_THRESHOLD)
    args = p.parse_args()
    os.environ["IMAGE_TOOLS_DEVICE"] = args.device
    if not 0.0 <= args.fidelity <= 1.0:
        _common.fail(f"--fidelity must be within 0..1, got {args.fidelity}", code=2)

    np, _ort, Image = _common.require_deps()
    from PIL import ImageFilter  # noqa: WPS433

    detector, restorer = _resolve_sessions(args.model)
    try:
        image = _common.load_oriented_image(args.image, MAX_INPUT_PIXELS)[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)
    out, count = restore_image(np, Image, ImageFilter, detector, restorer, image, args.fidelity, args.score_threshold)
    if count == 0:
        _common.fail("no face detected in the input image; nothing to restore", code=9)
    try:
        out.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)
    return {"faces": count}


if __name__ == "__main__":
    main()
