"""Pure (torch-free) unit checks for the diffusers family adapter table.

Run standalone: ``python3 -m image_tools_sidecar.test_diffusers_adapters`` (exits
non-zero with a message on any mismatch). The Go test TestDiffusersSidecarContract
executes this inside ``go test`` so the param→kwarg mapping is covered in CI
without a separate pytest harness. It asserts ONLY the declared call contract —
no model is loaded — mirroring the Go arg-builder discipline.
"""

from __future__ import annotations

import io
import json
import os
from unittest.mock import patch

import numpy as np
from PIL import Image

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


def test_inpaint_composite_preserves_outside_binary_mask():
    original = Image.new("RGB", (13, 9), (13, 29, 41))
    generated = Image.new("RGB", (8, 8), (220, 170, 70))
    mask = Image.new("L", original.size, 0)
    for y in range(2, 7):
        for x in range(4, 10):
            mask.putpixel((x, y), 255)
    result = _diffusers._composite_only_mask(original, mask, generated)
    np.testing.assert_array_equal(np.asarray(result)[np.asarray(mask) == 0], np.asarray(original)[np.asarray(mask) == 0])
    assert np.all(np.asarray(result)[np.asarray(mask) > 0] == (220, 170, 70))


def test_inpaint_png_embeds_provenance():
    output = io.BytesIO()
    receipt = {"seed": 42, "model_id": "sd-1.5-inpainting"}
    _diffusers._save_provenance_png(Image.new("RGB", (2, 2)), output, receipt)
    decoded = Image.open(io.BytesIO(output.getvalue()))
    actual = json.loads(decoded.info["vrooli_inpaint_provenance"])
    _eq(actual, receipt, "inpaint provenance PNG text")


def test_inpaint_repository_selects_pinned_weight_variant():
    captured = {}

    class Loader:
        @staticmethod
        def from_pretrained(path, **kwargs):
            captured["path"] = path
            captured.update(kwargs)
            return object()

    class FakeTorch:
        float32 = "float32"

    class FakeCuda:
        @staticmethod
        def is_available():
            return False

    FakeTorch.cuda = FakeCuda
    fake_diffusers = type("FakeDiffusers", (), {"AutoPipelineForInpainting": Loader})
    with patch.dict(os.environ, {"IMAGE_TOOLS_DEVICE": "cpu"}):
        _diffusers._load_inpaint_pipe(
            fake_diffusers, FakeTorch, "sdxl", "/models/sdxl", True, None, "fp16"
        )
    _eq(captured.get("variant"), "fp16", "repo fp16 variant")


def test_inpaint_mask_crop_keeps_context_and_full_resolution_outside_pixels():
    original = Image.new("RGB", (1200, 800), (15, 35, 55))
    mask = Image.new("L", original.size, 0)
    for y in range(100, 700):
        for x in range(200, 1000):
            mask.putpixel((x, y), 255)
    crop_image, crop_mask, box, receipt = _diffusers._prepare_inpaint_crop(original, mask, 32, 512)
    _eq(box, (168, 68, 1032, 732), "mask crop with context")
    _eq(crop_image.size, (512, 392), "crop inference dimensions")
    _eq(crop_mask.size, crop_image.size, "crop image/mask dimensions")
    _eq(receipt["source_to_inference_matrix_row_major"][0][2], -168 * (512 / 864), "crop translation")
    generated = Image.new("RGB", crop_image.size, (230, 170, 90))
    result = _diffusers._composite_cropped_inpaint(original, mask, generated, box)
    source = np.asarray(original)
    output = np.asarray(result)
    binary = np.asarray(mask) >= 128
    np.testing.assert_array_equal(output[~binary], source[~binary])
    assert np.all(output[binary] == (230, 170, 90))


