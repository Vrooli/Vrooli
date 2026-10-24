"""Fixed iPad-only cup refinement over the pinned full study environment."""
import hashlib
import importlib.util
import json
import math
import struct
from pathlib import Path

import bpy
from mathutils import Matrix, Quaternion, Vector


ROOT = Path(__file__).resolve().parent
INPUTS = ROOT / "inputs"
OUTPUTS = ROOT / "outputs"
MANIFEST = ROOT / "ipad-scene-manifest.json"
HELPER_SHA = "2a36b813c050497524a75d170deb53f4d23fb8d3c39054f3d5836a34abfa6659"
INVENTORY_SHA = "1cadc29e06491aae213963d731b7edcc7145ebbb54072baef48522b2ce7564b9"
MATH_HELPER_SHA = "c45a8996b631d51d48053ae345a01b8a668db68722fa62ce5b96db7a772610d2"
TABLE_MANIFEST_SHA = "a5587b445c029211dfece07a71709bb94c4287171526318fa8022d78170d86cf"
ENVIRONMENT_MANIFEST_SHA = "bc89ac5b69ef6cd58ef788072eee89fa9c12598c7f76664a838d25623d6617ea"
ENVIRONMENT_SHA = "c0f1450af8cc83b237159f2c5208bd964c48fc5dc9fe7f2ac2b2ee5409447f4d"
ENVIRONMENT_BYTES = 421404
EXPECTED_NODES = {
    "TableLeg-NE", "TableLeg-NW", "TableLeg-SE", "TableLeg-SW", "TableTop",
    "TeaCup", "TeaCupHandle", "TeaSurface", "RoomBackWall", "WindowLightPanel",
    "WindowFrameUprightLeft", "WindowFrameUprightRight", "WindowFrameRailLower",
    "WindowFrameRailUpper",
}
math_helper = None


def require(ok, message):
    if not ok:
        raise RuntimeError(message)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def manifest_vector3(value):
    if isinstance(value, dict):
        require(set(value) == {"x", "y", "z"}, "3D manifest vector must contain exactly x, y and z")
        value = (value["x"], value["y"], value["z"])
    require(isinstance(value, (list, tuple)) and len(value) == 3 and
            all(type(component) in (int, float) and math.isfinite(component) for component in value),
            "3D manifest vector must contain three finite numbers")
    return Vector(tuple(float(component) for component in value))


def manifest_quaternion(value):
    if isinstance(value, dict):
        require(set(value) == {"x", "y", "z", "w"}, "manifest quaternion must contain exactly x, y, z and w")
        xyzw = (value["x"], value["y"], value["z"], value["w"])
    else:
        xyzw = value
    require(isinstance(xyzw, (list, tuple)) and len(xyzw) == 4 and
            all(type(component) in (int, float) and math.isfinite(component) for component in xyzw),
            "manifest quaternion must contain four finite XYZW numbers")
    x, y, z, w = (float(component) for component in xyzw)
    require(abs(math.sqrt(x * x + y * y + z * z + w * w) - 1.0) <= 1e-5,
            "manifest quaternion must be normalized")
    return Quaternion((w, x, y, z))


def read_regular(path, maximum):
    require(path.is_file() and not path.is_symlink(), "fixed input must be a regular file: " + path.name)
    size = path.stat().st_size
    require(0 < size <= maximum, "fixed input exceeds its size bound: " + path.name)
    data = path.read_bytes()
    require(len(data) == size, "input changed while reading: " + path.name)
    return data


def glb(data):
    require(len(data) >= 20 and data[:4] == b"glTF", "input is not a bounded GLB")
    magic, version, declared = struct.unpack_from("<4sII", data, 0)
    require(version == 2 and declared == len(data), "GLB header differs")
    offset, chunks = 12, []
    while offset < len(data):
        require(offset + 8 <= len(data), "truncated GLB chunk")
        size, kind = struct.unpack_from("<II", data, offset)
        offset += 8
        require(size <= len(data) - offset, "GLB chunk exceeds bounds")
        chunks.append((kind, data[offset:offset + size]))
        offset += size
    require(len(chunks) == 2 and chunks[0][0] == 0x4E4F534A and chunks[1][0] == 0x004E4942,
            "GLB must have exactly JSON and BIN chunks")
    return json.loads(chunks[0][1].decode("utf-8").rstrip(" \t\r\n\0")), chunks[1][1]


