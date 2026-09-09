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

import json

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
inputs = program.inputs()
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
    "shapes": lambda: program_runtime.shapes.list(uncovered_only=True, min_occurrences=3),
    "failures": lambda: program_runtime.programs.mine(),
    "progs": lambda: program_runtime.programs.list(provenance="agent", since_seconds=30 * 24 * 60 * 60),
    "portfolio": lambda: program_runtime.programs.portfolio(window_days=30, scenario="", include_ad_hoc=False),
    "maturity": lambda: lib.program_runtime.portfolio_audit(min_rung="S3", window_days=30),
    "taskmetrics": lambda: optional_read("delivery_metrics"),
    "fragments": lambda: optional_read("fragment_list"),
    "advice": lambda: advice_read(),
}


def optional_read(method):
    """Return an empty typed handle when optional telemetry is not admitted."""
    try:
        target = tasks
    except NameError:
        return Handle([])
    return getattr(target, method)()


def advice_read():
    try:
        return lib.vrooli_memory.compare_outcomes(scope="program-runtime-usage", cohort_limit=10)
    except AttributeError:
        return Handle([])


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
            status, klass = program.classify(result)
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

    # agent-failure-rate: ask the binding for exactly the corpus named by the row.
    target = f"< {agent_failure_band}"
    sensor = "program-runtime programs list --provenance agent --since-seconds 2592000"
    if "progs" in h:
        agent_programs = h["progs"]
        total = agent_programs.count()
        failed = agent_programs.filter(lambda r: r.get("status") == "PROGRAM_STATUS_FAILED").count()
        rate = (failed / total) if total else None
        row("agent-failure-rate", {"failed": failed, "total": total, "rate": rate, "window": "last-30-days"}, target,
            rate is not None and rate < agent_failure_band,
            unavailable=(total == 0), reason=None if total else "unreliable:no agent-provenance programs in corpus",
            sensor=sensor)
    else:
        dead_row("agent-failure-rate", "progs", target, sensor)

    # program-adoption: the portfolio keeps this reading on the declared
    # portfolio boundary; the bounded agent-provenance list is the identity
    # proof. A daemon harness is an operator of the loop, not adoption.
    target, sensor = ">= 0.02 of agent runs", "program-runtime programs portfolio --window-days 30 + programs list --provenance agent --since-seconds 2592000"
    if "portfolio" in h and "progs" in h:
        agent_rows = h["progs"].head(1000)
        agent_runs = len(agent_rows)
        unattributed = 0
        daemon_rows = 0
        adopted = 0
        for item in agent_rows:
            caller = item.get("caller") if isinstance(item.get("caller"), dict) else {}
            run_id = str(item.get("callerRunId", item.get("caller_run_id", caller.get("runId", caller.get("run_id", "")))) or "").strip()
            harness = str(item.get("callerHarness", item.get("caller_harness", caller.get("harness", ""))) or "").strip().lower()
            if harness == "daemon":
                daemon_rows += 1
            elif run_id:
                adopted += 1
        eligible = max(agent_runs - daemon_rows, 0)
        unattributed = max(eligible - adopted, 0)
        rate = (adopted / eligible) if eligible else None
        row("program-adoption", {"adopted_agent_runs": adopted, "eligible_agent_runs": eligible,
                                  "unattributed_agent_runs": unattributed, "daemon_runs_excluded": daemon_rows,
                                  "rate": rate, "window": "last-30-days"}, target,
            rate is not None and rate >= 0.02, unavailable=(eligible == 0),
            reason=None if eligible else "unreliable:no agent-provenance runs in corpus", sensor=sensor)
    else:
        dead_row("program-adoption", "portfolio", target, sensor)

    # portfolio-maturity: consume the declared portfolio-audit contract; do not
    # recreate its eight-dimension score in the setpoint reader.
    target, sensor = ">= 0.8", "program-runtime library run program-runtime.portfolio-audit --input min_rung=S3 window_days=30"
    if "maturity" in h:
        audit_rows = h["maturity"].head(1)
        audit = audit_rows[0] if audit_rows else {}
        audit_status = audit.get("status")
        score = (audit.get("signals") or {}).get("score")
        if audit_status == "ok" and score is not None:
            score = float(score)
            row("portfolio-maturity", {"score": score, "scored": int((audit.get("signals") or {}).get("scored", 0))},
                target, score >= 0.8, sensor=sensor)
        else:
            reason = f"unreliable:portfolio-audit status={audit_status or 'missing'}"
            row("portfolio-maturity", None, target, None, unavailable=True, reason=reason, sensor=sensor)
    else:
        dead_row("portfolio-maturity", "maturity", target, sensor)

    # program-health: a declared program with any failed execution is unhealthy.
    target, sensor = "0 unhealthy declared programs", "program-runtime programs portfolio --window-days 30"
    if "portfolio" in h:
        unhealthy = []
        portfolio_rows = h["portfolio"].head(1000)
        for item in portfolio_rows:
            if float(item.get("failed", 0) or 0) > 0:
                unhealthy.append({"name": item.get("name"), "failed": int(float(item.get("failed", 0) or 0)),
                                  "success_rate": float(item.get("success_rate", item.get("successRate", 0)) or 0)})
        row("program-health", {"unhealthy": len(unhealthy), "declared_programs": len(portfolio_rows),
                                "examples": unhealthy[:10]}, target, len(unhealthy) == 0, sensor=sensor)
    else:
        dead_row("program-health", "portfolio", target, sensor)

    # unexercised-contracts: the portfolio metadata owns this exact set.
    target, sensor = "0", "program-runtime programs portfolio --window-days 30 → never_executed"
    if "portfolio" in h:
        portfolio_meta = h["portfolio"].meta() or {}
        never_executed = portfolio_meta.get("neverExecuted", portfolio_meta.get("never_executed", [])) or []
        count = len(never_executed)
        row("unexercised-contracts", {"count": count, "examples": list(never_executed)[:10]}, target, count == 0, sensor=sensor)
    else:
        dead_row("unexercised-contracts", "portfolio", target, sensor)

    target, sensor = "0", "program-runtime durable task metrics: blocked deliveries older than 24 hours"
    if "taskmetrics" in h:
        metric = (h["taskmetrics"].head(1) or [{}])[0]
        count = int(metric.get("blocked_deliveries_aged", 0) or 0)
        row("blocked-deliveries-aged", count, target, count == 0, sensor=sensor)
    else:
        dead_row("blocked-deliveries-aged", "taskmetrics", target, sensor)

    target, sensor = ">= 0.2", "vrooli-memory.compare-outcomes advice-outcomes over the last 30 days"
    if "advice" in h:
        comparison = (h["advice"].head(1) or [{}])[0]
        advice_rows = ((comparison.get("signals") or {}).get("rows") or [])
        advice = next((item.get("reading") for item in advice_rows if item.get("row") == "advice-outcomes"), None)
        cohorts = (advice or {}).get("cohorts", []) if isinstance(advice, dict) else []
        aggregate = (((comparison.get("signals") or {}).get("aggregate") or {}).get("advice") or {})
        if isinstance(aggregate, dict) and ("applied" in aggregate or "rejected" in aggregate):
            applied = int(aggregate.get("applied", 0) or 0)
            rejected = int(aggregate.get("rejected", 0) or 0)
        else:
            applied = sum(int(c.get("appliedAdvice", 0) or 0) for c in cohorts if isinstance(c, dict))
            rejected = sum(int(c.get("rejectedAdvice", 0) or 0) for c in cohorts if isinstance(c, dict))
        total = applied + rejected
        ratio = (applied / total) if total else None
        reliability = ((comparison.get("signals") or {}).get("reliability") or {})
        row("advice-application-ratio", {"applied": applied, "rejected": rejected, "ratio": ratio,
                                          "source_reliable": reliability.get("reliable"),
                                          "source_reason": reliability.get("reason")}, target,
            ratio is not None and ratio >= 0.2, unavailable=(total == 0), reason=None if total else "unreliable:no advice decisions", sensor=sensor)
    else:
        dead_row("advice-application-ratio", "advice", target, sensor)

    target, sensor = ">= 0.5 after verified fragment", "program-runtime retained successful fragment observations"
    if "fragments" in h:
        payload = (h["fragments"].head(1) or [{}])[0]
        fragments = payload.get("fragments") or []
        verified = sum(int(f.get("verified", 0) or 0) for f in fragments if isinstance(f, dict))
        cached = sum(int(f.get("cached_runs", 0) or 0) for f in fragments if isinstance(f, dict))
        total = verified
        rate = (cached / total) if total else None
        row("fragment-cache-hit-rate", {"cached_runs": cached, "act_runs": total, "rate": rate}, target,
            rate is not None and rate >= 0.5, unavailable=(verified == 0), reason=None if verified else "pending_telemetry", sensor=sensor)
        promotable_rows = [f for f in fragments if isinstance(f, dict) and int(f.get("verified", 0) or 0) >= 5 and int(f.get("contexts", 0) or 0) >= 2 and int(f.get("contradicted_since_edit", 0) or 0) == 0]
        promotable = len(promotable_rows)
        envelope_fragment_reading = {"cached_runs": cached, "act_runs": total, "rate": rate,
                                     "candidates": [{"step_key": f.get("step_key"), "verified": f.get("verified")} for f in promotable_rows[:10]]}
        envelope["signals"]["rows"][-1]["reading"] = envelope_fragment_reading
        row("promotable-fragments", promotable, "0 without a filed route", promotable == 0,
            sensor="program-runtime durable fragment promotion candidates")
    else:
        dead_row("fragment-cache-hit-rate", "fragments", target, sensor)
        row("promotable-fragments", None, "0 without a filed route", None, unavailable=True, reason="pending_telemetry",
            sensor="program-runtime durable fragment promotion candidates")

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

    # uncovered-recurring-shapes: nominations are the actionable reuse gap.
    target, sensor = "0 nominated shapes with no declared contract", "program-runtime shapes list --uncovered --min-occurrences 3"
    if "shapes" in h:
        nominated = h["shapes"].count()
        row("uncovered-recurring-shapes", {"nominated": nominated}, target, nominated == 0, sensor=sensor)
    else:
        dead_row("uncovered-recurring-shapes", "shapes", target, sensor)

    # Rows owned elsewhere: read by the named program, or waiting on a measure that does not exist yet.
    row("attribution", None, "program_id on agent-manager facts", None, unavailable=True, reason="pending_telemetry")
    row("external-friction", None, "0 recurring fingerprints", None, unavailable=True,
        reason="read_elsewhere:agent-manager.friction-digest")
    row("fleet-improve-coverage", None, "all high-volume callers conformant", None, unavailable=True,
        reason="read_elsewhere:prompt-manager.skill-set-read")

    if "failures" in h:
        # Recurring binding sets are reuse opportunities, not failure causes.
        # The improve router consumes only the failure miner's shape/count
        # projection; full exemplars would waste output and truncate the board.
        envelope["signals"]["failure_shapes"] = h["failures"].map(
            lambda item: {"shape": item["shape"], "count": int(item.get("count", 0))}
        ).sort("count", reverse=True).head(6)
        envelope["signals"]["failure_shapes_window"] = "all-time"
    # Permanent and unreliable rows do not lower the status; only a failed read does.
    envelope["status"] = "partial" if dead else "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    # Keep the envelope inside the declared output budget even as rows are
    # added. The row readings are authoritative; evidence is a bounded sample.
    if len(json.dumps(envelope, allow_nan=False, separators=(",", ":")).encode()) > 3800:
        envelope["evidence"] = envelope["evidence"][:6]
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
