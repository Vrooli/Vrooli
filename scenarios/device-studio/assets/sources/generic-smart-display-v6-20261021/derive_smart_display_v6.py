#!/usr/bin/env python3
"""Combine v5's rounded contour with v4's coincident-position shell normals."""
import hashlib, json, math, struct
from pathlib import Path
ROOT = Path(__file__).resolve().parents[3]
SRC = ROOT / "api/internal/physicalruntime/assets/smart-display-v5/generic-smart-display-v5.glb"
OUT = ROOT / "api/internal/physicalruntime/assets/smart-display-v6/generic-smart-display-v6.glb"
MANIFEST = ROOT / "api/internal/physicalruntime/assets/smart-display-v6/asset-manifest-v6.json"
def build():
    raw=SRC.read_bytes()
    if raw[:4]!=b"glTF" or struct.unpack_from('<I',raw,4)[0]!=2 or struct.unpack_from('<I',raw,8)[0]!=len(raw): raise ValueError('expected complete GLB 2.0')
    jl,jt=struct.unpack_from('<II',raw,12)
    if jt!=0x4e4f534a: raise ValueError('missing JSON chunk')
    doc=json.loads(raw[20:20+jl].decode().rstrip(' \0'))
    bh=20+jl; bl,bt=struct.unpack_from('<II',raw,bh)
    if bt!=0x004e4942: raise ValueError('missing BIN chunk')
    binary=bytearray(raw[bh+8:bh+8+bl])
    nodes=[(i,n) for i,n in enumerate(doc['nodes']) if n.get('name')=='Display rear shell']
    if len(nodes)!=1 or nodes[0][1].get('mesh')!=4: raise ValueError('v5 shell binding changed')
    prim=doc['meshes'][4]['primitives'][0]; pa=doc['accessors'][prim['attributes']['POSITION']]; na=doc['accessors'][prim['attributes']['NORMAL']]
    if pa['type']!='VEC3' or na['type']!='VEC3' or pa['count']!=na['count'] or pa['count']%3: raise ValueError('unexpected shell accessor topology')
    def offset(a):
        v=doc['bufferViews'][a['bufferView']]
        if v.get('byteStride',12)!=12: raise ValueError('interleaved accessor unsupported')
        return v.get('byteOffset',0)+a.get('byteOffset',0)
    po,no=offset(pa),offset(na); count=pa['count']; positions=[struct.unpack_from('<3f',binary,po+12*i) for i in range(count)]
    sums={}; faces=[]
    for k in range(0,count,3):
        a,b,c=positions[k:k+3]; ab=[b[j]-a[j] for j in range(3)]; ac=[c[j]-a[j] for j in range(3)]
        cross=(ab[1]*ac[2]-ab[2]*ac[1],ab[2]*ac[0]-ab[0]*ac[2],ab[0]*ac[1]-ab[1]*ac[0]); area=math.sqrt(sum(v*v for v in cross))
        if area<1e-14: continue
        normal=tuple(v/area for v in cross); faces.append((k,normal,area))
        for i in range(k,k+3):
            key=tuple(round(v,7) for v in positions[i]); acc=sums.setdefault(key,[0.,0.,0.])
            for axis in range(3): acc[axis]+=normal[axis]*area
    for k,_,_ in faces:
        for i in range(k,k+3):
            avg=sums[tuple(round(v,7) for v in positions[i])]; length=math.sqrt(sum(v*v for v in avg))
            if length<1e-12: raise ValueError('degenerate normal group')
            struct.pack_into('<3f',binary,no+12*i,*(v/length for v in avg))
    doc['asset']['generator']='Device Studio generic smart display v6 high-contour continuous shell normals'
    doc['extras']={'v6':{'source':'generic-smart-display-v5','shellNodeIndex':nodes[0][0],'shellMeshIndex':4,'shellPositionCount':count,'coincidentPositionGroups':len(sums),'normalSmoothing':'area-weighted across exact rounded-position groups only','screenNode':'MainDisplayActivePlane','screenMesh':6}}
    jb=json.dumps(doc,separators=(',',':'),ensure_ascii=True).encode(); jb+=b' '*((-len(jb))%4); bb=bytes(binary)+b'\0'*((-len(binary))%4)
    out=struct.pack('<4sII',b'glTF',2,12+8+len(jb)+8+len(bb))+struct.pack('<II',len(jb),0x4e4f534a)+jb+struct.pack('<II',len(bb),0x004e4942)+bb
    OUT.parent.mkdir(parents=True,exist_ok=True); OUT.write_bytes(out)
    manifest={'schema_version':1,'status':'candidate_only_unqualified','artifact_id':'vrooli-generic-smart-display-concept-v6','output':{'path':'scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v6/generic-smart-display-v6.glb','sha256':hashlib.sha256(out).hexdigest(),'size_bytes':len(out)},'source':{'asset_id':'vrooli-generic-smart-display-concept-v5','path':'scenarios/device-studio/api/internal/physicalruntime/assets/smart-display-v5/generic-smart-display-v5.glb','sha256':hashlib.sha256(raw).hexdigest(),'license':'AGPL-3.0-only','authored_by':'Vrooli'},'derivation':{'generator':'derive_smart_display_v6.py','runtime':'Python standard library; deterministic and offline','changes':['Preserve v5 48-segment rounded shell contour, positions, topology, material, and display mapping.','Recompute only shell normals by area-weighted averaging triangle normals over coincident positions rounded to 1e-7; separated positions retain hard edges.'],'shell_node_index':nodes[0][0],'shell_mesh_index':4,'shell_position_count':count,'coincident_position_groups':len(sums),'screen_node':'MainDisplayActivePlane','screen_mesh_index':6,'screen_primitive_index':0,'uv_orientation':'V=0 top'},'qualification_limits':['Generic authored concept; comparison dimensions are not product CAD or identity.','Candidate requires a native appearance review.']}
    MANIFEST.write_text(json.dumps(manifest,indent=2)+'\n')
    return manifest
if __name__=='__main__':
    m=build(); print(m['output']['size_bytes'],m['output']['sha256'],m['derivation']['coincident_position_groups'])
