"""Private, local MIR adapter used by Music Tools.

The service never downloads inputs. Music Tools passes verified blob paths and a
request-scoped output directory; this process verifies both against the caller's
owned blob root before reading or writing. Model and analysis dependencies are
loaded only from the resource's isolated runtime.
"""
from __future__ import annotations

import hashlib
import ipaddress
import json
import math
import os
from pathlib import Path
import shutil
import subprocess
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import Request, urlopen

VERSION = "0.1.0"
MODEL_TYPE = os.environ.get("MUSIC_MIR_MODEL_TYPE", "sub_roformer")
CONFIG_PATH = Path(os.environ.get("MUSIC_MIR_CONFIG_PATH", ""))
CHECKPOINT_PATH = Path(os.environ.get("MUSIC_MIR_CHECKPOINT_PATH", ""))
MODEL_REVISION = os.environ.get("MUSIC_MIR_MODEL_REVISION", "unconfigured")
HOST = os.environ.get("MUSIC_MIR_HOST", "127.0.0.1")
PORT = int(os.environ.get("MUSIC_MIR_PORT", "8897"))
MAX_REQUEST_BYTES = 65536
MAX_SOURCE_BYTES = 1 << 30
MAX_ANALYSIS_SECONDS = 7 * 60
MAX_STEM_BYTES = 128 * 1024 * 1024
STEMS = ("vocals", "drums", "bass", "other")
FFMPEG_BINARY = os.environ.get("MUSIC_MIR_FFMPEG", "")
WHISPER_URL = os.environ.get("MUSIC_MIR_WHISPER_URL", "http://127.0.0.1:8090").rstrip("/")
_model_lock = threading.Lock()
_separator: Any = None
_model_error = "MIR model is not installed or configured"
_model_device = "unavailable"
_last_model_use = 0.0
_idle_ttl_seconds = max(30, int(os.environ.get("MUSIC_MIR_IDLE_TTL_SECONDS", "900")))
_active_lock = threading.Lock()
_active_operations: dict[str, tuple[threading.Event, threading.Event]] = {}
_cancelled_requests: dict[str, float] = {}


def local_whisper_url() -> str:
    """Only permit loopback Whisper URLs so private reference audio cannot egress."""
    try:
        parsed = urlsplit(WHISPER_URL)
        host = parsed.hostname or ""
        loopback = host.lower() == "localhost" or ipaddress.ip_address(host).is_loopback
        if (parsed.scheme == "http" and loopback and not parsed.username and
                not parsed.password and not parsed.query and not parsed.fragment):
            return WHISPER_URL
    except (ValueError, TypeError):
        pass
    return ""


def whisper_readiness() -> dict[str, Any]:
    """Report local Whisper companion readiness without making it a hard dependency."""
    endpoint = local_whisper_url()
    if not endpoint:
        return {"configured": False, "ready": False, "status": "not_configured"}
    try:
        with urlopen(endpoint + "/", timeout=1) as response:
            ready = 200 <= response.status < 400
        return {"configured": True, "ready": ready, "status": "ready" if ready else "unavailable"}
    except (HTTPError, URLError, TimeoutError, OSError, ValueError):
        return {"configured": True, "ready": False, "status": "unavailable"}


def digest_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as audio:
        while chunk := audio.read(1024 * 1024):
            digest.update(chunk)
    return digest.hexdigest()


