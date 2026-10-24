"""MoGe-2 scene geometry inference with camera recovery and durable archive output.

The upstream ONNX graph exports ``forward()``, not ``infer()``. This module
implements the missing focal/shift fit, positive-depth validity, pinhole
reprojection, and metric scale application before writing the arrays.
"""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import math
import os
import zipfile

import numpy as np

from . import _common


MAX_PIXELS = 4_000_000
MAX_ARCHIVE_BYTES = 128 * 1024 * 1024
MAX_ARRAY_BYTES = 124 * 1024 * 1024
FOCAL_FIT_SIZE = 64


def normalized_view_plane_uv(width: int, height: int) -> np.ndarray:
    """Match MoGe's pixel-center normalized view-plane coordinates."""
    aspect = width / height
    span_x = aspect / math.sqrt(1.0 + aspect * aspect)
    span_y = 1.0 / math.sqrt(1.0 + aspect * aspect)
    u = np.linspace(-span_x * (width - 1) / width, span_x * (width - 1) / width, width, dtype=np.float32)
    v = np.linspace(-span_y * (height - 1) / height, span_y * (height - 1) / height, height, dtype=np.float32)
    uu, vv = np.meshgrid(u, v, indexing="xy")
    return np.stack((uu, vv), axis=-1)


def _fit_residual(points: np.ndarray, uv: np.ndarray, shift: float) -> tuple[float, np.ndarray]:
    denom = points[:, 2] + shift
    if not np.isfinite(denom).all() or float(denom.min()) <= 1e-7:
        return float("inf"), np.full_like(uv, np.inf)
    xy_projected = points[:, :2] / denom[:, None]
    divisor = float(np.square(xy_projected).sum())
    if divisor <= 1e-12:
        return float("inf"), np.full_like(uv, np.inf)
    focal = float((xy_projected * uv).sum() / divisor)
    if not math.isfinite(focal) or focal <= 1e-7:
        return float("inf"), np.full_like(uv, np.inf)
    residual = focal * xy_projected - uv
    return float(np.square(residual).sum()), residual


def recover_focal_shift(points: np.ndarray, mask: np.ndarray) -> tuple[float, float]:
    """Recover MoGe's relative-diagonal focal and Z shift using 1-D LM.

    This minimizes the same least-squares reprojection error as upstream's
    ``solve_optimal_focal_shift`` while avoiding a SciPy runtime dependency.
    """
    height, width = points.shape[:2]
    rows = np.minimum((np.arange(FOCAL_FIT_SIZE) * height / FOCAL_FIT_SIZE).astype(np.intp), height - 1)
    cols = np.minimum((np.arange(FOCAL_FIT_SIZE) * width / FOCAL_FIT_SIZE).astype(np.intp), width - 1)
    sampled_points = points[rows[:, None], cols[None, :]].reshape(-1, 3)
    sampled_mask = mask[rows[:, None], cols[None, :]].reshape(-1)
    sampled_uv = normalized_view_plane_uv(width, height)[rows[:, None], cols[None, :]].reshape(-1, 2)
    keep = sampled_mask & np.isfinite(sampled_points).all(axis=1) & np.isfinite(sampled_uv).all(axis=1)
    xyz, uv = sampled_points[keep], sampled_uv[keep]
    if len(xyz) < 8:
        raise ValueError("MoGe camera recovery needs at least 8 valid samples")

    # Upstream starts the Levenberg-Marquardt solve at shift=0. Clamp steps only
    # to keep all candidate camera-space Z values positive and numerically safe.
    lower = -float(xyz[:, 2].min()) + 1e-5
    shift = max(0.0, lower + 1e-5)
    damping = 1e-3
    cost, residual = _fit_residual(xyz, uv, shift)
    if not math.isfinite(cost):
        shift = lower + max(1e-4, abs(lower) * 1e-3)
        cost, residual = _fit_residual(xyz, uv, shift)
    for _ in range(100):
        eps = max(1e-5, abs(shift) * 1e-4)
        _, r_hi = _fit_residual(xyz, uv, shift + eps)
        _, r_lo = _fit_residual(xyz, uv, max(lower + 1e-6, shift - eps))
        jac = ((r_hi - r_lo) / (2.0 * eps)).reshape(-1)
        r = residual.reshape(-1)
        denom = float(jac @ jac) + damping
        if not math.isfinite(denom) or denom <= 1e-12:
            break
        step = -float(jac @ r) / denom
        step = float(np.clip(step, -max(0.1, abs(shift) * 0.5 + 0.1), max(0.1, abs(shift) * 0.5 + 0.1)))
        candidate = max(lower + 1e-6, shift + step)
        candidate_cost, candidate_residual = _fit_residual(xyz, uv, candidate)
        if candidate_cost < cost:
            shift, cost, residual = candidate, candidate_cost, candidate_residual
            damping = max(damping * 0.3, 1e-9)
            if abs(step) < 1e-7:
                break
        else:
            damping = min(damping * 10.0, 1e9)

    # Compute final focal using the same least-squares projection model.
    denominator = xyz[:, 2] + shift
    xy_projected = xyz[:, :2] / denominator[:, None]
    focal = float((xy_projected * uv).sum() / np.square(xy_projected).sum())
    if not math.isfinite(focal) or focal <= 0:
        raise ValueError("MoGe camera recovery produced an invalid focal estimate")
    return focal, shift


