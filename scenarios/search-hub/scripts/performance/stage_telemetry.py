#!/usr/bin/env python3
"""Reduce and verify complete run/case-correlated Search Hub stage spans.

SH4 never queries Insights for stage data and never reads owner private keys.
The stage owner exports sanitized per-case telemetry, this adapter validates
and reduces it to an unsigned signing payload, and a separately admitted owner
signs the payload. Until a purpose-limited trust key is reviewed and pinned,
verification fails closed.
"""

from __future__ import annotations

import base64
import hashlib
from typing import Any

from scope_receipts import canonical_bytes, rsa_pkcs1_sha256_verify, validate_trust_policy


STAGE_PURPOSE = "search_hub_stage_telemetry"
STAGE_IDS = ("classifier", "resolver", "fanout", "rerank", "provider")
EXPORT_FIELDS = {"schema_version", "run_id", "arm", "phase", "complete", "truncated", "case_coverage"}
CASE_FIELDS = {"case_id", "query_sha256", "stages"}
STAGE_FIELDS = {"executed", "duration_ms"}
STAGE_RESOURCE_FIELDS = {"cpu_ms", "peak_memory_bytes", "resource_method_id", "resource_scope_receipt_id",
                        "resource_scope_uri", "resource_scope_sha256"}
SOURCE_FIELDS = {"receipt_id", "evidence_uri", "evidence_sha256"}
RECEIPT_FIELDS = {"schema_version", "receipt_id", "issuer", "key_id", "signature_algorithm", "run_id", "arm",
                  "phase", "case_query_set_sha256", "case_count", "stages", "case_coverage", "source_evidence",
                  "signature_base64"}


def is_sha256(value: Any) -> bool:
    return isinstance(value, str) and len(value) == 64 and all(ch in "0123456789abcdef" for ch in value)


def query_set_digest(pairs: list[dict[str, str]]) -> str:
    normalized = sorted(pairs, key=lambda row: row["case_id"])
    return hashlib.sha256(canonical_bytes(normalized)).hexdigest()


