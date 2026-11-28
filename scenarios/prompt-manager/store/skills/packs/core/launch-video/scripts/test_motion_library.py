import unittest

from motion_library import compile_plan


def request(**overrides):
    value = {"profile": "launch", "audio": {"bpm": 120, "first_beat_seconds": 0.1},
             "scenes": [{"id": "open", "start_seconds": 0, "end_seconds": 4,
                         "words": 3, "camera": "push"},
                        {"id": "proof", "start_seconds": 4, "end_seconds": 10,
                         "words": 0, "camera": "drift-left"}]}
    value.update(overrides)
    return value


class MotionLibraryTests(unittest.TestCase):
    def test_resolves_beat_phase_and_typed_hold(self):
        plan = compile_plan(request())
        opening = plan["scenes"][0]
        self.assertAlmostEqual(plan["audio"]["beat_seconds"], 0.5)
        self.assertAlmostEqual(opening["nearest_beat_seconds"], 0.1)
        self.assertAlmostEqual(opening["beat_error_seconds"], -0.1)
        self.assertGreaterEqual(opening["settled_hold_seconds"], 1.2)
        self.assertEqual(opening["camera_scale_delta"], 0.045)

    def test_profile_changes_pacing_and_camera_strength(self):
        scenes = [{"id": "one", "start_seconds": 0, "end_seconds": 8, "camera": "push"}]
        launch = compile_plan({"profile": "launch", "audio": {"bpm": 120}, "scenes": scenes})
        explainer = compile_plan({"profile": "explainer", "audio": {"bpm": 120}, "scenes": scenes})
        self.assertEqual(launch["scenes"][0]["pacing"], "hold")
        self.assertEqual(explainer["scenes"][0]["pacing"], "target")
        self.assertGreater(launch["scenes"][0]["camera_scale_delta"], explainer["scenes"][0]["camera_scale_delta"])

    def test_rejects_gap_and_overlap(self):
        scenes = [{"start_seconds": 0, "end_seconds": 2}, {"start_seconds": 2.01, "end_seconds": 4}]
        with self.assertRaisesRegex(ValueError, "contiguous"):
            compile_plan({"profile": "launch", "audio": {"bpm": 120}, "scenes": scenes})

    def test_rejects_transition_that_consumes_scene(self):
        scenes = [{"start_seconds": 0, "end_seconds": 1, "transition_beats": 1.1}]
        with self.assertRaisesRegex(ValueError, "consume"):
            compile_plan({"profile": "launch", "audio": {"bpm": 120}, "scenes": scenes})

    def test_reading_hold_respects_profile_minimum(self):
        scenes = [{"start_seconds": 0, "end_seconds": 4, "words": 2, "settled_hold_seconds": 0.5}]
        with self.assertRaisesRegex(ValueError, "below product_story minimum"):
            compile_plan({"profile": "product_story", "audio": {"bpm": 100}, "scenes": scenes})

    def test_rejects_invalid_tempo(self):
        with self.assertRaisesRegex(ValueError, "audio.bpm"):
            compile_plan({"profile": "launch", "audio": {"bpm": 0}, "scenes": [{"start_seconds": 0, "end_seconds": 2}]})


if __name__ == "__main__":
    unittest.main()
