"""Fixed exporter adding the approved study's wall, window and frame to the table GLB."""
import hashlib
import json
import math
import struct
from pathlib import Path

import bpy


ROOT = Path(__file__).resolve().parent
MANIFEST = ROOT / "full-scene-manifest.json"
INPUTS = ROOT / "inputs"
OUTPUTS = ROOT / "outputs"
EXPECTED_TABLE_NODES = {
    "TableLeg-NE", "TableLeg-NW", "TableLeg-SE", "TableLeg-SW",
    "TableTop", "TeaCup", "TeaCupHandle", "TeaSurface",
}
EXPECTED_ROOM_NODES = {
    "RoomBackWall", "WindowLightPanel", "WindowFrameUprightLeft",
    "WindowFrameUprightRight", "WindowFrameRailLower", "WindowFrameRailUpper",
}
TABLE_MATERIALS = {
    "Warm pale timber support legs", "Pale warm mineral sides",
    "Travertine001 top material slot", "Warm ivory glazed ceramic", "Dark tea scale cue",
}


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def read_manifest():
    require(MANIFEST.is_file() and not MANIFEST.is_symlink(), "fixed manifest must be a regular staged file")
    require(MANIFEST.stat().st_size <= 64 * 1024, "manifest exceeds fixed bound")
    raw = MANIFEST.read_bytes()
    manifest = json.loads(raw.decode("utf-8"))
    require(manifest["profile_id"] == "device-studio-phone-study-environment-export-v1", "wrong fixed profile")
    require(list(bpy.app.version[:3]) == manifest["blender_version"], "Blender version differs from manifest")
    spec = manifest["input"]["scene_glb"]
    source = INPUTS / spec["staged_name"]
    require(source.is_file() and not source.is_symlink(), "table input must be a regular staged file")
    require(source.stat().st_size == spec["size_bytes"] <= 2 * 1024 * 1024, "table input size mismatch or exceeds bound")
    data = source.read_bytes()
    require(len(data) == spec["size_bytes"] and sha256(data) == spec["sha256"], "table source digest mismatch")
    require(manifest["expected_output"]["glb_name"] == "phone-study-environment.glb" and
            manifest["expected_output"]["inventory_name"] == "environment-inventory.json" and
            manifest["expected_output"]["max_bytes"] == 33554432, "fixed output contract differs")
    return manifest, sha256(raw), data


def principled(name, color, roughness):
    material = bpy.data.materials.new(name)
    material.diffuse_color = (*color, 1.0)
    material.use_nodes = True
    shader = material.node_tree.nodes.get("Principled BSDF")
    require(shader is not None, "Blender Principled shader unavailable")
    shader.inputs["Base Color"].default_value = (*color, 1.0)
    shader.inputs["Roughness"].default_value = roughness
    shader.inputs["Metallic"].default_value = 0.0
    return material


def emission(name, color):
    material = bpy.data.materials.new(name)
    material.diffuse_color = (*color, 1.0)
    material.use_nodes = True
    nodes = material.node_tree.nodes
    links = material.node_tree.links
    nodes.clear()
    output = nodes.new("ShaderNodeOutputMaterial")
    shader = nodes.new("ShaderNodeBsdfPrincipled")
    shader.inputs["Base Color"].default_value = (0.0, 0.0, 0.0, 1.0)
    shader.inputs["Metallic"].default_value = 0.0
    shader.inputs["Roughness"].default_value = 1.0
    emission_color = shader.inputs.get("Emission Color") or shader.inputs.get("Emission")
    require(emission_color is not None, "Blender Principled emission input unavailable")
    emission_color.default_value = (*color, 1.0)
    strength = shader.inputs.get("Emission Strength")
    require(strength is not None, "Blender Principled emission strength unavailable")
    strength.default_value = 1.0
    links.new(shader.outputs["BSDF"], output.inputs["Surface"])
    return material


