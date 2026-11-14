"""Verify the declared closure and load upstream Medium using local paths only."""
from __future__ import annotations

import copy
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path
import re
import stat
import time

SOURCE_REVISION = "77e0398c26f7d22d801ec1bb036307e8b231f741"
MODEL_REVISION = "d4d3579395a8ac0df4ab13b5b1d7138ce7290f81"
MODEL_ID = "stabilityai/stable-audio-3-medium"
TORCH_ALLOCATOR_BYTES = 8 * 1024**3
BROKER_CLAIM_BYTES = 9 * 1024**3
HERE = Path(__file__).resolve().parent
REQUIRED_FILES = (
    "model.safetensors", "model_config.json", "t5gemma-b-b-ul2/config.json",
    "t5gemma-b-b-ul2/generation_config.json", "t5gemma-b-b-ul2/model.safetensors",
    "t5gemma-b-b-ul2/special_tokens_map.json", "t5gemma-b-b-ul2/tokenizer.json",
    "t5gemma-b-b-ul2/tokenizer.model", "t5gemma-b-b-ul2/tokenizer_config.json",
    "LICENSE.md", "LICENSE_GEMMA.md", "NOTICE",
)


def canonical(value: object) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False).encode()


def digest_file(path: Path, check=lambda: None) -> tuple[str, int]:
    """Refuse symlinks/special-file replacement before a potentially blocking open."""
    before = path.lstat()
    if not stat.S_ISREG(before.st_mode):
        raise ValueError(f"input is not a regular file: {path.name}")
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0)
    descriptor = os.open(path, flags)
    try:
        opened = os.fstat(descriptor)
        if not stat.S_ISREG(opened.st_mode) or (before.st_dev, before.st_ino) != (opened.st_dev, opened.st_ino):
            raise ValueError(f"input changed while opening: {path.name}")
        hasher = hashlib.sha256()
        size = 0
        with os.fdopen(descriptor, "rb", closefd=False) as stream:
            while True:
                check()
                chunk = stream.read(1024 * 1024)
                if not chunk:
                    break
                hasher.update(chunk)
                size += len(chunk)
        after = os.fstat(descriptor)
        if (opened.st_size, opened.st_mtime_ns) != (after.st_size, after.st_mtime_ns):
            raise ValueError(f"input changed while reading: {path.name}")
        return hasher.hexdigest(), size
    finally:
        os.close(descriptor)


def verify_inputs(model_dir: Path, lock_path: Path = HERE / "model-lock.json", check=lambda: None) -> dict:
    lock = json.loads(lock_path.read_bytes())
    if (lock.get("model_id"), lock.get("model_revision"), lock.get("source_revision")) != (MODEL_ID, MODEL_REVISION, SOURCE_REVISION):
        raise ValueError("unsupported model/source lock identity")
    entries = lock.get("files", [])
    if sorted(item.get("path", "") for item in entries) != sorted(REQUIRED_FILES):
        raise ValueError("model lock must declare the exact complete twelve-file closure")
    witnessed = {}
    for item in entries:
        expected = item.get("sha256", "")
        if not re.fullmatch(r"[0-9a-f]{64}", expected) or expected == "0" * 64:
            raise ValueError(f"unqualified SHA256: {item['path']}")
        relative = Path(item["path"])
        candidate = model_dir / relative
        # Parent-directory symlinks are also disallowed; no borrowed HF cache.
        if model_dir.is_symlink() or any((model_dir / parent).is_symlink() for parent in relative.parents if str(parent) != "."):
            raise ValueError("model closure contains a symlink directory")
        actual, size = digest_file(candidate, check)
        if actual != expected or size != item.get("bytes"):
            raise ValueError(f"model input checksum/size mismatch: {item['path']}")
        witnessed[item["path"]] = {"sha256": actual, "bytes": size}
    return {"model_id": MODEL_ID, "model_revision": MODEL_REVISION, "source_revision": SOURCE_REVISION,
            "license": lock["license"], "files": witnessed, "model_lock_sha256": digest_file(lock_path)[0]}


def local_config(original: dict, model_dir: Path) -> dict:
    effective = copy.deepcopy(original)
    configs = effective["model"]["conditioning"]["configs"]
    text = [item for item in configs if item.get("type") == "t5gemma"]
    if len(text) != 1 or text[0]["config"].get("model_name", "google/t5gemma-b-b-ul2") != "google/t5gemma-b-b-ul2":
        raise ValueError("expected exactly one supported T5Gemma conditioner")
    settings = text[0]["config"]
    settings["model_path"] = str(model_dir / "t5gemma-b-b-ul2")
    settings.pop("repo_id", None)
    settings.pop("subfolder", None)
    if original.get("sample_rate") != 44100 or type(original.get("sample_size")) is not int or original["sample_size"] < 48 * 44100:
        raise ValueError("unsupported Medium sample-rate/length configuration")
    return effective


