#!/usr/bin/env python3
"""Validate a complete Search Hub performance evidence manifest.

Consumes per-phase receipts from compare.py and paired idle measurement reports.
This tool validates evidence structure; it does not run workloads or certify a
performance improvement.
"""

from __future__ import annotations

import argparse
import contextlib
import hashlib
import io
import json
import math
import os
import re
import sys
import tempfile
from pathlib import Path
from typing import Any

from scope_receipts import read_scope_receipt, verify_key_admission_artifact, verify_scope_receipt
from run_telemetry import verify_receipt as verify_run_telemetry_receipt
import stage_telemetry


REQUIRED_PHASES = ("cold", "warm", "concurrent", "degraded")
ALLOWED_PHASES = set(REQUIRED_PHASES) | {"model-free"}
PLAN_REQUIRED_FIELDS = {"manifest_id", "admission_receipt_id", "corpus_snapshot_id", "resource_source", "resource_group",
                        "concurrency_by_phase", "phase_receipts", "idle_reports", "evidence_receipts"}
PLAN_OPTIONAL_FIELDS = {"minimum_repeats", "component_groups"}
PLAN_FIELDS = PLAN_REQUIRED_FIELDS | PLAN_OPTIONAL_FIELDS
PAIR_INPUT_FIELDS = {"baseline_report", "candidate_report", "baseline_grade_receipt", "candidate_grade_receipt"}
PAIR_INPUT_VARIANTS = (PAIR_INPUT_FIELDS, PAIR_INPUT_FIELDS | {"baseline_telemetry", "candidate_telemetry"},
                       PAIR_INPUT_FIELDS | {"baseline_stage_trace", "candidate_stage_trace"},
                       PAIR_INPUT_FIELDS | {"baseline_telemetry", "candidate_telemetry", "baseline_stage_trace", "candidate_stage_trace"})
T975 = {2: 4.303, 3: 3.182, 4: 2.776, 5: 2.571, 6: 2.447, 7: 2.365, 8: 2.306, 9: 2.262, 10: 2.228,
        11: 2.201, 12: 2.179, 13: 2.160, 14: 2.145, 15: 2.131, 16: 2.120, 17: 2.110, 18: 2.101,
        19: 2.093, 20: 2.086, 21: 2.080, 22: 2.074, 23: 2.069, 24: 2.064, 25: 2.060,
        26: 2.056, 27: 2.052, 28: 2.048, 29: 2.045, 30: 2.042}


def student_t_975(degrees_of_freedom: int) -> float:
    """Two-sided 95% Student-t critical value, including large repeat counts."""
    if type(degrees_of_freedom) is not int or degrees_of_freedom < 2:
        raise ValueError("Student-t interval requires at least two degrees of freedom")
    if degrees_of_freedom in T975:
        return T975[degrees_of_freedom]
    # Cornish-Fisher expansion around the standard normal quantile. The terms
    # through df^-3 avoid the prior over-narrow normal fallback above df=30.
    z = 1.959963984540054
    nu = float(degrees_of_freedom)
    return (z + (z**3 + z) / (4 * nu) + (5 * z**5 + 16 * z**3 + 3 * z) / (96 * nu**2) +
            (3 * z**7 + 19 * z**5 + 17 * z**3 - 15 * z) / (384 * nu**3))


def read_json(path: Path) -> dict[str, Any]:
    data = json.loads(path.read_text())
    if not isinstance(data, dict):
        raise ValueError(f"{path}: expected object")
    return data


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def resolve(root: Path, item: str) -> Path:
    path = Path(item)
    return path if path.is_absolute() else root / path


def plan_artifact_paths(plan: dict[str, Any], root: Path) -> list[Path]:
    paths = [resolve(root, entry["receipt"]) for entries in plan["phase_receipts"].values() for entry in entries]
    paths.extend(resolve(root, path) for arm_paths in plan["idle_reports"].values() for path in arm_paths)
    paths.extend(resolve(root, item["path"]) for item in plan["evidence_receipts"])
    paths.extend(resolve(root, artifact) for entries in plan["phase_receipts"].values()
                  for entry in entries for artifact in entry["inputs"].values())
    return paths


def write_new_atomic(path: Path, content: bytes) -> None:
    """Write a new artifact atomically without replacing an existing path."""
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temp_name = tempfile.mkstemp(prefix=f".{path.name}.", suffix=".tmp", dir=path.parent)
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
        os.link(temp_name, path)
    finally:
        try:
            os.unlink(temp_name)
        except FileNotFoundError:
            pass


def operator_audit_summary(bundle: dict[str, Any], bundle_sha256: str) -> dict[str, Any]:
    phases = {}
    rollup = bundle.get("phase_budget_rollup", {}).get("phases", {})
    for phase, rows in bundle.get("phase_receipts", {}).items():
        summary = rollup.get(phase, {})
        metrics = summary.get("metrics", {})
        cpu = metrics.get("cpu_per_accepted_part", {})
        memory = metrics.get("peak_memory", {})
        telemetry = summary.get("run_attributed_telemetry", {})
        stage_latency = summary.get("stage_latency", {})
        stage_metrics = stage_latency.get("metrics", {})
        phases[phase] = {
            "repeat_count": len(rows),
            "quality_gate_all_repeats": summary.get("quality_gate_all_repeats", False),
            "cpu_per_accepted_part": {"status": cpu.get("status", "unavailable"),
                "confidence_floor_status": cpu.get("confidence_floor_status", "unavailable"),
                "conditional_threshold_met": cpu.get("conditional_threshold_met", False)},
            "cpu_per_accepted_task": {"status": metrics.get("cpu_per_accepted_task", {}).get("status", "unavailable"),
                "confidence_floor_status": metrics.get("cpu_per_accepted_task", {}).get("confidence_floor_status", "unavailable"),
                "conditional_threshold_met": metrics.get("cpu_per_accepted_task", {}).get("conditional_threshold_met", False)},
            "peak_memory": {"status": memory.get("status", "unavailable"),
                "confidence_floor_status": memory.get("confidence_floor_status", "unavailable"),
                "conditional_threshold_met": memory.get("conditional_threshold_met", False)},
            "client_latency_p95": {"status": metrics.get("client_latency_p95", {}).get("status", "unavailable"),
                "confidence_floor_status": metrics.get("client_latency_p95", {}).get("confidence_floor_status", "unavailable"),
                "conditional_threshold_met": metrics.get("client_latency_p95", {}).get("conditional_threshold_met", False)},
            "tokens_and_cache": {"status": summary.get("tokens_and_cache", {}).get("status", "unavailable"),
                "metrics": {name: value.get("status", "unavailable") for name, value in telemetry.get("metrics", {}).items()}},
            "stage_latency": {"status": stage_latency.get("status", "unavailable"),
                "resource_attribution_status": stage_latency.get("resource_attribution_status", "unavailable"),
                "metrics": {name: value.get("status", "unavailable") for name, value in stage_metrics.items()},
                "resource_metrics": {name: {
                    "status": (value.get("resource_attribution") or {}).get("status", "unavailable"),
                    "method_id": (value.get("resource_attribution") or {}).get("method_id"),
                    "scope_receipt_id": (value.get("resource_attribution") or {}).get("scope_receipt_id"),
                    "scope_uri": (value.get("resource_attribution") or {}).get("scope_uri"),
                    "scope_sha256": (value.get("resource_attribution") or {}).get("scope_sha256"),
                    "observed_scope_evidence": (value.get("resource_attribution") or {}).get("observed_scope_evidence", []),
                    "quality_gate_all_repeats": (value.get("resource_attribution") or {}).get("quality_gate_all_repeats"),
                    "metrics": {metric_name: {"status": metric_value.get("status", "unavailable"),
                        "candidate_to_baseline_ratio": metric_value.get("candidate_to_baseline_ratio")}
                        for metric_name, metric_value in ((value.get("resource_attribution") or {}).get("metrics") or {}).items()}}
                    for name, value in stage_metrics.items()}},
            "component_resource_attribution": {"status": summary.get("component_resource_attribution", {}).get("status", "unavailable"),
                "roles": {role: {"status": value.get("status", "unavailable"),
                    "quality_gate_all_repeats": value.get("quality_gate_all_repeats"),
                    "metrics": {name: {"status": metric_data.get("status", "unavailable"),
                        "candidate_to_baseline_ratio": metric_data.get("candidate_to_baseline_ratio")}
                        for name, metric_data in value.get("metrics", {}).items()}}
                    for role, value in summary.get("component_resource_attribution", {}).get("roles", {}).items()}},
        }
    idle = bundle.get("idle_check", {})
    provider = bundle.get("provider_model_attribution", {})
    budget = bundle.get("phase_budget_rollup", {})
    return {
        "schema_version": 1,
        "bundle": {"name": None, "sha256": bundle_sha256, "plan_sha256": bundle.get("provenance", {}).get("plan", {}).get("sha256")},
        "status": "bundle_replayed_requires_approved_review",
        "mechanical_replay_verified": True,
        "performance_claim_allowed": False,
        "reason": "This summary reports evidence availability and gates only; it does not establish independent quality review, owner semantic coverage, or a performance floor.",
        "provider_model_scope": {"status": provider.get("status"), "receipt_artifact_hash_present": provider.get("receipt_artifact_hash_present", False)},
        "idle": {"method": idle.get("method"),
                 "cpu_within_baseline_uncertainty": idle.get("idle_cpu_within_baseline_uncertainty"),
                 "cpu_candidate_minus_baseline_ci95": idle.get("candidate_minus_baseline_cpu_ms_per_second_ci95"),
                 "cpu_baseline_uncertainty_margin": idle.get("baseline_cpu_uncertainty_margin"),
                 "memory_within_baseline_uncertainty": idle.get("idle_memory_within_baseline_uncertainty"),
                 "memory_candidate_minus_baseline_ci95_bytes": idle.get("candidate_minus_baseline_memory_bytes_ci95"),
                 "memory_baseline_uncertainty_margin_bytes": idle.get("baseline_memory_uncertainty_margin_bytes")},
        "phase_scope": budget.get("phase_scope"),
        "phases": phases,
        "showcase_gate_matrix": bundle.get("showcase_gate_matrix"),
        "required_review": ["SH1 independent per-case grade provenance and non-regression", "provider/model scope semantic coverage",
                            "telemetry owner provenance and key admission", "approved matched-load showcase review"],
    }


