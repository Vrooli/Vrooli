"""Headless, managed ACE-Step 1.5 service.

The resource owns model loading and one-take generation. Batching, provenance,
and pool state belong to music-tools. The server deliberately disables ACE's
planner rewrites unless a caller explicitly opts into them.
"""
from __future__ import annotations

import hashlib
import json
import math
import os
import re
import secrets
import threading
import time
from pathlib import Path
from typing import Literal

from fastapi import FastAPI, HTTPException, Response
from fastapi.responses import FileResponse
from pydantic import BaseModel, Field

ROOT = Path(os.environ.get("ACE_STEP_DATA_DIR", "."))
ROOT.mkdir(parents=True, exist_ok=True)
# ACE-Step and some of its dependencies use relative cache paths. Keep those
# runtime writes in the governed data directory instead of mutating the staged
# immutable artifact tree.
os.chdir(ROOT)
CHECKPOINT_DIR = Path(os.environ.get("ACE_STEP_CHECKPOINT_DIR", ROOT / "checkpoints"))
VARIANT = os.environ.get("ACE_STEP_VARIANT", "acestep-v15-turbo")
PLANNER = os.environ.get("ACE_STEP_PLANNER", "acestep-5Hz-lm-0.6B")
DEVICE = os.environ.get("ACE_STEP_DEVICE", "cuda")
OFFLOAD_DIT = os.environ.get("ACE_STEP_OFFLOAD_DIT", "0").lower() in {"1", "true", "yes", "on"}
OUTPUT_DIR = Path(os.environ.get("ACE_STEP_OUTPUT_DIR", ROOT / "outputs"))
RUNG_FILE = ROOT / "capacity-rung"
if RUNG_FILE.is_file():
    OFFLOAD_DIT = RUNG_FILE.read_text(encoding="utf-8").strip() == "offload-dit"

app = FastAPI(title="ace-step", version="1.0.0")
_lock = threading.Lock()
_generation_lock = threading.Lock()
_dit = None
_llm = None
_device_keepalive = None
_init_error = ""
_init_seconds = 0.0


@app.on_event("startup")
def eager_load() -> None:
    """Make managed-service readiness include model readiness, not just TCP."""
    try:
        _ensure_loaded()
    except Exception:
        # /health retains the structured initialization error for the control
        # plane; the process stays up so operators can inspect the failure.
        pass


class ComposeRequest(BaseModel):
    caption: str = Field(min_length=1, max_length=12000)
    lyrics: str = "[instrumental]"
    duration: float = Field(default=45.0, gt=0, le=240)
    bpm: float | None = Field(default=None, gt=0, le=400)
    keyscale: str = ""
    seed: int = Field(default_factory=lambda: secrets.randbits(32), strict=True, ge=0, le=2**32-1)
    inference_steps: int | None = Field(default=None, ge=1, le=100)
    guidance_scale: float | None = Field(default=None, ge=0, le=20)
    shift: float | None = Field(default=None, ge=1, le=5, allow_inf_nan=False)
    expected_variant: Literal["acestep-v15-turbo", "acestep-v15-sft"] | None = None
    rewrite_caption: bool = False
    offload_dit_to_cpu: bool | None = None
    expected_rung: str | None = None
    admitted_rungs: list[Literal["full", "offload-dit"]] | None = Field(default=None, min_length=1, max_length=2)


_loaded_identity: tuple[str, str] | None = None
_loaded_planner: str | None = None


