import hashlib
import importlib.util
import json
import math
import os
import sys
import traceback

import bpy
from mathutils import Matrix, Quaternion, Vector


def fail(message):
    raise RuntimeError(message)


def vector_measurement(name, actual, expected, tolerance):
    actual_values = [float(value) for value in actual]
    expected_values = [float(value) for value in expected]
    delta = [a - e for a, e in zip(actual_values, expected_values)]
    return {"name": name, "actual": actual_values, "expected": expected_values,
            "component_error": delta, "distance": math.sqrt(sum(x * x for x in delta)),
            "tolerance": float(tolerance)}


def persist_measurements(root, name, measurements):
    if name not in ("physical-device-landmark-diagnostics.json", "physical-hinge-direct-diagnostics.json", "physical-hinge-imported-diagnostics.json"):
        fail("semantic diagnostic filename is outside its fixed allowlist")
    data = json.dumps({"version": 1, "measurements": measurements}, sort_keys=True, separators=(",", ":"))
    if len(data) > 8192:
        fail("semantic diagnostic evidence exceeds its fixed bound")
    path = os.path.join(root, name)
    if os.path.commonpath((root, path)) != root:
        fail("semantic diagnostic evidence escaped the attempt root")
    with open(path, "x", encoding="utf-8") as stream:
        stream.write(data)
        stream.write("\n")


def require_measurement(measurement, description):
    if not math.isfinite(measurement["distance"]) or measurement["distance"] > measurement["tolerance"]:
        fail("%s: actual=%s expected=%s component_error=%s distance=%.12g tolerance=%.12g" % (
            description, measurement["actual"], measurement["expected"], measurement["component_error"],
            measurement["distance"], measurement["tolerance"]))


def read_manifest(path):
    if not os.path.isabs(path) or os.path.basename(path) != "physical-semantic-probe-manifest.json":
        fail("semantic probe requires its fixed absolute manifest")
    with open(path, "rb") as stream:
        data = stream.read(65537)
    if len(data) > 65536:
        fail("semantic probe manifest exceeds its bound")
    value = json.loads(data)
    if value.get("version") != 1 or value.get("adapter_name") != "physical_blender.py":
        fail("semantic probe manifest version or adapter name is invalid")
    root = os.path.dirname(path)
    adapter_path = os.path.join(root, "physical_blender.py")
    with open(adapter_path, "rb") as stream:
        adapter_bytes = stream.read((1 << 20) + 1)
    if len(adapter_bytes) > 1 << 20 or hashlib.sha256(adapter_bytes).hexdigest() != value.get("adapter_sha256"):
        fail("staged adapter bytes differ from the manifest")
    glb_name = value.get("hinge_glb_name")
    if glb_name != "physical-hinge-probe.glb" or value.get("hinge_glb_sha256") != "36d6091d2aa782f1f06f7447a6078520c586bcc7c4f3a4f48f697cd9a2ffc512" or value.get("hinge_glb_size_bytes") != 832:
        fail("importer-backed hinge fixture identity is not the fixed reviewed value")
    glb_path = os.path.join(root, glb_name)
    if os.path.commonpath((root, glb_path)) != root:
        fail("hinge GLB escaped the attempt root")
    with open(glb_path, "rb") as stream:
        glb_bytes = stream.read(833)
    if len(glb_bytes) != value["hinge_glb_size_bytes"] or hashlib.sha256(glb_bytes).hexdigest() != value["hinge_glb_sha256"]:
        fail("staged importer-backed hinge GLB differs from the pinned bytes")
    return root, value, adapter_path


def import_adapter(path):
    spec = importlib.util.spec_from_file_location("device_studio_physical_adapter", path)
    if spec is None or spec.loader is None:
        fail("cannot load the pinned physical adapter")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def links_to(links, source, target):
    return any(link.from_socket == source and link.to_socket == target for link in links)


