#!/usr/bin/env python3
"""Fail-closed verification for owner-signed provider/model serving scope receipts.

The trust policy is source-controlled and must pin the issuer public key. No
key is trusted by default. RSA PKCS#1 v1.5/SHA-256 verification uses Python's
standard library only; this verifies authenticity, not the truth of an owner's
measurement claims.
"""

from __future__ import annotations

import base64
import hashlib
import hmac
import json
from pathlib import Path
from typing import Any


RECEIPT_FIELDS = {
    "schema_version", "receipt_id", "issuer", "key_id", "signature_algorithm",
    "resource_source", "resource_group", "resource_group_definition_sha256",
    "coverage_status", "components", "unmeasured_costs", "signature_base64",
}
COMPONENT_FIELDS = {"component_id", "owner_id", "kind"}
TRUST_POLICY_FIELDS = {"schema_version", "keys"}
TRUST_KEY_FIELDS = {"issuer", "key_id", "modulus_hex", "public_exponent", "enabled", "purpose",
                    "key_fingerprint_sha256", "admission_receipt_id", "admission_receipt_sha256", "reviewer_ids"}
ADMISSION_FIELDS = {"schema_version", "receipt_id", "purpose", "issuer", "key_id", "key_fingerprint_sha256",
                    "approval_status", "owner_acknowledgement_ref", "manager_review_ref", "reviewer_ids"}
REQUIRED_KINDS = {"search_hub", "provider", "model"}
SCOPE_PURPOSE = "search_hub_provider_model_scope"
TELEMETRY_PURPOSE = "search_hub_run_telemetry"
SHA256_DER_PREFIX = bytes.fromhex("3031300d060960864801650304020105000420")


def canonical_bytes(value: dict[str, Any]) -> bytes:
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode("utf-8")


