#!/usr/bin/env python3
"""Validate and render an image-complete feature decision card.

The JSON input is a packet for one disappeared served feature. Evidence images
must be embedded as data URIs so the card stays readable when opened from a
phone without navigating to another file or private path.
"""

from __future__ import annotations

import argparse
import base64
import html
import json
import re
import sys
from pathlib import Path

ID = re.compile(r"[A-Z0-9][A-Z0-9_-]{0,127}\Z")
IMAGE = re.compile(r"data:image/(png|jpeg|webp);base64,([A-Za-z0-9+/=]+)\Z")
REQUIRED = {"schema", "id", "title", "location", "summary", "evidence", "options", "recommendation", "status", "terms"}


def validate(card: object) -> list[str]:
    errors: list[str] = []
    if not isinstance(card, dict):
        return ["card must be a JSON object"]
    missing = sorted(REQUIRED - set(card))
    if missing:
        errors.append("missing fields: " + ", ".join(missing))
    if card.get("schema") != "research-feature-decision/v1":
        errors.append("schema must be research-feature-decision/v1")
    if not isinstance(card.get("id"), str) or not ID.fullmatch(card.get("id", "")):
        errors.append("id must use uppercase letters, digits, underscores or hyphens")
    for field in ("title", "summary"):
        if not isinstance(card.get(field), str) or not card[field].strip():
            errors.append(f"{field} must be a non-empty sentence")
    summary = card.get("summary")
    if isinstance(summary, str) and summary.strip():
        sentences = [part for part in re.split(r"(?<=[.!?])\s+", summary.strip()) if part]
        if len(sentences) != 1 or not re.search(r"[.!?]$", summary.strip()):
            errors.append("summary must be one complete sentence")
        if len(summary.split()) > 28:
            errors.append("summary must use 28 words or fewer")
    location = card.get("location")
    if not isinstance(location, dict) or not isinstance(location.get("screen"), str) or not location["screen"].strip():
        errors.append("location.screen must name the screen")
    elif not isinstance(location.get("taps"), list) or not location["taps"] or not all(isinstance(step, str) and step.strip() for step in location["taps"]):
        errors.append("location.taps must list the taps to reach the screen")

    evidence = card.get("evidence")
    if not isinstance(evidence, list) or not evidence:
        errors.append("evidence must contain at least one embedded image")
    else:
        for index, item in enumerate(evidence, start=1):
            if not isinstance(item, dict) or not isinstance(item.get("alt"), str) or not item["alt"].strip():
                errors.append(f"evidence {index} needs short alt text")
                continue
            source = item.get("image")
            match = IMAGE.fullmatch(source) if isinstance(source, str) else None
            if not match:
                errors.append(f"evidence {index} must embed a PNG, JPEG or WebP data URI; file paths and links are not accepted")
                continue
            try:
                raw = base64.b64decode(match.group(2), validate=True)
                if not raw:
                    raise ValueError("empty image")
            except (ValueError, base64.binascii.Error):
                errors.append(f"evidence {index} has invalid base64 image data")
                continue
            if len(raw) > 1_048_576:
                errors.append(f"evidence {index} image exceeds 1 MiB")
                continue
            kind = match.group(1)
            valid_header = (
                raw.startswith(b"\x89PNG\r\n\x1a\n") if kind == "png" else
                raw.startswith(b"\xff\xd8\xff") if kind == "jpeg" else
                len(raw) >= 12 and raw.startswith(b"RIFF") and raw[8:12] == b"WEBP"
            )
            if not valid_header:
                errors.append(f"evidence {index} image bytes do not match {kind}")

    options = card.get("options")
    if not isinstance(options, list) or len(options) != 2:
        errors.append("options must contain exactly keep and drop")
        options = []
    by_key = {}
    for option in options:
        if not isinstance(option, dict) or option.get("key") not in ("keep", "drop"):
            errors.append("each option must use the key keep or drop")
            continue
        key = option["key"]
        if key in by_key:
            errors.append(f"option {key} is repeated")
        by_key[key] = option
        if not isinstance(option.get("label"), str) or not option["label"].strip():
            errors.append(f"option {key} needs a plain-language label")
        effect = option.get("effect")
        if not isinstance(effect, str) or not effect.strip():
            errors.append(f"option {key} must explain what happens")
        elif key == "keep" and not re.search(r"restore|retain|keep", effect, re.I):
            errors.append("keep must say that the feature will be restored or retained")
        elif key == "drop" and not re.search(r"delete|remove", effect, re.I):
            errors.append("drop must say that its code will be deleted or removed")
    if set(by_key) != {"keep", "drop"}:
        errors.append("both keep and drop options are required")
    if card.get("recommendation") not in ("keep", "drop"):
        errors.append("recommendation must be keep or drop")
    if card.get("recommendation") not in by_key:
        errors.append("recommendation must match an offered option")

    status = card.get("status")
    if status not in ("pending", "resolved"):
        errors.append("status must be pending or resolved")
    elif status == "resolved" and card.get("answer") not in ("keep", "drop"):
        errors.append("a resolved card must record answer keep or drop")
    elif status == "pending" and card.get("answer") is not None:
        errors.append("a pending card must not claim an answer")

    terms = card.get("terms")
    if not isinstance(terms, list) or not all(isinstance(term, dict) and isinstance(term.get("word"), str) and term["word"].strip() and isinstance(term.get("meaning"), str) and term["meaning"].strip() for term in terms):
        errors.append("terms must be a list that defines any needed word in plain language; use [] when there are no special words")
    return errors


