"""Pinned TripoSR single-image prop reconstruction and inspected GLB archive.

Foreground comes only from an explicit mask or source alpha. Opaque images with
no mask use a documented full-frame assumption; this module never downloads a
background remover. The upstream model's hidden surfaces and scale are inferred.
"""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import math
import os
from pathlib import Path
import resource
import sys
import zipfile

import numpy as np

MAX_PIXELS = 16_000_000
MAX_GLTF_BYTES = 120 * 1024 * 1024
MAX_ARCHIVE_BYTES = 128 * 1024 * 1024
FOREGROUND_RATIO = 0.85
SOURCE_COMMIT = "107cefdc244c39106fa830359024f6a2f1c78871"
WEIGHTS_REVISION = "5b521936b01fbe1890f6f9baed0254ab6351c04a"


def _sha256(path: str | Path) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def prepare_foreground(Image, image_path: str, mask_path: str | None = None):
    """Return a square gray-composited PIL image and source/mask provenance."""
    from . import _common

    rgba, source_raster = _common.load_oriented_image(image_path, MAX_PIXELS, mode="RGBA")
    width, height = rgba.size
    rgba_arr = np.asarray(rgba, dtype=np.uint8)
    source_alpha = rgba_arr[..., 3]
    explicit = mask_path is not None
    mask_raster = None
    if explicit:
        mask, mask_raster = _common.load_oriented_image(mask_path, MAX_PIXELS, mode="L")
        if mask.size != (width, height):
            raise ValueError("explicit foreground mask dimensions must match the oriented source image")
        mask_arr = np.asarray(mask, dtype=np.uint8)
        alpha = np.minimum(source_alpha, mask_arr)
        foreground_source = "explicit_mask_and_source_alpha" if int(source_alpha.min()) < 255 else "explicit_mask"
    elif int(source_alpha.min()) < 255:
        alpha = source_alpha
        foreground_source = "source_alpha"
    else:
        alpha = np.full((height, width), 255, dtype=np.uint8)
        foreground_source = "full_frame_assumption"

    foreground = alpha >= 128
    if not foreground.any():
        raise ValueError("foreground mask/alpha contains no foreground pixels")
    ys, xs = np.nonzero(foreground)
    x0, x1 = int(xs.min()), int(xs.max()) + 1
    y0, y1 = int(ys.min()), int(ys.max()) + 1
    cropped_rgb = rgba_arr[y0:y1, x0:x1, :3].astype(np.float32) / 255.0
    cropped_alpha = alpha[y0:y1, x0:x1].astype(np.float32) / 255.0
    composite = cropped_rgb * cropped_alpha[..., None] + (1.0 - cropped_alpha[..., None]) * 0.5
    crop_h, crop_w = composite.shape[:2]
    side = max(crop_w, crop_h)
    square = np.full((side, side, 3), 0.5, dtype=np.float32)
    ox, oy = (side - crop_w) // 2, (side - crop_h) // 2
    square[oy:oy + crop_h, ox:ox + crop_w] = composite
    target = max(1, int(math.ceil(side / FOREGROUND_RATIO)))
    canvas = np.full((target, target, 3), 0.5, dtype=np.float32)
    xoff, yoff = (target - side) // 2, (target - side) // 2
    canvas[yoff:yoff + side, xoff:xoff + side] = square
    prepared = Image.fromarray(np.clip(canvas * 255.0, 0, 255).astype(np.uint8), "RGB")
    provenance = {
        "foreground_source": foreground_source,
        "source_dimensions": {"width": width, "height": height},
        "source_raster": source_raster,
        "mask_raster": mask_raster,
        "foreground_bbox_xyxy_pixels": [x0, y0, x1, y1],
        "foreground_ratio": FOREGROUND_RATIO,
        "prepared_dimensions": {"width": target, "height": target},
        "mesh_scale": "model-normalized; no metric scale is available from one image",
    }
    return prepared, provenance


