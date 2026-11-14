#!/usr/bin/env python3
"""Build a deterministic, neutral, reference-proportioned book-fold GLB."""

import json
import math
import pathlib
import struct

ROOT = pathlib.Path(__file__).resolve().parents[3]
OUTPUT = ROOT / "api/internal/physicalruntime/assets/foldable/foldable-concept.glb"
HALF_W, HEIGHT, DEPTH = 0.03275, 0.1535, 0.0056
PANEL_X = HALF_W + 0.0008

binary = bytearray()
views, accessors, meshes, nodes = [], [], [], []


def align():
    binary.extend(b"\0" * ((4 - len(binary) % 4) % 4))


def accessor(values, component, kind, target, minimum=None, maximum=None):
    align()
    offset = len(binary)
    fmt = "f" if component == 5126 else "H"
    for row in values:
        items = row if isinstance(row, tuple) else (row,)
        binary.extend(struct.pack("<" + fmt * len(items), *items))
    width = len(values[0]) if isinstance(values[0], tuple) else 1
    count = len(values) * width if kind == "SCALAR" else len(values)
    view = len(views)
    views.append({"buffer": 0, "byteOffset": offset, "byteLength": len(binary) - offset, "target": target})
    result = {"bufferView": view, "componentType": component, "count": count, "type": kind}
    if minimum is not None:
        result["min"], result["max"] = minimum, maximum
    accessors.append(result)
    return len(accessors) - 1


def add_mesh(name, positions, triangles, material, uv=None, normals=None):
    if normals is None:
        normals = [(0.0, 0.0, 1.0)] * len(positions)
    if uv is None:
        uv = [(0.0, 0.0)] * len(positions)
    pos_min = [min(p[i] for p in positions) for i in range(3)]
    pos_max = [max(p[i] for p in positions) for i in range(3)]
    attrs = {
        "POSITION": accessor(positions, 5126, "VEC3", 34962, pos_min, pos_max),
        "NORMAL": accessor(normals, 5126, "VEC3", 34962),
        "TEXCOORD_0": accessor(uv, 5126, "VEC2", 34962),
    }
    index = accessor(triangles, 5123, "SCALAR", 34963)
    meshes.append({"name": name, "primitives": [{"attributes": attrs, "indices": index, "material": material}]})
    return len(meshes) - 1


def rounded_panel(name, material, width, height, depth, radius, bevel=0.00055):
    points = []
    for cx, cy, start in ((width / 2 - radius, height / 2 - radius, 0),
                          (-width / 2 + radius, height / 2 - radius, 90),
                          (-width / 2 + radius, -height / 2 + radius, 180),
                          (width / 2 - radius, -height / 2 + radius, 270)):
        for step in range(7):
            angle = math.radians(start + step * 90 / 6)
            points.append((cx + radius * math.cos(angle), cy + radius * math.sin(angle)))
    n = len(points)
    rings = [(-depth / 2, 0.0), (-depth / 2 + bevel, bevel),
             (depth / 2 - bevel, bevel), (depth / 2, 0.0)]
    positions = []
    for z, inset in rings:
        sx, sy = (width - 2 * inset) / width, (height - 2 * inset) / height
        positions.extend((x * sx, y * sy, z) for x, y in points)
    triangles = []
    for ring in range(3):
        for i in range(n):
            j = (i + 1) % n
            a, b, c, d = ring * n + i, ring * n + j, (ring + 1) * n + j, (ring + 1) * n + i
            triangles.extend(((a, b, c), (a, c, d)))
    back_center, front_center = len(positions), len(positions) + 1
    positions.extend(((0.0, 0.0, rings[1][0]), (0.0, 0.0, rings[2][0])))
    for i in range(n):
        j = (i + 1) % n
        triangles.extend(((back_center, j, i), (front_center, 2 * n + i, 2 * n + j)))
    normals = []
    for ring, (_, inset) in enumerate(rings):
        for x, y in points:
            length = math.hypot(x, y) or 1
            if ring == 0:
                normals.append((0.0, 0.0, -1.0))
            elif ring == 1:
                normals.append((x / length * 0.7, y / length * 0.7, -0.7))
            elif ring == 2:
                normals.append((x / length * 0.7, y / length * 0.7, 0.7))
            else:
                normals.append((0.0, 0.0, 1.0))
    normals.extend(((0.0, 0.0, -1.0), (0.0, 0.0, 1.0)))
    uvs = [((x + width / 2) / width, (y + height / 2) / height) for x, y, _ in positions]
    return add_mesh(name, positions, triangles, material, uvs, normals)


