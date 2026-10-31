#!/usr/bin/env python3
"""Deterministically author a generic, replaceable-screen round watch GLB."""
import json, math, struct, hashlib
from pathlib import Path

ROOT = Path(__file__).parent
OUT = ROOT / "generic-roundwatch-v2.glb"
materials = []
meshes = []
nodes = []
blob = bytearray()
views = []
accessors = []

def material(name, color, metal=0.0, rough=.45):
    materials.append({"name":name,"pbrMetallicRoughness":{"baseColorFactor":[*color,1],"metallicFactor":metal,"roughnessFactor":rough}})
    return len(materials)-1

CASE = material("Graphite titanium-like case",(0.18,.20,.22),.72,.28)
BEZEL = material("Brushed dark metal bezel",(.32,.35,.38),.78,.24)
GLASS = material("Neutral replaceable display",(.012,.018,.028),.04,.2)
SILICONE = material("Charcoal elastomer strap",(.045,.052,.061),.0,.78)
BUCKLE = material("Satin metal buckle",(.38,.40,.42),.8,.3)
BACK = material("Ceramic sensor back",(.10,.12,.14),.12,.52)
CONTACT = material("Soft contact pads",(.025,.03,.035),0,.9)

def add_accessor(data, component, typ, count, target, mins=None, maxs=None):
    while len(blob)%4: blob.append(0)
    off=len(blob); blob.extend(data)
    vi=len(views); views.append({"buffer":0,"byteOffset":off,"byteLength":len(data),"target":target})
    acc={"bufferView":vi,"byteOffset":0,"componentType":component,"count":count,"type":typ}
    if mins is not None: acc["min"]=mins; acc["max"]=maxs
    accessors.append(acc)
    return len(accessors)-1

def make_mesh(name, verts, tris, mat, uvs=None, normals=None):
    if normals is None:
        sums=[[0.,0.,0.] for _ in verts]
        for a,b,c in tris:
            p,q,r=verts[a],verts[b],verts[c]
            u=[q[i]-p[i] for i in range(3)]; v=[r[i]-p[i] for i in range(3)]
            n=[u[1]*v[2]-u[2]*v[1],u[2]*v[0]-u[0]*v[2],u[0]*v[1]-u[1]*v[0]]
            for k in (a,b,c): sums[k]=[sums[k][i]+n[i] for i in range(3)]
        normals=[]
        for x,y,z in sums:
            d=math.sqrt(x*x+y*y+z*z) or 1
            normals.append([x/d,y/d,z/d])
    pos=b''.join(struct.pack('<3f',*p) for p in verts)
    nrm=b''.join(struct.pack('<3f',*n) for n in normals)
    uvdata=b''.join(struct.pack('<2f',*p) for p in (uvs or [(0,0)]*len(verts)))
    inds=[i for tri in tris for i in tri]
    # positions and normals stored in meters, with local origin centered on the case.
    mins=[min(p[i] for p in verts) for i in range(3)]; maxs=[max(p[i] for p in verts) for i in range(3)]
    pa=add_accessor(pos,5126,"VEC3",len(verts),34962,mins,maxs)
    na=add_accessor(nrm,5126,"VEC3",len(verts),34962)
    ua=add_accessor(uvdata,5126,"VEC2",len(verts),34962)
    ia=add_accessor(struct.pack('<%dH'%len(inds),*inds),5123,"SCALAR",len(inds),34963)
    meshes.append({"name":name,"primitives":[{"attributes":{"POSITION":pa,"NORMAL":na,"TEXCOORD_0":ua},"indices":ia,"material":mat,"mode":4}]})
    return len(meshes)-1

def node(name, mesh, translation=(0,0,0), parent=0):
    ix=len(nodes); nodes.append({"name":name,"mesh":mesh,"translation":list(translation)})
    nodes[parent].setdefault("children",[]).append(ix)
    return ix

