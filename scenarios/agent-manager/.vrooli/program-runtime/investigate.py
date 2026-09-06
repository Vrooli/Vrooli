"""agent-manager.investigate v1 — one diagnosis-only delegated investigation.

Phases: validate -> delegate -> report. The Agent Manager workflow owns
execution and structured result validation; this wrapper owns stable caller
identity, explicit subjects, and one collect wait.
"""

import json

try:
    inputs
except NameError:
    inputs = {}

envelope = {
    "program": "agent-manager.investigate", "version": "1", "status": "failed", "phase": "validate",
    "inputs": {}, "signals": {"started": False, "collected": False, "execution_id": None, "disposition": "diagnosis_only"}, "errors": [], "evidence": [],
}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running", "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def step_validate():
    if not isinstance(inputs, dict):
        return fail("failed", "invalid_input", "inputs must be an object", "validate")
    required = ("caller_key", "subject_kind", "subject_ref", "question", "run_ids")
    if any(not isinstance(inputs.get(key), str) or not inputs[key].strip() for key in required[:4]):
        return fail("failed", "invalid_input", "caller key, subject identity, and question are required", "validate")
    run_ids = inputs.get("run_ids")
    if not isinstance(run_ids, list) or not 1 <= len(run_ids) <= 8 or any(not isinstance(value, str) or not value.strip() for value in run_ids):
        return fail("failed", "invalid_input", "run_ids must contain one to eight explicit values", "validate")
    wait_seconds = int(inputs.get("wait_seconds", 30))
    if wait_seconds < 1 or wait_seconds > 300:
        return fail("failed", "invalid_input", "wait_seconds must be in [1, 300]", "validate")
    depth = str(inputs.get("depth", "standard"))
    if depth not in ("quick", "standard", "deep"):
        return fail("failed", "invalid_input", "depth must be quick, standard, or deep", "validate")
    evidence_refs = inputs.get("evidence_refs", [])
    if not isinstance(evidence_refs, list) or len(evidence_refs) > 50 or any(not isinstance(value, dict) for value in evidence_refs):
        return fail("failed", "invalid_input", "evidence_refs must contain at most 50 objects", "validate")
    for index, ref in enumerate(evidence_refs):
        required_ref_fields = ("owner", "kind", "ref", "revision", "schemaVersion")
        if any(not isinstance(ref.get(key), str) or not ref[key].strip() for key in required_ref_fields):
            return fail("failed", "invalid_input", f"evidence_refs[{index}] is missing a typed identity", "validate")
        subject_run_ids = ref.get("subjectRunIds", [])
        if not isinstance(subject_run_ids, list) or len(subject_run_ids) > 50 or any(not isinstance(value, str) or not value.strip() for value in subject_run_ids):
            return fail("failed", "invalid_input", f"evidence_refs[{index}].subjectRunIds is invalid", "validate")
    envelope["inputs"] = {"caller_key": inputs["caller_key"], "subject_kind": inputs["subject_kind"], "subject_ref": inputs["subject_ref"], "question": inputs["question"][:512], "run_ids": list(inputs["run_ids"]), "depth": depth, "wait_seconds": wait_seconds, "evidence_refs": list(evidence_refs)}
    return "delegate"


def step_delegate():
    envelope["phase"] = "delegate"
    resolved = envelope["inputs"]
    request = {
        "owner": "agent-manager",
        "workflow_key": "agent-manager/investigate-typed",
        "input": {
            "context": resolved["question"],
            "depth": resolved["depth"],
            "runIds": resolved["run_ids"],
            "selection": {"subjectKind": resolved["subject_kind"], "subjectRef": resolved["subject_ref"], "callerKey": resolved["caller_key"], "diagnosisOnly": True},
            "evidenceRefs": resolved["evidence_refs"],
        },
        "idempotency_key": "program-investigation/" + resolved["caller_key"],
    }
    try:
        handle = agent.start(**request)
        envelope["signals"]["started"] = True
        row = agent.collect(handle, wait_seconds=resolved["wait_seconds"]).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
        if klass == "binding_error" and any(token in str(exc) for token in ("NOT_FOUND_WORKFLOW", "schema_mismatch", "unknown workflow")):
            klass = "workflow_rejected"
        return fail(status, klass, exc, "delegate")
    if not row:
        return fail("failed", "no_output", "delegated investigation returned no bounded result row", "delegate")
    result = row[0]
    execution_id = result.get("executionId") or result.get("execution_id") or result.get("id")
    result_status = str(result.get("status") or "").upper()
    pending = result_status.endswith("QUEUED") or result_status.endswith("RUNNING") or result_status.endswith("WAITING") or result_status.endswith("COLLECTING") or result_status.endswith("DIAGNOSING")
    successful = result_status.endswith("SUCCEEDED") or result_status.endswith("COMPLETED") or result_status == "OK"
    envelope["signals"].update({"collected": True, "execution_id": execution_id, "result_status": result.get("status"), "result_keys": sorted(result.keys())[:32], "pending": pending, "terminal": not pending, "successful": successful})
    if execution_id:
        envelope["evidence"] = [execution_id, resolved["caller_key"], resolved["subject_kind"] + ":" + resolved["subject_ref"]]
    if pending:
        envelope["status"] = "partial"
    elif successful:
        envelope["status"] = "ok"
    else:
        return fail("failed", "nested_failed", result.get("status") or "nested investigation was not successful", "delegate")
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
    return None


states = {"validate": step_validate, "delegate": step_delegate, "report": step_report}
state = "validate"
while state:
    try:
        envelope["phase"] = state
        state = states[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        try:
            status, klass = classify_transport(exc)
        except (NameError, AttributeError):
            status, klass = "failed", "kernel_runtime"
        state = fail(status, klass, exc, envelope.get("phase") or state)