def test_inpaint_crop_neutralizes_masked_condition_without_changing_retained_pixels():
    original = Image.new("RGB", (64, 48), (40, 90, 130))
    # A distinctive object color must not be visible in the pipeline init crop.
    for y in range(12, 36):
        for x in range(20, 48):
            original.putpixel((x, y), (220, 20, 30))
    mask = Image.new("L", original.size, 0)
    for y in range(12, 36):
        for x in range(20, 48):
            mask.putpixel((x, y), 255)
    crop, crop_mask, box, receipt = _diffusers._prepare_inpaint_crop(
        original, mask, 8, 512, neutralize_masked_pixels=True
    )
    _eq(box, (12, 4, 56, 44), "neutralized crop bounds")
    assert receipt["masked_input_neutralization"].startswith("nearest_unmasked_pixel_voronoi_gaussian_sigma_"), receipt
    assert receipt["neutralization_sigma_px"] > 0, receipt
    crop_pixels = np.asarray(crop)
    crop_mask_pixels = np.asarray(crop_mask) >= 128
    assert not np.any(np.all(crop_pixels[crop_mask_pixels] == (220, 20, 30), axis=1)), "masked source object color leaked into init image"
    generated = Image.new("RGB", crop.size, (200, 180, 80))
    result = _diffusers._composite_cropped_inpaint(original, mask, generated, box)
    result_pixels, source_pixels = np.asarray(result), np.asarray(original)
    full_mask = np.asarray(mask) >= 128
    np.testing.assert_array_equal(result_pixels[~full_mask], source_pixels[~full_mask])
    assert np.all(result_pixels[full_mask] == (200, 180, 80))


def test_inpaint_cuda_peak_provenance_uses_initialized_logical_device():
    class FakeCuda:
        reset_device = None

        def init(self):
            self.initialized = True

        def current_device(self):
            return 0

        def reset_peak_memory_stats(self, device):
            self.reset_device = device

        def max_memory_allocated(self, device):
            assert device == 0
            return 1234

        def max_memory_reserved(self, device):
            assert device == 0
            return 2345

    cuda = FakeCuda()
    torch = type("FakeTorch", (), {"cuda": cuda})
    device = _diffusers._begin_cuda_peak_measurement(torch, True)
    _eq(device, 0, "initialized CUDA logical device")
    _eq(cuda.reset_device, 0, "CUDA peak reset device")
    with patch.dict(os.environ, {"CUDA_VISIBLE_DEVICES": "GPU-qualified-uuid"}):
        receipt = _diffusers._cuda_peak_receipt(torch, device)
    _eq(receipt["gpu_uuid"], "GPU-qualified-uuid", "broker UUID provenance")
    _eq(receipt["gpu_peak_allocated_bytes"], 1234, "CUDA allocated peak")
    _eq(receipt["gpu_peak_reserved_bytes"], 2345, "CUDA reserved peak")
    assert "before inpaint model load" in receipt["gpu_peak_scope"]


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
        build_call_kwargs("longcat-image-edit", Params(prompt="p"))
    except RuntimeError:
        raised = True
    assert raised, "unproven family build must raise rather than guess kwargs"


def test_flux2_klein_defaults_follow_distilled_model_card():
    k = build_call_kwargs("flux-2-klein", Params(prompt="the same guinea pig wearing a hat"))
    _eq(
        k,
        {"prompt": "the same guinea pig wearing a hat", "num_inference_steps": 4, "guidance_scale": 1.0},
        "klein defaults (4 steps, guidance 1.0, size left to the reference/native)",
    )
    adapter = _diffusers.adapter_for("flux-2-klein")
    assert adapter.ready and adapter.generates and adapter.multi_image, "klein is a proven unified multi-reference family"


def test_flux2_klein_overrides_snap_size_to_16():
    k = build_call_kwargs(
        "flux-2-klein",
        Params(prompt="p", steps=8, guidance=2.5, width=1030, height=770, negative_prompt="ignored", image_guidance=3.0),
    )
    _eq(k["num_inference_steps"], 8, "klein steps override")
    _eq(k["guidance_scale"], 2.5, "klein guidance override")
    _eq((k["width"], k["height"]), (1024, 768), "klein size snaps down to a multiple of 16")
    for absent in ("negative_prompt", "true_cfg_scale", "image_guidance_scale"):
        assert absent not in k, f"klein pipeline takes no {absent}"


