"""browser-automation-studio.smoke-flow v1 — execute one persisted workflow and classify the outcome from its timeline.

Contract: smoke-flow.json (inputs, invariants, bindings, outputs).
Skill:    browser-automation-studio (usage) — the [S3] leaf "workflow exists, run it".

Phases: validate -> collect -> act -> classify -> report.
collect reads workflows/get. act holds the one write (workflows/execute, wait=True) and nothing else.
classify reads executions/get, executions/timeline (rows="entries") and executions/screenshots: read bindings,
which the classify phase may call (program-contracts.md: classify may not act; a read is not an act).
The failure class comes from the first failed timeline entry (aggregates.status == STEP_STATUS_FAILED, or
context.error present): its action type and context.errorCode. The execution's error prose is not parsed.
`parameters` is passed as an object (proto field `parameters`, message ExecutionParameters): the CLI flag
--parameters-file is CLI-local. No retry. No memory. The envelope is printed exactly once, on every path.
"""

try:
    inputs
except NameError:
    inputs = {}
workflow_id = str(inputs.get("workflow_id", "") or "")
parameters = inputs.get("parameters")  # ExecutionParameters object; --parameters-file is a CLI-local json_file alias
version = inputs.get("version")

envelope = {
    "program": "browser-automation-studio.smoke-flow", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"workflow_id": workflow_id, "parameters": bool(parameters), "version": version},
    "signals": {"workflow_name": None, "execution_id": None, "execution_status": None,
                "outcome": None, "failed_step": None, "failed_action": None, "failed_error_code": None,
                "timeline_entries": None, "screenshot_count": None},
    "errors": [], "evidence": [],
}
handles = {}
ELEMENT_ACTIONS = {"ACTION_TYPE_CLICK", "ACTION_TYPE_INPUT", "ACTION_TYPE_HOVER", "ACTION_TYPE_SELECT",
                   "ACTION_TYPE_FOCUS", "ACTION_TYPE_BLUR", "ACTION_TYPE_ASSERT", "ACTION_TYPE_EXTRACT",
                   "ACTION_TYPE_UPLOAD_FILE", "ACTION_TYPE_KEYBOARD", "ACTION_TYPE_SCROLL"}


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
    return "collect"


def step_collect():  # COLLECT · confirm the workflow exists before spending a run
    envelope["phase"] = "collect"
    try:
        rows = browser_automation_studio.workflows.get(workflow_id=workflow_id).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
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
        status, klass = classify_transport(exc)
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
                envelope["signals"]["execution_status"] = e.get("status") or envelope["signals"]["execution_status"]
        except Exception as exc:
            status, klass = classify_transport(exc)
            envelope["errors"].append({"class": klass, "detail": f"executions.get: {str(exc)[:160]}", "where": "classify"})
        try:
            shots = browser_automation_studio.executions.screenshots(execution_id=execution_id)
            envelope["signals"]["screenshot_count"] = shots.count()
        except Exception as exc:
            status, klass = classify_transport(exc)
            envelope["errors"].append({"class": klass, "detail": f"executions.screenshots: {str(exc)[:160]}", "where": "classify"})
    st = envelope["signals"]["execution_status"] or ""
    if st == "EXECUTION_STATUS_COMPLETED":
        envelope["signals"]["outcome"] = "passed"
        envelope["status"] = "ok" if not envelope["errors"] else "partial"
        return "report"
    if st in ("EXECUTION_STATUS_RUNNING", "EXECUTION_STATUS_PENDING"):
        envelope["signals"]["outcome"] = "still_running"
        return fail("partial", "timeout", f"execution {execution_id} is {st} after wait", "classify")
    if st == "EXECUTION_STATUS_CANCELLED":
        envelope["signals"]["outcome"] = "cancelled"
        return fail("failed", "step_failed", "execution cancelled", "classify")
    envelope["signals"]["outcome"] = "failed"
    if not execution_id:
        return fail("failed", "step_failed", f"execution status {st or 'unknown'} and no execution id", "classify")
    try:
        tl = browser_automation_studio.executions.timeline(execution_id=execution_id, rows="entries")
        entries = tl.head(200)
        envelope["signals"]["timeline_entries"] = tl.count()
    except Exception as exc:
        status, klass = classify_transport(exc)
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
    return fail("failed", classify_entry(entry),
                f"step {idx + 1} {envelope['signals']['failed_action']} {ctx.get('errorCode') or ''}: {str(ctx.get('error') or '')[:120]}", "classify")


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "act": step_act, "classify": step_classify, "report": step_report}
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
