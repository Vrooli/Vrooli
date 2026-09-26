"""browser-automation-studio.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json.
Skill:    browser-automation-studio-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Rows: pass-rate, flake-rate, selector-failure-rate, p95-execution-duration, step-failure-rate, failed-run-evidence,
external-friction — same order as the skill table. A row whose sensor has no
governed binding is reported unavailable with the reason; it is never computed by hand here.
"""

import json

inputs = program.inputs()
profile = inputs.get("profile", "operations")
window = int(inputs.get("window", 100))
evidence_sample = int(inputs.get("evidence_sample", 5))

envelope = {
    "program": "browser-automation-studio.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": ({"profile": profile} if profile == "rehabilitation" else {
        "window": window, "evidence_sample": evidence_sample, "profile": profile}),
    "signals": {"rows": [], "readable": 0, "unavailable": 0},
    "errors": [], "evidence": [],
}
program.attach(envelope)
handles = {}

# IDs are checked against the canonical qualification contract by refactor_contract.py.
REHABILITATION_ROWS = ['preservation', 'interactive-feedback', 'motion', 'readiness', 'capture', 'passive-fidelity', 'profile-durability', 'known-flow-reliability', 'cancellation-recovery', 'resource-budget', 'soak-stability', 'evidence-completeness', 'desktop-portability', 'agent-usefulness', 'structural-debt', 'workspace-usability', 'adversarial-review']



fail = program.fail


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


def latest_rehabilitation_run(runs, captured_at):
    """Select the exact composite phase's terminal result after this capture."""
    for run in runs:
        planned = run.get("plannedPhases", run.get("planned_phases", []))
        phases = run.get("phases", [])
        completed = run.get("completedAt", run.get("completed_at", ""))
        if (planned == ["rehabilitation-evidence"] and run.get("status") in ("passed", "failed")
                and len(phases) == 1 and phases[0].get("name") == "rehabilitation-evidence"
                and phases[0].get("status") in ("passed", "failed") and captured_at and completed
                and completed >= captured_at):
            return run
    return None


def capability_standing(findings, capability_id):
    """Read one provider-owned capability from the persisted phase presentation."""
    for phase in findings.get("phases", []):
        if phase.get("name") != "rehabilitation-evidence":
            continue
        presentation = phase.get("phasePresentation", phase.get("phase_presentation", {})) or {}
        for capability in presentation.get("capabilities", []):
            if capability.get("id") == capability_id:
                return {
                    "level": capability.get("currentLevel", capability.get("current_level")),
                    "clean": capability.get("clean"),
                    "label": capability.get("currentLevelLabel", capability.get("current_level_label")),
                }
    return None


def capability_reading(standing):
    """Keep missing maturity evidence unknown instead of calling it a product failure."""
    level = standing.get("level")
    label = standing.get("label")
    clean = standing.get("clean")
    if level == "L1" and clean is True:
        return True, False, None
    if level == "L0" or label == "Unavailable":
        return None, True, "provider maturity is unavailable; no applicable owner receipt"
    if level:
        return False, False, f"capability remains at {level}"
    return None, True, "provider maturity standing is incomplete"


def candidate_freshness(response):
    """Require a complete lifecycle verdict before crediting rehabilitation evidence."""
    if not isinstance(response, dict) or response.get("success") is not True:
        return False, "lifecycle freshness response is missing or unsuccessful"
    checks = response.get("checks")
    if not isinstance(checks, list) or not checks:
        return False, "lifecycle freshness response contains no artifact checks"
    # Proto3 JSON omits false scalar values unless the transport asks it to
    # emit defaults. A missing stale field therefore means false; true remains
    # an explicit fail-closed verdict.
    stale_checks = [check for check in checks if not isinstance(check, dict) or check.get("stale") is True]
    if response.get("stale") is True or stale_checks:
        first = stale_checks[0] if stale_checks else {}
        target = first.get("target") or "managed artifact"
        cause = first.get("cause") or "stale or incomplete verdict"
        file = first.get("file")
        detail = f"lifecycle reports {target} stale: {cause}"
        if file:
            detail += f" ({file})"
        return False, detail
    return True, None


def step_validate():
    if profile not in ("operations", "rehabilitation"):
        return fail("failed", "invalid_input", "unknown qualification profile", "validate")
    if profile == "rehabilitation":
        return "collect"
    if not (10 <= window <= 100):
        return fail("failed", "invalid_input", f"window={window} outside 10..100 (executions list caps at 100)", "validate")
    if not (0 <= evidence_sample <= 10):
        return fail("failed", "invalid_input", f"evidence_sample={evidence_sample} outside 0..10", "validate")
    return "collect"