def test_flux1_kontext_defaults_follow_the_model_card_and_stage_both_encoders():
    k = build_call_kwargs("flux-1-kontext", Params(prompt="make the hat red"))
    _eq(k, {"prompt": "make the hat red", "num_inference_steps": 28, "guidance_scale": 2.5}, "kontext defaults (28 steps, guidance 2.5)")
    adapter = _diffusers.adapter_for("flux-1-kontext")
    assert adapter.ready and adapter.staged and not adapter.generates and not adapter.multi_image
    _eq(adapter.text_components, ("text_encoder", "tokenizer", "text_encoder_2", "tokenizer_2"), "kontext text components")
    _eq(adapter.embed_kwargs(("EMB", "POOLED", "IDS")), {"prompt_embeds": "EMB", "pooled_prompt_embeds": "POOLED"}, "kontext embedding kwargs")

    events = []

    class Module:
        def __init__(self, name):
            self.name = name

        def to(self, device):
            events.append(f"{self.name}->{device}")
            return self

    class FluxKontextPipeline:
        @classmethod
        def from_pretrained(cls, path, **kwargs):
            pipe = cls()
            if kwargs.get("transformer", "x") is None:
                events.append("load encoders-only")
                pipe.text_encoder, pipe.text_encoder_2 = Module("clip"), Module("t5")
            else:
                for name in ("text_encoder", "tokenizer", "text_encoder_2", "tokenizer_2"):
                    assert kwargs.get(name, "x") is None, f"denoiser stage must not reload {name}"
                events.append("load denoiser")
            return pipe

        def encode_prompt(self, prompt, device):
            events.append("encode")
            return "EMB", "POOLED", "IDS"

        def to(self, device):
            events.append(f"denoiser->{device}")
            return self

    class NoGrad:
        def __enter__(self):
            return None

        def __exit__(self, *exc):
            return False

    class Cuda:
        @staticmethod
        def empty_cache():
            events.append("empty_cache")

    fake_torch = type("T", (), {"no_grad": staticmethod(lambda: NoGrad()), "cuda": Cuda})
    fake_diffusers = type("D", (), {"FluxKontextPipeline": FluxKontextPipeline})
    kwargs = {"prompt": "p", "num_inference_steps": 28}
    _diffusers._load_staged(fake_diffusers, fake_torch, adapter, "/m", {"torch_dtype": "bf16"}, kwargs)
    _eq(kwargs, {"num_inference_steps": 28, "prompt_embeds": "EMB", "pooled_prompt_embeds": "POOLED"}, "kontext prompt replaced by both embeddings")
    _eq(
        events,
        ["load encoders-only", "clip->cuda", "t5->cuda", "encode", "empty_cache", "load denoiser", "denoiser->cuda"],
        "both encoders run before the denoiser loads",
    )


def test_staged_placement_encodes_then_runs_denoiser_from_embeddings():
    events = []

    class Module:
        def __init__(self, name):
            self.name = name

        def to(self, device):
            events.append(f"{self.name}->{device}")
            return self

    class Flux2KleinPipeline:
        @classmethod
        def from_pretrained(cls, path, **kwargs):
            pipe = cls()
            if kwargs.get("transformer", "x") is None:
                events.append("load encoder-only")
                pipe.text_encoder = Module("text_encoder")
            else:
                assert kwargs.get("text_encoder", "x") is None, "denoiser stage must not reload the text encoder"
                events.append("load denoiser")
            return pipe

        def encode_prompt(self, prompt, device):
            events.append(f"encode {prompt!r} on {device}")
            return "EMBEDS", "IDS"

        def to(self, device):
            events.append(f"denoiser->{device}")
            return self

    class NoGrad:
        def __enter__(self):
            return None

        def __exit__(self, *exc):
            return False

    class Cuda:
        @staticmethod
        def empty_cache():
            events.append("empty_cache")

    fake_torch = type("T", (), {"no_grad": staticmethod(lambda: NoGrad()), "cuda": Cuda})
    fake_diffusers = type("D", (), {"Flux2KleinPipeline": Flux2KleinPipeline})
    kwargs = {"prompt": "a hat", "num_inference_steps": 4}
    pipe, placement = _diffusers._load_staged(
        fake_diffusers, fake_torch, _diffusers.adapter_for("flux-2-klein"), "/m", {"torch_dtype": "bf16"}, kwargs
    )
    _eq(placement, "staged_text_encoder_then_denoiser:gpu_resident", "placement label")
    _eq(kwargs, {"num_inference_steps": 4, "prompt_embeds": "EMBEDS"}, "prompt replaced by embeddings")
    _eq(
        events,
        ["load encoder-only", "text_encoder->cuda", "encode 'a hat' on cuda", "empty_cache", "load denoiser", "denoiser->cuda"],
        "encoder is freed before the denoiser loads",
    )
    assert isinstance(pipe, Flux2KleinPipeline)


