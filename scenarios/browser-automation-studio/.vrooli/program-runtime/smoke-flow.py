"""browser-automation-studio.smoke-flow v2 — execute one persisted workflow and classify the outcome from its timeline.

Contract: smoke-flow.json (inputs, invariants, bindings, outputs).
Skill:    browser-automation-studio (usage) — the [S3] leaf "workflow exists, run it".

Phases: validate -> collect -> act -> classify -> report.
collect reads workflows/get. act holds the one write (workflows/execute, wait=True) and nothing else.
classify reads executions/get, executions/timeline (rows="entries") and executions/screenshots: read bindings,
which the classify phase may call (program-contracts.md: classify may not act; a read is not an act).
The failure class comes from the first failed timeline entry (aggregates.status == STEP_STATUS_FAILED, or
context.error present): its action type and context.errorCode. The execution's error prose is not parsed.
`parameters` is passed as an object (proto field `parameters`, message ExecutionParameters): the CLI flag
--parameters-file is CLI-local. No retry. Memory: a verified run notes a preference for the exact revision, a structural failure notes an avoid. The envelope is printed exactly once, on every path.
"""

import json

inputs = program.inputs()
workflow_id = str(inputs.get("workflow_id", "") or "")
parameters = inputs.get("parameters")  # ExecutionParameters object; --parameters-file is a CLI-local json_file alias
version = inputs.get("version")
learn.task(scope="bas-usage", operation="browser-automation-studio.smoke-flow", key={"workflow_id": workflow_id, "version": version})

envelope = {
    "program": "browser-automation-studio.smoke-flow", "version": "3",
    "status": "failed", "phase": "validate",
    "inputs": {"workflow_id": workflow_id, "parameters": bool(parameters), "version": version},
    "signals": {"workflow_name": None, "execution_id": None, "execution_status": None,
                "outcome": "unknown", "failed_step": None, "failed_action": None, "failed_error_code": None,
                "timeline_entries": None, "screenshot_count": None},
    "errors": [], "evidence": [],
}
handles = {}
ELEMENT_ACTIONS = {"ACTION_TYPE_CLICK", "ACTION_TYPE_INPUT", "ACTION_TYPE_HOVER", "ACTION_TYPE_SELECT",
                   "ACTION_TYPE_FOCUS", "ACTION_TYPE_BLUR", "ACTION_TYPE_ASSERT", "ACTION_TYPE_EXTRACT",
                   "ACTION_TYPE_UPLOAD_FILE", "ACTION_TYPE_KEYBOARD", "ACTION_TYPE_SCROLL"}


fail = program.fail


def entry_failed(entry):
    """A timeline entry is the failed one when its aggregate status says so or its context carries an error."""
    agg = entry.get("aggregates") or {}
    ctx = entry.get("context") or {}
    return agg.get("status") == "STEP_STATUS_FAILED" or bool(ctx.get("error"))


def classify_entry(entry):
    """Deterministic table over the failed entry's structure: the action type and the driver's error code.
    A timeout on an element-targeting action is a selector failure; a timeout on a wait or navigate is a timeout."""
    action = str((entry.get("action") or {}).get("type") or "")
    code = str((entry.get("context") or {}).get("errorCode") or "").upper()
    if action in ELEMENT_ACTIONS and code in ("ELEMENT_NOT_FOUND", "NOT_FOUND", "NO_BOUNDING_BOX"):
        return "selector_not_found"
    if code in ("UNAUTHORIZED", "FORBIDDEN", "AUTH_REQUIRED"):
        return "auth_required"
    if "TIMEOUT" in code:
        return "selector_not_found" if action in ELEMENT_ACTIONS else "timeout"
    if action == "ACTION_TYPE_WAIT":
        return "timeout"
    return "step_failed"


# ---- phases ----------------------------------------------------------------
def step_validate():
    if type(version) != int or version < 1:
        return fail("failed", "invalid_input", "Exact positive workflow version required", "validate")
    if not workflow_id:
        return fail("failed", "invalid_input", "workflow_id is required", "validate")
    if parameters is not None and not isinstance(parameters, dict):
        return fail("failed", "invalid_input", "parameters must be an ExecutionParameters object, not a file path", "validate")
    extraction = inputs.get("extraction", [])
    if not isinstance(extraction, list) or len(extraction) > 16 or any(not isinstance(e, dict) or not e.get("name") or not e.get("selector") or type(e.get("limit", 10)) is not int or not 0 <= e.get("limit", 10) <= 100 for e in extraction):
        return fail("failed", "invalid_input", "Invalid extraction contract", "validate")
    return "collect"


