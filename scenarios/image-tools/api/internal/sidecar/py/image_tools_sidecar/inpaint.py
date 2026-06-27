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

from . import _common, _diffusers


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="diffusers base-checkpoint inpaint (masked regenerate)")
    p.add_argument("--model", required=True, help="path to the diffusers model dir or single-file checkpoint")
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
    return p.parse_args()


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
    _diffusers.run_inpaint(
        architecture=args.architecture,
        model_dir=args.model,
        image_path=args.image,
        mask_path=args.mask,
        out_path=args.out,
        params=params,
    )


if __name__ == "__main__":
    main()
