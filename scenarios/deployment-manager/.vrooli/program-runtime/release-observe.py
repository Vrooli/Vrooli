try:
    inputs
except NameError:
    inputs = {}
review_key = str(inputs.get("review_key", "")).strip(); limit = int(inputs.get("limit", 20))
envelope = {"program": "deployment-manager.release-observe", "version": "1", "status": "failed", "phase": "validate", "inputs": {"review_key": review_key, "limit": limit}, "signals": {}, "errors": [], "evidence": []}; handles = {}
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
    if not review_key or limit < 1 or limit > 20: return fail("failed", "invalid_input", "review_key required and limit must be 1..20", "validate")
    return "collect"
def step_collect():
    envelope["phase"] = "collect"
    try:
        handles["findings"], handles["evidence"] = gather(lambda: deployment_manager.readiness_reviews.get(review_key=review_key, rows="findings"), lambda: deployment_manager.readiness_reviews.get(review_key=review_key, rows="evidence")); return "classify"
    except Exception as exc:
        status, klass = classify_transport(exc); return fail(status, klass, exc, "collect")
def step_classify():
    envelope["phase"] = "classify"; fm = handles["findings"].meta() or {}
    envelope["signals"] = {"review_status": fm.get("status"), "comparison_mode": fm.get("comparisonMode"), "goal_ref": fm.get("goalRef"), "evidence_count": handles["evidence"].count(), "finding_count": handles["findings"].count(), "evidence": handles["evidence"].head(limit), "findings": handles["findings"].head(limit)}
    envelope["evidence"] = [review_key]; envelope["status"] = "ok"; return "report"
def step_report(): envelope["phase"] = "report"; print(envelope); return None
STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}; state = "validate"
while state:
    try: state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report": raise
        envelope["status"] = "failed"; envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state}); state = "report"
