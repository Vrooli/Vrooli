"""search-hub.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json.
Skill:    search-hub-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
The routing rows read the newest run of the composed router.routing suite; when that
run carries a strategy-compare tag it is the candidate arm, and the rows say so rather
than reporting the incumbent. Row `reason` values are the closed vocabulary in
program-contracts.md; a row with a permanent reason (pending_telemetry, unreliable:*)
does not lower the status, only a transient scenario_unreachable row or a failed read does.
"""

try:
    inputs
except NameError:
    inputs = {}
window_token = str(inputs.get("window_token", "TIME_WINDOW_TOKEN_LAST_7D"))
insights_window = str(inputs.get("insights_window", "7") or "7")
routing_suite = str(inputs.get("routing_suite", "router.routing"))

envelope = {
    "program": "search-hub.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"window_token": window_token, "insights_window": insights_window, "routing_suite": routing_suite},
    "signals": {"rows": [], "readable": 0, "unavailable": 0, "unavailable_transient": 0},
    "errors": [], "evidence": [],
}
handles = {}
WINDOWS = ("TIME_WINDOW_TOKEN_THIS_WEEK", "TIME_WINDOW_TOKEN_LAST_7D", "TIME_WINDOW_TOKEN_LAST_30D",
           "TIME_WINDOW_TOKEN_THIS_MONTH", "TIME_WINDOW_TOKEN_LAST_MONTH", "TIME_WINDOW_TOKEN_THIS_QUARTER")
PERMANENT_REASONS = ("no_governed_binding", "kernel_invoke_budget", "pending_telemetry", "read_elsewhere:", "unreliable:")


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


def row(name, reading, target, in_band, unavailable=False, reason=None, sensor=None):
    """One setpoint row. target/in_band are None when the row has no band; reason is the closed vocabulary."""
    envelope["signals"]["rows"].append({
        "row": name, "reading": reading, "target": target,
        "in_band": None if in_band is None else (bool(in_band) and not unavailable),
        "unavailable": unavailable, "reason": reason,
    })
    if sensor and sensor not in envelope["evidence"]:
        envelope["evidence"].append(sensor[:70])
    if unavailable:
        envelope["signals"]["unavailable"] += 1
        if not str(reason or "").startswith(PERMANENT_REASONS):
            envelope["signals"]["unavailable_transient"] += 1
    else:
        envelope["signals"]["readable"] += 1


def safe(fn):
    try:
        return fn(), None
    except Exception as exc:
        return None, exc


def read_error(err, where):
    """Record a failed governed read once and return the row reason for it."""
    _, klass = classify_transport(err)
    envelope["errors"].append({"class": klass, "detail": f"{where}: {str(err)[:140]}", "where": "collect"})
    return klass  # scenario_unreachable is transient; any other class is a failed read and also lowers the status


