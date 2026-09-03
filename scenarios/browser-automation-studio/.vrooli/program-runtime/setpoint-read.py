"""browser-automation-studio.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json.
Skill:    browser-automation-studio-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Rows: pass-rate, flake-rate, selector-failure-rate, p95-step-duration, failed-run-evidence,
external-friction — same order as the skill table. A row whose sensor has no
governed binding is reported unavailable with the reason; it is never computed by hand here.
"""

try:
    inputs
except NameError:
    inputs = {}
window = int(inputs.get("window", 100))
evidence_sample = int(inputs.get("evidence_sample", 5))

envelope = {
    "program": "browser-automation-studio.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"window": window, "evidence_sample": evidence_sample},
    "signals": {"rows": [], "readable": 0, "unavailable": 0},
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


def is_selector_failure(text):
    t = (text or "").lower()
    return any(k in t for k in ("waiting for selector", "waiting for locator", "element to be visible",
                                "element not found", "no element", "selector", "locator"))


def row(name, reading, target, in_band, unavailable=False, reason=None, sensor=None):
    envelope["signals"]["rows"].append({
        "row": name, "reading": reading, "target": target,
        "in_band": None if (in_band is None or unavailable) else bool(in_band),   # canon: null when the row has no band "unavailable": unavailable, "reason": reason,
    })
    if sensor:
        envelope["evidence"].append(sensor)
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1


def step_validate():
    if not (10 <= window <= 100):
        return fail("failed", "invalid_input", f"window={window} outside 10..100 (executions list caps at 100)", "validate")
    if not (0 <= evidence_sample <= 10):
        return fail("failed", "invalid_input", f"evidence_sample={evidence_sample} outside 0..10", "validate")
    return "collect"


def step_collect():  # COLLECT · one governed read; the evidence sample is read per failed execution
    envelope["phase"] = "collect"
    try:
        handles["ex"] = browser_automation_studio.executions.list(limit=window)
        handles["ex"].count()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    # evidence sample: governed reads belong in collect; an outage mid-sample is classified, not counted as unreadable
    failed_h = handles["ex"].filter(lambda r: r.get("status") == "EXECUTION_STATUS_FAILED")
    handles["failed_h"] = failed_h
    handles["sample"] = failed_h.head(evidence_sample) if evidence_sample else []
    handles["with_shots"] = 0
    handles["sample_read"] = 0
    for r in handles["sample"]:
        try:
            if browser_automation_studio.executions.screenshots(execution_id=r.get("executionId")).count() > 0:
                handles["with_shots"] += 1
            handles["sample_read"] += 1
        except Exception as exc:
            status, klass = classify_transport(exc)
            if status == "unavailable":
                return fail(status, klass, exc, "collect")
            envelope["errors"].append({"class": klass, "detail": f"executions.screenshots: {str(exc)[:120]}", "where": "collect"})
    return "classify"


def step_classify():  # CLASSIFY · every reading is count or filter in the kernel
    envelope["phase"] = "classify"
    ex = handles["ex"]
    completed = ex.filter(lambda r: r.get("status") == "EXECUTION_STATUS_COMPLETED").count()
    failed_h = handles["failed_h"]
    failed = failed_h.count()
    terminal = completed + failed
    rate = (completed / terminal) if terminal else None
    row("pass-rate", {"completed": completed, "failed": failed, "window": window}, ">= 0.9", rate is not None and rate >= 0.9,
        unavailable=(terminal == 0), reason=None if terminal else "no terminal executions in window",
        sensor=f"browser-automation-studio executions list --limit {window} (in-kernel count by status)")

    row("flake-rate", None, "<= 0.05", False, unavailable=True,
        reason="no run-group key on executions; the same workflow's runs cannot be grouped into re-runs",
        sensor="executions list (no grouping key)")

    sel = failed_h.filter(lambda r: is_selector_failure(r.get("error"))).count()
    sel_rate = (sel / failed) if failed else None
    row("selector-failure-rate", {"selector_failures": sel, "failed": failed}, "<= 0.2 of failures", sel_rate is not None and sel_rate <= 0.2,
        unavailable=(failed == 0), reason=None if failed else "no failed executions in window",
        sensor="executions list (in-kernel classify of error text)")

    row("p95-step-duration", None, "<= 5000 ms", False, unavailable=True,
        reason="no fleet sensor: uxmetrics workflow-aggregate is per workflow id and returns an untyped Struct",
        sensor="browser-automation-studio uxmetrics workflow-aggregate <workflow-id>")

    sample = handles["sample"]
    with_shots = handles["with_shots"]
    n = handles["sample_read"]
    share = (with_shots / n) if n else None
    row("failed-run-evidence", {"sampled": len(sample), "with_screenshot": with_shots, "read": n}, "1.0", share == 1.0,
        unavailable=(n == 0), reason=None if n else "no failed executions sampled",
        sensor="browser-automation-studio executions screenshots <execution-id> over the most recent failed executions")

    row("external-friction", None, "0 recurring fingerprints", False, unavailable=True,
        reason="agent-manager.friction-digest exists but lib.<name>() accepts no inputs, so the scenario filter cannot be passed in-kernel (W1); run it as its own submission with inputs={\"scenario\": \"browser-automation-studio\"}",
        sensor="run agent-manager.friction-digest scenario=browser-automation-studio window_days=7")

    # Canon (program-contracts.md): a permanent reason does not lower the status. Only a row the
    # owner failed to answer this time, or a read that failed outright, makes the board partial.
    _transient = [r for r in envelope["signals"]["rows"]
                  if r.get("unavailable") and str(r.get("reason") or "").startswith("scenario_unreachable")]
    envelope["status"] = "partial" if (_transient or envelope["errors"]) else "ok"
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
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