def read_scope_receipt(path: Path) -> dict[str, Any]:
    """Read a signed receipt while rejecting duplicate-key JSON ambiguity."""
    def unique_object(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
        result: dict[str, Any] = {}
        for key, value in pairs:
            if key in result:
                raise ValueError(f"duplicate JSON field in scope receipt: {key}")
            result[key] = value
        return result
    receipt = json.loads(path.read_text(), object_pairs_hook=unique_object)
    if not isinstance(receipt, dict):
        raise ValueError("provider/model scope receipt must be a JSON object")
    return receipt


def group_definition_sha256(value: dict[str, Any]) -> str:
    return hashlib.sha256(canonical_bytes(value)).hexdigest()


def public_key_fingerprint(issuer: str, key_id: str, modulus_hex: str, exponent: int) -> str:
    normalized_modulus = format(int(modulus_hex, 16), "x")
    return hashlib.sha256(canonical_bytes({"issuer": issuer, "key_id": key_id,
        "modulus_hex": normalized_modulus, "public_exponent": exponent})).hexdigest()


def validate_trust_policy(policy: dict[str, Any], *, expected_purpose: str = SCOPE_PURPOSE) -> dict[str, dict[str, Any]]:
    if not isinstance(policy, dict) or set(policy) != TRUST_POLICY_FIELDS or type(policy.get("schema_version")) is not int or policy["schema_version"] != 1 or not isinstance(policy.get("keys"), list):
        raise ValueError("invalid purpose-limited trust-policy-v1 shape")
    indexed: dict[str, dict[str, Any]] = {}
    for entry in policy["keys"]:
        if not isinstance(entry, dict) or set(entry) != TRUST_KEY_FIELDS:
            raise ValueError("trusted scope key does not match the key-admission field allowlist")
        if any(not isinstance(entry.get(field), str) or not entry[field] for field in
               ("issuer", "key_id", "modulus_hex", "purpose", "admission_receipt_id")):
            raise ValueError("trusted key requires issuer, key ID, public modulus, purpose, and admission receipt ID")
        if entry["purpose"] != expected_purpose or type(entry.get("enabled")) is not bool:
            raise ValueError("trusted key has an unsupported purpose or invalid enabled flag")
        try:
            modulus = int(entry["modulus_hex"], 16)
        except ValueError as exc:
            raise ValueError("trusted RSA modulus must be hexadecimal") from exc
        exponent = entry.get("public_exponent")
        if modulus.bit_length() < 2048 or type(exponent) is not int or exponent < 3 or exponent % 2 == 0:
            raise ValueError("trusted RSA key must have a >=2048-bit modulus and odd public exponent >=3")
        if entry["modulus_hex"] != format(modulus, "x"):
            raise ValueError("trusted modulus must use lowercase canonical hexadecimal without leading zeros")
        if entry.get("key_fingerprint_sha256") != public_key_fingerprint(entry["issuer"], entry["key_id"], entry["modulus_hex"], exponent):
            raise ValueError("trusted public-key fingerprint does not match key material")
        if not isinstance(entry.get("admission_receipt_sha256"), str) or len(entry["admission_receipt_sha256"]) != 64 or any(ch not in "0123456789abcdef" for ch in entry["admission_receipt_sha256"]):
            raise ValueError("trusted key requires a lowercase SHA-256 key-admission receipt digest")
        reviewers = entry.get("reviewer_ids")
        if not isinstance(reviewers, list) or len(reviewers) < 2 or any(not isinstance(value, str) or not value for value in reviewers) or len(set(reviewers)) != len(reviewers):
            raise ValueError("key admission requires at least two distinct reviewer IDs")
        if entry["key_id"] in indexed:
            raise ValueError("duplicate trusted scope key ID")
        indexed[entry["key_id"]] = entry
    return indexed


def validate_key_admission_receipt(receipt: dict[str, Any], key: dict[str, Any]) -> dict[str, Any]:
    if not isinstance(receipt, dict) or set(receipt) != ADMISSION_FIELDS:
        raise ValueError("key-admission receipt field allowlist mismatch")
    if type(receipt.get("schema_version")) is not int or receipt["schema_version"] != 1:
        raise ValueError("unsupported key-admission receipt version")
    for field, expected in (("receipt_id", key["admission_receipt_id"]), ("purpose", key["purpose"]),
                            ("issuer", key["issuer"]), ("key_id", key["key_id"]),
                            ("key_fingerprint_sha256", key["key_fingerprint_sha256"]),
                            ("approval_status", "approved"), ("reviewer_ids", key["reviewer_ids"])):
        if receipt.get(field) != expected:
            raise ValueError(f"key-admission receipt {field} differs from pinned trust policy")
    for field in ("owner_acknowledgement_ref", "manager_review_ref"):
        if not isinstance(receipt.get(field), str) or not receipt[field]:
            raise ValueError(f"key-admission receipt requires {field}")
    return {"receipt_id": receipt["receipt_id"], "issuer": receipt["issuer"], "key_id": receipt["key_id"],
            "key_fingerprint_sha256": receipt["key_fingerprint_sha256"], "reviewer_ids": receipt["reviewer_ids"],
            "approval_status": "approved"}


def verify_key_admission_artifact(receipt: dict[str, Any], artifact_sha256: str,
                                  scope_verification: dict[str, Any]) -> dict[str, Any]:
    if artifact_sha256 != scope_verification.get("key_admission_receipt_sha256"):
        raise ValueError("key-admission receipt hash differs from source-pinned trust policy")
    expected = {
        "admission_receipt_id": scope_verification.get("key_admission_receipt_id"),
        "purpose": scope_verification.get("purpose"),
        "issuer": scope_verification.get("issuer"), "key_id": scope_verification.get("key_id"),
        "key_fingerprint_sha256": scope_verification.get("key_fingerprint_sha256"),
        "reviewer_ids": scope_verification.get("reviewer_ids"),
    }
    return validate_key_admission_receipt(receipt, expected)


def rsa_pkcs1_sha256_verify(message: bytes, signature: bytes, modulus: int, exponent: int) -> bool:
    """Verify strict EMSA-PKCS1-v1_5 encoding; caller must enforce key policy."""
    if modulus <= 0 or exponent < 3 or exponent % 2 == 0:
        return False
    size = (modulus.bit_length() + 7) // 8
    if len(signature) != size or int.from_bytes(signature, "big") >= modulus:
        return False
    encoded = pow(int.from_bytes(signature, "big"), exponent, modulus).to_bytes(size, "big")
    digest_info = SHA256_DER_PREFIX + hashlib.sha256(message).digest()
    padding_size = size - len(digest_info) - 3
    if padding_size < 8:
        return False
    expected = b"\x00\x01" + b"\xff" * padding_size + b"\x00" + digest_info
    return hmac.compare_digest(encoded, expected)


def verify_scope_receipt(receipt: dict[str, Any], policy: dict[str, Any], *, resource_source: str,
                         resource_group: str, group_definition: dict[str, Any]) -> dict[str, Any]:
    if set(receipt) != RECEIPT_FIELDS:
        raise ValueError("provider/model scope receipt field allowlist mismatch")
    if type(receipt.get("schema_version")) is not int or receipt["schema_version"] != 1:
        raise ValueError("unsupported provider/model scope receipt schema version")
    if receipt.get("signature_algorithm") != "rsa-pkcs1v15-sha256":
        raise ValueError("unsupported provider/model scope signature algorithm")
    for field in ("receipt_id", "issuer", "key_id", "resource_group"):
        if not isinstance(receipt.get(field), str) or not receipt[field]:
            raise ValueError(f"scope receipt requires {field}")
    if receipt.get("resource_source") != resource_source or receipt.get("resource_group") != resource_group:
        raise ValueError("scope receipt resource source/group differs from measured attribution")
    expected_definition_hash = group_definition_sha256(group_definition)
    if receipt.get("resource_group_definition_sha256") != expected_definition_hash:
        raise ValueError("scope receipt resource-group definition hash mismatch")
    components = receipt.get("components")
    if not isinstance(components, list) or not components:
        raise ValueError("scope receipt requires component coverage rows")
    component_ids = []
    component_kinds = set()
    for item in components:
        if not isinstance(item, dict) or set(item) != COMPONENT_FIELDS:
            raise ValueError("scope component rows must contain only component_id, owner_id, and kind")
        if any(not isinstance(item.get(field), str) or not item[field] for field in COMPONENT_FIELDS):
            raise ValueError("scope component identity fields must be nonempty strings")
        component_ids.append(item["component_id"])
        component_kinds.add(item["kind"])
    if len(component_ids) != len(set(component_ids)) or not REQUIRED_KINDS.issubset(component_kinds):
        raise ValueError("scope must uniquely cover Search Hub, provider, and model components")
    if receipt.get("coverage_status") != "complete" or receipt.get("unmeasured_costs") != []:
        raise ValueError("scope receipt has incomplete or unmeasured serving cost")
    trusted_keys = validate_trust_policy(policy)
    key = trusted_keys.get(receipt["key_id"])
    if key is None:
        raise ValueError("scope signer key is not uniquely pinned by source trust policy")
    if key.get("issuer") != receipt["issuer"] or key.get("enabled") is not True:
        raise ValueError("scope signer is disabled or issuer identity differs from pinned key")
    try:
        modulus = int(key["modulus_hex"], 16)
        exponent = key["public_exponent"]
        signature = base64.b64decode(receipt["signature_base64"], validate=True)
    except (KeyError, TypeError, ValueError) as exc:
        raise ValueError("invalid pinned public key or base64 signature") from exc
    if type(exponent) is not int or modulus.bit_length() < 2048:
        raise ValueError("pinned scope key must use an integer exponent and at least a 2048-bit RSA modulus")
    payload = {field: value for field, value in receipt.items() if field != "signature_base64"}
    if not rsa_pkcs1_sha256_verify(canonical_bytes(payload), signature, modulus, exponent):
        raise ValueError("provider/model scope receipt signature verification failed")
    return {"receipt_id": receipt["receipt_id"], "issuer": receipt["issuer"], "key_id": receipt["key_id"],
            "resource_source": resource_source, "resource_group": resource_group,
            "resource_group_definition_sha256": expected_definition_hash,
            "component_count": len(components), "component_kinds": sorted(component_kinds),
            "coverage_status": "complete", "signature_status": "verified",
            "key_admission_receipt_id": key["admission_receipt_id"],
            "purpose": key["purpose"],
            "key_admission_receipt_sha256": key["admission_receipt_sha256"],
            "key_fingerprint_sha256": key["key_fingerprint_sha256"],
            "reviewer_ids": key["reviewer_ids"]}
