"""SpeechBrain ECAPA-TDNN speaker-verification HTTP server.

Exposes multi-clip enrollment + verification + target-speaker extraction over a
small FastAPI surface. The contract here is consumed byte-for-byte by the
audio-tools Go client (scenarios/audio-tools/api/internal/stt/pipeline/
speaker_client.go) -- endpoint paths, multipart field names, and response JSON
keys MUST NOT drift.

Embeddings are 192-dimensional ECAPA-TDNN vectors computed from 16 kHz mono
audio. Both enrollment and verification embed only the VOICED span of the clip
(see vad.py) through one shared helper, so the two stay symmetric and silence /
room noise does not dilute the voiceprint.

A profile is one IDENTITY holding N labeled enrollment clips, each storing its
own embedding. Scoring is max-over-clips (no centroid): the score of a test
embedding against a profile is the largest cosine similarity between the test
embedding and any single clip's embedding. We dropped centroid aggregation in
v0.4 because spectrally-divergent enrollment clips (whisper + normal + phone +
laptop) pull the mean toward neutral and depress genuine scores; the max is
mathematically dominated by — and therefore strictly better than — the hybrid
``max(centroid, best_clip)`` mode that preceded it.

At enrollment we also compute a self-consistency score: the cosine similarity
between the new clip and the strongest existing clip in the same profile. A low
self-score warns (does NOT block) — it tells the user the clip is recorded in
substantially different conditions than the others and may not help recognition.
The clip is still stored so the user controls their own data.
"""

from __future__ import annotations

import asyncio
import io
import json
import os
import subprocess
import tempfile
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

import torch
import torchaudio
from fastapi import FastAPI, File, Form, UploadFile
from fastapi.responses import JSONResponse, Response

from vad import build_vad

# ---------------------------------------------------------------------------
# Configuration (all overridable via environment for compose / GPU overlays)
# ---------------------------------------------------------------------------

SERVER_VERSION = "0.4.0"
MODEL_NAME = os.environ.get(
    "SPEAKER_VERIFICATION_MODEL", "speechbrain/spkrec-ecapa-voxceleb"
)
EMBEDDING_DIM = 192
SAMPLE_RATE = 16000

# Voice-activity trimming + minimum-voiced-duration guards. ECAPA needs a few
# seconds of voiced speech for a stable enrollment embedding; a sub-second
# window is not reliable. Enrollment rejects below MIN_ENROLL; verification
# returns sufficient:false (not a fabricated score) below MIN_VERIFY.
MIN_ENROLL_VOICED_SECONDS = float(
    os.environ.get("SPEAKER_MIN_ENROLL_VOICED_SECONDS", "3.0")
)
MIN_VERIFY_VOICED_SECONDS = float(
    os.environ.get("SPEAKER_MIN_VERIFY_VOICED_SECONDS", "1.0")
)

# Self-consistency warning threshold at enrollment time. When a newly enrolled
# clip's max cosine against any existing clip in the profile is below this, the
# response carries a `self_consistency_warning` flag and the score so the
# operator can decide whether to re-record. The clip is stored regardless.
SELF_CONSISTENCY_THRESHOLD = float(
    os.environ.get("SPEAKER_SELF_CONSISTENCY_THRESHOLD", "0.5")
)


def _env_bool(name: str, default: bool = False) -> bool:
    raw = os.environ.get(name)
    if raw is None:
        return default
    return raw.strip().lower() in ("1", "true", "yes", "on")


# Optional spectral denoise of the audio BEFORE embedding. Default OFF: the
# resource-side VAD trim is the primary noise mitigation (silence is the
# dominant diluter), and spectral denoise (ffmpeg afftdn) can distort timbre and
# hurt ECAPA. This is an embedding-path knob independent of the audio-tools
# transcription denoise toggle — speaker identity never depends on it.
EMBED_DENOISE = _env_bool("SPEAKER_EMBED_DENOISE", False)

# The one canonical default verify threshold. Every layer (manifest, Go config,
# CLI/handlers, this form default) resolves to the same value; calibrate against
# real voices after VAD trim lands and record the chosen number in
# scenarios/audio-tools/docs/reference/configuration.md.
DEFAULT_VERIFY_THRESHOLD = os.environ.get("SPEAKER_DEFAULT_THRESHOLD", "0.5")

