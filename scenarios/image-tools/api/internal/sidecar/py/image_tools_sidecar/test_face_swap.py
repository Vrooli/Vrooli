"""face_swap contracts with ONNX-session stand-ins; no model download or GPU."""
import os
import tempfile
import unittest
from types import SimpleNamespace
from unittest import mock

import numpy as np
from PIL import Image, ImageFilter

from . import face_swap
from .test_face_restore import FakeDetector, yunet_outputs

# Five landmarks of an upright face centred on (320, 250), eyes 40px apart.
MARKS = [(300, 230), (340, 230), (320, 250), (305, 270), (335, 270)]


def detector_with(faces):
    """A YuNet stand-in that reports every (row, col, size, marks) in ``faces``."""
    outputs = yunet_outputs(16, *faces[0])
    for row, col, wh, marks in faces[1:]:
        extra = yunet_outputs(16, row, col, wh, marks)
        for key in outputs:
            outputs[key] = outputs[key] + extra[key]
    return FakeDetector(outputs)


class FakeEmbedder:
    """ArcFace stand-in: a fixed 512-d vector regardless of the face."""

    def __init__(self, vector):
        self.vector, self.seen = np.asarray(vector, np.float32), None

    def get_inputs(self):
        return [SimpleNamespace(name="input", shape=[1, 3, 112, 112])]

    def get_outputs(self):
        return [SimpleNamespace(name="embedding")]

    def run(self, _names, feed):
        self.seen = feed["input"]
        return [self.vector.reshape(1, -1)]


class FakeSwapper:
    """Swapper stand-in: paints a constant colour and records its inputs."""

    def __init__(self, size=256, value=0.0):
        self.size, self.value, self.feed = size, value, None

    def get_inputs(self):
        return [SimpleNamespace(name="source", shape=[1, 512]), SimpleNamespace(name="target", shape=[1, 3, self.size, self.size])]

    def get_outputs(self):
        return [SimpleNamespace(name="output")]

    def run(self, _names, feed):
        self.feed = feed
        return [np.full((1, 3, self.size, self.size), self.value, np.float32)]


class FakeOccluder:
    """XSeg stand-in: face visible everywhere except the left half."""

    def get_inputs(self):
        return [SimpleNamespace(name="input", shape=[1, 256, 256, 3])]

    def get_outputs(self):
        return [SimpleNamespace(name="mask")]

    def run(self, _names, feed):
        assert feed["input"].shape == (1, 256, 256, 3)
        mask = np.ones((1, 256, 256, 1), np.float32)
        mask[:, :, :128] = 0.0
        return [mask]


def graphs_with(family="hyperswap", swapper=None, occluder=None, embedding=None, detector=None):
    swapper = swapper or FakeSwapper(face_swap.PROFILES[family]["size"])
    return face_swap.Graphs(
        detector,
        FakeEmbedder(np.arange(1, 513) if embedding is None else embedding),
        swapper,
        occluder,
        dict(face_swap.PROFILES[family], family=family),
        "/models/%s.onnx" % family,
    )


