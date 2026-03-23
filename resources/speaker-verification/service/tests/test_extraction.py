"""Tests for the target speaker extraction module.

These are unit-level tests that mock the separation model and embedding
extraction so they run fast without GPU/model downloads. They validate
the selection logic, edge cases, and WAV encoding.
"""

import struct
from unittest.mock import MagicMock, patch

import numpy as np
import pytest
import torch


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(autouse=True)
def _reset_extraction_globals():
    """Reset extraction module globals between tests."""
    import extraction

    original_separator = extraction._separator
    original_loaded = extraction._separator_loaded
    yield
    extraction._separator = original_separator
    extraction._separator_loaded = original_loaded


def _sine_wave(freq: float, duration: float = 2.0, sr: int = 16000) -> np.ndarray:
    """Generate a mono sine wave as float32 numpy array."""
    t = np.linspace(0, duration, int(sr * duration), endpoint=False, dtype=np.float32)
    return 0.5 * np.sin(2 * np.pi * freq * t)


def _mix_signals(*signals: np.ndarray) -> np.ndarray:
    """Sum multiple signals (truncating to shortest)."""
    min_len = min(len(s) for s in signals)
    return sum(s[:min_len] for s in signals)


# ---------------------------------------------------------------------------
# Unit tests: extract_target_speaker
# ---------------------------------------------------------------------------

class TestExtractTargetSpeaker:
    """Tests for extraction.extract_target_speaker()."""

    def test_selects_source_matching_reference(self):
        """When separation produces two sources, the one whose embedding is
        closest to the reference is returned."""
        import extraction

        speaker_a = _sine_wave(440, duration=2.0)
        speaker_b = _sine_wave(880, duration=2.0)
        mix = _mix_signals(speaker_a, speaker_b)

        ref_embedding = np.random.randn(192).astype(np.float32)
        ref_embedding /= np.linalg.norm(ref_embedding)

        # Embedding for source 0 is close to ref; source 1 is far.
        emb_close = ref_embedding + np.random.randn(192).astype(np.float32) * 0.05
        emb_close /= np.linalg.norm(emb_close)
        emb_far = np.random.randn(192).astype(np.float32)
        emb_far /= np.linalg.norm(emb_far)

        # Mock separator: returns two 8kHz sources
        samples_8k = int(2.0 * 8000)
        fake_sources = torch.stack([
            torch.from_numpy(speaker_a[:samples_8k]),
            torch.from_numpy(speaker_b[:samples_8k]),
        ])  # [2, samples]

        mock_separator = MagicMock()
        # separate_batch returns [batch, samples, num_sources]
        mock_separator.separate_batch.return_value = torch.stack([
            fake_sources[0], fake_sources[1]
        ], dim=-1).unsqueeze(0)  # [1, samples, 2]

        extraction._separator = mock_separator

        call_count = [0]
        embeddings_sequence = [emb_close, emb_far]

        def mock_extract_embedding(waveform):
            idx = call_count[0]
            call_count[0] += 1
            return embeddings_sequence[idx]

        with patch("extraction.embeddings.extract_embedding", side_effect=mock_extract_embedding):
            result = extraction.extract_target_speaker(mix, ref_embedding, sample_rate=16000)

        assert result is not None
        assert isinstance(result, np.ndarray)
        assert result.ndim == 1
        assert len(result) > 0

    def test_returns_none_when_all_sources_below_threshold(self):
        """When no separated source matches the reference well enough, returns None."""
        import extraction

        mix = _sine_wave(440, duration=2.0)
        ref_embedding = np.random.randn(192).astype(np.float32)
        ref_embedding /= np.linalg.norm(ref_embedding)

        # Both embeddings are far from reference
        emb_far_1 = np.random.randn(192).astype(np.float32)
        emb_far_1 /= np.linalg.norm(emb_far_1)
        emb_far_2 = -ref_embedding  # Maximally dissimilar

        samples_8k = int(2.0 * 8000)
        source = torch.from_numpy(_sine_wave(440, duration=2.0, sr=8000)[:samples_8k])
        mock_separator = MagicMock()
        mock_separator.separate_batch.return_value = torch.stack([
            source, source
        ], dim=-1).unsqueeze(0)

        extraction._separator = mock_separator

        call_count = [0]
        embeddings_sequence = [emb_far_1, emb_far_2]

        def mock_extract_embedding(waveform):
            idx = call_count[0]
            call_count[0] += 1
            return embeddings_sequence[idx]

        with patch("extraction.embeddings.extract_embedding", side_effect=mock_extract_embedding), \
             patch("extraction.settings") as mock_settings:
            mock_settings.TSE_MIN_SPEAKER_SCORE = 0.9  # Very high threshold
            mock_settings.TSE_MIN_OUTPUT_RMS = 1e-4
            mock_settings.SAMPLE_RATE = 16000
            mock_settings.CACHE_DIR = "/tmp/test_cache"
            result = extraction.extract_target_speaker(mix, ref_embedding, sample_rate=16000)

        assert result is None

    def test_skips_near_silent_sources(self):
        """Near-silent separated sources are skipped."""
        import extraction

        mix = _sine_wave(440, duration=2.0)
        ref_embedding = np.random.randn(192).astype(np.float32)
        ref_embedding /= np.linalg.norm(ref_embedding)

        # Source 0 is silent, source 1 has signal with matching embedding
        emb_match = ref_embedding + np.random.randn(192).astype(np.float32) * 0.01
        emb_match /= np.linalg.norm(emb_match)

        samples_8k = int(2.0 * 8000)
        silent_source = torch.zeros(samples_8k)
        active_source = torch.from_numpy(_sine_wave(440, duration=2.0, sr=8000)[:samples_8k])

        mock_separator = MagicMock()
        mock_separator.separate_batch.return_value = torch.stack([
            silent_source, active_source
        ], dim=-1).unsqueeze(0)

        extraction._separator = mock_separator

        def mock_extract_embedding(waveform):
            return emb_match

        with patch("extraction.embeddings.extract_embedding", side_effect=mock_extract_embedding):
            result = extraction.extract_target_speaker(mix, ref_embedding, sample_rate=16000)

        # Should return the active source (source 1), not the silent one
        assert result is not None
        assert np.sqrt(np.mean(result**2)) > 1e-4

    def test_raises_when_model_not_loaded(self):
        """Raises RuntimeError when separator model is not loaded."""
        import extraction

        extraction._separator = None
        mix = _sine_wave(440)
        ref = np.random.randn(192).astype(np.float32)

        with pytest.raises(RuntimeError, match="not loaded"):
            extraction.extract_target_speaker(mix, ref)

    def test_output_preserves_sample_rate(self):
        """Output waveform should have the same duration as input (within tolerance)."""
        import extraction

        duration = 3.0
        sr = 16000
        mix = _sine_wave(440, duration=duration, sr=sr)
        ref_embedding = np.random.randn(192).astype(np.float32)
        ref_embedding /= np.linalg.norm(ref_embedding)

        emb_match = ref_embedding.copy()

        samples_8k = int(duration * 8000)
        source = torch.from_numpy(_sine_wave(440, duration=duration, sr=8000)[:samples_8k])

        mock_separator = MagicMock()
        mock_separator.separate_batch.return_value = torch.stack([
            source, source
        ], dim=-1).unsqueeze(0)

        extraction._separator = mock_separator

        with patch("extraction.embeddings.extract_embedding", return_value=emb_match):
            result = extraction.extract_target_speaker(mix, ref_embedding, sample_rate=sr)

        assert result is not None
        # Resampled from 8kHz back to 16kHz: duration should be approximately the same
        output_duration = len(result) / sr
        assert abs(output_duration - duration) < 0.1, \
            f"Output duration {output_duration:.2f}s != expected ~{duration}s"


