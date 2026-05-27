"""SpeechBrain ECAPA-TDNN speaker-verification HTTP server.

Exposes enrollment + verification + (reserved) target-speaker extraction over a
small FastAPI surface. The contract here is consumed byte-for-byte by the
audio-tools Go client (scenarios/audio-tools/api/internal/stt/pipeline/
speaker_client.go) -- endpoint paths, multipart field names, and response JSON
keys MUST NOT drift.

Embeddings are 192-dimensional ECAPA-TDNN vectors computed from 16 kHz mono
audio. Verification is the cosine similarity between an enrolled embedding and a
test-clip embedding compared against a caller-supplied threshold.
"""

from __future__ import annotations

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

# ---------------------------------------------------------------------------
# Configuration (all overridable via environment for compose / GPU overlays)
# ---------------------------------------------------------------------------

SERVER_VERSION = "0.3.0"
MODEL_NAME = os.environ.get(
    "SPEAKER_VERIFICATION_MODEL", "speechbrain/spkrec-ecapa-voxceleb"
)
EMBEDDING_DIM = 192
SAMPLE_RATE = 16000

# Target-speaker extraction = source separation (split the mixture into N voices)
# + ECAPA target-selection (pick the separated source whose embedding best
# matches the enrolled profile). The published SepFormer separation checkpoints
# (wsj02mix / libri2mix, 2 speakers) run at 8 kHz; the 16 kHz checkpoints are
# enhancement (single-source denoise), not separation. So the default is the
# 8 kHz 2-speaker model and audio is resampled 16k↔8k around it. Which variant +
# match threshold perform best on real two-speaker audio is an empirical spike
# (GPU) — see docs/extraction.md — hence model, rate, and threshold are tunable.
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

# Lazily-initialized model handles. Loaded on first use so /ready can report
# liveness even before the (one-time) model download completes.
_classifier = None
_separator = None


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


def _embed(raw: bytes) -> Tuple[List[float], float]:
    """Compute an L2-normalized 192-dim embedding from audio bytes.

    Returns (embedding, audio_seconds).
    """
    waveform, audio_seconds = _decode_to_waveform(raw)
    return _embed_waveform(waveform), audio_seconds


def _cosine(a: List[float], b: List[float]) -> float:
    ta = torch.tensor(a, dtype=torch.float32)
    tb = torch.tensor(b, dtype=torch.float32)
    return float(torch.nn.functional.cosine_similarity(ta, tb, dim=0).item())


