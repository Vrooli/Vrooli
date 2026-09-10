"""search-hub.research v1 — one federated project-research call at a named effort level.

Contract: research.json.
Skill:    search-hub (usage) — the [S3] leaf of the question-shape tree.

Phases: validate -> collect -> classify -> report. Read-only. One binding call:
search-hub/query/query with rows="ranked"; the per-provider `groups` field arrives in
meta() on the same call. Effort maps to selector, depth, and limit; explicit `types`
always override the classifier. Zero results and degraded legs are verdicts in
signals, never errors; unreachable is unavailable, never zero.

When the router's reranker leg is absent (`rerankerLeg` is "none" or missing) the
response carries no fused `ranked` rows and every hit sits in `groups[].hits`. The
program reads the groups and reports verdict `answered_degraded` with error class
`reranker_absent`; a reranker outage never reads as zero_result and never sets
escalate_to_web.
"""

inputs = program.inputs()
query = str(inputs.get("query", "") or "").strip()
effort = str(inputs.get("effort", "standard") or "standard").lower()
types = list(inputs.get("types", []) or [])
group = str(inputs.get("group", "") or "")
scope = str(inputs.get("scope", "") or "")

EFFORT = {  # effort -> (fan out to every provider, per-provider limit, rows returned)
    "fast": (False, 5, 5),
    "standard": (False, 10, 6),
    "deep": (True, 20, 6),
}

envelope = {
    "program": "search-hub.research", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"query": query, "effort": effort, "types": types, "group": group, "scope": scope},
    "signals": {
        "verdict": "unavailable", "ranked": [], "ranked_count": 0, "ranked_source": None,
        "corpora_searched": [], "corpora_count": 0, "providers_hit": {},
        "degraded": None, "routing_degrade_reason": None, "partial": None, "pending_providers": None,
        "latency_ms": None, "selector_leg": None, "reranker_leg": None, "ordered_by": None,
        "routing_explanation": None, "escalate_to_web": False,
    },
    "errors": [], "evidence": [],
}
handles = {}


fail = program.fail


def hit_row(r):
    return {"id": str(r.get("id", ""))[:80], "title": str(r.get("title", ""))[:60], "provider_id": r.get("providerId"),
            "type": r.get("type"), "path": str(r.get("path", ""))[:80], "score": round(float(r.get("score") or 0), 3),
            "rerank_score": round(float(r.get("rerankScore") or 0), 3), "snippet": str(r.get("snippet", ""))[:60]}


def step_validate():
    if not query:
        return fail("failed", "invalid_input", "query is required", "validate")
    if effort not in EFFORT:
        return fail("failed", "invalid_input", f"effort={effort!r} not in {tuple(EFFORT)}", "validate")
    if any(not isinstance(t, str) or not t for t in types):
        return fail("failed", "invalid_input", "types must be non-empty strings", "validate")
    return "collect"


def step_collect():  # COLLECT · one governed read
    envelope["phase"] = "collect"
    fan_out, limit, _ = EFFORT[effort]
    kwargs = {"text": query, "limit": limit, "explain": True, "rows": "ranked"}
    if types:
        kwargs["type"] = types          # explicit tokens override the classifier
    elif fan_out:
        kwargs["all"] = True
    if group:
        kwargs["group"] = group
    if scope:
        kwargs["scope"] = scope
    try:
        handles["q"] = search_hub.query.query(**kwargs)
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · verdict table over the response meta; bounded head
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    q = handles["q"]
    meta = q.meta() or {}
    _, _, rows_n = EFFORT[effort]
    corpora = list(meta.get("corporaSearched") or [])
    sig["corpora_searched"] = corpora[:6]
    sig["corpora_count"] = len(corpora)
    sig["degraded"] = bool(meta.get("degraded", False))
    sig["routing_degrade_reason"] = meta.get("routingDegradeReason") or None
    sig["partial"] = bool(meta.get("partial", False))
    sig["pending_providers"] = meta.get("pendingProviders")
    lat = meta.get("latencyMs")  # int64 arrives as a protojson string
    sig["latency_ms"] = int(lat) if lat not in (None, "") else None
    sig["selector_leg"] = meta.get("selectorLeg")
    sig["reranker_leg"] = meta.get("rerankerLeg")
    sig["ordered_by"] = meta.get("orderedBy")
    expl = meta.get("routingExplanation")
    sig["routing_explanation"] = str(expl)[:400] if expl else None

    ranked_count = q.count()
    rows = q.head(rows_n)
    sig["ranked_source"] = "ranked"
    reranker_absent = False
    if ranked_count == 0:
        # No fused rows: read the per-provider groups from the same response before calling it zero.
        group_hits = []
        for g in (meta.get("groups") or []):
            if isinstance(g, dict):
                group_hits.extend(h for h in (g.get("hits") or []) if isinstance(h, dict))
        if group_hits:
            reranker_absent = str(sig["reranker_leg"] or "none") == "none"
            ranked_count = len(group_hits)
            rows = sorted(group_hits, key=lambda h: float(h.get("score") or 0), reverse=True)[:rows_n]
            sig["ranked_source"] = "groups"
    sig["ranked"] = [hit_row(r) for r in rows]
    sig["ranked_count"] = ranked_count
    hits = {}
    for r in rows:
        pid = r.get("providerId")
        hits[pid] = hits.get(pid, 0) + 1
    sig["providers_hit"] = hits
    envelope["evidence"] = [r["id"] for r in sig["ranked"][:5] if r.get("id")]

    if sig["ranked_count"] == 0 and sig["corpora_count"] == 0:
        sig["verdict"] = "no_provider_selected"
    elif sig["ranked_count"] == 0:
        sig["verdict"] = "zero_result"
    elif sig["degraded"] or sig["partial"] or reranker_absent:
        sig["verdict"] = "answered_degraded"
    else:
        sig["verdict"] = "answered"
    sig["escalate_to_web"] = sig["verdict"] in ("zero_result", "no_provider_selected") and "web" not in types
    envelope["status"] = "partial" if sig["verdict"] == "answered_degraded" else "ok"
    if reranker_absent:
        envelope["errors"].append({"class": "reranker_absent", "detail": "no reranker leg; hits read from groups, ordered by provider score, not fused", "where": "classify"})
    if sig["degraded"] or sig["partial"]:
        envelope["errors"].append({"class": "degraded_leg", "detail": str(sig["routing_degrade_reason"] or "partial: a provider leg did not answer")[:120], "where": "classify"})
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