def check_device_landmark(adapter, root):
    def trs_dict(translation, quaternion, scale=(1, 1, 1)):
        return {"translation_m": dict(zip(("x", "y", "z"), translation)),
                "rotation_xyzw": dict(zip(("x", "y", "z", "w"), quaternion)),
                "scale": dict(zip(("x", "y", "z"), scale))}

    # Imported hierarchy: C·N_parent·N_child·p_mesh. All transforms are
    # deliberately noncommuting, and the expected final point is an
    # independent hand-computed oracle: (2, -8.4, 5.3) Blender meters.
    master = bpy.data.objects.new("probe-imported-master", None)
    child = bpy.data.objects.new("probe-imported-mesh", None)
    bpy.context.scene.collection.objects.link(master)
    bpy.context.scene.collection.objects.link(child)
    c = adapter.canonical_to_blender()
    n_parent = trs_dict((0, 0, 1), (0, 0, 0.7071067811865476, 0.7071067811865476))
    n_child = trs_dict((0, 2, 0), (0, 0, 0, 1))
    master.matrix_world = c @ adapter.trs(n_parent)
    child.parent = master
    child.matrix_parent_inverse = Matrix.Identity(4)
    child.matrix_basis = adapter.trs(n_child)

    device = {
        "source_to_canonical": trs_dict((0.1, 0.2, 0.3), (0, 0, 0, 1), (2, 2, 2)),
        "support_frames": [{"id": "device-support", "part_id": "body", "local_frame": trs_dict((0, 0, 0), (0, 0, 0, 1))}],
        "parts": [{"id": "body", "node_index": 1}],
    }
    placement = {
        "device_support_id": "device-support",
        "scene_support_id": "table-support",
        "selected_camera_id": "camera-main",
        "scene_support_world": trs_dict((1, 2, 3), (0, 0, 0.7071067811865476, 0.7071067811865476)),
        "support_pivot_m": {"x": 0.7, "y": 0.8, "z": 0.9},
        "device_root_rest": trs_dict((0, 0, 0), (0.7071067811865476, 0, 0, 0.7071067811865476)),
    }
    sampled_root = trs_dict((0.4, 0.5, 0.6), (0, 0.7071067811865476, 0, 0.7071067811865476))
    manifest = {"placement": placement, "output": {"samples": [{"device_root": sampled_root}]}}
    if child.parent is not master:
        fail("landmark fixture is not a parented imported hierarchy")
    adapter.place_device([master, child], device, manifest, {adapter.alias("device", 1): child})
    bpy.context.view_layer.update()
    actual = child.evaluated_get(bpy.context.evaluated_depsgraph_get()).matrix_world @ Vector((1, 0, 0, 1))
    expected = Vector((2.0, -8.4, 5.3, 1.0))
    measurement = vector_measurement("parented-device-world-landmark", actual, expected, 2e-5)
    persist_measurements(root, "physical-device-landmark-diagnostics.json", [measurement])
    require_measurement(measurement, "actual parented device placement differs from independent transform landmark")


