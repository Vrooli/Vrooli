"""Admit one bounded Bridge qualification request through Test Genie's owner contract."""
import json
import re

inputs = program.inputs()
envelope = program.envelope("vrooli-bridge.qualification", "1")
envelope["status"] = "failed"
envelope["phase"] = "validate"
envelope["signals"] = {"outcome": "unknown"}

request_key = str(inputs.get("request_key", "") or "").strip()
candidate_digest = str(inputs.get("candidate_digest", "") or "").strip()
target_ids = inputs.get("target_ids", [])
authorized_target_ids = inputs.get("authorized_target_ids", [])
authorization_ref = str(inputs.get("authorization_ref", "") or "").strip()
phases = inputs.get("phases", ["business", "storage", "contracts"])
evidence_cells = inputs.get("evidence_cells", [])
input_local_run_ids = inputs.get("local_run_ids", [])
plan_only = inputs.get("plan_only", False)


def is_digest(value):
    return bool(re.fullmatch(r"sha256:[0-9a-f]{64}", value))


def normalized_strings(values, label):
    if not isinstance(values, list) or not values or len(values) > 32:
        raise ValueError(label + " must be a non-empty list of at most 32 strings")
    normalized = []
    for value in values:
        if not isinstance(value, str) or not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}", value):
            raise ValueError(label + " contains an invalid identity")
        if value not in normalized:
            normalized.append(value)
    return sorted(normalized)


def step_validate():
    if not request_key or len(request_key) > 128 or not re.fullmatch(r"[A-Za-z0-9_.:-]+", request_key):
        return program.fail("failed", "invalid_input", "request_key must be a stable bounded identity", "validate")
    if not is_digest(candidate_digest):
        return program.fail("failed", "invalid_input", "candidate_digest must be sha256:<64 lowercase hex>", "validate")
    try:
        requested = normalized_strings(target_ids, "target_ids")
        authorized = sorted(set(normalized_strings(authorized_target_ids, "authorized_target_ids"))) if authorized_target_ids else []
    except ValueError as exc:
        return program.fail("failed", "invalid_input", str(exc), "validate")
    if not authorization_ref:
        return program.fail("refused", "no_grant", "explicit authorization_ref is required before qualification admission", "validate")
    missing = sorted(set(requested) - set(authorized))
    if missing:
        return program.fail("refused", "no_grant", "target authority does not cover: " + ", ".join(missing[:8]), "validate")
    if not isinstance(phases, list) or not phases or len(phases) > 8:
        return program.fail("failed", "invalid_input", "phases must contain 1..8 explicit Test Genie phase names", "validate")
    if any(not isinstance(phase, str) or not re.fullmatch(r"[a-z][a-z0-9-]{0,63}", phase) for phase in phases):
        return program.fail("failed", "invalid_input", "phases contain an invalid phase name", "validate")
    normalized_phases = sorted(set(phases))
    if not isinstance(evidence_cells, list) or len(evidence_cells) > 32:
        return program.fail("failed", "invalid_input", "evidence_cells must contain at most 32 rows", "validate")
    cells = []
    allowed_statuses = {"passed", "failed", "stale", "unavailable", "pending"}
    for cell in evidence_cells[:8]:
        if not isinstance(cell, dict) or not isinstance(cell.get("cell_id"), str) or not cell.get("cell_id"):
            return program.fail("failed", "invalid_input", "each evidence cell requires cell_id", "validate")
        status = cell.get("status", "pending")
        if status not in allowed_statuses:
            return program.fail("failed", "invalid_input", "evidence cell status must be passed, failed, stale, unavailable, or pending", "validate")
        cells.append({"cell_id": cell["cell_id"], "status": status, "evidence_ref": str(cell.get("evidence_ref", ""))[:256], "source": "caller_observation", "lane_eligible": False})
    if not isinstance(input_local_run_ids, list) or len(input_local_run_ids) > 8:
        return program.fail("failed", "invalid_input", "local_run_ids must contain at most 8 Bridge run ids", "validate")
    if any(not isinstance(run_id, str) or not re.fullmatch(r"[A-Za-z0-9_.:-]{1,128}", run_id) for run_id in input_local_run_ids):
        return program.fail("failed", "invalid_input", "local_run_ids contain an invalid Bridge run identity", "validate")
    local_run_ids = sorted(set(input_local_run_ids))
    local_receipts = []
    if local_run_ids:
        for run_id in local_run_ids:
            try:
                rows = vrooli_bridge.runs.local_receipt(id=run_id).head(1)
            except Exception as exc:
                status, klass = program.classify(exc)
                return program.fail("unavailable", klass, "Bridge local receipt read failed for " + run_id, "validate")
            receipt = rows[0] if rows else None
            if not isinstance(receipt, dict) or receipt.get("runId") != run_id:
                return program.fail("failed", "binding_error", "Bridge local receipt read returned no matching persisted run", "validate")
            if receipt.get("lane") != "local-contract" or receipt.get("targetClass") != "local-fixture" or receipt.get("provenance") != "local-fixture" or receipt.get("laneEligible") is not False:
                return program.fail("failed", "binding_error", "Bridge receipt is not pinned to local-only provenance", "validate")
            run_status = str(receipt.get("status", ""))
            exit_code = receipt.get("exitCode")
            if run_status not in {"passed", "failed", "aborted", "failed_delivery"}:
                return program.fail("failed", "binding_error", "Bridge receipt is not terminal", "validate")
            if run_status == "passed" and exit_code != 0:
                return program.fail("failed", "binding_error", "Bridge passed receipt has a nonzero exit code", "validate")
            if run_status == "failed" and exit_code == 0:
                return program.fail("failed", "binding_error", "Bridge failed receipt has a zero exit code", "validate")
            local_status = "local_pass_observed" if run_status == "passed" and exit_code == 0 else "local_failure_observed"
            local_receipts.append({
                "run_id": run_id, "target_id": str(receipt.get("targetId", "")),
                "scenario": str(receipt.get("scenario", "")), "verb": str(receipt.get("verb", "")),
                "status": local_status, "run_status": run_status, "exit_code": exit_code,
                "evidence_refs": [str(ref)[:256] for ref in (receipt.get("evidenceRefs", []) or [])[:8]],
                "lane": "local-contract", "target_class": "local-fixture", "provenance": "local-fixture",
                "lane_eligible": False, "qualifies_remote_lane": False,
            })
    envelope["signals"]["request"] = {"request_key": request_key, "candidate_digest": candidate_digest, "target_ids": requested, "phases": normalized_phases}
    envelope["signals"]["authority"] = {"authorized": True, "authorization_ref": authorization_ref, "target_count": len(requested)}
    envelope["signals"]["cells"] = cells
    envelope["signals"]["local_receipts"] = local_receipts
    return "act"


