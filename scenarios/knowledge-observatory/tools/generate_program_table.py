#!/usr/bin/env python3
"""Render the Getting Started program table from Program Runtime contracts."""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[3]
SCENARIO = ROOT / "scenarios/knowledge-observatory"
DOC = SCENARIO / "docs/guides/getting-started.md"
CONTRACTS = SCENARIO / ".vrooli/program-runtime"
START = "<!-- BEGIN GENERATED PROGRAM TABLE"
END = "<!-- END GENERATED PROGRAM TABLE -->"


def cell(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", " ").strip()


def render() -> str:
    rows: list[tuple[str, str, str, str]] = []
    for path in sorted(CONTRACTS.glob("*.json")):
        contract = json.loads(path.read_text(encoding="utf-8"))
        name = contract.get("name", "")
        if not name.startswith("knowledge-observatory."):
            continue
        inputs = []
        for key, spec in contract.get("inputs", {}).items():
            kind = spec.get("type", "value")
            if spec.get("required"):
                detail = f"{kind}, required"
            elif "default" in spec:
                detail = f"{kind}, default {json.dumps(spec['default'], ensure_ascii=False)}"
            else:
                detail = f"{kind}, optional"
            inputs.append(f"`{key}` ({detail})")
        signals = contract.get("outputs", {}).get("signals", {})
        outputs = "; ".join(f"**{key}**: {value}" for key, value in signals.items())
        rows.append((name, ", ".join(inputs) if inputs else "None", contract.get("purpose", ""), outputs or "No detailed signal fields"))

    if not rows:
        raise ValueError(f"no Knowledge Observatory contracts found in {CONTRACTS}")
    lines = [
        "| Program | Contract inputs | Purpose | Output signals |",
        "|---|---|---|---|",
    ]
    lines.extend(
        f"| `{cell(name)}` | {cell(inputs)} | {cell(purpose)} | {cell(outputs)} |"
        for name, inputs, purpose, outputs in rows
    )
    return "\n".join(lines)


def source_digest() -> str:
    digest = hashlib.sha256()
    paths = sorted(CONTRACTS.glob("*.json"))
    if not paths:
        raise ValueError(f"no contract JSON files found in {CONTRACTS}")
    for path in paths:
        digest.update(path.name.encode("utf-8"))
        digest.update(b"\0")
        digest.update(path.read_bytes())
        digest.update(b"\0")
    return digest.hexdigest()


def replace_block(document: str, block: str, source_hash: str) -> str:
    if document.count(START) != 1 or document.count(END) != 1:
        raise ValueError("document must contain exactly one generated-table marker pair")
    marker_start = document.index(START)
    marker_end = document.index("\n", marker_start)
    before = document[:marker_start]
    remainder = document[marker_end + 1 :]
    _, after = remainder.split(END, 1)
    block_hash = hashlib.sha256(block.encode("utf-8")).hexdigest()
    marker = f"{START} source=.vrooli/program-runtime/*.json source_sha256={source_hash} block_sha256={block_hash} -->"
    return f"{before}{marker}\n{block}\n{END}{after}"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--write", action="store_true", help="replace the generated block")
    mode.add_argument("--check", action="store_true", help="fail if the block is stale")
    args = parser.parse_args()
    current = DOC.read_text(encoding="utf-8")
    expected = replace_block(current, render(), source_digest())
    if args.check:
        if current != expected:
            print(f"generated program table is stale: {DOC.relative_to(ROOT)}", file=sys.stderr)
            return 1
        print(f"generated program table is current ({DOC.relative_to(ROOT)})")
        return 0
    if current != expected:
        DOC.write_text(expected, encoding="utf-8")
        print(f"updated {DOC.relative_to(ROOT)}")
    else:
        print(f"already current: {DOC.relative_to(ROOT)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
