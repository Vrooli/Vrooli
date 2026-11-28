#!/usr/bin/env python3
"""Versioned, renderer-neutral motion timing presets for launch-video compositions.

This compiles authored scene timing and music cue metadata into explicit beat,
transition, camera, and type entrance parameters. It does not analyze audio or
claim that a detected beat is musically correct; cues must come from the chosen
track's retained cue sheet or an editor-reviewed beat map.
"""
from __future__ import annotations

import argparse
import json
import math
from pathlib import Path

VERSION = 1
PROFILES = {
    # Ranges are starting points, not fixed taste rules. Scene/reading constraints win.
    "launch": {"beat_subdivision": 1, "target_cut_beats": 8, "min_cut_beats": 4,
               "max_cut_beats": 16, "reading_min": 1.2, "transition_beats": 0.5,
               "camera_scale": 0.045, "camera_px": 24},
    "product_story": {"beat_subdivision": 1, "target_cut_beats": 12, "min_cut_beats": 6,
                      "max_cut_beats": 20, "reading_min": 1.6, "transition_beats": 0.5,
                      "camera_scale": 0.025, "camera_px": 14},
    "explainer": {"beat_subdivision": 1, "target_cut_beats": 16, "min_cut_beats": 8,
                  "max_cut_beats": 24, "reading_min": 2.0, "transition_beats": 0.0,
                  "camera_scale": 0.012, "camera_px": 8},
}


def _number(value, name, *, minimum=None):
    if isinstance(value, bool) or not isinstance(value, (int, float)) or not math.isfinite(value):
        raise ValueError(f"{name} must be a finite number")
    if minimum is not None and value < minimum:
        raise ValueError(f"{name} must be >= {minimum}")
    return float(value)


