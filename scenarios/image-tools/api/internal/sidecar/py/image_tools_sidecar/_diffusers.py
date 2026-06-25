"""_diffusers — the single generic runner for diffusers-backed image edits.

This replaces the previous one-model-only hardcode: edit_instruct.py imported
exactly ``StableDiffusionInstructPix2PixPipeline`` and called it with
IP2P-specific kwargs, so the diffusers backend genuinely supported one model.
Execution is now DECLARED, not coded — each pipeline architecture is a *family*
adapter that maps normalized op params onto that family's call kwargs and load
story. Adding a mainstream model is a registry row; a new architecture is one
adapter entry here, mirrored by its Go twin in internal/models/families.go and
asserted in lockstep by a conformance test.

Two seams keep this honest:
  * ``build_call_kwargs`` (and the FAMILIES table) is PURE — no torch/diffusers
    import — so the param→kwarg mapping is unit-testable on any host.
  * loading + running needs the heavy backend and FAILS LOUD (never a fake
    success) when it is not provisioned, exactly like the gated standalone
    backends. An adapter that is declared but not yet proven (ready=False) refuses
    to run rather than guessing at an unstable upstream pipeline.
"""

from __future__ import annotations

import os
from typing import Any, Callable, NamedTuple, Optional

from . import _common


class Params(NamedTuple):
    """Normalized, family-agnostic edit-instruct parameters (the Go arg-builder
    emits exactly these; every adapter maps from here)."""

    prompt: str
    steps: int = 0  # 0 → use the family default
    guidance: Optional[float] = None  # text guidance scale (cfg_scale)
    image_guidance: Optional[float] = None  # identity-preservation knob (strength)
    negative_prompt: Optional[str] = None
    seed: int = 0


class Adapter(NamedTuple):
    family: str
    pipeline_class: str
    dtype: str  # "float16" | "bfloat16" | "" (device default)
    multi_image: bool  # call expects image=[...] (True) vs a single image
    offload: bool  # prefer enable_model_cpu_offload() over .to('cuda') (big models)
    disable_safety: bool  # pass safety_checker=None at load (SD-family only)
    ready: bool  # adapter is proven runnable; False ⇒ refuse to run
    build: Callable[["Params"], dict]  # → scalar call kwargs (no image/generator)
    pending: str = ""


def _ip2p_kwargs(p: "Params") -> dict:
    return {
        "prompt": p.prompt,
        "num_inference_steps": p.steps or 20,
        "guidance_scale": 7.5 if p.guidance is None else p.guidance,
        "image_guidance_scale": 1.5 if p.image_guidance is None else p.image_guidance,
    }


def _qwen_edit_plus_kwargs(p: "Params") -> dict:
    # Qwen-Image-Edit-2509 (QwenImageEditPlusPipeline): the identity/edit-strength
    # knob is true_cfg_scale (NOT image_guidance_scale); guidance_scale is fixed at
    # 1.0; a non-empty negative_prompt is required. See the model card usage.
    return {
        "prompt": p.prompt,
        "num_inference_steps": p.steps or 40,
        "true_cfg_scale": 4.0 if p.image_guidance is None else p.image_guidance,
        "negative_prompt": p.negative_prompt or " ",
        "guidance_scale": 1.0 if p.guidance is None else p.guidance,
        "num_images_per_prompt": 1,
    }


def _unproven(_p: "Params") -> dict:  # pragma: no cover - guarded before call
    raise RuntimeError("family adapter is not yet proven (ready=False)")


_ADAPTERS = {
    "instruct-pix2pix": Adapter(
        family="instruct-pix2pix",
        pipeline_class="StableDiffusionInstructPix2PixPipeline",
        dtype="float16",
        multi_image=False,
        offload=False,
        disable_safety=True,
        ready=True,
        build=_ip2p_kwargs,
    ),
    "qwen-image-edit-plus": Adapter(
        family="qwen-image-edit-plus",
        pipeline_class="QwenImageEditPlusPipeline",
        dtype="bfloat16",
        multi_image=True,
        offload=True,
        disable_safety=False,
        ready=True,
        build=_qwen_edit_plus_kwargs,
    ),
    "flux-2-klein": Adapter(
        family="flux-2-klein",
        pipeline_class="Flux2KleinPipeline",
        dtype="bfloat16",
        multi_image=True,
        offload=True,
        disable_safety=False,
        ready=False,
        build=_unproven,
        pending=(
            "diffusers Flux2KleinPipeline is install-from-git as of 2026-06; pin a "
            "released version + prove an attended run before enabling."
        ),
    ),
    "longcat-image-edit": Adapter(
        family="longcat-image-edit",
        pipeline_class="LongCatImageEditPipeline",
        dtype="bfloat16",
        multi_image=True,
        offload=True,
        disable_safety=False,
        ready=False,
        build=_unproven,
        pending=(
            "diffusers LongCatImageEditPipeline needs a recent/custom diffusers; pin "
            "the version + prove an attended run before enabling."
        ),
    ),
}

