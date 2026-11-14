#!/usr/bin/env python3
"""Validate the recommendation grammar before a needs-you file is written."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

HEADING = re.compile(r"^### ([A-Z0-9][A-Z0-9_-]{0,127}) — (.+?)\s*$")
OPTION = re.compile(r"^([a-z0-9-]{1,32}) — (.+)$")
DEFAULT_KEYS = {"accept", "hold"}


def validate(text: str, section: str) -> list[str]:
    errors: list[str] = []
    active = False
    item_id: str | None = None
    options: list[str] | None = None
    recommendation: tuple[str, int] | None = None

    def finish() -> None:
        if item_id is None or recommendation is None:
            return
        key, line = recommendation
        choices = options if options else sorted(DEFAULT_KEYS)
        if key not in choices:
            errors.append(
                f"line {line} ({item_id}): Recommended key {key!r} is not offered. "
                f"Use one of: {', '.join(choices)}. Fix the Recommended key or add it to Options:."
            )

    for number, raw in enumerate(text.replace("\r\n", "\n").split("\n"), start=1):
        line = raw.strip()
        if line.startswith("## "):
            finish()
            active = line[3:].strip() == section
            item_id, options, recommendation = None, None, None
            continue
        if not active:
            continue
        if line.startswith("### "):
            finish()
            match = HEADING.fullmatch(line)
            item_id = match.group(1) if match else None
            options, recommendation = None, None
            if item_id is None:
                errors.append(f"line {number}: malformed item heading; use ### <ID> — <title>.")
            continue
        if item_id is None:
            continue
        name, separator, value = line.partition(":")
        if not separator:
            continue
        value = value.strip()
        if name == "Options":
            options = []
            for option in value.split("; "):
                match = OPTION.fullmatch(option.strip())
                if not match:
                    errors.append(f"line {number} ({item_id}): malformed option {option!r}; use <key> — <label>.")
                    continue
                if match.group(1) in options:
                    errors.append(f"line {number} ({item_id}): duplicate option key {match.group(1)!r}.")
                options.append(match.group(1))
        elif name == "Recommended":
            match = OPTION.fullmatch(value)
            if not match:
                recommendation = None
                errors.append(f"line {number} ({item_id}): malformed Recommended line; use Recommended: <key> — <label>.")
            else:
                recommendation = (match.group(1), number)
    finish()
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("validate", choices=["validate"])
    parser.add_argument("--file", required=True, type=Path, help="Draft markdown file to validate")
    parser.add_argument("--section", required=True, help='Section heading, for example "Needs you"')
    args = parser.parse_args()
    try:
        errors = validate(args.file.read_text(encoding="utf-8"), args.section)
    except OSError as exc:
        print(f"cannot read draft {args.file}: {exc}", file=sys.stderr)
        return 2
    if errors:
        print("Needs-you draft rejected:", file=sys.stderr)
        print("\n".join(f"- {error}" for error in errors), file=sys.stderr)
        return 1
    print(f"Needs-you draft valid: {args.file} (## {args.section})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
