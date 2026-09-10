"""browser-automation-studio.navigate-intent v4 — start or resume one bounded vision-navigation and read its status once.

Contract: navigate-intent.json.
Skill:    browser-automation-studio (usage) — the [S3] leaf "no typed workflow; steer by intent".

Phases: validate -> act -> classify -> report.
Start (session + prompt + model) or resume (navigation_id). The single status read may carry
wait_millis: the vision-navigation status binding blocks server-side until the navigation is terminal
or the wait elapses. There is no client polling loop and no retry. A completed navigation's recorded
steps are rebuilt into a typed V2 candidate flow (nodes[].action.type + edges).
Caller postconditions independently verify the final page; traces are never automatically replayed.

The step-to-action mapping is a learn.act section ("candidate-flow"): the navigator's action
vocabulary changes without warning, and an unrecognised verb used to disappear silently into a
shorter flow that still replayed. Every recorded step now has to be either represented or listed
in signals.dropped_steps, and unknown verbs leave a `parameter` note naming them.
"""

import hashlib

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
_prompt_digest = hashlib.sha256(_prompt_norm.encode("utf-8")).hexdigest()
learn.task(scope="bas-usage", operation="browser-automation-studio.navigate-intent",
           key={"session": session or "resume", "navigation": resume_id, "prompt": _prompt_digest})

