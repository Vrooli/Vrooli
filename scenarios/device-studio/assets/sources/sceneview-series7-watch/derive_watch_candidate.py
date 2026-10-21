#!/usr/bin/env python3
"""Create the pinned neutral-screen, UV-mapped Series 7 GLB candidate."""

from __future__ import annotations

import copy
import hashlib
import json
import math
import struct
import zlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[5]
SOURCE = ROOT / "scenarios/device-studio/docs/internal/goal/evidence/E4/free-models/sceneview-research/apple_watch_series_7.glb"
CREATOR = ROOT / "scenarios/device-studio/docs/internal/goal/evidence/E4/free-models/sceneview-research/watch-creator-api.json"
OUTPUT = ROOT / "scenarios/device-studio/api/internal/physicalruntime/assets/watch/series7-generic-neutral-screen.glb"
MANIFEST = ROOT / "scenarios/device-studio/api/internal/physicalruntime/assets/watch/asset-manifest.json"
SOURCE_SHA = "ac3c33a32bc27c562e6655245e20327d17c77c988f11aab3016496d851e4e5e3"
CREATOR_SHA = "900a4e5d90c06106979b2c2bc8431f43e4311e0426df43cea65ce95b7f60051a"
IMAGE_SHA = "bbdb946a84b0229752fef36683dc2c832a0b231c3eaa8cd970e7e13cb272754b"
SOURCE_IMAGE_BYTES = 139751
SCREEN_NODE, SCREEN_MESH, SCREEN_PRIMITIVE, SCREEN_MATERIAL = 27, 14, 0, 11
ROOT_NODE, ROOT_NAME, ROOT_SCALE = 28, "sceneview_root", 0.6430670822618203


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def load_glb(data: bytes) -> tuple[dict, bytes]:
    if data[:4] != b"glTF" or struct.unpack_from("<II", data, 4) != (2, len(data)):
        raise ValueError("pinned input is not a bounded GLB 2.0 file")
    json_size, kind = struct.unpack_from("<II", data, 12)
    if kind != 0x4E4F534A:
        raise ValueError("GLB JSON chunk is absent")
    bin_header = 20 + json_size
    bin_size, bin_kind = struct.unpack_from("<II", data, bin_header)
    if bin_kind != 0x004E4942 or bin_header + 8 + bin_size != len(data):
        raise ValueError("GLB BIN chunk is absent or has trailing bytes")
    return json.loads(data[20:bin_header]), data[bin_header + 8 :]


def pack_glb(doc: dict, binary: bytes) -> bytes:
    json_chunk = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    json_chunk += b" " * ((-len(json_chunk)) % 4)
    binary += b"\0" * ((-len(binary)) % 4)
    total = 12 + 8 + len(json_chunk) + 8 + len(binary)
    return (b"glTF" + struct.pack("<II", 2, total) +
            struct.pack("<II", len(json_chunk), 0x4E4F534A) + json_chunk +
            struct.pack("<II", len(binary), 0x004E4942) + binary)


def png_chunk(kind: bytes, payload: bytes) -> bytes:
    return (struct.pack(">I", len(payload)) + kind + payload +
            struct.pack(">I", zlib.crc32(kind + payload) & 0xFFFFFFFF))


def neutral_png() -> bytes:
    width = height = 1024
    rows = bytearray()
    for y in range(height):
        rows.append(0)
        for x in range(width):
            edge = max(abs(x - 511.5), abs(y - 511.5)) / 511.5
            value = 24 - int(edge * 8)
            rows.extend((value, value + 2, value + 5))
    header = struct.pack(">IIBBBBB", width, height, 8, 2, 0, 0, 0)
    return (b"\x89PNG\r\n\x1a\n" + png_chunk(b"IHDR", header) +
            png_chunk(b"IDAT", zlib.compress(bytes(rows), 9)) + png_chunk(b"IEND", b""))


def accessor_rows(doc: dict, binary: bytes, index: int, fmt: str, item_bytes: int) -> list[tuple]:
    accessor = doc["accessors"][index]
    view = doc["bufferViews"][accessor["bufferView"]]
    stride = view.get("byteStride", item_bytes)
    if stride < item_bytes:
        raise ValueError("accessor byte stride is smaller than its item")
    start = view.get("byteOffset", 0) + accessor.get("byteOffset", 0)
    return [struct.unpack_from(fmt, binary, start + row * stride) for row in range(accessor["count"])]