# Target-speaker extraction = source separation (split the mixture into N voices)
# + ECAPA target-selection (pick the separated source whose embedding best
# matches any of the enrolled clips by max cosine). The published SepFormer
# separation checkpoints (wsj02mix / libri2mix, 2 speakers) run at 8 kHz; the
# 16 kHz checkpoints are enhancement (single-source denoise), not separation. So
# the default is the 8 kHz 2-speaker model and audio is resampled 16k<->8k.
EXTRACTION_MODEL = os.environ.get(
    "SPEAKER_EXTRACTION_MODEL", "speechbrain/sepformer-wsj02mix"
)
EXTRACTION_SAMPLE_RATE = int(
    os.environ.get("SPEAKER_EXTRACTION_SAMPLE_RATE", "8000")
)
EXTRACTION_MATCH_THRESHOLD = float(
    os.environ.get("SPEAKER_EXTRACTION_MATCH_THRESHOLD", "0.25")
)

# Device resolution. "auto" (and the empty default) pick cuda when a GPU is
# visible, else cpu. An explicit "cuda" still downgrades to cpu when no GPU is
# present so the same CUDA image runs unchanged on CPU-only hosts. The chosen
# device is logged once at import so operators can confirm GPU use in the logs.
_DEVICE_REQUEST = os.environ.get("SPEAKER_VERIFICATION_DEVICE", "auto").strip().lower()
_CUDA_AVAILABLE = torch.cuda.is_available()
if _DEVICE_REQUEST in ("", "auto"):
    DEVICE = "cuda" if _CUDA_AVAILABLE else "cpu"
elif _DEVICE_REQUEST == "cuda" and not _CUDA_AVAILABLE:
    DEVICE = "cpu"
else:
    DEVICE = _DEVICE_REQUEST

_gpu_name = torch.cuda.get_device_name(0) if DEVICE == "cuda" and _CUDA_AVAILABLE else None
print(
    f"[speaker-verification] torch={torch.__version__} "
    f"device_request={_DEVICE_REQUEST or 'auto'} cuda_available={_CUDA_AVAILABLE} "
    f"device={DEVICE}" + (f" gpu={_gpu_name}" if _gpu_name else ""),
    flush=True,
)
if _DEVICE_REQUEST == "cuda" and not _CUDA_AVAILABLE:
    print(
        "[speaker-verification] WARNING: cuda requested but no GPU is visible to "
        "the container; falling back to cpu. Ensure the GPU compose overlay "
        "(runtime: nvidia + device reservation) is applied.",
        flush=True,
    )

PROFILE_STORE_DIR = Path(
    os.environ.get("SPEAKER_VERIFICATION_PROFILE_DIR", "/data/profiles")
)
MODEL_CACHE_DIR = Path(
    os.environ.get("SPEAKER_VERIFICATION_MODEL_CACHE", "/data/model-cache")
)
TEMP_DIR = Path(os.environ.get("SPEAKER_VERIFICATION_TEMP_DIR", "/tmp/speaker-verification"))

PROFILE_STORE_DIR.mkdir(parents=True, exist_ok=True)
MODEL_CACHE_DIR.mkdir(parents=True, exist_ok=True)
TEMP_DIR.mkdir(parents=True, exist_ok=True)

app = FastAPI(title="Speaker Verification", version=SERVER_VERSION)

# Model handles are eagerly warmed in a startup task.  Requests retain the
# idempotent accessors for safety, while /ready stays non-200 until both models
# can serve work.
_classifier = None
_separator = None
_vad = None
_model_warm_error: Optional[str] = None


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _load_model():
    """Load (and cache) the ECAPA-TDNN encoder. Idempotent."""
    global _classifier
    if _classifier is not None:
        return _classifier
    # Imported lazily so the module imports cleanly without model weights.
    from speechbrain.inference.speaker import EncoderClassifier

    _classifier = EncoderClassifier.from_hparams(
        source=MODEL_NAME,
        savedir=str(MODEL_CACHE_DIR / "spkrec-ecapa-voxceleb"),
        run_opts={"device": DEVICE},
    )
    return _classifier


def _get_vad():
    """Load and cache the VAD separately from the GPU-backed model."""
    global _vad
    if _vad is None:
        _vad = build_vad()
    return _vad


def _warm_models() -> None:
    """Warm the serving models without blocking FastAPI from exposing /ready."""
    global _model_warm_error
    try:
        _load_model()
        _get_vad()
        _model_warm_error = None
    except Exception as exc:  # noqa: BLE001 -- expose readiness failure safely
        _model_warm_error = str(exc)
        print(f"[speaker-verification] model warmup failed: {exc}", flush=True)


