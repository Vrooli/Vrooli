import hashlib
import importlib.util
import json
import os
import sys

import bpy
import math
from mathutils import Matrix, Vector
from mathutils.bvhtree import BVHTree
from bpy_extras.object_utils import world_to_camera_view


def digest(path):
    h = hashlib.sha256()
    with open(path, "rb") as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def mesh_snapshot(obj, depsgraph):
    evaluated = obj.evaluated_get(depsgraph)
    mesh = evaluated.to_mesh()
    mesh.calc_loop_triangles()
    vertices = [evaluated.matrix_world @ v.co for v in mesh.vertices]
    triangles = [tuple(t.vertices) for t in mesh.loop_triangles]
    tree = BVHTree.FromPolygons(vertices, triangles, all_triangles=True)
    evaluated.to_mesh_clear()
    return vertices, triangles, tree


def bounds(points):
    return [[min(p[i] for p in points), max(p[i] for p in points)] for i in range(3)]


def boxes_overlap(a, b):
    return all(a[i][0] <= b[i][1] and b[i][0] <= a[i][1] for i in range(3))


def analytic_swept_bounds(vertices, pivot_world, upper):
    inverse = pivot_world.inverted()
    out = []
    for world in vertices:
        local = inverse @ world
        candidates = [0.0, upper]
        for k in range(-2, 4):
            for angle in (math.atan2(-local.z, local.y) + k * math.pi,
                          math.atan2(local.y, local.z) + k * math.pi):
                if 0.0 < angle < upper:
                    candidates.append(angle)
        for angle in candidates:
            rotated = Matrix.Rotation(angle, 4, "X") @ local.to_4d()
            out.append((pivot_world @ rotated).to_3d())
    return bounds(out)


def sampled_clearance(adapter, job, device, device_list, part_by_id,
                      device_root, camera, camera_spec, hinge):
    fixed_ids = ("laptop-base-geometry", "laptop-hinge-review-geometry")
    moving_ids = ("laptop-lid-geometry", "laptop-screen-surface")
    owners = {}
    objects = {}
    for part_id in fixed_ids + moving_ids:
        part = part_by_id[part_id]
        obj = device.get(adapter.alias("device", part["node_index"]))
        if obj is None or obj.type != "MESH":
            raise RuntimeError("required clearance component missing: " + part_id)
        owners[part_id] = {"part_id": part_id, "node_index": part["node_index"],
                           "object": obj.name, "motion": "moving_with_lid" if part_id in moving_ids else
                           ("fixed_base" if part_id == fixed_ids[0] else "unresolved_review_fixed_for_diagnostic")}
        objects[part_id] = obj

    sample0 = dict(job["output"]["samples"][0])
    sample0["joint_angles"] = {"laptop_lid_hinge": 0.0}
    adapter.apply_motion_sample(device_root, camera, camera_spec, job["placement"],
                                job["device"], sample0, hinge)
    depsgraph = bpy.context.evaluated_depsgraph_get()
    snapshots = {pid: mesh_snapshot(objects[pid], depsgraph) for pid in fixed_ids + moving_ids}
    for part_id, snapshot in snapshots.items():
        owners[part_id]["triangle_count"] = len(snapshot[1])
    pivot = device.get(adapter.alias("device", part_by_id["laptop-lid-pivot"]["node_index"]))
    pivot_world = pivot.matrix_world.copy()
    swept = {pid: analytic_swept_bounds(snapshots[pid][0], pivot_world, 1.919) for pid in moving_ids}
    fixed_bounds = {pid: bounds(snapshots[pid][0]) for pid in fixed_ids}
    broadphase = [{"moving_part_id": moving, "fixed_part_id": fixed,
                   "moving_triangle_count": len(snapshots[moving][1]),
                   "fixed_triangle_count": len(snapshots[fixed][1]),
                   "moving_swept_aabb_m": swept[moving], "fixed_aabb_m": fixed_bounds[fixed],
                   "candidate_pair": boxes_overlap(swept[moving], fixed_bounds[fixed])}
                  for moving in moving_ids for fixed in fixed_ids]

    measurements = []
    sample_count = 49
    for ordinal in range(sample_count):
        angle = 1.919 * ordinal / (sample_count - 1)
        sample = dict(job["output"]["samples"][0])
        sample["joint_angles"] = {"laptop_lid_hinge": angle}
        sample["sampled_motion_tick"] = angle
        adapter.apply_motion_sample(device_root, camera, camera_spec, job["placement"],
                                    job["device"], sample, hinge)
        depsgraph = bpy.context.evaluated_depsgraph_get()
        for moving in moving_ids:
            moving_vertices, moving_triangles, moving_tree = mesh_snapshot(objects[moving], depsgraph)
            for fixed in fixed_ids:
                if not boxes_overlap(swept[moving], fixed_bounds[fixed]):
                    continue
                fixed_vertices, fixed_triangles, fixed_tree = snapshots[fixed]
                pairs = moving_tree.overlap(fixed_tree)
                best = (float("inf"), None, None)
                for point in moving_vertices:
                    nearest = fixed_tree.find_nearest(point)
                    if nearest and nearest[3] < best[0]:
                        best = (nearest[3], None, nearest[2])
                for point in fixed_vertices:
                    nearest = moving_tree.find_nearest(point)
                    if nearest and nearest[3] < best[0]:
                        best = (nearest[3], nearest[2], None)
                measurements.append({"angle_rad": angle, "moving_part_id": moving,
                    "fixed_part_id": fixed, "bvh_intersection_pair_count": len(pairs),
                    "bvh_intersection_triangle_pairs": [{"moving_triangle_id": a, "fixed_triangle_id": b}
                        for a, b in pairs[:32]],
                    "intersection_pairs_truncated": len(pairs) > 32,
                    "sampled_raw_vertex_to_opposite_surface_separation_m": best[0] if best[0] != float("inf") else None,
                    "nearest_moving_triangle_id": best[1], "nearest_fixed_triangle_id": best[2]})
    return {"angle_range_rad": [0.0, 1.919], "angular_samples": sample_count,
            "triangle_ownership": owners, "analytic_swept_aabb_broadphase": broadphase,
            "sampled_bvh_measurements": measurements,
            "numerical_oracle_tolerance_m": 2e-6,
            "claim_limit": "Bounded angular samples and analytic swept AABBs are diagnostic evidence; they do not prove continuous clearance or mechanical qualification."}


