"""Target Speaker Extraction using blind separation + speaker identification.

Approach: SpeechBrain SepFormer separates all sources from a mixture,
then TitaNet embeddings identify which separated source matches the
enrolled speaker profile. This avoids TSE models that require reference
audio clips — we reuse the existing 192-dim TitaNet embeddings directly.

Pipeline:
  1. SepFormer separates the mixture into N sources (typically 2).
  2. TitaNet extracts an embedding from each separated source.
  3. Cosine similarity selects the source closest to the enrolled profile.
  4. If the best similarity exceeds a minimum threshold, return that source;
     otherwise the target speaker is considered absent.
"""

import logging
import time
from typing import Optional

import numpy as np
import torch

from config import settings

logger = logging.getLogger(__name__)

# Global model reference — loaded lazily so startup is fast when TSE is disabled.
_separator = None
_separator_loaded = False


def load_model() -> None:
    """Load the SepFormer separation model. Called once at startup if TSE is enabled."""
    global _separator, _separator_loaded

    if _separator_loaded:
        return

    logger.info("Loading TSE separation model %s ...", settings.TSE_MODEL)
    try:
        from speechbrain.inference.separation import SepformerSeparation

        _separator = SepformerSeparation.from_hparams(
            source=settings.TSE_MODEL,
            savedir=str(settings.CACHE_DIR / "speechbrain" / "sepformer"),
        )
        _separator_loaded = True
        logger.info("TSE separation model loaded successfully")
    except Exception:
        logger.exception("Failed to load TSE separation model")
        _separator_loaded = True  # Prevent repeated load attempts
        raise


def is_model_loaded() -> bool:
    """Check if the separation model is loaded and ready."""
    return _separator is not None


def separate_sources(waveform: torch.Tensor) -> torch.Tensor:
    """Separate a mono waveform into individual source signals.

    Args:
        waveform: Tensor of shape [1, samples] at 8kHz (SepFormer's expected rate).

    Returns:
        Tensor of shape [num_sources, samples] with separated waveforms.

    Raises:
        RuntimeError: If the separation model is not loaded.
    """
    if _separator is None:
        raise RuntimeError("TSE separation model not loaded")

    with torch.no_grad():
        # SepFormer expects [batch, samples]; returns [batch, samples, num_sources]
        est_sources = _separator.separate_batch(waveform)
        # Squeeze batch dim and transpose to [num_sources, samples]
        return est_sources.squeeze(0).T


def _resample(waveform: torch.Tensor, orig_sr: int, target_sr: int) -> torch.Tensor:
    """Resample a waveform tensor."""
    if orig_sr == target_sr:
        return waveform
    import torchaudio

    resampler = torchaudio.transforms.Resample(orig_freq=orig_sr, new_freq=target_sr)
    return resampler(waveform)


def extract_target_speaker(
    waveform: np.ndarray,
    reference_embedding: np.ndarray,
    sample_rate: int = 16000,
) -> Optional[np.ndarray]:
    """Extract the target speaker's waveform from a mixture.

    Uses blind source separation (SepFormer) followed by speaker identification
    (TitaNet embedding cosine similarity) to isolate the enrolled speaker.

    Args:
        waveform: Mono audio as float32 numpy array at ``sample_rate`` Hz.
        reference_embedding: Enrolled speaker's TitaNet embedding (192-dim).
        sample_rate: Sample rate of the input waveform (default 16000).

    Returns:
        Extracted speaker waveform as float32 numpy array at ``sample_rate`` Hz,
        or ``None`` if the target speaker is not detected in the mixture
        (best similarity below threshold or output is near-silent).
    """
    if _separator is None:
        raise RuntimeError("TSE separation model not loaded")

    import embeddings
    import verification

    start = time.monotonic()

    # SepFormer models are trained on 8kHz audio — resample down, separate, resample back.
    separator_sr = 8000
    wav_tensor = torch.from_numpy(waveform).float()
    if wav_tensor.dim() == 1:
        wav_tensor = wav_tensor.unsqueeze(0)  # [1, samples]

    wav_8k = _resample(wav_tensor, sample_rate, separator_sr)

    # Separate into individual sources
    sources = separate_sources(wav_8k)  # [num_sources, samples]
    num_sources = sources.shape[0]
    logger.debug("Separated %d sources in %.1fms", num_sources, (time.monotonic() - start) * 1000)

    # Resample each source back to the original rate for TitaNet embedding extraction.
    best_score = -1.0
    best_idx = -1
    best_source = None

    for i in range(num_sources):
        source_8k = sources[i].unsqueeze(0)  # [1, samples]
        source_orig = _resample(source_8k, separator_sr, sample_rate)

        # Skip near-silent sources
        rms = torch.sqrt(torch.mean(source_orig**2)).item()
        if rms < settings.TSE_MIN_OUTPUT_RMS:
            logger.debug("Source %d is near-silent (RMS=%.6f), skipping", i, rms)
            continue

        # Extract TitaNet embedding and compare with reference
        source_embedding = embeddings.extract_embedding(source_orig)
        score = verification.cosine_similarity(reference_embedding, source_embedding)
        logger.debug("Source %d: cosine_similarity=%.4f, RMS=%.6f", i, score, rms)

        if score > best_score:
            best_score = score
            best_idx = i
            best_source = source_orig

    elapsed_ms = (time.monotonic() - start) * 1000
    logger.info(
        "TSE completed in %.1fms: %d sources, best_source=%d, best_score=%.4f, threshold=%.4f",
        elapsed_ms, num_sources, best_idx, best_score, settings.TSE_MIN_SPEAKER_SCORE,
    )

    # If no source matches well enough, the target speaker is absent.
    if best_score < settings.TSE_MIN_SPEAKER_SCORE:
        logger.info("Target speaker not detected (best_score=%.4f < threshold=%.4f)", best_score, settings.TSE_MIN_SPEAKER_SCORE)
        return None

    # Return the best-matching source as a numpy array [samples].
    return best_source.squeeze(0).numpy()
