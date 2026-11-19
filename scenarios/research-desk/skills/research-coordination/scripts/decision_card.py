#!/usr/bin/env python3
"""Check a feature decision card, and render it as a Notification Hub decision page.

A card (research-feature-decision/v1) is one served feature that disappeared, as BAS's
feature-inventory diff emits it; BAS runs `validate` on every card it writes. `render`
writes the card as a Hub explainer page (the kit the Hub styles, answer buttons that fill
in the answer), writes BRIEF.md beside it (the packet a worker executes on the answer), and
prints the needs-you item that names both, so the card reaches the operator through the
normal decision route. Render into the level's work/<ID>/. Evidence images are embedded data: URIs.
"""

from __future__ import annotations

import argparse
import base64
import binascii
import datetime as dt
import html
import json
import re
import sys
from pathlib import Path

ID = re.compile(r"[A-Z0-9][A-Z0-9_-]{0,127}")
IMAGE = re.compile(r"data:image/(png|jpeg|webp);base64,([A-Za-z0-9+/=]+)")
MAGIC = {"png": lambda b: b.startswith(b"\x89PNG\r\n\x1a\n"), "jpeg": lambda b: b.startswith(b"\xff\xd8\xff"),
         "webp": lambda b: b[:4] == b"RIFF" and b[8:12] == b"WEBP"}
REQUIRED = {"schema", "id", "title", "location", "summary", "evidence", "options", "recommendation", "status", "terms"}
EFFECT = {"keep": (r"restore|retain|keep", "keep must say that the feature will be restored or retained"),
          "drop": (r"delete|remove", "drop must say that its code will be deleted or removed")}


def text(value: object) -> bool:
    return isinstance(value, str) and bool(value.strip())


def image_problem(index: int, source: object) -> str | None:
    match = IMAGE.fullmatch(source) if isinstance(source, str) else None
    if not match:
        return f"evidence {index} must embed a PNG, JPEG or WebP data URI; file paths and links are not accepted"
    try:
        raw = base64.b64decode(match[2], validate=True)
    except (ValueError, binascii.Error):
        return f"evidence {index} has invalid base64 image data"
    if not raw or len(raw) > 1_048_576:
        return f"evidence {index} image must be 1 byte to 1 MiB"
    return None if MAGIC[match[1]](raw) else f"evidence {index} image bytes do not match {match[1]}"


def validate(card: object) -> list[str]:
    if not isinstance(card, dict):
        return ["card must be a JSON object"]
    errors = []
    check = lambda ok, message: ok or errors.append(message)
    missing = sorted(REQUIRED - set(card))
    check(not missing, "missing fields: " + ", ".join(missing))
    check(card.get("schema") == "research-feature-decision/v1", "schema must be research-feature-decision/v1")
    check(isinstance(card.get("id"), str) and ID.fullmatch(card["id"]), "id must use uppercase letters, digits, underscores or hyphens")
    for field in ("title", "summary"):
        check(text(card.get(field)), f"{field} must be a non-empty sentence")
    summary = str(card.get("summary") or "").strip()
    if summary:
        check(len(re.split(r"(?<=[.!?])\s+", summary)) == 1 and summary[-1] in ".!?", "summary must be one complete sentence")
        check(len(summary.split()) <= 28, "summary must use 28 words or fewer")
    location = card.get("location") if isinstance(card.get("location"), dict) else {}
    check(text(location.get("screen")), "location.screen must name the screen")
    taps = location.get("taps")
    check(isinstance(taps, list) and taps and all(map(text, taps)), "location.taps must list the taps to reach the screen")
    evidence = card.get("evidence")
    if check(isinstance(evidence, list) and evidence, "evidence must contain at least one embedded image"):
        for index, item in enumerate(evidence, start=1):
            if not isinstance(item, dict) or not text(item.get("alt")):
                errors.append(f"evidence {index} needs short alt text")
            elif problem := image_problem(index, item.get("image")):
                errors.append(problem)
    options = card.get("options") if isinstance(card.get("options"), list) else []
    keys = [o.get("key") for o in options if isinstance(o, dict)]
    check(sorted(keys) == ["drop", "keep"] and len(options) == 2, "options must be exactly keep and drop")
    for option in (o for o in options if isinstance(o, dict) and o.get("key") in EFFECT):
        key = option["key"]
        check(text(option.get("label")), f"option {key} needs a plain-language label")
        if check(text(option.get("effect")), f"option {key} must explain what happens"):
            check(re.search(EFFECT[key][0], option["effect"], re.I), EFFECT[key][1])
    check(card.get("recommendation") in keys and card.get("recommendation") in EFFECT, "recommendation must be keep or drop, and offered")
    status, answer = card.get("status"), card.get("answer")
    check(status in ("pending", "resolved"), "status must be pending or resolved")
    check(status != "resolved" or answer in EFFECT, "a resolved card must record answer keep or drop")
    check(status != "pending" or answer is None, "a pending card must not claim an answer")
    terms = card.get("terms")
    check(isinstance(terms, list) and all(isinstance(t, dict) and text(t.get("word")) and text(t.get("meaning")) for t in terms),
          "terms must be a list that defines any needed word in plain language; use [] when there are no special words")
    return errors