def validate_plan_shape(plan: dict[str, Any]) -> None:
    """Reject ambiguous or incomplete plan contracts before reading evidence artifacts."""
    if not isinstance(plan, dict) or not PLAN_REQUIRED_FIELDS.issubset(plan) or set(plan) - PLAN_FIELDS:
        missing = PLAN_REQUIRED_FIELDS - set(plan) if isinstance(plan, dict) else PLAN_REQUIRED_FIELDS
        extra = set(plan) - PLAN_FIELDS if isinstance(plan, dict) else set()
        raise ValueError(f"plan field allowlist mismatch (missing={sorted(missing)}, extra={sorted(extra)})")
    for field in ("manifest_id", "admission_receipt_id", "corpus_snapshot_id", "resource_source", "resource_group"):
        if not isinstance(plan.get(field), str) or not plan[field].strip():
            raise ValueError(f"plan requires nonempty string {field}")
    if plan["resource_source"] not in ("cgroup_groups", "process_groups", "pss_groups"):
        raise ValueError("plan resource_source must select one supported measurement group source")
    if "minimum_repeats" in plan and (type(plan["minimum_repeats"]) is not int or plan["minimum_repeats"] < 3):
        raise ValueError("plan minimum_repeats must be an integer of at least three")
    component_groups = plan.get("component_groups")
    if component_groups is not None:
        if not isinstance(component_groups, dict) or not component_groups:
            raise ValueError("plan component_groups must be a nonempty role-to-group mapping when supplied")
        for role, name in component_groups.items():
            if not isinstance(role, str) or not re.fullmatch(r"[a-z][a-z0-9_.-]{0,63}", role):
                raise ValueError(f"invalid component group role {role!r}")
            if not isinstance(name, str) or not name.strip() or name != name.strip() or name == plan["resource_group"]:
                raise ValueError(f"component group for role {role!r} must be nonempty and differ from primary resource_group")
    phases = plan.get("phase_receipts")
    if not isinstance(phases, dict) or not set(REQUIRED_PHASES).issubset(phases) or set(phases) - ALLOWED_PHASES:
        raise ValueError(f"plan phase_receipts must contain required phases and only {sorted(ALLOWED_PHASES)}")
    concurrency = plan.get("concurrency_by_phase")
    if not isinstance(concurrency, dict) or set(concurrency) != set(phases):
        raise ValueError("plan concurrency_by_phase must exactly match phase_receipts keys")
    for phase, value in concurrency.items():
        if type(value) is not int or value < 1 or (phase == "concurrent" and value < 2):
            raise ValueError(f"plan concurrency for {phase!r} must be positive; concurrent must be at least two")
    for phase, entries in phases.items():
        if not isinstance(entries, list) or len(entries) < plan.get("minimum_repeats", 3):
            raise ValueError(f"plan phase {phase!r} requires at least minimum_repeats entries")
        for index, entry in enumerate(entries):
            if not isinstance(entry, dict) or set(entry) != {"receipt", "inputs"}:
                raise ValueError(f"plan phase {phase!r} entry {index} must contain only receipt and inputs")
            if not isinstance(entry["receipt"], str) or not entry["receipt"].strip():
                raise ValueError(f"plan phase {phase!r} entry {index} requires a receipt path")
            inputs = entry["inputs"]
            if not isinstance(inputs, dict) or set(inputs) not in PAIR_INPUT_VARIANTS:
                raise ValueError(f"plan phase {phase!r} entry {index} requires report/grade paths and optional paired token/cache and/or stage receipts")
            if any(not isinstance(value, str) or not value.strip() for value in inputs.values()):
                raise ValueError(f"plan phase {phase!r} entry {index} has an empty or invalid artifact path")
    idle_reports = plan.get("idle_reports")
    if not isinstance(idle_reports, dict) or set(idle_reports) != {"baseline", "candidate"}:
        raise ValueError("plan idle_reports must contain exactly baseline and candidate arms")
    if any(not isinstance(paths, list) or len(paths) < 3 or any(not isinstance(path, str) or not path.strip() for path in paths)
           for paths in idle_reports.values()):
        raise ValueError("each idle arm requires at least three nonempty report paths")
    evidence = plan.get("evidence_receipts")
    if not isinstance(evidence, list):
        raise ValueError("plan evidence_receipts must be a list")
    ids = set()
    for index, item in enumerate(evidence):
        if not isinstance(item, dict) or set(item) != {"receipt_id", "path", "sha256"}:
            raise ValueError(f"plan evidence_receipts entry {index} must contain only receipt_id, path, sha256")
        receipt_id, path, checksum = item.get("receipt_id"), item.get("path"), item.get("sha256")
        if not isinstance(receipt_id, str) or not receipt_id or receipt_id in ids:
            raise ValueError("plan evidence receipt IDs must be nonempty and unique")
        if not isinstance(path, str) or not path.strip():
            raise ValueError(f"plan evidence receipt {receipt_id!r} requires a path")
        if not isinstance(checksum, str) or len(checksum) != 64 or any(char not in "0123456789abcdef" for char in checksum):
            raise ValueError(f"plan evidence receipt {receipt_id!r} requires a lowercase SHA-256 digest")
        ids.add(receipt_id)


def preflight_plan(plan_path: Path) -> dict[str, Any]:
    """Inventory referenced artifacts without grading, trusting, or building a final bundle."""
    report: dict[str, Any] = {"schema_version": 1, "plan": {"name": plan_path.name, "sha256": None},
                              "status": "blocked_invalid_plan", "plan_shape": "invalid", "artifacts": [],
                              "trust_gates": {}, "claims": "none; inventory only, not full validation or measurement evidence"}
    try:
        report["plan"]["sha256"] = digest(plan_path)
        plan = read_json(plan_path)
        validate_plan_shape(plan)
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
        report["blockers"] = [f"plan: {exc}"]
        return report
    root = plan_path.parent
    report["plan_shape"] = "valid"
    blockers: list[str] = []

    def inventory(role: str, value: str, expected_sha256: str | None = None,
                  require_expected_hash: bool = False) -> dict[str, Any]:
        path = resolve(root, value)
        item: dict[str, Any] = {"role": role, "name": path.name, "expected_sha256": expected_sha256,
                                "actual_sha256": None, "status": "missing"}
        if require_expected_hash and (not isinstance(expected_sha256, str) or len(expected_sha256) != 64 or
                                      any(char not in "0123456789abcdef" for char in expected_sha256)):
            item["status"] = "missing_expected_hash"
            blockers.append(f"{role}:missing_expected_hash:{path.name}")
        if path.is_file():
            try:
                item["actual_sha256"] = digest(path)
                if item["status"] != "missing_expected_hash":
                    item["status"] = "present" if expected_sha256 is None or item["actual_sha256"] == expected_sha256 else "hash_mismatch"
            except OSError as exc:
                item["status"] = "unreadable"
                blockers.append(f"{role}:unreadable:{path.name}:{exc}")
        if item["status"] in ("missing", "hash_mismatch"):
            blockers.append(f"{role}:{item['status']}:{path.name}")
        report["artifacts"].append(item)
        return item

    telemetry_receipt_count = 0
    stage_receipt_count = 0
    scope_receipt_ids: set[str] = set()
    for phase, entries in plan["phase_receipts"].items():
        for index, entry in enumerate(entries):
            label = f"phase:{phase}:repeat:{index + 1}"
            receipt_path = resolve(root, entry["receipt"])
            pair_item = inventory(label + ":pair_receipt", entry["receipt"])
            pair_body = None
            if pair_item["status"] == "present":
                try:
                    pair_body = read_json(receipt_path)
                except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
                    pair_item["status"] = "invalid_json"
                    blockers.append(f"{label}:pair_receipt:invalid_json:{exc}")
            receipt_inputs = ((pair_body or {}).get("receipt") or {}).get("inputs") or {}
            if not isinstance(receipt_inputs, dict):
                receipt_inputs = {}
            for role, artifact_path in entry["inputs"].items():
                expected = receipt_inputs.get(role) or {}
                if not isinstance(expected, dict):
                    expected = {}
                inventory(label + ":" + role, artifact_path, expected.get("sha256"), require_expected_hash=True)
                if role.endswith("_telemetry"):
                    telemetry_receipt_count += 1
                if role.endswith("_stage_trace"):
                    stage_receipt_count += 1
            comparison = (pair_body or {}).get("comparison") or {}
            scope_id = comparison.get("serving_scope_receipt_id")
            if scope_id:
                scope_receipt_ids.add(scope_id)
    for arm, paths in plan["idle_reports"].items():
        for index, path in enumerate(paths):
            inventory(f"idle:{arm}:{index + 1}", path)
    for item in plan["evidence_receipts"]:
        inventory("authority:" + item["receipt_id"], item["path"], item["sha256"], require_expected_hash=True)

    policy_status = {}
    for name, filename in (("telemetry", "run-telemetry-trust-v1.json"),
                           ("provider_model_scope", "provider-model-scope-trust-v1.json"),
                           ("stage_telemetry", "stage-telemetry-trust-v1.json")):
        try:
            policy = read_json(Path(__file__).with_name(filename))
            keys = policy.get("keys")
            policy_status[name] = "empty_allowlist_owner_admission_pending" if keys == [] else (
                f"{len(keys)}_key_entries_require_full_manifest_verification" if isinstance(keys, list) else "invalid_policy_shape")
        except (OSError, ValueError, TypeError, json.JSONDecodeError):
            policy_status[name] = "unreadable_policy"
    report["trust_gates"] = {**policy_status,
        "telemetry_receipts_present": telemetry_receipt_count > 0,
        "telemetry_receipts_required_together": telemetry_receipt_count == 0 or telemetry_receipt_count % 2 == 0,
        "stage_receipts_present": stage_receipt_count > 0,
        "stage_receipts_required_together": stage_receipt_count == 0 or stage_receipt_count % 2 == 0,
        "serving_scope_reference_present": bool(scope_receipt_ids),
        "serving_scope_reference_count": len(scope_receipt_ids),
        "serving_scope_reference_consistent": len(scope_receipt_ids) <= 1,
        "full_signed_grade_scope_and_idle_validation": "required; preflight does not establish acceptance"}
    if telemetry_receipt_count % 2:
        blockers.append("telemetry:unpaired_receipt_paths")
    if stage_receipt_count % 2:
        blockers.append("stage_telemetry:unpaired_receipt_paths")
    trust_blocked = any(value == "empty_allowlist_owner_admission_pending" for value in policy_status.values())
    if not scope_receipt_ids:
        blockers.append("provider_model_scope:receipt_reference_missing")
    if len(scope_receipt_ids) > 1:
        blockers.append("provider_model_scope:receipt_reference_drift")
    if trust_blocked:
        blockers.append("trust:owner_key_admission_pending")
    if any(value in ("invalid_policy_shape", "unreadable_policy") for value in policy_status.values()):
        blockers.append("trust:policy_invalid_or_unreadable")
    statuses = [row["status"] for row in report["artifacts"]]
    report["artifact_status_counts"] = {status: statuses.count(status) for status in sorted(set(statuses))}
    report["blockers"] = blockers
    report["status"] = "blocked_missing_or_mismatched_artifacts" if any(status != "present" for status in statuses) else (
        "blocked_prerequisite_or_trust" if blockers else "inputs_present_pending_full_manifest_validation")
    return report


