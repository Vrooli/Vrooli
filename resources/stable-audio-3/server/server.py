"""Local Medium service: one generation, offline inputs, atomic audio/receipt bundles."""
from __future__ import annotations

import asyncio
import ctypes
import hashlib
import json
import math
import os
from pathlib import Path
import resource
import shutil
import tempfile
import threading
import time
import uuid
from typing import Literal

from fastapi import FastAPI, HTTPException, Request, Response
from fastapi.responses import FileResponse
from pydantic import BaseModel, ConfigDict, Field, ValidationError

from loader import MODEL_ID, MODEL_REVISION, canonical, digest_file, load_local

ROOT = Path(os.environ.get("SA3_DATA_DIR", ".")).resolve()
MODEL_DIR = Path(os.environ.get("SA3_MODEL_DIR", ROOT / "models/medium"))
OUTPUT_DIR = Path(os.environ.get("SA3_OUTPUT_DIR", ROOT / "outputs"))
app = FastAPI(title="stable-audio-3", version="0.1.0")
_loaded = None
_init_error = ""
_generation = threading.Lock()
_state_lock = threading.Lock()
_active = None
MAX_BODY = 65536


class ComposeRequest(BaseModel):
    model_config = ConfigDict(extra="forbid", allow_inf_nan=False, strict=True)
    prompt: str = Field(min_length=1, max_length=12000)
    duration: float = Field(default=48, ge=1, le=120)
    seed: int = Field(strict=True, ge=0, le=2**32 - 1)
    steps: int = Field(default=8, strict=True, ge=1, le=32)
    cfg_scale: float = Field(default=1.0, ge=0, le=20)
    timeout_seconds: int = Field(default=300, strict=True, ge=1, le=300)
    expected_model_revision: Literal[MODEL_REVISION] = MODEL_REVISION


class GenerationCancelled(Exception):
    pass


@app.on_event("startup")
def initialize() -> None:
    global _loaded, _init_error
    ROOT.mkdir(parents=True, exist_ok=True)
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    os.chdir(ROOT)
    try:
        _loaded = load_local(MODEL_DIR)
        _init_error = ""
    except Exception as exc:
        _loaded = None
        _init_error = f"{type(exc).__name__}: {exc}"


@app.get("/health")
def health(response: Response) -> dict:
    ready = _loaded is not None and not _init_error
    response.status_code = 200 if ready else 503
    return {"status": "ok" if ready else "error" if _init_error else "loading", "model_loaded": ready,
            "model_id": MODEL_ID if ready else None, "model_revision": MODEL_REVISION if ready else None,
            "device": "cuda" if ready else None, "active_operation": _active,
            "initialization": _loaded["initialization"] if ready else None, "error": _init_error}


def check_operation(cancel: threading.Event, deadline: float) -> None:
    if cancel.is_set():
        raise GenerationCancelled("caller disconnected/cancelled; generation drained before publication")
    if time.monotonic() >= deadline:
        raise GenerationCancelled("generation deadline reached; no final bundle published")


def publish_directory(stage: Path, final: Path) -> None:
    """Linux-only resource: atomic publication must never replace an existing directory."""
    libc = ctypes.CDLL(None, use_errno=True)
    rename = libc.renameat2
    rename.argtypes = [ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint]
    rename.restype = ctypes.c_int
    if rename(-100, os.fsencode(stage), -100, os.fsencode(final), 1) != 0:
        code = ctypes.get_errno()
        raise OSError(code, os.strerror(code), str(final))


def sampling_shift(model) -> dict:
    shift = model.model.sampling_dist_shift
    if type(shift).__module__ != "stable_audio_3.inference.distribution_shift" or type(shift).__name__ not in {
        "IdentityDistributionShift", "FluxDistributionShift", "DistributionShift", "LogSNRShift"
    }:
        raise RuntimeError("unsupported loaded sampling shift")
    parameters = vars(shift)
    if len(parameters) > 16 or any(not isinstance(value, (int, float, bool)) or not math.isfinite(value) for value in parameters.values()):
        raise RuntimeError("loaded sampling shift cannot be represented by bounded numeric parameters")
    return {"class": type(shift).__module__ + "." + type(shift).__name__, "parameters": dict(parameters)}