def owned_path(raw: str, allowed_root: Path, *, directory: bool) -> Path:
    candidate = Path(raw)
    if not candidate.is_absolute():
        raise ValueError("paths must be absolute")
    resolved_root = allowed_root.resolve(strict=True)
    # Reject symlinked path components, including the leaf, before resolve.
    current = Path(candidate.anchor)
    for part in candidate.parts[1:]:
        current = current / part
        if current.is_symlink():
            raise ValueError("symlinked paths are not accepted")
    resolved = candidate.resolve(strict=not directory)
    try:
        resolved.relative_to(resolved_root)
    except ValueError as exc:
        raise ValueError("path is outside the Music Tools blob root") from exc
    if directory:
        if resolved.exists() and not resolved.is_dir():
            raise ValueError("output_dir must be a directory")
        resolved.mkdir(parents=True, exist_ok=True, mode=0o700)
    elif not resolved.is_file():
        raise ValueError("input path must be a regular file")
    elif resolved.stat().st_size <= 0 or resolved.stat().st_size > MAX_SOURCE_BYTES:
        raise ValueError("input file is empty or exceeds the 1 GiB source limit")
    return resolved


def decode_source(source: Path, output_dir: Path) -> Path:
    executable = None
    if FFMPEG_BINARY:
        executable = shutil.which(FFMPEG_BINARY)
        if executable is None and Path(FFMPEG_BINARY).is_file():
            executable = FFMPEG_BINARY
    try:
        if executable is None:
            import imageio_ffmpeg
            executable = imageio_ffmpeg.get_ffmpeg_exe()
    except ImportError:
        pass
    if executable is None:
        executable = shutil.which("ffmpeg")
    if not executable:
        raise RuntimeError("no bundled or host ffmpeg executable is available for audio decoding")
    decoded = output_dir / ".music-mir-input.wav"
    try:
        subprocess.run(
            [executable, "-nostdin", "-hide_banner", "-v", "error", "-y", "-i", str(source),
             # Decode at most one second beyond the accepted limit. The size
             # check below rejects anything longer, while this prevents a
             # long compressed source from filling staging before validation.
             "-vn", "-t", str(MAX_ANALYSIS_SECONDS + 1), "-ac", "2", "-ar", "48000",
             "-c:a", "pcm_s16le", str(decoded)],
            check=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=300,
        )
        info = decoded.stat()
        max_decoded_bytes = int(MAX_ANALYSIS_SECONDS * 48000 * 2 * 2 + 65536)
        if info.st_size <= 44 or info.st_size > max_decoded_bytes:
            raise ValueError("decoded audio is empty or exceeds the 7-minute analysis limit")
        return decoded
    except Exception:
        decoded.unlink(missing_ok=True)
        raise


def initialize_model() -> None:
    global _separator, _model_error, _model_device
    if not CONFIG_PATH.is_file() or not CHECKPOINT_PATH.is_file():
        return
    try:
        import torch
        from msst import Separator

        if not torch.cuda.is_available():
            raise RuntimeError("CUDA is required; CPU fallback is disabled")
        separator = Separator(
            model_type=MODEL_TYPE,
            config_path=CONFIG_PATH,
            checkpoint_path=CHECKPOINT_PATH,
            device_ids=0,
        )
        found = {normalize_stem(name) for name in separator.instruments}
        if found != set(STEMS):
            separator.close()
            raise RuntimeError(f"configured checkpoint outputs {sorted(found)}, expected {sorted(STEMS)}")
        _separator = separator
        _model_device = str(separator.device)
        _model_error = ""
    except Exception as exc:  # service remains healthy for capabilities inspection
        _separator = None
        _model_device = "unavailable"
        _model_error = f"{type(exc).__name__}: {exc}"


def ensure_model_loaded() -> None:
    global _last_model_use
    with _model_lock:
        if _separator is None:
            initialize_model()
        if _separator is None:
            raise RuntimeError(_model_error or "MIR model is not ready")
        _last_model_use = time.monotonic()