def _mesh_inspection(mesh, glb: bytes) -> dict:
    vertices = np.asarray(mesh.vertices, dtype=np.float64)
    faces = np.asarray(mesh.faces)
    if vertices.ndim != 2 or vertices.shape[1] != 3 or not np.isfinite(vertices).all():
        raise ValueError("TripoSR generated non-finite or malformed vertices")
    if faces.ndim != 2 or faces.shape[1] != 3 or len(faces) < 4:
        raise ValueError("TripoSR generated an empty or non-triangle mesh")
    if len(vertices) > 1_500_000 or len(faces) > 3_000_000:
        raise ValueError("generated mesh exceeds the 1.5M vertex / 3M triangle inspection budget")
    if int(faces.min()) < 0 or int(faces.max()) >= len(vertices):
        raise ValueError("generated mesh contains an out-of-range face index")
    bounds = np.asarray(mesh.bounds, dtype=np.float64)
    extents = np.asarray(mesh.extents, dtype=np.float64)
    if bounds.shape != (2, 3) or extents.shape != (3,) or not np.isfinite(bounds).all() or not np.isfinite(extents).all() or np.any(extents <= 0):
        raise ValueError("generated mesh has invalid bounds")
    colors = getattr(mesh.visual, "vertex_colors", None)
    has_vertex_colors = colors is not None and np.asarray(colors).shape[0] == len(vertices)
    components = [type(mesh)(vertices=vertices, faces=faces[indices], process=False) for indices in _face_components(mesh)]
    closed_components = [component for component in components if component.is_watertight]
    open_components = len(components) - len(closed_components)
    closed_outward = bool(closed_components) and all(float(component.volume) > 0 and component.is_winding_consistent for component in closed_components)
    return {
        "glb_bytes": len(glb),
        "glb_sha256": hashlib.sha256(glb).hexdigest(),
        "vertex_count": int(len(vertices)),
        "triangle_count": int(len(faces)),
        "bounds_model_units": bounds.tolist(),
        "extents_model_units": extents.tolist(),
        "watertight": open_components == 0,
        "winding_consistent": bool(mesh.is_winding_consistent),
        "connected_components": len(components),
        "closed_component_count": len(closed_components),
        "open_component_count": open_components,
        "closed_components_outward": closed_outward,
        "vertex_colors": bool(has_vertex_colors),
        "uv_texture_coordinates": False,
        "volume_model_units_cubed": float(mesh.volume) if open_components == 0 else None,
        "hidden_surfaces": "model-generated from one view; inspect before physical use",
        "metric_scale": "unavailable; model-normalized bounds are not measurements",
    }


def _ensure_outward_winding(mesh) -> None:
    """Normalize closed-mesh orientation to positive signed volume.

    TripoSR's isosurface backend can return a consistently inward mesh even
    when every adjacent face agrees. For a closed triangle mesh, reverse all
    faces in that case; local winding consistency alone does not establish
    outward normals.
    """
    faces = np.asarray(mesh.faces).copy()
    changed = False
    for face_indices in _face_components(mesh):
        component = type(mesh)(vertices=mesh.vertices, faces=faces[face_indices], process=False)
        if component.is_watertight and not math.isfinite(float(component.volume)):
            raise ValueError("watertight TripoSR component has a non-finite signed volume")
        if component.is_watertight and float(component.volume) < 0.0:
            faces[face_indices] = faces[face_indices][:, [0, 2, 1]]
            changed = True
    if changed:
        mesh.faces = faces


def _face_components(mesh) -> list[np.ndarray]:
    """Return face-index components joined by shared edges."""
    count = len(mesh.faces)
    parent = list(range(count))

    def find(value: int) -> int:
        while parent[value] != value:
            parent[value] = parent[parent[value]]
            value = parent[value]
        return value

    adjacency = np.asarray(mesh.face_adjacency, dtype=np.int64).reshape(-1, 2)
    for left, right in adjacency:
        a, b = find(int(left)), find(int(right))
        if a != b:
            parent[b] = a
    groups: dict[int, list[int]] = {}
    for face in range(count):
        groups.setdefault(find(face), []).append(face)
    return [np.asarray(indices, dtype=np.int64) for indices in groups.values()]


def build_archive(mesh, manifest: dict) -> bytes:
    _ensure_outward_winding(mesh)
    glb = mesh.export(file_type="glb")
    if not isinstance(glb, bytes) or not glb or len(glb) > MAX_GLTF_BYTES:
        raise ValueError("generated GLB is empty or exceeds the 120 MiB limit")
    manifest = dict(manifest)
    manifest["mesh_inspection"] = _mesh_inspection(mesh, glb)
    manifest_bytes = json.dumps(manifest, sort_keys=True, separators=(",", ":"), allow_nan=False).encode("utf-8")
    output = io.BytesIO()
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
        archive.writestr("model.glb", glb)
        archive.writestr("manifest.json", manifest_bytes)
    data = output.getvalue()
    if len(data) > MAX_ARCHIVE_BYTES:
        raise ValueError("image_to_object ZIP exceeds 128 MiB")
    return data


