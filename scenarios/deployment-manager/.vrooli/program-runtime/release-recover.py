import json
inputs = program.inputs()
review_key = str(inputs.get("review_key", "")).strip(); release_id = str(inputs.get("release_id", "")).strip(); candidate_id = str(inputs.get("candidate_id", "")).strip(); destination_revision_id = str(inputs.get("destination_revision_id", "")).strip(); action = str(inputs.get("action", "halt")).strip(); execute = bool(inputs.get("execute", False)); confirmation = str(inputs.get("confirmation", "")); dry_run_reference = str(inputs.get("dry_run_reference", "")); expected_predecessor_revision = int(inputs.get("expected_predecessor_revision", 0)); data_compatibility = str(inputs.get("data_compatibility", "")).strip(); repair_artifact_ids = inputs.get("repair_artifact_ids", {}) or {}
envelope = {"program": "deployment-manager.release-recover", "version": "1", "status": "failed", "phase": "validate", "inputs": {"review_key": review_key, "release_id": release_id, "candidate_id": candidate_id, "destination_revision_id": destination_revision_id, "action": action, "execute": execute, "confirmation": bool(confirmation), "dry_run_reference": dry_run_reference, "expected_predecessor_revision": expected_predecessor_revision, "data_compatibility": data_compatibility, "repair_artifact_ids": repair_artifact_ids}, "signals": {}, "errors": [], "evidence": []}; handle = None
fail = program.fail
def step_validate():
    if not review_key: return fail("failed", "invalid_input", "review_key is required", "validate")
    if execute and (not release_id or not candidate_id or not destination_revision_id or not action): return fail("failed", "invalid_input", "execute requires release, candidate, destination, and action identity", "validate")
    if action not in ("halt", "withdraw", "rollback", "forward_repair"): return fail("failed", "invalid_input", "action must be halt, withdraw, rollback, or forward_repair", "validate")
    if execute and (not confirmation or not dry_run_reference): return fail("refused", "no_grant", "execute requires confirmation and dry-run evidence", "validate")
    if execute and action in ("rollback", "forward_repair") and data_compatibility != "compatible": return fail("refused", "binding_error", "rollback and forward repair require data_compatibility=compatible", "validate")
    if execute and action == "rollback" and expected_predecessor_revision <= 0: return fail("failed", "invalid_input", "rollback requires expected_predecessor_revision", "validate")
    if execute and action == "forward_repair" and not isinstance(repair_artifact_ids, dict): return fail("failed", "invalid_input", "forward_repair requires repair_artifact_ids", "validate")
    return "collect"
def step_collect():
    global handle; envelope["phase"] = "collect"
    try: handle = deployment_manager.readiness_reviews.get(review_key=review_key, rows="findings"); return "decide"
    except Exception as exc:
        status, klass = program.classify(exc); return fail(status, klass, exc, "collect")
def step_decide():
    envelope["phase"] = "decide"; meta = handle.meta() or {}; status = meta.get("status"); decision = "observe" if status in ("approved", "promoted") else "repair_blockers"
    envelope["signals"] = {"decision": decision, "review_status": status, "target_identity": review_key, "required_controls": ["exact owner binding", "explicit grant", "confirmation", "dry-run evidence"]}; envelope["evidence"] = [review_key] + ([dry_run_reference] if dry_run_reference else [])
    if execute and status not in ("approved", "promoted"):
        return fail("refused", "not_run_eligible", "recovery execution requires an approved or promoted canonical review", "decide")
    return "act" if execute else "report"
def step_act():
    envelope["phase"] = "act"
    try:
        result = deployment_manager.releases.recover(release_id=release_id, review_key=review_key, candidate_id=candidate_id, destination_revision_id=destination_revision_id, action=action, expected_predecessor_revision=expected_predecessor_revision, data_compatibility=data_compatibility, repair_artifact_ids=repair_artifact_ids, confirmation=confirmation, dry_run=False).head(1)
        row = result[0] if result else {}
        receipt = row.get("receipt") if isinstance(row, dict) else None
        if (not isinstance(receipt, dict) or receipt.get("release_id") != release_id or
                receipt.get("candidate_id") != candidate_id or
                receipt.get("destination_revision_id") != destination_revision_id or
                not receipt.get("deployment_id") or not receipt.get("external_receipt") or
                receipt.get("action") != action or
                receipt.get("outcome") not in ("halted", "withdrawn", "rolled_back", "forward_repaired")):
            return fail("failed", "ambiguous_response", "owner recovery did not return a durable effect receipt", "act")
        envelope["signals"].update({"owner_receipt": receipt, "outcome": "recovered"}); envelope["evidence"].append("receipt:" + str(receipt["external_receipt"])); envelope["status"] = "ok"; return "report"
    except Exception as exc:
        message = str(exc)
        lowered = message.lower()
        if "owner" in lowered and ("unavailable" in lowered or "unsupported" in lowered or "no receipt" in lowered):
            return fail("unavailable", "no_governed_binding", message, "act")
        status, klass = program.classify(exc); return fail(status, klass, exc, "act")
def step_report(): envelope["phase"] = "report"; envelope["status"] = "ok" if not envelope["errors"] else envelope["status"]; print(json.dumps(envelope, allow_nan=False)); return None
STATES = {"validate": step_validate, "collect": step_collect, "decide": step_decide, "act": step_act, "report": step_report}; state = "validate"
program.run(STATES, state)
