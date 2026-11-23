"""Decision contracts of the bundled minor-safety reviewer.

The models are replaced by stubs: these tests pin what the reviewer DOES with
model outputs (when it refuses, when it does not, what it reports), not what the
weights predict. Measured detector behaviour on real faces lives in
scripts/minor-review-eval.py and docs/internal/SECURITY.md.
"""

import base64
import io
import json
import unittest
from unittest import mock

import numpy as np
from PIL import Image

from image_tools_sidecar import minor_review as mr

LABELS = ["0-2", "3-9", "10-19", "20-29", "30-39", "40-49", "50-59", "60-69", "more than 70"]


def face(size=120.0, p_under_20=None):
    row = {"size_px": size, "box": [0, 0, size, size]}
    if p_under_20 is not None:
        row["p_under_20"] = p_under_20
    return row


def png_b64(color=(120, 90, 70), size=(64, 64)):
    buf = io.BytesIO()
    Image.new("RGB", size, color).save(buf, format="PNG")
    return base64.b64encode(buf.getvalue()).decode()


class FaceCropTests(unittest.TestCase):
    def test_crop_is_square_centred_and_scaled_by_the_margin(self):
        left, top, right, bottom = mr.face_crop_box((100, 100, 200, 200), 1000, 1000, margin=2.0)
        self.assertEqual((right - left, bottom - top), (200, 200))
        self.assertEqual(((left + right) / 2, (top + bottom) / 2), (150, 150))

    def test_crop_clamps_at_the_raster_border_instead_of_shifting(self):
        left, top, right, bottom = mr.face_crop_box((0, 0, 100, 100), 150, 150, margin=2.0)
        self.assertEqual((left, top), (0, 0))
        self.assertLessEqual(right, 150)
        self.assertLessEqual(bottom, 150)

    def test_degenerate_box_has_no_crop(self):
        self.assertIsNone(mr.face_crop_box((10, 10, 10.5, 10.5), 1000, 1000, margin=1.0))


class AgeMassTests(unittest.TestCase):
    def test_mass_sums_the_under_twenty_bins_only(self):
        probs = [0.05, 0.10, 0.20, 0.30, 0.15, 0.10, 0.05, 0.03, 0.02]
        self.assertAlmostEqual(mr.under_age_mass(LABELS, probs), 0.35)

    def test_the_whole_ten_to_nineteen_bin_counts_as_minor_mass(self):
        # 18-19 year olds sit in this bin and the model cannot split it: the error
        # is deliberately one-sided toward refusing.
        probs = [0, 0, 1.0, 0, 0, 0, 0, 0, 0]
        self.assertGreaterEqual(mr.under_age_mass(LABELS, probs), mr.MINOR_THRESHOLD)


class JudgeImageTests(unittest.TestCase):
    def test_sexual_image_with_a_minor_appearing_face_is_refused(self):
        j = mr.judge_image(0.95, [face(p_under_20=0.80)])
        self.assertTrue(j["sexual"] and j["minor_face"] and j["refuse"])

    def test_sexual_image_of_an_adult_passes(self):
        j = mr.judge_image(0.95, [face(p_under_20=0.02)])
        self.assertTrue(j["sexual"])
        self.assertFalse(j["refuse"])

    def test_non_sexual_portrait_of_a_child_is_not_refused_but_is_reported(self):
        j = mr.judge_image(0.01, [face(p_under_20=0.97)])
        self.assertTrue(j["minor_face"])
        self.assertFalse(j["sexual"])
        self.assertFalse(j["refuse"])

    def test_sexual_image_with_no_face_is_refused_as_unassessable(self):
        j = mr.judge_image(0.95, [])
        self.assertTrue(j["unassessable"] and j["refuse"])

    def test_sexual_image_whose_only_faces_are_too_small_to_read_is_unassessable(self):
        j = mr.judge_image(0.95, [face(size=mr.MIN_FACE_PX - 1)])
        self.assertTrue(j["unassessable"] and j["refuse"])

    def test_a_tiny_background_face_does_not_block_an_assessed_adult(self):
        j = mr.judge_image(0.95, [face(p_under_20=0.01), face(size=10)])
        self.assertFalse(j["refuse"])
        self.assertEqual(j["assessed_faces"], 1)

    def test_any_minor_appearing_face_among_several_refuses(self):
        j = mr.judge_image(0.95, [face(p_under_20=0.01), face(p_under_20=0.60)])
        self.assertTrue(j["refuse"])

    def test_thresholds_are_inclusive(self):
        self.assertTrue(mr.judge_image(mr.SEXUAL_THRESHOLD, [face(p_under_20=mr.MINOR_THRESHOLD)])["refuse"])
        self.assertFalse(mr.judge_image(mr.SEXUAL_THRESHOLD - 0.01, [face(p_under_20=1.0)])["refuse"])

    def test_flags_only_lower_the_sexual_bar(self):
        self.assertEqual(mr.sexual_threshold(False, False), mr.SEXUAL_THRESHOLD)
        self.assertEqual(mr.sexual_threshold(True, False), mr.STRICT_SEXUAL_THRESHOLD)
        self.assertEqual(mr.sexual_threshold(False, True), 0.0)
        self.assertLess(mr.STRICT_SEXUAL_THRESHOLD, mr.SEXUAL_THRESHOLD)

    def test_either_classifier_can_call_an_image_sexual(self):
        # The precise classifier alone, and the high-recall one alone, each cross
        # their own bar; neither is required to agree.
        self.assertTrue(mr.sexual_decision(mr.SEXUAL_THRESHOLD, 0.0, strict=False))
        self.assertTrue(mr.sexual_decision(0.0, mr.RECALL_SEXUAL_THRESHOLD, strict=False))

    def test_the_measured_sfw_false_positive_does_not_cross_the_recall_bar(self):
        # AdamCodd scored the clothed studio portraits up to 0.652; the recall
        # threshold is above that so a clothed adult is not called sexual by it.
        self.assertFalse(mr.sexual_decision(0.01, 0.652, strict=False))
        self.assertGreater(mr.RECALL_SEXUAL_THRESHOLD, 0.652)

    def test_strict_mode_lowers_both_bars(self):
        self.assertTrue(mr.sexual_decision(0.0, mr.STRICT_RECALL_SEXUAL_THRESHOLD, strict=True))
        self.assertLess(mr.recall_sexual_threshold(True, False), mr.recall_sexual_threshold(False, False))


