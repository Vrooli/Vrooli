"""Fixed Device Studio physical still adapter; all decisions arrive in one validated manifest."""
import json
import hashlib
import math
import os
import struct
import sys
import traceback

sys.dont_write_bytecode = True
if os.environ.get("PYTHONPYCACHEPREFIX"):
    sys.pycache_prefix = os.environ["PYTHONPYCACHEPREFIX"]

import bpy
from mathutils import Matrix, Quaternion, Vector


def fail(message):
    raise RuntimeError(message)


def persist_gpu_diagnostics(root, runtime):
    """Persist selection proof before rendering, then atomically update completion."""
    destination = os.path.join(root, "physical-gpu-diagnostics.json")
    temporary = destination + ".tmp"
    with open(temporary, "w", encoding="utf-8") as diagnostic_file:
        json.dump(runtime, diagnostic_file, sort_keys=True, separators=(",", ":"))
        diagnostic_file.write("\n")
        diagnostic_file.flush()
        os.fsync(diagnostic_file.fileno())
    os.replace(temporary, destination)


def initialize_empty_scene():
    bpy.ops.wm.read_factory_settings(use_empty=True)
    if len(bpy.data.objects) != 0:
        fail("physical renderer did not start from an empty factory scene")


def read_manifest(path):
    if not os.path.isabs(path) or os.path.basename(path) != "physical-worker-input.json":
        fail("worker input is not the fixed absolute attempt file")
    with open(path, "rb") as stream:
        raw = stream.read((4 << 20) + 1)
    if len(raw) > (4 << 20):
        fail("worker input exceeds its bound")
    manifest = json.loads(raw)
    valid_still = manifest.get("version") == 1 and manifest.get("output", {}).get("kind") == "still_png"
    output, profile = manifest.get("output", {}), manifest.get("profile", {})
    valid_preview = (
        manifest.get("version") == 3 and output.get("kind") == "still_png"
        and output.get("profile_id") == "device-studio-physical-preview-960x540-v1"
        and output.get("profile_version") == 1 and output.get("width") == 960
        and output.get("height") == 540 and output.get("output_name") == "physical-preview.png"
        and profile.get("samples") == 64 and profile.get("wall_time_seconds") == 30
    )
    valid_diagnostic = (
        manifest.get("version") == 3 and output.get("kind") == "still_png"
        and output.get("profile_id") == "device-studio-physical-diagnostic-960x540-v1"
        and output.get("profile_version") == 1 and output.get("width") == 960
        and output.get("height") == 540 and output.get("output_name") == "physical-diagnostic.png"
        and profile.get("samples") == 4 and profile.get("wall_time_seconds") == 30
    )
    valid_motion = manifest.get("version") == 2 and manifest.get("output", {}).get("kind") == "motion_mp4"
    valid_recording = (
        manifest.get("version") == 4 and manifest.get("output", {}).get("kind") == "motion_mp4"
        and manifest.get("output", {}).get("profile_id") == "device-studio-physical-recording-1080p-v1"
        and manifest.get("screen", {}).get("recording") is not None
    )
    valid_diagnostic_recording = (
        manifest.get("version") == 4 and output.get("kind") == "motion_mp4"
        and output.get("profile_id") == "device-studio-physical-recording-diagnostic-1080p-v1"
        and output.get("profile_version") == 1 and output.get("width") == 1920
        and output.get("height") == 1080 and output.get("frame_count", 0) <= 48
        and output.get("motion_profile") == "device-studio-physical-recording-diagnostic-motion-v1"
        and manifest.get("screen", {}).get("recording") is not None
        and profile.get("samples") == 4 and profile.get("wall_time_seconds") == 600
    )
    output = manifest.get("output", {})
    articulated = (
        manifest.get("version") == 5
        and output.get("articulation_profile") == "device-studio-single-revolute-hinge-v1"
        and ((output.get("kind") == "still_png" and output.get("profile_id") in (
            "device-studio-physical-still-2560-v1", "device-studio-physical-preview-960x540-v1",
            "device-studio-physical-diagnostic-960x540-v1"))
            or (output.get("kind") == "motion_mp4" and output.get("profile_id") in (
                "device-studio-physical-motion-1080p-v1", "device-studio-physical-motion-1080p-v2",
                "device-studio-physical-recording-1080p-v1")))
    )
    if not (valid_still or valid_preview or valid_diagnostic or valid_motion or valid_recording or valid_diagnostic_recording or articulated):
        fail("unsupported physical render manifest")
    return os.path.dirname(path), manifest


def alias(prefix, index, kind="NODE"):
    return "DS_%s_%s_%05d" % (prefix, kind, index)


def canonical_to_blender():
    # Canonical +Y up to Blender +Z up: C(x,y,z)=(x,-z,y).
    return Matrix(((1, 0, 0, 0), (0, 0, -1, 0), (0, 1, 0, 0), (0, 0, 0, 1)))


def trs(value):
    t, q, s = value["translation_m"], value["rotation_xyzw"], value["scale"]
    return Matrix.Translation(Vector((t["x"], t["y"], t["z"]))) @ Quaternion((q["w"], q["x"], q["y"], q["z"])).to_matrix().to_4x4() @ Matrix.Diagonal((s["x"], s["y"], s["z"], 1.0))


def as_blender_transform(value):
    c = canonical_to_blender()
    return c @ trs(value) @ c.inverted()


def import_glb(root, filename, prefix):
    path = os.path.join(root, filename)
    if os.path.commonpath((root, path)) != root or os.path.basename(path) != filename:
        fail("GLB escaped the attempt root")
    before = {obj.as_pointer() for obj in bpy.data.objects}
    bpy.ops.import_scene.gltf(filepath=path)
    imported = [obj for obj in bpy.data.objects if obj.as_pointer() not in before]
    by_name = {}
    for obj in imported:
        if obj.name in by_name:
            fail("Blender produced ambiguous normalized node names")
        by_name[obj.name] = obj
    return imported, by_name


def wrap_roots(name, objects, transform):
    root = bpy.data.objects.new(name, None)
    bpy.context.scene.collection.objects.link(root)
    root.matrix_world = transform
    for child in objects:
        if child.parent is None:
            previous = child.matrix_world.copy()
            child.parent = root
            child.matrix_world = transform @ previous
    return root


def link_node(nodes, kind, x, y):
    node = nodes.new(kind)
    node.location = (x, y)
    return node


def math_node(nodes, operation, x, y, left=None, right=None):
    node = link_node(nodes, "ShaderNodeMath", x, y)
    node.operation = operation
    if left is not None:
        node.inputs[0].default_value = left
    if right is not None:
        node.inputs[1].default_value = right
    return node


def blender_uv_layer_name(uv_layers, gltf_attribute):
    prefix = "TEXCOORD_"
    if not isinstance(gltf_attribute, str) or not gltf_attribute.startswith(prefix):
        fail("material UV binding is not a glTF TEXCOORD attribute")
    suffix = gltf_attribute[len(prefix):]
    if not suffix.isdigit() or (len(suffix) > 1 and suffix[0] == "0"):
        fail("material UV attribute index is malformed")
    index = int(suffix)
    if index >= len(uv_layers):
        fail("imported mesh lacks the requested glTF UV attribute")
    layer = uv_layers[index]
    if not layer.name or sum(1 for item in uv_layers if item.name == layer.name) != 1:
        fail("imported Blender UV layer name is ambiguous")
    return layer.name


