"""Compose identity resolution and one owner-verified device operation."""
import json

inputs = program.inputs()
identity = learn.task(operation="device-control.do-task", key={"device": inputs.get("device", ""), "context_key": inputs.get("context_key", "")})

envelope = {"program": "device-control.do-task", "version": "1", "status": "failed",
            "phase": "validate", "signals": {"outcome": "unknown"}, "errors": [], "evidence": []}


def child_result(handle):
    rows = handle.head(1)
    if not rows or not isinstance(rows[0], dict):
        raise ValueError("Child program returned no result envelope")
    return rows[0]


try:
    if not isinstance(inputs, dict):
        raise ValueError("Object inputs required")
    for name in ("device", "context_key", "actor"):
        if not isinstance(inputs.get(name), str) or not inputs[name].strip():
            raise ValueError("Required string: " + name)
    if inputs.get("flow") and inputs.get("flow_id"):
        raise ValueError("Select a candidate or a saved flow, not both")
    if inputs.get("flow_id") and (type(inputs.get("version")) is not int or inputs["version"] < 1):
        raise ValueError("Saved flow requires an exact positive revision")
    envelope["phase"] = "collect"
    with learn.step("find") as find_step:
        prepared = child_result(lib.device_control.prepare_task(
            device=inputs["device"], context_key=inputs["context_key"]))
        find_step.outcome((prepared.get("signals") or {}).get("outcome", "unknown"), prepared.get("evidence", []))
    if prepared.get("status") != "ok":
        envelope.update(status=prepared.get("status", "failed"),
                        errors=prepared.get("errors", []), evidence=prepared.get("evidence", []))
    elif not inputs.get("flow_id") and not inputs.get("flow"):
        envelope.update(status="partial", signals={**prepared.get("signals", {}), "outcome": "unknown"},
                        errors=[{"class": "selection_required", "where": "collect",
                                 "detail": "Select a saved flow and revision or supply an assertion-bearing candidate"}])
        flows = envelope["signals"].get("flows", [])
        options = [str(f["id"]) + "@" + str(f["version"]) for f in flows if f.get("id") and f.get("version")]
        if options:
            choice = learn.choose(options, options[0])
            envelope["signals"]["recommended_flow"] = choice["selected_id"]
            envelope["signals"]["learning"] = choice["learning"]
    else:
        device_id = prepared["signals"]["device_id"]
        envelope["phase"] = "act"
        step_name = "replay" if inputs.get("flow_id") else "author"
        with learn.step(step_name) as flow_step:
            if inputs.get("flow_id"):
                child = child_result(lib.device_control.replay_flow(
                    device_id=device_id, context_key=inputs["context_key"], actor=inputs["actor"],
                    flow_id=inputs["flow_id"], version=inputs["version"]))
            else:
                child = child_result(lib.device_control.author_flow(
                    device_id=device_id, context_key=inputs["context_key"], actor=inputs["actor"],
                    flow=inputs["flow"]))
            flow_step.outcome((child.get("signals") or {}).get("outcome", "unknown"), child.get("evidence", []))
        envelope.update(status=child.get("status", "failed"),
                        signals={**child.get("signals", {}), "device_id": device_id},
                        evidence=child.get("evidence", [])[:10], errors=child.get("errors", [])[:5])
        # Child programs establish completed assertions, not command acknowledgement.
        envelope["signals"]["outcome"] = child.get("signals", {}).get("outcome", "unknown")
        envelope["signals"].setdefault("learning", {}).setdefault("measurements", {})["reused_workflow"] = bool(inputs.get("flow_id"))
        if envelope["signals"]["outcome"] == "verified_success":
            flow_id = child.get("signals", {}).get("flow_id", inputs.get("flow_id", ""))
            version = child.get("signals", {}).get("version", inputs.get("version", 0))
            if flow_id and version:
                learn.note("preference", {"option_id": str(flow_id) + "@" + str(version)})
except Exception as exc:
    envelope["errors"].append({"class": "invalid_input" if isinstance(exc, ValueError) else "operation_unavailable",
                               "where": envelope["phase"], "detail": str(exc)[:160]})
    envelope["status"] = "failed" if isinstance(exc, ValueError) else "unavailable"
envelope["phase"] = "report"
envelope["signals"].setdefault("learning", {}).update({"attempt_id": identity.get("attempt_id"), "task_id": identity.get("task_id"),
    "feedback_ref": learn.result("task")})
status = envelope["signals"].get("outcome", "unknown")
earlier = inputs.get("advice_attempt_id", "")
if earlier and status in ("verified_success", "failed") and envelope["evidence"]:
    envelope["signals"]["learning"]["feedback"] = learn.feedback(earlier, "supported" if status == "verified_success" else "contradicted", envelope["evidence"][:20])
learn.outcome(status if status in ("verified_success", "failed", "unavailable", "unknown") else "unknown",
              envelope["evidence"] if status == "verified_success" else [], measurements={"reused_workflow": bool(inputs.get("flow_id"))})
print(json.dumps(envelope))