def step_collect():  # COLLECT · one governed read; the evidence sample is read per failed execution
    envelope["phase"] = "collect"
    if profile == "rehabilitation":
        try:
            api_status = vrooli.scenario.status(name="browser-automation-studio").raw() or {}
            runtime = api_status.get("runtime") or {}
            scenario = api_status.get("scenario") or {}
            handles["current_build"] = (
                runtime.get("buildIdentity", runtime.get("build_identity", ""))
                or scenario.get("buildIdentity", scenario.get("build_identity", ""))
            )
            if not handles["current_build"]:
                handles["runtime_error"] = "scenario status omitted managed build identity"
        except Exception as exc:
            _, klass = program.classify(exc)
            handles["current_build"] = ""
            handles["runtime_error"] = klass
        try:
            freshness = vrooli.scenario.freshness(name="browser-automation-studio").raw() or {}
            handles["artifact_freshness"] = freshness
            fresh, reason = candidate_freshness(freshness)
            handles["artifact_fresh"] = fresh
            handles["artifact_freshness_reason"] = reason
            if not fresh:
                handles["freshness_error"] = reason
        except Exception as exc:
            _, klass = program.classify(exc)
            handles["artifact_fresh"] = False
            handles["artifact_freshness"] = {}
            handles["artifact_freshness_reason"] = f"lifecycle freshness unavailable: {klass}"
            handles["freshness_error"] = klass
        if not handles.get("current_build") or not handles.get("artifact_fresh"):
            return "classify"
        try:
            result = performance_health.sweep.workload_get(scenario="browser-automation-studio", workload="capture")
            readings = result.head(1)
            handles["capture"] = readings[0] if readings else {}
        except Exception as exc:
            _, klass = program.classify(exc)
            handles["capture"] = {}
            handles["capture_error"] = klass
        try:
            handles["test_genie_runs"] = test_genie.runs.list(scenario="browser-automation-studio", limit=10).head(10)
        except Exception as exc:
            _, klass = program.classify(exc)
            handles["test_genie_runs"] = []
            handles["test_genie_error"] = klass
        capture = handles.get("capture", {})
        current_build = handles.get("current_build", "")
        if (capture.get("buildIdentity", capture.get("build_identity", "")) == current_build
                and current_build):
            run = latest_rehabilitation_run(handles.get("test_genie_runs", []), capture.get("capturedAt", capture.get("captured_at", "")))
            if run:
                try:
                    findings = test_genie.runs.findings(
                        scenario="browser-automation-studio",
                        run_id=run.get("runId", run.get("run_id", "")),
                    )
                    handles["owner_run"] = run
                    handles["owner_findings"] = findings.head(10)
                except Exception as exc:
                    _, klass = program.classify(exc)
                    handles["owner_findings"] = []
                    handles["findings_error"] = klass
        return "classify"
    try:
        handles["ex"] = browser_automation_studio.executions.list(limit=window)
        handles["ex"].count()
    except Exception as exc:
        status, klass = program.classify(exc)
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
            status, klass = program.classify(exc)
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
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": f"measures.{name}: {str(exc)[:120]}", "where": "collect"})
    return "classify"


