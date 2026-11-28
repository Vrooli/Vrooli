"""Colorize contracts for the ab-output DDColor export; no model download or GPU."""
import unittest
from types import SimpleNamespace

import numpy as np
from PIL import Image

from . import colorize


class FakeAbSession:
    """An ONNX session stand-in with the DDColor ab contract: (1,3,H,W) in, (1,2,H,W) out."""

    def __init__(self, a, b, size=64):
        self.size, self.a, self.b, self.seen = size, a, b, None

    def get_inputs(self):
        return [SimpleNamespace(name="input", shape=[1, 3, self.size, self.size])]

    def get_outputs(self):
        return [SimpleNamespace(name="output", shape=[1, 2, self.size, self.size])]

    def run(self, _names, feed):
        self.seen = feed["input"]
        out = np.empty((1, 2, self.size, self.size), dtype=np.float32)
        out[0, 0], out[0, 1] = self.a, self.b
        return [out]


class ColorizeContracts(unittest.TestCase):
    def test_lab_round_trip_preserves_srgb(self):
        rng = np.random.default_rng(7)
        rgb = rng.random((16, 16, 3)).astype(np.float32)
        back = colorize._lab_to_srgb(np, colorize._srgb_to_lab(np, rgb))
        np.testing.assert_allclose(back, rgb, atol=2e-3)

    def test_ab_export_keeps_full_resolution_luminance_and_adds_chroma(self):
        gradient = np.tile(np.linspace(20, 235, 200, dtype=np.uint8), (120, 1))
        grey = Image.fromarray(np.stack([gradient] * 3, axis=-1), mode="RGB")
        session = FakeAbSession(a=40.0, b=30.0)  # warm red-orange chroma
        out = colorize._colorize_ab(np, Image, session, grey, 64, 64)
        self.assertEqual(out.size, grey.size)
        self.assertEqual(session.seen.shape, (1, 3, 64, 64))
        # The model input is the grey rendering of L: all three channels agree.
        np.testing.assert_allclose(session.seen[0, 0], session.seen[0, 1], atol=1e-3)
        arr = np.asarray(out, dtype=np.float32)
        self.assertTrue((arr[..., 0] > arr[..., 2]).mean() > 0.9, "red channel should dominate for a>0,b>0")
        # Luminance is the source's: L of the output tracks L of the input.
        l_in = colorize._srgb_to_lab(np, np.asarray(grey, dtype=np.float32) / 255.0)[..., 0]
        l_out = colorize._srgb_to_lab(np, arr / 255.0)[..., 0]
        # In-gamut clipping bounds the drift at saturated extremes; the mid-tones match.
        mid = (l_in > 30) & (l_in < 80)
        np.testing.assert_allclose(l_out[mid], l_in[mid], atol=4.0)

    def test_zero_chroma_returns_the_grey_source(self):
        gradient = np.tile(np.linspace(0, 255, 100, dtype=np.uint8), (50, 1))
        grey = Image.fromarray(np.stack([gradient] * 3, axis=-1), mode="RGB")
        out = colorize._colorize_ab(np, Image, FakeAbSession(a=0.0, b=0.0), grey, 32, 32)
        np.testing.assert_allclose(np.asarray(out, dtype=np.float32), np.asarray(grey, dtype=np.float32), atol=2.0)


if __name__ == "__main__":
    unittest.main()
