"""pose_extract contracts with ONNX-session stand-ins; no model download or GPU."""
import unittest
from types import SimpleNamespace

import numpy as np
from PIL import Image, ImageDraw

from . import _common, pose_extract

POSE_HW = (384, 288)


class FakeSession:
    def __init__(self, shape, outputs, name="input"):
        self.shape, self.outputs, self.name, self.feed = shape, outputs, name, None

    def get_inputs(self):
        return [SimpleNamespace(name=self.name, shape=self.shape)]

    def run(self, _names, feed):
        self.feed = feed
        return self.outputs


def simcc_for(points, hw=POSE_HW, conf=1.0):
    """SimCC x/y heatmaps with one spike per keypoint at crop-input coordinates ``points``."""
    height, width = hw
    sx = np.zeros((1, 133, int(width * 2)), np.float32)
    sy = np.zeros((1, 133, int(height * 2)), np.float32)
    for i in range(133):
        x, y = points.get(i, (0, 0))
        c = conf if i in points else 0.0
        sx[0, i, int(x * 2)] = c
        sy[0, i, int(y * 2)] = c
    return [sx, sy]


class CropGeometry(unittest.TestCase):
    def test_crop_matches_estimator_aspect_and_pads(self):
        centre, scale = pose_extract.crop_scale([100, 100, 200, 400], 288 / 384)
        self.assertEqual(centre, (150.0, 250.0))
        self.assertAlmostEqual(scale[0] / scale[1], 288 / 384)
        self.assertGreaterEqual(scale[1], 300 * pose_extract.POSE_PADDING - 1e-6)

    def test_wide_box_grows_height_not_width(self):
        _, scale = pose_extract.crop_scale([0, 0, 400, 100], 288 / 384)
        self.assertAlmostEqual(scale[0], 400 * pose_extract.POSE_PADDING)
        self.assertGreater(scale[1], 100 * pose_extract.POSE_PADDING)

    def test_crop_outside_the_image_takes_the_page_colour(self):
        page = Image.new("RGB", (100, 100), (250, 250, 250))
        crop = pose_extract.crop_person(np, Image, page, (0.0, 0.0), (80.0, 106.0), POSE_HW)
        self.assertEqual(crop.size, (288, 384))
        self.assertGreater(np.asarray(crop).min(), 240)


class Decode(unittest.TestCase):
    def test_keypoint_lands_back_in_image_pixels(self):
        sx, sy = simcc_for({0: (144, 192)})
        whole = pose_extract.decode_simcc(np, sx, sy, (500.0, 400.0), (288.0, 384.0), POSE_HW)
        self.assertAlmostEqual(whole[0][0], 500.0, delta=1.0)
        self.assertAlmostEqual(whole[0][1], 400.0, delta=1.0)
        self.assertGreater(whole[0][2], 0.9)
        self.assertEqual(whole[1][2], 0.0)

    def test_body18_names_and_neck(self):
        whole = np.zeros((133, 3))
        whole[5] = (80, 50, 0.9)   # left shoulder
        whole[6] = (120, 50, 0.8)  # right shoulder
        whole[9] = (60, 120, 0.7)  # left wrist
        whole[10] = (140, 118, 0.6)  # right wrist
        rows = pose_extract.body18(np, whole)
        self.assertEqual(len(rows), 18)
        names = pose_extract.BODY_NAMES
        self.assertEqual(rows[names.index("neck")][:2], [100.0, 50.0])
        self.assertAlmostEqual(rows[names.index("neck")][2], 0.8)
        self.assertEqual(rows[names.index("left_wrist")], [60.0, 120.0, 0.7])
        self.assertEqual(rows[names.index("right_wrist")], [140.0, 118.0, 0.6])

    def test_neck_needs_both_shoulders(self):
        whole = np.zeros((133, 3))
        whole[5] = (80, 50, 0.9)
        rows = pose_extract.body18(np, whole)
        self.assertEqual(rows[pose_extract.BODY_NAMES.index("neck")][2], 0.0)


