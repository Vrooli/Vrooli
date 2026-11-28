#!/usr/bin/env python3
"""Producer adapter and fail-closed verifier for run-correlated token/cache receipts.

The adapter reduces owner-exported, case-correlated metadata into a signing
payload. The authorized telemetry owner signs that payload; the private key is
never read by SH4. Verification needs an explicitly pinned purpose-limited key.
"""

from __future__ import annotations

import base64
import hashlib
from typing import Any

from scope_receipts import TELEMETRY_PURPOSE, canonical_bytes, rsa_pkcs1_sha256_verify, validate_trust_policy


RECEIPT_FIELDS = {"schema_version", "receipt_id", "issuer", "key_id", "signature_algorithm", "run_id", "arm", "phase",
                  "case_query_set_sha256", "case_count", "gateway", "cache", "source_evidence", "signature_base64"}
SOURCE_FIELDS = {"kind", "receipt_id", "evidence_uri", "evidence_sha256"}
GATEWAY_EXPORT_FIELDS = {"schema_version", "run_id", "complete", "truncated", "case_coverage", "events"}
GATEWAY_COVERAGE_FIELDS = {"case_id", "query_sha256", "complete"}
GATEWAY_EVENT_FIELDS = {"event_id", "case_id", "query_sha256", "input_tokens", "output_tokens"}
CACHE_EXPORT_FIELDS = {"schema_version", "run_id", "complete", "case_coverage"}
CACHE_COVERAGE_FIELDS = {"case_id", "query_sha256", "complete", "hits", "misses"}


def is_sha256(value: Any) -> bool:
    return isinstance(value, str) and len(value) == 64 and all(char in "0123456789abcdef" for char in value)


def query_set_digest(query_pairs: list[dict[str, str]]) -> str:
    normalized = [{"case_id": row["case_id"], "query_sha256": row["query_sha256"]} for row in sorted(query_pairs, key=lambda r: r["case_id"])]
    return hashlib.sha256(canonical_bytes(normalized)).hexdigest()


def require_exact_pairs(rows: Any, expected: list[dict[str, str]], fields: set[str], label: str) -> dict[str, dict[str, Any]]:
    if not isinstance(rows, list):
        raise ValueError(f"{label} case coverage must be a list")
    indexed = {}
    for row in rows:
        if not isinstance(row, dict) or set(row) != fields:
            raise ValueError(f"{label} row field allowlist mismatch")
        case_id = row.get("case_id")
        if not isinstance(case_id, str) or not case_id or case_id in indexed:
            raise ValueError(f"{label} requires unique nonempty case IDs")
        indexed[case_id] = row
    expected_map = {row["case_id"]: row["query_sha256"] for row in expected}
    if indexed.keys() != expected_map.keys() or any(indexed[key].get("query_sha256") != value for key, value in expected_map.items()):
        raise ValueError(f"{label} case/query coverage does not exactly match the frozen workload")
    return indexed


