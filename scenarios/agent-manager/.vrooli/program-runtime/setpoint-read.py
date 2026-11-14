"""agent-manager.setpoint-read v2 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json (inputs, invariants, bindings, outputs).
Skill:    agent-manager-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Row shape and the `reason` vocabulary are program-contracts.md's: a row the program declines
to evaluate carries reading null and a reason; a permanent reason (no_governed_binding,
pending_telemetry) never lowers the status. Every reading is a value the sensor returned or null.
"""

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
import json

inputs = program.inputs()
window_token = str(inputs.get("window_token", "TIME_WINDOW_TOKEN_LAST_7D"))
coverage_band = float(inputs.get("coverage_band", 0.95))
investigation_tag_prefix = str(inputs.get("investigation_tag_prefix", "agent-manager-investigation"))

WINDOW_TOKENS = {"TIME_WINDOW_TOKEN_THIS_WEEK", "TIME_WINDOW_TOKEN_LAST_7D", "TIME_WINDOW_TOKEN_LAST_30D",
                 "TIME_WINDOW_TOKEN_THIS_MONTH", "TIME_WINDOW_TOKEN_LAST_MONTH", "TIME_WINDOW_TOKEN_THIS_QUARTER"}
PERMANENT = ("no_governed_binding", "kernel_invoke_budget", "read_elsewhere:", "pending_telemetry")
# RunStatus values from packages/proto/schemas/agent-manager/v1/domain/types.proto that end a run.
TERMINAL = ("RUN_STATUS_NEEDS_REVIEW", "RUN_STATUS_COMPLETE", "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED", "RUN_STATUS_UNKNOWN")

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "agent-manager.setpoint-read", "version": "2",
    "status": "failed", "phase": "validate",
    "inputs": {"window_token": window_token, "coverage_band": coverage_band,
               "investigation_tag_prefix": investigation_tag_prefix},
    "signals": {"rows": [], "readable": 0, "unavailable": 0, "unavailable_permanent": 0, "detail": {}},
    "errors": [], "evidence": [],
}
handles = {}


def fail(status, klass, detail, where):
    """The one place a bad path is recorded. Sets status, appends the error, routes to report."""
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def row(name, reading, target, in_band, reason=None):
    """Append one setpoint row in the improve skill's table order. reason set means the row was not evaluated."""
    unavailable = reason is not None
    envelope["signals"]["rows"].append({
        "row": name, "reading": None if unavailable else reading, "target": target,
        "in_band": None if (unavailable or target is None) else bool(in_band),
        "unavailable": unavailable, "reason": reason,
    })
    sig = envelope["signals"]
    sig["unavailable" if unavailable else "readable"] += 1
    if unavailable and reason.startswith(PERMANENT):
        sig["unavailable_permanent"] += 1


def measure(handle):
    """A measures binding returns one record as its only row (tool-friction's summary declares the whole
    response the record); None when the handle carries no row."""
    rows = handle.head(1)
    return rows[0] if rows else None


def gate(record):
    """(state, reason) of a measure's validity; an empty record is its own gate failure."""
    if record is None:
        return (None, "unreliable:empty_measure")
    validity = record.get("validity", {}) or {}
    state = validity.get("state")
    if state == "available":
        return (state, None)
    return (state, f"unreliable:{validity.get('reason') or state or 'no validity state'}")


def rate_row(name, record, band_text, predicate):
    """One friction or throughput measure: a failed validity gate is reported with its text, never banded.
    protojson omits a double that is exactly 0.0; when the measure says it is available, an absent rate IS zero."""
    state, reason = gate(record)
    if reason:
        row(name, None, band_text, None, reason=reason)
        return
    rate = float(record.get("rate", 0.0))
    row(name, rate, band_text, predicate(rate))


# ---- state machine ---------------------------------------------------------------
def safe(fn):
    """Run one optional read; return (handle, error) so one dead dependency marks its rows, not the board."""
    try:
        return fn(), None
    except Exception as exc:
        return None, exc


