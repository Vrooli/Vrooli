"""Unit tests for profile centroid + score aggregation.

These exercise ``server._centroid`` and ``server._score_profile``, which use
torch, so the whole module is skipped when torch is unavailable. It runs inside
the resource image:

    python -m unittest test_profiles
"""

from __future__ import annotations

import unittest

try:  # importing server pulls torch + constructs the VAD; container-only.
    import server

    _HAVE = True
except Exception:  # noqa: BLE001
    _HAVE = False


def _unit(*components: float):
    """A simple (unnormalized) embedding; cosine handles normalization."""
    return list(components)


@unittest.skipUnless(_HAVE, "torch/server unavailable on host")
class CentroidTest(unittest.TestCase):
    def test_empty(self):
        self.assertEqual(server._centroid([]), [])

    def test_single_clip_is_normalized_direction(self):
        clips = [{"embedding": _unit(3.0, 0.0, 0.0)}]
        centroid = server._centroid(clips)
        self.assertAlmostEqual(centroid[0], 1.0, places=5)
        self.assertAlmostEqual(centroid[1], 0.0, places=5)

    def test_mean_then_renormalize(self):
        clips = [
            {"embedding": _unit(1.0, 0.0)},
            {"embedding": _unit(0.0, 1.0)},
        ]
        centroid = server._centroid(clips)
        # mean = (0.5, 0.5) -> normalized = (0.707, 0.707)
        self.assertAlmostEqual(centroid[0], centroid[1], places=5)
        self.assertAlmostEqual(centroid[0], 0.70710677, places=5)


@unittest.skipUnless(_HAVE, "torch/server unavailable on host")
class ScoreProfileTest(unittest.TestCase):
    def _record(self):
        clips = [
            {"embedding": _unit(1.0, 0.0), "label": "normal"},
            {"embedding": _unit(0.0, 1.0), "label": "whisper"},
        ]
        return {"centroid": server._centroid(clips), "clips": clips}

    def test_hybrid_prefers_best_clip_over_centroid(self):
        orig = server.SCORE_AGG
        server.SCORE_AGG = "hybrid"
        try:
            rec = self._record()
            # A test vector aligned with the "whisper" clip: per-clip cosine = 1.0
            # beats the centroid cosine (~0.707), so hybrid returns the clip hit.
            score, label = server._score_profile(_unit(0.0, 1.0), rec)
            self.assertAlmostEqual(score, 1.0, places=5)
            self.assertEqual(label, "whisper")
        finally:
            server.SCORE_AGG = orig

    def test_centroid_only_mode(self):
        orig = server.SCORE_AGG
        server.SCORE_AGG = "centroid"
        try:
            rec = self._record()
            score, label = server._score_profile(_unit(0.0, 1.0), rec)
            self.assertAlmostEqual(score, 0.70710677, places=5)
            self.assertEqual(label, "")
        finally:
            server.SCORE_AGG = orig

    def test_max_mode_returns_label(self):
        orig = server.SCORE_AGG
        server.SCORE_AGG = "max"
        try:
            rec = self._record()
            score, label = server._score_profile(_unit(1.0, 0.0), rec)
            self.assertAlmostEqual(score, 1.0, places=5)
            self.assertEqual(label, "normal")
        finally:
            server.SCORE_AGG = orig


@unittest.skipUnless(_HAVE, "torch/server unavailable on host")
class TotalVoicedTest(unittest.TestCase):
    def test_sum(self):
        clips = [{"voiced_seconds": 3.0}, {"voiced_seconds": 1.5}, {}]
        self.assertAlmostEqual(server._total_voiced_seconds(clips), 4.5)


if __name__ == "__main__":
    unittest.main()
