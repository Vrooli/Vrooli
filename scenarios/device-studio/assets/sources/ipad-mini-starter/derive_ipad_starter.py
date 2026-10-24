"""Fixed stdlib GLB rewrite replacing only the iPad source screen image."""
import copy
import hashlib
import json
import math
import struct
import sys
import zlib
from pathlib import Path


ROOT = Path(__file__).resolve().parent
INPUTS = ROOT / "inputs"
OUTPUTS = ROOT / "outputs"
MANIFEST_PATH = ROOT / "ipad-starter-manifest.json"
SOURCE_NAME = "ipad_mini_6.glb"
MAX_SOURCE_BYTES = 2_100_000
PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
JSON_CHUNK = 0x4E4F534A
BIN_CHUNK = 0x004E4942
UV_NORMALIZATION_MANIFEST = ROOT / "ipad-screen-uv-normalization-manifest.json"
UV_NORMALIZATION_SOURCE = "ipad-mini-neutral-screen.glb"
UV_ORIENTATION_MANIFEST = ROOT / "ipad-screen-uv-orientation-manifest.json"
UV_ORIENTATION_SOURCE = "ipad-mini-screen-uv-normalized.glb"
UV_ORIENTATION_OUTPUTS = ROOT / "outputs-oriented"


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def chunk(kind, payload):
    return struct.pack(">I", len(payload)) + kind + payload + struct.pack(">I", zlib.crc32(kind + payload) & 0xffffffff)


def neutral_png(width, height, rgba):
    require(width == 512 and height == 1024 and rgba == [128, 128, 128, 255], "neutral raster differs from fixed profile")
    row = b"\0" + bytes(rgba) * width
    raw = row * height
    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)
    return (PNG_SIGNATURE + chunk(b"IHDR", ihdr) + chunk(b"sRGB", b"\0") +
            chunk(b"IDAT", zlib.compress(raw, 9)) + chunk(b"IEND", b""))


def inspect_neutral_png(data, expected_width, expected_height, expected_rgba):
    require(data.startswith(PNG_SIGNATURE), "neutral screen is not PNG")
    offset, chunks, compressed = len(PNG_SIGNATURE), [], bytearray()
    while offset < len(data):
        require(offset + 12 <= len(data), "truncated PNG chunk")
        length = struct.unpack_from(">I", data, offset)[0]
        kind = data[offset + 4:offset + 8]
        end = offset + 8 + length
        require(end + 4 <= len(data), "PNG chunk exceeds file")
        payload = data[offset + 8:end]
        crc = struct.unpack_from(">I", data, end)[0]
        require(crc == (zlib.crc32(kind + payload) & 0xffffffff), "PNG chunk CRC mismatch")
        chunks.append(kind)
        if kind == b"IHDR":
            width, height, depth, color, compression, filtering, interlace = struct.unpack(">IIBBBBB", payload)
            require((width, height, depth, color, compression, filtering, interlace) ==
                    (expected_width, expected_height, 8, 6, 0, 0, 0), "neutral PNG format differs")
        elif kind == b"sRGB":
            require(payload == b"\0", "neutral PNG sRGB rendering intent differs")
        elif kind == b"IDAT":
            compressed.extend(payload)
        elif kind == b"IEND":
            require(length == 0 and end + 4 == len(data), "PNG IEND or trailing bytes are invalid")
        else:
            raise ValueError("unexpected PNG chunk")
        offset = end + 4
    require(chunks == [b"IHDR", b"sRGB", b"IDAT", b"IEND"],
            "PNG chunk sequence differs")
    decoded = zlib.decompress(bytes(compressed))
    row_size = 1 + expected_width * 4
    require(len(decoded) == row_size * expected_height, "decoded PNG raster size differs")
    expected_row = b"\0" + bytes(expected_rgba) * expected_width
    require(all(decoded[y * row_size:(y + 1) * row_size] == expected_row for y in range(expected_height)),
            "neutral screen PNG is not uniformly opaque neutral RGBA")
    return width, height