class Detector(unittest.TestCase):
    def raw(self, cx, cy, w, h, score, cls_scores=1.0):
        """One YOLOX output row at stride 8 grid cell (cx/8, cy/8)."""
        rows = np.zeros((8400, 85), np.float32)
        gx, gy = int(cx // 8), int(cy // 8)
        index = gy * 80 + gx
        rows[:, 2:4] = -20.0
        rows[index, 0], rows[index, 1] = cx / 8 - gx, cy / 8 - gy
        rows[index, 2], rows[index, 3] = np.log(w / 8), np.log(h / 8)
        rows[index, 4], rows[index, 5] = score, cls_scores
        return rows[None]

    def test_person_box_is_scaled_back_to_the_image(self):
        session = FakeSession([1, 3, 640, 640], [self.raw(320, 320, 160, 320, 0.9)])
        image = Image.new("RGB", (1280, 1280), "white")  # ratio 0.5
        found = pose_extract.detect_people(np, Image, session, image)
        self.assertEqual(len(found), 1)
        x0, y0, x1, y1, score = found[0]
        self.assertAlmostEqual(x0, 480, delta=10)
        self.assertAlmostEqual(y1 - y0, 640, delta=10)
        self.assertAlmostEqual(score, 0.9, places=3)
        feed = next(iter(session.feed.values()))
        self.assertEqual(feed.shape, (1, 3, 640, 640))

    def test_low_score_finds_nobody(self):
        session = FakeSession([1, 3, 640, 640], [self.raw(320, 320, 160, 320, 0.1)])
        self.assertEqual(pose_extract.detect_people(np, Image, session, Image.new("RGB", (640, 640))), [])

    def test_overlapping_boxes_collapse_to_the_best(self):
        keep = pose_extract.nms(np, np.array([[0, 0, 100, 100], [5, 5, 105, 105], [300, 300, 400, 400]], float), np.array([0.6, 0.9, 0.5]), 0.45)
        self.assertEqual(keep, [1, 2])


class Foreground(unittest.TestCase):
    def test_drawing_box_ignores_the_blank_page(self):
        page = Image.new("RGB", (400, 600), "white")
        ImageDraw.Draw(page).rectangle([100, 150, 250, 480], outline="black", width=3)
        box = pose_extract.foreground_box(np, page)
        self.assertEqual([round(v) for v in box], [100, 150, 251, 481])

    def test_blank_page_has_none(self):
        self.assertIsNone(pose_extract.foreground_box(np, Image.new("RGB", (50, 50), "white")))


class LineArt(unittest.TestCase):
    def drawing(self):
        page = Image.new("RGB", (400, 600), "white")
        draw = ImageDraw.Draw(page)
        draw.ellipse([150, 150, 250, 450], outline="black", width=3)  # a closed outline
        return page

    def test_a_drawing_is_line_art_and_a_photo_is_not(self):
        self.assertTrue(pose_extract.is_line_art(np, self.drawing()))
        photo = Image.fromarray(np.random.default_rng(1).integers(0, 255, (200, 200, 3), dtype=np.uint8))
        self.assertFalse(pose_extract.is_line_art(np, photo))
        self.assertFalse(pose_extract.is_line_art(np, Image.new("RGB", (200, 200), (200, 40, 40))))

    def test_outline_becomes_a_filled_figure_and_the_page_stays_white(self):
        figure = np.asarray(pose_extract.line_art_to_figure(np, Image, self.drawing()))
        centre, corner = figure[300, 200], figure[10, 10]
        self.assertTrue((corner > 250).all(), corner)
        self.assertLess(int(centre.max()), 240)  # filled, no longer page colour
        self.assertGreater(int(centre[0]) - int(centre[2]), 20)  # skin toned (warm)

    def test_prepare_modes_decide_what_the_estimator_sees(self):
        seen = []

        class Recorder(FakeSession):
            def run(self, names, feed):
                seen.append(next(iter(feed.values())).copy())
                return super().run(names, feed)

        def estimator():
            return Recorder([1, 3, *POSE_HW], simcc_for({0: (144, 96)}))

        page = self.drawing()
        for mode in ("none", "auto", "line_art"):
            pose_extract.extract(np, Image, pose_extract.Graphs(None, estimator()), page, {"prepare": mode, "box": [140, 140, 260, 460]})
        raw, auto, forced = seen
        self.assertFalse(np.allclose(raw, auto), "auto prepares a drawing")
        self.assertTrue(np.allclose(auto, forced), "auto on line art equals line_art")
        with self.assertRaises(_common.InputError):
            pose_extract.extract(np, Image, pose_extract.Graphs(None, estimator()), page, {"prepare": "sketch"})

    def test_prepared_keypoints_come_back_in_page_pixels(self):
        pose = FakeSession([1, 3, *POSE_HW], simcc_for({0: (144, 192)}))
        document = pose_extract.extract(np, Image, pose_extract.Graphs(None, pose), self.drawing(), {"box": [150, 150, 250, 450]})
        nose = document["people"][0]["body"][0]
        self.assertAlmostEqual(nose[0], 200, delta=12)
        self.assertAlmostEqual(nose[1], 300, delta=12)


class Extract(unittest.TestCase):
    def graphs(self, detector=None):
        pose = FakeSession([1, 3, *POSE_HW], simcc_for({0: (144, 96), 5: (100, 150), 6: (188, 150), 91: (10, 20)}))
        return pose_extract.Graphs(detector, pose)

    def test_drawing_falls_back_to_its_foreground_box(self):
        page = Image.new("RGB", (400, 600), "white")
        ImageDraw.Draw(page).rectangle([100, 150, 250, 480], outline="black", width=3)
        document = pose_extract.extract(np, Image, self.graphs(), page, {})
        person = document["people"][0]
        self.assertEqual(person["box_source"], "foreground")
        self.assertEqual(len(person["body"]), 18)
        self.assertEqual(len(person["left_hand"]), 21)
        self.assertGreater(person["body"][0][2], 0.9)
        self.assertEqual(document["body_keypoints"][1], "neck")
        self.assertEqual(document["image"], {"width": 400, "height": 600})

    def test_detector_boxes_win_over_the_page(self):
        detector = FakeSession([1, 3, 640, 640], [Detector().raw(320, 320, 160, 320, 0.9)])
        document = pose_extract.extract(np, Image, self.graphs(detector), Image.new("RGB", (640, 640), "white"), {})
        self.assertEqual(document["people"][0]["box_source"], "detector")

    def test_given_box_is_used_and_validated(self):
        page = Image.new("RGB", (400, 600), "white")
        document = pose_extract.extract(np, Image, self.graphs(), page, {"box": [10, 20, 210, 420]})
        self.assertEqual(document["people"][0]["box_source"], "given")
        with self.assertRaises(_common.InputError):
            pose_extract.extract(np, Image, self.graphs(), page, {"box": [10, 20, 5, 420]})

    def test_blank_page_is_refused_as_given(self):
        with self.assertRaises(_common.InputError):
            pose_extract.extract(np, Image, self.graphs(), Image.new("RGB", (80, 80), "white"), {})


if __name__ == "__main__":
    unittest.main()