def _load_separator():
    """Load (and cache) the SepFormer source-separation model. Idempotent.

    Imported lazily so the module imports cleanly without model weights, and so
    a deployment that never enables extraction never pays the download/load
    cost.
    """
    global _separator
    if _separator is not None:
        return _separator
    from speechbrain.inference.separation import SepformerSeparation

    _separator = SepformerSeparation.from_hparams(
        source=EXTRACTION_MODEL,
        savedir=str(MODEL_CACHE_DIR / "sepformer-extraction"),
        run_opts={"device": DEVICE},
    )
    return _separator


def _embed_waveform(waveform: "torch.Tensor") -> List[float]:
    """L2-normalized 192-dim ECAPA embedding from an in-memory 16 kHz mono
    waveform tensor of shape (1, num_samples)."""
    classifier = _load_model()
    with torch.no_grad():
        emb = classifier.encode_batch(waveform)
    vec = emb.squeeze().detach().cpu()
    vec = torch.nn.functional.normalize(vec, dim=0)
    return vec.tolist()


def _waveform_to_pcm16(waveform: "torch.Tensor") -> bytes:
    """Encode a mono float waveform (1, n) in [-1, 1] to little-endian s16 PCM."""
    samples = waveform.squeeze().detach().cpu()
    clipped = torch.clamp(samples, -1.0, 1.0)
    ints = (clipped * 32767.0).round().to(torch.int16)
    return ints.numpy().tobytes()


# ---------------------------------------------------------------------------
# Audio decoding + embedding
# ---------------------------------------------------------------------------


def _decode_to_waveform(raw: bytes) -> Tuple[torch.Tensor, float]:
    """Decode arbitrary audio bytes (WebM/Opus/WAV/PCM/...) to a mono 16 kHz
    float waveform tensor of shape (1, num_samples).

    Tries torchaudio first; falls back to an ffmpeg transcode to WAV for
    container formats torchaudio cannot read directly (e.g. WebM/Opus).
    Returns (waveform, audio_seconds).
    """
    waveform = None
    sr = None
    try:
        waveform, sr = torchaudio.load(io.BytesIO(raw))
    except Exception:
        waveform = None

    if waveform is None:
        # ffmpeg fallback: decode any container to 16 kHz mono signed-16 WAV.
        with tempfile.NamedTemporaryFile(
            dir=str(TEMP_DIR), suffix=".in", delete=False
        ) as src:
            src.write(raw)
            src_path = src.name
        dst_path = src_path + ".wav"
        try:
            subprocess.run(
                [
                    "ffmpeg",
                    "-hide_banner",
                    "-loglevel",
                    "error",
                    "-y",
                    "-i",
                    src_path,
                    "-ac",
                    "1",
                    "-ar",
                    str(SAMPLE_RATE),
                    "-f",
                    "wav",
                    dst_path,
                ],
                check=True,
                capture_output=True,
            )
            waveform, sr = torchaudio.load(dst_path)
        finally:
            for path in (src_path, dst_path):
                try:
                    os.unlink(path)
                except OSError:
                    pass

    # Downmix to mono.
    if waveform.dim() == 2 and waveform.size(0) > 1:
        waveform = waveform.mean(dim=0, keepdim=True)
    elif waveform.dim() == 1:
        waveform = waveform.unsqueeze(0)

    # Resample to the model's expected rate.
    if sr != SAMPLE_RATE:
        waveform = torchaudio.functional.resample(waveform, sr, SAMPLE_RATE)

    audio_seconds = float(waveform.size(-1)) / float(SAMPLE_RATE)
    return waveform, audio_seconds


def _denoise_waveform(waveform: "torch.Tensor") -> "torch.Tensor":
    """Best-effort ffmpeg afftdn denoise of a 16 kHz mono waveform via a temp-WAV
    round-trip. Returns the input unchanged on any failure."""
    src = TEMP_DIR / f"dn-{uuid.uuid4().hex}.wav"
    dst = TEMP_DIR / f"dn-{uuid.uuid4().hex}.out.wav"
    try:
        torchaudio.save(str(src), waveform, SAMPLE_RATE)
        subprocess.run(
            [
                "ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
                "-i", str(src), "-af", "afftdn",
                "-ar", str(SAMPLE_RATE), "-ac", "1", str(dst),
            ],
            check=True,
            capture_output=True,
        )
        out, _ = torchaudio.load(str(dst))
        return out
    except Exception:  # noqa: BLE001 -- denoise is best-effort, never fatal
        return waveform
    finally:
        for path in (src, dst):
            try:
                path.unlink()
            except OSError:
                pass