def step_validate():
    if window_token not in WINDOWS:
        return fail("failed", "invalid_input", f"window_token={window_token} not a TimeWindowToken name", "validate")
    if not routing_suite:
        return fail("failed", "invalid_input", "routing_suite is required", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only, concurrent; each may fail alone
    envelope["phase"] = "collect"
    win = {"token": window_token}
    results = gather(
        lambda: safe(lambda: search_hub.evals.runs(suite_id=routing_suite, limit=1)),
        lambda: safe(lambda: search_hub.metrics.provider_degradation_rate(window=win)),
        lambda: safe(lambda: search_hub.metrics.degraded_query_rate(window=win)),
        lambda: safe(lambda: search_hub.metrics.federated_latency(window=win)),
        lambda: safe(lambda: search_hub.insights.insights(window=insights_window, rows="providers")),
        lambda: safe(lambda: search_hub.federation.status(rows="providers")),
        lambda: safe(lambda: search_hub.evals.list()),
    )
    names = ("routing", "pdr", "dqr", "lat", "ins", "fed", "suites")
    for name, (h, err) in zip(names, results):
        handles[name] = h
        handles[name + "_err"] = err
    if all(handles[n] is None for n in names):
        status, klass = classify_transport(handles["routing_err"])
        return fail(status, klass, handles["routing_err"], "collect")
    # newest run per suite for the eval-floors row (bounded: registered suites only, limit 1 each)
    if handles["suites"] is not None:
        suite_ids = [s["sid"] for s in handles["suites"].map(lambda r: {"sid": r.get("suiteId")}).head(60) if s["sid"] and s["sid"] != routing_suite]
        handles["suite_ids"] = suite_ids
        handles["suite_runs"] = list(gather(*[(lambda sid=sid: safe(lambda: search_hub.evals.runs(suite_id=sid, limit=1))) for sid in suite_ids])) if suite_ids else []
    return "classify"


def step_classify():  # CLASSIFY · rows in the skill's table order; every reading is head/meta/count
    envelope["phase"] = "classify"
    h = handles
    routing_rows = (("routing-precision", "routingPrecision", ">= 0.90", 0.90), ("e2e-pass-rate", "passRate", ">= 0.50", 0.50), ("retrieval-recall", "retrievalRecall", ">= 0.85", 0.85))
    r_sensor = f"search-hub evals runs {routing_suite} --limit 1 (aggregate)"
    if h["routing"] is None:
        reason = read_error(h["routing_err"], f"evals runs {routing_suite}")
        for name, _, target, _ in routing_rows:
            row(name, None, target, False, unavailable=True, reason=reason, sensor=r_sensor)
    else:
        runs = h["routing"].head(1)
        if not runs:
            for name, _, target, _ in routing_rows:
                row(name, None, target, False, unavailable=True, reason="unreliable:no_run", sensor=r_sensor)
        else:
            r = runs[0]
            agg = r.get("aggregate") or {}
            tag = str(r.get("tag") or "")
            ctx = {"run_id": r.get("runId"), "created_at": r.get("createdAt"), "tag": tag, "tier": r.get("tier"),
                   "arm": "candidate" if tag.startswith("strategy-compare:") else "incumbent-or-untagged"}
            envelope["evidence"].append(f"search-hub://eval-run/{r.get('runId')}")
            extra = {"routingPrecision": {"graded": agg.get("gradedCases"), "unavailable_cases": agg.get("unavailableCases")},
                     "passRate": {"met": agg.get("met"), "below": agg.get("below"), "cases": agg.get("cases")}, "retrievalRecall": {}}
            for name, key, target, floor in routing_rows:
                v = agg.get(key)
                if v is None:  # the run stored no aggregate for this measure: the sensor answered, its validity gate failed
                    row(name, dict(ctx, value=None), target, False, unavailable=True, reason="unreliable:no_aggregate", sensor=r_sensor)
                else:
                    row(name, dict(ctx, value=v, **extra[key]), target, float(v) >= floor, sensor=r_sensor)

    for name, key, target, band in (("provider-degradation", "pdr", "<= 0.20", lambda v: v <= 0.20),
                                     ("degraded-query-rate", "dqr", "<= 0.20", lambda v: v <= 0.20)):
        sensor = f"search-hub metrics {'provider-degradation-rate' if key == 'pdr' else 'degraded-query-rate'} --window {window_token.split('_TOKEN_')[-1].lower()}"
        if h[key] is None:
            row(name, None, target, False, unavailable=True, reason=read_error(h[key + "_err"], sensor), sensor=sensor)
        else:
            m = h[key].meta() or {}
            # The field is `rate` on both responses (verified against the proto); protojson omits it when it is
            # exactly 0.0, which is the in-band value, so an absent rate on a reachable measure is zero.
            v = float(m.get("rate", 0.0))
            row(name, {"rate": v, "window": window_token, **{k: m.get(k) for k in ("degradedCount", "timesRouted", "degradedQueries", "totalQueries") if k in m}}, target, band(v), sensor=sensor)

    lat_sensor = f"search-hub metrics federated-latency --window {window_token.split('_TOKEN_')[-1].lower()}"
    if h["lat"] is None:
        row("federated-latency", None, "p95 <= 4000 ms", False, unavailable=True, reason=read_error(h["lat_err"], lat_sensor), sensor=lat_sensor)
    else:
        m = h["lat"].meta() or {}
        p95 = float(m.get("p95Ms", 0.0))  # absent on a reachable measure is zero: no query in the window
        no_sample = (p95 == 0.0)
        row("federated-latency", {"p50_ms": m.get("p50Ms"), "p95_ms": p95, "window": window_token}, "p95 <= 4000 ms",
            p95 <= 4000, unavailable=no_sample, reason="unreliable:no_sample" if no_sample else None, sensor=lat_sensor)

    ins_sensor = f"search-hub insights insights --window {insights_window}"
    if h["ins"] is None:
        reason = read_error(h["ins_err"], ins_sensor)
        row("under-utilized-providers", None, "0 retirement candidates", False, unavailable=True, reason=reason, sensor=ins_sensor)
        row("zero-result-rate", None, "<= 0.10", False, unavailable=True, reason=reason, sensor=ins_sensor)
    else:
        m = h["ins"].meta() or {}
        rc = [c.get("providerId") for c in (m.get("retirementCandidates") or []) if isinstance(c, dict)]
        row("under-utilized-providers", {"retirement_candidates": rc, "providers_in_window": h["ins"].count()}, "0 retirement candidates", len(rc) == 0, sensor=ins_sensor)
        sufficient = bool(m.get("sampleSufficient", False))
        zr = float(m.get("zeroResultRate", 0.0))  # absent on a sufficient sample is zero
        row("zero-result-rate", {"rate": zr, "zero_result_queries": m.get("zeroResultQueries"), "total_queries": m.get("totalQueries"), "sample_sufficient": sufficient, "minimum_sample_count": m.get("minimumSampleCount")},
            "<= 0.10", zr <= 0.10, unavailable=not sufficient, reason=None if sufficient else "unreliable:sample_too_small", sensor=ins_sensor)

    fed_sensor = "search-hub federation status (reachable per provider)"
    if h["fed"] is None:
        row("provider-reachability", None, "0 unreachable active providers", False, unavailable=True, reason=read_error(h["fed_err"], fed_sensor), sensor=fed_sensor)
    else:
        unreachable = [r.get("providerId") for r in h["fed"].filter(lambda r: r.get("reachable") is False).head(40)]
        row("provider-reachability", {"unreachable": unreachable, "providers": h["fed"].count()}, "0 unreachable active providers", len(unreachable) == 0, sensor=fed_sensor)

    # eval-floors: no suite records a floor (search-hub-improve §4), so the row has no band; the newest
    # pass rates are reported for the improve skill's derivation and nothing is compared against 1.0.
    ev_sensor = "search-hub evals runs <suite_id> --limit 1 for every registered suite (aggregate.passRate)"
    if h["suites"] is None:
        row("eval-floors", None, None, None, unavailable=True, reason=read_error(h["suites_err"], "evals list"), sensor=ev_sensor)
    else:
        newest, no_run, no_agg, errs = [], [], [], 0
        for sid, (run, err) in zip(h.get("suite_ids", []), h.get("suite_runs", [])):
            if err is not None:
                errs += 1; continue
            first = run.head(1)
            if not first:
                no_run.append(sid); continue
            pr = (first[0].get("aggregate") or {}).get("passRate")
            if pr is None:
                no_agg.append(sid); continue
            newest.append({"suite_id": sid, "pass_rate": round(float(pr), 2)})
        newest.sort(key=lambda e: e["pass_rate"])
        row("eval-floors", {"suites": len(h.get("suite_ids", [])), "with_run": len(newest), "lowest_newest_pass_rates": newest[:5],
                            "no_run": no_run[:8], "no_aggregate": no_agg[:8], "read_errors": errs, "floors_recorded": 0},
            None, None, unavailable=True, reason="pending_telemetry", sensor=ev_sensor)

    envelope["status"] = "partial" if envelope["signals"]["unavailable_transient"] or envelope["errors"] else "ok"
    return "report"


def step_report():  # REPORT
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
