#!/usr/bin/env python3
"""Remove the coplanar front cap hidden by the v6 smart-display bezel."""

import hashlib, json, math, struct
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SRC = ROOT / "api/internal/physicalruntime/assets/smart-display-v6/generic-smart-display-v6.glb"
OUT = ROOT / "api/internal/physicalruntime/assets/smart-display-v7/generic-smart-display-v7.glb"
MANIFEST = ROOT / "api/internal/physicalruntime/assets/smart-display-v7/asset-manifest-v7.json"
EXPECTED_V6_SHA = "0b77cc12cf175c5c2d8640e0b81b3b1d5ac216233db3fc1fff4ba2d8b4f428e4"
EXPECTED_FRONT_TRIANGLES = 192
FRONT_DOT_MIN = 0.99999
PLANE_DISTANCE = 0.0267138936
PLANE_TOLERANCE = 2e-7

def parse_glb(raw: bytes) -> tuple[dict, bytearray]:
    if raw[:4] != b"glTF" or struct.unpack_from("<II", raw, 4) != (2, len(raw)): raise ValueError("expected a complete GLB 2.0")
    json_length, json_type = struct.unpack_from("<II", raw, 12)
    if json_type != 0x4E4F534A: raise ValueError("missing JSON chunk")
    doc = json.loads(raw[20 : 20 + json_length].decode().rstrip(" \0"))
    bin_offset = 20 + json_length
    bin_length, bin_type = struct.unpack_from("<II", raw, bin_offset)
    if bin_type != 0x004E4942 or bin_offset + 8 + bin_length != len(raw): raise ValueError("missing or malformed BIN chunk")
    return doc, bytearray(raw[bin_offset + 8 : bin_offset + 8 + bin_length])

def accessor_bytes(doc: dict, binary: bytearray, accessor_index: int) -> tuple[dict, bytes]:
    accessor = doc["accessors"][accessor_index]
    view = doc["bufferViews"][accessor["bufferView"]]
    if view.get("byteStride") is not None: raise ValueError("interleaved accessors are unsupported")
    start = view.get("byteOffset", 0) + accessor.get("byteOffset", 0)
    count = accessor["count"] * {5121: 1, 5123: 2, 5125: 4, 5126: 4}[accessor["componentType"]] * {"SCALAR": 1, "VEC2": 2, "VEC3": 3, "VEC4": 4, "MAT2": 4, "MAT3": 9, "MAT4": 16}[accessor["type"]]
    return accessor, bytes(binary[start : start + count])

def read_float_vec3(doc: dict, binary: bytearray, accessor_index: int) -> list[tuple[float, float, float]]:
    accessor = doc["accessors"][accessor_index]
    if accessor["componentType"] != 5126 or accessor["type"] != "VEC3": raise ValueError("expected FLOAT VEC3 accessor")
    _, raw = accessor_bytes(doc, binary, accessor_index)
    return [struct.unpack_from("<3f", raw, i * 12) for i in range(accessor["count"])]

def read_indices(doc: dict, binary: bytearray, accessor_index: int) -> tuple[list[int], str]:
    accessor = doc["accessors"][accessor_index]
    if accessor["type"] != "SCALAR": raise ValueError("expected scalar index accessor")
    fmt = {5121: "B", 5123: "H", 5125: "I"}.get(accessor["componentType"])
    if not fmt: raise ValueError("unsupported index component type")
    _, raw = accessor_bytes(doc, binary, accessor_index)
    return [x[0] for x in struct.iter_unpack("<" + fmt, raw)], fmt

