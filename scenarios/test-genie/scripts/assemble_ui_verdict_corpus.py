#!/usr/bin/env python3
"""Assemble an evaluator-blind verdict corpus from fixed cases and review ballots.

The case packet must contain frozen case identity and an evidence reference, but
no phase predictions. Two reviewer files must cover the packet exactly. A third
adjudicator is required for disagreements or any insufficient vote. Insufficient
cases are written to a separate exclusions file and never enter the score
denominator. Output rows use the contract consumed by score_ui_verdicts.py.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path
from typing import Any

LABELS = {"defect", "clean", "insufficient"}
HASH_RE = re.compile(r"^[0-9a-f]{64}$")
IDENTITY_FIELDS = (
    "phase",
    "scenario",
    "build_identity",
    "route",
    "state",
    "viewport",
    "snapshot_sha256",
)
FORBIDDEN_CASE_FIELDS = {
    "truth",
    "verdict",
    "prediction",
    "predictions",
    "findings",
    "phase_output",
    "evaluator_result",
}


def find_prediction_fields(value: Any, path: str = "") -> list[str]:
    """Find evaluator/label fields anywhere in a packet, including nested evidence."""
    found: list[str] = []
    if isinstance(value, dict):
        for key, child in value.items():
            child_path = f"{path}.{key}" if path else str(key)
            if str(key).lower() in FORBIDDEN_CASE_FIELDS:
                found.append(child_path)
            found.extend(find_prediction_fields(child, child_path))
    elif isinstance(value, list):
        for index, child in enumerate(value):
            found.extend(find_prediction_fields(child, f"{path}[{index}]"))
    return found


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    return read_jsonl_bytes(path, path.read_bytes())


def read_jsonl_bytes(path: Path, content: bytes) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    try:
        lines = content.decode("utf-8").splitlines()
    except UnicodeDecodeError as exc:
        raise ValueError(f"{path}: input is not UTF-8: {exc}") from exc
    for number, line in enumerate(lines, 1):
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        try:
            value = json.loads(line)
        except json.JSONDecodeError as exc:
            raise ValueError(f"{path}:{number}: invalid JSON: {exc}") from exc
        if not isinstance(value, dict):
            raise ValueError(f"{path}:{number}: expected a JSON object")
        rows.append(value)
    return rows


def index_cases(rows: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    exact_units: set[tuple[str, ...]] = set()
    for row in rows:
        case_id = row.get("case_id")
        if not isinstance(case_id, str) or not case_id.strip():
            raise ValueError("case rows require a non-empty case_id")
        if case_id in result:
            raise ValueError(f"duplicate case_id: {case_id}")
        leaked = sorted(find_prediction_fields(row))
        if leaked:
            raise ValueError(f"{case_id}: frozen review packet contains evaluator output fields: {', '.join(leaked)}")
        missing = [field for field in IDENTITY_FIELDS if not isinstance(row.get(field), str) or not row[field].strip()]
        if missing:
            raise ValueError(f"{case_id}: missing case identity: {', '.join(missing)}")
        if not HASH_RE.fullmatch(row["snapshot_sha256"]):
            raise ValueError(f"{case_id}: snapshot_sha256 must be 64 lowercase hexadecimal characters")
        if not isinstance(row.get("evidence_ref"), str) or not row["evidence_ref"].strip():
            raise ValueError(f"{case_id}: evidence_ref is required")
        identity = tuple(row[field] for field in IDENTITY_FIELDS) + (str(row.get("rule_code", "")),)
        if identity in exact_units:
            raise ValueError(f"{case_id}: duplicate case identity tuple")
        exact_units.add(identity)
        result[case_id] = row
    if not result:
        raise ValueError("case packet is empty")
    return result


def index_ballots(rows: list[dict[str, Any]], label: str) -> dict[str, dict[str, Any]]:
    result: dict[str, dict[str, Any]] = {}
    for row in rows:
        case_id = row.get("case_id")
        if not isinstance(case_id, str) or not case_id.strip():
            raise ValueError(f"{label} ballots require a non-empty case_id")
        if case_id in result:
            raise ValueError(f"{label}: duplicate ballot for {case_id}")
        if not isinstance(row.get("truth"), str) or row["truth"] not in LABELS:
            raise ValueError(f"{label}:{case_id}: truth must be defect, clean, or insufficient")
        if row["truth"] == "insufficient":
            if not isinstance(row.get("insufficient_reason"), str) or not row["insufficient_reason"].strip():
                raise ValueError(f"{label}:{case_id}: insufficient labels require insufficient_reason")
        else:
            if not isinstance(row.get("reason"), str) or not row["reason"].strip():
                raise ValueError(f"{label}:{case_id}: defect/clean labels require reason")
            if not isinstance(row.get("evidence_node"), str) or not row["evidence_node"].strip():
                raise ValueError(f"{label}:{case_id}: defect/clean labels require evidence_node")
        result[case_id] = row
    return result


def require_exact_coverage(case_ids: set[str], ballots: dict[str, dict[str, Any]], label: str) -> None:
    missing = sorted(case_ids - ballots.keys())
    extra = sorted(ballots.keys() - case_ids)
    if missing or extra:
        bits = []
        if missing:
            bits.append(f"missing {len(missing)} case(s): {', '.join(missing[:5])}")
        if extra:
            bits.append(f"unknown {len(extra)} case(s): {', '.join(extra[:5])}")
        raise ValueError(f"{label} coverage does not match case packet ({'; '.join(bits)})")


def assemble(
    case_rows: list[dict[str, Any]],
    reviewer_a_rows: list[dict[str, Any]],
    reviewer_b_rows: list[dict[str, Any]],
    adjudicator_rows: list[dict[str, Any]] | None,
    rubric_ref: str,
) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    if not rubric_ref.strip():
        raise ValueError("rubric_ref is required")
    cases = index_cases(case_rows)
    reviewer_a = index_ballots(reviewer_a_rows, "reviewer A")
    reviewer_b = index_ballots(reviewer_b_rows, "reviewer B")
    case_ids = set(cases)
    require_exact_coverage(case_ids, reviewer_a, "reviewer A")
    require_exact_coverage(case_ids, reviewer_b, "reviewer B")

    adjudicator = index_ballots(adjudicator_rows or [], "adjudicator")
    a_cases = {
        case_id
        for case_id in case_ids
        if reviewer_a[case_id]["truth"] == reviewer_b[case_id]["truth"]
        and reviewer_a[case_id]["truth"] != "insufficient"
    }
    needs_adjudication = case_ids - a_cases
    extra_adjudications = adjudicator.keys() - needs_adjudication
    missing_adjudications = needs_adjudication - adjudicator.keys()
    if extra_adjudications or missing_adjudications:
        parts = []
        if missing_adjudications:
            parts.append(f"missing {len(missing_adjudications)} adjudication(s)")
        if extra_adjudications:
            parts.append(f"{len(extra_adjudications)} unnecessary adjudication(s)")
        raise ValueError("adjudicator coverage invalid (" + ", ".join(parts) + ")")

    scored: list[dict[str, Any]] = []
    excluded: list[dict[str, Any]] = []
    for case_id in sorted(cases):
        case = cases[case_id]
        a, b = reviewer_a[case_id], reviewer_b[case_id]
        adjudication = adjudicator.get(case_id)
        label = adjudication["truth"] if adjudication else a["truth"]
        ballot_count = 3 if adjudication else 2
        if label == "insufficient":
            excluded.append({
                "case_id": case_id,
                "phase": case["phase"],
                "scenario": case["scenario"],
                "snapshot_sha256": case["snapshot_sha256"],
                "reason": (adjudication or a).get("insufficient_reason", "adjudicated insufficient"),
                "reviewer_labels": [a["truth"], b["truth"]],
                "adjudicator_label": None if adjudication is None else adjudication["truth"],
            })
            continue

        row: dict[str, Any] = {
            "case_id": case_id,
            "phase": case["phase"],
            "truth": label,
            "evidence": case["evidence_ref"],
            "scenario": case["scenario"],
            "build_identity": case["build_identity"],
            "route": case["route"],
            "state": case["state"],
            "viewport": case["viewport"],
            "snapshot_sha256": case["snapshot_sha256"],
            "adjudication": {
                "method": "blind-independent",
                "phase_blinded": True,
                "reviewer_count": ballot_count,
                "rubric_ref": rubric_ref,
            },
        }
        if case.get("rule_code"):
            row["rule_code"] = case["rule_code"]
        scored.append(row)
    return scored, excluded


def write_jsonl(path: Path, rows: list[dict[str, Any]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8", newline="\n") as stream:
        for row in rows:
            stream.write(json.dumps(row, sort_keys=True, ensure_ascii=False) + "\n")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cases", type=Path, required=True, help="frozen case identity JSONL, without evaluator verdicts")
    parser.add_argument("--reviewer-a", type=Path, required=True)
    parser.add_argument("--reviewer-b", type=Path, required=True)
    parser.add_argument("--adjudicator", type=Path, help="required only for disagreements or insufficient votes")
    parser.add_argument("--rubric-ref", required=True, help="versioned rubric path or immutable URI")
    parser.add_argument("--output", type=Path, required=True, help="scorer-compatible adjudicated corpus JSONL")
    parser.add_argument("--excluded-output", type=Path, required=True, help="audit trail for insufficient cases")
    parser.add_argument("--receipt-output", type=Path, required=True, help="write hashes and counts for this assembly")
    args = parser.parse_args()
    try:
        inputs = [args.cases, args.reviewer_a, args.reviewer_b] + ([args.adjudicator] if args.adjudicator else [])
        outputs = [args.output, args.excluded_output, args.receipt_output]
        input_path_list = [path.resolve() for path in inputs]
        input_paths = set(input_path_list)
        output_paths = [path.resolve() for path in outputs]
        if len(input_paths) != len(input_path_list):
            raise ValueError("case, reviewer, and adjudicator inputs must be different paths")
        if len(set(output_paths)) != len(output_paths):
            raise ValueError("--output, --excluded-output, and --receipt-output must be different paths")
        if input_paths.intersection(output_paths):
            raise ValueError("output paths must not overwrite case or reviewer input files")
        case_bytes = args.cases.read_bytes()
        reviewer_a_bytes = args.reviewer_a.read_bytes()
        reviewer_b_bytes = args.reviewer_b.read_bytes()
        adjudicator_bytes = args.adjudicator.read_bytes() if args.adjudicator else None
        case_rows = read_jsonl_bytes(args.cases, case_bytes)
        reviewer_a_rows = read_jsonl_bytes(args.reviewer_a, reviewer_a_bytes)
        reviewer_b_rows = read_jsonl_bytes(args.reviewer_b, reviewer_b_bytes)
        adjudicator_rows = read_jsonl_bytes(args.adjudicator, adjudicator_bytes) if args.adjudicator and adjudicator_bytes is not None else None
        scored, excluded = assemble(
            case_rows,
            reviewer_a_rows,
            reviewer_b_rows,
            adjudicator_rows,
            args.rubric_ref,
        )
        write_jsonl(args.output, scored)
        write_jsonl(args.excluded_output, excluded)
        scored_bytes = args.output.read_bytes()
        excluded_bytes = args.excluded_output.read_bytes()
        receipt = {
            "schema": "ui-verdict-corpus-assembly/v1",
            "rubric_ref": args.rubric_ref,
            "case_packet_sha256": hashlib.sha256(case_bytes).hexdigest(),
            "reviewer_a_sha256": hashlib.sha256(reviewer_a_bytes).hexdigest(),
            "reviewer_b_sha256": hashlib.sha256(reviewer_b_bytes).hexdigest(),
            "adjudicator_sha256": hashlib.sha256(adjudicator_bytes).hexdigest() if adjudicator_bytes is not None else None,
            "input_cases": len(case_rows),
            "scored_cases": len(scored),
            "excluded_insufficient_cases": len(excluded),
            "scored_corpus_sha256": hashlib.sha256(scored_bytes).hexdigest(),
            "excluded_audit_sha256": hashlib.sha256(excluded_bytes).hexdigest(),
        }
        args.receipt_output.parent.mkdir(parents=True, exist_ok=True)
        args.receipt_output.write_text(json.dumps(receipt, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    except (OSError, ValueError) as exc:
        print(f"assemble_ui_verdict_corpus: {exc}", file=sys.stderr)
        return 2
    packet_hash = hashlib.sha256(case_bytes).hexdigest()
    print(f"Assembled {len(scored)} scored cases; excluded {len(excluded)} insufficient cases.")
    print(f"Frozen case packet SHA-256: {packet_hash}")
    print(f"Scored corpus: {args.output}")
    print(f"Excluded-case audit: {args.excluded_output}")
    print(f"Assembly receipt: {args.receipt_output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
