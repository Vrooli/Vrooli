#!/usr/bin/env python3
"""Build a deterministic, generic smart-display study from dimensioned references.

The mesh is authored directly into a bounded GLB 2.0 payload. It contains no
external resources, scripts, logos, or screen artwork. Coordinates are metres,
Y-up, with the active display facing +Z.
"""
from __future__ import annotations

import hashlib
import json
import math
import pathlib
import struct

ROOT = pathlib.Path(__file__).resolve().parent
OUT = ROOT / "generic-smart-display-v3.glb"

MATERIALS = [
    {"name": "Soft-touch graphite polymer", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.080, 0.088, 0.096, 1], "metallicFactor": 0.10, "roughnessFactor": 0.38}},
    {"name": "Satin graphite bezel", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.105, 0.114, 0.123, 1], "metallicFactor": 0.16, "roughnessFactor": 0.31}},
    {"name": "Replaceable active display plane", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.006, 0.008, 0.011, 1], "metallicFactor": 0.08, "roughnessFactor": 0.10}},
    {"name": "Acoustic woven fabric", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.205, 0.198, 0.185, 1], "metallicFactor": 0, "roughnessFactor": 0.93}},
    {"name": "Acoustic port recess", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.045, 0.049, 0.051, 1], "metallicFactor": 0, "roughnessFactor": 0.88}},
    {"name": "Smoked sensor glass", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.018, 0.023, 0.030, 1], "metallicFactor": 0.20, "roughnessFactor": 0.16}},
    {"name": "Silicone contact pads", "doubleSided": True,
     "pbrMetallicRoughness": {"baseColorFactor": [0.025, 0.028, 0.030, 1], "metallicFactor": 0, "roughnessFactor": 0.80}},
    {"name": "Status indicator", "doubleSided": True,
     "emissiveFactor": [0.20, 0.42, 0.34],
     "pbrMetallicRoughness": {"baseColorFactor": [0.16, 0.42, 0.30, 1], "metallicFactor": 0, "roughnessFactor": 0.25}},
]

doc = {
    "asset": {"version": "2.0", "generator": "Device Studio generic smart display v3 deterministic geometry"},
    "scene": 0,
    "scenes": [{"nodes": [0]}],
    "nodes": [{"name": "GenericSmartDisplayV3", "children": []}],
    "meshes": [], "materials": MATERIALS,
    "buffers": [{"byteLength": 0}], "bufferViews": [], "accessors": [],
}
binary = bytearray()


def append_accessor(payload: bytes, component: int, count: int, kind: str,
                    target: int, low=None, high=None) -> int:
    binary.extend(b"\0" * ((-len(binary)) % 4))
    offset = len(binary)
    binary.extend(payload)
    doc["bufferViews"].append({"buffer": 0, "byteOffset": offset,
                               "byteLength": len(payload), "target": target})
    item = {"bufferView": len(doc["bufferViews"]) - 1,
            "componentType": component, "count": count, "type": kind}
    if low is not None:
        item["min"], item["max"] = low, high
    doc["accessors"].append(item)
    return len(doc["accessors"]) - 1


def add_mesh(name: str, triangles: list[tuple[tuple[float, float, float], tuple[float, float]]],
             material: int) -> int:
    if not triangles:
        raise ValueError(f"empty mesh {name}")
    positions: list[tuple[float, float, float]] = []
    uvs: list[tuple[float, float]] = []
    normals: list[tuple[float, float, float]] = []
    indices: list[int] = []
    for tri_start in range(0, len(triangles), 3):
        tri = triangles[tri_start:tri_start + 3]
        a, b, c = (v[0] for v in tri)
        ab = (b[0]-a[0], b[1]-a[1], b[2]-a[2])
        ac = (c[0]-a[0], c[1]-a[1], c[2]-a[2])
        n = (ab[1]*ac[2]-ab[2]*ac[1], ab[2]*ac[0]-ab[0]*ac[2], ab[0]*ac[1]-ab[1]*ac[0])
        length = math.sqrt(sum(x*x for x in n)) or 1.0
        normal = tuple(x/length for x in n)
        for p, uv in tri:
            indices.append(len(positions))
            positions.append(p); uvs.append(uv); normals.append(normal)
    xs, ys, zs = zip(*positions)
    pos = append_accessor(b"".join(struct.pack("<3f", *v) for v in positions), 5126,
                          len(positions), "VEC3", 34962,
                          [min(xs), min(ys), min(zs)], [max(xs), max(ys), max(zs)])
    nor = append_accessor(b"".join(struct.pack("<3f", *v) for v in normals), 5126,
                          len(normals), "VEC3", 34962)
    us, vs = zip(*uvs)
    uv = append_accessor(b"".join(struct.pack("<2f", *v) for v in uvs), 5126,
                         len(uvs), "VEC2", 34962, [min(us), min(vs)], [max(us), max(vs)])
    index_type = 5123 if len(indices) <= 65535 else 5125
    fmt = "<" + ("H" if index_type == 5123 else "I") * len(indices)
    idx = append_accessor(struct.pack(fmt, *indices), index_type, len(indices), "SCALAR", 34963,
                          [0], [max(indices)])
    doc["meshes"].append({"name": name, "primitives": [{
        "attributes": {"POSITION": pos, "NORMAL": nor, "TEXCOORD_0": uv},
        "indices": idx, "material": material,
    }]})
    return len(doc["meshes"]) - 1


