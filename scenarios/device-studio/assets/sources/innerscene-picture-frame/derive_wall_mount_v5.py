#!/usr/bin/env python3
"""Relocate only the Vrooli-authored wall cleat to the frame back for v5."""
from __future__ import annotations

import hashlib
import json
import pathlib
import struct

ROOT = pathlib.Path(__file__).resolve().parent
ASSET_DIR = ROOT.parents[2] / "api" / "internal" / "physicalruntime" / "assets" / "digital-frame"
SOURCE = ASSET_DIR / "innerscene-frame-wall-v4-glassless.glb"
OUTPUT = ASSET_DIR / "innerscene-frame-wall-v5-mounted.glb"
MANIFEST = ASSET_DIR / "asset-manifest-v5-wall-mounted.json"


def chunks(data: bytes):
    if len(data) < 20 or data[:4] != b"glTF" or struct.unpack_from("<I", data, 4)[0] != 2:
        raise ValueError("expected GLB 2.0")
    if struct.unpack_from("<I", data, 8)[0] != len(data):
        raise ValueError("GLB declared length mismatch")
    out, pos = [], 12
    while pos < len(data):
        size = struct.unpack_from("<I", data, pos)[0]
        kind = data[pos + 4:pos + 8]
        payload = data[pos + 8:pos + 8 + size]
        if len(payload) != size:
            raise ValueError("truncated GLB chunk")
        out.append((kind, payload))
        pos += 8 + size
    return out


def build(source: pathlib.Path = SOURCE, output: pathlib.Path = OUTPUT):
    raw = source.read_bytes()
    parts = chunks(raw)
    json_chunks = [i for i, (kind, _) in enumerate(parts) if kind == b"JSON"]
    bin_chunks = [i for i, (kind, _) in enumerate(parts) if kind == b"BIN\x00"]
    if len(json_chunks) != 1 or len(bin_chunks) != 1:
        raise ValueError("expected one JSON and one BIN chunk")
    ji, bi = json_chunks[0], bin_chunks[0]
    doc = json.loads(parts[ji][1].decode().rstrip(" \t\r\n\0"))
    matches = [(i, node) for i, node in enumerate(doc.get("nodes", []))
               if node.get("name") == "WallMountCleat"]
    if len(matches) != 1 or matches[0][1].get("mesh") is None:
        raise ValueError("expected exactly one mesh-backed WallMountCleat node")
    index, cleat = matches[0]
    translation = list(cleat.get("translation", (0, 0, 0)))
    if len(translation) != 3 or abs(translation[2] + 0.108) > 1e-6:
        raise ValueError("v4 cleat position changed; refusing an unreviewed derivation")
    mesh_index = cleat["mesh"]
    prim = doc["meshes"][mesh_index]["primitives"][0]
    accessor = doc["accessors"][prim["attributes"]["POSITION"]]
    low, high = accessor["min"][2], accessor["max"][2]
    if abs(low + 0.004) > 1e-6 or abs(high - 0.004) > 1e-6:
        raise ValueError("unexpected cleat local depth")
    # Frame rear bound is -9 mm. Put cleat front face at that same plane;
    # its rear face then extends 8 mm behind the rails for a plausible mount pad.
    rear_rail_z = -0.009
    translation[2] = rear_rail_z - high
    cleat["translation"] = translation
    doc["asset"]["generator"] = "Device Studio wall frame v5: source-preserving authored-cleat placement repair"
    encoded = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    encoded += b" " * ((-len(encoded)) % 4)
    out_chunks = []
    for i, (kind, payload) in enumerate(parts):
        if i == ji:
            payload = encoded
        out_chunks.append(struct.pack("<I", len(payload)) + kind + payload)
    body = b"".join(out_chunks)
    result = b"glTF" + struct.pack("<II", 2, 12 + len(body)) + body
    output.write_bytes(result)
    manifest = {
        "schema_version": 1,
        "status": "candidate_only_unqualified",
        "asset_id": "innerscene-frame-digital-neutral-wall-v5-mounted",
        "source": {
            "path": "innerscene-frame-wall-v4-glassless.glb",
            "bytes": len(raw),
            "sha256": hashlib.sha256(raw).hexdigest(),
            "license": "CC0-1.0 source; AGPL-3.0-only existing Vrooli wall-cleat/adaptation",
        },
        "derivative": {
            "generator": "scenarios/device-studio/assets/sources/innerscene-picture-frame/derive_wall_mount_v5.py",
            "runtime": "Python standard library; deterministic and offline",
            "path": output.name,
            "bytes": len(result),
            "sha256": hashlib.sha256(result).hexdigest(),
            "cleat_node_index": index,
            "cleat_mesh_index": mesh_index,
            "cleat_translation_m": translation,
            "changes": ["Only the existing Vrooli-authored WallMountCleat node translation changed; its front face now aligns with the frame-rail rear bound at z=-9 mm. All mesh/accessor/image/material records, screen geometry, UVs, and BIN bytes are unchanged."],
        },
        "wall_mount_pose": {
            "frame_rail_rear_bound_m": rear_rail_z,
            "cleat_local_z_bounds_m": [translation[2] + low, translation[2] + high],
            "room_wall_front_relative_to_support_frame_m": -0.0025,
            "desired_rear_clearance_m": 0.002,
            "device_root_rest_z_m": 0.0165,
            "derivation": "RoomBackWall center is z=-0.480 m with 35 mm thickness; support frame is at center+20 mm=-0.460 m. The wall front is -0.4625 m. Place the new cleat rear bound at wall front+2 mm, yielding root rest z=+16.5 mm from the support frame. Source-derived candidate pose, not a measured hardware mount or load rating.",
        },
        "display_binding": {
            "screen_node": "Unprinted frame insert",
            "screen_mesh_index": 7,
            "screen_material_index": 5,
            "uv_bounds": [[0, 0], [1, 1]],
            "screen_geometry_unchanged": True,
        },
        "qualification_limits": [
            "This is a generic CC0-source picture-frame adaptation, not a manufacturer electronic frame.",
            "The wall cleat is a Vrooli-authored candidate mount only; fastener design, substrate, load rating, and installation safety are not established.",
            "The 2 mm room-wall clearance is a source-scene placement assumption, not measured hardware clearance.",
        ],
    }
    MANIFEST.write_text(json.dumps(manifest, indent=2) + "\n")
    return result, manifest


if __name__ == "__main__":
    data, manifest = build()
    print(f"{OUTPUT.name} {len(data)} {manifest['derivative']['sha256']}")