def produce_unsigned_receipt(*, receipt_id: str, issuer: str, key_id: str, run_id: str, arm: str, phase: str,
                             query_pairs: list[dict[str, str]], export: dict[str, Any],
                             source_evidence: dict[str, Any]) -> dict[str, Any]:
    """Validate exact case/stage coverage and return the owner-signing payload."""
    for label, value in (("receipt_id", receipt_id), ("issuer", issuer), ("key_id", key_id),
                         ("run_id", run_id), ("arm", arm), ("phase", phase)):
        if not isinstance(value, str) or not value:
            raise ValueError(f"{label} must be nonempty")
    if not query_pairs:
        raise ValueError("query_pairs must contain the frozen workload")
    expected: dict[str, str] = {}
    for row in query_pairs:
        if not isinstance(row, dict) or set(row) != {"case_id", "query_sha256"}:
            raise ValueError("query_pairs require only case_id and query_sha256")
        case_id, query_hash = row["case_id"], row["query_sha256"]
        if not isinstance(case_id, str) or not case_id or not is_sha256(query_hash) or case_id in expected:
            raise ValueError("query_pairs require unique IDs and lowercase SHA-256 query hashes")
        expected[case_id] = query_hash
    if not isinstance(export, dict) or set(export) != EXPORT_FIELDS:
        raise ValueError("stage export field allowlist mismatch")
    if type(export.get("schema_version")) is not int or export["schema_version"] != 1:
        raise ValueError("unsupported stage export version")
    if any(export.get(field) != value for field, value in (("run_id", run_id), ("arm", arm), ("phase", phase))):
        raise ValueError("stage export run/arm/phase differs from measured report")
    if export.get("complete") is not True or export.get("truncated") is not False:
        raise ValueError("stage export must explicitly be complete and not truncated")
    coverage = export.get("case_coverage")
    if not isinstance(coverage, list):
        raise ValueError("stage case_coverage must be a list")
    indexed: dict[str, dict[str, Any]] = {}
    for row in coverage:
        if not isinstance(row, dict) or set(row) != CASE_FIELDS:
            raise ValueError("stage case coverage field allowlist mismatch")
        case_id = row.get("case_id")
        if not isinstance(case_id, str) or not case_id or case_id in indexed or expected.get(case_id) != row.get("query_sha256"):
            raise ValueError("stage coverage is duplicate or outside the exact frozen case/query set")
        stages = row.get("stages")
        if not isinstance(stages, dict) or set(stages) != set(STAGE_IDS):
            raise ValueError("every case requires explicit coverage for every registered stage")
        normalized_stages: dict[str, dict[str, Any]] = {}
        for stage_id in STAGE_IDS:
            item = stages[stage_id]
            if not isinstance(item, dict) or set(item) not in (STAGE_FIELDS, STAGE_FIELDS | STAGE_RESOURCE_FIELDS) or type(item.get("executed")) is not bool:
                raise ValueError(f"stage {stage_id} coverage field allowlist mismatch")
            duration = item.get("duration_ms")
            if item["executed"]:
                if type(duration) is not int or duration <= 0:
                    raise ValueError(f"executed stage {stage_id} requires a positive integer duration")
            elif duration is not None:
                raise ValueError(f"non-executed stage {stage_id} must have null duration, never zero")
            normalized = {"executed": item["executed"], "duration_ms": duration}
            if set(item) == STAGE_FIELDS | STAGE_RESOURCE_FIELDS:
                cpu_ms, memory_bytes = item["cpu_ms"], item["peak_memory_bytes"]
                method_id = item["resource_method_id"]
                scope_id, scope_uri, scope_hash = item["resource_scope_receipt_id"], item["resource_scope_uri"], item["resource_scope_sha256"]
                if (not isinstance(method_id, str) or not method_id or not isinstance(scope_id, str) or not scope_id or
                        not isinstance(scope_uri, str) or not scope_uri or not is_sha256(scope_hash)):
                    raise ValueError(f"stage {stage_id} resource attribution requires method ID and hash-linked scope evidence")
                if item["executed"]:
                    if type(cpu_ms) is not int or cpu_ms < 0 or type(memory_bytes) is not int or memory_bytes <= 0:
                        raise ValueError(f"executed stage {stage_id} resource attribution requires nonnegative CPU and positive peak memory")
                elif any(value is not None for value in (cpu_ms, memory_bytes)):
                    raise ValueError(f"non-executed stage {stage_id} must have null resource values")
                normalized.update({"cpu_ms": cpu_ms, "peak_memory_bytes": memory_bytes,
                                   "resource_method_id": method_id, "resource_scope_receipt_id": scope_id,
                                   "resource_scope_uri": scope_uri, "resource_scope_sha256": scope_hash})
            normalized_stages[stage_id] = normalized
        indexed[case_id] = {"case_id": case_id, "query_sha256": row["query_sha256"], "stages": normalized_stages}
    if indexed.keys() != expected.keys():
        raise ValueError("stage coverage must exactly match every frozen case")
    if not isinstance(source_evidence, dict) or set(source_evidence) != SOURCE_FIELDS:
        raise ValueError("stage source evidence must contain receipt_id, evidence_uri and evidence_sha256")
    if any(not isinstance(source_evidence.get(key), str) or not source_evidence[key]
           for key in ("receipt_id", "evidence_uri")) or not is_sha256(source_evidence.get("evidence_sha256")):
        raise ValueError("stage source evidence is missing identity, URI or lowercase digest")
    stage_summary = {}
    for stage_id in STAGE_IDS:
        values = [indexed[case_id]["stages"][stage_id]["duration_ms"] for case_id in sorted(indexed)
                  if indexed[case_id]["stages"][stage_id]["executed"]]
        not_applicable = len(indexed) - len(values)
        summary = {"executed_case_count": len(values), "not_applicable_case_count": not_applicable,
                   "complete_case_count": len(indexed), "durations_ms": values}
        resource_rows = [indexed[case_id]["stages"][stage_id] for case_id in sorted(indexed)
                         if indexed[case_id]["stages"][stage_id]["executed"]]
        has_resource_fields = ["cpu_ms" in row for row in resource_rows]
        if any(has_resource_fields) and not all(has_resource_fields):
            raise ValueError(f"stage {stage_id} resource attribution is partially populated")
        if resource_rows and all(has_resource_fields):
            methods = {row["resource_method_id"] for row in resource_rows}
            scope_ids = {row["resource_scope_receipt_id"] for row in resource_rows}
            scope_uris = {row["resource_scope_uri"] for row in resource_rows}
            scopes = {row["resource_scope_sha256"] for row in resource_rows}
            if len(methods) != 1 or len(scope_ids) != 1 or len(scope_uris) != 1 or len(scopes) != 1:
                raise ValueError(f"stage {stage_id} resource method or scope changes within receipt")
            summary.update({"resource_status": "owner_attributed_diagnostic",
                "resource_method_id": next(iter(methods)), "resource_scope_receipt_id": next(iter(scope_ids)),
                "resource_scope_uri": next(iter(scope_uris)), "resource_scope_sha256": next(iter(scopes)),
                "cpu_ms_total": sum(row["cpu_ms"] for row in resource_rows),
                "peak_memory_bytes": max(row["peak_memory_bytes"] for row in resource_rows)})
        stage_summary[stage_id] = summary
    return {"schema_version": 1, "receipt_id": receipt_id, "issuer": issuer, "key_id": key_id,
            "signature_algorithm": "rsa-pkcs1v15-sha256", "run_id": run_id, "arm": arm, "phase": phase,
            "case_query_set_sha256": query_set_digest(query_pairs), "case_count": len(expected),
            "stages": stage_summary, "case_coverage": [indexed[key] for key in sorted(indexed)],
            "source_evidence": source_evidence}