def test_staged_placement_streams_each_stage_when_vram_is_contended():
    events = []

    class Stage:
        def __init__(self, encoder):
            self.text_encoder = object() if encoder else None

        def enable_sequential_cpu_offload(self):
            events.append("sequential " + ("encoder" if self.text_encoder else "denoiser"))

        def encode_prompt(self, prompt, device):
            events.append("encode")
            return "EMBEDS", "IDS"

    class Flux2KleinPipeline(Stage):
        @classmethod
        def from_pretrained(cls, path, **kwargs):
            pipe = cls(kwargs.get("transformer", "x") is None)
            return pipe

    class NoGrad:
        def __enter__(self):
            return None

        def __exit__(self, *exc):
            return False

    fake_torch = type("T", (), {"no_grad": staticmethod(lambda: NoGrad()), "cuda": type("C", (), {"empty_cache": staticmethod(lambda: None)})})
    kwargs = {"prompt": "p"}
    _, placement = _diffusers._load_staged(
        type("D", (), {"Flux2KleinPipeline": Flux2KleinPipeline}), fake_torch,
        _diffusers.adapter_for("flux-2-klein"), "/m", {}, kwargs, on_gpu=False,
    )
    _eq(placement, "staged_text_encoder_then_denoiser:sequential_cpu_offload", "contended placement label")
    _eq(events, ["sequential encoder", "encode", "sequential denoiser"], "each stage streams with sequential offload")


def test_text_only_run_refused_for_edit_only_family():
    raised = None
    try:
        _diffusers.run(family="instruct-pix2pix", model_dir="/nonexistent", image_paths=[], out_path="/dev/null", params=Params(prompt="p"), op="text_to_image")
    except SystemExit as exc:
        raised = exc.code
    _eq(raised, 2, "an edit-only family refuses a text-only run before loading anything")


def test_family_cli_refuses_conditioning_adapters():
    import argparse

    args = argparse.Namespace(
        family="flux-2-klein", model="/m", out="/o", prompt="p", negative_prompt="", steps=0,
        guidance=None, seed=0, width=0, height=0, lora=["/l.safetensors:0.8"], controlnet=[], ip_adapter=[],
    )
    raised = None
    with patch.object(_diffusers, "run") as run:
        try:
            _diffusers.run_family_op(args, op="text_to_image", image_paths=[])
        except SystemExit as exc:
            raised = exc.code
    _eq(raised, 2, "family route refuses LoRA rather than dropping it")
    assert not run.called, "refused request must not reach the pipeline"


def test_family_cli_maps_generate_args_onto_params():
    import argparse

    args = argparse.Namespace(
        family="flux-2-klein", model="/m", out="/o", prompt="a fox", negative_prompt="", steps=6,
        guidance=None, seed=7, width=1024, height=768, lora=[], controlnet=[], ip_adapter=[],
    )
    with patch.object(_diffusers, "run") as run:
        _diffusers.run_family_op(args, op="text_to_image", image_paths=[])
    kwargs = run.call_args.kwargs
    _eq(kwargs["op"], "text_to_image", "op threaded to the runner")
    _eq(kwargs["image_paths"], [], "text_to_image has no reference image")
    _eq(kwargs["params"], Params(prompt="a fox", steps=6, seed=7, width=1024, height=768), "generate params")


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
    _eq(k["guidance_scale"], 6.0, "inpaint default guidance")
    _eq(k["strength"], 1.0, "inpaint default strength")
    assert "negative_prompt" not in k, "inpaint omits negative_prompt when empty"


def test_sdxl_inpainting_model_uses_upstream_reference_strength():
    from ._diffusers import TransformParams, build_inpaint_kwargs

    params = TransformParams(prompt="continue the tabletop")
    sdxl = build_inpaint_kwargs(params, "sdxl-inpainting-0.1")
    generic = build_inpaint_kwargs(params, "some-other-sdxl-model")
    explicit = build_inpaint_kwargs(
        TransformParams(prompt="continue the tabletop", strength=0.7),
        "sdxl-inpainting-0.1",
    )
    _eq(sdxl["strength"], 0.99, "SDXL inpainting card default")
    _eq(generic["strength"], 1.0, "generic inpaint default gives a replacement garment the full noise budget")
    _eq(explicit["strength"], 0.7, "explicit strength overrides model default")


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


def _sparse_component(root, component, size):
    os.makedirs(os.path.join(root, component), exist_ok=True)
    with open(os.path.join(root, component, "model.safetensors"), "wb") as handle:
        handle.truncate(size)  # sparse: only the size matters


