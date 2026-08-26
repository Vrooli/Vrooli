#!/usr/bin/env python3
"""Wrap catalog component exports with the shared className seam.

This is intentionally a small, idempotent migration tool. It only edits the
files named by the restyle-contract gate and wraps exported function
components; forwardRef and re-export adapters are handled explicitly by the
caller because their public types need more care.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path


IMPORT = 'import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";\n'


def body_brace(source: str, start: int) -> int:
    """Find the function body brace after the parameter list."""
    parens = 0
    i = start
    quote = None
    while i < len(source):
        char = source[i]
        if quote:
            if char == "\\":
                i += 2
                continue
            if char == quote:
                quote = None
            i += 1
            continue
        if char in "'\"`":
            quote = char
        elif char == "(":
            parens += 1
        elif char == ")":
            parens -= 1
        elif char == "{" and parens == 0:
            return i
        i += 1
    raise ValueError("could not find function body")


def matching_brace(source: str, opening: int) -> int:
    depth = 0
    i = opening
    quote = None
    line_comment = False
    block_comment = False
    while i < len(source):
        char = source[i]
        nxt = source[i + 1] if i + 1 < len(source) else ""
        if line_comment:
            if char == "\n":
                line_comment = False
            i += 1
            continue
        if block_comment:
            if char == "*" and nxt == "/":
                block_comment = False
                i += 2
                continue
            i += 1
            continue
        if quote:
            if char == "\\":
                i += 2
                continue
            if char == quote:
                quote = None
            i += 1
            continue
        if char == "/" and nxt == "/":
            line_comment = True
            i += 2
            continue
        if char == "/" and nxt == "*":
            block_comment = True
            i += 2
            continue
        if char in "'\"`":
            quote = char
        elif char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return i
        i += 1
    raise ValueError("unbalanced function body")


def add_import(source: str) -> str:
    if IMPORT.strip() in source:
        return source
    end = source.find("*/")
    if source.lstrip().startswith("/**") and end >= 0:
        end += 2
        return source[:end] + "\n" + IMPORT + source[end:]
    return IMPORT + source


def wrap(path: Path) -> bool:
    source = path.read_text()
    if "withClassName" in source or "forwardRef" in source:
        return False
    matches = list(re.finditer(r"export function (\w+)", source))
    if not matches:
        return False
    # Work from the end so offsets remain stable.
    for match in reversed(matches):
        opening = body_brace(source, match.end())
        closing = matching_brace(source, opening)
        name = match.group(1)
        replacement = f"export const {name} = withClassName(function {name}"
        source = source[: match.start()] + replacement + source[match.end() :]
        # Recompute the closing location after replacing the declaration.
        delta = len(replacement) - (match.end() - match.start())
        source = source[: closing + delta] + "});" + source[closing + delta + 1 :]
    path.write_text(add_import(source))
    return True


def main() -> int:
    changed = 0
    for raw in sys.argv[1:]:
        if wrap(Path(raw)):
            changed += 1
    print(f"wrapped {changed} export file(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
