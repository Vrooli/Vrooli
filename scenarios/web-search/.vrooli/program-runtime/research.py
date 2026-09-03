"""web-search.research v1 — answer one question at one ladder rung, stored findings first.

Contract: research.json (inputs, invariants, bindings, outputs).
Skill:    web-search (usage) — the [S3] leaf of the ladder tree.

Phases: validate -> collect -> classify -> act -> report.
Order of evidence: the findings ledger is read first on every effort; the live
rung requested by `effort` runs alongside it. L3 has no governed binding
(`research l3` is run-ineligible) and is reported unavailable, never emulated.
`mark_used` records `findings use` for every strong stored hit that answered.
"""

try:
    inputs
except NameError:
    inputs = {}
query = str(inputs.get("query", "") or "").strip()
effort = str(inputs.get("effort", "l1") or "l1").lower()
limit = int(inputs.get("limit", 5))
top_n = int(inputs.get("top_n", 3))
mark_used = bool(inputs.get("mark_used", False))
capture_findings = bool(inputs.get("capture", False))

envelope = {
    "program": "web-search.research", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"query": query, "effort": effort, "limit": limit, "top_n": top_n, "mark_used": mark_used, "capture": capture_findings},
    "signals": {
        "answer_kind": "none", "confidence": None, "citations": [],
        "stored_hits": [], "stored_hit_count": 0, "strong_hit_count": 0,
        "live_result_count": 0, "cached": None, "degraded": None, "degraded_reason": None,
        "abstained": None, "captured_finding_ids": [], "marked_used": [], "live_rung": None,
    },
    "errors": [], "evidence": [],
}
handles = {}
VALID_EFFORT = ("l0", "l1", "l2", "l3")


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:160], "where": where})
    return "report"


def note(status, klass, detail, where):
    """Record a non-fatal problem and keep going; a problem after classify downgrades ok to partial."""
    envelope["errors"].append({"class": klass, "detail": str(detail)[:160], "where": where})
    if envelope["status"] == "ok":
        envelope["status"] = "partial"


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


def citation_rows(items):
    out = []
    for c in items or []:
        if isinstance(c, dict):
            out.append({"url": c.get("url"), "title": c.get("title")})
    return out[:8]


USABLE_CONFIDENCE = 0.5     # web-search usage skill §4: the ledger-answers gate
USABLE_AGE_DAYS = 180       # one decay half-life


def age_days(ts):
    """Days since an RFC3339 retrieval date, or None when the response carries none."""
    if not ts:
        return None
    try:
        import datetime
        t = datetime.datetime.fromisoformat(str(ts).replace("Z", "+00:00"))
        return (datetime.datetime.now(datetime.timezone.utc) - t).days
    except Exception:
        return None


def step_validate():
    if not query:
        return fail("failed", "invalid_input", "query is required", "validate")
    if effort not in VALID_EFFORT:
        return fail("failed", "invalid_input", f"effort={effort!r} not in {VALID_EFFORT}", "validate")
    if not (1 <= limit <= 50):
        return fail("failed", "invalid_input", f"limit={limit} outside 1..50", "validate")
    if not (1 <= top_n <= 10):
        return fail("failed", "invalid_input", f"top_n={top_n} outside 1..10", "validate")
    return "collect"


def live_call():
    """The one live binding call this effort permits, run in act. L3 has no governed binding."""
    if effort == "l0":
        return web_search.search.search(query=query, limit=limit, rows="results")
    if effort == "l1":
        return web_search.search.search(query=query, limit=limit, synthesis=True, rows="results")
    if effort == "l2":
        return web_search.research.l2(query=query, top_n=top_n, capture=capture_findings, rows="excerpts")
    return None


def step_collect():  # COLLECT · the ledger read only; reads never reach the live web
    envelope["phase"] = "collect"
    try:
        handles["stored"] = web_search.findings.search(query=query, limit=limit)
        handles["stored"].count()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    return "act_live"


def step_act_live():  # ACT · the live rung. l2 with capture=true persists findings, so it is never a collect.
    envelope["phase"] = "act"
    if effort == "l3":
        # A capability with no governed binding is failed, never unavailable (program-contracts.md).
        handles["live"] = None
        note("failed", "no_governed_binding",
             "research l3 has no governed binding; run `web-search research l3 <query>` from a shell and read the result there",
             "act")
        return "classify"
    try:
        handles["live"] = live_call()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act")
    return "classify"


