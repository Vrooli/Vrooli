#!/usr/bin/env python3
"""Create a reachable-node neutral wall scene; preserve v1 bytes and source data."""
import hashlib, json, struct
from pathlib import Path
ROOT = Path(__file__).resolve().parents[3]
SOURCE = ROOT / "api/internal/physicalruntime/assets/laptop-study-environment/laptop-study-environment-no-cup.glb"
OUTPUT = ROOT / "api/internal/physicalruntime/assets/vehicle-studio/vehicle-neutral-studio-wall-v2.glb"
MANIFEST = ROOT / "api/internal/physicalruntime/assets/vehicle-studio/asset-manifest-v2.json"
SOURCE_SHA = "68a8ad34f5af76f681e9aab7337ce80514221bda333c6713b58f624556660f56"
def build():
    source=SOURCE.read_bytes()
    if hashlib.sha256(source).hexdigest()!=SOURCE_SHA: raise ValueError("pinned room source changed")
    if source[:4]!=b"glTF" or struct.unpack_from("<II",source,4)!=(2,len(source)): raise ValueError("expected GLB 2.0")
    jl,jt=struct.unpack_from("<II",source,12)
    if jt!=0x4E4F534A: raise ValueError("missing JSON chunk")
    doc=json.loads(source[20:20+jl].decode().rstrip(" \0")); bo=20+jl
    bl,bt=struct.unpack_from("<II",source,bo)
    if bt!=0x004E4942 or bo+8+bl!=len(source): raise ValueError("malformed source BIN")
    nodes=doc.get("nodes",[])
    walls=[i for i,node in enumerate(nodes) if node.get("name")=="RoomBackWall"]
    if len(walls)!=1 or len(doc.get("scenes",[]))!=1: raise ValueError("expected one pinned wall in one-scene room")
    wall=nodes[walls[0]]
    if wall.get("children"): raise ValueError("wall node unexpectedly owns child nodes")
    doc["nodes"]=[wall]
    doc["scenes"][0]["nodes"]=[0]
    doc["scene"]=0
    doc["asset"]["generator"]="Device Studio vehicle-neutral studio wall context v2"
    doc["extras"]={"vehicle_studio_context_v2":{"source":"Vrooli-authored laptop-study room candidate","active_scene_nodes":["RoomBackWall"],"omitted_nodes":[n.get("name","") for i,n in enumerate(nodes) if i!=walls[0]],"change":"Retain only the RoomBackWall node in the glTF node table and active scene. Keep all source meshes, materials, accessors and BIN bytes; this avoids unreachable inactive nodes in scene validation.","limits":"Neutral studio backdrop only; it is not an automobile interior or installation/fit claim."}}
    jb=json.dumps(doc,separators=(",",":"),ensure_ascii=True).encode(); jb+=b" "*(-len(jb)%4)
    binary=source[bo+8:bo+8+bl]
    out=struct.pack("<4sII",b"glTF",2,12+8+len(jb)+8+len(binary))+struct.pack("<II",len(jb),0x4E4F534A)+jb+struct.pack("<II",len(binary),0x004E4942)+binary
    OUTPUT.parent.mkdir(parents=True,exist_ok=True); OUTPUT.write_bytes(out)
    manifest={"schema_version":1,"status":"candidate_only_unqualified","asset_id":"vehicle-neutral-studio-wall-v2","output":{"path":"scenarios/device-studio/api/internal/physicalruntime/assets/vehicle-studio/vehicle-neutral-studio-wall-v2.glb","sha256":hashlib.sha256(out).hexdigest(),"size_bytes":len(out)},"source":{"asset_id":"device-studio-laptop-candidate-room-no-cup-v1","sha256":SOURCE_SHA,"license":"AGPL-3.0-only","authored_by":"Vrooli"},"derivation":{"generator":"derive_vehicle_studio_context_v2.py","runtime":"Python standard library; deterministic and offline","active_nodes":["RoomBackWall"],"omitted_node_records":[n.get("name","") for i,n in enumerate(nodes) if i!=walls[0]],"preserved":"The wall node, all meshes/materials/accessors and the complete source BIN are retained. Only the node table and scene roots are reduced so every node is reachable."},"limits":["Neutral studio background only; no automobile interior, install fit, or vehicle compatibility claim.","Wall support and device placement remain candidate-only and require native visual review."]}
    MANIFEST.write_text(json.dumps(manifest,separators=(",",":"))+"\n"); print(manifest["output"]["size_bytes"],manifest["output"]["sha256"])
if __name__=="__main__": build()
