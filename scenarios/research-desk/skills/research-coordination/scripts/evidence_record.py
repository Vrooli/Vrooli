#!/usr/bin/env python3
"""Validate and write reusable Notification Hub evidence records for a research run."""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

sys.dont_write_bytecode = True
SCHEMA = Path(__file__).resolve().parents[4] / "notification-hub/skills/notification-hub/kit/schemas/evidence.schema.json"
ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,79}")


def prepare(record: dict) -> dict:
    """Return the concise, human-facing form written to a run's evidence folder."""
    record = json.loads(json.dumps(record))
    facts = []
    for fact in record.get("facts", []):
        a, b = fact.get("a"), fact.get("b")
        label = fact.get("label", "").lower()
        if "speedup" in label and record.get("items"):
            observed = record["items"][0]
            if isinstance(observed.get("a"), (int, float)) and isinstance(observed.get("b"), (int, float)) and observed["b"]:
                a, b = 1.0, observed["a"] / observed["b"]
        if a == b or isinstance(a, (list, dict)) or isinstance(b, (list, dict)):
            continue
        if "speedup" in label and isinstance(a, (int, float)) and isinstance(b, (int, float)) and a != b:
            fact["win"] = "a" if a > b else "b"
        if fact.get("win") not in ("a", "b"):
            continue
        if isinstance(a, (int, float)) and isinstance(b, (int, float)):
            suffix = "×" if "speedup" in label else ""
            fact["a"] = f"{a:.1f}{suffix}" if suffix else f"{a:g}"
            fact["b"] = f"{b:.1f}{suffix}" if suffix else f"{b:g}"
        facts.append(fact)
    if "facts" in record:
        record["facts"] = facts
    if "items" in record:
        record["items"] = [item for item in record["items"] if item.get("a") != item.get("b")]
    if record.get("format") == "measure" and record.get("items"):
        item = record["items"][0]
        a, b = item.get("a"), item.get("b")
        unit = item.get("unit", "").lower()
        if isinstance(a, (int, float)) and isinstance(b, (int, float)) and a > b > 0 and unit in ("ms", "s", "sec", "seconds"):
            ratio = a / b
            record.setdefault("verdict", {})["headline"] = f"{ratio:.1f}× faster"
            old = record.setdefault("verdict", {}).get("text", "")
            clauses = re.split(r"(?<=[.;])\s+", old)
            distinct = [clause.strip(" .;") for clause in clauses
                        if clause.strip() and not re.search(r"\d+(?:\.\d+)?\s*[×x]\s*faster", clause, re.IGNORECASE)]
            record["verdict"]["text"] = "; ".join(distinct) + "." if distinct else "The measured values are shown above."
    return record


def validate(record: dict, schema_path: Path = SCHEMA) -> None:
    try:
        import jsonschema
    except ImportError as exc:
        raise ValueError("Python package jsonschema is required to validate evidence") from exc
    schema = json.loads(schema_path.read_text(encoding="utf-8"))
    jsonschema.Draft202012Validator(schema).validate(record)


def write(run: Path, record: dict, schema_path: Path = SCHEMA) -> Path:
    if not isinstance(record.get("workstream"), str) or not record["workstream"].strip():
        raise ValueError("new evidence records require a workstream id")
    record = prepare(record)
    validate(record, schema_path)
    ident = record["id"]
    if not ID.fullmatch(ident) or ident in (".", ".."):
        raise ValueError("evidence id must be a safe directory name")
    root = run.resolve() / "evidence"
    target = root / ident
    if not target.resolve().is_relative_to(root.resolve()):
        raise ValueError("evidence id escapes run evidence directory")
    target.mkdir(parents=True, exist_ok=True)
    (target / "media").mkdir(exist_ok=True)
    path = target / "evidence.json"
    path.write_text(json.dumps(record, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    check = sub.add_parser("validate")
    check.add_argument("record", type=Path)
    save = sub.add_parser("write")
    save.add_argument("run", type=Path)
    save.add_argument("record", type=Path, help="record JSON file")
    args = parser.parse_args()
    try:
        record = json.loads(args.record.read_text(encoding="utf-8"))
        validate(record)
        if args.command == "write":
            print(write(args.run, record))
        else:
            print(json.dumps({"ok": True, "id": record["id"]}))
    except (OSError, UnicodeError, json.JSONDecodeError, ValueError, KeyError) as exc:
        print(f"evidence record: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
