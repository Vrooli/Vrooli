"""face_restore contracts with ONNX-session stand-ins; no model download or GPU."""
import unittest
from types import SimpleNamespace

import numpy as np
from PIL import Image, ImageFilter

from . import face_restore

STRIDES = face_restore.DETECTOR_STRIDES
SIZE = face_restore.DETECTOR_SIZE


def yunet_outputs(cell_stride, row, col, box_wh, landmarks):
    """YuNet heads with one confident face at (row, col) of one stride."""
    out = {}
    for stride in STRIDES:
        n = (SIZE // stride) ** 2
        out[f"cls_{stride}"] = np.zeros((1, n, 1), np.float32)
        out[f"obj_{stride}"] = np.zeros((1, n, 1), np.float32)
        out[f"bbox_{stride}"] = np.zeros((1, n, 4), np.float32)
        out[f"kps_{stride}"] = np.zeros((1, n, 10), np.float32)
    idx = row * (SIZE // cell_stride) + col
    out[f"cls_{cell_stride}"][0, idx, 0] = 0.9
    out[f"obj_{cell_stride}"][0, idx, 0] = 0.9
    out[f"bbox_{cell_stride}"][0, idx] = [0.5, 0.5, np.log(box_wh / cell_stride), np.log(box_wh / cell_stride)]
    kps = [(x / cell_stride - col, y / cell_stride - row) for x, y in landmarks]
    out[f"kps_{cell_stride}"][0, idx] = np.array(kps, np.float32).reshape(-1)
    return out


class FakeDetector:
    def __init__(self, outputs):
        self.outputs, self.seen = outputs, None

    def get_inputs(self):
        return [SimpleNamespace(name="input", shape=[1, 3, SIZE, SIZE])]

    def get_outputs(self):
        return [SimpleNamespace(name=name) for name in self.outputs]

    def run(self, _names, feed):
        self.seen = feed["input"]
        return list(self.outputs.values())


class FakeRestorer:
    """Returns a constant mid-grey 'restored' face; records the fidelity weight."""

    def __init__(self, with_weight, value=0.0, colour=False):
        self.with_weight, self.value, self.colour, self.weight = with_weight, value, colour, None

    def get_inputs(self):
        inputs = [SimpleNamespace(name="input", shape=[1, 3, 512, 512])]
        if self.with_weight:
            inputs.append(SimpleNamespace(name="weight", shape=[]))
        return inputs

    def get_outputs(self):
        return [SimpleNamespace(name="output")]

    def run(self, _names, feed):
        assert feed["input"].shape == (1, 3, 512, 512)
        self.weight = feed.get("weight")
        out = np.full((1, 3, 512, 512), self.value, np.float32)
        if self.colour:
            out[0, 2] = 1.0  # strong blue cast, as CodeFormer invents on B&W
        return [out]


class FaceRestoreContracts(unittest.TestCase):
    def test_similarity_transform_recovers_scale_rotation_translation(self):
        template = np.array(face_restore.FFHQ_TEMPLATE_512)
        theta, scale, shift = 0.3, 0.25, np.array([40.0, 70.0])
        rot = np.array([[np.cos(theta), -np.sin(theta)], [np.sin(theta), np.cos(theta)]])
        src = (template @ rot.T) * scale + shift
        m = face_restore.similarity_transform(np, src, template)
        mapped = (m @ np.c_[src, np.ones(5)].T).T[:, :2]
        np.testing.assert_allclose(mapped, template, atol=1e-6)

    def test_decode_yunet_maps_cell_offsets_and_scale_back_to_source(self):
        marks = [(300, 310), (340, 310), (320, 330), (305, 350), (335, 350)]
        outputs = yunet_outputs(16, row=20, col=19, box_wh=64, landmarks=marks)
        faces = face_restore.decode_yunet(np, outputs, scale=2.0)
        self.assertEqual(len(faces), 1)
        x1, y1, x2, y2 = faces[0]["box"]
        self.assertAlmostEqual((x1 + x2) / 2, (19 + 0.5) * 16 * 2.0, places=3)
        self.assertAlmostEqual(x2 - x1, 64 * 2.0, places=3)
        np.testing.assert_allclose(faces[0]["landmarks"], np.array(marks) * 2.0, atol=1e-3)
        self.assertGreater(faces[0]["score"], 0.85)

    def test_decode_yunet_drops_low_scores_and_suppresses_duplicates(self):
        marks = [(300, 310), (340, 310), (320, 330), (305, 350), (335, 350)]
        outputs = yunet_outputs(16, 20, 19, 64, marks)
        outputs["cls_16"][0, 20 * 40 + 20, 0] = 0.8  # overlapping neighbour
        outputs["obj_16"][0, 20 * 40 + 20, 0] = 0.8
        outputs["bbox_16"][0, 20 * 40 + 20] = [0.0, 0.5, np.log(4.0), np.log(4.0)]
        self.assertEqual(len(face_restore.decode_yunet(np, outputs, 1.0)), 1)
        self.assertEqual(face_restore.decode_yunet(np, outputs, 1.0, score_threshold=0.95), [])

    def test_restore_image_pastes_only_the_face_region_and_feeds_fidelity(self):
        image = Image.fromarray(np.full((480, 640, 3), (200, 40, 40), np.uint8))
        marks = [(300, 230), (340, 230), (320, 250), (305, 270), (335, 270)]
        detector = FakeDetector(yunet_outputs(16, 15, 19, 80, marks))
        restorer = FakeRestorer(with_weight=True, value=0.0)
        out, count = face_restore.restore_image(np, Image, ImageFilter, detector, restorer, image, fidelity=0.7)
        self.assertEqual(count, 1)
        self.assertEqual(out.size, image.size)
        self.assertAlmostEqual(float(restorer.weight), 0.7)
        arr = np.asarray(out)
        np.testing.assert_array_equal(arr[5, 5], [200, 40, 40])  # far from the face
        self.assertTrue(np.allclose(arr[250, 320], [128, 128, 128], atol=2))  # face centre restored

    def test_monochrome_source_keeps_only_restored_luminance(self):
        image = Image.fromarray(np.full((480, 640, 3), 90, np.uint8))
        self.assertTrue(face_restore.is_monochrome(np, image))
        marks = [(300, 230), (340, 230), (320, 250), (305, 270), (335, 270)]
        detector = FakeDetector(yunet_outputs(16, 15, 19, 80, marks))
        out, _ = face_restore.restore_image(np, Image, ImageFilter, detector, FakeRestorer(False, colour=True), image)
        arr = np.asarray(out).astype(int)
        self.assertEqual(int(np.abs(arr[..., 0] - arr[..., 2]).max()), 0)

    def test_no_face_returns_zero_count(self):
        image = Image.fromarray(np.zeros((100, 100, 3), np.uint8))
        detector = FakeDetector(yunet_outputs(16, 0, 0, 16, [(0, 0)] * 5))
        detector.outputs["cls_16"][:] = 0
        _out, count = face_restore.restore_image(np, Image, ImageFilter, detector, FakeRestorer(False), image)
        self.assertEqual(count, 0)


if __name__ == "__main__":
    unittest.main()
