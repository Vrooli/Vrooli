"""Pure (torch-free) unit checks for the diffusers family adapter table.

Run standalone: ``python3 -m image_tools_sidecar.test_diffusers_adapters`` (exits
non-zero with a message on any mismatch). The Go test TestDiffusersSidecarContract
executes this inside ``go test`` so the param→kwarg mapping is covered in CI
without a separate pytest harness. It asserts ONLY the declared call contract —
no model is loaded — mirroring the Go arg-builder discipline.
"""

from __future__ import annotations

from . import _diffusers
from ._diffusers import Params, build_call_kwargs


def _eq(got, want, ctx):
    if got != want:
        raise AssertionError(f"{ctx}: got {got!r}, want {want!r}")


def test_ip2p_defaults():
    k = build_call_kwargs("instruct-pix2pix", Params(prompt="make it winter"))
    _eq(k["prompt"], "make it winter", "ip2p prompt")
    _eq(k["num_inference_steps"], 20, "ip2p default steps")
    _eq(k["guidance_scale"], 7.5, "ip2p default guidance")
    _eq(k["image_guidance_scale"], 1.5, "ip2p default image_guidance")
    assert "true_cfg_scale" not in k, "ip2p must not emit true_cfg_scale"


def test_ip2p_overrides():
    k = build_call_kwargs(
        "instruct-pix2pix",
        Params(prompt="p", steps=30, guidance=5.0, image_guidance=2.0),
    )
    _eq(k["num_inference_steps"], 30, "ip2p steps override")
    _eq(k["guidance_scale"], 5.0, "ip2p guidance override")
    _eq(k["image_guidance_scale"], 2.0, "ip2p image_guidance override")


def test_qwen_defaults():
    k = build_call_kwargs("qwen-image-edit-plus", Params(prompt="add a hat"))
    _eq(k["prompt"], "add a hat", "qwen prompt")
    _eq(k["num_inference_steps"], 40, "qwen default steps")
    _eq(k["true_cfg_scale"], 4.0, "qwen default true_cfg_scale")
    _eq(k["negative_prompt"], " ", "qwen default negative_prompt")
    _eq(k["guidance_scale"], 1.0, "qwen fixed guidance_scale")
    _eq(k["num_images_per_prompt"], 1, "qwen num_images_per_prompt")
    assert "image_guidance_scale" not in k, "qwen must not emit image_guidance_scale"


def test_qwen_overrides():
    k = build_call_kwargs(
        "qwen-image-edit-plus",
        Params(prompt="p", steps=8, guidance=2.0, image_guidance=3.0, negative_prompt="blurry"),
    )
    _eq(k["num_inference_steps"], 8, "qwen steps override")
    _eq(k["true_cfg_scale"], 3.0, "qwen image_guidance→true_cfg_scale override")
    _eq(k["guidance_scale"], 2.0, "qwen guidance override")
    _eq(k["negative_prompt"], "blurry", "qwen negative_prompt override")


def test_unproven_family_refuses():
    raised = False
    try:
        build_call_kwargs("flux-2-klein", Params(prompt="p"))
    except RuntimeError:
        raised = True
    assert raised, "unproven family build must raise rather than guess kwargs"


def test_unknown_family_exits():
    raised = False
    try:
        _diffusers.adapter_for("nope-not-a-family")
    except SystemExit:
        raised = True
    assert raised, "unknown family must fail loud (SystemExit), not return None"


def main() -> int:
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
    print(f"ok - {len(tests)} diffusers adapter checks passed")
    return 0


if __name__ == "__main__":
    import sys

    sys.exit(main())
