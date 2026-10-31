"""Contract tests for foreground handling and the exported GLB ZIP envelope."""

import io
import importlib.util
import json
import os
import struct
import subprocess
import zipfile
from dataclasses import dataclass, field
from pathlib import Path

import numpy as np
import pytest
from PIL import Image

from image_tools_sidecar import image_to_object as prop


class FakeVisual:
    vertex_colors = np.zeros((4, 4), dtype=np.uint8)


class FakeMesh:
    face_adjacency = np.array([[0, 1], [1, 2], [2, 3]], dtype=np.int64)

    def __init__(self, vertices=None, faces=None, process=False):
        del process
        self.vertices = np.asarray(vertices if vertices is not None else [[0, 0, 0], [1, 0, 0], [0, 1, 0], [0, 0, 1]], dtype=np.float64)
        self.faces = np.asarray(faces if faces is not None else [[0, 2, 1], [0, 1, 3], [0, 3, 2], [1, 2, 3]], dtype=np.int32)
        self.bounds = np.stack([self.vertices.min(axis=0), self.vertices.max(axis=0)])
        self.extents = self.bounds[1] - self.bounds[0]
        self.visual = FakeVisual()
        self.is_watertight = len(self.faces) == 4
        self.is_winding_consistent = True
        a, b, c = self.vertices[self.faces[:, 0]], self.vertices[self.faces[:, 1]], self.vertices[self.faces[:, 2]]
        self.volume = float(np.einsum("ij,ij->i", a, np.cross(b, c)).sum() / 6.0)

    def invert(self):
        self.faces = self.faces[:, [0, 2, 1]]
        self.volume = -self.volume

    def split(self, only_watertight=False):
        return [self]

    def export(self, file_type):
        assert file_type == "glb"
        return b"glTF" + struct.pack("<II", 2, 20) + struct.pack("<I4s", 0, b"JSON")


def _write(path, array, mode):
    Image.fromarray(array, mode=mode).save(path)


def _installed_triposr_source():
    path = Path.home() / ".vrooli/opt/triposr-source/adapter/source"
    if not (path / "tsr/config_adapter.py").is_file():
        pytest.skip("requires the governed native TripoSR source adapter profile")
    return path


def test_installed_config_adapter_strict_types_defaults_and_checkpoint_config(tmp_path):
    source = _installed_triposr_source()
    spec = importlib.util.spec_from_file_location("triposr_config_adapter_test", source / "tsr/config_adapter.py")
    adapter = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(adapter)

    @dataclass
    class Nested:
        count: int
        label: str = "default"

    @dataclass
    class Config:
        nested: Nested
        options: dict = field(default_factory=dict)

    config_path = tmp_path / "valid.yaml"
    config_path.write_text("nested:\n  count: 3\noptions:\n  enabled: true\n", encoding="utf-8")
    parsed = adapter.load_config(config_path, Config)
    assert parsed.nested.count == 3
    assert parsed.nested.label == "default"
    assert parsed.options == {"enabled": True}

    config_path.write_text("nested:\n  count: invalid\n", encoding="utf-8")
    with pytest.raises(TypeError, match="expected integer"):
        adapter.load_config(config_path, Config)
    config_path.write_text("nested:\n  count: 3\n  unexpected: true\n", encoding="utf-8")
    with pytest.raises(ValueError, match="unknown fields"):
        adapter.load_config(config_path, Config)
    config_path.write_text("nested:\n  count: 3\noptions: ${unsupported}", encoding="utf-8")
    with pytest.raises(ValueError, match="interpolation is unsupported"):
        adapter.load_config(config_path, Config)
    config_path.write_text("nested:\n  count: 3\n", encoding="utf-8")
    actual = adapter.load_config(config_path, Config)
    assert actual.nested.count == 3

    model_config = Path.home() / ".vrooli/data/vrooli/image-tools/models/triposr-vision-to-mesh/config.yaml"
    if not model_config.is_file():
        pytest.skip("requires the governed TripoSR checkpoint config")

    @dataclass
    class TSRConfig:
        cond_image_size: int
        image_tokenizer_cls: str
        image_tokenizer: dict
        tokenizer_cls: str
        tokenizer: dict
        backbone_cls: str
        backbone: dict
        post_processor_cls: str
        post_processor: dict
        decoder_cls: str
        decoder: dict
        renderer_cls: str
        renderer: dict

    loaded = adapter.load_config(model_config, TSRConfig)
    assert loaded.cond_image_size == 512
    assert loaded.backbone["in_channels"] == 1024