def trs_matrix(node):
    require("matrix" not in node, "only decomposed glTF TRS is supported")
    t = Vector(node.get("translation", (0, 0, 0)))
    q = node.get("rotation", (0, 0, 0, 1))
    r = Quaternion((q[3], q[0], q[1], q[2]))
    s = Vector(node.get("scale", (1, 1, 1)))
    return Matrix.Translation(t) @ r.to_matrix().to_4x4() @ Matrix.Diagonal((s.x, s.y, s.z, 1))


def accessor_vec3(doc, binary, index):
    accessor = doc["accessors"][index]
    require(accessor["componentType"] == 5126 and accessor["type"] == "VEC3", "positions must be float VEC3")
    view = doc["bufferViews"][accessor["bufferView"]]
    stride = view.get("byteStride", 12)
    offset = view.get("byteOffset", 0) + accessor.get("byteOffset", 0)
    count = accessor["count"]
    require(count <= 200000 and offset + (count - 1) * stride + 12 <= len(binary), "position accessor exceeds bounds")
    return [Vector(struct.unpack_from("<3f", binary, offset + i * stride)) for i in range(count)]


def device_points(data, placement):
    doc, binary = glb(data)
    nodes = doc["nodes"]
    parents = [None] * len(nodes)
    for parent, node in enumerate(nodes):
        for child in node.get("children", []):
            require(child < len(nodes) and parents[child] is None, "device hierarchy is invalid")
            parents[child] = parent
    world = {}

    def visit(index, parent_matrix):
        matrix = parent_matrix @ trs_matrix(nodes[index])
        world[index] = matrix
        for child in nodes[index].get("children", []):
            visit(child, matrix)

    for root in doc["scenes"][0]["nodes"]:
        visit(root, Matrix.Identity(4))
    support = Matrix.Translation(manifest_vector3(placement["scene_support_world"]["translation_m"]))
    pivot = Matrix.Translation(manifest_vector3(placement["support_pivot_m"]))
    rest = placement["device_root_rest"]
    root = Matrix.Translation(manifest_vector3(rest["translation_m"])) @ manifest_quaternion(rest["rotation_xyzw"]).to_matrix().to_4x4()
    canonical = placement["source_to_canonical"]
    source = manifest_quaternion(canonical["rotation_xyzw"]).to_matrix().to_4x4()
    scale = canonical["scale"]
    source @= Matrix.Diagonal((scale["x"], scale["y"], scale["z"], 1))
    candidate = support @ pivot @ root @ source
    points, display_points = [], []
    display_index = placement["display_node_index"]
    for index, node in enumerate(nodes):
        if "mesh" not in node:
            continue
        for primitive in doc["meshes"][node["mesh"]]["primitives"]:
            local_points = accessor_vec3(doc, binary, primitive["attributes"]["POSITION"])
            transformed = [candidate @ world[index] @ point.to_4d() for point in local_points]
            points.extend(Vector(point[:3]) for point in transformed)
            if index == display_index:
                display_points.extend(Vector(point[:3]) for point in transformed)
    require(points and display_points, "pinned iPad geometry or display vertices are missing")
    return points, display_points


def camera_project(points, spec):
    camera = Matrix.Translation(manifest_vector3(spec["translation_m"])) @ manifest_quaternion(spec["rotation_xyzw"]).to_matrix().to_4x4()
    inverse = camera.inverted()
    return [math_helper.project_camera_local(tuple(inverse @ point.to_4d())[:3],
                                             spec["focal_length_mm"],
                                             spec["sensor_width_mm"], spec["aspect"])
            for point in points]


def bounds2(points):
    return (min(p[0] for p in points), min(p[1] for p in points),
            max(p[0] for p in points), max(p[1] for p in points))


def projected_gap(a, b):
    return max(a[0] - b[2], b[0] - a[2], a[1] - b[3], b[1] - a[3])