def uv_vector(nodes, links, item, x, y):
    if item["coordinates"] == "UV_SET_AFFINE_V1":
        uv = link_node(nodes, "ShaderNodeUVMap", x, y)
        uv.uv_map = item.get("blender_uv_name") or item["uv_set"]
        separate = link_node(nodes, "ShaderNodeSeparateXYZ", x + 160, y)
        links.new(uv.outputs["UV"], separate.inputs[0])
        a, b, c, d, e, f = item["affine_uv"]
        u1 = math_node(nodes, "MULTIPLY", x + 300, y, a)
        links.new(separate.outputs["X"], u1.inputs[1])
        u2 = math_node(nodes, "MULTIPLY", x + 300, y - 50, b)
        links.new(separate.outputs["Y"], u2.inputs[1])
        u3 = math_node(nodes, "ADD", x + 450, y)
        links.new(u1.outputs[0], u3.inputs[0]); links.new(u2.outputs[0], u3.inputs[1])
        u4 = math_node(nodes, "ADD", x + 600, y, c)
        links.new(u3.outputs[0], u4.inputs[1])
        v1 = math_node(nodes, "MULTIPLY", x + 300, y - 150, d)
        links.new(separate.outputs["X"], v1.inputs[1])
        v2 = math_node(nodes, "MULTIPLY", x + 300, y - 200, e)
        links.new(separate.outputs["Y"], v2.inputs[1])
        v3 = math_node(nodes, "ADD", x + 450, y - 150)
        links.new(v1.outputs[0], v3.inputs[0]); links.new(v2.outputs[0], v3.inputs[1])
        v4 = math_node(nodes, "ADD", x + 600, y - 150, f)
        links.new(v3.outputs[0], v4.inputs[1])
        combine = link_node(nodes, "ShaderNodeCombineXYZ", x + 760, y - 60)
        links.new(u4.outputs[0], combine.inputs["X"]); links.new(v4.outputs[0], combine.inputs["Y"])
        return combine.outputs[0]
    if item["coordinates"] == "GENERATED_BOUNDS_X_NEG_Z_V1":
        uv_name = item.get("generated_uv_map")
        if not uv_name:
            fail("generated-bounds material has no mesh-local derived UV layer")
        uv = link_node(nodes, "ShaderNodeUVMap", x, y)
        uv.uv_map = uv_name
        return uv.outputs["UV"]
    fail("unsupported material coordinate mapping")


def set_alpha(material, spec, shader):
    mode = spec["alpha_mode"]
    if mode not in ("OPAQUE", "MASK", "BLEND"):
        fail("unsupported material alpha mode")
    if mode == "OPAQUE":
        return
    if mode == "MASK":
        cutoff = spec.get("alpha_cutoff")
        if cutoff is None or not hasattr(shader.inputs, "get") or shader.inputs.get("Alpha") is None:
            fail("masked alpha is not supported by this Blender build/profile")
        return cutoff
    material.surface_render_method = "DITHERED"
    return None


def make_material(spec, textures, uv_layers):
    spec = dict(spec)
    spec["inputs"] = [dict(item) for item in spec.get("inputs", [])]
    for item in spec["inputs"]:
        if item.get("coordinates") == "UV_SET_AFFINE_V1":
            item["blender_uv_name"] = blender_uv_layer_name(uv_layers, item.get("uv_set"))
    material = bpy.data.materials.new("physical-material:" + spec["id"])
    material.use_nodes = True
    nodes, links = material.node_tree.nodes, material.node_tree.links
    nodes.clear()
    output = link_node(nodes, "ShaderNodeOutputMaterial", 900, 0)
    if spec["surface_profile"] == "EMISSION_V1":
        shader = link_node(nodes, "ShaderNodeEmission", 650, 0)
        shader.inputs["Color"].default_value = (*spec["base_color_linear_rgba"][:3], 1.0)
        shader.inputs["Strength"].default_value = 1.0
        alpha_output = None
        for index, item in enumerate(spec.get("inputs", [])):
            if item["socket"] != "base_color":
                fail("emission profile only supports its declared base-color input")
            path = textures.get(item["texture_binding_id"])
            if not path:
                fail("emission material references an unstaged texture")
            # Texture interpretation is part of each input binding. A single
            # Blender Image datablock cannot safely serve both sRGB and
            # Non-Color inputs because colorspace_settings is mutable.
            image = bpy.data.images.load(path, check_existing=False)
            image.colorspace_settings.name = "sRGB"
            tex = link_node(nodes, "ShaderNodeTexImage", 0, -260 * index)
            tex.image = image
            tex.interpolation = "Linear"
            tex.extension = "REPEAT" if item["sampler"] == "REPEAT" else "EXTEND"
            links.new(uv_vector(nodes, links, item, -500, -260 * index), tex.inputs["Vector"])
            links.new(tex.outputs["Color"], shader.inputs["Color"])
            alpha_output = tex.outputs.get("Alpha")
        mode = spec["alpha_mode"]
        if mode != "OPAQUE":
            alpha = alpha_output
            if alpha is None:
                value = link_node(nodes, "ShaderNodeValue", 350, -400)
                value.outputs[0].default_value = spec["base_color_linear_rgba"][3]
                alpha = value.outputs[0]
            else:
                factor = spec["base_color_linear_rgba"][3]
                if factor != 1.0:
                    multiply = math_node(nodes, "MULTIPLY", 200, -400, factor)
                    links.new(alpha, multiply.inputs[1])
                    alpha = multiply.outputs[0]
            if mode == "MASK":
                # glTF MASK keeps alpha >= cutoff. Invert the strict-less
                # predicate so the exact cutoff boundary remains opaque.
                threshold = math_node(nodes, "LESS_THAN", 450, -400)
                links.new(alpha, threshold.inputs[0])
                threshold.inputs[1].default_value = spec["alpha_cutoff"]
                alpha = threshold.outputs[0]
            transparent = link_node(nodes, "ShaderNodeBsdfTransparent", 650, -250)
            mix = link_node(nodes, "ShaderNodeMixShader", 850, -100)
            if mode == "MASK":
                links.new(alpha, mix.inputs[0]); links.new(shader.outputs[0], mix.inputs[1]); links.new(transparent.outputs[0], mix.inputs[2])
            else:
                links.new(alpha, mix.inputs[0]); links.new(transparent.outputs[0], mix.inputs[1]); links.new(shader.outputs[0], mix.inputs[2])
            links.new(mix.outputs[0], output.inputs["Surface"])
            material.surface_render_method = "DITHERED"
        else:
            links.new(shader.outputs[0], output.inputs["Surface"])
        return material
    if spec["surface_profile"] != "PRINCIPLED_V1":
        fail("unsupported scene material surface profile")
    shader = link_node(nodes, "ShaderNodeBsdfPrincipled", 650, 0)
    shader.inputs["Base Color"].default_value = spec["base_color_linear_rgba"]
    shader.inputs["Metallic"].default_value = spec["metallic"]
    shader.inputs["Roughness"].default_value = spec["roughness"]
    base_color = shader.inputs["Base Color"]
    normal_output, bump_node, alpha_output = None, None, None
    for index, item in enumerate(spec.get("inputs", [])):
        image_path = textures.get(item["texture_binding_id"])
        if not image_path:
            fail("scene material references an unstaged texture role")
        image = bpy.data.images.load(image_path, check_existing=False)
        image.colorspace_settings.name = "sRGB" if item["color_space"] == "SRGB" else "Non-Color"
        tex = link_node(nodes, "ShaderNodeTexImage", 0, -260 * index)
        tex.image = image
        tex.interpolation = "Linear"
        tex.extension = "REPEAT" if item["sampler"] == "REPEAT" else "EXTEND"
        links.new(uv_vector(nodes, links, item, -500, -260 * index), tex.inputs["Vector"])
        socket = item["socket"]
        if socket == "base_color":
            links.new(tex.outputs["Color"], base_color)
            alpha_output = tex.outputs.get("Alpha")
        elif socket == "roughness":
            if item["channel"] != "RED":
                fail("roughness input must select its declared red channel")
            split = link_node(nodes, "ShaderNodeSeparateColor", 220, -260 * index - 80)
            split.mode = "RGB"
            links.new(tex.outputs["Color"], split.inputs["Color"])
            links.new(split.outputs["Red"], shader.inputs["Roughness"])
        elif socket == "normal":
            if item.get("normal_convention") != "OPENGL_TANGENT_PLUS_Y_V1":
                fail("unsupported tangent-space normal convention")
            normal = link_node(nodes, "ShaderNodeNormalMap", 350, -260 * index)
            normal.inputs["Strength"].default_value = item.get("normal_strength", 1.0)
            normal.uv_map = item.get("generated_uv_map") or blender_uv_layer_name(uv_layers, item.get("uv_set", ""))
            if not normal.uv_map:
                fail("tangent normal input lacks a declared UV basis")
            links.new(tex.outputs["Color"], normal.inputs["Color"])
            normal_output = normal.outputs["Normal"]
        elif socket == "bump_height":
            bump = link_node(nodes, "ShaderNodeBump", 350, -260 * index)
            bump.inputs["Strength"].default_value = item.get("bump_strength", 0.0)
            bump.inputs["Distance"].default_value = item.get("bump_distance_m", 0.0)
            split = link_node(nodes, "ShaderNodeSeparateColor", 220, -260 * index - 80)
            split.mode = "RGB"
            links.new(tex.outputs["Color"], split.inputs["Color"])
            links.new(split.outputs["Red"], bump.inputs["Height"])
            bump_node = bump
        else:
            fail("unsupported material input socket")
    grade = spec.get("grade")
    if grade is not None:
        if grade["recipe"] != "HUE_SATURATION_THEN_LINEAR_MIX_V1":
            fail("unsupported scene material grade recipe")
        hue = link_node(nodes, "ShaderNodeHueSaturation", 350, 250)
        hue.inputs["Hue"].default_value = 0.5
        hue.inputs["Saturation"].default_value = grade["saturation"]
        hue.inputs["Value"].default_value = grade["value"]
        source = base_color.links[0].from_socket if base_color.is_linked else None
        if source is None:
            rgb = link_node(nodes, "ShaderNodeRGB", 100, 250)
            rgb.outputs[0].default_value = spec["base_color_linear_rgba"]
            source = rgb.outputs[0]
        links.new(source, hue.inputs["Color"])
        mix = link_node(nodes, "ShaderNodeMixRGB", 500, 250)
        mix.blend_type = "MIX"
        mix.inputs[0].default_value = grade["mix_factor"]
        links.new(hue.outputs["Color"], mix.inputs[1])
        mix.inputs[2].default_value = (*grade["mix_linear_rgb"], 1.0)
        for old_link in list(base_color.links):
            links.remove(old_link)
        links.new(mix.outputs[0], base_color)
    if bump_node is not None:
        if normal_output is not None:
            links.new(normal_output, bump_node.inputs["Normal"])
        links.new(bump_node.outputs["Normal"], shader.inputs["Normal"])
    elif normal_output is not None:
        links.new(normal_output, shader.inputs["Normal"])
    cutoff = set_alpha(material, spec, shader)
    if cutoff is not None:
        if alpha_output is None:
            alpha_value = link_node(nodes, "ShaderNodeValue", 250, -700)
            alpha_value.outputs[0].default_value = spec["base_color_linear_rgba"][3]
            alpha_source = alpha_value.outputs[0]
        elif spec["base_color_linear_rgba"][3] == 1.0:
            alpha_source = alpha_output
        else:
            alpha_scale = math_node(nodes, "MULTIPLY", 250, -700, spec["base_color_linear_rgba"][3])
            links.new(alpha_output, alpha_scale.inputs[1])
            alpha_source = alpha_scale.outputs[0]
        # MASK semantics retain equality at alpha_cutoff (alpha >= cutoff).
        threshold = link_node(nodes, "ShaderNodeMath", 450, -700)
        threshold.operation = "LESS_THAN"
        threshold.inputs[1].default_value = cutoff
        links.new(alpha_source, threshold.inputs[0])
        transparent = link_node(nodes, "ShaderNodeBsdfTransparent", 650, -250)
        mix_shader = link_node(nodes, "ShaderNodeMixShader", 850, -100)
        links.new(threshold.outputs[0], mix_shader.inputs[0])
        links.new(shader.outputs[0], mix_shader.inputs[1])
        links.new(transparent.outputs[0], mix_shader.inputs[2])
        links.new(mix_shader.outputs[0], output.inputs["Surface"])
        material.surface_render_method = "DITHERED"
    else:
        if spec["alpha_mode"] == "BLEND":
            if alpha_output is None:
                shader.inputs["Alpha"].default_value = spec["base_color_linear_rgba"][3]
            elif spec["base_color_linear_rgba"][3] == 1.0:
                links.new(alpha_output, shader.inputs["Alpha"])
            else:
                alpha_scale = math_node(nodes, "MULTIPLY", 250, -700, spec["base_color_linear_rgba"][3])
                links.new(alpha_output, alpha_scale.inputs[1])
                links.new(alpha_scale.outputs[0], shader.inputs["Alpha"])
        links.new(shader.outputs[0], output.inputs["Surface"])
    return material


