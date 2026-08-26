#!/usr/bin/env python3
"""Add one stable catalog-rooted selector to the first uncovered control."""

from __future__ import annotations

import re
import sys
from pathlib import Path


INTERACTIVE = re.compile(r"<(button|a|input|select|textarea)\b[^>]*>", re.DOTALL)


def add_selector(asset_id: str, file_name: str) -> bool:
    path = Path(file_name)
    source = path.read_text()
    for match in INTERACTIVE.finditer(source):
        tag = match.group(0)
        if re.search(r"\bdata-testid\s*=", tag):
            continue
        replacement = tag.replace(
            f"<{match.group(1)}",
            f'<{match.group(1)} data-testid="{asset_id}"',
            1,
        )
        path.write_text(source[: match.start()] + replacement + source[match.end() :])
        return True
    return False


def main() -> int:
    changed = 0
    for line in sys.stdin:
        line = line.rstrip("\n")
        if not line:
            continue
        asset_id, file_name = line.split("\t", 1)
        if add_selector(asset_id, file_name):
            changed += 1
    print(f"added selectors to {changed} file(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
