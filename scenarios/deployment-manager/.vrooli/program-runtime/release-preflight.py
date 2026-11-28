inputs = program.inputs()
review_key = str(inputs.get("review_key", "")).strip()
envelope = {"program": "deployment-manager.release-preflight", "version": "1", "status": "failed", "phase": "validate", "inputs": {"review_key": review_key}, "signals": {}, "errors": [], "evidence": []}
handles = {}
def step_validate(): return program.fail("failed", "invalid_input", "review_key is required", "validate") if not review_key else "collect"
def step_collect():
    envelope["phase"] = "collect"
    try:
        handles["review"], handles["policy"] = gather(lambda: deployment_manager.readiness_reviews.get(review_key=review_key, rows="findings"), lambda: deployment_manager.readiness_reviews.policy_check())
        return "classify"
    except Exception as exc:
        return program.fail_exception(exc, "collect")
def step_classify():
    envelope["phase"] = "classify"; review = handles["review"]; meta = review.meta() or {}; policy = handles["policy"].head(1); policy_row = policy[0] if policy else {}
    ready = meta.get("status") == "approved" and bool(policy_row.get("matches", False))
    envelope["signals"] = {"ready": ready, "review_status": meta.get("status"), "unresolved": review.count(), "policy_matches": bool(policy_row.get("matches", False)), "findings": review.head(20)}
    if not ready:
        return program.fail("refused", "not_run_eligible", "the review is not currently approved by the canonical policy projection", "classify")
    envelope["evidence"] = [review_key, f"policy:{policy_row.get('policyVersion', 0)}"]
    envelope["status"] = "ok"; return "report"
def step_report(): envelope["phase"] = "report"; program.report(envelope); return None
STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}; state = "validate"
program.run(STATES, state)