def test_staged_gpu_residency_floor_follows_the_largest_component():
    import tempfile

    gib = 1024**3
    with tempfile.TemporaryDirectory() as small, tempfile.TemporaryDirectory() as large:
        # klein 4B: ~8GB components keep the historical 10 GiB floor.
        _sparse_component(small, "text_encoder", 8 * 10**9)
        _sparse_component(small, "transformer", 7_750_000_000)
        _eq(_diffusers.staged_min_free_bytes(small), 10 * gib, "4B floor")
        # klein 9B: an 18.2GB transformer needs more than any 16GB card holds.
        _sparse_component(large, "text_encoder", 16_400_000_000)
        _sparse_component(large, "transformer", 18_157_190_880)
        need = _diffusers.staged_min_free_bytes(large)
        assert need > 16 * gib, f"9B residency floor {need} must exceed a 16 GiB card"
    _eq(_diffusers.staged_min_free_bytes("/nonexistent/model"), 10 * gib, "missing dir floor")


def test_fast_only_tokenizers_are_found_by_missing_vocab_files():
    """A tokenizer dir saved in the transformers 5 layout (tokenizer.json, no vocab.json)
    cannot become a slow CLIPTokenizer, so the loader must supply a fast one."""
    import tempfile

    with tempfile.TemporaryDirectory() as root:

        def make(sub, *files):
            os.makedirs(os.path.join(root, sub), exist_ok=True)
            for name in files:
                open(os.path.join(root, sub, name), "w").close()

        make("tokenizer", "tokenizer.json", "tokenizer_config.json")  # fast only
        make("tokenizer_2", "tokenizer.json", "vocab.json", "merges.txt")  # complete: leave alone
        make("tokenizer_3", "spiece.model", "tokenizer.json")  # sentencepiece: leave alone
        _eq(_diffusers.fast_only_tokenizers(root), ["tokenizer"], "fast-only tokenizers")
        _eq(_diffusers.fast_only_tokenizers(os.path.join(root, "missing")), [], "missing dir")
        _eq(_diffusers.repo_tokenizer_overrides(os.path.join(root, "tokenizer_2")), {}, "no override needed")


def test_fast_only_clip_tokenizer_is_rebuilt_as_the_slow_class_diffusers_expects():
    """diffusers rejects a fast tokenizer for a declared CLIPTokenizer; the override must be the
    slow class carrying the same vocabulary and merges."""
    import tempfile

    try:
        import transformers
    except ImportError:  # the pure checks above run without the heavy backend
        return
    vocab = {"<|startoftext|>": 0, "<|endoftext|>": 1, "a</w>": 2, "b</w>": 3, "a": 4, "b": 5, "ab</w>": 6}
    tokenizer_json = {"model": {"type": "BPE", "vocab": vocab, "merges": [["a", "b</w>"]]}}
    config = {"unk_token": "<|endoftext|>", "bos_token": "<|startoftext|>", "eos_token": "<|endoftext|>", "pad_token": "<|endoftext|>", "model_max_length": 77}
    with tempfile.TemporaryDirectory() as root:
        os.makedirs(os.path.join(root, "tokenizer"))
        with open(os.path.join(root, "tokenizer", "tokenizer.json"), "w") as fh:
            json.dump(tokenizer_json, fh)
        with open(os.path.join(root, "tokenizer", "tokenizer_config.json"), "w") as fh:
            json.dump(config, fh)
        got = _diffusers.repo_tokenizer_overrides(root)
        _eq(list(got), ["tokenizer"], "override names")
        assert isinstance(got["tokenizer"], transformers.CLIPTokenizer), type(got["tokenizer"])
        _eq(got["tokenizer"]("ab").input_ids, [0, 6, 1], "merged token ids")
        _eq(got["tokenizer"].model_max_length, 77, "CLIP context length kept (padding='max_length' depends on it)")


def test_edm_scheduler_is_misdeclared_only_beside_an_epsilon_model():
    _eq(_diffusers.edm_scheduler_misdeclared({"_class_name": "EDMDPMSolverMultistepScheduler", "prediction_type": "epsilon"}), True, "EDM + epsilon")
    _eq(_diffusers.edm_scheduler_misdeclared({"_class_name": "EDMDPMSolverMultistepScheduler", "prediction_type": "v_prediction"}), False, "EDM + v_prediction (Playground)")
    _eq(_diffusers.edm_scheduler_misdeclared({"_class_name": "EulerDiscreteScheduler", "prediction_type": "epsilon"}), False, "Euler")
    _eq(_diffusers.edm_scheduler_misdeclared({}), False, "empty config")