def step_classify():  # CLASSIFY · deterministic; bounded heads only
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    hits = handles["stored"].head(limit)
    rows = []
    strong = []
    for h in hits:
        f = h.get("finding", {}) or {}
        row = {"id": f.get("id"), "claim": str(f.get("claim", ""))[:140], "status": str(f.get("status", "")).replace("FINDING_STATUS_", "").lower(),
               "confidence": f.get("confidence"), "score": round(float(h.get("score") or 0), 3), "weak": bool(h.get("weak", False)),
               "retrieval_date": f.get("retrievalDate"), "age_days": age_days(f.get("retrievalDate"))}
        # The usage skill's ledger-answers gate, applied here so a use is never recorded for a
        # finding the skill would not answer from: active, not weak, confidence >= 0.5, <= 180 days.
        row["usable"] = (row["status"] == "active" and not row["weak"]
                         and float(row["confidence"] or 0) >= USABLE_CONFIDENCE
                         and (row["age_days"] is None or row["age_days"] <= USABLE_AGE_DAYS))
        rows.append(row)
        if row["usable"]:
            strong.append(row)
    sig["stored_hits"] = rows[:10]  # bounded output: counts cover every hit, the envelope carries at most ten
    sig["stored_hit_count"] = len(rows)
    sig["strong_hit_count"] = len(strong)
    envelope["evidence"].extend([r["id"] for r in rows if r.get("id")])

    live = handles.get("live")
    if live is not None:
        meta = live.meta() or {}
        sig["live_rung"] = effort
        if effort in ("l0", "l1"):
            sig["live_result_count"] = live.count()
            sig["cached"] = bool(meta.get("cached", False))
            sig["degraded"] = bool(meta.get("degraded", False))
            sig["degraded_reason"] = meta.get("degradedReason") or None
            synth = meta.get("synthesis") or {}
            if effort == "l1" and synth:
                sig["abstained"] = bool(synth.get("abstained", False))
                sig["citations"] = citation_rows(synth.get("citations"))
            if sig["degraded"]:
                note("partial", "rate_limited" if "governor" in str(sig["degraded_reason"]).lower() or "rate" in str(sig["degraded_reason"]).lower() else "upstream_degraded",
                     sig["degraded_reason"], "classify")
        elif effort == "l2":
            # RunL2Response.brief carries no citations; the excerpts rows (url, title, excerpt) selected by rows="excerpts" are the citations.
            sig["abstained"] = bool(meta.get("abstained", False))
            sig["citations"] = [{"url": e.get("url"), "title": e.get("title")} for e in live.head(8) if isinstance(e, dict) and e.get("url")]
            sig["live_result_count"] = live.count()
            sig["captured_finding_ids"] = list(meta.get("capturedFindingIds") or [])[:20]
            envelope["evidence"].extend(sig["captured_finding_ids"])

    # Answer kind and confidence: a strong stored finding outranks a synthesis; a synthesis outranks raw hits.
    if strong:
        sig["answer_kind"] = "stored_finding"
        sig["confidence"] = max(float(r.get("confidence") or 0.0) for r in strong)
        if not sig["citations"]:
            sig["citations"] = [{"finding_id": r["id"]} for r in strong[:5]]
    elif sig["citations"] and sig["abstained"] is False:
        sig["answer_kind"] = "cited_synthesis"
        sig["confidence"] = None
    elif sig["live_result_count"]:
        sig["answer_kind"] = "raw_hits"
        sig["confidence"] = None
    else:
        sig["answer_kind"] = "none"
    # status is settled here, once; act may only downgrade it through note()
    envelope["status"] = "partial" if envelope["errors"] else "ok"
    return "act" if (mark_used and strong) else "report"


def step_act():  # ACT · declared write: findings use, once per strong hit, in plan order
    envelope["phase"] = "act"
    for r in envelope["signals"]["stored_hits"]:
        if not r.get("usable") or not r.get("id"):
            continue   # disputed, stale, low-confidence and weak hits are read but never marked used
        try:
            web_search.findings.use(id=r["id"])
            envelope["signals"]["marked_used"].append(r["id"])
        except Exception as exc:
            status, klass = classify_transport(exc)
            note(status, klass, exc, "act")
            break  # a refusal stops the program; no retry
    return "report"


def step_report():  # REPORT · one envelope, every path
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "act_live": step_act_live, "classify": step_classify, "act": step_act, "report": step_report}
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
