"""Fixed, evidence-only laptop role split and neutral-screen GLB derivation."""
import hashlib
import itertools
import json
import math
import os
import struct
import traceback
from pathlib import Path

import bmesh
import bpy
from mathutils import Matrix, Vector
from mathutils.bvhtree import BVHTree

ROOT = Path(__file__).resolve().parent
MANIFEST = ROOT / "laptop-starter-manifest.json"
INPUTS = ROOT / "inputs"
OUTPUTS = ROOT / "outputs"
SOURCE_NAME = "normalized-macbook.glb"
INVENTORY_NAME = "component-candidates.json"
SCREEN_NODE = "Object_123"
SCREEN_MATERIAL_INDEX = 14
SCREEN_IMAGE_INDEX = 14
ROLE_COLORS = {
    "base": (0.25, 0.38, 0.52, 1.0),
    "lid": (0.76, 0.40, 0.19, 1.0),
    "hinge_review": (0.92, 0.72, 0.12, 1.0),
}
EXPECTED_LID_COMPONENT_OVERRIDES = [
    {"source_node_index": 0, "source_object": "Object_10", "source_component": 310,
     "source_candidate_role": "hinge-review", "candidate_role": "lid", "triangle_count": 184},
    {"source_node_index": 0, "source_object": "Object_10", "source_component": 313,
     "source_candidate_role": "hinge-review", "candidate_role": "lid", "triangle_count": 148},
    {"source_node_index": 2, "source_object": "Object_20", "source_component": 92,
     "source_candidate_role": "hinge-review", "candidate_role": "lid", "triangle_count": 456},
]
GLB_TO_BLENDER = Matrix.Rotation(math.pi / 2, 4, "X")


def require(ok, message):
    if not ok:
        raise RuntimeError("laptop starter derivation: " + message)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def persist_diagnostic(path, value):
    with path.open("w", encoding="utf-8") as stream:
        json.dump(value, stream, sort_keys=True)
        stream.write("\n")
        stream.flush()
        os.fsync(stream.fileno())


def read_regular(path, max_bytes):
    require(path.is_file() and not path.is_symlink(), "missing or unsafe fixed input " + str(path))
    require(0 < path.stat().st_size <= max_bytes, "fixed input size outside bound")
    data = path.read_bytes()
    require(len(data) == path.stat().st_size, "input changed while reading")
    return data


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        require(key not in result, "duplicate JSON key")
        result[key] = value
    return result


def parse_glb(data):
    require(len(data) >= 20, "short source GLB")
    magic, version, declared = struct.unpack_from("<4sII", data)
    require(magic == b"glTF" and version == 2 and declared == len(data), "bad GLB header")
    offset, chunks = 12, []
    while offset < len(data):
        require(offset + 8 <= len(data), "truncated chunk header")
        size, kind = struct.unpack_from("<I4s", data, offset)
        offset += 8
        require(size <= len(data) - offset, "chunk exceeds GLB")
        chunks.append((kind, data[offset:offset + size]))
        offset += size
    require(offset == len(data) and len(chunks) == 2, "unexpected source chunk set")
    require(chunks[0][0] == b"JSON" and chunks[1][0] == b"BIN\0", "source GLB needs JSON and BIN only")
    doc = json.loads(chunks[0][1].rstrip(b" \t\r\n\0"), object_pairs_hook=unique_object)
    require(not any(item.get("uri") for item in doc.get("buffers", [])), "external source buffer is forbidden")
    require(not any(item.get("uri") for item in doc.get("images", [])), "external source image is forbidden")
    return doc, chunks[1][1]


def extract_image_bytes(doc, binary, index):
    images = doc.get("images", [])
    require(0 <= index < len(images), "screen image index is absent")
    view_index = images[index].get("bufferView")
    require(isinstance(view_index, int) and 0 <= view_index < len(doc["bufferViews"]), "screen image must be embedded")
    view = doc["bufferViews"][view_index]
    start, size = view.get("byteOffset", 0), view.get("byteLength", 0)
    require(size > 0 and start >= 0 and start + size <= len(binary), "image buffer range is invalid")
    return binary[start:start + size]


def read_inputs():
    raw_manifest = read_regular(MANIFEST, 128 * 1024)
    cfg = json.loads(raw_manifest.decode("utf-8"), object_pairs_hook=unique_object)
    require(cfg["schema_version"] == 1 and cfg["profile_id"] == "device-studio-laptop-starter-derive-v9", "wrong fixed profile")
    require(list(bpy.app.version[:3]) == cfg["blender_version"], "Blender version differs from fixed manifest")
    require(bpy.app.version_string == cfg["blender_version_string"], "Blender build differs from fixed manifest")
    require(cfg["backend"] == "CPU" and cfg["render"]["engine"] == "CYCLES", "only pinned Cycles CPU is supported")
    require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed outputs must not already exist")
    source = read_regular(INPUTS / SOURCE_NAME, 8 * 1024 * 1024)
    inventory_bytes = read_regular(INPUTS / INVENTORY_NAME, 2 * 1024 * 1024)
    require(len(source) == cfg["source"]["size_bytes"] and digest(source) == cfg["source"]["sha256"], "source identity mismatch")
    require(len(inventory_bytes) == cfg["component_inventory"]["size_bytes"] and digest(inventory_bytes) == cfg["component_inventory"]["sha256"], "component inventory identity mismatch")
    document, binary = parse_glb(source)
    image_records = cfg["source"]["embedded_images"]
    require(len(document.get("images", [])) == len(image_records) == 18, "embedded image census changed")
    for index, record in enumerate(image_records):
        image_data = extract_image_bytes(document, binary, index)
        require(len(record) == 5 and len(image_data) == record[1] and digest(image_data) == record[2], f"embedded image {index} differs")
    screen = cfg["source"]["screen_binding"]
    node = document["nodes"][screen["node_index"]]
    primitive = document["meshes"][node["mesh"]]["primitives"][0]
    material_index = primitive.get("material")
    material = document["materials"][material_index]
    texture_index = material.get("emissiveTexture", {}).get("index")
    require(node.get("name") == SCREEN_NODE and node.get("mesh") == screen["mesh_index"], "screen node/mesh binding changed")
    require(material_index == SCREEN_MATERIAL_INDEX and material.get("name") == screen["material_name"], "screen material binding changed")
    require(texture_index == screen["texture_index"] and document["textures"][texture_index].get("source") == SCREEN_IMAGE_INDEX, "screen emissive image mapping changed")
    image = document["images"][SCREEN_IMAGE_INDEX]
    require(image.get("mimeType") == "image/png", "screen raster type changed")
    return cfg, digest(raw_manifest), source, document, json.loads(inventory_bytes), binary