def load_local(model_dir: Path, lock_path: Path = HERE / "model-lock.json") -> dict:
    # Set before importing transformers/hub/upstream; loading never needs a token.
    os.environ["HF_HUB_OFFLINE"] = "1"
    os.environ["TRANSFORMERS_OFFLINE"] = "1"
    os.environ["ENABLE_TORCH_COMPILE"] = "0"
    started = time.monotonic()
    identity = verify_inputs(model_dir, lock_path)
    original = json.loads((model_dir / "model_config.json").read_bytes())
    effective = local_config(original, model_dir)
    import torch
    import flash_attn
    from flash_attn import flash_attn_func, flash_attn_varlen_func
    from stable_audio_3.loading_utils import load_diffusion_cond
    from stable_audio_3.model import StableAudioModel
    from stable_audio_3.models import transformer as attention

    if not torch.cuda.is_available() or torch.version.cuda != "12.6" or str(torch.__version__) != "2.7.1+cu126":
        raise RuntimeError("qualified runtime requires Torch2.7.1+cu126 and available CUDA")
    if not torch._C._GLIBCXX_USE_CXX11_ABI or not callable(flash_attn_func) or not callable(flash_attn_varlen_func):
        raise RuntimeError("qualified FlashAttention2 ABI/functions unavailable")
    if attention.flash_attn_func is not flash_attn_func or attention.flash_attn_varlen_func is not flash_attn_varlen_func or attention.index_first_axis is None:
        raise RuntimeError("upstream FlashAttention2 bindings are unavailable; fallback is unqualified")
    if importlib.metadata.version("transformers") != "5.10.4" or importlib.metadata.version("flash-attn").split("+")[0] != "2.8.3.post2":
        raise RuntimeError("unreviewed transformer/attention runtime version")
    declared = os.environ.get("SA3_COMPOSED_SHA256", "")
    if not re.fullmatch(r"[0-9a-f]{64}", declared) or declared == "0" * 64:
        raise RuntimeError("unqualified composed artifact identity")
    total = torch.cuda.get_device_properties(0).total_memory
    if type(total) is not int or total < BROKER_CLAIM_BYTES:
        raise RuntimeError("GPU total memory cannot support the fixed pilot budget")
    torch.cuda.set_per_process_memory_fraction(TORCH_ALLOCATOR_BYTES / total, device=0)
    torch.cuda.reset_peak_memory_stats()
    # Upstream casts after its .to(device); cast on CPU first to avoid a transient
    # full-FP32 GPU allocation. Final precision matches from_pretrained's default.
    loaded = load_diffusion_cond(effective, str(model_dir / "model.safetensors"), device="cpu", model_half=True)
    loaded.to("cuda")
    loaded.use_lora = False
    loaded.lora_names = []
    model = StableAudioModel(loaded, effective, "cuda", True)
    torch.cuda.synchronize()
    source_files = {name: digest_file(HERE / name)[0] for name in ("server.py", "loader.py", "requirements.lock")}
    text_id = next(item["id"] for item in effective["model"]["conditioning"]["configs"] if item["type"] == "t5gemma")
    main_dtype = str(next(loaded.model.parameters()).dtype)
    text_dtype = str(next(loaded.conditioner.conditioners[text_id].model.parameters()).dtype)
    return {"model": model, "identity": identity, "config": effective, "text_conditioner_id": text_id,
            "effective_config_sha256": hashlib.sha256(canonical(effective)).hexdigest(),
            "runtime": {"python": __import__("platform").python_version(), "torch": str(torch.__version__),
                        "cuda": torch.version.cuda, "flash_attn": importlib.metadata.version("flash-attn"),
                        "flash_attn_module_version": flash_attn.__version__, "transformers": "5.10.4",
                        "cxx11_abi": True, "device": torch.cuda.get_device_name(), "model_half": True,
                        "torch_compile": False, "loading_strategy": "cpu-fp16-then-cuda; conditioner-owned-first-use-transfer",
                        "main_parameter_dtype": main_dtype, "conditioner_parameter_dtype": text_dtype,
                        "declared_composed_sha256": declared, "source_files": source_files,
                        "torch_allocator_ceiling_bytes": TORCH_ALLOCATOR_BYTES, "broker_claim_bytes": BROKER_CLAIM_BYTES,
                        "non_torch_headroom_bytes": BROKER_CLAIM_BYTES - TORCH_ALLOCATOR_BYTES,
                        "non_torch_peak_bytes": None,
                        "attention": {"upstream_fa2_bindings_verified": True, "actual_selected_paths": None}},
            "initialization": {"seconds": time.monotonic() - started,
                               "cuda_peak_allocated_bytes": torch.cuda.max_memory_allocated(),
                               "cuda_peak_reserved_bytes": torch.cuda.max_memory_reserved()}}
