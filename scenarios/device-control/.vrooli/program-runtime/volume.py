"""Run one typed, lease-owned semantic volume operation."""
import json

inputs = program.inputs()
learn.task(operation="device-control.volume", key={"device": inputs.get("device", "")})

envelope = {"program": "device-control.volume", "version": "1", "status": "failed", "phase": "validate", "signals": {"outcome": "unknown"}, "errors": [], "evidence": []}


def compact_state(state):
    """Keep only bounded volume evidence; full device state is service-owned."""
    if not isinstance(state, dict):
        return {}
    properties = {}
    for name in ("volume", "muted"):
        value = state.get("properties", {}).get(name)
        if isinstance(value, dict):
            properties[name] = {
                key: (value[key][:240] if isinstance(value[key], str) else value[key])
                for key in ("value", "status", "reason", "source_transport", "state_domain") if key in value
            }
    result = {"properties": properties}
    for key in ("unavailable", "conflicts"):
        if state.get(key):
            result[key] = state[key]
    return result


def compact_result(payload):
    """Return a stable, small receipt that remains valid under runtime output limits."""
    result = {}
    for key in ("status", "operation_id", "device_id", "device_name", "verification_class", "recovery_attempts", "next_action"):
        if key in payload:
            result[key] = payload[key]
    if isinstance(payload.get("plan"), dict):
        plan = payload["plan"]
        result["plan"] = {key: plan[key] for key in (
            "action_transport", "action", "absolute_setpoint", "math_basis", "approximation", "intent") if key in plan}
    result["before"] = compact_state(payload.get("before"))
    result["after"] = compact_state(payload.get("after"))
    return result

try:
    if not isinstance(inputs, dict):
        raise ValueError("object inputs required")
    for name in ("device", "actor"):
        if not isinstance(inputs.get(name), str) or not inputs[name].strip():
            raise ValueError("required string: " + name)
    if not inputs.get("goal") and not inputs.get("operation"):
        raise ValueError("goal or operation is required")
    envelope["phase"] = "act"
    kwargs = {key: value for key, value in inputs.items() if value is not None and key not in ("device", "confirm")}
    kwargs["device"] = inputs["device"]
    # Device-control's semantic volume endpoint is a governed root binding.
    # `lib` is reserved for scenario-owned program contracts; using it here
    # makes the runtime look for a contract named `device` and fails before
    # the endpoint is invoked.
    response = device_control.device.volume(**kwargs, _confirm=bool(inputs.get("confirm", False)))
    payload = response.raw() if hasattr(response, "raw") else response.meta()
    if not isinstance(payload, dict):
        raise ValueError("volume binding returned no result object")
    envelope["status"] = payload.get("status", "failed")
    envelope["signals"].update({"operation_id": payload.get("operationId", payload.get("operation_id", "")), "verification_class": payload.get("verificationClass", payload.get("verification_class", "unavailable")), "result": compact_result(payload)})
    envelope["signals"]["outcome"] = {"verified": "verified_success", "acknowledged": "unknown", "unverified": "unknown", "conflicted": "failed"}.get(envelope["signals"]["verification_class"], "unavailable")
    envelope["evidence"] = payload.get("evidence", [])[:10]
    if envelope["signals"]["outcome"] == "verified_success":
        learn.note("parameter", {"name": "verification_policy", "value": inputs.get("verification_policy", "physical_output")})
except ValueError as exc:
    envelope["errors"].append({"class": "invalid_input", "where": envelope["phase"], "detail": str(exc)[:180]})
except Exception as exc:
    envelope["status"] = "unavailable"
    envelope["errors"].append({"class": "binding_error", "where": envelope["phase"], "detail": str(exc)[:180]})
envelope["phase"] = "report"
status = envelope["signals"].get("outcome", "unknown")
# No external artifact: the task reference is what a later correction names.
envelope["signals"]["learning"] = {"feedback_ref": learn.result("volume")}
learn.outcome(status if status in ("verified_success", "failed", "unavailable", "unknown") else "unknown",
              envelope["evidence"] if status == "verified_success" else [])
print(json.dumps(envelope, allow_nan=False))
