"""Embedding extraction using NeMo TitaNet."""

import logging
from typing import Optional

import numpy as np
import torch

from config import settings

logger = logging.getLogger(__name__)

# Global model reference
_model = None
_device = None


def _resolve_device() -> str:
    """Resolve the compute device based on config and availability."""
    device = settings.DEVICE
    if device == "auto":
        if torch.cuda.is_available():
            return "cuda"
        return "cpu"
    return device


def load_model() -> None:
    """Load the TitaNet model. Called once at startup."""
    global _model, _device

    _device = _resolve_device()
    logger.info("Loading model %s on device %s...", settings.MODEL, _device)

    try:
        import nemo.collections.asr as nemo_asr

        _model = nemo_asr.models.EncDecSpeakerLabelModel.from_pretrained(
            model_name=settings.MODEL
        )
        _model.eval()
        if _device == "cuda":
            _model = _model.cuda()
        logger.info("Model loaded successfully")
    except Exception:
        logger.exception("Failed to load model")
        raise


def is_model_loaded() -> bool:
    """Check if the model is loaded and ready."""
    return _model is not None


def get_embedding_dim() -> int:
    """Return the embedding dimension of the loaded model."""
    if _model is None:
        return 0
    try:
        # TitaNet models expose decoder output dim
        return _model.decoder.linear.out_features
    except AttributeError:
        return 192  # Default TitaNet-Large dimension


def get_device() -> str:
    """Return the active compute device."""
    return _device or "unknown"


def extract_embedding(waveform: torch.Tensor) -> np.ndarray:
    """Extract speaker embedding from a waveform tensor.

    Args:
        waveform: Tensor of shape [1, samples] at target sample rate

    Returns:
        numpy array of shape [embedding_dim]
    """
    if _model is None:
        raise RuntimeError("Model not loaded")

    with torch.no_grad():
        if _device == "cuda":
            waveform = waveform.cuda()

        # NeMo expects [batch, samples] without the channel dim
        if waveform.dim() == 2 and waveform.shape[0] == 1:
            audio_signal = waveform
        else:
            audio_signal = waveform.unsqueeze(0)

        audio_length = torch.tensor([audio_signal.shape[1]], dtype=torch.long)
        if _device == "cuda":
            audio_length = audio_length.cuda()

        _, embedding = _model.forward(
            input_signal=audio_signal, input_signal_length=audio_length
        )

        return embedding.cpu().numpy().flatten()


def extract_embeddings_multi(waveforms: list[torch.Tensor]) -> list[np.ndarray]:
    """Extract embeddings from multiple waveforms.

    Structured for future multi-sample enrollment support.
    """
    return [extract_embedding(w) for w in waveforms]


def aggregate_embeddings(embeddings: list[np.ndarray]) -> np.ndarray:
    """Aggregate multiple embeddings into a canonical profile embedding.

    Uses mean + L2 normalization for stability.
    """
    if len(embeddings) == 1:
        emb = embeddings[0]
    else:
        emb = np.mean(embeddings, axis=0)

    # L2 normalize
    norm = np.linalg.norm(emb)
    if norm > 0:
        emb = emb / norm

    return emb