def parse_glb(data):
    require(20 <= len(data) <= MAX_SOURCE_BYTES + 64 * 1024, "GLB size outside fixed bound")
    magic, version, declared = struct.unpack_from("<4sII", data, 0)
    require(magic == b"glTF" and version == 2 and declared == len(data), "GLB header is invalid")
    offset = 12
    chunks = []
    while offset < len(data):
        require(offset + 8 <= len(data), "truncated GLB chunk header")
        size, kind = struct.unpack_from("<II", data, offset)
        offset += 8
        require(size <= len(data) - offset, "GLB chunk exceeds declared file")
        chunks.append((kind, data[offset:offset + size]))
        offset += size
    require(len(chunks) == 2 and chunks[0][0] == JSON_CHUNK and chunks[1][0] == BIN_CHUNK,
            "fixed profile requires exactly JSON then BIN chunks")
    doc = json.loads(chunks[0][1].decode("utf-8").rstrip(" \t\r\n\0"))
    return doc, chunks[1][1]


def buffer_view_bytes(doc, binary, index):
    require(0 <= index < len(doc.get("bufferViews", [])), "bufferView index is invalid")
    view = doc["bufferViews"][index]
    start, length = int(view.get("byteOffset", 0)), int(view["byteLength"])
    require(view.get("buffer", 0) == 0 and start >= 0 and length > 0 and start <= len(binary) and length <= len(binary) - start,
            "bufferView bounds or buffer binding is invalid")
    return binary[start:start + length]


def validate_source(doc, binary, manifest):
    source = manifest["source"]
    require(doc.get("asset", {}).get("version") == "2.0", "source glTF version differs")
    require(not doc.get("extensionsUsed") and not doc.get("extensionsRequired"), "source extensions are outside this fixed profile")
    buffers = doc.get("buffers", [])
    require(len(buffers) == 1 and "uri" not in buffers[0] and buffers[0]["byteLength"] == len(binary),
            "source must have one embedded BIN buffer")
    images = doc.get("images", [])
    require(len(images) == 3, "source image count differs")
    for item, expected in zip(images, source["embedded_images"]):
        require("uri" not in item and item.get("mimeType") == expected["mime_type"], "embedded image declaration differs")
        payload = buffer_view_bytes(doc, binary, int(item["bufferView"]))
        require(len(payload) == expected["size_bytes"] and sha256(payload) == expected["sha256"],
                "embedded image bytes differ from pinned acquisition")
    bind = source["screen_binding"]
    nodes, meshes = doc.get("nodes", []), doc.get("meshes", [])
    require(nodes[bind["node_index"]].get("mesh") == bind["mesh_index"], "screen node/mesh binding differs")
    primitives = meshes[bind["mesh_index"]].get("primitives", [])
    require(len(primitives) == 1 and primitives[bind["primitive_index"]].get("mode", 4) == 4,
            "screen mesh primitive structure differs")
    primitive = primitives[bind["primitive_index"]]
    require(primitive.get("material") == bind["material_index"], "screen primitive material differs")
    material = doc["materials"][bind["material_index"]]
    texture_index = material.get("pbrMetallicRoughness", {}).get("baseColorTexture", {}).get("index")
    require(texture_index == bind["texture_index"], "screen material texture differs")
    texture = doc["textures"][bind["texture_index"]]
    require(texture.get("source") == bind["image_index"] and texture.get("sampler") == 1,
            "screen texture/image/sampler binding differs")
    target_view = int(images[bind["image_index"]]["bufferView"])
    require(target_view == 1 and all(int(image.get("bufferView", -1)) != target_view
                                     for index, image in enumerate(images) if index != bind["image_index"]),
            "screen bufferView is not private to the pinned image")
    require(all(int(accessor.get("bufferView", -1)) != target_view for accessor in doc.get("accessors", [])),
            "screen bufferView is shared with an accessor")
    uv_index = primitive["attributes"].get("TEXCOORD_0")
    accessor = doc["accessors"][uv_index]
    require(accessor.get("componentType") == 5126 and accessor.get("type") == "VEC2", "screen UV accessor format differs")
    view = doc["bufferViews"][accessor["bufferView"]]
    stride = int(view.get("byteStride", 8))
    base = int(view.get("byteOffset", 0)) + int(accessor.get("byteOffset", 0))
    values = []
    for i in range(int(accessor["count"])):
        require(base + i * stride + 8 <= len(binary), "screen UV accessor exceeds BIN")
        values.extend(struct.unpack_from("<2f", binary, base + i * stride))
    actual_min = [min(values[0::2]), min(values[1::2])]
    actual_max = [max(values[0::2]), max(values[1::2])]
    for actual, expected in zip(actual_min, bind["uv_min"]):
        require(abs(actual - expected) < 2e-6, "screen UV minimum differs from audit")
    for actual, expected in zip(actual_max, bind["uv_max"]):
        require(abs(actual - expected) < 2e-6, "screen UV maximum differs from audit")
    spans = []
    for i, view in enumerate(doc["bufferViews"]):
        start, end = int(view.get("byteOffset", 0)), int(view.get("byteOffset", 0)) + int(view["byteLength"])
        require(start >= 0 and end <= len(binary), f"bufferView {i} exceeds BIN")
        spans.append((start, end, i))
    spans.sort()
    cursor = 0
    for start, end, _ in spans:
        require(start >= cursor, "bufferViews overlap")
        require(not any(binary[cursor:start]), "source has unsupported nonzero bytes between bufferViews")
        cursor = end
    require(not any(binary[cursor:]), "source has unsupported nonzero bytes after final bufferView")
    return {"uv_min": actual_min, "uv_max": actual_max, "sorted_buffer_view_indices": [row[2] for row in spans]}


