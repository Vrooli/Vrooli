#!/usr/bin/env python3
"""Smooth only the rounded rear-shell normals in the retained smart-display v3 GLB."""
from __future__ import annotations
import argparse, hashlib, json, math, pathlib, struct

ROOT = pathlib.Path(__file__).resolve().parent
SOURCE = ROOT.parent / "generic-smart-display-v3-20261021" / "generic-smart-display-v3.glb"
OUTPUT = ROOT / "generic-smart-display-v4.glb"
MANIFEST = ROOT / "manifest.json"

def chunks(data: bytes):
    if len(data) < 20 or data[:4] != b"glTF" or struct.unpack_from("<I", data, 4)[0] != 2:
        raise ValueError("input must be a GLB 2.0 file")
    declared = struct.unpack_from("<I", data, 8)[0]
    if declared != len(data): raise ValueError("GLB declared length does not match file size")
    out=[]; pos=12
    while pos < len(data):
        size = struct.unpack_from("<I", data, pos)[0]; kind = data[pos+4:pos+8]
        payload = data[pos+8:pos+8+size]
        if len(payload) != size: raise ValueError("truncated GLB chunk")
        out.append((kind,payload)); pos += 8+size
    return out

def accessor_payload(doc, binary, accessor_index):
    a=doc["accessors"][accessor_index]; view=doc["bufferViews"][a["bufferView"]]
    offset=view.get("byteOffset",0)+a.get("byteOffset",0)
    return a,view,offset

def smooth_shell_normals(doc, binary):
    matches=[(i,n) for i,n in enumerate(doc["nodes"]) if n.get("name")=="Display rear shell"]
    if len(matches)!=1 or matches[0][1].get("mesh")!=4: raise ValueError("v3 shell node contract changed")
    mesh=doc["meshes"][4]; prim=mesh["primitives"][0]
    pos_i=prim["attributes"]["POSITION"]; nor_i=prim["attributes"]["NORMAL"]
    pa,pv,po=accessor_payload(doc,binary,pos_i); na,nv,no=accessor_payload(doc,binary,nor_i)
    if pa["type"]!="VEC3" or na["type"]!="VEC3" or pa["count"]!=na["count"]:
        raise ValueError("shell position/normal accessors do not align")
    stride=pv.get("byteStride",12); nstride=nv.get("byteStride",12)
    if stride!=12 or nstride!=12: raise ValueError("unexpected interleaved shell accessors")
    count=pa["count"]; positions=[struct.unpack_from("<3f",binary,po+i*stride) for i in range(count)]
    # Area-weighted averaging across duplicate triangle vertices smooths the authored
    # bevel strips while retaining the exact source positions, indices, and topology.
    sums={}; faces=[]
    for k in range(0,count,3):
        a,b,c=positions[k:k+3]
        ab=[b[j]-a[j] for j in range(3)]; ac=[c[j]-a[j] for j in range(3)]
        cross=[ab[1]*ac[2]-ab[2]*ac[1],ab[2]*ac[0]-ab[0]*ac[2],ab[0]*ac[1]-ab[1]*ac[0]]
        area=math.sqrt(sum(v*v for v in cross))
        if area < 1e-14: continue
        normal=[v/area for v in cross]; faces.append((k,normal,area))
        for i in range(k,k+3):
            key=tuple(round(v,7) for v in positions[i])
            acc=sums.setdefault(key,[0.0,0.0,0.0])
            for axis in range(3): acc[axis]+=normal[axis]*area
    for k,normal,area in faces:
        for i in range(k,k+3):
            key=tuple(round(v,7) for v in positions[i]); avg=sums[key]
            length=math.sqrt(sum(v*v for v in avg))
            if length < 1e-12: raise ValueError("degenerate smooth-normal group")
            result=tuple(v/length for v in avg)
            struct.pack_into("<3f",binary,no+i*nstride,*result)
    return {"node_index":matches[0][0],"mesh_index":4,"primitive_index":0,"position_accessor":pos_i,"normal_accessor":nor_i,"vertex_count":count,"smoothed_position_groups":len(sums)}

