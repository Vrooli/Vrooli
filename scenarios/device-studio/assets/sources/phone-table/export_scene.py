"""Fixed Blender exporter for the E3 phone-study tabletop/cup geometry candidate."""
import hashlib
import json
import math
import struct
from pathlib import Path

import bpy


ROOT = Path(__file__).resolve().parent
MANIFEST_PATH = ROOT / "export-manifest.json"
INPUTS = ROOT / "inputs"
OUTPUTS = ROOT / "outputs"
GLB_NAME = "phone-table-scene.glb"
INVENTORY_NAME = "scene-inventory.json"
EXPECTED_NODES = {
    "TableTop", "TableLeg-NW", "TableLeg-NE", "TableLeg-SW", "TableLeg-SE",
    "TeaCup", "TeaCupHandle", "TeaSurface",
}
EXPECTED_MATERIALS = {"Pale warm mineral sides", "Travertine001 top material slot",
                      "Warm pale timber support legs", "Warm ivory glazed ceramic",
                      "Dark tea scale cue"}
EXPECTED_PROVENANCE_ROLES = {"phone_model", "screen_raster", "table_color", "table_normal", "table_roughness", "table_height"}
EXPECTED_INPUT_NAMES = {
    "phone_model": "phone.glb", "screen_raster": "phone-screen.png",
    "table_color": "table-color.jpg", "table_normal": "table-normal.jpg",
    "table_roughness": "table-roughness.jpg", "table_height": "table-height.jpg",
}


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def load_inputs():
    require(MANIFEST_PATH.is_file() and not MANIFEST_PATH.is_symlink(), "manifest must be a regular staged file")
    require(MANIFEST_PATH.stat().st_size <= 128 * 1024, "manifest exceeds fixed bound")
    raw = MANIFEST_PATH.read_bytes()
    require(len(raw) <= 128 * 1024, "manifest exceeds fixed bound")
    manifest = json.loads(raw.decode("utf-8"))
    require(manifest["profile_id"] == "device-studio-phone-table-scene-export-v1", "wrong export profile")
    require(list(bpy.app.version[:3]) == manifest["blender_version"], "Blender version mismatch")
    provenance = manifest["provenance_assets"]
    require(set(provenance) == EXPECTED_PROVENANCE_ROLES, "provenance role set differs")
    verified = {}
    for role, spec in provenance.items():
        require(spec["size_bytes"] > 0 and len(spec["sha256"]) == 64 and len(spec["source_evidence"]) <= 256,
                "provenance identity is malformed: " + role)
        require(spec.get("staged_name") == EXPECTED_INPUT_NAMES[role], "staged input name differs: " + role)
        path = INPUTS / spec["staged_name"]
        require(path.is_file() and not path.is_symlink(), "input must be a regular staged file: " + role)
        require(path.stat().st_size <= spec["max_bytes"], "input exceeds fixed size bound: " + role)
        data = path.read_bytes()
        require(0 < len(data) == spec["size_bytes"] <= spec["max_bytes"], "input size mismatch: " + role)
        require(sha256(data) == spec["sha256"], "input digest mismatch: " + role)
        verified[role] = {"sha256": sha256(data), "size_bytes": len(data)}
    require(manifest["output"] == {"glb_name": GLB_NAME, "inventory_name": INVENTORY_NAME, "max_bytes": 33554432},
            "fixed output contract differs")
    return manifest, sha256(raw), verified


def principled(name, color, roughness, metallic=0.0):
    value = bpy.data.materials.new(name)
    value.diffuse_color = (*color, 1.0)
    value.use_nodes = True
    shader = value.node_tree.nodes.get("Principled BSDF")
    shader.inputs["Base Color"].default_value = (*color, 1.0)
    shader.inputs["Roughness"].default_value = roughness
    shader.inputs["Metallic"].default_value = metallic
    return value


def cube(name, dimensions, location, material, bevel):
    bpy.ops.mesh.primitive_cube_add(size=1.0, location=location)
    obj = bpy.context.object
    obj.name = name
    obj.dimensions = dimensions
    bpy.ops.object.transform_apply(location=False, rotation=False, scale=True)
    obj.data.materials.append(material)
    if bevel:
        edge = obj.modifiers.new("Study edge radius", "BEVEL")
        edge.width, edge.segments, edge.limit_method = bevel, 3, "ANGLE"
        normals = obj.modifiers.new("Weighted normals", "WEIGHTED_NORMAL")
        normals.keep_sharp = True
        normals.weight = 50
    return obj


def apply_modifiers(obj):
    triangulate = obj.modifiers.new("Stable triangle primitives", "TRIANGULATE")
    triangulate.quad_method = "BEAUTY"
    bpy.ops.object.select_all(action="DESELECT")
    obj.select_set(True)
    bpy.context.view_layer.objects.active = obj
    for modifier in list(obj.modifiers):
        bpy.ops.object.modifier_apply(modifier=modifier.name)