def metric(report: dict[str, Any], source: str, name: str) -> tuple[float, float, dict[str, Any]]:
    groups = report.get(source)
    if not isinstance(groups, dict) or not isinstance(groups.get(name), dict):
        raise ValueError(f"idle report lacks {source}.{name}")
    group = groups[name]
    if group.get("available") is False:
        raise ValueError(f"idle attribution {source}.{name} is unavailable")
    cpu = group.get("cpu_ms")
    memory_key = {"cgroup_groups": "peak_memory_current_bytes_sum", "process_groups": "peak_rss_kib_sum", "pss_groups": "peak_pss_kib_sum"}[source]
    memory = group.get(memory_key)
    if not isinstance(cpu, (int, float)) or cpu < 0 or not isinstance(memory, (int, float)) or memory <= 0:
        raise ValueError(f"idle attribution {source}.{name} has invalid CPU or memory")
    return float(cpu), float(memory), group


def mean_ci(values: list[float]) -> dict[str, float]:
    if len(values) < 3:
        raise ValueError("at least three paired idle samples are required")
    average = sum(values) / len(values)
    variance = sum((value - average) ** 2 for value in values) / (len(values) - 1)
    margin = student_t_975(len(values) - 1) * math.sqrt(variance / len(values))
    return {"mean": average, "ci95_low": max(0.0, average - margin), "ci95_high": average + margin, "sample_count": len(values)}


def paired_ratio(rows: list[dict[str, Any]], metric_name: str, getter: Any, floor: float | None = None,
                 provider_scope_hash_present: bool = False, require_provider_scope: bool = True) -> dict[str, Any]:
    """Summarize matched repeat ratios; missing telemetry stays explicitly unavailable."""
    pairs = []
    for row in rows:
        pair = getter(row)
        if not isinstance(pair, (tuple, list)) or len(pair) != 2:
            return {"status": "unavailable", "reason": f"missing or invalid {metric_name} in at least one pair", "sample_count": len(pairs), "expected_pair_count": len(rows), "unavailable_receipt_id": row.get("receipt_id")}
        baseline, candidate = pair
        if any(type(v) not in (int, float) or not math.isfinite(v) or v <= 0 for v in (baseline, candidate)):
            return {"status": "unavailable", "reason": f"{metric_name} requires positive finite arm values in every pair", "sample_count": len(pairs), "expected_pair_count": len(rows), "unavailable_receipt_id": row.get("receipt_id")}
        pairs.append({"receipt_id": row.get("receipt_id"), "ratio": candidate / baseline})
    ratios = [row["ratio"] for row in pairs]
    if len(ratios) < 3:
        return {"status": "unavailable", "reason": "fewer than three complete matched pairs", "sample_count": len(ratios), "expected_pair_count": len(rows)}
    interval = mean_ci(ratios)
    result: dict[str, Any] = {"status": "summarized", "candidate_to_baseline_ratio": interval,
                              "paired_ratios": pairs, "method": "two-sided 95% Student-t interval over within-phase matched candidate/baseline ratios"}
    if floor is not None:
        result["required_max_ratio"] = floor
        result["quality_eligible"] = all(row.get("quality_gate") is True for row in rows)
        result["provider_model_scope_receipt_hash_present"] = provider_scope_hash_present
        result["provider_model_scope_required_for_metric"] = require_provider_scope
        result["confidence_floor_eligible"] = result["quality_eligible"] and (provider_scope_hash_present or not require_provider_scope)
        result["conditional_threshold_met"] = result["confidence_floor_eligible"] and interval["ci95_high"] <= floor + 1e-12
        result["confidence_floor_status"] = (
            "ineligible_quality_regression" if not result["quality_eligible"] else
            "ineligible_missing_provider_model_scope_receipt" if require_provider_scope and not provider_scope_hash_present else
            ("conditional_interval_within_threshold_pending_scope_semantic_review" if require_provider_scope else
             "conditional_interval_within_threshold_pending_showcase_review") if result["conditional_threshold_met"] else
            "interval_above_threshold"
        )
    return result


def phase_budget_rollup(phases: dict[str, list[dict[str, Any]]], provider_scope_hash_present: bool) -> dict[str, Any]:
    """Keep each condition separate and avoid treating absent telemetry as zero."""
    output = {}
    metric_getters = {
        "cpu_per_accepted_part": (lambda row: row.get("cpu_part"), 0.80),
        "cpu_per_accepted_task": (lambda row: row.get("cpu_task"), 0.80),
        "peak_memory": (lambda row: row.get("peak_memory"), 0.85),
        "client_latency_p95": (lambda row: row.get("client_p95"), 0.70),
        "api_latency_p95": (lambda row: row.get("api_p95"), None),
    }
    for phase, rows in phases.items():
        measurements = []
        quality_gates = []
        for row in rows:
            measurements.append(row.get("budget_metrics", {}))
            quality_gates.append(row.get("baseline_acceptance_nonregression"))
        normalized = []
        for metrics, gate in zip(measurements, quality_gates):
            normalized.append({**metrics, "quality_gate": gate})
        budget = {name: paired_ratio(normalized, name, getter, floor, provider_scope_hash_present,
                                     require_provider_scope=name not in ("client_latency_p95", "api_latency_p95"))
                  for name, (getter, floor) in metric_getters.items()}
        telemetry = run_attributed_telemetry(rows)
        component_roles = sorted({role for row in rows for role in (row.get("component_budget_metrics") or {})})
        component_rollup = {}
        for role in component_roles:
            role_entries = [(row, (row.get("component_budget_metrics") or {}).get(role)) for row in rows]
            group_names = {item.get("group") for _, item in role_entries if item}
            group_definition_values = [item.get("group_definition") for _, item in role_entries if item]
            group_definition_keys = {json.dumps(value, sort_keys=True, separators=(",", ":")) for value in group_definition_values}
            component_metrics = {}
            for name in ("cpu_part", "cpu_task", "peak_memory"):
                normalized_rows = [{"receipt_id": row.get("receipt_id"),
                                    "quality_gate": row.get("baseline_acceptance_nonregression"),
                                    "component_metric": item.get(name) if item else None} for row, item in role_entries]
                component_metrics[name] = paired_ratio(normalized_rows, name,
                    lambda row: row.get("component_metric"), floor=None, require_provider_scope=False)
            complete = all(metric.get("status") == "summarized" for metric in component_metrics.values())
            component_rollup[role] = {
                "status": "descriptive_component_group_rollup" if complete else "component_group_rollup_unavailable",
                "group": next(iter(group_names)) if len(group_names) == 1 else None,
                "group_definition": group_definition_values[0] if len(group_definition_keys) == 1 and group_definition_values else None,
                "quality_gate_all_repeats": bool(rows) and all(row.get("baseline_acceptance_nonregression") is True for row in rows),
                "metrics": component_metrics,
                "interpretation": "diagnostic only; groups are reported separately, possible overlap is unknown, and values are not added to the primary serving total",
            }
        stage_rollup = {}
        for stage_id in stage_telemetry.STAGE_IDS:
            stage_rows = [((row.get("stage_latency") or {}).get("stages") or {}).get(stage_id, {}) for row in rows]
            ratios = [item.get("candidate_to_baseline_total_duration_ratio") for item in stage_rows]
            if not stage_rows or any(item.get("status") != "descriptive_pair_stage_latency" for item in stage_rows):
                stage_rollup[stage_id] = {"status": "unavailable_missing_or_invalid_stage_receipt",
                    "sample_count": 0, "expected_pair_count": len(rows),
                    "reason": "every pair requires verified complete case-correlated stage coverage"}
            elif any(type(value) not in (int, float) or not math.isfinite(value) or value <= 0 for value in ratios):
                stage_rollup[stage_id] = {"status": "unavailable_missing_or_invalid_stage_ratio",
                    "sample_count": 0, "expected_pair_count": len(rows)}
            elif len(ratios) < 3:
                stage_rollup[stage_id] = {"status": "unavailable_fewer_than_three_pairs",
                    "sample_count": len(ratios), "expected_pair_count": len(rows)}
            else:
                stage_rollup[stage_id] = {"status": "descriptive_repeated_pair_latency",
                    "quality_eligible": all(row.get("baseline_acceptance_nonregression") is True for row in rows),
                    "candidate_to_baseline_total_duration_ratio": mean_ci([float(value) for value in ratios]),
                    "paired_ratios": [{"receipt_id": row.get("receipt_id"), "ratio": ratio}
                                      for row, ratio in zip(rows, ratios)],
                    "method": "two-sided 95% Student-t interval over per-run paired stage-duration ratios; diagnostic only, no stage floor"}
            resource_attributions = [(stage.get("resource_attribution") or {}) for stage in stage_rows]
            resource_complete = bool(resource_attributions) and all(
                item.get("status") == "descriptive_owner_attributed_stage_resource" for item in resource_attributions)
            resource_scopes = {(item.get("method_id"), item.get("scope_receipt_id"), item.get("scope_uri"), item.get("scope_sha256")) for item in resource_attributions
                               if item.get("status") == "descriptive_owner_attributed_stage_resource"}
            observed_resource_scopes = sorted([{"method_id": method, "receipt_id": receipt_id, "uri": uri, "sha256": checksum}
                for method, receipt_id, uri, checksum in resource_scopes], key=lambda value: (value["receipt_id"], value["sha256"]))
            resource_scope_stable = resource_complete and len(resource_scopes) == 1
            resource_metrics = {}
            for resource_name in ("cpu_ms", "cpu_ms_per_accepted_part", "cpu_ms_per_accepted_task", "peak_memory_bytes"):
                resource_inputs = [{"receipt_id": row.get("receipt_id"),
                    "resource_pair": (stage.get("resource_attribution") or {}).get(resource_name) if resource_scope_stable else None}
                    for row, stage in zip(rows, stage_rows)]
                resource_metrics[resource_name] = paired_ratio(resource_inputs, resource_name,
                    lambda item: (item.get("resource_pair", {}).get("baseline"), item.get("resource_pair", {}).get("candidate"))
                        if isinstance(item.get("resource_pair"), dict) else None,
                    floor=None, require_provider_scope=False)
            stage_rollup[stage_id]["resource_attribution"] = {
                "status": "descriptive_repeated_pair_stage_resource" if resource_scope_stable and all(item.get("status") == "summarized" for item in resource_metrics.values()) else "unavailable_stage_resource_method_or_scope_drift" if resource_complete and not resource_scope_stable else "unavailable_stage_cpu_memory_not_measured",
                "method_id": next(iter(resource_scopes))[0] if resource_scope_stable else None,
                "scope_receipt_id": next(iter(resource_scopes))[1] if resource_scope_stable else None,
                "scope_uri": next(iter(resource_scopes))[2] if resource_scope_stable else None,
                "scope_sha256": next(iter(resource_scopes))[3] if resource_scope_stable else None,
                "observed_scope_evidence": observed_resource_scopes,
                "metrics": resource_metrics,
                "quality_gate_all_repeats": bool(rows) and all(row.get("baseline_acceptance_nonregression") is True for row in rows),
                "interpretation": "owner-attributed diagnostics only; separate stage memory peaks are not summed and stage values do not replace the primary total-cost floor"}
        telemetry_available = all(item.get("status") == "summarized" for item in telemetry["metrics"].values())
        output[phase] = {
            "repeat_count": len(rows),
            "quality_gate_all_repeats": bool(rows) and all(value is True for value in quality_gates),
            "metrics": budget,
            "tokens_and_cache": {"status": "run_attributed_summary_available" if telemetry_available else "run_attributed_summary_unavailable; ambient snapshots retained separately and never converted to serving CPU",
                                 "repeats": [{"receipt_id": row["receipt_id"], "evidence": row.get("separate_evidence"), "scope": row.get("telemetry_scope")} for row in rows]},
            "run_attributed_telemetry": telemetry,
            "stage_latency": {"status": "complete_descriptive_stage_latency" if all(item.get("status") == "descriptive_repeated_pair_latency" for item in stage_rollup.values()) else "stage_latency_incomplete_or_unavailable",
                              "metrics": stage_rollup,
                              "resource_attribution_status": "descriptive_stage_resource_diagnostics_available" if any(item.get("resource_attribution", {}).get("status") == "descriptive_repeated_pair_stage_resource" for item in stage_rollup.values()) else "unavailable_stage_cpu_memory_not_measured",
                              "interpretation": "latency and optional owner-attributed resource diagnostics only; no stage floor or replacement for the primary total-cost measurement"},
            "component_resource_attribution": {
                "status": "descriptive_component_groups_available" if component_roles and all(item["status"] == "descriptive_component_group_rollup" for item in component_rollup.values()) else "unavailable_component_groups_not_configured" if not component_roles else "component_group_rollup_incomplete",
                "roles": component_rollup,
                "interpretation": "selected primary group remains the total-cost input; component groups are never summed and do not establish group disjointness"},
        }
    return {"phase_scope": "independent; cold, warm, concurrent, degraded, and optional model-free values are never pooled",
            "provider_model_scope_receipt_hash_present": provider_scope_hash_present,
            "acceptance": "descriptive_audit_rollup_only; floors require complete measurements, quality eligibility, provider/model scope receipt hash, SH1 review, and approved showcase review",
            "phases": output}


