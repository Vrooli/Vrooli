import os
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "tools"))
import qualify  # noqa: E402


class QualificationHarnessTests(unittest.TestCase):
    def test_default_run_covers_each_voice_and_prompt_without_quality_claim(self):
        voices = [{"id": "kenzie"}, {"id": "luna"}]
        sample = {"latency_ms": 10.0, "duration_seconds": 1.0, "peak": 0.3,
                  "rms": 0.1, "clipped_sample_count": 0, "wav_sha256": "abc"}
        with patch.object(qualify, "request_json", side_effect=[
                {"ready": True, "model": "kale-tts-1-nano"},
                {"voices": voices}]), patch.object(qualify, "synthesize", return_value=sample) as synth:
            receipt = qualify.run("http://127.0.0.1:18896")
        self.assertEqual(receipt["request_count"], 4)
        self.assertEqual(receipt["voice_ids"], ["kenzie", "luna"])
        self.assertFalse(receipt["quality_claim"])
        self.assertEqual(synth.call_count, 4)

    def test_requested_voice_must_be_in_catalog(self):
        with patch.object(qualify, "request_json", side_effect=[
                {"ready": True, "model": "kale-tts-1-nano"}, {"voices": [{"id": "kenzie"}]}]):
            with self.assertRaisesRegex(ValueError, "voice catalogue"):
                qualify.run("http://127.0.0.1:18896", ["missing"])


if __name__ == "__main__":
    unittest.main()