def bind_scene_materials(root, manifest, scene_candidate, objects):
    materials = {item["id"]: item for item in scene_candidate["materials"]}
    textures = {asset["binding_id"]: os.path.join(root, asset["name"]) for asset in manifest["assets"] if asset["role"] == "scene_texture"}
    artifacts = {artifact["id"]: artifact for artifact in scene_candidate["artifacts"]}
    assigned = {}
    for slot in scene_candidate["material_slots"]:
        part = next((p for p in scene_candidate["parts"] if p["id"] == slot["part_id"]), None)
        if part is None:
            fail("material slot references an unknown scene part")
        obj = objects.get(alias("scene", part["node_index"]))
        if obj is None or obj.type != "MESH":
            fail("scene material slot does not resolve to the imported primitive slot")
        node = manifest["scene_inventory"]["nodes"][part["node_index"]]
        if node.get("mesh") is None:
            fail("scene material part has no pinned mesh")
        mesh = manifest["scene_inventory"]["meshes"][node["mesh"]]
        primitive = next((p for p in mesh["primitives"] if p["index"] == slot["primitive_index"]), None)
        if primitive is None:
            fail("scene material primitive is outside the pinned mesh inventory")
        material_alias = alias("scene", primitive["material"], "MATERIAL")
        slots = [i for i, material in enumerate(obj.data.materials) if material and material.name == material_alias]
        if len(slots) != 1:
            fail("scene material alias did not resolve to exactly one imported slot")
        assignment_key = (obj.name, slots[0])
        if assignment_key in assigned and assigned[assignment_key] != slot["material_id"]:
            fail("one imported material slot maps to conflicting candidate primitive materials")
        assigned[assignment_key] = slot["material_id"]
        source_spec = materials.get(slot["material_id"])
        if source_spec is None:
            fail("scene material slot references a missing material")
        spec = dict(source_spec)
        artifact = artifacts.get(part["artifact_binding_id"])
        if artifact is None:
            fail("scene part has no geometry artifact binding")
        unit_to_m = {"m": 1.0, "cm": 0.01, "mm": 0.001}.get(artifact["source_unit"])
        if unit_to_m is None:
            fail("scene geometry source unit is unsupported")
        spec["inputs"] = [dict(item) for item in source_spec.get("inputs", [])]
        for input_index, item in enumerate(spec["inputs"]):
            if item["coordinates"] == "GENERATED_BOUNDS_X_NEG_Z_V1":
                item["generated_uv_map"] = install_generated_bounds_uv(obj, part["node_index"], primitive, item["physical_repeat_m"], unit_to_m, slot, input_index)
            elif item["coordinates"] == "UV_SET_AFFINE_V1":
                item["blender_uv_name"] = blender_uv_layer_name(obj.data.uv_layers, item.get("uv_set"))
        obj.data.materials[slots[0]] = make_material(spec, textures, obj.data.uv_layers)


def install_generated_bounds_uv(obj, node_index, primitive, repeat_m, unit_to_m, slot, input_index):
    if not repeat_m or len(repeat_m) != 2 or min(repeat_m) <= 0:
        fail("generated-bounds material lacks physical repeat dimensions")
    layer_name = "DS_GEN_%05d_%05d_%05d" % (node_index, slot["primitive_index"], input_index)
    if len(layer_name) > 63:
        fail("derived UV layer name exceeds Blender's bound")
    if obj.data.users > 1:
        obj.data = obj.data.copy()
    layer = obj.data.uv_layers.get(layer_name) or obj.data.uv_layers.new(name=layer_name)
    primitive_min = primitive["position_min"]
    primitive_max = primitive["position_max"]
    min_x = primitive_min[0]
    min_y = -primitive_max[2]
    max_x = primitive_max[0]
    max_y = -primitive_min[2]
    if max_x <= min_x or max_y <= min_y:
        fail("generated-bounds primitive has degenerate canonical X/-Z bounds")
    repeat_x, repeat_y = repeat_m
    for loop in obj.data.loops:
        vertex = obj.data.vertices[loop.vertex_index].co
        uv = layer.data[loop.index].uv
        uv[0] = ((vertex.x - min_x) * unit_to_m) / repeat_x
        uv[1] = ((vertex.y - min_y) * unit_to_m) / repeat_y
    return layer_name


