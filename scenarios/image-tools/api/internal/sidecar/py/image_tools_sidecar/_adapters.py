"""_adapters — conditioning-adapter application for the diffusers runner.

Phase 4 ships LoRA application; ControlNet / IP-Adapter land in later phases. The
Go arg-builders (internal/technique) emit repeatable ``--lora <path>:<scale>``
flags from the resolved conditioning stack; this module parses them and fuses the
LoRA weights onto a constructed diffusers pipeline.

Two seams keep this honest (mirroring _diffusers):
  * ``parse_lora_spec`` / ``parse_lora_specs`` are PURE (no torch) — the
    ``<path>:<scale>`` wire contract is unit-testable on any host and the Go
    parity test (TestLoRASpecParityPython) asserts the Go emitter and this parser
    agree.
  * ``apply_loras`` needs the heavy pipeline; it is exercised with a fake pipe in
    the torch-free contract test (it only drives ``load_lora_weights`` /
    ``set_adapters``, both recorded by the fake), and for real on the attended GPU
    e2e (plan Phase 4) before the LoRA adapters flip Ready.
"""

from __future__ import annotations

from typing import Any, List, NamedTuple, Sequence


class LoRASpec(NamedTuple):
    path: str
    scale: float


def parse_lora_spec(spec: str) -> LoRASpec:
    """Parse one ``<path>:<scale>`` LoRA spec. The scale is the trailing
    colon-delimited field WHEN it parses as a float; otherwise the whole string is
    the path and the scale defaults to 1.0 (so a bare path, or a path that happens
    to contain a colon with a non-numeric tail, is handled correctly). Mirrors the
    Go emitter in internal/technique (LoRAArgs)."""
    s = spec.strip()
    idx = s.rfind(":")
    if idx > 0:
        tail = s[idx + 1 :]
        try:
            scale = float(tail)
        except ValueError:
            return LoRASpec(path=s, scale=1.0)
        return LoRASpec(path=s[:idx], scale=scale)
    return LoRASpec(path=s, scale=1.0)


def parse_lora_specs(specs: Sequence[str]) -> List[LoRASpec]:
    """Parse every non-empty ``--lora`` spec, preserving order (the Go resolver
    already ordered the stack LoRA→ControlNet→IP-Adapter)."""
    return [parse_lora_spec(s) for s in specs if s and s.strip()]


def apply_loras(pipe: Any, specs: Sequence[str]) -> List[str]:
    """Load + activate each LoRA on a constructed diffusers pipeline with its
    per-adapter scale, stacking multiple. Returns the adapter names registered (for
    diagnostics). A no-op for an empty spec list. Must be called BEFORE any CPU
    offload is enabled (offload moves weights off the GPU, breaking late fusion)."""
    parsed = parse_lora_specs(specs)
    if not parsed:
        return []
    names: List[str] = []
    weights: List[float] = []
    for i, sp in enumerate(parsed):
        name = f"lora_{i}"
        pipe.load_lora_weights(sp.path, adapter_name=name)
        names.append(name)
        weights.append(sp.scale)
    # set_adapters activates the stack with per-adapter weights; fuse_lora bakes
    # them into the UNet so a single forward pass reflects all of them.
    pipe.set_adapters(names, adapter_weights=weights)
    return names


# =============================================================================
# ControlNet (Phase 5). A ControlNet conditions generation on a preprocessed
# control image (canny edges / depth / pose / segmentation). The Go arg-builder
# (internal/technique.ControlNetArgs) emits repeatable
# ``--controlnet <dir>:<scale>:<image>`` flags; the trailing scale + image are
# colon-free local paths so the spec splits cleanly off the right. The conditioning
# image is ALREADY the final control map (the auto-preprocess path runs the
# adapter's preprocessor op as a Look step first); this module never re-preprocesses.
# =============================================================================


class ControlNetSpec(NamedTuple):
    path: str  # the ControlNet diffusers repo dir (ControlNetModel.from_pretrained)
    scale: float  # controlnet_conditioning_scale
    image: str  # path to the (already preprocessed) control image