def make_table(manifest, side, top, leg):
    geometry = manifest["geometry"]
    table = cube("TableTop", geometry["table_top_dimensions_meters"], geometry["table_top_center_meters_blender"], side,
                 geometry["table_top_bevel_meters"])
    table.data.materials.append(top)
    for face in table.data.polygons:
        if face.normal.z > 0.99:
            face.material_index = 1
    table["scene_role"] = "horizontal_support_surface"
    apply_modifiers(table)
    for name, center in geometry["table_legs"]["named_centers_meters_blender"].items():
        obj = cube(name, geometry["table_legs"]["dimensions_meters"], center, leg,
                   geometry["table_legs"]["bevel_meters"])
        obj["scene_role"] = "table_support_leg"
        apply_modifiers(obj)
    return table


def make_cup(manifest, ceramic, tea):
    geometry = manifest["geometry"]
    center = geometry["cup_center_meters_blender"]
    top_z = geometry["cup_base_height_meters_blender"]
    profile = geometry["cup_profile_rings_radius_height_meters"]
    segments, vertices, faces = geometry["cup_ring_segments"], [], []
    for radius, height in profile:
        vertices.extend((center[0] + radius * math.cos(2 * math.pi * i / segments),
                         center[1] + radius * math.sin(2 * math.pi * i / segments), top_z + height)
                        for i in range(segments))
    for ring in range(len(profile) - 1):
        for i in range(segments):
            a, b = ring * segments + i, ring * segments + (i + 1) % segments
            faces.append((a + segments, b + segments, b, a))
    floor_center, base_center = len(vertices), len(vertices) + 1
    vertices.extend(((center[0], center[1], top_z + profile[0][1]),
                     (center[0], center[1], top_z + profile[-1][1])))
    base_ring = (len(profile) - 1) * segments
    for i in range(segments):
        nxt = (i + 1) % segments
        faces.extend(((floor_center, i, nxt), (base_center, base_ring + nxt, base_ring + i)))
    mesh = bpy.data.meshes.new("TeaCup hollow vessel mesh")
    mesh.from_pydata(vertices, [], faces)
    mesh.update()
    cup = bpy.data.objects.new("TeaCup", mesh)
    bpy.context.scene.collection.objects.link(cup)
    cup.data.materials.append(ceramic)
    rim = cup.modifiers.new("Soft ceramic rim", "BEVEL")
    rim.width, rim.segments = geometry["cup_rim_bevel_meters"], geometry["cup_rim_bevel_segments"]
    cup.modifiers.new("Weighted vessel normals", "WEIGHTED_NORMAL")
    cup["scene_role"] = "ceramic_drink_vessel"
    apply_modifiers(cup)
    handle_spec = geometry["cup_handle"]
    bpy.ops.mesh.primitive_torus_add(major_radius=handle_spec["major_radius_meters"],
                                     minor_radius=handle_spec["minor_radius_meters"],
                                     major_segments=handle_spec["major_segments"],
                                     minor_segments=handle_spec["minor_segments"],
                                     location=(center[0] + handle_spec["offset_meters"][0],
                                               center[1] + handle_spec["offset_meters"][1],
                                               top_z + handle_spec["offset_meters"][2]),
                                     rotation=(math.pi / 2, 0, 0))
    handle = bpy.context.object
    handle.name = "TeaCupHandle"
    handle.data.materials.append(ceramic)
    handle["scene_role"] = "ceramic_vessel_handle"
    apply_modifiers(handle)
    tea_spec = geometry["tea_surface"]
    bpy.ops.mesh.primitive_cylinder_add(vertices=tea_spec["vertices"], radius=tea_spec["radius_meters"],
                                        depth=tea_spec["depth_meters"],
                                        location=(center[0], center[1], top_z + tea_spec["center_height_from_base_meters"]))
    surface = bpy.context.object
    surface.name = "TeaSurface"
    surface.data.materials.append(tea)
    surface["scene_role"] = "dark_tea_scale_cue"
    apply_modifiers(surface)


