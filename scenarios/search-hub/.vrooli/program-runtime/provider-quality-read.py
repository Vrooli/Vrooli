"""search-hub.provider-quality-read v1 — one table of per-provider utilization, degradation, reachability, and latest eval outcome.

Contract: provider-quality-read.json.
Skill:    search-hub-improve §3 (sensors), §5 (routes filed against provider owners).

Phases: validate -> collect -> classify -> report. Read-only. Joins insights providers,
providers list, federation status, and the newest run of every registered suite into
one bounded table (one row per provider), so the improve skill can name the provider
and the measure that proves the defect in one filing.
"""

try:
    inputs
except NameError:
    inputs = {}
raw_window = inputs.get("insights_window", "7")
raw_group = inputs.get("provider_group", "")
raw_max_rows = inputs.get("max_rows", 5)
insights_window = str(raw_window or "7")
provider_group = str(raw_group or "")
cfg = {"max_rows": 5}  # parsed in validate; a bad value is invalid_input, not a crash before the envelope exists

envelope = {
    "program": "search-hub.provider-quality-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"insights_window": insights_window, "provider_group": provider_group, "max_rows": raw_max_rows},
    "signals": {
        "providers": [], "provider_count": 0, "degraded_over_band": [], "unreachable": [], "under_utilized": [],
        "suites_without_runs": [], "suites_below_full_pass": [], "window": {}, "fleet": {},
    },
    "errors": [], "evidence": [],
}
handles = {}
DEGRADED_BAND = 0.20  # the declared degraded_rate_max every provider descriptor carries


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


def safe(fn):
    try:
        return fn(), None
    except Exception as exc:
        return None, exc


def step_validate():
    try:
        cfg["max_rows"] = int(raw_max_rows)
    except (TypeError, ValueError):
        return fail("failed", "invalid_input", f"max_rows={raw_max_rows!r} is not an integer", "validate")
    envelope["inputs"]["max_rows"] = cfg["max_rows"]
    if not (1 <= cfg["max_rows"] <= 100):
        return fail("failed", "invalid_input", f"max_rows={cfg['max_rows']} outside 1..100", "validate")
    if not insights_window.strip():
        return fail("failed", "invalid_input", "insights_window must be a day count or a duration such as 15m", "validate")
    return "collect"


def step_collect():  # COLLECT · four reads concurrently, then one newest-run read per suite
    envelope["phase"] = "collect"
    try:
        ins, prov, fed, suites = gather(
            lambda: search_hub.insights.insights(window=insights_window, rows="providers"),
            lambda: search_hub.providers.list(rows="providers"),
            lambda: search_hub.federation.status(rows="providers"),
            lambda: search_hub.evals.list(),
        )
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    handles.update(ins=ins, prov=prov, fed=fed, suites=suites)
    # select() raises on an omitted key; map with .get so a suite without providerId cannot kill the program
    suite_rows = suites.map(lambda r: {"suiteId": r.get("suiteId"), "providerId": r.get("providerId")}).head(60)
    if provider_group:
        suite_rows = [s for s in suite_rows if str(s.get("providerId", "")).startswith(provider_group + ".") or s.get("providerId") == provider_group]
    handles["suite_rows"] = suite_rows
    calls = [(lambda sid=s.get("suiteId"): safe(lambda: search_hub.evals.runs(suite_id=sid, limit=1))) for s in suite_rows]
    handles["runs"] = list(gather(*calls)) if calls else []
    for _, err in handles["runs"]:
        if err is not None and classify_transport(err)[0] == "unavailable":
            return fail("unavailable", "scenario_unreachable", err, "collect")  # search-hub went away mid-collect: nothing is known
    return "classify"


