"""Host-side regression for the pure envelope gate in physical_blender.py."""

import ast
import hashlib
import math
import os
import pathlib
import tempfile
import unittest


WORKER = pathlib.Path(__file__).with_name("physical_blender.py")
tree = ast.parse(WORKER.read_text(encoding="utf-8"))
functions = [
    node for node in tree.body
    if isinstance(node, ast.FunctionDef) and node.name in {
        "validate_photographic_camera_envelope", "validate_photographic_render_mode",
        "source_camera_plate_path", "validate_source_plate_aspect", "triangle_within_support_plane",
        "support_plane_quad", "rotate_by_quaternion", "opencv_plane_to_world",
        "make_support_shadow_catchers"
    }
]
namespace = {"math": math, "os": os, "hashlib": hashlib,
             "fail": lambda message: (_ for _ in ()).throw(ValueError(message))}
exec(compile(ast.Module(body=functions, type_ignores=[]), str(WORKER), "exec"), namespace)
validate = namespace["validate_photographic_camera_envelope"]
validate_render_mode = namespace["validate_photographic_render_mode"]
resolve_plate = namespace["source_camera_plate_path"]
validate_plate_aspect = namespace["validate_source_plate_aspect"]
triangle_within_plane = namespace["triangle_within_support_plane"]
support_plane_quad = namespace["support_plane_quad"]
opencv_plane_to_world = namespace["opencv_plane_to_world"]
make_support_shadow_catchers = namespace["make_support_shadow_catchers"]


class IdentityTransform:
    def __matmul__(self, value):
        return value

    def inverted(self):
        return self


class MeshData:
    def __init__(self, vertices):
        self.vertices = [type("Vertex", (), {"co": point}) for point in vertices]
        self.polygons = []
        self.created_vertices = []
        self.created_faces = []

    def from_pydata(self, vertices, edges, faces):
        self.created_vertices = vertices
        self.created_faces = faces

    def update(self):
        pass


class SceneObjects:
    def __init__(self):
        self.items = []

    def link(self, value):
        self.items.append(value)


class FakeBpy:
    def __init__(self):
        self.meshes = []
        self.objects = []
        self.data = type("Data", (), {})()
        self.data.meshes = type("Meshes", (), {"new": self.new_mesh})()
        self.data.objects = type("Objects", (), {"new": self.new_object})()

    def new_mesh(self, name):
        mesh = MeshData([])
        mesh.name = name
        self.meshes.append(mesh)
        return mesh

    def new_object(self, name, mesh):
        obj = type("Object", (), {})()
        obj.name = name
        obj.data = mesh
        self.objects.append(obj)
        return obj


