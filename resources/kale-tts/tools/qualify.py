#!/usr/bin/env python3
"""Bounded Kale runtime/integrity sweep; this is not a speech-quality evaluator."""
import argparse
from datetime import datetime, timezone
import hashlib
import json
import math
import statistics
import time
import urllib.request
import wave
from io import BytesIO


# Fixed short prompts exercise ordinary prose, punctuation, and numbers. They
# are runtime probes, not a held-out recognition or naturalness benchmark.
PROMPTS = (
    "The quiet garden is ready for visitors this morning.",
    "Room 14 opens at 3:30 p.m.; please arrive five minutes early.",
)


def request_json(base_url, path, timeout):
    req = urllib.request.Request(base_url.rstrip("/") + path)
    with urllib.request.urlopen(req, timeout=timeout) as response:
        return json.load(response)


def synthesize(base_url, voice, text, timeout):
    body = json.dumps({"model": "kale-tts-1-nano", "input": text,
                       "voice": voice, "response_format": "wav"}).encode()
    req = urllib.request.Request(base_url.rstrip("/") + "/v1/audio/speech",
                                data=body, headers={"Content-Type": "application/json"})
    started = time.monotonic()
    with urllib.request.urlopen(req, timeout=timeout) as response:
        audio = response.read()
    elapsed_ms = round((time.monotonic() - started) * 1000, 2)
    with wave.open(BytesIO(audio), "rb") as wav:
        if wav.getnchannels() != 1 or wav.getframerate() != 24000 or wav.getsampwidth() != 2:
            raise ValueError("unexpected WAV format (expected mono 24 kHz PCM16)")
        raw = wav.readframes(wav.getnframes())
        samples = [int.from_bytes(raw[i:i + 2], "little", signed=True)
                   for i in range(0, len(raw), 2)]
    if not samples:
        raise ValueError("synthesis returned no samples")
    peak = max(abs(sample) for sample in samples) / 32768
    rms = math.sqrt(sum(sample * sample for sample in samples) / len(samples)) / 32768
    duration = len(samples) / 24000
    if not (0.1 <= duration <= 30.0):
        raise ValueError(f"implausible duration: {duration:.3f}s")
    if peak < 0.01 or rms < 0.001:
        raise ValueError("audio is silent or near-silent")
    return {"latency_ms": elapsed_ms, "duration_seconds": round(duration, 4),
            "peak": round(peak, 6), "rms": round(rms, 6),
            "clipped_sample_count": sum(abs(sample) >= 32767 for sample in samples),
            "wav_sha256": hashlib.sha256(audio).hexdigest()}


def run(base_url, voices=None, timeout=45):
    health = request_json(base_url, "/healthz", timeout)
    if not health.get("ready") or health.get("model") != "kale-tts-1-nano":
        raise ValueError(f"service is not ready for Nano: {health}")
    catalog = request_json(base_url, "/v1/audio/voices", timeout)
    available = [voice["id"] for voice in catalog.get("voices", [])]
    selected = voices or available
    if not selected or any(voice not in available for voice in selected):
        raise ValueError("requested voices must be present in the service voice catalogue")
    rows = []
    for voice in selected:
        for index, prompt in enumerate(PROMPTS, start=1):
            rows.append({"voice": voice, "prompt_id": f"probe-{index}",
                         **synthesize(base_url, voice, prompt, timeout)})
    latencies = [row["latency_ms"] for row in rows]
    return {"schema_version": 1, "qualification": "runtime-integrity-smoke",
            "quality_claim": False,
            "measured_at_utc": datetime.now(timezone.utc).isoformat(),
            "endpoint": base_url.rstrip("/"),
            "model": "kale-tts-1-nano", "voice_ids": selected,
            "prompt_ids": [f"probe-{i}" for i in range(1, len(PROMPTS) + 1)],
            "request_count": len(rows), "latency_ms": {
                "median": round(statistics.median(latencies), 2),
                "max": max(latencies)}, "results": rows}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--url", default="http://127.0.0.1:18896")
    parser.add_argument("--voice", action="append", help="limit to voice ID; repeatable (default: all)")
    parser.add_argument("--timeout", type=float, default=45)
    args = parser.parse_args()
    print(json.dumps(run(args.url, args.voice, args.timeout), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
