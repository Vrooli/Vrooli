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


def test_inpaint_kwargs_defaults():
    from ._diffusers import TransformParams, build_inpaint_kwargs

    k = build_inpaint_kwargs(TransformParams(prompt="a clear blue sky"))
    _eq(k["prompt"], "a clear blue sky", "inpaint prompt")
    _eq(k["num_inference_steps"], 30, "inpaint default steps")
    _eq(k["guidance_scale"], 7.5, "inpaint default guidance")
    _eq(k["strength"], 0.85, "inpaint default strength")
    assert "negative_prompt" not in k, "inpaint omits negative_prompt when empty"


def test_inpaint_kwargs_overrides():
    from ._diffusers import TransformParams, build_inpaint_kwargs

    k = build_inpaint_kwargs(
        TransformParams(prompt="p", negative_prompt="blurry", strength=0.6, steps=28, guidance=6.0)
    )
    _eq(k["num_inference_steps"], 28, "inpaint steps override")
    _eq(k["guidance_scale"], 6.0, "inpaint guidance override")
    _eq(k["strength"], 0.6, "inpaint strength override")
    _eq(k["negative_prompt"], "blurry", "inpaint negative_prompt override")


def test_inpaint_class_known_archs():
    _eq(_diffusers._inpaint_class("sd15"), "StableDiffusionInpaintPipeline", "sd15 inpaint class")
    _eq(_diffusers._inpaint_class("sdxl"), "StableDiffusionXLInpaintPipeline", "sdxl inpaint class")


def test_inpaint_class_unknown_arch_exits():
    raised = False
    try:
        _diffusers._inpaint_class("not-an-arch")
    except SystemExit:
        raised = True
    assert raised, "unknown architecture must fail loud (SystemExit)"


def test_txt2img_kwargs_defaults():
    from ._diffusers import GenerateParams, build_txt2img_kwargs

    k = build_txt2img_kwargs(GenerateParams(prompt="a castle"))
    _eq(k["prompt"], "a castle", "txt2img prompt")
    _eq(k["num_inference_steps"], 30, "txt2img default steps")
    _eq(k["guidance_scale"], 7.5, "txt2img default guidance")
    assert "width" not in k and "height" not in k, "txt2img omits size when unset"
    assert "negative_prompt" not in k, "txt2img omits negative_prompt when empty"


def test_txt2img_kwargs_overrides_and_size_rounding():
    from ._diffusers import GenerateParams, build_txt2img_kwargs

    k = build_txt2img_kwargs(
        GenerateParams(prompt="p", negative_prompt="blurry", steps=24, guidance=6.0, width=515, height=768)
    )
    _eq(k["num_inference_steps"], 24, "txt2img steps override")
    _eq(k["guidance_scale"], 6.0, "txt2img guidance override")
    _eq(k["negative_prompt"], "blurry", "txt2img negative_prompt override")
    _eq(k["width"], 512, "txt2img width rounded to multiple of 8")
    _eq(k["height"], 768, "txt2img height unchanged when already a multiple of 8")


def test_img2img_kwargs_defaults_and_strength():
    from ._diffusers import GenerateParams, build_img2img_kwargs

    k = build_img2img_kwargs(GenerateParams(prompt="make it autumn"))
    _eq(k["prompt"], "make it autumn", "img2img prompt")
    _eq(k["num_inference_steps"], 30, "img2img default steps")
    _eq(k["strength"], 0.7, "img2img default strength")
    assert "width" not in k and "height" not in k, "img2img takes size from the init image"

    k2 = build_img2img_kwargs(GenerateParams(prompt="p", strength=0.4, steps=20))
    _eq(k2["strength"], 0.4, "img2img strength override")
    _eq(k2["num_inference_steps"], 20, "img2img steps override")


def test_generate_class_known_archs():
    _eq(_diffusers._generate_class("sdxl", _diffusers._TXT2IMG_SINGLE_FILE, "text_to_image"), "StableDiffusionXLPipeline", "sdxl txt2img class")
    _eq(_diffusers._generate_class("sd15", _diffusers._IMG2IMG_SINGLE_FILE, "image_to_image"), "StableDiffusionImg2ImgPipeline", "sd15 img2img class")