class ProfileContracts(unittest.TestCase):
    def test_family_is_taken_from_the_swapper_filename(self):
        self.assertEqual(face_swap.profile_for("hyperswap_1a_256.onnx")[0], "hyperswap")
        self.assertEqual(face_swap.profile_for("/m/inswapper_128_fp16.onnx")[0], "inswapper")
        self.assertEqual(face_swap.profile_for("alphaface_256.onnx")[0], "alphaface")
        self.assertEqual(face_swap.profile_for("mystery.onnx"), (None, None))

    def test_every_profile_declares_the_fields_the_pipeline_reads(self):
        for family, profile in face_swap.PROFILES.items():
            self.assertEqual(set(profile), {"size", "mean", "std", "embedding", "denormalize"}, family)
            self.assertIn(profile["embedding"], {"emap", "unit", "raw"}, family)

    def test_roles_are_assigned_by_onnx_signature_not_filename(self):
        sessions = {
            "face_detection_yunet_2023mar.onnx": detector_with([(15, 19, 80, MARKS)]),
            "arcface_w600k_r50.onnx": FakeEmbedder(np.ones(512)),
            "hyperswap_1a_256.onnx": FakeSwapper(256),
            "xseg_1.onnx": FakeOccluder(),
        }
        with tempfile.TemporaryDirectory() as directory:
            for name in sessions:
                open(os.path.join(directory, name), "wb").close()
            with mock.patch.object(face_swap._common, "make_session", lambda path: sessions[os.path.basename(path)]):
                graphs = face_swap.resolve_graphs(directory)
        self.assertIs(graphs.swapper, sessions["hyperswap_1a_256.onnx"])
        self.assertIs(graphs.embedder, sessions["arcface_w600k_r50.onnx"])
        self.assertIs(graphs.occluder, sessions["xseg_1.onnx"])
        self.assertIs(graphs.detector, sessions["face_detection_yunet_2023mar.onnx"])
        self.assertEqual(graphs.profile["family"], "hyperswap")

    def test_missing_embedder_fails_loud(self):
        with tempfile.TemporaryDirectory() as directory:
            open(os.path.join(directory, "hyperswap_1a_256.onnx"), "wb").close()
            with mock.patch.object(face_swap._common, "make_session", lambda path: FakeSwapper(256)):
                with self.assertRaises(SystemExit) as raised:
                    face_swap.resolve_graphs(directory)
        self.assertEqual(raised.exception.code, 4)


class EmbeddingContracts(unittest.TestCase):
    def setUp(self):
        self.raw = np.arange(1, 513, dtype=np.float32)
        self.unit = self.raw / np.linalg.norm(self.raw)

    def test_raw_family_feeds_the_unnormalised_vector(self):
        out = face_swap.source_embedding(np, graphs_with("alphaface"), self.raw, self.unit)
        np.testing.assert_allclose(out, self.raw.reshape(1, -1))

    def test_unit_family_feeds_the_normalised_vector(self):
        out = face_swap.source_embedding(np, graphs_with("hyperswap"), self.raw, self.unit)
        self.assertAlmostEqual(float(np.linalg.norm(out)), 1.0, places=5)

    def test_emap_family_projects_then_renormalises(self):
        emap = np.eye(512, dtype=np.float32) * 2.0
        with mock.patch.object(face_swap, "_emap", lambda path: emap):
            out = face_swap.source_embedding(np, graphs_with("inswapper"), self.raw, self.unit)
        self.assertAlmostEqual(float(np.linalg.norm(out)), 1.0, places=5)
        np.testing.assert_allclose(out.ravel(), self.unit, atol=1e-5)


class MaskContracts(unittest.TestCase):
    def test_box_mask_is_opaque_inside_and_transparent_at_the_border(self):
        mask = face_swap.box_mask(np, Image, ImageFilter, 256)
        self.assertEqual(mask.shape, (256, 256))
        self.assertGreater(float(mask[128, 128]), 0.99)
        self.assertLess(float(mask[0, 0]), 0.05)
        self.assertLess(float(mask[128, 0]), 0.2)

    def test_occlusion_mask_protects_what_the_occluder_hides(self):
        crop = np.full((256, 256, 3), 120, np.uint8)
        mask = face_swap.occlusion_mask(np, Image, ImageFilter, graphs_with(occluder=FakeOccluder()), crop)
        self.assertLess(float(mask[128, 20]), 0.05)  # hidden: keep the original pixel
        self.assertGreater(float(mask[128, 236]), 0.95)  # visible: take the swap

    def test_colour_match_pulls_the_swap_toward_the_original_statistics(self):
        swapped = np.full((64, 64, 3), (200, 60, 60), np.uint8)
        original = np.full((64, 64, 3), (60, 60, 200), np.uint8)
        full = np.ones((64, 64), np.float32)
        matched = face_swap.match_colour(np, swapped, original, full, 1.0)
        self.assertLess(int(abs(int(matched[10, 10, 0]) - 60)), 6)
        self.assertLess(int(abs(int(matched[10, 10, 2]) - 200)), 6)
        np.testing.assert_array_equal(face_swap.match_colour(np, swapped, original, full, 0.0), swapped)


