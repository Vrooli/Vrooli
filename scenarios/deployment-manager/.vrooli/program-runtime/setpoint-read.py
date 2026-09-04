try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "deployment-manager.setpoint-read", "version": "1", "status": "failed", "phase": "collect", "inputs": {}, "signals": {"rows": [], "readable": 0, "unavailable": 0}, "errors": [], "evidence": []}; handles = {}
def row(name, reading, target, in_band, unavailable=False, reason=None): envelope["signals"]["rows"].append({"row": name, "reading": None if unavailable else reading, "target": target, "in_band": None if unavailable else in_band, "unavailable": unavailable, "reason": reason}); envelope["signals"]["unavailable" if unavailable else "readable"] += 1
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
def guarded(call):
    def run():
        try: return call()
        except Exception as exc: return exc
    return run
def step_collect():
    results = gather(guarded(lambda: deployment_manager.readiness_reviews.policy_check()), guarded(lambda: deployment_manager.profiles.list(page_size=1)), guarded(lambda: deployment_manager.readiness_reviews.list(page_size=100)), guarded(lambda: deployment_manager.readiness_review_waivers.list(page_size=100)))
    for name, result in zip(("policy", "profiles", "reviews", "waivers"), results):
        if isinstance(result, Exception):
            status, klass = classify_transport(result); envelope["errors"].append({"class": klass, "detail": str(result)[:160], "where": "collect"})
        else: handles[name] = result
    if not handles: envelope["status"] = "unavailable"; return "report"
    return "classify"
def step_classify():
    envelope["phase"] = "classify"
    if "policy" in handles:
        values = handles["policy"].head(1); value = values[0] if values else {}; row("policy-projection", {"version": value.get("policyVersion", 0), "criteria": value.get("criterionCount", 0)}, "matches=true and criteria>=30", bool(value.get("matches")) and int(value.get("criterionCount", 0)) >= 30); envelope["evidence"].append("deployment-manager/readiness-reviews/policy-check")
    else: row("policy-projection", None, "matches=true and criteria>=30", None, True, "scenario_unreachable")
    if "profiles" in handles: row("profile-surface-live", {"profiles_sampled": handles["profiles"].count()}, "reachable", True); envelope["evidence"].append("deployment-manager/profiles/list")
    else: row("profile-surface-live", None, "reachable", None, True, "scenario_unreachable")
    if "reviews" in handles:
        reviews = handles["reviews"]; total = reviews.count(); statuses = dict(reviews.group_by("status")); finished = statuses.get("approved", 0) + statuses.get("promoted", 0)
        row("review-completion-rate", {"complete": finished, "total": total, "rate": (finished / total) if total else None}, ">=0.9", total > 0 and finished / total >= 0.9, total == 0, "unreliable:no readiness reviews" if total == 0 else None)
        comparison = dict(reviews.group_by("comparisonMode")); eligible = total - comparison.get("first_release", 0); comparable = comparison.get("comparable", 0)
        row("predecessor-comparison-coverage", {"comparable": comparable, "eligible": eligible, "rate": (comparable / eligible) if eligible else None}, ">=0.95", eligible > 0 and comparable / eligible >= 0.95, eligible == 0, "unreliable:no predecessor-eligible reviews" if eligible == 0 else None)
        sample = reviews.head(100); evidence = [entry for review in sample for entry in (review.get("evidence") or [])]; required = len(evidence); available = sum(1 for entry in evidence if entry.get("status") in ("passed", "waived", "not_applicable")); stale = sum(1 for entry in evidence if entry.get("status") == "stale")
        row("required-evidence-availability", {"available": available, "observed": required, "rate": (available / required) if required else None}, ">=0.98", required > 0 and available / required >= 0.98, required == 0, "unreliable:no evidence rows" if required == 0 else None)
        row("stale-evidence-rate", {"stale": stale, "observed": required, "rate": (stale / required) if required else None}, "<=0.02", required > 0 and stale / required <= 0.02, required == 0, "unreliable:no evidence rows" if required == 0 else None)
        row("goal-synchronization-lag", None, "<=300s", None, True, "pending_telemetry")
        envelope["evidence"].append("deployment-manager/readiness-reviews/list")
    else:
        for name, target in (("review-completion-rate", ">=0.9"), ("required-evidence-availability", ">=0.98"), ("stale-evidence-rate", "<=0.02"), ("goal-synchronization-lag", "<=300s"), ("predecessor-comparison-coverage", ">=0.95")):
            row(name, None, target, None, True, "scenario_unreachable")
    if "waivers" in handles:
        row("waiver-count", {"count": handles["waivers"].count(), "bounded_at": 100}, "review and reduce", None); envelope["evidence"].append("deployment-manager/readiness-review-waivers/list")
    else: row("waiver-count", None, "review and reduce", None, True, "scenario_unreachable")
    row("program-success-rate", None, ">=0.9", None, True, "read_elsewhere:program-runtime.failure-triage")
    envelope["status"] = "partial" if envelope["errors"] else "ok"; return "report"
def step_report(): envelope["phase"] = "report"; print(envelope); return None
STATES = {"collect": step_collect, "classify": step_classify, "report": step_report}; state = "collect"
while state:
    try: state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report": raise
        envelope["status"] = "failed"; envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state}); state = "report"