SCRIPT = ('document.addEventListener("click",function(e){var b=e.target.closest&&e.target.closest("[data-od-answer]");'
          'if(b&&window.hub&&window.hub.prefill)window.hub.prefill("answer",b.getAttribute("data-od-answer"))});')


def question(card: dict) -> str:
    return f"Keep or drop “{card['title']}”?"


def render(card: dict) -> str:
    """The card as a Hub explainer page, in the markup api-core operatordecision.Page writes."""
    e = lambda value: html.escape(str(value), quote=True)
    where = " → ".join(e(step) for step in card["location"]["taps"])
    figures = "".join(f'<figure class="hub-figure"><img src="{e(i["image"])}" alt="{e(i["alt"])}"><figcaption>{e(i["alt"])}</figcaption></figure>'
                      for i in card["evidence"])
    recommended = '<section class="is-recommended"><span class="hub-chip is-primary">Recommended</span>'
    choices = "".join(
        (recommended if o["key"] == card["recommendation"] else "<section>") +
        f'<h3>{e(o["label"])}</h3><p>{e(o["effect"])}</p><button type="button" data-od-answer="{e(o["key"])}">Choose this</button></section>'
        for o in card["options"])
    answer = (f'<div class="hub-callout is-success"><strong>Answered:</strong> {e(card["answer"])}. This page replays an answered choice.</div>'
              if card["status"] == "resolved" else "")
    terms = "".join(f"<dt>{e(t['word'])}</dt><dd>{e(t['meaning'])}</dd>" for t in card["terms"])
    terms = '<h2>Words used</h2><dl class="od-facts">' + terms + "</dl>" if terms else ""
    return ('<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">'
            f'<title>{e(question(card))}</title><style>.od-facts{{display:grid;grid-template-columns:auto 1fr;gap:.25rem .75rem}}'
            '.od-facts dt{font-weight:600}.od-facts dd{margin:0}</style></head><body>'
            f'<h1>{e(question(card))}</h1><p>{e(card["summary"])}</p>{answer}'
            f'<dl class="od-facts"><dt>Where</dt><dd>{e(card["location"]["screen"])}</dd><dt>How to get there</dt><dd>{where}</dd></dl>'
            f'{figures}<h2>Your choices</h2><div class="hub-tradeoffs">{choices}</div>'
            f'{terms}<script>{SCRIPT}</script></body></html>')


def brief(card: dict) -> str:
    """The packet: what a worker does on each answer, with no other context."""
    taps = " → ".join(card["location"]["taps"])
    effects = "\n".join(f"- **`{o['key']}`{' (recommended)' if o['key'] == card['recommendation'] else ''}.** {o['effect']}"
                         for o in card["options"])
    checked = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%MZ")
    return (f"# {card['id']} · packet: {question(card)}\n\n{card['summary']}\n\n"
            f"Checked: {checked} the served feature inventory diff reports this feature gone (card {card['id']}, "
            f"rendered from it just now)\n\n"
            f"**Where:** {card['location']['screen']} ({taps}). **Evidence:** the decision page beside this file.\n\n"
            f"## On each answer\n\n{effects}\n- **`other`.** The operator's own words are the instruction; "
            "do only what they say, and if they ask for neither keep nor drop, change nothing and raise it again.\n\n"
            "## Proof\n\nThe served screen shows the feature (keep), or no code provides it (drop), and the feature "
            "inventory diff reports this card resolved.\n")


def needs_you_item(card: dict, page: Path, packet: Path) -> str:
    """The needs-you item that sends the card to the operator with its page and packet."""
    label = {o["key"]: o["label"] for o in card["options"]}
    return "\n".join([
        f"### {card['id']} — {question(card)}",
        f"{card['summary']} It is on {card['location']['screen']}. The page shows it before and after the change.",
        f"Recommended: {card['recommendation']} — {label[card['recommendation']]}",
        f"Options: keep — {label['keep']}; drop — {label['drop']}",
        f"Packet: {packet.resolve()}",
        f"Explainer: {page.resolve()}",
    ])


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["validate", "render"])
    parser.add_argument("--file", required=True, type=Path)
    parser.add_argument("--output", type=Path, help="Where render writes the page")
    args = parser.parse_args()
    try:
        card = json.loads(args.file.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"cannot read card JSON: {exc}", file=sys.stderr)
        return 2
    if errors := validate(card):
        print("Feature decision card rejected:\n" + "\n".join(f"- {error}" for error in errors), file=sys.stderr)
        return 1
    if args.action == "validate":
        print(f"Feature decision card valid: {args.file}")
        return 0
    if args.output is None or args.output.parent.name != card["id"]:
        print(f"render requires --output <level>/work/{card['id']}/page.html: a decision's page and brief live in its own folder",
              file=sys.stderr)
        return 2
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(render(card), encoding="utf-8")
    packet = args.output.with_name("BRIEF.md")
    print(f"Feature decision page rendered: {args.output}")
    if card["status"] == "pending":
        packet.write_text(brief(card), encoding="utf-8")
        print(f"Packet written: {packet}")
        print("Needs-you item (add it with needs_you.py write):\n" + needs_you_item(card, args.output, packet))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
