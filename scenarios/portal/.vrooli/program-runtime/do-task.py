"""portal.do-task v1 — cross-owner browser export then desktop verification."""
try:
    inputs
except NameError:
    inputs = {}

envelope = {"program": "portal.do-task", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {}, "errors": [], "evidence": []}

def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:160], "where": where})
    return "report"

def classify(exc):
    text = str(exc)
    if "saved flow not found" in text or "device/context mismatch" in text:
        return "failed", "desktop_failed"
    if "unreachable" in text or "connection refused" in text or "scenario_not_running" in text:
        return "unavailable", "scenario_unreachable"
    if "grant" in text or "permission" in text:
        return "refused", "no_grant"
    if "deadline" in text or "timeout" in text:
        return "failed", "deadline_exceeded"
    return "failed", "binding_error"

def step_validate():
    if not isinstance(inputs, dict):
        return fail("failed", "invalid_input", "Object inputs required", "validate")
    required = ("workflow_id", "workflow_version", "device_id", "context_key", "flow_id", "flow_version", "actor")
    for name in required:
        value = inputs.get(name)
        if name.endswith("_version"):
            if type(value) is not int or value < 1:
                return fail("failed", "invalid_input", name + " must be a positive integer", "validate")
        elif not isinstance(value, str) or not value.strip() or len(value) > 512:
            return fail("failed", "invalid_input", name + " is required", "validate")
    envelope["inputs"] = {name: ("<redacted>" if name == "actor" else inputs[name]) for name in required}
    return "act"

def step_act():
    envelope["phase"] = "act"
    try:
        browser_handle = browser_automation_studio.workflows.execute(
            workflow_id=inputs["workflow_id"], workflow_version=inputs["workflow_version"], wait_for_completion=True
        )
        browser = browser_handle.head(1)
        browser_meta = browser_handle.meta()
        if not browser and browser_meta:
            browser = [browser_meta]
        if len(browser) != 1:
            return fail("failed", "browser_failed", "browser owner returned no verified result", "act:browser")
        browser_result = browser[0]
        browser_status = str(browser_result.get("status", browser_result.get("disposition", ""))).lower()
        if browser_status not in ("ok", "passed", "complete", "completed", "success", "execution_status_succeeded", "execution_status_completed", "succeeded"):
            return fail("failed", "browser_failed", "browser owner did not verify export", "act:browser")
        envelope["evidence"].append("browser:" + str(browser_result.get("executionId", browser_result.get("id", "verified"))))
        desktop_handle = device_control.flow.replay(
            id=inputs["flow_id"], version=inputs["flow_version"], device_id=inputs["device_id"], context_key=inputs["context_key"], actor=inputs["actor"], rows="chapters"
        )
        desktop = desktop_handle.head(1)
        desktop_meta = desktop_handle.meta()
        if not desktop and desktop_meta:
            desktop = [desktop_meta]
        if len(desktop) != 1:
            return fail("failed", "desktop_failed", "desktop owner returned no verification result", "act:desktop")
        desktop_result = dict(desktop_meta) if isinstance(desktop_meta, dict) else {}
        desktop_result.update(desktop[0])
        desktop_status = str(desktop_result.get("status", desktop_result.get("disposition", ""))).lower()
        if desktop_status not in ("ok", "passed", "complete", "completed", "success"):
            return fail("failed", "desktop_failed", "desktop owner did not verify artifact", "act:desktop")
        envelope["evidence"].append("desktop:" + str(desktop_result.get("runId", desktop_result.get("id", "verified"))))
        envelope["signals"] = {"browser": {"status": browser_status}, "desktop": {"status": desktop_status}}
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = classify(exc)
        return fail(status, klass, str(exc), "act")
    return "report"

def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None

STEPS = {"validate": step_validate, "act": step_act, "report": step_report}
state = "validate"
while state:
    try:
        state = STEPS[state]()
    except Exception as exc:
        if state == "report":
            raise
        state = fail("failed", "kernel_runtime", str(exc)[:160], state)
