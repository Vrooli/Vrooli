"""web-search.setpoint-read v1 — read every improve-setpoint row in one submission.

Contract: setpoint-read.json.
Skill:    web-search-improve §2 (the rows), §3 (the sensors).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Rows whose sensor is not a governed binding (cache hit rate, budget exhaustion,
steps at S3) are reported unavailable with the reason; nothing is estimated.
"""

try:
    inputs
except NameError:
    inputs = {}
effectiveness_limit = int(inputs.get("effectiveness_limit", 500))
decayed_below = float(inputs.get("decayed_below", 0.5))
count_window = str(inputs.get("count_window", "TIME_WINDOW_TOKEN_LAST_30D"))

envelope = {
    "program": "web-search.setpoint-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"effectiveness_limit": effectiveness_limit, "decayed_below": decayed_below, "count_window": count_window},
    "signals": {"rows": [], "readable": 0, "unavailable": 0},
    "errors": [], "evidence": [],
}
handles = {}
WINDOWS = ("TIME_WINDOW_TOKEN_THIS_WEEK", "TIME_WINDOW_TOKEN_LAST_7D", "TIME_WINDOW_TOKEN_LAST_30D",
           "TIME_WINDOW_TOKEN_THIS_MONTH", "TIME_WINDOW_TOKEN_LAST_MONTH", "TIME_WINDOW_TOKEN_THIS_QUARTER")


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
    envelope["signals"]["rows"].append({
        "row": name, "reading": reading, "target": target,
        "in_band": None if (in_band is None or unavailable) else bool(in_band),   # canon: null when the row has no band "unavailable": unavailable, "reason": reason,
    })
    if sensor and sensor not in envelope["evidence"]:
        envelope["evidence"].append(sensor[:70])
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1


def safe(fn):
    """Run one optional read; return (handle, error) so one dead dependency marks its rows, not the board."""
    try:
        return fn(), None
    except Exception as exc:
        return None, exc


def step_validate():
    if not (1 <= effectiveness_limit <= 1000):
        return fail("failed", "invalid_input", f"effectiveness_limit={effectiveness_limit} outside 1..1000", "validate")
    if not (0.0 < decayed_below <= 1.0):
        return fail("failed", "invalid_input", f"decayed_below={decayed_below} outside (0, 1]", "validate")
    if count_window not in WINDOWS:
        return fail("failed", "invalid_input", f"count_window={count_window} not a TimeWindowToken name", "validate")
    return "collect"


def step_collect():  # COLLECT · web-search reads and search-hub reads, concurrent; each may fail alone
    envelope["phase"] = "collect"
    (eff, eff_err), (cnt, cnt_err), (ins, ins_err), (prov, prov_err) = gather(
        lambda: safe(lambda: web_search.findings.effectiveness(limit=effectiveness_limit, include_disputed=True)),
        lambda: safe(lambda: web_search.findings.count(window={"token": count_window})),
        lambda: safe(lambda: search_hub.insights.insights(rows="providers")),
        lambda: safe(lambda: search_hub.providers.list(rows="providers")),
    )
    handles.update(eff=eff, eff_err=eff_err, cnt=cnt, cnt_err=cnt_err, ins=ins, ins_err=ins_err, prov=prov, prov_err=prov_err)
    for name, err in (("web-search", eff_err), ("search-hub", ins_err)):
        if err is not None:
            status, klass = classify_transport(err)
            envelope["errors"].append({"class": klass, "detail": f"{name}: {str(err)[:100]}", "where": "collect"})
    return "classify"


