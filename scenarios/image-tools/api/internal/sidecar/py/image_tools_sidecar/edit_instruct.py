"""edit_instruct — natural-language, identity-preserving image editing via the
diffusers instruction-edit pipeline class (InstructPix2Pix / Qwen-Image-Edit).

Unlike bg_removal/denoise (CPU-tractable onnxruntime), this op is heavy: it
needs torch + diffusers and a downloaded diffusers model directory, and a GPU is
strongly recommended (the CPU default, InstructPix2Pix, is viable but slow). It
follows the project's no-vaporware rule by being REAL diffusers code that runs
wherever torch+diffusers are provisioned — and by FAILING LOUD with an
actionable message (never a fake success) when they are not, exactly like the
gated standalone backends. See docs/reference/backends.md for provisioning.

The instruction rides `--prompt` ("make it winter", "add sunglasses");
`--guidance` is the text-guidance scale and `--image-guidance` controls how
faithful the result stays to the source (higher = closer to the original, the
identity-preservation knob).
"""

from __future__ import annotations

import argparse

from . import _common


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="diffusers instruction-edit (identity-preserving)")
    p.add_argument("--model", required=True, help="path to the diffusers model directory")
    p.add_argument("--image", required=True, help="input image path")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--prompt", required=True, help="the natural-language edit instruction")
    p.add_argument("--steps", type=int, default=20, help="sampler steps")
    p.add_argument("--guidance", type=float, default=7.5, help="text guidance scale")
    p.add_argument(
        "--image-guidance",
        dest="image_guidance",
        type=float,
        default=1.5,
        help="image guidance scale (higher = stay closer to the source / preserve identity)",
    )
    p.add_argument("--seed", type=int, default=0, help="RNG seed (0 = nondeterministic)")
    return p.parse_args()


def main() -> None:
    args = _parse_args()

    if not args.prompt.strip():
        _common.fail("edit_instruct requires a non-empty --prompt (the edit instruction)", code=2)

    try:
        import torch  # noqa: WPS433
        from diffusers import StableDiffusionInstructPix2PixPipeline  # noqa: WPS433
        from PIL import Image  # noqa: WPS433
    except ImportError as exc:  # pragma: no cover - exercised on un-provisioned hosts
        _common.fail(
            f"missing instruction-edit dependency ({exc}). Provision the heavy backend: "
            "pip install torch diffusers transformers accelerate pillow  "
            "(GPU strongly recommended; see docs/reference/backends.md)",
            code=3,
        )
        raise

    try:
        image = Image.open(args.image).convert("RGB")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open input image {args.image!r}: {exc}", code=6)

    use_cuda = torch.cuda.is_available()
    dtype = torch.float16 if use_cuda else torch.float32
    try:
        pipe = StableDiffusionInstructPix2PixPipeline.from_pretrained(
            args.model,
            torch_dtype=dtype,
            safety_checker=None,
        )
        pipe = pipe.to("cuda" if use_cuda else "cpu")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to load instruction-edit model {args.model!r}: {exc}", code=5)
        raise

    generator = None
    if args.seed:
        generator = torch.Generator(device="cuda" if use_cuda else "cpu").manual_seed(args.seed)

    try:
        result = pipe(
            args.prompt,
            image=image,
            num_inference_steps=args.steps,
            guidance_scale=args.guidance,
            image_guidance_scale=args.image_guidance,
            generator=generator,
        ).images[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"instruction edit failed: {exc}", code=8)
        raise

    try:
        result.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)


if __name__ == "__main__":
    main()
