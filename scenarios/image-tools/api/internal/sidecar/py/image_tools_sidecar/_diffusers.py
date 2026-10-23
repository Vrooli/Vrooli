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
import json
import resource
import time
from typing import Any, Callable, NamedTuple, Optional, Sequence

from PIL import Image
from PIL.PngImagePlugin import PngInfo

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
    use_cuda = _common.use_cuda(torch)
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

    use_cuda = _common.use_cuda(torch)
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

# Per-model inpaint defaults. SDXL Inpainting 0.1's upstream model card uses
# strength=0.99 for its reference inpaint call; generic SD15 inpaint keeps the
# established 0.85 default. Keep this table explicit and the kwargs mapping
# torch-free so the model-specific choice is testable without loading weights.
_INPAINT_DEFAULT_STEPS = 30
_INPAINT_DEFAULT_STRENGTH = 0.85
_INPAINT_DEFAULT_STRENGTH_BY_MODEL = {
    "sdxl-inpainting-0.1": 0.99,
}


def build_inpaint_kwargs(params: "TransformParams", model_id: str = "") -> dict:
    """Pure param→kwargs mapping for a base-checkpoint inpaint (no torch).
    Unit-testable; mirrors build_call_kwargs for the family path."""
    out = {
        "prompt": params.prompt,
        "num_inference_steps": params.steps or _INPAINT_DEFAULT_STEPS,
        "guidance_scale": 7.5 if params.guidance is None else params.guidance,
        "strength": (
            _INPAINT_DEFAULT_STRENGTH_BY_MODEL.get(model_id, _INPAINT_DEFAULT_STRENGTH)
            if params.strength is None else params.strength
        ),
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


def inpaint_smoke(*, architecture: str, model_dir: str, deep: bool = False, variant: str = "") -> str:
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

    pipe = _load_inpaint_pipe(diffusers, torch, architecture, model_dir, repo_index, single, variant)
    return f"diffusers inpaint smoke OK (deep): constructed {type(pipe).__name__} from {model_dir}"


def _load_inpaint_pipe(diffusers, torch, architecture: str, model_dir: str, repo_index: bool, single, variant: str = ""):
    use_cuda = _common.use_cuda(torch)
    dtype = _resolve_dtype(torch, "float16", use_cuda)
    load_kwargs = {"torch_dtype": dtype}
    if repo_index and variant:
        load_kwargs["variant"] = variant
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
    provenance: Optional[dict] = None,
    started: Optional[float] = None,
    variant: str = "",
    crop_to_mask: bool = False,
    crop_context_pixels: int = 0,
    crop_max_edge_px: int = 0,
    neutralize_masked_pixels: bool = False,
) -> None:
    """Load a base checkpoint into its architecture's inpaint pipeline and run one
    masked regenerate (white mask = repaint, black = keep), writing a PNG. FAILS
    LOUD on a missing backend or load failure — never a fabricated success.
    lora_specs are fused before placement."""
    started = time.perf_counter() if started is None else started
    diffusers, torch, Image = _import_diffusers_backend()

    try:
        image, source_raster = _common.load_oriented_image(image_path, 16_000_000, mode="RGB")
        mask, mask_raster = _common.load_oriented_image(mask_path, 16_000_000, mode="L")
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

    use_cuda = _common.use_cuda(torch)
    cuda_device = _begin_cuda_peak_measurement(torch, use_cuda)
    pipe = _load_inpaint_pipe(diffusers, torch, architecture, model_dir, repo_index, single, variant)
    _adapters.apply_loras(pipe, lora_specs or [])
    pipe = _place_pipe(pipe, torch, use_cuda, offload=True)

    kwargs = build_inpaint_kwargs(params, str((provenance or {}).get("model_id", "")))
    inference_image, inference_mask = image, mask
    crop_receipt = {"crop_to_mask": False}
    crop_box = None
    if crop_to_mask:
        native_max_edge = crop_max_edge_px or (1024 if architecture == "sdxl" else 512)
        inference_image, inference_mask, crop_box, crop_receipt = _prepare_inpaint_crop(
            image, mask, crop_context_pixels, native_max_edge,
            neutralize_masked_pixels=neutralize_masked_pixels,
        )
    elif crop_context_pixels or crop_max_edge_px:
        raise ValueError("inpaint crop context and max edge require crop_to_mask")
    elif neutralize_masked_pixels:
        raise ValueError("neutralize_masked_pixels requires crop_to_mask")
    kwargs["image"] = inference_image
    kwargs["mask_image"] = inference_mask
    # Pin the output geometry to the (multiple-of-8) input size so the pipeline does
    # not silently fall back to its native default and return a differently-sized
    # canvas the caller did not ask for.
    w, h = inference_image.size
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
    cuda_receipt = _cuda_peak_receipt(torch, cuda_device)
    try:
        if crop_box is None:
            result = _composite_only_mask(image, mask, result)
        else:
            result = _composite_cropped_inpaint(image, mask, result, crop_box)
        receipt = dict(provenance or {})
        receipt["effective_strength"] = kwargs["strength"]
        receipt.update({
            "source_raster": source_raster,
            "mask_raster": mask_raster,
            "elapsed_ms": round((time.perf_counter() - started) * 1000, 3),
            "process_peak_rss_bytes": int(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * 1024),
            "process_peak_rss_scope": "cumulative sidecar process high-water mark",
            "sidecar_pid": os.getpid(),
            "execution_device": "cuda" if use_cuda else "cpu",
            **cuda_receipt,
            "inpaint_crop": crop_receipt,
        })
        _save_provenance_png(result, out_path, receipt)
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {out_path!r}: {exc}", code=7)