def test_parse_lora_spec():
    from ._adapters import parse_lora_spec

    s = parse_lora_spec("/models/adapters/lcm/lcm.safetensors:0.8")
    _eq(s.path, "/models/adapters/lcm/lcm.safetensors", "lora spec path")
    _eq(s.scale, 0.8, "lora spec scale")

    # A bare path (no scale) defaults to 1.0.
    bare = parse_lora_spec("/x/y.safetensors")
    _eq(bare.path, "/x/y.safetensors", "bare lora path")
    _eq(bare.scale, 1.0, "bare lora default scale")

    # A path whose trailing colon-field is non-numeric stays part of the path.
    weird = parse_lora_spec("/x/y:name.safetensors")
    _eq(weird.path, "/x/y:name.safetensors", "non-numeric tail stays in path")
    _eq(weird.scale, 1.0, "non-numeric tail → default scale")


def test_parse_lora_specs_skips_blanks():
    from ._adapters import parse_lora_specs

    specs = parse_lora_specs(["/a.safetensors:1", "", "  ", "/b.safetensors:0.5"])
    _eq(len(specs), 2, "blank specs skipped")
    _eq(specs[0].scale, 1.0, "first scale")
    _eq(specs[1].scale, 0.5, "second scale")


class _FakePipe:
    """Records load_lora_weights / set_adapters calls (torch-free) so apply_loras
    is covered without a real pipeline."""

    def __init__(self):
        self.loaded = []  # (path, adapter_name)
        self.activated = None  # (names, weights)

    def load_lora_weights(self, path, adapter_name=None):
        self.loaded.append((path, adapter_name))

    def set_adapters(self, names, adapter_weights=None):
        self.activated = (list(names), list(adapter_weights or []))


def test_apply_loras_stacks_and_activates():
    from ._adapters import apply_loras

    pipe = _FakePipe()
    names = apply_loras(pipe, ["/a.safetensors:0.8", "/b.safetensors:0.5"])
    _eq(names, ["lora_0", "lora_1"], "registered adapter names")
    _eq(len(pipe.loaded), 2, "two loras loaded")
    _eq(pipe.loaded[0], ("/a.safetensors", "lora_0"), "first lora load")
    _eq(pipe.activated[0], ["lora_0", "lora_1"], "activated names")
    _eq(pipe.activated[1], [0.8, 0.5], "activated weights")


def test_apply_loras_empty_is_noop():
    from ._adapters import apply_loras

    pipe = _FakePipe()
    _eq(apply_loras(pipe, []), [], "empty specs → no adapters")
    _eq(len(pipe.loaded), 0, "empty specs load nothing")
    assert pipe.activated is None, "empty specs do not activate"


def test_parse_controlnet_spec():
    from ._adapters import parse_controlnet_spec

    s = parse_controlnet_spec("/models/adapters/cn-canny:0.9:/tmp/cond-0.png")
    _eq(s.path, "/models/adapters/cn-canny", "controlnet dir")
    _eq(s.scale, 0.9, "controlnet scale")
    _eq(s.image, "/tmp/cond-0.png", "controlnet image")


def test_parse_controlnet_spec_rejects_malformed():
    from ._adapters import parse_controlnet_spec

    raised = False
    try:
        parse_controlnet_spec("/only/path")
    except ValueError:
        raised = True
    assert raised, "a controlnet spec missing scale+image must raise"


def test_parse_controlnet_specs_skips_blanks():
    from ._adapters import parse_controlnet_specs

    specs = parse_controlnet_specs(["/a:1:/i.png", "", "  ", "/b:0.5:/j.png"])
    _eq(len(specs), 2, "blank controlnet specs skipped")
    _eq(specs[1].scale, 0.5, "second controlnet scale")


def test_parse_ip_adapter_spec():
    from ._adapters import parse_ip_adapter_spec

    s = parse_ip_adapter_spec("/models/adapters/ip/ip.safetensors:0.6:/tmp/ref.png")
    _eq(s.path, "/models/adapters/ip/ip.safetensors", "ip-adapter weight file")
    _eq(s.scale, 0.6, "ip-adapter scale")
    _eq(s.image, "/tmp/ref.png", "ip-adapter reference image")


def test_apply_ip_adapter_rejects_multiple():
    from ._adapters import apply_ip_adapter

    raised = False
    try:
        apply_ip_adapter(_FakePipe(), ["/a.safetensors:0.5:/i.png", "/b.safetensors:0.5:/j.png"])
    except ValueError:
        raised = True
    assert raised, "more than one ip-adapter must raise (one per pipeline)"


def test_apply_ip_adapter_empty_is_none():
    from ._adapters import apply_ip_adapter

    assert apply_ip_adapter(_FakePipe(), []) is None, "no ip-adapter specs → None"


def main() -> int:
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
    print(f"ok - {len(tests)} diffusers adapter checks passed")
    return 0


if __name__ == "__main__":
    import sys

    sys.exit(main())
