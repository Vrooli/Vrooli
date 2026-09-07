"""Compose identity resolution and one owner-verified device operation."""
import json

try:
    inputs
except NameError:
    inputs = {}

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
    prepared = child_result(lib.device_control.prepare_task(
        device=inputs["device"], context_key=inputs["context_key"]))
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
            choice = child_result(lib.vrooli_memory.choose_option(options=options, default_id=options[0]))
            if choice.get("status") == "ok":
                envelope["signals"]["recommended_flow"] = choice["signals"]["selected_id"]
                envelope["signals"]["learning"] = choice["signals"]["learning"]
    else:
        device_id = prepared["signals"]["device_id"]
        envelope["phase"] = "act"
        if inputs.get("flow_id"):
            child = child_result(lib.device_control.replay_flow(
                device_id=device_id, context_key=inputs["context_key"], actor=inputs["actor"],
                flow_id=inputs["flow_id"], version=inputs["version"]))
        else:
            child = child_result(lib.device_control.author_flow(
                device_id=device_id, context_key=inputs["context_key"], actor=inputs["actor"],
                flow=inputs["flow"]))
        envelope.update(status=child.get("status", "failed"),
                        signals={**child.get("signals", {}), "device_id": device_id},
                        evidence=child.get("evidence", [])[:10], errors=child.get("errors", [])[:5])
        # Child programs establish completed assertions, not command acknowledgement.
        envelope["signals"]["outcome"] = child.get("signals", {}).get("outcome", "unknown")
        envelope["signals"].setdefault("learning", {}).setdefault("measurements", {})["reused_workflow"] = bool(inputs.get("flow_id"))
except Exception as exc:
    envelope["errors"].append({"class": "invalid_input" if isinstance(exc, ValueError) else "operation_unavailable",
                               "where": envelope["phase"], "detail": str(exc)[:160]})
    envelope["status"] = "failed" if isinstance(exc, ValueError) else "unavailable"
envelope["phase"] = "report"
print(json.dumps(envelope))
