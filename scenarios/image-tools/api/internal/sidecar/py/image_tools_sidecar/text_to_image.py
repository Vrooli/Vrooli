"""text_to_image — base-checkpoint generate via the generic diffusers runner
(_diffusers.run_txt2img).

This is the executable side of a *base* text2image checkpoint's NATIVE generate
op on the diffusers backend. It exists for diffusers-REPO models (sharded
unet/vae/text_encoder weights) that stable-diffusion.cpp cannot load; a
single-file checkpoint takes the sd.cpp path instead. The concrete pipeline class
is resolved by ``--architecture`` (the registry's Model.architecture), not
hardcoded, so every base architecture generates through one runner.

Thin CLI: argv → _diffusers.GenerateParams → _diffusers.run_txt2img.
"""

from __future__ import annotations

import argparse

from . import _common, _diffusers


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="diffusers base-checkpoint text2image generate")
    p.add_argument("--model", required=True, help="path to the diffusers model dir or single-file checkpoint")
    p.add_argument("--architecture", required=True, help="model weight-lineage architecture (sd15|sdxl)")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--prompt", required=True, help="text prompt")
    p.add_argument("--negative-prompt", dest="negative_prompt", default="", help="optional negative prompt")
    p.add_argument("--width", type=int, default=0, help="output width (0 = pipeline default)")
    p.add_argument("--height", type=int, default=0, help="output height (0 = pipeline default)")
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
        _common.fail("text_to_image requires a non-empty --prompt", code=2)

    params = _diffusers.GenerateParams(
        prompt=args.prompt,
        negative_prompt=args.negative_prompt or None,
        steps=args.steps,
        guidance=args.guidance,
        width=args.width,
        height=args.height,
        seed=args.seed,
    )
    _diffusers.run_txt2img(
        architecture=args.architecture,
        model_dir=args.model,
        out_path=args.out,
        params=params,
        lora_specs=args.lora,
        controlnet_specs=args.controlnet,
        ip_adapter_specs=args.ip_adapter,
    )


if __name__ == "__main__":
    main()