def _ensure_loaded() -> None:
    global _dit, _llm, _device_keepalive, _init_error, _init_seconds, _loaded_identity, _loaded_planner
    if _dit is not None and _llm is not None:
        return
    with _lock:
        if _dit is not None and _llm is not None:
            return
        started = time.monotonic()
        try:
            # The upstream VAE config is 425 bytes, below binaryfetch's
            # minimum remote-artifact floor. Keep its exact pinned contents in
            # the reviewed application boundary and materialize it into the
            # governed data directory before offline model loading.
            vae_dir = CHECKPOINT_DIR / "vae"
            vae_dir.mkdir(parents=True, exist_ok=True)
            vae_config = {
                "_class_name": "AutoencoderOobleck",
                "_diffusers_version": "0.34.0",
                "_name_or_path": "/root/data/repo/gongjunmin/ACE-Step-1.5/checkpoints/vae/",
                "audio_channels": 2,
                "channel_multiples": [1, 2, 4, 8, 16],
                "decoder_channels": 128,
                "decoder_input_channels": 64,
                "downsampling_ratios": [2, 4, 4, 6, 10],
                "encoder_hidden_size": 128,
                "sampling_rate": 48000,
            }
            (vae_dir / "config.json").write_text(json.dumps(vae_config, indent=2) + "\n", encoding="utf-8")
            text_dir = CHECKPOINT_DIR / "Qwen3-Embedding-0.6B"
            text_dir.mkdir(parents=True, exist_ok=True)
            (text_dir / "added_tokens.json").write_text(json.dumps({
                "</think>": 151668, "</tool_call>": 151658, "</tool_response>": 151666,
                "<think>": 151667, "<tool_call>": 151657, "<tool_response>": 151665,
                "<|box_end|>": 151649, "<|box_start|>": 151648, "<|endoftext|>": 151643,
                "<|file_sep|>": 151664, "<|fim_middle|>": 151660, "<|fim_pad|>": 151662,
                "<|fim_prefix|>": 151659, "<|fim_suffix|>": 151661, "<|im_end|>": 151645,
                "<|im_start|>": 151644, "<|image_pad|>": 151655, "<|object_ref_end|>": 151647,
                "<|object_ref_start|>": 151646, "<|quad_end|>": 151651, "<|quad_start|>": 151650,
                "<|repo_name|>": 151663, "<|video_pad|>": 151656, "<|vision_end|>": 151653,
                "<|vision_pad|>": 151654, "<|vision_start|>": 151652,
            }, indent=2) + "\n", encoding="utf-8")
            (text_dir / "special_tokens_map.json").write_text(json.dumps({
                "additional_special_tokens": ["<|im_start|>", "<|im_end|>", "<|object_ref_start|>", "<|object_ref_end|>", "<|box_start|>", "<|box_end|>", "<|quad_start|>", "<|quad_end|>", "<|vision_start|>", "<|vision_end|>", "<|vision_pad|>", "<|image_pad|>", "<|video_pad|>"],
                "eos_token": {"content": "<|im_end|>", "lstrip": False, "normalized": False, "rstrip": False, "single_word": False},
                "pad_token": {"content": "<|endoftext|>", "lstrip": False, "normalized": False, "rstrip": False, "single_word": False},
            }, indent=2) + "\n", encoding="utf-8")
            from acestep.handler import AceStepHandler
            from acestep.llm_inference import LLMHandler

            load_offload_dit = OFFLOAD_DIT
            load_planner = PLANNER
            load_variant = VARIANT
            dit = AceStepHandler()
            message, ok = dit.initialize_service(
                project_root=str(ROOT),
                config_path=load_variant,
                device=DEVICE,
                offload_to_cpu=True,
                offload_dit_to_cpu=load_offload_dit,
            )
            if not ok:
                raise RuntimeError(f"DiT initialization failed: {message}")
            llm = LLMHandler()
            message, ok = llm.initialize(
                checkpoint_dir=str(CHECKPOINT_DIR),
                lm_model_path=load_planner,
                backend="pt",
                device=DEVICE,
                offload_to_cpu=True,
            )
            if not ok:
                raise RuntimeError(f"planner initialization failed: {message}")
            keepalive = None
            if DEVICE == "cuda":
                import torch
                keepalive = torch.empty((1,), device="cuda")
            # Publish immutable loaded identity only after all initialization
            # succeeds; failed keepalive must not make a retry look loaded.
            _loaded_identity = (f"ACE-Step/Ace-Step1.5:{load_variant}", "offload-dit" if load_offload_dit else "full")
            _loaded_planner = load_planner
            _dit, _llm, _device_keepalive = dit, llm, keepalive
            _init_error = ""
        except Exception as exc:  # noqa: BLE001 - surfaced through health
            _init_error = f"{type(exc).__name__}: {exc}"
            raise
        finally:
            _init_seconds = time.monotonic() - started