# FAMILIES is the pure, importable mirror of the Go internal/models/families.go
# registry. The Go conformance test (TestDiffusersFamilyAdaptersMirrorPython)
# imports this and asserts name/pipeline_class/ready parity, so the registry, the
# Go doctor, and this runner cannot drift.
FAMILIES = {
    name: {"pipeline_class": a.pipeline_class, "ready": a.ready}
    for name, a in _ADAPTERS.items()
}


def adapter_for(family: str) -> Adapter:
    a = _ADAPTERS.get(family)
    if a is None:
        _common.fail(
            f"unknown diffusers family {family!r} (no registered adapter); "
            f"known: {sorted(_ADAPTERS)}",
            code=4,
        )
    return a  # type: ignore[return-value]


def build_call_kwargs(family: str, params: "Params") -> dict:
    """Pure param→kwargs mapping for a family (no torch). Unit-testable."""
    return adapter_for(family).build(params)


def _resolve_dtype(torch, dtype: str, use_cuda: bool):
    if not use_cuda:
        return torch.float32
    if dtype == "bfloat16":
        return torch.bfloat16
    if dtype == "float16":
        return torch.float16
    return torch.float16


def _find_single_file(model_dir: str) -> Optional[str]:
    """Return a single-file diffusers checkpoint inside model_dir, if that is the
    install shape (e.g. the instruct-pix2pix .safetensors asset) rather than a
    diffusers repo tree (model_index.json present)."""
    for entry in sorted(os.listdir(model_dir)):
        if entry.endswith((".safetensors", ".ckpt")):
            return os.path.join(model_dir, entry)
    return None


def smoke(*, family: str, model_dir: str, deep: bool = False) -> str:
    """Install-time load-smoke for a diffusers-family model. Proves the provisioned
    runtime can CONSTRUCT this model before a user op depends on it, WITHOUT a full
    forward pass.

    Default (cheap) checks, sufficient to catch the common "installed but broken"
    cases without loading multi-GB weights:
      * the family adapter is registered and proven (ready=True);
      * diffusers/torch import from the provisioned venv;
      * the declared pipeline_class is an attribute of the installed diffusers
        (catches "runtime too old for this pipeline architecture");
      * the model dir is a recognizable install shape (repo tree with
        model_index.json, or a single-file checkpoint).
    deep=True additionally LOADS the pipeline (from_pretrained/from_single_file)
    on a low-mem path — opt-in, because it reads the full weights (prohibitive for
    a 57GB model at install time). Returns a human summary; raises SystemExit via
    _common.fail on any failure (never a fabricated pass).
    """
    adapter = adapter_for(family)
    if not adapter.ready:
        _common.fail(
            f"diffusers family {family!r} adapter is not yet proven: {adapter.pending}",
            code=4,
        )

    try:
        import diffusers  # noqa: WPS433
        import torch  # noqa: WPS433,F401
    except ImportError as exc:  # pragma: no cover - exercised on un-provisioned hosts
        _common.fail(
            f"missing diffusers backend ({exc}); the Python venv is not provisioned. "
            "Ensure `vrooli host install uv`, then restart image-tools to build/repair the venv.",
            code=3,
        )

    cls = getattr(diffusers, adapter.pipeline_class, None)
    if cls is None:
        _common.fail(
            f"installed diffusers has no pipeline class {adapter.pipeline_class!r} for "
            f"family {family!r} (runtime too old? check runtime.min_runtime)",
            code=5,
        )

    if not os.path.isdir(model_dir):
        _common.fail(f"model dir {model_dir!r} does not exist", code=4)
    single = _find_single_file(model_dir)
    repo_index = os.path.isfile(os.path.join(model_dir, "model_index.json"))
    if not repo_index and single is None:
        _common.fail(
            f"model dir {model_dir!r} has neither model_index.json (repo) nor a "
            "single-file .safetensors/.ckpt checkpoint (incomplete download?)",
            code=5,
        )

    if not deep:
        shape = "repo" if repo_index else "single-file"
        return (
            f"diffusers smoke OK (shallow): family={family} class={adapter.pipeline_class} "
            f"shape={shape}; pipeline class resolves and the install shape is valid"
        )

    # deep: actually construct the pipeline (reads full weights).
    use_cuda = torch.cuda.is_available()
    dtype = _resolve_dtype(torch, adapter.dtype, use_cuda)
    load_kwargs = {"torch_dtype": dtype}
    if adapter.disable_safety:
        load_kwargs["safety_checker"] = None
    try:
        if repo_index:
            pipe = diffusers.DiffusionPipeline.from_pretrained(model_dir, **load_kwargs)
        else:
            pipe = cls.from_single_file(single, **load_kwargs)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"deep smoke: failed to construct {family!r} from {model_dir!r}: {exc}", code=5)
        raise
    got = type(pipe).__name__
    if got != adapter.pipeline_class:
        _common.fail(
            f"deep smoke: model at {model_dir!r} loaded as {got}, expected "
            f"{adapter.pipeline_class} for family {family!r}",
            code=5,
        )
    return f"diffusers smoke OK (deep): constructed {got} from {model_dir}"


