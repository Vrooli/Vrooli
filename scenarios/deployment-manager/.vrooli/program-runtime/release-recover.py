try:
    inputs
except NameError:
    inputs = {}
review_key = str(inputs.get("review_key", "")).strip(); execute = bool(inputs.get("execute", False)); confirmation = str(inputs.get("confirmation", "")); dry_run_reference = str(inputs.get("dry_run_reference", ""))
envelope = {"program": "deployment-manager.release-recover", "version": "1", "status": "failed", "phase": "validate", "inputs": {"review_key": review_key, "execute": execute, "confirmation": bool(confirmation), "dry_run_reference": dry_run_reference}, "signals": {}, "errors": [], "evidence": []}; handle = None
def fail(status, klass, detail, where): envelope["status"] = status; envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where}); return "report"
def classify_transport(exc):
    if isinstance(exc, (NameError, AttributeError)): raise exc
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running", "no running runtime ports", "connection refused"):
        if needle in text: return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text: return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text: return ("refused", "not_run_eligible")
    if "inference spend" in text: return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text: return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text: return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text: return ("failed", "invalid_input")
    if "deadline" in text: return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")
def step_validate():
    if not review_key: return fail("failed", "invalid_input", "review_key is required", "validate")
    if execute and (not confirmation or not dry_run_reference): return fail("refused", "no_grant", "execute requires confirmation and dry-run evidence", "validate")
    return "collect"
def step_collect():
    global handle; envelope["phase"] = "collect"
    try: handle = deployment_manager.readiness_reviews.get(review_key=review_key, rows="findings"); return "decide"
    except Exception as exc:
        status, klass = classify_transport(exc); return fail(status, klass, exc, "collect")
def step_decide():
    envelope["phase"] = "decide"; meta = handle.meta() or {}; status = meta.get("status"); decision = "observe" if status in ("approved", "promoted") else "repair_blockers"
    envelope["signals"] = {"decision": decision, "review_status": status, "target_identity": review_key, "required_controls": ["exact owner binding", "explicit grant", "confirmation", "dry-run evidence"]}; envelope["evidence"] = [review_key] + ([dry_run_reference] if dry_run_reference else [])
    return "act" if execute else "report"
def step_act(): return fail("failed", "no_governed_binding", "no target-owner recovery mutation is declared; refusing to improvise", "act")
def step_report(): envelope["phase"] = "report"; envelope["status"] = "ok" if not envelope["errors"] else envelope["status"]; print(envelope); return None
STATES = {"validate": step_validate, "collect": step_collect, "decide": step_decide, "act": step_act, "report": step_report}; state = "validate"
while state:
    try: state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report": raise
        envelope["status"] = "failed"; envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state}); state = "report"