def parse_controlnet_spec(spec: str) -> ControlNetSpec:
    """Parse one ``<dir>:<scale>:<image>`` ControlNet spec. The scale and image are
    the trailing two colon-delimited fields (both colon-free); the remainder is the
    repo dir (so a dir that itself contained a colon would be preserved). Mirrors the
    Go emitter in internal/technique (ControlNetArgs)."""
    s = spec.strip()
    parts = s.rsplit(":", 2)
    if len(parts) != 3:
        raise ValueError(f"malformed controlnet spec {spec!r}; want <dir>:<scale>:<image>")
    path, scale_s, image = parts
    try:
        scale = float(scale_s)
    except ValueError as exc:
        raise ValueError(f"controlnet spec {spec!r} has a non-numeric scale {scale_s!r}") from exc
    return ControlNetSpec(path=path, scale=scale, image=image)


def parse_controlnet_specs(specs: Sequence[str]) -> List[ControlNetSpec]:
    """Parse every non-empty ``--controlnet`` spec, preserving order."""
    return [parse_controlnet_spec(s) for s in specs if s and s.strip()]


def load_controlnets(diffusers: Any, torch: Any, dtype: Any, specs: Sequence[str]):
    """Load each ControlNet model + open its control image. Returns
    (models, images, scales) parallel lists ready to hand a ControlNet pipeline
    (``controlnet=models`` at construction; ``image=images`` +
    ``controlnet_conditioning_scale=scales`` at call). Empty for no specs. Needs the
    heavy backend (exercised on the attended GPU e2e before ControlNet flips Ready);
    the parsing seam above is the unit-tested contract."""
    parsed = parse_controlnet_specs(specs)
    if not parsed:
        return [], [], []
    from PIL import Image  # noqa: WPS433

    models = []
    images = []
    scales = []
    for sp in parsed:
        models.append(diffusers.ControlNetModel.from_pretrained(sp.path, torch_dtype=dtype))
        images.append(Image.open(sp.image).convert("RGB"))
        scales.append(sp.scale)
    return models, images, scales


# =============================================================================
# IP-Adapter (Phase 6). An IP-Adapter conditions generation on a reference image
# (identity / style transfer). The Go arg-builder
# (internal/technique.IPAdapterArgs) emits repeatable
# ``--ip-adapter <weightfile>:<scale>:<reference>`` flags. diffusers loads a single
# IP-Adapter per pipeline; the first spec wins and any extra is reported.
# =============================================================================


class IPAdapterSpec(NamedTuple):
    path: str  # the IP-Adapter weight file (.safetensors)
    scale: float  # set_ip_adapter_scale
    image: str  # path to the reference image (the adapter's "prompt")


def parse_ip_adapter_spec(spec: str) -> IPAdapterSpec:
    """Parse one ``<weightfile>:<scale>:<reference>`` IP-Adapter spec (same
    right-split discipline as ControlNet). Mirrors internal/technique.IPAdapterArgs."""
    s = spec.strip()
    parts = s.rsplit(":", 2)
    if len(parts) != 3:
        raise ValueError(f"malformed ip-adapter spec {spec!r}; want <weightfile>:<scale>:<reference>")
    path, scale_s, image = parts
    try:
        scale = float(scale_s)
    except ValueError as exc:
        raise ValueError(f"ip-adapter spec {spec!r} has a non-numeric scale {scale_s!r}") from exc
    return IPAdapterSpec(path=path, scale=scale, image=image)


def parse_ip_adapter_specs(specs: Sequence[str]) -> List[IPAdapterSpec]:
    """Parse every non-empty ``--ip-adapter`` spec, preserving order."""
    return [parse_ip_adapter_spec(s) for s in specs if s and s.strip()]


def apply_ip_adapter(pipe: Any, specs: Sequence[str]):
    """Load + scale the IP-Adapter on a constructed pipeline and return the opened
    reference image (passed as ``ip_adapter_image`` at call time), or None for no
    specs. diffusers supports one IP-Adapter per pipeline; a second spec is an
    error rather than a silent drop. Must run before CPU offload."""
    parsed = parse_ip_adapter_specs(specs)
    if not parsed:
        return None
    if len(parsed) > 1:
        raise ValueError("only one ip-adapter is supported per generation")
    from PIL import Image  # noqa: WPS433

    sp = parsed[0]
    # The weight file lives at <dir>/<name>.safetensors; load_ip_adapter takes the
    # directory, a subfolder, and the weight filename.
    import os  # noqa: WPS433

    weight_dir = os.path.dirname(sp.path)
    weight_name = os.path.basename(sp.path)
    pipe.load_ip_adapter(weight_dir, subfolder="", weight_name=weight_name)
    pipe.set_ip_adapter_scale(sp.scale)
    return Image.open(sp.image).convert("RGB")
