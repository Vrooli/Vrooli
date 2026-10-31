"""Repair contracts independent of learned image quality."""
import unittest
from types import SimpleNamespace

import numpy as np
from PIL import Image

from image_tools_sidecar.lama import effective_removal_mask, repair


class Export:
    def get_inputs(self):
        return [SimpleNamespace(name="image", type="tensor(float)", shape=["batch", 3, 512, 512]),
                SimpleNamespace(name="mask", type="tensor(float)", shape=["batch", 1, 512, 512])]

    def run(self, _outputs, inputs):
        self.calls = getattr(self, "calls", 0) + 1
        self.inputs = inputs
        result = np.zeros((1, 3, 512, 512), dtype=np.float32)
        result[:, 2] = 255
        return [result]


class Export1024(Export):
    def get_inputs(self):
        return [SimpleNamespace(name="image", type="tensor(float)", shape=["batch", 3, 1024, 1024]),
                SimpleNamespace(name="mask", type="tensor(float)", shape=["batch", 1, 1024, 1024])]

    def run(self, _outputs, inputs):
        self.calls = getattr(self, "calls", 0) + 1
        self.inputs = inputs
        result = np.zeros((1, 3, 1024, 1024), dtype=np.float32)
        result[:, 2] = 255
        return [result]


class LaMaRepairTest(unittest.TestCase):
    def fixture(self):
        source = Image.new("RGB", (180, 90), (213, 47, 9))
        mask = Image.new("L", source.size)
        for y in range(30, 50):
            for x in range(70, 90):
                mask.putpixel((x, y), 255)
        return source, mask

    def test_white_region_repaired_without_resizing_or_changing_surroundings(self):
        source, mask = self.fixture()
        session = Export()
        result, receipt = repair(session, source, mask)
        self.assertEqual(result.size, source.size)
        original, output = np.asarray(source), np.asarray(result)
        selected = np.asarray(mask) > 0
        np.testing.assert_array_equal(output[~selected], original[~selected])
        self.assertTrue((output[selected] == (0, 0, 255)).all())
        self.assertEqual(receipt["removed_pixels"], 400)
        self.assertEqual(session.inputs["image"].shape, (1, 3, 512, 512))
        self.assertTrue(np.isin(session.inputs["mask"], (0, 1)).all())
        self.assertGreater(receipt["letterbox_offset_pixels"][1], 0)

    def test_fixed_1024_profile_preserves_the_same_mask_contract(self):
        source, mask = self.fixture()
        session = Export1024()
        result, receipt = repair(session, source, mask)
        selected = np.asarray(mask) >= 128
        original, output = np.asarray(source), np.asarray(result)
        np.testing.assert_array_equal(output[~selected], original[~selected])
        self.assertEqual(receipt["model_raster"], [1024, 1024])
        self.assertEqual(session.inputs["image"].shape, (1, 3, 1024, 1024))

    def test_mask_requires_matching_raster_and_surrounding_context(self):
        source, _mask = self.fixture()
        for mask in (Image.new("L", (10, 10)), Image.new("L", source.size),
                     Image.new("L", source.size, 255)):
            with self.subTest(mask=mask.size):
                with self.assertRaises(ValueError):
                    repair(Export(), source, mask)

    def test_unqualified_tensor_contract_and_nonfinite_output_are_refused(self):
        source, mask = self.fixture()
        class WrongShape(Export):
            def get_inputs(self):
                return [SimpleNamespace(name="image", type="tensor(float)", shape=[1, 3, 256, 256])]
        with self.assertRaisesRegex(ValueError, "unqualified"):
            repair(WrongShape(), source, mask)
        class Nonfinite(Export):
            def run(self, outputs, inputs):
                return [np.full((1, 3, 512, 512), np.nan)]
        with self.assertRaisesRegex(ValueError, "finite"):
            repair(Nonfinite(), source, mask)

    def test_explicit_expansion_and_normalized_regions_only_add_to_mask(self):
        _source, mask = self.fixture()
        before = np.asarray(mask).copy()
        effective = effective_removal_mask(mask, 3, [{"x": 0.5, "y": 0.1, "width": 0.1, "height": 0.1}])
        original, actual = before >= 128, np.asarray(effective) >= 128
        self.assertTrue(np.all(actual[original]))
        self.assertGreater(np.count_nonzero(actual & ~original), 0)
        self.assertTrue(actual[9, 90])  # normalized region is painted into the source raster
        np.testing.assert_array_equal(np.asarray(mask), before)  # caller's mask is immutable

    def test_explicit_effective_mask_preserves_all_pixels_outside_it(self):
        source, mask = self.fixture()
        session = Export()
        result, receipt = repair(session, source, mask, 4, [{"x": .5, "y": .1, "width": .1, "height": .1}])
        original, output = np.asarray(source), np.asarray(result)
        selected = np.asarray(effective_removal_mask(mask, 4, [{"x": .5, "y": .1, "width": .1, "height": .1}])) >= 128
        np.testing.assert_array_equal(output[~selected], original[~selected])
        self.assertEqual(receipt["mask_expansion_pixels"], 4)
        self.assertGreater(receipt["effective_mask_added_pixels"], 0)
        self.assertIn("effective_mask_png_base64", receipt)

    def test_expansion_and_regions_are_bounded(self):
        _source, mask = self.fixture()
        with self.assertRaisesRegex(ValueError, "0 to 64"):
            effective_removal_mask(mask, 65)
        with self.assertRaisesRegex(ValueError, "within 0..1"):
            effective_removal_mask(mask, 0, [{"x": .8, "y": .8, "width": .5, "height": .5}])

    def test_large_photographic_masks_use_overlapping_crops_and_preserve_outside(self):
        source = Image.new("RGB", (1400, 900), (213, 47, 9))
        mask_pixels = np.zeros((900, 1400), dtype=np.uint8)
        mask_pixels[80:820, 100:1300] = 255
        mask = Image.fromarray(mask_pixels)
        session = Export()
        result, receipt = repair(session, source, mask)
        self.assertEqual(session.calls, len(receipt["tile_crops"]))
        self.assertGreater(session.calls, 1)
        self.assertTrue(receipt["tiled_inference"])
        original, output = np.asarray(source), np.asarray(result)
        selected = np.asarray(mask) >= 128
        np.testing.assert_array_equal(output[~selected], original[~selected])


if __name__ == "__main__":
    unittest.main()