def postprocess(points: np.ndarray, normals: np.ndarray, mask_probability: np.ndarray,
                metric_scale: np.ndarray) -> dict[str, np.ndarray | float]:
    """Convert MoGe raw forward outputs to the full camera-space geometry."""
    points = np.asarray(points, dtype=np.float32)
    normals = np.asarray(normals, dtype=np.float32)
    mask_probability = np.asarray(mask_probability, dtype=np.float32)
    if points.ndim != 3 or points.shape[-1] != 3:
        raise ValueError(f"MoGe point map must have shape HxWx3, got {points.shape}")
    if normals.shape != points.shape or mask_probability.shape != points.shape[:2]:
        raise ValueError("MoGe normals and mask dimensions do not match point map")
    scale = float(np.asarray(metric_scale, dtype=np.float32).reshape(-1)[0])
    if not math.isfinite(scale) or scale <= 0:
        raise ValueError("MoGe metric scale must be a finite positive value")

    mask = (mask_probability > 0.5) & np.isfinite(points).all(axis=-1)
    focal, shift = recover_focal_shift(points, mask)
    height, width = points.shape[:2]
    aspect = width / height
    diagonal_ratio = math.sqrt(1.0 + aspect * aspect)
    # These are normalized-image focal lengths, matching MoGe's use of
    # intrinsics_from_focal_center rather than pixel-unit OpenCV K directly.
    fx_normalized = focal * diagonal_ratio / (2.0 * aspect)
    fy_normalized = focal * diagonal_ratio / 2.0
    intrinsics = np.array([[fx_normalized, 0.0, 0.5], [0.0, fy_normalized, 0.5], [0.0, 0.0, 1.0]], dtype=np.float32)

    depth = (points[..., 2] + np.float32(shift)) * np.float32(scale)
    valid = mask & np.isfinite(depth) & (depth > 0)
    # Construct a projection-consistent point map from depth and calibrated K.
    px = (np.arange(width, dtype=np.float32) + 0.5) / width
    py = (np.arange(height, dtype=np.float32) + 0.5) / height
    xx, yy = np.meshgrid(px, py, indexing="xy")
    camera_points = np.stack(((xx - 0.5) * depth / fx_normalized,
                              (yy - 0.5) * depth / fy_normalized,
                              depth), axis=-1).astype(np.float32)
    camera_points[~valid] = np.nan
    depth[~valid] = np.nan
    unit_normals = normals / np.maximum(np.linalg.norm(normals, axis=-1, keepdims=True), 1e-12)
    unit_normals[~np.isfinite(unit_normals).all(axis=-1)] = 0.0
    unit_normals[~valid] = 0.0
    return {
        "points": camera_points,
        "depth": depth.astype(np.float32),
        "normals": unit_normals.astype(np.float32),
        "validity": valid.astype(np.uint8),
        "intrinsics": intrinsics,
        "metric_scale": scale,
        "focal_relative_half_diagonal": focal,
        "shift_before_metric_scale": shift,
    }


