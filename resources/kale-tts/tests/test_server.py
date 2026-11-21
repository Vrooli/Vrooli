import os
import json
import sys
import tempfile
import unittest
from unittest.mock import patch

import numpy as np
from fastapi.testclient import TestClient

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "server"))
import server as sut  # noqa: E402


class FakeTTS:
    def list_voices(self):
        return [{"id": "s057b", "name": "Kenzie"}]

    def stream(self, _text, voice=None):
        if voice not in ("kenzie", "s057b"):
            raise ValueError("unknown voice")
        return [np.array([0.0, 0.5, -0.5], dtype=np.float32)]


class ServerContractTests(unittest.TestCase):
    def setUp(self):
        self.client = TestClient(sut.app)
        self.engine_patch = patch.object(sut, "engine", return_value=FakeTTS())
        self.engine_patch.start()

    def tearDown(self):
        self.engine_patch.stop()
        self.client.close()

    def test_voice_list_shape(self):
        with tempfile.TemporaryDirectory() as model:
            with open(os.path.join(model, "voices.json"), "w", encoding="utf-8") as target:
                json.dump({"voices": [{"id": "s057b", "name": "Kenzie", "accent": "en-us"}]}, target)
            with patch.object(sut, "MODEL_DIR", model):
                response = self.client.get("/v1/audio/voices")
        self.assertEqual(response.status_code, 200)
        self.assertEqual(response.json()["model"], "kale-tts-1-nano")
        self.assertEqual(response.json()["voices"][0]["id"], "s057b")

    def test_wav_and_pcm_responses(self):
        payload = {"model": "kale-tts-1-nano", "input": "hi", "voice": "kenzie"}
        wav = self.client.post("/v1/audio/speech", json=payload)
        pcm = self.client.post("/v1/audio/speech", json={**payload, "response_format": "pcm"})
        self.assertEqual(wav.status_code, 200)
        self.assertEqual(wav.headers["content-type"], "audio/wav")
        self.assertTrue(wav.content.startswith(b"RIFF"))
        self.assertEqual(pcm.status_code, 200)
        self.assertEqual(pcm.headers["content-type"], "audio/pcm")
        self.assertEqual(pcm.headers["x-sample-rate"], "24000")
        self.assertEqual(len(pcm.content), 6)

    def test_invalid_request_is_rejected(self):
        for payload in (
            {"model": "wrong", "input": "hi"},
            {"model": "kale-tts-1-nano", "input": " "},
            {"model": "kale-tts-1-nano", "input": "hi", "response_format": "mp3"},
            {"model": "kale-tts-1-nano", "input": "x" * 5001},
        ):
            with self.subTest(payload=payload):
                self.assertGreaterEqual(self.client.post("/v1/audio/speech", json=payload).status_code, 400)

    def test_health_does_not_load_model(self):
        with tempfile.TemporaryDirectory() as model, tempfile.TemporaryDirectory() as frontend:
            for name in ("kale-tts-1-nano.onnx", "voices.json", "tokenizer.json"):
                with open(os.path.join(model, name), "wb"):
                    pass
            os.makedirs(os.path.join(frontend, "words"))
            for name in ("lexicon.tsv", "override.public.tsv", "terms.tsv", "g2p.bin"):
                with open(os.path.join(frontend, "words", name), "wb"):
                    pass
            bundle = os.path.join(frontend, "kale_frontend.js")
            with open(bundle, "wb"):
                pass
            with patch.object(sut, "MODEL_DIR", model), patch.object(sut, "MODEL_FILE", "kale-tts-1-nano.onnx"), \
                    patch.object(sut, "FRONTEND_DIR", frontend), patch.object(sut, "FRONTEND_BUNDLE", bundle), \
                    patch.object(sut, "_engine", None):
                response = self.client.get("/healthz")
            self.assertEqual(response.status_code, 200)
            self.assertEqual(response.json(), {"status": "ok", "model": "kale-tts-1-nano", "loaded": False, "ready": True})

    def test_health_reports_missing_artifacts_as_unavailable(self):
        with tempfile.TemporaryDirectory() as model, tempfile.TemporaryDirectory() as frontend, \
                patch.object(sut, "MODEL_DIR", model), patch.object(sut, "FRONTEND_DIR", frontend):
            response = self.client.get("/healthz")
        self.assertEqual(response.status_code, 503)
        self.assertEqual(response.json()["status"], "model_missing")
        self.assertFalse(response.json()["ready"])


if __name__ == "__main__":
    unittest.main()