def bind_screen(root, manifest, mapping, objects):
    obj = objects.get(alias("device", mapping["node_index"]))
    if obj is None or obj.type != "MESH":
        fail("screen node did not import as one mesh")
    material_name = alias("device", mapping["material_index"], "MATERIAL")
    slots = [slot for slot in obj.material_slots if slot.material and slot.material.name == material_name]
    if len(slots) != 1:
        fail("screen material did not resolve uniquely after import")
    recording = manifest["screen"].get("recording") is not None
    raster_name = "physical-recording-screen.png" if recording else manifest["screen"]["raster"]["name"]
    image = bpy.data.images.load(os.path.join(root, raster_name), check_existing=False)
    image.colorspace_settings.name = "sRGB"
    material = bpy.data.materials.new("physical-screen-pinned")
    material.use_nodes = True
    nodes, links = material.node_tree.nodes, material.node_tree.links
    nodes.clear()
    output = link_node(nodes, "ShaderNodeOutputMaterial", 500, 0)
    emission = link_node(nodes, "ShaderNodeEmission", 250, 0)
    texcoord = link_node(nodes, "ShaderNodeUVMap", -350, 0)
    texcoord.uv_map = blender_uv_layer_name(obj.data.uv_layers, manifest["screen"]["uv_set"])
    if mapping.get("blender_uv_layer") != texcoord.uv_map:
        fail("screen UV layer differs from the pinned GLB importer mapping")
    texture = link_node(nodes, "ShaderNodeTexImage", -100, 0)
    texture.image = image
    texture.interpolation = "Linear"
    links.new(texcoord.outputs["UV"], texture.inputs["Vector"])
    links.new(texture.outputs["Color"], emission.inputs["Color"])
    links.new(emission.outputs[0], output.inputs["Surface"])
    slots[0].material = material
    return image


def build_world(scene, environment):
    if environment["model"] != "CONSTANT_V1":
        fail("unsupported scene environment model")
    world = bpy.data.worlds.new("physical-world")
    world.use_nodes = True
    background = world.node_tree.nodes.get("Background")
    if background is None:
        fail("Blender world background node is unavailable")
    background.inputs["Color"].default_value = (*environment["linear_rgb"], 1.0)
    background.inputs["Strength"].default_value = environment["strength"]
    scene.world = world


def build_lights(scene, lights):
    c = canonical_to_blender()
    for spec in lights:
        if spec["local_axis_profile"] != "AREA_XY_MINUS_Z_V1":
            fail("unsupported scene light axis profile")
        data = bpy.data.lights.new("physical-light:" + spec["id"], "AREA")
        data.shape = "RECTANGLE" if spec["shape"] == "RECTANGLE" else "DISK"
        data.size = spec["dimensions_m"][0]
        if data.shape == "RECTANGLE":
            data.size_y = spec["dimensions_m"][1]
        data.energy = spec["power_watts"]
        data.color = spec["linear_rgb"]
        obj = bpy.data.objects.new(data.name, data)
        scene.collection.objects.link(obj)
        obj.matrix_world = c @ trs(spec["frame"])


def resolve_camera_focus(camera_spec, camera_object, device_root, scene_objects):
    focus = camera_spec.get("focus")
    fstop = camera_spec.get("dof_f_number")
    if not focus or fstop is None:
        camera_object.data.dof.use_dof = False
        return
    target = None
    if focus["mode"] == "SELECTED_DEVICE_ROOT":
        target = device_root
    elif focus["mode"] == "SCENE_PART":
        part = next((p for p in scene_objects["candidate"]["parts"] if p["id"] == focus.get("scene_part_id")), None)
        target = scene_objects["objects"].get(alias("scene", part["node_index"])) if part else None
    elif focus["mode"] == "DISTANCE":
        distance = focus.get("distance_m")
        if distance is None or distance <= 0:
            fail("camera focus distance is invalid")
        empty = bpy.data.objects.new("physical-focus-distance", None)
        bpy.context.scene.collection.objects.link(empty)
        empty.location = camera_object.location + camera_object.matrix_world.to_3x3() @ Vector((0, 0, -distance))
        target = empty
    else:
        fail("unsupported camera focus mode")
    if target is None:
        fail("selected camera focus target is unavailable")
    camera_object.data.dof.use_dof = True
    camera_object.data.dof.focus_object = target
    camera_object.data.dof.aperture_fstop = fstop


def setup_camera(scene, camera_spec, device_root, scene_candidate, scene_objects, camera_motion=None):
    data = bpy.data.cameras.new("physical-camera")
    obj = bpy.data.objects.new("physical-camera", data)
    scene.collection.objects.link(obj)
    frame = trs(camera_spec["frame"])
    if camera_motion is not None:
        frame = frame @ Matrix.Translation(Vector((camera_motion["translation_m"]["x"], camera_motion["translation_m"]["y"], camera_motion["translation_m"]["z"]))) @ Quaternion((camera_motion["rotation_xyzw"]["w"], camera_motion["rotation_xyzw"]["x"], camera_motion["rotation_xyzw"]["y"], camera_motion["rotation_xyzw"]["z"])).to_matrix().to_4x4()
    obj.matrix_world = canonical_to_blender() @ frame
    if camera_spec["projection"] == "PERSPECTIVE":
        data.type = "PERSP"
        data.lens = camera_spec["focal_length_mm"]
        data.sensor_width = camera_spec["sensor_width_mm"]
        data.sensor_height = camera_spec["sensor_height_mm"]
        data.sensor_fit = camera_spec["sensor_fit"]
    elif camera_spec["projection"] == "ORTHOGRAPHIC":
        data.type = "ORTHO"
        data.ortho_scale = camera_spec["orthographic_vertical_span_m"]
    else:
        fail("unsupported camera projection")
    data.clip_start = camera_spec["near_clip_m"]
    data.clip_end = camera_spec["far_clip_m"]
    scene.camera = obj
    resolve_camera_focus(camera_spec, obj, device_root, {"candidate": scene_candidate, "objects": scene_objects})
    return obj


def configure_scene(scene, manifest, root):
    profile, output = manifest["profile"], manifest["output"]
    if profile["color_management"] != "device-studio-blender-standard-v1":
        fail("unsupported physical color-management profile")
    scene.display_settings.display_device = "sRGB"
    scene.render.engine = "CYCLES"
    scene.cycles.samples = profile["samples"]
    scene.cycles.seed = profile["seed"]
    scene.cycles.use_adaptive_sampling = profile["adaptive_sampling"]
    scene.cycles.use_denoising = profile["denoise"]
    scene.cycles.max_bounces = 8
    scene.cycles.caustics_reflective = False
    scene.cycles.caustics_refractive = False
    scene.render.threads_mode = "FIXED"
    scene.render.threads = profile["threads"]
    scene.render.resolution_x = output["width"]
    scene.render.resolution_y = output["height"]
    scene.render.resolution_percentage = 100
    scene.render.image_settings.file_format = "PNG"
    scene.render.image_settings.color_mode = "RGBA"
    scene.render.image_settings.color_depth = "8"
    scene.render.film_transparent = False
    scene.view_settings.view_transform = profile["color_view"]
    scene.view_settings.look = profile["color_look"]
    scene.view_settings.exposure = float(profile["exposure"])
    scene.view_settings.gamma = float(profile["gamma"])
    scene.render.filepath = os.path.join(root, output["output_name"])