def build() -> dict:
    raw = SRC.read_bytes()
    source_sha = hashlib.sha256(raw).hexdigest()
    if source_sha != EXPECTED_V6_SHA:
        raise ValueError(f"v6 input changed: {source_sha}")
    doc, binary = parse_glb(raw)
    shell_node = [(i, n) for i, n in enumerate(doc["nodes"]) if n.get("name") == "Display rear shell"]
    screen_node = [(i, n) for i, n in enumerate(doc["nodes"]) if n.get("name") == "MainDisplayActivePlane"]
    if len(shell_node) != 1 or shell_node[0][1].get("mesh") != 4 or len(screen_node) != 1: raise ValueError("v6 named hierarchy or shell binding changed")

    shell_primitive = doc["meshes"][4]["primitives"][0]
    if "indices" not in shell_primitive or shell_primitive.get("mode", 4) != 4: raise ValueError("v6 shell must remain indexed triangles")
    positions = read_float_vec3(doc, binary, shell_primitive["attributes"]["POSITION"])
    indices, index_fmt = read_indices(doc, binary, shell_primitive["indices"])
    bezel_primitive = doc["meshes"][5]["primitives"][0]
    bezel_normals = read_float_vec3(doc, binary, bezel_primitive["attributes"]["NORMAL"])
    front = bezel_normals[0]
    if len(indices) % 3: raise ValueError("shell indices are not triangle aligned")

    kept: list[int] = []
    removed: list[tuple[int, float, float]] = []
    for offset in range(0, len(indices), 3):
        tri = indices[offset : offset + 3]
        a, b, c = (positions[i] for i in tri)
        ab = tuple(b[k] - a[k] for k in range(3))
        ac = tuple(c[k] - a[k] for k in range(3))
        cross = (ab[1] * ac[2] - ab[2] * ac[1], ab[2] * ac[0] - ab[0] * ac[2], ab[0] * ac[1] - ab[1] * ac[0])
        length = math.sqrt(sum(x * x for x in cross))
        if length < 1e-14:
            kept.extend(tri)
            continue
        normal = tuple(x / length for x in cross)
        facing = sum(normal[k] * front[k] for k in range(3))
        distance = sum(front[k] * a[k] for k in range(3))
        all_on_plane = all(abs(sum(front[k] * positions[i][k] for k in range(3)) - PLANE_DISTANCE) <= PLANE_TOLERANCE for i in tri)
        if facing > FRONT_DOT_MIN and all_on_plane:
            removed.append((offset // 3, facing, distance))
        else:
            kept.extend(tri)

    if len(removed) != EXPECTED_FRONT_TRIANGLES: raise ValueError(f"expected {EXPECTED_FRONT_TRIANGLES} coplanar front-cap triangles; found {len(removed)}")
    if len(kept) + 3 * len(removed) != len(indices): raise ValueError("triangle accounting failed")

    index_accessor = doc["accessors"][shell_primitive["indices"]]
    component_type = index_accessor["componentType"]
    index_code = {5121: "B", 5123: "H", 5125: "I"}[component_type]
    encoded_indices = b"".join(struct.pack("<" + index_code, i) for i in kept)
    while len(binary) % 4:
        binary.append(0)
    new_offset = len(binary)
    binary.extend(encoded_indices)
    view_index = len(doc["bufferViews"])
    doc["bufferViews"].append({"buffer": 0, "byteOffset": new_offset, "byteLength": len(encoded_indices), "target": 34963})
    accessor_index = len(doc["accessors"])
    doc["accessors"].append({"bufferView": view_index, "componentType": component_type, "count": len(kept), "type": "SCALAR", "min": [min(kept)], "max": [max(kept)]})
    shell_primitive["indices"] = accessor_index
    doc["buffers"][0]["byteLength"] = len(binary)
    doc["asset"]["generator"] = "Device Studio generic smart display v7 coplanar cap removal"
    doc["extras"] = {"v7": {
        "source": "generic-smart-display-v6",
        "shellNodeIndex": shell_node[0][0],
        "shellMeshIndex": 4,
        "removedFrontCapTriangleCount": len(removed),
        "removedFrontCapPlaneDistanceMeters": PLANE_DISTANCE,
        "removedFrontCapPlaneToleranceMeters": PLANE_TOLERANCE,
        "frontNormalFromBezel": list(front),
        "retainedShellTriangleCount": len(kept) // 3,
        "retainedVertexAttributes": True,
        "screenNode": "MainDisplayActivePlane",
        "screenMesh": 6,
        "screenUvOrientation": "V=0 top",
    }}
    json_bytes = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    json_bytes += b" " * (-len(json_bytes) % 4)
    binary_bytes = bytes(binary) + b"\0" * (-len(binary) % 4)
    out = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(binary_bytes))
    out += struct.pack("<II", len(json_bytes), 0x4E4F534A) + json_bytes
    out += struct.pack("<II", len(binary_bytes), 0x004E4942) + binary_bytes
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_bytes(out)
    manifest = {
        "schema_version": 1,
        "status": "candidate_only_unqualified",
        "artifact_id": "vrooli-generic-smart-display-concept-v7",
        "output": {"path": "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v7/generic-smart-display-v7.glb", "sha256": hashlib.sha256(out).hexdigest(), "size_bytes": len(out)},
        "source": {"asset_id": "vrooli-generic-smart-display-concept-v6", "path": "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v6/generic-smart-display-v6.glb", "sha256": source_sha, "license": "AGPL-3.0-only", "authored_by": "Vrooli"},
        "derivation": {"generator": "derive_smart_display_v7.py", "runtime": "Python standard library; deterministic and offline", "changes": ["Remove only the 192 front-cap triangles of the Display rear shell that are coplanar with the continuous bezel ring within 0.2 micrometers.", "Retain every shell vertex attribute and all shell side/back triangles; leave display plane, UVs, materials, nodes, and transforms unchanged."], "shell_node_index": shell_node[0][0], "shell_mesh_index": 4, "removed_triangle_count": len(removed), "retained_triangle_count": len(kept) // 3, "front_cap_plane_distance_meters": PLANE_DISTANCE, "screen_node": "MainDisplayActivePlane", "screen_mesh_index": 6, "screen_primitive_index": 0, "uv_orientation": "V=0 top"},
        "qualification_limits": ["Generic authored concept; comparison dimensions are not product CAD or identity.", "Candidate requires a native appearance review; v6 remains unchanged and is not replaced."],
    }
    MANIFEST.parent.mkdir(parents=True, exist_ok=True)
    MANIFEST.write_text(json.dumps(manifest, separators=(",", ":")) + "\n")
    return manifest

if __name__ == "__main__":
    result = build()
    print(result["output"]["size_bytes"], result["output"]["sha256"], result["derivation"]["removed_triangle_count"])
