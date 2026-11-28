import unittest
from types import SimpleNamespace
from unittest import mock
import tempfile
import os

import numpy as np
from PIL import Image

from image_tools_sidecar import _common
from image_tools_sidecar import nsfw
from image_tools_sidecar.nsfw import _preprocess


class NSFWPreprocessTests(unittest.TestCase):
    def test_dynamic_onnx_axes_use_the_model_specific_fallback(self):
        session = SimpleNamespace(get_inputs=lambda: [SimpleNamespace(shape=[1, 3, "height", "width"])])
        self.assertEqual(_common.model_input_hw(session, default_hw=(384, 384)), (384, 384))

    def test_preprocess_matches_pinned_vit_processor_contract(self):
        image = Image.new("RGB", (1, 1), (0, 128, 255))
        tensor = _preprocess(np, image, (1, 1), resample=Image.Resampling.BILINEAR)

        expected = np.array([-1.0, 1.0 / 255.0, 1.0], dtype=np.float32)
        self.assertEqual(tensor.shape, (1, 3, 1, 1))
        np.testing.assert_allclose(tensor[0, :, 0, 0], expected, atol=1e-6)


class NSFWProfileTests(unittest.TestCase):
    def classify(self, scores):
        profile = {
            "name": "test", "reviewer_version": "test",
            "classifiers": [
                {"id": "falconsai-nsfw", "threshold": 0.3},
                {"id": "adamcodd-vit-nsfw", "threshold": 0.8},
            ],
            "face": {"id": "yunet", "score_threshold": 0.5, "min_px": 40, "crop_margin": 1.35, "max_faces": 16},
            "age": {"id": "fairface-vit-age", "threshold": 0.2},
            "on_minor_face": "refuse", "on_unassessable": "refuse",
        }
        payloads = {
            "falconsai-nsfw": {"score": scores[0], "categories": [{"label": "nsfw", "score": scores[0]}]},
            "adamcodd-vit-nsfw": {"score": scores[1], "categories": [{"label": "nsfw", "score": scores[1]}]},
        }
        with tempfile.TemporaryDirectory() as tmp, \
                mock.patch("image_tools_sidecar.nsfw._common.require_deps", return_value=(np, None, Image)), \
                mock.patch("image_tools_sidecar.minor_review._Models.require"), \
                mock.patch("image_tools_sidecar.minor_review._Models.classifier", side_effect=lambda cfg: cfg), \
                mock.patch("image_tools_sidecar.minor_review._classifier_payload", side_effect=lambda _np, _Image, model, _image: payloads[model["id"]]):
            path = os.path.join(tmp, "in.png")
            Image.new("RGB", (8, 8), (1, 2, 3)).save(path)
            return nsfw.classify(path, "/models", __import__("json").dumps(profile), None)

    def test_the_winning_classifier_sets_the_reported_score_and_threshold(self):
        # The precise classifier is below its bar; the high-recall one crossed.
        # The reported score/threshold must be the one that decided, not a score
        # below the threshold it is displayed beside.
        out = self.classify((0.2, 0.9))
        self.assertTrue(out["sexual"])
        self.assertEqual(out["score"], 0.9)
        self.assertEqual(out["threshold"], 0.8)
        self.assertEqual(out["primary_score"], 0.2)
        self.assertEqual(out["categories"][0]["score"], 0.9)

    def test_a_clear_image_reports_the_primary_classifier(self):
        out = self.classify((0.1, 0.2))
        self.assertFalse(out["sexual"])
        self.assertEqual(out["score"], 0.1)
        self.assertEqual(out["threshold"], 0.3)
        self.assertEqual(out["reason_code"], "clean")
