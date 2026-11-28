#!/usr/bin/env python3
"""Derive fixed neutral screen candidates for Device Studio's missing concepts."""
from __future__ import annotations

import copy
import hashlib
import json
import math
import pathlib
import struct

ROOT = pathlib.Path(__file__).resolve().parents[3]
OUT = ROOT / "api/internal/physicalruntime/assets"
FRAME_SOURCE = ROOT / "assets/sources/innerscene-picture-frame/picture-frame.glb"
FRAME_SOURCE_SHA = "7c9f5cf3e63c7f01cc911b1db9eea2c95b5e1bdadd9b75c3076b10dca70cef10"


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def glb_parts(data: bytes) -> tuple[dict, bytes]:
    if len(data) < 28 or data[:4] != b"glTF" or struct.unpack_from("<II", data, 4) != (2, len(data)):
        raise ValueError("expected bounded GLB 2.0")
    json_len, json_type = struct.unpack_from("<II", data, 12)
    if json_type != 0x4E4F534A:
        raise ValueError("GLB JSON chunk is missing")
    bin_header = 20 + json_len
    bin_len, bin_type = struct.unpack_from("<II", data, bin_header)
    if bin_type != 0x004E4942 or bin_header + 8 + bin_len != len(data):
        raise ValueError("GLB BIN chunk is missing or unbounded")
    return json.loads(data[20:bin_header]), data[bin_header + 8 :]


def pack_glb(doc: dict, binary: bytes) -> bytes:
    encoded = json.dumps(doc, separators=(",", ":"), ensure_ascii=True).encode()
    encoded += b" " * ((-len(encoded)) % 4)
    binary += b"\0" * ((-len(binary)) % 4)
    size = 12 + 8 + len(encoded) + 8 + len(binary)
    return b"glTF" + struct.pack("<II", 2, size) + struct.pack("<II", len(encoded), 0x4E4F534A) + encoded + struct.pack("<II", len(binary), 0x004E4942) + binary


def append_accessor(doc: dict, binary: bytearray, payload: bytes, component: int, count: int, kind: str, target: int, low=None, high=None) -> int:
    binary.extend(b"\0" * ((-len(binary)) % 4))
    offset = len(binary)
    binary.extend(payload)
    doc["bufferViews"].append({"buffer": 0, "byteOffset": offset, "byteLength": len(payload), "target": target})
    view = len(doc["bufferViews"]) - 1
    item = {"bufferView": view, "componentType": component, "count": count, "type": kind}
    if low is not None:
        item["min"], item["max"] = low, high
    doc["accessors"].append(item)
    return len(doc["accessors"]) - 1


def box_mesh(doc: dict, binary: bytearray, name: str, dimensions: tuple[float, float, float], material: int) -> int:
    x, y, z = (v / 2 for v in dimensions)
    faces = [((-1, 0, 0), [(-x,-y,-z),(-x,-y,z),(-x,y,z),(-x,y,-z)]), ((1, 0, 0), [(x,-y,z),(x,-y,-z),(x,y,-z),(x,y,z)]),
             ((0, -1, 0), [(-x,-y,z),(-x,-y,-z),(x,-y,-z),(x,-y,z)]), ((0, 1, 0), [(-x,y,-z),(-x,y,z),(x,y,z),(x,y,-z)]),
             ((0, 0, -1), [(x,-y,-z),(-x,-y,-z),(-x,y,-z),(x,y,-z)]), ((0, 0, 1), [(-x,-y,z),(x,-y,z),(x,y,z),(-x,y,z)])]
    positions, normals, uvs, indices = [], [], [], []
    for normal, corners in faces:
        base = len(positions)
        positions.extend(corners)
        normals.extend([normal] * 4)
        uvs.extend(((0,0),(1,0),(1,1),(0,1)))
        indices.extend((base, base+1, base+2, base, base+2, base+3))
    pos = append_accessor(doc, binary, b"".join(struct.pack("<fff", *v) for v in positions), 5126, len(positions), "VEC3", 34962, [-x,-y,-z], [x,y,z])
    nor = append_accessor(doc, binary, b"".join(struct.pack("<fff", *v) for v in normals), 5126, len(normals), "VEC3", 34962)
    uv = append_accessor(doc, binary, b"".join(struct.pack("<ff", *v) for v in uvs), 5126, len(uvs), "VEC2", 34962, [0,0], [1,1])
    idx = append_accessor(doc, binary, struct.pack("<" + "H" * len(indices), *indices), 5123, len(indices), "SCALAR", 34963, [0], [max(indices)])
    doc["meshes"].append({"name": name, "primitives": [{"attributes": {"POSITION": pos, "NORMAL": nor, "TEXCOORD_0": uv}, "indices": idx, "material": material}]})
    return len(doc["meshes"]) - 1


