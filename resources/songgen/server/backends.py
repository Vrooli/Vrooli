"""Model-specific adapters for the recipes qualified by the retained trials."""
import gc
import math
import random
import weakref


def generate_minimax(root, stage, profile, request):
    import bitsandbytes as bnb
    import numpy as np
    import soundfile as sf
    import torch
    from transformers import BitsAndBytesConfig
    from diffusers.guiders import ClassifierFreeGuidance
    from diffusers.modular_pipelines.minimax_music3.modular_blocks_minimax_music3 import (
        MiniMaxMusic3SemanticGenerationStep, MiniMaxMusic3CoreDenoiseStep,
    )
    from diffusers.modular_pipelines.minimax_music3.decoders import MiniMaxMusic3VocoderDecodeStep

    model_dir = str(root / profile["model_dir"])
    quantization = dict(profile["quantization"])
    if quantization.pop("bnb_4bit_compute_dtype") != "bfloat16":
        raise ValueError("unsupported compute dtype")
    quant = BitsAndBytesConfig(**quantization, bnb_4bit_compute_dtype=torch.bfloat16)
    generator = torch.Generator("cuda").manual_seed(request["seed"])

    def release(pipe, names):
        refs = [weakref.ref(getattr(pipe, name)) for name in names]
        pipe.unload_components(names)
        gc.collect()
        torch.cuda.synchronize()
        torch.cuda.empty_cache()
        if any(ref() is not None for ref in refs):
            raise RuntimeError("model stage references survived release")

    def load(block, names):
        pipe = block.init_pipeline(model_dir, components_manager=None)
        pipe.load_components(dtype=torch.bfloat16, pretrained_model_name_or_path=model_dir, local_files_only=True)
        if any(getattr(pipe, name, None) is None for name in names):
            raise RuntimeError("incomplete MiniMax component load")
        pipe.to("cuda")
        return pipe

    semantic = MiniMaxMusic3SemanticGenerationStep().init_pipeline(model_dir, components_manager=None)
    semantic.load_components(names=["language_model"], dtype=torch.bfloat16,
        pretrained_model_name_or_path=model_dir, local_files_only=True,
        quantization_config={"language_model": quant}, device_map={"language_model": {"": 0}})
    if not semantic.language_model.is_loaded_in_4bit or not isinstance(semantic.language_model.lm_head, bnb.nn.Linear4bit):
        raise RuntimeError("tested NF4/output-head quantization was not applied")
    semantic.load_components(names=["tokenizer", "rvq_depth_decoder"], dtype=torch.bfloat16,
        pretrained_model_name_or_path=model_dir, local_files_only=True)
    semantic.rvq_depth_decoder.to("cuda")
    frames = semantic(prompt=request["prompt"], lyrics=request["lyrics"], audio_duration=request["duration"],
                      generator=generator, output="frame_hiddens").detach().cpu().contiguous()
    if frames.ndim != 3 or not 0 < frames.shape[1] <= math.floor(request["duration"] * semantic.frame_rate) or not torch.isfinite(frames).all():
        raise RuntimeError("invalid MiniMax semantic frames")
    release(semantic, ["language_model", "rvq_depth_decoder"])
    del semantic
    denoise = load(MiniMaxMusic3CoreDenoiseStep(), ["condition_encoder", "transformer", "scheduler", "guider"])
    denoise.update_components(guider=ClassifierFreeGuidance(guidance_scale=request["guidance"]))
    actual_guidance = float(denoise.guider.config.guidance_scale)
    if actual_guidance != request["guidance"]:
        raise RuntimeError("MiniMax did not apply requested CFG guidance")
    chunks = denoise(frame_hiddens=frames, generator=generator, num_inference_steps=request["steps"], output="latent_chunks")
    actual_steps = len(denoise.scheduler.timesteps)
    if actual_steps != request["steps"]:
        raise RuntimeError("MiniMax did not apply requested denoising steps")
    chunks = [chunk.detach().cpu().contiguous() for chunk in chunks]
    release(denoise, ["condition_encoder", "transformer", "scheduler"])
    del denoise, frames
    decode = load(MiniMaxMusic3VocoderDecodeStep(), ["vocoder"])
    audio = decode(latent_chunks=[chunk.to("cuda") for chunk in chunks], output_type="np", output="audios")
    rate = decode.sampling_rate
    release(decode, ["vocoder"])
    del decode, chunks
    if audio.shape[0] != 1 or audio.shape[1] != 2 or not np.isfinite(audio).all():
        raise RuntimeError("invalid MiniMax decoded samples")
    sf.write(stage / "audio.wav", audio[0].T, rate, subtype="PCM_24")
    return {"duration_control": "upper_bound", "sampling": {"steps": actual_steps, "guidance": actual_guidance},
            "voice_reference_used": False, "limitations": ["NF4 output head changes semantic logits", "section and voice directions are soft"]}