def _voiced_embedding(raw: bytes) -> Tuple[Optional[List[float]], float, float]:
    """Decode -> (optional denoise) -> VAD-trim to the voiced span -> embed.

    The single embedding path for both enrollment and verification, so they stay
    symmetric. Returns (embedding, audio_seconds, voiced_seconds). When the clip
    has no voiced audio the embedding is None and voiced_seconds is 0.0; callers
    treat that as "insufficient voiced audio".
    """
    waveform, audio_seconds = _decode_to_waveform(raw)
    if EMBED_DENOISE:
        waveform = _denoise_waveform(waveform)
    vad = _get_vad()
    voiced, voiced_seconds = vad.trim(waveform, SAMPLE_RATE)
    if voiced_seconds <= 0.0 or int(voiced.size(-1)) == 0:
        return None, audio_seconds, 0.0
    return _embed_waveform(voiced), audio_seconds, voiced_seconds


def _cosine(a: List[float], b: List[float]) -> float:
    ta = torch.tensor(a, dtype=torch.float32)
    tb = torch.tensor(b, dtype=torch.float32)
    return float(torch.nn.functional.cosine_similarity(ta, tb, dim=0).item())


def _total_voiced_seconds(clips: List[Dict[str, Any]]) -> float:
    return float(sum(float(c.get("voiced_seconds", 0.0)) for c in clips))


def _best_match(
    test_embedding: List[float], clips: List[Dict[str, Any]]
) -> Tuple[float, str, str]:
    """Max-over-clips cosine. Returns (best_score, best_clip_label, best_clip_id).

    When the profile has no clips with embeddings, returns (-1.0, "", "").
    """
    best_score = -1.0
    best_label = ""
    best_id = ""
    for clip in clips:
        emb = clip.get("embedding")
        if not emb:
            continue
        score = _cosine(emb, test_embedding)
        if score > best_score:
            best_score = score
            best_label = clip.get("label", "")
            best_id = clip.get("clip_id", "")
    return best_score, best_label, best_id


def _self_consistency(
    new_embedding: List[float], existing_clips: List[Dict[str, Any]]
) -> Tuple[float, str, str]:
    """Max cosine between the new clip embedding and any existing clip in the
    same profile. Returns (-1.0, "", "") if there are no existing clips (the
    first clip in a profile has no self-consistency to check).
    """
    return _best_match(new_embedding, existing_clips)


def _extract_target(
    waveform: "torch.Tensor", clips: List[Dict[str, Any]]
) -> Tuple[bytes, float]:
    """Isolate the enrolled speaker from a 16 kHz mono mixture.

    Separates the mixture into candidate sources, embeds each with ECAPA, and
    returns (pcm16, score) for the source whose embedding has the highest
    max-cosine to any clip in the profile. The returned PCM is 16 kHz mono
    s16le, matching the input contract so it can re-enter the STT pipeline.
    """
    separator = _load_separator()

    mix = waveform  # (1, n) @ SAMPLE_RATE
    if EXTRACTION_SAMPLE_RATE != SAMPLE_RATE:
        mix = torchaudio.functional.resample(mix, SAMPLE_RATE, EXTRACTION_SAMPLE_RATE)

    with torch.no_grad():
        # SepformerSeparation.separate_batch expects (batch, time) and returns
        # (batch, time, n_sources).
        est = separator.separate_batch(mix)
    est = est.squeeze(0)  # (time, n_sources)
    n_sources = int(est.shape[-1])

    best_pcm: Optional[bytes] = None
    best_score = -1.0
    for i in range(n_sources):
        src = est[:, i].unsqueeze(0)  # (1, time) @ EXTRACTION_SAMPLE_RATE
        # Separators attenuate sources; renormalize so amplitude/embedding are
        # comparable to enrollment.
        peak = float(src.abs().max())
        if peak > 0:
            src = src / peak
        if EXTRACTION_SAMPLE_RATE != SAMPLE_RATE:
            src = torchaudio.functional.resample(src, EXTRACTION_SAMPLE_RATE, SAMPLE_RATE)
        emb = _embed_waveform(src)
        score, _, _ = _best_match(emb, clips)
        if score > best_score:
            best_score = score
            best_pcm = _waveform_to_pcm16(src)

    if best_pcm is None:
        # No sources came back (degenerate model output): return the mixture so
        # recognition still proceeds rather than dropping the audio.
        return _waveform_to_pcm16(waveform), 0.0
    return best_pcm, best_score


