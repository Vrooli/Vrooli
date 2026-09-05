"""browser-automation-studio.navigate-intent v1 — start one bounded vision-navigation and read its status once.

Contract: navigate-intent.json.
Skill:    browser-automation-studio (usage) — the [S3] leaf "no typed workflow; steer by intent".

Phases: validate -> act -> classify -> report.
There is no wait verb on vision-navigation. The program starts the navigation, reads status
exactly once, and reports what that read said: reached, human_pause, budget, in_progress, or failed.
No polling. No retry. The caller re-reads with `browser-automation-studio vision-navigation status <id>`.
"""

try:
    inputs
except NameError:
    inputs = {}
session = str(inputs.get("session", "") or "").strip()
prompt = str(inputs.get("prompt", "") or "").strip()
max_steps = int(inputs.get("max_steps", 10))
navigator = str(inputs.get("navigator", "") or "").strip()
model = str(inputs.get("model", "") or "").strip()  # required: no model slug is defaulted in a caller (ai-gateway guardrail)

envelope = {
    "program": "browser-automation-studio.navigate-intent", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"session": session, "prompt": prompt, "max_steps": max_steps, "navigator": navigator or None, "model": model},
    "signals": {"navigation_id": None, "start_status": None, "status": None, "step_count": None,
                "total_tokens": None, "navigator_type": None, "outcome": None},
    "errors": [], "evidence": [],
}
OUTCOME = {"completed": "reached", "awaiting_human": "human_pause", "max_steps_reached": "budget",
           "loop_detected": "budget", "navigating": "in_progress", "idle": "in_progress",
           "failed": "failed", "aborted": "failed"}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def step_validate():
    if not session:
        return fail("failed", "invalid_input", "session (existing browser session id) is required", "validate")
    if not prompt:
        return fail("failed", "invalid_input", "prompt is required", "validate")
    if not (1 <= max_steps <= 25):
        return fail("failed", "invalid_input", f"max_steps={max_steps} outside 1..25", "validate")
    if not model:
        return fail("failed", "model_required", "model is required: the vision-navigation binding has no default and callers must not hardcode a slug", "validate")
    return "act"


def step_act():  # ACT · one start, one status read
    envelope["phase"] = "act"
    kwargs = {"session": session, "prompt": prompt, "max_steps": max_steps, "model": model}
    if navigator:
        kwargs["navigator"] = navigator
    try:
        rows = browser_automation_studio.vision_navigation.start(**kwargs).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act")
    if not rows:
        return fail("failed", "binding_error", "start returned no row", "act")
    r = rows[0]
    nav_id = r.get("navigationId")
    envelope["signals"]["navigation_id"] = nav_id
    envelope["signals"]["start_status"] = r.get("status")
    envelope["signals"]["navigator_type"] = r.get("navigatorType")
    if not nav_id:
        return fail("failed", "binding_error", "start returned no navigationId", "act")
    envelope["evidence"].append(f"navigation:{nav_id}")
    try:
        srows = browser_automation_studio.vision_navigation.status(navigation_id=nav_id).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
        envelope["errors"].append({"class": klass, "detail": f"status read: {str(exc)[:160]}", "where": "act"})
        return "classify"
    if srows:
        s = srows[0]
        envelope["signals"]["status"] = s.get("status")
        envelope["signals"]["step_count"] = s.get("stepCount")
        envelope["signals"]["total_tokens"] = s.get("totalTokens")
        envelope["signals"]["navigator_type"] = s.get("navigatorType") or envelope["signals"]["navigator_type"]
    return "classify"


def step_classify():  # CLASSIFY · closed map from the driver's status string
    envelope["phase"] = "classify"
    st = envelope["signals"]["status"]
    if st is None:
        envelope["signals"]["outcome"] = "unknown"
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "navigation_pending", "detail": "status read returned no row; re-read by hand", "where": "classify"})
        return "report"
    outcome = OUTCOME.get(str(st), "unknown")
    envelope["signals"]["outcome"] = outcome
    if outcome == "reached":
        envelope["status"] = "ok"
    elif outcome in ("human_pause", "in_progress", "unknown"):
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "navigation_pending", "detail": f"navigation {envelope['signals']['navigation_id']} is {st}; re-read with vision-navigation status", "where": "classify"})
    elif outcome == "budget":
        return fail("failed", "budget_exhausted", f"navigation ended with {st} after {envelope['signals']['step_count']} steps", "classify")
    else:
        return fail("failed", "navigation_failed", f"navigation status {st}", "classify")
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "act": step_act, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:  # the one catch that guarantees an envelope on every path
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
