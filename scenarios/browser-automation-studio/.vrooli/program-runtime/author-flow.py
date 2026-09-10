inputs = program.inputs()
raw_inputs = inputs
inputs = inputs if isinstance(inputs, dict) else {}
learn.task(scope="bas-usage", operation="browser-automation-studio.author-flow", key={"project_id": inputs.get("project_id", ""), "workflow_id": inputs.get("workflow_id", "")})
flow = inputs.get("flow")
project_id = inputs.get("project_id", "")
workflow_id = inputs.get("workflow_id", "")
expected_version = inputs.get("expected_version", 0)
name = str(inputs.get("name", "") or "candidate").strip()
folder = str(inputs.get("folder", "") or "candidates").strip()

envelope = {
    "program": "browser-automation-studio.author-flow", "version": "4",
    "status": "failed", "phase": "validate",
    "inputs": {"flow_nodes": len((flow or {}).get("nodes", [])) if isinstance(flow, dict) else None,
               "name": name, "folder": folder},
    "signals": {"valid": None, "validation_errors": None, "validation_warnings": None, "node_count": None,
                "execution_id": None, "execution_status": None, "outcome": "unknown", "persistable": False, "qualification": "candidate",
                "workflow_id": None, "version": None},
    "errors": [], "evidence": [],
}
handles = {}


fail = program.fail


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
        status, klass = program.classify(exc)
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
            status, klass = program.classify(exc)
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
        status, klass = program.classify(exc)
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
        if r.get("status") == "EXECUTION_STATUS_FAILED":
            envelope["signals"]["outcome"] = "failed"
        return fail("failed", classify_failure(r.get("error")), r.get("error") or f"status {r.get('status')}", "act")
    # Validation requires an assertion, but only execution telemetry proves it ran.
    assertions_verified = False
    try:
        timeline = browser_automation_studio.executions.timeline(execution_id=r["executionId"], rows="entries")
        entries = timeline.head(200)
        assertions = [e for e in entries if (e.get("action") or {}).get("type") == "ACTION_TYPE_ASSERT"]
        assertions_verified = bool(assertions) and timeline.count() == len(entries) and all(
            (e.get("aggregates") or {}).get("status") == "STEP_STATUS_COMPLETED"
            for e in entries) and all(
            (e.get("aggregates") or {}).get("status") == "STEP_STATUS_COMPLETED"
            and ((e.get("context") or {}).get("assertion") or {}).get("success") is True
            for e in assertions) and not any((e.get("context") or {}).get("error")
                or (e.get("aggregates") or {}).get("status") == "STEP_STATUS_FAILED" for e in entries)
    except Exception as exc:
        status, klass = program.classify(exc)
        envelope["errors"].append({"class":klass,"detail":"Assertion evidence unavailable","where":"act:verify"})
    if not assertions_verified or envelope["errors"]:
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "verification_required", "detail": "Candidate retained by caller; no qualified workflow persisted without complete assertion evidence", "where": "act:verify"})
        return "report"
    envelope["signals"]["persistable"] = True
    envelope["signals"]["qualification"] = "qualified"
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
        envelope["status"] = "partial" if envelope["errors"] else "ok"
        if assertions_verified and not envelope["errors"]:
            envelope["signals"]["outcome"] = "verified_success"
            learn.note("preference", {"option_id": saved["id"] + "@" + str(saved["version"])})
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "act:persist")
    return "report"


def step_report():
    envelope["phase"] = "report"
    status = envelope["signals"].get("outcome", "unknown")
    learn.outcome(status if status in ("verified_success", "failed", "unavailable", "unknown") else "unknown",
                  envelope["evidence"] if status == "verified_success" else [],
                  measurements={"reused_workflow": False})
    envelope["signals"]["learning"] = {"feedback_ref": learn.result("workflow", artifact={"kind": "workflow", "owner": "browser-automation-studio", "id": envelope["signals"].get("workflow_id") or "candidate", "revision": str(envelope["signals"].get("version") or 0)})}
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "act": step_act, "report": step_report}
state = "validate"
program.run(STATES, state)
