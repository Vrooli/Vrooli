"""Known-camera regressions for MoGe full-geometry postprocessing."""

import os
import io
import struct
import zipfile
import warnings
from unittest import mock
import tempfile
import unittest

import numpy as np

from image_tools_sidecar.geometry import (
    _read_geometry_array,
    normalized_view_plane_uv,
    fit_support_plane,
    postprocess,
    recover_focal_shift,
    validate_geometry_archive,
    write_geometry_archive,
)


class GeometryPostprocessTest(unittest.TestCase):
    def make_raw_scene(self):
        width, height = 96, 72
        uv = normalized_view_plane_uv(width, height)
        focal = 1.37
        shift = 0.42
        true_depth = 2.4 + 0.55 * uv[..., 0] - 0.31 * uv[..., 1]
        points = np.empty((height, width, 3), dtype=np.float32)
        points[..., :2] = uv * true_depth[..., None] / focal
        points[..., 2] = true_depth - shift
        normals = np.zeros_like(points)
        normals[..., 2] = -1.0
        mask = np.ones((height, width), dtype=np.float32)
        scale = np.asarray([1.6], dtype=np.float32)
        return uv, focal, shift, true_depth, points, normals, mask, scale

    def test_recovers_known_camera_and_reprojects_points(self):
        uv, expected_focal, expected_shift, true_depth, raw, normals, mask, scale = self.make_raw_scene()
        focal, shift = recover_focal_shift(raw, mask > 0.5)
        self.assertAlmostEqual(focal, expected_focal, delta=0.015)
        self.assertAlmostEqual(shift, expected_shift, delta=0.02)

        result = postprocess(raw, normals, mask, scale)
        self.assertEqual(result["points"].dtype, np.float32)
        self.assertEqual(result["depth"].dtype, np.float32)
        self.assertTrue(np.all(result["validity"] == 1))
        np.testing.assert_allclose(result["depth"], true_depth * 1.6, atol=0.04, rtol=0.015)
        self.assertAlmostEqual(float(result["intrinsics"][0, 2]), 0.5, places=7)
        self.assertAlmostEqual(float(result["intrinsics"][1, 2]), 0.5, places=7)
        np.testing.assert_allclose(result["points"][..., 0] / result["points"][..., 2],
                                   uv[..., 0] / expected_focal, atol=0.004)
        np.testing.assert_allclose(result["points"][..., 1] / result["points"][..., 2],
                                   uv[..., 1] / expected_focal, atol=0.004)

    def test_invalid_prediction_samples_remain_explicitly_invalid(self):
        _uv, _f, _s, _z, points, normals, mask, scale = self.make_raw_scene()
        mask[:10, :10] = 0.0
        result = postprocess(points, normals, mask, scale)
        self.assertEqual(int(result["validity"][:10, :10].sum()), 0)
        self.assertTrue(np.isnan(result["depth"][:10, :10]).all())
        self.assertTrue(np.isnan(result["points"][:10, :10]).all())

    def test_archive_schema_keeps_arrays_and_provenance_readable(self):
        _uv, _f, _s, _z, raw, normals, mask, scale = self.make_raw_scene()
        result = postprocess(raw, normals, mask, scale)
        arrays = {
            f"{name}.npy": {"dtype": str(result[name].dtype), "shape": list(result[name].shape)}
            for name in ("points", "depth", "normals", "validity", "intrinsics")
        }
        manifest = {
            "format": "vrooli.image-tools.scene-geometry.v1",
            "coordinates": {"frame": "OpenCV camera"},
            "units": {"scale_kind": "estimated_metric"},
            "arrays": arrays,
        }
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "geometry.npz")
            write_geometry_archive(path, result, manifest)
            self.assertEqual(validate_geometry_archive(path)["format"], manifest["format"])

    def test_archive_validation_rejects_camera_projection_mismatch(self):
        _uv, _f, _s, _z, raw, normals, mask, scale = self.make_raw_scene()
        result = postprocess(raw, normals, mask, scale)
        result["points"] = result["points"].copy()
        result["points"][20, 20, 0] += 0.1
        arrays = {
            f"{name}.npy": {"dtype": str(result[name].dtype), "shape": list(result[name].shape)}
            for name in ("points", "depth", "normals", "validity", "intrinsics")
        }
        manifest = {"format": "vrooli.image-tools.scene-geometry.v1",
                    "coordinates": {"frame": "OpenCV camera"},
                    "units": {"scale_kind": "estimated_metric"}, "arrays": arrays}
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "bad.zip")
            with self.assertRaisesRegex(ValueError, "reproject"):
                write_geometry_archive(path, result, manifest)

    def test_npy_header_cannot_allocate_more_than_retained_bytes(self):
        header = b"{'descr': '<f4', 'fortran_order': False, 'shape': (1000000000, 3), }\n"
        raw = b"\x93NUMPY\x01\x00" + struct.pack("<H", len(header)) + header
        with mock.patch("image_tools_sidecar.geometry.np.load") as load:
            with self.assertRaisesRegex(ValueError, "byte count"):
                _read_geometry_array(raw, "points", {"shape": [1000000000, 3], "dtype": "float32"})
            load.assert_not_called()

    def test_geometry_array_refuses_foreign_dtype_before_allocation(self):
        stream = io.BytesIO()
        np.save(stream, np.ones((2, 3), dtype=np.float64), allow_pickle=False)
        with mock.patch("image_tools_sidecar.geometry.np.load") as load:
            with self.assertRaisesRegex(ValueError, "float32/uint8"):
                _read_geometry_array(stream.getvalue(), "depth", {"shape": [2, 3], "dtype": "float64"})
            load.assert_not_called()

    def test_geometry_archive_rejects_duplicate_members(self):
        with tempfile.TemporaryDirectory() as directory:
            path = os.path.join(directory, "duplicate.zip")
            with zipfile.ZipFile(path, "w") as archive:
                for name in ("points.npy", "depth.npy", "normals.npy", "validity.npy", "intrinsics.npy", "manifest.json"):
                    archive.writestr(name, b"not a valid artifact")
                with warnings.catch_warnings():
                    warnings.simplefilter("ignore", UserWarning)
                    archive.writestr("points.npy", b"shadowed geometry")
            with self.assertRaisesRegex(ValueError, "missing, unexpected, or corrupt"):
                validate_geometry_archive(path)

    def test_support_plane_fit_handles_outliers_and_roi_exclusion(self):
        height, width = 80, 100
        uv = normalized_view_plane_uv(width, height)
        depth = np.full((height, width), 3.0, dtype=np.float32)
        points = np.stack((uv[..., 0] * depth, uv[..., 1] * depth, depth), axis=-1)
        validity = np.ones((height, width), dtype=np.uint8)
        normals = np.zeros_like(points)
        normals[..., 2] = 1.0
        # Make a deterministic patch of gross prediction outliers inside ROI.
        points[30:40, 40:50, 2] = 5.0
        roi = {"x": 0.1, "y": 0.1, "width": 0.8, "height": 0.8}
        plane, rejected = fit_support_plane(points, validity, roi, threshold_m=0.02,
                                             normals=normals, trials=256)
        self.assertGreater(plane["inlier_fraction"], 0.9)
        self.assertEqual(plane["metric_status"], "estimated_monocular_scale_not_measured")
        self.assertAlmostEqual(abs(plane["normal_opencv_camera"][2]), 1.0, delta=1e-4)
        self.assertAlmostEqual(abs(plane["offset_m_estimate"]), 3.0, delta=1e-4)
        self.assertEqual(int(rejected[35, 45]), 1)
        self.assertEqual(int(rejected[5, 5]), 0)
        self.assertGreater(plane["rejected_roi_pixels"], 0)

        archived = {"points": points, "depth": depth, "normals": normals,
                    "validity": validity, "intrinsics": np.eye(3, dtype=np.float32),
                    "support_plane_rejected": rejected}
        specs = {f"{name}.npy": {"dtype": str(value.dtype), "shape": list(value.shape)}
                 for name, value in archived.items()}
        manifest = {"format": "vrooli.image-tools.scene-geometry.v1",
                    "coordinates": {"frame": "OpenCV camera"},
                    "units": {"scale_kind": "estimated_metric"},
                    "arrays": specs, "support_plane": plane}
        # Replace synthetic non-normalized coordinates with projection-consistent
        # data through a known pinhole camera before checking the archive contract.
        archived["intrinsics"] = np.array([[1.0, 0.0, 0.5], [0.0, 1.0, 0.5],
                                           [0.0, 0.0, 1.0]], dtype=np.float32)
        xx = (np.arange(width, dtype=np.float32) + 0.5) / width
        yy = (np.arange(height, dtype=np.float32) + 0.5) / height
        uu, vv = np.meshgrid(xx, yy)
        archived["points"] = np.stack(((uu - 0.5) * depth, (vv - 0.5) * depth, depth), axis=-1)
        manifest["arrays"] = {f"{name}.npy": {"dtype": str(value.dtype), "shape": list(value.shape)}
                               for name, value in archived.items()}
        with tempfile.TemporaryDirectory() as tmp:
            path = os.path.join(tmp, "support.zip")
            write_geometry_archive(path, archived, manifest)
            self.assertEqual(validate_geometry_archive(path)["support_plane"]["inlier_points"], plane["inlier_points"])

    def test_support_plane_requires_a_valid_normalized_roi(self):
        _uv, _f, _s, _z, points, _normals, _mask, _scale = self.make_raw_scene()
        with self.assertRaisesRegex(ValueError, "normalized rectangle"):
            fit_support_plane(points, np.ones(points.shape[:2]), {"x": 0.5, "y": 0.5, "width": 0.7, "height": 0.5})


if __name__ == "__main__":
    unittest.main()
