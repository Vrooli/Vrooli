try:
    inputs
except NameError:
    inputs = {}
review_key = str(inputs.get("review_key", "")).strip()
envelope = {"program": "deployment-manager.release-preflight", "version": "1", "status": "failed", "phase": "validate", "inputs": {"review_key": review_key}, "signals": {}, "errors": [], "evidence": []}
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
handles = {}
def step_validate(): return fail("failed", "invalid_input", "review_key is required", "validate") if not review_key else "collect"
def step_collect():
    envelope["phase"] = "collect"
    try:
        handles["review"], handles["policy"] = gather(lambda: deployment_manager.readiness_reviews.get(review_key=review_key, rows="findings"), lambda: deployment_manager.readiness_reviews.policy_check())
        return "classify"
    except Exception as exc:
        status, klass = classify_transport(exc); return fail(status, klass, exc, "collect")
def step_classify():
    envelope["phase"] = "classify"; review = handles["review"]; meta = review.meta() or {}; policy = handles["policy"].head(1); policy_row = policy[0] if policy else {}
    envelope["signals"] = {"ready": meta.get("status") == "approved", "review_status": meta.get("status"), "unresolved": review.count(), "policy_matches": bool(policy_row.get("matches", False)), "findings": review.head(20)}
    envelope["evidence"] = [review_key, f"policy:{policy_row.get('policyVersion', 0)}"]
    envelope["status"] = "ok"; return "report"
def step_report(): envelope["phase"] = "report"; print(envelope); return None
STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}; state = "validate"
while state:
    try: state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report": raise
        envelope["status"] = "failed"; envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state}); state = "report"