def _composite_only_mask(original, mask, generated):
    """Keep original RGB pixels outside the explicit binary regeneration mask."""
    if original.size != mask.size:
        raise ValueError("inpaint mask must match the oriented source image raster")
    if generated.size != original.size:
        generated = generated.resize(original.size, resample=Image.Resampling.LANCZOS)
    binary = mask.convert("L").point(lambda value: 255 if value >= 128 else 0)
    result = original.convert("RGB").copy()
    result.paste(generated.convert("RGB"), mask=binary)
    return result


def _begin_cuda_peak_measurement(torch, use_cuda: bool):
    """Reset this process's CUDA allocator peaks after initializing logical 0."""
    if not use_cuda:
        return None
    torch.cuda.init()
    device = torch.cuda.current_device()
    torch.cuda.reset_peak_memory_stats(device)
    return device


def _cuda_peak_receipt(torch, device):
    if device is None:
        return {"gpu_uuid": None, "gpu_peak_allocated_bytes": None, "gpu_peak_reserved_bytes": None}
    return {
        "gpu_uuid": os.environ.get("CUDA_VISIBLE_DEVICES") or None,
        "gpu_logical_device": int(device),
        "gpu_peak_allocated_bytes": int(torch.cuda.max_memory_allocated(device)),
        "gpu_peak_reserved_bytes": int(torch.cuda.max_memory_reserved(device)),
        "gpu_peak_scope": "torch CUDA allocator peak reset before inpaint model load",
    }


def _composite_cropped_inpaint(original, mask, generated, crop_box):
    """Restore a resized crop and apply only the full-resolution source mask."""
    x0, y0, x1, y1 = crop_box
    source_crop = original.crop(crop_box)
    mask_crop = mask.crop(crop_box)
    crop_result = _composite_only_mask(source_crop, mask_crop, generated)
    full = original.convert("RGB").copy()
    full.paste(crop_result, (x0, y0))
    return full


def _prepare_inpaint_crop(
    image, mask, context_pixels: int, max_edge_px: int, *, neutralize_masked_pixels: bool = False
):
    """Crop to the explicit white-mask bounds for inference; return source-space map."""
    if context_pixels < 0 or context_pixels > 512:
        raise ValueError("inpaint crop context_pixels must be between 0 and 512")
    if max_edge_px not in (512, 1024):
        raise ValueError("inpaint crop max_edge_px must be 512 or 1024 (or omitted for model-native)")
    if image.size != mask.size:
        raise ValueError("inpaint mask must match the oriented source image raster")
    binary = mask.convert("L").point(lambda value: 255 if value >= 128 else 0)
    bounds = binary.getbbox()
    if bounds is None:
        raise ValueError("crop_to_mask requires a non-empty white mask")
    x0, y0, x1, y1 = bounds
    x0, y0 = max(0, x0 - context_pixels), max(0, y0 - context_pixels)
    x1, y1 = min(image.width, x1 + context_pixels), min(image.height, y1 + context_pixels)
    source_box = (x0, y0, x1, y1)
    source_crop = image.crop(source_box)
    mask_crop = binary.crop(source_box)
    neutralization_sigma = 0
    if neutralize_masked_pixels:
        source_crop, neutralization_sigma = _neutralize_masked_pixels(source_crop, mask_crop)
    source_w, source_h = source_crop.size
    scale = min(1.0, max_edge_px / max(source_w, source_h))
    infer_w = max(8, int(round(source_w * scale / 8)) * 8)
    infer_h = max(8, int(round(source_h * scale / 8)) * 8)
    if (infer_w, infer_h) != source_crop.size:
        source_crop = source_crop.resize((infer_w, infer_h), resample=Image.Resampling.LANCZOS)
        mask_crop = mask_crop.resize((infer_w, infer_h), resample=Image.Resampling.NEAREST)
    sx, sy = infer_w / (x1 - x0), infer_h / (y1 - y0)
    receipt = {
        "crop_to_mask": True,
        "source_box_xyxy": [x0, y0, x1, y1],
        "source_crop_size": [x1 - x0, y1 - y0],
        "inference_size": [infer_w, infer_h],
        "context_pixels": context_pixels,
        "max_edge_px": max_edge_px,
        "source_to_inference_matrix_row_major": [[sx, 0, -x0 * sx], [0, sy, -y0 * sy], [0, 0, 1]],
        "mask_resampling": "nearest",
        "neutralization_sigma_px": neutralization_sigma,
        "masked_input_neutralization": (
            f"nearest_unmasked_pixel_voronoi_gaussian_sigma_{neutralization_sigma}"
            if neutralize_masked_pixels else "none"
        ),
    }
    return source_crop, mask_crop, source_box, receipt


