"""Denoise tiling contract; no model download or GPU. Run with unittest."""
import unittest
from types import SimpleNamespace

import numpy as np

from . import denoise


class BoxSession:
    """A shift-invariant stand-in network: a 3x3 box blur with zero-fill borders.
    Its receptive field (1px) is far below the tile margin, so tiled output must
    equal one whole-image pass."""

    def __init__(self):
        self.shapes = []

    def get_inputs(self):
        return [SimpleNamespace(name="image")]

    def run(self, _names, feed):
        x = feed["image"]
        self.shapes.append(x.shape)
        padded = np.pad(x, ((0, 0), (0, 0), (1, 1), (1, 1)), mode="edge")
        acc = sum(padded[:, :, dy : dy + x.shape[2], dx : dx + x.shape[3]] for dy in range(3) for dx in range(3))
        return [acc / 9.0]


class DenoiseTiling(unittest.TestCase):
    def test_tiled_result_matches_a_single_pass_and_keeps_the_input_size(self):
        rng = np.random.default_rng(3)
        rgb = rng.random((150, 203, 3)).astype(np.float32)
        tiled_session = BoxSession()
        tiled = denoise.denoise_array(np, tiled_session, rgb, tile=64, margin=8)
        whole = denoise.denoise_array(np, BoxSession(), rgb, tile=512, margin=8)
        self.assertEqual(tiled.shape, rgb.shape)
        self.assertGreater(len(tiled_session.shapes), 1, "a small tile must force several passes")
        np.testing.assert_allclose(tiled, whole, atol=1e-5)
        self.assertLess(float(tiled.std()), float(rgb.std()), "the stand-in denoiser must smooth")

    def test_every_pass_is_a_multiple_of_eight(self):
        session = BoxSession()
        denoise.denoise_array(np, session, np.zeros((97, 131, 3), dtype=np.float32), tile=64, margin=8)
        for shape in session.shapes:
            self.assertEqual(shape[1], 3)
            self.assertEqual(shape[2] % 8, 0)
            self.assertEqual(shape[3] % 8, 0)


if __name__ == "__main__":
    unittest.main()