def fit_support_plane(points: np.ndarray, validity: np.ndarray, roi: dict,
                      threshold_m: float = 0.01, normals: np.ndarray | None = None,
                      trials: int = 512) -> tuple[dict, np.ndarray]:
    """Fit a robust plane in an explicit normalized ROI; all distances are estimates.

    A deterministic RANSAC seed makes repeated jobs reproducible. The returned
    rejected mask is one for pixels inside the ROI that are invalid or excluded
    from the best consensus, and zero outside the requested ROI.
    """
    points = np.asarray(points, dtype=np.float32)
    validity = np.asarray(validity, dtype=bool)
    if points.ndim != 3 or points.shape[-1] != 3 or validity.shape != points.shape[:2]:
        raise ValueError("support-plane points and validity dimensions do not match")
    if not isinstance(roi, dict):
        raise ValueError("support_roi must be an object with normalized x, y, width, height")
    try:
        x, y, width, height = (float(roi[key]) for key in ("x", "y", "width", "height"))
    except (KeyError, TypeError, ValueError) as exc:
        raise ValueError("support_roi must contain numeric x, y, width, height") from exc
    if (not np.isfinite([x, y, width, height]).all() or x < 0 or y < 0 or width <= 0 or height <= 0
            or x + width > 1 or y + height > 1):
        raise ValueError("support_roi must be a non-empty normalized rectangle within the image")
    if not math.isfinite(threshold_m) or threshold_m <= 0 or threshold_m > 0.25:
        raise ValueError("support plane threshold must be in (0, 0.25] estimated meters")
    h, w = validity.shape
    ys, xs = np.mgrid[:h, :w]
    u, v = (xs + 0.5) / w, (ys + 0.5) / h
    roi_mask = (u >= x) & (u < x + width) & (v >= y) & (v < y + height)
    candidates = roi_mask & validity & np.isfinite(points).all(axis=-1)
    q = points[candidates].astype(np.float64)
    if normals is not None and np.asarray(normals).shape != points.shape:
        raise ValueError("support-plane normals dimensions do not match points")
    if len(q) < 16:
        raise ValueError(f"support ROI has only {len(q)} valid geometry samples; at least 16 are required")

    rng = np.random.default_rng(0)
    hypothesis_q = q
    if len(q) > 50_000:
        hypothesis_q = q[np.sort(rng.choice(len(q), 50_000, replace=False))]
    best = None
    for _ in range(max(32, min(int(trials), 4096))):
        sample = hypothesis_q[rng.choice(len(hypothesis_q), 3, replace=False)]
        normal = np.cross(sample[1] - sample[0], sample[2] - sample[0])
        norm = float(np.linalg.norm(normal))
        if norm < 1e-10:
            continue
        normal /= norm
        offset = -float(normal @ sample[0])
        residual = np.abs(hypothesis_q @ normal + offset)
        inliers = residual <= threshold_m
        count = int(inliers.sum())
        if count < 16:
            continue
        score = (count, -float(np.median(residual[inliers])))
        if best is None or score > best[0]:
            best = (score, inliers)
    if best is None:
        raise ValueError("support ROI did not contain a stable plane consensus")

    first_sample = hypothesis_q[best[1]]
    center = np.mean(first_sample, axis=0)
    _u, _s, vh = np.linalg.svd(first_sample - center, full_matrices=False)
    normal = vh[-1]
    offset = -float(normal @ center)
    inliers = np.abs(q @ normal + offset) <= threshold_m
    for _ in range(4):
        selected = q[inliers]
        center = np.mean(selected, axis=0)
        _u, _s, vh = np.linalg.svd(selected - center, full_matrices=False)
        normal = vh[-1]
        offset = -float(normal @ center)
        residual = np.abs(q @ normal + offset)
        refined = residual <= threshold_m
        if int(refined.sum()) < 16 or np.array_equal(refined, inliers):
            break
        inliers = refined
    selected = q[inliers]
    center = np.mean(selected, axis=0)
    _u, _s, vh = np.linalg.svd(selected - center, full_matrices=False)
    normal = vh[-1]
    offset = -float(normal @ center)
    if normals is not None:
        normal_samples = np.asarray(normals, dtype=np.float32)[candidates][inliers]
        mean_normal = np.mean(normal_samples, axis=0)
        if np.linalg.norm(mean_normal) > 1e-6 and float(normal @ mean_normal) < 0:
            normal, offset = -normal, -offset
    residual = np.abs(q @ normal + offset)
    inlier_residual = residual[inliers]
    rejected = np.zeros((h, w), dtype=np.uint8)
    selected_pixels = np.zeros((h, w), dtype=bool)
    selected_pixels[candidates] = inliers
    rejected[roi_mask & ~selected_pixels] = 1
    yy, xx = np.where(roi_mask)
    plane = {
        "method": "deterministic_ransac_pca",
        "roi_normalized": {"x": x, "y": y, "width": width, "height": height},
        "roi_pixels_exclusive_bounds": {"x0": int(xx.min()), "y0": int(yy.min()),
                                        "x1": int(xx.max()) + 1, "y1": int(yy.max()) + 1},
        "normal_opencv_camera": normal.astype(np.float32).tolist(),
        "offset_m_estimate": float(offset),
        "distance_threshold_m_estimate": float(threshold_m),
        "selected_valid_points": int(len(q)),
        "inlier_points": int(inliers.sum()),
        "rejected_roi_pixels": int(rejected.sum()),
        "inlier_fraction": float(inliers.mean()),
        "inlier_absolute_residual_m_estimate_quantiles": np.quantile(inlier_residual, [0.5, 0.9, 0.95, 1.0]).astype(float).tolist(),
        "metric_status": "estimated_monocular_scale_not_measured",
    }
    return plane, rejected