def place_device(root_objects, device_candidate, manifest, device_objects):
    candidate, placement = device_candidate, manifest["placement"]
    support = next((x for x in candidate["support_frames"] if x["id"] == placement["device_support_id"]), None)
    part = next((x for x in candidate["parts"] if support and x["id"] == support["part_id"]), None)
    node = device_objects.get(alias("device", part["node_index"])) if part else None
    if support is None or part is None or node is None:
        fail("selected device support frame did not resolve to imported node")
    source = trs(candidate["source_to_canonical"])
    sample = manifest["output"]["samples"][0]
    sampled = sample.get("device_root")
    root_rest = trs(placement["device_root_rest"])
    if sampled is not None:
        sampled_root = trs({"translation_m": sampled["translation_m"], "rotation_xyzw": sampled["rotation_xyzw"], "scale": {"x": 1.0, "y": 1.0, "z": 1.0}})
        # Motion root is a delta in the support frame and therefore precedes
        # the saved device rest transform in column-vector composition.
        root_rest = sampled_root @ root_rest
    wrapper_matrix = physical_device_placement_matrix(placement, root_rest, source)
    # The selected support ID is a compatibility/contact assertion only. The
    # saved root origin and source transform define placement; no inverse of
    # the support node is allowed to cancel that pinned source correction.
    return wrap_roots("physical-device-placement", root_objects, wrapper_matrix)


def attach_physical_hinge(device_candidate, motion_candidate, device_objects):
    joint_ids = [track["target"]["joint_id"] for track in motion_candidate.get("tracks", [])
                 if track.get("target", {}).get("joint_id")]
    if len(joint_ids) != 1:
        fail("single-hinge profile requires exactly one joint motion track")
    joint_id = joint_ids[0]
    joints = [joint for joint in device_candidate.get("definition", {}).get("joints", [])
              if joint.get("id") == joint_id]
    if len(joints) != 1:
        fail("single-hinge motion does not resolve one device joint")
    joint = joints[0]
    if joint.get("status") not in ("candidate_only", "qualified"):
        fail("single-hinge joint is not admitted for local review rendering")
    axis, pivot = joint.get("axis"), joint.get("pivot_m")
    lower, upper = joint.get("lower_rad"), joint.get("upper_rad")
    if (not isinstance(axis, dict) or not isinstance(pivot, dict) or lower is None or upper is None
            or not all(math.isfinite(float(axis.get(k, float("nan")))) for k in ("x", "y", "z"))
            or abs(math.sqrt(sum(float(axis[k]) ** 2 for k in ("x", "y", "z"))) - 1.0) > 1e-6
            or not all(math.isfinite(float(pivot.get(k, float("nan")))) for k in ("x", "y", "z"))
            or not math.isfinite(float(lower)) or not math.isfinite(float(upper)) or float(lower) >= float(upper)):
        fail("single-hinge axis, pivot, or limits are invalid")
    parts = {part["id"]: part for part in device_candidate.get("parts", [])}
    parent_part, child_part = parts.get(joint.get("parent_part")), parts.get(joint.get("child_part"))
    if parent_part is None or child_part is None or child_part.get("parent_id") != parent_part["id"]:
        fail("single-hinge joint differs from semantic part hierarchy")
    parent = device_objects.get(alias("device", parent_part["node_index"]))
    child = device_objects.get(alias("device", child_part["node_index"]))
    if parent is None or child is None or parent is child:
        fail("single-hinge semantic parts did not resolve to distinct imported nodes")
    current = child.parent
    while current is not None and current is not parent:
        if current.type == "MESH" and current.data is not None:
            fail("single-hinge path contains an unbound mesh intermediary")
        current = current.parent
    if current is not parent:
        fail("single-hinge child is not below its parent in the imported GLB")
    # place_device has just changed the imported root's parent transform. Flush
    # that transform before deriving a relative rest matrix from world space.
    bpy.context.view_layer.update()
    rest_relative = parent.matrix_world.inverted() @ child.matrix_world
    hinge = bpy.data.objects.new("physical-hinge:" + joint_id, None)
    bpy.context.scene.collection.objects.link(hinge)
    hinge.parent = parent
    hinge.matrix_parent_inverse = Matrix.Identity(4)
    hinge.matrix_basis = Matrix.Identity(4)
    child.parent = hinge
    child.matrix_parent_inverse = Matrix.Identity(4)
    child.matrix_basis = rest_relative
    basis = canonical_to_blender().to_3x3()
    # Candidate hinge coordinates are parent-local canonical +Y-up values;
    # Blender's imported node-local basis is +Z-up, so convert the point and
    # direction once without applying the outer source/root transform again.
    axis_blender = basis @ Vector((axis["x"], axis["y"], axis["z"]))
    pivot_blender = basis @ Vector((pivot["x"], pivot["y"], pivot["z"]))
    return {"object": hinge, "id": joint_id, "axis": axis_blender,
            "pivot": pivot_blender,
            "lower": float(lower), "upper": float(upper)}


def apply_physical_hinge(binding, sample):
    if binding is None:
        return
    angles = sample.get("joint_angles", {})
    if set(angles) != {binding["id"]}:
        fail("sample does not contain exactly the selected hinge angle")
    angle = float(angles[binding["id"]])
    if not math.isfinite(angle) or angle < binding["lower"] or angle > binding["upper"]:
        fail("sampled hinge angle is outside its pinned range")
    move = Matrix.Translation(binding["pivot"])
    rotate = Matrix.Rotation(angle, 4, binding["axis"])
    binding["object"].matrix_basis = move @ rotate @ move.inverted()


def apply_motion_sample(device_root, camera_object, camera_spec, placement, device_candidate, sample, hinge=None):
    sampled = sample.get("device_root")
    if sampled is not None:
        sampled_root = trs({"translation_m": sampled["translation_m"], "rotation_xyzw": sampled["rotation_xyzw"], "scale": {"x": 1.0, "y": 1.0, "z": 1.0}})
        device_root.matrix_world = physical_device_placement_matrix(placement, sampled_root @ trs(placement["device_root_rest"]), trs(device_candidate["source_to_canonical"]))
    if camera_object is not None:
        camera_motion = sample.get("camera_transforms", {}).get(camera_spec["id"])
        frame = trs(camera_spec["frame"])
        if camera_motion is not None:
            frame = frame @ trs(camera_motion)
        camera_object.matrix_world = canonical_to_blender() @ frame
    apply_physical_hinge(hinge, sample)
    bpy.context.view_layer.update()


def validate_photographic_camera_envelope(scene_candidate, camera_spec, samples):
    """Reject camera motion that exceeds the reviewed photo reconstruction envelope."""
    photograph = scene_candidate.get("photographic_background")
    if not photograph:
        return
    max_translation = float(photograph.get("camera_envelope_max_translation_m", 0.0))
    max_rotation = float(photograph.get("camera_envelope_max_rotation_degrees", 0.0))
    reviewed = photograph.get("overscan_reviewed") is True
    if not math.isfinite(max_translation) or not math.isfinite(max_rotation) or max_translation < 0 or max_rotation < 0:
        fail("photographic camera envelope is invalid")
    if not reviewed and (max_translation > 0 or max_rotation > 0):
        fail("photographic camera motion requires reviewed overscan")
    for sample in samples:
        motion = sample.get("camera_transforms", {}).get(camera_spec["id"])
        if motion is None:
            continue
        translation = motion.get("translation_m", {})
        rotation = motion.get("rotation_xyzw", {})
        values = [float(translation.get(axis, 0.0)) for axis in ("x", "y", "z")]
        quaternion = [float(rotation.get(axis, 0.0)) for axis in ("x", "y", "z", "w")]
        if not all(math.isfinite(value) for value in values + quaternion):
            fail("photographic camera motion contains non-finite values")
        distance = math.sqrt(sum(value * value for value in values))
        qnorm = math.sqrt(sum(value * value for value in quaternion))
        if abs(qnorm - 1.0) > 1e-5:
            fail("photographic camera motion quaternion is not normalized")
        angle = math.degrees(2.0 * math.acos(min(1.0, abs(quaternion[3]))))
        if distance > max_translation + 1e-9 or angle > max_rotation + 1e-7:
            fail("sampled camera motion exceeds the reviewed photographic envelope")


