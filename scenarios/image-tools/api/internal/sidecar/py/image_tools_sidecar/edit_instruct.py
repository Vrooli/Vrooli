"""edit_instruct — natural-language, identity-preserving image editing via the
generic diffusers runner (_diffusers.py).

This module is now a thin CLI: it normalizes argv into _diffusers.Params and
hands off to the family adapter named by ``--family`` (instruct-pix2pix,
qwen-image-edit-plus, …). The pipeline class + call contract are DECLARED per
family in _diffusers.py (mirrored from the Go registry), not hardcoded here — so
the diffusers backend supports every registered edit family, not just one.

The instruction rides ``--prompt`` ("make it winter", "add sunglasses");
``--guidance`` is the text-guidance scale and ``--image-guidance`` is the
identity-preservation / edit-strength knob (its exact meaning is family-specific:
image_guidance_scale for InstructPix2Pix, true_cfg_scale for Qwen-Image-Edit).
"""

from __future__ import annotations

import argparse

from . import _common, _diffusers


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="diffusers instruction-edit (identity-preserving)")
    p.add_argument("--model", required=True, help="path to the diffusers model dir or checkpoint")
    p.add_argument("--family", required=True, help="registered diffusers family adapter (runtime.family)")
    p.add_argument("--image", required=True, action="append", help="input image path (repeatable for multi-reference families)")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--prompt", required=True, help="the natural-language edit instruction")
    p.add_argument("--negative-prompt", dest="negative_prompt", default="", help="optional negative prompt")
    p.add_argument("--steps", type=int, default=0, help="sampler steps (0 = family default)")
    p.add_argument("--guidance", type=float, default=None, help="text guidance scale")
    p.add_argument(
        "--image-guidance",
        dest="image_guidance",
        type=float,
        default=None,
        help="image-guidance / edit-strength scale (higher = stay closer to the source)",
    )
    p.add_argument("--seed", type=int, default=0, help="RNG seed (0 = nondeterministic)")
    return p.parse_args()


def main() -> None:
    args = _parse_args()

    if not args.prompt.strip():
        _common.fail("edit_instruct requires a non-empty --prompt (the edit instruction)", code=2)

    params = _diffusers.Params(
        prompt=args.prompt,
        steps=args.steps,
        guidance=args.guidance,
        image_guidance=args.image_guidance,
        negative_prompt=args.negative_prompt or None,
        seed=args.seed,
    )
    _diffusers.run(
        family=args.family,
        model_dir=args.model,
        image_paths=args.image,
        out_path=args.out,
        params=params,
    )


if __name__ == "__main__":
    main()