class PhotographicEnvelopeTests(unittest.TestCase):
    def setUp(self):
        self.scene = {"photographic_background": {
            "camera_envelope_max_translation_m": 0.02,
            "camera_envelope_max_rotation_degrees": 1.0,
            "overscan_reviewed": True,
        }}
        self.camera = {"id": "source-camera"}

    def sample(self, x=0.0, angle_degrees=0.0):
        half = math.radians(angle_degrees) / 2
        return {"camera_transforms": {"source-camera": {
            "translation_m": {"x": x, "y": 0.0, "z": 0.0},
            "rotation_xyzw": {"x": 0.0, "y": 0.0, "z": math.sin(half), "w": math.cos(half)},
        }}}

    def test_accepts_samples_within_reviewed_envelope(self):
        validate(self.scene, self.camera, [self.sample(0.01, 0.5)])

    def test_rejects_translation_and_rotation_outside_envelope(self):
        for sample in (self.sample(0.0201), self.sample(0.0, 1.01)):
            with self.subTest(sample=sample), self.assertRaisesRegex(ValueError, "exceeds"):
                validate(self.scene, self.camera, [sample])

    def test_rejects_motion_without_overscan_review_and_malformed_quaternion(self):
        unreviewed = {"photographic_background": dict(self.scene["photographic_background"], overscan_reviewed=False)}
        with self.assertRaisesRegex(ValueError, "requires reviewed overscan"):
            validate(unreviewed, self.camera, [])
        bad = self.sample()
        bad["camera_transforms"]["source-camera"]["rotation_xyzw"]["w"] = 2.0
        with self.assertRaisesRegex(ValueError, "not normalized"):
            validate(self.scene, self.camera, [bad])

    def test_nonphotographic_scene_is_unchanged(self):
        validate({}, self.camera, [self.sample(5.0, 90.0)])

    def test_source_camera_plate_allows_device_only_motion_but_rejects_camera_drift(self):
        photo = {"photographic_background": {"render_mode": "SOURCE_CAMERA_PLATE_V1"}}
        validate_render_mode(photo, self.camera, [{"camera_transforms": {}}, {"device_transform": {"x": 1}}])
        with self.assertRaisesRegex(ValueError, "rejects camera motion"):
            validate_render_mode(photo, self.camera, [self.sample(0.001)])
        with self.assertRaisesRegex(ValueError, "rejects camera motion"):
            validate_render_mode(photo, self.camera, [self.sample(0, 0.01)])

    def test_source_camera_plate_resolves_only_exact_staged_raster(self):
        data = b"reviewed plate bytes"
        digest = hashlib.sha256(data).hexdigest()
        artifact = {"id": "background-texture", "ref": {"role": "texture_image", "sha256": digest, "size_bytes": str(len(data))}}
        manifest = {"assets": [{"binding_id": "background-texture", "role": "scene_texture", "name": "plate.png"}]}
        scene = {"photographic_background": {
            "route": "PHOTOGRAPH_ROUTE_EMPTY_SCENE", "source_raster_sha256": digest,
        }, "artifacts": [artifact]}
        with tempfile.TemporaryDirectory() as root:
            path = os.path.join(root, "plate.png")
            with open(path, "wb") as output:
                output.write(data)
            self.assertEqual(resolve_plate(root, manifest, scene), path)
            with open(path, "ab") as output:
                output.write(b"changed")
            with self.assertRaisesRegex(ValueError, "unexpected size"):
                resolve_plate(root, manifest, scene)

    def test_source_camera_plate_device_removal_binds_reviewed_clean_plate_not_source(self):
        plate = b"reviewed authored clean plate"
        source = b"different original source raster"
        plate_digest = hashlib.sha256(plate).hexdigest()
        artifact = {"id": "background-texture", "ref": {
            "role": "texture_image", "sha256": plate_digest, "size_bytes": str(len(plate)),
        }}
        manifest = {"assets": [{
            "binding_id": "background-texture", "role": "scene_texture", "name": "plate.png",
        }]}
        scene = {"photographic_background": {
            "route": "PHOTOGRAPH_ROUTE_DEVICE_REMOVAL",
            "source_raster_sha256": hashlib.sha256(source).hexdigest(),
            "repair_provenance_sha256": "reviewed-repair-digest",
        }, "artifacts": [artifact]}
        with tempfile.TemporaryDirectory() as root:
            path = os.path.join(root, "plate.png")
            with open(path, "wb") as output:
                output.write(plate)
            self.assertEqual(resolve_plate(root, manifest, scene), path)
            scene["photographic_background"]["repair_provenance_sha256"] = ""
            with self.assertRaisesRegex(ValueError, "lacks reviewed repair provenance"):
                resolve_plate(root, manifest, scene)

    def test_source_camera_plate_rejects_stretch_and_accepts_matching_aspect(self):
        validate_plate_aspect(1672, 941, 1920, 1080)
        with self.assertRaisesRegex(ValueError, "aspect ratio differs"):
            validate_plate_aspect(1536, 1024, 1920, 1080)

    def test_support_catcher_selects_only_triangles_on_reviewed_plane(self):
        normal = (0.0, 1.0, 0.0)
        point = (0.0, 0.25, 0.0)
        flat = [(0.0, 0.25, 0.0), (1.0, 0.253, 0.0), (0.0, 0.246, 1.0)]
        warped = [(0.0, 0.25, 0.0), (1.0, 0.25, 0.0), (0.0, 0.281, 1.0)]
        self.assertTrue(triangle_within_plane(flat, normal, point, 0.01))
        self.assertFalse(triangle_within_plane(warped, normal, point, 0.01))
        self.assertFalse(triangle_within_plane(flat[:2], normal, point, 0.01))

    def test_shadow_catcher_quad_is_two_triangles_on_reviewed_plane_with_padding(self):
        points = [(-0.1, 0.01, -0.2), (0.1, 0.01, -0.2),
                  (0.1, 0.01, 0.2), (-0.1, 0.01, 0.2),
                  (-0.1, 0.02, -0.2), (0.1, 0.02, -0.2),
                  (0.1, 0.02, 0.2), (-0.1, 0.02, 0.2)]
        quad = support_plane_quad((0.0, 1.0, 0.0), (0.0, 0.0, 0.0), points, 0.25)
        self.assertEqual(len(quad["corners"]), 4)
        self.assertAlmostEqual(quad["footprint"][0], -0.1)
        self.assertAlmostEqual(quad["footprint"][1], -0.2)
        self.assertAlmostEqual(quad["footprint"][2], 0.1)
        self.assertAlmostEqual(quad["footprint"][3], 0.2)
        for corner in quad["corners"]:
            self.assertAlmostEqual(corner[1], 0.0, places=7)
        self.assertAlmostEqual(abs(quad["corners"][1][0] - quad["corners"][0][0]), 0.7)
        self.assertAlmostEqual(abs(quad["corners"][2][2] - quad["corners"][1][2]), 0.9)
        self.assertAlmostEqual(quad["signed_distance_min"], 0.01)
        self.assertAlmostEqual(quad["signed_distance_max"], 0.02)

    def test_shadow_catcher_quad_rejects_invalid_plane_and_padding(self):
        points = [(-1, 0, -1), (1, 0, -1), (1, 0, 1), (-1, 0, 1)]
        with self.assertRaisesRegex(ValueError, "normal is not normalized"):
            support_plane_quad((0.0, 2.0, 0.0), (0, 0, 0), points, 0.1)
        with self.assertRaisesRegex(ValueError, "margin is invalid"):
            support_plane_quad((0, 1, 0), (0, 0, 0), points, float("nan"))

    def test_actual_support_catcher_invocation_builds_two_triangles_after_observed_check(self):
        fake_bpy = FakeBpy()
        namespace["bpy"] = fake_bpy
        namespace["Vector"] = lambda value: value
        namespace["canonical_to_blender"] = lambda: IdentityTransform()
        scene = type("Scene", (), {"collection": type("Collection", (), {"objects": SceneObjects()})()})()
        device_points = [
            (-0.05, 0.002, -0.1), (0.05, 0.002, -0.1),
            (0.05, 0.002, 0.1), (-0.05, 0.002, 0.1),
            (-0.05, 0.006, -0.1), (0.05, 0.006, -0.1),
            (0.05, 0.006, 0.1), (-0.05, 0.006, 0.1),
        ]
        device = type("Device", (), {
            "type": "MESH", "data": object(), "bound_box": device_points,
            "matrix_world": IdentityTransform(),
        })()
        source = type("Source", (), {
            "type": "MESH", "data": MeshData([
                (-0.05, 0.0, -0.1), (0.05, 0.0, -0.1),
                (0.05, 0.0, 0.1), (-0.05, 0.0, 0.1), (0.0, 0.0, 0.0),
            ]), "matrix_world": IdentityTransform(),
        })()
        photograph = {
            "support_plane_normal_opencv": [0.0, -1.0, 0.0],
            "support_plane_offset_estimate_m": 0.0,
            "support_plane_residual_p95_estimate_m": 0.003,
        }
        catchers = make_support_shadow_catchers(scene, [source], [device], photograph, {
            "translation_m": {"x": 0.0, "y": 0.0, "z": 0.0},
            "rotation_xyzw": {"x": 0.0, "y": 0.0, "z": 0.0, "w": 1.0},
        })
        self.assertEqual(len(catchers), 1)
        self.assertEqual(len(fake_bpy.meshes[0].created_vertices), 4)
        self.assertEqual(fake_bpy.meshes[0].created_faces, [(0, 1, 2), (0, 2, 3)])
        self.assertTrue(catchers[0].is_shadow_catcher)
        self.assertEqual(scene.collection.objects.items, catchers)

    def test_actual_support_catcher_rejects_device_away_from_plane_before_mesh_creation(self):
        fake_bpy = FakeBpy()
        namespace["bpy"] = fake_bpy
        namespace["Vector"] = lambda value: value
        namespace["canonical_to_blender"] = lambda: IdentityTransform()
        scene = type("Scene", (), {"collection": type("Collection", (), {"objects": SceneObjects()})()})()
        device = type("Device", (), {
            "type": "MESH", "data": object(),
            "bound_box": [(x, 0.3, z) for x in (-0.05, 0.05) for z in (-0.1, 0.1)],
            "matrix_world": IdentityTransform(),
        })()
        with self.assertRaisesRegex(ValueError, "do not intersect"):
            make_support_shadow_catchers(scene, [], [device], {
                "support_plane_normal_opencv": [0.0, -1.0, 0.0],
                "support_plane_offset_estimate_m": 0.0,
                "support_plane_residual_p95_estimate_m": 0.003,
            }, {"translation_m": {"x": 0.0, "y": 0.0, "z": 0.0},
                "rotation_xyzw": {"x": 0.0, "y": 0.0, "z": 0.0, "w": 1.0}})
        self.assertEqual(fake_bpy.meshes, [])

    def test_support_plane_moves_from_opencv_camera_into_exact_scene_world(self):
        camera = {
            "translation_m": {"x": 1.0, "y": 2.0, "z": 3.0},
            "rotation_xyzw": {"x": 0.0, "y": 0.0, "z": math.sin(math.pi / 4), "w": math.cos(math.pi / 4)},
        }
        normal, point = opencv_plane_to_world((0.0, -1.0, 0.0), 1.0, camera)
        for actual, expected in zip(normal, (-1.0, 0.0, 0.0)):
            self.assertAlmostEqual(actual, expected, places=6)
        for actual, expected in zip(point, (2.0, 2.0, 3.0)):
            self.assertAlmostEqual(actual, expected, places=6)

    def test_composite_uses_blender_5_standalone_compositor_group(self):
        source = WORKER.read_text(encoding="utf-8")
        self.assertIn('bpy.data.node_groups.new("photographic-device-composite", "CompositorNodeTree")', source)
        self.assertIn('scene.compositing_node_group = compositor_tree', source)
        self.assertIn('nodes.new("NodeGroupOutput")', source)
        self.assertNotIn("scene.node_tree", source)
        self.assertNotIn("scene.use_nodes", source)


if __name__ == "__main__":
    unittest.main()
