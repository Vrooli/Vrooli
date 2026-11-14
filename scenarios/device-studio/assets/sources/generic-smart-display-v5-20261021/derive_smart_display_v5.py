#!/usr/bin/env python3
"""Build smart-display v5 with sufficient rounded-shell contour tessellation."""
from __future__ import annotations

import hashlib
import json
import pathlib
import runpy
import struct
import tempfile

ROOT = pathlib.Path(__file__).resolve().parent
SOURCE_SCRIPT = ROOT.parent / "generic-smart-display-v3-20261021" / "derive_smart_display.py"
OUTPUT = ROOT.parents[2] / "api" / "internal" / "physicalruntime" / "assets" / "smart-display-v5" / "generic-smart-display-v5.glb"
MANIFEST = ROOT.parents[2] / "api" / "internal" / "physicalruntime" / "assets" / "smart-display-v5" / "asset-manifest-v5.json"


def glb_parts(data: bytes):
    if len(data) < 20 or data[:4] != b"glTF" or struct.unpack_from("<I", data, 4)[0] != 2 or struct.unpack_from("<I", data, 8)[0] != len(data):
        raise ValueError("expected complete GLB 2.0")
    chunks, pos = [], 12
    while pos < len(data):
        size = struct.unpack_from("<I", data, pos)[0]
        kind, payload = data[pos + 4:pos + 8], data[pos + 8:pos + 8 + size]
        if len(payload) != size:
            raise ValueError("truncated GLB chunk")
        chunks.append((kind, payload))
        pos += size + 8
    return chunks


def rewrite_json(chunks, transform):
    indexes = [i for i, (kind, _) in enumerate(chunks) if kind == b"JSON"]
    if len(indexes) != 1:
        raise ValueError("expected one GLB JSON chunk")
    index = indexes[0]
    document = json.loads(chunks[index][1].decode().rstrip(" \t\r\n\0"))
    transform(document)
    encoded = json.dumps(document, separators=(",", ":"), ensure_ascii=True).encode()
    encoded += b" " * ((-len(encoded)) % 4)
    body = b"".join(struct.pack("<I", len(encoded) if i == index else len(payload)) + kind + (encoded if i == index else payload)
                     for i, (kind, payload) in enumerate(chunks))
    return b"glTF" + struct.pack("<II", 2, 12 + len(body)) + body, document


def build():
    original = SOURCE_SCRIPT.read_text()
    source_sha = hashlib.sha256(original.encode()).hexdigest()
    out_line = 'OUT = ROOT / "generic-smart-display-v3.glb"'
    shell_call = 'rounded_prism("Display rear shell",.1714,.1025,.0150,screen_y,screen_z,.0125,tilt,0,12)'
    if original.count(out_line) != 1 or original.count(shell_call) != 1:
        raise ValueError("v3 deterministic source recipe changed; review the v5 derivation")
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="device-studio-smart-v5-") as temp:
        script = pathlib.Path(temp) / "generate.py"
        generated = pathlib.Path(temp) / "smart-display.glb"
        recipe = original.replace(out_line, f"OUT = pathlib.Path({str(generated)!r})", 1)
        recipe = recipe.replace(shell_call, shell_call[:-3] + "48)", 1)
        script.write_text(recipe)
        runpy.run_path(str(script), run_name="__main__")
        raw = generated.read_bytes()
    chunks = glb_parts(raw)

    def adjust(doc):
        nodes = [(i, node) for i, node in enumerate(doc.get("nodes", [])) if node.get("name") == "Display rear shell"]
        if len(nodes) != 1 or nodes[0][1].get("mesh") != 4 or len(doc.get("meshes", [])) != 10:
            raise ValueError("v3 generated smart-display structure changed")
        mesh = doc["meshes"][4]
        count = doc["accessors"][mesh["primitives"][0]["attributes"]["POSITION"]]["count"]
        if count < 2400:
            raise ValueError(f"shell subdivision did not increase sufficiently: {count} vertices")
        material = doc["materials"][0]["pbrMetallicRoughness"]
        material["metallicFactor"] = 0.04
        material["roughnessFactor"] = 0.56
        doc["asset"]["generator"] = "Device Studio generic smart display v5 rounded-shell tessellation derivative"
        doc["extras"] = {"v5": {"shellNode": nodes[0][0], "shellMesh": 4, "shellPositionCount": count,
                                  "screenNode": "MainDisplayActivePlane", "screenMesh": 6,
                                  "shellRoundedRectSegmentsPerCorner": 12}}

    result, doc = rewrite_json(chunks, adjust)
    OUTPUT.write_bytes(result)
    screen = [(i, n) for i, n in enumerate(doc["nodes"]) if n.get("name") == "MainDisplayActivePlane"]
    if len(screen) != 1 or screen[0][1].get("mesh") != 6:
        raise ValueError("screen node changed during derivation")
    manifest = {
        "schema_version": 1,
        "status": "candidate_only_unqualified",
        "artifact_id": "vrooli-generic-smart-display-concept-v5",
        "output": {"path": "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v5/generic-smart-display-v5.glb",
                   "sha256": hashlib.sha256(result).hexdigest(), "size_bytes": len(result)},
        "source": {"asset_id": "vrooli-generic-smart-display-concept-v3",
                   "path": "scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v3/generic-smart-display-v3.glb",
                   "sha256": "9890e82e4ded55f098fb2eb99548bb3d0331c1a599193ee2041c9773a338cf50",
                   "generator_sha256": source_sha, "license": "AGPL-3.0-only", "authored_by": "Vrooli"},
        "derivation": {"generator": "scenarios/device-studio/assets/sources/generic-smart-display-v5-20261021/derive_smart_display_v5.py",
                       "runtime": "Python standard library; deterministic and offline",
                       "changes": ["Rebuild only the Display rear shell using 48 rounded-rectangle segments per corner (v3 uses 12), eliminating the three-facet-per-corner contour; all other component geometry and all display UVs/bindings match the original v3 recipe.",
                                   "Set graphite polymer metallicFactor to 0.04 and roughnessFactor to 0.56, matching the v4 lower-specularity treatment."],
                       "shell_node_index": 6, "shell_mesh_index": 4, "shell_position_count": doc["extras"]["v5"]["shellPositionCount"],
                       "screen_node_index": screen[0][0], "screen_node_name": "MainDisplayActivePlane", "screen_mesh_index": 6,
                       "screen_primitive_index": 0, "screen_material_index": 2, "uv_orientation": "V=0 top; V=1 bottom"},
        "design_envelope_mm": [177.4, 120.4, 69.5],
        "qualification_limits": ["Generic authored concept; Google reference envelope is comparison-only, not product CAD or identity.",
                                 "Candidate requires a native appearance review; no professional-appearance or product qualification is claimed."]}
    MANIFEST.write_text(json.dumps(manifest, indent=2) + "\n")
    return result, manifest


if __name__ == "__main__":
    data, manifest = build()
    print(f"{OUTPUT.name} {len(data)} {manifest['output']['sha256']}")
