"""program-runtime.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json (inputs, invariants, bindings, outputs).
Skill:    program-runtime-improve §2 (the rows), §3 (the sensors).
Canon:    program-contracts.md §"The envelope" (row shape, `reason` vocabulary, permanent rows).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Every governed read is guarded on its own worker, so one dead binding yields one row with
reason `scenario_unreachable` and the other rows survive. A row whose reason is permanent
(`no_governed_binding`, `kernel_invoke_budget`, `read_elsewhere:<program>`, `pending_telemetry`)
never lowers the status; only a failed read makes the board `partial`.
"""

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
governance_window_seconds = int(inputs.get("governance_window_seconds", 604800))
agent_failure_band = float(inputs.get("agent_failure_band", 0.15))

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "program-runtime.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"governance_window_seconds": governance_window_seconds, "agent_failure_band": agent_failure_band},
    "signals": {"rows": [], "readable": 0, "unavailable": 0, "failure_shapes": []},
    "errors": [], "evidence": [],
}
handles = {}
dead = {}  # read name -> (status, class) for a read that raised


def fail(status, klass, detail, where):
    """The one place a bad path is recorded. Sets status, appends the error, routes to report."""
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


def guarded(call):
    """Run one read on its own worker; an exception becomes the result so the other reads survive."""
    def run():
        try:
            return call()
        except Exception as exc:
            return exc
    return run


def row(name, reading, target, in_band, unavailable=False, reason=None, sensor=None):
    """Append one setpoint row in the improve skill's table order and keep the sensor as evidence.

    `target` and `in_band` are None when the row has no band. A row that is declined carries
    reading None and a reason from the closed vocabulary.
    """
    envelope["signals"]["rows"].append({
        "row": name, "reading": None if unavailable else reading, "target": target,
        "in_band": None if (unavailable or target is None) else bool(in_band),
        "unavailable": unavailable, "reason": reason,
    })
    if sensor:
        envelope["evidence"].append(sensor)
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1


def dead_row(name, key, target, sensor):
    """A row whose read raised: reason scenario_unreachable when the scenario did not answer, else unreliable."""
    status, klass = dead[key]
    reason = "scenario_unreachable" if status == "unavailable" else f"unreliable:read failed ({klass})"
    row(name, None, target, None, unavailable=True, reason=reason, sensor=sensor)


CALLS = {
    "gov": lambda: program_runtime.programs.governance_share(window_seconds=governance_window_seconds),
    "act": lambda: program_runtime.bindings.act(),
    "cond": lambda: program_runtime.bindings.condition(scenario="program-runtime", window_seconds=governance_window_seconds, rows="conditions"),
    "deleg": lambda: program_runtime.sessions.delegations(),
    "lib": lambda: program_runtime.library.list(),
    "shapes": lambda: program_runtime.programs.mine(include_operator=False),
    "progs": lambda: program_runtime.programs.list(),
}


