#!/usr/bin/env python3
"""Create visible-screen digital-frame variants by detaching only the source glazing node."""
from __future__ import annotations
import hashlib, json, pathlib, struct

ROOT=pathlib.Path(__file__).resolve().parent
ASSET_DIR=ROOT.parents[2]/"api"/"internal"/"physicalruntime"/"assets"/"digital-frame"
MANIFEST=ASSET_DIR/"asset-manifest-v4.json"
VARIANTS=(
    {"profile":"tabletop","input":"innerscene-frame-tabletop-v2.glb","input_sha":"bc6b22fb722672b0601dc2ea4424517e6bed589e3ebf40de9ecd0257d20fb4f4","output":"innerscene-frame-tabletop-v4-glassless.glb","parent_id":"innerscene-frame-digital-neutral-tabletop-v2","id":"innerscene-frame-digital-neutral-tabletop-v4-glassless","device_id":"innerscene-digital-frame-tabletop-candidate-v4-glassless","device_name":"innerscene-digital-frame-tabletop-candidate-v4-glassless"},
    {"profile":"wall","input":"innerscene-frame-wall-v3.glb","input_sha":"86c216e9bc574ac076bc78231107cb53b4c178bdea4121239df43ca8963e17bf","output":"innerscene-frame-wall-v4-glassless.glb","parent_id":"innerscene-frame-digital-neutral-wall-v3","id":"innerscene-frame-digital-neutral-wall-v4-glassless","device_id":"innerscene-digital-frame-wall-candidate-v4-glassless","device_name":"innerscene-digital-frame-wall-candidate-v4-glassless"},
)

def parse_glb(data:bytes):
    if len(data)<20 or data[:4]!=b"glTF" or struct.unpack_from("<I",data,4)[0]!=2 or struct.unpack_from("<I",data,8)[0]!=len(data):
        raise ValueError("expected complete GLB 2.0")
    chunks=[]; pos=12
    while pos<len(data):
        size=struct.unpack_from("<I",data,pos)[0]; kind=data[pos+4:pos+8]; payload=data[pos+8:pos+8+size]
        if len(payload)!=size: raise ValueError("truncated GLB chunk")
        chunks.append((kind,payload));pos+=8+size
    json_indexes=[i for i,(kind,_) in enumerate(chunks) if kind==b"JSON"]
    bin_indexes=[i for i,(kind,_) in enumerate(chunks) if kind==b"BIN\x00"]
    if len(json_indexes)!=1 or len(bin_indexes)!=1: raise ValueError("expected one JSON and one BIN chunk")
    ji,bi=json_indexes[0],bin_indexes[0]; doc=json.loads(chunks[ji][1].decode().rstrip(" \t\r\n\0"))
    return chunks,ji,bi,doc

