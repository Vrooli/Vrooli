#!/usr/bin/env python3
"""Build reproducible speech auditions using Audio Tools synthesis and timed pauses."""

import argparse
import json
from pathlib import Path
import re
import subprocess
import tempfile
import wave


SAMPLE_RATE = 24000


def render(recipe, output, cli):
    variants = recipe["variants"]
    ids = [item["id"] for item in variants]
    if len(ids) != len(set(ids)) or any(not re.fullmatch(r"[a-z][a-z0-9-]*", item) for item in ids):
        raise ValueError("Take IDs must be unique lowercase slugs")
    lines = recipe["lines"]
    plans = []
    for variant in variants:
        delivery = variant.get("delivery", {})
        if delivery.keys() - lines.keys():
            raise ValueError(f"Unknown line in {variant['id']}")
        plan = []
        for name, text in lines.items():
            controls = delivery.get(name, {})
            speed = controls.get("speed", 1)
            pause_ms = controls.get("pause_ms", 0)
            if not text.strip() or not .5 <= speed <= 4 or not 0 <= pause_ms <= 5000:
                raise ValueError(f"Invalid text, speed, or pause for {name}")
            plan.append((name, text, speed, pause_ms))
        plans.append(plan)
    if not lines or not variants:
        raise ValueError("An audition needs at least one spoken line and variant")

    output.mkdir(parents=True, exist_ok=True)
    receipts = []
    cache = {}
    with tempfile.TemporaryDirectory(dir=output) as scratch:
        root = Path(scratch)
        raw = root / "synthesis.wav"
        for variant, plan in zip(variants, plans):
            take = root / (variant["id"] + ".wav")
            segments = []
            with wave.open(str(take), "wb") as audio:
                audio.setparams((1, 2, SAMPLE_RATE, 0, "NONE", "not compressed"))
                for name, text, speed, pause_ms in plan:
                    voice = variant["voice"]
                    adapter_voice = variant.get("adapter_voice")
                    key = (voice, adapter_voice, text, speed)
                    if key not in cache:
                        command = [
                            cli, "tts", "synthesize", "--text", text, "--voice", voice,
                            "--speed", str(speed), "--format", "wav", "--out", str(raw),
                            "--json", "--json-metadata-only",
                        ]
                        if adapter_voice:
                            command.extend(["--adapter-voice", adapter_voice])
                        response = subprocess.check_output(command, text=True, timeout=120)
                        pcm = subprocess.check_output([
                            "ffmpeg", "-nostdin", "-v", "error", "-i", str(raw), "-f", "s16le",
                            "-ac", "1", "-ar", str(SAMPLE_RATE), "-",
                        ], timeout=30)
                        if not pcm:
                            raise ValueError(f"No decoded speech for {name}")
                        cache[key] = (json.loads(response), pcm)
                    provider, pcm = cache[key]
                    for field in ("provider_id", "model_id", "voice_used"):
                        if field in variant.get("expect", {}) and variant["expect"][field] != provider.get(field):
                            raise ValueError(f"{variant['id']} expected {field}={variant['expect'][field]}, got {provider.get(field)}")
                    start = audio.getnframes() / SAMPLE_RATE
                    audio.writeframes(pcm)
                    end = audio.getnframes() / SAMPLE_RATE
                    audio.writeframes(bytes(round(SAMPLE_RATE * pause_ms / 1000) * 2))
                    segments.append(dict(line=name, start=start, end=end, text=text, speed=speed,
                                         pause_ms=pause_ms, provider=provider))
            receipts.append(dict(file=take.name, voice=variant["voice"],
                                 hypothesis=variant["hypothesis"], adapter_voice=variant.get("adapter_voice"), segments=segments))
        for receipt in receipts:
            (root / receipt["file"]).replace(output / receipt["file"])
    (output / "takes.json").write_text(json.dumps(receipts, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("recipe", type=Path)
    parser.add_argument("output", type=Path)
    parser.add_argument("--cli", default="audio-tools")
    args = parser.parse_args()
    render(json.loads(args.recipe.read_text()), args.output, args.cli)