def parse_glb(data):
    require(len(data) >= 20 and data[:4] == b"glTF", "output is not a GLB")
    magic, version, declared = struct.unpack_from("<4sII", data, 0)
    require(version == 2 and declared == len(data), "GLB header length/version mismatch")
    chunks, offset = [], 12
    while offset < len(data):
        require(offset + 8 <= len(data), "truncated GLB chunk header")
        size, kind = struct.unpack_from("<II", data, offset)
        offset += 8
        require(size <= len(data) - offset, "GLB chunk exceeds declared file length")
        chunks.append((kind, data[offset:offset + size]))
        offset += size
    require(offset == len(data) and len(chunks) == 2 and chunks[0][0] == 0x4E4F534A and
            chunks[1][0] == 0x004E4942, "GLB must contain exactly one JSON and one BIN chunk")
    size, kind = struct.unpack_from("<II", data, 12)
    require(kind == 0x4E4F534A and 20 + size <= len(data), "GLB JSON chunk missing")
    doc = json.loads(data[20:20 + size].decode("utf-8").rstrip(" \t\r\n\0"))
    require(not any("uri" in item for group in ("buffers", "images") for item in doc.get(group, [])), "external GLB URI is forbidden")
    nodes, meshes = doc.get("nodes", []), doc.get("meshes", [])
    found = {}
    occurrences = []
    for index, node in enumerate(nodes):
        if node.get("name") in EXPECTED_NODES:
            require(node["name"] not in found, "duplicate geometry node name: " + node["name"])
            occurrences.append(node["name"])
            found[node["name"]] = {"node_index": index, "mesh_index": node.get("mesh"),
                                    "translation": node.get("translation", [0, 0, 0]),
                                    "rotation_xyzw": node.get("rotation", [0, 0, 0, 1]),
                                    "scale": node.get("scale", [1, 1, 1])}
    require(set(found) == EXPECTED_NODES, "exported geometry node inventory differs")
    require(len(occurrences) == len(EXPECTED_NODES), "duplicate geometry node name")
    materials = doc.get("materials", [])
    require({item.get("name") for item in materials} == EXPECTED_MATERIALS, "exported material inventory differs")
    accessors = doc.get("accessors", [])
    material_by_node = {}
    for name, node in found.items():
        require(node["mesh_index"] is not None, "scene node lacks mesh: " + name)
        primitives = meshes[node["mesh_index"]].get("primitives", [])
        require(primitives, "empty mesh primitive: " + name)
        node["primitives"] = []
        minima, maxima = [], []
        for i, primitive in enumerate(primitives):
            require(primitive.get("mode", 4) == 4, "only triangle primitives are supported")
            material_index = primitive.get("material")
            require(material_index is not None and material_index < len(materials), "primitive has no exported material")
            position = primitive.get("attributes", {}).get("POSITION")
            require(position is not None and position < len(accessors), "primitive has no bounded POSITION accessor")
            accessor = accessors[position]
            require(len(accessor.get("min", [])) == 3 and len(accessor.get("max", [])) == 3, "POSITION bounds missing")
            minima.append(accessor["min"])
            maxima.append(accessor["max"])
            material_name = materials[material_index]["name"]
            node["primitives"].append({"primitive_index": i, "mode": primitive.get("mode", 4), "material_index": material_index,
                                        "material_name": material_name, "position_accessor": position})
        node["mesh_bounds_local_meters"] = {"min": [min(row[a] for row in minima) for a in range(3)],
                                             "max": [max(row[a] for row in maxima) for a in range(3)]}
        material_by_node[name] = {item["material_name"] for item in node["primitives"]}
    require(material_by_node["TableTop"] == {"Pale warm mineral sides", "Travertine001 top material slot"}, "table top material primitives differ")
    require(all(material_by_node[name] == {"Warm pale timber support legs"} for name in EXPECTED_NODES if name.startswith("TableLeg-")), "table leg material differs")
    require(material_by_node["TeaCup"] == {"Warm ivory glazed ceramic"} and material_by_node["TeaCupHandle"] == {"Warm ivory glazed ceramic"}, "ceramic material slots differ")
    require(material_by_node["TeaSurface"] == {"Dark tea scale cue"}, "tea material slot differs")
    has_lights = any("KHR_lights_punctual" in node.get("extensions", {}) for node in nodes)
    require(not doc.get("cameras") and not doc.get("animations") and not doc.get("skins") and
            "KHR_lights_punctual" not in doc.get("extensionsUsed", []) and not has_lights,
            "camera, light, animation or skin data is unexpected")
    mesh_node_names = [node.get("name") for node in nodes if "mesh" in node]
    require(len(mesh_node_names) == len(EXPECTED_NODES) and set(mesh_node_names) == EXPECTED_NODES,
            "unexpected or missing mesh node")
    require(len(data) <= 32 * 1024 * 1024, "scene GLB exceeds fixed 32 MiB bound")
    return doc, found


