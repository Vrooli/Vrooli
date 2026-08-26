#!/usr/bin/env python3
"""Repair the one known story-contract corruption mechanically.

The 2026-08-24 migration converted element nodes to text nodes containing the
HTML tag name. This transform is intentionally narrow: it only rewrites the
allow-listed tag-name text values and uses each story's existing expectations
as child content where available. It is safe to re-run and is kept as an
auditable migration tool rather than a runtime workaround.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

ROOT = Path(__file__).resolve().parents[1] / "library"
TAGS = {
    "div", "p", "span", "button", "nav", "section", "article", "ul", "ol", "li",
    "h1", "h2", "h3", "h4", "h5", "h6", "header", "footer", "main", "aside",
    "table", "form", "label", "img", "a", "strong", "em", "small", "code", "pre",
}


def expected_texts(story: dict[str, Any]) -> list[str]:
    values: list[str] = []
    for expectation in story.get("expect", []):
        value = expectation.get("value") or expectation.get("name")
        if isinstance(value, str) and value.strip():
            values.append(value)
    return values or ["Content"]


def rewrite(value: Any, texts: list[str], index: list[int]) -> Any:
    if isinstance(value, list):
        return [rewrite(item, texts, index) for item in value]
    if isinstance(value, dict):
        if set(value) == {"$text"} and isinstance(value["$text"], str) and value["$text"].lower() in TAGS:
            tag = value["$text"]
            text = texts[index[0] % len(texts)]
            index[0] += 1
            return {"$node": tag, "children": [{"$text": text}]}
        return {key: rewrite(child, texts, index) for key, child in value.items()}
    return value


def repair(path: Path) -> bool:
    document = json.loads(path.read_text())
    changed = False
    for story in document.get("stories", []):
        before = json.dumps(story.get("args"), sort_keys=True)
        story["args"] = rewrite(story.get("args", {}), expected_texts(story), [0])
        changed |= before != json.dumps(story["args"], sort_keys=True)
    fields = document.get("args", {}).get("fields", [])
    for field in fields:
        if "default" not in field:
            continue
        before = json.dumps(field["default"], sort_keys=True)
        field["default"] = rewrite(field["default"], ["Content"], [0])
        changed |= before != json.dumps(field["default"], sort_keys=True)
    if changed:
        path.write_text(json.dumps(document, indent=2, ensure_ascii=False) + "\n")
    return changed


def main() -> int:
    changed = [path for path in sorted(ROOT.glob("**/story.json")) if repair(path)]
    print(json.dumps({"changedFiles": [str(path) for path in changed], "count": len(changed)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