def check_material_graph(adapter, root, manifest):
    texture_path = os.path.join(root, manifest["texture_name"])
    spec = {
        "id": "semantic-probe-material",
        "surface_profile": "PRINCIPLED_V1",
        "base_color_linear_rgba": [0.8, 0.7, 0.6, 0.9],
        "metallic": 0.0,
        "roughness": 0.8,
        "alpha_mode": "MASK",
        "alpha_cutoff": 0.5,
        "inputs": [
            {"socket": "base_color", "texture_binding_id": "probe", "color_space": "SRGB", "channel": "RGB", "sampler": "REPEAT", "coordinates": "UV_SET_AFFINE_V1", "uv_set": "TEXCOORD_0", "affine_uv": [1, 0, 0, 0, 1, 0]},
            {"socket": "roughness", "texture_binding_id": "probe", "color_space": "NON_COLOR", "channel": "RED", "sampler": "REPEAT", "coordinates": "UV_SET_AFFINE_V1", "uv_set": "TEXCOORD_0", "affine_uv": [1, 0, 0, 0, 1, 0]},
            {"socket": "normal", "texture_binding_id": "probe", "color_space": "NON_COLOR", "channel": "RGB", "sampler": "REPEAT", "coordinates": "UV_SET_AFFINE_V1", "uv_set": "TEXCOORD_0", "affine_uv": [1, 0, 0, 0, 1, 0], "normal_convention": "OPENGL_TANGENT_PLUS_Y_V1", "normal_strength": 0.75},
            {"socket": "bump_height", "texture_binding_id": "probe", "color_space": "NON_COLOR", "channel": "RED", "sampler": "REPEAT", "coordinates": "UV_SET_AFFINE_V1", "uv_set": "TEXCOORD_0", "affine_uv": [1, 0, 0, 0, 1, 0], "bump_strength": 0.25, "bump_distance_m": 0.001},
        ],
        "grade": {"recipe": "HUE_SATURATION_THEN_LINEAR_MIX_V1", "saturation": 1.1, "value": 1.0, "mix_factor": 0.2, "mix_linear_rgb": [0.2, 0.3, 0.4]},
    }
    mesh = bpy.data.meshes.new("semantic-probe-logical-uv0")
    mesh.uv_layers.new(name="UVMap")
    material = adapter.make_material(spec, {"probe": texture_path}, mesh.uv_layers)
    nodes, links = material.node_tree.nodes, material.node_tree.links
    principled = next((n for n in nodes if n.bl_idname == "ShaderNodeBsdfPrincipled"), None)
    normal = next((n for n in nodes if n.bl_idname == "ShaderNodeNormalMap"), None)
    bump = next((n for n in nodes if n.bl_idname == "ShaderNodeBump"), None)
    hue = next((n for n in nodes if n.bl_idname == "ShaderNodeHueSaturation"), None)
    mix = next((n for n in nodes if n.bl_idname == "ShaderNodeMixRGB"), None)
    separate = [n for n in nodes if n.bl_idname == "ShaderNodeSeparateColor"]
    threshold = next((n for n in nodes if n.bl_idname == "ShaderNodeMath" and n.operation == "LESS_THAN"), None)
    mix_shader = next((n for n in nodes if n.bl_idname == "ShaderNodeMixShader"), None)
    textures = [n for n in nodes if n.bl_idname == "ShaderNodeTexImage"]
    if not all((principled, normal, bump, hue, mix, threshold, mix_shader)) or len(separate) != 2 or len(textures) != 4:
        fail("supported material nodes were not constructed")
    if not links_to(links, separate[0].outputs["Red"], principled.inputs["Roughness"]) and not links_to(links, separate[1].outputs["Red"], principled.inputs["Roughness"]):
        fail("roughness did not use the declared red channel")
    if not links_to(links, normal.outputs["Normal"], bump.inputs["Normal"]) or not links_to(links, bump.outputs["Normal"], principled.inputs["Normal"]):
        fail("normal and bump nodes were not chained into the shader")
    if not links_to(links, hue.outputs["Color"], mix.inputs[1]) or not links_to(links, mix.outputs[0], principled.inputs["Base Color"]):
        fail("grade nodes are not wired in the declared order")
    bump_red = any(links_to(links, node.outputs["Red"], bump.inputs["Height"]) for node in separate)
    if not bump_red:
        fail("bump height did not use the declared red channel")
    if not links_to(links, threshold.outputs[0], mix_shader.inputs[0]):
        fail("MASK alpha cutoff is not applied to the surface")
    transparent = next((n for n in nodes if n.bl_idname == "ShaderNodeBsdfTransparent"), None)
    if (abs(threshold.inputs[1].default_value - 0.5) > 1e-8 or
            not any(link.from_node == principled for link in mix_shader.inputs[1].links) or
            transparent is None or not any(link.from_node == transparent for link in mix_shader.inputs[2].links)):
        fail("MASK equality boundary does not retain opaque alpha >= cutoff")
    spaces = [node.image.colorspace_settings.name for node in textures]
    if "sRGB" not in spaces or "Non-Color" not in spaces:
        fail("sRGB and Non-Color interpretations were not independently staged")
    if len({node.image.as_pointer() for node in textures}) != len(textures):
        fail("texture colorspace bindings share mutable Blender Image datablocks")
    uv_nodes = [node for node in nodes if node.bl_idname == "ShaderNodeUVMap"]
    if len(uv_nodes) != 4 or any(node.uv_map != "UVMap" for node in uv_nodes):
        fail("logical TEXCOORD_0 did not resolve to Blender's imported UVMap layer")
    output = next((n for n in nodes if n.bl_idname == "ShaderNodeOutputMaterial"), None)
    if output is None or not links_to(links, mix_shader.outputs[0], output.inputs["Surface"]):
        fail("MASK surface output is not connected")