def segmented_hinge(name, material, radius, height, segments=7, sides=12):
    positions, triangles, normals, uvs = [], [], [], []
    gap = height * 0.018
    segment_height = (height - gap * (segments - 1)) / segments
    for segment in range(segments):
        low = -height / 2 + segment * (segment_height + gap)
        first = len(positions)
        for y in (low, low + segment_height):
            for side in range(sides):
                angle = 2 * math.pi * side / sides
                nx, nz = math.cos(angle), math.sin(angle)
                positions.append((radius * nx, y, radius * nz))
                normals.append((nx, 0.0, nz))
                uvs.append((side / sides, (y + height / 2) / height))
        for side in range(sides):
            nxt = (side + 1) % sides
            triangles.extend(((first + side, first + nxt, first + sides + nxt),
                              (first + side, first + sides + nxt, first + sides + side)))
    return add_mesh(name, positions, triangles, material, uvs, normals)


def screen(name, material, width, height, z, reverse=False):
    positions = [(-width / 2, -height / 2, z), (width / 2, -height / 2, z),
                 (width / 2, height / 2, z), (-width / 2, height / 2, z)]
    uv = [(0, 0), (1, 0), (1, 1), (0, 1)]
    indices = [(0, 1, 2), (0, 2, 3)]
    normal = (0, 0, 1)
    if reverse:
        indices = [(0, 2, 1), (0, 3, 2)]
        normal = (0, 0, -1)
    return add_mesh(name, positions, indices, material, uv, [normal] * 4)


case = rounded_panel("LeftRigidHousing", 0, HALF_W * 2, HEIGHT, DEPTH, 0.004)
left_inner = screen("InnerDisplayLeft", 1, 0.0605, 0.144, DEPTH / 2 + 0.00008)
right_inner = screen("InnerDisplayRight", 2, 0.0605, 0.144, DEPTH / 2 + 0.00008)
right_outer = screen("OuterDisplay", 3, 0.058, 0.126, -DEPTH / 2 - 0.00008, True)
moving_case = rounded_panel("RightRigidHousing", 0, HALF_W * 2, HEIGHT, DEPTH, 0.004)
hinge = segmented_hinge("HingeBarrelMesh", 4, 0.00315, HEIGHT)

nodes = [
    {"name": "FoldConceptRoot", "children": [1, 3, 7]},
    {"name": "LeftHousing", "mesh": case, "translation": [-PANEL_X, 0, 0], "children": [2]},
    {"name": "LeftInnerDisplay", "mesh": left_inner, "translation": [0, 0, 0]},
    {"name": "MainHingePivot", "children": [4]},
    {"name": "RightHousing", "mesh": moving_case, "translation": [PANEL_X, 0, 0], "children": [5, 6]},
    {"name": "RightInnerDisplay", "mesh": right_inner},
    {"name": "OuterDisplay", "mesh": right_outer},
    {"name": "HingeBarrel", "mesh": hinge},
]

document = {
    "asset": {"version": "2.0", "generator": "Device Studio foldable concept derivation"},
    "scene": 0, "scenes": [{"nodes": [0]}], "nodes": nodes,
    "meshes": meshes, "materials": [
        {"name": "Graphite ceramic housing", "pbrMetallicRoughness": {"baseColorFactor": [0.12, 0.13, 0.15, 1], "metallicFactor": 0.42, "roughnessFactor": 0.34}},
        {"name": "Inner display left neutral glass", "pbrMetallicRoughness": {"baseColorFactor": [0.004, 0.005, 0.007, 1], "metallicFactor": 0.04, "roughnessFactor": 0.28}, "emissiveFactor": [0.001, 0.0012, 0.0015]},
        {"name": "Inner display right neutral glass", "pbrMetallicRoughness": {"baseColorFactor": [0.004, 0.005, 0.007, 1], "metallicFactor": 0.04, "roughnessFactor": 0.28}, "emissiveFactor": [0.001, 0.0012, 0.0015]},
        {"name": "Outer display neutral glass", "pbrMetallicRoughness": {"baseColorFactor": [0.004, 0.005, 0.007, 1], "metallicFactor": 0.04, "roughnessFactor": 0.28}, "emissiveFactor": [0.001, 0.0012, 0.0015]},
        {"name": "Hinge metal", "pbrMetallicRoughness": {"baseColorFactor": [0.22, 0.23, 0.25, 1], "metallicFactor": 0.72, "roughnessFactor": 0.3}},
    ], "accessors": accessors, "bufferViews": views,
    "buffers": [{"byteLength": len(binary)}],
}

json_bytes = json.dumps(document, separators=(",", ":")).encode("utf-8")
json_bytes += b" " * ((4 - len(json_bytes) % 4) % 4)
binary.extend(b"\0" * ((4 - len(binary) % 4) % 4))
total = 12 + 8 + len(json_bytes) + 8 + len(binary)
OUTPUT.parent.mkdir(parents=True, exist_ok=True)
with OUTPUT.open("wb") as stream:
    stream.write(struct.pack("<III", 0x46546C67, 2, total))
    stream.write(struct.pack("<II", len(json_bytes), 0x4E4F534A))
    stream.write(json_bytes)
    stream.write(struct.pack("<II", len(binary), 0x004E4942))
    stream.write(binary)
