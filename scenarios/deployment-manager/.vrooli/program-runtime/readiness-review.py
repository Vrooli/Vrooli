"""Prepare one immutable readiness review through its governed binding."""
try:
    inputs
except NameError:
    inputs = {}
keys = ("scenario", "profile_id", "candidate_commit", "artifact_digest", "target", "channel")
resolved = {key: str(inputs.get(key, "")).strip() for key in keys}
resolved["deliverable"] = str(inputs.get("deliverable", "release readiness"))
resolved["trigger"] = str(inputs.get("trigger", "operator requested readiness review"))
envelope = {"program": "deployment-manager.readiness-review", "version": "1", "status": "failed", "phase": "validate", "inputs": resolved, "signals": {}, "errors": [], "evidence": []}

def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"

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
    missing = [key for key in keys if not resolved[key]]
    if missing: return fail("failed", "invalid_input", "missing: " + ",".join(missing), "validate")
    return "act"

def step_act():
    envelope["phase"] = "act"
    try:
        handle = deployment_manager.readiness_reviews.prepare(scenario=resolved["scenario"], profile_id=resolved["profile_id"], candidate_commit=resolved["candidate_commit"], artifact_digest=resolved["artifact_digest"], targets=[resolved["target"]], channel=resolved["channel"], policy_version=2, deliverable=resolved["deliverable"], trigger=resolved["trigger"], rows="findings")
        meta = handle.meta() or {}
        findings = handle.head(20)
        envelope["signals"] = {"review_key": meta.get("reviewKey"), "review_status": meta.get("status"), "comparison_mode": meta.get("comparisonMode"), "goal_ref": meta.get("goalRef"), "unresolved": handle.count(), "findings": findings, "next_actions": (meta.get("nextActions") or [])[:20]}
        envelope["evidence"] = [value for value in (meta.get("reviewKey"), meta.get("goalRef")) if value]
        envelope["status"] = "ok"
        return "report"
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act")

def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None

STATES = {"validate": step_validate, "act": step_act, "report": step_report}
state = "validate"
while state:
    try: state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report": raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
