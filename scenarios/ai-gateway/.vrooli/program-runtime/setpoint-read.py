"""ai-gateway.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json (inputs, invariants, bindings, outputs).
Skill:    ai-gateway-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
The ten route measures take a TimeWindow message; the window token is an input.
Cost is unavailable when no priced route rows exist; per-caller cost remains a follow-up projection.
"""

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
window_token = str(inputs.get("window_token", "TIME_WINDOW_TOKEN_LAST_7D"))
evidence_sample = int(inputs.get("evidence_sample", 50))

WINDOW_TOKENS = {"TIME_WINDOW_TOKEN_THIS_WEEK", "TIME_WINDOW_TOKEN_LAST_7D", "TIME_WINDOW_TOKEN_LAST_30D",
                 "TIME_WINDOW_TOKEN_THIS_MONTH", "TIME_WINDOW_TOKEN_LAST_MONTH", "TIME_WINDOW_TOKEN_THIS_QUARTER"}

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "ai-gateway.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"window_token": window_token, "evidence_sample": evidence_sample},
    "signals": {"rows": [], "readable": 0, "unavailable": 0},
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


def row(name, reading, target, in_band, unavailable=False, reason=None, sensor=None):
    """Append one setpoint row in the improve skill's table order and keep the sensor as evidence."""
    envelope["signals"]["rows"].append({
        "row": name, "reading": reading, "target": target,
        "in_band": None if (in_band is None or unavailable) else bool(in_band),   # canon: null when the row has no band "unavailable": unavailable, "reason": reason,
    })
    if sensor:
        envelope["evidence"].append(sensor)
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1


