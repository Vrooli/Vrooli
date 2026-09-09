"""browser-automation-studio.navigate-intent v2 — start or resume one bounded vision-navigation and read its status once.

Contract: navigate-intent.json.
Skill:    browser-automation-studio (usage) — the [S3] leaf "no typed workflow; steer by intent".

Phases: validate -> act -> classify -> report.
Start (session + prompt + model) or resume (navigation_id). The single status read may carry
wait_millis: the vision-navigation status binding blocks server-side until the navigation is terminal
or the wait elapses. There is no client polling loop and no retry. A completed navigation's recorded
steps are rebuilt into a typed V2 candidate flow (nodes[].action.type + edges) that ends in an assertion,
so the caller can author and verify it as a saved workflow.
"""

inputs = program.inputs()
session = str(inputs.get("session", "") or "").strip()
prompt = str(inputs.get("prompt", "") or "").strip()
max_steps = int(inputs.get("max_steps", 10))
navigator = str(inputs.get("navigator", "") or "").strip()
model = str(inputs.get("model", "") or "").strip()  # required to start: no model slug is defaulted in a caller (ai-gateway guardrail)
resume_id = str(inputs.get("navigation_id", "") or "").strip()
wait_millis = inputs.get("wait_millis", 0)
# Identity: the prompt is free text, so the key is the session (or the resumed navigation) plus a
# digest of the normalized prompt; two different prompts on one session stay distinct, one prompt
# retyped with different spacing does not.
_prompt_norm = " ".join(prompt.lower().split())
_prompt_digest = "".join("%02x" % b for b in _prompt_norm.encode("utf-8")[:8]) + str(len(_prompt_norm))
learn.task(scope="bas-usage", operation="browser-automation-studio.navigate-intent",
           key={"session": session or "resume", "navigation": resume_id, "prompt": _prompt_digest})

envelope = {
    "program": "browser-automation-studio.navigate-intent", "version": "2",
    "status": "failed", "phase": "validate",
    "inputs": {"session": session or None, "prompt": prompt[:160], "max_steps": max_steps, "navigator": navigator or None,
               "model": model or None, "navigation_id": resume_id or None, "wait_millis": wait_millis},
    "signals": {"navigation_id": resume_id or None, "start_status": None, "status": None, "step_count": None,
                "total_tokens": None, "navigator_type": None, "outcome": None, "terminal": None, "recorded_steps": 0},
    "errors": [], "evidence": [],
}
OUTCOME = {"completed": "reached", "awaiting_human": "human_pause", "max_steps_reached": "budget",
           "loop_detected": "budget", "navigating": "in_progress", "idle": "in_progress", "started": "in_progress",
           "failed": "failed", "aborted": "failed"}
ACTION_TYPES = {
    "navigate": "ACTION_TYPE_NAVIGATE", "goto": "ACTION_TYPE_NAVIGATE", "open": "ACTION_TYPE_NAVIGATE",
    "click": "ACTION_TYPE_CLICK", "tap": "ACTION_TYPE_CLICK",
    "type": "ACTION_TYPE_INPUT", "input": "ACTION_TYPE_INPUT", "fill": "ACTION_TYPE_INPUT", "enter_text": "ACTION_TYPE_INPUT",
    "assert": "ACTION_TYPE_ASSERT", "verify": "ACTION_TYPE_ASSERT",
}


fail = program.fail


def candidate_flow(steps, final_url=""):
    """Rebuild recorded navigation steps as a V2 flow: nodes carry a typed action, edges chain them,
    and the flow ends in an assertion so author-flow's require_assertion gate has something to verify.
    Steps the executor cannot replay (scroll, screenshots, model reasoning) are dropped, not guessed."""
    nodes = []
    last_url = ""
    for entry in (steps or [])[:64]:
        if not isinstance(entry, dict) or entry.get("success") is False:
            continue
        raw = entry.get("action_type", entry.get("actionType", entry.get("type", "")))
        kind = str(raw or "").strip().lower().removeprefix("action_type_")
        action_type = ACTION_TYPES.get(kind)
        selector = str(entry.get("selector") or "").strip()
        value = entry.get("value")
        url = str(entry.get("url") or "").strip()
        if url:
            last_url = url
        node_id = "step-" + str(len(nodes) + 1)
        if action_type == "ACTION_TYPE_NAVIGATE" and url:
            nodes.append({"id": node_id, "action": {"type": action_type, "navigate": {"url": url}}})
        elif action_type == "ACTION_TYPE_CLICK" and selector:
            nodes.append({"id": node_id, "action": {"type": action_type, "click": {"selector": selector}}})
        elif action_type == "ACTION_TYPE_INPUT" and selector:
            nodes.append({"id": node_id, "action": {"type": action_type, "input": {"selector": selector, "value": str(value if value is not None else "")}}})
        elif action_type == "ACTION_TYPE_ASSERT" and selector:
            nodes.append({"id": node_id, "action": {"type": action_type, "assert": {"selector": selector, "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": 5000}}})
    if not nodes:
        return None
    if nodes[-1]["action"]["type"] != "ACTION_TYPE_ASSERT":
        # The navigator declared the goal reached; the replay proves the page it reached still renders.
        target = last_url or final_url
        assertion = {"selector": "body", "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": 5000}
        nodes.append({"id": "step-" + str(len(nodes) + 1), "action": {"type": "ACTION_TYPE_ASSERT", "assert": assertion}})
        if target:
            nodes[-1]["meta"] = {"reached_url": target[:512]}
    edges = [{"source": nodes[i]["id"], "target": nodes[i + 1]["id"], "type": "WORKFLOW_EDGE_TYPE_SMOOTHSTEP"} for i in range(len(nodes) - 1)]
    return {"nodes": nodes[:64], "edges": edges[:64]}