def showcase_gate_matrix(budget_rollup: dict[str, Any], provider_model_attribution: dict[str, Any],
                         idle_check: dict[str, Any]) -> dict[str, Any]:
    """Combine floor evidence statuses without turning conditional results into claims."""
    floor_metrics = ("cpu_per_accepted_part", "cpu_per_accepted_task", "peak_memory", "client_latency_p95")
    phases: dict[str, Any] = {}
    blockers: list[str] = []
    reporting_gaps: list[str] = []
    all_phase_gates = True
    for phase, phase_data in budget_rollup.get("phases", {}).items():
        quality_ok = phase_data.get("quality_gate_all_repeats") is True
        metrics = phase_data.get("metrics", {})
        gates = {}
        for name in floor_metrics:
            metric = metrics.get(name, {})
            if not quality_ok:
                status = "ineligible_quality_regression"
            elif metric.get("status") != "summarized":
                status = "unavailable_missing_or_invalid_measurements"
            elif metric.get("provider_model_scope_required_for_metric") and not metric.get("provider_model_scope_receipt_hash_present"):
                status = "ineligible_missing_provider_model_scope_receipt"
            elif metric.get("conditional_threshold_met") is not True:
                status = "candidate_threshold_not_met"
            else:
                status = "conditional_threshold_met_pending_review"
            gates[name] = {"status": status, "required_max_ratio": metric.get("required_max_ratio"),
                           "candidate_to_baseline_ratio": metric.get("candidate_to_baseline_ratio"),
                           "quality_eligible": metric.get("quality_eligible", quality_ok),
                           "conditional_threshold_met": status == "conditional_threshold_met_pending_review"}
            if gates[name]["conditional_threshold_met"] is not True:
                blockers.append(f"phase:{phase}:{name}:{status}")
        phase_ok = quality_ok and all(gate["conditional_threshold_met"] for gate in gates.values())
        all_phase_gates = all_phase_gates and phase_ok
        telemetry_status = phase_data.get("tokens_and_cache", {}).get("status", "unavailable")
        stage_status = phase_data.get("stage_latency", {}).get("status", "unavailable")
        component_status = phase_data.get("component_resource_attribution", {}).get("status", "unavailable")
        phases[phase] = {"quality_gate_all_repeats": quality_ok, "cost_and_latency_gates": gates,
                         "token_cache_reporting": telemetry_status, "stage_latency_attribution": stage_status,
                         "component_resource_attribution": component_status}
        if not quality_ok:
            blockers.append(f"phase:{phase}:quality:nonregression_gate_failed")
        if telemetry_status != "run_attributed_summary_available":
            reporting_gaps.append(f"phase:{phase}:run_correlated_token_cache_metrics_unavailable")
        if stage_status != "complete_descriptive_stage_latency":
            reporting_gaps.append(f"phase:{phase}:run_correlated_stage_latency_unavailable")

    scope_verification = provider_model_attribution.get("verification") or {}
    admission_verification = scope_verification.get("key_admission_verification") or {}
    scope_verified = (scope_verification.get("signature_status") == "verified" and
                      admission_verification.get("approval_status") == "approved")
    if not scope_verified:
        blockers.append("provider_model_scope:owner_key_or_scope_verification_unavailable")
    idle_cpu = idle_check.get("idle_cpu_within_baseline_uncertainty")
    idle_memory = idle_check.get("idle_memory_within_baseline_uncertainty")
    if idle_cpu is not True:
        blockers.append("idle:cpu_outside_or_missing_baseline_uncertainty")
    if idle_memory is not True:
        blockers.append("idle:memory_outside_or_missing_baseline_uncertainty")
    all_cost_quality_gates = all_phase_gates and scope_verified and idle_cpu is True and idle_memory is True
    reporting_complete = not reporting_gaps
    status = ("conditional_thresholds_and_reporting_met_pending_review" if all_cost_quality_gates and reporting_complete else
              "conditional_cost_thresholds_met_telemetry_reporting_incomplete" if all_cost_quality_gates else
              "incomplete_or_blocked")
    return {
        "status": status,
        "performance_claim_allowed": False,
        "conditional_cost_quality_and_idle_gates_met": all_cost_quality_gates,
        "telemetry_reporting_complete": reporting_complete,
        "thresholds": {"cpu_per_accepted_part_ratio_max": 0.80, "cpu_per_accepted_task_ratio_max": 0.80,
                        "peak_memory_ratio_max": 0.85, "client_latency_p95_ratio_max": 0.70},
        "phases": phases,
        "provider_model_scope_signature_and_admission_verified": scope_verified,
        "idle_cpu_within_baseline_uncertainty": idle_cpu,
        "idle_memory_within_baseline_uncertainty": idle_memory,
        "blockers": blockers,
        "reporting_gaps": reporting_gaps,
        "review_required_even_when_all_thresholds_met": ["SH1 grade provenance and independent review",
            "provider/model coverage semantics", "telemetry completeness and owner provenance",
            "approved matched-load showcase review"],
    }


def signed_mean_ci(values: list[float]) -> dict[str, float]:
    if len(values) < 3:
        raise ValueError("at least three paired telemetry values are required")
    average = sum(values) / len(values)
    variance = sum((value - average) ** 2 for value in values) / (len(values) - 1)
    margin = student_t_975(len(values) - 1) * math.sqrt(variance / len(values))
    return {"mean": average, "ci95_low": average - margin, "ci95_high": average + margin, "sample_count": len(values)}


def run_attributed_telemetry(rows: list[dict[str, Any]]) -> dict[str, Any]:
    """Only compare token/cache inputs explicitly correlated to this exact run and query cohort."""
    per_metric: dict[str, list[dict[str, Any]]] = {"input_tokens_per_accepted_part": [], "output_tokens_per_accepted_part": [], "cache_hit_rate_delta": []}
    unavailable: dict[str, str] = {}
    for row in rows:
        evidence_bundle = row.get("separate_evidence") or {}
        evidence = (evidence_bundle.get("run_attributed", evidence_bundle) or {}) if isinstance(evidence_bundle, dict) else {}
        for metric_name, source_keys in (("input_tokens_per_accepted_part", ("baseline_tokens", "candidate_tokens")),
                                         ("output_tokens_per_accepted_part", ("baseline_tokens", "candidate_tokens")),
                                         ("cache_hit_rate_delta", ("baseline_cache", "candidate_cache"))):
            if metric_name in unavailable:
                continue
            arms = [evidence.get(key) for key in source_keys]
            if any(not isinstance(item, dict) or item.get("available") is not True or item.get("signature_verified") is not True or item.get("attribution_status") != "run_correlated" or item.get("run_id") != row.get("run_id") or item.get("case_query_set_sha256") != row.get("grade_case_query_set_sha256") for item in arms):
                unavailable[metric_name] = "requires complete evidence bound to this pair run_id and frozen case/query-set digest"
                continue
            if metric_name.startswith("input_tokens") or metric_name.startswith("output_tokens"):
                field = "input_tokens" if metric_name.startswith("input_tokens") else "output_tokens"
                counts = row.get("accepted_parts") or {}
                b_total, c_total = arms[0].get(field), arms[1].get(field)
                b_den, c_den = counts.get("baseline"), counts.get("candidate")
                if any(type(value) is not int or value < 0 for value in (b_total, c_total)) or any(type(value) is not int or value <= 0 for value in (b_den, c_den)):
                    unavailable[metric_name] = "requires nonnegative integer tokens and positive accepted-part denominators"
                    continue
                per_metric[metric_name].append({"receipt_id": row.get("receipt_id"), "baseline": b_total / b_den, "candidate": c_total / c_den})
            else:
                def rate(item: dict[str, Any]) -> float | None:
                    hits, misses = item.get("resolver_cache_hit_delta"), item.get("resolver_cache_miss_delta")
                    if type(hits) is not int or type(misses) is not int or hits < 0 or misses < 0 or hits + misses == 0:
                        return None
                    return hits / (hits + misses)
                base_rate, candidate_rate = rate(arms[0]), rate(arms[1])
                if base_rate is None or candidate_rate is None:
                    unavailable[metric_name] = "requires nonnegative run-attributed resolver hit/miss counts with a nonzero denominator"
                    continue
                per_metric[metric_name].append({"receipt_id": row.get("receipt_id"), "delta": candidate_rate - base_rate})
    output: dict[str, Any] = {}
    for name, pairs in per_metric.items():
        if name in unavailable or len(pairs) < 3:
            output[name] = {"status": "unavailable", "reason": unavailable.get(name, "requires at least three complete paired runs"), "available_pair_count": len(pairs), "expected_pair_count": len(rows)}
            continue
        if name == "cache_hit_rate_delta":
            values = [pair["delta"] for pair in pairs]
            output[name] = {"status": "summarized", "candidate_minus_baseline_rate": signed_mean_ci(values), "paired_deltas": pairs,
                            "method": "within-run candidate minus baseline resolver-cache hit-rate differences; units are rate points"}
        else:
            values = [pair["candidate"] / pair["baseline"] for pair in pairs]
            output[name] = {"status": "summarized", "candidate_to_baseline_ratio": mean_ci(values), "paired_ratios": [{"receipt_id": pair["receipt_id"], "ratio": ratio} for pair, ratio in zip(pairs, values)],
                            "method": "paired within-run token count per independently accepted part; 95% Student-t interval over candidate/baseline ratios"}
    return {"quality_gate_all_repeats": bool(rows) and all(row.get("baseline_acceptance_nonregression") is True for row in rows), "metrics": output,
            "eligibility_rule": "all telemetry must be explicitly run_correlated and bind exact run_id plus frozen case/query-set digest; this source is currently unavailable for stock Insights/Gateway window aggregates"}