def blender_world_points(obj):
    # Blender Z-up to this GLB profile's canonical X-right/Y-up/Z-front frame.
    return [Vector((*math_helper.blender_to_canonical(tuple(obj.matrix_world @ vertex.co.to_4d())[:3]), 1))
            for vertex in obj.data.vertices]


def blender_xyz_points(obj):
    return [obj.matrix_world @ vertex.co.to_4d() for vertex in obj.data.vertices]


def cup_target(manifest, geometry, device_glb):
    projection = manifest["camera_projection"]
    placement = projection["device_placement"]
    device, display = device_points(device_glb, placement)
    device_box = bounds2(camera_project(device, projection["camera"]))
    display_box = bounds2(camera_project(display, projection["camera"]))
    frame_margin = projection["image_margin_normalized"]
    require(math_helper.bounds_within_frame(device_box, frame_margin),
            "complete tablet silhouette exceeds the framing margin")
    objects = [bpy.data.objects.get(name) for name in ("TeaCup", "TeaCupHandle", "TeaSurface")]
    require(all(obj is not None and obj.type == "MESH" for obj in objects), "cup group is incomplete")
    original = geometry["cup_center_meters_blender"]
    current_points = [point for obj in objects for point in blender_xyz_points(obj)]
    mins = [min(point[i] for point in current_points) for i in range(3)]
    maxs = [max(point[i] for point in current_points) for i in range(3)]
    corners = [Vector((x, y, z, 1)) for x in (mins[0], maxs[0])
               for y in (mins[1], maxs[1]) for z in (mins[2], maxs[2])]
    table = geometry
    tabletop = projection["tabletop_blender_xy_bounds_meters"]
    edge = tabletop["edge_margin"]
    x_min, y_min, x_max, y_max = math_helper.table_bounds_from_dimensions(
        table["table_top_center_meters_blender"][:2],
        table["table_top_dimensions_meters"][:2], edge)
    declared = (tabletop["x_min"], tabletop["y_min"], tabletop["x_max"], tabletop["y_max"])
    require(all(abs(actual - expected) <= 1e-9 for actual, expected in zip(
        (x_min - edge, y_min - edge, x_max + edge, y_max + edge), declared)),
        "declared tabletop bounds differ from the pinned table geometry")
    search = projection["placement_search_blender_xy_meters"]
    math_helper.validate_search(search)
    step = search["step"]
    gap_required = projection["minimum_gap_from_device_normalized"]
    best = None
    x = search["x_min"]
    while x <= search["x_max"] + step / 2:
        y = search["y_min"]
        while y <= search["y_max"] + step / 2:
            dx, dy = x - original[0], y - original[1]
            if all(x_min <= corner.x + dx <= x_max and y_min <= corner.y + dy <= y_max
                   for corner in corners):
                canonical_delta = math_helper.canonical_delta_from_blender_xy(dx, dy)
                shifted = [Vector((p.x + canonical_delta[0], p.y + canonical_delta[1],
                                   p.z + canonical_delta[2], 1)) for p in
                           (Vector((*math_helper.blender_to_canonical(tuple(corner)[:3]), 1)) for corner in corners)]
                box = bounds2(camera_project(shifted, projection["camera"]))
                if math_helper.bounds_within_frame(box, frame_margin):
                    gap = projected_gap(box, device_box)
                    if gap >= gap_required:
                        distance = (dx * dx + dy * dy)
                        score = (distance, -gap)
                        if best is None or score < best[0]:
                            best = (score, (x, y), box, gap)
            y += step
        x += step
    require(best is not None, "no cup placement clears the full iPad silhouette with the required projected margin")
    return objects, original, best[1], device_box, display_box, best[2], best[3]