class AggregateTests(unittest.TestCase):
    @staticmethod
    def img(sexual, refuse, minor_face=False, nsfw=0.5, max_p=0.0):
        return {"sexual": sexual, "refuse": refuse, "minor_face": minor_face, "nsfw_score": nsfw,
                "max_p_under_20": max_p, "faces_found": 1, "assessed_faces": 1}

    def test_minor_is_the_co_occurrence_verdict(self):
        # A child portrait (not sexual) next to an adult sexual image must not look
        # like "minor and sexual" to the host gate, which refuses on that conjunction.
        verdict = mr.aggregate([self.img(False, False, minor_face=True, max_p=0.9), self.img(True, False)])
        self.assertFalse(verdict["minor"])
        self.assertTrue(verdict["sexual"])
        self.assertTrue(verdict["detail"].startswith("passed"))

    def test_one_refused_image_refuses_the_request(self):
        verdict = mr.aggregate([self.img(True, False), self.img(True, True, minor_face=True, nsfw=0.97, max_p=0.8)])
        self.assertTrue(verdict["minor"] and verdict["sexual"])
        self.assertIn("image 1", verdict["detail"])
        self.assertIn("minor-appearing", verdict["detail"])

    def test_unassessable_refusal_says_the_age_could_not_be_established(self):
        refused = self.img(True, True, nsfw=0.9)
        refused["faces_found"], refused["assessed_faces"] = 0, 0
        self.assertIn("could not be established", mr.aggregate([refused])["detail"])


class ReviewImageStubTests(unittest.TestCase):
    def run_review(self, nsfw, faces, probs, **kw):
        image = Image.new("RGB", (400, 400), (90, 80, 70))
        detected = [{"box": f, "score": 0.9, "landmarks": None} for f in faces]
        with mock.patch.object(mr, "_nsfw_scores", return_value=(nsfw, 0.0)), \
                mock.patch("image_tools_sidecar.face_restore.detect_faces", return_value=detected), \
                mock.patch.object(mr, "_age_probs", return_value=probs), \
                mock.patch.object(mr._Models, "require"), \
                mock.patch.object(mr._Models, "detector", return_value=object()), \
                mock.patch.object(mr._Models, "age", return_value=(object(), LABELS)):
            return mr.review_image(np, Image, mr._Models("/nowhere"), image, **kw)

    def test_non_sexual_image_never_loads_the_face_or_age_models(self):
        with mock.patch.object(mr, "_nsfw_scores", return_value=(0.01, 0.0)), \
                mock.patch.object(mr._Models, "detector", side_effect=AssertionError("detector loaded")), \
                mock.patch.object(mr._Models, "age", side_effect=AssertionError("age model loaded")):
            out = mr.review_image(np, Image, mr._Models("/nowhere"), Image.new("RGB", (64, 64)), strict=False)
        self.assertFalse(out["sexual"] or out["refuse"])
        self.assertEqual(out["faces_found"], 0)

    def test_sexual_image_with_a_young_face_is_refused_with_numbers(self):
        probs = [[0.1, 0.7, 0.1, 0.05, 0.03, 0.01, 0.01, 0.0, 0.0]]
        out = self.run_review(0.97, [(100, 100, 220, 220)], probs, strict=False)
        self.assertTrue(out["refuse"] and out["minor_face"])
        self.assertAlmostEqual(out["max_p_under_20"], 0.9, places=3)
        self.assertEqual(out["faces"][0]["age_probs"]["3-9"], 0.7)

    def test_strict_mode_reaches_images_the_normal_bar_would_wave_through(self):
        probs = [[0.0, 0.0, 0.9, 0.1, 0, 0, 0, 0, 0]]
        normal = self.run_review(0.2, [(100, 100, 220, 220)], probs, strict=False)
        strict = self.run_review(0.2, [(100, 100, 220, 220)], probs, strict=True)
        self.assertFalse(normal["refuse"])
        self.assertTrue(strict["refuse"])

    def test_assume_sexual_probe_judges_the_age_finding_alone(self):
        probs = [[0.9, 0.05, 0.05, 0, 0, 0, 0, 0, 0]]
        out = self.run_review(0.0, [(100, 100, 220, 220)], probs, strict=False, assume_sexual=True)
        self.assertTrue(out["refuse"])