def render(card: dict) -> str:
    evidence = "".join(
        f'<figure><img src="{html.escape(item["image"], quote=True)}" alt="{html.escape(item["alt"], quote=True)}"><figcaption>{html.escape(item["alt"])}</figcaption></figure>'
        for item in card["evidence"]
    )
    options = "".join(
        f'<article class="choice"><h2>{html.escape(option["label"])}</h2><p>{html.escape(option["effect"])}</p></article>'
        for option in card["options"]
    )
    answer = ""
    if card["status"] == "resolved":
        answer = f'<p class="answer">Recorded answer: {html.escape(card["answer"].title())}. This is a replay of an answered choice.</p>'
    terms = "".join(f'<li><strong>{html.escape(term["word"])}</strong>: {html.escape(term["meaning"])}</li>' for term in card.get("terms", []))
    taps = " &rarr; ".join(html.escape(step) for step in card["location"]["taps"])
    return f'''<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(card["title"])}</title>
<style>
*{{box-sizing:border-box}}body{{margin:0;background:#f5f7fa;color:#182230;font:16px/1.5 system-ui,sans-serif}}main{{max-width:720px;margin:auto;padding:16px}}h1{{font-size:1.35rem;line-height:1.25}}h2{{font-size:1rem;margin:0 0 6px}}p{{margin:8px 0}}.location,.answer{{padding:12px;border-radius:10px;background:#e8eef5}}.evidence{{display:grid;gap:12px}}figure{{margin:0;background:#fff;border:1px solid #cbd5e1;border-radius:12px;padding:8px}}img{{display:block;width:100%;height:auto;border-radius:8px}}figcaption{{font-size:.875rem;color:#46576b;padding-top:6px}}.choices{{display:grid;gap:10px}}.choice{{padding:12px;border:1px solid #b6c4d3;border-radius:10px;background:#fff}}.recommend{{font-weight:700;color:#155e45}}@media(min-width:640px){{.evidence{{grid-template-columns:1fr 1fr}}}}
</style><main>
<p class="recommend">Recommendation: {html.escape(card["recommendation"].title())}</p>
<h1>{html.escape(card["title"])}</h1><p>{html.escape(card["summary"])}</p>
<p class="location"><strong>Where:</strong> {html.escape(card["location"]["screen"])}<br><strong>Tap:</strong> {taps}</p>
<section class="evidence" aria-label="Evidence images">{evidence}</section>
{answer}<section class="choices" aria-label="Choices">{options}</section>
{('<section><h2>Word meanings</h2><ul>'+terms+'</ul></section>') if terms else ''}
</main></html>'''


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("validate", choices=["validate", "render"])
    parser.add_argument("--file", required=True, type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    try:
        card = json.loads(args.file.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"cannot read card JSON: {exc}", file=sys.stderr)
        return 2
    errors = validate(card)
    if errors:
        print("Feature decision card rejected:\n" + "\n".join(f"- {error}" for error in errors), file=sys.stderr)
        return 1
    if args.validate == "render":
        if args.output is None:
            print("render requires --output", file=sys.stderr)
            return 2
        args.output.write_text(render(card), encoding="utf-8")
        print(f"Feature decision card rendered: {args.output}")
    else:
        print(f"Feature decision card valid: {args.file}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
