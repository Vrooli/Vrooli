#!/usr/bin/env python3
"""Append a private 180-degree UV view for the derive16 laptop screen only."""

import hashlib
import json
import math
import pathlib
import struct

ROOT = pathlib.Path(__file__).resolve().parents[3]
SOURCE = ROOT / "api/internal/physicalruntime/assets/laptop/laptop-neutral-rig.glb"
OUTPUT = ROOT / "api/internal/physicalruntime/assets/laptop/laptop-neutral-rig-screen-upright-180-v1.glb"
SOURCE_SHA = "3005e1e8b12e2017565292712aaad20dfb35cda2c42519a937d2c74afb6f5925"
SCREEN_NODE = "LaptopScreenSurface"


def chunks(data):
    if len(data) < 20 or struct.unpack_from("<III", data) != (0x46546C67, 2, len(data)):
        raise ValueError("expected a complete glTF 2.0 binary")
    cursor = 12
    result = []
    while cursor < len(data):
        size, kind = struct.unpack_from("<II", data, cursor)
        cursor += 8
        payload = data[cursor : cursor + size]
        if len(payload) != size:
            raise ValueError("truncated GLB chunk")
        result.append((kind, payload))
        cursor += size
    if cursor != len(data) or len(result) != 2 or result[0][0] != 0x4E4F534A or result[1][0] != 0x004E4942:
        raise ValueError("expected exactly JSON followed by BIN")
    return result


def encode(document, binary):
    raw_json = json.dumps(document, separators=(",", ":")).encode()
    raw_json += b" " * ((4 - len(raw_json) % 4) % 4)
    binary += b"\0" * ((4 - len(binary) % 4) % 4)
    total = 12 + 8 + len(raw_json) + 8 + len(binary)
    return struct.pack("<III", 0x46546C67, 2, total) + struct.pack("<II", len(raw_json), 0x4E4F534A) + raw_json + struct.pack("<II", len(binary), 0x004E4942) + binary


def derive(data):
    if hashlib.sha256(data).hexdigest() != SOURCE_SHA:
        raise ValueError("derive16 source candidate hash changed")
    json_kind, json_bytes = chunks(data)[0]
    _, original_bin = chunks(data)[1]
    document = json.loads(json_bytes)
    nodes = [node for node in document["nodes"] if node.get("name") == SCREEN_NODE]
    if len(nodes) != 1:
        raise ValueError("derive16 screen node is not unique")
    mesh_index = nodes[0].get("mesh")
    primitives = document["meshes"][mesh_index]["primitives"]
    if len(primitives) != 1:
        raise ValueError("derive16 screen mesh must have one primitive")
    attributes = primitives[0]["attributes"]
    if (attributes.get("POSITION"), attributes.get("NORMAL"), primitives[0].get("indices")) != (92, 93, 95):
        raise ValueError("derive16 screen geometry accessors changed")
    old_uv_index = attributes.get("TEXCOORD_0")
    old_uv = document["accessors"][old_uv_index]
    if old_uv.get("componentType") != 5126 or old_uv.get("type") != "VEC2" or old_uv.get("count") != 95:
        raise ValueError("derive16 screen UV accessor differs from the reviewed inventory")
    view = document["bufferViews"][old_uv["bufferView"]]
    if view.get("buffer") != 0 or view.get("byteLength") != old_uv["count"] * 8:
        raise ValueError("screen UV data is not a compact private accessor")
    offset = view.get("byteOffset", 0)
    source_uv = struct.unpack_from("<" + "f" * (old_uv["count"] * 2), original_bin, offset)
    if not all(math.isfinite(value) and 0 <= value <= 1 for value in source_uv):
        raise ValueError("normalized screen UVs are invalid")
    transformed = []
    for u, v in zip(source_uv[::2], source_uv[1::2]):
        transformed.extend((1.0 - u, 1.0 - v))
    private_bin = bytearray(original_bin)
    private_bin.extend(b"\0" * ((4 - len(private_bin) % 4) % 4))
    new_offset = len(private_bin)
    private_bin.extend(struct.pack("<" + "f" * len(transformed), *transformed))
    view_index = len(document["bufferViews"])
    document["bufferViews"].append({"buffer": 0, "byteOffset": new_offset, "byteLength": len(transformed) * 4, "target": 34962})
    new_accessor = dict(old_uv)
    new_accessor["bufferView"] = view_index
    new_accessor["min"], new_accessor["max"] = [0.0, 0.0], [1.0, 1.0]
    attributes["TEXCOORD_0"] = len(document["accessors"])
    document["accessors"].append(new_accessor)
    document["buffers"][0]["byteLength"] = len(private_bin)
    return encode(document, bytes(private_bin)), original_bin


def main():
    source = SOURCE.read_bytes()
    result, original_bin = derive(source)
    OUTPUT.write_bytes(result)
    print(json.dumps({"source_sha256": hashlib.sha256(source).hexdigest(), "source_bin_prefix_sha256": hashlib.sha256(original_bin).hexdigest(), "output_sha256": hashlib.sha256(result).hexdigest(), "output_bytes": len(result)}, sort_keys=True))


if __name__ == "__main__":
    main()