def replace_screen_image(source_data, manifest):
    doc, source_bin = parse_glb(source_data)
    source = manifest["source"]
    require(len(source_data) == source["size_bytes"] and sha256(source_data) == source["sha256"], "source GLB pin mismatch")
    uv_facts = validate_source(doc, source_bin, manifest)
    source_doc = copy.deepcopy(doc)
    target_image = int(source["screen_binding"]["image_index"])
    target_view = int(doc["images"][target_image]["bufferView"])
    replacement = neutral_png(manifest["neutral_screen"]["width"], manifest["neutral_screen"]["height"],
                              manifest["neutral_screen"]["rgba_srgb"])
    inspect_neutral_png(replacement, manifest["neutral_screen"]["width"], manifest["neutral_screen"]["height"],
                        manifest["neutral_screen"]["rgba_srgb"])
    result_bin = bytearray()
    copied_hashes = {}
    original_views = []
    for index, view in enumerate(doc["bufferViews"]):
        original_views.append(buffer_view_bytes(doc, source_bin, index))
    ordered = sorted(range(len(doc["bufferViews"])), key=lambda index: (int(doc["bufferViews"][index].get("byteOffset", 0)), index))
    for index in ordered:
        view = doc["bufferViews"][index]
        while len(result_bin) % 4:
            result_bin.append(0)
        view["byteOffset"] = len(result_bin)
        payload = replacement if index == target_view else original_views[index]
        view["byteLength"] = len(payload)
        result_bin.extend(payload)
        if index != target_view:
            copied_hashes[str(index)] = sha256(payload)
    while len(result_bin) % 4:
        result_bin.append(0)
    doc["buffers"][0]["byteLength"] = len(result_bin)
    doc["images"][target_image]["mimeType"] = "image/png"
    for index in range(len(doc["bufferViews"])):
        if index != target_view:
            require(buffer_view_bytes(doc, result_bin, index) == original_views[index],
                    f"non-target bufferView {index} payload changed")
    require(sha256(buffer_view_bytes(doc, result_bin, 0)) == source["embedded_images"][0]["sha256"] and
            sha256(buffer_view_bytes(doc, result_bin, 2)) == source["embedded_images"][2]["sha256"],
            "non-screen image payload changed")
    require(source["embedded_images"][1]["sha256"] != sha256(replacement) and
            bytes(result_bin).find(buffer_view_bytes(source_doc, source_bin, target_view)) < 0,
            "private source screen payload remains in derivative BIN")
    expected_doc = copy.deepcopy(source_doc)
    for index, view in enumerate(doc["bufferViews"]):
        expected_doc["bufferViews"][index]["byteOffset"] = view["byteOffset"]
        expected_doc["bufferViews"][index]["byteLength"] = view["byteLength"]
    expected_doc["buffers"][0]["byteLength"] = len(result_bin)
    expected_doc["images"][target_image]["mimeType"] = "image/png"
    require(doc == expected_doc, "source JSON changed outside the declared image/layout updates")
    json_bytes = json.dumps(doc, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    json_bytes += b" " * ((-len(json_bytes)) % 4)
    binary = bytes(result_bin)
    output = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(binary))
    output += struct.pack("<II", len(json_bytes), JSON_CHUNK) + json_bytes
    output += struct.pack("<II", len(binary), BIN_CHUNK) + binary
    require(len(output) <= manifest["output"]["max_bytes"], "derived GLB exceeds fixed output bound")
    out_doc, out_bin = parse_glb(output)
    require(out_doc == doc, "serialized GLB JSON differs from validated rewritten document")
    require(validate_source_except_screen(out_doc, out_bin, source, target_image), "output invariant validation failed")
    derived_screen = buffer_view_bytes(out_doc, out_bin, target_view)
    require(derived_screen == replacement, "round-tripped neutral screen payload changed")
    inspect_neutral_png(derived_screen, manifest["neutral_screen"]["width"], manifest["neutral_screen"]["height"],
                        manifest["neutral_screen"]["rgba_srgb"])
    for index in range(len(out_doc["bufferViews"])):
        if index != target_view:
            require(buffer_view_bytes(out_doc, out_bin, index) == original_views[index],
                    f"round-tripped non-target bufferView {index} changed")
    return output, replacement, uv_facts, copied_hashes