def generate_heartmula(root, stage, profile, request):
    import numpy as np
    import soundfile as sf
    import torch
    from heartlib import HeartMuLaGenPipeline

    lyrics, tags = stage / "lyrics.txt", stage / "tags.txt"
    lyrics.write_text(request["lyrics"])
    tags.write_text(request["prompt"])
    random.seed(request["seed"])
    torch.manual_seed(request["seed"])
    torch.cuda.manual_seed_all(request["seed"])
    pipe = HeartMuLaGenPipeline.from_pretrained(str(root / profile["model_dir"]),
        device={"mula": torch.device("cuda"), "codec": torch.device("cuda")},
        dtype={"mula": torch.bfloat16, "codec": torch.float32}, version=profile["version"], lazy_load=True)
    inputs = {"lyrics": str(lyrics), "tags": str(tags)}
    milliseconds = int(request["duration"] * 1000)
    prompt_len = pipe.preprocess(inputs, cfg_scale=request["guidance"])["tokens"].shape[1]
    cache_len = ((prompt_len + milliseconds // 80 + 2 + 127) // 128) * 128
    if cache_len > 8192:
        raise ValueError("HeartMuLa prompt plus duration exceeds the bounded cache")
    original_compile = pipe._apply_compile
    def configure(model):
        model.backbone.max_seq_len = cache_len
        original_compile(model)
    pipe._apply_compile = configure
    def unload():
        for attr in ("_mula", "_codec"):
            model = getattr(pipe, attr, None)
            if model is not None:
                setattr(pipe, attr, None)
                del model
        gc.collect()
        torch.cuda.empty_cache()
    pipe._unload = unload
    original_postprocess = pipe.postprocess
    def postprocess(outputs, **kwargs):
        scalar = pipe.codec.scalar_model.to("cpu")
        original_decode = scalar.decode
        scalar.decode = lambda x: original_decode(x.to("cpu"))
        return original_postprocess(outputs, **kwargs)
    pipe.postprocess = postprocess
    with torch.no_grad():
        pre, forward, post = pipe._sanitize_parameters(max_audio_length_ms=milliseconds,
            save_path=str(stage / "audio.wav"), topk=profile["topk"], temperature=profile["temperature"], cfg_scale=request["guidance"])
        if forward.get("cfg_scale") != request["guidance"]:
            raise RuntimeError("HeartMuLa did not preserve requested CFG guidance")
        outputs = pipe._forward(pipe.preprocess(inputs, **pre), **forward)
        torch.manual_seed(request["seed"])
        torch.cuda.manual_seed_all(request["seed"])
        pipe.postprocess(outputs, **post)
    audio, rate = sf.read(stage / "audio.wav", always_2d=True)
    if not len(audio) or not np.isfinite(audio).all():
        raise RuntimeError("invalid HeartMuLa decoded samples")
    sf.write(stage / "audio.wav", audio, rate, subtype="PCM_24")
    return {"duration_control": "upper_bound", "sampling": {"guidance": request["guidance"],
            "topk": profile["topk"], "temperature": profile["temperature"], "decoder_seed": request["seed"], "cache_tokens": cache_len},
            "voice_reference_used": False, "limitations": ["section and voice directions are soft"]}