def validate_photographic_render_mode(scene_candidate, camera_spec, samples):
    """Return the explicit photo composite mode and enforce its camera contract."""
    photograph = scene_candidate.get("photographic_background") or {}
    mode = photograph.get("render_mode", "")
    if mode not in ("", "SOURCE_CAMERA_PLATE_V1", "DEPTH_REPROJECTED_GEOMETRY_V1"):
        fail("photographic render mode is unsupported")
    if mode == "SOURCE_CAMERA_PLATE_V1":
        camera_id = camera_spec["id"]
        for sample in samples:
            transforms = sample.get("camera_transforms", {})
            transform = transforms.get(camera_id)
            if transform is None:
                continue
            translation = transform.get("translation_m", {})
            rotation = transform.get("rotation_xyzw", {})
            moved = any(abs(float(translation.get(axis, 0.0))) > 1e-9 for axis in ("x", "y", "z"))
            rotated = (abs(float(rotation.get("x", 0.0))) > 1e-9 or
                       abs(float(rotation.get("y", 0.0))) > 1e-9 or
                       abs(float(rotation.get("z", 0.0))) > 1e-9 or
                       abs(float(rotation.get("w", 1.0)) - 1.0) > 1e-9)
            if moved or rotated:
                fail("source-camera plate mode rejects camera motion")
    return mode


def source_camera_plate_path(root, manifest, scene_candidate):
    """Resolve the clean-plate only from its immutable staged scene texture."""
    photograph = scene_candidate.get("photographic_background") or {}
    textures = [item for item in scene_candidate.get("artifacts", [])
                if item.get("ref", {}).get("role") == "texture_image"]
    if len(textures) != 1:
        fail("source-camera plate mode requires exactly one staged scene texture")
    artifact = textures[0]
    reference = artifact.get("ref", {})
    route = photograph.get("route", "")
    if route == "PHOTOGRAPH_ROUTE_EMPTY_SCENE":
        if reference.get("sha256") != photograph.get("source_raster_sha256"):
            fail("source-camera plate differs from the reviewed source raster")
    elif route == "PHOTOGRAPH_ROUTE_DEVICE_REMOVAL":
        if not photograph.get("repair_provenance_sha256"):
            fail("device-removal source-camera plate lacks reviewed repair provenance")
    else:
        fail("source-camera plate route is unsupported")
    binding = next((item for item in manifest.get("assets", [])
                    if item.get("binding_id") == artifact.get("id") and item.get("role") == "scene_texture"), None)
    if binding is None or not binding.get("name"):
        fail("source-camera plate has no immutable staged scene texture binding")
    path = os.path.join(root, binding["name"])
    try:
        expected_size = int(reference.get("size_bytes", 0))
    except (TypeError, ValueError):
        expected_size = 0
    if not os.path.isfile(path) or expected_size <= 0 or os.path.getsize(path) != expected_size:
        fail("staged source-camera plate is missing or has an unexpected size")
    digest = hashlib.sha256()
    with open(path, "rb") as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b""):
            digest.update(chunk)
    if digest.hexdigest() != reference.get("sha256"):
        fail("staged source-camera plate differs from its immutable digest")
    return path


def validate_source_plate_aspect(source_width, source_height, output_width, output_height, tolerance=0.002):
    """Reject source-camera composites that would stretch the immutable plate."""
    dimensions = (source_width, source_height, output_width, output_height)
    if any(not isinstance(value, int) or value <= 0 for value in dimensions):
        fail("source-camera plate and output dimensions must be positive integers")
    if not math.isfinite(tolerance) or tolerance < 0:
        fail("source-camera aspect tolerance is invalid")
    source_aspect = source_width / source_height
    output_aspect = output_width / output_height
    if abs(source_aspect / output_aspect - 1.0) > tolerance:
        fail("source-camera plate aspect ratio differs from output; crop and re-review the source first")


def triangle_within_support_plane(points, normal, plane_point, tolerance):
    if len(points) != 3 or not math.isfinite(tolerance) or tolerance <= 0:
        return False
    return all(
        abs(sum(float(normal[axis]) * (float(point[axis]) - float(plane_point[axis])) for axis in range(3))) <= tolerance
        for point in points
    )


def support_plane_quad(normal, plane_point, device_points, margin):
    """Build a bounded, planar catcher around projected device bounds."""
    if len(normal) != 3 or len(plane_point) != 3 or len(device_points) < 4:
        raise ValueError("support plane or device bounds are incomplete")
    if not math.isfinite(margin) or margin < 0:
        raise ValueError("support catcher margin is invalid")
    n = tuple(float(value) for value in normal)
    plen = math.sqrt(sum(value * value for value in n))
    if not math.isfinite(plen) or abs(plen - 1.0) > 1e-4:
        raise ValueError("support plane normal is not normalized")
    p0 = tuple(float(value) for value in plane_point)
    axis = (1.0, 0.0, 0.0) if abs(n[0]) < 0.9 else (0.0, 0.0, 1.0)
    projection = sum(axis[i] * n[i] for i in range(3))
    u_axis = tuple(axis[i] - projection * n[i] for i in range(3))
    u_length = math.sqrt(sum(value * value for value in u_axis))
    u_axis = tuple(value / u_length for value in u_axis)
    v_axis = (
        n[1] * u_axis[2] - n[2] * u_axis[1],
        n[2] * u_axis[0] - n[0] * u_axis[2],
        n[0] * u_axis[1] - n[1] * u_axis[0],
    )
    projected = []
    signed_distances = []
    for point in device_points:
        delta = tuple(float(point[i]) - p0[i] for i in range(3))
        signed = sum(n[i] * delta[i] for i in range(3))
        signed_distances.append(signed)
        on_plane = tuple(float(point[i]) - signed * n[i] for i in range(3))
        relative = tuple(on_plane[i] - p0[i] for i in range(3))
        projected.append((sum(relative[i] * u_axis[i] for i in range(3)),
                          sum(relative[i] * v_axis[i] for i in range(3))))
    footprint = (
        min(value[0] for value in projected), min(value[1] for value in projected),
        max(value[0] for value in projected), max(value[1] for value in projected),
    )
    u_min, v_min, u_max, v_max = footprint
    u_min -= margin
    v_min -= margin
    u_max += margin
    v_max += margin
    corners = []
    for u, v in ((u_min, v_min), (u_max, v_min), (u_max, v_max), (u_min, v_max)):
        corners.append(tuple(p0[i] + u * u_axis[i] + v * v_axis[i] for i in range(3)))
    return {
        "corners": tuple(corners),
        "basis_u": u_axis,
        "basis_v": v_axis,
        "footprint": footprint,
        "signed_distance_min": min(signed_distances),
        "signed_distance_max": max(signed_distances),
    }


def rotate_by_quaternion(q, vector):
    x, y, z = float(q["x"]), float(q["y"]), float(q["z"])
    w = float(q["w"])
    vx, vy, vz = (float(value) for value in vector)
    return (
        (1 - 2 * (y*y + z*z))*vx + 2 * (x*y - z*w)*vy + 2 * (x*z + y*w)*vz,
        2 * (x*y + z*w)*vx + (1 - 2 * (x*x + z*z))*vy + 2 * (y*z - x*w)*vz,
        2 * (x*z - y*w)*vx + 2 * (y*z + x*w)*vy + (1 - 2 * (x*x + y*y))*vz,
    )


def opencv_plane_to_world(plane_normal_opencv, offset, camera_frame):
    # OpenCV +x right,+y down,+z forward to canonical +x right,+y up,+z
    # front, then through the immutable source camera frame.
    local_normal = (float(plane_normal_opencv[0]), -float(plane_normal_opencv[1]), -float(plane_normal_opencv[2]))
    local_point = tuple(-float(offset) * value for value in local_normal)
    q = camera_frame["rotation_xyzw"]
    t = camera_frame["translation_m"]
    normal_world = rotate_by_quaternion(q, local_normal)
    point_rotated = rotate_by_quaternion(q, local_point)
    point_world = tuple(point_rotated[i] + float(t[axis]) for i, axis in enumerate(("x", "y", "z")))
    return normal_world, point_world