def classify_rehabilitation():
    if not handles.get("artifact_fresh") or not handles.get("current_build"):
        reason = (handles.get("artifact_freshness_reason")
                  or handles.get("runtime_error")
                  or "lifecycle freshness is unavailable")
        envelope["signals"]["candidate_freshness"] = {
            "fresh": False,
            "reason": reason,
            "build_identity": handles.get("current_build", ""),
        }
        envelope["evidence"].append("vrooli/scenario/freshness")
        row_reason = ("managed artifacts are stale; see candidate_freshness"
                      if reason.startswith("lifecycle reports")
                      else "lifecycle freshness unavailable; see candidate_freshness")
        for name in REHABILITATION_ROWS:
            row(name, None, "bas-rehabilitation-v1#" + name, None,
                unavailable=True, reason=row_reason)
        envelope["signals"]["required"] = len(REHABILITATION_ROWS)
        envelope["signals"]["unmet"] = len(REHABILITATION_ROWS)
        envelope["signals"]["product_qualified"] = False
        envelope["status"] = "partial"
        return "report"

    capture = handles.get("capture", {})
    applicable = (capture.get("outcome") == "WORKLOAD_OUTCOME_MEASURED"
                  and capture.get("sampleCount") == 100
                  and capture.get("declaredWarmups") == 1
                  and capture.get("budgetMs") == 2000
                  and bool(capture.get("operationId"))
                  and bool(capture.get("receiptSha256"))
                  and capture.get("buildIdentity", capture.get("build_identity", ""))
                  == handles.get("current_build", "")
                  and bool(handles.get("current_build")))
    owner_run = handles.get("owner_run") if applicable else None
    owner_findings = {"phases": handles.get("owner_findings", [])}
    if owner_run and owner_findings["phases"]:
        # These values are shared by the owner-qualified rows. Store them
        # once; duplicating the build digest and run receipt in every row pushed
        # the governed CLI result past its 4 KiB output bound.
        envelope["signals"]["owner_evidence"] = {
            "run_id": owner_run.get("runId", owner_run.get("run_id")),
            "completed_at": owner_run.get("completedAt", owner_run.get("completed_at")),
            "build_identity": handles.get("current_build"),
            "phase_status": next((p.get("status") for p in owner_run.get("phases", [])
                                   if p.get("name") == "rehabilitation-evidence"), None),
            "evidence_tier": owner_run.get("evidenceTier", owner_run.get("evidence_tier")),
            "source": "test-genie/runs/findings",
        }
    for name in REHABILITATION_ROWS:
        if name == "capture" and applicable:
            reading = {"p95_ms": capture.get("p95Ms"), "wall_p95_ms": capture.get("wallP95Ms"),
                       "samples": capture.get("sampleCount"), "operation_id": capture.get("operationId"),
                       "build_identity": capture.get("buildIdentity"), "captured_at": capture.get("capturedAt")}
            row(name, reading, "bas-rehabilitation-v1#capture", capture.get("withinBudget") is True,
                sensor="performance-health sweep workload-get browser-automation-studio capture")
        elif name in ("motion", "passive-fidelity", "profile-durability", "cancellation-recovery", "resource-budget", "evidence-completeness") and applicable and owner_run and owner_findings["phases"]:
            standing = capability_standing(owner_findings, name)
            if standing and standing.get("level"):
                reading = {"current_level": standing.get("level"),
                           "clean": standing.get("clean")}
                in_band, unavailable, reason = capability_reading(standing)
                row(name, reading, "bas-rehabilitation-v1#" + name, in_band,
                    unavailable=unavailable, reason=reason,
                    sensor="test-genie/runs/findings")
            else:
                row(name, None, "bas-rehabilitation-v1#" + name, None, unavailable=True,
                    reason="exact phase findings lack the named capability standing",
                    sensor="test-genie/runs/findings")
        else:
            reason = "pending_telemetry"
            if name == "capture":
                reason = handles.get("capture_error") or handles.get("runtime_error") or capture.get("reason") or "capture evidence missing or build identity is stale"
                if capture.get("operationId") or capture.get("operation_id"):
                    reading = {
                        "operation_id": capture.get("operationId", capture.get("operation_id")),
                        "workload_build_identity": capture.get("buildIdentity", capture.get("build_identity", "")),
                        "live_build_identity": handles.get("current_build", ""),
                    }
                    reason = reason or "capture build identity does not match live BAS"
                else:
                    reading = None
            elif name in ("motion", "passive-fidelity", "profile-durability", "cancellation-recovery", "resource-budget"):
                reason = (handles.get("test_genie_error") or handles.get("findings_error")
                          or handles.get("runtime_error") or "no_matching_current_candidate_phase_receipt")
                reading = None
            else:
                reading = None
            row(name, reading, "bas-rehabilitation-v1#" + name, None, unavailable=True, reason=reason)
    envelope["signals"]["required"] = len(REHABILITATION_ROWS)
    envelope["signals"]["unmet"] = sum(r["in_band"] is not True for r in envelope["signals"]["rows"])
    envelope["signals"]["product_qualified"] = envelope["signals"]["unmet"] == 0
    envelope["signals"]["candidate_freshness"] = {
        "fresh": True,
        "check_count": len((handles.get("artifact_freshness") or {}).get("checks") or []),
        "build_identity": handles.get("current_build"),
    }
    envelope["evidence"].append("vrooli/scenario/freshness")
    envelope["status"] = "partial" if any(handles.get(key) for key in ("capture_error", "test_genie_error", "findings_error", "runtime_error", "freshness_error")) else "ok"
    return "report"


def step_classify():  # CLASSIFY · every reading is count or filter in the kernel
    envelope["phase"] = "classify"
    if profile == "rehabilitation":
        return classify_rehabilitation()
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
        status, klass = program.classify(exc)
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
    print(json.dumps(envelope, separators=(",", ":"), allow_nan=False))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
