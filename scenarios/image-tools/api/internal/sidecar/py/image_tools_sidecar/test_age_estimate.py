"""Contracts of the age_estimate op (stub session: what it does with model outputs)."""

import json
import os
import tempfile
import unittest

import numpy as np
from PIL import Image

from image_tools_sidecar import age_estimate as ae

LABELS = ["0-2", "3-9", "10-19", "20-29", "30-39", "40-49", "50-59", "60-69", "more than 70"]


class StubSession:
    """Returns fixed logits per batch row; records the batch it was given."""

    def __init__(self, logits):
        self.logits = np.asarray(logits, dtype=np.float32)
        self.batch = None

    class _In:
        name = "pixel_values"

    def get_inputs(self):
        return [self._In()]

    def run(self, _outputs, feed):
        self.batch = feed["pixel_values"]
        return [self.logits[: len(self.batch)]]


def write_config(directory, labels=LABELS):
    with open(os.path.join(directory, "config.json"), "w") as fh:
        json.dump({"id2label": {str(i): name for i, name in enumerate(labels)}}, fh)


class AgeEstimateTests(unittest.TestCase):
    def test_under_age_mass_counts_only_the_three_minor_bins(self):
        probs = [0.1, 0.2, 0.3, 0.2, 0.1, 0.05, 0.03, 0.01, 0.01]
        self.assertAlmostEqual(ae.under_age_mass(LABELS, probs), 0.6)

    def test_labels_come_from_config_and_must_include_the_minor_bins(self):
        with tempfile.TemporaryDirectory() as d:
            write_config(d)
            self.assertEqual(ae.read_labels(d), LABELS)
        with tempfile.TemporaryDirectory() as d:
            write_config(d, ["a", "b", "c"])
            with self.assertRaises(ValueError):
                ae.read_labels(d)

    def test_crop_tensor_is_chw_224_and_normalised_to_unit_range(self):
        t = ae.crop_tensor(np, Image, Image.new("RGB", (100, 60), (255, 0, 0)))
        self.assertEqual(t.shape, (3, 224, 224))
        self.assertAlmostEqual(float(t[0].max()), 1.0, places=5)
        self.assertAlmostEqual(float(t[1].min()), -1.0, places=5)

    def test_probabilities_average_each_crop_with_its_mirror_and_sum_to_one(self):
        # Row i is the crop, row i+n is its mirror: logits differ so the average is visible.
        logits = [[4.0, 0, 0, 0, 0, 0, 0, 0, 0], [0, 4.0, 0, 0, 0, 0, 0, 0, 0]]
        session = StubSession(logits)
        tensor = ae.crop_tensor(np, Image, Image.new("RGB", (32, 32), (10, 20, 30)))
        out = ae.mirrored_probs(np, session, [tensor])
        self.assertEqual(session.batch.shape[0], 2)  # the crop and its mirror, one batch
        self.assertAlmostEqual(sum(out[0]), 1.0, places=5)
        self.assertAlmostEqual(out[0][0], out[0][1], places=5)  # symmetric average of the two logits

    def test_no_crops_runs_nothing(self):
        session = StubSession([[0] * 9])
        self.assertEqual(ae.mirrored_probs(np, session, []), [])
        self.assertIsNone(session.batch)

    def test_payload_is_the_documented_json(self):
        probs = [0.05, 0.1, 0.15, 0.4, 0.2, 0.05, 0.03, 0.01, 0.01]
        out = ae.payload(LABELS, probs)
        self.assertEqual(set(out), {"labels", "probs", "p_under_20"})
        self.assertEqual(out["probs"]["20-29"], 0.4)
        self.assertAlmostEqual(out["p_under_20"], 0.3, places=5)
        json.dumps(out)


if __name__ == "__main__":
    unittest.main()
