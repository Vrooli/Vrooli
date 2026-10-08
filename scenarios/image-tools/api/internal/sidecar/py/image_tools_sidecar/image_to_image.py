"""image_to_image — base-checkpoint transform via the generic diffusers runner
(_diffusers.run_img2img).

This is the executable side of a base text2image checkpoint's architecture-derived
img2img transform on the diffusers backend, for diffusers-REPO models that
stable-diffusion.cpp cannot load (a single-file checkpoint takes the sd.cpp
sd-img2img path instead). ``--strength`` controls how far the output may drift
from the init image; the concrete pipeline class is resolved by ``--architecture``.

Thin CLI: argv → _diffusers.GenerateParams → _diffusers.run_img2img.
"""

from __future__ import annotations

import argparse

from . import _common, _diffusers


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="diffusers base-checkpoint img2img transform")
    p.add_argument("--model", required=True, help="path to the diffusers model dir or single-file checkpoint")
    p.add_argument("--architecture", required=True, help="model weight-lineage architecture (sd15|sdxl)")
    p.add_argument("--image", required=True, help="init image path")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--prompt", required=True, help="prompt steering the transform")
    p.add_argument("--negative-prompt", dest="negative_prompt", default="", help="optional negative prompt")
    p.add_argument("--strength", type=float, default=None, help="how far from the init image (0..1)")
    p.add_argument("--steps", type=int, default=0, help="sampler steps (0 = default)")
    p.add_argument("--guidance", type=float, default=None, help="text guidance scale (cfg_scale)")
    p.add_argument("--seed", type=int, default=0, help="RNG seed (0 = nondeterministic)")
    p.add_argument("--lora", action="append", default=[], help="conditioning LoRA as <path>:<scale> (repeatable)")
    p.add_argument("--controlnet", action="append", default=[], help="ControlNet as <dir>:<scale>:<image> (repeatable)")
    p.add_argument("--ip-adapter", dest="ip_adapter", action="append", default=[], help="IP-Adapter as <weightfile>:<scale>:<reference> (repeatable)")
    return p.parse_args()


def main() -> None:
    args = _parse_args()
    if not args.prompt.strip():
        _common.fail("image_to_image requires a non-empty --prompt", code=2)

    params = _diffusers.GenerateParams(
        prompt=args.prompt,
        negative_prompt=args.negative_prompt or None,
        strength=args.strength,
        steps=args.steps,
        guidance=args.guidance,
        seed=args.seed,
    )
    _diffusers.run_img2img(
        architecture=args.architecture,
        model_dir=args.model,
        image_path=args.image,
        out_path=args.out,
        params=params,
        lora_specs=args.lora,
        controlnet_specs=args.controlnet,
        ip_adapter_specs=args.ip_adapter,
    )


if __name__ == "__main__":
    main()