def test_installed_marching_cubes_preserves_xyz_and_outward_winding():
    source = _installed_triposr_source()
    interpreter = Path.home() / ".vrooli/data/vrooli/image-tools/pyenv/bin/python"
    if not interpreter.is_file():
        pytest.skip("requires the governed Image Tools Python runtime")
    code = r'''
import torch
import trimesh
from tsr.models.isosurface import MarchingCubeHelper
resolution=32
axis=torch.linspace(0.0,1.0,resolution)
x,y,z=torch.meshgrid(axis,axis,axis,indexing="ij")
center=torch.tensor([0.27,0.46,0.68])
radius=0.18
field=(x-center[0])**2+(y-center[1])**2+(z-center[2])**2-radius**2
vertices,faces=MarchingCubeHelper(resolution)(-field.reshape(-1))
assert torch.allclose(vertices.mean(dim=0),center,atol=0.015),vertices.mean(dim=0)
mesh=trimesh.Trimesh(vertices=vertices.numpy(),faces=faces.numpy(),process=False)
assert mesh.is_watertight and mesh.is_winding_consistent and mesh.volume>0,mesh.volume
triangles=vertices[faces]
normals=torch.cross(triangles[:,1]-triangles[:,0],triangles[:,2]-triangles[:,0],dim=1)
outward=(normals*(triangles.mean(dim=1)-center)).sum(dim=1)
assert torch.quantile(outward,0.05).item()>0,torch.quantile(outward,torch.tensor([.05,.5,.95]))
'''
    env = dict(os.environ, PYTHONDONTWRITEBYTECODE="1", PYTHONPATH=str(source))
    result = subprocess.run([str(interpreter), "-B", "-c", code], env=env, capture_output=True, text=True, check=False)
    assert result.returncode == 0, result.stdout + result.stderr


def test_closed_asymmetric_mesh_inward_winding_is_repaired():
    interpreter = Path.home() / ".vrooli/data/vrooli/image-tools/pyenv/bin/python"
    if not interpreter.is_file():
        pytest.skip("requires the governed Image Tools Python runtime")
    code = r'''
import numpy as np
import trimesh
from image_tools_sidecar.image_to_object import _ensure_outward_winding
mesh=trimesh.creation.icosphere(subdivisions=2, radius=1.0)
mesh.apply_scale([1.0,2.0,3.0])
mesh.apply_translation([0.7,-0.4,1.2])
mesh.invert()
assert mesh.is_watertight and mesh.is_winding_consistent and mesh.volume < 0
_ensure_outward_winding(mesh)
assert mesh.volume > 0
centers=mesh.triangles_center-mesh.center_mass
assert np.all(np.einsum("ij,ij->i",centers,mesh.face_normals)>0)
assert np.allclose(mesh.extents,[2.0,4.0,6.0],atol=0.2)
'''
    env = dict(os.environ, PYTHONDONTWRITEBYTECODE="1", PYTHONPATH=str(Path(__file__).resolve().parents[1]))
    result = subprocess.run([str(interpreter), "-B", "-c", code], env=env, capture_output=True, text=True, check=False)
    assert result.returncode == 0, result.stdout + result.stderr


