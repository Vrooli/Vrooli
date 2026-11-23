"""Deblur runs a dynamic-shape model at the image's own size; no weights, no GPU."""
import unittest
from types import SimpleNamespace

import numpy as np

from . import deblur


class StrictSession:
    """A shift-invariant stand-in with NAFNet's constraints: every side must be a
    multiple of 32 and at least 384, or the real model's Pad node fails."""

    def __init__(self, dynamic=True):
        self.shapes = []
        self.dynamic = dynamic

    def get_inputs(self):
        shape = [1, 3, "height", "width"] if self.dynamic else [1, 3, 256, 256]
        return [SimpleNamespace(name="lq", shape=shape)]

    def run(self, _names, feed):
        x = feed["lq"]
        self.shapes.append(x.shape)
        assert x.shape[2] % 32 == 0 and x.shape[3] % 32 == 0, x.shape
        assert min(x.shape[2:]) >= 384, x.shape
        padded = np.pad(x, ((0, 0), (0, 0), (1, 1), (1, 1)), mode="edge")
        acc = sum(padded[:, :, dy : dy + x.shape[2], dx : dx + x.shape[3]] for dy in range(3) for dx in range(3))
        return [acc / 9.0]


class DeblurAtNativeSize(unittest.TestCase):
    def test_a_small_image_is_padded_up_to_a_size_the_network_accepts(self):
        session = StrictSession()
        rgb = np.random.default_rng(1).random((120, 90, 3)).astype(np.float32)
        out = deblur.deblur_array(np, session, rgb)
        self.assertEqual(out.shape, rgb.shape)
        self.assertEqual(len(session.shapes), 1)

    def test_tiled_result_matches_one_pass_and_keeps_the_input_size(self):
        rgb = np.random.default_rng(2).random((500, 610, 3)).astype(np.float32)
        tiled_session = StrictSession()
        tiled = deblur.deblur_array(np, tiled_session, rgb, tile=256, margin=8)
        whole = deblur.deblur_array(np, StrictSession(), rgb, tile=2048, margin=8)
        self.assertEqual(tiled.shape, rgb.shape)
        self.assertGreater(len(tiled_session.shapes), 1, "a small tile must force several passes")
        np.testing.assert_allclose(tiled, whole, atol=1e-5)

    def test_dynamic_detection(self):
        self.assertTrue(deblur._is_dynamic(StrictSession(dynamic=True)))
        self.assertFalse(deblur._is_dynamic(StrictSession(dynamic=False)))


if __name__ == "__main__":
    unittest.main()
