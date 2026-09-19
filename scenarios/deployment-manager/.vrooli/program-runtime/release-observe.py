import json
inputs = program.inputs()
review_key = str(inputs.get("review_key", "")).strip(); limit = int(inputs.get("limit", 20))
envelope = {"program": "deployment-manager.release-observe", "version": "1", "status": "failed", "phase": "validate", "inputs": {"review_key": review_key, "limit": limit}, "signals": {}, "errors": [], "evidence": []}; handles = {}
fail = program.fail
def step_validate():
    if not review_key or limit < 1 or limit > 20: return fail("failed", "invalid_input", "review_key required and limit must be 1..20", "validate")
    return "collect"
def step_collect():
    envelope["phase"] = "collect"
    try:
        handles["findings"], handles["evidence"] = gather(lambda: deployment_manager.readiness_reviews.get(review_key=review_key, rows="findings"), lambda: deployment_manager.readiness_reviews.get(review_key=review_key, rows="evidence")); return "classify"
    except Exception as exc:
        status, klass = program.classify(exc); return fail(status, klass, exc, "collect")
def step_classify():
    envelope["phase"] = "classify"; fm = handles["findings"].meta() or {}
    envelope["signals"] = {"review_status": fm.get("status"), "comparison_mode": fm.get("comparisonMode"), "goal_ref": fm.get("goalRef"), "evidence_count": handles["evidence"].count(), "finding_count": handles["findings"].count(), "evidence": handles["evidence"].head(limit), "findings": handles["findings"].head(limit)}
    envelope["evidence"] = [review_key]; envelope["status"] = "ok"; return "report"
def step_report(): envelope["phase"] = "report"; print(json.dumps(envelope, allow_nan=False)); return None
STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}; state = "validate"
program.run(STATES, state)
