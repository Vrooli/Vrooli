"""browser-automation-studio.author-flow v1 — validate a V2 flow object, run it once ad hoc, and report whether it may be persisted.

Contract: author-flow.json.
Skill:    browser-automation-studio (usage) — the [S3] leaf "no workflow fits; author one".

Phases: validate -> collect -> classify -> act -> report.
collect: workflows/validate on the flow object (read). classify: valid or not. act: one execute-adhoc with wait.
The flow arrives as an object (input `flow`): the CLI's --flow-file is CLI-local and has no proto field;
the proto fields are `workflow` (validate) and `flow_definition` (execute-adhoc), both verified live.
Persisting into folder "candidates" needs workflows/create, which has NO governed binding today, and the
`--ai-prompt` path lives only on that command. Both are reported failed/no_governed_binding (a missing binding is
permanent, never `unavailable`); the caller persists by hand:
  browser-automation-studio workflows create --project-id <id> --name <name> --folder-path candidates --flow-file <file>
Success shape: status "partial" with signals.persistable=true and signals.persist_command; "ok" is unreachable until
workflows/create is a governed binding, so the contract's status enum omits it.
Kwargs use the proto field names (workflow, flow_definition, wait_for_completion, metadata): the CLI flag --flow-file is
CLI-local and maps to no field, so flag-name kwargs are not possible for this binding.
An unvalidated or failing draft is never reported as persistable.
"""

try:
    inputs
except NameError:
    inputs = {}
flow = inputs.get("flow")
ai_prompt = str(inputs.get("ai_prompt", "") or "").strip()
name = str(inputs.get("name", "") or "candidate").strip()
folder = str(inputs.get("folder", "") or "candidates").strip()

envelope = {
    "program": "browser-automation-studio.author-flow", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"flow_nodes": len((flow or {}).get("nodes", [])) if isinstance(flow, dict) else None,
               "ai_prompt": ai_prompt or None, "name": name, "folder": folder},
    "signals": {"valid": None, "validation_errors": None, "validation_warnings": None, "node_count": None,
                "execution_id": None, "execution_status": None, "outcome": None, "persistable": False,
                "persist_command": None},
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
    if ai_prompt and not flow:
        return fail("failed", "no_governed_binding",
                    "AI generation lives on `workflows create --ai-prompt`, which has no governed binding; write the flow and pass it as `flow`", "validate")
    if not isinstance(flow, dict) or not flow.get("nodes"):
        return fail("failed", "invalid_input", "flow must be a V2 flow object with a non-empty nodes list", "validate")
    return "collect"


def step_collect():  # COLLECT · schema validation of the draft (read effect)
    envelope["phase"] = "collect"
    try:
        rows = browser_automation_studio.workflows.validate(workflow=flow).head(1)
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
    if r.get("status") != "EXECUTION_STATUS_COMPLETED":
        envelope["signals"]["outcome"] = "failed"
        return fail("failed", classify_failure(r.get("error")), r.get("error") or f"status {r.get('status')}", "act")
    envelope["signals"]["outcome"] = "passed"
    envelope["signals"]["persistable"] = True
    envelope["signals"]["persist_command"] = (
        f"browser-automation-studio workflows create --project-id <project-id> --name '{name}' --folder-path {folder} --flow-file <path to this flow as JSON>")
    envelope["errors"].append({"class": "no_governed_binding",
                               "detail": "workflows/create has no governed binding; persist with the command in signals.persist_command",
                               "where": "act"})
    envelope["status"] = "partial"
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
