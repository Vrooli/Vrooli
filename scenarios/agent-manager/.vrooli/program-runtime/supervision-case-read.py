"""Join one watch decision with bounded optional actions and outcomes reads."""

import json

NAME = "agent-manager.supervision-case-read"
KEYS = ["watch_id"]
inputs = program.inputs()
envelope = {
    "program": NAME,
    "version": "1",
    "status": "ok",
    "phase": "validate",
    "inputs": {},
    "signals": {},
    "errors": [],
    "evidence": [],
}
work = {}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:180], "where": where})
    return "report"


def safe(call):
    try:
        return call(), None
    except Exception as exc:
        return None, exc


def step_validate():
    if (
        not isinstance(inputs, dict)
        or set(inputs) - set(KEYS)
        or any(not isinstance(inputs.get(key), str) or not inputs[key].strip() or len(inputs[key]) > 128 for key in KEYS)
    ):
        return fail("failed", "invalid_input", "Supply the declared bounded nonempty owner ID", "validate")
    envelope["inputs"] = dict(inputs)
    return "collect"


def step_collect():
    results = gather(
        lambda: safe(lambda: agent_manager.watch.inspect(watch_id=inputs["watch_id"], event_limit=16)),
        lambda: safe(lambda: agent_manager.watch.actions(watch_id=inputs["watch_id"], limit=8)),
        lambda: safe(lambda: agent_manager.watch.policy_outcomes(watch_id=inputs["watch_id"], limit=16)),
    )
    for name, (handle, error) in zip(("inspection", "actions", "outcomes"), results):
        work[name] = None if error is not None else handle
        if error is None:
            continue
        status, klass = program.classify(error)
        envelope["errors"].append({"class": klass, "detail": f"{name}: {str(error)[:150]}", "where": "collect"})
        if name == "inspection":
            envelope["status"] = status
            return "report"
        envelope["status"] = "partial"
    return "classify"


def step_classify():
    meta = work["inspection"].meta()
    watch = meta.get("watch") or {}
    if watch.get("watchId") != inputs["watch_id"]:
        return fail("failed", "identity_mismatch", "watch response mismatch", "classify")

    decision = watch.get("lastDecision") or {}
    actions_available = work["actions"] is not None
    outcomes_available = work["outcomes"] is not None
    actions = work["actions"].head(8) if actions_available else []
    outcomes = work["outcomes"].head(16) if outcomes_available else []
    signals = {
        "coverage": work["outcomes"].meta().get("coverage") if outcomes_available else "unavailable",
        "watch_status": watch.get("status"),
        "revision": watch.get("revision"),
        "policy_version": (watch.get("spec") or {}).get("policyVersion"),
        "decision_id": decision.get("decisionId"),
        "disposition": decision.get("disposition"),
        "classification": decision.get("classification"),
        "cursor_reset": meta.get("cursorResetRequired", False),
        "pending_event_count": work["inspection"].count(),
        "actions_available": actions_available,
        "actions": [
            {"id": row.get("actionId"), "state": row.get("state"), "decision_id": row.get("decisionId"), "target": row.get("targetRunId")}
            for row in actions
        ],
        "outcomes_available": outcomes_available,
        "assessed_sample": sum(bool(row.get("observedClass")) for row in outcomes) if outcomes_available else None,
        "unassessed_sample": sum(not row.get("observedClass") for row in outcomes) if outcomes_available else None,
        "sample_may_be_truncated": (
            (len(outcomes) == 16 if outcomes_available else False) or (len(actions) == 8 if actions_available else False)
            if outcomes_available or actions_available
            else None
        ),
        "history_scope": "latest decision and bounded owner records",
    }
    child_states = meta.get("subjectStates") or []
    signals["current_children"] = [
        {key: row.get(key) for key in ("runId", "status", "terminal", "frictionUnavailable")}
        for row in child_states[:8]
    ]
    signals["child_states_truncated"] = len(child_states) > 8
    signals["child_state_unavailable"] = meta.get("subjectStateUnavailable")
    signals["next_action"] = "reconcile_cursor" if signals["cursor_reset"] else "inspect_outcome_evidence"
    envelope["signals"] = signals
    envelope["evidence"] = [inputs["watch_id"]] + ([decision["decisionId"]] if decision.get("decisionId") else [])
    return "report"


def step_report():
    envelope["phase"] = "report"
    encoded = json.dumps(envelope, sort_keys=True, separators=(",", ":"))
    if len(encoded.encode()) > 4096:
        envelope["status"] = "failed"
        envelope["signals"] = {}
        envelope["errors"] = [{"class": "output_budget_exhausted", "detail": "owner summary exceeds bound", "where": "report"}]
        encoded = json.dumps(envelope, separators=(",", ":"))
    print(encoded)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        envelope["phase"] = state
        state = STATES[state]()
    except Exception as exc:
        if state == "report":
            raise
        try:
            status, klass = program.classify(exc)
        except (NameError, AttributeError):
            status, klass = "failed", "kernel_runtime"
        state = fail(status, klass, exc, state)
