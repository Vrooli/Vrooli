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