class SwapImageContracts(unittest.TestCase):
    def target(self):
        return Image.fromarray(np.full((480, 640, 3), (200, 40, 40), np.uint8))

    def reference(self):
        return Image.fromarray(np.full((480, 640, 3), (40, 40, 200), np.uint8))

    def test_swaps_only_the_face_region_and_keeps_the_target_size(self):
        graphs = graphs_with(detector=detector_with([(15, 19, 80, MARKS)]))
        out, summary = face_swap.swap_image(np, Image, ImageFilter, graphs, self.target(), self.reference(), {"color_match": 0})
        self.assertEqual(out.size, (640, 480))
        arr = np.asarray(out)
        np.testing.assert_array_equal(arr[5, 5], [200, 40, 40])  # far from the face
        self.assertTrue(np.allclose(arr[255, 320], [128, 128, 128], atol=3))  # face centre swapped (mid grey)
        self.assertEqual((summary["faces_swapped"], summary["target_faces"], summary["reference_faces"]), (1, 1, 1))
        self.assertEqual(summary["swapper"], "hyperswap")

    def test_the_swapper_receives_the_source_identity_and_a_target_crop_in_its_range(self):
        swapper = FakeSwapper(256)
        graphs = graphs_with(swapper=swapper, detector=detector_with([(15, 19, 80, MARKS)]))
        face_swap.swap_image(np, Image, ImageFilter, graphs, self.target(), self.reference(), {})
        self.assertEqual(swapper.feed["source"].shape, (1, 512))
        self.assertEqual(swapper.feed["target"].shape, (1, 3, 256, 256))
        self.assertGreaterEqual(float(swapper.feed["target"].min()), -1.0)  # hyperswap range
        self.assertLessEqual(float(swapper.feed["target"].max()), 1.0)

    def test_occlusion_mask_keeps_the_original_where_the_face_is_covered(self):
        graphs = graphs_with(occluder=FakeOccluder(), detector=detector_with([(15, 19, 80, MARKS)]))
        out, summary = face_swap.swap_image(np, Image, ImageFilter, graphs, self.target(), self.reference(), {"color_match": 0})
        arr = np.asarray(out).astype(int)
        self.assertTrue(summary["occlusion_mask"])
        self.assertLess(abs(arr[255, 298, 0] - 200), 30)  # covered (left of the face): original red kept
        self.assertTrue(np.allclose(arr[255, 345], [128, 128, 128], atol=6))  # visible: swapped
        off = face_swap.swap_image(np, Image, ImageFilter, graphs, self.target(), self.reference(), {"color_match": 0, "occlusion": False})[0]
        self.assertTrue(np.allclose(np.asarray(off)[255, 298], [128, 128, 128], atol=6))

    def test_target_face_index_orders_by_size_and_all_faces_swaps_every_face(self):
        big, small = (15, 19, 80, MARKS), (30, 8, 40, [(130, 470), (150, 470), (140, 480), (132, 490), (148, 490)])
        graphs = graphs_with(detector=detector_with([big, small]))
        target = self.target()
        first = np.asarray(face_swap.swap_image(np, Image, ImageFilter, graphs, target, self.reference(), {"color_match": 0})[0])
        self.assertTrue(np.allclose(first[255, 320], [128, 128, 128], atol=3))
        self.assertEqual(tuple(first[480 - 1, 140]), (200, 40, 40))  # the smaller face untouched by default
        second, summary = face_swap.swap_image(np, Image, ImageFilter, graphs, target, self.reference(), {"color_match": 0, "target_face": 1})
        self.assertTrue(np.allclose(np.asarray(second)[255, 320], [200, 40, 40], atol=1))  # largest left alone
        self.assertEqual(summary["faces_swapped"], 1)
        every = face_swap.swap_image(np, Image, ImageFilter, graphs, target, self.reference(), {"color_match": 0, "all_faces": True})[1]
        self.assertEqual(every["faces_swapped"], 2)

    def test_face_strength_is_met_by_measuring_the_identity_the_swapper_produced(self):
        """A swapper that saturates: the face is already fully the source's at 40%
        of the embedding mix. Half strength must land at half the identity
        distance anyway, so the weight is found by measurement, not set to 0.5."""

        class SaturatingSwapper(FakeSwapper):
            def run(self, _names, feed):
                self.feed = feed
                source = feed["source"].reshape(-1)
                share = float(source[0] / (source[0] + source[1]))
                colour = min(1.0, 2.5 * share)  # the painted face, 0 = target's, 1 = source's
                return [np.full((1, 3, self.size, self.size), 2.0 * colour - 1.0, np.float32)]

        class ColourEmbedder(FakeEmbedder):
            """ArcFace stand-in: a white face is person 0, a black face person 1, grey is between."""

            def run(self, _names, feed):
                colour = (float(feed["input"].mean()) + 1.0) / 2.0
                vector = np.zeros((1, 512), np.float32)
                vector[0, 0], vector[0, 1] = colour, 1.0 - colour
                return [vector]

        def swap(strength):
            graphs = graphs_with(swapper=SaturatingSwapper(256), detector=detector_with([(15, 19, 80, MARKS)]))
            graphs.embedder = ColourEmbedder(np.zeros(512))
            white = Image.fromarray(np.full((480, 640, 3), 255, np.uint8))
            black = Image.fromarray(np.zeros((480, 640, 3), np.uint8))
            return face_swap.swap_image(np, Image, ImageFilter, graphs, black, white, {"color_match": 0, "face_strength": strength})[1]

        half = swap(0.5)
        self.assertEqual(half["face_strength"], 0.5)
        self.assertAlmostEqual(half["identity_progress"], 0.5, delta=0.05)
        self.assertAlmostEqual(half["identity_mix"], 0.2, delta=0.03)  # not the naive 0.5
        quarter = swap(0.25)
        self.assertLess(quarter["identity_mix"], half["identity_mix"])
        self.assertNotIn("identity_mix", swap(1.0))  # the full swap needs no calibration

    def test_solve_mix_inverts_a_saturating_curve_and_keeps_the_ends(self):
        curve = lambda w: min(1.0, 3.0 * w)  # noqa: E731
        for target in (0.25, 0.5, 0.75):
            weight = face_swap.solve_mix(curve, target)
            self.assertAlmostEqual(curve(weight), target, delta=0.06)
        self.assertEqual(face_swap.solve_mix(curve, 0.0), 0.0)
        self.assertGreaterEqual(curve(face_swap.solve_mix(curve, 1.0)), 0.95)

    def test_two_near_identical_faces_fall_back_to_the_requested_weight(self):
        """With nothing to measure the identity span is below the floor, so the
        strength is used as the mix weight directly."""
        graphs = graphs_with(detector=detector_with([(15, 19, 80, MARKS)]))
        summary = face_swap.swap_image(np, Image, ImageFilter, graphs, self.target(), self.reference(), {"color_match": 0, "face_strength": 0.4})[1]
        self.assertEqual(summary["identity_mix"], 0.4)

    def test_a_face_index_out_of_range_is_a_usage_error(self):
        graphs = graphs_with(detector=detector_with([(15, 19, 80, MARKS)]))
        for options in ({"target_face": 3}, {"source_face": 2}):
            with self.assertRaises(SystemExit) as raised:
                face_swap.swap_image(np, Image, ImageFilter, graphs, self.target(), self.reference(), options)
            self.assertEqual(raised.exception.code, 2)

    def test_no_face_in_either_image_fails_loud(self):
        quiet = FakeDetector(yunet_outputs(16, 0, 0, 16, [(0, 0)] * 5))
        quiet.outputs["cls_16"][:] = 0
        with self.assertRaises(SystemExit) as raised:
            face_swap.swap_image(np, Image, ImageFilter, graphs_with(detector=quiet), self.target(), self.reference(), {})
        self.assertEqual(raised.exception.code, 9)


if __name__ == "__main__":
    unittest.main()
