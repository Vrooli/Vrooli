#!/usr/bin/env python3
"""Keep only the neutral room wall in a versioned dashboard studio scene."""
import hashlib
import json
import struct
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SOURCE = ROOT / "api/internal/physicalruntime/assets/laptop-study-environment/laptop-study-environment-no-cup.glb"
OUTPUT = ROOT / "api/internal/physicalruntime/assets/vehicle-studio/vehicle-neutral-studio-wall-v1.glb"
MANIFEST = ROOT / "api/internal/physicalruntime/assets/vehicle-studio/asset-manifest-v1.json"
SOURCE_SHA = "68a8ad34f5af76f681e9aab7337ce80514221bda333c6713b58f624556660f56"


def build():
    source = SOURCE.read_bytes()
    if hashlib.sha256(source).hexdigest() != SOURCE_SHA:
        raise ValueError("pinned cup-free room source changed")
    if source[:4] != b"glTF" or struct.unpack_from("<II", source, 4) != (2, len(source)):
        raise ValueError("expected complete GLB 2.0")
    json_size, json_type = struct.unpack_from("<II", source, 12)
    if json_type != 0x4E4F534A:
        raise ValueError("missing JSON chunk")
    doc = json.loads(source[20:20 + json_size].decode().rstrip(" \0"))
    if len(doc.get("scenes", [])) != 1 or len(doc.get("nodes", [])) != 11:
        raise ValueError("unexpected source room hierarchy")
    walls = [i for i, node in enumerate(doc["nodes"]) if node.get("name") == "RoomBackWall"]
    if len(walls) != 1:
        raise ValueError("expected one neutral back-wall node")
    doc["scenes"][0]["nodes"] = walls
    doc["asset"]["generator"] = "Device Studio vehicle-neutral studio wall context v1"
    doc["extras"] = {"vehicle_studio_context_v1": {
        "source": "Vrooli-authored laptop-study room candidate",
        "active_scene_nodes": ["RoomBackWall"],
        "omitted_context": ["table legs", "table top", "window frame and light panel"],
        "change": "Only the active scene root list changes; all node, mesh, material, accessor, and BIN records are retained.",
        "limits": "Neutral studio backdrop only; it is not an automobile interior or an installation/fit claim."
    }}
    json_bytes = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    json_bytes += b" " * (-len(json_bytes) % 4)
    offset = 20 + json_size
    bin_size, bin_type = struct.unpack_from("<II", source, offset)
    if bin_type != 0x004E4942 or offset + 8 + bin_size != len(source):
        raise ValueError("malformed source BIN chunk")
    binary = source[offset + 8:offset + 8 + bin_size]
    out = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(binary))
    out += struct.pack("<II", len(json_bytes), 0x4E4F534A) + json_bytes
    out += struct.pack("<II", len(binary), 0x004E4942) + binary
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_bytes(out)
    manifest = {"schema_version": 1, "status": "candidate_only_unqualified", "asset_id": "vehicle-neutral-studio-wall-v1",
        "output": {"path": "scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-studio/vehicle-neutral-studio-wall-v1.glb", "sha256": hashlib.sha256(out).hexdigest(), "size_bytes": len(out)},
        "source": {"asset_id": "device-studio-laptop-candidate-room-no-cup-v1", "sha256": SOURCE_SHA, "license": "AGPL-3.0-only", "authored_by": "Vrooli"},
        "derivation": {"generator": "derive_vehicle_studio_context.py", "runtime": "Python standard library; deterministic and offline", "active_nodes": ["RoomBackWall"], "omitted_from_active_scene": ["table legs", "table top", "window frame and light panel"], "preserved": "All source node, mesh, material, accessor and BIN records; only the active scene root list changes."},
        "limits": ["Neutral studio background only; no automobile interior, install fit, or vehicle compatibility claim.", "The wall support and device placement remain candidate-only and require native visual review."]}
    MANIFEST.write_text(json.dumps(manifest, separators=(",", ":")) + "\n")
    return manifest


if __name__ == "__main__":
    result = build()
    print(result["output"]["size_bytes"], result["output"]["sha256"])