def step_classify():  # CLASSIFY · one join keyed by providerId; deterministic thresholds only
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    h = handles
    ins_meta = h["ins"].meta() or {}
    sig["window"] = {"from": ins_meta.get("windowFrom"), "to": ins_meta.get("windowTo"), "insights_window": insights_window}
    sig["fleet"] = {k: ins_meta.get(k) for k in ("totalQueries", "zeroResultQueries", "zeroResultRate", "degradedQueries", "latencyP95Ms")}
    retire = {r.get("providerId"): r.get("reason") for r in (ins_meta.get("retirementCandidates") or []) if isinstance(r, dict)}

    ins_rows = {r.get("providerId"): r for r in h["ins"].head(100)}
    fed_rows = {r.get("providerId"): r for r in h["fed"].head(100)}
    prov_rows = h["prov"].head(100)
    if provider_group:
        prov_rows = [p for p in prov_rows if str(p.get("providerGroup", "")) == provider_group]

    latest = {}
    for s, (run, err) in zip(h["suite_rows"], h["runs"]):
        sid = s.get("suiteId")
        pid = s.get("providerId")
        if err is not None:
            _, klass = classify_transport(err)
            envelope["errors"].append({"class": klass, "detail": f"evals runs {sid}: {str(err)[:160]}", "where": "collect"})
            continue
        rows = run.head(1)
        if not rows:
            sig["suites_without_runs"].append(sid)
            continue
        r = rows[0]
        agg = r.get("aggregate") or {}
        entry = {"suite_id": sid, "run_id": r.get("runId"), "created_at": r.get("createdAt"), "tier": r.get("tier"), "tag": r.get("tag"),
                 "pass_rate": agg.get("passRate"), "routing_precision": agg.get("routingPrecision"), "retrieval_recall": agg.get("retrievalRecall"),
                 "met": agg.get("met"), "below": agg.get("below"), "cases": agg.get("cases"), "unavailable_cases": agg.get("unavailableCases")}
        if entry["pass_rate"] is not None and float(entry["pass_rate"]) < 1.0:
            sig["suites_below_full_pass"].append({"suite_id": sid, "pass_rate": round(float(entry["pass_rate"]), 3)})
        latest.setdefault(pid, []).append(entry)
        if len(envelope["evidence"]) < 6:   # bounded: the run ids are a sample, the counts carry the rest
            envelope["evidence"].append(f"search-hub://eval-run/{r.get('runId')}")

    table = []
    for p in prov_rows:
        pid = p.get("providerId")
        i = ins_rows.get(pid, {})
        f = fed_rows.get(pid, {})
        deg = i.get("degradationRate")
        reasons = [(d.get("reason"), d.get("count")) for d in (i.get("degradationReasons") or []) if isinstance(d, dict)]
        ev = latest.get(pid, [])
        row = {
            "provider_id": pid, "lifecycle": str(p.get("lifecycle", "")).replace("LIFECYCLE_", "").lower(),
            "reachable": f.get("reachable"), "circuit": f.get("circuitState"),
            "routed": i.get("timesRouted"), "hits": i.get("totalHits"), "degradation": None if deg is None else round(float(deg), 3),
            "reasons": reasons[:2],
            "p95_ms": i.get("latencyP95Ms"), "retire": retire.get(pid),
            "eval": [{"suite": e["suite_id"][:40], "pass": e["pass_rate"], "prec": e["routing_precision"], "at": str(e["created_at"] or "")[:10]} for e in ev[:1]],
        }
        table.append(row)
        if deg is not None and float(deg) > DEGRADED_BAND:
            sig["degraded_over_band"].append({"provider_id": pid, "degradation_rate": deg})
        unreachable_count = sum(int(c or 0) for reason, c in reasons if reason == "unreachable")
        if (f and f.get("reachable") is False) or unreachable_count:
            sig["unreachable"].append({"provider_id": pid, "reachability": f.get("reachability"), "now": f.get("reachable"), "unreachable_in_window": unreachable_count})
        if pid in retire:
            sig["under_utilized"].append({"provider_id": pid, "reason": retire[pid]})
    sig["provider_count"] = len(table)
    sig["providers"] = table
    sig["under_utilized"] = sig["under_utilized"][:6]
    sig["unreachable"] = sig["unreachable"][:6]
    envelope["evidence"] = envelope["evidence"][:4]
    envelope["status"] = "partial" if envelope["errors"] else "ok"
    return "report"


def step_report():  # REPORT · bounded: every list carries a count beside a five-row sample
    envelope["phase"] = "report"
    sig = envelope["signals"]
    for key in ("providers", "suites_below_full_pass", "degraded_over_band",
                "unreachable", "under_utilized", "suites_without_runs"):
        value = sig.get(key)
        if isinstance(value, list):
            if key != "providers":            # provider_count is already set from the full table
                sig[key + "_count"] = len(value)
            sig[key] = value[:3 if key != "providers" else cfg["max_rows"]]
    envelope["evidence"] = envelope["evidence"][:4]
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