def normalize_screen_uv(source_data, manifest):
    """Add a private normalized UV accessor; preserve every existing BIN view."""
    doc, source_bin = parse_glb(source_data)
    source = manifest["source"]
    binding = source["screen_binding"]
    section = manifest.get("orientation", manifest.get("normalization"))
    require(len(source_data) == source["size_bytes"] and sha256(source_data) == source["sha256"],
            "neutral derivative source GLB pin mismatch")
    require(doc.get("asset", {}).get("version") == "2.0" and not doc.get("extensionsUsed") and
            not doc.get("extensionsRequired"), "neutral derivative is outside the pinned GLB subset")
    require(len(doc.get("nodes", [])) == section["node_count"] and
            len(doc.get("meshes", [])) == section["mesh_count"] and
            len(doc.get("images", [])) == 3, "neutral derivative inventory differs")
    require(doc["nodes"][binding["node_index"]].get("mesh") == binding["mesh_index"],
            "neutral derivative screen node binding differs")
    primitive = doc["meshes"][binding["mesh_index"]]["primitives"][binding["primitive_index"]]
    require(primitive.get("material") == binding["material_index"] and
            primitive.get("mode", 4) == 4, "neutral derivative screen primitive differs")
    texture_index = doc["materials"][binding["material_index"]]["pbrMetallicRoughness"]["baseColorTexture"]["index"]
    require(texture_index == binding["texture_index"] and
            doc["textures"][texture_index].get("source") == binding["image_index"],
            "neutral derivative screen texture chain differs")
    image = doc["images"][binding["image_index"]]
    neutral_bytes = buffer_view_bytes(doc, source_bin, int(image["bufferView"]))
    require(image.get("mimeType") == "image/png" and sha256(neutral_bytes) == source["neutral_screen_sha256"],
            "neutral derivative screen raster differs")
    inspect_neutral_png(neutral_bytes, 512, 1024, [128, 128, 128, 255])
    attributes = primitive["attributes"]
    uv_accessor_index = attributes.get("TEXCOORD_0")
    position_accessor_index = attributes.get("POSITION")
    require(uv_accessor_index is not None and position_accessor_index is not None,
            "screen primitive lacks required UV0 or POSITION")
    uv_accessor = doc["accessors"][uv_accessor_index]
    pos_accessor = doc["accessors"][position_accessor_index]
    require(uv_accessor.get("componentType") == 5126 and uv_accessor.get("type") == "VEC2" and
            pos_accessor.get("componentType") == 5126 and pos_accessor.get("type") == "VEC3" and
            uv_accessor["count"] == pos_accessor["count"] == section["vertex_count"],
            "screen UV/position accessor layout differs from reviewed inventory")
    uv_values = read_float_accessor(doc, source_bin, uv_accessor, 2)
    positions = read_float_accessor(doc, source_bin, pos_accessor, 3)
    measured_min = [min(row[axis] for row in uv_values) for axis in range(2)]
    measured_max = [max(row[axis] for row in uv_values) for axis in range(2)]
    if "source_bin_sha256" in section:
        require(sha256(source_bin) == section["source_bin_sha256"],
                "normalized source BIN bytes differ from the pinned orientation profile")
    for actual, expected in zip(measured_min + measured_max,
                                section["uv_min"] + section["uv_max"]):
        require(abs(actual - expected) <= 2e-6, "neutral derivative UV range differs from source audit")
    normalized = []
    max_affine_residual = 0.0
    rotation = section.get("rotation_degrees", 0)
    require(rotation in (0, 180), "UV rotation is outside the fixed profile")
    a, b, c, d, e, f = section["uv_to_position_xy_affine"]
    for uv, position in zip(uv_values, positions):
        require(all(math.isfinite(value) for value in uv + position), "screen accessor contains non-finite values")
        predicted_x, predicted_y = a * uv[0] + b * uv[1] + c, d * uv[0] + e * uv[1] + f
        max_affine_residual = max(max_affine_residual, math.hypot(predicted_x - position[0], predicted_y - position[1]))
        u = (uv[0] - measured_min[0]) / (measured_max[0] - measured_min[0])
        v = (uv[1] - measured_min[1]) / (measured_max[1] - measured_min[1])
        normalized.extend((1.0 - u, 1.0 - v) if rotation == 180 else (u, v))
    require(max_affine_residual <= section["max_uv_position_residual_m"],
            "screen UVs are not the pinned affine rectangular chart")
    require(min(normalized) >= -1e-7 and max(normalized) <= 1.0000001,
            "normalized screen UV values escape [0,1]")
    output_doc = copy.deepcopy(doc)
    referenced_uv = [(mesh_index, primitive_index, attribute)
                     for mesh_index, mesh in enumerate(doc["meshes"])
                     for primitive_index, primitive in enumerate(mesh.get("primitives", []))
                     for attribute, accessor_index in primitive.get("attributes", {}).items()
                     if accessor_index == uv_accessor_index]
    require(referenced_uv == [(binding["mesh_index"], binding["primitive_index"], "TEXCOORD_0")],
            "screen UV accessor is shared outside the pinned primitive")
    new_uv = len(output_doc["bufferViews"])
    new_bin = bytearray(source_bin)
    while len(new_bin) % 4:
        new_bin.append(0)
    byte_offset = len(new_bin)
    new_bin.extend(struct.pack("<" + "f" * len(normalized), *normalized))
    output_doc["bufferViews"].append({"buffer": 0, "byteOffset": byte_offset,
                                      "byteLength": len(normalized) * 4, "target": 34962})
    output_doc["accessors"][uv_accessor_index]["bufferView"] = new_uv
    output_doc["accessors"][uv_accessor_index].pop("byteOffset", None)
    output_doc["accessors"][uv_accessor_index]["min"] = [0.0, 0.0]
    output_doc["accessors"][uv_accessor_index]["max"] = [1.0, 1.0]
    output_doc["buffers"][0]["byteLength"] = len(new_bin)
    expected_doc = copy.deepcopy(doc)
    expected_doc["bufferViews"].append(copy.deepcopy(output_doc["bufferViews"][-1]))
    expected_doc["accessors"][uv_accessor_index]["bufferView"] = new_uv
    expected_doc["accessors"][uv_accessor_index].pop("byteOffset", None)
    expected_doc["accessors"][uv_accessor_index]["min"] = [0.0, 0.0]
    expected_doc["accessors"][uv_accessor_index]["max"] = [1.0, 1.0]
    expected_doc["buffers"][0]["byteLength"] = len(new_bin)
    require(output_doc == expected_doc, "normalization changed JSON outside the declared UV binding")
    json_bytes = json.dumps(output_doc, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    json_bytes += b" " * ((-len(json_bytes)) % 4)
    binary = bytes(new_bin)
    output = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(binary))
    output += struct.pack("<II", len(json_bytes), JSON_CHUNK) + json_bytes
    output += struct.pack("<II", len(binary), BIN_CHUNK) + binary
    require(len(output) <= manifest["output"]["max_bytes"], "normalized derivative exceeds fixed bound")
    check_doc, check_bin = parse_glb(output)
    expected_uv_sha = section.get("oriented_uv_sha256", section.get("normalized_uv_sha256"))
    require(check_doc == output_doc and sha256(buffer_view_bytes(check_doc, check_bin, new_uv)) == expected_uv_sha,
            "derived accessor serialization differs")
    require(check_bin[:len(source_bin)] == source_bin, "normalization changed existing BIN payload bytes")
    values = read_float_accessor(check_doc, check_bin, check_doc["accessors"][uv_accessor_index], 2)
    actual_min = [min(row[axis] for row in values) for axis in range(2)]
    actual_max = [max(row[axis] for row in values) for axis in range(2)]
    require(actual_min == [0.0, 0.0] and actual_max == [1.0, 1.0],
            "normalized UV accessor is not an exact unit rectangle")
    facts = {"source_uv_min": measured_min, "source_uv_max": measured_max,
                    "normalized_uv_min": actual_min, "normalized_uv_max": actual_max,
                    "max_uv_position_residual_m": max_affine_residual,
                    "source_uv_accessor": uv_accessor_index, "new_buffer_view": new_uv,
                    "new_uv_accessor_sha256": sha256(buffer_view_bytes(check_doc, check_bin, new_uv)),
                    "source_bin_sha256": sha256(source_bin),
                    "preserved_source_bin_prefix_sha256": sha256(check_bin[:len(source_bin)])}
    if rotation == 180:
        facts["rotation"] = "(u,v) -> (1-u,1-v)"
    return output, facts


