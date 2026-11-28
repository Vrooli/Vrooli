"""OCR post-processing contract: geometry, CTC decoding and reading order. No model
download or GPU. Run with unittest."""
import unittest
from types import SimpleNamespace

import numpy as np

from . import ocr


class MinAreaRect(unittest.TestCase):
    def test_a_rotated_bar_gets_a_tight_rotated_rectangle(self):
        # A 60x10 bar rotated 30 degrees: the rectangle must be ~60x10, far smaller
        # than its axis-aligned bounding box.
        theta = np.deg2rad(30)
        rot = np.array([[np.cos(theta), -np.sin(theta)], [np.sin(theta), np.cos(theta)]])
        bar = np.array([[x, y] for x in np.linspace(0, 60, 40) for y in np.linspace(0, 10, 8)])
        pts = bar @ rot.T + np.array([100.0, 50.0])
        rect = ocr._min_area_rect(np, pts)
        w, h = sorted(ocr._rect_size(rect))
        self.assertAlmostEqual(w, 10, delta=0.5)
        self.assertAlmostEqual(h, 60, delta=0.5)

    def test_grown_rectangle_is_larger_on_every_side(self):
        rect = ocr._min_area_rect(np, np.array([[0, 0], [40, 0], [40, 10], [0, 10]], dtype=float))
        grown = ocr._rect_corners(np, rect, grow=3.0)
        self.assertAlmostEqual(float(grown[:, 0].max() - grown[:, 0].min()), 46, delta=0.01)
        self.assertAlmostEqual(float(grown[:, 1].max() - grown[:, 1].min()), 16, delta=0.01)


class ReadingOrder(unittest.TestCase):
    def block(self, text, x, y, w=50, h=20):
        return {"text": text, "confidence": 1.0, "box": {"x": x, "y": y, "width": w, "height": h}}

    def test_rows_read_top_to_bottom_and_left_to_right(self):
        blocks = [self.block("c", 0, 100), self.block("b", 120, 4), self.block("a", 0, 0)]
        ordered, rows = ocr._reading_order(blocks)
        self.assertEqual([b["text"] for b in ordered], ["a", "b", "c"])
        self.assertEqual(len(rows), 2, "a and b share a row")


class CTCDecoding(unittest.TestCase):
    class Session:
        def __init__(self, classes):
            self.classes = classes

        def get_inputs(self):
            return [SimpleNamespace(name="x")]

        def run(self, _names, feed):
            # The timesteps spell: blank h h blank i i  => "hi" after collapsing.
            steps = [0, 2, 2, 0, 3, 3]
            logits = np.zeros((1, len(steps), self.classes), dtype=np.float32)
            for t, index in enumerate(steps):
                logits[0, t, index] = 9.0
            return [logits]

    def test_collapses_repeats_and_drops_blanks(self):
        from PIL import Image

        chars = ["blank", "x", "h", "i", " "]
        text, confidence = ocr._recognise(np, self.Session(len(chars)), chars, Image.new("RGB", (60, 20), "white"))
        self.assertEqual(text, "hi")
        self.assertAlmostEqual(confidence, 9.0, places=3)

    def test_a_dictionary_that_does_not_match_the_recogniser_is_refused(self):
        from PIL import Image

        with self.assertRaises(SystemExit):
            ocr._recognise(np, self.Session(7), ["blank", "x"], Image.new("RGB", (60, 20), "white"))

    def test_the_dictionary_gains_a_blank_first_and_a_space_last(self):
        import tempfile, os

        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "dict.txt")
            with open(path, "w", encoding="utf-8") as f:
                f.write("a\nb\n")
            self.assertEqual(ocr._load_dictionary(path), ["blank", "a", "b", " "])


class ScalePyramid(unittest.TestCase):
    def image(self, w, h):
        return SimpleNamespace(size=(w, h))

    def test_a_large_photo_is_read_at_its_capped_size_and_each_distinct_target(self):
        self.assertEqual(ocr._pyramid(self.image(4284, 5712)), [4000, 3072, 2048, 1280])

    def test_a_scale_close_to_one_already_chosen_is_skipped(self):
        # 2100 is within 1.25x of 2048, 1300 within 1.25x of 1280.
        self.assertEqual(ocr._pyramid(self.image(1400, 2100)), [3072, 2100, 1280])

    def test_a_small_image_is_also_read_enlarged(self):
        self.assertEqual(ocr._pyramid(self.image(500, 600)), [3072, 2048, 1280, 600])


class ScaleMerge(unittest.TestCase):
    def block(self, text, confidence, x, y, w, h):
        quad = [[x, y], [x + w, y], [x + w, y + h], [x, y + h]]
        return {"text": text, "confidence": confidence, "quad": quad}

    def test_a_whole_line_beats_a_better_scored_fragment_of_it(self):
        line = self.block("THE HUNGER GAMES", 0.95, 0, 0, 160, 20)
        fragment = self.block("HUN", 0.999, 50, 2, 30, 16)
        kept = ocr._merge_scales([fragment, line])
        self.assertEqual([b["text"] for b in kept], ["THE HUNGER GAMES"])

    def test_separate_lines_are_all_kept(self):
        top = self.block("COLLINS", 0.99, 0, 0, 70, 20)
        bottom = self.block("THE REAPING", 0.98, 0, 40, 110, 20)
        self.assertEqual(len(ocr._merge_scales([top, bottom])), 2)


if __name__ == "__main__":
    unittest.main()
