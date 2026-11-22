#!/usr/bin/env python3
"""Score paired UI-phase verdicts against an immutable adjudicated case corpus.

Corpus rows are JSONL with case_id, phase, truth (defect|clean), and evidence.
Prediction rows are JSONL with case_id, phase, verdict (passed|failed|not-evaluated).
The exact corpus is the denominator: missing, duplicate, extra, or phase-mismatched
predictions are errors. A not-evaluated result is never treated as pass; on defect
cases it counts as a false negative and it is separately reported as coverage loss.
"""

from __future__ import annotations

import argparse
import json
from collections import defaultdict
from pathlib import Path
from typing import Any

TRUTHS = {"defect", "clean"}
VERDICTS = {"passed", "failed", "not-evaluated"}


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    rows = []
    with path.open(encoding="utf-8") as stream:
        for number, line in enumerate(stream, 1):
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


def index_corpus(rows: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    indexed: dict[str, dict[str, Any]] = {}
    for row in rows:
        case_id, phase, truth = row.get("case_id"), row.get("phase"), row.get("truth")
        if not all(isinstance(item, str) and item.strip() for item in (case_id, phase)):
            raise ValueError("corpus rows require non-empty case_id and phase")
        if truth not in TRUTHS:
            raise ValueError(f"{case_id}: truth must be defect or clean")
        if not isinstance(row.get("evidence"), str) or not row["evidence"].strip():
            raise ValueError(f"{case_id}: evidence is required")
        if case_id in indexed:
            raise ValueError(f"duplicate corpus case_id: {case_id}")
        indexed[case_id] = row
    if not indexed:
        raise ValueError("corpus is empty")
    return indexed


def score(corpus_rows: list[dict[str, Any]], prediction_rows: list[dict[str, Any]]) -> dict[str, dict[str, int]]:
    corpus = index_corpus(corpus_rows)
    predictions: dict[str, dict[str, Any]] = {}
    for row in prediction_rows:
        case_id = row.get("case_id")
        if not isinstance(case_id, str) or case_id not in corpus:
            raise ValueError(f"prediction references unknown case_id: {case_id!r}")
        if case_id in predictions:
            raise ValueError(f"duplicate prediction case_id: {case_id}")
        if row.get("phase") != corpus[case_id]["phase"]:
            raise ValueError(f"{case_id}: prediction phase does not match corpus")
        if row.get("verdict") not in VERDICTS:
            raise ValueError(f"{case_id}: verdict must be passed, failed, or not-evaluated")
        predictions[case_id] = row
    missing = sorted(corpus.keys() - predictions.keys())
    if missing:
        raise ValueError(f"missing predictions for {len(missing)} corpus cases: {', '.join(missing[:8])}")

    result: dict[str, dict[str, Any]] = defaultdict(lambda: {"counts": defaultdict(int), "rules": defaultdict(lambda: defaultdict(int))})
    for case_id, case in corpus.items():
        phase, truth, verdict = case["phase"], case["truth"], predictions[case_id]["verdict"]
        rule = case.get("rule_code", "unspecified")
        for bucket in (result[phase]["counts"], result[phase]["rules"][rule]):
            bucket["cases"] += 1
            bucket["defects"] += truth == "defect"
            bucket["clean"] += truth == "clean"
            bucket["not_evaluated"] += verdict == "not-evaluated"
            bucket["evaluated"] += verdict != "not-evaluated"
            # A skipped evaluator cannot improve its score on known defects.
            bucket["false_positives"] += truth == "clean" and verdict == "failed"
            bucket["false_negatives"] += truth == "defect" and verdict != "failed"
            bucket["true_positives"] += truth == "defect" and verdict == "failed"
            bucket["true_negatives"] += truth == "clean" and verdict != "failed"
    return {
        phase: {"counts": dict(value["counts"]), "rules": {rule: dict(counts) for rule, counts in sorted(value["rules"].items())}}
        for phase, value in sorted(result.items())
    }


def validate_evaluator_manifest(path: Path, prediction_rows: list[dict[str, Any]], label: str) -> dict[str, str]:
    """Require traceable evaluator/build/command identity for a scored snapshot."""
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError(f"{label} evaluator manifest cannot be read: {exc}") from exc
    required = ("evaluator", "build_identity", "command", "execution_mode", "source_sha256")
    missing = [key for key in required if not isinstance(manifest.get(key), str) or not manifest[key].strip()]
    if missing:
        raise ValueError(f"{label} evaluator manifest missing: {', '.join(missing)}")
    if manifest["execution_mode"] not in {"fresh", "cached", "replay"}:
        raise ValueError(f"{label} evaluator manifest execution_mode must be fresh, cached, or replay")
    mismatched = [row.get("case_id", "?") for row in prediction_rows if row.get("evaluator") != manifest["evaluator"]]
    if mismatched:
        raise ValueError(f"{label} predictions do not match declared evaluator identity for {len(mismatched)} cases")
    return {key: manifest[key] for key in required}


def independence_audit(corpus_rows: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    """Count auditable independent case units without inferring independence.

    A case is eligible only when its author supplies a blind-review record and
    enough identity fields to distinguish the frozen UI context. Rows sharing a
    snapshot hash are one cluster, regardless of how many rule claims they carry.
    """
    by_phase: dict[str, dict[str, Any]] = defaultdict(
        lambda: {"claims": 0, "snapshot_clusters": set(), "scenarios": set(), "eligible_clusters": set(), "ineligible_claims": 0}
    )
    required_identity = ("scenario", "build_identity", "route", "state", "viewport", "snapshot_sha256")
    for row in index_corpus(corpus_rows).values():
        phase = row["phase"]
        audit = by_phase[phase]
        audit["claims"] += 1
        snapshot = row.get("snapshot_sha256")
        scenario = row.get("scenario") or (row.get("observation") or {}).get("scenario")
        if scenario:
            audit["scenarios"].add(scenario)
        cluster = snapshot or row["case_id"]
        audit["snapshot_clusters"].add(cluster)
        adjudication = row.get("adjudication")
        eligible = (
            all(row.get(field) for field in required_identity)
            and isinstance(adjudication, dict)
            and adjudication.get("method") == "blind-independent"
            and adjudication.get("phase_blinded") is True
            and isinstance(adjudication.get("reviewer_count"), int)
            and adjudication["reviewer_count"] >= 2
            and bool(adjudication.get("rubric_ref"))
        )
        if eligible:
            audit["eligible_clusters"].add(cluster)
        else:
            audit["ineligible_claims"] += 1
    return {
        phase: {
            "claims": value["claims"],
            "snapshot_clusters": len(value["snapshot_clusters"]),
            "scenario_clusters": len(value["scenarios"]),
            "independently_adjudicated_clusters": len(value["eligible_clusters"]),
            "ineligible_claims": value["ineligible_claims"],
        }
        for phase, value in sorted(by_phase.items())
    }


def independent_case_failures(audit: dict[str, dict[str, Any]], requested_phases: list[str], minimum: int) -> list[str]:
    """Return requested phases that do not meet an independent-case threshold."""
    return [phase for phase in requested_phases if audit.get(phase, {}).get("independently_adjudicated_clusters", 0) < minimum]


def pct(numerator: int, denominator: int) -> str:
    return "n/a" if denominator == 0 else f"{100 * numerator / denominator:.1f}%"


def report(label: str, results: dict[str, dict[str, Any]], requested_phases: list[str], snapshot_count: int) -> str:
    lines = [f"### {label}", "", "| Phase | Cases | Defect / clean | FP / clean | FP rate | FN / defect | FN rate | Not evaluated | Coverage |", "|---|---:|---:|---:|---:|---:|---:|---:|---:|"]
    for phase, count in results.items():
        count = count["counts"]
        lines.append(
            f"| {phase} | {count['cases']} | {count['defects']} / {count['clean']} "
            f"| {count['false_positives']} / {count['clean']} "
            f"| {pct(count['false_positives'], count['clean'])} "
            f"| {count['false_negatives']} / {count['defects']} "
            f"| {pct(count['false_negatives'], count['defects'])} "
            f"| {count['not_evaluated']} | {pct(count['evaluated'], count['cases'])} |"
        )
    lines += ["", "Rule-specific breakdown:", "", "| Phase | Rule | Cases | Defect / clean | FP rate | FN rate | Not evaluated |", "|---|---|---:|---:|---:|---:|---:|"]
    for phase, value in results.items():
        for rule, count in value["rules"].items():
            lines.append(f"| {phase} | {rule} | {count['cases']} | {count['defects']} / {count['clean']} | {pct(count['false_positives'], count['clean'])} | {pct(count['false_negatives'], count['defects'])} | {count['not_evaluated']} |")
    missing = sorted(set(requested_phases) - set(results))
    if missing:
        lines += ["", "Unmeasured phases (no labeled cases or paired predictions in this corpus):"]
        lines.extend(f"- `{phase}`: unmeasured; no phase-specific denominator. This is not a zero error rate." for phase in missing)
    total = sum(value["counts"]["cases"] for value in results.values())
    lines += ["", f"FP rate = clean cases incorrectly failed / all clean cases. FN rate = defective cases not failed / all defect cases; `not-evaluated` is an FN and is also shown separately. Coverage = evaluated cases / all applicable cases. These {total} rule-specific claims are clustered across {snapshot_count} unique source snapshots and are not independent fleet samples."]
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", type=Path, required=True)
    parser.add_argument("--before", type=Path, required=True, help="pre-change predictions JSONL")
    parser.add_argument("--after", type=Path, required=True, help="post-change predictions JSONL")
    parser.add_argument("--output", type=Path, help="write Markdown report; stdout by default")
    parser.add_argument("--phases", default="experience,experience-contract,workflow,component-tests", help="comma-separated canonical phases to display, including unmeasured phases")
    parser.add_argument("--minimum-independent-cases", type=int, default=0, help="fail unless every requested phase has this many distinct, explicitly blind-adjudicated snapshot clusters")
    parser.add_argument("--before-evaluator", type=Path, help="JSON manifest naming the actual before evaluator, build, command, mode, and source SHA-256")
    parser.add_argument("--after-evaluator", type=Path, help="JSON manifest naming the actual after evaluator, build, command, mode, and source SHA-256")
    args = parser.parse_args()
    cases = read_jsonl(args.corpus)
    before_rows, after_rows = read_jsonl(args.before), read_jsonl(args.after)
    before_identity = after_identity = None
    if args.before_evaluator or args.after_evaluator:
        if not args.before_evaluator or not args.after_evaluator:
            raise ValueError("--before-evaluator and --after-evaluator must be supplied together")
        before_identity = validate_evaluator_manifest(args.before_evaluator, before_rows, "before")
        after_identity = validate_evaluator_manifest(args.after_evaluator, after_rows, "after")
    before = score(cases, before_rows)
    after = score(cases, after_rows)
    phases = [phase.strip() for phase in args.phases.split(",") if phase.strip()]
    snapshots = {row.get("snapshot_sha256") for row in cases if row.get("snapshot_sha256")}
    snapshot_count = len(snapshots) or len(index_corpus(cases))
    audit = independence_audit(cases)
    audit_lines = ["| Phase | Claims | Snapshot clusters | Scenarios | Independently adjudicated clusters | Ineligible claims |", "|---|---:|---:|---:|---:|---:|"]
    for phase in sorted(set(phases) | set(audit)):
        item = audit.get(phase, {"claims": 0, "snapshot_clusters": 0, "scenario_clusters": 0, "independently_adjudicated_clusters": 0, "ineligible_claims": 0})
        audit_lines.append(f"| {phase} | {item['claims']} | {item['snapshot_clusters']} | {item['scenario_clusters']} | {item['independently_adjudicated_clusters']} | {item['ineligible_claims']} |")
    identity_section = ""
    if before_identity and after_identity:
        identity_section = "## Evaluator identities\n\n" + "\n".join(
            f"- {label}: `{identity['evaluator']}` · build `{identity['build_identity']}` · {identity['execution_mode']} · source SHA-256 `{identity['source_sha256']}` · command: `{identity['command']}`"
            for label, identity in (("Before", before_identity), ("After", after_identity))
        ) + "\n\n"
    text = f"# UI verdict trust score\n\nCorpus: `{args.corpus}` ({len(index_corpus(cases))} labeled rule claims across {snapshot_count} unique source snapshot(s))\n\n" + identity_section + "## Independence audit\n\n" + "\n".join(audit_lines) + "\n\nIndependence is never inferred from unique IDs: eligibility requires frozen scenario/build/route/state/viewport/snapshot identity plus a phase-blinded record of at least two reviewers and a separately authored rubric. Multiple claims from one snapshot count as one cluster.\n\n" + report("Before", before, phases, snapshot_count) + "\n\n" + report("After", after, phases, snapshot_count) + "\n"
    if args.minimum_independent_cases:
        deficient = independent_case_failures(audit, phases, args.minimum_independent_cases)
        if deficient:
            text += f"\nIndependent-case gate: FAIL; fewer than {args.minimum_independent_cases} independently adjudicated clusters for: {', '.join(deficient)}.\n"
            if args.output:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(text, encoding="utf-8")
            else:
                print(text, end="")
            return 2
        text += f"\nIndependent-case gate: PASS; every requested phase has at least {args.minimum_independent_cases} independently adjudicated clusters.\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(text, encoding="utf-8")
    else:
        print(text, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