def lathe(name, profile, mat, parent=0, seg=128):
    verts=[]; tris=[]; uvs=[]
    for radius,z in profile:
        for j in range(seg):
            a=2*math.pi*j/seg
            verts.append([radius*math.cos(a), radius*math.sin(a),z])
            uvs.append([j/seg,0 if len(profile)==1 else len(verts)//seg/len(profile)])
    for k in range(len(profile)-1):
        for j in range(seg):
            a=k*seg+j; b=k*seg+(j+1)%seg; c=(k+1)*seg+(j+1)%seg; d=(k+1)*seg+j
            tris.extend([(a,b,c),(a,c,d)])
    m=make_mesh(name,verts,tris,mat,uvs)
    return node(name,m,parent=parent)

def disk(name, radius, z, mat, parent=0, seg=128):
    verts=[[0,0,z]]; uvs=[[.5,.5]]; tris=[]
    for j in range(seg):
        a=2*math.pi*j/seg
        x=radius*math.cos(a); y=radius*math.sin(a)
        verts.append([x,y,z]); uvs.append([.5+x/(2*radius),.5-y/(2*radius)])
    for j in range(seg): tris.append((0,j+1,(j+1)%seg+1))
    normals=[[0,0,1] for _ in verts]
    m=make_mesh(name,verts,tris,mat,uvs,normals)
    return node(name,m,parent=parent)

def rounded_box(name, sx, sy, sz, bevel, mat, center=(0,0,0), parent=0, seg=8):
    # Beveled rectangular prism with rounded vertical corners; front/back rings are
    # smoothly connected to the smaller bevel rings to avoid primitive cube edges.
    x0,y0,z0=center; rx=sx/2; ry=sy/2
    outline=[]
    corners=[(rx-bevel,ry-bevel,0),( -rx+bevel,ry-bevel,90),(-rx+bevel,-ry+bevel,180),(rx-bevel,-ry+bevel,270)]
    for cx,cy,start in corners:
        for k in range(seg+1):
            a=math.radians(start+k*90/seg)
            outline.append((cx+bevel*math.cos(a),cy+bevel*math.sin(a)))
    verts=[]; uvs=[]; n=len(outline)
    for zz,scale in [(-sz/2+bevel,.94),(-sz/2,1),(sz/2,1),(sz/2-bevel,.94)]:
        for x,y in outline:
            verts.append([x*scale,y*scale,zz]); uvs.append([(x/sx)+.5,.5-(y/sy)])
    tris=[]
    for ring in range(3):
        for j in range(n):
            a=ring*n+j;b=ring*n+(j+1)%n;c=(ring+1)*n+(j+1)%n;d=(ring+1)*n+j
            tris.extend([(a,b,c),(a,c,d)])
    # front/back caps
    for j in range(1,n-1):
        tris.extend([(0,j+1,j),(3*n,3*n+j,3*n+j+1)])
    m=make_mesh(name,verts,tris,mat,uvs)
    return node(name,m,center,parent)

# Root case center is at Y=.057 m. Reference envelope comparison: 47.4 x 47.1 x 12.1 mm.
nodes.append({"name":"GenericRoundWatchV2","children":[],"translation":[0,.057,0]})
cx,cy=.0,.057
caseprofile=[(.0186,-.00605),(.0218,-.0059),(.0232,-.0044),(.0237,-.0022),(.0237,.0035),(.0231,.0052),(.0219,.00605),(.0192,.00605)]
case_node=lathe("Rounded monocoque case with soft bevel",caseprofile,CASE)
ringprofile=[(.01845,.0060),(.0200,.0060),(.02065,.0068),(.02065,.0078),(.0201,.0084),(.01855,.0084)]
lathe("Continuous satin circular bezel",ringprofile,BEZEL)
disk("One clean circular replaceable display",.01835,.00845,GLASS)
# Rear inset and sensor circle remain visible only from the reverse side.
lathe("Ceramic caseback perimeter",[(.0168,-.0063),(.0187,-.0062),(.0187,-.0058)],BACK)
lathe("Optical sensor ring",[(.0072,-.0071),(.0090,-.0071),(.0090,-.0062),(.0072,-.0062)],BACK)
lathe("Optical sensor lens",[(.0070,-.00715),(.0,-.00715)],CASE,seg=64)
# Four stepped lug blocks, strap lands, and dark insert contact pads.
for ysign,label in [(-1,"lower"),(1,"upper")]:
    y=ysign*.0253
    rounded_box(label+" lug bridge",.018,.009,.008,.003,CASE,(0,y,-.001))
    # A single broad silicone band with softened corners, projected above/below case.
    if ysign<0:
        sy=.038; centerY=-.0235-.019
    else:
        sy=.052; centerY=.0235+.026
    rounded_box(label+" charcoal silicone strap",.019,sy,.0048,.004,SILICONE,(0,centerY,-.0018))
    rounded_box(label+" strap inner contact",.014,.007,.002,.003,CONTACT,(0,(-.053 if ysign<0 else .048),-.0048))
# Buckle, understated side crown and one flush pusher.
rounded_box("Satin strap buckle crossbar",.017,.003,.003,.0012,BUCKLE,(0,.073,-.002))
rounded_box("Satin strap buckle side",.003,.012,.003,.0012,BUCKLE,(-.008,.067,-.002))
rounded_box("Satin strap buckle side",.003,.012,.003,.0012,BUCKLE,(.008,.067,-.002))
# crown shafts point along X; 48 segments produce a smooth edge in close review.
def cylinder_x(name,x,y,z,r,length,mat,segments=48):
    verts=[]; tris=[]; uvs=[]
    for xx in [x-length/2,x+length/2]:
        for j in range(segments):
            a=2*math.pi*j/segments
            verts.append([xx,y+r*math.cos(a),z+r*math.sin(a)]); uvs.append([j/segments,xx])
    for j in range(segments):
        a=j;b=(j+1)%segments;c=segments+b;d=segments+a;tris.extend([(a,b,c),(a,c,d)])
    m=make_mesh(name,verts,tris,mat,uvs)
    node(name,m,parent=0)
cylinder_x("Recessed digital crown",.0241,0,.0003,.0041,.0054,BEZEL)
cylinder_x("Flush lower action button",.0240,-.009,.0002,.0022,.0048,CASE)
# crown grip ribs
for i in range(12):
    a=2*math.pi*i/12
    cylinder_x("Crown knurl",.0270,.0041*math.cos(a),.0003+.0041*math.sin(a),.00035,.0007,CASE,12)

doc={"asset":{"version":"2.0","generator":"Device Studio deterministic generic round-watch v2 generator"},"scene":0,"scenes":[{"nodes":[0]}],"nodes":nodes,"meshes":meshes,"materials":materials,"buffers":[{"byteLength":0}],"bufferViews":views,"accessors":accessors}
while len(blob)%4: blob.append(0)
doc["buffers"][0]["byteLength"]=len(blob)
j=json.dumps(doc,separators=(',',':')).encode()
while len(j)%4: j+=b' '
total=12+8+len(j)+8+len(blob)
OUT.write_bytes(struct.pack('<4sII',b'glTF',2,total)+struct.pack('<I4s',len(j),b'JSON')+j+struct.pack('<I4s',len(blob),b'BIN\x00')+blob)
print(len(OUT.read_bytes()),hashlib.sha256(OUT.read_bytes()).hexdigest(),len(nodes),len(meshes))