def index_values(doc: dict, binary: bytes, index: int) -> list[int]:
    accessor = doc["accessors"][index]
    view = doc["bufferViews"][accessor["bufferView"]]
    formats = {5121: "B", 5123: "H", 5125: "I"}
    code = formats.get(accessor["componentType"])
    if accessor["type"] != "SCALAR" or code is None or view.get("byteStride", 0):
        raise ValueError("target index accessor must be an unsigned packed scalar")
    start = view.get("byteOffset", 0) + accessor.get("byteOffset", 0)
    return list(struct.unpack_from(f"<{accessor['count']}{code}", binary, start))


def add_view(doc: dict, binary: bytearray, payload: bytes, target: int) -> int:
    binary.extend(b"\0" * ((-len(binary)) % 4))
    offset = len(binary)
    binary.extend(payload)
    doc["bufferViews"].append({"buffer": 0, "byteOffset": offset, "byteLength": len(payload), "target": target})
    return len(doc["bufferViews"]) - 1


def write_exact(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if path.exists() and path.read_bytes() != data:
        raise ValueError(f"refusing to overwrite a changed reviewed output: {path}")
    if not path.exists():
        path.write_bytes(data)


def derive() -> tuple[bytes, dict]:
    source, creator = SOURCE.read_bytes(), CREATOR.read_bytes()
    if sha(source) != SOURCE_SHA or sha(creator) != CREATOR_SHA:
        raise ValueError("pinned Series 7 source GLB or creator record hash changed")
    doc, source_bin = load_glb(source)
    if len(doc["nodes"]) != 29 or len(doc["meshes"]) != 15 or len(doc["materials"]) != 12:
        raise ValueError("source node/mesh/material inventory differs from the retained review")
    root = doc["nodes"][ROOT_NODE]
    if root.get("name") != ROOT_NAME or root.get("scale") != [ROOT_SCALE] * 3:
        raise ValueError("source root name/scale changed; preserve the explicit 0.643 scale")
    screen = doc["nodes"][SCREEN_NODE]
    if screen.get("name") != "Object_27" or screen.get("mesh") != SCREEN_MESH:
        raise ValueError("reviewed screen node/mesh binding changed")
    mesh = doc["meshes"][SCREEN_MESH]
    if mesh.get("name") != "Object_14" or len(mesh["primitives"]) != 1:
        raise ValueError("reviewed screen mesh changed")
    primitive = mesh["primitives"][SCREEN_PRIMITIVE]
    if primitive.get("material") != SCREEN_MATERIAL or primitive.get("mode", 4) != 4:
        raise ValueError("reviewed screen primitive/material binding changed")
    if len(doc["images"]) != 1 or doc["images"][0].get("mimeType") != "image/jpeg" or doc["images"][0].get("bufferView") != 0:
        raise ValueError("source embedded display image chain changed")
    texture_users = [i for i, material in enumerate(doc["materials"])
                     if material.get("pbrMetallicRoughness", {}).get("baseColorTexture", {}).get("index") == 0]
    if texture_users != [SCREEN_MATERIAL] or doc["textures"][0].get("source") != 0:
        raise ValueError("source image 0 is shared beyond the exact display material")
    image_view = doc["bufferViews"][0]
    image_start, image_size = image_view.get("byteOffset", 0), image_view["byteLength"]
    source_image = source_bin[image_start : image_start + image_size]
    if image_size != SOURCE_IMAGE_BYTES or sha(source_image) != IMAGE_SHA:
        raise ValueError("branded source image bytes differ from the retained rights audit")

    attributes = primitive["attributes"]
    if set(attributes) != {"POSITION", "NORMAL", "TEXCOORD_0"}:
        raise ValueError("screen primitive attribute set changed")
    position_index, normal_index, uv_index = attributes["POSITION"], attributes["NORMAL"], attributes["TEXCOORD_0"]
    if uv_index != 45 or doc["accessors"][position_index]["count"] != 86 or doc["accessors"][normal_index]["count"] != 86:
        raise ValueError("screen POSITION/NORMAL/UV accessor inventory changed")
    source_uv_accessor = doc["accessors"][uv_index]
    if source_uv_accessor["componentType"] != 5126 or source_uv_accessor["type"] != "VEC2":
        raise ValueError("source screen UV accessor must be float32 VEC2")
    uv_rows = accessor_rows(doc, source_bin, uv_index, "<ff", 8)
    indices = index_values(doc, source_bin, primitive["indices"])
    if len(indices) != 90 or len(indices) % 3:
        raise ValueError("source display topology is not the reviewed 30-triangle mesh")
    referenced = sorted(set(indices))
    if len(referenced) != 86 or referenced != list(range(86)):
        raise ValueError("screen primitive no longer references all 86 reviewed vertices")
    source_min = [min(uv_rows[i][axis] for i in referenced) for axis in range(2)]
    source_max = [max(uv_rows[i][axis] for i in referenced) for axis in range(2)]
    if min(source_max[i] - source_min[i] for i in range(2)) <= 0:
        raise ValueError("source screen UV envelope is degenerate")
    remapped = [tuple((uv_rows[i][axis] - source_min[axis]) / (source_max[axis] - source_min[axis]) for axis in range(2)) for i in range(len(uv_rows))]
    for vertex in referenced:
        if any(not math.isfinite(value) or value < -1e-6 or value > 1 + 1e-6 for value in remapped[vertex]):
            raise ValueError("normalized display UV contains a non-finite or out-of-range value")
    uv_bytes = b"".join(struct.pack("<ff", *value) for value in remapped)

    neutral = neutral_png()
    if len(neutral) >= image_size:
        raise ValueError("generated neutral screen PNG does not fit the pinned image buffer view")
    padded_neutral = neutral + bytes(image_size - len(neutral))
    changed_bin = bytearray(source_bin)
    changed_bin[image_start : image_start + image_size] = padded_neutral
    image_non_target = source_bin[:image_start] + source_bin[image_start + image_size :]
    neutral_image_hash = sha(neutral)
    uv_view = add_view(doc, changed_bin, uv_bytes, 34962)
    uv_accessor = {"bufferView": uv_view, "componentType": 5126, "count": 86, "type": "VEC2", "min": [0, 0], "max": [1, 1]}
    doc["accessors"].append(uv_accessor)
    new_uv_index = len(doc["accessors"]) - 1
    primitive["attributes"]["TEXCOORD_0"] = new_uv_index
    doc["images"][0]["mimeType"] = "image/png"
    material = doc["materials"][SCREEN_MATERIAL]
    material["name"] = "Neutral replaceable smartwatch display"
    pbr = material["pbrMetallicRoughness"]
    pbr["baseColorFactor"] = [1, 1, 1, 1]
    pbr["metallicFactor"] = 0
    pbr["roughnessFactor"] = 0.38
    doc["buffers"][0]["byteLength"] = len(changed_bin)
    output = pack_glb(doc, bytes(changed_bin))
    output_doc, output_bin = load_glb(output)
    output_image_view = output_doc["bufferViews"][0]
    output_neutral = output_bin[output_image_view.get("byteOffset", 0):output_image_view.get("byteOffset", 0) + output_image_view["byteLength"]]
    if output_neutral != padded_neutral or output_bin[:image_start] + output_bin[image_start + image_size:len(source_bin)] != image_non_target:
        raise ValueError("derivative changed non-image source BIN bytes")

    script_sha = sha(Path(__file__).read_bytes())
    manifest = {
        "schema_version": 1,
        "status": "candidate_only_unqualified",
        "artifact_id": "sceneview-series7-generic-neutral-watch-v1",
        "source": {
            "path": str(SOURCE.relative_to(ROOT)), "sha256": SOURCE_SHA, "size_bytes": len(source),
            "title": "Apple Watch Series 7 - Free Watch-Face SDC™",
            "creator_record_sha256": CREATOR_SHA,
            "creator": "SDC PERFORMANCE™; description attributes base model to bharathsrk13",
            "listing_url": "https://sketchfab.com/3d-models/apple-watch-series-7-free-watch-face-sdctm-cf15ecbf159f42a28df5e56bdfa666bb",
            "license": "CC BY 4.0",
            "attribution": "Apple Watch Series 7 - Free Watch-Face SDC™ by SDC PERFORMANCE™; base model attribution to bharathsrk13; adapted under CC BY 4.0.",
        },
        "output": {"path": str(OUTPUT.relative_to(ROOT)), "sha256": sha(output), "size_bytes": len(output)},
        "derivation": {
            "script_sha256": script_sha,
            "source_image_sha256": IMAGE_SHA,
            "neutral_png_sha256": neutral_image_hash,
            "neutral_png_bytes": len(neutral),
            "padded_image_view_bytes": image_size,
            "image_view_offset": image_start,
            "source_non_image_bin_sha256": sha(image_non_target),
            "output_non_image_source_bin_sha256": sha(output_bin[:image_start] + output_bin[image_start + image_size:len(source_bin)]),
            "source_bin_bytes": len(source_bin),
            "output_source_bin_prefix_bytes": len(source_bin),
            "appended_uv_buffer_view": uv_view,
            "appended_uv_accessor": new_uv_index,
            "appended_uv_bytes_sha256": sha(uv_bytes),
            "appended_uv_bytes": len(uv_bytes),
        },
        "display_binding": {
            "node_index": SCREEN_NODE, "node_name": "Object_27", "mesh_index": SCREEN_MESH,
            "mesh_name": "Object_14", "primitive_index": SCREEN_PRIMITIVE,
            "material_index": SCREEN_MATERIAL, "texture_index": 0, "image_index": 0,
            "source_uv_accessor": uv_index, "derivative_uv_accessor": new_uv_index,
            "source_uv_min": source_min, "source_uv_max": source_max,
            "derivative_uv_min": [0, 0], "derivative_uv_max": [1, 1],
            "referenced_vertex_count": len(referenced), "index_count": len(indices), "triangle_count": len(indices) // 3,
            "position_accessor": position_index, "normal_accessor": normal_index,
            "target_material_policy": {"base_color_factor": [1, 1, 1, 1], "metallic_factor": 0, "roughness_factor": 0.38},
        },
        "preservation": {
            "source_bin_non_image_bytes_unchanged": True,
            "source_positions_normals_indices_and_triangle_topology_unchanged": True,
            "non_target_materials_textures_and_images_unchanged": True,
            "original_uv_accessor_and_all_non_target_uv_references_unchanged": True,
            "source_hierarchy_and_transforms_unchanged": True,
            "root_node": ROOT_NODE, "root_name": ROOT_NAME, "root_scale_preserved": ROOT_SCALE,
            "only_source_image_0_and_target_screen_material_and_uv_binding_changed": True,
        },
        "geometry": {
            "source_axis_envelope_mm": [25.201, 57.579, 54.549],
            "case_glass_screen_display_frame_mm": [28.344, 24.613, 7.942],
            "source_scale_policy": "Preserve the exact source hierarchy including sceneview_root scale 0.643; no nominal-size fitting or scale normalization.",
            "variant": "unknown",
            "measurement_limit": "GLB-derived source geometry under the glTF metre assumption; not independent physical measurement, not a 41 mm/45 mm variant, and the full model AABB is band-dominated.",
        },
        "rights_and_claims": {
            "adaptation_notice": "Replaced branded source image 0 with a generated neutral dark display image, normalized only the selected display TEXCOORD_0 chart, and changed only display material 11 to neutral replaceable-screen values.",
            "rights_limit": "CC BY 4.0 attribution applies to the model adaptation; the original branded image is excluded. No manufacturer endorsement, trademark clearance, physical qualification, or round-watch family claim.",
            "candidate_scope": "Generic rectangular smartwatch-style candidate only; not a round-watch family implementation.",
        },
    }
    return output, manifest


def main() -> None:
    output, manifest = derive()
    write_exact(OUTPUT, output)
    encoded = (json.dumps(manifest, indent=2) + "\n").encode()
    write_exact(MANIFEST, encoded)
    print(f"output={OUTPUT} bytes={len(output)} sha256={manifest['output']['sha256']}")
    print(f"manifest={MANIFEST} sha256={sha(encoded)}")
    print("screen=node27/mesh14/primitive0/material11; 30 triangles and 0.643 root scale preserved")


if __name__ == "__main__":
    main()