def run(*, family: str, model_dir: str, image_paths, out_path: str, params: "Params") -> None:
    """Load the family's pipeline from model_dir and run one edit, writing a PNG.

    Supports both diffusers install shapes: a repo tree (model_index.json →
    from_pretrained, class auto-resolved) and a single-file checkpoint
    (from_single_file with the declared pipeline_class). FAILS LOUD on a missing
    backend, an unproven family, or a class mismatch — never a fabricated success.
    """
    adapter = adapter_for(family)
    if not adapter.ready:
        _common.fail(
            f"diffusers family {family!r} adapter is not yet proven: {adapter.pending}",
            code=4,
        )

    try:
        import diffusers  # noqa: WPS433
        import torch  # noqa: WPS433
        from PIL import Image  # noqa: WPS433
    except ImportError as exc:  # pragma: no cover - exercised on un-provisioned hosts
        _common.fail(
            f"missing diffusers backend ({exc}); the Python venv is not provisioned. "
            "Ensure `vrooli host install uv`, then restart image-tools to build/repair "
            "the lock-pinned venv (versions come from internal/pydeps/requirements.lock; "
            "GPU strongly recommended — see docs/reference/backends.md).",
            code=3,
        )
        raise

    if not image_paths:
        _common.fail("edit_instruct requires at least one input image", code=2)
    try:
        images = [Image.open(p).convert("RGB") for p in image_paths]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image: {exc}", code=6)

    use_cuda = torch.cuda.is_available()
    dtype = _resolve_dtype(torch, adapter.dtype, use_cuda)
    load_kwargs = {"torch_dtype": dtype}
    if adapter.disable_safety:
        load_kwargs["safety_checker"] = None

    try:
        single = _find_single_file(model_dir) if os.path.isdir(model_dir) else None
        repo_index = os.path.isfile(os.path.join(model_dir, "model_index.json"))
        if repo_index:
            pipe = diffusers.DiffusionPipeline.from_pretrained(model_dir, **load_kwargs)
        elif single is not None:
            cls = getattr(diffusers, adapter.pipeline_class, None)
            if cls is None:
                _common.fail(
                    f"diffusers has no pipeline class {adapter.pipeline_class!r} for family "
                    f"{family!r} (runtime too old? check runtime.min_runtime)",
                    code=5,
                )
            pipe = cls.from_single_file(single, **load_kwargs)
        else:
            _common.fail(
                f"model dir {model_dir!r} has neither model_index.json (repo) nor a "
                "single-file .safetensors/.ckpt checkpoint",
                code=5,
            )
    except SystemExit:
        raise
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to load {family!r} model {model_dir!r}: {exc}", code=5)
        raise

    got = type(pipe).__name__
    if got != adapter.pipeline_class:
        _common.fail(
            f"model at {model_dir!r} loaded as {got}, expected {adapter.pipeline_class} "
            f"for family {family!r} (registry runtime descriptor mismatch)",
            code=5,
        )

    if use_cuda and adapter.offload:
        # Big models stream layers to fit consumer VRAM. Pick the offload mode from
        # the ACTUAL free VRAM: model-level offload (one whole component resident,
        # faster) when there is comfortable headroom, else sequential offload
        # (per-submodule streaming, ~1-2GB peak, slower) so a contended 16GB GPU
        # still completes instead of OOMing.
        free_bytes, _ = torch.cuda.mem_get_info()
        if free_bytes < 10 * 1024**3:
            pipe.enable_sequential_cpu_offload()
        else:
            pipe.enable_model_cpu_offload()
    else:
        pipe = pipe.to("cuda" if use_cuda else "cpu")

    kwargs = adapter.build(params)
    kwargs["image"] = images if adapter.multi_image else images[0]
    if params.seed:
        kwargs["generator"] = torch.Generator(
            device="cuda" if use_cuda else "cpu"
        ).manual_seed(params.seed)

    try:
        result = pipe(**kwargs).images[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"{family!r} instruction edit failed: {exc}", code=8)
        raise

    try:
        result.save(out_path, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)