def step_act():
    envelope["phase"] = "act"
    intent = {
        "schema_version": 1,
        "idempotency_key": request_key,
        "caller_scenario": "vrooli-bridge",
        "caller_execution_id": request_key,
        "purpose": "VALIDATION_PURPOSE_PHASE",
        "required_strength": "VALIDATION_STRENGTH_TARGETED",
        "targets": [{"kind": "VALIDATION_TARGET_KIND_SCENARIO", "id": "vrooli-bridge", "root": "scenarios/vrooli-bridge"}],
        "phases": envelope["signals"]["request"]["phases"],
        "reuse_policy": {"mode": "REUSE_MODE_ATTACH_OR_TERMINAL", "maximum_age": "3600s"},
        "concurrency_policy": {"mode": "CONCURRENCY_MODE_SHARED_COMPATIBLE", "maximum_parallelism": 1},
        "evidence_policy": {"required_evidence_kinds": ["test-genie-run"]},
        "deadline_policy": {"queue_budget": "300s", "execution_budget": "3600s", "maximum_attempts": 1},
        "content_inputs": [{"name": "candidate", "root": "scenarios/vrooli-bridge", "selections": [{"glob": "**", "required": True}]}],
        "caller_attributes": {"candidate_digest": candidate_digest, "target_ids": ",".join(envelope["signals"]["request"]["target_ids"]), "authorization_ref": authorization_ref},
    }
    envelope["signals"]["intent"] = {"idempotency_key": request_key, "candidate_digest": candidate_digest, "target_count": len(envelope["signals"]["request"]["target_ids"]), "phases": intent["phases"]}
    envelope["evidence"] = ["candidate:" + candidate_digest, "authorization:" + authorization_ref]
    envelope["evidence"].extend("bridge-local-run:" + receipt["run_id"] for receipt in envelope["signals"].get("local_receipts", []))
    if plan_only:
        envelope["status"] = "ok"
        envelope["signals"]["outcome"] = "planned"
        return "report"
    try:
        handle = test_genie.validation.create(intent=intent)
        rows = handle.head(1)
        receipt = rows[0] if rows else handle.meta().get("receipt", {})
        if "receipt" in receipt:
            receipt = receipt["receipt"]
        receipt_id = receipt.get("receiptId", "")
        if not re.fullmatch(r"[A-Za-z0-9_-]{1,128}", receipt_id):
            return program.fail("failed", "binding_error", "Test Genie returned no valid receipt id", "act")
        envelope["status"] = "ok"
        envelope["signals"]["outcome"] = "admitted"
        envelope["signals"]["receipt"] = {key: receipt.get(key) for key in ("receiptId", "state", "revision", "reasonCode", "createdAt", "updatedAt")}
        envelope["signals"]["resume_command"] = "test-genie validation get " + receipt_id + " --json"
        envelope["signals"]["wait_command"] = "test-genie validation wait " + receipt_id + " --wait-id " + receipt_id + "-bridge-qualification --json"
        envelope["evidence"].append("test-genie-validation:" + receipt_id)
    except program.BindingError as exc:
        if "validation idempotency key was reused with different intent" in str(exc):
            return program.fail("refused", "idempotency_conflict", str(exc)[:240], "act")
        status, klass = program.classify(exc)
        return program.fail(status, klass, str(exc)[:240], "act")
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "act": step_act, "report": step_report}, "validate")
