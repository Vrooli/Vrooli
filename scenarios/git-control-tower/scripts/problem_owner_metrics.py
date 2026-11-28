#!/usr/bin/env python3
"""Verify Git Control Tower's row in the portfolio owner ratchet.

This adapter is read-only. It emits counts, timestamps and a content digest,
never the PROBLEMS.md text. The steward index remains the portfolio source.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import math
import os
from pathlib import Path
import sys
import time

OWNER = "scenario:git-control-tower"
PROBLEMS_PATH = "scenarios/git-control-tower/docs/internal/PROBLEMS.md"
RATCHET_PATH = Path.home() / ".vrooli/portfolio/steward/bin/problem_ratchet.py"


def _load_ratchet():
    if str(RATCHET_PATH.parent) not in sys.path:
        sys.path.insert(0, str(RATCHET_PATH.parent))
    spec = importlib.util.spec_from_file_location("problem_ratchet", RATCHET_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load canonical ratchet module at {RATCHET_PATH}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def measure(index: dict, repo: Path, now: float | None = None) -> dict:
    """Validate and return the GCT owner row plus non-content source evidence."""
    if index.get("index_schema_version") != 3:
        raise ValueError("portfolio problem index schema v3 is required")
    active = [row for row in index.get("_active_problem_rows", [])
              if row.get("path") == PROBLEMS_PATH]
    if len(active) != 1:
        raise ValueError(f"expected one active GCT PROBLEMS source row, found {len(active)}")
    source_row = active[0]
    path = repo / PROBLEMS_PATH
    stat = path.stat()
    if stat.st_size != source_row.get("bytes"):
        raise ValueError("indexed GCT PROBLEMS byte count does not match the live source")

    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(128 * 1024), b""):
            digest.update(chunk)
    owner_record = next((row for row in index.get("records", [])
                         if row.get("id") == f"problems-md:{PROBLEMS_PATH}"), None)
    if not owner_record or owner_record.get("owner") != OWNER or owner_record.get("state") != "open":
        raise ValueError("GCT PROBLEMS source pointer is absent, misowned, or not open")
    if owner_record.get("evidence", {}).get("bytes") != stat.st_size:
        raise ValueError("GCT source pointer byte evidence does not match the live source")
    if owner_record.get("fingerprint") != digest.hexdigest():
        raise ValueError("GCT source pointer fingerprint does not match the live source")

    row = _load_ratchet().owner_measurements(index).get(OWNER)
    if not isinstance(row, dict):
        raise ValueError("canonical ratchet produced no GCT owner measurement")
    age = source_row.get("age_days")
    if isinstance(age, bool) or not isinstance(age, (int, float)) or not math.isfinite(age) or age < 0:
        raise ValueError("indexed GCT source age must be a finite non-negative number")
    observed = time.time() if now is None else now
    live_age = round(max(0.0, observed - stat.st_mtime) / 86400, 1)
    if float(age) != live_age:
        raise ValueError(f"indexed source age {age} is stale; live PROBLEMS file age is {live_age} days")
    # The ratchet retains the source scanner's fractional day precision. Verify
    # it without substituting issue dates from the document body.
    expected_age = float(age)
    if row.get("eligible_count") != 1 or row.get("bytes") != stat.st_size:
        raise ValueError("canonical ratchet count/bytes do not match GCT's one active source")
    if row.get("max_age_days") != expected_age:
        raise ValueError(f"canonical ratchet age {row.get('max_age_days')} does not equal indexed source age {expected_age}")

    return {
        "owner": OWNER,
        "measurement": row,
        "source": {
            "path": PROBLEMS_PATH,
            "bytes": stat.st_size,
            "sha256": digest.hexdigest(),
            "modified_at_epoch": int(stat.st_mtime),
            "modified_age_days_exact": round(max(0.0, observed - stat.st_mtime) / 86400, 3),
            "indexed_age_days": age,
            "age_precision": "indexed source age in fractional days",
            "age_semantics": "PROBLEMS.md modification age; issue dates inside the document are not parsed",
        },
        "closure_claims": [],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("index", type=Path, help="canonical steward state/problems.json")
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[3],
                        help="Vrooli repository root")
    args = parser.parse_args()
    try:
        index = json.loads(args.index.read_text(encoding="utf-8"))
        print(json.dumps(measure(index, args.repo.resolve()), indent=2, sort_keys=True))
    except (OSError, ValueError, RuntimeError) as error:
        print(json.dumps({"ok": False, "error": str(error)}, indent=2), file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