def check_color_profile(adapter, root):
    profile = {
        "color_management": "device-studio-blender-standard-v1",
        "samples": 256,
        "seed": 1307,
        "adaptive_sampling": False,
        "denoise": False,
        "color_view": "Standard",
        "color_look": "None",
        "exposure": "0",
        "gamma": "1",
        "threads": 2,
    }
    output = {"width": 2560, "height": 1440, "output_name": "physical-output.png"}
    scene = bpy.context.scene
    adapter.configure_scene(scene, {"profile": profile, "output": output}, root)
    if scene.display_settings.display_device != "sRGB" or scene.view_settings.view_transform != "Standard" or scene.view_settings.look != "None":
        fail("pinned physical display and view transforms were not applied")
    if (scene.cycles.max_bounces != 8 or scene.cycles.caustics_reflective or
            scene.cycles.caustics_refractive):
        fail("approved Cycles bounce and caustics settings were not pinned")


def check_world(adapter):
    scene = bpy.context.scene
    environment = {"model": "CONSTANT_V1", "linear_rgb": [0.53, 0.56, 0.60], "strength": 0.2}
    adapter.build_world(scene, environment)
    background = scene.world.node_tree.nodes.get("Background")
    if background is None:
        fail("canonical CONSTANT_V1 environment did not create a world background")
    color = tuple(background.inputs["Color"].default_value)
    expected = (0.53, 0.56, 0.60, 1.0)
    if any(abs(actual - want) > 1e-7 for actual, want in zip(color, expected)):
        fail("canonical scene linear RGB did not reach the Blender world unchanged")
    if abs(background.inputs["Strength"].default_value - 0.2) > 1e-7:
        fail("canonical scene environment strength did not reach the Blender world")


def check_photographic_compositor(adapter, texture_path):
    scene = bpy.context.scene
    previous = scene.compositing_node_group
    tree = None
    plate = None
    try:
        tree, plate = adapter.build_photographic_compositor(scene, texture_path)
        if scene.compositing_node_group is not tree:
            fail("Blender 5 compositor node group was not assigned to the scene")
        if not hasattr(scene, "compositing_node_group") or hasattr(scene, "node_tree"):
            fail("scene compositor API does not match pinned Blender 5")
        node_types = [node.bl_idname for node in tree.nodes]
        expected_nodes = ("CompositorNodeImage", "CompositorNodeRLayers", "CompositorNodeAlphaOver", "NodeGroupOutput")
        if sorted(node_types) != sorted(expected_nodes):
            fail("photographic compositor nodes differ from the approved plate-over-device graph")
        if len(tree.links) != 3 or not tree.interface.items_tree:
            fail("photographic compositor graph is missing required links or output interface")
    finally:
        scene.compositing_node_group = previous
        if plate is not None:
            bpy.data.images.remove(plate)
        if tree is not None:
            bpy.data.node_groups.remove(tree, do_unlink=True)


