"""Unit tests for max-over-clips scoring + self-consistency.

These exercise ``server._best_match`` and ``server._self_consistency``, which
use torch, so the whole module is skipped when torch is unavailable. It runs
inside the resource image:

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
class BestMatchTest(unittest.TestCase):
    def test_empty_profile_returns_negative(self):
        score, label, clip_id = server._best_match(_unit(1.0, 0.0), [])
        self.assertEqual(score, -1.0)
        self.assertEqual(label, "")
        self.assertEqual(clip_id, "")

    def test_max_over_clips_picks_best_label_and_id(self):
        clips = [
            {"embedding": _unit(1.0, 0.0), "label": "normal", "clip_id": "n1"},
            {"embedding": _unit(0.0, 1.0), "label": "whisper", "clip_id": "w1"},
        ]
        # A test vector aligned with whisper: per-clip cosine = 1.0
        score, label, clip_id = server._best_match(_unit(0.0, 1.0), clips)
        self.assertAlmostEqual(score, 1.0, places=5)
        self.assertEqual(label, "whisper")
        self.assertEqual(clip_id, "w1")

    def test_max_over_clips_picks_better_of_two(self):
        clips = [
            {"embedding": _unit(1.0, 0.0), "label": "normal", "clip_id": "n1"},
            {"embedding": _unit(0.5, 0.5), "label": "mixed", "clip_id": "m1"},
        ]
        # Vector closer to "normal" than "mixed"
        score, label, _id = server._best_match(_unit(0.9, 0.1), clips)
        self.assertEqual(label, "normal")
        self.assertGreater(score, 0.95)


@unittest.skipUnless(_HAVE, "torch/server unavailable on host")
class SelfConsistencyTest(unittest.TestCase):
    def test_no_existing_clips_returns_negative(self):
        score, label, clip_id = server._self_consistency(_unit(1.0, 0.0), [])
        self.assertEqual(score, -1.0)
        self.assertEqual(label, "")
        self.assertEqual(clip_id, "")

    def test_consistent_clip_scores_high(self):
        existing = [{"embedding": _unit(1.0, 0.0), "label": "normal", "clip_id": "n1"}]
        # New clip extremely similar to existing
        score, label, _id = server._self_consistency(_unit(0.99, 0.05), existing)
        self.assertGreater(score, 0.95)
        self.assertEqual(label, "normal")

    def test_divergent_clip_scores_low(self):
        existing = [{"embedding": _unit(1.0, 0.0), "label": "normal", "clip_id": "n1"}]
        # Orthogonal new clip
        score, _label, _id = server._self_consistency(_unit(0.0, 1.0), existing)
        self.assertLess(score, 0.1)


@unittest.skipUnless(_HAVE, "torch/server unavailable on host")
class TotalVoicedTest(unittest.TestCase):
    def test_sum(self):
        clips = [{"voiced_seconds": 3.0}, {"voiced_seconds": 1.5}, {}]
        self.assertAlmostEqual(server._total_voiced_seconds(clips), 4.5)


if __name__ == "__main__":
    unittest.main()
