"""Fixed Device Studio physical still adapter; all decisions arrive in one validated manifest."""
import json
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
    valid_motion = manifest.get("version") == 2 and manifest.get("output", {}).get("kind") == "motion_mp4"
    valid_recording = (
        manifest.get("version") == 4 and manifest.get("output", {}).get("kind") == "motion_mp4"
        and manifest.get("output", {}).get("profile_id") == "device-studio-physical-recording-1080p-v1"
        and manifest.get("screen", {}).get("recording") is not None
    )
    output = manifest.get("output", {})
    articulated = (
        manifest.get("version") == 5
        and output.get("articulation_profile") == "device-studio-single-revolute-hinge-v1"
        and ((output.get("kind") == "still_png" and output.get("profile_id") in (
            "device-studio-physical-still-2560-v1", "device-studio-physical-preview-960x540-v1"))
            or (output.get("kind") == "motion_mp4" and output.get("profile_id") in (
                "device-studio-physical-motion-1080p-v1", "device-studio-physical-motion-1080p-v2",
                "device-studio-physical-recording-1080p-v1")))
    )
    if not (valid_still or valid_preview or valid_motion or valid_recording or articulated):
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
    camera_motion = manifest["output"]["samples"][0].get("camera_transforms", {}).get(camera_id)
    camera_object = setup_camera(scene, camera_spec, device_root, scene_candidate, scene_objects, camera_motion)
    configure_scene(scene, manifest, root)
    backend = manifest["profile"]["backend"]
    if backend == "CPU":
        scene.cycles.device = "CPU"
    else:
        addon = bpy.context.preferences.addons.get("cycles")
        if addon is None:
            fail("Cycles backend preferences are unavailable")
        prefs = addon.preferences
        prefs.compute_device_type = backend
        prefs.get_devices()
        usable = [item for item in prefs.devices if item.type == backend]
        if not usable:
            fail("pinned Cycles backend has no usable device")
        for item in prefs.devices:
            item.use = item.type == backend
        scene.cycles.device = "GPU"
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
        bpy.ops.render.render(write_still=True)
        if not os.path.isfile(output_path) or os.path.getsize(output_path) <= 0:
            fail("Blender did not produce the fixed still output")


if __name__ == "__main__":
    try:
        main()
    except Exception:
        traceback.print_exc(file=sys.stderr)
        sys.exit(17)
