"""Voice-activity detection for speaker embedding.

Trimming non-voiced audio before ECAPA embedding is the single biggest lever for
speaker-verification quality. ECAPA-TDNN pools statistics across the whole clip,
so silence and room noise dilute the voiceprint and collapse genuine and impostor
scores toward a common low value. Both enrollment and verification embed only the
voiced audio through one shared path so the two stay symmetric.

The detector is selected via the ``SPEAKER_VAD`` environment variable
(``energy`` | ``none``). The energy detector is dependency-free: it operates on
the already-present torch waveform tensor and a little pure-Python math. A
``silero`` branch can be added later behind this same ``trim`` interface with no
re-architecting; it is intentionally NOT implemented here because it would add a
model dependency.

The energy/threshold/mask math is pure Python (operates on lists of frame RMS
values) so it is unit-testable without torch; only ``EnergyVAD.trim`` touches the
tensor, to frame the signal and gather the voiced frames back together.
"""

from __future__ import annotations

import os
from typing import List, Optional, Protocol, Tuple

try:  # torch is a runtime dep of the detectors, NOT of the pure helpers below,
    import torch  # so the threshold/mask math stays importable (and testable)
except ImportError:  # on a host without torch.
    torch = None  # type: ignore[assignment]


# ---------------------------------------------------------------------------
# Pure helpers (no torch) — independently unit-testable
# ---------------------------------------------------------------------------


def percentile(values: List[float], q: float) -> float:
    """Linear-interpolated percentile of ``values`` at quantile ``q`` in [0,1].

    Pure Python so the threshold logic can be tested without numpy/torch.
    """
    if not values:
        return 0.0
    if len(values) == 1:
        return values[0]
    ordered = sorted(values)
    pos = max(0.0, min(1.0, q)) * (len(ordered) - 1)
    lo = int(pos)
    hi = min(lo + 1, len(ordered) - 1)
    frac = pos - lo
    return ordered[lo] * (1.0 - frac) + ordered[hi] * frac


def estimate_threshold(
    rms: List[float], noise_percentile: float, margin_fraction: float
) -> Tuple[float, float, float]:
    """Adaptive voiced-frame threshold from per-frame RMS energies.

    The threshold sits a fixed fraction of the dynamic range above the noise
    floor: ``noise + margin_fraction * (peak - noise)``. This adapts to both a
    near-constant signal (a pure tone: peak ~= noise, so nearly everything is
    voiced) and speech-over-silence (low noise floor, high peak, so silence
    frames fall below the threshold). Returns ``(threshold, peak, noise)``.
    """
    if not rms:
        return 0.0, 0.0, 0.0
    peak = max(rms)
    noise = percentile(rms, noise_percentile)
    threshold = noise + margin_fraction * (peak - noise)
    return threshold, peak, noise


def smooth_mask(mask: List[bool], hangover_frames: int, pad_frames: int) -> List[bool]:
    """Merge short silence gaps between voiced runs, then dilate by ``pad_frames``.

    ``hangover_frames`` bridges brief inter-word pauses so a single utterance is
    not chopped into fragments; ``pad_frames`` keeps a little context around each
    voiced region (onsets/offsets carry speaker-identity cues).
    """
    n = len(mask)
    if n == 0:
        return []
    out = list(mask)

    voiced = [i for i, m in enumerate(mask) if m]
    if voiced:
        for a, b in zip(voiced, voiced[1:]):
            if b - a - 1 <= hangover_frames:
                for j in range(a + 1, b):
                    out[j] = True

    if pad_frames > 0:
        dilated = list(out)
        for i, m in enumerate(out):
            if m:
                for j in range(max(0, i - pad_frames), min(n, i + pad_frames + 1)):
                    dilated[j] = True
        out = dilated

    return out


# ---------------------------------------------------------------------------
# Detector interface + implementations
# ---------------------------------------------------------------------------