def unload_after_idle() -> None:
    global _separator, _model_device, _model_error
    while True:
        time.sleep(min(30, max(5, _idle_ttl_seconds // 6)))
        if _separator is None or not _last_model_use or time.monotonic() - _last_model_use < _idle_ttl_seconds:
            continue
        if not _model_lock.acquire(blocking=False):
            continue
        try:
            if _separator is not None and time.monotonic() - _last_model_use >= _idle_ttl_seconds:
                _separator.close()
                _separator = None
                _model_device = "unloaded"
                _model_error = ""
                try:
                    import torch
                    torch.cuda.empty_cache()
                except Exception:
                    pass
        finally:
            _model_lock.release()


def unload_model() -> None:
    global _separator, _model_device, _model_error
    with _model_lock:
        if _separator is None:
            return
        _separator.close()
        _separator = None
        _model_device = "unloaded"
        _model_error = ""
        try:
            import torch
            torch.cuda.empty_cache()
        except Exception:
            pass


def normalize_stem(name: str) -> str:
    aliases = {"vocal": "vocals", "vocals": "vocals", "drum": "drums", "drums": "drums",
               "bass": "bass", "other": "other", "instrumental": "other"}
    return aliases.get(name.strip().lower(), name.strip().lower())


def transcribe_segments(path: Path) -> dict[str, Any]:
    """Request timestamped local Whisper output without making MIR depend on Whisper availability."""
    endpoint = local_whisper_url()
    if not endpoint:
        return {"status": "not_configured", "segments": [], "confidence": "unmeasured"}
    boundary = "----music-mir-" + hashlib.sha256(os.urandom(24)).hexdigest()
    chunks: list[bytes] = []
    for name, value in (("task", "transcribe"),):
        chunks.append(f"--{boundary}\r\nContent-Disposition: form-data; name=\"{name}\"\r\n\r\n{value}\r\n".encode())
    chunks.append((f"--{boundary}\r\nContent-Disposition: form-data; name=\"audio_file\"; filename=\"analysis.wav\"\r\n"
                   "Content-Type: audio/wav\r\n\r\n").encode())
    with path.open("rb") as source:
        while chunk := source.read(1024 * 1024):
            chunks.append(chunk)
    chunks.append(f"\r\n--{boundary}--\r\n".encode())
    request = Request(endpoint + "/asr?output=json", data=b"".join(chunks), method="POST",
                      headers={"Content-Type": f"multipart/form-data; boundary={boundary}", "Accept": "application/json"})
    try:
        with urlopen(request, timeout=600) as response:
            raw = response.read(8 * 1024 * 1024 + 1)
        if len(raw) > 8 * 1024 * 1024:
            return {"status": "invalid_response", "segments": [], "confidence": "unmeasured"}
        payload = json.loads(raw)
        if not isinstance(payload, dict) or not isinstance(payload.get("segments"), list):
            return {"status": "invalid_response", "segments": [], "confidence": "unmeasured"}
        segments = []
        for item in payload["segments"][:10000]:
            if not isinstance(item, dict) or not isinstance(item.get("text"), str):
                continue
            start, end = item.get("start"), item.get("end")
            if (isinstance(start, (int, float)) and math.isfinite(start) and
                    isinstance(end, (int, float)) and math.isfinite(end) and 0 <= start <= end <= MAX_ANALYSIS_SECONDS):
                segments.append({"start_seconds": round(float(start), 3),
                                 "end_seconds": round(float(end), 3), "text": item["text"][:4000]})
        language = payload.get("language")
        return {"status": "available", "service": "whisper", "language": language if isinstance(language, str) else None,
                "segments": segments, "confidence": "not supplied by the service"}
    except (HTTPError, URLError, TimeoutError, OSError, ValueError, json.JSONDecodeError):
        return {"status": "unavailable", "service": "whisper", "segments": [], "confidence": "unmeasured",
                "limits": ["Local Whisper did not return a usable timestamped transcript."]}


def audio_analysis(path: Path, source_digest: str, stems: dict[str, Any], stem_sample_rate: int,
                   transcription: dict[str, Any]) -> dict[str, Any]:
    """Return measured audio features while retaining low-confidence limits."""
    import librosa
    import numpy as np

    audio, sample_rate = librosa.load(path, sr=22050, mono=True)
    if audio.size == 0 or not np.isfinite(audio).all():
        raise ValueError("decoded audio is empty or contains non-finite samples")
    duration = float(audio.size / sample_rate)
    hop = 512
    rms = librosa.feature.rms(y=audio, frame_length=2048, hop_length=hop)[0]
    onset = librosa.onset.onset_strength(y=audio, sr=sample_rate, hop_length=hop)
    tempo, beat_frames = librosa.beat.beat_track(y=audio, sr=sample_rate, hop_length=hop, units="frames")
    tempo_value = float(np.asarray(tempo).reshape(-1)[0])
    beat_times = librosa.frames_to_time(beat_frames, sr=sample_rate, hop_length=hop)

    chroma = librosa.feature.chroma_cqt(y=audio, sr=sample_rate, hop_length=hop)
    profile_major = np.asarray([6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88])
    profile_minor = np.asarray([6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17])
    chroma_mean = np.mean(chroma, axis=1)
    candidates: list[tuple[float, str]] = []
    note_names = ("C", "C♯", "D", "D♯", "E", "F", "F♯", "G", "G♯", "A", "A♯", "B")
    for shift, note in enumerate(note_names):
        for mode, profile in (("major", profile_major), ("minor", profile_minor)):
            score = float(np.corrcoef(chroma_mean, np.roll(profile, shift))[0, 1])
            if math.isfinite(score):
                candidates.append((score, f"{note} {mode}"))
    candidates.sort(reverse=True)
    key_margin = candidates[0][0] - candidates[1][0] if len(candidates) > 1 else 0.0

    # One-second RMS/onset/brightness points are useful, timestamped acoustic
    # observations. They are not semantic labels or a substitute for listening.
    centroid = librosa.feature.spectral_centroid(y=audio, sr=sample_rate, hop_length=hop)[0]
    frame_times = librosa.frames_to_time(np.arange(rms.size), sr=sample_rate, hop_length=hop)
    curve: list[dict[str, float]] = []
    for second in np.arange(0.0, duration, 1.0):
        mask = (frame_times >= second) & (frame_times < second + 1.0)
        if np.any(mask):
            curve.append({"time_seconds": round(float(second), 3),
                          "rms": round(float(np.mean(rms[mask])), 6),
                          "spectral_centroid_hz": round(float(np.mean(centroid[mask])), 2)})

    # Candidate cuts come from abrupt novelty peaks and are explicitly tagged
    # low confidence. Exact musical sections/hooks need a trained segmenter and
    # operator correction; this heuristic must not claim verse/chorus truth.
    novelty = np.asarray(onset, dtype=float)
    novelty = np.abs(np.diff(novelty, prepend=novelty[:1]))
    novelty_times = librosa.frames_to_time(np.arange(novelty.size), sr=sample_rate, hop_length=hop)
    cut_candidates: list[dict[str, Any]] = []
    if novelty.size:
        threshold = float(np.mean(novelty) + 1.5 * np.std(novelty))
        last = -12.0
        for idx in np.flatnonzero(novelty >= threshold):
            at = float(novelty_times[idx])
            if at >= 5.0 and at < duration - 5.0 and at - last >= 8.0:
                cut_candidates.append({"time_seconds": round(at, 3), "confidence": "low",
                                      "signal": "onset-novelty-change"})
                last = at
    stem_energy_curve: list[dict[str, Any]] = []
    chunk_samples = sample_rate * 5
    for start in range(0, audio.size, chunk_samples):
        end = min(start + chunk_samples, audio.size)
        powers: dict[str, float] = {}
        for name, samples in stems.items():
            stem_samples = np.asarray(samples)
            if stem_samples.ndim == 2:
                stem_samples = np.mean(stem_samples, axis=0)
            left = int(start * stem_sample_rate / sample_rate)
            right = int(end * stem_sample_rate / sample_rate)
            region = stem_samples[left:right]
            powers[name] = float(np.mean(np.square(region))) if region.size else 0.0
        total = sum(powers.values()) or 1.0
        stem_energy_curve.append({"time_seconds": round(start / sample_rate, 3),
                                  "relative_power": {name: round(value / total, 4) for name, value in powers.items()}})
    return {
        "schema_version": 1,
        "status": "partial",
        "source_sha256": source_digest,
        "media": {"sample_rate_hz": sample_rate, "duration_seconds": round(duration, 3),
                  "channels_analyzed": 1, "decoder": "librosa", "model_device": _model_device},
        "beat": {"tempo_bpm_estimate": round(tempo_value, 3),
                 "beat_times_seconds": [round(float(value), 3) for value in beat_times],
                 "method": "librosa beat tracker", "confidence": "unmeasured"},
        "key": {"primary_estimate": candidates[0][1] if candidates else None,
                "alternatives": [{"key": key, "score": round(score, 4)} for score, key in candidates[:4]],
                "margin": round(key_margin, 4), "confidence": "unmeasured"},
        "energy_curve": curve,
        "stem_energy_curve": stem_energy_curve,
        "section_boundaries": cut_candidates,
        "lyric_alignment": transcription,
        "descriptions": [],
        "limits": ["Acoustic cuts are low-confidence candidates, not verified song sections.",
                   "Whisper transcript segments are hypotheses; confidence and word-level alignment are not supplied.",
                   "No semantic audio captioning or verified section labels are configured."],
    }


def ensure_request_root(request: dict[str, Any]) -> tuple[Path, Path, Path, str]:
    root_value = request.get("allowed_root")
    if not isinstance(root_value, str) or not root_value:
        raise ValueError("allowed_root is required from the Music Tools owner")
    root = Path(root_value)
    if not root.is_absolute() or not root.is_dir():
        raise ValueError("allowed_root must be an existing absolute directory")
    source = request.get("input")
    if not isinstance(source, dict) or not isinstance(source.get("path"), str):
        raise ValueError("input.path is required")
    source_path = owned_path(source["path"], root, directory=False)
    output_path = owned_path(str(request.get("output_dir", "")), root, directory=True)
    request_id = str(request.get("request_id", ""))
    if not request_id or output_path.name != request_id or output_path.parent.name != ".staging" or output_path.parent.parent.name != "analysis":
        raise ValueError("output_dir must be analysis/.staging/<request_id>")
    digest = str(source.get("sha256", ""))
    if len(digest) != 64 or digest_file(source_path) != digest.lower():
        raise ValueError("input SHA-256 does not match the source file")
    return root, source_path, output_path, digest.lower()


def separate_in_memory(path: Path, cancel_requested: threading.Event) -> tuple[dict[str, Any], int]:
    if _separator is None:
        raise RuntimeError(_model_error or "MIR separation backend is unavailable")
    global _last_model_use
    with _model_lock:
        if cancel_requested.is_set():
            raise RuntimeError("operation cancelled before model execution")
        output = _separator.separate_file(path)
        sample_rate = int(_separator.sample_rate)
        _last_model_use = time.monotonic()
    normalized: dict[str, Any] = {}
    for name, samples in output.items():
        normalized[normalize_stem(name)] = samples
    if set(normalized) != set(STEMS):
        raise RuntimeError(f"separation returned {sorted(normalized)}, expected {sorted(STEMS)}")
    return normalized, sample_rate


def separate(path: Path, output_dir: Path, cancel_requested: threading.Event) -> dict[str, Any]:
    import numpy as np
    import soundfile as sf

    normalized, sample_rate = separate_in_memory(path, cancel_requested)
    result: dict[str, Any] = {}
    for stem in STEMS:
        filename = f"{stem}.wav"
        destination = output_dir / filename
        samples = np.asarray(normalized[stem])
        if samples.ndim != 2 or not np.isfinite(samples).all():
            raise RuntimeError(f"invalid {stem} waveform returned by model")
        if samples.size * 3 + 65536 > MAX_STEM_BYTES:
            raise ValueError(f"{stem} stem would exceed Music Tools' 128 MiB owned-blob limit")
        # MSST returns (channels, samples); soundfile accepts (samples, channels).
        sf.write(destination, samples.T, sample_rate, subtype="PCM_24", format="WAV")
        result[stem] = {"file": filename, "sha256": digest_file(destination), "media_type": "audio/wav"}
    return result


class Handler(BaseHTTPRequestHandler):
    server_version = "music-mir/" + VERSION

    def log_message(self, fmt: str, *args: Any) -> None:
        # Keep local paths and source identifiers out of routine service logs.
        print("music-mir " + (fmt % args), flush=True)

    def send_json(self, status: int, body: dict[str, Any]) -> None:
        encoded = json.dumps(body, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def do_GET(self) -> None:  # noqa: N802 - stdlib HTTP handler contract
        if self.path == "/health":
            model_configured = CONFIG_PATH.is_file() and CHECKPOINT_PATH.is_file()
            self.send_json(200 if model_configured else 503,
                           {"status": "ok" if model_configured else "not_ready",
                                 "resource_version": VERSION,
                                 "model_configured": model_configured,
                                 "model_loaded": _separator is not None, "model": MODEL_TYPE,
                                 "model_revision": MODEL_REVISION, "device": _model_device,
                                 "model_error": _model_error or None,
                                 "whisper": whisper_readiness()})
        elif self.path == "/v1/capabilities":
            model_configured = CONFIG_PATH.is_file() and CHECKPOINT_PATH.is_file()
            whisper = whisper_readiness()
            self.send_json(200, {"resource_version": VERSION,
                                 "capabilities": {"analyze": {"available": model_configured,
                                                                "features": ["source-separation", "tempo", "beat-times", "key-candidates", "energy-curve", "acoustic-cut-candidates"]},
                                                  "separate": {"available": model_configured,
                                                                  "stems": list(STEMS)},
                                                  "lyric-transcription": {**whisper,
                                                                            "service_url": local_whisper_url() or None}},
                                 "model": MODEL_TYPE, "model_revision": MODEL_REVISION,
                                 "device": _model_device, "limitations": ["No semantic section labels or audio captions yet.",
                                                                            "Whisper returns timestamped segments, not word-level alignment or confidence."]})
        else:
            self.send_json(404, {"error": "not found"})

    def do_POST(self) -> None:  # noqa: N802 - stdlib HTTP handler contract
        if self.path == "/v1/cancel":
            self.cancel_operation()
            return
        if self.path not in ("/v1/analyze", "/v1/separate"):
            self.send_json(404, {"error": "not found"})
            return
        length = int(self.headers.get("Content-Length", "0"))
        if length < 1 or length > MAX_REQUEST_BYTES:
            self.send_json(413, {"error": "request body must be between 1 byte and 64 KiB"})
            return
        try:
            request = json.loads(self.rfile.read(length))
            if not isinstance(request, dict):
                raise ValueError("request body must be a JSON object")
            _, source_path, output_dir, digest = ensure_request_root(request)
            if any(output_dir.iterdir()):
                raise ValueError("request-scoped output directory must be empty")
            request_id = str(request["request_id"])
            drained = threading.Event()
            cancel_requested = threading.Event()
            with _active_lock:
                expired = [key for key, at in _cancelled_requests.items() if time.monotonic() - at > 86400]
                for key in expired:
                    _cancelled_requests.pop(key, None)
                if request_id in _cancelled_requests:
                    raise ValueError("request_id was already cancelled")
                if request_id in _active_operations:
                    raise ValueError("request_id is already active")
                _active_operations[request_id] = (drained, cancel_requested)
            started = time.monotonic()
            response_payload: dict[str, Any] | None = None
            try:
                decoded_path = decode_source(source_path, output_dir)
                ensure_model_loaded()
                if cancel_requested.is_set():
                    raise RuntimeError("operation cancelled before model execution")
                if self.path == "/v1/separate":
                    stems = separate(decoded_path, output_dir, cancel_requested)
                    response_payload = {"resource_version": VERSION, "model": MODEL_TYPE,
                                        "backend_revision": MODEL_REVISION, "stems": stems,
                                        "timing": {"elapsed_seconds": round(time.monotonic() - started, 3)},
                                        "confidence": {"source_sha256_verified": True}}
                else:
                    separated, stem_sample_rate = separate_in_memory(decoded_path, cancel_requested)
                    transcription = transcribe_segments(decoded_path)
                    analysis = audio_analysis(decoded_path, digest, separated, stem_sample_rate, transcription)
                    response_payload = {"resource_version": VERSION, "model": MODEL_TYPE,
                                        "backend_revision": MODEL_REVISION, "analysis": analysis,
                                        "timing": {"elapsed_seconds": round(time.monotonic() - started, 3)},
                                        "confidence": {"key_margin": analysis["key"]["margin"],
                                                       "cut_candidates": "low"},
                                        "alternatives": {"key": analysis["key"]["alternatives"]}}
            finally:
                try:
                    (output_dir / ".music-mir-input.wav").unlink(missing_ok=True)
                finally:
                    try:
                        # The Music Tools runner owns the active capacity lease.
                        # Free resident VRAM before its HTTP call returns and that
                        # lease can be released; an idle MIR model must not become
                        # invisible to the broker between requests.
                        unload_model()
                    finally:
                        with _active_lock:
                            _active_operations.pop(request_id, None)
                            drained.set()
            if response_payload is None:
                self.send_json(503, {"error": "MIR operation produced no response"})
            else:
                self.send_json(200, response_payload)
        except (ValueError, json.JSONDecodeError) as exc:
            self.send_json(400, {"error": str(exc)})
        except Exception as exc:
            self.send_json(503, {"error": f"{type(exc).__name__}: {exc}"})

    def cancel_operation(self) -> None:
        length = int(self.headers.get("Content-Length", "0"))
        if length < 1 or length > 4096:
            self.send_json(413, {"error": "cancel request body must be between 1 byte and 4 KiB"})
            return
        try:
            request = json.loads(self.rfile.read(length))
            if not isinstance(request, dict):
                raise ValueError("cancel request body must be a JSON object")
            request_id = str(request.get("request_id", ""))
            if not request_id:
                raise ValueError("request_id is required")
            with _active_lock:
                active = _active_operations.get(request_id)
                if active is None:
                    _cancelled_requests[request_id] = time.monotonic()
            if active is None:
                self.send_json(200, {"request_id": request_id, "drained": True, "already_finished": True})
                return
            # MSST's public file API does not expose a forward-pass interrupt.
            # Acknowledge cancellation only after its current inference drains;
            # Music Tools must retain the single GPU lease while awaiting this.
            drained, cancel_requested = active
            cancel_requested.set()
            if not drained.wait(timeout=1800):
                self.send_json(504, {"request_id": request_id, "drained": False,
                                     "error": "inference did not drain within 30 minutes"})
                return
            self.send_json(200, {"request_id": request_id, "drained": True})
        except (ValueError, json.JSONDecodeError, AttributeError) as exc:
            self.send_json(400, {"error": str(exc)})


if __name__ == "__main__":
    threading.Thread(target=unload_after_idle, daemon=True, name="music-mir-idle-unload").start()
    server = ThreadingHTTPServer((HOST, PORT), Handler)
    server.daemon_threads = True
    print(json.dumps({"resource": "music-mir", "version": VERSION, "host": HOST,
                      "port": PORT, "model_loaded": _separator is not None,
                      "device": _model_device, "model_error": _model_error or None}), flush=True)
    server.serve_forever()