class RunContractTests(unittest.TestCase):
    def test_undecodable_image_is_an_error_not_a_pass(self):
        request = {"images": [{"media_type": "image/png", "base64": base64.b64encode(b"not an image").decode()}]}
        with self.assertRaises(SystemExit) as cm:
            mr.run(request, "/nowhere")
        self.assertEqual(cm.exception.code, 6)

    def test_empty_request_is_an_error(self):
        with self.assertRaises(SystemExit):
            mr.run({"images": []}, "/nowhere")

    def test_verdict_carries_the_documented_contract_fields(self):
        request = {"images": [{"media_type": "image/png", "base64": png_b64()}], "strict": True}
        with mock.patch.object(mr, "_nsfw_scores", return_value=(0.02, 0.0)):
            verdict = mr.run(request, "/nowhere")
        self.assertEqual({"minor", "sexual", "detail", "images", "thresholds"}, set(verdict))
        self.assertFalse(verdict["minor"] or verdict["sexual"])
        self.assertEqual(verdict["thresholds"]["sexual"], mr.STRICT_SEXUAL_THRESHOLD)
        json.dumps(verdict)  # the host parses stdout as JSON

    def test_missing_classifier_exits_with_the_missing_weights_code_naming_it(self):
        import os
        import tempfile
        with tempfile.TemporaryDirectory() as root, mock.patch("sys.stdin", io.StringIO('{"images": []}')):
            os.makedirs(os.path.join(root, "models"))
            with self.assertRaises(SystemExit) as cm, mock.patch("sys.stderr", new_callable=io.StringIO) as err:
                mr.main(["--models-root", root])
            self.assertEqual(cm.exception.code, mr.EXIT_MISSING_WEIGHTS)
            self.assertIn("missing-weights=" + mr.MODEL_NSFW, err.getvalue())

    def test_classifier_alone_is_enough_to_review_a_clear_image(self):
        # The face and age weights are not installed: a clear image must not need them.
        import os
        import tempfile
        request = {"images": [{"media_type": "image/png", "base64": png_b64()}]}
        with tempfile.TemporaryDirectory() as root:
            for model in mr.CLASSIFIER_MODELS:
                os.makedirs(os.path.join(root, "models", model))
            with mock.patch.object(mr, "_nsfw_scores", return_value=(0.01, 0.0)):
                verdict = mr.run(request, root)
        self.assertFalse(verdict["minor"] or verdict["sexual"])

    def test_a_sexual_image_without_the_face_and_age_weights_exits_naming_them(self):
        import os
        import tempfile
        request = {"images": [{"media_type": "image/png", "base64": png_b64()}]}
        with tempfile.TemporaryDirectory() as root:
            for model in mr.CLASSIFIER_MODELS:
                os.makedirs(os.path.join(root, "models", model))
            with mock.patch.object(mr, "_nsfw_scores", return_value=(0.95, 0.0)), \
                    self.assertRaises(SystemExit) as cm, \
                    mock.patch("sys.stderr", new_callable=io.StringIO) as err:
                mr.run(request, root)
        self.assertEqual(cm.exception.code, mr.EXIT_MISSING_WEIGHTS)
        for model in mr.ESCALATION_MODELS:
            self.assertIn(model, err.getvalue())


if __name__ == "__main__":
    unittest.main()