def smooth_cup_group(objects, geometry):
    cup = bpy.data.objects["TeaCup"]
    # Smooth only the circumferential vessel surfaces. Preserve sharp floor,
    # underside, and lip transitions; the authored rim bevel remains in place.
    segments = geometry["cup_ring_segments"]
    profile_count = len(geometry["cup_profile_rings_radius_height_meters"])
    faces_per_ring = segments * 2
    face_limit = (profile_count - 1) * faces_per_ring
    require(len(cup.data.polygons) >= face_limit, "cup triangulation census is too small")
    for polygon in cup.data.polygons:
        polygon.use_smooth = False
        if segments * 2 <= polygon.index < 5 * segments * 2:
            polygon.use_smooth = True
    if not hasattr(cup.data, "set_sharp_from_angle"):
        raise RuntimeError("pinned Blender mesh sharp-edge API is unavailable")
    cup.data.set_sharp_from_angle(angle=math.radians(35))
    handle = bpy.data.objects["TeaCupHandle"]
    for polygon in handle.data.polygons:
        polygon.use_smooth = True
    handle.data.set_sharp_from_angle(angle=math.radians(35))
    surface = bpy.data.objects["TeaSurface"]
    for polygon in surface.data.polygons:
        polygon.use_smooth = abs(polygon.normal.z) < 0.5
    surface.data.set_sharp_from_angle(angle=math.radians(35))