def source_mapping(document, records, cfg):
    nodes = document["nodes"]
    mesh_nodes = [(index, node) for index, node in enumerate(nodes) if "mesh" in node]
    require(len(mesh_nodes) == 19, "source mesh-node count changed")
    expected = {(row[0], row[3]) for row in cfg["mesh_node_map"]}
    mapping = {}
    actual = []
    for node_index, node in mesh_nodes:
        name = node.get("name")
        require(name and (node_index, 0) in expected, "source node is not pinned")
        require(name not in mapping, "duplicate imported node names are unsupported")
        mesh_index = node["mesh"]
        primitives = document["meshes"][mesh_index]["primitives"]
        require(len(primitives) == 1 and primitives[0].get("mode", 4) == 4, "source must use one triangle primitive per mesh")
        count = document["accessors"][primitives[0]["indices"]]["count"]
        require(count % 3 == 0, "source index count is not triangular")
        actual.append([node_index, name, mesh_index, 0, count // 3])
        mapping[name] = (node_index, mesh_index)
    require(actual == cfg["mesh_node_map"], "source node/mesh/primitive ordering differs")
    lookup = {(row["name"], row["primitive"], row["component"]): row for row in records}
    require(len(lookup) == len(records) == 587, "component inventory keys are duplicate or census differs")
    return mapping, lookup


def connected_components(obj):
    points, key_to_id, welded = [], {}, []
    for vertex in obj.data.vertices:
        point = obj.matrix_world @ vertex.co
        key = (round(float(point.x), 7), round(float(point.z), 7), round(float(-point.y), 7))
        if key not in key_to_id:
            key_to_id[key] = len(key_to_id)
        welded.append(key_to_id[key])
    parent = list(range(len(key_to_id)))

    def find(value):
        while parent[value] != value:
            parent[value] = parent[parent[value]]
            value = parent[value]
        return value

    for polygon in obj.data.polygons:
        require(len(polygon.vertices) == 3, "imported mesh has non-triangle face")
        roots = [find(welded[index]) for index in polygon.vertices]
        for root in roots[1:]:
            parent[find(root)] = find(roots[0])
    groups = {}
    for polygon in obj.data.polygons:
        root = find(welded[polygon.vertices[0]])
        groups.setdefault(root, []).append(polygon.index)
    result, face_components = {}, [-1] * len(obj.data.polygons)
    for ordinal, indices in enumerate(sorted(groups.values(), key=lambda values: values[0])):
        for face_index in indices:
            face_components[face_index] = ordinal
        result[ordinal] = {"indices": indices, "first": indices[0], "count": len(indices),
                           "sha256": digest(struct.pack("<" + "I" * len(indices), *indices))}
    require(all(component >= 0 for component in face_components), "triangle lacks a connected component")
    return result, face_components


def validate_import(document, lookup, mapping, cfg):
    overrides = cfg["role_assignment"].get("lid_component_overrides")
    require(overrides == EXPECTED_LID_COMPONENT_OVERRIDES, "exact source-backed lid component overrides differ")
    override_map = {(row["source_object"], row["source_component"]): row
                    for row in EXPECTED_LID_COMPONENT_OVERRIDES}
    objects = {}
    for obj in bpy.context.scene.objects:
        if obj.type == "MESH":
            require(obj.name not in objects, "ambiguous Blender mesh name")
            objects[obj.name] = obj
    require(set(objects) == set(mapping), "Blender imported mesh set changed")
    roles, total_triangles, total_components = {}, 0, 0
    for name, (node_index, mesh_index) in mapping.items():
        obj = objects[name]
        primitive = document["meshes"][mesh_index]["primitives"][0]
        count = document["accessors"][primitive["indices"]]["count"] // 3
        require(len(obj.data.polygons) == count, "imported primitive triangle count changed")
        face_roles = [None] * count
        component_summaries, face_components = connected_components(obj)
        for component, summary in component_summaries.items():
            row = lookup.get((name, 0, component))
            require(row is not None and row["node"] == node_index, "component identity missing")
            require(row["first_triangle"] == summary["first"] and row["triangles"] == summary["count"] and row["triangle_indices_sha256"] == summary["sha256"], "component triangle signature changed")
            override = override_map.get((name, component))
            if override is not None:
                require(row["candidate"] == override["source_candidate_role"] and
                        row["node"] == override["source_node_index"] and
                        row["triangles"] == override["triangle_count"],
                        "source role or triangle count changed for exact lid override")
                role = override["candidate_role"]
            else:
                role = row["candidate"]
            require(role in ("base", "lid", "hinge-review"), "unknown candidate role")
            for face_index in summary["indices"]:
                require(face_roles[face_index] is None, "source triangle assigned twice")
                face_roles[face_index] = role
            total_components += 1
        require(all(role is not None for role in face_roles), "source triangle omitted")
        roles[name] = (obj, face_roles, face_components, node_index)
        total_triangles += count
    require(total_triangles == 112502 and total_components == 587, "full source census differs")
    return roles


def neutralize_screen(cfg):
    baseline = cfg["source"]["embedded_images"][SCREEN_IMAGE_INDEX]
    matches = [image for image in bpy.data.images if image.packed_file and digest(bytes(image.packed_file.data)) == baseline[2]]
    require(len(matches) == 1, "screen texture image is not uniquely identifiable by its pinned bytes")
    image = matches[0]
    require(tuple(image.size) == tuple(baseline[3:5]), "screen image dimensions changed")
    rgba = cfg["neutral_screen_linear_rgba"]
    require(len(rgba) == 4 and rgba[0] == rgba[1] == rgba[2] and rgba[3] == 1.0 and 0 < rgba[0] < 1, "neutral screen color must be opaque gray")
    width, height = map(int, image.size)
    image.pixels.foreach_set(tuple(rgba) * (width * height))
    image.update()
    image.pack()
    return image, {"source_sha256": baseline[2], "source_size_bytes": baseline[1], "dimensions_pixels": [width, height], "replacement_linear_rgba": rgba}


def make_fragment(source, roles, component_ids, node_index, role, collection):
    clone = source.copy()
    clone.data = source.data.copy()
    clone.name = source.name + "__" + role
    clone.data.name = source.data.name + "__" + role
    collection.objects.link(clone)
    bm = bmesh.new()
    bm.from_mesh(clone.data)
    bm.faces.ensure_lookup_table()
    bm.faces.index_update()
    require(len(bm.faces) == len(roles), "BMesh source face ordering changed")
    node_layer = bm.faces.layers.int.new("source_node_index")
    component_layer = bm.faces.layers.int.new("source_component_id")
    source_triangle_layer = bm.faces.layers.int.new("source_triangle_index")
    for face in bm.faces:
        face[node_layer] = node_index
        face[component_layer] = component_ids[face.index]
        face[source_triangle_layer] = face.index
    bmesh.ops.delete(bm, geom=[face for face in bm.faces if roles[face.index] != role], context="FACES")
    bmesh.ops.delete(bm, geom=[edge for edge in bm.edges if not edge.link_faces], context="EDGES")
    bmesh.ops.delete(bm, geom=[vert for vert in bm.verts if not vert.link_edges], context="VERTS")
    bm.to_mesh(clone.data)
    bm.free()
    clone.data.update()
    require(len(clone.data.polygons) > 0, "empty role fragment should not be emitted")
    clone["candidate_role"] = role
    clone["source_node"] = source.name
    return clone


def preserve_parent_world(obj, parent):
    matrix = obj.matrix_world.copy()
    obj.parent = parent
    obj.matrix_parent_inverse = parent.matrix_world.inverted()
    obj.matrix_world = matrix


def create_role_assembly(roles, axis):
    root = bpy.data.objects.new("LaptopStarterRoot", None)
    root.empty_display_type = "PLAIN_AXES"
    bpy.context.scene.collection.objects.link(root)
    base_root = bpy.data.objects.new("LaptopBaseCandidate", None)
    base_root.empty_display_type = "PLAIN_AXES"
    bpy.context.scene.collection.objects.link(base_root)
    base_root.parent = root
    review_root = bpy.data.objects.new("LaptopHingeReview", None)
    review_root.empty_display_type = "PLAIN_AXES"
    bpy.context.scene.collection.objects.link(review_root)
    review_root.parent = root
    pivot = bpy.data.objects.new("LaptopLidPivotCandidate", None)
    pivot.empty_display_type = "CIRCLE"
    bpy.context.scene.collection.objects.link(pivot)
    pivot.parent = root
    pivot.location = axis
    fragments = {"base": [], "lid": [], "hinge-review": []}
    source_role_inventory = {}
    for name in sorted(roles):
        source, face_roles, component_ids, node_index = roles[name]
        source_role_inventory[name] = {role: face_roles.count(role) for role in fragments}
        for role in fragments:
            if role not in face_roles:
                continue
            fragment = make_fragment(source, face_roles, component_ids, node_index, role, bpy.context.scene.collection)
            fragments[role].append(fragment)
    require(all(fragments[role] for role in fragments), "candidate role unexpectedly empty")
    screens = [obj for obj in fragments["lid"] if obj.get("source_node") == SCREEN_NODE]
    require(len(screens) == 1, "screen surface fragment is not uniquely retained")
    screen = screens[0]
    screen.name = "LaptopScreenSurface"
    screen.data.name = "LaptopScreenSurfaceMesh"
    fragments["lid"].remove(screen)
    groups = (("base", "LaptopBaseGeometry", base_root),
              ("lid", "LaptopLidGeometry", pivot),
              ("hinge-review", "LaptopHingeReviewGeometry", review_root))
    for role, name, parent in groups:
        role_fragments = fragments[role]
        if role == "hinge-review":
            trims = [obj for obj in role_fragments
                     if obj.get("source_node") == "Object_42" and
                     set(obj.data.attributes["source_component_id"].data[index].value for index in range(len(obj.data.polygons))) == {12}]
            require(len(trims) == 1 and len(trims[0].data.polygons) == 584,
                    "pinned Object_42 component 12 is not a unique intact trim candidate")
            trim = trims[0]
            trim.name = "LaptopHingeTrimCandidate"
            trim.data.name = "LaptopHingeTrimCandidateMesh"
            preserve_parent_world(trim, parent)
            role_fragments = [obj for obj in role_fragments if obj is not trim]
            joined = join_fragments(role_fragments, name)
            preserve_parent_world(joined, parent)
            fragments[role] = [joined, trim]
        else:
            joined = join_fragments(role_fragments, name)
            preserve_parent_world(joined, parent)
            fragments[role] = [joined]
    preserve_parent_world(screen, pivot)
    fragments["lid"].append(screen)
    source_role_inventory[SCREEN_NODE]["screen_surface"] = "separate lid child for display mapping"
    all_objects = [obj for objects in fragments.values() for obj in objects]
    return root, base_root, pivot, review_root, fragments, all_objects, source_role_inventory


def join_fragments(fragments, name):
    require(fragments, "cannot join an empty semantic role")
    bpy.ops.object.select_all(action="DESELECT")
    for obj in fragments:
        obj.select_set(True)
    bpy.context.view_layer.objects.active = fragments[0]
    if len(fragments) > 1:
        require(bpy.ops.object.join() == {"FINISHED"}, "failed to join a semantic role")
    joined = bpy.context.view_layer.objects.active
    require(joined is not None and joined.type == "MESH", "joined role is not a mesh")
    joined.name = name
    joined.data.name = name + "Mesh"
    joined["candidate_role"] = name
    return joined


def role_triangle_counts(fragments):
    result = {role: sum(len(obj.data.polygons) for obj in objects) for role, objects in fragments.items()}
    require(result == {"base": 99866, "lid": 8470, "hinge-review": 4166}, "role triangle totals differ from pinned inventory and exact overrides")
    return result


def align_fixed_hinge_trim(fragments, source_axis, fitted_axis, cfg):
    trim_spec = cfg["rig"]["fixed_trim_alignment"]
    trims = [obj for obj in fragments["hinge-review"] if obj.name == "LaptopHingeTrimCandidate"]
    require(len(trims) == 1, "authored fixed hinge trim candidate is missing")
    delta = fitted_axis - source_axis
    require(trim_spec == {"source_object": "Object_42", "source_component": 12,
                          "transform": "fitted_axis_delta", "parent": "LaptopHingeReview",
                          "rotates_with_lid": False}, "fixed trim alignment profile differs")
    trim = trims[0]
    trim.matrix_world = Matrix.Translation(delta) @ trim.matrix_world
    bpy.context.view_layer.update()
    return {
        "status": "authored_fixed_base_trim_alignment_hypothesis",
        "source_object": "Object_42", "source_component": 12, "triangle_count": len(trim.data.polygons),
        "parent": "LaptopHingeReview", "rotates_with_lid": False,
        "translation_world_m": [float(value) for value in delta],
        "translation_matches_fitted_axis_delta": True,
        "claim": "visual alignment experiment only; component ownership and hinge function remain unverified",
    }


def source_open_override_geometry(roles, fragments, cfg):
    """Verify reparented source triangles retain their world-space geometry at zero lid rotation."""
    bpy.context.view_layer.update()
    require(cfg["role_assignment"].get("lid_component_overrides") == EXPECTED_LID_COMPONENT_OVERRIDES,
            "source-open invariant does not use the exact pinned override list")

    def triangle_signatures(obj, polygons):
        signatures = []
        for polygon in polygons:
            points = [obj.matrix_world @ obj.data.vertices[index].co for index in polygon.vertices]
            signatures.append(tuple(sorted(tuple(round(float(value), 7) for value in point) for point in points)))
        return signatures

    def triangle_world_points(obj, polygon):
        return sorted(tuple(float(value) for value in obj.matrix_world @ obj.data.vertices[index].co)
                      for index in polygon.vertices)

    checks = []
    tolerance_m = float(cfg["role_assignment"]["source_open_geometry_tolerance_m"])
    require(tolerance_m == 1e-7, "source-open geometry tolerance differs from the pinned source precision")
    for row in EXPECTED_LID_COMPONENT_OVERRIDES:
        source_obj, face_roles, face_components, source_node = roles[row["source_object"]]
        require(source_node == row["source_node_index"], "source-open override node identity changed")
        source_faces = [polygon for polygon in source_obj.data.polygons
                        if face_components[polygon.index] == row["source_component"]]
        require(len(source_faces) == row["triangle_count"] and
                all(face_roles[polygon.index] == "lid" for polygon in source_faces),
                "source-open override role or triangle count changed")
        derived_by_object = {}
        for obj in fragments["lid"]:
            node_attr = obj.data.attributes.get("source_node_index")
            component_attr = obj.data.attributes.get("source_component_id")
            triangle_attr = obj.data.attributes.get("source_triangle_index")
            if node_attr is None or component_attr is None or triangle_attr is None:
                continue
            faces = [polygon for polygon in obj.data.polygons
                     if int(node_attr.data[polygon.index].value) == row["source_node_index"] and
                     int(component_attr.data[polygon.index].value) == row["source_component"]]
            if faces:
                derived_by_object[obj] = faces
        require(sum(len(faces) for faces in derived_by_object.values()) == row["triangle_count"],
                "derived source-open override triangle count changed")
        source_by_triangle = {polygon.index: triangle_world_points(source_obj, polygon) for polygon in source_faces}
        derived_by_triangle = {}
        for obj, faces in derived_by_object.items():
            triangle_attr = obj.data.attributes["source_triangle_index"].data
            derived_by_triangle.update({int(triangle_attr[polygon.index].value): triangle_world_points(obj, polygon)
                                        for polygon in faces})
        require(set(source_by_triangle) == set(derived_by_triangle), "source triangle identity was lost during role parenting")
        distances = []
        for triangle_index, source_signature in source_by_triangle.items():
            derived_signature = derived_by_triangle[triangle_index]
            pairing = min(itertools.permutations(derived_signature), key=lambda candidate: sum(
                sum((source_signature[vertex][axis] - candidate[vertex][axis]) ** 2 for axis in range(3))
                for vertex in range(3)))
            distances.extend(math.sqrt(sum((source_signature[vertex][axis] - pairing[vertex][axis]) ** 2
                                           for axis in range(3))) for vertex in range(3))
        max_residual = max(distances, default=0.0)
        rms_residual = math.sqrt(sum(value * value for value in distances) / max(1, len(distances)))
        original = sorted(triangle_signatures(source_obj, source_faces))
        derived = sorted(signature for obj, faces in derived_by_object.items()
                         for signature in triangle_signatures(obj, faces))
        diagnostic = {"source_object": row["source_object"], "source_node_index": row["source_node_index"],
                      "source_component": row["source_component"],
                      "triangle_count": len(source_faces), "max_vertex_residual_m": max_residual,
                      "rms_vertex_residual_m": rms_residual,
                      "tolerance_m": tolerance_m, "within_tolerance": max_residual <= tolerance_m,
                      "quantized_signature_equal": original == derived,
                      "source_matrix_world": [float(value) for line in source_obj.matrix_world for value in line],
                      "derived_matrix_world": [[float(value) for line in obj.matrix_world for value in line]
                                                for obj in derived_by_object]}
        checks.append(diagnostic)
        require(len(checks) <= 3, "source-open diagnostic component list exceeds fixed bound")
        print("SOURCE_OPEN_INVARIANCE_DIAGNOSTIC " + json.dumps(diagnostic, sort_keys=True))
    all_within = len(checks) == len(EXPECTED_LID_COMPONENT_OVERRIDES) and all(row["within_tolerance"] for row in checks)
    for row in checks:
        row["world_triangle_geometry_within_tolerance"] = row["within_tolerance"]
    persist_diagnostic(ROOT / "source-open-invariance-diagnostic.json", {
        "status": "measured_before_tolerance_assertion", "tolerance_m": tolerance_m,
        "all_components_within_tolerance": all_within, "components": checks})
    require(all_within, "source-open world-space geometry exceeds the pinned 1e-7 m tolerance")
    return {"method": "source triangle index matched before/after parenting; raw world-space vertex residual measured per vertex; rounded 1e-7 m signatures retained as diagnostics only",
            "pose_rotation_degrees": 0.0, "tolerance_m": tolerance_m,
            "all_components_within_tolerance": all_within,
            "components": checks}


def remove_source_face_attribution(fragments):
    names = ("source_node_index", "source_component_id", "source_triangle_index")
    for objects in fragments.values():
        for obj in objects:
            for name in names:
                attribute = obj.data.attributes.get(name)
                if attribute is not None:
                    obj.data.attributes.remove(attribute)


def configure_scene(cfg):
    scene = bpy.context.scene
    world = bpy.data.worlds.new("Laptop diagnostic world")
    world.use_nodes = True
    background = world.node_tree.nodes.get("Background")
    require(background is not None, "world background node is unavailable")
    background.inputs["Color"].default_value = cfg["render"]["world_background_linear_rgba"]
    background.inputs["Strength"].default_value = cfg["render"]["world_background_strength"]
    world.color = tuple(cfg["render"]["world_background_linear_rgba"][:3])
    scene.world = world
    scene.render.engine = "CYCLES"
    scene.cycles.device = "CPU"
    scene.cycles.samples = cfg["render"]["samples"]
    scene.cycles.seed = cfg["render"]["seed"]
    scene.render.resolution_x = cfg["render"]["width"]
    scene.render.resolution_y = cfg["render"]["height"]
    scene.render.resolution_percentage = 100
    scene.render.threads_mode = "FIXED"
    scene.render.threads = 2
    scene.render.image_settings.file_format = "PNG"
    scene.render.film_transparent = False
    scene.view_settings.view_transform = "Standard"
    scene.view_settings.look = "None"
    scene.render.image_settings.color_mode = "RGBA"


def add_lights(center):
    for name, delta, power, size in (
        ("DiagnosticKey", (0.30, -0.45, 0.48), 28.0, 0.45),
        ("DiagnosticFill", (-0.38, -0.1, 0.20), 12.0, 0.5),
        ("DiagnosticRim", (0.0, 0.35, 0.3), 16.0, 0.42),
    ):
        data = bpy.data.lights.new(name, "AREA")
        data.energy, data.shape, data.size = power, "DISK", size
        light = bpy.data.objects.new(name, data)
        bpy.context.scene.collection.objects.link(light)
        light.location = Vector(center) + Vector(delta)
        light.rotation_euler = (Vector(center) - light.location).to_track_quat("-Z", "Y").to_euler()


def tint_role(obj, role):
    replacements = []
    for index, source_material in enumerate(obj.data.materials):
        material = source_material.copy() if source_material else bpy.data.materials.new(f"Diagnostic {role} {index}")
        material.name = f"Laptop diagnostic {role} {obj.name} {index}"
        material.use_nodes = True
        material.node_tree.nodes.clear()
        shader = material.node_tree.nodes.new("ShaderNodeBsdfPrincipled")
        output = material.node_tree.nodes.new("ShaderNodeOutputMaterial")
        material.node_tree.links.new(shader.outputs["BSDF"], output.inputs["Surface"])
        base = shader.inputs.get("Base Color")
        require(base is not None, "diagnostic shader lacks base color")
        base.default_value = ROLE_COLORS[role]
        shader.inputs["Metallic"].default_value = 0.0
        shader.inputs["Roughness"].default_value = 0.75
        replacements.append((index, material))
    return replacements


def replace_role_materials(fragments):
    restore = []
    for role, objects in fragments.items():
        for obj in objects:
            for index, material in tint_role(obj, "hinge_review" if role == "hinge-review" else role):
                original = obj.data.materials[index]
                restore.append((obj, index, original))
                obj.data.materials[index] = material
    return restore


def restore_materials(restore):
    for obj, index, material in restore:
        obj.data.materials[index] = material


def camera(name, center, delta, ortho):
    data = bpy.data.cameras.new(name)
    data.type = "ORTHO"
    data.ortho_scale = ortho
    obj = bpy.data.objects.new(name, data)
    bpy.context.scene.collection.objects.link(obj)
    target = Vector(center)
    obj.location = target + Vector(delta)
    obj.rotation_euler = (target - obj.location).to_track_quat("-Z", "Y").to_euler()
    return obj


def actual_points(objects):
    bpy.context.view_layer.update()
    return [obj.matrix_world @ vertex.co for obj in objects for vertex in obj.data.vertices]


def percentile(values, fraction):
    require(values and 0.0 <= fraction <= 1.0, "invalid bounded percentile input")
    ordered = sorted(float(value) for value in values)
    return ordered[min(len(ordered) - 1, int(round(fraction * (len(ordered) - 1))))]


def normalized(vector):
    require(vector.length > 1e-9, "cannot normalize a degenerate plane normal")
    return vector.normalized()


def surface_normal(obj, selector=None):
    weighted = Vector((0.0, 0.0, 0.0))
    total_area = 0.0
    for polygon in obj.data.polygons:
        corners = [obj.matrix_world @ obj.data.vertices[index].co for index in polygon.vertices]
        if len(corners) < 3:
            continue
        cross = (corners[1] - corners[0]).cross(corners[2] - corners[0])
        area = cross.length * 0.5
        if area <= 1e-12:
            continue
        normal = cross.normalized()
        center = sum(corners, Vector((0.0, 0.0, 0.0))) / len(corners)
        if selector is None or selector(normal, center):
            weighted += normal * area
            total_area += area
    require(total_area > 1e-8, "no area found for requested reference plane")
    return normalized(weighted), total_area


def plane_alignment_angle(screen_normal, base_normal):
    require(abs(screen_normal.x) < 0.08 and abs(base_normal.x) < 0.08,
            "screen/base plane alignment is not representable by the fixed X hinge axis")
    screen = normalized(Vector((0.0, screen_normal.y, screen_normal.z)))
    base = normalized(Vector((0.0, base_normal.y, base_normal.z)))

    def angle_to(target):
        target = normalized(Vector((0.0, target.y, target.z)))
        cross = screen.y * target.z - screen.z * target.y
        dot = screen.y * target.y + screen.z * target.z
        return math.degrees(math.atan2(cross, dot))

    same = angle_to(base)
    opposite = angle_to(-base)
    selected = opposite + 360.0 if opposite < 0.0 else opposite
    require(70.0 <= selected <= 130.0, "plane-derived closed pose is outside the bounded near-closed branch")
    return selected, same, opposite


def rotate_about_x(points, pivot, angle_degrees):
    angle = math.radians(angle_degrees)
    cosine, sine = math.cos(angle), math.sin(angle)
    result = []
    for point in points:
        relative = point - pivot
        result.append(Vector((point.x,
                              pivot.y + relative.y * cosine - relative.z * sine,
                              pivot.z + relative.y * sine + relative.z * cosine)))
    return result


def fit_closed_pivot(fragments, screen, pivot, source_axis, cfg):
    base_objects = fragments["base"]
    lid_objects = fragments["lid"]
    base_points = actual_points(base_objects)
    lid_points = actual_points(lid_objects)
    screen_points = actual_points([screen])
    base_top = percentile([point.z for point in base_points], cfg["rig"]["base_top_percentile"])
    base_normal, base_area = surface_normal(
        base_objects[0],
        lambda normal, center: normal.z > 0.75 and center.z >= base_top - cfg["rig"]["base_plane_band_m"],
    )
    screen_normal, screen_area = surface_normal(screen)
    angle, same_angle, opposite_angle = plane_alignment_angle(screen_normal, base_normal)
    rotated_lid = rotate_about_x(lid_points, source_axis, angle)
    rotated_screen = rotate_about_x(screen_points, source_axis, angle)
    base_plane_reference = Vector((0.0, 0.0, base_top))
    screen_plane_distances = [(point - base_plane_reference).dot(base_normal) for point in rotated_screen]
    target_screen_gap = cfg["rig"]["target_screen_deck_gap_m"]
    screen_gap_delta = target_screen_gap - percentile(screen_plane_distances, 0.5)
    depth_fraction = cfg["rig"]["depth_percentile"]
    base_depth = [point.y for point in base_points]
    lid_depth = [point.y for point in rotated_lid]
    base_depth_center = (percentile(base_depth, 1.0 - depth_fraction) + percentile(base_depth, depth_fraction)) / 2.0
    lid_depth_center = (percentile(lid_depth, 1.0 - depth_fraction) + percentile(lid_depth, depth_fraction)) / 2.0
    delta_depth = base_depth_center - lid_depth_center
    require(abs(base_normal.z) > 0.98, "base deck normal is too tilted for bounded vertical pivot fit")
    delta_height = (screen_gap_delta - base_normal.y * delta_depth) / base_normal.z
    radians = math.radians(angle)
    a, sine = 1.0 - math.cos(radians), math.sin(radians)
    determinant = a * a + sine * sine
    require(determinant > 1e-6, "closed-angle pivot fit is singular")
    delta_axis_depth = (a * delta_depth - sine * delta_height) / determinant
    delta_axis_height = (sine * delta_depth + a * delta_height) / determinant
    fitted_axis = source_axis + Vector((0.0, delta_axis_depth, delta_axis_height))
    require((fitted_axis - source_axis).length <= cfg["rig"]["max_axis_adjustment_m"],
            "plane/footprint fit requires an unbounded pivot adjustment")

    original_world = [(obj, obj.matrix_world.copy(), obj.matrix_basis.copy()) for obj in lid_objects]
    pivot.location = fitted_axis
    bpy.context.view_layer.update()
    inverse = pivot.matrix_world.inverted()
    for obj, world, basis in original_world:
        obj.matrix_parent_inverse = inverse @ world @ basis.inverted()
    bpy.context.view_layer.update()
    for obj, world, _basis in original_world:
        difference = max(abs(obj.matrix_world[row][column] - world[row][column]) for row in range(4) for column in range(4))
        require(difference <= 1e-6, "changing the fitted pivot moved the source-open lid geometry")

    pivot.rotation_euler.x = math.radians(angle)
    bpy.context.view_layer.update()
    closed_base = actual_points(base_objects)
    closed_lid = actual_points(lid_objects)
    closed_screen = actual_points([screen])
    actual_screen_gaps = [(point - base_plane_reference).dot(base_normal) for point in closed_screen]
    screen_gap_min = min(actual_screen_gaps)
    screen_gap_median = percentile(actual_screen_gaps, 0.5)
    screen_gap_p95 = percentile(actual_screen_gaps, 0.95)
    global_min_point = min(closed_lid, key=lambda point: point.z)
    global_min_gap = global_min_point.z - base_top
    closed_base_depth = [point.y for point in closed_base]
    closed_lid_depth = [point.y for point in closed_lid]
    actual_base_center = (percentile(closed_base_depth, 1.0 - depth_fraction) + percentile(closed_base_depth, depth_fraction)) / 2.0
    actual_lid_center = (percentile(closed_lid_depth, 1.0 - depth_fraction) + percentile(closed_lid_depth, depth_fraction)) / 2.0
    actual_center_delta = actual_lid_center - actual_base_center
    require(cfg["rig"]["minimum_screen_deck_gap_m"] <= screen_gap_median <= cfg["rig"]["maximum_screen_deck_gap_m"],
            f"closed screen/deck median gap is outside target ({screen_gap_median:.6f} m)")
    require(abs(actual_center_delta) <= cfg["rig"]["maximum_depth_center_error_m"],
            f"closed candidate footprint centers remain misaligned ({actual_center_delta:.6f} m)")
    pivot.rotation_euler.x = 0.0
    bpy.context.view_layer.update()

    glb_axis = GLB_TO_BLENDER.inverted() @ fitted_axis.to_4d()
    return {
        "method": "screen outward normal aligned opposite the area-weighted base top-plane normal; YZ pivot translation solved so the screen median signed distance to the base deck plane reaches the declared target while robust depth centers align",
        "source_axis_blender_m": list(source_axis), "fitted_axis_blender_m": list(fitted_axis),
        "fitted_axis_glb_m": [float(value) for value in glb_axis[:3]],
        "axis_adjustment_blender_m": [0.0, float(delta_axis_depth), float(delta_axis_height)],
        "screen_plane_normal_blender": list(screen_normal), "base_top_plane_normal_blender": list(base_normal),
        "screen_plane_area_m2": screen_area, "base_top_plane_area_m2": base_area,
        "screen_to_base_plane_alignment_angle_degrees": same_angle,
        "screen_to_opposite_base_plane_alignment_angle_degrees": opposite_angle,
        "selected_closed_angle_degrees": angle,
        "base_deck_reference_height_m": base_top, "target_screen_deck_gap_m": target_screen_gap,
        "screen_deck_gap_min_m": screen_gap_min, "screen_deck_gap_median_m": screen_gap_median,
        "screen_deck_gap_p95_m": screen_gap_p95,
        "lid_geometry_lowest_point_relative_to_deck_m": global_min_gap,
        "lid_geometry_lowest_point_world_m": [float(value) for value in global_min_point],
        "lid_geometry_minimum_is_diagnostic_only": True,
        "realized_depth_center_error_at_diagnostic_pose_m": actual_center_delta,
        "diagnostic_angle_degrees": angle,
        "uncertainty": "heuristic fit to source mesh planes and vertex envelopes; no physical measurement, tolerance, hinge mechanism, or continuous clearance claim",
    }


def projection_margin(objects, camera_obj, cfg):
    bpy.context.view_layer.update()
    inverse = camera_obj.matrix_world.inverted()
    half_width = camera_obj.data.ortho_scale / 2
    half_height = half_width * cfg["render"]["height"] / cfg["render"]["width"]
    margin = 1.0
    for point in actual_points(objects):
        projected = inverse @ point
        margin = min(margin, (half_width - abs(projected.x)) / half_width,
                     (half_height - abs(projected.y)) / half_height)
    require(margin >= cfg["render"]["minimum_projected_margin_fraction"], f"pose cropped at fixed camera (margin {margin:.4f})")
    return margin


def render_png(path, camera_obj):
    scene = bpy.context.scene
    scene.camera = camera_obj
    scene.render.filepath = str(path)
    bpy.ops.render.render(write_still=True)
    require(path.is_file() and not path.is_symlink() and 0 < path.stat().st_size <= cfg_size_limit(), "diagnostic PNG absent or outside bound")


def cfg_size_limit():
    return 8 * 1024 * 1024


def sampled_overlap_pairs(fragments):
    depsgraph = bpy.context.evaluated_depsgraph_get()
    lids = fragments["lid"]
    fixed = fragments["base"] + fragments["hinge-review"]

    def world_tree(obj):
        evaluated = obj.evaluated_get(depsgraph)
        mesh = evaluated.to_mesh()
        try:
            matrix = evaluated.matrix_world
            vertices = [matrix @ vertex.co for vertex in mesh.vertices]
            polygons = [list(poly.vertices) for poly in mesh.polygons]
            return BVHTree.FromPolygons(vertices, polygons, all_triangles=False, epsilon=1e-7)
        finally:
            evaluated.to_mesh_clear()

    candidates = 0
    for lid in lids:
        lid_tree = world_tree(lid)
        for stationary in fixed:
            other = world_tree(stationary)
            candidates += len(lid_tree.overlap(other))
    return candidates


def local_closed_surface_witnesses(fragments, screen, cfg):
    """Sample real downward lid faces against upward base faces in world space."""
    depsgraph = bpy.context.evaluated_depsgraph_get()

    def face_cloud(objects, direction):
        vertices, polygons, samples, face_origins = [], [], [], []
        for obj in objects:
            evaluated = obj.evaluated_get(depsgraph)
            mesh = evaluated.to_mesh()
            try:
                offset = len(vertices)
                matrix = evaluated.matrix_world
                vertices.extend(matrix @ vertex.co for vertex in mesh.vertices)
                for polygon in mesh.polygons:
                    normal = (matrix.to_3x3() @ polygon.normal).normalized()
                    if normal.z * direction < 0.75 or len(polygon.vertices) != 3:
                        continue
                    corners = [vertices[offset + index] for index in polygon.vertices]
                    polygons.append([offset + index for index in polygon.vertices])
                    node_attr = mesh.attributes.get("source_node_index")
                    component_attr = mesh.attributes.get("source_component_id")
                    require(node_attr is not None and component_attr is not None, "source face attribution was lost during role join")
                    source_node_index = int(node_attr.data[polygon.index].value)
                    source_component = int(component_attr.data[polygon.index].value)
                    samples.append((obj.name, polygon.index, sum(corners, Vector()) / 3.0,
                                    source_node_index, source_component))
                    face_origins.append((source_node_index, source_component))
            finally:
                evaluated.to_mesh_clear()
        require(polygons, "local gap sample has no selected opposing faces")
        return BVHTree.FromPolygons(vertices, polygons, all_triangles=True, epsilon=1e-8), samples, face_origins

    base_tree, _base_faces, base_origins = face_cloud(fragments["base"], 1.0)
    source_names = {row[0]: row[1] for row in cfg["mesh_node_map"]}
    results = {"base_to_lid_underside": {}, "base_to_screen_surface": {}}
    for label, objects in (("base_to_lid_underside", [obj for obj in fragments["lid"] if obj is not screen]),
                           ("base_to_screen_surface", [screen])):
        _unused_tree, underside_faces, _underside_origins = face_cloud(objects, -1.0)
        distances, signed_gaps = [], []
        witnesses, negative_samples = [], []
        for owner, face_index, point, source_node_index, source_component in underside_faces:
            nearest = base_tree.find_nearest(point)
            if nearest is None or nearest[3] > 0.02:
                continue
            distance = float(nearest[3])
            signed_gap = float((point - nearest[0]).dot(nearest[1]))
            distances.append(distance)
            signed_gaps.append(signed_gap)
            base_node_index, base_component = base_origins[int(nearest[2])]
            origin = {"source_object": source_names[source_node_index], "source_component": source_component,
                      "role_mesh_polygon": face_index,
                      "base_source_object": source_names[base_node_index], "base_source_component": base_component,
                      "lid_point_world_m": [float(v) for v in point],
                      "base_point_world_m": [float(v) for v in nearest[0]],
                      "euclidean_distance_m": distance, "signed_gap_to_base_normal_m": signed_gap}
            if signed_gap < 0.0 and len(negative_samples) < 256:
                negative_samples.append(origin)
            if len(witnesses) < 12 or distance < max(item["distance_m"] for item in witnesses):
                witnesses.append({"lid_object": owner, "lid_polygon": face_index,
                                  "source_object": source_names[source_node_index], "source_component": source_component,
                                  "role_mesh_polygon": face_index,
                                  "lid_point_world_m": [float(v) for v in point],
                                  "base_point_world_m": [float(v) for v in nearest[0]],
                                  "base_source_object": source_names[base_node_index], "base_source_component": base_component,
                                  "distance_m": distance, "signed_gap_to_base_normal_m": signed_gap})
                witnesses.sort(key=lambda item: item["distance_m"])
                witnesses = witnesses[:12]
        require(distances, "no underside samples fell within 20 mm of the base top")
        results[label] = {
            "method": "nearest points from actual selected world-space triangle centroids; base faces have upward normals and lid faces downward normals; samples farther than 20 mm omitted",
            "sample_count": len(distances), "within_20mm_count": len(distances),
            "min_distance_m": min(distances), "median_distance_m": percentile(distances, 0.5),
            "p95_distance_m": percentile(distances, 0.95), "max_distance_m": max(distances),
            "signed_gap_min_m": min(signed_gaps), "signed_gap_median_m": percentile(signed_gaps, 0.5),
            "signed_gap_p95_m": percentile(signed_gaps, 0.95),
            "signed_gap_negative_sample_count": sum(value < 0.0 for value in signed_gaps),
            "signed_gap_scope": "downward-face centroid samples only; signed normal projection is local diagnostic, not continuous clearance",
            "negative_sample_records": negative_samples,
            "negative_sample_records_truncated": len(negative_samples) < sum(value < 0.0 for value in signed_gaps),
            "closest_witnesses": witnesses,
            "screen_mesh_included": label == "base_to_screen_surface",
            "qualification": "discrete centroid proximity only; not continuous clearance or a fitted physical tolerance",
        }
    return results


def render_closed_material_macros(fragments, base_points, fitted_axis_y, cfg, output_records):
    """Two fixed side crops using retained source materials except neutralized screen."""
    base_y = [point.y for point in base_points]
    base_x = [point.x for point in base_points]
    targets = (("closed-hinge-material-macro.png", fitted_axis_y),
               ("closed-front-material-macro.png", percentile(base_y, 0.12)))
    for filename, target_y in targets:
        center = (closed_hinge_macro_target(base_points, fitted_axis_y, cfg) if filename == "closed-hinge-material-macro.png"
                  else (percentile(base_x, 0.5), target_y, percentile([point.z for point in base_points], 0.9) + 0.006))
        close = camera("Closed source-material seam macro", center, (1.0, 0.0, 0.0), cfg["render"]["macro_ortho_width_m"])
        scene = bpy.context.scene
        old_resolution = (scene.render.resolution_x, scene.render.resolution_y)
        scene.render.resolution_x, scene.render.resolution_y = cfg["render"]["macro_width"], cfg["render"]["macro_height"]
        render_png(OUTPUTS / filename, close)
        data = (OUTPUTS / filename).read_bytes()
        output_records[filename] = {"sha256": digest(data), "size_bytes": len(data),
                                    "width": cfg["render"]["macro_width"], "height": cfg["render"]["macro_height"],
                                    "target_world_y_m": target_y,
                                    "materials": "original non-screen source slots; display image remains neutralized"}
        scene.render.resolution_x, scene.render.resolution_y = old_resolution
    bpy.data.objects.remove(close, do_unlink=True)


def closed_hinge_macro_target(base_points, fitted_axis_y, cfg):
    """Shared rear-band anchor for material and component-key close-ups."""
    anchor = cfg["render"]["component_macro_anchor"]
    require(anchor["rule"] == "closed-hinge-material-macro-v1", "component macro anchor rule changed")
    return (percentile([point.x for point in base_points], 0.5), fitted_axis_y,
            percentile([point.z for point in base_points], anchor["base_z_percentile"]) + anchor["base_z_offset_m"])


def component_diagnostic_material(color, name):
    material = bpy.data.materials.new(name)
    material.diffuse_color = color
    material.use_nodes = True
    material.node_tree.nodes.clear()
    shader = material.node_tree.nodes.new("ShaderNodeBsdfPrincipled")
    output = material.node_tree.nodes.new("ShaderNodeOutputMaterial")
    material.node_tree.links.new(shader.outputs["BSDF"], output.inputs["Surface"])
    shader.inputs["Base Color"].default_value = color
    shader.inputs["Metallic"].default_value = 0.0
    shader.inputs["Roughness"].default_value = 0.75
    return material


def render_component_keyed_diagnostics(fragments, side_camera, base_points, fitted_axis_y, cfg, output_records):
    """Color exact source components in the already-fitted closed pose; no transforms change."""
    require(all(obj.visible_get() for objects in fragments.values() for obj in objects),
            "closed-pose diagnostic expects all candidate and context meshes visible")
    bindings = cfg["output"]["component_diagnostic_bindings"]
    require(len(bindings) == 6, "component diagnostic binding count changed")
    targets = {(int(row["source_node_index"]), int(row["source_component"])): row for row in bindings}
    expected_components = {
        (0, 310): ("lid", "LaptopLidGeometry", 184),
        (0, 313): ("lid", "LaptopLidGeometry", 148),
        (2, 92): ("lid", "LaptopLidGeometry", 456),
        (8, 12): ("hinge-review", "LaptopHingeTrimCandidate", 584),
        (3, 0): ("hinge-review", "LaptopHingeReviewGeometry", 3582),
        (0, 316): ("lid", "LaptopLidGeometry", 298),
    }
    require(set(targets) == set(expected_components), "component diagnostic source identities differ from the fixed profile")
    for key, row in targets.items():
        role, mesh_name, triangles = expected_components[key]
        require(row["candidate_role"] == role and row["derived_mesh_name"] == mesh_name and
                int(row["triangle_count"]) == triangles,
                f"component {row['label']} has unexpected role/derived mesh/count binding")
    matched = {key: 0 for key in targets}
    transform_before = {}
    changed = {}
    materials_added = []
    target_meshes = set()
    for objects in fragments.values():
        for obj in objects:
            transform_before[id(obj)] = (obj, obj.matrix_world.copy())
            node_attr = obj.data.attributes.get("source_node_index")
            component_attr = obj.data.attributes.get("source_component_id")
            if node_attr is None or component_attr is None:
                continue
            rows_by_key = {}
            for polygon in obj.data.polygons:
                key = (int(node_attr.data[polygon.index].value), int(component_attr.data[polygon.index].value))
                row = targets.get(key)
                if row is None:
                    continue
                rows_by_key.setdefault(key, []).append(polygon.index)
            if not rows_by_key:
                continue
            target_meshes.add(obj.name)
            original_slot_count = len(obj.data.materials)
            old_indices = [polygon.material_index for polygon in obj.data.polygons]
            slots = {}
            for key, polygon_ids in rows_by_key.items():
                row = targets[key]
                if obj.name != row["derived_mesh_name"]:
                    raise RuntimeError(f"laptop starter derivation: component {row['label']} mapped to {obj.name}, expected {row['derived_mesh_name']}")
                material = component_diagnostic_material(row["color_rgba"], "Component key " + row["label"])
                obj.data.materials.append(material)
                materials_added.append((obj.data, material))
                slots[key] = original_slot_count + len(slots)
                for polygon_id in polygon_ids:
                    obj.data.polygons[polygon_id].material_index = slots[key]
                matched[key] += len(polygon_ids)
            changed[id(obj)] = (obj, old_indices, original_slot_count)
    wanted_meshes = {row["derived_mesh_name"] for row in bindings}
    require(target_meshes == wanted_meshes, "component keys no longer map to the fixed lid/review meshes")
    for key, row in targets.items():
        require(matched[key] == int(row["triangle_count"]),
                "component diagnostic triangle attribution changed for " + row["label"])
    scene = bpy.context.scene
    scene.camera = side_camera
    side_name = "closed-component-key-side.png"
    render_png(OUTPUTS / side_name, side_camera)
    side_bytes = (OUTPUTS / side_name).read_bytes()
    output_records[side_name] = {"sha256": digest(side_bytes), "size_bytes": len(side_bytes),
                                 "width": cfg["render"]["width"], "height": cfg["render"]["height"],
                                 "diagnostic": "same closed pose and transforms; target components keyed by manifest colors"}
    points = []
    for objects in fragments.values():
        for obj in objects:
            node_attr = obj.data.attributes.get("source_node_index")
            component_attr = obj.data.attributes.get("source_component_id")
            if node_attr is None or component_attr is None:
                continue
            for polygon in obj.data.polygons:
                key = (int(node_attr.data[polygon.index].value), int(component_attr.data[polygon.index].value))
                if key in targets:
                    points.extend(obj.matrix_world @ obj.data.vertices[index].co for index in polygon.vertices)
    require(points, "component diagnostic has no keyed geometry points")
    # Match the existing hinge-material macro's rear-band anchor. The broad
    # chassis sidewall remains visible context but must not pull the crop center.
    center = closed_hinge_macro_target(base_points, fitted_axis_y, cfg)
    macro = camera("Closed component-key hinge macro", center, (1.0, 0.0, 0.0), cfg["render"]["macro_ortho_width_m"])
    old_resolution = (scene.render.resolution_x, scene.render.resolution_y)
    scene.render.resolution_x, scene.render.resolution_y = cfg["render"]["macro_width"], cfg["render"]["macro_height"]
    macro_name = "closed-component-key-hinge-macro.png"
    render_png(OUTPUTS / macro_name, macro)
    macro_bytes = (OUTPUTS / macro_name).read_bytes()
    output_records[macro_name] = {"sha256": digest(macro_bytes), "size_bytes": len(macro_bytes),
                                  "width": cfg["render"]["macro_width"], "height": cfg["render"]["macro_height"],
                                  "diagnostic": "same closed pose; exact candidate components keyed by manifest colors",
                                  "camera_target_rule": "closed-hinge-material-macro-v1",
                                  "camera_target_world_m": [float(value) for value in center],
                                  "camera_target_region_world_m": {
                                      "y_min": center[1] - cfg["render"]["macro_ortho_width_m"] / 2,
                                      "y_max": center[1] + cfg["render"]["macro_ortho_width_m"] / 2,
                                      "z_min": center[2] - cfg["render"]["macro_ortho_width_m"] * cfg["render"]["macro_height"] / cfg["render"]["macro_width"] / 2,
                                      "z_max": center[2] + cfg["render"]["macro_ortho_width_m"] * cfg["render"]["macro_height"] / cfg["render"]["macro_width"] / 2},
                                  "camera_ortho_width_m": cfg["render"]["macro_ortho_width_m"],
                                  "target_excludes_component": "Object_22/0 full chassis sidewall; retained only as keyed context"}
    scene.render.resolution_x, scene.render.resolution_y = old_resolution
    bpy.data.objects.remove(macro, do_unlink=True)
    scene.camera = side_camera
    for obj, old_indices, _slot_start in changed.values():
        for polygon, old_index in zip(obj.data.polygons, old_indices):
            polygon.material_index = old_index
    for mesh, material in reversed(materials_added):
        slot = next((index for index, current in enumerate(mesh.materials) if current == material), None)
        if slot is not None:
            mesh.materials.pop(index=slot)
        bpy.data.materials.remove(material)
    for obj, before in transform_before.values():
        require(all(abs(a - b) <= 1e-7 for row_a, row_b in zip(before, obj.matrix_world)
                    for a, b in zip(row_a, row_b)), "component diagnostic changed object transform")
    return [{"source_object": row["source_object"], "source_component": row["source_component"],
             "source_node_index": row["source_node_index"], "triangle_count": matched[(row["source_node_index"], row["source_component"])],
             "candidate_role": row["candidate_role"], "derived_mesh_name": row["derived_mesh_name"],
             "label": row["label"], "color_rgba": row["color_rgba"], "visibility": "visible",
             "object_transform_unchanged": True} for row in bindings]


def export_candidate(root, cfg):
    glb_path = OUTPUTS / cfg["output"]["glb_name"]
    bpy.ops.object.select_all(action="DESELECT")
    for obj in bpy.context.scene.objects:
        if obj.type == "MESH" or obj.name in ("LaptopStarterRoot", "LaptopBaseCandidate", "LaptopLidPivotCandidate", "LaptopHingeReview"):
            obj.select_set(True)
    require(bpy.context.view_layer.objects.active is not None, "no candidate objects selected")
    result = bpy.ops.export_scene.gltf(filepath=str(glb_path), export_format="GLB", use_selection=True,
                                      export_yup=True, export_apply=False, export_cameras=False,
                                      export_lights=False, export_materials="EXPORT", export_extras=True,
                                      export_texcoords=True, export_normals=True, export_animations=False)
    require(result == {"FINISHED"} and glb_path.is_file() and not glb_path.is_symlink(), "candidate GLB export failed")
    data = glb_path.read_bytes()
    require(len(data) == glb_path.stat().st_size and 0 < len(data) <= cfg["output"]["max_glb_bytes"], "candidate GLB size invalid")
    doc, binary = parse_glb(data)
    require(not doc.get("animations") and not doc.get("skins") and not doc.get("cameras"), "candidate contains unapproved animation/camera/skin")
    require(all(not image.get("uri") for image in doc.get("images", [])), "candidate externalizes source images")
    nodes = doc["nodes"]
    names = {node.get("name") for node in nodes}
    require({"LaptopStarterRoot", "LaptopBaseCandidate", "LaptopLidPivotCandidate", "LaptopHingeReview"}.issubset(names), "candidate hierarchy roles missing")
    meshes = [node for node in nodes if "mesh" in node]
    triangle_count = 0
    for node in meshes:
        for primitive in doc["meshes"][node["mesh"]].get("primitives", []):
            require(primitive.get("mode", 4) == 4, "candidate primitive is not triangles")
            triangle_count += doc["accessors"][primitive["indices"]]["count"] // 3
    require(triangle_count == 112502, "candidate GLB triangle census differs")
    screen_node = next((node for node in nodes if node.get("name") == "LaptopScreenSurface"), None)
    require(screen_node is not None, "display plane is not attached to the lid role")
    material_index = doc["meshes"][screen_node["mesh"]]["primitives"][0].get("material")
    material = doc["materials"][material_index]
    texture = material.get("emissiveTexture", {}).get("index")
    image_index = doc["textures"][texture].get("source")
    image_bytes = extract_image_bytes(doc, binary, image_index)
    require(digest(image_bytes) != cfg["source"]["embedded_images"][SCREEN_IMAGE_INDEX][2], "fictional source display pixels remain")
    check_path = OUTPUTS / ".screen-neutral-check.png"
    check_path.write_bytes(image_bytes)
    checked = bpy.data.images.load(str(check_path), check_existing=False)
    expected_dimensions = cfg["source"]["embedded_images"][SCREEN_IMAGE_INDEX][3:5]
    require(list(checked.size) == expected_dimensions, "neutral screen raster dimensions changed")
    pixels = [0.0] * (expected_dimensions[0] * expected_dimensions[1] * 4)
    checked.pixels.foreach_get(pixels)
    channels = [[pixels[index] for index in range(channel, len(pixels), 4)] for channel in range(4)]
    require(all(max(values) - min(values) < 0.01 for values in channels[:3]), "derived screen raster is not uniform gray")
    require(min(channels[3]) > 0.99, "derived screen raster is not opaque")
    bpy.data.images.remove(checked)
    check_path.unlink()
    return data, doc, image_bytes, triangle_count


def main():
    cfg, manifest_sha, source_data, source_doc, component_data, _source_binary = read_inputs()
    require(len(component_data["components"]) == cfg["component_inventory"]["component_count"] == 587, "component record count differs")
    require(component_data["all_triangles_accounted_for"] == cfg["source"]["triangle_count"] == 112502, "component triangle total differs")
    bpy.ops.wm.read_factory_settings(use_empty=True)
    require(bpy.ops.import_scene.gltf(filepath=str(INPUTS / SOURCE_NAME)) == {"FINISHED"}, "pinned laptop GLB import failed")
    bpy.context.view_layer.update()
    mapping, lookup = source_mapping(source_doc, component_data["components"], cfg)
    roles = validate_import(source_doc, lookup, mapping, cfg)
    screen_name = source_doc["nodes"][cfg["source"]["screen_binding"]["node_index"]]["name"]
    require(screen_name == SCREEN_NODE and "lid" in roles[screen_name][1], "screen plane is not in lid role")
    neutral_image, neutral_facts = neutralize_screen(cfg)
    axis_glb = Vector(cfg["rig"]["source_axis_glb_m"]).to_4d()
    axis = (GLB_TO_BLENDER @ axis_glb).to_3d()
    OUTPUTS.mkdir()
    root, base_root, pivot, review_root, fragments, all_fragments, source_role_inventory = create_role_assembly(roles, axis)
    source_open_geometry = source_open_override_geometry(roles, fragments, cfg)
    counts = role_triangle_counts(fragments)
    screen = next((obj for obj in fragments["lid"] if obj.name == "LaptopScreenSurface"), None)
    require(screen is not None, "screen surface is missing from derived lid")
    closed_fit = fit_closed_pivot(fragments, screen, pivot, axis, cfg)
    trim_alignment = align_fixed_hinge_trim(fragments, axis, Vector(closed_fit["fitted_axis_blender_m"]), cfg)
    # Drop the imported unsplit originals so only the derived assembly is exported.
    for obj in list(bpy.context.scene.objects):
        if obj.type == "MESH" and obj.name in roles:
            bpy.data.objects.remove(obj, do_unlink=True)
    configure_scene(cfg)
    points = actual_points(all_fragments)
    center = [(min(float(point[i]) for point in points) + max(float(point[i]) for point in points)) / 2 for i in range(3)]
    add_lights(center)
    side = camera("Laptop close side", center, (1.0, 0.0, 0.0), cfg["render"]["ortho_width_m"])
    rear = camera("Laptop close rear", center, (0.0, -1.0, 0.0), cfg["render"]["ortho_width_m"])
    restore = replace_role_materials(fragments)
    pose_records, output_records = {}, {}
    diagnostic_poses = (
        ("source-open", float(cfg["rig"]["diagnostic_angles_degrees"][0])),
        ("intermediate", float(cfg["rig"]["diagnostic_angles_degrees"][1])),
        ("closed", float(closed_fit["selected_closed_angle_degrees"])),
    )
    for pose_label, angle in diagnostic_poses:
        pivot.rotation_euler.x = math.radians(angle)
        bpy.context.view_layer.update()
        overlaps = sampled_overlap_pairs(fragments)
        pose_record = {
            "pose_label": pose_label,
            "closure_from_source_open_degrees": angle,
            "candidate_triangle_surface_overlap_pairs": overlaps,
            "sampled_envelope_status": "overlap_candidates_detected" if overlaps else "no_candidates_at_sampled_pose_unqualified",
            "views": {},
        }
        for view_name, camera_obj in (("side", side), ("rear", rear)):
            margin = projection_margin(all_fragments, camera_obj, cfg)
            filename = f"{pose_label}-{view_name}.png"
            render_png(OUTPUTS / filename, camera_obj)
            data = (OUTPUTS / filename).read_bytes()
            output_records[filename] = {"sha256": digest(data), "size_bytes": len(data), "width": cfg["render"]["width"], "height": cfg["render"]["height"]}
            pose_record["views"][view_name] = {"projected_margin_fraction": margin, "output": filename}
        pose_records[pose_label] = pose_record
    local_witnesses = local_closed_surface_witnesses(fragments, screen, cfg)
    restore_materials(restore)
    render_closed_material_macros(fragments, actual_points(fragments["base"]),
                                  closed_fit["fitted_axis_blender_m"][1], cfg, output_records)
    component_bindings = render_component_keyed_diagnostics(
        fragments, side, actual_points(fragments["base"]), closed_fit["fitted_axis_blender_m"][1], cfg, output_records)
    # The diagnostic keeps trim separate so its authored fixed transform is visible.
    # Rejoin it only after measurements so the candidate GLB hierarchy stays closed.
    merged_review = join_fragments(fragments["hinge-review"], "LaptopHingeReviewGeometry")
    preserve_parent_world(merged_review, review_root)
    fragments["hinge-review"] = [merged_review]
    remove_source_face_attribution(fragments)
    # Put the pose-fit pivot back at source-open before publishing the starter.
    pivot.rotation_euler.x = 0.0
    bpy.context.view_layer.update()
    data, output_doc, screen_bytes, total = export_candidate(root, cfg)
    glb_path = OUTPUTS / cfg["output"]["glb_name"]
    output_records[glb_path.name] = {"sha256": digest(data), "size_bytes": len(data)}
    role_inventory = {
        "profile_id": cfg["profile_id"], "status": "derived_candidate_unreviewed",
        "source_sha256": cfg["source"]["sha256"], "source_size_bytes": cfg["source"]["size_bytes"],
        "component_inventory_sha256": cfg["component_inventory"]["sha256"],
        "manifest_sha256": manifest_sha, "script_sha256": digest(Path(__file__).read_bytes()),
        "blender_version": bpy.app.version_string, "total_source_triangles": total,
        "triangles_by_candidate_role": counts, "mesh_node_count": sum(1 for node in output_doc["nodes"] if "mesh" in node),
        "source_node_role_triangles": source_role_inventory,
        "role_assignment": {**cfg["role_assignment"], "source_open_geometry_invariance": source_open_geometry},
        "hierarchy": {"root": "LaptopStarterRoot", "base": "LaptopBaseCandidate", "lid_pivot": "LaptopLidPivotCandidate", "lid_geometry": "LaptopLidGeometry", "lid_screen_node": "LaptopScreenSurface", "unresolved_fixed": "LaptopHingeReview"},
        "pose_fit_axis_glb_m": closed_fit["fitted_axis_glb_m"], "closed_pose_fit": closed_fit,
        "hinge_trim_alignment": trim_alignment,
        "closed_local_surface_witnesses": local_witnesses,
        "component_diagnostic_bindings": component_bindings,
        "source_face_attribution": {"method": "temporary per-face source node/component integer attributes retained through role joins for diagnostic sample attribution",
                                    "temporary_attributes_removed_before_candidate_export": True},
        "screen_binding": cfg["source"]["screen_binding"],
        "neutral_screen": {**neutral_facts, "derived_png_sha256": digest(screen_bytes), "claim": "uniform neutral screen; not the original browser capture"},
        "embedded_source_images": cfg["source"]["embedded_images"], "candidate_hierarchy_names": [node.get("name") for node in output_doc["nodes"]],
        "poses": pose_records,
        "world_background": {"linear_rgba": cfg["render"]["world_background_linear_rgba"], "strength": cfg["render"]["world_background_strength"]},
        "limits": "Screen/deck-plane visual fit only. The exact 788-triangle near-screen-plane cluster follows the lid as a source-backed visual ownership hypothesis; Object_42/12 and residual hinge-review geometry stay fixed and unresolved. Lid-geometry global minimum (the display mesh is reported separately) and nearest-face signed local samples are diagnostics; they do not establish continuous clearance. Candidate overlap counts are discrete BVH surface-pair observations, not a continuous swept-clearance proof, measured mechanism, or manufacturer validation.",
        "outputs": output_records,
    }
    png_bytes = sum(record["size_bytes"] for name, record in output_records.items() if name.endswith(".png"))
    require(png_bytes <= cfg["output"]["max_png_aggregate_bytes"], "fixed diagnostic PNG aggregate exceeds manifest bound")
    receipt_path = OUTPUTS / cfg["output"]["receipt_name"]
    receipt_path.write_text(json.dumps(role_inventory, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        persist_diagnostic(ROOT / "failure-diagnostic.json", {"failure_type": type(error).__name__,
                           "failure": str(error)[:512], "traceback": traceback.format_exc(limit=4)[:3072]})
        raise