# ---------------------------------------------------------------------------
# Profile store (one JSON file per profile, keyed by profile_id)
#
# Record shape (v3 — max-over-clips, no centroid):
#   { id, display_name, notes, model_name, embedding_dim, sample_rate,
#     clips:[ {clip_id, label, embedding:[192], voiced_seconds, audio_seconds,
#             self_consistency_score, vad_model, created_at} ],
#     created_at, updated_at }
#
# Centroid was removed in v0.4: it's mathematically dominated by max-over-clips
# and spectrally-divergent enrollment conditions dilute it badly. Profile files
# written by older servers may still carry a "centroid" field; the loader
# ignores it (forward-only — no migration code).
# ---------------------------------------------------------------------------


def _profile_path(profile_id: str) -> Path:
    return PROFILE_STORE_DIR / f"{profile_id}.json"


def _save_profile(record: Dict[str, Any]) -> None:
    # Strip any legacy centroid so we don't keep paying for a stale aggregate.
    record.pop("centroid", None)
    path = _profile_path(record["id"])
    tmp = path.with_suffix(".json.tmp")
    with open(tmp, "w", encoding="utf-8") as handle:
        json.dump(record, handle)
    tmp.replace(path)


def _load_profile(profile_id: str) -> Optional[Dict[str, Any]]:
    path = _profile_path(profile_id)
    if not path.exists():
        return None
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def _list_profiles() -> List[Dict[str, Any]]:
    records: List[Dict[str, Any]] = []
    for path in sorted(PROFILE_STORE_DIR.glob("*.json")):
        try:
            with open(path, "r", encoding="utf-8") as handle:
                records.append(json.load(handle))
        except (OSError, json.JSONDecodeError):
            continue
    return records


def _clip_public(clip: Dict[str, Any]) -> Dict[str, Any]:
    """Clip metadata surfaced to clients (never the raw embedding)."""
    return {
        "clip_id": clip.get("clip_id", ""),
        "label": clip.get("label", ""),
        "voiced_seconds": clip.get("voiced_seconds", 0.0),
        "audio_seconds": clip.get("audio_seconds", 0.0),
        "self_consistency_score": clip.get("self_consistency_score", -1.0),
        "vad_model": clip.get("vad_model", ""),
        "created_at": clip.get("created_at", ""),
        "embedding_dim": len(clip.get("embedding") or []) or EMBEDDING_DIM,
    }


def _public_profile(record: Dict[str, Any]) -> Dict[str, Any]:
    """Strip raw embeddings; surface only metadata expected by the client."""
    clips = record.get("clips") or []
    return {
        "id": record.get("id", ""),
        "display_name": record.get("display_name", ""),
        "created_at": record.get("created_at", ""),
        "updated_at": record.get("updated_at", ""),
        "model_name": record.get("model_name", MODEL_NAME),
        "embedding_dim": record.get("embedding_dim", EMBEDDING_DIM),
        "sample_rate": record.get("sample_rate", SAMPLE_RATE),
        "clip_count": len(clips),
        "total_voiced_seconds": _total_voiced_seconds(clips),
        "notes": record.get("notes", ""),
    }


# ---------------------------------------------------------------------------
# Endpoints
# ---------------------------------------------------------------------------


@app.on_event("startup")
async def warm_models_on_startup() -> None:
    """Start model warmup without making the liveness listener unavailable."""
    asyncio.create_task(asyncio.to_thread(_warm_models))


@app.get("/ready")
def ready() -> JSONResponse:
    profile_store_ok = PROFILE_STORE_DIR.is_dir() and os.access(
        PROFILE_STORE_DIR, os.W_OK
    )
    temp_dir_ok = TEMP_DIR.is_dir() and os.access(TEMP_DIR, os.W_OK)
    model_loaded = _classifier is not None and _vad is not None
    payload = {
        "status": "ok" if model_loaded and profile_store_ok and temp_dir_ok else "starting",
        "model_loaded": model_loaded,
        "profile_store_ok": bool(profile_store_ok),
        "temp_dir_ok": bool(temp_dir_ok),
    }
    if _model_warm_error:
        payload["model_error"] = _model_warm_error
    return JSONResponse(status_code=200 if model_loaded and profile_store_ok and temp_dir_ok else 503, content=payload)