def derive(spec):
    src=ASSET_DIR/spec["input"]; out=ASSET_DIR/spec["output"]; raw=src.read_bytes()
    source_sha=hashlib.sha256(raw).hexdigest()
    if source_sha!=spec["input_sha"]: raise ValueError(f"unexpected parent GLB hash for {src.name}: {source_sha}")
    chunks,ji,bi,doc=parse_glb(raw)
    named=[(i,n) for i,n in enumerate(doc["nodes"]) if n.get("name")=="Frame Stand"]
    glass=[(i,n) for i,n in enumerate(doc["nodes"]) if n.get("name")=="Clear protective picture glazing"]
    if len(named)!=1 or len(glass)!=1: raise ValueError("expected one Frame Stand root and one protective glazing node")
    root_i,root=named[0]; glass_i,glass_node=glass[0]
    if root_i not in doc["scenes"][doc.get("scene",0)]["nodes"] or glass_i not in root.get("children",[]):
        raise ValueError("frame hierarchy changed; refuse to detach a different node")
    if glass_i!=0 or glass_node.get("mesh")!=0 or len(doc["meshes"][0]["primitives"])!=1 or doc["meshes"][0]["primitives"][0].get("material")!=0:
        raise ValueError("glazing geometry/material contract changed")
    old_root_i=root_i
    # The rendering validator requires every node to be reachable. Remove the single
    # glazing node from the node list, remap only node/child/scene indices, and leave
    # its mesh/material/image records and the complete BIN payload intact.
    def mapped(index):
        if index==glass_i: raise ValueError("unexpected reference to removed glazing node")
        return index-1 if index>glass_i else index
    doc["nodes"]=[node for i,node in enumerate(doc["nodes"]) if i!=glass_i]
    for node in doc["nodes"]:
        node["children"]=[mapped(child) for child in node.get("children",[]) if child!=glass_i]
    for scene in doc["scenes"]:
        scene["nodes"]=[mapped(node) for node in scene.get("nodes",[]) if node!=glass_i]
    root_i=mapped(root_i)
    doc["asset"]["generator"]="Device Studio digital frame v4 visible-screen derivative; protective glazing node omitted"
    encoded=json.dumps(doc,separators=(",",":"),ensure_ascii=True).encode(); encoded+=b" "*((-len(encoded))%4)
    out_chunks=[]
    for i,(kind,payload) in enumerate(chunks):
        if i==ji: payload=encoded
        out_chunks.append(struct.pack("<I",len(payload))+kind+payload)
    body=b"".join(out_chunks); derived=b"glTF"+struct.pack("<II",2,12+len(body))+body
    out.write_bytes(derived)
    return {**spec,"source_bytes":len(raw),"source_sha256":source_sha,"derived_bytes":len(derived),"derived_sha256":hashlib.sha256(derived).hexdigest(),"root_node_index":root_i,"parent_root_node_index":old_root_i,"removed_glazing_node_index":glass_i,"retained_unreferenced_glazing_mesh_index":0,"retained_glazing_material_index":0,"retained_unreferenced_mesh_material_records":True,"screen_node_index":6 if spec["profile"]=="tabletop" else 4,"screen_mesh_index":7,"screen_material_index":5,"binary_chunk_sha256":hashlib.sha256(chunks[bi][1]).hexdigest(),"binary_chunk_byte_identical_to_parent":True}

def main():
    results=[derive(v) for v in VARIANTS]
    source="scenarios/device-studio/assets/sources/innerscene-picture-frame/derive_visible_frame.py"
    manifest={"schema_version":1,"status":"candidate_only_unqualified","derivative":"generic digital-frame variants with the high-transmission source glazing node removed; its original mesh/material/image records and complete BIN payload remain preserved but unreferenced","generator":source,"runtime":"Python standard library","deterministic":True,"adaptation_license":"CC0-1.0 for retained Innerscene source content; AGPL-3.0-only for Vrooli-authored candidate adaptation and the existing wall cleat.","source":{"name":"Innerscene Frame Stand - Picture Frame","path":"scenarios/device-studio/assets/sources/innerscene-picture-frame/picture-frame.glb","sha256":"7c9f5cf3e63c7f01cc911b1db9eea2c95b5e1bdadd9b75c3076b10dca70cef10","license":"CC0-1.0 as stated by source package/source page","original_geometry_and_source_file_preserved":True},"variants":results,"change_scope":"For each parent variant, remove only the Clear protective picture glazing node from the nodes array and remap references to the remaining nodes. The glazing mesh/material/image records, all BIN bytes, inserts, support geometry, screen UVs and every remaining node record are retained unchanged.","display_binding":{"screen_name":"Unprinted frame insert","screen_mesh_index":7,"screen_material_index":5,"uv_bounds":[[0,0],[1,1]],"screen_target_unchanged":True},"claim_limits":["The active digital-frame render omits the protective pane to preserve screen legibility in the current native presentation path; its source mesh/material/image records remain unreferenced in the GLB and the complete BIN payload is unchanged.","Not an Innerscene electronic product or measured digital frame.","Unqualified candidate geometry; tabletop support stability and wall fastening/load capacity are not established.","Native stills are still required to accept the visible-screen variants."]}
    MANIFEST.write_text(json.dumps(manifest,indent=2)+"\n")
    for v in results: print(v["output"],v["derived_bytes"],v["derived_sha256"])
if __name__=="__main__": main()