def produce_unsigned_receipt(*, receipt_id: str, issuer: str, key_id: str, run_id: str, arm: str, phase: str,
                             query_pairs: list[dict[str, str]], gateway_export: dict[str, Any],
                             cache_export: dict[str, Any], source_evidence: list[dict[str, Any]]) -> dict[str, Any]:
    """Aggregate only complete case-correlated exports; return data for owner signing."""
    for label, value in (("receipt_id", receipt_id), ("issuer", issuer), ("key_id", key_id), ("run_id", run_id), ("arm", arm), ("phase", phase)):
        if not isinstance(value, str) or not value:
            raise ValueError(f"{label} must be nonempty")
    if not isinstance(query_pairs, list) or not query_pairs:
        raise ValueError("query_pairs must contain the exact frozen workload")
    normalized = []
    for row in query_pairs:
        if not isinstance(row, dict) or set(row) != {"case_id", "query_sha256"} or not isinstance(row["case_id"], str) or not row["case_id"] or not is_sha256(row["query_sha256"]):
            raise ValueError("query_pairs require case IDs and exact SHA-256 query hashes")
        normalized.append({"case_id": row["case_id"], "query_sha256": row["query_sha256"]})
    if len({row["case_id"] for row in normalized}) != len(normalized):
        raise ValueError("query_pairs contain duplicate case IDs")
    expected = sorted(normalized, key=lambda row: row["case_id"])
    for label, export, fields in (("Gateway", gateway_export, GATEWAY_EXPORT_FIELDS), ("cache", cache_export, CACHE_EXPORT_FIELDS)):
        if not isinstance(export, dict) or set(export) != fields or type(export.get("schema_version")) is not int or export["schema_version"] != 1:
            raise ValueError(f"{label} export schema/field allowlist mismatch")
        if export.get("run_id") != run_id or export.get("complete") is not True:
            raise ValueError(f"{label} export must be complete and bound to this run_id")
    if gateway_export.get("truncated") is not False:
        raise ValueError("Gateway export must explicitly attest truncated=false")
    gateway_cases = require_exact_pairs(gateway_export["case_coverage"], expected, GATEWAY_COVERAGE_FIELDS, "Gateway")
    cache_cases = require_exact_pairs(cache_export["case_coverage"], expected, CACHE_COVERAGE_FIELDS, "cache")
    if any(row.get("complete") is not True for row in gateway_cases.values()) or any(row.get("complete") is not True for row in cache_cases.values()):
        raise ValueError("every case must have explicit complete Gateway and cache coverage")
    event_ids: set[str] = set()
    input_tokens = output_tokens = 0
    for event in gateway_export["events"]:
        if not isinstance(event, dict) or set(event) != GATEWAY_EVENT_FIELDS:
            raise ValueError("Gateway event field allowlist mismatch")
        case_id = event.get("case_id")
        if case_id not in gateway_cases or event.get("query_sha256") != gateway_cases[case_id]["query_sha256"]:
            raise ValueError("Gateway event is outside the exact run/query cohort")
        event_id = event.get("event_id")
        if not isinstance(event_id, str) or not event_id or event_id in event_ids:
            raise ValueError("Gateway event IDs must be unique and nonempty")
        event_ids.add(event_id)
        for field in ("input_tokens", "output_tokens"):
            value = event.get(field)
            if type(value) is not int or value < 0:
                raise ValueError("Gateway token counts must be nonnegative integers")
        input_tokens += event["input_tokens"]
        output_tokens += event["output_tokens"]
    hits = misses = 0
    for row in cache_cases.values():
        for field in ("hits", "misses"):
            value = row.get(field)
            if type(value) is not int or value < 0:
                raise ValueError("cache hit/miss counts must be nonnegative integers")
        hits += row["hits"]
        misses += row["misses"]
    if not isinstance(source_evidence, list) or len(source_evidence) != 2:
        raise ValueError("exactly two source evidence references are required: gateway_export and cache_export")
    sources = {}
    for item in source_evidence:
        if not isinstance(item, dict) or set(item) != SOURCE_FIELDS or item.get("kind") not in ("gateway_export", "cache_export"):
            raise ValueError("source evidence field/kind allowlist mismatch")
        if any(not isinstance(item.get(field), str) or not item[field] for field in ("receipt_id", "evidence_uri")) or not is_sha256(item.get("evidence_sha256")):
            raise ValueError("source evidence needs a receipt ID, URI, and SHA-256 digest")
        if item["kind"] in sources:
            raise ValueError("duplicate source evidence kind")
        sources[item["kind"]] = item
    if set(sources) != {"gateway_export", "cache_export"}:
        raise ValueError("Gateway and cache owner evidence are both required")
    return {
        "schema_version": 1, "receipt_id": receipt_id, "issuer": issuer, "key_id": key_id,
        "signature_algorithm": "rsa-pkcs1v15-sha256", "run_id": run_id, "arm": arm, "phase": phase,
        "case_query_set_sha256": query_set_digest(expected), "case_count": len(expected),
        "gateway": {"complete": True, "coverage_case_count": len(gateway_cases), "event_count": len(event_ids),
                    "input_tokens": input_tokens, "output_tokens": output_tokens},
        "cache": {"complete": True, "coverage_case_count": len(cache_cases), "hits": hits, "misses": misses,
                  "lookup_count": hits + misses},
        "source_evidence": [sources["gateway_export"], sources["cache_export"]],
    }


def signing_bytes(unsigned_receipt: dict[str, Any]) -> bytes:
    if "signature_base64" in unsigned_receipt:
        raise ValueError("signing payload must omit signature_base64")
    return canonical_bytes(unsigned_receipt)