def validate_archive(data: bytes) -> dict:
    """Validate the public v1 ZIP envelope without extracting files to disk."""
    if not data or len(data) > MAX_ARCHIVE_BYTES or not data.startswith(b"PK\x03\x04"):
        raise ValueError("invalid or oversized image_to_object ZIP archive")
    try:
        with zipfile.ZipFile(io.BytesIO(data), "r") as archive:
            infos = archive.infolist()
            if {info.filename for info in infos} != {"model.glb", "manifest.json"} or len(infos) != 2:
                raise ValueError("image_to_object ZIP must contain only model.glb and manifest.json")
            if any(info.file_size > MAX_GLTF_BYTES for info in infos):
                raise ValueError("image_to_object ZIP member exceeds its size limit")
            glb = archive.read("model.glb")
            raw_manifest = archive.read("manifest.json")
            if not glb.startswith(b"glTF") or len(glb) < 20:
                raise ValueError("image_to_object model.glb has an invalid GLB signature")
            manifest = json.loads(raw_manifest)
            if not isinstance(manifest, dict) or manifest.get("schema_version") != 1:
                raise ValueError("image_to_object manifest schema_version must be 1")
            inspection = manifest.get("mesh_inspection")
            if not isinstance(inspection, dict) or inspection.get("glb_sha256") != hashlib.sha256(glb).hexdigest():
                raise ValueError("image_to_object manifest mesh inspection hash does not match model.glb")
            if inspection.get("uv_texture_coordinates") is not False or inspection.get("metric_scale") != "unavailable; model-normalized bounds are not measurements":
                raise ValueError("image_to_object mesh disclosure fields are missing or inconsistent")
            runtime = manifest.get("runtime")
            if isinstance(runtime, dict) and str(runtime.get("device", "")).startswith("cuda"):
                allocated = runtime.get("cuda_peak_allocated_bytes")
                reserved = runtime.get("cuda_peak_reserved_bytes")
                if (
                    not isinstance(runtime.get("gpu_uuid"), str)
                    or not runtime["gpu_uuid"].strip()
                    or not isinstance(runtime.get("worker_pid"), int)
                    or runtime["worker_pid"] <= 1
                    or not isinstance(runtime.get("process_peak_rss_bytes"), int)
                    or runtime["process_peak_rss_bytes"] <= 0
                    or not isinstance(allocated, int)
                    or not isinstance(reserved, int)
                    or allocated < 0
                    or reserved < allocated
                ):
                    raise ValueError("CUDA image_to_object archive is missing worker/GPU memory measurements")
            if int(inspection.get("vertex_count", 0)) <= 0 or int(inspection.get("triangle_count", 0)) <= 0:
                raise ValueError("image_to_object mesh counts must be positive")
            return manifest
    except (zipfile.BadZipFile, json.JSONDecodeError, KeyError, TypeError, ValueError) as exc:
        if isinstance(exc, ValueError):
            raise
        raise ValueError(f"invalid image_to_object ZIP archive: {exc}") from exc