def _profile_name(variant: str, planner: str | None) -> str:
    return {
        ("acestep-v15-turbo", "acestep-5Hz-lm-0.6B"): "fast",
        ("acestep-v15-turbo", "acestep-5Hz-lm-1.7B"): "balanced",
        ("acestep-v15-sft", "acestep-5Hz-lm-1.7B"): "detail",
    }.get((variant, planner), "custom")


def _sampling_headers(audio: dict, seed: int, steps: int, guidance: float, shift: float, planner: str) -> dict[str, str]:
    actual = audio.get("params")
    if not isinstance(actual, dict):
        raise RuntimeError("upstream resolved sampling parameters are unavailable")
    resolved_seed = actual.get("seed")
    if type(resolved_seed) is not int or not 0 <= resolved_seed <= 2**32-1 or resolved_seed != seed:
        raise RuntimeError("upstream resolved seed differs from requested uint32 seed")
    for field, expected in [("inference_steps", steps), ("guidance_scale", guidance), ("shift", shift)]:
        value = actual.get(field)
        numeric = not isinstance(value, bool) and isinstance(value, (int, float))
        exact_steps = field != "inference_steps" or type(value) is int
        if not numeric or not exact_steps or not math.isfinite(value) or value != expected:
            raise RuntimeError(f"upstream resolved {field} differs from requested effective profile")
    if planner not in {"acestep-5Hz-lm-0.6B", "acestep-5Hz-lm-1.7B"}:
        raise RuntimeError("loaded planner identity is unavailable or unsupported")
    headers = {
        "X-ACE-Step-Seed": str(resolved_seed),
        "X-ACE-Step-Planner": planner,
        "X-ACE-Step-Inference-Steps": str(steps),
        "X-ACE-Step-Guidance-Scale": format(guidance, ".17g"),
        "X-ACE-Step-Shift": format(shift, ".17g"),
    }
    codes = actual.get("audio_codes")
    if codes:
        if not isinstance(codes, str) or len(codes) > 2*1024*1024:
            raise RuntimeError("upstream semantic-code witness exceeds its bounded text contract")
        count = len(re.findall(r"<\|audio_code_\d+\|>", codes))
        if count > 100000:
            raise RuntimeError("upstream semantic-code count exceeds its bound")
        if count:
            headers["X-ACE-Step-LM-Code-SHA256"] = hashlib.sha256(codes.encode()).hexdigest()
            headers["X-ACE-Step-LM-Code-Count"] = str(count)
    return headers


@app.get("/health")
def health(response: Response) -> dict[str, object]:
    loaded = _dit is not None and _llm is not None
    ready = loaded and not _init_error
    identity = _loaded_identity
    loaded_variant = identity[0].rsplit(":", 1)[-1] if identity else None
    loaded_planner = _loaded_planner
    response.status_code = 200 if ready else 503
    return {
        "status": "ok" if ready else "degraded" if _init_error else "initializing",
        "model_loaded": loaded,
        "variant": loaded_variant or VARIANT,
        "planner": loaded_planner or PLANNER,
        "configured_variant": VARIANT,
        "configured_planner": PLANNER,
        "configured_profile": _profile_name(VARIANT, PLANNER),
        "loaded_variant": loaded_variant,
        "loaded_planner": loaded_planner,
        "loaded_profile": _profile_name(loaded_variant, loaded_planner) if loaded_variant and loaded_planner else None,
        "device": DEVICE,
        "offload_dit_to_cpu": OFFLOAD_DIT,
        "initialization_seconds": _init_seconds,
        "error": _init_error,
    }


@app.get("/v1/variants")
def variants() -> dict[str, object]:
    return {"default": VARIANT, "planner": PLANNER, "variants": ["acestep-v15-turbo", "acestep-v15-sft"]}


