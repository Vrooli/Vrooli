#!/usr/bin/env python3
"""Create an unstrapped floating watch-body candidate from immutable v3 bytes."""
import hashlib, json, struct
from pathlib import Path
ROOT = Path(__file__).resolve().parents[3]
SRC = ROOT / "api/internal/physicalruntime/assets/roundwatch-v3/generic-roundwatch-v3.glb"
OUT = ROOT / "api/internal/physicalruntime/assets/roundwatch-body-v4/generic-roundwatch-body-v4.glb"
MANIFEST = ROOT / "api/internal/physicalruntime/assets/roundwatch-body-v4/asset-manifest-body-v4.json"
EXPECTED_SOURCE_SHA = "faa7f39f305c77d77078080599cecdf2df4c2af3d28a06c2763c2707c1a7188c"
OMIT = {"lower lug bridge", "lower charcoal silicone strap", "lower strap inner contact", "upper lug bridge", "upper charcoal silicone strap", "upper strap inner contact", "Satin strap buckle crossbar", "Satin strap buckle side"}
def build():
    raw = SRC.read_bytes(); source_sha = hashlib.sha256(raw).hexdigest()
    if source_sha != EXPECTED_SOURCE_SHA: raise ValueError(f"v3 input changed: {source_sha}")
    if raw[:4] != b"glTF" or struct.unpack_from("<II", raw, 4) != (2, len(raw)): raise ValueError("expected complete GLB 2.0")
    jl, jt = struct.unpack_from("<II", raw, 12)
    if jt != 0x4E4F534A: raise ValueError("missing JSON chunk")
    doc = json.loads(raw[20:20+jl].decode().rstrip(" \0")); bo = 20 + jl
    bl, bt = struct.unpack_from("<II", raw, bo)
    if bt != 0x004E4942 or bo + 8 + bl != len(raw): raise ValueError("missing or malformed BIN chunk")
    nodes = doc["nodes"]; children = nodes[0].get("children", [])
    remove = [i for i in children if nodes[i].get("name") in OMIT]
    removed = [nodes[i].get("name", "") for i in remove]
    if len(remove) != 9 or not OMIT.issubset(set(removed)): raise ValueError(f"unexpected watch strap assembly: {removed}")
    remove_set = set(remove)
    kept = [i for i in range(len(nodes)) if i not in remove_set]
    remap = {old: new for new, old in enumerate(kept)}
    reduced = [nodes[i] for i in kept]
    for node in reduced:
        if "children" in node:
            node["children"] = [remap[index] for index in node["children"] if index in remap]
    doc["nodes"] = nodes = reduced
    doc["asset"]["generator"] = "Device Studio generic floating round-watch body v4"
    doc["extras"] = {"v4": {"source":"generic-roundwatch-v3", "change":"The active scene omits only upper/lower lug, strap, strap-contact and buckle nodes. Case, circular display, crown, buttons and all source meshes/materials/BIN bytes remain.", "omittedActiveNodes":removed, "pose":"floating watch body; no strap or attachment is represented"}}
    jb = json.dumps(doc, separators=(",",":"), ensure_ascii=True).encode(); jb += b" " * (-len(jb)%4)
    binary = raw[bo+8:bo+8+bl]
    out = struct.pack("<4sII",b"glTF",2,12+8+len(jb)+8+len(binary))+struct.pack("<II",len(jb),0x4E4F534A)+jb+struct.pack("<II",len(binary),0x004E4942)+binary
    OUT.parent.mkdir(parents=True,exist_ok=True); OUT.write_bytes(out)
    manifest = {"schema_version":1,"status":"candidate_only_unqualified","artifact_id":"vrooli-generic-round-watch-body-concept-v4","output":{"path":"scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-body-v4/generic-roundwatch-body-v4.glb","sha256":hashlib.sha256(out).hexdigest(),"size_bytes":len(out)},"source":{"asset_id":"vrooli-generic-round-watch-concept-v3","path":"scenarios/device-studio/api/internal/physicalruntime/assets/roundwatch-v3/generic-roundwatch-v3.glb","sha256":source_sha,"license":"AGPL-3.0-only","authored_by":"Vrooli"},"derivation":{"generator":"derive_roundwatch_body_v4.py","runtime":"Python standard library; deterministic and offline","changes":["Active hierarchy omits strap, buckle, inner-contact and lug nodes only; all mesh/material/accessor/binary records remain retained."],"omitted_active_nodes":removed,"preserved_nodes":["Rounded monocoque case with soft bevel","Continuous satin circular bezel","One clean circular replaceable display","Recessed digital crown","Flush lower action button"],"pose":"Floating body-only review pose; no strap, stand, mount, or physical attachment is claimed.","screen_uv_orientation":"V=0 top"},"qualification_limits":["Authored generic concept, not product CAD or identity.","The candidate represents the watch body without a strap; it is not a complete wearable watch.","Candidate requires native appearance review; v3 remains unchanged."]}
    MANIFEST.write_text(json.dumps(manifest,separators=(",",":"))+"\n"); print(manifest["output"]["size_bytes"],manifest["output"]["sha256"])
if __name__ == "__main__": build()
