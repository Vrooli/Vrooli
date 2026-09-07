"""Compose shared learning measurements with Device Control's binding condition."""
import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "device-control.setpoint-read", "version": "2", "status": "ok",
            "phase": "report", "signals": {"rows": []}, "errors": [], "evidence": []}
try:
    if not isinstance(inputs, dict) or set(inputs) - {"from", "to", "operation", "context_key"}:
        raise ValueError("Only explicit comparison selectors are accepted")
    learning = lib.vrooli_memory.compare_outcomes(scope="device-control-usage", **inputs)
    rows = learning.head(1)
    if not rows:
        raise RuntimeError("Shared comparison returned no envelope")
    child = rows[0]
    envelope["signals"]["rows"] = child.get("signals", {}).get("rows", [])
    envelope["errors"] = child.get("errors", [])[:5]
    envelope["evidence"] = child.get("evidence", [])[:10]
    envelope["status"] = child.get("status", "failed")
except Exception as exc:
    envelope["status"] = "failed" if isinstance(exc, ValueError) else "partial"
    envelope["errors"].append({"class": "invalid_input" if isinstance(exc, ValueError) else "comparison_unavailable",
                               "where": "collect", "detail": str(exc)[:160]})

if envelope["status"] != "failed":
    try:
        condition = program_runtime.bindings.condition(scenario="device-control", window_seconds=604800, rows="conditions")
        valid = condition.count() > 0
        unproven = condition.filter(lambda c: c.get("status") != "CONDITION_STATUS_HEALTHY").count()
        envelope["signals"]["rows"].append({"row": "binding-condition", "reading": {"unproven": unproven},
            "target": "used bindings healthy", "in_band": unproven == 0 if valid else None,
            "unavailable": not valid, "reason": None if valid else "unreliable:no_condition"})
    except Exception as exc:
        envelope["status"] = "partial"
        envelope["signals"]["rows"].append({"row": "binding-condition", "reading": None,
            "target": "used bindings healthy", "in_band": None, "unavailable": True, "reason": "binding_unavailable"})
        envelope["errors"].append({"class": "binding_error", "where": "condition", "detail": str(exc)[:160]})
    envelope["signals"]["rows"].append({"row": "external-friction", "reading": None,
        "target": "no recurring failures in a representative window", "in_band": None,
        "unavailable": True, "reason": "read_elsewhere:agent-manager.friction-digest"})
print(json.dumps(envelope))
