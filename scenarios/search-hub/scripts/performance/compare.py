#!/usr/bin/env python3
"""Compare paired Search Hub measurement reports against an independent grade receipt.

The grade receipt is produced by SH1 and contains only case IDs and accepted
part/task counts. Resource groups are selected explicitly so inclusive cgroup
totals are not accidentally added to process/PSS estimates.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path
from typing import Any

from run_telemetry import verify_receipt as verify_run_telemetry_receipt
import stage_telemetry


def read_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text())
    if not isinstance(value, dict):
        raise ValueError(f"{path}: expected a JSON object")
    return value


def file_sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def is_sha256(value: Any) -> bool:
    return isinstance(value, str) and len(value) == 64 and all(char in "0123456789abcdef" for char in value)


def resource_pattern(report: dict[str, Any], source: str, name: str) -> dict[str, Any]:
    pattern_key = {"cgroup_groups": "cgroup_group_patterns", "process_groups": "process_group_patterns", "pss_groups": "pss_group_patterns"}[source]
    patterns = report.get(pattern_key)
    if not isinstance(patterns, list):
        raise ValueError(f"report lacks {pattern_key}")
    selected = [item for item in patterns if isinstance(item, dict) and item.get("name") == name]
    if len(selected) != 1:
        raise ValueError(f"report must declare exactly one {pattern_key} entry for {name!r}")
    return selected[0]


def grade_counts(path: Path, case_hashes: dict[str, str], expected_build: str, expected_tier: str) -> dict[str, Any]:
    """Validate sanitized SH1 per-case grades against this exact build and query set."""
    receipt = read_json(path)
    cases = receipt.get("cases")
    required_fields = {"schema_version", "issuer", "receipt_id", "evidence_uri", "evidence_sha256", "run_tag", "evaluation_run_ids", "tier", "build_id", "acceptance_policy_id", "cohort_id", "task_set_sha256", "cases"}
    if set(receipt) != required_fields:
        raise ValueError("grade receipt does not match the SH1 grade-export-v1 field allowlist")
    if type(receipt.get("schema_version")) is not int or receipt.get("schema_version") != 1:
        raise ValueError("unsupported SH1 grade export schema version")
    identity_fields = ("receipt_id", "cohort_id", "run_tag", "tier", "build_id", "acceptance_policy_id", "evidence_uri")
    if (receipt.get("issuer") != "SH1" or any(not isinstance(receipt.get(key), str) or not receipt[key] for key in identity_fields) or
            not isinstance(receipt.get("evaluation_run_ids"), list) or not receipt["evaluation_run_ids"] or
            any(not isinstance(value, str) or not value for value in receipt["evaluation_run_ids"]) or
            len(set(receipt["evaluation_run_ids"])) != len(receipt["evaluation_run_ids"]) or
            not is_sha256(receipt.get("evidence_sha256")) or not is_sha256(receipt.get("task_set_sha256")) or
            not isinstance(cases, dict)):
        raise ValueError("grade receipt requires issuer, receipt/cohort/task-set IDs, acceptance policy, run_tag, tier, build_id and cases")
    if receipt["build_id"] != expected_build:
        raise ValueError("independent grade build_id does not match the measured arm build")
    if receipt["tier"] != expected_tier:
        raise ValueError(f"independent grade tier must be {expected_tier!r}")
    missing = case_hashes.keys() - cases.keys()
    extra = cases.keys() - case_hashes.keys()
    if missing or extra:
        raise ValueError(f"grade case mismatch: missing={len(missing)} extra={len(extra)}")
    totals = {"accepted_parts": 0, "total_parts": 0, "accepted_tasks": 0, "total_tasks": 0}
    case_verdicts: dict[str, dict[str, list[bool]]] = {}
    for case_id, query_hash in case_hashes.items():
        case = cases[case_id]
        if not isinstance(case, dict) or set(case) != {"query_sha256", "parts", "tasks"}:
            raise ValueError(f"grade for {case_id!r} must match the redacted case-grade field allowlist")
        if not is_sha256(case.get("query_sha256")) or case.get("query_sha256") != query_hash:
            raise ValueError(f"independent grade query hash differs for case {case_id!r}")
        case_verdicts[case_id] = {}
        for dimension in ("parts", "tasks"):
            counts = case.get(dimension)
            allowed_counts = {"accepted", "total", "verdicts"}
            if not isinstance(counts, dict) or set(counts) != allowed_counts:
                raise ValueError(f"grade for {case_id!r} requires {dimension} counts")
            accepted, total = counts.get("accepted"), counts.get("total")
            if type(accepted) is not int or type(total) is not int or total < 0 or accepted < 0 or accepted > total:
                raise ValueError(f"invalid {dimension} counts for {case_id!r}")
            verdicts = counts.get("verdicts")
            if not isinstance(verdicts, list) or len(verdicts) != total or any(not isinstance(value, bool) for value in verdicts) or sum(verdicts) != accepted:
                raise ValueError(f"{dimension} verdicts must be per-item booleans matching accepted/total for {case_id!r}")
            if dimension == "tasks" and (total != 1 or len(verdicts) != 1):
                raise ValueError(f"task grade for {case_id!r} must contain exactly one complete-task verdict")
            case_verdicts[case_id][dimension] = verdicts
            totals[f"accepted_{dimension}"] += accepted
            totals[f"total_{dimension}"] += total
    if not totals["total_parts"] or not totals["total_tasks"]:
        raise ValueError("grade receipt must include at least one graded part and task")
    query_pairs = [{"case_id": case_id, "query_sha256": case_hashes[case_id]} for case_id in sorted(case_hashes)]
    return {"receipt_id": receipt["receipt_id"], "run_tag": receipt["run_tag"], "tier": receipt["tier"],
            "build_id": receipt["build_id"], "cohort_id": receipt["cohort_id"],
            "task_set_sha256": receipt["task_set_sha256"], "acceptance_policy_id": receipt["acceptance_policy_id"],
            "evidence_uri": receipt["evidence_uri"], "evidence_sha256": receipt["evidence_sha256"],
            "evaluation_run_ids": receipt["evaluation_run_ids"],
            "case_query_set_sha256": hashlib.sha256(json.dumps(query_pairs, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
            **totals, "_case_verdicts": case_verdicts}


def chosen_group(report: dict[str, Any], source: str, name: str) -> tuple[float, float]:
    groups = report.get(source)
    if not isinstance(groups, dict) or name not in groups:
        raise ValueError(f"report lacks {source}.{name}")
    group = groups[name]
    if not isinstance(group, dict) or group.get("available") is False:
        raise ValueError(f"{source}.{name} is unavailable")
    cpu = group.get("cpu_ms")
    memory_key = {"cgroup_groups": "peak_memory_current_bytes_sum", "process_groups": "peak_rss_kib_sum", "pss_groups": "peak_pss_kib_sum"}[source]
    memory = group.get(memory_key)
    if not isinstance(cpu, (int, float)) or not isinstance(memory, (int, float)) or cpu <= 0 or memory <= 0:
        raise ValueError(f"{source}.{name} lacks positive CPU or peak memory attribution")
    return float(cpu), float(memory)


def parse_component_groups(raw: list[str], primary_group: str) -> dict[str, str]:
    """Parse diagnostic ROLE=GROUP entries; outputs remain separate, never summed."""
    result: dict[str, str] = {}
    for item in raw:
        role, separator, group = item.partition("=")
        role, group = role.strip(), group.strip()
        if not separator or not group or not re.fullmatch(r"[a-z][a-z0-9_.-]{0,63}", role):
            raise ValueError(f"invalid --component-group {item!r}; expected lowercase role=GROUP")
        if role in result:
            raise ValueError(f"duplicate component role {role!r}")
        if group == primary_group:
            raise ValueError("component group must differ from the primary serving-total group")
        result[role] = group
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", required=True, type=Path)
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--baseline-grades", required=True, type=Path)
    parser.add_argument("--candidate-grades", required=True, type=Path)
    parser.add_argument("--baseline-telemetry", type=Path, help="owner-signed per-run token/cache receipt for the baseline arm")
    parser.add_argument("--candidate-telemetry", type=Path, help="owner-signed per-run token/cache receipt for the candidate arm")
    parser.add_argument("--baseline-stage-trace", type=Path, help="owner-signed per-case stage-latency receipt for the baseline arm")
    parser.add_argument("--candidate-stage-trace", type=Path, help="owner-signed per-case stage-latency receipt for the candidate arm")
    parser.add_argument("--resource-source", required=True, choices=("cgroup_groups", "process_groups", "pss_groups"))
    parser.add_argument("--resource-group", required=True, help="one named group present in both reports")
    parser.add_argument("--component-group", action="append", default=[], metavar="ROLE=GROUP",
                        help="diagnostic process/cgroup/PSS group; repeated groups are reported separately, never summed")
    parser.add_argument("--grade-tier", choices=("federated", "provider_direct"), default="federated", help="SH1 independent grading tier; serving Query normally uses federated")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    if args.output.exists():
        parser.error(f"output already exists: {args.output}")
    try:
        base, cand = read_json(args.baseline), read_json(args.candidate)
        component_groups = parse_component_groups(args.component_group, args.resource_group)
        if base.get("run_id") != cand.get("run_id"):
            raise ValueError("paired arms must use the same run_id")
        if base.get("arm") == cand.get("arm"):
            raise ValueError("baseline and candidate arm labels must differ")
        for field in ("workload", "phase", "concurrency", "sample_interval_ms", "transport"):
            if base.get(field) != cand.get(field):
                raise ValueError(f"paired arms differ in {field}")
        base_context = base.get("execution_context")
        cand_context = cand.get("execution_context")
        required_context = ("build_id", "corpus_snapshot_id", "fault_schedule_id", "state_receipt_id", "admission_receipt_id")
        if not isinstance(base_context, dict) or not isinstance(cand_context, dict) or any(not base_context.get(k) or not cand_context.get(k) for k in required_context):
            raise ValueError("paired reports require verified build, corpus snapshot, fault schedule, and state receipt IDs")
        for field in ("corpus_snapshot_id", "fault_schedule_id", "admission_receipt_id"):
            if base_context[field] != cand_context[field]:
                raise ValueError(f"paired arms differ in execution_context.{field}")
        if base_context.get("serving_scope_receipt_id") != cand_context.get("serving_scope_receipt_id"):
            raise ValueError("paired arms differ in execution_context.serving_scope_receipt_id")
        if base.get("telemetry_scope") != cand.get("telemetry_scope"):
            raise ValueError("paired reports differ in telemetry scope definitions")
        if base.get("workload", {}).get("cases", 0) < 1:
            raise ValueError("comparison requires a non-empty workload")
        b_rows = {str(row["case_id"]): row for row in base.get("cases", [])}
        c_rows = {str(row["case_id"]): row for row in cand.get("cases", [])}
        if not b_rows or b_rows.keys() != c_rows.keys():
            raise ValueError("paired arm case IDs differ or are missing")
        if len(b_rows) != base["workload"]["cases"] or len(c_rows) != cand["workload"]["cases"]:
            raise ValueError("report row count does not match workload case count")
        query_pairs = []
        for case_id in sorted(b_rows):
            if b_rows[case_id].get("query_sha256") != c_rows[case_id].get("query_sha256"):
                raise ValueError(f"query hash differs for case {case_id!r}")
            if not b_rows[case_id].get("query_sha256"):
                raise ValueError(f"missing query hash for case {case_id!r}")
            query_pairs.append({"case_id": case_id, "query_sha256": b_rows[case_id]["query_sha256"]})
        query_hashes = {case_id: row["query_sha256"] for case_id, row in b_rows.items()}
        bg = grade_counts(args.baseline_grades, query_hashes, base_context["build_id"], args.grade_tier)
        cg = grade_counts(args.candidate_grades, query_hashes, cand_context["build_id"], args.grade_tier)
        if (bg["cohort_id"], bg["task_set_sha256"], bg["tier"], bg["acceptance_policy_id"], bg["case_query_set_sha256"]) != (cg["cohort_id"], cg["task_set_sha256"], cg["tier"], cg["acceptance_policy_id"], cg["case_query_set_sha256"]):
            raise ValueError("independent grade receipts are from different frozen cohorts")
        for case_id in query_hashes:
            if any(len(bg["_case_verdicts"][case_id][dimension]) != len(cg["_case_verdicts"][case_id][dimension]) for dimension in ("parts", "tasks")):
                raise ValueError(f"independent grade denominator changed for case {case_id!r}")
        if bg["run_tag"] == cg["run_tag"]:
            raise ValueError("baseline and candidate grade receipts must reference distinct SH1 run tags")
        if (bg["total_parts"], bg["total_tasks"]) != (cg["total_parts"], cg["total_tasks"]):
            raise ValueError("independent grade denominators differ between arms")
        telemetry = None
        if bool(args.baseline_telemetry) != bool(args.candidate_telemetry):
            raise ValueError("baseline and candidate per-run telemetry receipts must be supplied together")
        if args.baseline_telemetry:
            trust_policy = read_json(Path(__file__).with_name("run-telemetry-trust-v1.json"))
            case_set_hash = hashlib.sha256(json.dumps(query_pairs, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
            telemetry = {}
            for arm_name, arm_path, expected_arm in (("baseline", args.baseline_telemetry, base.get("arm")),
                                                      ("candidate", args.candidate_telemetry, cand.get("arm"))):
                source_receipt = read_json(arm_path)
                verified = verify_run_telemetry_receipt(source_receipt, trust_policy, run_id=base.get("run_id"),
                    arm=expected_arm, phase=base.get("phase"), case_query_set_sha256=case_set_hash, case_count=len(b_rows))
                telemetry[arm_name] = {"path": arm_path, "sha256": file_sha256(arm_path), "receipt": source_receipt, "verified": verified}
        stage_traces = None
        if bool(args.baseline_stage_trace) != bool(args.candidate_stage_trace):
            raise ValueError("baseline and candidate stage trace receipts must be supplied together")
        if args.baseline_stage_trace:
            stage_policy = read_json(Path(__file__).with_name("stage-telemetry-trust-v1.json"))
            stage_traces = {}
            for arm_name, arm_path, expected_arm in (("baseline", args.baseline_stage_trace, base.get("arm")),
                                                      ("candidate", args.candidate_stage_trace, cand.get("arm"))):
                source_receipt = read_json(arm_path)
                verified = stage_telemetry.verify_receipt(source_receipt, stage_policy, run_id=base.get("run_id"),
                    arm=expected_arm, phase=base.get("phase"), query_pairs=query_pairs)
                stage_traces[arm_name] = {"path": arm_path, "sha256": file_sha256(arm_path),
                                          "receipt": source_receipt, "verified": verified}
        cpu_b, mem_b = chosen_group(base, args.resource_source, args.resource_group)
        cpu_c, mem_c = chosen_group(cand, args.resource_source, args.resource_group)
        pattern_b = resource_pattern(base, args.resource_source, args.resource_group)
        pattern_c = resource_pattern(cand, args.resource_source, args.resource_group)
        if pattern_b != pattern_c:
            raise ValueError("selected resource group definition differs between paired reports")
        for grade in (bg, cg):
            if grade["accepted_parts"] == 0 or grade["accepted_tasks"] == 0:
                raise ValueError("cannot compute cost per accepted result when a denominator is zero")
        component_attribution = {}
        for role, component_name in sorted(component_groups.items()):
            component_b_cpu, component_b_mem = chosen_group(base, args.resource_source, component_name)
            component_c_cpu, component_c_mem = chosen_group(cand, args.resource_source, component_name)
            component_pattern_b = resource_pattern(base, args.resource_source, component_name)
            component_pattern_c = resource_pattern(cand, args.resource_source, component_name)
            if component_pattern_b != component_pattern_c:
                raise ValueError(f"component group definition differs for role {role!r}")
            scale = 1024 if args.resource_source in ("process_groups", "pss_groups") else 1
            component_b_mem_bytes, component_c_mem_bytes = component_b_mem * scale, component_c_mem * scale
            component_attribution[role] = {
                "group": component_name, "group_definition": component_pattern_b,
                "cpu_ms": {"baseline": component_b_cpu, "candidate": component_c_cpu},
                "cpu_ms_per_accepted_part": {"baseline": component_b_cpu / bg["accepted_parts"], "candidate": component_c_cpu / cg["accepted_parts"]},
                "cpu_ms_per_accepted_task": {"baseline": component_b_cpu / bg["accepted_tasks"], "candidate": component_c_cpu / cg["accepted_tasks"]},
                "peak_memory_bytes": {"baseline": component_b_mem_bytes, "candidate": component_c_mem_bytes},
                "cpu_per_accepted_part_ratio": (component_c_cpu / cg["accepted_parts"]) / (component_b_cpu / bg["accepted_parts"]),
                "cpu_per_accepted_task_ratio": (component_c_cpu / cg["accepted_tasks"]) / (component_b_cpu / bg["accepted_tasks"]),
                "peak_memory_ratio": component_c_mem_bytes / component_b_mem_bytes,
                "interpretation": "diagnostic component group; never summed with other groups; possible overlap is unknown",
            }
        b_part_rate = bg["accepted_parts"] / bg["total_parts"]
        c_part_rate = cg["accepted_parts"] / cg["total_parts"]
        b_task_rate = bg["accepted_tasks"] / bg["total_tasks"]
        c_task_rate = cg["accepted_tasks"] / cg["total_tasks"]
        part_regressions = sum(
            baseline_ok and not candidate_ok
            for case_id, baseline_case in bg["_case_verdicts"].items()
            for baseline_ok, candidate_ok in zip(baseline_case["parts"], cg["_case_verdicts"][case_id]["parts"])
        )
        task_regressions = sum(
            baseline_ok and not candidate_ok
            for case_id, baseline_case in bg["_case_verdicts"].items()
            for baseline_ok, candidate_ok in zip(baseline_case["tasks"], cg["_case_verdicts"][case_id]["tasks"])
        )
        part_nonregression = part_regressions == 0
        task_nonregression = task_regressions == 0
        # Aggregate rates can hide per-part loss, so gate every frozen part and task independently.
        quality_gate = part_nonregression and task_nonregression and c_part_rate >= b_part_rate and c_task_rate >= b_task_rate
        for component in component_attribution.values():
            component["quality_gate"] = quality_gate
        stage_pair = (stage_telemetry.compare_verified_receipts(stage_traces["baseline"]["verified"],
            stage_traces["candidate"]["verified"], quality_gate=quality_gate,
            baseline_accepted_parts=bg["accepted_parts"], candidate_accepted_parts=cg["accepted_parts"],
            baseline_accepted_tasks=bg["accepted_tasks"], candidate_accepted_tasks=cg["accepted_tasks"])
            if stage_traces else None)
        if stage_pair is None:
            stage_pair = {"status": "unavailable_stage_receipts_not_supplied", "quality_gate": quality_gate,
                          "measurement_dimension": "latency_only",
                          "resource_attribution_status": "unavailable_stage_cpu_memory_not_measured",
                          "stages": {stage_id: {"status": "unavailable_stage_receipts_not_supplied",
                                                 "candidate_to_baseline_total_duration_ratio": None}
                                     for stage_id in stage_telemetry.STAGE_IDS}}
        b_mem_bytes = mem_b * (1024 if args.resource_source in ("process_groups", "pss_groups") else 1)
        c_mem_bytes = mem_c * (1024 if args.resource_source in ("process_groups", "pss_groups") else 1)
        result = {
            "schema_version": 1,
            "receipt": {
                "type": "search_hub_performance_pair",
                "id": "sh4-pair-" + hashlib.sha256(json.dumps({"base": file_sha256(args.baseline), "candidate": file_sha256(args.candidate), "base_grades": file_sha256(args.baseline_grades), "candidate_grades": file_sha256(args.candidate_grades), "baseline_telemetry": telemetry["baseline"]["sha256"] if telemetry else None, "candidate_telemetry": telemetry["candidate"]["sha256"] if telemetry else None, "baseline_stage_trace": stage_traces["baseline"]["sha256"] if stage_traces else None, "candidate_stage_trace": stage_traces["candidate"]["sha256"] if stage_traces else None, "resource_source": args.resource_source, "resource_group": args.resource_group, "component_groups": component_groups}, sort_keys=True).encode()).hexdigest()[:24],
                "inputs": {
                    "baseline_report": {"name": args.baseline.name, "sha256": file_sha256(args.baseline)},
                    "candidate_report": {"name": args.candidate.name, "sha256": file_sha256(args.candidate)},
                    "baseline_grade_receipt": {"name": args.baseline_grades.name, "sha256": file_sha256(args.baseline_grades)},
                    "candidate_grade_receipt": {"name": args.candidate_grades.name, "sha256": file_sha256(args.candidate_grades)},
                    **({"baseline_telemetry": {"name": telemetry["baseline"]["path"].name, "sha256": telemetry["baseline"]["sha256"]}, "candidate_telemetry": {"name": telemetry["candidate"]["path"].name, "sha256": telemetry["candidate"]["sha256"]}} if telemetry else {}),
                    **({"baseline_stage_trace": {"name": stage_traces["baseline"]["path"].name, "sha256": stage_traces["baseline"]["sha256"]}, "candidate_stage_trace": {"name": stage_traces["candidate"]["path"].name, "sha256": stage_traces["candidate"]["sha256"]}} if stage_traces else {}),
                },
            },
            "comparison": {"run_id": base.get("run_id"), "baseline_arm": base.get("arm"), "candidate_arm": cand.get("arm"), "phase": base["phase"], "concurrency": base["concurrency"], "transport": base["transport"], "sample_interval_ms": base["sample_interval_ms"], "workload_sha256": base["workload"]["sha256"], "case_count": len(b_rows), "case_query_set_sha256": hashlib.sha256(json.dumps(query_pairs, sort_keys=True, separators=(",", ":")).encode()).hexdigest(), "correlated_telemetry": {arm: {"receipt_id": data["verified"]["receipt_id"], "sha256": data["sha256"], "key_admission_receipt_id": data["verified"]["key_admission_receipt_id"], "key_admission_receipt_sha256": data["verified"]["key_admission_receipt_sha256"], "key_fingerprint_sha256": data["verified"]["key_fingerprint_sha256"], "reviewer_ids": data["verified"]["reviewer_ids"], "source_evidence": data["verified"]["source_evidence"]} for arm, data in telemetry.items()} if telemetry else None, "correlated_stage_telemetry": {arm: {"receipt_id": data["verified"]["receipt_id"], "sha256": data["sha256"], "key_admission_receipt_id": data["verified"]["key_admission_receipt_id"], "key_admission_receipt_sha256": data["verified"]["key_admission_receipt_sha256"], "key_fingerprint_sha256": data["verified"]["key_fingerprint_sha256"], "reviewer_ids": data["verified"]["reviewer_ids"], "source_evidence": data["verified"]["source_evidence"], "resource_scopes": {stage_id: {"receipt_id": item["resource_scope_receipt_id"], "uri": item["resource_scope_uri"], "sha256": item["resource_scope_sha256"], "method_id": item["resource_method_id"]} for stage_id, item in data["verified"]["stages"].items() if item.get("resource_status") == "owner_attributed_diagnostic"}} for arm, data in stage_traces.items()} if stage_traces else None, "corpus_snapshot_id": base_context["corpus_snapshot_id"], "fault_schedule_id": base_context["fault_schedule_id"], "admission_receipt_id": base_context["admission_receipt_id"], "serving_scope_receipt_id": base_context.get("serving_scope_receipt_id"), "measurement_windows": {"baseline": {"started_at_unix_ms": base.get("run_started_at_unix_ms"), "ended_at_unix_ms": base.get("run_ended_at_unix_ms")}, "candidate": {"started_at_unix_ms": cand.get("run_started_at_unix_ms"), "ended_at_unix_ms": cand.get("run_ended_at_unix_ms")}}, "latency_ms": {"baseline": base.get("latency_ms"), "candidate": cand.get("latency_ms")}, "api_response_latency_ms": {"baseline": base.get("api_response_latency_ms"), "candidate": cand.get("api_response_latency_ms")}, "state_receipt_ids": {"baseline": base_context["state_receipt_id"], "candidate": cand_context["state_receipt_id"]}, "build_ids": {"baseline": base_context["build_id"], "candidate": cand_context["build_id"]}},
            "stage_latency": stage_pair,
            "independent_grades": {"baseline": {k: v for k, v in bg.items() if k != "_case_verdicts"}, "candidate": {k: v for k, v in cg.items() if k != "_case_verdicts"}, "acceptance_nonregression_gate": quality_gate, "per_part_nonregression": part_nonregression, "regressed_part_count": part_regressions, "per_task_nonregression": task_nonregression, "regressed_task_count": task_regressions,
                "part_acceptance_rate": {"baseline": b_part_rate, "candidate": c_part_rate},
                "task_acceptance_rate": {"baseline": b_task_rate, "candidate": c_task_rate}},
            "resource_attribution": {"source": args.resource_source, "group": args.resource_group, "group_definition": pattern_b,
                "cpu_ms": {"baseline": cpu_b, "candidate": cpu_c},
                "cpu_ms_per_accepted_part": {"baseline": cpu_b / bg["accepted_parts"], "candidate": cpu_c / cg["accepted_parts"]},
                "cpu_ms_per_accepted_task": {"baseline": cpu_b / bg["accepted_tasks"], "candidate": cpu_c / cg["accepted_tasks"]},
                "peak_memory_bytes": {"baseline": b_mem_bytes, "candidate": c_mem_bytes},
                "component_groups": component_attribution,
                "component_group_policy": "separate diagnostic groups; do not sum because process/cgroup overlap is unknown; selected primary group alone is total-cost floor input",
                "peak_memory_change_fraction": (c_mem_bytes / b_mem_bytes - 1) if b_mem_bytes else None,
                "cpu_per_accepted_part_change_fraction": (cpu_c / cg["accepted_parts"]) / (cpu_b / bg["accepted_parts"]) - 1,
                "cpu_per_accepted_task_change_fraction": (cpu_c / cg["accepted_tasks"]) / (cpu_b / bg["accepted_tasks"]) - 1,
                "cpu_floor_point_estimate_pass": (cpu_c / cg["accepted_parts"]) <= 0.8 * (cpu_b / bg["accepted_parts"]),
                "cpu_task_floor_point_estimate_pass": (cpu_c / cg["accepted_tasks"]) <= 0.8 * (cpu_b / bg["accepted_tasks"]),
                "memory_floor_point_estimate_pass": b_mem_bytes > 0 and c_mem_bytes <= 0.85 * b_mem_bytes,
                "selected_group_only": True,
                "floor_status": "ineligible_quality_regression" if not quality_gate else (
                    "ineligible_missing_provider_model_scope_receipt" if not base_context.get("serving_scope_receipt_id") else (
                        "selected_group_point_estimates_only_pending_scope_review" if (cpu_c / cg["accepted_parts"] <= 0.8 * (cpu_b / bg["accepted_parts"]) and cpu_c / cg["accepted_tasks"] <= 0.8 * (cpu_b / bg["accepted_tasks"]) and b_mem_bytes > 0 and c_mem_bytes <= 0.85 * b_mem_bytes) else "selected_group_point_estimate_floor_shortfall"
                    )
                )},
            "separate_evidence": {"ambient": {"baseline_cache": base.get("cache_tokens_and_grades", {}).get("search_hub_cache"), "candidate_cache": cand.get("cache_tokens_and_grades", {}).get("search_hub_cache"), "baseline_tokens": base.get("cache_tokens_and_grades", {}).get("ai_gateway_tokens"), "candidate_tokens": cand.get("cache_tokens_and_grades", {}).get("ai_gateway_tokens")}, "run_attributed": ({arm + "_tokens": {"available": True, "attribution_status": "run_correlated", "run_id": data["verified"]["run_id"], "case_query_set_sha256": data["verified"]["case_query_set_sha256"], "input_tokens": data["verified"]["gateway"]["input_tokens"], "output_tokens": data["verified"]["gateway"]["output_tokens"], "source_receipt_id": data["verified"]["receipt_id"]} for arm, data in telemetry.items()} | {arm + "_cache": {"available": True, "attribution_status": "run_correlated", "run_id": data["verified"]["run_id"], "case_query_set_sha256": data["verified"]["case_query_set_sha256"], "resolver_cache_hit_delta": data["verified"]["cache"]["hits"], "resolver_cache_miss_delta": data["verified"]["cache"]["misses"], "source_receipt_id": data["verified"]["receipt_id"]} for arm, data in telemetry.items()} if telemetry else None)},
            "provider_model_attribution": {"scope_receipt_id": base_context.get("serving_scope_receipt_id"), "status": "owner_receipt_reference_only; group inclusion and owner coverage unverified" if base_context.get("serving_scope_receipt_id") else "missing_owner_scope_receipt", "selected_resource_group": {"source": args.resource_source, "name": args.resource_group, "definition": pattern_b}},
            "telemetry_scope": {"baseline": base.get("telemetry_scope"), "candidate": cand.get("telemetry_scope"), "tokens_comparable": False, "cache_comparable": False, "reason": "current token events are scenario-window scoped and cache metrics are rolling-window aggregates; neither is correlated to this run or accepted cases"},
            "limitations": ["Aggregate batch CPU and peak memory are divided by independent accepted-part/task counts; they are not per-request stage allocation.", "Cache and token evidence is retained separately and can be contaminated by concurrent traffic.", "A passing non-regression gate does not by itself satisfy SH4's repeated-run uncertainty, idle, provider/model attribution, or outage requirements."],
        }
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, indent=2) + "\n")
        print(json.dumps({"output": str(args.output), "quality_gate": quality_gate, "cpu_part_change": result["resource_attribution"]["cpu_per_accepted_part_change_fraction"], "cpu_task_change": result["resource_attribution"]["cpu_per_accepted_task_change_fraction"], "cpu_part_floor_point_pass": result["resource_attribution"]["cpu_floor_point_estimate_pass"], "cpu_task_floor_point_pass": result["resource_attribution"]["cpu_task_floor_point_estimate_pass"], "memory_change": result["resource_attribution"]["peak_memory_change_fraction"]}, separators=(",", ":")))
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    sys.exit(main())