def validate_idle(plan: dict[str, Any], root: Path, source: str, group: str, admission: str, corpus: str) -> dict[str, Any]:
    idle = plan.get("idle_reports")
    if not isinstance(idle, dict):
        raise ValueError("plan requires idle_reports baseline/candidate paths")
    arms = {}
    for arm in ("baseline", "candidate"):
        paths = idle.get(arm)
        if not isinstance(paths, list) or len(paths) < 3:
            raise ValueError(f"idle_reports.{arm} requires at least three reports")
        rows = []
        for item in paths:
            path = resolve(root, item)
            report = read_json(path)
            context = report.get("execution_context") or {}
            workload = report.get("workload")
            if report.get("phase") != "idle" or not isinstance(workload, dict) or workload.get("cases") != 0:
                raise ValueError(f"{path}: idle phase must have zero workload cases")
            if context.get("admission_receipt_id") != admission or context.get("corpus_snapshot_id") != corpus:
                raise ValueError(f"{path}: admission/corpus identity differs from manifest")
            if not all(context.get(key) for key in ("build_id", "state_receipt_id", "fault_schedule_id")):
                raise ValueError(f"{path}: idle report lacks build/state/fault context")
            cpu, memory, _ = metric(report, source, group)
            seconds = report.get("idle_seconds")
            if not isinstance(seconds, (int, float)) or seconds <= 0:
                raise ValueError(f"{path}: idle_seconds must be positive")
            started, ended = report.get("run_started_at_unix_ms"), report.get("run_ended_at_unix_ms")
            if not isinstance(started, (int, float)) or not isinstance(ended, (int, float)) or ended <= started:
                raise ValueError(f"{path}: idle report needs a valid measurement time window")
            pattern_key = {"cgroup_groups": "cgroup_group_patterns", "process_groups": "process_group_patterns", "pss_groups": "pss_group_patterns"}[source]
            patterns = report.get(pattern_key, [])
            selected = [item for item in patterns if isinstance(item, dict) and item.get("name") == group]
            if len(selected) != 1:
                raise ValueError(f"{path}: selected resource group is not uniquely declared")
            memory_scale = 1024 if source in ("process_groups", "pss_groups") else 1
            rows.append({"path": path.name, "sha256": digest(path), "run_id": report.get("run_id"), "build_id": context["build_id"], "state_receipt_id": context["state_receipt_id"], "fault_schedule_id": context["fault_schedule_id"], "serving_scope_receipt_id": context.get("serving_scope_receipt_id"), "seconds": float(seconds), "sample_interval_ms": report.get("sample_interval_ms"), "started_at_unix_ms": started, "ended_at_unix_ms": ended, "cpu_ms_per_second": cpu / seconds, "memory_source_value": memory, "memory_bytes": memory * memory_scale, "memory_source_unit": "KiB" if memory_scale == 1024 else "bytes", "group_definition": selected[0]})
        arms[arm] = rows
    baseline_by_run = {row["run_id"]: row for row in arms["baseline"]}
    candidate_by_run = {row["run_id"]: row for row in arms["candidate"]}
    if None in baseline_by_run or baseline_by_run.keys() != candidate_by_run.keys():
        raise ValueError("baseline/candidate idle reports must have matching unique run IDs")
    if len(baseline_by_run) != len(arms["baseline"]) or len(candidate_by_run) != len(arms["candidate"]):
        raise ValueError("idle run IDs must be unique")
    cpu_base, memory_base = [], []
    cpu_candidate, memory_candidate = [], []
    cpu_deltas, memory_deltas = [], []
    for run_id in baseline_by_run:
        b, c = baseline_by_run[run_id], candidate_by_run[run_id]
        for field in ("seconds", "sample_interval_ms", "group_definition"):
            if b[field] != c[field]:
                raise ValueError(f"paired idle runs differ in {field}")
        if not (b["ended_at_unix_ms"] <= c["started_at_unix_ms"] or c["ended_at_unix_ms"] <= b["started_at_unix_ms"]):
            raise ValueError("paired idle arms must run sequentially, not concurrently")
        cpu_base.append(b["cpu_ms_per_second"])
        memory_base.append(b["memory_bytes"])
        cpu_candidate.append(c["cpu_ms_per_second"])
        memory_candidate.append(c["memory_bytes"])
        cpu_deltas.append(c["cpu_ms_per_second"] - b["cpu_ms_per_second"])
        memory_deltas.append(c["memory_bytes"] - b["memory_bytes"])
    cpu_ci, memory_ci = mean_ci(cpu_base), mean_ci(memory_base)
    candidate_cpu_mean = sum(cpu_candidate) / len(cpu_candidate)
    candidate_memory_mean = sum(memory_candidate) / len(memory_candidate)
    cpu_delta_ci, memory_delta_ci = signed_mean_ci(cpu_deltas), signed_mean_ci(memory_deltas)
    # Treat the baseline's 95% confidence half-width as an equivalence margin.
    # Require the entire paired candidate-minus-baseline interval to fit inside
    # that margin; a candidate mean alone can hide substantial run-to-run noise.
    cpu_margin = (cpu_ci["ci95_high"] - cpu_ci["ci95_low"]) / 2
    memory_margin = (memory_ci["ci95_high"] - memory_ci["ci95_low"]) / 2
    cpu_within_uncertainty = cpu_delta_ci["ci95_low"] >= -cpu_margin and cpu_delta_ci["ci95_high"] <= cpu_margin
    memory_within_uncertainty = memory_delta_ci["ci95_low"] >= -memory_margin and memory_delta_ci["ci95_high"] <= memory_margin
    return {
        "resource_source": source,
        "resource_group": group,
        "baseline_reports": arms["baseline"],
        "candidate_reports": arms["candidate"],
        "baseline_cpu_ms_per_second_ci95": cpu_ci,
        "candidate_cpu_ms_per_second_mean": candidate_cpu_mean,
        "candidate_minus_baseline_cpu_ms_per_second_ci95": cpu_delta_ci,
        "baseline_cpu_uncertainty_margin": cpu_margin,
        "idle_cpu_within_baseline_uncertainty": cpu_within_uncertainty,
        "baseline_memory_ci95": memory_ci,
        "baseline_memory_ci95_unit": "bytes",
        "candidate_memory_mean": candidate_memory_mean,
        "candidate_memory_mean_unit": "bytes",
        "candidate_minus_baseline_memory_bytes_ci95": memory_delta_ci,
        "baseline_memory_uncertainty_margin_bytes": memory_margin,
        "idle_memory_within_baseline_uncertainty": memory_within_uncertainty,
        "method": "paired candidate-minus-baseline 95% Student-t interval must fit wholly within the baseline 95% confidence half-width equivalence margin; minimum three matched sequential pairs",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--plan", required=True, type=Path, help="reviewed JSON plan listing phase receipts and idle report artifacts")
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--output", type=Path, help="write a newly validated evidence bundle")
    action.add_argument("--verify-bundle", type=Path, help="replay the plan and verify this bundle byte-for-byte")
    action.add_argument("--preflight-report", type=Path, help="write a non-authoritative artifact/readiness inventory")
    parser.add_argument("--audit-report", type=Path, help="with --output, emit a compact summary after bundle replay verification")
    args = parser.parse_args()
    if args.audit_report is not None:
        if args.output is None:
            parser.error("--audit-report requires --output")
        targets = (args.output, args.audit_report)
        if any(path.exists() or path.is_symlink() for path in targets):
            parser.error("audit bundle and report paths must be new")
        if args.output.resolve() == args.audit_report.resolve() or any(path.resolve() == args.plan.resolve() for path in targets):
            parser.error("audit outputs must be distinct from each other and the plan")
        try:
            plan = read_json(args.plan)
            validate_plan_shape(plan)
            protected = {args.plan.resolve(), Path(__file__).resolve(),
                Path(__file__).with_name("provider-model-scope-trust-v1.json").resolve(),
                Path(__file__).with_name("run-telemetry-trust-v1.json").resolve()}
            protected.update(path.resolve() for path in plan_artifact_paths(plan, args.plan.parent))
            if any(path.resolve() in protected for path in targets):
                parser.error("audit outputs must not overwrite the plan, trust policy, or a referenced artifact")
            original_argv = sys.argv
            with tempfile.TemporaryDirectory(prefix="sh4-showcase-audit-") as temp_dir:
                staged_bundle = Path(temp_dir) / "validated-bundle.json"
                sys.argv = [original_argv[0], "--plan", str(args.plan), "--output", str(staged_bundle)]
                with contextlib.redirect_stdout(io.StringIO()):
                    main()
                sys.argv = [original_argv[0], "--plan", str(args.plan), "--verify-bundle", str(staged_bundle)]
                with contextlib.redirect_stdout(io.StringIO()):
                    main()
                bundle_bytes = staged_bundle.read_bytes()
            bundle = json.loads(bundle_bytes)
            bundle_hash = hashlib.sha256(bundle_bytes).hexdigest()
            summary = operator_audit_summary(bundle, bundle_hash)
            summary["bundle"]["name"] = args.output.name
            summary["summary_file"] = args.audit_report.name
            bundle_written = False
            try:
                write_new_atomic(args.output, bundle_bytes)
                bundle_written = True
                write_new_atomic(args.audit_report, (json.dumps(summary, indent=2) + "\n").encode())
            except OSError:
                if bundle_written:
                    args.output.unlink(missing_ok=True)
                raise
            print(json.dumps({"bundle": str(args.output), "summary": str(args.audit_report),
                              "mechanical_replay_verified": True, "performance_claim_allowed": False,
                              "sha256": bundle_hash}, separators=(",", ":")))
            return 0
        except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            parser.error(str(exc))
        finally:
            if "original_argv" in locals():
                sys.argv = original_argv
    if args.preflight_report is not None:
        report_path = args.preflight_report
        if report_path.exists() or report_path.is_symlink() or report_path.resolve() == args.plan.resolve():
            parser.error("preflight report path must be new and must not overwrite the plan")
        try:
            report = preflight_plan(args.plan)
            if report["plan_shape"] == "valid":
                plan = read_json(args.plan)
                paths = plan_artifact_paths(plan, args.plan.parent)
                if report_path.resolve() in {path.resolve() for path in paths}:
                    parser.error("preflight report path must not overwrite a referenced source artifact")
            report_path.parent.mkdir(parents=True, exist_ok=True)
            report_path.write_text(json.dumps(report, indent=2) + "\n")
            print(json.dumps({"preflight_report": str(report_path), "status": report["status"],
                              "artifact_status_counts": report.get("artifact_status_counts", {}),
                              "blocker_count": len(report.get("blockers", []))}, separators=(",", ":")))
            return 0
        except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            parser.error(str(exc))
    if args.verify_bundle is not None:
        if not args.verify_bundle.is_file():
            parser.error(f"bundle does not exist: {args.verify_bundle}")
        try:
            original_argv = sys.argv
            with tempfile.TemporaryDirectory(prefix="sh4-manifest-replay-") as temp_dir:
                replay_output = Path(temp_dir) / "replayed-bundle.json"
                sys.argv = [original_argv[0], "--plan", str(args.plan), "--output", str(replay_output)]
                with contextlib.redirect_stdout(io.StringIO()):
                    main()
                replayed = replay_output.read_bytes()
            stored = args.verify_bundle.read_bytes()
            if stored != replayed:
                parser.error("bundle replay differs from saved bytes; an input, trust policy, or bundle artifact changed")
            print(json.dumps({"verified": True, "bundle": str(args.verify_bundle),
                              "sha256": hashlib.sha256(stored).hexdigest()}, separators=(",", ":")))
            return 0
        except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            parser.error(str(exc))
        finally:
            sys.argv = original_argv
    if args.output.exists():
        parser.error(f"output already exists: {args.output}")
    try:
        plan = read_json(args.plan)
        validate_plan_shape(plan)
        root = args.plan.parent
        if not plan.get("manifest_id"):
            raise ValueError("plan requires manifest_id")
        source, group = plan.get("resource_source"), plan.get("resource_group")
        if source not in ("cgroup_groups", "process_groups", "pss_groups") or not group:
            raise ValueError("plan requires valid resource_source and resource_group")
        admission, corpus = plan.get("admission_receipt_id"), plan.get("corpus_snapshot_id")
        if not admission or not corpus:
            raise ValueError("plan requires manager admission and fixed corpus snapshot IDs")
        minimum_repeats = plan.get("minimum_repeats", 3)
        if type(minimum_repeats) is not int or minimum_repeats < 3:
            raise ValueError("minimum_repeats must be at least three")
        phase_files = plan.get("phase_receipts")
        if not isinstance(phase_files, dict) or not set(REQUIRED_PHASES).issubset(phase_files):
            raise ValueError(f"plan must include phase_receipts for: {', '.join(REQUIRED_PHASES)}")
        concurrency_plan = plan.get("concurrency_by_phase")
        if not isinstance(concurrency_plan, dict) or any(phase not in concurrency_plan for phase in phase_files):
            raise ValueError("plan requires explicit concurrency_by_phase entries for every listed phase")
        if any(type(concurrency_plan[phase]) is not int or concurrency_plan[phase] < 1 for phase in phase_files):
            raise ValueError("phase concurrency values must be positive integers")
        if type(concurrency_plan.get("concurrent")) is not int or concurrency_plan["concurrent"] < 2:
            raise ValueError("concurrent phase requires explicit concurrency of at least two")
        allowed = set(REQUIRED_PHASES) | {"model-free"}
        if set(phase_files) - allowed:
            raise ValueError("unknown phase key in phase_receipts")
        phases: dict[str, list[dict[str, Any]]] = {}
        shared = None
        seen_receipts: set[str] = set()
        all_windows: list[tuple[int, int, str]] = []
        referenced_evidence_ids = {admission, corpus}
        telemetry_evidence_hashes: dict[str, str] = {}
        stage_evidence_hashes: dict[str, str] = {}
        stage_key_admission_checks: list[dict[str, Any]] = []
        for phase, files in phase_files.items():
            if not isinstance(files, list) or len(files) < minimum_repeats:
                raise ValueError(f"phase {phase!r} requires at least {minimum_repeats} comparison receipts")
            rows = []
            phase_fault_schedule = None
            phase_run_ids: set[str] = set()
            for item in files:
                if not isinstance(item, dict) or not isinstance(item.get("inputs"), dict):
                    raise ValueError(f"phase {phase!r} entries require receipt and inputs paths")
                path = resolve(root, item.get("receipt", ""))
                receipt = read_json(path)
                comparison = receipt.get("comparison") or {}
                grades = receipt.get("independent_grades") or {}
                resource = receipt.get("resource_attribution") or {}
                provider_attribution = receipt.get("provider_model_attribution") or {}
                identity = receipt.get("receipt") or {}
                if identity.get("type") != "search_hub_performance_pair" or not identity.get("id") or identity["id"] in seen_receipts:
                    raise ValueError(f"{path}: invalid or duplicate pair receipt")
                seen_receipts.add(identity["id"])
                receipt_inputs = identity.get("inputs") or {}
                required_inputs = ("baseline_report", "candidate_report", "baseline_grade_receipt", "candidate_grade_receipt")
                input_names = set(item["inputs"])
                telemetry_names = {"baseline_telemetry", "candidate_telemetry"}
                stage_names = {"baseline_stage_trace", "candidate_stage_trace"}
                if input_names not in (set(required_inputs), set(required_inputs) | telemetry_names,
                                       set(required_inputs) | stage_names, set(required_inputs) | telemetry_names | stage_names):
                    raise ValueError(f"{path}: inputs must name report/grade artifacts and optional paired telemetry/stage receipts")
                verified_inputs: dict[str, dict[str, str]] = {}
                for input_name in sorted(input_names):
                    artifact = resolve(root, item["inputs"][input_name])
                    metadata = receipt_inputs.get(input_name) or {}
                    artifact_hash = digest(artifact) if artifact.is_file() else None
                    if not artifact.is_file() or artifact.name != metadata.get("name") or artifact_hash != metadata.get("sha256"):
                        raise ValueError(f"{path}: source artifact hash/name mismatch for {input_name}")
                    verified_inputs[input_name] = {"name": artifact.name, "sha256": artifact_hash}
                if comparison.get("phase") != phase:
                    raise ValueError(f"{path}: receipt phase does not match plan key {phase!r}")
                telemetry_verified: dict[str, dict[str, Any]] = {}
                comparison_telemetry = comparison.get("correlated_telemetry")
                if telemetry_names.issubset(input_names):
                    if not isinstance(comparison_telemetry, dict) or set(comparison_telemetry) != {"baseline", "candidate"}:
                        raise ValueError(f"{path}: telemetry artifacts are present but paired receipt linkage is missing")
                    telemetry_policy = read_json(Path(__file__).with_name("run-telemetry-trust-v1.json"))
                    for arm_name, receipt_arm in (("baseline", comparison.get("baseline_arm")), ("candidate", comparison.get("candidate_arm"))):
                        telemetry_path = resolve(root, item["inputs"][arm_name + "_telemetry"])
                        telemetry_receipt = read_scope_receipt(telemetry_path)
                        verified = verify_run_telemetry_receipt(telemetry_receipt, telemetry_policy,
                            run_id=comparison.get("run_id"), arm=receipt_arm, phase=phase,
                            case_query_set_sha256=comparison.get("case_query_set_sha256"), case_count=comparison.get("case_count"))
                        linked = comparison_telemetry[arm_name]
                        if linked.get("receipt_id") != verified["receipt_id"] or linked.get("sha256") != digest(telemetry_path):
                            raise ValueError(f"{path}: correlated telemetry receipt identity/hash mismatch for {arm_name}")
                        for evidence in verified["source_evidence"]:
                            referenced_evidence_ids.add(evidence["receipt_id"])
                            prior = telemetry_evidence_hashes.setdefault(evidence["receipt_id"], evidence["evidence_sha256"])
                            if prior != evidence["evidence_sha256"]:
                                raise ValueError("telemetry source receipt ID maps to multiple digests")
                        admission_id = verified["key_admission_receipt_id"]
                        referenced_evidence_ids.add(admission_id)
                        stage_key_admission_checks.append(verified)
                        pinned_hash = linked.get("key_admission_receipt_sha256")
                        if linked.get("key_admission_receipt_id") != admission_id or pinned_hash != verified["key_admission_receipt_sha256"] or linked.get("source_evidence") != verified["source_evidence"]:
                            raise ValueError(f"{path}: key-admission/source evidence linkage mismatch for {arm_name}")
                        prior = telemetry_evidence_hashes.setdefault(admission_id, pinned_hash)
                        if prior != pinned_hash:
                            raise ValueError("telemetry key-admission ID maps to multiple digests")
                        verified["source_receipt_verified"] = True
                        telemetry_verified[arm_name] = verified
                elif comparison_telemetry is not None:
                    raise ValueError(f"{path}: pair receipt links correlated telemetry without source artifacts")
                stage_verified: dict[str, dict[str, Any]] = {}
                comparison_stage = comparison.get("correlated_stage_telemetry")
                if stage_names.issubset(input_names):
                    if not isinstance(comparison_stage, dict) or set(comparison_stage) != {"baseline", "candidate"}:
                        raise ValueError(f"{path}: stage artifacts are present but paired receipt linkage is missing")
                    stage_policy = read_json(Path(__file__).with_name("stage-telemetry-trust-v1.json"))
                    reports_for_stages = {}
                    for arm_name, report_role, receipt_arm in (("baseline", "baseline_report", comparison.get("baseline_arm")),
                                                               ("candidate", "candidate_report", comparison.get("candidate_arm"))):
                        report_path = resolve(root, item["inputs"][report_role])
                        report_data = read_json(report_path)
                        case_pairs = [{"case_id": str(row["case_id"]), "query_sha256": row["query_sha256"]}
                                      for row in report_data.get("cases", [])]
                        stage_path = resolve(root, item["inputs"][arm_name + "_stage_trace"])
                        stage_receipt = read_scope_receipt(stage_path)
                        verified = stage_telemetry.verify_receipt(stage_receipt, stage_policy,
                            run_id=comparison.get("run_id"), arm=receipt_arm, phase=phase, query_pairs=case_pairs)
                        linked = comparison_stage[arm_name]
                        if linked.get("receipt_id") != verified["receipt_id"] or linked.get("sha256") != digest(stage_path):
                            raise ValueError(f"{path}: stage receipt identity/hash mismatch for {arm_name}")
                        scope_links = {stage_id: {"receipt_id": data["resource_scope_receipt_id"],
                            "uri": data["resource_scope_uri"], "sha256": data["resource_scope_sha256"],
                            "method_id": data["resource_method_id"]}
                            for stage_id, data in verified["stages"].items()
                            if data.get("resource_status") == "owner_attributed_diagnostic"}
                        if linked.get("resource_scopes", {}) != scope_links:
                            raise ValueError(f"{path}: stage resource-scope provenance linkage mismatch for {arm_name}")
                        for evidence in [verified["source_evidence"]]:
                            evidence_id = evidence["receipt_id"]
                            referenced_evidence_ids.add(evidence_id)
                            prior = stage_evidence_hashes.setdefault(evidence_id, evidence["evidence_sha256"])
                            if prior != evidence["evidence_sha256"]:
                                raise ValueError("stage source receipt ID maps to multiple digests")
                        for scope in scope_links.values():
                            referenced_evidence_ids.add(scope["receipt_id"])
                            prior = stage_evidence_hashes.setdefault(scope["receipt_id"], scope["sha256"])
                            if prior != scope["sha256"]:
                                raise ValueError("stage resource-scope receipt ID maps to multiple digests")
                        admission_id = verified["key_admission_receipt_id"]
                        referenced_evidence_ids.add(admission_id)
                        if (linked.get("key_admission_receipt_id") != admission_id or
                                linked.get("key_admission_receipt_sha256") != verified["key_admission_receipt_sha256"] or
                                linked.get("source_evidence") != verified["source_evidence"]):
                            raise ValueError(f"{path}: stage key-admission/source evidence linkage mismatch for {arm_name}")
                        prior = stage_evidence_hashes.setdefault(admission_id, verified["key_admission_receipt_sha256"])
                        if prior != verified["key_admission_receipt_sha256"]:
                            raise ValueError("stage key-admission ID maps to multiple digests")
                        stage_verified[arm_name] = verified
                        reports_for_stages[arm_name] = report_data
                    computed_stage_pair = stage_telemetry.compare_verified_receipts(stage_verified["baseline"],
                        stage_verified["candidate"], quality_gate=grades.get("acceptance_nonregression_gate") is True,
                        baseline_accepted_parts=(grades.get("baseline") or {}).get("accepted_parts"),
                        candidate_accepted_parts=(grades.get("candidate") or {}).get("accepted_parts"),
                        baseline_accepted_tasks=(grades.get("baseline") or {}).get("accepted_tasks"),
                        candidate_accepted_tasks=(grades.get("candidate") or {}).get("accepted_tasks"))
                    if receipt.get("stage_latency") != computed_stage_pair:
                        raise ValueError(f"{path}: stage latency pair summary does not match verified stage receipts")
                elif comparison_stage is not None:
                    raise ValueError(f"{path}: pair receipt links stage traces without source artifacts")
                else:
                    supplied_stage = receipt.get("stage_latency") or {}
                    if any(((supplied_stage.get("stages") or {}).get(stage_id, {}).get("candidate_to_baseline_total_duration_ratio") is not None)
                           for stage_id in stage_telemetry.STAGE_IDS):
                        raise ValueError(f"{path}: stage latency values cannot be accepted without signed stage receipt artifacts")
                if provider_attribution.get("scope_receipt_id") != comparison.get("serving_scope_receipt_id"):
                    raise ValueError(f"{path}: provider/model scope receipt linkage mismatch")
                if comparison.get("concurrency") != concurrency_plan[phase]:
                    raise ValueError(f"{path}: concurrency differs from plan for {phase!r}")
                windows = comparison.get("measurement_windows") or {}
                bw, cw = windows.get("baseline") or {}, windows.get("candidate") or {}
                bstart, bend = bw.get("started_at_unix_ms"), bw.get("ended_at_unix_ms")
                cstart, cend = cw.get("started_at_unix_ms"), cw.get("ended_at_unix_ms")
                if not all(isinstance(v, (int, float)) for v in (bstart, bend, cstart, cend)) or bend <= bstart or cend <= cstart:
                    raise ValueError(f"{path}: pair receipt lacks valid arm time windows")
                if not (bend <= cstart or cend <= bstart):
                    raise ValueError(f"{path}: paired arms must run sequentially, not concurrently")
                all_windows.extend(((int(bstart), int(bend), f"{path.name}:baseline"), (int(cstart), int(cend), f"{path.name}:candidate")))
                run_id = comparison.get("run_id")
                if not run_id or run_id in phase_run_ids:
                    raise ValueError(f"{path}: phase receipts require unique run_id values")
                phase_run_ids.add(run_id)
                if comparison.get("admission_receipt_id") != admission or comparison.get("corpus_snapshot_id") != corpus:
                    raise ValueError(f"{path}: admission/corpus identity differs from plan")
                fault_id = comparison.get("fault_schedule_id")
                if not fault_id or (phase == "degraded" and fault_id == "none"):
                    raise ValueError(f"{path}: degraded phase requires a non-none fault schedule receipt")
                if phase_fault_schedule is None:
                    phase_fault_schedule = fault_id
                elif phase_fault_schedule != fault_id:
                    raise ValueError(f"{path}: fault schedule differs between repeats of {phase!r}")
                if fault_id != "none":
                    referenced_evidence_ids.add(fault_id)
                if resource.get("source") != source or resource.get("group") != group:
                    raise ValueError(f"{path}: selected resource attribution differs from plan")
                bg, cg = grades.get("baseline") or {}, grades.get("candidate") or {}
                grade_identity = (bg.get("cohort_id"), bg.get("task_set_sha256"), bg.get("total_parts"), bg.get("total_tasks"), bg.get("tier"), bg.get("case_query_set_sha256"), bg.get("acceptance_policy_id"))
                candidate_grade_identity = (cg.get("cohort_id"), cg.get("task_set_sha256"), cg.get("total_parts"), cg.get("total_tasks"), cg.get("tier"), cg.get("case_query_set_sha256"), cg.get("acceptance_policy_id"))
                if not all(grade_identity) or grade_identity != candidate_grade_identity:
                    raise ValueError(f"{path}: independent grade identity/denominator mismatch")
                expected_components = plan.get("component_groups", {})
                recorded_components = resource.get("component_groups") or {}
                if not isinstance(recorded_components, dict) or set(recorded_components) != set(expected_components):
                    raise ValueError(f"{path}: component-group roles differ from the reviewed manifest plan")
                component_budget_metrics: dict[str, Any] = {}
                if expected_components:
                    base_report = read_json(resolve(root, item["inputs"]["baseline_report"]))
                    candidate_report = read_json(resolve(root, item["inputs"]["candidate_report"]))
                    pattern_key = {"cgroup_groups": "cgroup_group_patterns", "process_groups": "process_group_patterns",
                                   "pss_groups": "pss_group_patterns"}[source]
                    for role, component_name in sorted(expected_components.items()):
                        component = recorded_components[role]
                        required_component_fields = {"group", "group_definition", "cpu_ms", "cpu_ms_per_accepted_part",
                            "cpu_ms_per_accepted_task", "peak_memory_bytes", "cpu_per_accepted_part_ratio",
                            "cpu_per_accepted_task_ratio", "peak_memory_ratio", "quality_gate", "interpretation"}
                        if not isinstance(component, dict) or set(component) != required_component_fields or component.get("group") != component_name:
                            raise ValueError(f"{path}: component attribution contract mismatch for role {role!r}")
                        if component_name == group:
                            raise ValueError(f"{path}: diagnostic component group {role!r} duplicates primary total group")
                        baseline_component_cpu, baseline_component_memory, _ = metric(base_report, source, component_name)
                        candidate_component_cpu, candidate_component_memory, _ = metric(candidate_report, source, component_name)
                        patterns = []
                        for report_data in (base_report, candidate_report):
                            matches = [item for item in report_data.get(pattern_key, [])
                                       if isinstance(item, dict) and item.get("name") == component_name]
                            if len(matches) != 1:
                                raise ValueError(f"{path}: component group {component_name!r} is not uniquely declared")
                            patterns.append(matches[0])
                        if patterns[0] != patterns[1] or patterns[0] != component["group_definition"]:
                            raise ValueError(f"{path}: component group definition differs for role {role!r}")
                        memory_scale = 1024 if source in ("process_groups", "pss_groups") else 1
                        baseline_component_memory *= memory_scale
                        candidate_component_memory *= memory_scale
                        expected_values = {
                            "cpu_ms": {"baseline": baseline_component_cpu, "candidate": candidate_component_cpu},
                            "cpu_ms_per_accepted_part": {"baseline": baseline_component_cpu / bg["accepted_parts"], "candidate": candidate_component_cpu / cg["accepted_parts"]},
                            "cpu_ms_per_accepted_task": {"baseline": baseline_component_cpu / bg["accepted_tasks"], "candidate": candidate_component_cpu / cg["accepted_tasks"]},
                            "peak_memory_bytes": {"baseline": baseline_component_memory, "candidate": candidate_component_memory},
                            "cpu_per_accepted_part_ratio": (candidate_component_cpu / cg["accepted_parts"]) / (baseline_component_cpu / bg["accepted_parts"]),
                            "cpu_per_accepted_task_ratio": (candidate_component_cpu / cg["accepted_tasks"]) / (baseline_component_cpu / bg["accepted_tasks"]),
                            "peak_memory_ratio": candidate_component_memory / baseline_component_memory,
                            "quality_gate": grades.get("acceptance_nonregression_gate"),
                        }
                        for field, expected_value in expected_values.items():
                            actual = component.get(field)
                            if isinstance(expected_value, dict):
                                if actual != expected_value:
                                    raise ValueError(f"{path}: component {field} does not match source reports/grades for {role!r}")
                            elif type(expected_value) in (int, float):
                                if type(actual) not in (int, float) or not math.isclose(actual, expected_value, rel_tol=1e-12, abs_tol=1e-12):
                                    raise ValueError(f"{path}: component {field} does not match source reports/grades for {role!r}")
                            elif actual != expected_value:
                                raise ValueError(f"{path}: component {field} does not match source reports/grades for {role!r}")
                        component_budget_metrics[role] = {
                            "cpu_part": (expected_values["cpu_ms_per_accepted_part"]["baseline"], expected_values["cpu_ms_per_accepted_part"]["candidate"]),
                            "cpu_task": (expected_values["cpu_ms_per_accepted_task"]["baseline"], expected_values["cpu_ms_per_accepted_task"]["candidate"]),
                            "peak_memory": (baseline_component_memory, candidate_component_memory),
                            "group": component_name, "group_definition": patterns[0]}
                state_ids = comparison.get("state_receipt_ids") or {}
                if not state_ids.get("baseline") or not state_ids.get("candidate"):
                    raise ValueError(f"{path}: state receipt IDs missing from phase pair")
                referenced_evidence_ids.update((state_ids["baseline"], state_ids["candidate"]))
                build_ids = comparison.get("build_ids") or {}
                component_signature = tuple((role, recorded_components[role]["group"],
                    json.dumps(recorded_components[role]["group_definition"], sort_keys=True, separators=(",", ":")))
                    for role in sorted(recorded_components))
                common = (comparison.get("baseline_arm"), comparison.get("candidate_arm"), comparison.get("workload_sha256"), comparison.get("case_query_set_sha256"), comparison.get("transport"), comparison.get("sample_interval_ms"), comparison.get("corpus_snapshot_id"), grade_identity, build_ids.get("baseline"), build_ids.get("candidate"), resource.get("group_definition"), bg.get("run_tag"), cg.get("run_tag"), bg.get("evaluation_run_ids"), cg.get("evaluation_run_ids"), bg.get("evidence_uri"), bg.get("evidence_sha256"), cg.get("evidence_uri"), cg.get("evidence_sha256"), comparison.get("serving_scope_receipt_id"), component_signature)
                if shared is None:
                    shared = common
                elif common != shared:
                    raise ValueError(f"{path}: phase receipt differs in shared cohort/arm/measurement contract")
                latency = comparison.get("latency_ms") or {}
                api_latency = comparison.get("api_response_latency_ms") or {}
                def arm_ratio(pair: Any) -> tuple[Any, Any] | None:
                    if not isinstance(pair, dict):
                        return None
                    return pair.get("baseline"), pair.get("candidate")
                def p95_ratio(pair: Any) -> tuple[Any, Any] | None:
                    if not isinstance(pair, dict):
                        return None
                    baseline, candidate = pair.get("baseline"), pair.get("candidate")
                    if not isinstance(baseline, dict) or not isinstance(candidate, dict):
                        return None
                    return baseline.get("p95"), candidate.get("p95")
                separate = receipt.get("separate_evidence") or {}
                if telemetry_verified:
                    correlated = {}
                    for arm_name, verified in telemetry_verified.items():
                        correlated[arm_name + "_tokens"] = {"available": True, "attribution_status": "run_correlated",
                            "signature_verified": True, "run_id": verified["run_id"], "case_query_set_sha256": verified["case_query_set_sha256"],
                            "input_tokens": verified["gateway"]["input_tokens"], "output_tokens": verified["gateway"]["output_tokens"],
                            "source_receipt_id": verified["receipt_id"]}
                        correlated[arm_name + "_cache"] = {"available": True, "attribution_status": "run_correlated",
                            "signature_verified": True, "run_id": verified["run_id"], "case_query_set_sha256": verified["case_query_set_sha256"],
                            "resolver_cache_hit_delta": verified["cache"]["hits"], "resolver_cache_miss_delta": verified["cache"]["misses"],
                            "source_receipt_id": verified["receipt_id"]}
                    separate = {"ambient": separate.get("ambient", separate), "run_attributed": correlated}
                receipt["separate_evidence"] = separate
                rows.append({"path": path.name, "sha256": digest(path), "receipt_id": identity["id"],
                    "source_artifacts": {"pair_receipt": {"name": path.name, "sha256": digest(path)}, "inputs": verified_inputs},
                    "phase": phase, "run_id": comparison.get("run_id"), "concurrency": comparison.get("concurrency"), "fault_schedule_id": comparison.get("fault_schedule_id"), "measurement_windows": windows, "grade_receipt_ids": {"baseline": bg.get("receipt_id"), "candidate": cg.get("receipt_id")}, "grade_run_tags": {"baseline": bg.get("run_tag"), "candidate": cg.get("run_tag")}, "grade_evidence": {"baseline_uri": bg.get("evidence_uri"), "baseline_sha256": bg.get("evidence_sha256"), "candidate_uri": cg.get("evidence_uri"), "candidate_sha256": cg.get("evidence_sha256")}, "grade_evaluation_run_ids": {"baseline": bg.get("evaluation_run_ids"), "candidate": cg.get("evaluation_run_ids")}, "grade_tier": bg.get("tier"), "grade_case_query_set_sha256": bg.get("case_query_set_sha256"), "accepted_parts": {"baseline": bg.get("accepted_parts"), "candidate": cg.get("accepted_parts")}, "state_receipt_ids": comparison.get("state_receipt_ids"), "baseline_acceptance_nonregression": grades.get("acceptance_nonregression_gate"), "point_floor_status": resource.get("floor_status"), "provider_model_scope_receipt_id": comparison.get("serving_scope_receipt_id"), "provider_model_attribution": receipt.get("provider_model_attribution"), "telemetry_scope": receipt.get("telemetry_scope"), "budget_metrics": {"cpu_part": arm_ratio(resource.get("cpu_ms_per_accepted_part")), "cpu_task": arm_ratio(resource.get("cpu_ms_per_accepted_task")), "peak_memory": arm_ratio(resource.get("peak_memory_bytes")), "client_p95": p95_ratio(latency), "api_p95": p95_ratio(api_latency)}, "stage_latency": receipt.get("stage_latency"), "component_budget_metrics": component_budget_metrics, "separate_evidence": receipt.get("separate_evidence")})
                rows[-1]["correlated_telemetry"] = comparison.get("correlated_telemetry")
            phases[phase] = rows
        if shared and shared[19]:
            referenced_evidence_ids.add(shared[19])
        idle = validate_idle(plan, root, source, group, admission, corpus)
        for rows in (idle["baseline_reports"], idle["candidate_reports"]):
            for row in rows:
                referenced_evidence_ids.update((row["state_receipt_id"], row["fault_schedule_id"]))
                if row["fault_schedule_id"] == "none":
                    referenced_evidence_ids.discard("none")
        if any(row["build_id"] != shared[8] for row in idle["baseline_reports"]) or any(row["build_id"] != shared[9] for row in idle["candidate_reports"]):
            raise ValueError("idle and loaded samples must use the same per-arm build IDs")
        if any(row["serving_scope_receipt_id"] != shared[19] for row in idle["baseline_reports"] + idle["candidate_reports"]):
            raise ValueError("idle and loaded reports must use the same provider/model serving-scope receipt")
        if any(row["group_definition"] != shared[10] for row in idle["baseline_reports"] + idle["candidate_reports"]):
            raise ValueError("idle and loaded reports must use the same selected resource-group definition")
        for arm in ("baseline_reports", "candidate_reports"):
            all_windows.extend((int(row["started_at_unix_ms"]), int(row["ended_at_unix_ms"]), f"idle:{arm}:{row['path']}") for row in idle[arm])
        all_windows.sort()
        for previous, current in zip(all_windows, all_windows[1:]):
            if current[0] < previous[1]:
                raise ValueError(f"measurement windows overlap: {previous[2]} and {current[2]}")
        authority = plan.get("evidence_receipts")
        if not isinstance(authority, list):
            raise ValueError("plan requires evidence_receipts with path/hash references for admission, corpus, fault, and prepared-state receipts")
        authority_index: dict[str, dict[str, Any]] = {}
        authority_paths: dict[str, Path] = {}
        for item in authority:
            if not isinstance(item, dict) or not item.get("receipt_id") or item["receipt_id"] in authority_index:
                raise ValueError("evidence_receipts entries require unique receipt_id values")
            evidence_path = resolve(root, item.get("path", ""))
            if not evidence_path.is_file() or digest(evidence_path) != item.get("sha256"):
                raise ValueError(f"authority receipt missing or hash mismatch: {item.get('receipt_id')}")
            authority_index[item["receipt_id"]] = {"path": evidence_path.name, "sha256": item["sha256"]}
            authority_paths[item["receipt_id"]] = evidence_path
        for telemetry_id, telemetry_hash in telemetry_evidence_hashes.items():
            if telemetry_id in authority_index and authority_index[telemetry_id]["sha256"] != telemetry_hash:
                raise ValueError(f"telemetry authority receipt hash differs from signed run receipt: {telemetry_id}")
        for evidence_id, evidence_hash in stage_evidence_hashes.items():
            if evidence_id not in authority_index or authority_index[evidence_id]["sha256"] != evidence_hash:
                raise ValueError(f"stage authority receipt missing or hash differs from signed stage receipt: {evidence_id}")
        for stage_check in stage_key_admission_checks:
            admission_id = stage_check["key_admission_receipt_id"]
            stage_check["key_admission_verification"] = verify_key_admission_artifact(
                read_scope_receipt(authority_paths[admission_id]), authority_index[admission_id]["sha256"], stage_check)
        provider_scope_hash_present = bool(shared[19] and shared[19] in authority_index)
        provider_scope_verification = None
        if provider_scope_hash_present:
            scope_receipt = read_scope_receipt(authority_paths[shared[19]])
            trust_policy = read_json(Path(__file__).with_name("provider-model-scope-trust-v1.json"))
            provider_scope_verification = verify_scope_receipt(scope_receipt, trust_policy,
                resource_source=source, resource_group=group, group_definition=shared[10])
            referenced_evidence_ids.add(provider_scope_verification["key_admission_receipt_id"])
        missing_authority = referenced_evidence_ids - authority_index.keys()
        if missing_authority:
            raise ValueError(f"authority receipt files missing for {len(missing_authority)} referenced IDs")
        if provider_scope_hash_present:
            admission_id = provider_scope_verification["key_admission_receipt_id"]
            admission_receipt = read_scope_receipt(authority_paths[admission_id])
            provider_scope_verification["key_admission_verification"] = verify_key_admission_artifact(
                admission_receipt, authority_index[admission_id]["sha256"], provider_scope_verification)
        provider_model_attribution = {"scope_receipt_id": shared[19], "receipt_artifact_hash_present": provider_scope_hash_present,
            "verification": provider_scope_verification,
            "status": "signed receipt verified against pinned issuer key and measured group; the owner's coverage assertion still requires reviewer evidence audit" if provider_scope_hash_present else "missing; provider/model serving attribution is incomplete"}
        budget_rollup = phase_budget_rollup(phases, provider_scope_hash_present)
        acceptance_matrix = showcase_gate_matrix(budget_rollup, provider_model_attribution, idle)
        result = {
            "schema_version": 1,
            "provenance": {"plan": {"name": args.plan.name, "sha256": digest(args.plan)},
                           "source_path_policy": "artifact basenames and SHA-256 only; full paths remain in the local reviewed plan"},
            "manifest": {"id": plan["manifest_id"], "admission_receipt_id": admission, "corpus_snapshot_id": corpus, "resource_source": source, "resource_group": group, "minimum_repeats": minimum_repeats, "concurrency_by_phase": concurrency_plan, "common_contract": {"baseline_arm": shared[0], "candidate_arm": shared[1], "workload_sha256": shared[2], "case_query_set_sha256": shared[3], "transport": shared[4], "sample_interval_ms": shared[5], "cohort_id": shared[7][0], "task_set_sha256": shared[7][1], "total_parts": shared[7][2], "total_tasks": shared[7][3], "grade_tier": shared[7][4], "grade_case_query_set_sha256": shared[7][5], "acceptance_policy_id": shared[7][6], "baseline_build_id": shared[8], "candidate_build_id": shared[9], "resource_group_definition": shared[10], "baseline_grade_run_tag": shared[11], "candidate_grade_run_tag": shared[12], "baseline_evaluation_run_ids": shared[13], "candidate_evaluation_run_ids": shared[14], "baseline_grade_evidence": {"uri": shared[15], "sha256": shared[16]}, "candidate_grade_evidence": {"uri": shared[17], "sha256": shared[18]}, "serving_scope_receipt_id": shared[19], "component_groups": [{"role": role, "group": group_name, "group_definition": json.loads(definition_json)} for role, group_name, definition_json in shared[20]]}},
            "phase_receipts": phases,
            "provider_model_attribution": provider_model_attribution,
            "phase_budget_rollup": budget_rollup,
            "showcase_gate_matrix": acceptance_matrix,
            "evidence_receipts": authority_index,
            "idle_check": idle,
            "acceptance": "auditable_evidence_bundle_only; performance showcase and all speed/footprint/quality/floor claims remain unverified until reviewed with approved measurements",
        }
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, indent=2) + "\n")
        print(json.dumps({"output": str(args.output), "phase_counts": {key: len(value) for key, value in phases.items()}, "idle_cpu_within_baseline_uncertainty": idle["idle_cpu_within_baseline_uncertainty"], "idle_memory_within_baseline_uncertainty": idle["idle_memory_within_baseline_uncertainty"]}, separators=(",", ":")))
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    sys.exit(main())
