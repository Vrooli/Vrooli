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
from typing import Any, Callable, NamedTuple, Optional, Sequence

from . import _adapters, _common


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

# ARCHITECTURES is the pure, importable mirror of the Go architecture→technique
# derivation table (internal/models/architecture.go `architectureTechniques`).
# It maps a model's weight-lineage architecture to the operations it can DERIVE
# (beyond its declared/native ops) through a named technique, with a `ready` gate
# (False ⇒ declared but not yet proven on this architecture — an honest
# derived_pipeline_unproven state, flipped only by the attended GPU acceptance
# run). The Go parity test (TestArchitectureTechniquesMirrorPython) asserts the
# {op, technique, ready} triples agree, so capability derivation cannot drift
# between the selector (Go) and the runtime (Python). The quality `caveat` prose
# is Go-only (UI text) and intentionally excluded from the parity contract.
ARCHITECTURES = {
    "sd15": [
        {"op": "image_to_image", "technique": "sd-img2img", "ready": True},
        {"op": "inpaint", "technique": "diffusers-inpaint", "ready": True},
        {"op": "outpaint", "technique": "diffusers-outpaint", "ready": False},
        {"op": "edit_instruct", "technique": "edit-via-img2img", "ready": False},
    ],
    "sdxl": [
        {"op": "image_to_image", "technique": "sd-img2img", "ready": True},
        {"op": "inpaint", "technique": "diffusers-inpaint", "ready": True},
        {"op": "outpaint", "technique": "diffusers-outpaint", "ready": False},
        {"op": "edit_instruct", "technique": "edit-via-img2img", "ready": False},
    ],
    "flux": [
        {"op": "image_to_image", "technique": "sd-img2img", "ready": False},
        {"op": "edit_instruct", "technique": "edit-via-img2img", "ready": False},
    ],
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


# =============================================================================
# Derived base-checkpoint transforms (inpaint via AutoPipeline).
#
# These are NOT family adapters. A *base* text2image checkpoint (any sd15/sdxl
# model) is loaded into the standard inpaint pipeline for its architecture and run
# as a masked regenerate. There is no per-model call contract — the only knobs are
# the generic {prompt, negative, strength, steps, guidance, seed}. Offerability is
# gated UPSTREAM in Go (the architecture→technique `Ready` flag in
# internal/models/architecture.go, mirrored in ARCHITECTURES above); this runner
# only executes, so it does not re-check Ready (a caller reaches it only for a
# technique the Go selector already deemed offerable). A dedicated *-inpainting
# checkpoint blends masked edges more cleanly — that quality trade-off is the
# derived caveat the Go table carries, not an execution error here.
# =============================================================================


class TransformParams(NamedTuple):
    """Normalized params for a base-checkpoint masked regenerate (the Go
    DiffusersInpaint arg-builder emits exactly these)."""

    prompt: str
    negative_prompt: Optional[str] = None
    strength: Optional[float] = None  # how much the masked region may change (0..1)
    steps: int = 0  # 0 → default
    guidance: Optional[float] = None  # text guidance scale (cfg_scale)
    seed: int = 0


# arch → concrete inpaint pipeline class. from_single_file needs a concrete class
# (it cannot auto-resolve a bare checkpoint); a diffusers repo tree is loaded with
# AutoPipelineForInpainting instead, which auto-detects the architecture.
_INPAINT_SINGLE_FILE = {
    "sd15": "StableDiffusionInpaintPipeline",
    "sdxl": "StableDiffusionXLInpaintPipeline",
}

# Per-architecture inpaint dtype + default sampler steps. Kept tiny + explicit so
# the cheap path (kwargs) stays torch-free and unit-testable.
_INPAINT_DEFAULT_STEPS = 30


def build_inpaint_kwargs(params: "TransformParams") -> dict:
    """Pure param→kwargs mapping for a base-checkpoint inpaint (no torch).
    Unit-testable; mirrors build_call_kwargs for the family path."""
    out = {
        "prompt": params.prompt,
        "num_inference_steps": params.steps or _INPAINT_DEFAULT_STEPS,
        "guidance_scale": 7.5 if params.guidance is None else params.guidance,
        "strength": 0.85 if params.strength is None else params.strength,
    }
    if params.negative_prompt:
        out["negative_prompt"] = params.negative_prompt
    return out


def _inpaint_class(architecture: str) -> str:
    cls = _INPAINT_SINGLE_FILE.get(architecture)
    if cls is None:
        _common.fail(
            f"no inpaint pipeline class for architecture {architecture!r}; "
            f"known: {sorted(_INPAINT_SINGLE_FILE)}",
            code=4,
        )
    return cls  # type: ignore[return-value]


def _import_diffusers_backend():
    try:
        import diffusers  # noqa: WPS433
        import torch  # noqa: WPS433
        from PIL import Image  # noqa: WPS433

        return diffusers, torch, Image
    except ImportError as exc:  # pragma: no cover - exercised on un-provisioned hosts
        _common.fail(
            f"missing diffusers backend ({exc}); the Python venv is not provisioned. "
            "Ensure `vrooli host install uv`, then restart image-tools to build/repair "
            "the lock-pinned venv (versions come from internal/pydeps/requirements.lock; "
            "GPU strongly recommended — see docs/reference/backends.md).",
            code=3,
        )
        raise


def _place_pipe(pipe, torch, use_cuda: bool, offload: bool):
    """Move a constructed pipeline onto the device, choosing an offload mode from
    the ACTUAL free VRAM (mirrors run()): model-level offload with headroom, else
    sequential per-submodule streaming so a contended GPU completes instead of
    OOMing."""
    if use_cuda and offload:
        free_bytes, _ = torch.cuda.mem_get_info()
        if free_bytes < 10 * 1024**3:
            pipe.enable_sequential_cpu_offload()
        else:
            pipe.enable_model_cpu_offload()
        return pipe
    return pipe.to("cuda" if use_cuda else "cpu")


def inpaint_smoke(*, architecture: str, model_dir: str, deep: bool = False) -> str:
    """Install-time load-smoke for a base-checkpoint inpaint, the analogue of
    smoke() for the family path. Cheap path: the architecture's inpaint pipeline
    class resolves against the installed diffusers and the install shape is valid.
    deep=True constructs the pipeline (reads full weights). Never a fabricated
    pass — fails loud via _common.fail."""
    cls_name = _inpaint_class(architecture)
    diffusers, torch, _Image = _import_diffusers_backend()

    if getattr(diffusers, cls_name, None) is None:
        _common.fail(
            f"installed diffusers has no inpaint pipeline class {cls_name!r} for "
            f"architecture {architecture!r} (runtime too old? check runtime.min_runtime)",
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
            f"diffusers inpaint smoke OK (shallow): arch={architecture} class={cls_name} "
            f"shape={shape}; pipeline class resolves and the install shape is valid"
        )

    pipe = _load_inpaint_pipe(diffusers, torch, architecture, model_dir, repo_index, single)
    return f"diffusers inpaint smoke OK (deep): constructed {type(pipe).__name__} from {model_dir}"


def _load_inpaint_pipe(diffusers, torch, architecture: str, model_dir: str, repo_index: bool, single):
    use_cuda = torch.cuda.is_available()
    dtype = _resolve_dtype(torch, "float16", use_cuda)
    load_kwargs = {"torch_dtype": dtype}
    # SD-1.5 base checkpoints ship a safety checker that blanks flagged output; the
    # scenario owns its own NSFW gate, so disable the in-pipeline one (SD-family
    # only — SDXL's inpaint pipeline takes no safety_checker arg).
    if architecture == "sd15":
        load_kwargs["safety_checker"] = None
    try:
        if repo_index:
            pipe = diffusers.AutoPipelineForInpainting.from_pretrained(model_dir, **load_kwargs)
        else:
            cls = getattr(diffusers, _inpaint_class(architecture))
            pipe = cls.from_single_file(single, **load_kwargs)
    except Exception as exc:  # noqa: BLE001
        _common.fail(
            f"failed to load inpaint pipeline for arch {architecture!r} from {model_dir!r}: {exc}",
            code=5,
        )
        raise
    return pipe


def run_inpaint(
    *,
    architecture: str,
    model_dir: str,
    image_path: str,
    mask_path: str,
    out_path: str,
    params: "TransformParams",
    lora_specs: Optional[Sequence[str]] = None,
) -> None:
    """Load a base checkpoint into its architecture's inpaint pipeline and run one
    masked regenerate (white mask = repaint, black = keep), writing a PNG. FAILS
    LOUD on a missing backend or load failure — never a fabricated success.
    lora_specs are fused before placement."""
    diffusers, torch, Image = _import_diffusers_backend()

    try:
        image = Image.open(image_path).convert("RGB")
        mask = Image.open(mask_path).convert("L")  # white=repaint, black=keep
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image/mask: {exc}", code=6)

    single = _find_single_file(model_dir) if os.path.isdir(model_dir) else None
    repo_index = os.path.isfile(os.path.join(model_dir, "model_index.json"))
    if not repo_index and single is None:
        _common.fail(
            f"model dir {model_dir!r} has neither model_index.json (repo) nor a "
            "single-file .safetensors/.ckpt checkpoint",
            code=5,
        )

    pipe = _load_inpaint_pipe(diffusers, torch, architecture, model_dir, repo_index, single)
    _adapters.apply_loras(pipe, lora_specs or [])
    use_cuda = torch.cuda.is_available()
    pipe = _place_pipe(pipe, torch, use_cuda, offload=True)

    kwargs = build_inpaint_kwargs(params)
    kwargs["image"] = image
    kwargs["mask_image"] = mask
    # Pin the output geometry to the (multiple-of-8) input size so the pipeline does
    # not silently fall back to its native default and return a differently-sized
    # canvas the caller did not ask for.
    w, h = image.size
    kwargs["width"], kwargs["height"] = w - (w % 8), h - (h % 8)
    if params.seed:
        kwargs["generator"] = torch.Generator(
            device="cuda" if use_cuda else "cpu"
        ).manual_seed(params.seed)

    try:
        result = pipe(**kwargs).images[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"inpaint ({architecture}) failed: {exc}", code=8)
        raise
    try:
        result.save(out_path, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)


# =============================================================================
# Generic base-checkpoint generate (text2image) + transform (img2img).
#
# A *base* text2image checkpoint (any sd15/sdxl/flux model) generates from a
# prompt and transforms an init image — the two ops a base diffusers REPO serves
# directly (a single-file checkpoint takes the stable-diffusion.cpp path instead;
# this runner exists so a diffusers-repo import, which sd.cpp cannot load, still
# generates). text_to_image is a NATIVE op of the imported base; image_to_image is
# its architecture-derived transform. Like the inpaint runner these are NOT family
# adapters — the only knobs are the generic {prompt, negative, steps, guidance,
# size, strength, seed}, and offerability is gated upstream in Go.
# =============================================================================


class GenerateParams(NamedTuple):
    """Normalized params for a base-checkpoint generate/transform (the Go
    DiffusersText2Image / DiffusersImg2Img arg-builders emit exactly these)."""

    prompt: str
    negative_prompt: Optional[str] = None
    steps: int = 0  # 0 → default
    guidance: Optional[float] = None  # text guidance scale (cfg_scale)
    strength: Optional[float] = None  # img2img only: how far from the init image (0..1)
    width: int = 0  # 0 → pipeline default
    height: int = 0  # 0 → pipeline default
    seed: int = 0


# arch → concrete txt2img / img2img pipeline class. from_single_file needs a
# concrete class; a diffusers repo tree is loaded with the AutoPipeline, which
# auto-detects the architecture.
_TXT2IMG_SINGLE_FILE = {
    "sd15": "StableDiffusionPipeline",
    "sdxl": "StableDiffusionXLPipeline",
}
_IMG2IMG_SINGLE_FILE = {
    "sd15": "StableDiffusionImg2ImgPipeline",
    "sdxl": "StableDiffusionXLImg2ImgPipeline",
}

_GENERATE_DEFAULT_STEPS = 30


def build_txt2img_kwargs(params: "GenerateParams") -> dict:
    """Pure param→kwargs mapping for a base-checkpoint text2image (no torch)."""
    out = {
        "prompt": params.prompt,
        "num_inference_steps": params.steps or _GENERATE_DEFAULT_STEPS,
        "guidance_scale": 7.5 if params.guidance is None else params.guidance,
    }
    if params.negative_prompt:
        out["negative_prompt"] = params.negative_prompt
    if params.width:
        out["width"] = params.width - (params.width % 8)
    if params.height:
        out["height"] = params.height - (params.height % 8)
    return out


def build_img2img_kwargs(params: "GenerateParams") -> dict:
    """Pure param→kwargs mapping for a base-checkpoint img2img (no torch). Carries
    the generic txt2img knobs plus the init-image strength; width/height are taken
    from the init image by the pipeline, so they are not emitted here."""
    out = {
        "prompt": params.prompt,
        "num_inference_steps": params.steps or _GENERATE_DEFAULT_STEPS,
        "guidance_scale": 7.5 if params.guidance is None else params.guidance,
        "strength": 0.7 if params.strength is None else params.strength,
    }
    if params.negative_prompt:
        out["negative_prompt"] = params.negative_prompt
    return out


def _generate_class(architecture: str, single_file_map: dict, op: str) -> str:
    cls = single_file_map.get(architecture)
    if cls is None:
        _common.fail(
            f"no {op} pipeline class for architecture {architecture!r}; "
            f"known: {sorted(single_file_map)}",
            code=4,
        )
    return cls  # type: ignore[return-value]


def _load_generate_pipe(diffusers, torch, architecture, model_dir, repo_index, single, auto_cls, single_file_map, op):
    use_cuda = torch.cuda.is_available()
    dtype = _resolve_dtype(torch, "float16", use_cuda)
    load_kwargs = {"torch_dtype": dtype}
    if architecture == "sd15":
        # The scenario owns its NSFW gate; disable the in-pipeline SD-1.5 checker.
        load_kwargs["safety_checker"] = None
    try:
        if repo_index:
            pipe = getattr(diffusers, auto_cls).from_pretrained(model_dir, **load_kwargs)
        else:
            cls = getattr(diffusers, _generate_class(architecture, single_file_map, op))
            pipe = cls.from_single_file(single, **load_kwargs)
    except Exception as exc:  # noqa: BLE001
        _common.fail(
            f"failed to load {op} pipeline for arch {architecture!r} from {model_dir!r}: {exc}",
            code=5,
        )
        raise
    return pipe


def run_txt2img(
    *,
    architecture: str,
    model_dir: str,
    out_path: str,
    params: "GenerateParams",
    lora_specs: Optional[Sequence[str]] = None,
) -> None:
    """Load a base checkpoint into its architecture's text2image pipeline and
    generate one image from the prompt, writing a PNG. FAILS LOUD on a missing
    backend or load failure — never a fabricated success. lora_specs (the resolved
    ``--lora <path>:<scale>`` conditioning stack) are fused before placement."""
    diffusers, torch, _Image = _import_diffusers_backend()
    single = _find_single_file(model_dir) if os.path.isdir(model_dir) else None
    repo_index = os.path.isfile(os.path.join(model_dir, "model_index.json"))
    if not repo_index and single is None:
        _common.fail(
            f"model dir {model_dir!r} has neither model_index.json (repo) nor a "
            "single-file .safetensors/.ckpt checkpoint",
            code=5,
        )
    pipe = _load_generate_pipe(
        diffusers, torch, architecture, model_dir, repo_index, single,
        "AutoPipelineForText2Image", _TXT2IMG_SINGLE_FILE, "text_to_image",
    )
    _adapters.apply_loras(pipe, lora_specs or [])
    use_cuda = torch.cuda.is_available()
    pipe = _place_pipe(pipe, torch, use_cuda, offload=True)

    kwargs = build_txt2img_kwargs(params)
    if params.seed:
        kwargs["generator"] = torch.Generator(device="cuda" if use_cuda else "cpu").manual_seed(params.seed)
    try:
        result = pipe(**kwargs).images[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"text_to_image ({architecture}) failed: {exc}", code=8)
        raise
    try:
        result.save(out_path, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)


def run_img2img(
    *,
    architecture: str,
    model_dir: str,
    image_path: str,
    out_path: str,
    params: "GenerateParams",
    lora_specs: Optional[Sequence[str]] = None,
) -> None:
    """Load a base checkpoint into its architecture's img2img pipeline and
    transform the init image guided by the prompt, writing a PNG. FAILS LOUD on a
    missing backend or load failure — never a fabricated success. lora_specs are
    fused before placement."""
    diffusers, torch, Image = _import_diffusers_backend()
    try:
        init = Image.open(image_path).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open init image: {exc}", code=6)
        raise

    single = _find_single_file(model_dir) if os.path.isdir(model_dir) else None
    repo_index = os.path.isfile(os.path.join(model_dir, "model_index.json"))
    if not repo_index and single is None:
        _common.fail(
            f"model dir {model_dir!r} has neither model_index.json (repo) nor a "
            "single-file .safetensors/.ckpt checkpoint",
            code=5,
        )
    pipe = _load_generate_pipe(
        diffusers, torch, architecture, model_dir, repo_index, single,
        "AutoPipelineForImage2Image", _IMG2IMG_SINGLE_FILE, "image_to_image",
    )
    _adapters.apply_loras(pipe, lora_specs or [])
    use_cuda = torch.cuda.is_available()
    pipe = _place_pipe(pipe, torch, use_cuda, offload=True)

    kwargs = build_img2img_kwargs(params)
    kwargs["image"] = init
    if params.seed:
        kwargs["generator"] = torch.Generator(device="cuda" if use_cuda else "cpu").manual_seed(params.seed)
    try:
        result = pipe(**kwargs).images[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"image_to_image ({architecture}) failed: {exc}", code=8)
        raise
    try:
        result.save(out_path, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)