def step_validate():
    if type(wait_millis) is not int or not 0 <= wait_millis <= 300000:
        return fail("failed", "invalid_input", "wait_millis must be an integer in 0..300000", "validate")
    if resume_id:
        return "act"
    if not session:
        return fail("failed", "invalid_input", "session (existing browser session id) or navigation_id is required", "validate")
    if not prompt:
        return fail("failed", "invalid_input", "prompt is required", "validate")
    if not (1 <= max_steps <= 25):
        return fail("failed", "invalid_input", f"max_steps={max_steps} outside 1..25", "validate")
    if not model:
        return fail("failed", "model_required", "model is required: the vision-navigation binding has no default and callers must not hardcode a slug", "validate")
    return "act"


def step_act():  # ACT · at most one start, exactly one status read
    envelope["phase"] = "act"
    nav_id = resume_id
    if not nav_id:
        kwargs = {"session": session, "prompt": prompt, "max_steps": max_steps, "model": model}
        if navigator:
            kwargs["navigator"] = navigator
        try:
            rows = browser_automation_studio.vision_navigation.start(**kwargs).head(1)
        except Exception as exc:
            status, klass = program.classify(exc)
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
    status_kwargs = {"navigation_id": nav_id}
    if wait_millis:
        status_kwargs["wait_millis"] = wait_millis
    try:
        srows = browser_automation_studio.vision_navigation.status(**status_kwargs).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        low = str(exc).lower()
        if resume_id and ("not found" in low or "404" in low):
            return fail("failed", "session_not_found", f"navigation {nav_id} is unknown", "act")
        envelope["errors"].append({"class": klass, "detail": f"status read: {str(exc)[:160]}", "where": "act"})
        return "classify"
    if srows:
        s = srows[0]
        envelope["signals"]["status"] = s.get("status")
        envelope["signals"]["step_count"] = s.get("stepCount")
        envelope["signals"]["total_tokens"] = s.get("totalTokens")
        envelope["signals"]["terminal"] = s.get("terminal")
        envelope["signals"]["navigator_type"] = s.get("navigatorType") or envelope["signals"]["navigator_type"]
        steps = s.get("steps") or []
        envelope["signals"]["recorded_steps"] = len(steps) if isinstance(steps, list) else 0
        if str(s.get("status")) == "completed":
            flow = candidate_flow(steps)
            if flow:
                envelope["signals"]["candidate_flow"] = flow
    return "classify"


def step_classify():  # CLASSIFY · closed map from the driver's status string
    envelope["phase"] = "classify"
    st = envelope["signals"]["status"]
    if st is None:
        envelope["signals"]["outcome"] = "unknown"
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "navigation_pending", "detail": "status read returned no row; resume with navigation_id", "where": "classify"})
        return "report"
    outcome = OUTCOME.get(str(st), "unknown")
    envelope["signals"]["outcome"] = outcome
    if outcome == "reached":
        envelope["status"] = "ok"
        if not envelope["signals"].get("candidate_flow"):
            envelope["errors"].append({"class": "no_candidates", "detail": "navigation completed but recorded no replayable steps", "where": "classify"})
            envelope["status"] = "partial"
        return "report"
    if outcome in ("human_pause", "in_progress", "unknown"):
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "navigation_pending", "detail": f"navigation {envelope['signals']['navigation_id']} is {st}; resume with navigation_id (and wait_millis)", "where": "classify"})
        return "report"
    if outcome == "budget":
        learn.note("avoid", {"fingerprint": "navigation:" + str(st)}, evidence=envelope["evidence"][:5])
        return fail("failed", "budget_exhausted", f"navigation ended with {st} after {envelope['signals']['step_count']} steps", "classify")
    learn.note("avoid", {"fingerprint": "navigation:" + str(st)}, evidence=envelope["evidence"][:5])
    return fail("failed", "navigation_failed", f"navigation status {st}", "classify")


def step_report():
    envelope["phase"] = "report"
    # Completed navigation is not verification: only author-flow/smoke-flow assertions verify.
    status = "failed" if envelope["signals"].get("outcome") in ("failed", "budget") else "unknown"
    learn.outcome(status, [], measurements={"visual_reasoning_calls": int(envelope["signals"].get("step_count") or 0)} if envelope["signals"].get("step_count") is not None else None)
    print(envelope)
    return None


STATES = {"validate": step_validate, "act": step_act, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