@app.get("/v1/info")
def info() -> Dict[str, Any]:
    return {
        "backend": "speechbrain",
        "model": MODEL_NAME,
        "device": DEVICE,
        "torch_version": torch.__version__,
        "cuda_available": _CUDA_AVAILABLE,
        "sample_rate": SAMPLE_RATE,
        "version": SERVER_VERSION,
        "embedding_dim": EMBEDDING_DIM,
        "vad": _get_vad().name,
        "vad_model": _get_vad().name,
        "score_agg": "max",
        "embed_denoise": EMBED_DENOISE,
        "min_enroll_voiced_seconds": MIN_ENROLL_VOICED_SECONDS,
        "min_verify_voiced_seconds": MIN_VERIFY_VOICED_SECONDS,
        "self_consistency_threshold": SELF_CONSISTENCY_THRESHOLD,
        "default_threshold": float(DEFAULT_VERIFY_THRESHOLD),
        "extraction_model": EXTRACTION_MODEL,
        "extraction_sample_rate": EXTRACTION_SAMPLE_RATE,
        "extraction_match_threshold": EXTRACTION_MATCH_THRESHOLD,
    }


@app.get("/v1/profiles")
def list_profiles() -> Dict[str, Any]:
    records = [_public_profile(r) for r in _list_profiles()]
    return {"profiles": records, "count": len(records)}


@app.get("/v1/profiles/{profile_id}")
def get_profile(profile_id: str) -> JSONResponse:
    record = _load_profile(profile_id.strip())
    if record is None:
        return JSONResponse(status_code=404, content={"error": "profile not found"})
    public = _public_profile(record)
    public["clips"] = [_clip_public(c) for c in record.get("clips", [])]
    return JSONResponse(status_code=200, content=public)


@app.get("/v1/profiles/{profile_id}/clips")
def list_clips(profile_id: str) -> JSONResponse:
    record = _load_profile(profile_id.strip())
    if record is None:
        return JSONResponse(status_code=404, content={"error": "profile not found"})
    clips = [_clip_public(c) for c in record.get("clips", [])]
    return JSONResponse(
        status_code=200,
        content={
            "profile_id": record.get("id", profile_id.strip()),
            "clips": clips,
            "count": len(clips),
        },
    )


@app.post("/v1/profiles")
async def enroll(
    profile_id: str = Form(""),
    display_name: str = Form(""),
    notes: str = Form(""),
    label: str = Form(""),
    audio: UploadFile = File(...),
) -> JSONResponse:
    """Append one labeled enrollment clip to a profile (creating it if new).

    The clip is embedded over its voiced span only; clips with less than
    MIN_ENROLL_VOICED_SECONDS of voiced audio are rejected (422). If the
    profile already has clips, the new clip's max-cosine against the existing
    ones is reported as ``self_consistency_score``; when below
    ``SPEAKER_SELF_CONSISTENCY_THRESHOLD`` (default 0.5), ``self_consistency_warning``
    is true. The clip is stored either way — the warning is informational.
    """
    raw = await audio.read()
    if not raw:
        return JSONResponse(status_code=400, content={"error": "empty audio upload"})

    try:
        embedding, audio_seconds, voiced_seconds = _voiced_embedding(raw)
    except Exception as exc:  # noqa: BLE001 -- surface decode/model failures
        return JSONResponse(
            status_code=400, content={"error": f"failed to embed audio: {exc}"}
        )

    if embedding is None or voiced_seconds < MIN_ENROLL_VOICED_SECONDS:
        return JSONResponse(
            status_code=422,
            content={
                "error": "insufficient voiced audio",
                "voiced_seconds": voiced_seconds,
                "audio_seconds": audio_seconds,
                "min_voiced_seconds": MIN_ENROLL_VOICED_SECONDS,
                "vad_model": _get_vad().name,
            },
        )

    pid = profile_id.strip() or uuid.uuid4().hex
    now = _now_iso()

    existing = _load_profile(pid)
    existing_clips: List[Dict[str, Any]] = []
    if existing is not None:
        if existing.get("model_name", MODEL_NAME) != MODEL_NAME:
            return JSONResponse(
                status_code=409,
                content={
                    "error": "profile model mismatch",
                    "profile_model": existing.get("model_name", ""),
                    "server_model": MODEL_NAME,
                },
            )
        existing_clips = existing.get("clips") or []

    self_score, self_label, self_clip_id = _self_consistency(embedding, existing_clips)
    self_warning = bool(
        existing_clips and self_score >= 0.0 and self_score < SELF_CONSISTENCY_THRESHOLD
    )

    clip = {
        "clip_id": uuid.uuid4().hex,
        "label": label.strip(),
        "embedding": embedding,
        "voiced_seconds": voiced_seconds,
        "audio_seconds": audio_seconds,
        "self_consistency_score": self_score,
        "vad_model": _get_vad().name,
        "created_at": now,
    }

    if existing is not None:
        record = existing
        if display_name:
            record["display_name"] = display_name
        if notes:
            record["notes"] = notes
        record["clips"].append(clip)
        record["updated_at"] = now
    else:
        record = {
            "id": pid,
            "display_name": display_name,
            "notes": notes,
            "model_name": MODEL_NAME,
            "embedding_dim": EMBEDDING_DIM,
            "sample_rate": SAMPLE_RATE,
            "clips": [clip],
            "created_at": now,
            "updated_at": now,
        }
    _save_profile(record)

    clips = record["clips"]
    return JSONResponse(
        status_code=200,
        content={
            "profile_id": pid,
            "clip_id": clip["clip_id"],
            "label": clip["label"],
            "voiced_seconds": voiced_seconds,
            "audio_seconds": audio_seconds,
            "clip_count": len(clips),
            "total_voiced_seconds": _total_voiced_seconds(clips),
            "embedding_dim": EMBEDDING_DIM,
            "sample_rate": SAMPLE_RATE,
            "model_name": MODEL_NAME,
            "vad_model": _get_vad().name,
            "self_consistency_score": self_score,
            "self_consistency_threshold": SELF_CONSISTENCY_THRESHOLD,
            "self_consistency_warning": self_warning,
            "self_consistency_best_clip_label": self_label,
            "self_consistency_best_clip_id": self_clip_id,
            "created_at": record["created_at"],
        },
    )


