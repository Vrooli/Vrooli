"""Headless, managed ACE-Step 1.5 service.

The resource owns model loading and one-take generation. Batching, provenance,
and pool state belong to music-tools. The server deliberately disables ACE's
planner rewrites unless a caller explicitly opts into them.
"""
from __future__ import annotations

import os
import json
import threading
import time
from pathlib import Path

from fastapi import FastAPI, HTTPException
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
    seed: int = 0
    inference_steps: int | None = Field(default=None, ge=1, le=100)
    guidance_scale: float | None = Field(default=None, ge=0, le=20)
    rewrite_caption: bool = False
    offload_dit_to_cpu: bool | None = None


def _ensure_loaded() -> None:
    global _dit, _llm, _device_keepalive, _init_error, _init_seconds
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

            dit = AceStepHandler()
            message, ok = dit.initialize_service(
                project_root=str(ROOT),
                config_path=VARIANT,
                device=DEVICE,
                offload_to_cpu=True,
                offload_dit_to_cpu=OFFLOAD_DIT,
            )
            if not ok:
                raise RuntimeError(f"DiT initialization failed: {message}")
            llm = LLMHandler()
            message, ok = llm.initialize(
                checkpoint_dir=str(CHECKPOINT_DIR),
                lm_model_path=PLANNER,
                backend="pt",
                device=DEVICE,
                offload_to_cpu=True,
            )
            if not ok:
                raise RuntimeError(f"planner initialization failed: {message}")
            _dit, _llm = dit, llm
            if DEVICE == "cuda":
                import torch
                _device_keepalive = torch.empty((1,), device="cuda")
            _init_error = ""
        except Exception as exc:  # noqa: BLE001 - surfaced through health
            _init_error = f"{type(exc).__name__}: {exc}"
            raise
        finally:
            _init_seconds = time.monotonic() - started


@app.get("/health")
def health() -> dict[str, object]:
    return {
        "status": "ok" if not _init_error else "degraded",
        "model_loaded": _dit is not None and _llm is not None,
        "variant": VARIANT,
        "planner": PLANNER,
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
    try:
        _ensure_loaded()
        from acestep.inference import GenerationConfig, GenerationParams, generate_music

        turbo = "turbo" in VARIANT
        params = GenerationParams(
            task_type="text2music",
            thinking=True,
            caption=request.caption,
            lyrics=request.lyrics,
            duration=request.duration,
            bpm=request.bpm,
            keyscale=request.keyscale,
            inference_steps=request.inference_steps or (8 if turbo else 50),
            guidance_scale=request.guidance_scale or (1.0 if turbo else 7.0),
            seed=request.seed,
            vocal_language="en",
            use_cot_caption=request.rewrite_caption,
            use_cot_metas=request.rewrite_caption,
        )
        OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
        output = OUTPUT_DIR / f"take-{time.time_ns()}"
        output.mkdir()
        result = generate_music(
            _dit,
            _llm,
            params=params,
            config=GenerationConfig(batch_size=1, audio_format="wav"),
            save_dir=str(output),
        )
        if not result.success or not result.audios:
            raise RuntimeError(result.status_message)
        path = result.audios[0].get("path")
        if not path or not Path(path).is_file():
            raise RuntimeError("ACE-Step returned no audio file")
        response = FileResponse(path, media_type="audio/wav", filename="take.wav")
        response.headers["X-ACE-Step-Profile-Rung"] = "offload-dit" if OFFLOAD_DIT else "full"
        return response
    except Exception as exc:  # noqa: BLE001 - stable HTTP boundary
        raise HTTPException(status_code=503, detail=str(exc)) from exc


@app.post("/v1/capacity/degrade")
def degrade(to: str) -> dict[str, object]:
    global OFFLOAD_DIT
    if to not in {"full", "offload-dit"}:
        raise HTTPException(status_code=400, detail="supported rungs: full, offload-dit")
    OFFLOAD_DIT = to == "offload-dit"
    ROOT.mkdir(parents=True, exist_ok=True)
    RUNG_FILE.write_text(to + "\n", encoding="utf-8")
    return {"applied_rung": to, "restart_required": _dit is not None}
