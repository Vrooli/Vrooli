"""Verification logic: cosine similarity, thresholding, result shaping."""

import time

import numpy as np

from config import settings


def cosine_similarity(a: np.ndarray, b: np.ndarray) -> float:
    """Compute cosine similarity between two embeddings."""
    dot = np.dot(a, b)
    norm_a = np.linalg.norm(a)
    norm_b = np.linalg.norm(b)

    if norm_a == 0 or norm_b == 0:
        return 0.0

    return float(dot / (norm_a * norm_b))


def verify(
    profile_embedding: np.ndarray,
    candidate_embedding: np.ndarray,
    threshold: float | None = None,
) -> dict:
    """Compare a candidate embedding against a stored profile embedding.

    Args:
        profile_embedding: The enrolled speaker's embedding
        candidate_embedding: The embedding from verification audio
        threshold: Optional threshold override (uses default if None)

    Returns:
        Dict with score, matched, threshold
    """
    if threshold is None:
        threshold = settings.DEFAULT_THRESHOLD

    start_time = time.monotonic()
    score = cosine_similarity(profile_embedding, candidate_embedding)
    duration_ms = (time.monotonic() - start_time) * 1000

    matched = score >= threshold

    return {
        "score": round(score, 6),
        "matched": matched,
        "threshold": threshold,
        "duration_ms": round(duration_ms, 2),
    }
