"""Federated project research and optional bounded skill discovery."""

inputs = program.inputs()
query = str(inputs.get("query", "") or "").strip()
effort = str(inputs.get("effort", "standard") or "standard").lower()
types = list(inputs.get("types", []) or [])
group = str(inputs.get("group", "") or "")
scope = str(inputs.get("scope", "") or "")
include_skill = inputs.get("include_skill", False)
include_discover = inputs.get("include_discover", False)
capture_note = inputs.get("capture_note", False)
repeat_after_status = inputs.get("repeat_after_status", False)

EFFORT = {  # effort -> (fan out to every provider, per-provider limit, rows returned)
    "fast": (False, 5, 5),
    "standard": (False, 10, 6),
    "deep": (True, 20, 6),
}

envelope = {
    "program": "search-hub.research", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"query": query, "effort": effort, "types": types, "group": group, "scope": scope,
               "include_skill": include_skill, "include_discover": include_discover,
               "capture_note": capture_note, "repeat_after_status": repeat_after_status},
    "signals": {
        "verdict": "unavailable", "ranked": [], "ranked_count": 0, "ranked_source": None,
        "corpora_searched": [], "corpora_count": 0, "providers_hit": {},
        "degraded": None, "routing_degrade_reason": None, "partial": None, "pending_providers": None,
        "latency_ms": None, "selector_leg": None, "reranker_leg": None, "ordered_by": None,
        "routing_explanation": None, "escalate_to_web": False, "skill_lookup": None,
        "skill_discovery": None, "memory_capture": None, "status_check": None,
        "first_query": None,
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
    if any(not isinstance(flag, bool) for flag in (include_skill, include_discover, capture_note, repeat_after_status)):
        return fail("failed", "invalid_input", "workflow options must be booleans", "validate")
    if capture_note and not include_discover:
        return fail("failed", "invalid_input", "capture_note requires include_discover", "validate")
    if include_skill and types and types != ["skill"]:
        return fail("failed", "invalid_input", "include_skill requires types=[skill]", "validate")
    return "collect"


def step_collect():  # COLLECT · one governed read
    envelope["phase"] = "collect"
    fan_out, limit, _ = EFFORT[effort]
    kwargs = {"text": query, "limit": limit, "explain": True, "rows": "ranked"}
    if include_skill:
        kwargs["type"] = ["skill"]
    elif types:
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
    if repeat_after_status:
        handles["first_q"] = handles["q"]
        try:
            status_rows = vrooli.scenario.status(name="search-hub").materialize(1)
            status_row = status_rows[0] if status_rows else {}
            if isinstance(status_row, dict) and isinstance(status_row.get("scenario"), dict):
                status_row = status_row["scenario"]
            lifecycle_status = str(status_row.get("status") or "") if isinstance(status_row, dict) else ""
            envelope["signals"]["status_check"] = {"status": lifecycle_status or "unknown"}
        except Exception as exc:
            _, klass = program.classify(exc)
            envelope["signals"]["status_check"] = {"status": "unavailable"}
            envelope["errors"].append({"class": klass, "detail": "Search Hub status unavailable; repeated query skipped", "where": "status_check"})
            envelope["status"] = "unavailable"
            return "report"
        if lifecycle_status != "running":
            envelope["errors"].append({"class": "scenario_unreachable", "detail": "Search Hub is not running; repeated query skipped", "where": "status_check"})
            envelope["status"] = "unavailable"
            return "report"
        try:
            handles["q"] = search_hub.query.query(**kwargs)
            envelope["signals"]["first_query"] = {
                "ranked_count": handles["first_q"].count(),
                "corpora_count": len(list(handles["first_q"].meta().get("corporaSearched") or [])),
            }
        except Exception as exc:
            _, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": "repeated Search Hub query failed", "where": "repeat_query"})
            envelope["status"] = "unavailable"
            return "report"
    if include_discover:
        try:
            handles["discover"] = prompt_manager.discover.discover(
                queries=[query], complexity="moderate", limit=5, type="skill", rows="results")
        except Exception as exc:
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": str(exc)[:120], "where": "skill_discover"})
            envelope["signals"]["skill_discovery"] = {"status": "unavailable"}
            if envelope["status"] == "ok":
                envelope["status"] = "partial"
    return "classify"


def step_classify():  # CLASSIFY · verdict table over the response meta; bounded head
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    include_skill = envelope.get("inputs", {}).get("include_skill", False)
    include_discover = envelope.get("inputs", {}).get("include_discover", False)
    sig.setdefault("skill_lookup", None)
    q = handles["q"]
    meta = q.meta() or {}
    groups = [g for g in (meta.get("groups") or []) if isinstance(g, dict)]
    _, _, rows_n = EFFORT[effort]
    corpora = list(meta.get("corporaSearched") or [])
    sig["corpora_searched"] = corpora[:6]
    sig["corpora_count"] = len(corpora)
    sig["degraded"] = bool(meta.get("degraded", False))
    sig["routing_degrade_reason"] = meta.get("routingDegradeReason") or None
    if not sig["routing_degrade_reason"] and sig["degraded"]:
        provider_notes = []
        for group in groups:
            if not group.get("degraded"):
                continue
            provider_id = str(group.get("providerId") or "").strip()
            note = str(group.get("note") or "").strip()
            if provider_id and note:
                provider_notes.append(f"{provider_id}: {note}")
        if provider_notes:
            sig["routing_degrade_reason"] = "; ".join(provider_notes)[:400]
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
        for g in groups:
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

    if sig["ranked_count"] == 0 and (sig["degraded"] or sig["partial"]):
        # A searched corpus is not a completed negative when one of its legs did not answer.
        sig["verdict"] = "unavailable"
    elif sig["ranked_count"] == 0 and sig["corpora_count"] == 0:
        sig["verdict"] = "no_provider_selected"
    elif sig["ranked_count"] == 0:
        sig["verdict"] = "zero_result"
    elif sig["degraded"] or sig["partial"] or reranker_absent:
        sig["verdict"] = "answered_degraded"
    else:
        sig["verdict"] = "answered"
    sig["escalate_to_web"] = sig["verdict"] in ("zero_result", "no_provider_selected") and "web" not in types

    if sig["verdict"] == "answered_degraded":
        envelope["status"] = "partial"
    elif sig["verdict"] == "unavailable":
        envelope["status"] = "unavailable"
    else:
        envelope["status"] = "ok"
    if include_skill and sig["skill_lookup"] and sig["skill_lookup"].get("status") in ("unavailable", "partial") and envelope["status"] == "ok":
        envelope["status"] = "partial"

    if include_skill and sig["skill_lookup"] is None:
        source = "prompt-manager.discover" if include_discover else "search-hub"
        if include_discover:
            candidates = handles.get("discover")
            candidates = candidates.filter(lambda r: r.get("type") == "skill") if candidates is not None else None
            candidates = candidates.head(1) if candidates is not None else []
        else:
            candidates = [r for r in rows if r.get("type") == "skill" and r.get("providerId") == "prompt-manager.skill"][:1]
        skill_id = str(candidates[0].get("id") or "") if candidates else ""
        if not skill_id:
            sig["skill_lookup"] = {"status": "not_found", "source": source, "id": "", "content": "", "truncated": False}
        else:
            try:
                skills = prompt_manager.skill.read(identifiers=[skill_id], allow_missing=True, rows="skills")
                skill = next((r for r in skills.head(1) if r.get("id") == skill_id), None)
                content = skill.get("content") if isinstance(skill, dict) else None
                if not isinstance(content, str):
                    sig["skill_lookup"] = {"status": "not_found", "source": source, "id": skill_id, "content": "", "truncated": False}
                else:
                    raw = content.encode("utf-8")
                    truncated = len(raw) > 8500
                    content = raw[:8500].decode("utf-8", "ignore") if truncated else content
                    sig["skill_lookup"] = {"status": "partial" if truncated else "ok", "source": source,
                                           "id": skill_id, "content": content, "truncated": truncated}
                    if truncated:
                        envelope["status"] = "partial"
            except Exception as exc:
                status, klass = program.classify(exc)
                envelope["errors"].append({"class": klass, "detail": str(exc)[:120], "where": "skill_read"})
                sig["skill_lookup"] = {"status": "unavailable", "source": source, "id": skill_id, "content": "", "truncated": False}
                if envelope["status"] in ("ok", "partial"):
                    envelope["status"] = "partial"
    if include_discover and handles.get("discover") is not None:
        discovery = handles["discover"]
        envelope["signals"]["skill_discovery"] = {"status": "ok", "matched_count": discovery.count()}
    if capture_note:
        discovery = handles.get("discover")
        if discovery is None or envelope["status"] == "unavailable":
            envelope["signals"]["memory_capture"] = {"status": "not_written"}
        else:
            note_body = f"Search Hub discovery outcome: verdict={sig['verdict']}; results={sig['ranked_count']}; discovered={discovery.count()}."
            try:
                receipt = vrooli_memory.journal.note(
                    body=note_body, scope="search-hub-usage", kind="query-record",
                    trigger="task-skill discovery", approach="search-hub.research then prompt-manager.discover",
                    evidence=f"verdict={sig['verdict']};results={sig['ranked_count']};discovered={discovery.count()}",
                    outcome=str(sig["verdict"])[:40],
                )
                entry = (receipt.meta() or {}).get("entry") or {}
                entry_id = entry.get("id") or entry.get("entryId") if isinstance(entry, dict) else ""
                envelope["signals"]["memory_capture"] = {"status": "written", "entry_ref": str(entry_id)[:80]}
            except Exception as exc:
                _, klass = program.classify(exc)
                envelope["signals"]["memory_capture"] = {"status": "failed"}
                envelope["errors"].append({"class": klass, "detail": "memory note failed; search result retained", "where": "memory_capture"})
                if envelope["status"] == "ok":
                    envelope["status"] = "partial"
    if reranker_absent:
        envelope["errors"].append({"class": "reranker_absent", "detail": "no reranker leg; hits read from groups, ordered by provider score, not fused", "where": "classify"})
    if sig["degraded"] or sig["partial"]:
        envelope["errors"].append({"class": "degraded_leg", "detail": str(sig["routing_degrade_reason"] or "partial: a provider leg did not answer")[:120], "where": "classify"})
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    program.report(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