def main():
    root = os.path.dirname(os.path.abspath(sys.argv[sys.argv.index("--") + 1]))
    with open(os.path.join(root, "physical-semantic-probe-manifest.json"), encoding="utf-8") as f:
        m = json.load(f)
    pinned_inputs = (m["worker_input"], m["device_glb"], m["scene_glb"], m["screen_png"])
    for item in pinned_inputs + tuple(m.get("scene_textures", [])):
        path = os.path.join(root, item["name"])
        if os.path.getsize(path) != item["size_bytes"] or digest(path) != item["sha256"]:
            raise RuntimeError("archived input identity changed: " + item["name"])
    if digest(os.path.join(root, m["adapter_name"])) != m["adapter_sha256"]:
        raise RuntimeError("adapter identity changed")
    if digest(__file__) != m["probe_sha256"]:
        raise RuntimeError("diagnostic source identity changed")
    spec = importlib.util.spec_from_file_location("physical_adapter", os.path.join(root, m["adapter_name"]))
    adapter = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(adapter)
    with open(os.path.join(root, m["worker_input"]["name"]), encoding="utf-8") as f:
        job = json.load(f)
    bpy.ops.wm.read_factory_settings(use_empty=True)
    device_list, device = adapter.import_glb(root, "device-normalized.glb", "device")
    scene_list, scene_objects = adapter.import_glb(root, "scene-normalized.glb", "scene")
    scene = bpy.context.scene
    imported_world = {obj.name: obj.matrix_world.copy() for obj in device_list}
    device_root = adapter.place_device(device_list, job["device"], job, device)
    pre_attach_world = {obj.name: obj.matrix_world.copy() for obj in device_list}
    hinge = adapter.attach_physical_hinge(job["device"], job["motion"], device)
    for artifact in job["scene"]["artifacts"]:
        if artifact["ref"]["role"] == "geometry_glb":
            unit = {"m": 1.0, "cm": .01, "mm": .001}[artifact["source_unit"]]
            c = adapter.canonical_to_blender()
            source = c @ adapter.trs(artifact["source_to_canonical"]) @ adapter.Matrix.Scale(unit, 4) @ c.inverted()
            adapter.wrap_roots("probe-scene-source", scene_list, source)
            break
    adapter.bind_scene_materials(root, job, job["scene"], scene_objects)
    adapter.bind_screen(root, job, job["screen_geometry"], device)
    adapter.build_world(scene, job["scene"]["environment"])
    adapter.build_lights(scene, job["scene"]["light_frames"])
    scene.render.resolution_x = job["output"]["width"]
    scene.render.resolution_y = job["output"]["height"]
    scene.render.resolution_percentage = 100
    camera_spec = next(x for x in job["scene"]["camera_frames"] if x["id"] == job["placement"]["selected_camera_id"])
    camera = adapter.setup_camera(scene, camera_spec, device_root, job["scene"], scene_objects)
    scene.camera = camera
    parts = {x["id"]: x for x in job["device"]["parts"]}
    dep = bpy.context.evaluated_depsgraph_get()
    frames = []
    rest_error = 0.0
    hierarchy = []
    frame_zero_world = {}
    for ordinal in (0, 24, 47):
        sample = job["output"]["samples"][ordinal]
        adapter.apply_motion_sample(device_root, camera, camera_spec, job["placement"], job["device"], sample, hinge)
        dep = bpy.context.evaluated_depsgraph_get()
        if ordinal == 0:
            for obj in device_list:
                ev = obj.evaluated_get(dep)
                expected = device_root.matrix_world @ imported_world[obj.name]
                error = max(abs(ev.matrix_world[r][c] - expected[r][c]) for r in range(4) for c in range(4))
                hierarchy.append({"object": obj.name, "type": obj.type,
                    "parent": obj.parent.name if obj.parent else "",
                    "imported_world": [[float(x) for x in row] for row in imported_world[obj.name]],
                    "pre_attach_world": [[float(x) for x in row] for row in pre_attach_world[obj.name]],
                    "placed_expected_world": [[float(x) for x in row] for row in expected],
                    "evaluated_world": [[float(x) for x in row] for row in ev.matrix_world],
                    "matrix_error": error})
                rest_error = max(rest_error, error)
                frame_zero_world[obj.name] = ev.matrix_world.copy()
        entries = []
        for pid, part in parts.items():
            obj = device.get(adapter.alias("device", part["node_index"]))
            if obj is None or obj.type != "MESH":
                continue
            ev = obj.evaluated_get(dep)
            corners = [obj.matrix_world @ Vector(c) for c in obj.bound_box]
            projected = [world_to_camera_view(scene, camera, p) for p in corners]
            center = sum(corners, Vector()) / len(corners)
            direction = center - camera.matrix_world.translation
            hit, loc, normal, face, hit_obj, hit_matrix = scene.ray_cast(dep, camera.matrix_world.translation, direction.normalized())
            distance = (loc - camera.matrix_world.translation).length if hit else None
            materials = [slot.material.name if slot.material else "" for slot in obj.material_slots]
            entries.append({"part_id": pid, "node_index": part["node_index"], "object": obj.name,
                "world_bounds": [[min(p[i] for p in corners), max(p[i] for p in corners)] for i in range(3)],
                "camera_bounds": [[min(p[i] for p in projected), max(p[i] for p in projected)] for i in range(3)],
                "visible_get": obj.visible_get(), "hide_render": obj.hide_render,
                "hide_viewport": obj.hide_viewport, "materials": materials,
                "ray": {"hit": hit, "first_object": hit_obj.name if hit_obj else "", "distance": distance,
                        "target_object": obj.name, "target_distance": (center-camera.matrix_world.translation).length}})
        frames.append({"ordinal": ordinal, "motion_tick": sample["sampled_motion_tick"], "parts": entries})
    last_world = {obj.name: obj.matrix_world.copy() for obj in device_list}
    adapter.apply_motion_sample(device_root, camera, camera_spec, job["placement"], job["device"], job["output"]["samples"][0], hinge)
    dep = bpy.context.evaluated_depsgraph_get()
    aba_error = max(max(abs(obj.evaluated_get(dep).matrix_world[r][c] - frame_zero_world[obj.name][r][c])
                         for r in range(4) for c in range(4)) for obj in device_list)
    travel_error = max(max(abs(obj.matrix_world[r][c] - last_world[obj.name][r][c])
                           for r in range(4) for c in range(4)) for obj in device_list)
    mode = m["adapter_mode"]
    status = {"original": "diagnostic-only", "current": "corrected-adapter-measurements",
              "archive": "pilot3-archive-diagnostic"}[mode]
    report = {"version": 1, "status": status, "job_id": m["job_id"], "attempt_id": m["attempt_id"],
        "adapter_sha256": m["adapter_sha256"], "worker_input_sha256": m["worker_input"]["sha256"],
        "zero_pose_matrix_error": rest_error, "a_b_a_matrix_error": aba_error,
        "travel_matrix_delta_from_open": travel_error, "frame_zero_hierarchy": hierarchy, "frames": frames}
    out = os.path.join(root, "physical-laptop-hierarchy-diagnostics.json")
    with open(out, "x", encoding="utf-8") as f:
        json.dump(report, f, sort_keys=True, separators=(",", ":"))
        f.flush()
        os.fsync(f.fileno())
    if m["mode"] == "archived-laptop-clearance-v1":
        clearance = sampled_clearance(adapter, job, device, device_list, parts,
                                      device_root, camera, camera_spec, hinge)
        clearance["version"] = 1
        clearance["job_id"] = m["job_id"]
        clearance["attempt_id"] = m["attempt_id"]
        clearance["worker_input_sha256"] = m["worker_input"]["sha256"]
        clearance["adapter_sha256"] = m["adapter_sha256"]
        out = os.path.join(root, "physical-laptop-clearance-diagnostics.json")
        with open(out, "x", encoding="utf-8") as f:
            json.dump(clearance, f, sort_keys=True, separators=(",", ":"))
            f.flush()
            os.fsync(f.fileno())


if __name__ == "__main__":
    main()