def step_collect():  # COLLECT · confirm the workflow exists before spending a run
    envelope["phase"] = "collect"
    eligibility = learn.result_status(artifact={"kind": "workflow", "owner": "browser-automation-studio", "id": workflow_id, "revision": str(version)})
    if not eligibility.get("available") or not eligibility.get("eligible"):
        return fail("refused", "learning_ineligible", "Workflow revision is contradicted or eligibility unavailable", "collect")
    try:
        rows = browser_automation_studio.workflows.get(workflow_id=workflow_id).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        low = str(exc).lower()
        if klass == "binding_error":
            # A malformed id and an absent workflow are different callers' mistakes and
            # branch differently in the usage skill's tree.
            if "invalid workflow id" in low or "invalid_argument" in low:
                return fail("failed", "invalid_input",
                            f"{workflow_id!r} is not a workflow id; read one from `workflows list`", "collect")
            if "not found" in low or "404" in low:
                return fail("failed", "workflow_not_found", exc, "collect")
        return fail(status, klass, exc, "collect")
    if not rows:
        return fail("failed", "workflow_not_found", f"workflows.get returned no row for {workflow_id}", "collect")
    first = rows[0]
    wfrow = first.get("workflow", first)  # GetWorkflowResponse wraps the workflow; a bare row is accepted too
    envelope["signals"]["workflow_name"] = wfrow.get("name")
    conditions = inputs.get("postconditions", [])
    extraction = inputs.get("extraction", [])
    if conditions or extraction or inputs.get("effect_policy") == "read_only":
        if wfrow.get("version") != version:
            return fail("failed", "version_conflict", "Workflow revision changed before compatibility check", "collect")
        definition = wfrow.get("flowDefinition", wfrow.get("flow_definition", {})) or {}
        actions = [n.get("action", {}) for n in definition.get("nodes", [])]
        if not actions:
            return fail("partial", "verification_required", "Workflow definition unavailable for compatibility check", "collect")
        assertions = [a.get("assert") for a in actions if a.get("type") == "ACTION_TYPE_ASSERT"]
        def enforces(actual, condition):
            modes = {"exists": "ASSERTION_MODE_EXISTS", "text_equals": "ASSERTION_MODE_TEXT_EQUALS", "text_contains": "ASSERTION_MODE_TEXT_CONTAINS"}
            mode = modes.get(condition.get("mode"), condition.get("mode"))
            if actual.get("negated") or actual.get("selector") != condition.get("selector") or actual.get("mode") != mode:
                return False
            if mode == "ASSERTION_MODE_EXISTS":
                return True
            return actual.get("expected") == condition.get("expected") and actual.get("caseSensitive", actual.get("case_sensitive")) is True
        if any(not any(enforces(actual, condition) for actual in assertions) for condition in conditions):
            return fail("refused", "verification_required", "Saved workflow does not enforce the requested postconditions", "collect")
        extractors = [a.get("extract", {}) for a in actions if a.get("type") == "ACTION_TYPE_EXTRACT"]
        for spec in extraction:
            if not any(e.get("extractType", e.get("extract_type", "EXTRACT_TYPE_TEXT")) in (("EXTRACT_TYPE_ATTRIBUTE",) if spec.get("attribute") else ("EXTRACT_TYPE_TEXT", "EXTRACT_TYPE_UNSPECIFIED")) and e.get("selector") == spec.get("selector") and e.get("storeAs", e.get("store_as")) == spec.get("name") and (e.get("attributeName", e.get("attribute_name", "")) or "") == (spec.get("attribute") or "") for e in extractors):
                return fail("refused", "verification_required", "Saved workflow does not implement requested extraction", "collect")
        # Only observational actions qualify. Navigation/click/input can trigger domain effects.
        observational = {"ACTION_TYPE_ASSERT", "ACTION_TYPE_EXTRACT", "ACTION_TYPE_SCREENSHOT", "ACTION_TYPE_WAIT"}
        if inputs.get("effect_policy") == "read_only" and any(a.get("type") not in observational for a in actions):
            return fail("refused", "effect_policy_unavailable", "Workflow includes actions without a read-only guarantee", "collect")
    envelope["evidence"].append(f"workflow:{workflow_id}")
    return "act"


def step_act():  # ACT · exactly one execution with wait; nothing else
    envelope["phase"] = "act"
    kwargs = {"workflow_id": workflow_id, "wait": True}
    if parameters is not None:
        kwargs["parameters"] = parameters
    if version is not None:
        kwargs["version"] = int(version)
    try:
        rows = browser_automation_studio.workflows.execute(**kwargs).head(1)
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "act")
    if not rows:
        return fail("failed", "binding_error", "execute returned no row", "act")
    r = rows[0]
    envelope["signals"]["execution_id"] = r.get("executionId")
    envelope["signals"]["execution_status"] = r.get("status")
    if r.get("executionId"):
        envelope["evidence"].append(f"execution:{r.get('executionId')}")
    return "classify"