def add_node(name: str, mesh: int) -> None:
    doc["nodes"].append({"name": name, "mesh": mesh})
    doc["nodes"][0]["children"].append(len(doc["nodes"]) - 1)


def rounded_rect(width: float, height: float, radius: float, segments: int = 10) -> list[tuple[float, float]]:
    """CCW rounded-rectangle contour in XY, with deterministic corner samples."""
    hx, hy = width / 2, height / 2
    r = min(radius, hx * 0.96, hy * 0.96)
    corners = [(hx-r, -hy+r, -90), (hx-r, hy-r, 0),
               (-hx+r, hy-r, 90), (-hx+r, -hy+r, 180)]
    points = []
    for cx, cy, start in corners:
        for j in range(segments):
            angle = math.radians(start + 90 * j / segments)
            points.append((cx + r * math.cos(angle), cy + r * math.sin(angle)))
    return points


def fan_cap(contour: list[tuple[float, float]], y_center: float, z_at,
            reverse: bool = False) -> list[tuple[tuple[float, float, float], tuple[float, float]]]:
    # The contour is CCW in X/Y, so the front fan faces +Z.
    center = (0.0, y_center, z_at(y_center))
    result = []
    for i, p in enumerate(contour):
        q = contour[(i+1) % len(contour)]
        tri = [(center, (.5,.5)), ((p[0], y_center+p[1], z_at(y_center+p[1])), (0,0)),
               ((q[0], y_center+q[1], z_at(y_center+q[1])), (1,1))]
        if reverse:
            tri.reverse()
        result.extend(tri)
    return result


def rounded_prism(name: str, width: float, height: float, radius: float,
                  center_y: float, front_z: float, depth: float, slope: float,
                  material: int, segments: int = 10) -> None:
    # A small layered bevel carries the rounded silhouette across the front and back.
    layers = [(-depth, width-.004, height-.004, max(.001, radius-.002)),
              (-depth+.002, width, height, radius),
              (-.0018, width, height, radius),
              (0.0, width-.002, height-.002, max(.001, radius-.001))]
    contours = [rounded_rect(w, h, r, segments) for _,w,h,r in layers]
    z_at = lambda y, d=0.0: front_z - slope * (y-center_y) + d
    tris: list[tuple[tuple[float,float,float],tuple[float,float]]] = []
    for layer_i in range(len(layers)-1):
        d0, *_ = layers[layer_i]; d1, *_ = layers[layer_i+1]
        a, b = contours[layer_i], contours[layer_i+1]
        for i in range(len(a)):
            j = (i+1) % len(a)
            v00=(a[i][0],center_y+a[i][1],z_at(center_y+a[i][1],d0))
            v01=(a[j][0],center_y+a[j][1],z_at(center_y+a[j][1],d0))
            v11=(b[j][0],center_y+b[j][1],z_at(center_y+b[j][1],d1))
            v10=(b[i][0],center_y+b[i][1],z_at(center_y+b[i][1],d1))
            tris.extend([(v00,(0,0)),(v01,(1,0)),(v11,(1,1)),
                         (v00,(0,0)),(v11,(1,1)),(v10,(0,1))])
    tris.extend(fan_cap(contours[0], center_y, lambda y:z_at(y,layers[0][0]), reverse=True))
    tris.extend(fan_cap(contours[-1], center_y, lambda y:z_at(y,layers[-1][0])))
    add_node(name, add_mesh(name, tris, material))


def ring_front(name: str, outer_wh: tuple[float,float,float], inner_wh: tuple[float,float,float],
               center_y: float, front_z: float, slope: float, material: int, segments: int=10) -> None:
    outer=rounded_rect(*outer_wh,segments); inner=rounded_rect(*inner_wh,segments)
    if len(outer) != len(inner): raise ValueError("ring contours must match")
    tris=[]
    for i in range(len(outer)):
        j=(i+1)%len(outer)
        def v(point):
            x,y=point; yy=center_y+y
            return ((x,yy,front_z-slope*(yy-center_y)), ((x/outer_wh[0])+.5, 1-((y/outer_wh[1])+.5)))
        oi,oj,ij,ii=v(outer[i]),v(outer[j]),v(inner[j]),v(inner[i])
        tris.extend([oi,oj,ij, oi,ij,ii])
    add_node(name, add_mesh(name, tris, material))