def verify_receipt(receipt: dict[str, Any], policy: dict[str, Any], *, run_id: str, arm: str, phase: str,
                   case_query_set_sha256: str, case_count: int) -> dict[str, Any]:
    if not isinstance(receipt, dict) or set(receipt) != RECEIPT_FIELDS:
        raise ValueError("run telemetry receipt field allowlist mismatch")
    if type(receipt.get("schema_version")) is not int or receipt["schema_version"] != 1:
        raise ValueError("unsupported run telemetry receipt schema version")
    for field, expected in (("run_id", run_id), ("arm", arm), ("phase", phase),
                            ("case_query_set_sha256", case_query_set_sha256), ("case_count", case_count)):
        if receipt.get(field) != expected:
            raise ValueError(f"run telemetry {field} differs from measured arm")
    for field in ("receipt_id", "issuer", "key_id"):
        if not isinstance(receipt.get(field), str) or not receipt[field]:
            raise ValueError(f"run telemetry receipt requires {field}")
    if receipt.get("signature_algorithm") != "rsa-pkcs1v15-sha256":
        raise ValueError("unsupported run telemetry signature algorithm")
    gateway, cache = receipt.get("gateway"), receipt.get("cache")
    if not isinstance(gateway, dict) or set(gateway) != {"complete", "coverage_case_count", "event_count", "input_tokens", "output_tokens"}:
        raise ValueError("Gateway telemetry summary field allowlist mismatch")
    if not isinstance(cache, dict) or set(cache) != {"complete", "coverage_case_count", "hits", "misses", "lookup_count"}:
        raise ValueError("cache telemetry summary field allowlist mismatch")
    if gateway.get("complete") is not True or cache.get("complete") is not True or gateway.get("coverage_case_count") != case_count or cache.get("coverage_case_count") != case_count:
        raise ValueError("token/cache receipt coverage is incomplete")
    counts = [gateway.get(key) for key in ("event_count", "input_tokens", "output_tokens")] + [cache.get(key) for key in ("hits", "misses", "lookup_count")]
    if any(type(value) is not int or value < 0 for value in counts) or cache["lookup_count"] != cache["hits"] + cache["misses"]:
        raise ValueError("token/cache summary counts are invalid")
    sources = receipt.get("source_evidence")
    if not isinstance(sources, list) or len(sources) != 2:
        raise ValueError("run telemetry receipt must reference both owner source exports")
    kinds = set()
    for item in sources:
        if not isinstance(item, dict) or set(item) != SOURCE_FIELDS or item.get("kind") not in ("gateway_export", "cache_export") or item["kind"] in kinds:
            raise ValueError("run telemetry source evidence rows are invalid")
        if not isinstance(item.get("receipt_id"), str) or not item["receipt_id"] or not isinstance(item.get("evidence_uri"), str) or not item["evidence_uri"] or not is_sha256(item.get("evidence_sha256")):
            raise ValueError("run telemetry source evidence needs receipt IDs, URIs, and SHA-256 hashes")
        kinds.add(item["kind"])
    if kinds != {"gateway_export", "cache_export"}:
        raise ValueError("run telemetry source evidence is missing Gateway or cache export")
    try:
        signature = base64.b64decode(receipt["signature_base64"], validate=True)
    except (KeyError, TypeError, ValueError) as exc:
        raise ValueError("invalid run telemetry signature encoding") from exc
    keys = validate_trust_policy(policy, expected_purpose=TELEMETRY_PURPOSE)
    key = keys.get(receipt["key_id"])
    if key is None or key["issuer"] != receipt["issuer"] or key["enabled"] is not True:
        raise ValueError("run telemetry signer is not enabled in the pinned telemetry trust policy")
    modulus = int(key["modulus_hex"], 16)
    unsigned = {field: value for field, value in receipt.items() if field != "signature_base64"}
    if not rsa_pkcs1_sha256_verify(canonical_bytes(unsigned), signature, modulus, key["public_exponent"]):
        raise ValueError("run telemetry receipt signature verification failed")
    return {"receipt_id": receipt["receipt_id"], "issuer": receipt["issuer"], "key_id": receipt["key_id"],
            "run_id": run_id, "arm": arm, "phase": phase, "case_query_set_sha256": case_query_set_sha256,
            "case_count": case_count, "gateway": gateway, "cache": cache, "source_evidence": sources,
            "key_admission_receipt_id": key["admission_receipt_id"],
            "key_admission_receipt_sha256": key["admission_receipt_sha256"],
            "key_fingerprint_sha256": key["key_fingerprint_sha256"], "reviewer_ids": key["reviewer_ids"],
            "verified": True}