def plane_mesh(doc: dict, binary: bytearray, name: str, width: float, height: float, material: int) -> int:
    pos = append_accessor(doc, binary, struct.pack("<12f", -width/2,-height/2,0, width/2,-height/2,0, width/2,height/2,0, -width/2,height/2,0), 5126, 4, "VEC3", 34962, [-width/2,-height/2,0], [width/2,height/2,0])
    nor = append_accessor(doc, binary, struct.pack("<12f", *([0,0,1]*4)), 5126, 4, "VEC3", 34962)
    uv = append_accessor(doc, binary, struct.pack("<8f", 0,0,1,0,1,1,0,1), 5126, 4, "VEC2", 34962, [0,0], [1,1])
    idx = append_accessor(doc, binary, struct.pack("<6H", 0,1,2,0,2,3), 5123, 6, "SCALAR", 34963, [0], [3])
    doc["meshes"].append({"name": name, "primitives": [{"attributes": {"POSITION": pos, "NORMAL": nor, "TEXCOORD_0": uv}, "indices": idx, "material": material}]})
    return len(doc["meshes"]) - 1


def material(name: str, color: list[float], metallic: float, roughness: float) -> dict:
    return {"name": name, "doubleSided": True, "pbrMetallicRoughness": {"baseColorFactor": color, "metallicFactor": metallic, "roughnessFactor": roughness}}


def authored_document(name: str, materials: list[dict]) -> tuple[dict, bytearray]:
    return {"asset": {"version": "2.0", "generator": "Device Studio fixed generic screen concepts"}, "scene": 0,
            "scenes": [{"nodes": [0]}], "nodes": [{"name": name, "children": []}], "meshes": [], "materials": materials,
            "buffers": [{"byteLength": 0}], "bufferViews": [], "accessors": []}, bytearray()


def add_node(doc: dict, name: str, mesh: int, at=(0.0,0.0,0.0), rotation=None, parent=0) -> int:
    node = {"name": name, "mesh": mesh, "translation": list(at)}
    if rotation is not None:
        node["rotation"] = list(rotation)
    doc["nodes"].append(node)
    index = len(doc["nodes"]) - 1
    doc["nodes"][parent].setdefault("children", []).append(index)
    return index


def prune_unreachable_nodes(doc: dict) -> None:
    reachable = set()
    pending = [root for scene in doc.get("scenes", []) for root in scene.get("nodes", [])]
    while pending:
        index = pending.pop()
        if index in reachable:
            continue
        if not isinstance(index, int) or index < 0 or index >= len(doc["nodes"]):
            raise ValueError("scene hierarchy contains an invalid node reference")
        reachable.add(index)
        pending.extend(doc["nodes"][index].get("children", []))
    ordered = sorted(reachable)
    remap = {old: new for new, old in enumerate(ordered)}
    nodes = []
    for old in ordered:
        node = copy.deepcopy(doc["nodes"][old])
        if "children" in node:
            node["children"] = [remap[child] for child in node["children"]]
        nodes.append(node)
    for scene in doc.get("scenes", []):
        scene["nodes"] = [remap[root] for root in scene.get("nodes", [])]
    doc["nodes"] = nodes


def smart_display() -> bytes:
    doc, binary = authored_document("Generic Wedge Smart Display", [material("matte graphite polymer", [.09,.10,.12,1], .08,.55), material("neutral active display", [.003,.004,.007,1], 0,.08), material("speaker cloth", [.025,.028,.032,1], 0,.82), material("speaker perforations", [.004,.005,.007,1], .1,.45), material("polished edge", [.24,.25,.27,1], .5,.3)])
    body = box_mesh(doc,binary,"Wedge enclosure",(.180,.125,.090),0)
    add_node(doc,"SmartDisplayRoot",body,(0,.0625,0))
    screen = plane_mesh(doc,binary,"Active display plane",.151,.085,1)
    add_node(doc,"MainDisplay",screen,(0,.084,.048),(-.173648,0,0,.984808))
    grille = box_mesh(doc,binary,"Speaker grille",(.138,.018,.002),2)
    add_node(doc,"SpeakerGrille",grille,(0,.022,.046))
    for i in range(11):
        hole = box_mesh(doc,binary,f"Speaker port {i:02d}",(.002,.002,.001),3)
        add_node(doc,f"SpeakerPort{i:02d}",hole,((i-5)*.010,.022,.047))
    edge = box_mesh(doc,binary,"Display lower trim",(.156,.003,.002),4)
    add_node(doc,"DisplayLowerTrim",edge,(0,.039,.046))
    doc["buffers"][0]["byteLength"] = len(binary)
    return pack_glb(doc,bytes(binary))