def step_classify():  # CLASSIFY · every reading is count/head/filter; rows in the skill's table order
    envelope["phase"] = "classify"
    h = handles
    eff_sensor = "web-search findings effectiveness --limit N --include-disputed"
    if h["eff"] is None:
        reason = "web-search unreachable: " + str(h["eff_err"])[:60]
        for name, target in (("surfaced-rate", ">= 0.50"), ("used-rate", ">= 0.20"), ("never-surfaced-share", "<= 0.20")):
            row(name, None, target, False, unavailable=True, reason=reason, sensor=eff_sensor)
    else:
        rows_ = h["eff"].head(effectiveness_limit)
        total = len(rows_)
        surfaced = sum(1 for r in rows_ if int(r.get("surfacedCount", 0) or 0) > 0)
        used = sum(1 for r in rows_ if int(r.get("usedCount", 0) or 0) > 0)
        nsd = sum(1 for r in rows_ if int(r.get("surfacedCount", 0) or 0) == 0 and float(r.get("effectiveConfidence", 0.0) or 0.0) < decayed_below)
        s_rate = (surfaced / total) if total else None
        u_rate = (used / surfaced) if surfaced else None
        ns_share = (nsd / total) if total else None
        row("surfaced-rate", {"surfaced": surfaced, "total": total, "rate": s_rate}, ">= 0.50",
            s_rate is not None and s_rate >= 0.50, unavailable=(total == 0), reason=None if total else "no findings in ledger", sensor=eff_sensor)
        row("used-rate", {"used": used, "surfaced": surfaced, "rate": u_rate}, ">= 0.20",
            u_rate is not None and u_rate >= 0.20, unavailable=(surfaced == 0), reason=None if surfaced else "no finding has been surfaced yet", sensor=eff_sensor)
        row("never-surfaced-share", {"never_surfaced_decayed": nsd, "total": total, "share": ns_share, "decayed_below": decayed_below}, "<= 0.20",
            ns_share is not None and ns_share <= 0.20, unavailable=(total == 0), reason=None if total else "no findings in ledger", sensor=eff_sensor)

    ins_sensor = "search-hub insights insights (providers web-search.live vs web-search.learnings, timesRouted)"
    if h["ins"] is None:
        row("live-vs-local-ratio", None, None, None, unavailable=True, reason="scenario_unreachable: search-hub: " + str(h["ins_err"])[:50], sensor=ins_sensor)
    else:
        prows = h["ins"].filter(lambda r: str(r.get("providerGroup", "")) == "web-search").head(10)
        routed = {r.get("providerId"): int(r.get("timesRouted", 0) or 0) for r in prows}
        live = routed.get("web-search.live", 0)
        local = routed.get("web-search.learnings", 0)
        ratio = (live / local) if local else None
        # No band: the improve skill states a direction ("reach the live web progressively less"),
        # and insights has no windowed read, so a single ratio cannot be in or out of band.
        row("live-vs-local-ratio", {"live_routed": live, "learnings_routed": local, "ratio": ratio}, None, None,
            unavailable=(local == 0 and live == 0),
            reason="pending_telemetry: insights has no windowed read; this ratio is all-time and is a reference point"
                   if (local or live) else "unreliable:no routed calls recorded for web-search providers",
            sensor=ins_sensor)

    row("cache-hit-rate", None, ">= 0.30", False, unavailable=True,
        reason="pending_telemetry: SearchResponse.cached exists per call; no measure aggregates it", sensor="measure not declared (measures-adoption)")
    row("budget-exhaustion", None, "0 governor-declined calls per day", False, unavailable=True,
        reason="pending_telemetry: degraded_reason is per call; no measure counts governor declines", sensor="measure not declared (measures-adoption)")

    prov_sensor = "search-hub providers list (providerGroup web-search, lifecycle)"
    if h["prov"] is None:
        row("provider-lifecycle", None, "learnings production; live fixture or better", False, unavailable=True,
            reason="search-hub unreachable: " + str(h["prov_err"])[:60], sensor=prov_sensor)
    else:
        prows = h["prov"].filter(lambda r: str(r.get("providerGroup", "")) == "web-search").head(10)
        life = {r.get("providerId"): str(r.get("lifecycle", "")) for r in prows}
        learn_ok = life.get("web-search.learnings", "").endswith("PRODUCTION")
        row("provider-lifecycle", life, "learnings production; live fixture or better", learn_ok and "web-search.live" in life,
            unavailable=(not life), reason=None if life else "web-search providers not registered", sensor=prov_sensor)

    cnt_sensor = f"web-search findings count --window <token>"
    if h["cnt"] is None:
        row("capture-volume", None, None, None, unavailable=True, reason="scenario_unreachable: web-search: " + str(h["cnt_err"])[:50], sensor=cnt_sensor)
    else:
        c = int((h["cnt"].meta() or {}).get("count", 0) or 0)  # protojson omits a zero count: absent means 0 on a reachable measure
        # pending-baseline: the band is a direction across windows, which one reading cannot decide.
        row("capture-volume", {"count": c, "window": count_window}, None, None, sensor=cnt_sensor)

    # Canon (program-contracts.md): a permanent reason does not lower the status. Only a row the
    # owner failed to answer this time, or a read that failed outright, makes the board partial.
    _transient = [r for r in envelope["signals"]["rows"]
                  if r.get("unavailable") and str(r.get("reason") or "").startswith("scenario_unreachable")]
    envelope["status"] = "partial" if (_transient or envelope["errors"]) else "ok"
    if h["eff"] is None and h["ins"] is None:
        envelope["status"] = "unavailable"
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
