#!/usr/bin/env python3
"""Deterministically derive a neutral, single-plane vehicle display from the retained CC0 Grab3D GLB."""
from __future__ import annotations
import argparse, hashlib, json, math, struct
from pathlib import Path

ROOT = Path(__file__).resolve().parent
SRC = ROOT / 'source' / 'car-dashboard-source-original.glb'
OUT = ROOT / 'generic-vehicle-dashboard-display-v1.glb'
MANIFEST = ROOT / 'manifest.json'

def clamp01(v):
    v=round(v,8)
    return max(0.0,min(1.0,v))

def rounded_rect(cx, cy, w, h, r, seg=5):
    pts=[]
    for ox,oy,start in ((cx+w/2-r,cy+h/2-r,0),(cx-w/2+r,cy+h/2-r,90),(cx-w/2+r,cy-h/2+r,180),(cx+w/2-r,cy-h/2+r,270)):
        # CCW around perimeter, in X/Y plane.
        for k in range(seg+1):
            a=math.radians(start+k*90/seg)
            pts.append((ox+r*math.cos(a),oy+r*math.sin(a)))
    return pts

def mesh_data(vertices, faces, uv_by_vertex):
    # Split per-face vertices for crisp bezel side/front normals.
    pos=[]; nrm=[]; uv=[]; idx=[]
    for face in faces:
        a,b,c=(vertices[i] for i in face)
        ab=[b[j]-a[j] for j in range(3)]; ac=[c[j]-a[j] for j in range(3)]
        nn=[ab[1]*ac[2]-ab[2]*ac[1],ab[2]*ac[0]-ab[0]*ac[2],ab[0]*ac[1]-ab[1]*ac[0]]
        ll=math.sqrt(sum(x*x for x in nn)) or 1
        nn=[x/ll for x in nn]
        for i in face:
            idx.append(len(pos)//3); pos.extend(vertices[i]); nrm.extend(nn); uv.extend(uv_by_vertex[i])
    return pos,nrm,uv,idx

def build_mesh():
    # Source upper-screen envelope in the model's Y-up local coordinates; node translation retained.
    # The source's front plane normal is (+Y,+Z) ~= (0,.208,.978).
    cy=.60515; z0=.1022; slope=-.21258
    def xyz(x,y,offset=0): return [x,y,z0+slope*(y-cy)+offset]
    # Rounded 1.91:1 active display face, V=0 at the upper edge. Front and bezel are separate primitives.
    iw,ih,ir=.246,.128,.0055
    outer=rounded_rect(0,cy,.28,.147,.012,6)
    inner=rounded_rect(0,cy,iw,ih,ir,6)
    # Outer back/front loops, inner edge loop. Faces point toward camera (+Y,+Z).
    verts=[]
    for loop,off in ((outer,-.0015),(outer,.004),(inner,.004)):
        verts.extend(xyz(x,y,off) for x,y in loop)
    n=len(outer); faces=[]
    # bezel top face annular band, triangulated as corresponding ring quads (reverse winding for +Y/+Z normal)
    for i in range(n):
        j=(i+1)%n
        faces.extend(((i,j,2*n+j),(i,2*n+j,2*n+i)))
        faces.extend(((i,n+i,n+j),(i,n+j,j)))
        faces.extend(((n+i,2*n+i,2*n+j),(n+i,2*n+j,n+j)))
    # For this XY ring, CCW triangles point +Z; source display front is tilted, recalc actual normals from geometry.
    uv=[(clamp01(x/.28+.5), clamp01(.5-(y-cy)/.147)) for x,y in outer]
    uv += uv[:] + [(clamp01(x/.28+.5), clamp01(.5-(y-cy)/.147)) for x,y in inner]
    bezel=mesh_data(verts,faces,uv)
    # Active rounded panel as a triangle fan, front normal +Y/+Z; exact uv upper-left origin.
    panel=rounded_rect(0,cy,iw,ih,ir,6)
    pverts=[xyz(x,y,.0046) for x,y in panel]
    center=len(pverts); pverts.append(xyz(0,cy,.0046))
    pfaces=[]
    # Perimeter above is CCW in XY; CCW normal is +Z. Fan orientation remains front facing.
    for i in range(len(panel)): pfaces.append((center,i,(i+1)%len(panel)))
    puv=[(clamp01(x/iw+.5),clamp01(.5-(y-cy)/ih)) for x,y in panel]+[(.5,.5)]
    screen=mesh_data(pverts,pfaces,puv)
    return bezel,screen

def main():
    ap=argparse.ArgumentParser(); ap.add_argument('--source',type=Path,default=SRC); ap.add_argument('--output',type=Path,default=OUT); a=ap.parse_args()
    raw=a.source.read_bytes(); source_sha=hashlib.sha256(raw).hexdigest()
    if len(raw)<20 or raw[:4]!=b'glTF': raise SystemExit('source is not GLB')
    jlen=struct.unpack_from('<I',raw,12)[0]; j=json.loads(raw[20:20+jlen]); pos=20+jlen
    if raw[pos+4:pos+8]!=b'BIN\x00': raise SystemExit('expected BIN chunk')
    blen=struct.unpack_from('<I',raw,pos)[0]; binary=bytearray(raw[pos+8:pos+8+blen])
    if len(j['nodes'])<4 or j['nodes'][3].get('mesh')!=3: raise SystemExit('source node contract changed')
    j['materials'][0]['name']='vehicle-dashboard-matte-charcoal'
    p=j['materials'][0]['pbrMetallicRoughness']; p.pop('baseColorTexture',None); p['baseColorFactor']=[.105,.112,.12,1]; p['metallicFactor']=.02; p['roughnessFactor']=.84
    j['materials'][9]['name']='vehicle-steering-wheel-dark-leather'
    p=j['materials'][9]['pbrMetallicRoughness']; p.pop('baseColorTexture',None); p['baseColorFactor']=[.072,.068,.066,1]; p['metallicFactor']=0; p['roughnessFactor']=.73
    # Replace playful ambient/emissive tints with restrained, non-emissive analog cluster inks.
    colors={1:[.16,.17,.18,1],2:[.70,.72,.73,1],3:[.18,.19,.20,1],4:[.42,.12,.075,1],5:[.72,.75,.78,1],6:[.27,.29,.30,1],7:[.72,.50,.18,1],8:[.30,.39,.48,1]}
    for i,c in colors.items():
        m=j['materials'][i]; m['pbrMetallicRoughness']={'baseColorFactor':c,'metallicFactor':.18 if i==1 else 0,'roughnessFactor':.72 if i==1 else .58}; m.pop('emissiveFactor',None); m.pop('extensions',None)
    if not any('KHR_materials_emissive_strength' in m.get('extensions',{}) for m in j['materials']):
        j.pop('extensionsUsed',None); j.pop('extensionsRequired',None)
    j['materials'].append({'name':'vehicle-display-active-glass','pbrMetallicRoughness':{'baseColorFactor':[.012,.016,.020,1],'metallicFactor':.06,'roughnessFactor':.16},'alphaMode':'OPAQUE','doubleSided':False})
    j['materials'].append({'name':'vehicle-display-satin-bezel','pbrMetallicRoughness':{'baseColorFactor':[.075,.082,.088,1],'metallicFactor':.08,'roughnessFactor':.42},'alphaMode':'OPAQUE','doubleSided':False})
    def append_accessor(data,typ,component,kind,minv=None,maxv=None):
        while len(binary)%4: binary.append(0)
        off=len(binary); fmt={'VEC3':'<3f','VEC2':'<2f','SCALAR':'<I'}[typ]
        for n in range(0,len(data),{'VEC3':3,'VEC2':2,'SCALAR':1}[typ]): binary.extend(struct.pack(fmt,*data[n:n+{'VEC3':3,'VEC2':2,'SCALAR':1}[typ]]))
        bv=len(j['bufferViews']); j['bufferViews'].append({'buffer':0,'byteOffset':off,'byteLength':len(binary)-off,'target':34963 if kind=='index' else 34962})
        ac=len(j['accessors']); item={'bufferView':bv,'componentType':component,'count':len(data)//({'VEC3':3,'VEC2':2,'SCALAR':1}[typ]),'type':typ}
        if minv is not None: item['min']=minv
        if maxv is not None: item['max']=maxv
        j['accessors'].append(item); return ac
    def make_primitive(md,material):
        pp,nn,uv,ix=md
        def mm(data,k):
            stride={'POSITION':3,'NORMAL':3,'TEXCOORD_0':2,'indices':1}[k]
            rows=[data[i:i+stride] for i in range(0,len(data),stride)]
            mn=[min(r[c] for r in rows) for c in range(stride)]; mx=[max(r[c] for r in rows) for c in range(stride)]
            return append_accessor(data,'SCALAR' if stride==1 else ('VEC2' if stride==2 else 'VEC3'),5125 if stride==1 else 5126,k.lower(),mn,mx)
        return {'attributes':{'POSITION':mm(pp,'POSITION'),'NORMAL':mm(nn,'NORMAL'),'TEXCOORD_0':mm(uv,'TEXCOORD_0')},'indices':mm(ix,'indices'),'material':material,'mode':4}
    bezel,screen=build_mesh()
    j['meshes'][3]={'name':'vehicle-display-assembly','primitives':[make_primitive(screen,10),make_primitive(bezel,11)]}
    j['nodes'][3]['name']='vehicle-display-assembly'
    j['asset'].setdefault('extras',{})['derivation']='Vehicle dashboard host recolor and single clean active display; original retained unchanged.'
    j['buffers'][0]['byteLength']=len(binary)
    jb=json.dumps(j,separators=(',',':')).encode(); jb+=b' '*((-len(jb))%4)
    binary.extend(b' ' *((-len(binary))%4))
    chunks=struct.pack('<I',len(jb))+b'JSON'+jb+struct.pack('<I',len(binary))+b'BIN\x00'+binary
    out=b'glTF'+struct.pack('<II',2,12+len(chunks))+chunks
    a.output.write_bytes(out)
    sha=hashlib.sha256(out).hexdigest()
    manifest={
      'id':'generic-vehicle-dashboard-display-v1','source':{'name':'Grab3D Car Dashboard','url':'https://grab3d.com/models/vehicles/car-dashboard/','download_url':'https://grab3d.com/dl/car-dashboard/car-dashboard.glb','license_claim':'CC0-1.0, stated by source package LICENSE.txt and source manifest','license_reference':'https://creativecommons.org/publicdomain/zero/1.0/','source_file':'source/car-dashboard-source-original.glb','source_bytes':len(raw),'source_sha256':source_sha,'original_preserved_byte_for_byte':True,'acquired_at':'2026-10-21'},
      'derived_file':a.output.name,'derived_bytes':len(out),'derived_sha256':sha,'generator':'derive_vehicle_host.py (Python standard library only)',
      'geometry':{'units':'meters (source-reported nominal real scale)','axis':'Y up; front screen normal points +Y,+Z','root_node_index':5,'display_node_index':3,'display_node_name':'vehicle-display-assembly','mesh_index':3,'active_display_primitive_index':0,'active_display_material_index':10,'bezel_primitive_index':1,'bezel_material_index':11,'uv_origin':'upper-left; U increases right, V increases downward','active_plane_local_bounds_m':{'x':[-.123,.123],'y':[.54115,.66915],'z_approx':[.105,.120]},'support_frame_node_index':0,'support_frame':'dashboard body; approximate world-space Y=0 contact plane; no stability/contact qualification'},
      'changes':['Retained dashboard body, ambient trim, instrument cluster and steering wheel geometry.','Removed all four original screen/map/routing overlay primitives and installed one clean rounded active plane with a separate satin bezel.','Removed mint emissive colors; set dashboard and wheel to dark neutral matte PBR, and cluster details to restrained analog colors.','Dropped source base-color maps only on body and steering wheel to avoid their mint stylization; retained source roughness/metallic and normal maps.'],
      'limitations':['Generic host adaptation; no OEM identity or CarPlay integration claim.','Display proportions and placement are a bounded derivative of the source screen envelope; these are not a manufacturer dimensional specification.','Source preview and GLB geometry were reviewed, but no native renderer acceptance claim is made here.','The generated support plane is an inferred asset bound, not proof of physical stability.']}
    MANIFEST.write_text(json.dumps(manifest,indent=2)+'\n')
    print(json.dumps({'output':str(a.output),'bytes':len(out),'sha256':sha,'source_sha256':source_sha,'manifest':str(MANIFEST)},indent=2))
if __name__=='__main__': main()
