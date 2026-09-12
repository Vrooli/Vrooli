#!/usr/bin/env python3
"""Add the catalog id to the first rendered host element of each live asset."""

import json
from pathlib import Path
import re


ROOT = Path("scenarios/react-component-library/library")
RETURN = re.compile(r"\breturn\b")


def tags_after(source: str, offset: int):
    for match in re.finditer(r"<([A-Za-z][A-Za-z0-9_.]*)\b", source[offset:]):
        start = offset + match.start()
        name = match.group(1)
        previous = source[start - 1] if start else ""
        if previous.isalnum() or previous in "_$":
            continue
        if name in {"style", "Fragment", "React.Fragment"}:
            continue
        end = start + 1
        quote = None
        depth = 0
        while end < len(source):
            char = source[end]
            if quote:
                if char == "\\":
                    end += 2
                    continue
                if char == quote:
                    quote = None
            elif char in "'\"`":
                quote = char
            elif char == "{":
                depth += 1
            elif char == "}" and depth:
                depth -= 1
            elif char == ">" and depth == 0:
                yield start, end + 1, name
                break
            end += 1


def first_rendered_tag(source: str):
    for returned in RETURN.finditer(source):
        for tag in tags_after(source, returned.end()):
            return tag
    return next(tags_after(source, 0), None)


changed = []
for manifest in sorted(ROOT.glob("components/*/component.json")) + sorted(ROOT.glob("primitives/*/component.json")):
    metadata = json.loads(manifest.read_text())
    catalog_id = metadata.get("catalogId")
    latest = metadata.get("latest")
    if not catalog_id or not latest:
        continue
    version = manifest.parent / "versions" / latest
    source_path = version / f"{manifest.parent.name}.tsx"
    if not source_path.exists():
        sources = sorted(version.glob("*.tsx")) + sorted(version.glob("*.ts"))
        if not sources:
            continue
        source_path = sources[0]
    source = source_path.read_text()
    if re.search(r"data-testid\s*=\s*[\"'`][^\"'`]*" + re.escape(catalog_id), source):
        continue
    tag = first_rendered_tag(source)
    if not tag:
        continue
    start, end, name = tag
    opening = source[start:end]
    if "data-testid" in opening:
        continue
    replacement = source[start : start + 1 + len(name)] + f' data-testid="{catalog_id}"' + source[start + 1 + len(name) : end]
    source_path.write_text(source[:start] + replacement + source[end:])
    changed.append(str(source_path))

print(f"added root selectors to {len(changed)} sources")
for path in changed:
    print(path)
