#!/usr/bin/env python3
"""Build the v2 display-only primitive/UV derivative from frozen monitor UV v1."""

from __future__ import annotations

import copy
import hashlib
import json
import struct
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SOURCE = ROOT / "assets/sources/innerscene-ultrawide-monitor/ultrawide-computer-monitor-screen-uv-v1.glb"
ORIGINAL_SHA = "67e5c979d6755dd561c291f39ae5748849cb26f7bb78f97821166ecbd05e2bed"
SOURCE_SHA = "eff6e9a20a4fd26003046e7f0c7b5d0ea737d83d4ce9adb66b1c118561155266"
SOURCE_MANIFEST_SHA = "507989d4027b4a31f18629b03fa67c12eaed5c30bdab5ef8246f716e40c10e54"
OUTPUT = Path(__file__).with_name("monitor-screen-split-v2.glb")
MANIFEST = Path(__file__).with_name("monitor-screen-split-v2-manifest.json")
FRONT_TRIANGLES = ((110, 136, 27), (110, 27, 1))
FRONT_VERTEX_IDS = (1, 27, 110, 136)


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def load_glb(data: bytes) -> tuple[dict, bytes]:
    if data[:4] != b"glTF" or struct.unpack_from("<II", data, 4) != (2, len(data)):
        raise ValueError("source is not a bounded GLB 2.0 file")
    json_size, json_kind = struct.unpack_from("<II", data, 12)
    if json_kind != 0x4E4F534A:
        raise ValueError("source JSON chunk is missing")
    json_start = 20
    bin_header = json_start + json_size
    bin_size, bin_kind = struct.unpack_from("<II", data, bin_header)
    if bin_kind != 0x004E4942 or bin_header + 8 + bin_size != len(data):
        raise ValueError("source BIN chunk is missing or has trailing data")
    return json.loads(data[json_start:bin_header]), data[bin_header + 8 :]


def pack_glb(doc: dict, binary_chunk: bytes) -> bytes:
    json_chunk = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    json_chunk += b" " * ((-len(json_chunk)) % 4)
    binary_chunk += b"\0" * ((-len(binary_chunk)) % 4)
    total = 12 + 8 + len(json_chunk) + 8 + len(binary_chunk)
    return b"glTF" + struct.pack("<II", 2, total) + struct.pack("<II", len(json_chunk), 0x4E4F534A) + json_chunk + struct.pack("<II", len(binary_chunk), 0x004E4942) + binary_chunk


def accessor_bytes(doc: dict, binary_chunk: bytes, index: int) -> tuple[dict, bytes]:
    accessor = doc["accessors"][index]
    view = doc["bufferViews"][accessor["bufferView"]]
    start = view.get("byteOffset", 0) + accessor.get("byteOffset", 0)
    size = view["byteLength"] - accessor.get("byteOffset", 0)
    return accessor, binary_chunk[start : start + size]


def vec2_values(doc: dict, binary_chunk: bytes, accessor_index: int) -> list[tuple[float, float]]:
    accessor, raw = accessor_bytes(doc, binary_chunk, accessor_index)
    if accessor["componentType"] != 5126 or accessor["type"] != "VEC2" or not accessor.get("bufferView"):
        raise ValueError("source TEXCOORD_0 must be float32 VEC2")
    view = doc["bufferViews"][accessor["bufferView"]]
    stride = view.get("byteStride", 8)
    offset = accessor.get("byteOffset", 0)
    return [struct.unpack_from("<ff", binary_chunk, view.get("byteOffset", 0) + offset + i * stride) for i in range(accessor["count"])]


def read_indices(doc: dict, binary_chunk: bytes, accessor_index: int) -> list[int]:
    accessor, _ = accessor_bytes(doc, binary_chunk, accessor_index)
    view = doc["bufferViews"][accessor["bufferView"]]
    if accessor["componentType"] != 5123 or accessor["type"] != "SCALAR" or view.get("byteStride", 0):
        raise ValueError("source index accessor must be tightly packed uint16 SCALAR")
    start = view.get("byteOffset", 0) + accessor.get("byteOffset", 0)
    return list(struct.unpack_from(f"<{accessor['count']}H", binary_chunk, start))


def append_view(doc: dict, binary_chunk: bytearray, payload: bytes, target: int) -> int:
    binary_chunk.extend(b"\0" * ((-len(binary_chunk)) % 4))
    offset = len(binary_chunk)
    binary_chunk.extend(payload)
    doc["bufferViews"].append({"buffer": 0, "byteOffset": offset, "byteLength": len(payload), "target": target})
    return len(doc["bufferViews"]) - 1