def signing_bytes(payload: dict[str, Any]) -> bytes:
    if "signature_base64" in payload:
        raise ValueError("signing payload must omit signature_base64")
    return canonical_bytes(payload)


def verify_receipt(receipt: dict[str, Any], policy: dict[str, Any], *, run_id: str, arm: str, phase: str,
                   query_pairs: list[dict[str, str]]) -> dict[str, Any]:
    """Verify signer and independently recompute all coverage/stage summaries."""
    if not isinstance(receipt, dict) or set(receipt) != RECEIPT_FIELDS:
        raise ValueError("stage receipt field allowlist mismatch")
    expected_digest = query_set_digest(query_pairs)
    if receipt.get("run_id") != run_id or receipt.get("arm") != arm or receipt.get("phase") != phase:
        raise ValueError("stage receipt run/arm/phase differs from measured report")
    if receipt.get("case_query_set_sha256") != expected_digest or receipt.get("case_count") != len(query_pairs):
        raise ValueError("stage receipt differs from the frozen case/query set")
    sources = receipt.get("source_evidence")
    if not isinstance(sources, dict) or set(sources) != SOURCE_FIELDS:
        raise ValueError("stage receipt source evidence field allowlist mismatch")
    payload = {key: value for key, value in receipt.items() if key != "signature_base64"}
    source_hash = sources.get("evidence_sha256")
    if not is_sha256(source_hash):
        raise ValueError("stage source evidence digest is invalid")
    try:
        signature = base64.b64decode(receipt.get("signature_base64", ""), validate=True)
    except (TypeError, ValueError) as exc:
        raise ValueError("stage receipt signature encoding is invalid") from exc
    keys = validate_trust_policy(policy, expected_purpose=STAGE_PURPOSE)
    key_id = receipt.get("key_id")
    key = keys.get(key_id)
    if not key or key.get("enabled") is not True or key.get("issuer") != receipt.get("issuer"):
        raise ValueError("stage signer is not enabled in the pinned stage telemetry trust policy")
    if not rsa_pkcs1_sha256_verify(signing_bytes(payload), signature, int(key["modulus_hex"], 16), key["public_exponent"]):
        raise ValueError("stage receipt signature verification failed")
    normalized = produce_unsigned_receipt(receipt_id=receipt["receipt_id"], issuer=receipt["issuer"], key_id=key_id,
        run_id=run_id, arm=arm, phase=phase, query_pairs=query_pairs,
        export={"schema_version": 1, "run_id": run_id, "arm": arm, "phase": phase, "complete": True,
                "truncated": False, "case_coverage": receipt["case_coverage"]}, source_evidence=sources)
    if normalized != payload:
        raise ValueError("stage receipt summaries do not match per-case coverage")
    return {"receipt_id": receipt["receipt_id"], "issuer": receipt["issuer"], "key_id": key_id,
            "purpose": key["purpose"],
            "run_id": run_id, "arm": arm, "phase": phase, "case_query_set_sha256": expected_digest,
            "case_count": len(query_pairs), "stages": normalized["stages"], "source_evidence": sources,
            "case_coverage": normalized["case_coverage"],
            "key_admission_receipt_id": key["admission_receipt_id"],
            "key_admission_receipt_sha256": key["admission_receipt_sha256"],
            "key_fingerprint_sha256": key["key_fingerprint_sha256"], "reviewer_ids": key["reviewer_ids"],
            "verified": True}