def check_screen_uv_binding(adapter, root, manifest):
    mesh = bpy.data.meshes.new("semantic-probe-screen-mesh")
    mesh.from_pydata([(0, 0, 0), (1, 0, 0), (1, 0, 1), (0, 0, 1)], [], [(0, 1, 2, 3)])
    mesh.uv_layers.new(name="UVMap")
    original = bpy.data.materials.new(adapter.alias("device", 0, "MATERIAL"))
    mesh.materials.append(original)
    obj = bpy.data.objects.new(adapter.alias("device", 0), mesh)
    bpy.context.scene.collection.objects.link(obj)
    screen = {"uv_set": "TEXCOORD_0", "raster": {"name": manifest["texture_name"]}}
    adapter.bind_screen(root, {"screen": screen}, {"node_index": 0, "material_index": 0, "blender_uv_layer": "UVMap"}, {adapter.alias("device", 0): obj})
    bound = obj.data.materials[0]
    uv_nodes = [node for node in bound.node_tree.nodes if node.bl_idname == "ShaderNodeUVMap"]
    if len(uv_nodes) != 1 or uv_nodes[0].uv_map != "UVMap":
        fail("logical screen TEXCOORD_0 did not resolve to imported Blender UVMap")


def check_absolute_hinge_pose(adapter, root):
    body = bpy.data.objects.new("probe-hinge-body", None)
    lid = bpy.data.objects.new("probe-hinge-lid", None)
    screen = bpy.data.objects.new("probe-hinge-screen-landmark", None)
    for obj in (body, lid, screen):
        bpy.context.scene.collection.objects.link(obj)
    body.matrix_world = Matrix.Translation((1, 2, 3)) @ Matrix.Rotation(math.pi / 2, 4, "Z")
    lid.parent = body
    lid.matrix_parent_inverse = Matrix.Identity(4)
    lid.matrix_basis = Matrix.Translation((0, 0, 2)) @ Matrix.Rotation(math.pi / 2, 4, "Y")
    screen.parent = lid
    screen.matrix_parent_inverse = Matrix.Identity(4)
    screen.matrix_basis = Matrix.Translation((1, 0, 1))
    bpy.context.view_layer.update()

    device = {
        "definition": {"joints": [{
            "id": "lid-hinge", "parent_part": "base", "child_part": "lid",
            "axis": {"x": 1, "y": 0, "z": 0}, "pivot_m": {"x": 0, "y": 0, "z": -1},
            "lower_rad": 0.0, "upper_rad": math.pi / 2, "status": "candidate_only",
        }]},
        "parts": [{"id": "base", "node_index": 0}, {"id": "lid", "parent_id": "base", "node_index": 1}],
    }
    motion = {"tracks": [{"target": {"joint_id": "lid-hinge"}}]}
    objects = {adapter.alias("device", 0): body, adapter.alias("device", 1): lid}
    binding = adapter.attach_physical_hinge(device, motion, objects)
    pivot = Vector((0, 1, 0, 1))

    def pose(angle):
        adapter.apply_physical_hinge(binding, {"joint_angles": {"lid-hinge": angle}})
        bpy.context.view_layer.update()

    pose(0.0)
    matrix_a = lid.matrix_world.copy()
    first = screen.matrix_world.translation.copy()
    pose(math.pi / 2)
    second = screen.matrix_world.translation.copy()
    moved_pivot = binding["object"].matrix_world @ pivot
    rest_pivot = body.matrix_world @ pivot
    expected_pivot = Vector((0, 2, 3, 1))
    expected_b = Vector((1, 3, 2, 1))
    actual_b = screen.matrix_world @ Vector((0, 0, 0, 1))
    pivot_local = vector_measurement("hinge-parent-local-pivot-invariant", binding["object"].matrix_basis @ pivot, pivot, 1e-7)
    pivot_same = vector_measurement("hinge-world-pivot-vs-parent-world-pivot", moved_pivot, rest_pivot, 2e-6)
    # Blender stores object transforms as float32; at this 2–3 m world
    # coordinate, one ULP is about 2.38e-7 (as measured in hinge-import-5).
    # Keep the parent-relative invariant at 1e-7, while using the existing
    # 2e-6 independent world-landmark budget for this composed transform.
    pivot_expected = vector_measurement("hinge-pivot-independent-world-oracle", moved_pivot, expected_pivot, 2e-6)
    landmark = vector_measurement("hinge-child-screen-independent-world-oracle", actual_b, expected_b, 2e-6)
    pose(0.0)
    matrix_a_again = lid.matrix_world.copy()
    matrix_error = max(abs(matrix_a[r][c] - matrix_a_again[r][c]) for r in range(4) for c in range(4))
    moved_distance = (second - first).length
    persist_measurements(root, "physical-hinge-direct-diagnostics.json", [
        pivot_local, pivot_same, pivot_expected, landmark,
        {"name": "child-screen-motion-distance", "actual": [moved_distance], "expected": [0.5], "component_error": [moved_distance - 0.5], "distance": 0.0 if moved_distance >= 0.5 else 0.5 - moved_distance, "tolerance": 0.0},
        {"name": "hinge-a-b-a-matrix-max-component-error", "actual": [matrix_error], "expected": [0.0], "component_error": [matrix_error], "distance": matrix_error, "tolerance": 1e-7},
    ])
    require_measurement(pivot_local, "hinge matrix_basis did not preserve its parent-local pivot")
    require_measurement(pivot_same, "hinge world pivot differs from the parent world pivot")
    require_measurement(pivot_expected, "hinge pivot differs from the independent world oracle")
    require_measurement(landmark, "hinge did not move the child screen to the independent landmark")
    require_measurement({"actual": [moved_distance], "expected": [0.5], "component_error": [moved_distance - 0.5], "distance": 0.0 if moved_distance >= 0.5 else 0.5 - moved_distance, "tolerance": 0.0}, "hinge child screen did not move by the minimum distance")
    require_measurement({"actual": [matrix_error], "expected": [0.0], "component_error": [matrix_error], "distance": matrix_error, "tolerance": 1e-7}, "A→B→A hinge sampling accumulated transforms instead of restoring the absolute pose")


