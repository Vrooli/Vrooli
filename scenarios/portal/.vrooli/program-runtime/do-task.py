"""portal.do-task v1 — cross-owner browser export then desktop verification."""
import json

inputs = program.inputs()

envelope = program.envelope('portal.do-task', '1', phase="validate", inputs={})


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
        return program.fail("failed", "invalid_input", "Object inputs required", "validate", detail_limit=160)
    required = ("workflow_id", "workflow_version", "device_id", "context_key", "flow_id", "flow_version", "actor")
    for name in required:
        value = inputs.get(name)
        if name.endswith("_version"):
            if type(value) is not int or value < 1:
                return program.fail("failed", "invalid_input", name + " must be a positive integer", "validate", detail_limit=160)
        elif not isinstance(value, str) or not value.strip() or len(value) > 512:
            return program.fail("failed", "invalid_input", name + " is required", "validate", detail_limit=160)
    envelope["inputs"] = {name: ("<redacted>" if name == "actor" else inputs[name]) for name in required}
    return "act"

def step_act():
    envelope["phase"] = "act"
    try:
        browser_handle = browser_automation_studio.workflows.execute(
            workflow_id=inputs["workflow_id"], workflow_version=inputs["workflow_version"], wait_for_completion=False
        )
        browser = browser_handle.head(1)
        if len(browser) != 1:
            return program.fail("failed", "browser_failed", "browser owner returned no execution identity", "act:browser", detail_limit=160)
        browser_result = browser[0]
        execution_id = browser_result.get("executionId", browser_result.get("execution_id"))
        if not isinstance(execution_id, str) or not execution_id.strip():
            return program.fail("failed", "browser_failed", "browser owner returned no execution ID", "act:browser", detail_limit=160)

        # Bind the wait to the exact ID returned by the async admission call.
        # If the bounded observation expires, stop that same owner execution
        # and require its durable terminal receipt before advancing.
        try:
            wait_handle = browser_automation_studio.executions.wait_for_execution(
                execution_id=execution_id, timeout_ms=50000
            )
            waited = wait_handle.head(1)
            if len(waited) != 1:
                raise RuntimeError("browser owner returned no terminal wait receipt")
        except Exception:
            stop_handle = browser_automation_studio.executions.stop(execution_id=execution_id)
            stopped = stop_handle.head(1)
            if len(stopped) != 1 or str(stopped[0].get("status", "")).lower() not in ("stopped", "completed"):
                return program.fail("failed", "browser_failed", "browser execution timeout has no acknowledged stop receipt", "act:browser", detail_limit=160)
            envelope["evidence"].append("browser-stop:" + execution_id)
            execution_handle = browser_automation_studio.executions.get(execution_id=execution_id)
            executions = execution_handle.head(1)
            if len(executions) != 1:
                return program.fail("failed", "browser_failed", "browser stop has no durable execution receipt", "act:browser", detail_limit=160)
        else:
            execution_handle = browser_automation_studio.executions.get(execution_id=execution_id)
            executions = execution_handle.head(1)
            if len(executions) != 1:
                return program.fail("failed", "browser_failed", "browser owner returned no durable execution receipt", "act:browser", detail_limit=160)

        execution = executions[0].get("execution", executions[0])

        receipt_id = execution.get("executionId", execution.get("execution_id"))
        workflow_id = execution.get("workflowId", execution.get("workflow_id"))
        workflow_version = execution.get("workflowVersion", execution.get("workflow_version"))
        status = str(execution.get("status", "")).strip().lower().replace("execution_status_", "")
        if (receipt_id != execution_id or workflow_id != inputs["workflow_id"] or
                type(workflow_version) is not int or workflow_version != inputs["workflow_version"] or
                status != "completed"):
            return program.fail("failed", "browser_failed", "browser execution receipt does not match the requested workflow revision", "act:browser", detail_limit=160)
        browser_status = status
        envelope["evidence"].append("browser:" + execution_id)
        desktop_handle = device_control.flow.replay(
            id=inputs["flow_id"], version=inputs["flow_version"], device_id=inputs["device_id"], context_key=inputs["context_key"], actor=inputs["actor"], rows="chapters"
        )
        desktop = desktop_handle.head(1)
        desktop_meta = desktop_handle.meta()
        if not desktop and desktop_meta:
            desktop = [desktop_meta]
        if len(desktop) != 1:
            return program.fail("failed", "desktop_failed", "desktop owner returned no verification result", "act:desktop", detail_limit=160)
        desktop_result = dict(desktop_meta) if isinstance(desktop_meta, dict) else {}
        desktop_result.update(desktop[0])
        desktop_status = str(desktop_result.get("status", desktop_result.get("disposition", ""))).lower()
        if desktop_status not in ("ok", "passed", "complete", "completed", "success"):
            return program.fail("failed", "desktop_failed", "desktop owner did not verify artifact", "act:desktop", detail_limit=160)
        envelope["evidence"].append("desktop:" + str(desktop_result.get("runId", desktop_result.get("id", "verified"))))
        envelope["signals"] = {"browser": {"status": browser_status}, "desktop": {"status": desktop_status}}
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = classify(exc)
        return program.fail(status, klass, str(exc), "act", detail_limit=160)
    return "report"

def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, separators=(",", ":")))
    return None

STEPS = {"validate": step_validate, "act": step_act, "report": step_report}
state = "validate"
while state:
    try:
        state = STEPS[state]()
    except Exception as exc:
        if state == "report":
            raise
        state = program.fail("failed", "kernel_runtime", str(exc)[:160], state, detail_limit=160)
