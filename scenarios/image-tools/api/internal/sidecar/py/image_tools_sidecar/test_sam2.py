"""SAM 2.1 prompt contracts, kept independent from model weights."""

import unittest
from types import SimpleNamespace
from unittest.mock import patch

from . import sam2
from .sam2 import _prompts


class SAM2PromptContracts(unittest.TestCase):
    def test_positive_and_negative_points_keep_normalized_image_coordinates(self):
        prompts = _prompts(
            '{"points":[{"x":0.75,"y":0.25},{"x":0.1,"y":0.9,"negative":true}]}',
            (800, 400),
        )
        self.assertEqual(prompts["input_points"], [[[[600.0, 100.0], [80.0, 360.0]]]])
        self.assertEqual(prompts["input_labels"], [[[1, 0]]])

    def test_box_uses_xyxy_pixel_coordinates_and_can_refine_with_points(self):
        prompts = _prompts(
            '{"box":{"x":0.2,"y":0.1,"width":0.5,"height":0.75},"points":[{"x":0.4,"y":0.3}]}',
            (800, 400),
        )
        self.assertEqual(prompts["input_boxes"], [[[160.0, 40.0, 560.0, 340.0]]])
        self.assertEqual(prompts["input_points"], [[[[320.0, 120.0]]]])
        self.assertEqual(prompts["input_labels"], [[[1]]])

    def test_invalid_prompts_fail_closed(self):
        invalid = (
            "{}",
            '{"points":[{"x":0.5,"y":0.5,"negative":true}]}',
            '{"points":[{"x":1.2,"y":0.5}]}',
            '{"box":{"x":0.5,"y":0.5,"width":0.6,"height":0.1}}',
            '{"points":[{"x":0.5,"y":0.5}],"box":{"x":0,"y":0,"width":1,"height":1},"extra":true}',
        )
        for raw in invalid:
            with self.subTest(raw=raw), self.assertRaises((ValueError, TypeError)):
                _prompts(raw, (640, 480))

    def test_peak_rss_metric_is_optional_on_windows(self):
        with patch.object(sam2, "resource", None):
            self.assertIsNone(sam2._max_rss_kib())

    def test_peak_rss_metric_normalizes_bytes_on_macos(self):
        usage = SimpleNamespace(ru_maxrss=1_234_567)
        fake_resource = SimpleNamespace(RUSAGE_SELF=0, getrusage=lambda _: usage)
        with patch.object(sam2, "resource", fake_resource), patch.object(sam2.sys, "platform", "darwin"):
            self.assertEqual(sam2._max_rss_kib(), 1205)


if __name__ == "__main__":
    unittest.main()
