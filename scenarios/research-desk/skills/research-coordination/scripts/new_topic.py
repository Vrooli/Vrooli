#!/usr/bin/env python3
"""Start a research topic from its mode's template.

  new_topic.py --mode build --title "Title" --intent "operator words" --source "chat, 2026-11-15" name
  new_topic.py --mode build --title "Title" --intent-file WORDS.md name   (every quote, verbatim)

Copies templates/topics/common plus templates/topics/<mode> into <home>/topics/<name>, fills the
title, date, mode and the operator's words, and prints mission_check's verdict. The new mission
fails the check until every `<fill: …>` marker is replaced, so a template can never pass for a
mission. Refuses an existing folder. Standard library; run with python3 -I.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent
sys.dont_write_bytecode = True
sys.path.insert(0, str(SCRIPTS))
import mission_check as mc  # noqa: E402

TEMPLATES = SCRIPTS.parent / "templates" / "topics"
NAME = re.compile(r"[a-z0-9][a-z0-9-]{0,63}")
PLACEHOLDER = re.compile(r"\{\{([a-z_]+)\}\}")


def fill(text: str, values: dict[str, str]) -> str:
    return PLACEHOLDER.sub(lambda m: values[m.group(1)], text)


def intent_block(intent: str | None, source: str | None, intent_file: Path | None, today: dt.date) -> str:
    """The Operator intent section: one quote from --intent, or every `>` quote block of a file
    copied verbatim (each keeps its own date and source)."""
    if intent_file is None:
        return f'> "{intent.strip().strip(chr(34) + "“”")}" — {today:%Y-%m-%d}, {source}'
    blocks = [b.strip() for b in re.split(r"\n\s*\n", intent_file.read_text(encoding="utf-8")) if b.strip().startswith(">")]
    if not blocks:
        raise SystemExit(f"{intent_file} holds no '>' quote blocks; quote the operator's words verbatim")
    return "\n\n".join(blocks)


def create(home: Path, name: str, mode: str, title: str, intent: str | None, source: str | None, today: dt.date,
           intent_file: Path | None = None) -> Path:
    if not NAME.fullmatch(name):
        raise SystemExit(f"topic name {name!r} must match {NAME.pattern}")
    if mode not in mc.MODES:
        raise SystemExit(f"mode must be one of {', '.join(mc.MODES)}")
    target = home / "topics" / name
    if target.exists():
        raise SystemExit(f"{target} already exists; audit it instead (research-review)")
    spec = json.loads((TEMPLATES / mode / "mode.json").read_text(encoding="utf-8"))
    values = {"title": title, "topic": name, "mode": mode, "date": f"{today:%Y-%m-%d}",
              "intent_block": intent_block(intent, source, intent_file, today), **spec}
    for layer in (TEMPLATES / "common", TEMPLATES / mode):
        for src in sorted(p for p in layer.rglob("*") if p.is_file() and p.name != "mode.json"):
            dest = target / src.relative_to(layer)
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_text(fill(src.read_text(encoding="utf-8"), values), encoding="utf-8")
    return target


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("name")
    ap.add_argument("--mode", required=True, choices=mc.MODES)
    ap.add_argument("--title", required=True)
    words = ap.add_mutually_exclusive_group(required=True)
    words.add_argument("--intent", help="one quote of the operator's words, verbatim (with --source)")
    words.add_argument("--intent-file", type=Path, help="Markdown whose '>' quote blocks are the operator's words, each with its date and source")
    ap.add_argument("--source", help="where and when the operator said the --intent quote")
    ap.add_argument("--home", type=Path, default=Path.home() / ".vrooli" / "research")
    a = ap.parse_args(argv)
    if a.intent and not a.source:
        ap.error("--intent needs --source")
    target = create(a.home, a.name, a.mode, a.title, a.intent, a.source, dt.date.today(), a.intent_file)
    rep = mc.check_mission(a.name, target / "MISSION.md")
    print(f"Created {target}")
    for f in sorted(p.relative_to(target) for p in target.rglob("*") if p.is_file()):
        print(f"  {f}")
    print(f"mission_check: {len(rep.errors)} errors (expected until every <fill: …> is replaced)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