def step_validate():
    if window_token not in WINDOW_TOKENS:
        return fail("failed", "invalid_input", f"window_token={window_token} not a TimeWindowToken", "validate")
    if not (0.0 < coverage_band <= 1.0):
        return fail("failed", "invalid_input", f"coverage_band={coverage_band} outside (0, 1]", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only, concurrent
    envelope["phase"] = "collect"
    window = {"token": window_token}
    # Each read is guarded: one dead binding marks its own rows unavailable, it never empties the board.
    _results = gather(
        lambda: safe(lambda: agent_manager.run.list(tag_prefix=investigation_tag_prefix, limit=50)),
        lambda: safe(lambda: agent_manager.measures.tool_friction(window=window)),
        lambda: safe(lambda: agent_manager.measures.run_success_rate(window=window)),
    )
    _names = ("inv", "tools", "success")
    _dead = {}
    for _n, (_h, _e) in zip(_names, _results):
        if _e is not None:
            _dead[_n] = program.classify(_e)
            envelope["errors"].append({"class": _dead[_n][1], "detail": f"{_n}: {str(_e)[:110]}", "where": "collect"})
    inv, tools, success = [h for h, _ in _results]
    if len(_dead) == len(_names):      # nothing answered: the board is unknown, not merely partial
        status, klass = next(iter(_dead.values()))
        return fail(status, klass, "every agent-manager read failed", "collect")
    handles.update(inv=inv, tools=tools, success=success, dead=_dead)
    cli_window = window_token.replace("TIME_WINDOW_TOKEN_", "").lower()
    envelope["evidence"] = [
        f"agent-manager run list --tag-prefix {investigation_tag_prefix}",
        f"agent-manager measures get tool-friction --window {cli_window}",
        f"agent-manager measures get run-success-rate --window {cli_window}",
    ]
    return "classify"


def step_classify():  # CLASSIFY · deterministic; every reading is count, head(1), meta, or group_by
    envelope["phase"] = "classify"
    h = handles
    detail = envelope["signals"]["detail"]
    dead = h["dead"]

    # tool-friction: one record whose summary carries the window's rates and coverage; protojson omits zeros.
    tools = None if "tools" in dead else measure(h["tools"])
    state, tools_reason = gate(tools) if "tools" not in dead else (None, "unreliable:tool_friction_read_failed")
    summary = (tools or {}).get("summary") or {}
    coverage = summary.get("coverage") or {}
    vrooli_calls = int(coverage.get("vrooliCommands") or 0)
    vrooli_known = int(coverage.get("vrooliOutcomeKnown") or 0)
    detail["tool-friction"] = {"sentence": coverage.get("sentence"), "profiles": int(summary.get("profiles") or 0),
                               "failed_calls": int(summary.get("failedCalls") or 0), "context_tax_tokens": int(summary.get("contextTaxTokens") or 0)}

    # command-outcome-coverage: Vrooli CLI calls whose outcome is known. Below the band, failure rates cannot be read.
    share = (vrooli_known / vrooli_calls) if vrooli_calls else None
    row("command-outcome-coverage", share, f">= {coverage_band}", share is not None and share >= coverage_band,
        reason=tools_reason or (None if vrooli_calls else "unreliable:no_vrooli_cli_calls"))

    # investigation-completion-rate: runs by tag prefix, complete over terminal. status may be omitted at its zero value.
    if "inv" in dead:
        row("investigation-completion-rate", None, ">= 0.8", None, reason="unreliable:run_list_read_failed")
    else:
        statuses = {k: int(v) for k, v in h["inv"].map(lambda r: {"s": r.get("status") or "RUN_STATUS_UNSPECIFIED"}).group_by("s").items()}
        terminal = sum(v for k, v in statuses.items() if k in TERMINAL)
        complete = statuses.get("RUN_STATUS_COMPLETE", 0)
        inv_rate = (complete / terminal) if terminal else None
        detail["investigation-completion-rate"] = {"complete": complete, "terminal": terminal, "by_status": statuses}
        row("investigation-completion-rate", inv_rate, ">= 0.8", inv_rate is not None and inv_rate >= 0.8,
            reason=None if terminal else "unreliable:no_terminal_runs_with_tag_prefix")

    row("skill-and-program-identity-on-facts", None, "skill_id or program_id on every fact", None,
        reason="pending_telemetry")

    # tool-friction-validity: the profiles' own validity gate.
    row("tool-friction-validity", state, "available", state == "available", reason=None if state else tools_reason)

    # bypass-calls: descriptive, no band yet: calls that reached a scenario's database, port or state around its CLI.
    row("bypass-calls", None if tools_reason else int(summary.get("bypassCalls") or 0), None, None, reason=tools_reason)

    rate_row("run-success-rate", None if "success" in dead else measure(h["success"]), ">= 0.85", lambda r: r >= 0.85)

    # The three tool-friction rates read only when their gate and the outcome coverage both hold.
    rates_reason = tools_reason or (None if share is not None and share >= coverage_band else "unreliable:outcome_coverage_below_band")
    shape = float(summary.get("shapeGuessRate") or 0.0)
    row("shape-guess-rate", shape, "<= 0.03", shape <= 0.03, reason=rates_reason)
    failure = float(summary.get("vrooliFailureRate") or 0.0)
    row("vrooli-cli-failure-rate", failure, "<= 0.05", failure <= 0.05, reason=rates_reason)
    polled = float(summary.get("polledShare") or 0.0)
    row("polled-share", None if tools_reason else polled, None, None, reason=tools_reason)

    for name in ("supervision-safety", "supervision-calibration", "supervision-outcome-coverage"):
        row(name, None, None, None, reason="read_elsewhere:agent-manager.friction-digest")

    # A permanent or unreliable row does not lower the status; only a failed read does (program-contracts.md).
    envelope["status"] = "partial" if dead else "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(json.dumps(envelope,separators=(",",":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