def _as_hwc(np, value: np.ndarray, channels: int) -> np.ndarray:
    arr = np.asarray(value, dtype=np.float32)
    if arr.ndim == 4:
        arr = arr[0]
    if arr.ndim == 3 and arr.shape[0] == channels and arr.shape[-1] != channels:
        arr = arr.transpose(1, 2, 0)
    if arr.ndim != 3 or arr.shape[-1] != channels:
        raise ValueError(f"unexpected MoGe output shape {arr.shape}; expected HxWx{channels}")
    return arr


def _sha256_file(path: str) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def write_geometry_archive(path: str, geometry: dict, manifest: dict) -> None:
    """Write and validate a self-describing ZIP of NPY arrays."""
    array_names = ("points", "depth", "normals", "validity", "intrinsics")
    if "support_plane_rejected" in geometry:
        array_names += ("support_plane_rejected",)
    uncompressed_bytes = sum(np.asarray(geometry[name]).nbytes for name in array_names)
    if uncompressed_bytes > MAX_ARRAY_BYTES:
        raise ValueError(f"geometry arrays exceed the {MAX_ARRAY_BYTES}-byte archive limit")
    os.makedirs(os.path.dirname(os.path.abspath(path)), exist_ok=True)
    with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
        for name in array_names:
            buffer = io.BytesIO()
            np.save(buffer, geometry[name], allow_pickle=False)
            archive.writestr(f"{name}.npy", buffer.getvalue())
        archive.writestr("manifest.json", json.dumps(manifest, sort_keys=True, indent=2).encode("utf-8"))
    validate_geometry_archive(path)


def _read_geometry_array(raw: bytes, name: str, spec: dict) -> np.ndarray:
    """Reject oversized/foreign NPY headers before numpy can allocate an array."""
    stream = io.BytesIO(raw)
    version = np.lib.format.read_magic(stream)
    if version == (1, 0):
        shape, fortran, dtype = np.lib.format.read_array_header_1_0(stream, max_header_size=4096)
    elif version == (2, 0):
        shape, fortran, dtype = np.lib.format.read_array_header_2_0(stream, max_header_size=4096)
    else:
        raise ValueError("geometry NPY version must be 1.0 or 2.0")
    expected_dtype = np.dtype(np.uint8 if name in ("validity", "support_plane_rejected") else np.float32)
    if dtype != expected_dtype or dtype.hasobject or fortran:
        raise ValueError(f"geometry array {name} requires native float32/uint8 C-order data")
    if (not shape or len(shape) > 3 or any(type(size) is not int or size <= 0 for size in shape)
            or list(shape) != spec.get("shape") or str(dtype) != spec.get("dtype")):
        raise ValueError(f"geometry array {name} does not match its manifest shape/dtype")
    data_bytes = math.prod(shape) * dtype.itemsize
    if data_bytes > MAX_ARRAY_BYTES or data_bytes != len(raw) - stream.tell():
        raise ValueError(f"geometry array {name} byte count exceeds bounds or disagrees with its header")
    return np.load(io.BytesIO(raw), allow_pickle=False)