def make_support_shadow_catchers(scene, scene_objects, device_objects, photograph, camera_frame):
    normal = photograph.get("support_plane_normal_opencv")
    offset = float(photograph.get("support_plane_offset_estimate_m", float("nan")))
    residual95 = float(photograph.get("support_plane_residual_p95_estimate_m", float("nan")))
    if not isinstance(normal, list) or len(normal) != 3 or not all(math.isfinite(float(value)) for value in normal):
        fail("photographic support-plane normal is invalid")
    if not math.isfinite(offset) or not math.isfinite(residual95) or residual95 < 0:
        fail("photographic support-plane offset or residual is invalid")
    n, point = opencv_plane_to_world(normal, offset, camera_frame)
    nlen = math.sqrt(sum(value * value for value in n))
    if abs(nlen - 1.0) > 1e-4:
        fail("photographic support-plane normal is not normalized")
    tolerance = min(0.03, max(0.008, residual95 * 3.0))
    blender_to_canonical = canonical_to_blender().inverted()
    device_points = []
    for device in device_objects:
        if device.type != "MESH" or device.data is None:
            continue
        for corner in device.bound_box:
            world = device.matrix_world @ Vector(corner)
            canonical = blender_to_canonical @ world
            device_points.append(tuple(float(value) for value in canonical))
    if not device_points:
        fail("photographic support catcher has no device geometry bounds")
    signed_distances = [sum(n[i] * (p[i] - point[i]) for i in range(3)) for p in device_points]
    if min(signed_distances) > tolerance or max(signed_distances) < -tolerance:
        fail("device bounds do not intersect the reviewed photographic support plane")
    footprint_margin = 0.18
    try:
        quad = support_plane_quad(n, point, device_points, footprint_margin)
    except ValueError as exc:
        fail(str(exc))

    # Confirm that the reconstructed source actually contains observed mesh
    # vertices near the device footprint on this plane before using a planar
    # catcher. The high-resolution background mesh remains the beauty pass.
    footprint_u_min, footprint_v_min, footprint_u_max, footprint_v_max = quad["footprint"]
    observed = []
    for source in scene_objects:
        if source.type != "MESH" or source.data is None:
            continue
        for vertex in source.data.vertices:
            world = source.matrix_world @ vertex.co
            canonical = blender_to_canonical @ world
            relative = tuple(float(canonical[i]) - point[i] for i in range(3))
            distance = sum(n[i] * relative[i] for i in range(3))
            if abs(distance) > tolerance:
                continue
            on_plane = tuple(float(canonical[i]) - distance * n[i] for i in range(3))
            relative = tuple(on_plane[i] - point[i] for i in range(3))
            u = sum(relative[i] * quad["basis_u"][i] for i in range(3))
            v = sum(relative[i] * quad["basis_v"][i] for i in range(3))
            if (footprint_u_min - 0.03 <= u <= footprint_u_max + 0.03
                    and footprint_v_min - 0.03 <= v <= footprint_v_max + 0.03):
                observed.append((u, v))
    if len(observed) < 3:
        fail("reconstructed scene has no observed support vertices near the device footprint")
    if (max(value[0] for value in observed) - min(value[0] for value in observed) < 0.02
            or max(value[1] for value in observed) - min(value[1] for value in observed) < 0.02):
        fail("observed support vertices do not span the device footprint")

    mesh = bpy.data.meshes.new("photographic-shadow-catcher-mesh")
    mesh.from_pydata(quad["corners"], [], [(0, 1, 2), (0, 2, 3)])
    mesh.update()
    catcher = bpy.data.objects.new("photographic-support-shadow-catcher", mesh)
    scene.collection.objects.link(catcher)
    catcher.matrix_world = canonical_to_blender()
    catcher.is_shadow_catcher = True
    print(
        "PHOTOGRAPHIC_SUPPORT_CATCHER triangles=2 footprint_margin_m=%.3f "
        "plane_tolerance_m=%.6f observed_support_vertices=%d device_signed_distance_m=[%.6f,%.6f]"
        % (footprint_margin, tolerance, len(observed), min(signed_distances), max(signed_distances)),
        flush=True,
    )
    return [catcher]


def build_photographic_compositor(scene, background_path):
    """Build the pinned Blender 5 standalone compositor graph for the baked plate."""
    compositor_tree = bpy.data.node_groups.new("photographic-device-composite", "CompositorNodeTree")
    scene.compositing_node_group = compositor_tree
    compositor_tree.interface.new_socket(name="Image", in_out="OUTPUT", socket_type="NodeSocketColor")
    nodes = compositor_tree.nodes
    nodes.clear()
    plate = bpy.data.images.load(background_path, check_existing=False)
    plate.colorspace_settings.name = "sRGB"
    validate_source_plate_aspect(
        int(plate.size[0]), int(plate.size[1]),
        int(scene.render.resolution_x), int(scene.render.resolution_y),
    )
    plate.scale(scene.render.resolution_x, scene.render.resolution_y)
    plate_node = nodes.new("CompositorNodeImage")
    plate_node.image = plate
    rendered = nodes.new("CompositorNodeRLayers")
    alpha_over = nodes.new("CompositorNodeAlphaOver")
    alpha_over.inputs["Factor"].default_value = 1.0
    nodes_output = nodes.new("NodeGroupOutput")
    compositor_tree.links.new(plate_node.outputs["Image"], alpha_over.inputs["Background"])
    compositor_tree.links.new(rendered.outputs["Image"], alpha_over.inputs["Foreground"])
    compositor_tree.links.new(alpha_over.outputs["Image"], nodes_output.inputs["Image"])
    return compositor_tree, plate


def render_photographic_composite(scene, root, scene_objects, device_objects, photograph, camera_frame, output_path, source_plate_path=None):
    """Render baked plate, then transparent device/shadow-catcher layer over it."""
    if not scene_objects or not device_objects:
        fail("photographic composite requires staged background and device geometry")
    background_path = source_plate_path or os.path.join(root, "photographic-background-pass.png")
    if source_plate_path is None and os.path.exists(background_path):
        os.remove(background_path)
    compositor_was_enabled = scene.render.use_compositing
    previous_compositor = scene.compositing_node_group
    compositor_tree = None
    scene.render.use_compositing = False
    scene.render.film_transparent = False
    scene.render.filepath = background_path
    device_visibility = [(obj, obj.hide_render) for obj in device_objects]
    catchers = []
    scene_visibility = [(obj, obj.hide_render) for obj in scene_objects]
    try:
        for obj, _ in device_visibility:
            if obj.type == "MESH":
                obj.hide_render = True
        if source_plate_path is None:
            bpy.ops.render.render(write_still=True)
            if not os.path.isfile(background_path) or os.path.getsize(background_path) <= 0:
                fail("photographic background pass produced no bounded image")
        for obj, hidden in device_visibility:
            obj.hide_render = hidden
        # The geometry beauty pass used to force dependency-graph evaluation
        # before the device bounds were tested. Source-camera plate mode skips
        # that pass, so refresh transforms explicitly before placing the catcher.
        bpy.context.view_layer.update()
        catchers = make_support_shadow_catchers(scene, scene_objects, device_objects, photograph, camera_frame)
        for obj, _ in scene_visibility:
            if obj.type == "MESH":
                obj.hide_render = True
        scene.render.film_transparent = True
        # Blender 5 stores compositor graphs as independent node groups;
        # Scene.use_nodes and Scene.node_tree are obsolete/removed APIs.
        compositor_tree, plate = build_photographic_compositor(scene, background_path)
        scene.render.use_compositing = True
        scene.render.filepath = output_path
        bpy.ops.render.render(write_still=True)
        bpy.data.images.remove(plate)
        if not os.path.isfile(output_path) or os.path.getsize(output_path) <= 0:
            fail("photographic device and contact-shadow composite produced no image")
    finally:
        for obj, hidden in device_visibility:
            obj.hide_render = hidden
        for obj, hidden in scene_visibility:
            obj.hide_render = hidden
        for catcher in catchers:
            mesh = catcher.data
            bpy.data.objects.remove(catcher, do_unlink=True)
            if mesh is not None and mesh.users == 0:
                bpy.data.meshes.remove(mesh)
        scene.render.film_transparent = False
        scene.render.use_compositing = compositor_was_enabled
        scene.compositing_node_group = previous_compositor
        scene.render.filepath = output_path
        if compositor_tree is not None:
            bpy.data.node_groups.remove(compositor_tree, do_unlink=True)
        if source_plate_path is None and os.path.exists(background_path):
            os.remove(background_path)


def write_frame_record(ordinal, data):
    fd = 3
    header = struct.pack(">IQ", ordinal, len(data))
    for chunk in (header, data):
        view = memoryview(chunk)
        while view:
            written = os.write(fd, view)
            if written <= 0:
                fail("motion frame pipe closed before the complete frame was sent")
            view = view[written:]