def _extract_target(
    waveform: "torch.Tensor", profile_embedding: List[float]
) -> Tuple[bytes, float]:
    """Isolate the enrolled speaker from a 16 kHz mono mixture.

    Separates the mixture into candidate sources, embeds each with ECAPA, and
    returns (pcm16, score) for the source whose embedding is the closest cosine
    match to the enrolled profile. The returned PCM is 16 kHz mono s16le,
    matching the input contract so it can re-enter the STT pipeline directly.
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
        score = _cosine(profile_embedding, _embed_waveform(src))
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
# ---------------------------------------------------------------------------


def _profile_path(profile_id: str) -> Path:
    return PROFILE_STORE_DIR / f"{profile_id}.json"


def _save_profile(record: Dict[str, Any]) -> None:
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


def _public_profile(record: Dict[str, Any]) -> Dict[str, Any]:
    """Strip the raw embedding; surface only metadata expected by the client."""
    return {
        "id": record.get("id", ""),
        "display_name": record.get("display_name", ""),
        "created_at": record.get("created_at", ""),
        "updated_at": record.get("updated_at", ""),
        "model_name": record.get("model_name", MODEL_NAME),
        "embedding_dim": record.get("embedding_dim", EMBEDDING_DIM),
        "sample_rate": record.get("sample_rate", SAMPLE_RATE),
        "enrollment_audio_seconds": record.get("enrollment_audio_seconds", 0.0),
        "notes": record.get("notes", ""),
    }


# ---------------------------------------------------------------------------
# Endpoints
# ---------------------------------------------------------------------------


@app.get("/ready")
def ready() -> Dict[str, Any]:
    profile_store_ok = PROFILE_STORE_DIR.is_dir() and os.access(
        PROFILE_STORE_DIR, os.W_OK
    )
    temp_dir_ok = TEMP_DIR.is_dir() and os.access(TEMP_DIR, os.W_OK)
    return {
        "status": "ok",
        "model_loaded": _classifier is not None,
        "profile_store_ok": bool(profile_store_ok),
        "temp_dir_ok": bool(temp_dir_ok),
    }


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
        "extraction_model": EXTRACTION_MODEL,
        "extraction_sample_rate": EXTRACTION_SAMPLE_RATE,
        "extraction_match_threshold": EXTRACTION_MATCH_THRESHOLD,
    }


@app.get("/v1/profiles")
def list_profiles() -> Dict[str, Any]:
    records = [_public_profile(r) for r in _list_profiles()]
    return {"profiles": records, "count": len(records)}


@app.post("/v1/profiles")
async def enroll(
    profile_id: str = Form(""),
    display_name: str = Form(""),
    notes: str = Form(""),
    audio: UploadFile = File(...),
) -> JSONResponse:
    raw = await audio.read()
    if not raw:
        return JSONResponse(
            status_code=400, content={"error": "empty audio upload"}
        )

    try:
        embedding, audio_seconds = _embed(raw)
    except Exception as exc:  # noqa: BLE001 -- surface decode/model failures
        return JSONResponse(
            status_code=400, content={"error": f"failed to embed audio: {exc}"}
        )

    pid = profile_id.strip() or uuid.uuid4().hex
    created_at = _now_iso()
    record = {
        "id": pid,
        "display_name": display_name,
        "notes": notes,
        "embedding": embedding,
        "embedding_dim": EMBEDDING_DIM,
        "sample_rate": SAMPLE_RATE,
        "model_name": MODEL_NAME,
        "enrollment_audio_seconds": audio_seconds,
        "created_at": created_at,
        "updated_at": created_at,
    }
    # Preserve original created_at when re-enrolling an existing profile id.
    existing = _load_profile(pid)
    if existing is not None:
        record["created_at"] = existing.get("created_at", created_at)
    _save_profile(record)

    return JSONResponse(
        status_code=200,
        content={
            "profile_id": pid,
            "display_name": display_name,
            "embedding_dim": EMBEDDING_DIM,
            "sample_rate": SAMPLE_RATE,
            "enrollment_audio_seconds": audio_seconds,
            "model_name": MODEL_NAME,
            "created_at": record["created_at"],
        },
    )


@app.post("/v1/verify")
async def verify(
    profile_id: str = Form(...),
    threshold: str = Form("0.25"),
    audio: UploadFile = File(...),
) -> JSONResponse:
    record = _load_profile(profile_id.strip())
    if record is None:
        return JSONResponse(
            status_code=404, content={"error": "profile not found"}
        )

    try:
        thr = float(threshold)
    except (TypeError, ValueError):
        return JSONResponse(
            status_code=400, content={"error": "invalid threshold"}
        )

    raw = await audio.read()
    if not raw:
        return JSONResponse(
            status_code=400, content={"error": "empty audio upload"}
        )

    start = time.perf_counter()
    try:
        embedding, audio_seconds = _embed(raw)
    except Exception as exc:  # noqa: BLE001
        return JSONResponse(
            status_code=400, content={"error": f"failed to embed audio: {exc}"}
        )
    score = _cosine(record["embedding"], embedding)
    duration_ms = (time.perf_counter() - start) * 1000.0

    return JSONResponse(
        status_code=200,
        content={
            "profile_id": profile_id.strip(),
            "matched": bool(score >= thr),
            "score": score,
            "threshold": thr,
            "duration_ms": duration_ms,
            "backend": "speechbrain",
            "model": MODEL_NAME,
            "audio_seconds": audio_seconds,
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

    raw = await audio.read()
    if not raw:
        return JSONResponse(status_code=400, content={"error": "empty audio upload"})

    start = time.perf_counter()
    try:
        waveform, audio_seconds = _decode_to_waveform(raw)
        cleaned_pcm, score = _extract_target(waveform, record["embedding"])
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
