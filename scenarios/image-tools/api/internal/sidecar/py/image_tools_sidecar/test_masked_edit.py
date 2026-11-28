"""Selected-area edit geometry: the guarantees the paste-back makes without any model."""
import unittest

import numpy as np
from PIL import Image

from image_tools_sidecar import _diffusers, masked_edit


def _noise_image(size, seed=3):
    rng = np.random.default_rng(seed)
    return Image.fromarray(rng.integers(0, 256, (size[1], size[0], 3), dtype=np.uint8), mode="RGB")


class PlanCropTest(unittest.TestCase):
    def test_small_selection_is_upscaled_to_a_workable_size_on_a_snapped_grid(self):
        plan = masked_edit.plan_crop((1152, 1536), (340, 555, 465, 650))
        self.assertEqual(max(plan.infer_size), masked_edit.MIN_LONG_EDGE)
        self.assertTrue(all(side % masked_edit.SNAP == 0 for side in plan.infer_size))
        x0, y0, x1, y1 = plan.box
        self.assertLessEqual((x0, y0), (340, 555))
        self.assertGreaterEqual((x1, y1), (465, 650))

    def test_large_selection_is_capped_at_the_requested_max_edge(self):
        plan = masked_edit.plan_crop((4000, 3000), (100, 100, 3900, 2900), max_edge=1024)
        self.assertEqual(max(plan.infer_size), 1024)

    def test_crop_never_leaves_the_frame(self):
        plan = masked_edit.plan_crop((200, 100), (0, 0, 200, 100), context_px=512)
        self.assertEqual(plan.box, (0, 0, 200, 100))

    def test_context_always_covers_the_feather(self):
        plan = masked_edit.plan_crop((1000, 1000), (500, 500, 520, 520), blend_px=100)
        self.assertGreaterEqual(plan.context_px, 100)
        self.assertEqual(plan.box, (400, 400, 620, 620))

    def test_rejects_out_of_range_options(self):
        with self.assertRaises(ValueError):
            masked_edit.plan_crop((100, 100), (0, 0, 10, 10), max_edge=64)
        with self.assertRaises(ValueError):
            masked_edit.plan_crop((100, 100), (0, 0, 10, 10), context_px=600)

    def test_empty_mask_is_refused(self):
        with self.assertRaises(ValueError):
            masked_edit.mask_bounds(np.zeros((8, 8), dtype=bool))


class CompositeTest(unittest.TestCase):
    def setUp(self):
        self.source = _noise_image((120, 90))
        self.generated = Image.new("RGB", (60, 60), (250, 10, 10))
        self.mask = np.zeros((90, 120), dtype=bool)
        self.mask[30:60, 40:80] = True

    def _run(self, grow, feather):
        grown = masked_edit.grow_mask(self.mask, grow)
        box = (20, 10, 100, 80)
        alpha = masked_edit.blend_alpha(grown[10:80, 20:100], feather)
        result = masked_edit.composite_crop(self.source, self.generated, alpha, box)
        return grown, result

    def test_pixels_outside_the_blend_zone_are_the_source_bytes(self):
        grow, feather = 4, 10
        _, result = self._run(grow, feather)
        zone = masked_edit.grow_mask(self.mask, grow + feather)
        changed = (np.asarray(result) != np.asarray(self.source)).any(axis=2)
        self.assertEqual(int(np.count_nonzero(changed & ~zone)), 0)

    def test_mask_interior_takes_the_generated_pixels(self):
        _, result = self._run(0, 0)
        self.assertEqual(result.getpixel((60, 45)), (250, 10, 10))

    def test_feather_produces_a_gradient_not_a_cut(self):
        _, result = self._run(0, 12)
        edge_row = [result.getpixel((x, 45))[0] for x in range(30, 50)]
        distinct = len(set(edge_row))
        self.assertGreater(distinct, 4)

    def test_zero_feather_is_a_hard_edge(self):
        grown, result = self._run(0, 0)
        arr = np.asarray(result)
        src = np.asarray(self.source)
        self.assertTrue((arr[~grown] == src[~grown]).all())

    def test_grow_expands_the_mask_by_the_requested_distance(self):
        grown = masked_edit.grow_mask(self.mask, 5)
        self.assertTrue(grown[30, 35])
        self.assertFalse(grown[30, 33])


class FamilyParamsTest(unittest.TestCase):
    def test_klein_receives_the_planned_inference_size(self):
        plan = masked_edit.plan_crop((1152, 1536), (340, 555, 465, 650))
        params = _diffusers.Params(prompt="make the cup a red mug", width=plan.infer_size[0], height=plan.infer_size[1])
        kwargs = _diffusers.build_call_kwargs("flux-2-klein", params)
        self.assertEqual((kwargs["width"], kwargs["height"]), plan.infer_size)


if __name__ == "__main__":
    unittest.main()