def vehicle_console() -> bytes:
    doc, binary = authored_document("Generic Vehicle Center Stack Concept", [material("soft-touch charcoal", [.07,.075,.082,1], .12,.48), material("neutral active display", [.002,.003,.006,1], 0,.07), material("vent recess", [.015,.019,.023,1], .18,.42), material("control satin metal", [.32,.34,.37,1], .72,.27), material("button polymer", [.12,.13,.15,1], .25,.38)])
    panel = box_mesh(doc,binary,"Center stack shell",(.380,.250,.120),0)
    add_node(doc,"VehicleCenterStackRoot",panel,(0,.125,0))
    display = plane_mesh(doc,binary,"Center display active plane",.292,.126,1)
    add_node(doc,"CenterDisplay",display,(0,.168,.061),(-.087156,0,0,.996195))
    for row, y in enumerate((.079,.056)):
        vent = box_mesh(doc,binary,f"Air vent recess {row}",(.205,.010,.003),2)
        add_node(doc,f"AirVent{row}",vent,(0,y,.062))
    for i, x in enumerate((-.113,-.075,.075,.113)):
        knob = box_mesh(doc,binary,f"Rotary control {i}",(.018,.018,.012),3)
        add_node(doc,f"RotaryControl{i}",knob,(x,.035,.063))
    for i, x in enumerate((-.042,0,.042)):
        button = box_mesh(doc,binary,f"Physical button {i}",(.026,.012,.004),4)
        add_node(doc,f"PhysicalButton{i}",button,(x,.035,.061))
    doc["buffers"][0]["byteLength"] = len(binary)
    return pack_glb(doc,bytes(binary))


def digital_frame(source: bytes, wall: bool) -> bytes:
    if sha(source) != FRAME_SOURCE_SHA:
        raise ValueError("pinned Innerscene source hash changed")
    doc, original_bin = glb_parts(source)
    if len(doc["nodes"]) != 9 or doc["nodes"][7].get("name") != "Unprinted frame insert" or doc["meshes"][7]["primitives"][0].get("material") != 4:
        raise ValueError("Innerscene frame display insertion point changed")
    doc = copy.deepcopy(doc)
    binary = bytearray(original_bin)
    primitive = doc["meshes"][7]["primitives"][0]
    source_uv_index = primitive["attributes"].get("TEXCOORD_0")
    if source_uv_index is None:
        raise ValueError("Innerscene frame display has no target UV accessor")
    source_uv = doc["accessors"][source_uv_index]
    if source_uv.get("componentType") != 5126 or source_uv.get("type") != "VEC2":
        raise ValueError("Innerscene frame display UV accessor is not float VEC2")
    source_view = doc["bufferViews"][source_uv["bufferView"]]
    source_offset = source_view.get("byteOffset", 0)
    values = struct.unpack_from("<" + "f" * source_uv["count"] * 2, original_bin, source_offset)
    low = [min(values[axis::2]) for axis in (0, 1)]
    high = [max(values[axis::2]) for axis in (0, 1)]
    if any(not math.isfinite(v) for v in values) or any(high[i] <= low[i] for i in range(2)):
        raise ValueError("Innerscene frame screen UV bounds are invalid")
    normalized = []
    for u, v in zip(values[::2], values[1::2]):
        normalized.extend(((u - low[0]) / (high[0] - low[0]), (v - low[1]) / (high[1] - low[1])))
    private_uv = append_accessor(doc, binary, struct.pack("<" + "f" * len(normalized), *normalized), 5126, source_uv["count"], "VEC2", 34962, [0, 0], [1, 1])
    primitive["attributes"]["TEXCOORD_0"] = private_uv
    doc["materials"].append(copy.deepcopy(doc["materials"][4]))
    screen_material = doc["materials"][-1]
    screen_material["name"] = "Neutral project display surface"
    screen_material["pbrMetallicRoughness"]["baseColorFactor"] = [.003,.004,.006,1]
    screen_material["pbrMetallicRoughness"]["roughnessFactor"] = .09
    doc["meshes"][7]["primitives"][0]["material"] = len(doc["materials"]) - 1
    if wall:
        doc["nodes"][8]["children"] = [i for i in doc["nodes"][8]["children"] if i not in (5,6)]
        cleat = box_mesh(doc,binary,"Authored wall cleat",(.105,.018,.008),3)
        add_node(doc,"WallMountCleat",cleat,(0,.174,-.108), parent=8)
        prune_unreachable_nodes(doc)
        doc["buffers"][0]["byteLength"] = len(binary)
        return pack_glb(doc,bytes(binary))
    doc["buffers"][0]["byteLength"] = len(binary)
    return pack_glb(doc,bytes(binary))


def write(path: pathlib.Path, data: bytes) -> dict:
    if path.exists() and path.read_bytes() != data:
        raise ValueError(f"refusing to replace changed output: {path}")
    path.parent.mkdir(parents=True, exist_ok=True)
    if not path.exists():
        path.write_bytes(data)
    return {"path": path.relative_to(ROOT).as_posix(), "bytes": len(data), "sha256": sha(data)}


def main() -> None:
    outputs = [write(OUT/"smart-display/generic-wedge-smart-display.glb", smart_display()),
               write(OUT/"vehicle-center-stack/generic-center-stack.glb", vehicle_console())]
    source = FRAME_SOURCE.read_bytes()
    outputs.extend([write(OUT/"digital-frame/innerscene-frame-tabletop-v2.glb", digital_frame(source, False)),
                    write(OUT/"digital-frame/innerscene-frame-wall-v3.glb", digital_frame(source, True))])
    for item in outputs:
        print(f"{item['path']} {item['bytes']} {item['sha256']}")


if __name__ == "__main__":
    main()