def add_box(name, spec, material):
    bpy.ops.mesh.primitive_cube_add(size=1.0, location=spec["location"])
    obj = bpy.context.object
    obj.name = name
    obj.dimensions = spec["dimensions"]
    bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
    obj.data.materials.append(material)
    bevel_width = spec["bevel"]
    if bevel_width:
        bevel = obj.modifiers.new("Study edge radius", "BEVEL")
        bevel.width, bevel.segments, bevel.limit_method = bevel_width, 3, "ANGLE"
        normals = obj.modifiers.new("Weighted normals", "WEIGHTED_NORMAL")
        normals.keep_sharp = True
        normals.weight = 50
        bpy.ops.object.select_all(action="DESELECT")
        obj.select_set(True)
        bpy.context.view_layer.objects.active = obj
        bpy.ops.object.modifier_apply(modifier=bevel.name)
        bpy.ops.object.modifier_apply(modifier=normals.name)
    return obj


def glb_document(data):
    require(len(data) >= 20 and data[:4] == b"glTF", "export is not GLB")
    magic, version, declared = struct.unpack_from("<4sII", data, 0)
    require(version == 2 and declared == len(data), "GLB version/length mismatch")
    offset, chunks = 12, []
    while offset < len(data):
        require(offset + 8 <= len(data), "truncated GLB chunk header")
        size, kind = struct.unpack_from("<II", data, offset)
        offset += 8
        require(size <= len(data) - offset, "GLB chunk exceeds file")
        chunks.append((kind, data[offset:offset + size]))
        offset += size
    require(len(chunks) == 2 and chunks[0][0] == 0x4E4F534A and chunks[1][0] == 0x004E4942,
            "GLB must have JSON and BIN chunks only")
    doc = json.loads(chunks[0][1].decode("utf-8").rstrip(" \t\r\n\0"))
    require(not any("uri" in item for group in ("buffers", "images") for item in doc.get(group, [])),
            "external GLB URI is forbidden")
    return doc


def build_inventory(doc, manifest, manifest_sha, input_bytes, output_bytes):
    nodes, meshes = doc.get("nodes", []), doc.get("meshes", [])
    expected = set(manifest["expected_output"]["expected_mesh_nodes"])
    require(len(nodes) == len(expected) and {n.get("name") for n in nodes} == expected,
            "emitted node set differs from closed profile")
    require(all("mesh" in node and not node.get("children") for node in nodes),
            "unexpected non-mesh node or hierarchy")
    require(not doc.get("cameras") and not doc.get("animations") and not doc.get("skins") and
            not doc.get("images") and not doc.get("textures") and
            "KHR_lights_punctual" not in doc.get("extensionsUsed", []), "unexpected camera/light/animation/texture data")
    materials = doc.get("materials", [])
    required_materials = TABLE_MATERIALS | {"Warm plaster backdrop", "Soft luminous window"}
    require({m.get("name") for m in materials} == required_materials, "material inventory differs")
    accessors = doc.get("accessors", [])
    inventory = {}
    total_triangles = 0
    for ni, node in enumerate(nodes):
        mesh_index = node["mesh"]
        primitives = meshes[mesh_index].get("primitives", [])
        require(primitives, "mesh node has no primitives")
        rows = []
        for pi, primitive in enumerate(primitives):
            require(primitive.get("mode", 4) == 4, "only triangle primitives are supported")
            position = primitive.get("attributes", {}).get("POSITION")
            require(position is not None and position < len(accessors), "POSITION accessor absent")
            acc = accessors[position]
            require(acc.get("type") == "VEC3" and acc.get("count", 0) >= 3 and
                    len(acc.get("min", [])) == 3 and len(acc.get("max", [])) == 3, "POSITION bounds are invalid")
            count = primitive.get("indices")
            index_count = accessors[count]["count"] if count is not None else acc["count"]
            require(index_count % 3 == 0, "triangle index count is not divisible by three")
            total_triangles += index_count // 3
            mat_index = primitive.get("material")
            require(mat_index is not None and 0 <= mat_index < len(materials), "material binding missing")
            rows.append({"primitive_index": pi, "mode": 4, "gltf_material_index": mat_index,
                         "material_name": materials[mat_index]["name"], "position_accessor": position,
                         "vertex_count": acc["count"], "local_bounds_meters": {"min": acc["min"], "max": acc["max"]}})
        inventory[node["name"]] = {
            "node_index": ni, "mesh_index": mesh_index, "parent_index": None,
            "translation": node.get("translation", [0, 0, 0]),
            "rotation_xyzw": node.get("rotation", [0, 0, 0, 1]),
            "scale": node.get("scale", [1, 1, 1]), "primitives": rows,
            "role": "environment_backdrop" if node["name"] in EXPECTED_ROOM_NODES else "imported_table_scene_part",
        }
    return {
        "profile_id": manifest["profile_id"], "status": "geometry_candidate_unreviewed",
        "coordinate_convention": manifest["coordinate_convention"],
        "source_table_glb_sha256": manifest["input"]["scene_glb"]["sha256"],
        "source_table_glb_bytes": input_bytes, "manifest_sha256": manifest_sha,
        "export_script_sha256": sha256(Path(__file__).read_bytes()),
        "glb_sha256": sha256(output_bytes), "glb_size_bytes": len(output_bytes),
        "triangle_count": total_triangles, "gltf_asset": doc.get("asset"), "nodes": inventory,
        "materials": [{"index": i, "name": m.get("name"),
                       "pbr": m.get("pbrMetallicRoughness", {}),
                       "emissive_factor_linear_rgb": m.get("emissiveFactor", [0, 0, 0])}
                      for i, m in enumerate(materials)],
        "gltf_extensions": {"used": doc.get("extensionsUsed", []), "required": doc.get("extensionsRequired", [])},
        "support_interface": manifest["support_interface"],
        "window_emission_conversion": manifest["materials"]["window"],
        "scene_scope": "table/cup GLB plus wall, luminous window panel, and four window-frame bars; no phone, camera, light, animation or embedded/external image",
    }