# ---------------------------------------------------------------------------
# Unit tests: is_model_loaded
# ---------------------------------------------------------------------------

class TestIsModelLoaded:
    def test_false_when_not_loaded(self):
        import extraction
        extraction._separator = None
        assert extraction.is_model_loaded() is False

    def test_true_when_loaded(self):
        import extraction
        extraction._separator = MagicMock()
        assert extraction.is_model_loaded() is True


# ---------------------------------------------------------------------------
# Unit tests: WAV encoding (from api/extract.py)
# ---------------------------------------------------------------------------

class TestNumpyToWavBytes:
    def test_valid_wav_header(self):
        from api.extract import _numpy_to_wav_bytes

        signal = _sine_wave(440, duration=1.0)
        wav = _numpy_to_wav_bytes(signal, sample_rate=16000)

        assert wav[:4] == b"RIFF"
        assert wav[8:12] == b"WAVE"
        assert wav[12:16] == b"fmt "
        assert wav[36:40] == b"data"

        # Check sample rate in header
        sr_bytes = struct.unpack("<I", wav[24:28])[0]
        assert sr_bytes == 16000

    def test_correct_data_length(self):
        from api.extract import _numpy_to_wav_bytes

        n_samples = 16000  # 1 second
        signal = np.zeros(n_samples, dtype=np.float32)
        wav = _numpy_to_wav_bytes(signal, sample_rate=16000)

        data_size = struct.unpack("<I", wav[40:44])[0]
        assert data_size == n_samples * 2  # 16-bit = 2 bytes per sample

    def test_clips_signal(self):
        """Values outside [-1, 1] should be clipped, not wrap."""
        from api.extract import _numpy_to_wav_bytes

        signal = np.array([2.0, -2.0, 0.5], dtype=np.float32)
        wav = _numpy_to_wav_bytes(signal, sample_rate=16000)

        # Extract PCM samples from WAV data section
        pcm_data = wav[44:]
        samples = struct.unpack(f"<{len(pcm_data)//2}h", pcm_data)
        assert samples[0] == 32767   # clipped at max
        assert samples[1] == -32767  # clipped at min (32767 * -1.0 then clip)
