"""Integration tests for the /v1/extract API endpoint.

These tests use FastAPI's TestClient with mocked models to validate
request/response contracts without requiring actual model weights.
"""

import io
import struct
import wave
from unittest.mock import MagicMock, patch

import numpy as np
import pytest
from fastapi.testclient import TestClient


def _wav_bytes(duration: float = 2.0, sr: int = 16000, freq: float = 440.0) -> bytes:
    """Generate a valid WAV file as bytes."""
    n_samples = int(sr * duration)
    t = np.linspace(0, duration, n_samples, endpoint=False)
    samples = (0.5 * np.sin(2 * np.pi * freq * t) * 32767).astype(np.int16)

    buf = io.BytesIO()
    with wave.open(buf, "wb") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(sr)
        wf.writeframes(samples.tobytes())
    return buf.getvalue()


@pytest.fixture
def client():
    """Create a TestClient with mocked models."""
    # Mock extraction and embeddings modules before importing app
    with patch("extraction.is_model_loaded", return_value=True), \
         patch("extraction._separator", MagicMock()), \
         patch("embeddings.is_model_loaded", return_value=True), \
         patch("embeddings.load_model"), \
         patch("extraction.load_model"):

        from app import app
        yield TestClient(app)


@pytest.fixture
def mock_profile_embedding():
    """A normalized 192-dim embedding."""
    emb = np.random.randn(192).astype(np.float32)
    emb /= np.linalg.norm(emb)
    return emb


class TestExtractEndpointReturnsWav:
    """Tests that /v1/extract returns valid WAV audio with correct headers."""

    def test_returns_wav_content_type(self, client, mock_profile_embedding):
        extracted = np.random.randn(32000).astype(np.float32) * 0.1

        with patch("profiles.load_embedding", return_value=mock_profile_embedding), \
             patch("audio.load_and_normalize") as mock_audio, \
             patch("extraction.extract_target_speaker", return_value=extracted), \
             patch("embeddings.extract_embedding", return_value=mock_profile_embedding), \
             patch("verification.verify", return_value={"score": 0.9, "matched": True, "threshold": 0.7, "duration_ms": 1.0}):

            import torch
            mock_audio.return_value = (torch.randn(1, 32000), 2.0)

            audio_data = _wav_bytes()
            resp = client.post(
                "/v1/extract",
                data={"profile_id": "test-profile", "verify": "true"},
                files={"audio": ("test.wav", audio_data, "audio/wav")},
            )

        assert resp.status_code == 200
        assert resp.headers["content-type"] == "audio/wav"
        assert resp.content[:4] == b"RIFF"
        assert resp.content[8:12] == b"WAVE"

    def test_returns_score_headers_when_verify_true(self, client, mock_profile_embedding):
        extracted = np.random.randn(32000).astype(np.float32) * 0.1

        with patch("profiles.load_embedding", return_value=mock_profile_embedding), \
             patch("audio.load_and_normalize") as mock_audio, \
             patch("extraction.extract_target_speaker", return_value=extracted), \
             patch("embeddings.extract_embedding", return_value=mock_profile_embedding), \
             patch("verification.verify", return_value={"score": 0.85, "matched": True, "threshold": 0.7, "duration_ms": 1.0}):

            import torch
            mock_audio.return_value = (torch.randn(1, 32000), 2.0)

            resp = client.post(
                "/v1/extract",
                data={"profile_id": "test-profile", "verify": "true"},
                files={"audio": ("test.wav", _wav_bytes(), "audio/wav")},
            )

        assert resp.status_code == 200
        assert "x-speaker-score" in resp.headers
        assert "x-speaker-matched" in resp.headers
        assert "x-duration-ms" in resp.headers
        assert "x-audio-seconds" in resp.headers
        assert resp.headers["x-speaker-matched"] == "true"

    def test_returns_matched_false_when_speaker_absent(self, client, mock_profile_embedding):
        """When extraction returns None, response has score 0 and matched false."""
        with patch("profiles.load_embedding", return_value=mock_profile_embedding), \
             patch("audio.load_and_normalize") as mock_audio, \
             patch("extraction.extract_target_speaker", return_value=None):

            import torch
            mock_audio.return_value = (torch.randn(1, 32000), 2.0)

            resp = client.post(
                "/v1/extract",
                data={"profile_id": "test-profile", "verify": "true"},
                files={"audio": ("test.wav", _wav_bytes(), "audio/wav")},
            )

        assert resp.status_code == 200
        assert resp.headers["x-speaker-matched"] == "false"
        assert resp.headers["x-speaker-score"] == "0.0"


class TestExtractEndpointErrors:
    """Tests for error responses from /v1/extract."""

    def test_unknown_profile_returns_404(self, client):
        with patch("extraction.is_model_loaded", return_value=True), \
             patch("profiles.load_embedding", return_value=None):

            resp = client.post(
                "/v1/extract",
                data={"profile_id": "nonexistent"},
                files={"audio": ("test.wav", _wav_bytes(), "audio/wav")},
            )

        assert resp.status_code == 404

    def test_short_audio_returns_400(self, client, mock_profile_embedding):
        short_wav = _wav_bytes(duration=0.3)  # Less than 1s minimum

        with patch("profiles.load_embedding", return_value=mock_profile_embedding), \
             patch("audio.load_and_normalize") as mock_audio, \
             patch("audio.validate_duration") as mock_validate:

            import torch
            mock_audio.return_value = (torch.randn(1, 4800), 0.3)

            from audio import AudioProcessingError
            mock_validate.side_effect = AudioProcessingError("Audio too short: 0.3s < 1s minimum")

            resp = client.post(
                "/v1/extract",
                data={"profile_id": "test-profile"},
                files={"audio": ("test.wav", short_wav, "audio/wav")},
            )

        assert resp.status_code == 400

    def test_model_not_loaded_returns_503(self, client):
        with patch("extraction.is_model_loaded", return_value=False):
            resp = client.post(
                "/v1/extract",
                data={"profile_id": "test-profile"},
                files={"audio": ("test.wav", _wav_bytes(), "audio/wav")},
            )

        assert resp.status_code == 503

    def test_empty_audio_returns_400(self, client, mock_profile_embedding):
        with patch("profiles.load_embedding", return_value=mock_profile_embedding):
            resp = client.post(
                "/v1/extract",
                data={"profile_id": "test-profile"},
                files={"audio": ("test.wav", b"", "audio/wav")},
            )

        assert resp.status_code == 400
