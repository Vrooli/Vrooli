#!/usr/bin/env python3
"""Flatten only broad strap-face normals on the immutable round-watch v2."""
import hashlib, json, struct
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SRC = ROOT / "api/internal/physicalruntime/assets/roundwatch-v2/generic-roundwatch-v2.glb"
OUT = ROOT / "api/internal/physicalruntime/assets/roundwatch-v3/generic-roundwatch-v3.glb"
MANIFEST = ROOT / "api/internal/physicalruntime/assets/roundwatch-v3/asset-manifest-v3.json"
EXPECTED_V2_SHA = "a609b435f5f4485504f1071e35cf74e95ba56095e5420aaa242255267c87750e"
STRAPS = ("lower charcoal silicone strap", "upper charcoal silicone strap")


def parse(raw):
    if raw[:4] != b"glTF" or struct.unpack_from("<II", raw, 4) != (2, len(raw)):
        raise ValueError("expected complete GLB 2.0")
    jl, jt = struct.unpack_from("<II", raw, 12)
    if jt != 0x4E4F534A:
        raise ValueError("missing JSON chunk")
    doc = json.loads(raw[20:20 + jl].decode().rstrip(" \0"))
    bo = 20 + jl
    bl, bt = struct.unpack_from("<II", raw, bo)
    if bt != 0x004E4942 or bo + 8 + bl != len(raw):
        raise ValueError("missing or malformed BIN chunk")
    return doc, bytearray(raw[bo + 8:bo + 8 + bl])


def accessor_location(doc, accessor_id):
    accessor = doc["accessors"][accessor_id]
    view = doc["bufferViews"][accessor["bufferView"]]
    if view.get("byteStride") is not None or accessor["componentType"] != 5126 or accessor["type"] != "VEC3":
        raise ValueError("expected tightly packed FLOAT VEC3")
    return accessor, view.get("byteOffset", 0) + accessor.get("byteOffset", 0)


def build():
    raw = SRC.read_bytes()
    source_sha = hashlib.sha256(raw).hexdigest()
    if source_sha != EXPECTED_V2_SHA:
        raise ValueError(f"v2 input changed: {source_sha}")
    doc, binary = parse(raw)
    straps, changed = {}, {}
    for name in STRAPS:
        nodes = [n for n in doc["nodes"] if n.get("name") == name]
        if len(nodes) != 1:
            raise ValueError(f"expected one {name} node")
        mesh = doc["meshes"][nodes[0]["mesh"]]["primitives"][0]
        pos, _ = accessor_location(doc, mesh["attributes"]["POSITION"])
        normals, no = accessor_location(doc, mesh["attributes"]["NORMAL"])
        if pos["count"] != 144 or normals["count"] != pos["count"] or pos["count"] % 4:
            raise ValueError(f"unexpected four-ring strap topology for {name}")
        ring = pos["count"] // 4
        straps[name] = nodes[0]["mesh"]
        changed[name] = 0
        # Cap rings are planar strap faces; keep their bevel/side rings separate in the normal data.
        for i in list(range(ring)) + list(range(3 * ring, 4 * ring)):
            sign = -1.0 if i < ring else 1.0
            before = struct.unpack_from("<3f", binary, no + 12 * i)
            after = (0.0, 0.0, sign)
            if before != after:
                struct.pack_into("<3f", binary, no + 12 * i, *after)
                changed[name] += 1
        if changed[name] != 2 * ring:
            raise ValueError(f"expected {2 * ring} planar cap normals to change for {name}; got {changed[name]}")

    doc["asset"]["generator"] = "Device Studio generic round-watch v3 planar silicone strap caps"
    doc["extras"] = {"v3": {"source": "generic-roundwatch-v2", "changedMeshes": straps, "flatCapNormalCounts": changed, "change": "Only the planar cap rings of the two silicone straps receive constant ±Z normals; geometry, indices, material, and display remain unchanged."}}
    jb = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    jb += b" " * (-len(jb) % 4)
    bb = bytes(binary) + b"\0" * (-len(binary) % 4)
    out = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(jb) + 8 + len(bb))
    out += struct.pack("<II", len(jb), 0x4E4F534A) + jb + struct.pack("<II", len(bb), 0x004E4942) + bb
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_bytes(out)
    manifest = {"schema_version": 1, "status": "candidate_only_unqualified", "artifact_id": "vrooli-generic-round-watch-concept-v3",
        "output": {"path": "scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-v3/generic-roundwatch-v3.glb", "sha256": hashlib.sha256(out).hexdigest(), "size_bytes": len(out)},
        "source": {"asset_id": "vrooli-generic-round-watch-concept-v2", "path": "scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-v2/generic-roundwatch-v2.glb", "sha256": source_sha, "license": "AGPL-3.0-only", "authored_by": "Vrooli"},
        "derivation": {"generator": "derive_roundwatch_v3.py", "runtime": "Python standard library; deterministic and offline", "changes": ["Set the two broad silicone strap cap rings to planar ±Z normals; retain original bevel/side normals and all GLB geometry, indices, material values, and display mapping."], "strap_nodes": list(STRAPS), "normals_changed_per_strap": next(iter(changed.values())), "screen_node": "One clean circular replaceable display", "screen_uv_orientation": "V=0 top"},
        "qualification_limits": ["Authored generic concept, not product CAD or identity.", "Candidate requires native appearance review; v2 remains unchanged."]}
    MANIFEST.parent.mkdir(parents=True, exist_ok=True)
    MANIFEST.write_text(json.dumps(manifest, separators=(",", ":")) + "\n")
    return manifest


if __name__ == "__main__":
    m = build()
    print(m["output"]["size_bytes"], m["output"]["sha256"])