def _neutralize_masked_pixels(image, mask):
    """Remove object color/shape cues from the pipeline init image.

    Each masked pixel receives its nearest unmasked source pixel color. The
    generated crop is still composited through the original binary mask, so this
    preprocessing cannot alter the retained output region.
    """
    import numpy as np
    from scipy.ndimage import distance_transform_edt, gaussian_filter

    rgb = np.asarray(image.convert("RGB")).copy()
    masked = np.asarray(mask.convert("L")) >= 128
    if rgb.shape[:2] != masked.shape:
        raise ValueError("neutralization mask must match the inference image")
    if not masked.any():
        return image.convert("RGB"), 0
    if masked.all():
        raise ValueError("neutralize_masked_pixels requires unmasked context pixels")
    nearest = distance_transform_edt(masked, return_distances=False, return_indices=True)
    rgb[masked] = rgb[nearest[0][masked], nearest[1][masked]]
    # Voronoi propagation removes the object, but its piecewise-constant sectors
    # create strong synthetic edges that become conditioning cues. Smooth only
    # the filled mask before inference; final output compositing still uses the
    # untouched source and binary mask.
    sigma = max(8.0, min(32.0, max(image.size) * 0.02))
    smooth = gaussian_filter(rgb.astype("float32"), sigma=(sigma, sigma, 0), mode="nearest")
    rgb[masked] = np.clip(smooth[masked], 0, 255).astype("uint8")
    return Image.fromarray(rgb, mode="RGB"), round(sigma, 3)


def _save_provenance_png(image, out_path, receipt):
    info = PngInfo()
    info.add_text("vrooli_inpaint_provenance", json.dumps(receipt, sort_keys=True, separators=(",", ":")))
    image.save(out_path, format="PNG", pnginfo=info)


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

# arch → concrete ControlNet pipeline class per op (Phase 5). When a request
# carries ControlNet conditioning the base pipeline is swapped for its ControlNet
# variant, constructed with ``controlnet=[ControlNetModel...]``.
_CONTROLNET_TXT2IMG = {
    "sd15": "StableDiffusionControlNetPipeline",
    "sdxl": "StableDiffusionXLControlNetPipeline",
}
_CONTROLNET_IMG2IMG = {
    "sd15": "StableDiffusionControlNetImg2ImgPipeline",
    "sdxl": "StableDiffusionXLControlNetImg2ImgPipeline",
}


def _controlnet_class(architecture: str, class_map: dict, op: str) -> str:
    cls = class_map.get(architecture)
    if cls is None:
        _common.fail(
            f"no {op} controlnet pipeline class for architecture {architecture!r}; "
            f"known: {sorted(class_map)}",
            code=4,
        )
    return cls  # type: ignore[return-value]