def main():
    manifest, manifest_sha, verified_inputs = load_inputs()
    require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed output directory already exists or is a symlink")
    OUTPUTS.mkdir()
    bpy.ops.wm.read_factory_settings(use_empty=True)
    bpy.context.scene.unit_settings.system = "METRIC"
    bpy.context.scene.unit_settings.scale_length = 1.0
    materials = manifest["materials"]
    side = principled("Pale warm mineral sides", materials["table_sides"]["base_color_linear_rgb"], materials["table_sides"]["roughness"])
    top = principled("Travertine001 top material slot", materials["travertine_material_factor_linear_rgb"], materials["travertine_roughness_factor"])
    leg = principled("Warm pale timber support legs", materials["table_legs"]["base_color_linear_rgb"], materials["table_legs"]["roughness"])
    ceramic = principled("Warm ivory glazed ceramic", materials["ceramic"]["base_color_linear_rgb"], materials["ceramic"]["roughness"])
    tea = principled("Dark tea scale cue", materials["tea"]["base_color_linear_rgb"], materials["tea"]["roughness"])
    make_table(manifest, side, top, leg)
    make_cup(manifest, ceramic, tea)
    require({obj.name for obj in bpy.context.scene.objects if obj.type == "MESH"} == EXPECTED_NODES, "scene contains unexpected geometry")
    bpy.ops.object.select_all(action="DESELECT")
    for obj in bpy.context.scene.objects:
        if obj.type == "MESH":
            obj.select_set(True)
    path = OUTPUTS / GLB_NAME
    bpy.ops.export_scene.gltf(filepath=str(path), export_format="GLB", use_selection=True,
                              export_yup=True, export_apply=True, export_cameras=False,
                              export_lights=False, export_materials="EXPORT", export_extras=True,
                              export_texcoords=True, export_normals=True, export_animations=False)
    require(path.is_file() and not path.is_symlink(), "exported scene must be a regular GLB file")
    require(path.stat().st_size <= manifest["output"]["max_bytes"], "scene GLB exceeds manifest output bound")
    output = path.read_bytes()
    require(len(output) == path.stat().st_size, "scene GLB size changed while reading")
    gltf, nodes = parse_glb(output)
    require(not gltf.get("images") and not gltf.get("textures"), "external texture bindings must stay outside this GLB")
    parent = {}
    for index, node in enumerate(gltf.get("nodes", [])):
        for child in node.get("children", []):
            parent[child] = index
    for name in nodes:
        index = nodes[name]["node_index"]
        nodes[name]["parent_index"] = parent.get(index)
        nodes[name]["role"] = {
            "TableTop": "support_surface", "TeaCup": "prop_vessel", "TeaCupHandle": "prop_handle",
            "TeaSurface": "prop_surface", "TableLeg-NW": "support_leg", "TableLeg-NE": "support_leg",
            "TableLeg-SW": "support_leg", "TableLeg-SE": "support_leg",
        }[name]
    inventory = {
        "profile_id": manifest["profile_id"], "status": "geometry_candidate_unreviewed",
        "coordinate_convention": manifest["axis_export"]["target"],
        "axis_export": manifest["axis_export"],
        "provenance_assets": manifest["provenance_assets"], "verified_input_bytes": verified_inputs,
        "export_script_sha256": sha256(Path(__file__).read_bytes()),
        "manifest_sha256": manifest_sha, "glb_sha256": sha256(output),
        "glb_size_bytes": len(output), "glb_asset": gltf.get("asset"), "nodes": nodes,
        "materials": [{"index": i, "name": item["name"], "pbr": item.get("pbrMetallicRoughness", {})}
                      for i, item in enumerate(gltf.get("materials", []))],
        "gltf_extensions": {"used": gltf.get("extensionsUsed", []), "required": gltf.get("extensionsRequired", [])},
        "external_texture_artifacts": [
            {"binding_id": "travertine-color", "library_role": "texture_image", "socket": "base_color", "semantic": "base_color", **manifest["provenance_assets"]["table_color"]},
            {"binding_id": "travertine-normal", "library_role": "texture_image", "socket": "normal", "semantic": "normal_gl", **manifest["provenance_assets"]["table_normal"]},
            {"binding_id": "travertine-roughness", "library_role": "texture_image", "socket": "roughness", "semantic": "roughness", **manifest["provenance_assets"]["table_roughness"]},
            {"binding_id": "travertine-height", "library_role": "texture_image", "socket": "bump_height", "semantic": "bump_height", **manifest["provenance_assets"]["table_height"]},
        ],
        "scene_scope": "tabletop, four legs, vessel, handle and tea cue only; no phone, screen, camera, light or room backdrop",
        "rights": {"geometry": "Vrooli-authored procedural interpretation from the approved study scene; no manufacturer CAD claim",
                   "travertine_maps": "CC0-1.0 ambientCG Travertine001; separately bound texture artifacts, not embedded in this GLB"},
    }
    (OUTPUTS / INVENTORY_NAME).write_text(json.dumps(inventory, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
