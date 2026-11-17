"""plan-manager.investigate v1 — one bounded plan-to-diagnosis composition."""

import json

inputs = program.inputs()

envelope = {
    "program": "plan-manager.investigate", "version": "1", "status": "failed", "phase": "validate",
    "inputs": {}, "signals": {"disposition": "diagnosis_only", "pending": False, "investigation_id": None}, "errors": [], "evidence": [],
}




def classify(exc):
    text = str(exc)
    if any(token in text for token in ("unreachable", "connection refused", "scenario_not_running")):
        return "unavailable", "scenario_unreachable"
    if "deadline" in text:
        return "failed", "deadline_exceeded"
    return "failed", "binding_error"


def step_validate():
    if not isinstance(inputs, dict):
        return program.fail("failed", "invalid_input", "inputs must be an object", "validate", detail_limit=200)
    required = ("caller_key", "execution_id", "phase_id", "phase_generation", "question")
    if any(not isinstance(inputs.get(key), str) or not inputs[key].strip() for key in required):
        return program.fail("failed", "invalid_input", "caller, execution, phase, generation, and question are required", "validate", detail_limit=200)
    run_ids = inputs.get("run_ids")
    if not isinstance(run_ids, list) or not 1 <= len(run_ids) <= 8 or any(not isinstance(value, str) or not value.strip() for value in run_ids):
        return program.fail("failed", "invalid_input", "run_ids must contain one to eight explicit values", "validate", detail_limit=200)
    wait_seconds = int(inputs.get("wait_seconds", 30))
    if wait_seconds < 1 or wait_seconds > 300:
        return program.fail("failed", "invalid_input", "wait_seconds must be in [1, 300]", "validate", detail_limit=200)
    envelope["inputs"] = {key: inputs[key] for key in required}
    envelope["inputs"]["run_ids"] = list(run_ids)
    envelope["inputs"]["brief_ref"] = str(inputs.get("brief_ref", ""))[:256]
    envelope["inputs"]["wait_seconds"] = wait_seconds
    return "delegate"


def step_delegate():
    envelope["phase"] = "delegate"
    value = envelope["inputs"]
    context = {
        "executionId": value["execution_id"], "phaseId": value["phase_id"], "phaseGeneration": value["phase_generation"],
        "briefRef": value["brief_ref"], "question": value["question"],
    }
    evidence_refs = []
    if value["brief_ref"]:
        evidence_refs.append({
            "owner": "plan-manager", "kind": "investigation-brief", "ref": value["brief_ref"],
            "revision": value["phase_generation"], "schemaVersion": "plan-investigation-brief/v1",
        })
    try:
        result = lib.agent_manager.investigate(
            caller_key=value["caller_key"], subject_kind="plan-execution", subject_ref=value["execution_id"],
            question=json.dumps(context, sort_keys=True, separators=(",", ":")), run_ids=value["run_ids"], wait_seconds=value["wait_seconds"],
            evidence_refs=evidence_refs,
        ).head(1)
    except Exception as exc:
        status, klass = classify(exc)
        return program.fail(status, klass, klass, "delegate", detail_limit=200)
    if not result:
        return program.fail("failed", "no_output", "nested investigation returned no envelope", "delegate", detail_limit=200)
    child = result[0]
    child_signals = child.get("signals") or {}
    pending = bool(child_signals.get("pending")) or child.get("status") == "partial"
    envelope["signals"].update({"nested_status": child.get("status"), "nested_signals": child_signals, "pending": pending})
    envelope["evidence"] = [value["caller_key"], value["execution_id"], value["phase_id"], value["phase_generation"]] + (child.get("evidence") or [])[:4]
    envelope["status"] = child.get("status", "failed")
    envelope["errors"].extend((child.get("errors") or [])[:4])
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
        status, klass = classify(exc)
        state = program.fail(status, klass, klass, envelope.get("phase") or state, detail_limit=200)