def validate_geometry_archive(path: str) -> dict:
    """Read back the entire archive and enforce its public artifact contract."""
    expected = {"points.npy", "depth.npy", "normals.npy", "validity.npy", "intrinsics.npy", "manifest.json"}
    if os.path.getsize(path) > MAX_ARCHIVE_BYTES:
        raise ValueError(f"geometry archive exceeds {MAX_ARCHIVE_BYTES} bytes")
    with zipfile.ZipFile(path, "r") as archive:
        members = archive.infolist()
        names = set(archive.namelist())
        if len(members) != len(names) or names not in (expected, expected | {"support_plane_rejected.npy"}):
            raise ValueError("geometry archive has missing, unexpected, or corrupt members")
        if sum(member.file_size for member in members) > MAX_ARRAY_BYTES + 1024 * 1024:
            raise ValueError("geometry archive uncompressed members exceed the size limit")
        if archive.getinfo("manifest.json").file_size > 64 * 1024:
            raise ValueError("geometry manifest exceeds 64 KiB")
        if archive.testzip() is not None:
            raise ValueError("geometry archive contains corrupt members")
        manifest = json.loads(archive.read("manifest.json"))
        if manifest.get("support_plane") is not None:
            expected.add("support_plane_rejected.npy")
        if set(archive.namelist()) != expected:
            raise ValueError("geometry archive members do not match optional support-plane declaration")
        if manifest.get("format") != "vrooli.image-tools.scene-geometry.v1":
            raise ValueError("geometry archive manifest format is missing or unsupported")
        if manifest.get("coordinates", {}).get("frame") != "OpenCV camera":
            raise ValueError("geometry archive manifest must declare the OpenCV camera frame")
        if manifest.get("units", {}).get("scale_kind") != "estimated_metric":
            raise ValueError("geometry archive must mark metric scale as an estimate")
        arrays = {}
        array_names = ["points", "depth", "normals", "validity", "intrinsics"]
        if manifest.get("support_plane") is not None:
            array_names.append("support_plane_rejected")
        for name in array_names:
            spec = manifest.get("arrays", {}).get(f"{name}.npy", {})
            arrays[name] = _read_geometry_array(archive.read(f"{name}.npy"), name, spec)
    points, depth, normals, valid, intrinsics = (arrays[key] for key in ("points", "depth", "normals", "validity", "intrinsics"))
    if (points.ndim != 3 or points.shape[-1] != 3 or depth.shape != points.shape[:2]
            or normals.shape != points.shape or valid.shape != depth.shape or intrinsics.shape != (3, 3)):
        raise ValueError("geometry archive arrays have inconsistent dimensions")
    if valid.dtype != np.uint8 or not np.isin(valid, (0, 1)).all():
        raise ValueError("geometry validity array must be uint8 values 0 or 1")
    if "support_plane_rejected" in arrays:
        rejected = arrays["support_plane_rejected"]
        plane = manifest.get("support_plane", {})
        if rejected.shape != valid.shape or rejected.dtype != np.uint8 or not np.isin(rejected, (0, 1)).all():
            raise ValueError("support-plane rejected-pixel mask must be uint8 HxW values 0 or 1")
        if int(rejected.sum()) != int(plane.get("rejected_roi_pixels", -1)):
            raise ValueError("support-plane rejected-pixel count does not match manifest")
        if not np.isfinite(np.asarray(plane.get("normal_opencv_camera"), dtype=np.float64)).all():
            raise ValueError("support-plane normal is missing or non-finite")
        if plane.get("metric_status") != "estimated_monocular_scale_not_measured":
            raise ValueError("support-plane fit must mark its metric values as estimates")
    if not np.isfinite(intrinsics).all() or not np.allclose(intrinsics[2], [0.0, 0.0, 1.0]):
        raise ValueError("geometry intrinsics matrix is non-finite or malformed")
    if np.any(valid > 0):
        selected = valid > 0
        if not np.isfinite(depth[selected]).all() or np.any(depth[selected] <= 0) or not np.isfinite(points[selected]).all():
            raise ValueError("valid geometry samples must have finite, positive depth and finite points")
        if not np.allclose(points[selected][:, 2], depth[selected], atol=2e-5, rtol=2e-5):
            raise ValueError("geometry point Z must agree with the depth array")
        height, width = depth.shape
        ys, xs = np.nonzero(selected)
        u = (xs.astype(np.float64) + 0.5) / width
        v = (ys.astype(np.float64) + 0.5) / height
        fx, fy, cx, cy = float(intrinsics[0, 0]), float(intrinsics[1, 1]), float(intrinsics[0, 2]), float(intrinsics[1, 2])
        expected_x = (u - cx) * depth[selected] / fx
        expected_y = (v - cy) * depth[selected] / fy
        if (not np.allclose(points[selected, 0], expected_x, atol=2e-5, rtol=2e-5)
                or not np.allclose(points[selected, 1], expected_y, atol=2e-5, rtol=2e-5)):
            raise ValueError("geometry points do not reproject through the declared intrinsics")
    if not np.isfinite(normals).all():
        raise ValueError("geometry normals must be finite")
    valid_norms = np.linalg.norm(normals[valid > 0], axis=-1)
    if valid_norms.size and not np.allclose(valid_norms, 1.0, atol=2e-3):
        raise ValueError("valid geometry normals must be unit length")
    return manifest


