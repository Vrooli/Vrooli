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
import sys
from pathlib import Path
from typing import Any


def read_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text())
    if not isinstance(value, dict):
        raise ValueError(f"{path}: expected a JSON object")
    return value


def file_sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def resource_pattern(report: dict[str, Any], source: str, name: str) -> dict[str, Any]:
    pattern_key = {"cgroup_groups": "cgroup_group_patterns", "process_groups": "process_group_patterns", "pss_groups": "pss_group_patterns"}[source]
    patterns = report.get(pattern_key)
    if not isinstance(patterns, list):
        raise ValueError(f"report lacks {pattern_key}")
    selected = [item for item in patterns if isinstance(item, dict) and item.get("name") == name]
    if len(selected) != 1:
        raise ValueError(f"report must declare exactly one {pattern_key} entry for {name!r}")
    return selected[0]


def grade_counts(path: Path, case_ids: set[str]) -> dict[str, Any]:
    """Read SH1's sanitized receipt: {receipt_id, cases:{id:{parts:{accepted,total},tasks:{accepted,total}}}}."""
    receipt = read_json(path)
    cases = receipt.get("cases")
    if (receipt.get("issuer") != "SH1" or not receipt.get("receipt_id") or
            not receipt.get("cohort_id") or not receipt.get("task_set_sha256") or not isinstance(cases, dict)):
        raise ValueError("grade receipt requires issuer=SH1, receipt_id, cohort_id, task_set_sha256, and cases object")
    missing = case_ids - cases.keys()
    extra = cases.keys() - case_ids
    if missing or extra:
        raise ValueError(f"grade case mismatch: missing={len(missing)} extra={len(extra)}")
    totals = {"accepted_parts": 0, "total_parts": 0, "accepted_tasks": 0, "total_tasks": 0}
    for case_id in case_ids:
        case = cases[case_id]
        if not isinstance(case, dict):
            raise ValueError(f"grade for {case_id!r} must be an object")
        for dimension in ("parts", "tasks"):
            counts = case.get(dimension)
            if not isinstance(counts, dict):
                raise ValueError(f"grade for {case_id!r} requires {dimension} counts")
            accepted, total = counts.get("accepted"), counts.get("total")
            if not isinstance(accepted, int) or not isinstance(total, int) or total < 0 or accepted < 0 or accepted > total:
                raise ValueError(f"invalid {dimension} counts for {case_id!r}")
            totals[f"accepted_{dimension}"] += accepted
            totals[f"total_{dimension}"] += total
    if not totals["total_parts"] or not totals["total_tasks"]:
        raise ValueError("grade receipt must include at least one graded part and task")
    return {"receipt_id": receipt["receipt_id"], "cohort_id": receipt["cohort_id"], "task_set_sha256": receipt["task_set_sha256"], **totals}


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


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", required=True, type=Path)
    parser.add_argument("--candidate", required=True, type=Path)
    parser.add_argument("--baseline-grades", required=True, type=Path)
    parser.add_argument("--candidate-grades", required=True, type=Path)
    parser.add_argument("--resource-source", required=True, choices=("cgroup_groups", "process_groups", "pss_groups"))
    parser.add_argument("--resource-group", required=True, help="one named group present in both reports")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    if args.output.exists():
        parser.error(f"output already exists: {args.output}")
    try:
        base, cand = read_json(args.baseline), read_json(args.candidate)
        if base.get("run_id") != cand.get("run_id"):
            raise ValueError("paired arms must use the same run_id")
        if base.get("arm") == cand.get("arm"):
            raise ValueError("baseline and candidate arm labels must differ")
        for field in ("workload", "phase", "concurrency", "sample_interval_ms", "transport"):
            if base.get(field) != cand.get(field):
                raise ValueError(f"paired arms differ in {field}")
        base_context = base.get("execution_context")
        cand_context = cand.get("execution_context")
        required_context = ("build_id", "corpus_snapshot_id", "fault_schedule_id", "state_receipt_id")
        if not isinstance(base_context, dict) or not isinstance(cand_context, dict) or any(not base_context.get(k) or not cand_context.get(k) for k in required_context):
            raise ValueError("paired reports require verified build, corpus snapshot, fault schedule, and state receipt IDs")
        for field in ("corpus_snapshot_id", "fault_schedule_id"):
            if base_context[field] != cand_context[field]:
                raise ValueError(f"paired arms differ in execution_context.{field}")
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
        bg = grade_counts(args.baseline_grades, b_rows.keys())
        cg = grade_counts(args.candidate_grades, b_rows.keys())
        if (bg["cohort_id"], bg["task_set_sha256"]) != (cg["cohort_id"], cg["task_set_sha256"]):
            raise ValueError("independent grade receipts are from different frozen cohorts")
        if (bg["total_parts"], bg["total_tasks"]) != (cg["total_parts"], cg["total_tasks"]):
            raise ValueError("independent grade denominators differ between arms")
        cpu_b, mem_b = chosen_group(base, args.resource_source, args.resource_group)
        cpu_c, mem_c = chosen_group(cand, args.resource_source, args.resource_group)
        pattern_b = resource_pattern(base, args.resource_source, args.resource_group)
        pattern_c = resource_pattern(cand, args.resource_source, args.resource_group)
        if pattern_b != pattern_c:
            raise ValueError("selected resource group definition differs between paired reports")
        for grade in (bg, cg):
            if grade["accepted_parts"] == 0 or grade["accepted_tasks"] == 0:
                raise ValueError("cannot compute cost per accepted result when a denominator is zero")
        b_part_rate = bg["accepted_parts"] / bg["total_parts"]
        c_part_rate = cg["accepted_parts"] / cg["total_parts"]
        b_task_rate = bg["accepted_tasks"] / bg["total_tasks"]
        c_task_rate = cg["accepted_tasks"] / cg["total_tasks"]
        # A performance gain is ineligible when either independently graded acceptance rate regresses.
        quality_gate = c_part_rate >= b_part_rate and c_task_rate >= b_task_rate
        b_mem_bytes = mem_b * (1024 if args.resource_source in ("process_groups", "pss_groups") else 1)
        c_mem_bytes = mem_c * (1024 if args.resource_source in ("process_groups", "pss_groups") else 1)
        result = {
            "schema_version": 1,
            "receipt": {
                "type": "search_hub_performance_pair",
                "id": "sh4-pair-" + hashlib.sha256(json.dumps({"base": file_sha256(args.baseline), "candidate": file_sha256(args.candidate), "base_grades": file_sha256(args.baseline_grades), "candidate_grades": file_sha256(args.candidate_grades), "resource_source": args.resource_source, "resource_group": args.resource_group}, sort_keys=True).encode()).hexdigest()[:24],
                "inputs": {
                    "baseline_report": {"name": args.baseline.name, "sha256": file_sha256(args.baseline)},
                    "candidate_report": {"name": args.candidate.name, "sha256": file_sha256(args.candidate)},
                    "baseline_grade_receipt": {"name": args.baseline_grades.name, "sha256": file_sha256(args.baseline_grades)},
                    "candidate_grade_receipt": {"name": args.candidate_grades.name, "sha256": file_sha256(args.candidate_grades)},
                },
            },
            "comparison": {"run_id": base.get("run_id"), "baseline_arm": base.get("arm"), "candidate_arm": cand.get("arm"), "phase": base["phase"], "concurrency": base["concurrency"], "transport": base["transport"], "sample_interval_ms": base["sample_interval_ms"], "workload_sha256": base["workload"]["sha256"], "case_count": len(b_rows), "case_query_set_sha256": hashlib.sha256(json.dumps(query_pairs, sort_keys=True, separators=(",", ":")).encode()).hexdigest(), "corpus_snapshot_id": base_context["corpus_snapshot_id"], "fault_schedule_id": base_context["fault_schedule_id"], "state_receipt_ids": {"baseline": base_context["state_receipt_id"], "candidate": cand_context["state_receipt_id"]}, "build_ids": {"baseline": base_context["build_id"], "candidate": cand_context["build_id"]}},
            "independent_grades": {"baseline": bg, "candidate": cg, "acceptance_nonregression_gate": quality_gate,
                "part_acceptance_rate": {"baseline": b_part_rate, "candidate": c_part_rate},
                "task_acceptance_rate": {"baseline": b_task_rate, "candidate": c_task_rate}},
            "resource_attribution": {"source": args.resource_source, "group": args.resource_group, "group_definition": pattern_b,
                "cpu_ms": {"baseline": cpu_b, "candidate": cpu_c},
                "cpu_ms_per_accepted_part": {"baseline": cpu_b / bg["accepted_parts"], "candidate": cpu_c / cg["accepted_parts"]},
                "cpu_ms_per_accepted_task": {"baseline": cpu_b / bg["accepted_tasks"], "candidate": cpu_c / cg["accepted_tasks"]},
                "peak_memory_bytes": {"baseline": b_mem_bytes, "candidate": c_mem_bytes},
                "peak_memory_change_fraction": (c_mem_bytes / b_mem_bytes - 1) if b_mem_bytes else None,
                "cpu_per_accepted_part_change_fraction": (cpu_c / cg["accepted_parts"]) / (cpu_b / bg["accepted_parts"]) - 1,
                "cpu_per_accepted_task_change_fraction": (cpu_c / cg["accepted_tasks"]) / (cpu_b / bg["accepted_tasks"]) - 1,
                "cpu_floor_point_estimate_pass": (cpu_c / cg["accepted_parts"]) <= 0.8 * (cpu_b / bg["accepted_parts"]),
                "memory_floor_point_estimate_pass": b_mem_bytes > 0 and c_mem_bytes <= 0.85 * b_mem_bytes,
                "floor_status": "ineligible_quality_regression" if not quality_gate else (
                    "point_estimates_meet_floors" if (cpu_c / cg["accepted_parts"] <= 0.8 * (cpu_b / bg["accepted_parts"]) and b_mem_bytes > 0 and c_mem_bytes <= 0.85 * b_mem_bytes) else "point_estimate_floor_shortfall"
                )},
            "separate_evidence": {"baseline_cache": base.get("cache_tokens_and_grades", {}).get("search_hub_cache"), "candidate_cache": cand.get("cache_tokens_and_grades", {}).get("search_hub_cache"), "baseline_tokens": base.get("cache_tokens_and_grades", {}).get("ai_gateway_tokens"), "candidate_tokens": cand.get("cache_tokens_and_grades", {}).get("ai_gateway_tokens")},
            "limitations": ["Aggregate batch CPU and peak memory are divided by independent accepted-part/task counts; they are not per-request stage allocation.", "Cache and token evidence is retained separately and can be contaminated by concurrent traffic.", "A passing non-regression gate does not by itself satisfy SH4's repeated-run uncertainty, idle, provider/model attribution, or outage requirements."],
        }
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, indent=2) + "\n")
        print(json.dumps({"output": str(args.output), "quality_gate": quality_gate, "cpu_part_change": result["resource_attribution"]["cpu_per_accepted_part_change_fraction"], "cpu_task_change": result["resource_attribution"]["cpu_per_accepted_task_change_fraction"], "memory_change": result["resource_attribution"]["peak_memory_change_fraction"]}, separators=(",", ":")))
    except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        parser.error(str(exc))
    return 0


if __name__ == "__main__":
    sys.exit(main())