@app.post("/v1/verify")
async def verify(
    profile_id: str = Form(...),
    threshold: str = Form(DEFAULT_VERIFY_THRESHOLD),
    audio: UploadFile = File(...),
) -> JSONResponse:
    record = _load_profile(profile_id.strip())
    if record is None:
        return JSONResponse(status_code=404, content={"error": "profile not found"})

    try:
        thr = float(threshold)
    except (TypeError, ValueError):
        return JSONResponse(status_code=400, content={"error": "invalid threshold"})

    raw = await audio.read()
    if not raw:
        return JSONResponse(status_code=400, content={"error": "empty audio upload"})

    start = time.perf_counter()
    try:
        embedding, audio_seconds, voiced_seconds = _voiced_embedding(raw)
    except Exception as exc:  # noqa: BLE001
        return JSONResponse(
            status_code=400, content={"error": f"failed to embed audio: {exc}"}
        )

    clips = record.get("clips") or []
    n_clips = sum(1 for c in clips if c.get("embedding"))

    # Too little voiced audio to judge: report insufficiency rather than
    # fabricating a score the caller would treat as a real (low) match.
    if embedding is None or voiced_seconds < MIN_VERIFY_VOICED_SECONDS:
        duration_ms = (time.perf_counter() - start) * 1000.0
        return JSONResponse(
            status_code=200,
            content={
                "profile_id": profile_id.strip(),
                "matched": False,
                "score": 0.0,
                "threshold": thr,
                "sufficient": False,
                "voiced_seconds": voiced_seconds,
                "audio_seconds": audio_seconds,
                "min_voiced_seconds": MIN_VERIFY_VOICED_SECONDS,
                "duration_ms": duration_ms,
                "backend": "speechbrain",
                "model": MODEL_NAME,
                "score_agg": "max",
                "vad_model": _get_vad().name,
                "n_clips": n_clips,
                "best_clip_label": "",
                "best_clip_id": "",
                "best_clip_score": 0.0,
            },
        )

    score, best_label, best_id = _best_match(embedding, clips)
    if score < 0.0:
        # Profile has no clips with embeddings — surface as not-matched but
        # sufficient, so the caller sees a real diagnostic, not a fabricated 0.
        score = 0.0
    duration_ms = (time.perf_counter() - start) * 1000.0

    print(
        "[speaker-verification] verify "
        f"profile={profile_id.strip()} score={score:.4f} threshold={thr:.4f} "
        f"voiced_seconds={voiced_seconds:.3f} audio_seconds={audio_seconds:.3f} "
        f"vad={_get_vad().name} n_clips={n_clips} best_clip_id={best_id} "
        f"best_clip_label={best_label!r}",
        flush=True,
    )

    return JSONResponse(
        status_code=200,
        content={
            "profile_id": profile_id.strip(),
            "matched": bool(score >= thr),
            "score": score,
            "threshold": thr,
            "sufficient": True,
            "voiced_seconds": voiced_seconds,
            "audio_seconds": audio_seconds,
            "duration_ms": duration_ms,
            "backend": "speechbrain",
            "model": MODEL_NAME,
            "score_agg": "max",
            "vad_model": _get_vad().name,
            "n_clips": n_clips,
            "best_clip_label": best_label,
            "best_clip_id": best_id,
            "best_clip_score": score,
        },
    )