def read_input_frame_record(stream, expected_ordinal):
    header = read_exact(stream, 12)
    ordinal, size = struct.unpack(">IQ", header)
    if ordinal != expected_ordinal or size <= 0 or size > (64 << 20):
        fail("recording screen input is outside the bounded frame protocol")
    data = bytearray()
    while len(data) < size:
        chunk = read_exact(stream, min(1 << 20, size - len(data)))
        data.extend(chunk)
    if data[:8] != b"\x89PNG\r\n\x1a\n":
        fail("recording screen input is not PNG")
    return bytes(data)


def read_exact(stream, size):
    data = bytearray()
    while len(data) < size:
        chunk = stream.read(size - len(data))
        if not chunk:
            fail("recording screen input is truncated")
        data.extend(chunk)
    return bytes(data)


def write_recording_screen_frame(root, data):
    target = os.path.join(root, "physical-recording-screen.png")
    temporary = target + ".next"
    with open(temporary, "xb") as stream:
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temporary, target)


def physical_device_placement_matrix(placement, device_root, source_to_canonical):
    """Apply C·scene·pivot·motion·rest·source·C⁻¹ before imported C·N_glTF."""
    c = canonical_to_blender()
    pivot = placement["support_pivot_m"]
    return (c @ trs(placement["scene_support_world"])
            @ Matrix.Translation(Vector((pivot["x"], pivot["y"], pivot["z"])))
            @ device_root @ source_to_canonical @ c.inverted())


def main():
    if "--" not in sys.argv:
        fail("worker did not receive fixed input argument")
    args = sys.argv[sys.argv.index("--") + 1:]
    if len(args) != 1:
        fail("worker accepts exactly one fixed input path")
    root, manifest = read_manifest(args[0])
    initialize_empty_scene()
    output_path = os.path.join(root, manifest["output"]["output_name"])
    if os.path.exists(output_path):
        fail("reserved physical output already exists")
    device_candidate, scene_candidate = manifest["device"], manifest["scene"]
    device_objects_list, device_objects = import_glb(root, "device-normalized.glb", "device")
    scene_objects_list, scene_objects = import_glb(root, "scene-normalized.glb", "scene")
    scene = bpy.context.scene
    device_root = place_device(device_objects_list, device_candidate, manifest, device_objects)
    articulated = manifest["output"].get("articulation_profile") == "device-studio-single-revolute-hinge-v1"
    hinge = attach_physical_hinge(device_candidate, manifest["motion"], device_objects) if articulated else None
    for artifact in scene_candidate["artifacts"]:
        if artifact["ref"]["role"] != "geometry_glb":
            continue
        if artifact.get("source_to_canonical"):
            binding = next((a for a in manifest["assets"] if a["binding_id"] == artifact["id"]), None)
            if binding is None:
                fail("scene artifact has no immutable staged binding")
            # The scene GLB is imported once; the matching source transform is
            # applied to its top-level nodes, never to each semantic part.
            unit_scale = {"m": 1.0, "cm": 0.01, "mm": 0.001}.get(artifact.get("source_unit"))
            if unit_scale is None:
                fail("scene artifact source unit is unsupported")
            c = canonical_to_blender()
            source = c @ trs(artifact["source_to_canonical"]) @ Matrix.Scale(unit_scale, 4) @ c.inverted()
            wrap_roots("physical-scene-source-transform", scene_objects_list, source)
        break
    bind_scene_materials(root, manifest, scene_candidate, scene_objects)
    mapping = manifest.get("screen_geometry")
    if not mapping:
        fail("worker input lacks a Go-verified screen geometry binding")
    screen_image = bind_screen(root, manifest, mapping, device_objects)
    build_world(scene, scene_candidate["environment"])
    build_lights(scene, scene_candidate["light_frames"])
    camera_id = manifest["placement"]["selected_camera_id"]
    camera_spec = next((item for item in scene_candidate["camera_frames"] if item["id"] == camera_id), None)
    if camera_spec is None:
        fail("selected camera is missing from scene candidate")
    validate_photographic_camera_envelope(scene_candidate, camera_spec, manifest["output"]["samples"])
    photo_render_mode = validate_photographic_render_mode(scene_candidate, camera_spec, manifest["output"]["samples"])
    photo_plate_path = source_camera_plate_path(root, manifest, scene_candidate) if photo_render_mode == "SOURCE_CAMERA_PLATE_V1" else None
    camera_motion = manifest["output"]["samples"][0].get("camera_transforms", {}).get(camera_id)
    camera_object = setup_camera(scene, camera_spec, device_root, scene_candidate, scene_objects, camera_motion)
    configure_scene(scene, manifest, root)
    backend = manifest["profile"]["backend"]
    gpu_runtime = None
    if backend == "CPU":
        scene.cycles.device = "CPU"
    else:
        addon = bpy.context.preferences.addons.get("cycles")
        if addon is None:
            fail("Cycles backend preferences are unavailable")
        requested_uuid = os.environ.get("DEVICE_STUDIO_GPU_UUID", "").strip()
        if not requested_uuid:
            fail("GPU render requires a pinned physical GPU UUID")
        visible_uuids = [x.strip() for x in os.environ.get("CUDA_VISIBLE_DEVICES", "").split(",") if x.strip()]
        if visible_uuids != [requested_uuid]:
            fail("GPU visibility does not match the pinned physical UUID")
        prefs = addon.preferences
        prefs.compute_device_type = backend
        prefs.get_devices()
        usable = [item for item in prefs.devices if item.type == backend]
        if len(usable) != 1:
            fail("pinned Cycles backend must expose exactly one usable device")
        for item in prefs.devices:
            item.use = item in usable
        scene.cycles.device = "GPU"
        gpu_runtime = {
            "backend": backend,
            "uuid": requested_uuid,
            "selected_device": usable[0].name,
            "visible_devices": [item.name for item in prefs.devices if item.use],
            "cycles_device": scene.cycles.device,
            "render_completed": False,
        }
        persist_gpu_diagnostics(root, gpu_runtime)
    if manifest["output"]["kind"] == "motion_mp4":
        scratch_path = os.path.join(root, "physical-motion-frame.png")
        scene.render.filepath = scratch_path
        recording = manifest["screen"].get("recording") is not None
        input_stream = os.fdopen(4, "rb", buffering=0) if recording else None
        for sample in manifest["output"]["samples"]:
            if recording and sample["output_ordinal"] > 0:
                packet = read_input_frame_record(input_stream, sample["output_ordinal"])
                write_recording_screen_frame(root, packet)
                screen_image.reload()
            apply_motion_sample(device_root, camera_object, camera_spec, manifest["placement"], device_candidate, sample, hinge)
            if scene_candidate.get("photographic_background"):
                render_photographic_composite(scene, root, scene_objects_list, device_objects_list, scene_candidate["photographic_background"], camera_spec["frame"], scratch_path, photo_plate_path)
            else:
                bpy.ops.render.render(write_still=True)
            if not os.path.isfile(scratch_path) or os.path.getsize(scratch_path) <= 0:
                fail("Blender did not produce the bounded motion frame scratch image")
            with open(scratch_path, "rb") as frame_file:
                frame_bytes = frame_file.read(64 << 20)
                if frame_file.read(1):
                    fail("motion frame PNG exceeds its fixed byte bound")
            write_frame_record(sample["output_ordinal"], frame_bytes)
        os.remove(scratch_path)
    else:
        if articulated:
            apply_motion_sample(device_root, camera_object, camera_spec, manifest["placement"], device_candidate, manifest["output"]["samples"][0], hinge)
        if scene_candidate.get("photographic_background"):
            render_photographic_composite(scene, root, scene_objects_list, device_objects_list, scene_candidate["photographic_background"], camera_spec["frame"], output_path, photo_plate_path)
        else:
            bpy.ops.render.render(write_still=True)
        if not os.path.isfile(output_path) or os.path.getsize(output_path) <= 0:
            fail("Blender did not produce the fixed still output")
    if gpu_runtime is not None:
        gpu_runtime["render_completed"] = True
        persist_gpu_diagnostics(root, gpu_runtime)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        traceback.print_exc(file=sys.stderr)
        sys.exit(17)