def active_screen(name: str, width: float, height: float, radius: float,
                  center_y: float, front_z: float, slope: float, material: int) -> None:
    contour=rounded_rect(width,height,radius,12)
    def posuv(p):
        x,y=p; yy=center_y+y
        u=x/width+.5
        # glTF image V origin is at the top: top edge is V=0, bottom edge is V=1.
        v=0.5-y/height
        return ((x,yy,front_z-slope*(yy-center_y)+0.00005),(u,v))
    center=((0,center_y,front_z+0.00005),(.5,.5))
    tris=[]
    for i,p in enumerate(contour):
        q=contour[(i+1)%len(contour)]
        tris.extend([center,posuv(p),posuv(q)])
    add_node(name, add_mesh(name, tris, material))


def rounded_profile(points: list[tuple[float,float]], radius: float, segments: int=5) -> list[tuple[float,float]]:
    result=[]
    for i,current in enumerate(points):
        prev=points[(i-1)%len(points)]; nxt=points[(i+1)%len(points)]
        a=(prev[0]-current[0],prev[1]-current[1]); b=(nxt[0]-current[0],nxt[1]-current[1])
        la=math.hypot(*a); lb=math.hypot(*b)
        a=(a[0]/la,a[1]/la); b=(b[0]/lb,b[1]/lb)
        angle=math.acos(max(-1,min(1,a[0]*b[0]+a[1]*b[1])))
        dist=min(radius/max(1e-6,math.tan(angle/2)),la*.42,lb*.42)
        p1=(current[0]+a[0]*dist,current[1]+a[1]*dist)
        p2=(current[0]+b[0]*dist,current[1]+b[1]*dist)
        for j in range(segments+1):
            t=j/segments; omt=1-t
            result.append((omt*omt*p1[0]+2*omt*t*current[0]+t*t*p2[0],
                           omt*omt*p1[1]+2*omt*t*current[1]+t*t*p2[1]))
    return result


def wedge_base() -> None:
    # Y/Z section makes a stable, shallow acoustic pedestal with a sloped rear support.
    section=rounded_profile([(.001,.022),(.037,.018),(.037,-.014),(.009,-.044566),(.001,-.044566)],.003,5)
    half=.0887
    tris=[]
    for i,(y,z) in enumerate(section):
        j=(i+1)%len(section); y2,z2=section[j]
        # left and right end caps, plus the side wall around the extrusion
        tris.extend([((-half,y,z),(0,0)),((-half,y2,z2),(1,0)),((half,y2,z2),(1,1)),
                     ((-half,y,z),(0,0)),((half,y2,z2),(1,1)),((half,y,z),(0,1))])
    # Concave profile fan is avoided: end caps use a center fan, visually flat and bounded.
    cy=sum(p[0] for p in section)/len(section); cz=sum(p[1] for p in section)/len(section)
    cap_left=[]; cap_right=[]
    for i,(y,z) in enumerate(section):
        y2,z2=section[(i+1)%len(section)]
        cap_left.extend([((-half,cy,cz),(0,0)),((-half,y2,z2),(1,0)),((-half,y,z),(0,1))])
        cap_right.extend([((half,cy,cz),(0,0)),((half,y,z),(0,1)),((half,y2,z2),(1,0))])
    # Replace the side-wall triangles created above with the two end caps and clean side strips.
    tris=[]
    for i,(y,z) in enumerate(section):
        y2,z2=section[(i+1)%len(section)]
        tris.extend([((-half,y,z),(0,0)),((half,y,z),(1,0)),((half,y2,z2),(1,1)),
                     ((-half,y,z),(0,0)),((half,y2,z2),(1,1)),((-half,y2,z2),(0,1))])
    tris.extend(cap_left); tris.extend(cap_right)
    add_node("AcousticSpeakerBase", add_mesh("Rounded wedge speaker enclosure",tris,0))


def front_ellipse_tris(name: str, centers: list[tuple[float,float,float]], rx: float, ry: float,
                       material: int, segments: int=10) -> None:
    tris=[]
    for cx,cy,cz in centers:
        center=((cx,cy,cz),(.5,.5))
        ring=[]
        for i in range(segments):
            a=math.tau*i/segments
            x=cx+rx*math.cos(a); y=cy+ry*math.sin(a)
            # UV is only a local detail map; no image is assigned.
            ring.append(((x,y,cz),(.5+.5*math.cos(a),.5+.5*math.sin(a))))
        for i in range(segments): tris.extend([center,ring[i],ring[(i+1)%segments]])
    add_node(name, add_mesh(name,tris,material))