def _load_controlnet_pipe(diffusers, torch, architecture, model_dir, repo_index, single, class_map, op, controlnets):
    """Construct a ControlNet pipeline for the base checkpoint with the given
    ControlNetModel(s) attached. Mirrors _load_generate_pipe's repo/single-file
    branch but uses the concrete ControlNet class (the AutoPipeline does not take a
    controlnet= kwarg)."""
    use_cuda = _common.use_cuda(torch)
    dtype = _resolve_dtype(torch, "float16", use_cuda)
    # diffusers treats a list as a MultiControlNet (which then requires list-typed
    # image + conditioning_scale at call time); a lone ControlNet must be passed as
    # the bare model so the single-image call path (cn_images[0]) matches. Mirror
    # the >1-vs-1 shape the call-site uses for image/controlnet_conditioning_scale.
    controlnet = controlnets[0] if len(controlnets) == 1 else controlnets
    load_kwargs = {"torch_dtype": dtype, "controlnet": controlnet}
    if architecture == "sd15":
        load_kwargs["safety_checker"] = None
    cls = getattr(diffusers, _controlnet_class(architecture, class_map, op))
    try:
        if repo_index:
            pipe = cls.from_pretrained(model_dir, **load_kwargs)
        else:
            pipe = cls.from_single_file(single, **load_kwargs)
    except Exception as exc:  # noqa: BLE001
        _common.fail(
            f"failed to load {op} controlnet pipeline for arch {architecture!r} from {model_dir!r}: {exc}",
            code=5,
        )
        raise
    return pipe


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
    use_cuda = _common.use_cuda(torch)
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
    controlnet_specs: Optional[Sequence[str]] = None,
    ip_adapter_specs: Optional[Sequence[str]] = None,
) -> None:
    """Load a base checkpoint into its architecture's text2image pipeline and
    generate one image from the prompt, writing a PNG. FAILS LOUD on a missing
    backend or load failure — never a fabricated success. The conditioning stack is
    applied in canonical order: lora_specs (fused), controlnet_specs (structural
    control image), ip_adapter_specs (reference image) — all before placement."""
    diffusers, torch, _Image = _import_diffusers_backend()
    single = _find_single_file(model_dir) if os.path.isdir(model_dir) else None
    repo_index = os.path.isfile(os.path.join(model_dir, "model_index.json"))
    if not repo_index and single is None:
        _common.fail(
            f"model dir {model_dir!r} has neither model_index.json (repo) nor a "
            "single-file .safetensors/.ckpt checkpoint",
            code=5,
        )
    use_cuda = _common.use_cuda(torch)
    dtype = _resolve_dtype(torch, "float16", use_cuda)
    cn_models, cn_images, cn_scales = _adapters.load_controlnets(diffusers, torch, dtype, controlnet_specs or [])
    if cn_models:
        pipe = _load_controlnet_pipe(
            diffusers, torch, architecture, model_dir, repo_index, single,
            _CONTROLNET_TXT2IMG, "text_to_image", cn_models,
        )
    else:
        pipe = _load_generate_pipe(
            diffusers, torch, architecture, model_dir, repo_index, single,
            "AutoPipelineForText2Image", _TXT2IMG_SINGLE_FILE, "text_to_image",
        )
    _adapters.apply_loras(pipe, lora_specs or [])
    ip_image = _adapters.apply_ip_adapter(pipe, ip_adapter_specs or [])
    pipe = _place_pipe(pipe, torch, use_cuda, offload=True)

    kwargs = build_txt2img_kwargs(params)
    if cn_models:
        kwargs["image"] = cn_images if len(cn_images) > 1 else cn_images[0]
        kwargs["controlnet_conditioning_scale"] = cn_scales if len(cn_scales) > 1 else cn_scales[0]
    if ip_image is not None:
        kwargs["ip_adapter_image"] = ip_image
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
    controlnet_specs: Optional[Sequence[str]] = None,
    ip_adapter_specs: Optional[Sequence[str]] = None,
) -> None:
    """Load a base checkpoint into its architecture's img2img pipeline and
    transform the init image guided by the prompt, writing a PNG. FAILS LOUD on a
    missing backend or load failure — never a fabricated success. The conditioning
    stack (lora/controlnet/ip-adapter) is applied before placement; for ControlNet
    img2img the init rides ``image`` and the control map rides ``control_image``."""
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
    use_cuda = _common.use_cuda(torch)
    dtype = _resolve_dtype(torch, "float16", use_cuda)
    cn_models, cn_images, cn_scales = _adapters.load_controlnets(diffusers, torch, dtype, controlnet_specs or [])
    if cn_models:
        pipe = _load_controlnet_pipe(
            diffusers, torch, architecture, model_dir, repo_index, single,
            _CONTROLNET_IMG2IMG, "image_to_image", cn_models,
        )
    else:
        pipe = _load_generate_pipe(
            diffusers, torch, architecture, model_dir, repo_index, single,
            "AutoPipelineForImage2Image", _IMG2IMG_SINGLE_FILE, "image_to_image",
        )
    _adapters.apply_loras(pipe, lora_specs or [])
    ip_image = _adapters.apply_ip_adapter(pipe, ip_adapter_specs or [])
    pipe = _place_pipe(pipe, torch, use_cuda, offload=True)

    kwargs = build_img2img_kwargs(params)
    kwargs["image"] = init
    if cn_models:
        kwargs["control_image"] = cn_images if len(cn_images) > 1 else cn_images[0]
        kwargs["controlnet_conditioning_scale"] = cn_scales if len(cn_scales) > 1 else cn_scales[0]
    if ip_image is not None:
        kwargs["ip_adapter_image"] = ip_image
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