def compare_verified_receipts(baseline: dict[str, Any], candidate: dict[str, Any], *, quality_gate: bool,
                              baseline_accepted_parts: int | None = None, candidate_accepted_parts: int | None = None,
                              baseline_accepted_tasks: int | None = None, candidate_accepted_tasks: int | None = None) -> dict[str, Any]:
    """Make per-pair stage timing diagnostics over the same independently graded cases.

    These are descriptive latency summaries, not CPU/memory attribution or a
    floor. The manifest combines per-pair ratios over repeated runs.
    """
    if not baseline.get("verified") or not candidate.get("verified"):
        raise ValueError("both stage receipts must be signature-verified")
    if (baseline.get("run_id"), baseline.get("phase"), baseline.get("case_query_set_sha256"), baseline.get("case_count")) != (
            candidate.get("run_id"), candidate.get("phase"), candidate.get("case_query_set_sha256"), candidate.get("case_count")):
        raise ValueError("stage receipts must identify the same paired run, phase, and frozen case set")
    indexed = []
    for receipt in (baseline, candidate):
        cases = receipt.get("case_coverage")
        if not isinstance(cases, list):
            raise ValueError("verified stage receipt lacks per-case coverage")
        mapped = {row.get("case_id"): row for row in cases if isinstance(row, dict)}
        if len(mapped) != len(cases):
            raise ValueError("stage receipt has duplicate or malformed cases")
        indexed.append(mapped)
    base_cases, candidate_cases = indexed
    if base_cases.keys() != candidate_cases.keys():
        raise ValueError("paired stage receipts have different case coverage")

    def percentile(values: list[float], fraction: float) -> float | None:
        if not values:
            return None
        ordered = sorted(values)
        index = min(len(ordered) - 1, max(0, int((len(ordered) - 1) * fraction + 0.5)))
        return ordered[index]

    stage_results = {}
    for stage_id in STAGE_IDS:
        base_values, candidate_values = [], []
        coverage_mismatch = False
        for case_id in sorted(base_cases):
            base_row, candidate_row = base_cases[case_id], candidate_cases[case_id]
            if base_row.get("query_sha256") != candidate_row.get("query_sha256"):
                raise ValueError(f"stage query hash differs for case {case_id}")
            base_stage = base_row.get("stages", {}).get(stage_id, {})
            candidate_stage = candidate_row.get("stages", {}).get(stage_id, {})
            if base_stage.get("executed") is not candidate_stage.get("executed"):
                coverage_mismatch = True
            elif base_stage.get("executed") is True:
                base_values.append(base_stage["duration_ms"])
                candidate_values.append(candidate_stage["duration_ms"])
        if coverage_mismatch or not base_values:
            stage_results[stage_id] = {"status": "unavailable_stage_coverage_mismatch" if coverage_mismatch else "unavailable_no_executed_cases",
                                       "quality_gate": quality_gate, "paired_case_count": 0,
                                       "baseline_ms": None, "candidate_ms": None, "candidate_to_baseline_total_duration_ratio": None}
            continue
        base_total, candidate_total = sum(base_values), sum(candidate_values)
        resource_base = baseline.get("stages", {}).get(stage_id, {})
        resource_candidate = candidate.get("stages", {}).get(stage_id, {})
        resource_result = {"status": "unavailable_stage_cpu_memory_not_measured",
                           "cpu_ms": None, "cpu_ms_ratio": None,
                           "peak_memory_bytes": None, "peak_memory_ratio": None}
        if (resource_base.get("resource_status") == "owner_attributed_diagnostic" and
                resource_candidate.get("resource_status") == "owner_attributed_diagnostic" and
                resource_base.get("resource_method_id") == resource_candidate.get("resource_method_id") and
                resource_base.get("resource_scope_receipt_id") == resource_candidate.get("resource_scope_receipt_id") and
                resource_base.get("resource_scope_uri") == resource_candidate.get("resource_scope_uri") and
                resource_base.get("resource_scope_sha256") == resource_candidate.get("resource_scope_sha256")):
            base_cpu, candidate_cpu = resource_base["cpu_ms_total"], resource_candidate["cpu_ms_total"]
            base_mem, candidate_mem = resource_base["peak_memory_bytes"], resource_candidate["peak_memory_bytes"]
            resource_result = {"status": "descriptive_owner_attributed_stage_resource",
                "method_id": resource_base["resource_method_id"], "scope_sha256": resource_base["resource_scope_sha256"],
                "scope_receipt_id": resource_base["resource_scope_receipt_id"],
                "scope_uri": resource_base["resource_scope_uri"],
                "cpu_ms": {"baseline": base_cpu, "candidate": candidate_cpu},
                "cpu_ms_ratio": candidate_cpu / base_cpu if base_cpu > 0 else None,
                "peak_memory_bytes": {"baseline": base_mem, "candidate": candidate_mem},
                "peak_memory_ratio": candidate_mem / base_mem if base_mem > 0 else None,
                "interpretation": "owner-attributed stage diagnostics; separate stages are not summed and values do not feed the total serving floor"}
            denominator_pairs = {
                "cpu_ms_per_accepted_part": (baseline_accepted_parts, candidate_accepted_parts),
                "cpu_ms_per_accepted_task": (baseline_accepted_tasks, candidate_accepted_tasks),
            }
            for metric_name, (baseline_denominator, candidate_denominator) in denominator_pairs.items():
                resource_result[metric_name] = ({
                    "baseline": base_cpu / baseline_denominator,
                    "candidate": candidate_cpu / candidate_denominator,
                    "candidate_to_baseline_ratio": (candidate_cpu / candidate_denominator) / (base_cpu / baseline_denominator),
                } if type(baseline_denominator) is int and baseline_denominator > 0 and
                     type(candidate_denominator) is int and candidate_denominator > 0 and base_cpu > 0 else None)
        stage_results[stage_id] = {
            "status": "descriptive_pair_stage_latency",
            "quality_gate": quality_gate,
            "paired_case_count": len(base_values),
            "baseline_not_applicable_case_count": len(base_cases) - len(base_values),
            "candidate_not_applicable_case_count": len(candidate_cases) - len(candidate_values),
            "baseline_ms": {"total": base_total, "p50": percentile(base_values, 0.50), "p95": percentile(base_values, 0.95)},
            "candidate_ms": {"total": candidate_total, "p50": percentile(candidate_values, 0.50), "p95": percentile(candidate_values, 0.95)},
            "candidate_to_baseline_total_duration_ratio": candidate_total / base_total,
            "resource_attribution": resource_result,
            "interpretation": "latency diagnostics; optional owner-attributed resource values remain separate diagnostics and do not feed the serving floor",
        }
    resource_available = any((item.get("resource_attribution") or {}).get("status") == "descriptive_owner_attributed_stage_resource"
                             for item in stage_results.values())
    return {"status": "descriptive_pair_stage_latency", "quality_gate": quality_gate,
            "measurement_dimension": "latency_and_owner_attributed_resource_diagnostics" if resource_available else "latency_only",
            "resource_attribution_status": "descriptive_owner_attributed_stage_resources_available" if resource_available else "unavailable_stage_cpu_memory_not_measured",
            "stages": stage_results}
