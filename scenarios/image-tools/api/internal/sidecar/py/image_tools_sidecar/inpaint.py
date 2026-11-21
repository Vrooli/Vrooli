"""inpaint — masked-regenerate of a base checkpoint via the generic diffusers
runner (_diffusers.run_inpaint).

This is the executable side of the *derived* inpaint capability: any base
text2image checkpoint (architecture sd15/sdxl) is loaded into the standard inpaint
pipeline for its architecture and run as a masked regenerate. White mask pixels are
repainted, black pixels are kept; ``--prompt`` steers the synthesized region.

It is a thin CLI: argv → _diffusers.TransformParams → _diffusers.run_inpaint. The
concrete pipeline class is resolved by ``--architecture`` (the registry's
Model.architecture), not hardcoded, so the diffusers backend inpaints with every
base architecture the derivation table proves. Offerability is gated upstream in Go
(the architecture→technique Ready flag); a dedicated *-inpainting model blends
masked edges more cleanly — the derived quality caveat.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import time

from . import _common, _diffusers


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="diffusers base-checkpoint inpaint (masked regenerate)")
    p.add_argument("--model", required=True, help="path to the diffusers model dir or single-file checkpoint")
    p.add_argument("--model-id", default="", help="catalog model identifier")
    p.add_argument("--model-sha256", default="", help="pinned catalog checkpoint digest")
    p.add_argument("--model-revision", default="", help="pinned model repository revision")
    p.add_argument("--variant", default="", help="Diffusers weight variant for repository models")
    p.add_argument("--crop-to-mask", action="store_true", help="crop inference to the explicit mask bounds plus context")
    p.add_argument("--crop-context-pixels", type=int, default=0, help="source-pixel context around the mask crop")
    p.add_argument("--crop-max-edge-px", type=int, default=0, help="maximum inference crop edge (512 or 1024)")
    p.add_argument("--neutralize-masked-pixels", action="store_true", help="replace masked init-image pixels with nearest unmasked context before inference")
    p.add_argument("--architecture", required=True, help="model weight-lineage architecture (sd15|sdxl)")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--mask", required=True, help="mask image path (white=repaint, black=keep)")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--prompt", required=True, help="prompt steering the repainted region")
    p.add_argument("--negative-prompt", dest="negative_prompt", default="", help="optional negative prompt")
    p.add_argument("--strength", type=float, default=None, help="how much the masked region may change (0..1)")
    p.add_argument("--steps", type=int, default=0, help="sampler steps (0 = default)")
    p.add_argument("--guidance", type=float, default=None, help="text guidance scale (cfg_scale)")
    p.add_argument("--seed", type=int, default=0, help="RNG seed (0 = nondeterministic)")
    p.add_argument("--lora", action="append", default=[], help="conditioning LoRA as <path>:<scale> (repeatable)")
    return _common.parse_device_args(p)


def main() -> None:
    args = _parse_args()

    if not args.prompt.strip():
        _common.fail("inpaint requires a non-empty --prompt", code=2)

    params = _diffusers.TransformParams(
        prompt=args.prompt,
        negative_prompt=args.negative_prompt or None,
        strength=args.strength,
        steps=args.steps,
        guidance=args.guidance,
        seed=args.seed,
    )
    started = time.perf_counter()
    provenance = {
        "schema": "vrooli.image-tools.inpaint/v1",
        "model_id": args.model_id,
        "model_revision": args.model_revision,
        "weight_variant": args.variant,
        "architecture": args.architecture,
        "model_sha256": args.model_sha256 or _model_sha256(args.model),
        "source_sha256": _sha256(args.image),
        "mask_sha256": _sha256(args.mask),
        "mask_semantics": "white pixels regenerated; source pixels outside mask are preserved",
        "prompt": params.prompt,
        "negative_prompt": params.negative_prompt,
        "seed": params.seed,
        "steps": params.steps,
        "guidance": params.guidance,
        "strength": params.strength,
    }
    _diffusers.run_inpaint(
        architecture=args.architecture,
        model_dir=args.model,
        image_path=args.image,
        mask_path=args.mask,
        out_path=args.out,
        params=params,
        lora_specs=args.lora,
        provenance=provenance,
        started=started,
        variant=args.variant,
        crop_to_mask=args.crop_to_mask,
        crop_context_pixels=args.crop_context_pixels,
        crop_max_edge_px=args.crop_max_edge_px,
        neutralize_masked_pixels=args.neutralize_masked_pixels,
    )


def _sha256(path):
    digest = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _model_sha256(path):
    if os.path.isfile(path):
        return _sha256(path)
    candidates = [os.path.join(path, name) for name in os.listdir(path)
                  if name.lower().endswith((".ckpt", ".safetensors"))]
    if len(candidates) == 1:
        return _sha256(candidates[0])
    return ""


if __name__ == "__main__":
    main()