def one(handle, key, default=None):
    """A measure returns one record; protojson omits zero-valued fields, so a missing count is a real zero."""
    rows = handle.head(1)
    value = (rows[0] if rows else {}).get(key, default)
    return value


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
    if evidence_sample < 1 or evidence_sample > 50:
        return fail("failed", "invalid_input", f"evidence_sample={evidence_sample} outside [1, 50]; the binding caps at 50", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only, concurrent
    envelope["phase"] = "collect"
    window = {"token": window_token}
    # Each read is guarded: one dead binding marks its own rows unavailable, it never empties the board.
    results = gather(
        lambda: safe(lambda: ai_gateway.measures.total(window=window)),
        lambda: safe(lambda: ai_gateway.measures.success_rate(window=window)),
        lambda: safe(lambda: ai_gateway.measures.fallback_rate(window=window)),
        lambda: safe(lambda: ai_gateway.measures.failure_rate(window=window)),
        lambda: safe(lambda: ai_gateway.measures.breaker_open(window=window)),
        lambda: safe(lambda: ai_gateway.measures.capacity_rejections(window=window)),
        lambda: safe(lambda: ai_gateway.measures.latency_p95(window=window)),
        lambda: safe(lambda: ai_gateway.measures.cost(window=window)),
        lambda: safe(lambda: ai_gateway.measures.tokens(window=window)),
        lambda: safe(lambda: ai_gateway.measures.local_share(window=window)),
        lambda: safe(lambda: ai_gateway.routing.evidence_list(limit=evidence_sample)),
    )
    names = ("total", "success", "fallback", "failure", "breaker", "capacity", "p95", "cost", "tokens", "local_share", "evidence")
    dead = {}
    for name, (handle, err) in zip(names, results):
        handles[name] = handle
        if err is not None:
            dead[name] = classify_transport(err)
            envelope["errors"].append({"class": dead[name][1], "detail": f"{name}: {str(err)[:110]}", "where": "collect"})
    handles["dead"] = dead
    if len(dead) == len(names):        # nothing answered: the board is unknown, not merely partial
        envelope["status"] = next(iter(dead.values()))[0]
        return "report"
    return "classify"


def step_classify():  # CLASSIFY · deterministic; every reading is head(1), count, or group_by
    envelope["phase"] = "classify"
    h = handles
    cli_window = window_token.replace("TIME_WINDOW_TOKEN_", "").lower()

    cost = one(h["cost"], "costUsd") if h["cost"] is not None else None
    row("cost-per-caller", {"total_cost_usd": cost, "basis": "priced route_events rows"}, "cost_usd per (scenario, operation) readable in a window", False,
        unavailable=(h["cost"] is None), reason="route cost unavailable" if h["cost"] is None else "per-caller projection remains a follow-up",
        sensor=f"ai-gateway measures cost --window {cli_window}")

    # local-share is now a windowed measure over successful local-served routes.
    share = one(h["local_share"], "share") if h["local_share"] is not None else None
    row("local-share", {"share": share, "basis": "windowed route_events measure"}, ">= 0.80 of routes local", share is not None and share >= 0.80,
        unavailable=(share is None), reason="local-share measure unavailable" if share is None else None,
        sensor=f"ai-gateway measures local-share --window {cli_window}")

    tokens = one(h["tokens"], "totalTokens") if h["tokens"] is not None else None
    row("token-volume", {"total_tokens": tokens}, None, None, unavailable=(tokens is None),
        reason="token measure unavailable" if tokens is None else None,
        sensor=f"ai-gateway measures tokens --window {cli_window}")

    total = int(one(h["total"], "count", 0))
    # protojson omits a double that is exactly 0.0; when routes exist in the window an absent rate IS zero
    # (the same rule the counts already use). With no routes, a rate is genuinely unknown.
    fallback = float(one(h["fallback"], "rate", 0.0)) if total else one(h["fallback"], "rate")
    failure = float(one(h["failure"], "rate", 0.0)) if total else one(h["failure"], "rate")
    breaker = int(one(h["breaker"], "count", 0))
    capacity = int(one(h["capacity"], "count", 0))
    p95 = one(h["p95"], "latencyMs")
    success = float(one(h["success"], "rate", 0.0)) if total else one(h["success"], "rate")

    row("fallback-rate", {"rate": fallback, "routes": total}, "<= 0.02", fallback is not None and float(fallback) <= 0.02,
        unavailable=(total == 0), reason=None if total else "no routes in window",
        sensor=f"ai-gateway measures fallback-rate --window {cli_window}")
    row("failure-rate", {"rate": failure, "routes": total}, "<= 0.02", failure is not None and float(failure) <= 0.02,
        unavailable=(total == 0), reason=None if total else "no routes in window",
        sensor=f"ai-gateway measures failure-rate --window {cli_window}")
    row("breaker-open", {"count": breaker, "routes": total}, "0 routes blocked by an open breaker", breaker == 0 and total > 0,
        unavailable=(total == 0), reason=None if total else "no routes in window",
        sensor=f"ai-gateway measures breaker-open --window {cli_window}")
    row("latency-p95", {"latency_ms": p95}, "<= 4000 ms", p95 is not None and int(p95) <= 4000,
        unavailable=(p95 is None), reason=None if p95 is not None else "no latency sample in window",
        sensor=f"ai-gateway measures latency-p95 --window {cli_window}")
    row("capacity-rejections", {"count": capacity, "routes": total}, "0 local routes rejected for capacity", capacity == 0 and total > 0,
        unavailable=(total == 0), reason=None if total else "no routes in window",
        sensor=f"ai-gateway measures capacity-rejections --window {cli_window}")
    row("success-rate", {"rate": success, "routes": total}, ">= 0.97", success is not None and float(success) >= 0.97,
        unavailable=(success is None), reason=None if success is not None else "no routes in window",
        sensor=f"ai-gateway measures success-rate --window {cli_window}")
    row("route-volume", {"count": total}, "pending-baseline: rising with adoption", False,
        sensor=f"ai-gateway measures total --window {cli_window}")

    # Canon (program-contracts.md): a permanent reason does not lower the status. Only a row the
    # owner failed to answer this time, or a read that failed outright, makes the board partial.
    _transient = [r for r in envelope["signals"]["rows"]
                  if r.get("unavailable") and str(r.get("reason") or "").startswith("scenario_unreachable")]
    envelope["status"] = "partial" if (_transient or envelope["errors"]) else "ok"
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