def run_generation(request: ComposeRequest, raw: bytes, operation: str, cancel: threading.Event, deadline: float) -> dict:
    global _active
    stage = None
    started = time.monotonic()
    check = lambda *args, **kwargs: check_operation(cancel, deadline)
    try:
        check()
        import torch
        import numpy as np
        import soundfile as sf

        state = _loaded
        if state is None:
            raise RuntimeError("model is not loaded")
        torch.cuda.synchronize()
        torch.cuda.reset_peak_memory_stats()
        config = state["config"]
        conditioner = state["model"].model.conditioner.conditioners[state["text_conditioner_id"]]
        tokens = conditioner.tokenizer(request.prompt, truncation=False, add_special_tokens=True)["input_ids"]
        token_count = len(tokens)
        if token_count > conditioner.max_length:
            raise ValueError(f"prompt has {token_count} tokens; configured limit is {conditioner.max_length}; truncation refused")
        check()
        shift = sampling_shift(state["model"])
        inference_started = time.monotonic()
        generated = state["model"].generate(
            prompt=request.prompt, duration=request.duration, steps=request.steps, cfg_scale=request.cfg_scale,
            seed=request.seed, batch_size=1, sample_size=config["sample_size"],
            truncate_output_to_duration=True, duration_padding_sec=6.0, apg_scale=1.0,
            dist_shift=state["model"].model.sampling_dist_shift, chunked_decode=True, callback=check,
        )
        torch.cuda.synchronize()
        inference_seconds = time.monotonic() - inference_started
        check()
        if torch.initial_seed() != request.seed:
            raise RuntimeError("upstream effective Torch seed differs from request")
        audio = generated.detach().cpu().numpy()
        frames = int(request.duration * 44100)
        if tuple(audio.shape) != (1, 2, frames) or not np.isfinite(audio).all():
            raise RuntimeError("upstream output shape/finite-audio contract failed")
        stage = Path(tempfile.mkdtemp(prefix=".pending-", dir=OUTPUT_DIR))
        audio_path = stage / "audio.wav"
        sf.write(audio_path, audio[0].T, 44100, subtype="PCM_24", format="WAV")
        audio_sha, audio_bytes = digest_file(audio_path, check)
        details = sf.info(audio_path)
        if (details.frames, details.channels, details.samplerate, details.subtype) != (frames, 2, 44100, "PCM_24"):
            raise RuntimeError("encoded WAV contract failed")
        objective = state["model"].model.diffusion_objective
        receipt = {"schema_version": 1, "status": "completed", "operation_id": operation,
                   "request_sha256": hashlib.sha256(raw).hexdigest(), "effective_request": request.model_dump(),
                   "model": state["identity"], "runtime": state["runtime"],
                   "effective_config_sha256": state["effective_config_sha256"], "initialization": state["initialization"],
                   "sampling": {"seed": torch.initial_seed(), "steps": request.steps, "cfg_scale": request.cfg_scale,
                                "diffusion_objective": objective, "sampler": "pingpong" if objective == "rf_denoiser" else "euler",
                                "sampling_distribution_shift_options": config["model"]["diffusion"].get("sampling_distribution_shift_options"),
                                "loaded_sampling_shift": shift,
                                "duration_padding_sec": 6.0, "apg_scale": 1.0, "chunked_decode": True,
                                "prompt_tokens": token_count, "prompt_token_limit": conditioner.max_length, "prompt_truncated": False},
                   "output": {"name": "audio.wav", "sha256": audio_sha, "bytes": audio_bytes, "sample_rate": 44100,
                              "channels": 2, "frames": frames, "duration_seconds": frames / 44100, "format": "WAV", "subtype": "PCM_24"},
                   "measurements": {"generation_seconds": time.monotonic() - started,
                                    "upstream_generate_seconds": inference_seconds,
                                    "cuda_peak_allocated_bytes": torch.cuda.max_memory_allocated(),
                                    "cuda_peak_reserved_bytes": torch.cuda.max_memory_reserved(),
                                    "process_lifetime_maxrss_bytes": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * 1024,
                                    "whole_scope_peak_ram_bytes": None}, "audition": "unreviewed"}
        (stage / "generation-receipt.json").write_bytes(canonical(receipt) + b"\n")
        check()
        final = OUTPUT_DIR / operation
        if final.exists():
            raise RuntimeError("operation output already exists")
        publish_directory(stage, final)
        stage = None
        return receipt
    finally:
        try:
            if stage is not None:
                shutil.rmtree(stage)
        finally:
            with _state_lock:
                _active = None
            _generation.release()


@app.post("/v1/compose")
async def compose(http: Request) -> dict:
    global _active
    raw = bytearray()
    async for chunk in http.stream():
        if len(raw) + len(chunk) > MAX_BODY:
            raise HTTPException(413, "request exceeds 64 KiB")
        raw.extend(chunk)
    try:
        request = ComposeRequest.model_validate_json(bytes(raw))
        if not request.prompt.strip():
            raise ValueError("prompt must contain text")
    except (ValidationError, ValueError) as exc:
        raise HTTPException(400, str(exc)) from exc
    if _loaded is None:
        raise HTTPException(503, "model not ready; inspect /health")
    if not _generation.acquire(blocking=False):
        raise HTTPException(503, "generation busy or draining; no request queued")
    operation = uuid.uuid4().hex
    cancel = threading.Event()
    with _state_lock:
        _active = {"operation_id": operation, "state": "generating"}
    task = asyncio.create_task(asyncio.to_thread(run_generation, request, bytes(raw), operation, cancel,
                                               time.monotonic() + request.timeout_seconds))
    # Retrieve abandoned worker exceptions without weakening their lock/finally owner.
    task.add_done_callback(lambda done: done.exception() if not done.cancelled() else None)
    try:
        while not task.done():
            await asyncio.wait({task}, timeout=0.1)
            if not task.done() and await http.is_disconnected():
                cancel.set()
                with _state_lock:
                    if _active is not None and _active["operation_id"] == operation:
                        _active = {"operation_id": operation, "state": "draining"}
                raise HTTPException(499, "caller disconnected; generation draining; no export promised")
        return task.result()
    except asyncio.CancelledError:
        cancel.set()
        with _state_lock:
            if _active is not None and _active["operation_id"] == operation:
                _active = {"operation_id": operation, "state": "draining"}
        raise
    except GenerationCancelled as exc:
        raise HTTPException(408, str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(400, str(exc)) from exc
    except HTTPException:
        raise
    except Exception as exc:
        raise HTTPException(500, f"generation failed: {type(exc).__name__}: {exc}") from exc


@app.get("/v1/outputs/{operation}/audio")
def output(operation: str) -> FileResponse:
    if len(operation) != 32 or any(char not in "0123456789abcdef" for char in operation):
        raise HTTPException(400, "invalid operation identity")
    directory = OUTPUT_DIR / operation
    if not (directory / "generation-receipt.json").is_file() or not (directory / "audio.wav").is_file():
        raise HTTPException(404, "completed output unavailable")
    return FileResponse(directory / "audio.wav", media_type="audio/wav", filename="audio.wav")