def read_float_accessor(doc, binary, accessor, components):
    require(accessor["componentType"] == 5126 and accessor["type"] == {2: "VEC2", 3: "VEC3"}[components],
            "unsupported fixed float accessor")
    view = doc["bufferViews"][accessor["bufferView"]]
    stride = int(view.get("byteStride", components * 4))
    start = int(view.get("byteOffset", 0)) + int(accessor.get("byteOffset", 0))
    require(stride >= components * 4 and start >= 0 and
            start + max(0, int(accessor["count"]) - 1) * stride + components * 4 <= len(binary),
            "float accessor exceeds binary buffer")
    fmt = "<" + "f" * components
    return [struct.unpack_from(fmt, binary, start + i * stride) for i in range(accessor["count"])]


def validate_source_except_screen(doc, binary, source, target_image):
    images = doc.get("images", [])
    require(len(images) == len(source["embedded_images"]) and images[target_image].get("mimeType") == "image/png",
            "output image declarations differ")
    for image_index in (0, 2):
        expected = source["embedded_images"][image_index]
        payload = buffer_view_bytes(doc, binary, int(images[image_index]["bufferView"]))
        require(len(payload) == expected["size_bytes"] and sha256(payload) == expected["sha256"], "preserved image differs")
    return True


def read_manifest():
    require(MANIFEST_PATH.is_file() and not MANIFEST_PATH.is_symlink(), "fixed manifest must be a regular file")
    require(MANIFEST_PATH.stat().st_size <= 64 * 1024, "manifest exceeds fixed bound")
    raw = MANIFEST_PATH.read_bytes()
    require(len(raw) <= 64 * 1024, "manifest exceeds fixed bound")
    manifest = json.loads(raw.decode("utf-8"))
    require(manifest["profile_id"] == "device-studio-ipad-mini-source-preserving-starter-v1" and
            manifest["source"]["staged_name"] == SOURCE_NAME, "wrong fixed iPad profile")
    require(manifest["source"]["sha256"] == "a462088b7bd84b7752802c109beee04110047c3b1a4f355624c863f790e93d5e" and
            manifest["source"]["size_bytes"] == 1_986_244 and manifest["neutral_screen"]["width"] == 512 and
            manifest["neutral_screen"]["height"] == 1024, "fixed source or screen profile changed")
    return manifest, sha256(raw)