def append_accessor(doc: dict, view_index: int, *, component: int, count: int, kind: str, minimum=None, maximum=None) -> int:
    accessor = {"bufferView": view_index, "componentType": component, "count": count, "type": kind}
    if minimum is not None:
        accessor["min"] = minimum
    if maximum is not None:
        accessor["max"] = maximum
    doc["accessors"].append(accessor)
    return len(doc["accessors"]) - 1


def derive() -> tuple[bytes, dict]:
    source = SOURCE.read_bytes()
    if digest(source) != SOURCE_SHA:
        raise ValueError("frozen UV-v1 source hash changed")
    doc, original_bin = load_glb(source)
    if doc["asset"]["version"] != "2.0" or doc["buffers"][0]["byteLength"] > len(original_bin):
        raise ValueError("source GLB metadata differs from the reviewed input")
    mesh = doc["meshes"][1]
    primitive = mesh["primitives"][0]
    if mesh["name"] != "Cube.001" or primitive.get("material") != 1:
        raise ValueError("source display mesh/material chain changed")
    position_index = primitive["attributes"]["POSITION"]
    normal_index = primitive["attributes"]["NORMAL"]
    uv_index = primitive["attributes"]["TEXCOORD_0"]
    positions_accessor, _ = accessor_bytes(doc, original_bin, position_index)
    normals_accessor, _ = accessor_bytes(doc, original_bin, normal_index)
    position_view = doc["bufferViews"][positions_accessor["bufferView"]]
    normal_view = doc["bufferViews"][normals_accessor["bufferView"]]
    positions = [struct.unpack_from("<fff", original_bin, position_view.get("byteOffset", 0) + positions_accessor.get("byteOffset", 0) + i * 12) for i in range(positions_accessor["count"])]
    normals = [struct.unpack_from("<fff", original_bin, normal_view.get("byteOffset", 0) + normals_accessor.get("byteOffset", 0) + i * 12) for i in range(normals_accessor["count"])]
    indices = read_indices(doc, original_bin, primitive["indices"])
    triangles = tuple(tuple(indices[i : i + 3]) for i in range(0, len(indices), 3))
    if triangles[292:294] != FRONT_TRIANGLES or len(triangles) != 300:
        raise ValueError("reviewed planar display triangle identities changed")
    face_vertices = sorted(set(v for triangle in triangles[292:294] for v in triangle))
    if tuple(face_vertices) != FRONT_VERTEX_IDS:
        raise ValueError("planar display vertex identities changed")
    face_z = [positions[v][2] for v in face_vertices]
    if max(face_z) - min(face_z) > 1e-8 or any(normals[v][2] < 0.999 for v in face_vertices):
        raise ValueError("target display vertices are no longer one front-facing plane")

    uv_values = vec2_values(doc, original_bin, uv_index)
    face_uv = [uv_values[v] for v in face_vertices]
    u_min, u_max = min(v[0] for v in face_uv), max(v[0] for v in face_uv)
    v_min, v_max = min(v[1] for v in face_uv), max(v[1] for v in face_uv)
    if u_max <= u_min or v_max <= v_min or len(set(face_uv)) != 4:
        raise ValueError("front display UVs are not four distinct rectangle corners")
    uv_remap = list(uv_values)
    for vertex in face_vertices:
        u, v = uv_values[vertex]
        uv_remap[vertex] = ((u - u_min) / (u_max - u_min), (v - v_min) / (v_max - v_min))
    corner_values = {tuple(round(x, 6) for x in uv_remap[v]) for v in face_vertices}
    if corner_values != {(0.0, 0.0), (0.0, 1.0), (1.0, 0.0), (1.0, 1.0)}:
        raise ValueError("target-only UV remap did not produce all four unit-rectangle corners")

    front_indices = [vertex for triangle in triangles[292:294] for vertex in triangle]
    bevel_indices = [vertex for i, triangle in enumerate(triangles) if i not in (292, 293) for vertex in triangle]
    if len(front_indices) != 6 or len(bevel_indices) != 894 or sorted(front_indices + bevel_indices) != sorted(indices):
        raise ValueError("front/bevel index streams do not exactly partition source topology")
    source_triangles = sorted(tuple(sorted(triangle)) for triangle in triangles)
    derived_triangles = sorted(
        tuple(sorted(stream[i : i + 3]))
        for stream in (front_indices, bevel_indices)
        for i in range(0, len(stream), 3)
    )
    if source_triangles != derived_triangles:
        raise ValueError("derived triangles do not reproduce the exact source triangle multiset")
    binary_chunk = bytearray(original_bin)
    front_view = append_view(doc, binary_chunk, struct.pack("<6H", *front_indices), 34963)
    front_accessor = append_accessor(doc, front_view, component=5123, count=6, kind="SCALAR", minimum=[min(front_indices)], maximum=[max(front_indices)])
    bevel_view = append_view(doc, binary_chunk, struct.pack("<894H", *bevel_indices), 34963)
    bevel_accessor = append_accessor(doc, bevel_view, component=5123, count=894, kind="SCALAR", minimum=[min(bevel_indices)], maximum=[max(bevel_indices)])
    uv_raw = b"".join(struct.pack("<ff", *value) for value in uv_remap)
    uv_view = append_view(doc, binary_chunk, uv_raw, 34962)
    display_uv_accessor = append_accessor(doc, uv_view, component=5126, count=len(uv_remap), kind="VEC2", minimum=[0, 0], maximum=[1, 1])

    original_material = copy.deepcopy(doc["materials"][1])
    display_material = copy.deepcopy(original_material)
    display_material["name"] = "Target display surface v2 (source neutral glass clone)"
    doc["materials"].append(display_material)
    screen_primitive = copy.deepcopy(primitive)
    screen_primitive["attributes"]["TEXCOORD_0"] = display_uv_accessor
    screen_primitive["indices"] = front_accessor
    screen_primitive["material"] = len(doc["materials"]) - 1
    bevel_primitive = copy.deepcopy(primitive)
    bevel_primitive["indices"] = bevel_accessor
    bevel_primitive["material"] = 1
    mesh["primitives"] = [screen_primitive, bevel_primitive]
    doc["buffers"][0]["byteLength"] = len(binary_chunk)
    output = pack_glb(doc, bytes(binary_chunk))

    manifest = {
        "schema_version": 1,
        "artifact_id": "innerscene-ultrawide-monitor-screen-split-v2",
        "source_original_sha256": ORIGINAL_SHA,
        "source_v1_sha256": SOURCE_SHA,
        "source_manifest_sha256": SOURCE_MANIFEST_SHA,
        "output_sha256": digest(output),
        "output_size_bytes": len(output),
        "mesh_index": 1,
        "source_primitive_index": 0,
        "display_primitive_index": 0,
        "display_material_index": len(doc["materials"]) - 1,
        "bezel_primitive_index": 1,
        "bezel_material_index": 1,
        "display_source_triangle_indices": [292, 293],
        "display_source_vertex_indices": face_vertices,
        "source_triangle_count": len(triangles),
        "display_triangle_count": 2,
        "bezel_triangle_count": 298,
        "index_partition_exact": True,
        "position_normal_and_original_uv_accessors_unchanged": True,
        "source_bin_prefix_sha256": digest(original_bin),
        "source_bin_prefix_bytes": len(original_bin),
        "display_uv_adaptation": "A new TEXCOORD_0 accessor is used only by the planar display primitive. It clones v1 UV0 values and remaps the four referenced display corners to [0,1]; original UV0 accessor and every bezel UV reference remain unchanged.",
        "display_uv_corners": sorted([list(value) for value in corner_values]),
        "material_adaptation": "The front primitive uses a clone of original neutral glass material 1 so the screen binder can replace only that slot. All 298 bevel/chassis triangles retain original material 1 and its textures.",
        "identity": "Generic Innerscene CC0 monitor design; no manufacturer or model claim.",
        "qualification": "candidate_only_unqualified",
    }
    return output, manifest


def write_exact(path: Path, data: bytes) -> None:
    if path.exists() and path.read_bytes() != data:
        raise ValueError(f"refusing to overwrite changed derivative: {path}")
    if not path.exists():
        path.write_bytes(data)


def main() -> None:
    output, manifest = derive()
    write_exact(OUTPUT, output)
    encoded_manifest = (json.dumps(manifest, indent=2) + "\n").encode()
    write_exact(MANIFEST, encoded_manifest)
    print(f"output={OUTPUT} bytes={len(output)} sha256={manifest['output_sha256']}")
    print(f"manifest={MANIFEST} sha256={digest(encoded_manifest)}")
    print("topology=300 source triangles -> 2 display + 298 bevel; exact partition verified")


if __name__ == "__main__":
    main()
