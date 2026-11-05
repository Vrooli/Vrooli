#!/usr/bin/env python3
"""Pinned, byte-preserving screen-UV normalization for one CC0 monitor GLB."""
import hashlib
import json
import math
import pathlib
import struct

ROOT = pathlib.Path(__file__).resolve().parent
SOURCE = ROOT / "ultrawide-computer-monitor.glb"
OUTPUT = ROOT / "ultrawide-computer-monitor-screen-uv-v1.glb"
SOURCE_SHA256 = "67e5c979d6755dd561c291f39ae5748849cb26f7bb78f97821166ecbd05e2bed"
SOURCE_BYTES = 109512
TARGET = (1, 1, 0, 1, 6, 8, 78212, 1728, 216)
UV_MIN = (0.00028384834877215326, 0.000688016414642334)
UV_MAX = (0.9997161626815796, 0.9993119835853577)

def glb_chunks(data):
    if len(data) < 20 or data[:4] != b"glTF" or struct.unpack_from("<I", data, 4)[0] != 2 or struct.unpack_from("<I", data, 8)[0] != len(data):
        raise ValueError("source is not a size-consistent GLB v2")
    chunks=[]; offset=12
    while offset < len(data):
        if offset + 8 > len(data): raise ValueError("truncated GLB chunk header")
        size, kind=struct.unpack_from("<II", data, offset); offset += 8
        end=offset + size
        if end > len(data): raise ValueError("truncated GLB chunk")
        chunks.append((kind, offset, end, data[offset:end])); offset=end
    if offset != len(data) or len(chunks) != 2 or chunks[0][0] != 0x4e4f534a or chunks[1][0] != 0x004e4942:
        raise ValueError("expected one JSON chunk followed by one BIN chunk")
    return chunks

def require_source_profile(doc):
    if doc.get("asset", {}).get("version") != "2.0" or len(doc.get("scenes", [])) != 1 or doc.get("scene", 0) != 0:
        raise ValueError("monitor source scene structure changed")
    if len(doc.get("nodes", [])) != 6 or len(doc.get("meshes", [])) != 5 or len(doc.get("materials", [])) != 3 or len(doc.get("images", [])) != 4:
        raise ValueError("monitor source inventory changed")
    if doc.get("extensionsUsed", []) or doc.get("extensionsRequired", []) or doc.get("animations") or doc.get("skins") or doc.get("cameras"):
        raise ValueError("monitor source now includes unsupported or unreviewed features")
    node_index, mesh_index, primitive_index, material_index, uv_index, view_index, offset, length, count = TARGET
    node=doc["nodes"][node_index]; mesh=doc["meshes"][mesh_index]; primitive=mesh["primitives"][primitive_index]
    if node.get("name") != "Display glass" or node.get("mesh") != mesh_index or node.get("children", []) or primitive.get("material") != material_index or primitive.get("mode", 4) != 4:
        raise ValueError("source display node/primitive binding changed")
    if primitive.get("attributes", {}).get("TEXCOORD_0") != uv_index:
        raise ValueError("source display UV0 accessor changed")
    accessor=doc["accessors"][uv_index]; view=doc["bufferViews"][view_index]
    if accessor.get("componentType") != 5126 or accessor.get("count") != count or accessor.get("type") != "VEC2" or accessor.get("bufferView") != view_index or accessor.get("byteOffset", 0) != 0:
        raise ValueError("source display UV accessor layout changed")
    if view.get("buffer") != 0 or view.get("byteOffset") != offset or view.get("byteLength") != length or view.get("byteStride", 8) != 8:
        raise ValueError("source display UV buffer view layout changed")
    refs=sum(1 for item in doc["accessors"] if item.get("bufferView") == view_index)
    if refs != 1: raise ValueError("display UV buffer view is shared")
    # Display material uses the dark inactive base-color texture and the source normal map.
    material=doc["materials"][material_index]
    base=material.get("pbrMetallicRoughness", {}).get("baseColorTexture", {}).get("index")
    normal=material.get("normalTexture", {}).get("index")
    if base != 3 or normal != 2 or doc["textures"][base].get("source") != 2 or doc["textures"][normal].get("source") != 0:
        raise ValueError("display material image bindings changed")
    if any(image.get("uri") for image in doc.get("images", [])):
        raise ValueError("external images are unsupported")