# ---- state machine ---------------------------------------------------------------
def step_validate():
    if governance_window_seconds < 60:
        return fail("failed", "invalid_input", f"governance_window_seconds={governance_window_seconds} below 60", "validate")
    if not (0.0 < agent_failure_band <= 1.0):
        return fail("failed", "invalid_input", f"agent_failure_band={agent_failure_band} outside (0, 1]", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only, concurrent, each guarded
    envelope["phase"] = "collect"
    results = gather(*[guarded(call) for call in CALLS.values()])
    for name, result in zip(CALLS, results):
        if isinstance(result, Exception):
            if isinstance(result, (NameError, AttributeError)):
                raise result
            status, klass = classify_transport(result)
            dead[name] = (status, klass)
            envelope["errors"].append({"class": klass, "detail": f"{name}: {str(result)[:160]}", "where": "collect"})
            continue
        handles[name] = result
    if not handles:  # nothing answered: the board is unknown, with the first read's status
        envelope["status"] = next(iter(dead.values()))[0]
        return "report"
    return "classify"


def step_classify():  # CLASSIFY · deterministic; every reading is count/head/group_by/meta
    envelope["phase"] = "classify"
    h = handles

    # Permanent rows (skill §3): declined in-program with the reason that says what the reader does next.
    row("discovery-floor", None, "met >= floor", None, unavailable=True, reason="no_governed_binding",
        sensor="program-runtime discovery eval --suite evals/discovery.primary.json --mode judged --json")
    row("authoring-floor", None, "met >= floor", None, unavailable=True, reason="kernel_invoke_budget",
        sensor="program-runtime authoring eval --json")

    # agent-failure-rate: filter the corpus in the kernel; all-time because the binding has no window.
    target = f"< {agent_failure_band}"
    sensor = "program-runtime programs list (in-kernel filter provenance=PROVENANCE_AGENT)"
    if "progs" in h:
        agent_programs = h["progs"].filter(lambda r: r.get("provenance") == "PROVENANCE_AGENT")
        total = agent_programs.count()
        failed = agent_programs.filter(lambda r: r.get("status") == "PROGRAM_STATUS_FAILED").count()
        rate = (failed / total) if total else None
        row("agent-failure-rate", {"failed": failed, "total": total, "rate": rate, "window": "all-time"}, target,
            rate is not None and rate < agent_failure_band,
            unavailable=(total == 0), reason=None if total else "unreliable:no agent-provenance programs in corpus",
            sensor=sensor)
    else:
        dead_row("agent-failure-rate", "progs", target, sensor)

    # governance-share: the handle's rows are the observed (ungoverned) names; the share is a meta() scalar.
    # protojson omits a double at 0.0, so an absent governedShare on an available response is 0.0.
    target, sensor = "1.0", f"program-runtime programs governance-share --window-seconds {governance_window_seconds}"
    if "gov" in h:
        gm = h["gov"].meta() or {}
        share = float(gm.get("governedShare", 0.0))
        row("governance-share",
            {"governed_share": share, "governed_calls": int(gm.get("governedCalls", 0)),
             "observed_calls": int(gm.get("observedCalls", 0)), "observed_names": h["gov"].count()},
            target, share >= 1.0, sensor=sensor)
    else:
        dead_row("governance-share", "gov", target, sensor)

    # act-coverage: cells by verdict; in band when nothing is merely AUTHORED.
    target, sensor = "0 ACT_VERDICT_AUTHORED", "program-runtime bindings act"
    if "act" in h:
        verdicts = dict(h["act"].group_by("verdict"))
        row("act-coverage", verdicts, target, verdicts.get("ACT_VERDICT_AUTHORED", 0) == 0, sensor=sensor)
    else:
        dead_row("act-coverage", "act", target, sensor)

    # binding-condition: this scenario's own bindings by condition status (CONDITION_STATUS_* from bindings.proto).
    target = "0 CONDITION_STATUS_DEGRADED"
    sensor = f"program-runtime bindings condition --scenario program-runtime --window-seconds {governance_window_seconds}"
    if "cond" in h:
        by_status = dict(h["cond"].group_by("status"))
        row("binding-condition", {"by_status": by_status, "bindings": h["cond"].count()}, target,
            by_status.get("CONDITION_STATUS_DEGRADED", 0) == 0, sensor=sensor)
    else:
        dead_row("binding-condition", "cond", target, sensor)

    # delegation-live: any delegated run recorded (the binding has no window).
    target, sensor = ">= 1 per 7 days", "program-runtime sessions delegations"
    if "deleg" in h:
        deleg_count = h["deleg"].count()
        row("delegation-live", {"delegations": deleg_count, "window": "all-time"}, target, deleg_count >= 1, sensor=sensor)
    else:
        dead_row("delegation-live", "deleg", target, sensor)

    # library-hygiene: unpromoted agent-authored candidates, and promoted names sharing one called-binding set.
    target, sensor = "0 candidates older than one cycle; 0 duplicate binding sets", "program-runtime library list"
    if "lib" in h:
        lib = h["lib"]
        candidates = lib.filter(lambda r: r.get("origin") == "agent-authored").count()
        sets = {}
        for r in lib.filter(lambda r: r.get("origin") != "agent-authored").head(500):
            key = tuple(sorted(set(r.get("calledBindingIds") or [])))
            if key:
                sets.setdefault(key, set()).add(r.get("name"))
        duplicates = sum(1 for names in sets.values() if len(names) > 1)
        row("library-hygiene", {"candidates": candidates, "duplicate_binding_sets": duplicates}, target,
            candidates == 0 and duplicates == 0, sensor=sensor)
    else:
        dead_row("library-hygiene", "lib", target, sensor)

    # Rows owned elsewhere: read by the named program, or waiting on a measure that does not exist yet.
    row("attribution", None, "program_id on agent-manager facts", None, unavailable=True, reason="pending_telemetry")
    row("external-friction", None, "0 recurring fingerprints", None, unavailable=True,
        reason="read_elsewhere:agent-manager.friction-digest")
    row("fleet-improve-coverage", None, "all high-volume callers conformant", None, unavailable=True,
        reason="read_elsewhere:prompt-manager.skill-set-read")

    if "shapes" in h:
        envelope["signals"]["failure_shapes"] = h["shapes"].head(6)
    # Permanent and unreliable rows do not lower the status; only a failed read does.
    envelope["status"] = "partial" if dead else "ok"
    return "report"


def step_report():  # REPORT · bounded, always
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
