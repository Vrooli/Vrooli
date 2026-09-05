"""browser-automation-studio.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json.
Skill:    browser-automation-studio-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Rows: pass-rate, flake-rate, selector-failure-rate, p95-execution-duration, step-failure-rate, failed-run-evidence,
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
        "in_band": None if (in_band is None or unavailable) else bool(in_band),
        # Canon: unavailable rows are explicit and must carry a reason.
        "unavailable": bool(unavailable),
        "reason": reason,
    })
    if sensor:
        envelope["evidence"].append(sensor)
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1


def one(handle, key, default=None):
    """Read the first scalar record without materializing a result set."""
    rows = handle.head(1)
    return (rows[0] if rows else {}).get(key, default)


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
    # The quality rows are measure-backed. Keep the execution list only for the
    # evidence sample; each measure is independently guarded so one unavailable
    # domain does not hide the other readings.
    measure_window = {"token": "TIME_WINDOW_TOKEN_LAST_7D"}
    for name, call in (
        ("pass_rate", lambda: browser_automation_studio.measures.pass_rate(window=measure_window)),
        ("duration", lambda: browser_automation_studio.measures.p95_duration(window=measure_window)),
        ("step_failure", lambda: browser_automation_studio.measures.step_failure_rate(window=measure_window)),
        ("selector_failure", lambda: browser_automation_studio.measures.selector_failure_rate(window=measure_window)),
    ):
        try:
            handles[name] = call()
            handles[name].head(1)
        except Exception as exc:
            handles[name] = None
            status, klass = classify_transport(exc)
            envelope["errors"].append({"class": klass, "detail": f"measures.{name}: {str(exc)[:120]}", "where": "collect"})
    return "classify"


def step_classify():  # CLASSIFY · every reading is count or filter in the kernel
    envelope["phase"] = "classify"
    ex = handles["ex"]
    completed = ex.filter(lambda r: r.get("status") == "EXECUTION_STATUS_COMPLETED").count()
    failed_h = handles["failed_h"]
    failed = failed_h.count()
    terminal = completed + failed
    rate = (completed / terminal) if terminal else None
    pass_value = one(handles["pass_rate"], "rate") if handles["pass_rate"] is not None else None
    row("pass-rate", {"rate": pass_value, "window": "last_7d", "basis": "executions.pass-rate measure"}, ">= 0.9",
        pass_value is not None and float(pass_value) >= 0.9, unavailable=pass_value is None,
        reason=None if pass_value is not None else "executions.pass-rate measure unavailable", sensor="browser-automation-studio measures pass-rate --window last_7d")

    row("flake-rate", None, "<= 0.05", False, unavailable=True,
        reason="no run-group key on executions; the same workflow's runs cannot be grouped into re-runs",
        sensor="executions list (no grouping key)")

    selector_value = one(handles["selector_failure"], "rate", 0.0) if handles["selector_failure"] is not None else None
    row("selector-failure-rate", {"rate": selector_value, "basis": "telemetry.selector-failure-rate measure"}, "<= 0.2",
        selector_value is not None and float(selector_value) <= 0.2, unavailable=selector_value is None,
        reason=None if selector_value is not None else "telemetry.selector-failure-rate measure unavailable", sensor="browser-automation-studio measures selector-failure-rate --window last_7d")

    duration_value = one(handles["duration"], "durationMs") if handles["duration"] is not None else None
    row("p95-execution-duration", {"duration_ms": duration_value, "basis": "executions.p95-duration measure"}, "<= 5000 ms",
        duration_value is not None and float(duration_value) <= 5000, unavailable=duration_value is None,
        reason=None if duration_value is not None else "executions.p95-duration measure unavailable", sensor="browser-automation-studio measures p95-duration --window last_7d")

    step_value = one(handles["step_failure"], "rate", 0.0) if handles["step_failure"] is not None else None
    row("step-failure-rate", {"rate": step_value, "basis": "execution_metrics.step-failure-rate measure"}, "<= 0.2",
        step_value is not None and float(step_value) <= 0.2, unavailable=step_value is None,
        reason=None if step_value is not None else "execution_metrics.step-failure-rate measure unavailable", sensor="browser-automation-studio measures step-failure-rate --window last_7d")

    sample = handles["sample"]
    with_shots = handles["with_shots"]
    n = handles["sample_read"]
    share = (with_shots / n) if n else None
    row("failed-run-evidence", {"sampled": len(sample), "with_screenshot": with_shots, "read": n}, "1.0", share == 1.0,
        unavailable=(n < 5), reason=None if n >= 5 else "unreliable:fewer_than_five_failed_runs",
        sensor="browser-automation-studio executions screenshots <execution-id> over the most recent failed executions")

    # The friction digest is a declared contract with inputs. Read it through
    # the namespaced lib surface so this board exercises the same in-program
    # path available to an improve skill.
    try:
        friction = lib.agent_manager.friction_digest(
            scenario="browser-automation-studio", window_days=7
        ).head(1)
        digest = friction[0] if friction else {}
        digest_signals = digest.get("signals") or {}
        recurring = digest_signals.get("recurring_count")
        valid = (digest.get("status") == "ok" and recurring is not None
                 and not digest_signals.get("window_truncated_by_run_limit")
                 and not digest_signals.get("episode_reads_failed")
                 and not digest_signals.get("runs_unparseable_timestamp")
                 and not (digest_signals.get("owner_confidence") or {}).get("unknown"))
        row("external-friction", {"recurring_count": recurring}, "0 recurring fingerprints",
            recurring == 0 if valid else None, unavailable=not valid,
            reason=None if valid else "unreliable:incomplete_or_unattributed_window",
            sensor="lib.agent_manager.friction_digest")
    except Exception as exc:
        status, klass = classify_transport(exc)
        row("external-friction", None, "0 recurring fingerprints", False, unavailable=True,
            reason=f"agent-manager.friction-digest unavailable: {str(exc)[:160]}",
            sensor="lib.agent_manager.friction_digest(scenario=browser-automation-studio, window_days=7)")
        envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": "collect:external-friction"})

    # The detailed learning board is a separate bounded read: combining all
    # cohorts here would overflow the runtime's 4 KB result envelope.
    row("learning-effectiveness", None, None, None, unavailable=True,
        reason="read_elsewhere:browser-automation-studio.learning-read",
        sensor="browser-automation-studio.learning-read")

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