def run_model(model_path: str, image_path: str, out_path: str, model_id: str, revision: str,
              support_roi: dict | None = None) -> None:
    np, ort, Image = _common.require_deps()
    oriented, source_raster = _common.load_oriented_image(image_path, MAX_PIXELS)
    source_width, source_height = oriented.size
    scale_to_limit = min(1.0, 1024.0 / max(source_width, source_height))
    infer_width = max(32, int(round(source_width * scale_to_limit)))
    infer_height = max(32, int(round(source_height * scale_to_limit)))
    model_image = oriented.resize((infer_width, infer_height), Image.Resampling.LANCZOS)
    image_tensor = np.asarray(model_image, dtype=np.float32) / np.float32(255.0)
    image_tensor = image_tensor.transpose(2, 0, 1)[None, ...].astype(np.float32)

    options = ort.SessionOptions()
    options.intra_op_num_threads = max(1, min(8, os.cpu_count() or 1))
    session = ort.InferenceSession(model_path, sess_options=options, providers=["CPUExecutionProvider"])
    feed = {}
    for tensor in session.get_inputs():
        if tensor.name == "image" or len(tensor.shape) == 4:
            feed[tensor.name] = image_tensor
        elif "token" in tensor.name.lower():
            feed[tensor.name] = np.asarray(1800, dtype=np.int64)
        else:
            raise ValueError(f"unsupported MoGe ONNX input {tensor.name!r}")
    outputs = dict(zip((item.name for item in session.get_outputs()), session.run(None, feed)))
    # The official docs name this output ``metric_scale``; the published ViT-S
    # export at its pinned revision exposes the same final exp-head as ``scale``.
    if "metric_scale" not in outputs and "scale" in outputs:
        outputs["metric_scale"] = outputs["scale"]
    required = {"points", "normal", "mask", "metric_scale"}
    if not required.issubset(outputs):
        raise ValueError(f"MoGe ONNX outputs missing {sorted(required - outputs.keys())}; found {sorted(outputs)}")
    raw_points = _as_hwc(np, outputs["points"], 3)
    raw_normals = _as_hwc(np, outputs["normal"], 3)
    mask = np.asarray(outputs["mask"], dtype=np.float32).squeeze()
    if raw_points.shape[:2] != (infer_height, infer_width) or mask.shape != (infer_height, infer_width):
        raise ValueError(f"MoGe output raster mismatch: points {raw_points.shape}, mask {mask.shape}, expected {(infer_height, infer_width)}")
    if not np.isfinite(mask).all() or not np.isfinite(raw_points).all() or not np.isfinite(raw_normals).all():
        raise ValueError("MoGe produced non-finite raw predictions")
    geometry = postprocess(raw_points, raw_normals, mask, outputs["metric_scale"])
    support_plane = None
    if support_roi is not None:
        support_plane, rejected = fit_support_plane(
            geometry["points"], geometry["validity"], support_roi,
            normals=geometry["normals"],
        )
        geometry["support_plane_rejected"] = rejected

    source_raster["sha256"] = _sha256_file(image_path)
    manifest = {
        "format": "vrooli.image-tools.scene-geometry.v1",
        "model": {"id": model_id, "checkpoint_revision": revision, "runtime": "onnxruntime-cpu",
                  "checkpoint_sha256": _sha256_file(model_path), "license": "MIT",
                  "source": "https://huggingface.co/Ruicheng/moge-2-vits-normal"},
        "input": source_raster,
        "raster": {"width": infer_width, "height": infer_height,
                   "pixel_centers": True, "source_to_geometry_scale": [infer_width / source_width, infer_height / source_height]},
        "coordinates": {"frame": "OpenCV camera", "x": "right", "y": "down", "z": "forward",
                        "intrinsics": "normalized by width for row 1 and height for row 2; principal point (0.5, 0.5)",
                        "array_origin": "top-left; samples at pixel centers"},
        "units": {"depth": "meter estimate from MoGe metric-scale head; not measured calibration",
                  "points": "meter estimate from MoGe metric-scale head; not measured calibration",
                  "scale_kind": "estimated_metric", "metric_scale": geometry["metric_scale"],
                  "uncertainty": "monocular prediction; scene scale and intrinsics are estimates, not ground truth"},
        "camera_recovery": {"focal_relative_half_diagonal": geometry["focal_relative_half_diagonal"],
                            "z_shift_before_metric_scale": geometry["shift_before_metric_scale"],
                            "postprocess": "MoGe-2 infer focal/shift recovery and pinhole reprojection"},
        "arrays": {
            "points.npy": {"dtype": "float32", "shape": list(geometry["points"].shape), "meaning": "camera-space XYZ; invalid samples are NaN"},
            "depth.npy": {"dtype": "float32", "shape": list(geometry["depth"].shape), "meaning": "camera Z; invalid samples are NaN"},
            "normals.npy": {"dtype": "float32", "shape": list(geometry["normals"].shape), "meaning": "unit surface normal in camera frame; invalid samples are zero"},
            "validity.npy": {"dtype": "uint8", "shape": list(geometry["validity"].shape), "meaning": "1 for mask and positive finite depth; 0 otherwise"},
            "intrinsics.npy": {"dtype": "float32", "shape": [3, 3], "meaning": "normalized camera K"},
        },
        "archive_readme": "NPY arrays are top-level ZIP members and can be loaded with numpy.load. validity.npy is authoritative for samples.",
    }
    if support_plane is not None:
        manifest["arrays"]["support_plane_rejected.npy"] = {
            "dtype": "uint8", "shape": list(geometry["support_plane_rejected"].shape),
            "meaning": "1 for invalid or RANSAC-rejected pixels inside support ROI; 0 outside ROI or for inliers",
        }
        manifest["support_plane"] = support_plane
    write_geometry_archive(out_path, geometry, manifest)


def main() -> None:
    parser = argparse.ArgumentParser(description="MoGe-2 full scene geometry inference")
    parser.add_argument("--model", required=True)
    parser.add_argument("--image", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--model-id", default="moge-2-vits-normal")
    parser.add_argument("--revision", default="e50ffda41565591092adea54c6ac83d6212e1e23")
    parser.add_argument("--support-roi", help='optional normalized JSON rectangle: {"x":0,"y":0.8,"width":1,"height":0.2}')
    args = parser.parse_args()
    try:
        support_roi = json.loads(args.support_roi) if args.support_roi else None
        run_model(args.model, args.image, args.out, args.model_id, args.revision, support_roi)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"scene geometry inference failed: {exc}", code=8)


if __name__ == "__main__":
    main()
