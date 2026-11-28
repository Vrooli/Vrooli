import importlib.util
import math
from pathlib import Path
import unittest


MODULE_PATH = Path(__file__).with_name("ipad_environment_math.py")
SPEC = importlib.util.spec_from_file_location("ipad_environment_math_under_test", MODULE_PATH)
math3d = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(math3d)


class EnvironmentMathTests(unittest.TestCase):
    def test_projected_full_geometry_bounds_respect_inclusive_frame_margin(self):
        self.assertTrue(math3d.bounds_within_frame((0.04, 0.04, 0.96, 0.96), 0.04))
        self.assertFalse(math3d.bounds_within_frame((0.1, 0.1, 0.960001, 0.9), 0.04))
        self.assertFalse(math3d.bounds_within_frame((0.1, 0.039, 0.9, 0.9), 0.04))
        self.assertFalse(math3d.bounds_within_frame((0.1, 0.2, 0.9), 0.04))
        self.assertFalse(math3d.bounds_within_frame((0.1, 0.2, math.inf, 0.9), 0.04))

    def test_blender_up_and_depth_map_to_canonical_up_and_front(self):
        point = (2.0, 3.0, 4.0)
        delta = math3d.canonical_delta_from_blender_xy(0.1, 0.2)
        before = math3d.blender_to_canonical(point)
        after = math3d.blender_to_canonical((point[0] + 0.1, point[1] + 0.2, point[2]))
        self.assertEqual(before, (2.0, 4.0, -3.0))
        self.assertEqual(delta, (0.1, 0.0, -0.2))
        self.assertEqual(after, tuple(before[i] + delta[i] for i in range(3)))

    def test_table_bounds_are_xy_centered_and_respect_edge_margin(self):
        actual = math3d.table_bounds_from_dimensions((0, 0), (0.95, 0.78), 0.02)
        for got, want in zip(actual, (-0.455, -0.37, 0.455, 0.37)):
            self.assertAlmostEqual(got, want)
        with self.assertRaises(ValueError):
            math3d.table_bounds_from_dimensions((0, 0), (0.95, 0.78), 0.40)

    def test_center_and_sensor_edge_project_to_expected_normalized_points(self):
        aspect = 16 / 9
        self.assertEqual(math3d.project_camera_local((0, 0, -1), 85, 36, aspect), (0.5, 0.5))
        self.assertEqual(math3d.project_camera_local((18 / 85, 18 / (85 * aspect), -1), 85, 36, aspect),
                         (1.0, 0.0))

    def test_projected_offset_scales_with_focal_and_inverse_depth(self):
        aspect = 16 / 9
        near = math3d.project_camera_local((0.18, 0.1, -1), 85, 36, aspect)
        far = math3d.project_camera_local((0.18, 0.1, -2), 85, 36, aspect)
        self.assertAlmostEqual(near[0], 0.5 + 0.18 * 85 / 36)
        self.assertAlmostEqual(near[1], 0.5 - 0.1 * 85 * aspect / 36)
        self.assertAlmostEqual(far[0] - 0.5, (near[0] - 0.5) / 2)
        self.assertAlmostEqual(0.5 - far[1], (0.5 - near[1]) / 2)

    def test_invalid_camera_values_and_behind_camera_are_rejected(self):
        for point, focal in (((0, 0, 0), 85), ((0, math.nan, -1), 85), ((0, 0, -1), 0)):
            with self.subTest(point=point, focal=focal), self.assertRaises(ValueError):
                math3d.project_camera_local(point, focal, 36, 16 / 9)


if __name__ == "__main__":
    unittest.main()
