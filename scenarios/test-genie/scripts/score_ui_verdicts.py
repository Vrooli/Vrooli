#!/usr/bin/env python3
"""Score paired UI-phase verdicts against an immutable adjudicated case corpus.

Rows pair by (case_id, phase), with exact subject/evaluator provenance and a
hashed ballot-assembly receipt. Missing, extra, duplicate, or mismatched records
are errors. Not-evaluated never passes and counts as a false negative on a defect.
The CLI withholds all rates until a phase has 50 independently reviewed snapshot
clusters; the JSON output carries explicit phase status and denominators.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from collections import defaultdict
from pathlib import Path
from typing import Any

TRUTHS = {"defect", "clean"}
VERDICTS = {"passed", "failed", "not-evaluated"}
PHASES = ("experience", "workflow", "component-tests")
CASE_PROVENANCE_FIELDS = ("scenario", "build_identity", "route", "state", "viewport", "snapshot_sha256")
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")


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


def index_corpus(rows: list[dict[str, Any]]) -> dict[tuple[str, str], dict[str, Any]]:
    indexed: dict[tuple[str, str], dict[str, Any]] = {}
    for row in rows:
        case_id, phase, truth = row.get("case_id"), row.get("phase"), row.get("truth")
        if not all(isinstance(item, str) and item.strip() for item in (case_id, phase)):
            raise ValueError("corpus rows require non-empty case_id and phase")
        if not isinstance(truth, str) or truth not in TRUTHS:
            raise ValueError(f"{case_id}: truth must be defect or clean")
        if not isinstance(row.get("evidence"), str) or not row["evidence"].strip():
            raise ValueError(f"{case_id}: evidence is required")
        key = (case_id, phase)
        if key in indexed:
            raise ValueError(f"duplicate corpus case_id/phase: {case_id}/{phase}")
        indexed[key] = row
    if not indexed:
        raise ValueError("corpus is empty")
    return indexed


def score(corpus_rows: list[dict[str, Any]], prediction_rows: list[dict[str, Any]]) -> dict[str, dict[str, int]]:
    corpus = index_corpus(corpus_rows)
    predictions: dict[tuple[str, str], dict[str, Any]] = {}
    case_phases: dict[str, set[str]] = defaultdict(set)
    for case_id, phase in corpus:
        case_phases[case_id].add(phase)
    for row in prediction_rows:
        case_id = row.get("case_id")
        phase = row.get("phase")
        if not isinstance(case_id, str) or not isinstance(phase, str):
            raise ValueError(f"prediction references unknown case_id: {case_id!r}")
        key = (case_id, phase)
        if key not in corpus and case_id in case_phases:
            raise ValueError(f"{case_id}: prediction phase does not match corpus")
        if key not in corpus:
            raise ValueError(f"prediction references unknown case_id/phase: {case_id!r}/{phase!r}")
        if key in predictions:
            raise ValueError(f"duplicate prediction case_id/phase: {case_id}/{phase}")
        if not isinstance(row.get("verdict"), str) or row["verdict"] not in VERDICTS:
            raise ValueError(f"{case_id}: verdict must be passed, failed, or not-evaluated")
        predictions[key] = row
    missing = sorted(corpus.keys() - predictions.keys())
    if missing:
        examples = ", ".join(f"{case_id}/{phase}" for case_id, phase in missing[:8])
        raise ValueError(f"missing predictions for {len(missing)} corpus cases: {examples}")

    result: dict[str, dict[str, Any]] = defaultdict(lambda: {"counts": defaultdict(int), "rules": defaultdict(lambda: defaultdict(int))})
    for (case_id, phase), case in corpus.items():
        truth, verdict = case["truth"], predictions[(case_id, phase)]["verdict"]
        rule = case.get("rule_code")
        if not isinstance(rule, str) or not rule.strip():
            rule = "unspecified"
        for bucket in (result[phase]["counts"], result[phase]["rules"][rule]):
            bucket["cases"] += 1
            bucket["defects"] += truth == "defect"
            bucket["clean"] += truth == "clean"
            bucket["not_evaluated"] += verdict == "not-evaluated"
            bucket["evaluated"] += verdict != "not-evaluated"
            bucket["passed"] += verdict == "passed"
            bucket["failed"] += verdict == "failed"
            # A skipped evaluator cannot improve its score on known defects.
            bucket["false_positives"] += truth == "clean" and verdict == "failed"
            bucket["false_negatives"] += truth == "defect" and verdict != "failed"
            bucket["true_positives"] += truth == "defect" and verdict == "failed"
            bucket["true_negatives"] += truth == "clean" and verdict != "failed"
    return {
        phase: {"counts": dict(value["counts"]), "rules": {rule: dict(counts) for rule, counts in sorted(value["rules"].items())}}
        for phase, value in sorted(result.items())
    }


def validate_evaluator_manifest(
    path: Path,
    corpus: dict[tuple[str, str], dict[str, Any]],
    prediction_rows: list[dict[str, Any]],
    label: str,
) -> dict[str, dict[str, str]]:
    """Require exact case, target-build, phase, and evaluator-build provenance."""
    try:
        manifest = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError(f"{label} evaluator manifest cannot be read: {exc}") from exc
    if not isinstance(manifest, dict):
        raise ValueError(f"{label} evaluator manifest must be a JSON object")
    phases = manifest.get("phases")
    if phases is None and isinstance(manifest.get("phase"), str):
        phases = {manifest["phase"]: manifest}
    if not isinstance(phases, dict) or not phases:
        raise ValueError(f"{label} evaluator manifest requires a non-empty phases map")
    normalized: dict[str, dict[str, str]] = {}
    required = ("evaluator", "build_identity", "command", "execution_mode", "source_sha256")
    for phase, identity in phases.items():
        if not isinstance(phase, str) or not phase.strip() or not isinstance(identity, dict):
            raise ValueError(f"{label} evaluator manifest has an invalid phase identity")
        missing = [key for key in required if not isinstance(identity.get(key), str) or not identity[key].strip()]
        if missing:
            raise ValueError(f"{label} manifest phase {phase} missing: {', '.join(missing)}")
        if not SHA256_RE.fullmatch(identity["source_sha256"]):
            raise ValueError(f"{label} manifest phase {phase} source_sha256 must be a lowercase SHA-256 digest")
        if identity["execution_mode"] not in {"fresh", "cached", "replay"}:
            raise ValueError(f"{label} manifest phase {phase} execution_mode must be fresh, cached, or replay")
        normalized[phase] = {key: identity[key] for key in required}

    for row in prediction_rows:
        case_id, phase = row.get("case_id"), row.get("phase")
        if not isinstance(case_id, str) or not isinstance(phase, str):
            raise ValueError(f"{label} prediction requires string case_id and phase")
        key = (case_id, phase)
        if key not in corpus:
            raise ValueError(f"{label} prediction has unknown case_id/phase: {case_id!r}/{phase!r}")
        if phase not in normalized:
            raise ValueError(f"{label} evaluator manifest has no identity for phase {phase!r}")
        identity = normalized[phase]
        case = corpus[key]
        expected = {
            "evaluator": identity["evaluator"],
            "evaluator_build_identity": identity["build_identity"],
            "command": identity["command"],
            "execution_mode": identity["execution_mode"],
            "evaluator_source_sha256": identity["source_sha256"],
            "subject_build_identity": case.get("build_identity"),
            **{field: case.get(field) for field in CASE_PROVENANCE_FIELDS if field != "build_identity"},
        }
        mismatched = [field for field, value in expected.items() if not isinstance(value, str) or not value.strip() or row.get(field) != value]
        if not SHA256_RE.fullmatch(str(row.get("evaluator_source_sha256", ""))):
            mismatched.append("evaluator_source_sha256")
        if not SHA256_RE.fullmatch(str(case.get("snapshot_sha256", ""))):
            mismatched.append("snapshot_sha256")
        if mismatched:
            raise ValueError(f"{label} prediction {case_id}/{phase} has missing or stale provenance: {', '.join(mismatched)}")
    return normalized


def validate_assembly_receipt(path: Path, corpus_path: Path, corpus: dict[tuple[str, str], dict[str, Any]]) -> dict[str, Any]:
    """Bind adjudicated truth to the ballot receipt and exact corpus bytes."""
    try:
        receipt = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError(f"assembly receipt cannot be read: {exc}") from exc
    if not isinstance(receipt, dict) or receipt.get("schema") != "ui-verdict-corpus-assembly/v1":
        raise ValueError("assembly receipt must use ui-verdict-corpus-assembly/v1")
    if receipt.get("scored_corpus_sha256") != sha256_file(corpus_path):
        raise ValueError("assembly receipt scored_corpus_sha256 does not match the exact corpus bytes")
    if receipt.get("scored_cases") != len(corpus):
        raise ValueError("assembly receipt scored_cases does not match the corpus case/phase count")
    for field in ("case_packet_sha256", "reviewer_a_sha256", "reviewer_b_sha256", "scored_corpus_sha256", "excluded_audit_sha256"):
        if not isinstance(receipt.get(field), str) or not SHA256_RE.fullmatch(receipt[field]):
            raise ValueError(f"assembly receipt missing valid {field}")
    if receipt.get("adjudicator_sha256") is not None and not SHA256_RE.fullmatch(str(receipt["adjudicator_sha256"])):
        raise ValueError("assembly receipt adjudicator_sha256 must be null or a lowercase SHA-256 digest")
    rubric_ref = receipt.get("rubric_ref")
    if not isinstance(rubric_ref, str) or not rubric_ref.strip():
        raise ValueError("assembly receipt requires rubric_ref")
    for row in corpus.values():
        adjudication = row.get("adjudication")
        if not isinstance(adjudication, dict) or adjudication.get("rubric_ref") != rubric_ref:
            raise ValueError(f"{row.get('case_id')}: corpus adjudication does not match assembly receipt rubric")
    return receipt


def independence_audit(corpus_rows: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    """Count auditable independent case units without inferring independence.

    A case is eligible only when its author supplies a blind-review record and
    enough identity fields to distinguish the frozen UI context. Rows sharing a
    snapshot hash are one cluster, regardless of how many rule claims they carry.
    """
    by_phase: dict[str, dict[str, Any]] = defaultdict(
        lambda: {"claims": 0, "snapshot_clusters": set(), "scenarios": set(), "eligible_clusters": set(), "ineligible_claims": 0}
    )
    required_identity = CASE_PROVENANCE_FIELDS
    for row in index_corpus(corpus_rows).values():
        phase = row["phase"]
        audit = by_phase[phase]
        audit["claims"] += 1
        snapshot = row.get("snapshot_sha256")
        scenario = row.get("scenario")
        if isinstance(scenario, str) and scenario.strip():
            audit["scenarios"].add(scenario)
        cluster = snapshot if isinstance(snapshot, str) and SHA256_RE.fullmatch(snapshot) else row["case_id"]
        audit["snapshot_clusters"].add(cluster)
        adjudication = row.get("adjudication")
        eligible = (
            all(isinstance(row.get(field), str) and row[field].strip() for field in required_identity)
            and isinstance(adjudication, dict)
            and adjudication.get("method") == "blind-independent"
            and adjudication.get("phase_blinded") is True
            and isinstance(adjudication.get("reviewer_count"), int)
            and not isinstance(adjudication.get("reviewer_count"), bool)
            and adjudication["reviewer_count"] >= 2
            and isinstance(adjudication.get("rubric_ref"), str)
            and bool(adjudication["rubric_ref"].strip())
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


def rate_record(numerator: int, denominator: int) -> dict[str, int | float | None]:
    return {
        "numerator": numerator,
        "denominator": denominator,
        "rate": None if denominator == 0 else numerator / denominator,
    }


def phase_summaries(
    before: dict[str, dict[str, Any]],
    after: dict[str, dict[str, Any]],
    audit: dict[str, dict[str, Any]],
    phases: list[str],
    minimum: int,
) -> dict[str, dict[str, Any]]:
    summaries: dict[str, dict[str, Any]] = {}
    empty = {"counts": {key: 0 for key in ("cases", "defects", "clean", "false_positives", "false_negatives", "true_positives", "true_negatives", "not_evaluated", "evaluated", "passed", "failed")}, "rules": {}}
    for phase in phases:
        before_phase = before.get(phase, empty)
        after_phase = after.get(phase, empty)
        eligible = audit.get(phase, {}).get("independently_adjudicated_clusters", 0)
        measured = eligible >= minimum
        summary: dict[str, Any] = {
            "status": "measured" if measured else "insufficient-eligible-cases",
            "eligible_snapshot_clusters": eligible,
            "minimum_eligible_snapshot_clusters": minimum,
            "before": before_phase["counts"],
            "after": after_phase["counts"],
        }
        if measured:
            summary["rates"] = {
                label: {
                    "false_positive": rate_record(result[phase]["counts"]["false_positives"], result[phase]["counts"]["clean"]),
                    "false_negative": rate_record(result[phase]["counts"]["false_negatives"], result[phase]["counts"]["defects"]),
                    "coverage": rate_record(result[phase]["counts"]["evaluated"], result[phase]["counts"]["cases"]),
                }
                for label, result in (("before", before), ("after", after))
            }
        else:
            summary["rates"] = None
        summaries[phase] = summary
    return summaries


def report(
    label: str,
    results: dict[str, dict[str, Any]],
    requested_phases: list[str],
    audit: dict[str, dict[str, Any]],
    minimum: int,
) -> str:
    lines = [f"### {label}", "", "| Phase | Eligible clusters | Status | Cases | FP / clean | FP rate | FN / defect | FN rate | Passed / failed / not-evaluated | Coverage |", "|---|---:|---|---:|---:|---:|---:|---:|---:|---:|"]
    for phase in requested_phases:
        value = results.get(phase)
        counts = value["counts"] if value else {key: 0 for key in ("cases", "defects", "clean", "false_positives", "false_negatives", "not_evaluated", "evaluated", "passed", "failed")}
        eligible = audit.get(phase, {}).get("independently_adjudicated_clusters", 0)
        measured = eligible >= minimum
        status = "measured" if measured else "insufficient-eligible-cases"
        fp_rate = pct(counts["false_positives"], counts["clean"]) if measured else "withheld"
        fn_rate = pct(counts["false_negatives"], counts["defects"]) if measured else "withheld"
        coverage = pct(counts["evaluated"], counts["cases"]) if measured else "withheld"
        lines.append(
            f"| {phase} | {eligible}/{minimum} | {status} | {counts['cases']} | {counts['false_positives']} / {counts['clean']} "
            f"| {fp_rate} | {counts['false_negatives']} / {counts['defects']} | {fn_rate} "
            f"| {counts['passed']} / {counts['failed']} / {counts['not_evaluated']} | {coverage} |"
        )
    lines += ["", "Rule-specific counts (rates are withheld unless the phase reaches the independent-case minimum):", "", "| Phase | Rule | Cases | FP / clean | FN / defect | Passed / failed / not-evaluated |", "|---|---|---:|---:|---:|---:|"]
    for phase in requested_phases:
        value = results.get(phase)
        if not value:
            continue
        for rule, count in value["rules"].items():
            lines.append(f"| {phase} | {rule} | {count['cases']} | {count['false_positives']} / {count['clean']} | {count['false_negatives']} / {count['defects']} | {count['passed']} / {count['failed']} / {count['not_evaluated']} |")
    total = sum(value["counts"]["cases"] for value in results.values())
    lines += ["", f"FP = clean cases incorrectly failed / clean denominator. FN = defect cases not failed / defect denominator; `not-evaluated` counts as FN and is reported separately. Coverage = evaluated / all cases. Counts are available below the minimum, but rates are withheld. These {total} rule-specific claims are clustered across {sum(item['snapshot_clusters'] for item in audit.values())} phase snapshot clusters, not independent fleet samples."]
    return "\n".join(lines)


def write_text(path: Path | None, text: str) -> None:
    if path:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")
    else:
        print(text, end="")


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_artifact_index(directory: Path, input_paths: dict[str, Path], report_path: Path, json_path: Path) -> Path:
    """Write a deterministic discovery receipt for the reviewer handoff."""
    index_path = directory / "ui-verdict-trust-artifacts.json"
    payload = {
        "schema": "ui-verdict-trust-artifacts/v1",
        "invocation": [sys.executable, *sys.argv],
        "inputs": {name: {"path": str(path.resolve()), "sha256": sha256_file(path)} for name, path in sorted(input_paths.items())},
        "artifacts": {
            "markdown_report": {"path": str(report_path.resolve()), "sha256": sha256_file(report_path)},
            "json_report": {"path": str(json_path.resolve()), "sha256": sha256_file(json_path)},
        },
        "scorer": {"path": str(Path(__file__).resolve()), "sha256": sha256_file(Path(__file__))},
    }
    index_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return index_path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", type=Path, required=True)
    parser.add_argument("--before", type=Path, required=True, help="pre-change predictions JSONL")
    parser.add_argument("--after", type=Path, required=True, help="post-change predictions JSONL")
    parser.add_argument("--assembly-receipt", type=Path, required=True, help="hash receipt produced by assemble_ui_verdict_corpus.py")
    parser.add_argument("--output", type=Path, help="write Markdown report; stdout by default")
    parser.add_argument("--json-output", type=Path, help="write machine-readable phase status, counts, denominators, and eligible rates")
    parser.add_argument("--artifact-dir", type=Path, help="write deterministic reviewer bundle (Markdown, JSON, and discoverability receipt) to this directory")
    parser.add_argument("--phases", default=",".join(PHASES), help="comma-separated UI verdict phases to display")
    parser.add_argument("--minimum-independent-cases", type=int, default=50, help="minimum distinct blind-adjudicated snapshot clusters required before rates are published (default: 50)")
    parser.add_argument("--before-evaluator", type=Path, required=True, help="phase-keyed JSON manifest with exact evaluator build, command, mode, and source SHA-256")
    parser.add_argument("--after-evaluator", type=Path, required=True, help="phase-keyed JSON manifest with exact evaluator build, command, mode, and source SHA-256")
    args = parser.parse_args()
    if args.artifact_dir and (args.output or args.json_output):
        parser.error("--artifact-dir cannot be combined with --output or --json-output")
    if args.artifact_dir:
        inputs = (args.corpus, args.before, args.after, args.assembly_receipt, args.before_evaluator, args.after_evaluator)
        outputs = tuple(args.artifact_dir / name for name in ("ui-verdict-trust.md", "ui-verdict-trust.json", "ui-verdict-trust-artifacts.json"))
        collisions = {path.resolve() for path in inputs} & {path.resolve() for path in outputs}
        if collisions:
            parser.error("--artifact-dir would overwrite scoring input: " + ", ".join(sorted(map(str, collisions))))
    if args.minimum_independent_cases <= 0:
        parser.error("--minimum-independent-cases must be greater than zero")
    phases = [phase.strip() for phase in args.phases.split(",") if phase.strip()]
    if not phases or len(phases) != len(set(phases)):
        parser.error("--phases must contain one or more unique phase names")
    try:
        cases = read_jsonl(args.corpus)
        before_rows, after_rows = read_jsonl(args.before), read_jsonl(args.after)
        corpus = index_corpus(cases)
        assembly_receipt = validate_assembly_receipt(args.assembly_receipt, args.corpus, corpus)
        corpus_phases = {phase for _, phase in corpus}
        if not corpus_phases.issubset(set(phases)):
            raise ValueError(f"--phases omits corpus phases: {', '.join(sorted(corpus_phases - set(phases)))}")
        before_identity = validate_evaluator_manifest(args.before_evaluator, corpus, before_rows, "before")
        after_identity = validate_evaluator_manifest(args.after_evaluator, corpus, after_rows, "after")
        before = score(cases, before_rows)
        after = score(cases, after_rows)
        audit = independence_audit(cases)
    except ValueError as exc:
        parser.error(str(exc))

    audit_lines = ["| Phase | Claims | Snapshot clusters | Scenarios | Independently adjudicated clusters | Ineligible claims |", "|---|---:|---:|---:|---:|---:|"]
    for phase in sorted(set(phases) | set(audit)):
        item = audit.get(phase, {"claims": 0, "snapshot_clusters": 0, "scenario_clusters": 0, "independently_adjudicated_clusters": 0, "ineligible_claims": 0})
        audit_lines.append(f"| {phase} | {item['claims']} | {item['snapshot_clusters']} | {item['scenario_clusters']} | {item['independently_adjudicated_clusters']} | {item['ineligible_claims']} |")
    summaries = phase_summaries(before, after, audit, phases, args.minimum_independent_cases)
    identities = {"before": before_identity, "after": after_identity}
    input_hashes = {
        "corpus": sha256_file(args.corpus),
        "before_predictions": sha256_file(args.before),
        "after_predictions": sha256_file(args.after),
        "before_manifest": sha256_file(args.before_evaluator),
        "after_manifest": sha256_file(args.after_evaluator),
        "assembly_receipt": sha256_file(args.assembly_receipt),
    }
    identity_lines = [f"- {label}: `{phase}` evaluator `{identity['evaluator']}` · build `{identity['build_identity']}` · {identity['execution_mode']} · source SHA-256 `{identity['source_sha256']}` · command: `{identity['command']}`" for label, by_phase in identities.items() for phase, identity in sorted(by_phase.items())]
    text = (
        f"# UI verdict trust score\n\nCorpus: `{args.corpus}` ({len(corpus)} labeled case/phase rows; SHA-256 `{input_hashes['corpus']}`)\n\n"
        + "Input SHA-256: " + ", ".join(f"{key} `{value}`" for key, value in input_hashes.items()) + "\n\n"
        "## Evaluator identities\n\n" + "\n".join(identity_lines) + "\n\n"
        "## Independence audit\n\n" + "\n".join(audit_lines)
        + "\n\nIndependence requires frozen scenario/build/route/state/viewport/snapshot identity plus a phase-blinded record of at least two reviewers and a separate rubric. Multiple claims from one snapshot count as one cluster.\n\n"
        + report("Before", before, phases, audit, args.minimum_independent_cases)
        + "\n\n" + report("After", after, phases, audit, args.minimum_independent_cases) + "\n"
    )
    deficient = [phase for phase in phases if summaries[phase]["status"] != "measured"]
    text += f"\nMeasurement status: {'INSUFFICIENT' if deficient else 'MEASURED'}; "
    if deficient:
        text += f"rates withheld below {args.minimum_independent_cases} independently adjudicated snapshot clusters for: {', '.join(deficient)}.\n"
    else:
        text += f"all requested phases meet the {args.minimum_independent_cases}-cluster minimum.\n"
    output_path = args.output
    json_path = args.json_output
    if args.artifact_dir:
        args.artifact_dir.mkdir(parents=True, exist_ok=True)
        output_path = args.artifact_dir / "ui-verdict-trust.md"
        json_path = args.artifact_dir / "ui-verdict-trust.json"
    write_text(output_path, text)
    if json_path:
        payload = {
            "schema_version": 1,
            "corpus": str(args.corpus),
            "minimum_independent_snapshot_clusters": args.minimum_independent_cases,
            "status": "insufficient-eligible-cases" if deficient else "measured",
            "phase_results": summaries,
            "independence_audit": audit,
            "evaluator_manifests": identities,
            "assembly_receipt": assembly_receipt,
            "input_sha256": input_hashes,
        }
        json_path.parent.mkdir(parents=True, exist_ok=True)
        json_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    if args.artifact_dir:
        paths = {
            "corpus": args.corpus,
            "before_predictions": args.before,
            "after_predictions": args.after,
            "assembly_receipt": args.assembly_receipt,
            "before_evaluator": args.before_evaluator,
            "after_evaluator": args.after_evaluator,
        }
        index_path = write_artifact_index(args.artifact_dir, paths, output_path, json_path)
        print(f"Reviewer artifacts: {index_path}")
    return 2 if deficient else 0


if __name__ == "__main__":
    raise SystemExit(main())