def _load_source_profile(source_dir: str):
    code_dir = Path(source_dir)
    profile_path = code_dir.parent.parent / "source-profile.json"
    if not profile_path.is_file():
        raise RuntimeError("pinned TripoSR source profile is missing; install it through the governed host-tool route")
    try:
        profile = json.loads(profile_path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise RuntimeError("pinned TripoSR source profile is invalid") from exc
    if profile.get("schema_version") != 2 or profile.get("upstream_commit") != SOURCE_COMMIT or profile.get("adapter_version") != "vrooli-triposr-config-skimage-v6":
        raise RuntimeError("installed TripoSR source profile identity does not match the pinned commit")
    if not (code_dir / "tsr" / "system.py").is_file():
        raise RuntimeError("pinned TripoSR source profile is incomplete")
    sys.path.insert(0, str(code_dir))
    try:
        from tsr.system import TSR
    except ImportError as exc:
        raise RuntimeError(f"pinned TripoSR source or runtime dependency is unavailable: {exc}") from exc
    return TSR, profile


def _parse_args():
    parser = argparse.ArgumentParser(description="Pinned TripoSR prop reconstruction")
    parser.add_argument("--model", required=True, help="Installed model asset directory")
    parser.add_argument("--source-dir", required=True, help="Governed TripoSR source profile directory")
    parser.add_argument("--image", required=True)
    parser.add_argument("--mask", default=None)
    parser.add_argument("--out", required=True)
    parser.add_argument("--device", choices=("cpu", "cuda"), default="cpu")
    parser.add_argument("--mesh-resolution", type=int, choices=(128, 256), default=256)
    parser.add_argument("--chunk-size", type=int, default=8192)
    return parser.parse_args()


def main() -> None:
    import torch
    import trimesh
    from PIL import Image

    args = _parse_args()
    if args.device == "cuda" and not torch.cuda.is_available():
        raise RuntimeError("image_to_object selected CUDA but no CUDA runtime is available")
    source_path = Path(args.model)
    for required in ("config.yaml", "model.ckpt"):
        if not (source_path / required).is_file():
            raise RuntimeError(f"installed TripoSR model is missing {required}")
    TSR, source_profile = _load_source_profile(args.source_dir)
    image, prep = prepare_foreground(Image, args.image, args.mask)
    device = "cuda:0" if args.device == "cuda" else "cpu"
    # Only the exact pinned local model files are used; there is no model/network
    # lookup path in this execution handler.
    rss_before = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
    if device.startswith("cuda"):
        # The engine sets CUDA_VISIBLE_DEVICES to the broker-selected UUID. That
        # device is logical CUDA 0 in this isolated worker; use PyTorch's current
        # device here because passing the pre-visibility device string can be
        # rejected by some CUDA/PyTorch builds.
        torch.cuda.reset_peak_memory_stats()
    model = TSR.from_pretrained(str(source_path), config_name="config.yaml", weight_name="model.ckpt")
    model.renderer.set_chunk_size(args.chunk_size)
    model.to(device)
    if device.startswith("cuda"):
        torch.cuda.synchronize()
    with torch.inference_mode():
        codes = model([image], device=device)
        meshes = model.extract_mesh(codes, has_vertex_color=True, resolution=args.mesh_resolution)
    if device.startswith("cuda"):
        torch.cuda.synchronize()
    if len(meshes) != 1:
        raise RuntimeError("TripoSR returned an unexpected mesh count")
    mesh = meshes[0]
    if not isinstance(mesh, trimesh.Trimesh):
        raise RuntimeError("TripoSR did not produce a single triangle mesh")
    manifest = {
        "schema_version": 1,
        "operation": "image_to_object",
        "model": {
            "id": "triposr-vision-to-mesh",
            "repo": "stabilityai/TripoSR",
            "revision": WEIGHTS_REVISION,
            "config_sha256": _sha256(source_path / "config.yaml"),
            "checkpoint_sha256": _sha256(source_path / "model.ckpt"),
            "license": "MIT",
        },
        "source_code": {
            "repo": "https://github.com/VAST-AI-Research/TripoSR",
            "commit": SOURCE_COMMIT,
            "license": "MIT",
            "source_profile": source_profile,
        },
        "runtime": {
            "python": sys.version.split()[0],
            "torch": torch.__version__,
            "cuda_available": bool(torch.cuda.is_available()),
            "device": device,
            "gpu_uuid": os.environ.get("CUDA_VISIBLE_DEVICES") if device.startswith("cuda") else None,
            "mesh_resolution": args.mesh_resolution,
            "worker_pid": os.getpid(),
            "process_peak_rss_bytes": int(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * 1024),
            "rss_before_inference_bytes": int(rss_before * 1024),
            "cuda_peak_allocated_bytes": int(torch.cuda.max_memory_allocated()) if device.startswith("cuda") else None,
            "cuda_peak_reserved_bytes": int(torch.cuda.max_memory_reserved()) if device.startswith("cuda") else None,
        },
        "input": {"sha256": _sha256(args.image), "mask_sha256": _sha256(args.mask) if args.mask else None},
        "preprocessing": prep,
        "output": {"uv_texture": False, "material": "vertex colors"},
        "claims": {"metric_scale": False, "hidden_geometry_measured": False},
    }
    data = build_archive(mesh, manifest)
    validate_archive(data)
    Path(args.out).write_bytes(data)


if __name__ == "__main__":
    main()
