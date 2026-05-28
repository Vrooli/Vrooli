"""Unit tests for the voice-activity detector.

The pure energy/threshold/mask math is exercised without torch so it runs in any
Python (host or image). The end-to-end ``EnergyVAD.trim`` test is skipped when
torch is unavailable; it runs in the resource image (or any torch venv).

    python -m unittest test_vad
"""

from __future__ import annotations

import math
import unittest

import vad

try:  # torch is present in the resource image, not necessarily on a bare host.
    import torch

    _HAVE_TORCH = True
except Exception:  # noqa: BLE001
    _HAVE_TORCH = False


class PercentileTest(unittest.TestCase):
    def test_endpoints_and_interpolation(self):
        self.assertEqual(vad.percentile([], 0.5), 0.0)
        self.assertEqual(vad.percentile([7.0], 0.9), 7.0)
        self.assertEqual(vad.percentile([0.0, 1.0], 0.0), 0.0)
        self.assertEqual(vad.percentile([0.0, 1.0], 1.0), 1.0)
        self.assertAlmostEqual(vad.percentile([0.0, 1.0], 0.5), 0.5)
        self.assertAlmostEqual(vad.percentile([0.0, 10.0, 20.0, 30.0], 0.1), 3.0)


class ThresholdTest(unittest.TestCase):
    def test_constant_signal_keeps_everything(self):
        # A pure tone has near-constant RMS: peak ~= noise, so the threshold sits
        # at the floor and every frame is voiced (the 220Hz integration fixture).
        rms = [0.5] * 20
        threshold, peak, noise = vad.estimate_threshold(rms, 0.1, 0.08)
        self.assertAlmostEqual(peak, 0.5)
        self.assertAlmostEqual(noise, 0.5)
        self.assertTrue(all(v > threshold for v in rms) or threshold == peak)

    def test_speech_over_silence_separates(self):
        rms = [0.001] * 10 + [0.4] * 10 + [0.001] * 10
        threshold, peak, noise = vad.estimate_threshold(rms, 0.1, 0.08)
        self.assertLess(noise, 0.01)
        self.assertGreater(peak, 0.3)
        self.assertTrue(0.001 < threshold < 0.4)
        mask = [v > threshold for v in rms]
        self.assertEqual(mask, [False] * 10 + [True] * 10 + [False] * 10)


class SmoothMaskTest(unittest.TestCase):
    def test_empty(self):
        self.assertEqual(vad.smooth_mask([], 5, 2), [])

    def test_fills_short_gap(self):
        mask = [True, True, False, False, True, True]
        out = vad.smooth_mask(mask, hangover_frames=3, pad_frames=0)
        self.assertEqual(out, [True] * 6)

    def test_keeps_long_gap(self):
        mask = [True, False, False, False, False, True]
        out = vad.smooth_mask(mask, hangover_frames=2, pad_frames=0)
        self.assertEqual(out, [True, False, False, False, False, True])

    def test_pad_dilates(self):
        mask = [False, False, True, False, False]
        out = vad.smooth_mask(mask, hangover_frames=0, pad_frames=1)
        self.assertEqual(out, [False, True, True, True, False])


@unittest.skipUnless(_HAVE_TORCH, "torch unavailable on host")
class EnergyTrimTorchTest(unittest.TestCase):
    def setUp(self):
        self.sr = 16000
        self.det = vad.EnergyVAD()

    def _tone(self, seconds: float, freq: float = 220.0, amp: float = 0.5):
        n = int(self.sr * seconds)
        t = torch.arange(n, dtype=torch.float32) / self.sr
        return (amp * torch.sin(2 * math.pi * freq * t)).reshape(1, n)

    def test_silent_clip_yields_nothing(self):
        silence = torch.zeros(1, self.sr)
        voiced, secs = self.det.trim(silence, self.sr)
        self.assertEqual(voiced.numel(), 0)
        self.assertEqual(secs, 0.0)

    def test_trims_leading_and_trailing_silence(self):
        clip = torch.cat(
            [torch.zeros(1, self.sr), self._tone(1.0), torch.zeros(1, self.sr)], dim=1
        )
        voiced, secs = self.det.trim(clip, self.sr)
        # ~1s of tone survives (+/- hangover/pad), the 2s of silence is dropped.
        self.assertGreater(secs, 0.7)
        self.assertLess(secs, 1.6)
        self.assertEqual(voiced.dim(), 2)
        self.assertEqual(voiced.size(0), 1)

    def test_pure_tone_mostly_survives(self):
        clip = self._tone(2.0)
        voiced, secs = self.det.trim(clip, self.sr)
        self.assertGreater(secs, 1.5)

    def test_noop_passthrough(self):
        clip = self._tone(1.0)
        voiced, secs = vad.NoOpVAD().trim(clip, self.sr)
        self.assertEqual(voiced.size(-1), clip.size(-1))
        self.assertAlmostEqual(secs, 1.0, places=3)


class BuildVadTest(unittest.TestCase):
    def test_defaults_to_energy(self):
        self.assertEqual(vad.build_vad("").name, "energy")
        self.assertEqual(vad.build_vad("energy").name, "energy")

    def test_none(self):
        self.assertEqual(vad.build_vad("none").name, "none")

    def test_silero_reserved(self):
        with self.assertRaises(ValueError):
            vad.build_vad("silero")

    def test_unknown(self):
        with self.assertRaises(ValueError):
            vad.build_vad("bogus")


if __name__ == "__main__":
    unittest.main()