def rounded_plane(name: str,width: float,height: float,radius: float,center_y: float,front_z: float,
                  slope: float,material: int) -> None:
    contour=rounded_rect(width,height,radius,8)
    center=((0,center_y,front_z),(.5,.5)); tris=[]
    for i,p in enumerate(contour):
        q=contour[(i+1)%len(contour)]
        def v(point):
            x,y=point; yy=center_y+y
            return ((x,yy,front_z-slope*(yy-center_y)),(x/width+.5,.5-y/height))
        tris.extend([center,v(p),v(q)])
    add_node(name,add_mesh(name,tris,material))


def make_glb() -> bytes:
    # Dimension envelope follows Google's published Nest Hub 2nd gen dimensions;
    # the generic geometry has no copied product-specific markings or screen content.
    wedge_base()
    # Small silicone contact pads sit within the device envelope and ground its base.
    rounded_prism("Silicone contact pad blank",.026,.0024,.00055,.0012,-.010,.028,0,6,6)
    # The first foot is translated by geometry itself in a separate helper mesh.
    # Add feet as explicit rounded solids through translated-copy triangle vertices.
    def shifted_mesh(old_mesh_index: int, new_name: str, dx: float) -> None:
        mesh=doc["meshes"][old_mesh_index]
        prim=mesh["primitives"][0]
        # Accessor payloads are already authored; duplicate the mesh node with local translation.
        doc["nodes"].append({"name":new_name,"mesh":old_mesh_index,"translation":[dx,0,0]})
        doc["nodes"][0]["children"].append(len(doc["nodes"])-1)
    left_mesh=len(doc["meshes"])-1
    # The source rounded foot was centered at X=0; move it left/right by node transforms.
    doc["nodes"].pop(); doc["nodes"][0]["children"].pop()
    shifted_mesh(left_mesh,"Left silicone contact pad",-.050)
    shifted_mesh(left_mesh,"Right silicone contact pad",.050)

    # Speaker base woven grille: a soft rounded cloth bar, with real repeated port geometry.
    rounded_plane("Woven acoustic grille",.151,.018,.006,.019,.020,.11,3)
    centers=[]
    for row in range(4):
        y=.0125+row*.0042
        count=35
        for col in range(count):
            x=(col-(count-1)/2)*.00415 + (0.0019 if row%2 else 0)
            z=.022-.11*(y-.002)+.00018
            centers.append((x,y,z))
    front_ellipse_tris("Acoustic port microperforations",centers,.00048,.00056,4,8)

    # Screen back shell, satin border, and a fully mapped flush image plane.
    screen_y=.06915; screen_z=.0196; tilt=.105
    rounded_prism("Display rear shell",.1714,.1025,.0150,screen_y,screen_z,.0125,tilt,0,12)
    ring_front("Continuous display bezel",(.1700,.1016,.0140),(.1535,.0899,.0048),screen_y,screen_z,tilt,1,12)
    active_screen("MainDisplayActivePlane",.1530,.0894,.0038,screen_y,screen_z,tilt,2)

    # A few quiet optical and acoustic details belong to the generic hardware, not the UI.
    z_at=lambda y:screen_z-tilt*(y-screen_y)+.00016
    sensor_centers=[(-.045,.1151,z_at(.1151)),(.045,.1151,z_at(.1151)),(0,.11525,z_at(.11525))]
    front_ellipse_tris("Microphone and ambient-sensor apertures",sensor_centers,.00062,.00062,5,12)
    front_ellipse_tris("Ambient-light sensor lens",[(0,.1130,z_at(.1130))],.00105,.00105,5,16)
    front_ellipse_tris("Low-glare status indicator",[(.078,.0285,.0240)],.00072,.00072,7,12)

    doc["buffers"][0]["byteLength"]=len(binary)
    encoded=json.dumps(doc,separators=(",",":"),ensure_ascii=True).encode()
    encoded+=b" "*((-len(encoded))%4)
    bin_data=bytes(binary)+b"\0"*((-len(binary))%4)
    total=12+8+len(encoded)+8+len(bin_data)
    return b"glTF"+struct.pack("<II",2,total)+struct.pack("<II",len(encoded),0x4E4F534A)+encoded+struct.pack("<II",len(bin_data),0x004E4942)+bin_data


def main() -> None:
    data=make_glb()
    if OUT.exists() and OUT.read_bytes()!=data:
        raise SystemExit(f"refusing to replace changed output: {OUT}")
    OUT.write_bytes(data)
    print(f"{OUT.name} {len(data)} {hashlib.sha256(data).hexdigest()}")


if __name__ == "__main__":
    main()