def test_masked_foreground_is_cropped_and_disclosure_is_explicit(tmp_path):
    rgb = np.zeros((10, 14, 3), dtype=np.uint8)
    rgb[3:7, 4:9] = [220, 40, 10]
    mask = np.zeros((10, 14), dtype=np.uint8)
    mask[3:7, 4:9] = 255
    image_path, mask_path = tmp_path / "input.png", tmp_path / "mask.png"
    _write(image_path, rgb, "RGB")
    _write(mask_path, mask, "L")

    prepared, info = prop.prepare_foreground(Image, str(image_path), str(mask_path))

    assert prepared.width == prepared.height
    assert info["foreground_source"] == "explicit_mask"
    assert info["foreground_bbox_xyxy_pixels"] == [4, 3, 9, 7]
    pixels = np.asarray(prepared)
    assert tuple(pixels[pixels.shape[0] // 2, pixels.shape[1] // 2]) == (220, 40, 10)
    assert tuple(pixels[-1, -1]) == (127, 127, 127)


def test_mask_dimensions_and_empty_foreground_are_rejected(tmp_path):
    image_path, mask_path = tmp_path / "input.png", tmp_path / "mask.png"
    _write(image_path, np.zeros((8, 8, 3), dtype=np.uint8), "RGB")
    _write(mask_path, np.ones((7, 8), dtype=np.uint8) * 255, "L")
    with pytest.raises(ValueError, match="dimensions"):
        prop.prepare_foreground(Image, str(image_path), str(mask_path))

    _write(mask_path, np.zeros((8, 8), dtype=np.uint8), "L")
    with pytest.raises(ValueError, match="no foreground"):
        prop.prepare_foreground(Image, str(image_path), str(mask_path))


def test_opaque_image_records_full_frame_assumption(tmp_path):
    path = tmp_path / "opaque.png"
    _write(path, np.full((4, 6, 3), 80, dtype=np.uint8), "RGB")
    _prepared, info = prop.prepare_foreground(Image, str(path))
    assert info["foreground_source"] == "full_frame_assumption"
    assert info["foreground_bbox_xyxy_pixels"] == [0, 0, 6, 4]


def test_glb_archive_is_bounded_and_discloses_no_uv_or_metric_scale():
    archive = prop.build_archive(FakeMesh(), {"schema_version": 1, "operation": "image_to_object"})
    manifest = prop.validate_archive(archive)
    assert manifest["mesh_inspection"]["vertex_count"] == 4
    assert manifest["mesh_inspection"]["uv_texture_coordinates"] is False
    assert manifest["mesh_inspection"]["metric_scale"].startswith("unavailable")


def test_archive_rejects_manifest_or_signature_tampering():
    archive = prop.build_archive(FakeMesh(), {"schema_version": 1, "operation": "image_to_object"})
    with zipfile.ZipFile(io.BytesIO(archive), "r") as source:
        glb = source.read("model.glb")
        manifest = json.loads(source.read("manifest.json"))
    manifest["mesh_inspection"]["glb_sha256"] = "0" * 64
    stale_hash = io.BytesIO()
    with zipfile.ZipFile(stale_hash, "w", compression=zipfile.ZIP_DEFLATED) as output:
        output.writestr("model.glb", glb)
        output.writestr("manifest.json", json.dumps(manifest))
    with pytest.raises(ValueError, match="mesh inspection hash"):
        prop.validate_archive(stale_hash.getvalue())

    glb = b"bad!" + glb[4:]
    manifest["mesh_inspection"]["glb_sha256"] = prop.hashlib.sha256(glb).hexdigest()
    tampered = io.BytesIO()
    with zipfile.ZipFile(tampered, "w", compression=zipfile.ZIP_DEFLATED) as output:
        output.writestr("model.glb", glb)
        output.writestr("manifest.json", json.dumps(manifest))
    with pytest.raises(ValueError, match="signature"):
        prop.validate_archive(tampered.getvalue())


def test_cuda_archive_requires_reserved_device_and_worker_memory_metrics():
    archive = prop.build_archive(FakeMesh(), {"schema_version": 1, "operation": "image_to_object"})
    with zipfile.ZipFile(io.BytesIO(archive), "r") as source:
        glb = source.read("model.glb")
        manifest = json.loads(source.read("manifest.json"))
    manifest["runtime"] = {"device": "cuda:0"}
    tampered = io.BytesIO()
    with zipfile.ZipFile(tampered, "w", compression=zipfile.ZIP_DEFLATED) as output:
        output.writestr("model.glb", glb)
        output.writestr("manifest.json", json.dumps(manifest))
    with pytest.raises(ValueError, match="GPU memory measurements"):
        prop.validate_archive(tampered.getvalue())