def main():
    if sys.argv[1:] == ["--orient-screen-uv-180"]:
        manifest, manifest_sha = read_uv_orientation_manifest()
        source_path = INPUTS / UV_ORIENTATION_SOURCE
        require(source_path.is_file() and not source_path.is_symlink() and
                source_path.stat().st_size == manifest["source"]["size_bytes"],
                "fixed normalized source must be a pinned regular file")
        source_data = source_path.read_bytes()
        output, facts = normalize_screen_uv(source_data, manifest)
        require(not UV_ORIENTATION_OUTPUTS.exists() and not UV_ORIENTATION_OUTPUTS.is_symlink(),
                "fixed oriented output directory already exists")
        UV_ORIENTATION_OUTPUTS.mkdir()
        target = UV_ORIENTATION_OUTPUTS / manifest["output"]["glb_name"]
        target.write_bytes(output)
        receipt = {"profile_id": manifest["profile_id"], "status": "screen_uv_oriented_180_derivative_unreviewed",
                   "source_sha256": manifest["source"]["sha256"], "source_size_bytes": len(source_data),
                   "manifest_sha256": manifest_sha, "script_sha256": sha256(Path(__file__).read_bytes()),
                   "screen_uv_orientation": facts, "derived_glb_sha256": sha256(output),
                   "derived_glb_size_bytes": len(output), "rights_and_claims": manifest["rights_and_claims"]}
        (UV_ORIENTATION_OUTPUTS / manifest["output"]["receipt_name"]).write_text(
            json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        return
    if sys.argv[1:] == ["--normalize-screen-uv"]:
        manifest, manifest_sha = read_uv_normalization_manifest()
        source_path = INPUTS / UV_NORMALIZATION_SOURCE
        require(source_path.is_file() and not source_path.is_symlink() and
                source_path.stat().st_size == manifest["source"]["size_bytes"],
                "fixed neutral source must be a pinned regular file")
        source_data = source_path.read_bytes()
        output, facts = normalize_screen_uv(source_data, manifest)
        require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed UV output directory already exists")
        OUTPUTS.mkdir()
        target = OUTPUTS / manifest["output"]["glb_name"]
        target.write_bytes(output)
        receipt = {"profile_id": manifest["profile_id"], "status": "screen_uv_normalized_derivative_unreviewed",
                   "source_sha256": manifest["source"]["sha256"], "source_size_bytes": len(source_data),
                   "manifest_sha256": manifest_sha, "script_sha256": sha256(Path(__file__).read_bytes()),
                   "screen_uv_normalization": facts, "derived_glb_sha256": sha256(output),
                   "derived_glb_size_bytes": len(output), "rights_and_claims": manifest["rights_and_claims"]}
        (OUTPUTS / manifest["output"]["receipt_name"]).write_text(
            json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        return
    require(not sys.argv[1:], "only fixed default, normalization, and 180-degree UV modes are supported")
    manifest, manifest_sha = read_manifest()
    source_path = INPUTS / SOURCE_NAME
    require(source_path.is_file() and not source_path.is_symlink(), "fixed staged GLB must be a regular file")
    require(source_path.stat().st_size == manifest["source"]["size_bytes"] <= MAX_SOURCE_BYTES,
            "staged source size differs or exceeds bound")
    source_data = source_path.read_bytes()
    require(len(source_data) == manifest["source"]["size_bytes"], "staged source size differs")
    output, screen_png, uv_facts, view_hashes = replace_screen_image(source_data, manifest)
    require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed output directory already exists")
    OUTPUTS.mkdir()
    output_path = OUTPUTS / manifest["output"]["glb_name"]
    output_path.write_bytes(output)
    receipt = {
        "profile_id": manifest["profile_id"], "status": "neutral_screen_derivative_unreviewed",
        "source_sha256": manifest["source"]["sha256"], "source_size_bytes": len(source_data),
        "manifest_sha256": manifest_sha, "script_sha256": sha256(Path(__file__).read_bytes()),
        "source_image_hashes": [row["sha256"] for row in manifest["source"]["embedded_images"]],
        "source_screen_sha256": manifest["source"]["embedded_images"][1]["sha256"],
        "derived_screen_sha256": sha256(screen_png), "derived_screen_size_bytes": len(screen_png),
        "derived_screen_dimensions": [manifest["neutral_screen"]["width"], manifest["neutral_screen"]["height"]],
        "derived_screen_rgba_srgb": manifest["neutral_screen"]["rgba_srgb"],
        "preserved_non_target_buffer_view_sha256": view_hashes,
        "screen_binding": manifest["source"]["screen_binding"], "measured_uv": uv_facts,
        "derived_glb_sha256": sha256(output), "derived_glb_size_bytes": len(output),
        "rights_and_claims": manifest["rights_and_claims"],
    }
    receipt_path = OUTPUTS / manifest["output"]["receipt_name"]
    receipt_path.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def read_uv_normalization_manifest():
    path = UV_NORMALIZATION_MANIFEST
    require(path.is_file() and not path.is_symlink() and path.stat().st_size <= 64 * 1024,
            "fixed UV normalization manifest must be a bounded regular file")
    raw = path.read_bytes()
    manifest = json.loads(raw.decode("utf-8"))
    require(manifest["profile_id"] == "device-studio-ipad-mini-screen-uv-normalization-v1" and
            manifest["source"]["staged_name"] == UV_NORMALIZATION_SOURCE and
            manifest["source"]["sha256"] == "8cd65349dcd38cb4f47fd897779db97dbbbf0fc7f6c81cd6c2ab05eaa6b47b5e" and
            manifest["source"]["size_bytes"] == 1938260 and
            manifest["source"]["neutral_screen_sha256"] == "3cf4703017eeeac32557bf6712cca55514457ff9f20ea6082f2185142b703cf6",
            "wrong pinned neutral derivative UV-normalization profile")
    return manifest, sha256(raw)


def read_uv_orientation_manifest():
    path = UV_ORIENTATION_MANIFEST
    require(path.is_file() and not path.is_symlink() and path.stat().st_size <= 64 * 1024,
            "fixed UV orientation manifest must be a bounded regular file")
    raw = path.read_bytes()
    manifest = json.loads(raw.decode("utf-8"))
    orientation = manifest["orientation"]
    require(manifest["profile_id"] == "device-studio-ipad-mini-screen-uv-orientation-180-v2" and
            manifest["source"]["staged_name"] == UV_ORIENTATION_SOURCE and
            manifest["source"]["sha256"] == "16d88943240dffae0e14e4743beefa1ed61e7efb4b5aa6cb00ecf2ac4b63749c" and
            manifest["source"]["size_bytes"] == 1938936 and
            manifest["source"]["neutral_screen_sha256"] == "3cf4703017eeeac32557bf6712cca55514457ff9f20ea6082f2185142b703cf6" and
            orientation["rotation_degrees"] == 180 and orientation["mapping"] == "(u,v) -> (1-u,1-v)" and
            orientation["source_bin_sha256"] == "296bdea52cb2d2637acc0f15b806844e530ac839d6ce396e327814a426da679b" and
            orientation["oriented_uv_sha256"] == "e8661ce4dcfe996a6b4ddfb6c957c1985227ca237120ef4a8070f255c5d3ee62",
            "wrong pinned iPad UV orientation profile")
    manifest["normalization"] = orientation
    manifest["normalization"]["normalized_uv_sha256"] = orientation["oriented_uv_sha256"]
    return manifest, sha256(raw)


if __name__ == "__main__":
    main()
