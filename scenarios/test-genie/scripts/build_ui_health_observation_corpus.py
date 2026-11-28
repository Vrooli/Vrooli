#!/usr/bin/env python3
"""Freeze adjudicable claims from ui-health's browser-observation store.

The source is opened read-only. Exact duplicate evaluator inputs are collapsed;
each kept input yields only rule-specific claims backed by explicit client-side
expectations and measured geometry. The output contains the minimal geometry
fields required for the read-only replay command, not screenshots or page text.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sqlite3
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any

LAYOUT_FIELDS = (
    "expectations", "keyboardVisible", "viewport", "screen", "cssViewport",
    "shell", "header", "overlays", "controls", "body",
)
RULES = {
    "visual_mobile_toolbar_oversized": "header",
    "visual_drawable_canvas_unused": "canvas",
    "visual_nav_control_unreachable": "control",
}


def canon(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def sanitized_observation(row: dict[str, Any], layout: dict[str, Any]) -> dict[str, Any]:
    layout_subset = {key: layout[key] for key in LAYOUT_FIELDS if key in layout}
    return {
        "scenario": row.get("scenario", ""),
        "standalone": bool(row.get("standalone", False)),
        "embedded": bool(row.get("embedded", False)),
        "layoutJson": canon(layout_subset),
    }


def claims(row: dict[str, Any], layout: dict[str, Any]) -> list[tuple[str, str, str]]:
    """Return (rule, gold truth, concise evidence) only for objective contracts."""
    out = []
    expectations = layout.get("expectations", {})
    header_limit = expectations.get("headerMaxHeight")
    header_height = layout.get("header", {}).get("height")
    if isinstance(header_limit, (int, float)) and header_limit > 0 and isinstance(header_height, (int, float)):
        truth = "defect" if header_height > header_limit + 1 else "clean"
        out.append(("visual_mobile_toolbar_oversized", truth,
                    f"explicit headerMaxHeight={header_limit}px; observed header height={header_height}px"))

    viewport, screen = layout.get("viewport", {}), layout.get("screen", {})
    canvas, shell = layout.get("cssViewport", {}), layout.get("shell", {})
    if (expectations.get("fullScreenCanvas") is True and row.get("standalone") is True and
            row.get("embedded") is not True and layout.get("keyboardVisible") is not True and
            isinstance(viewport.get("width"), (int, float)) and
            isinstance(screen.get("width"), (int, float)) and
            abs(viewport["width"] - screen["width"]) <= 2 and
            isinstance(canvas.get("largeHeight"), (int, float)) and canvas["largeHeight"] > 0 and
            isinstance(screen.get("height"), (int, float)) and
            abs(canvas["largeHeight"] - screen["height"]) <= 2 and
            isinstance(shell.get("bottom"), (int, float))):
        unused = canvas["largeHeight"] - shell["bottom"]
        truth = "defect" if unused > 2 else "clean"
        out.append(("visual_drawable_canvas_unused", truth,
                    f"explicit fullScreenCanvas=true; drawable canvas={canvas['largeHeight']}px; shell bottom={shell['bottom']}px; unused={unused:.1f}px"))

    if not layout.get("overlays"):
        controls = [item for item in layout.get("controls", [])
                    if isinstance(item, dict) and isinstance(item.get("centerHit"), bool)]
        if controls:
            unreachable = sum(item["centerHit"] is False for item in controls)
            truth = "defect" if unreachable else "clean"
            out.append(("visual_nav_control_unreachable", truth,
                        f"client hit-test records={len(controls)}; misses={unreachable}"))
    return out


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--db", type=Path, required=True, help="ui-health.db, opened read-only")
    parser.add_argument("--corpus", type=Path, required=True)
    parser.add_argument("--before", type=Path, required=True)
    parser.add_argument("--summary", type=Path, required=True)
    args = parser.parse_args()

    uri = f"file:{args.db.resolve()}?mode=ro"
    db = sqlite3.connect(uri, uri=True)
    db.row_factory = sqlite3.Row
    rows = db.execute("SELECT id, scenario, client_id, received_at, payload FROM browser_layout_observations ORDER BY received_at, id").fetchall()
    db.close()
    # Deduplicate only on every value the evaluator reads. Other metadata and
    # repeated receipts do not create additional cases.
    snapshots: dict[str, tuple[dict[str, Any], dict[str, Any], list[str], str]] = {}
    duplicate_disagreements = 0
    raw_duplicates = 0
    for record in rows:
        outer = json.loads(record["payload"])
        try:
            layout = json.loads(outer["layoutJson"])
        except (KeyError, json.JSONDecodeError) as exc:
            raise ValueError(f"observation {record['id']} has invalid layoutJson") from exc
        observation = sanitized_observation(outer, layout)
        observation_key = canon(observation)
        fingerprint = hashlib.sha256(observation_key.encode()).hexdigest()
        finding_codes = sorted(f.get("code", "") for f in outer.get("findings", []))
        if fingerprint in snapshots:
            raw_duplicates += 1
            if snapshots[fingerprint][2] != finding_codes:
                duplicate_disagreements += 1
            continue
        snapshots[fingerprint] = (dict(record), layout, finding_codes, observation_key)

    cases: list[dict[str, Any]] = []
    before: list[dict[str, str]] = []
    by_rule: Counter[str] = Counter()
    by_truth: Counter[tuple[str, str]] = Counter()
    for fingerprint, (record, layout, stored_codes, _) in sorted(snapshots.items()):
        observation_outer = json.loads(record["payload"])
        obs = sanitized_observation(observation_outer, layout)
        for rule, truth, basis in claims(observation_outer, layout):
            case_id = f"uiobs:{fingerprint[:24]}:{rule}"
            source = f"scenarios/ui-health/data/ui-health.db#browser_layout_observations/{record['id']}"
            evidence = (f"{source}; observed_at={record['received_at']}; {basis}; "
                        f"layout_sha256={fingerprint}; labels come from the declared client expectation and measured geometry.")
            cases.append({
                "case_id": case_id,
                "phase": "experience",
                "rule_code": rule,
                "truth": truth,
                "evidence": evidence,
                "label_basis": "client expectation compared with actual client-measured geometry; no screenshot/page text copied",
                "source_record_id": record["id"],
                "observed_at": record["received_at"],
                "snapshot_sha256": fingerprint,
                "observation": obs,
            })
            before.append({
                "case_id": case_id,
                "phase": "experience",
                "verdict": "failed" if rule in stored_codes else "passed",
                "rule_code": rule,
                "provenance": "finding codes persisted with the original immutable observation row",
            })
            by_rule[rule] += 1
            by_truth[(rule, truth)] += 1

    if duplicate_disagreements:
        raise ValueError(f"{duplicate_disagreements} exact duplicate evaluator inputs have different stored predictions; resolve before scoring")
    if not cases:
        raise ValueError("no objective claims found in source observations")
    for path in (args.corpus, args.before, args.summary):
        path.parent.mkdir(parents=True, exist_ok=True)
    args.corpus.write_text("".join(canon(row) + "\n" for row in cases), encoding="utf-8")
    args.before.write_text("".join(canon(row) + "\n" for row in before), encoding="utf-8")
    summary = {
        "source_db": str(args.db.resolve()),
        "raw_rows": len(rows),
        "unique_evaluator_inputs": len(snapshots),
        "duplicate_rows_collapsed": raw_duplicates,
        "duplicate_prediction_disagreements": duplicate_disagreements,
        "adjudicated_claim_cases": len(cases),
        "case_counts_by_rule": dict(sorted(by_rule.items())),
        "case_counts_by_rule_and_truth": {f"{rule}/{truth}": count for (rule, truth), count in sorted(by_truth.items())},
        "label_scope": "objective rule claims only; one unique evaluator input can contribute distinct named rule claims; clustered within the git-control-tower scenario and not a fleet sample",
        "unmeasured_phases": {
            "workflow": "no matching adjudicated journey corpus extracted",
            "component-tests": "RCL reports exist, but no independent case-level human/oracle adjudication has yet been paired to phase predictions",
            "experience": "available candidate reports are historical or self-labeled and are not mixed into this UI Health corpus",
        },
    }
    args.summary.write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"cases": len(cases), "unique_evaluator_inputs": len(snapshots), "duplicate_rows_collapsed": raw_duplicates, "rules": dict(by_rule)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