def step_classify():  # CLASSIFY · read the execution record and its timeline; label from the failed entry's structure
    envelope["phase"] = "classify"
    execution_id = envelope["signals"]["execution_id"]
    if execution_id:
        try:
            ex = browser_automation_studio.executions.get(execution_id=execution_id).head(1)
            if ex:
                e = ex[0].get("execution", ex[0])
                result = e.get("result") or {}
                extracted = result.get("extractedData", result.get("extracted_data", {})) or {}
                output = {}
                for spec in inputs.get("extraction", []):
                    if spec["name"] not in extracted:
                        return fail("partial", "verification_required", "Execution omitted requested output", "classify")
                    value = extracted[spec["name"]]
                    output[spec["name"]] = (value if isinstance(value, list) else [value])[:max(1, min(100, spec.get("limit") or 10))]
                if len(json.dumps(output).encode()) > 32768:
                    return fail("partial", "verification_required", "Execution output exceeds byte budget", "classify")
                envelope["signals"]["output"] = output
                envelope["signals"]["execution_status"] = e.get("status") or envelope["signals"]["execution_status"]
        except Exception as exc:
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": f"executions.get: {str(exc)[:160]}", "where": "classify"})
        try:
            shots = browser_automation_studio.executions.screenshots(execution_id=execution_id)
            envelope["signals"]["screenshot_count"] = shots.count()
        except Exception as exc:
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": f"executions.screenshots: {str(exc)[:160]}", "where": "classify"})
    st = envelope["signals"]["execution_status"] or ""
    if st == "EXECUTION_STATUS_COMPLETED":
        if execution_id:
            try:
                timeline = browser_automation_studio.executions.timeline(execution_id=execution_id, rows="entries")
                entries = timeline.head(200)
                envelope["signals"]["timeline_entries"] = timeline.count()
                assertions = [e for e in entries if (e.get("action") or {}).get("type") == "ACTION_TYPE_ASSERT"]
                if assertions and timeline.count() == len(entries) and all(
                    (e.get("aggregates") or {}).get("status") == "STEP_STATUS_COMPLETED"
                    for e in entries) and not any(entry_failed(e) for e in entries) and all(
                    (e.get("aggregates") or {}).get("status") == "STEP_STATUS_COMPLETED"
                    and ((e.get("context") or {}).get("assertion") or {}).get("success") is True
                    for e in assertions) and not envelope["errors"]:
                    envelope["signals"]["outcome"] = "verified_success"
                    learn.note("preference", {"option_id": workflow_id + "@" + str(version)})
            except Exception as exc:
                status, klass = program.classify(exc)
                envelope["errors"].append({"class":klass,"detail":"Assertion evidence unavailable","where":"classify"})
        envelope["status"] = "ok" if not envelope["errors"] else "partial"
        return "report"
    if st in ("EXECUTION_STATUS_RUNNING", "EXECUTION_STATUS_PENDING"):
        return fail("partial", "timeout", f"execution {execution_id} is {st} after wait", "classify")
    if st == "EXECUTION_STATUS_CANCELLED":
        return fail("failed", "step_failed", "execution cancelled", "classify")
    if st != "EXECUTION_STATUS_FAILED":
        return fail("partial", "invalid_response", f"Unrecognized execution status {st!r}", "classify")
    envelope["signals"]["outcome"] = "failed"
    if not execution_id:
        return fail("failed", "step_failed", f"execution status {st or 'unknown'} and no execution id", "classify")
    try:
        tl = browser_automation_studio.executions.timeline(execution_id=execution_id, rows="entries")
        entries = tl.head(200)
        envelope["signals"]["timeline_entries"] = tl.count()
    except Exception as exc:
        status, klass = program.classify(exc)
        envelope["errors"].append({"class": klass, "detail": f"executions.timeline: {str(exc)[:160]}", "where": "classify"})
        return fail("failed", "step_failed", f"execution status {st}; timeline unreadable", "classify")
    failed = [(i, e) for i, e in enumerate(entries) if entry_failed(e)][:1]
    if not failed:
        return fail("failed", "step_failed", f"execution status {st}; no failed timeline entry", "classify")
    idx, entry = failed[0]
    ctx = entry.get("context") or {}
    envelope["signals"]["failed_step"] = idx + 1
    envelope["signals"]["failed_action"] = (entry.get("action") or {}).get("type")
    envelope["signals"]["failed_error_code"] = ctx.get("errorCode")
    if entry.get("nodeId"):
        envelope["evidence"].append(f"node:{entry.get('nodeId')}")
    # A revision that failed on a structural fault is excluded from the next recommendation.
    learn.note("avoid", {"option_id": workflow_id + "@" + str(version),
                         "fingerprint": classify_entry(entry) + ":" + str(envelope["signals"]["failed_action"] or "") + ":" + str(ctx.get("errorCode") or "")},
               evidence=envelope["evidence"][:5])
    return fail("failed", classify_entry(entry),
                f"step {idx + 1} {envelope['signals']['failed_action']} {ctx.get('errorCode') or ''}: {str(ctx.get('error') or '')[:120]}", "classify")


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    status = envelope["signals"].get("outcome", "unknown")
    learn.outcome(status if status in ("verified_success", "failed", "unavailable", "unknown") else "unknown",
                  envelope["evidence"] if status == "verified_success" else [],
                  measurements={"reused_workflow": True})
    envelope["signals"]["learning"] = {"feedback_ref": learn.result("workflow", artifact={"kind": "workflow", "owner": "browser-automation-studio", "id": workflow_id, "revision": str(version)})}
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "act": step_act, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