@app.post("/v1/extract")
async def extract(
    profile_id: str = Form(""),
    verify: str = Form("false"),  # noqa: ARG001 -- reserved; the body IS the cleaned audio
    audio: UploadFile = File(...),
):
    """Isolate the enrolled speaker's voice from a mixture (target-speaker
    extraction). Returns the cleaned audio as raw 16 kHz mono s16le PCM in the
    body, with X-Speaker-Score / X-Speaker-Matched / X-Duration-Ms /
    X-Audio-Seconds headers — the contract the audio-tools Go client parses.
    """
    record = _load_profile(profile_id.strip())
    if record is None:
        return JSONResponse(status_code=404, content={"error": "profile not found"})

    clips = record.get("clips") or []
    if not any(c.get("embedding") for c in clips):
        return JSONResponse(
            status_code=409, content={"error": "profile has no enrollment clips"}
        )

    raw = await audio.read()
    if not raw:
        return JSONResponse(status_code=400, content={"error": "empty audio upload"})

    start = time.perf_counter()
    try:
        waveform, audio_seconds = _decode_to_waveform(raw)
        cleaned_pcm, score = _extract_target(waveform, clips)
    except Exception as exc:  # noqa: BLE001 -- surface decode/model failures
        return JSONResponse(
            status_code=400, content={"error": f"extraction failed: {exc}"}
        )
    duration_ms = (time.perf_counter() - start) * 1000.0

    return Response(
        content=cleaned_pcm,
        media_type="application/octet-stream",
        headers={
            "X-Speaker-Score": f"{score:.6f}",
            "X-Speaker-Matched": "true" if score >= EXTRACTION_MATCH_THRESHOLD else "false",
            "X-Duration-Ms": f"{duration_ms:.3f}",
            "X-Audio-Seconds": f"{audio_seconds:.3f}",
        },
    )


@app.delete("/v1/profiles/{profile_id}/clips/{clip_id}")
def delete_clip(profile_id: str, clip_id: str) -> JSONResponse:
    record = _load_profile(profile_id.strip())
    if record is None:
        return JSONResponse(status_code=404, content={"error": "profile not found"})

    clips = record.get("clips") or []
    remaining = [c for c in clips if c.get("clip_id") != clip_id]
    if len(remaining) == len(clips):
        return JSONResponse(status_code=404, content={"error": "clip not found"})

    if not remaining:
        # Deleting the last clip removes the (now-empty) identity.
        try:
            _profile_path(profile_id.strip()).unlink()
        except OSError as exc:
            return JSONResponse(
                status_code=500, content={"error": f"failed to delete profile: {exc}"}
            )
        return JSONResponse(
            status_code=200,
            content={
                "profile_id": profile_id.strip(),
                "clip_id": clip_id,
                "deleted_profile": True,
                "clip_count": 0,
            },
        )

    record["clips"] = remaining
    record["updated_at"] = _now_iso()
    _save_profile(record)
    return JSONResponse(
        status_code=200,
        content={
            "profile_id": profile_id.strip(),
            "clip_id": clip_id,
            "deleted_profile": False,
            "clip_count": len(remaining),
            "total_voiced_seconds": _total_voiced_seconds(remaining),
        },
    )


@app.delete("/v1/profiles/{profile_id}")
def delete_profile(profile_id: str) -> JSONResponse:
    path = _profile_path(profile_id)
    if not path.exists():
        return JSONResponse(status_code=404, content={"error": "profile not found"})
    try:
        path.unlink()
    except OSError as exc:
        return JSONResponse(
            status_code=500, content={"error": f"failed to delete profile: {exc}"}
        )
    return JSONResponse(status_code=200, content={})