def main():
    manifest, manifest_sha, table_bytes = read_manifest()
    require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed output directory already exists")
    OUTPUTS.mkdir()
    bpy.ops.wm.read_factory_settings(use_empty=True)
    bpy.context.scene.unit_settings.system = "METRIC"
    bpy.context.scene.unit_settings.scale_length = 1.0
    source = INPUTS / manifest["input"]["scene_glb"]["staged_name"]
    bpy.ops.import_scene.gltf(filepath=str(source))
    imported = {obj.name for obj in bpy.context.scene.objects if obj.type == "MESH"}
    require(imported == EXPECTED_TABLE_NODES, "imported table node set differs from pinned input")
    room = manifest["room_geometry_blender_meters"]
    wall_spec = manifest["materials"]["wall"]
    wall = principled(wall_spec["name"], wall_spec["base_color_linear_rgb"], wall_spec["roughness"])
    window_spec = manifest["materials"]["window"]
    window = emission(window_spec["name"], window_spec["export_emissive_factor_linear_rgb"])
    for name in ("RoomBackWall", "WindowFrameUprightLeft", "WindowFrameUprightRight",
                 "WindowFrameRailLower", "WindowFrameRailUpper"):
        add_box(name, room[name], wall)
    add_box("WindowLightPanel", room["WindowLightPanel"], window)
    expected = set(manifest["expected_output"]["expected_mesh_nodes"])
    require({obj.name for obj in bpy.context.scene.objects if obj.type == "MESH"} == expected,
            "scene mesh set differs from fixed profile")
    bpy.ops.object.select_all(action="DESELECT")
    for obj in bpy.context.scene.objects:
        if obj.type == "MESH":
            obj.select_set(True)
    path = OUTPUTS / manifest["expected_output"]["glb_name"]
    bpy.ops.export_scene.gltf(filepath=str(path), export_format="GLB", use_selection=True,
                              export_yup=True, export_apply=True, export_cameras=False,
                              export_lights=False, export_materials="EXPORT", export_extras=True,
                              export_texcoords=True, export_normals=True, export_animations=False)
    require(path.is_file() and not path.is_symlink(), "output must be a regular GLB")
    require(path.stat().st_size <= manifest["expected_output"]["max_bytes"], "GLB exceeds fixed bound")
    output = path.read_bytes()
    require(len(output) == path.stat().st_size, "GLB changed while reading")
    doc = glb_document(output)
    inventory = build_inventory(doc, manifest, manifest_sha, len(table_bytes), output)
    (OUTPUTS / manifest["expected_output"]["inventory_name"]).write_text(
        json.dumps(inventory, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