def check_imported_hinge(adapter, root, manifest):
    imported, objects = adapter.import_glb(root, manifest["hinge_glb_name"], "device")
    body = objects.get(adapter.alias("device", 0))
    lid = objects.get(adapter.alias("device", 1))
    if body is None or lid is None or body.type != "EMPTY" or lid.type != "MESH" or lid.parent is not body:
        fail("pinned GLB parented hinge fixture did not import into its expected node graph")
    if len(lid.data.vertices) != 3:
        fail("pinned GLB child mesh lost its three source landmarks during import")
    device = {"definition": {"joints": [{
        "id": "lid-hinge", "parent_part": "base", "child_part": "lid",
        "axis": {"x": 0, "y": 1, "z": 0}, "pivot_m": {"x": 0.1, "y": 0.2, "z": 0.3},
        "lower_rad": 0.0, "upper_rad": math.pi / 2, "status": "candidate_only",
    }]}, "parts": [{"id": "base", "node_index": 0}, {"id": "lid", "parent_id": "base", "node_index": 1}]}
    motion = {"tracks": [{"target": {"joint_id": "lid-hinge"}}]}
    binding = adapter.attach_physical_hinge(device, motion, {adapter.alias("device", 0): body, adapter.alias("device", 1): lid})
    pivot = Vector((0.1, -0.3, 0.2, 1))
    vertex = lid.data.vertices[0].co

    def pose(angle):
        adapter.apply_physical_hinge(binding, {"joint_angles": {"lid-hinge": angle}})
        bpy.context.view_layer.update()
        return lid.matrix_world @ vertex

    point_a = pose(0.0)
    point_b = pose(math.pi / 2)
    world_pivot = binding["object"].matrix_world @ pivot
    parent_pivot = body.matrix_world @ pivot
    expected_a = Vector((1.0, -4.0, 3.0))
    expected_b = Vector((1.0, -2.4, 2.8))
    expected_pivot = Vector((0.8, -3.3, 2.1, 1.0))
    point_a_again = pose(0.0)
    axis_measure = vector_measurement("imported-hinge-axis-canonical-to-blender", binding["axis"], Vector((0, 0, 1)), 1e-7)
    local_pivot_measure = vector_measurement("imported-hinge-pivot-canonical-to-blender", binding["pivot"], Vector((0.1, -0.3, 0.2)), 1e-7)
    point_a_measure = vector_measurement("imported-glb-rest-landmark", point_a, expected_a, 2e-6)
    point_b_measure = vector_measurement("imported-glb-hinge-landmark", point_b, expected_b, 2e-6)
    pivot_local_measure = vector_measurement("imported-hinge-parent-local-pivot-invariant", binding["object"].matrix_basis @ pivot, pivot, 1e-7)
    pivot_parent_measure = vector_measurement("imported-hinge-world-pivot-vs-parent-world-pivot", world_pivot, parent_pivot, 2e-6)
    pivot_oracle_measure = vector_measurement("imported-hinge-pivot-independent-world-oracle", world_pivot, expected_pivot, 2e-6)
    point_a_repeat_measure = vector_measurement("imported-glb-a-b-a-landmark", point_a_again, point_a, 1e-7)
    persist_measurements(root, "physical-hinge-imported-diagnostics.json", [axis_measure, local_pivot_measure, point_a_measure, point_b_measure, pivot_local_measure, pivot_parent_measure, pivot_oracle_measure, point_a_repeat_measure])
    for measurement, description in (
        (axis_measure, "glTF hinge axis was not basis-converted exactly once"),
        (local_pivot_measure, "glTF hinge pivot was not basis-converted exactly once"),
        (point_a_measure, "imported GLB rest point differs from independent matrix oracle"),
        (point_b_measure, "imported GLB hinge point differs from independent matrix oracle"),
        (pivot_local_measure, "imported hinge matrix_basis did not preserve its parent-local pivot"),
        (pivot_parent_measure, "imported hinge world pivot differs from parent world pivot"),
        (pivot_oracle_measure, "imported hinge pivot differs from independent world oracle"),
        (point_a_repeat_measure, "imported hinge A→B→A pose accumulated transform state"),
    ):
        require_measurement(measurement, description)