def transform():
    if SOURCE.is_symlink() or not SOURCE.is_file(): raise ValueError("pinned source must be a regular file")
    source=SOURCE.read_bytes()
    if len(source) != SOURCE_BYTES or hashlib.sha256(source).hexdigest() != SOURCE_SHA256:
        raise ValueError("pinned source byte identity mismatch")
    chunks=glb_chunks(source); doc=json.loads(chunks[0][3].rstrip(b" \t\r\n\0")); require_source_profile(doc)
    _, bin_start, bin_end, _=chunks[1]; start=bin_start+TARGET[6]; end=start+TARGET[7]
    updated=bytearray(source); values=[]
    for index in range(TARGET[8]):
        u,v=struct.unpack_from("<ff", source, start + index*8)
        if not math.isfinite(u) or not math.isfinite(v): raise ValueError("source UV contains a non-finite value")
        values.append((u,v))
    actual_min=tuple(min(pair[axis] for pair in values) for axis in range(2))
    actual_max=tuple(max(pair[axis] for pair in values) for axis in range(2))
    if any(abs(actual_min[i]-UV_MIN[i]) > 1e-9 or abs(actual_max[i]-UV_MAX[i]) > 1e-9 for i in range(2)):
        raise ValueError("source UV bounds differ from pinned values")
    spans=tuple(actual_max[i]-actual_min[i] for i in range(2))
    if min(spans) <= 0: raise ValueError("source UV span is invalid")
    for index,(u,v) in enumerate(values):
        normalized=((u-actual_min[0])/spans[0], (v-actual_min[1])/spans[1])
        if any(not math.isfinite(q) or q < -1e-7 or q > 1+1e-7 for q in normalized): raise ValueError("normalized UV leaves the unit rectangle")
        struct.pack_into("<ff", updated, start + index*8, *normalized)
    result=bytes(updated)
    # The only changed source range is the one unshared 216-pair display UV accessor.
    if result[:start] != source[:start] or result[end:] != source[end:] or result[start:end] == source[start:end]:
        raise ValueError("derived GLB changed bytes outside the pinned UV accessor")
    changed=sum(a != b for a,b in zip(source[start:end],result[start:end]))
    normalized_values=[struct.unpack_from("<ff", result, start+i*8) for i in range(TARGET[8])]
    out_min=tuple(min(pair[axis] for pair in normalized_values) for axis in range(2)); out_max=tuple(max(pair[axis] for pair in normalized_values) for axis in range(2))
    if out_min != (0.0,0.0) or out_max != (1.0,1.0): raise ValueError("derived UV bounds are not exactly unit rectangle")
    if OUTPUT.exists():
        if OUTPUT.is_symlink() or not OUTPUT.is_file() or OUTPUT.read_bytes() != result: raise ValueError("existing output differs; refusing overwrite")
    else:
        OUTPUT.write_bytes(result)
    return {"profile_id":"innerscene-monitor-screen-uv-affine-v1","input":{"name":SOURCE.name,"sha256":SOURCE_SHA256,"size_bytes":len(source)},"output":{"name":OUTPUT.name,"sha256":hashlib.sha256(result).hexdigest(),"size_bytes":len(result)},"changed_range_bin_offset_bytes":[TARGET[6],TARGET[6]+TARGET[7]],"changed_bytes_in_range":changed,"modified_accessor":{"index":TARGET[4],"buffer_view":TARGET[5],"component":"float32 VEC2","count":TARGET[8],"source_uv_min":actual_min,"source_uv_max":actual_max,"derived_uv_min":out_min,"derived_uv_max":out_max},"unchanged":"All GLB bytes outside this one UV accessor, including JSON, node transforms, all vertex positions/indices, materials, textures and embedded images."}

if __name__ == "__main__": print(json.dumps(transform(), indent=2, sort_keys=True))
