"""agent-manager.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json (inputs, invariants, bindings, outputs).
Skill:    agent-manager-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Row shape and the `reason` vocabulary are program-contracts.md's: a row the program declines
to evaluate carries reading null and a reason; a permanent reason (no_governed_binding,
pending_telemetry) never lowers the status. Every reading is a value the sensor returned or null.
"""

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
window_token = str(inputs.get("window_token", "TIME_WINDOW_TOKEN_LAST_7D"))
recent_runs = int(inputs.get("recent_runs", 25))
attribution_band = float(inputs.get("attribution_band", 0.90))
investigation_tag_prefix = str(inputs.get("investigation_tag_prefix", "agent-manager-investigation"))

WINDOW_TOKENS = {"TIME_WINDOW_TOKEN_THIS_WEEK", "TIME_WINDOW_TOKEN_LAST_7D", "TIME_WINDOW_TOKEN_LAST_30D",
                 "TIME_WINDOW_TOKEN_THIS_MONTH", "TIME_WINDOW_TOKEN_LAST_MONTH", "TIME_WINDOW_TOKEN_THIS_QUARTER"}
PERMANENT = ("no_governed_binding", "kernel_invoke_budget", "read_elsewhere:", "pending_telemetry")
# RunStatus values from packages/proto/schemas/agent-manager/v1/domain/types.proto that end a run.
TERMINAL = ("RUN_STATUS_NEEDS_REVIEW", "RUN_STATUS_COMPLETE", "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED", "RUN_STATUS_UNKNOWN")

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "agent-manager.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"window_token": window_token, "recent_runs": recent_runs, "attribution_band": attribution_band,
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
    """A measures binding returns one record as its only row; None when the handle carries no row."""
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
    if recent_runs < 1 or recent_runs > 100:
        return fail("failed", "invalid_input", f"recent_runs={recent_runs} outside [1, 100]", "validate")
    if not (0.0 < attribution_band <= 1.0):
        return fail("failed", "invalid_input", f"attribution_band={attribution_band} outside (0, 1]", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only, concurrent
    envelope["phase"] = "collect"
    window = {"token": window_token}
    # Each read is guarded: one dead binding marks its own rows unavailable, it never empties the board.
    _results = gather(
        lambda: safe(lambda: agent_manager.run.list(limit=recent_runs)),
        lambda: safe(lambda: agent_manager.run.list(tag_prefix=investigation_tag_prefix, limit=50)),
        lambda: safe(lambda: agent_manager.measures.external_tool_share(window=window)),
        lambda: safe(lambda: agent_manager.measures.retry_rate(window=window)),
        lambda: safe(lambda: agent_manager.measures.tool_failure_rate(window=window)),
        lambda: safe(lambda: agent_manager.measures.repeated_work_rate(window=window)),
        lambda: safe(lambda: agent_manager.measures.run_success_rate(window=window)),
    )
    _names = ("runs", "inv", "ext", "retry", "toolfail", "repeat", "success")
    _dead = {}
    for _n, (_h, _e) in zip(_names, _results):
        if _e is not None:
            _dead[_n] = classify_transport(_e)
            envelope["errors"].append({"class": _dead[_n][1], "detail": f"{_n}: {str(_e)[:110]}", "where": "collect"})
    runs, inv, ext, retry, toolfail, repeat, success = [h for h, _ in _results]
    handles["dead"] = _dead
    if len(_dead) == len(_names):      # nothing answered: the board is unknown, not merely partial
        status, klass = next(iter(_dead.values()))
        exc = "every agent-manager read failed"
        return fail(status, klass, exc, "collect")
    # runs may be None when that one read failed; the episode rows then read unavailable, the board still reports.
    run_ids = [r["id"] for r in runs.head(recent_runs)] if runs is not None else []

    def read(run_id):
        try:
            return agent_manager.run.episodes(run_id=run_id)
        except Exception as exc:
            return exc

    episodes = gather(*[lambda i=i: read(i) for i in run_ids]) if run_ids else []
    handles.update(runs=runs, inv=inv, ext=ext, retry=retry, toolfail=toolfail, repeat=repeat, success=success,
                   episodes=episodes, run_ids=run_ids)
    cli_window = window_token.replace("TIME_WINDOW_TOKEN_", "").lower()
    envelope["evidence"] = [
        f"agent-manager run list --limit {recent_runs}; run episodes <id> x{len(run_ids)}",
        f"agent-manager run list --tag-prefix {investigation_tag_prefix}",
        f"agent-manager measures {{external-tool-share,retry-rate,tool-failure-rate,repeated-work-rate,run-success-rate}} --window {cli_window}",
    ]
    return "classify"


def step_classify():  # CLASSIFY · deterministic; every reading is count, head(1), meta, or group_by
    envelope["phase"] = "classify"
    h = handles
    detail = envelope["signals"]["detail"]

    # ownership-attribution-share: episodes over the most recent runs, by ownerConfidence.
    conf = {}
    failed_reads = 0
    for run_id, result in zip(h["run_ids"], h["episodes"]):
        if isinstance(result, Exception):
            failed_reads += 1
            if len(envelope["errors"]) < 3:  # bounded: the count is in detail, the first three carry text
                try:
                    raise result
                except Exception as exc:  # classify_transport re-raises a missing kernel name; the driver labels it
                    klass = classify_transport(exc)[1]
                envelope["errors"].append({"class": klass, "detail": str(result)[:120], "where": f"collect:episodes:{run_id}"})
            continue
        # ownerConfidence may be omitted by protojson on an unknown-owner episode: map with .get, then group_by
        for label, n in result.map(lambda e: {"c": e.get("ownerConfidence") or "unknown"}).group_by("c").items():
            conf[label] = conf.get(label, 0) + int(n)
    total = sum(conf.values())
    derived = conf.get("manifest-derived", 0)
    share = (derived / total) if total else None
    detail["ownership-attribution-share"] = {"manifest_derived": derived, "episodes": total, "runs": len(h["run_ids"]),
                                             "episode_reads_failed": failed_reads, "by_confidence": conf}
    row("ownership-attribution-share", share, f">= {attribution_band}", share is not None and share >= attribution_band,
        reason=None if total else "unreliable:no_episodes_on_recent_runs")

    # Rows with no governed binding: unavailable by construction; permanent, so they never lower the status.
    row("recurring-friction-publish-cadence", None, "published within 7 days of the 3rd run", None, reason="no_governed_binding")
    row("findings-with-resolved-effectiveness", None, ">= 0.5 measured at 14 days", None, reason="no_governed_binding")

    # investigation-completion-rate: runs by tag prefix, complete over terminal. status may be omitted at its zero value.
    statuses = {k: int(v) for k, v in h["inv"].map(lambda r: {"s": r.get("status") or "RUN_STATUS_UNSPECIFIED"}).group_by("s").items()}
    terminal = sum(v for k, v in statuses.items() if k in TERMINAL)
    complete = statuses.get("RUN_STATUS_COMPLETE", 0)
    inv_rate = (complete / terminal) if terminal else None
    detail["investigation-completion-rate"] = {"complete": complete, "terminal": terminal, "by_status": statuses}
    row("investigation-completion-rate", inv_rate, ">= 0.8", inv_rate is not None and inv_rate >= 0.8,
        reason=None if terminal else "unreliable:no_terminal_runs_with_tag_prefix")

    row("skill-and-program-identity-on-facts", None, "skill_id or program_id on every fact", None,
        reason="pending_telemetry")

    # friction-measure-validity: how many of the four friction measures pass their own validity gate.
    ext, retry, toolfail, repeat, success = (measure(h[k]) for k in ("ext", "retry", "toolfail", "repeat", "success"))
    gates = {name: gate(rec) for name, rec in
             (("external-tool-share", ext), ("retry-rate", retry), ("tool-failure-rate", toolfail), ("repeated-work-rate", repeat))}
    available = sum(1 for _, reason in gates.values() if reason is None)
    detail["friction-measure-validity"] = {  # states are on the rows themselves; only the shared counts live here
        "sample_size": int((retry or {}).get("validity", {}).get("sampleSize") or 0) if retry else None,
        "unknown_calls": int((ext or {}).get("unknownCalls") or 0) if ext else None,
    }
    row("friction-measure-validity", available, "4 of 4 available", available == 4)

    # external-tool-share: descriptive, no band yet (target null); read only when its own gate passes.
    _, ext_reason = gates["external-tool-share"]
    row("external-tool-share", None if ext_reason else float(ext.get("share", 0.0)), None, None, reason=ext_reason)

    rate_row("run-success-rate", success, ">= 0.85", lambda r: r >= 0.85)
    rate_row("retry-rate", retry, "<= 0.03", lambda r: r <= 0.03)
    rate_row("tool-failure-rate", toolfail, "<= 0.01", lambda r: r <= 0.01)
    rate_row("repeated-work-rate", repeat, "<= 0.05", lambda r: r <= 0.05)

    for name in ("supervision-safety", "supervision-calibration", "supervision-outcome-coverage"):
        row(name, None, None, None, reason="read_elsewhere:agent-manager.friction-digest")

    # A permanent or unreliable row does not lower the status; only a failed read does (program-contracts.md).
    envelope["status"] = "partial" if failed_reads else "ok"
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
