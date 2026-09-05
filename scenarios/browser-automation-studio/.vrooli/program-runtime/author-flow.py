try:
    inputs
except NameError:
    inputs = {}
raw_inputs = inputs
inputs = inputs if isinstance(inputs, dict) else {}
flow = inputs.get("flow")
project_id = inputs.get("project_id", "")
workflow_id = inputs.get("workflow_id", "")
expected_version = inputs.get("expected_version", 0)
name = str(inputs.get("name", "") or "candidate").strip()
folder = str(inputs.get("folder", "") or "candidates").strip()

envelope = {
    "program": "browser-automation-studio.author-flow", "version": "2",
    "status": "failed", "phase": "validate",
    "inputs": {"flow_nodes": len((flow or {}).get("nodes", [])) if isinstance(flow, dict) else None,
               "name": name, "folder": folder},
    "signals": {"valid": None, "validation_errors": None, "validation_warnings": None, "node_count": None,
                "execution_id": None, "execution_status": None, "outcome": None, "persistable": False,
                "workflow_id": None, "version": None},
    "errors": [], "evidence": [],
}
handles = {}


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


def classify_failure(text):
    t = (text or "").lower()
    if any(k in t for k in ("waiting for selector", "waiting for locator", "element to be visible",
                            "element not found", "no element", "selector", "locator")):
        return "selector_not_found"
    if any(k in t for k in ("timeout", "timed out", "exceeded")):
        return "timeout"
    if any(k in t for k in ("401", "403", "unauthorized", "forbidden", "sign in", "log in", "login")):
        return "auth_required"
    return "step_failed"


def step_validate():
    if not isinstance(raw_inputs, dict) or not isinstance(project_id,str) or not isinstance(workflow_id,str):
        return fail("failed", "invalid_input", "Expected object inputs and string identities", "validate")
    if not project_id and not workflow_id:
        return fail("failed", "invalid_input", "project_id for promotion or workflow_id for repair is required", "validate")
    if workflow_id and (not isinstance(expected_version,int) or isinstance(expected_version,bool) or expected_version < 1):
        return fail("failed", "invalid_input", "Repair requires exact expected_version", "validate")
    if not isinstance(flow, dict) or not flow.get("nodes"):
        return fail("failed", "invalid_input", "flow must be a V2 flow object with a non-empty nodes list", "validate")
    return "collect"


def step_collect():  # COLLECT · schema validation of the draft (read effect)
    envelope["phase"] = "collect"
    try:
        rows = browser_automation_studio.workflows.validate(workflow=flow, require_assertion=True, baseline_workflow_id=workflow_id, expected_version=expected_version).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    if not rows:
        return fail("failed", "binding_error", "validate returned no row", "collect")
    res = rows[0].get("result", rows[0])
    envelope["signals"]["valid"] = bool(res.get("valid"))
    envelope["signals"]["validation_errors"] = len(res.get("errors") or [])
    envelope["signals"]["validation_warnings"] = len(res.get("warnings") or [])
    envelope["signals"]["node_count"] = (res.get("stats") or {}).get("nodeCount")
    handles["issues"] = [f"{i.get('code')}: {str(i.get('message'))[:80]}" for i in (res.get("errors") or [])[:5]]
    if workflow_id:
        try:
            prior = browser_automation_studio.workflows.get(workflow_id=workflow_id).head(1)
            handles["previous"] = prior[0].get("workflow", prior[0]) if prior else {}
            if handles["previous"].get("id") != workflow_id or handles["previous"].get("version") != expected_version:
                return fail("failed", "version_conflict", "Repair baseline changed", "collect")
        except Exception as exc:
            status, klass = classify_transport(exc)
            return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · valid drafts run; invalid drafts stop here
    envelope["phase"] = "classify"
    if not envelope["signals"]["valid"]:
        return fail("failed", "validation_failed", "; ".join(handles.get("issues") or ["validate returned valid=false"]), "classify")
    return "act"


def step_act():  # ACT · one ad hoc execution with wait
    envelope["phase"] = "act"
    try:
        rows = browser_automation_studio.workflows.execute_adhoc(flow_definition=flow, wait_for_completion=True,
                                                                 metadata={"name": name}).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act")
    if not rows:
        return fail("failed", "binding_error", "execute-adhoc returned no row", "act")
    r = rows[0]
    envelope["signals"]["execution_id"] = r.get("executionId")
    envelope["signals"]["execution_status"] = r.get("status")
    if r.get("executionId"):
        envelope["evidence"].append(f"execution:{r.get('executionId')}")
    if r.get("status") in ("EXECUTION_STATUS_RUNNING", "EXECUTION_STATUS_PENDING"):
        return fail("partial", "execution_pending", "Execution still running; nothing persisted", "act")
    if not r.get("executionId"):
        return fail("failed", "invalid_response", "Execution evidence identity missing", "act")
    if r.get("status") != "EXECUTION_STATUS_COMPLETED":
        envelope["signals"]["outcome"] = "failed"
        return fail("failed", classify_failure(r.get("error")), r.get("error") or f"status {r.get('status')}", "act")
    envelope["signals"]["outcome"] = "passed"
    envelope["signals"]["persistable"] = True
    try:
        if workflow_id:
            old = handles["previous"]
            response = browser_automation_studio.workflows.update(
                workflow_id=workflow_id, expected_version=expected_version, flow_definition=flow,
                name=old.get("name",""), description=old.get("description",""),
                folder_path=old.get("folderPath",""), tags=old.get("tags",[]),
                change_description="Validated repair; execution:" + r["executionId"]).head(1)
        else:
            response = browser_automation_studio.workflows.create(
                project_id=project_id, name=name, folder_path=folder, flow_definition=flow).head(1)
        saved = response[0].get("workflow",response[0]) if response else {}
        if not saved.get("id") or not saved.get("version"):
            return fail("failed", "invalid_response", "Persistence returned no workflow identity", "act:persist")
        envelope["signals"]["workflow_id"] = saved["id"]
        envelope["signals"]["version"] = saved["version"]
        envelope["evidence"].append("workflow:" + saved["id"] + ":" + str(saved["version"]))
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act:persist")
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "act": step_act, "report": step_report}
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