def main():
    if "--" not in sys.argv or len(sys.argv[sys.argv.index("--") + 1:]) != 1:
        fail("semantic probe requires exactly one manifest argument")
    root, manifest, adapter_path = read_manifest(sys.argv[sys.argv.index("--") + 1])
    adapter = import_adapter(adapter_path)
    adapter.initialize_empty_scene()
    texture = os.path.join(root, manifest["texture_name"])
    if os.path.basename(texture) != manifest["texture_name"] or not os.path.isfile(texture):
        fail("probe texture is missing")
    check_device_landmark(adapter, root)
    check_absolute_hinge_pose(adapter, root)
    check_imported_hinge(adapter, root, manifest)
    check_material_graph(adapter, root, manifest)
    check_color_profile(adapter, root)
    check_world(adapter)
    check_photographic_compositor(adapter, texture)
    check_screen_uv_binding(adapter, root, manifest)
    receipt = {"version": 1, "adapter_sha256": manifest["adapter_sha256"], "status": "verified", "checks": ["empty-factory-scene", "actual-parented-device-landmark", "absolute-hinge-pose-and-child-screen-landmark", "imported-glb-hinge-basis-and-child-landmark", "affine-uv-node-construction", "red-channel-roughness-and-bump", "normal-bump-chain", "grade-order", "mask-cutoff", "pinned-color-management", "canonical-constant-world", "blender-5-photographic-compositor-node-group", "logical-uv0-to-blender-uvmap", "pinned-cycles-look"]}
    with open(os.path.join(root, "physical-semantic-probe-receipt.json"), "x", encoding="utf-8") as stream:
        json.dump(receipt, stream, sort_keys=True, separators=(",", ":"))
        stream.write("\n")


if __name__ == "__main__":
    try:
        main()
    except Exception:
        traceback.print_exc(file=sys.stderr)
        sys.exit(17)
