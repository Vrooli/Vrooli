#!/usr/bin/env python3
"""Bind an exact compiled motion plan into an HTML composition template.

This creates a new, reviewable composition artifact without rendering media.
The template must expose one placeholder for the plan hash, one JSON script
placeholder, and one data-motion-scene value per compiled scene.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
from html.parser import HTMLParser
from pathlib import Path


HASH_MARKER = "__MOTION_PLAN_SHA256__"
PLAN_MARKER = "__MOTION_PLAN_JSON__"
CONSUMER = "JSON.parse(document.getElementById('motion-plan-data').textContent)"
MOTION_FIELDS = ("start_seconds", "end_seconds", "transition_seconds",
                 "type_entrance_seconds", "easing", "camera_scale_delta",
                 "camera_translate_px", "camera")


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


class SceneBindings(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=False)
        self.bindings: list[str] = []

    def handle_starttag(self, _tag, attrs):
        values = dict(attrs)
        value = values.get("data-motion-scene")
        if value is not None:
            self.bindings.append(value)

    def handle_startendtag(self, tag, attrs):
        self.handle_starttag(tag, attrs)


def _validate_plan(plan: dict) -> None:
    audio = plan.get("audio")
    if audio is not None:
        if not isinstance(audio, dict):
            raise ValueError("compiled motion audio metadata must be an object")
        bpm = audio.get("bpm")
        if isinstance(bpm, bool) or not isinstance(bpm, (int, float)) or not math.isfinite(bpm) or bpm <= 0:
            raise ValueError("compiled motion audio.bpm must be a finite positive number")
        first_beat = audio.get("first_beat_seconds", 0)
        if (isinstance(first_beat, bool) or not isinstance(first_beat, (int, float))
                or not math.isfinite(first_beat)):
            raise ValueError("compiled motion audio.first_beat_seconds must be finite")
        subdivision = audio.get("beat_subdivision", 1)
        if isinstance(subdivision, bool) or not isinstance(subdivision, int) or subdivision not in (1, 2, 4):
            raise ValueError("compiled motion audio.beat_subdivision must be 1, 2, or 4")
    scenes = plan.get("scenes")
    if not isinstance(scenes, list) or not scenes:
        raise ValueError("compiled motion plan must contain scenes")
    seen: set[str] = set()
    previous_end = None
    for scene in scenes:
        scene_id = scene.get("id")
        if not isinstance(scene_id, str) or not scene_id or scene_id in seen:
            raise ValueError("compiled motion scene IDs must be unique non-empty strings")
        seen.add(scene_id)
        start, end = scene.get("start_seconds"), scene.get("end_seconds")
        if (isinstance(start, bool) or not isinstance(start, (int, float)) or not math.isfinite(start)
                or isinstance(end, bool) or not isinstance(end, (int, float)) or not math.isfinite(end)
                or start < 0 or end <= start):
            raise ValueError(f"{scene_id}: invalid scene timing")
        if previous_end is not None and abs(start - previous_end) > 1e-6:
            raise ValueError(f"{scene_id}: plan scenes are not contiguous")
        previous_end = end


def compile_composition(template_path: Path, plan_path: Path) -> tuple[bytes, dict]:
    template_bytes = template_path.read_bytes()
    plan_bytes = plan_path.read_bytes()
    template = template_bytes.decode("utf-8")
    plan = json.loads(plan_bytes)
    _validate_plan(plan)
    if template.count(HASH_MARKER) != 1 or template.count(PLAN_MARKER) != 1:
        raise ValueError("template must contain each motion plan placeholder exactly once")
    if CONSUMER not in template:
        raise ValueError("template must consume the embedded exact plan by motion-plan-data ID")
    unapplied = [field for field in MOTION_FIELDS if f"scene.{field}" not in template]
    if unapplied:
        raise ValueError(f"template does not apply compiled motion fields: {', '.join(unapplied)}")
    parser = SceneBindings()
    parser.feed(template)
    bindings = parser.bindings
    plan_ids = [scene["id"] for scene in plan["scenes"]]
    if len(bindings) != len(set(bindings)) or bindings != plan_ids:
        raise ValueError(f"template scene bindings {bindings} do not exactly match plan scenes {plan_ids}")

    plan_hash = sha256(plan_bytes)
    hold_audit = []
    for index, scene in enumerate(plan["scenes"]):
        duration = scene["end_seconds"] - scene["start_seconds"]
        incoming = scene.get("transition_seconds", 0) if index > 0 else 0
        outgoing = scene.get("transition_seconds", 0) if index + 1 < len(plan["scenes"]) else 0
        entrance = scene.get("type_entrance_seconds", 0)
        required = scene.get("settled_hold_seconds", 0)
        available = duration - incoming - outgoing - entrance
        if available < required - 1e-6:
            raise ValueError(f"{scene['id']}: actual timeline leaves {available:g}s settled text, below declared {required:g}s")
        hold_audit.append({"scene_id": scene["id"], "declared_settled_hold_seconds": required,
                           "timeline_available_seconds": round(available, 6),
                           "incoming_transition_seconds": incoming, "outgoing_transition_seconds": outgoing,
                           "type_entrance_seconds": entrance, "satisfied": True})
    embedded_json = json.dumps(plan, ensure_ascii=False, separators=(",", ":"))
    embedded_json = embedded_json.replace("<", "\\u003c").replace(">", "\\u003e").replace("&", "\\u0026")
    result = template.replace(HASH_MARKER, plan_hash).replace(PLAN_MARKER, embedded_json).encode("utf-8")
    audio = plan.get("audio", {})
    bpm = audio.get("bpm")
    beat_seconds = (60.0 / bpm if isinstance(bpm, (int, float)) and not isinstance(bpm, bool)
                    and math.isfinite(bpm) and bpm > 0 else None)
    subdivision = audio.get("beat_subdivision", 1) if isinstance(audio, dict) else 1
    cut_grid_seconds = beat_seconds / subdivision if beat_seconds is not None else None
    first_beat = audio.get("first_beat_seconds", 0)
    beat_cut_audit = []
    if (cut_grid_seconds is not None and isinstance(first_beat, (int, float))
            and not isinstance(first_beat, bool) and math.isfinite(first_beat)):
        for scene in plan["scenes"][1:]:
            start = scene["start_seconds"]
            nearest = first_beat + round((start - first_beat) / cut_grid_seconds) * cut_grid_seconds
            error = start - nearest
            beat_cut_audit.append({"scene_id": scene["id"], "cut_seconds": start,
                                   "nearest_cut_grid_seconds": round(nearest, 6),
                                   "offset_seconds": round(error, 6),
                                   "within_30ms_grid": abs(error) <= 0.03})
    record = {
        "schema": 1,
        "status": "composition-compiled-no-render",
        "template_sha256": sha256(template_bytes),
        "motion_plan_path": str(plan_path.resolve()),
        "motion_plan_sha256": plan_hash,
        "composition_sha256": sha256(result),
        "scene_bindings": bindings,
        "pacing_metrics": plan.get("pacing_metrics"),
        "applied_motion_fields": list(MOTION_FIELDS),
        "settled_hold_audit": hold_audit,
        "declared_cut_grid_audit": {
            "bpm": bpm,
            "first_beat_seconds": first_beat,
            "beat_seconds": round(beat_seconds, 8) if beat_seconds is not None else None,
            "beat_subdivision": subdivision,
            "cut_grid_seconds": round(cut_grid_seconds, 8) if cut_grid_seconds is not None else None,
            "cuts": beat_cut_audit,
            "on_grid_count": sum(1 for cut in beat_cut_audit if cut["within_30ms_grid"]),
            "limits": "Compares declared cut times with the supplied cue grid; does not analyze the audio or certify perceived synchronization.",
        },
        "limits": ["This binds declared plan bytes and scene IDs; it does not render or verify media, transitions, audio fit, or taste."],
    }
    return result, record


def write_new(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
    except Exception:
        path.unlink(missing_ok=True)
        raise


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("template", type=Path)
    parser.add_argument("plan", type=Path)
    parser.add_argument("--out", required=True, type=Path, help="new HTML output path")
    parser.add_argument("--receipt", required=True, type=Path, help="new JSON binding receipt path")
    args = parser.parse_args()
    written: list[Path] = []
    try:
        if args.out.resolve() == args.receipt.resolve():
            raise ValueError("composition and receipt require different paths")
        composition, receipt = compile_composition(args.template, args.plan)
        write_new(args.out, composition)
        written.append(args.out)
        receipt["composition_path"] = str(args.out.resolve())
        write_new(args.receipt, (json.dumps(receipt, indent=2) + "\n").encode())
        written.append(args.receipt)
        print(json.dumps({"status": receipt["status"], "composition_sha256": receipt["composition_sha256"],
                          "motion_plan_sha256": receipt["motion_plan_sha256"], "scenes": len(receipt["scene_bindings"])}))
    except (OSError, UnicodeError, ValueError, json.JSONDecodeError) as exc:
        for path in reversed(written):
            path.unlink(missing_ok=True)
        parser.error(str(exc))


if __name__ == "__main__":
    main()
