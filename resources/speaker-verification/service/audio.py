"""Audio processing: decode, normalize, validate."""

import subprocess
import tempfile
from pathlib import Path

import torch
import torchaudio

from config import settings


class AudioProcessingError(Exception):
    """Raised when audio processing fails."""


def _ffmpeg_to_wav(src: str, dst: str) -> None:
    """Use ffmpeg to convert any supported audio format to 16-bit PCM WAV."""
    result = subprocess.run(
        [
            "ffmpeg", "-y",
            "-i", src,
            "-f", "wav",
            "-acodec", "pcm_s16le",
            "-ar", str(settings.SAMPLE_RATE),
            "-ac", "1",
            dst,
        ],
        capture_output=True,
        timeout=30,
    )
    if result.returncode != 0:
        stderr = result.stderr.decode(errors="replace").strip().split("\n")[-1]
        raise AudioProcessingError(f"Failed to decode audio: {stderr}")


def load_and_normalize(audio_bytes: bytes) -> tuple[torch.Tensor, float]:
    """Load audio bytes, convert to mono 16kHz, return tensor and duration.

    Browsers record as WebM/Opus, but torchaudio's soundfile backend only
    handles formats that libsndfile supports (WAV, FLAC, OGG-Vorbis).
    We normalise through ffmpeg first so any input format works.

    Returns:
        Tuple of (waveform tensor [1, samples], duration in seconds)
    """
    # Write raw upload to a temp file (no extension — could be any format).
    with tempfile.NamedTemporaryFile(suffix="", delete=False) as tmp_in:
        tmp_in.write(audio_bytes)
        in_path = tmp_in.name

    wav_path = in_path + ".wav"
    try:
        # Convert to mono 16 kHz WAV via ffmpeg (handles WebM, MP3, OGG, etc.)
        _ffmpeg_to_wav(in_path, wav_path)
        waveform, sample_rate = torchaudio.load(wav_path)
    except AudioProcessingError:
        raise
    except Exception as e:
        raise AudioProcessingError(f"Failed to decode audio: {e}") from e
    finally:
        Path(in_path).unlink(missing_ok=True)
        Path(wav_path).unlink(missing_ok=True)

    # ffmpeg already converts to mono + target sample rate, but if the
    # settings change we still handle it here as a safety net.
    if waveform.shape[0] > 1:
        waveform = torch.mean(waveform, dim=0, keepdim=True)

    if sample_rate != settings.SAMPLE_RATE:
        resampler = torchaudio.transforms.Resample(
            orig_freq=sample_rate, new_freq=settings.SAMPLE_RATE
        )
        waveform = resampler(waveform)

    duration_seconds = waveform.shape[1] / settings.SAMPLE_RATE
    return waveform, duration_seconds


def validate_duration(duration_seconds: float, min_seconds: float) -> None:
    """Validate audio meets minimum duration requirement."""
    if duration_seconds < min_seconds:
        raise AudioProcessingError(
            f"Audio too short: {duration_seconds:.1f}s < {min_seconds}s minimum"
        )


def check_signal_presence(waveform: torch.Tensor) -> None:
    """Check that audio contains meaningful signal (not silence)."""
    rms = torch.sqrt(torch.mean(waveform**2)).item()
    if rms < 1e-6:
        raise AudioProcessingError("Audio appears to be silence")