def main():
    require(MANIFEST.is_file() and not MANIFEST.is_symlink(), "manifest must be a regular staged file")
    manifest_bytes = read_regular(MANIFEST, 128 * 1024)
    manifest = json.loads(manifest_bytes.decode("utf-8"))
    require(manifest["profile_id"] == "device-studio-ipad-study-environment-export-v2", "wrong iPad scene profile")
    require(list(bpy.app.version[:3]) == manifest["blender_version"], "pinned Blender version differs")
    require(set(manifest) == {"profile_id", "blender_version", "coordinate_convention", "input", "cup",
                            "camera_projection", "expected_output", "scene_scope", "candidate_identity"},
            "fixed scene manifest fields differ")
    projection = manifest["camera_projection"]
    camera_spec = projection["camera"]
    manifest_vector3(camera_spec["translation_m"])
    manifest_quaternion(camera_spec["rotation_xyzw"])
    require(all(type(camera_spec[key]) in (int, float) and math.isfinite(camera_spec[key]) and camera_spec[key] > 0
                for key in ("focal_length_mm", "sensor_width_mm", "aspect")), "camera optical values are invalid")
    require(all(type(projection[key]) in (int, float) and math.isfinite(projection[key]) and 0 <= projection[key] < 0.5
                for key in ("image_margin_normalized", "minimum_gap_from_device_normalized")),
            "projection margin or clearance is invalid")
    require(manifest["cup"]["radial_segments"] == 128, "fixed cup tessellation differs")
    require(manifest["expected_output"]["glb_name"] == "ipad-study-environment.glb" and
            manifest["expected_output"]["inventory_name"] == "ipad-environment-inventory.json" and
            0 < manifest["expected_output"]["max_bytes"] <= 32 * 1024 * 1024,
            "fixed output contract differs")
    source_spec = manifest["input"]["scene_glb"]
    source = read_regular(INPUTS / source_spec["staged_name"], 2 * 1024 * 1024)
    require(len(source) == source_spec["size_bytes"] == ENVIRONMENT_BYTES and digest(source) == source_spec["sha256"] == ENVIRONMENT_SHA,
            "phone-study environment source identity differs")
    device_spec = manifest["input"]["device_glb"]
    device = read_regular(INPUTS / device_spec["staged_name"], 8 * 1024 * 1024)
    require(len(device) == device_spec["size_bytes"] and digest(device) == device_spec["sha256"], "iPad screen-fit source identity differs")
    helper_spec = manifest["input"]["cup_generator"]
    helper_bytes = read_regular(ROOT / helper_spec["staged_name"], 1 * 1024 * 1024)
    require(digest(helper_bytes) == helper_spec["sha256"] == HELPER_SHA, "shared cup geometry helper differs")
    helper_path = ROOT / helper_spec["staged_name"]
    spec = importlib.util.spec_from_file_location("pinned_phone_table_export", helper_path)
    require(spec is not None and spec.loader is not None, "cannot load the pinned cup generator helper")
    helper = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(helper)
    math_spec = manifest["input"]["math_helper"]
    math_bytes = read_regular(ROOT / math_spec["staged_name"], 128 * 1024)
    require(digest(math_bytes) == math_spec["sha256"] == MATH_HELPER_SHA,
            "iPad coordinate helper differs")
    math_path = ROOT / math_spec["staged_name"]
    math_loader = importlib.util.spec_from_file_location("pinned_ipad_environment_math", math_path)
    require(math_loader is not None and math_loader.loader is not None, "cannot load iPad coordinate helper")
    global math_helper
    math_helper = importlib.util.module_from_spec(math_loader)
    math_loader.loader.exec_module(math_helper)
    inventory_spec = manifest["input"]["inventory_generator"]
    inventory_bytes = read_regular(ROOT / inventory_spec["staged_name"], 1 * 1024 * 1024)
    require(digest(inventory_bytes) == inventory_spec["sha256"] == INVENTORY_SHA, "shared scene inventory helper differs")
    inventory_path = ROOT / inventory_spec["staged_name"]
    inventory_module = importlib.util.spec_from_file_location("pinned_phone_environment_export", inventory_path)
    require(inventory_module is not None and inventory_module.loader is not None, "cannot load the pinned inventory helper")
    inventory_helper = importlib.util.module_from_spec(inventory_module)
    inventory_module.loader.exec_module(inventory_helper)
    table_spec = manifest["input"]["base_table_manifest"]
    table_manifest_bytes = read_regular(ROOT / table_spec["staged_name"], 128 * 1024)
    require(digest(table_manifest_bytes) == table_spec["sha256"] == TABLE_MANIFEST_SHA, "shared cup geometry manifest differs")
    table_manifest = json.loads(table_manifest_bytes.decode("utf-8"))
    geometry = table_manifest["geometry"]
    geometry["cup_ring_segments"] = manifest["cup"]["radial_segments"]
    environment_spec = manifest["input"]["base_environment_manifest"]
    environment_manifest_bytes = read_regular(ROOT / environment_spec["staged_name"], 128 * 1024)
    require(digest(environment_manifest_bytes) == environment_spec["sha256"] == ENVIRONMENT_MANIFEST_SHA, "shared environment manifest differs")
    inventory_manifest = json.loads(environment_manifest_bytes.decode("utf-8"))
    inventory_manifest["profile_id"] = manifest["profile_id"]
    inventory_manifest["coordinate_convention"] = manifest["coordinate_convention"]
    inventory_manifest["input"]["scene_glb"] = source_spec
    inventory_manifest["expected_output"] = manifest["expected_output"]
    require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed output directory already exists")
    OUTPUTS.mkdir()
    bpy.ops.wm.read_factory_settings(use_empty=True)
    bpy.context.scene.unit_settings.system = "METRIC"
    bpy.context.scene.unit_settings.scale_length = 1.0
    source_path = INPUTS / source_spec["staged_name"]
    bpy.ops.import_scene.gltf(filepath=str(source_path))
    imported = {obj.name for obj in bpy.context.scene.objects if obj.type == "MESH"}
    source_nodes = set(source_spec["expected_mesh_nodes"])
    output_nodes = set(manifest["expected_output"]["expected_mesh_nodes"])
    require(len(source_nodes) == len(source_spec["expected_mesh_nodes"]) and
            len(output_nodes) == len(manifest["expected_output"]["expected_mesh_nodes"]) and
            imported == source_nodes == output_nodes, "source/output scene node set differs")
    for name in ("TeaCup", "TeaCupHandle", "TeaSurface"):
        obj = bpy.data.objects.get(name)
        require(obj is not None and obj.type == "MESH", "source cup part is missing: " + name)
        bpy.data.objects.remove(obj, do_unlink=True)
    ceramic = bpy.data.materials.get("Warm ivory glazed ceramic")
    tea = bpy.data.materials.get("Dark tea scale cue")
    require(ceramic is not None and tea is not None, "authored cup material slots are missing")
    projection = manifest["camera_projection"]
    table_manifest["geometry"] = geometry
    helper.make_cup(table_manifest, ceramic, tea)
    objects = [bpy.data.objects[name] for name in ("TeaCup", "TeaCupHandle", "TeaSurface")]
    smooth_cup_group(objects, geometry)
    bpy.context.view_layer.update()
    objects, original, target, device_box, display_box, cup_box, gap = cup_target(manifest, geometry, device)
    delta = (target[0] - original[0], target[1] - original[1])
    for obj in objects:
        obj.location.x += delta[0]
        obj.location.y += delta[1]
    bpy.context.view_layer.update()
    actual_points = [point for obj in objects for point in blender_world_points(obj)]
    actual_box = bounds2(camera_project(actual_points, projection["camera"]))
    require(math_helper.bounds_within_frame(device_box, projection["image_margin_normalized"]),
            "complete tablet silhouette exceeds the framing margin")
    require(math_helper.bounds_within_frame(actual_box, projection["image_margin_normalized"]),
            "actual regenerated cup group exceeds the framing margin")
    actual_gap = projected_gap(actual_box, device_box)
    require(actual_gap >= projection["minimum_gap_from_device_normalized"], "actual cup group intrudes into the tablet projection")
    expected = set(manifest["expected_output"]["expected_mesh_nodes"])
    require({obj.name for obj in bpy.context.scene.objects if obj.type == "MESH"} == expected, "new scene mesh set differs")
    bpy.ops.object.select_all(action="DESELECT")
    for obj in bpy.context.scene.objects:
        if obj.type == "MESH":
            obj.select_set(True)
    output_path = OUTPUTS / manifest["expected_output"]["glb_name"]
    bpy.ops.export_scene.gltf(filepath=str(output_path), export_format="GLB", use_selection=True,
                              export_yup=True, export_apply=True, export_cameras=False,
                              export_lights=False, export_materials="EXPORT", export_extras=True,
                              export_texcoords=True, export_normals=True, export_animations=False)
    require(output_path.is_file() and not output_path.is_symlink() and output_path.stat().st_size <= manifest["expected_output"]["max_bytes"],
            "fixed scene output is absent or oversized")
    output = read_regular(output_path, manifest["expected_output"]["max_bytes"])
    doc, _ = glb(output)
    inventory = inventory_helper.build_inventory(doc, inventory_manifest, digest(manifest_bytes), len(source), output)
    inventory["profile_id"] = manifest["profile_id"]
    inventory["export_script_sha256"] = digest(read_regular(ROOT / "export_ipad_environment.py", 1 * 1024 * 1024))
    inventory["source_scene_glb_sha256"] = digest(source)
    inventory["source_scene_glb_bytes"] = len(source)
    inventory["projection_device_glb_sha256"] = digest(device)
    inventory["projection_device_glb_bytes"] = len(device)
    inventory.pop("source_table_glb_sha256", None)
    inventory.pop("source_table_glb_bytes", None)
    inventory["scene_scope"] = manifest["scene_scope"]
    inventory["candidate_identity"] = manifest["candidate_identity"]
    inventory["cup_refinement"] = {
        "shared_generator_sha256": digest(helper_bytes),
        "shared_inventory_helper_sha256": digest(inventory_bytes),
        "base_table_manifest_sha256": digest(table_manifest_bytes),
        "base_environment_manifest_sha256": digest(environment_manifest_bytes),
        "source_center_meters_blender_xy": original,
        "projected_center_meters_blender_xy": list(target),
        "placement_delta_meters_blender_xy": list(delta),
        "radial_segments": geometry["cup_ring_segments"],
        "smooth_surfaces": ["inner wall", "outer wall", "handle torus"],
        "hard_edge_angle_degrees": 35,
        "projected_tablet_bounds_normalized": list(device_box),
        "projected_display_bounds_normalized": list(display_box),
        "projected_cup_bounds_normalized": list(actual_box),
        "projected_clearance_from_full_tablet": actual_gap,
        "minimum_required_clearance": projection["minimum_gap_from_device_normalized"],
        "frame_margin": projection["image_margin_normalized"],
        "geometry_status": "authored_scene_candidate_unqualified",
    }
    (OUTPUTS / manifest["expected_output"]["inventory_name"]).write_text(json.dumps(inventory, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