def build(source: pathlib.Path, output: pathlib.Path):
    raw=source.read_bytes(); source_sha=hashlib.sha256(raw).hexdigest(); parts=chunks(raw)
    json_chunks=[i for i,(kind,_) in enumerate(parts) if kind==b"JSON"]
    bin_chunks=[i for i,(kind,_) in enumerate(parts) if kind==b"BIN\x00"]
    if len(json_chunks)!=1 or len(bin_chunks)!=1: raise ValueError("expected one JSON and one BIN chunk")
    ji,bi=json_chunks[0],bin_chunks[0]
    doc=json.loads(parts[ji][1].decode().rstrip(" \t\r\n\0")); binary=bytearray(parts[bi][1])
    if doc.get("asset",{}).get("version")!="2.0" or len(doc.get("materials",[]))<2:
        raise ValueError("not the expected smart-display v3 source")
    if doc["materials"][0].get("name")!="Soft-touch graphite polymer": raise ValueError("v3 shell material changed")
    shell=smooth_shell_normals(doc,binary)
    # Reduce broad specular response on the graphite shell while leaving bezel/glass untouched.
    doc["materials"][0]["pbrMetallicRoughness"]["metallicFactor"]=0.04
    doc["materials"][0]["pbrMetallicRoughness"]["roughnessFactor"]=0.56
    doc["asset"]["generator"]="Device Studio smart display v4 source-preserving shell-normal repair"
    encoded=json.dumps(doc,separators=(",",":"),ensure_ascii=True).encode(); encoded+=b" "*((-len(encoded))%4)
    out_chunks=[]
    for i,(kind,payload) in enumerate(parts):
        if i==ji: payload=encoded
        elif i==bi: payload=bytes(binary)
        out_chunks.append(struct.pack("<I",len(payload))+kind+payload)
    body=b"".join(out_chunks); result=b"glTF"+struct.pack("<II",2,12+len(body))+body
    output.write_bytes(result)
    manifest={
      "schema":"device-studio-source-asset/v1","asset_id":"generic-smart-display-v4-20261021","status":"candidate_only_unqualified",
      "source":{"asset_id":"generic-smart-display-concept-v3","path":"../generic-smart-display-v3-20261021/generic-smart-display-v3.glb","bytes":len(raw),"sha256":source_sha,"preserved_unchanged":True,"rights":"Vrooli-authored AGPL-3.0-only generic geometry; official Google reference media is comparison-only and not embedded."},
      "derivative":{"generator":"derive_smart_display_v4.py","runtime":"Python standard library","deterministic":True,"glb_file":output.name,"glb_bytes":len(result),"glb_sha256":hashlib.sha256(result).hexdigest(),"changes":["Area-weighted smooth normals on only the Display rear shell triangle vertices; POSITION, UVs, indices, node transforms, and all other geometry are unchanged.","Shell polymer material metallicFactor 0.10→0.04 and roughnessFactor 0.38→0.56 to reduce broad specular reflections."],"shell_binding":shell},
      "display_binding":{"node_index":8,"node_name":"MainDisplayActivePlane","mesh_index":6,"primitive_index":0,"material_index":2,"uv_v_orientation":"0 at top; 1 at bottom","screen_geometry_unchanged":True},
      "qualification_limits":["Preserves the existing v3 candidate envelope and display target; does not change v3 bytes or IDs.","This is an unreviewed asset-only repair. No Device Studio native v4 render was run; professional appearance remains unaccepted until root's later native pilot.","No manufacturer identity or product qualification is claimed."]}
    MANIFEST.write_text(json.dumps(manifest,indent=2)+"\n")
    return result,manifest

def main():
    ap=argparse.ArgumentParser(); ap.add_argument("--source",type=pathlib.Path,default=SOURCE); ap.add_argument("--output",type=pathlib.Path,default=OUTPUT); args=ap.parse_args()
    data,manifest=build(args.source,args.output); print(f"{args.output.name} {len(data)} {manifest['derivative']['glb_sha256']}")
if __name__=="__main__": main()