@app.post("/v1/compose")
def compose(request: ComposeRequest) -> FileResponse:
    generation_acquired = False
    try:
        _ensure_loaded()
        if _loaded_identity is None:
            raise RuntimeError("loaded model identity is unavailable")
        if request.expected_rung is not None and request.expected_rung != _loaded_identity[1]:
            raise RuntimeError("loaded capacity rung differs from admitted rung; inference refused")
        if request.admitted_rungs is not None and _loaded_identity[1] not in request.admitted_rungs:
            raise RuntimeError("loaded capacity rung is outside admitted rungs; inference refused")
        actual_variant = _loaded_identity[0].rsplit(":", 1)[-1]
        if request.expected_variant is not None and request.expected_variant != actual_variant:
            raise RuntimeError("loaded model variant differs from expected variant; inference refused")
        if not _generation_lock.acquire(blocking=False):
            raise RuntimeError("generation is busy")
        generation_acquired = True
        from acestep.inference import GenerationConfig, GenerationParams, generate_music

        turbo = "turbo" in actual_variant
        steps = request.inference_steps if request.inference_steps is not None else (8 if turbo else 50)
        if turbo and steps > 20:
            raise RuntimeError("turbo supports at most20 effective steps; inference refused")
        guidance = 1.0 if turbo else (request.guidance_scale if request.guidance_scale is not None else 7.0)
        shift = request.shift if request.shift is not None else (3.0 if turbo else 1.0)
        params = GenerationParams(
            task_type="text2music",
            thinking=True,
            caption=request.caption,
            lyrics=request.lyrics,
            duration=request.duration,
            bpm=request.bpm,
            keyscale=request.keyscale,
            inference_steps=steps,
            guidance_scale=guidance,
            shift=shift,
            seed=request.seed,
            vocal_language="en",
            use_cot_caption=request.rewrite_caption,
            use_cot_metas=request.rewrite_caption,
        )
        OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
        output = OUTPUT_DIR / f"take-{time.time_ns()}"
        output.mkdir()
        config = GenerationConfig(
            batch_size=1, audio_format="wav", use_random_seed=False,
            seeds=[request.seed],
        )
        import torch
        # Pinned upstream's single-item PT LM path ignores its seeds argument.
        # The dedicated lock protects this process-wide CPU/CUDA RNG and models.
        torch.manual_seed(request.seed)
        result = generate_music(
            _dit, _llm, params=params, config=config, save_dir=str(output),
        )
        if not result.success or not result.audios:
            raise RuntimeError(result.status_message)
        path = result.audios[0].get("path")
        if not path or not Path(path).is_file():
            raise RuntimeError("ACE-Step returned no audio file")
        sampling = _sampling_headers(result.audios[0], request.seed, steps, guidance, shift, _loaded_planner)
        response = FileResponse(path, media_type="audio/wav", filename="take.wav")
        response.headers.update(sampling)
        response.headers["X-ACE-Step-Profile-Rung"] = _loaded_identity[1]
        response.headers["X-ACE-Step-Model-ID"] = _loaded_identity[0]
        response.headers["X-ACE-Step-License-Lane"] = "permissive"
        return response
    except Exception as exc:  # noqa: BLE001 - stable HTTP boundary
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    finally:
        if generation_acquired:
            _generation_lock.release()


@app.post("/v1/capacity/degrade")
def degrade(to: str) -> dict[str, object]:
    global OFFLOAD_DIT
    if to not in {"full", "offload-dit"}:
        raise HTTPException(status_code=400, detail="supported rungs: full, offload-dit")
    # Never acknowledge desired configuration while another thread has already
    # snapshotted a different configuration for an unfinished model load.
    if not _lock.acquire(blocking=False):
        raise HTTPException(status_code=503, detail="model initialization is active; capacity acknowledgement unavailable")
    try:
        OFFLOAD_DIT = to == "offload-dit"
        ROOT.mkdir(parents=True, exist_ok=True)
        RUNG_FILE.write_text(to + "\n", encoding="utf-8")
        actual = _loaded_identity[1] if _loaded_identity is not None else to
        return {"applied_rung": actual, "restart_required": actual != to}
    finally:
        _lock.release()