def compile_plan(request):
    """Return resolved scene timing. Times are seconds from composition zero."""
    profile_name = request.get("profile")
    if profile_name not in PROFILES:
        raise ValueError(f"profile must be one of: {', '.join(PROFILES)}")
    cfg = PROFILES[profile_name]
    audio = request.get("audio", {})
    bpm = _number(audio.get("bpm"), "audio.bpm", minimum=40)
    if bpm > 240:
        raise ValueError("audio.bpm must be <= 240")
    offset = _number(audio.get("first_beat_seconds", 0), "audio.first_beat_seconds")
    beat = 60.0 / bpm
    subdivision = audio.get("beat_subdivision", cfg["beat_subdivision"])
    if isinstance(subdivision, bool) or not isinstance(subdivision, int) or subdivision not in (1, 2, 4):
        raise ValueError("audio.beat_subdivision must be 1, 2, or 4")
    cut_grid = beat / subdivision
    scenes = request.get("scenes")
    if not isinstance(scenes, list) or not scenes:
        raise ValueError("scenes must be a non-empty list")
    out = []
    previous_end = 0.0
    for i, scene in enumerate(scenes):
        sid = scene.get("id") or f"scene-{i+1}"
        start = _number(scene.get("start_seconds"), f"{sid}.start_seconds", minimum=0)
        end = _number(scene.get("end_seconds"), f"{sid}.end_seconds", minimum=0)
        if end <= start or (i and abs(start - previous_end) > 1e-6):
            raise ValueError(f"{sid}: scenes must be positive and contiguous")
        duration = end - start
        nearest_beat = offset + round((start - offset) / beat) * beat
        nearest_cut_grid = offset + round((start - offset) / cut_grid) * cut_grid
        transition = _number(scene.get("transition_beats", cfg["transition_beats"]), f"{sid}.transition_beats", minimum=0) * beat
        if transition * 2 >= duration:
            raise ValueError(f"{sid}: incoming and outgoing transitions consume the scene")
        words = scene.get("words", 0)
        if isinstance(words, bool) or not isinstance(words, int) or words < 0:
            raise ValueError(f"{sid}.words must be a non-negative integer")
        # Readability is a minimum settled hold, excluding both entrance and exit.
        entrance = min(0.42, max(0.24, beat * 0.72)) if words else 0.0
        settled = _number(scene.get("settled_hold_seconds", cfg["reading_min"] if words else 0), f"{sid}.settled_hold_seconds", minimum=0)
        if words and settled < cfg["reading_min"]:
            raise ValueError(f"{sid}: settled type hold {settled:g}s below {profile_name} minimum {cfg['reading_min']:g}s")
        incoming_transition = transition if i > 0 else 0.0
        outgoing_transition = transition if i < len(scenes) - 1 else 0.0
        if entrance + settled + incoming_transition + outgoing_transition > duration:
            available = duration - entrance - incoming_transition - outgoing_transition
            raise ValueError(
                f"{sid}: declared readable hold {settled:g}s exceeds {available:g}s after "
                "type entrance and scene-boundary transitions"
            )
        camera = scene.get("camera", "hold")
        if camera not in ("hold", "push", "drift-left", "drift-right"):
            raise ValueError(f"{sid}.camera is unsupported")
        cut_beats = duration / beat
        pacing = "hold" if cut_beats >= cfg["max_cut_beats"] else "quick" if cut_beats <= cfg["min_cut_beats"] else "target"
        out.append({"id": sid, "start_seconds": start, "end_seconds": end,
                    "nearest_beat_seconds": round(nearest_beat, 6),
                    "beat_error_seconds": round(start-nearest_beat, 6),
                    "nearest_cut_grid_seconds": round(nearest_cut_grid, 6),
                    "cut_grid_error_seconds": round(start-nearest_cut_grid, 6),
                    "duration_beats": round(cut_beats, 4), "pacing": pacing,
                    "transition_seconds": round(transition, 6),
                    "type_entrance_seconds": round(entrance, 6),
                    "settled_hold_seconds": settled, "camera": camera,
                    "camera_scale_delta": cfg["camera_scale"] if camera == "push" else 0,
                    "camera_translate_px": cfg["camera_px"] if camera.startswith("drift") else 0,
                    "easing": "power3.out" if words else "sine.inOut"})
        previous_end = end
    duration_seconds = out[-1]["end_seconds"] - out[0]["start_seconds"]
    cut_count = max(0, len(out) - 1)
    measured_cuts_per_minute = cut_count * 60.0 / duration_seconds
    reference_cut_seconds = cfg["target_cut_beats"] * beat
    reference_cuts_per_minute = 60.0 / reference_cut_seconds
    return {"version": VERSION, "profile": profile_name,
            "pacing_metrics": {
                "duration_seconds": round(duration_seconds, 6),
                "scene_count": len(out),
                "transition_count": cut_count,
                "measured_cuts_per_minute": round(measured_cuts_per_minute, 4),
                "profile_reference_cut_beats": cfg["target_cut_beats"],
                "profile_reference_cuts_per_minute": round(reference_cuts_per_minute, 4),
                "interpretation": "Measured from declared scene boundaries; reference is profile tempo math, not a taste or quality verdict."
            },
            "audio": {"bpm": bpm,
            "first_beat_seconds": offset, "beat_seconds": round(beat, 8),
            "beat_subdivision": subdivision, "cut_grid_seconds": round(cut_grid, 8)}, "scenes": out,
            "limits": ["Timing compiles declared cues; it does not analyze audio or certify taste.",
                       "Readable hold excludes entrance and transition; verify rendered text with text_entrance.py."]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("request", type=Path, help="JSON request with profile, audio and contiguous scenes")
    parser.add_argument("--out", type=Path, required=True, help="new plan JSON path")
    args = parser.parse_args()
    if args.out.exists():
        parser.error("output already exists")
    try:
        result = compile_plan(json.loads(args.request.read_text()))
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(result, indent=2) + "\n")
        print(json.dumps({"version": VERSION, "scenes": len(result["scenes"]), "out": str(args.out)}))
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
