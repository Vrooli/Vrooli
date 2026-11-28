#!/usr/bin/env python3
"""Qualify native voice direction against a pinned, locally staged Qwen checkpoint."""

import argparse
import json
from pathlib import Path
import re
import time


def render(recipe, checkpoint, output, device, threads):
    import numpy as np
    import soundfile as sf
    import torch
    from qwen_tts import Qwen3TTSModel

    take_id = recipe.get("take_id", "take")
    lines = recipe["lines"]
    ids = [item["id"] for item in lines]
    if (not lines or len(ids) != len(set(ids)) or take_id in ids
            or any(not re.fullmatch("[a-z][a-z0-9-]*", name) for name in [take_id, *ids])
            or any(not item["text"].strip() or not item["direction"].strip() for item in lines)):
        raise ValueError("Use unique slugs and supply transcript and native direction for every line")

    torch.set_num_threads(threads)
    torch.manual_seed(recipe.get("seed", 11))
    started = time.monotonic()
    model = Qwen3TTSModel.from_pretrained(str(checkpoint), device_map=device,
                                          dtype=torch.bfloat16, attn_implementation="sdpa")
    print(f"Checkpoint loaded in {time.monotonic() - started:.1f}s", flush=True)
    output.mkdir(parents=True, exist_ok=True)
    segments = []
    tracks = []
    rate = None
    cursor = 0
    for item in lines:
        name, text, direction = item["id"], item["text"], item["direction"]
        started = time.monotonic()
        wavs, sample_rate = model.generate_custom_voice(
            text=text, language="English", speaker=recipe["speaker"], instruct=direction,
            max_new_tokens=400, top_p=.9, temperature=.75)
        if rate is not None and sample_rate != rate:
            raise ValueError("Model changed sample rate between lines")
        rate = sample_rate
        wave = np.asarray(wavs[0], dtype=np.float32)
        if not wave.size or not np.isfinite(wave).all():
            raise ValueError(f"Invalid synthesis for {name}")
        active = np.flatnonzero(np.abs(wave) > .008)
        if active.size:
            wave = wave[max(0, active[0] - rate // 16):active[-1] + rate // 10]
        tracks.append(wave)
        provider = dict(provider_id="qwen3-customvoice-local-trial", model_id=checkpoint.name,
                        voice_used=recipe["speaker"], latency_ms=round((time.monotonic()-started)*1000))
        end = cursor + len(wave)
        segments.append(dict(line=name, start=cursor/rate, end=end/rate, text=text,
                             direction=direction, provider=provider))
        cursor = end
        sf.write(output / f"{name}.wav", wave, rate, subtype="PCM_16")
        print(f"{name}: {len(wave)/rate:.2f}s in {provider['latency_ms']/1000:.1f}s", flush=True)
    filename = f"{take_id}.wav"
    sf.write(output / filename, np.concatenate(tracks), rate, subtype="PCM_16")
    (output / "takes.json").write_text(json.dumps([dict(file=filename, segments=segments)], indent=2)+"\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("recipe", type=Path)
    parser.add_argument("checkpoint", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--device", default="cpu")
    parser.add_argument("--threads", type=int, default=8)
    args = parser.parse_args()
    render(json.loads(args.recipe.read_text()), args.checkpoint.resolve(strict=True), args.output,
           args.device, args.threads)