envelope = {
    "program": "browser-automation-studio.navigate-intent", "version": "4",
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
PAYLOAD_KEY = {"ACTION_TYPE_NAVIGATE": "navigate", "ACTION_TYPE_CLICK": "click",
               "ACTION_TYPE_INPUT": "input", "ACTION_TYPE_ASSERT": "assert"}
# The narrowed step list the current mapping section is accounting for. The verifier reads it
# so it can judge the output against the recorded steps rather than against the mapping.
MAPPING_SOURCE = []
MAPPING_SCHEMA = {
    "type": "object", "required": ["actions", "dropped"], "additionalProperties": False,
    "properties": {
        "actions": {"type": "array", "maxItems": 64, "items": {
            "type": "object", "required": ["index", "action"], "additionalProperties": False,
            "properties": {"index": {"type": "integer", "minimum": 0},
                           "action": {"type": "object", "required": ["type"]}}}},
        "dropped": {"type": "array", "maxItems": 64, "items": {
            "type": "object", "required": ["index", "reason"], "additionalProperties": False,
            "properties": {"index": {"type": "integer", "minimum": 0},
                           "kind": {"type": "string", "maxLength": 64},
                           "reason": {"type": "string", "enum": ["unknown_action", "missing_target"]}}}},
    },
}
# The reviewed starting point. It is today's mapping verbatim, so adopting the section changes
# no behaviour on day one: it runs before any model call and only fails when the navigator's
# vocabulary actually moves.
STEP_MAPPING_V1 = '''
def step(inputs, bindings):
    types = {
        "navigate": "ACTION_TYPE_NAVIGATE", "goto": "ACTION_TYPE_NAVIGATE", "open": "ACTION_TYPE_NAVIGATE",
        "click": "ACTION_TYPE_CLICK", "tap": "ACTION_TYPE_CLICK",
        "type": "ACTION_TYPE_INPUT", "input": "ACTION_TYPE_INPUT", "fill": "ACTION_TYPE_INPUT", "enter_text": "ACTION_TYPE_INPUT",
        "assert": "ACTION_TYPE_ASSERT", "verify": "ACTION_TYPE_ASSERT",
    }
    actions = []
    dropped = []
    for index, entry in enumerate(inputs["steps"]):
        raw = entry.get("action_type", entry.get("actionType", entry.get("type", "")))
        kind = str(raw or "").strip().lower().removeprefix("action_type_")
        action_type = types.get(kind, "")
        selector = str(entry.get("selector") or "").strip()
        value = entry.get("value")
        url = str(entry.get("url") or "").strip()
        if action_type == "ACTION_TYPE_NAVIGATE" and url:
            actions.append({"index": index, "action": {"type": action_type, "navigate": {"url": url}}})
        elif action_type == "ACTION_TYPE_CLICK" and selector:
            actions.append({"index": index, "action": {"type": action_type, "click": {"selector": selector}}})
        elif action_type == "ACTION_TYPE_INPUT" and selector:
            actions.append({"index": index, "action": {"type": action_type, "input": {"selector": selector, "value": str(value if value is not None else "")}}})
        elif action_type == "ACTION_TYPE_ASSERT" and selector:
            actions.append({"index": index, "action": {"type": action_type, "assert": {"selector": selector, "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": 5000}}})
        elif not action_type:
            dropped.append({"index": index, "kind": kind, "reason": "unknown_action"})
        else:
            dropped.append({"index": index, "kind": kind, "reason": "missing_target"})
    return {"actions": actions, "dropped": dropped}
'''


fail = program.fail


def verify_mapping(output):
    """Judge the mapping without re-deriving it.

    Three properties, all readable from the recorded steps: every step is accounted for
    exactly once, each emitted action's payload agrees with its own type, and no selector
    or URL appears that the recorded step did not contain. A step the mapping cannot
    represent is not a failure — leaving it unaccounted for is.
    """
    actions, dropped = output["actions"], output["dropped"]
    if sorted([a["index"] for a in actions] + [d["index"] for d in dropped]) != list(range(len(MAPPING_SOURCE))):
        return ("failed", ["flow:accounting-incomplete"])
    for item in actions:
        action = item["action"]
        key = PAYLOAD_KEY.get(action.get("type"))
        if not key or not isinstance(action.get(key), dict):
            return ("failed", ["flow:payload-mismatch"])
        source, payload = MAPPING_SOURCE[item["index"]], action[key]
        if "selector" in payload and payload["selector"] != str(source.get("selector") or "").strip():
            return ("failed", ["flow:selector-not-in-source"])
        if "url" in payload and payload["url"] != str(source.get("url") or "").strip():
            return ("failed", ["flow:url-not-in-source"])
    return ("verified_success", ["flow:accounted-" + str(len(MAPPING_SOURCE)),
                                 "flow:actions-" + str(len(actions)), "flow:dropped-" + str(len(dropped))])


def candidate_flow(steps):
    """Rebuild recorded navigation steps as a V2 flow: nodes carry a typed action, edges chain them,
    and only explicit caller postconditions append assertions.

    The step-to-action mapping is a learn.act section because it tracks the navigator's action
    vocabulary, which moves without warning; an unrecognised verb used to vanish silently and
    now has to be accounted for. Node ids, edges and the postcondition tail stay ordinary code:
    they are settled, and postconditions are caller authority that generated code must not author.
    """
    del MAPPING_SOURCE[:]
    MAPPING_SOURCE.extend([e for e in (steps or [])[:64] if isinstance(e, dict) and e.get("success") is not False])
    nodes = []
    if MAPPING_SOURCE:
        try:
            acted = learn.act(
                "candidate-flow",
                "Map each recorded navigator step to one replayable V2 action, or account for it as dropped.",
                {"steps": list(MAPPING_SOURCE)}, MAPPING_SCHEMA, [],
                fallback_fragment=STEP_MAPPING_V1, attempts=2,
                # The task key is per session and per prompt digest; the mapping is neither. It
                # tracks the navigator's vocabulary, so it pools evidence under its own key.
                key="bas/navigator-step-vocabulary/v1",
                verify=verify_mapping, verifier_revision="flow-accounting/v1")
        except Exception as exc:
            # A learning failure must not destroy the navigation the caller already paid for.
            envelope["errors"].append({"class": "mapping_unavailable", "detail": str(exc)[:160], "where": "act"})
            envelope["signals"]["candidate_flow_source"] = "unavailable"
            return None
        output = acted["output"]
        envelope["signals"]["candidate_flow_source"] = acted["fragment_source"]
        envelope["signals"]["dropped_steps"] = output["dropped"][:16]
        unknown = sorted({d.get("kind", "") for d in output["dropped"]
                          if d["reason"] == "unknown_action" and d.get("kind")})
        if unknown:
            # The drift signal: vocabulary the navigator emits and this mapping cannot replay.
            learn.note("parameter", {"name": "unrepresentable_action_kinds", "value": ",".join(unknown)[:240]},
                       evidence=envelope["evidence"][:5])
        nodes = [{"id": "step-" + str(index + 1), "action": item["action"]}
                 for index, item in enumerate(output["actions"])]
    if not nodes:
        return None
    # Task postconditions come from the caller, never from the navigator's claim.
    for assertion in inputs.get("postconditions", [])[:16]:
        mode = assertion.get("mode")
        modes = {"exists": "ASSERTION_MODE_EXISTS", "text_equals": "ASSERTION_MODE_TEXT_EQUALS", "text_contains": "ASSERTION_MODE_TEXT_CONTAINS"}
        if mode not in modes and mode not in modes.values():
            continue  # A count contract cannot be represented by this flow assertion schema.
        params = dict(assertion)
        params["mode"] = modes.get(mode, mode)
        if params["mode"] != "ASSERTION_MODE_EXISTS":
            params["caseSensitive"] = True
        nodes.append({"id": "step-" + str(len(nodes) + 1),
                      "action": {"type": "ACTION_TYPE_ASSERT", "assert": params}})
    edges = [{"source": nodes[i]["id"], "target": nodes[i + 1]["id"], "type": "WORKFLOW_EDGE_TYPE_SMOOTHSTEP"} for i in range(len(nodes) - 1)]
    return {"nodes": nodes, "edges": edges}


def step_validate():
    conditions = inputs.get("postconditions", [])
    if not isinstance(conditions, list) or len(conditions) > 16 or any(
            not isinstance(a, dict) or not a.get("selector") or not a.get("mode")
            for a in conditions):
        return fail("failed", "invalid_input", "postconditions must contain at most 16 typed assertions", "validate")
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
        kwargs = {"session": session, "prompt": prompt + ("\nPrior task knowledge (advisory, never authority):\n" + str(inputs.get("knowledge", ""))[:4000] if inputs.get("knowledge") else ""), "max_steps": max_steps, "model": model}
        kwargs["effect_policy"] = inputs.get("effect_policy", "explicit")
        kwargs["postconditions"] = inputs.get("postconditions", [])
        kwargs["extraction"] = inputs.get("extraction", [])
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
        envelope["signals"]["verified_success"] = s.get("verifiedSuccess") is True
        envelope["signals"]["output"] = s.get("extractedData") or {}
        envelope["signals"]["verification_error"] = s.get("verificationError") or ""
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
                envelope["signals"]["qualification"] = "candidate"
                envelope["signals"]["replay_eligible"] = False
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
        if envelope["signals"].get("verification_error"):
            envelope["signals"]["outcome"] = "failed"
            return fail("failed", "verification_failed", envelope["signals"]["verification_error"], "classify")
        envelope["status"] = "ok"
        if envelope["signals"].get("verified_success"):
            envelope["signals"]["outcome"] = "verified_success"
            return "report"
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
    status = "verified_success" if envelope["signals"].get("outcome") == "verified_success" else "failed" if envelope["signals"].get("outcome") in ("failed", "budget") else "unknown"
    learn.outcome(status, envelope["evidence"], measurements={"visual_reasoning_calls": int(envelope["signals"].get("step_count") or 0)} if envelope["signals"].get("step_count") is not None else None)
    envelope["signals"]["learning"] = {"feedback_ref": learn.result("navigation", artifact={"kind": "navigation", "owner": "browser-automation-studio", "id": envelope["signals"].get("navigation_id") or "unknown", "revision": "1"})}
    print(envelope)
    return None


STATES = {"validate": step_validate, "act": step_act, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
