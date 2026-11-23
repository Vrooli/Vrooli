"""Embedding preprocessing and output selection: a vision tower's preprocessing comes
from its preprocessor_config.json, and the vector is the pooled image embedding, not the
per-patch hidden states. No model download or GPU."""
import json
import os
import tempfile
import unittest
from types import SimpleNamespace

import numpy as np
from PIL import Image

from . import embedding


class Preprocessing(unittest.TestCase):
    def write_config(self, directory, config):
        with open(os.path.join(directory, "preprocessor_config.json"), "w", encoding="utf-8") as handle:
            json.dump(config, handle)
        return os.path.join(directory, "vision_model.onnx")

    def test_a_config_sets_mean_std_size_and_resampling(self):
        with tempfile.TemporaryDirectory() as directory:
            weight = self.write_config(
                directory,
                {"image_mean": [0.5, 0.5, 0.5], "image_std": [0.5, 0.5, 0.5], "size": {"height": 224, "width": 224}, "resample": 2},
            )
            mean, std, size_hw, resample = embedding._load_preprocessing(weight, (8, 8))
        self.assertEqual(mean, [0.5, 0.5, 0.5])
        self.assertEqual(std, [0.5, 0.5, 0.5])
        self.assertEqual(size_hw, (224, 224))
        self.assertEqual(resample, "BILINEAR")

    def test_without_a_config_imagenet_and_the_model_size_apply(self):
        with tempfile.TemporaryDirectory() as directory:
            mean, std, size_hw, resample = embedding._load_preprocessing(os.path.join(directory, "model.onnx"), (320, 320))
        self.assertEqual(mean, embedding.IMAGENET["image_mean"])
        self.assertEqual(std, embedding.IMAGENET["image_std"])
        self.assertEqual(size_hw, (320, 320))
        self.assertEqual(resample, "BICUBIC")

    def test_half_normalisation_maps_white_to_one_and_black_to_minus_one(self):
        white = Image.new("RGB", (4, 4), (255, 255, 255))
        black = Image.new("RGB", (4, 4), (0, 0, 0))
        half = ([0.5] * 3, [0.5] * 3)
        self.assertAlmostEqual(float(embedding._preprocess(np, white, (2, 2), *half).max()), 1.0, places=5)
        self.assertAlmostEqual(float(embedding._preprocess(np, black, (2, 2), *half).min()), -1.0, places=5)


class OutputSelection(unittest.TestCase):
    def session(self, names):
        return SimpleNamespace(get_outputs=lambda: [SimpleNamespace(name=n) for n in names])

    def test_the_pooled_embedding_is_chosen_over_patch_states(self):
        outputs = [np.zeros((1, 196, 768)), np.ones((1, 768))]
        picked = embedding._embedding_output(self.session(["last_hidden_state", "pooler_output"]), outputs)
        self.assertEqual(picked.shape, (1, 768))

    def test_a_token_grid_is_reduced_to_its_class_token(self):
        grid = np.arange(2 * 197 * 4, dtype=np.float32).reshape(1, 197 * 2, 4)[:, :197, :]
        picked = embedding._embedding_output(self.session(["last_hidden_state"]), [grid])
        self.assertEqual(picked.shape, (1, 4))
        np.testing.assert_array_equal(picked[0], grid[0, 0])

    def test_a_single_unnamed_output_is_used_as_is(self):
        outputs = [np.ones((1, 768))]
        self.assertIs(embedding._embedding_output(self.session(["output"]), outputs), outputs[0])


if __name__ == "__main__":
    unittest.main()