class VoiceActivityDetector(Protocol):
    """Trims a mono waveform down to its voiced span before embedding."""

    name: str

    def trim(self, waveform: "torch.Tensor", sr: int) -> Tuple["torch.Tensor", float]:
        """Return ``(voiced_waveform[1, k], voiced_seconds)``.

        ``voiced_waveform`` is the concatenation of the frames judged to contain
        speech, shape ``(1, k)``; ``voiced_seconds`` is its duration. When the
        clip is silent, ``k`` is 0 and ``voiced_seconds`` is 0.0 — callers must
        treat that as "insufficient voiced audio" and not embed it.
        """
        ...


class EnergyVAD:
    """Dependency-free energy-based voice-activity trimmer."""

    name = "energy"

    def __init__(
        self,
        frame_ms: float = 20.0,
        margin_fraction: float = 0.08,
        noise_percentile: float = 0.1,
        hangover_ms: float = 200.0,
        pad_ms: float = 50.0,
        abs_silence: float = 1e-4,
    ) -> None:
        self.frame_ms = frame_ms
        self.margin_fraction = margin_fraction
        self.noise_percentile = noise_percentile
        self.hangover_ms = hangover_ms
        self.pad_ms = pad_ms
        self.abs_silence = abs_silence

    def trim(self, waveform: "torch.Tensor", sr: int) -> Tuple["torch.Tensor", float]:
        mono = waveform.mean(dim=0) if waveform.dim() == 2 else waveform.reshape(-1)
        empty = mono.new_zeros((1, 0))
        n = int(mono.numel())
        if n == 0:
            return empty, 0.0

        frame_len = max(1, int(sr * self.frame_ms / 1000.0))
        if n < frame_len:
            # Too short to frame; keep the whole clip unless it is silent.
            if float(mono.abs().max()) < self.abs_silence:
                return empty, 0.0
            return mono.reshape(1, n), float(n) / float(sr)

        num_frames = n // frame_len
        frames = mono[: num_frames * frame_len].reshape(num_frames, frame_len)
        rms = frames.pow(2).mean(dim=1).clamp_min(0.0).sqrt()

        if float(rms.max()) < self.abs_silence:
            return empty, 0.0

        rms_list = rms.tolist()
        threshold, _, _ = estimate_threshold(
            rms_list, self.noise_percentile, self.margin_fraction
        )
        mask = [v > threshold for v in rms_list]
        hangover_frames = int(round(self.hangover_ms / self.frame_ms))
        pad_frames = int(round(self.pad_ms / self.frame_ms))
        mask = smooth_mask(mask, hangover_frames, pad_frames)

        voiced_idx = [i for i, m in enumerate(mask) if m]
        if not voiced_idx:
            return empty, 0.0

        idx = torch.tensor(voiced_idx, dtype=torch.long)
        voiced = frames.index_select(0, idx).reshape(1, -1)
        voiced_seconds = float(len(voiced_idx) * frame_len) / float(sr)
        return voiced, voiced_seconds


class NoOpVAD:
    """Passthrough detector: embeds the whole clip (no trimming)."""

    name = "none"

    def trim(self, waveform: "torch.Tensor", sr: int) -> Tuple["torch.Tensor", float]:
        wav = waveform if waveform.dim() == 2 else waveform.reshape(1, -1)
        return wav, float(wav.size(-1)) / float(sr)


def build_vad(name: Optional[str] = None) -> "VoiceActivityDetector":
    """Construct the configured detector. ``name`` defaults to ``$SPEAKER_VAD``."""
    selected = (name or os.environ.get("SPEAKER_VAD", "energy")).strip().lower()
    if selected in ("", "energy"):
        return EnergyVAD()
    if selected == "none":
        return NoOpVAD()
    if selected == "silero":
        raise ValueError(
            "SPEAKER_VAD=silero is reserved but not implemented in this build "
            "(it would add a model dependency); use 'energy' or 'none'"
        )
    raise ValueError(
        f"unknown SPEAKER_VAD={selected!r} (supported: energy, none)"
    )