def test_base_checkpoint_ddpm_is_replaced_by_euler_for_inpaint():
    import tempfile

    class DDPMScheduler:
        config = {"_class_name": "DDPMScheduler"}

    class Euler:
        @staticmethod
        def from_config(config):
            return {"euler", config["_class_name"]}

    class Pipe:
        scheduler = DDPMScheduler()

    diffusers = type("D", (), {"EulerDiscreteScheduler": Euler})
    with tempfile.TemporaryDirectory() as root:
        with open(os.path.join(root, "model_index.json"), "w") as fh:
            json.dump({"_class_name": "StableDiffusionXLPipeline"}, fh)
        got = _diffusers._prefer_euler_for_base_inpaint(diffusers, Pipe(), root)
    _eq(got.scheduler, {"euler", "DDPMScheduler"}, "base checkpoint gets Euler")

    class InpaintPipe:
        scheduler = DDPMScheduler()

    with tempfile.TemporaryDirectory() as root:
        with open(os.path.join(root, "model_index.json"), "w") as fh:
            json.dump({"_class_name": "StableDiffusionXLInpaintPipeline"}, fh)
        got = _diffusers._prefer_euler_for_base_inpaint(diffusers, InpaintPipe(), root)
    _eq(type(got.scheduler).__name__, "DDPMScheduler", "dedicated inpaint model keeps its scheduler")


def test_a_stable_scheduler_is_left_alone():
    class EulerScheduler:
        config = {}

    class Pipe:
        scheduler = EulerScheduler()

    got = _diffusers._prefer_euler_for_base_inpaint(object(), Pipe(), "/nowhere")
    _eq(type(got.scheduler).__name__, "EulerScheduler", "non-DDPM scheduler untouched")


def test_misdeclared_edm_scheduler_is_replaced_by_euler():
    import tempfile

    try:
        import diffusers
    except ImportError:
        return
    config = {"_class_name": "EDMDPMSolverMultistepScheduler", "prediction_type": "epsilon", "beta_start": 0.00085, "beta_end": 0.012, "beta_schedule": "scaled_linear", "num_train_timesteps": 1000}
    with tempfile.TemporaryDirectory() as root:
        os.makedirs(os.path.join(root, "scheduler"))
        with open(os.path.join(root, "scheduler", "scheduler_config.json"), "w") as fh:
            json.dump(config, fh)
        got = _diffusers.repo_component_overrides(diffusers, root)
        _eq(type(got["scheduler"]).__name__, "EulerDiscreteScheduler", "replacement scheduler")
        _eq(got["scheduler"].config.prediction_type, "epsilon", "prediction type kept")
    with tempfile.TemporaryDirectory() as root:
        _eq(_diffusers.repo_component_overrides(diffusers, root), {}, "no scheduler config, no override")


def test_ip_adapter_layout_finds_the_encoder_beside_or_above_the_weight():
    import tempfile

    from ._adapters import ip_adapter_layout

    def touch(path):
        os.makedirs(os.path.dirname(path), exist_ok=True)
        open(path, "wb").close()

    with tempfile.TemporaryDirectory() as root:
        flat = os.path.join(root, "sd15", "models")
        os.makedirs(os.path.join(flat, "image_encoder"))
        weight = os.path.join(flat, "ip-adapter_sd15.safetensors")
        touch(weight)
        _eq(ip_adapter_layout(weight), (flat, "", "image_encoder"), "encoder beside the weight (SD 1.5 snapshot)")

        sdxl = os.path.join(root, "sdxl")
        os.makedirs(os.path.join(sdxl, "models", "image_encoder"))
        weight = os.path.join(sdxl, "sdxl_models", "ip-adapter-plus_sdxl_vit-h.safetensors")
        touch(weight)
        _eq(ip_adapter_layout(weight), (sdxl, "sdxl_models", "models/image_encoder"), "encoder above the weight (SDXL snapshot)")

        bare = os.path.join(root, "bare", "w.safetensors")
        touch(bare)
        _eq(ip_adapter_layout(bare), (os.path.dirname(bare), "", "image_encoder"), "no encoder found keeps the default")


def main() -> int:
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    for t in tests:
        t()
    print(f"ok - {len(tests)} diffusers adapter checks passed")
    return 0


if __name__ == "__main__":
    import sys

    sys.exit(main())

