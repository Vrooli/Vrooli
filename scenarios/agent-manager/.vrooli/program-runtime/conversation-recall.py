"""agent-manager.conversation-recall v1 — bounded incomplete-memory recall.

Contract: conversation-recall.json. Skill: agent-manager §4.1.
Phases: validate -> collect -> classify -> report. Read-only, deterministic, content-minimized.
"""

import datetime
import json

try:
    inputs
except NameError:
    inputs = {}

clues = inputs.get("clues", [])
project_scope = str(inputs.get("project_scope", "")).strip()
harness = str(inputs.get("harness", "")).strip()
role = str(inputs.get("role", "")).strip()
occurred_after = str(inputs.get("occurred_after", "")).strip()
occurred_before = str(inputs.get("occurred_before", "")).strip()
policy = str(inputs.get("policy", "both")).strip()
per_query_limit = int(inputs.get("per_query_limit", 5))
max_candidates = int(inputs.get("max_candidates", 10))
context_candidates = int(inputs.get("context_candidates", 2))
context_before = int(inputs.get("context_before", 2))
context_after = int(inputs.get("context_after", 2))
max_context_bytes = int(inputs.get("max_context_bytes", 6000))
output_mode = str(inputs.get("output_mode", "metadata")).strip()

envelope = {
    "program": "agent-manager.conversation-recall", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {
        "clue_count": len(clues) if isinstance(clues, list) else None,
        "filters": {"project_scope": bool(project_scope), "harness": bool(harness), "role": bool(role),
                    "occurred_after": bool(occurred_after), "occurred_before": bool(occurred_before)},
        "policy": policy, "per_query_limit": per_query_limit, "max_candidates": max_candidates,
        "context_candidates": context_candidates, "context_before": context_before,
        "context_after": context_after, "max_context_bytes": max_context_bytes, "output_mode": output_mode,
    },
    "signals": {
        "query_variants": 0, "legs_planned": 0, "legs_succeeded": 0, "legs_failed": 0,
        "candidate_count": 0, "candidates": [], "sibling_provider_count": 0,
        "degradations": [], "next_actions": [], "content_bytes": 0,
    },
    "errors": [], "evidence": [],
}
state_data = {"variants": [], "calls": [], "results": [], "candidates": {}}


def fail(status, klass, detail, where):
    envelope["status"] = status
    # Never include exception text: a transport may echo a raw query.
    envelope["errors"].append({"class": klass, "detail": klass, "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc
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


def parse_time(value):
    if not value:
        return None
    try:
        return datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return False


def trim_bytes(value, budget):
    data = str(value or "").encode("utf-8")
    if len(data) <= budget:
        return data.decode("utf-8"), len(data)
    return data[:budget].decode("utf-8", errors="ignore"), budget


def step_validate():
    if not isinstance(clues, list) or not 1 <= len(clues) <= 3:
        return fail("failed", "invalid_input", "invalid_input", "validate")
    normalized = []
    for clue in clues:
        if not isinstance(clue, str):
            return fail("failed", "invalid_input", "invalid_input", "validate")
        value = " ".join(clue.split())
        if not value or len(value.encode("utf-8")) > 512:
            return fail("failed", "invalid_input", "invalid_input", "validate")
        if value not in normalized:
            normalized.append(value)
    if sum(len(item.encode("utf-8")) for item in normalized) > 1536:
        return fail("failed", "invalid_input", "invalid_input", "validate")
    if policy not in ("direct", "federated", "both") or output_mode not in ("metadata", "snippets", "context"):
        return fail("failed", "invalid_input", "invalid_input", "validate")
    bounds = ((per_query_limit, 1, 10), (max_candidates, 1, 20), (context_candidates, 0, 3),
              (context_before, 0, 5), (context_after, 0, 5), (max_context_bytes, 0, 12000))
    if any(value < low or value > high for value, low, high in bounds):
        return fail("failed", "invalid_input", "invalid_input", "validate")
    after, before = parse_time(occurred_after), parse_time(occurred_before)
    if after is False or before is False or (after and before and after > before):
        return fail("failed", "invalid_input", "invalid_input", "validate")
    variants = normalized[:]
    combined = " ".join(normalized)
    if combined not in variants:
        variants.append(combined)
    state_data["variants"] = variants[:4]
    envelope["signals"]["query_variants"] = len(state_data["variants"])
    return "collect"


def direct_search(query, mode):
    modes = {
        "hybrid": "CONVERSATION_SEARCH_MODE_HYBRID",
        "text": "CONVERSATION_SEARCH_MODE_TEXT",
    }
    args = {"query": query, "mode": modes[mode], "sort": "CONVERSATION_SEARCH_SORT_RELEVANCE", "page_size": per_query_limit}
    if project_scope:
        args["project_scopes"] = project_scope
    if harness:
        args["harnesses"] = harness
    if role:
        args["roles"] = role
    if occurred_after:
        args["occurred_after"] = occurred_after
    if occurred_before:
        args["occurred_before"] = occurred_before
    return agent_manager.conversation.search(**args)


def federated_search(query):
    return search_hub.query.query(text=query, limit=per_query_limit, explain=True, rows="groups")


def step_collect():
    envelope["phase"] = "collect"
    calls = []
    for query_index, query in enumerate(state_data["variants"]):
        if policy in ("direct", "both"):
            for mode in ("hybrid", "text"):
                calls.append({"surface": "direct", "query_index": query_index, "mode": mode,
                              "call": lambda q=query, m=mode: direct_search(q, m)})
        if policy in ("federated", "both"):
            calls.append({"surface": "federated", "query_index": query_index, "mode": "federated",
                          "call": lambda q=query: federated_search(q)})
    envelope["signals"]["legs_planned"] = len(calls)
    state_data["calls"] = calls
    results = gather(*[item["call"] for item in calls])
    state_data["results"] = list(zip(calls, results))
    return "classify"


def add_degradations(meta):
    known = {(item["reason"], item["leg"]) for item in envelope["signals"]["degradations"]}
    for degradation in (meta or {}).get("degradations", []) or []:
        item = {"reason": degradation.get("reason") or "unknown", "leg": degradation.get("leg") or "unknown",
                "retryable": bool(degradation.get("retryable"))}
        key = (item["reason"], item["leg"])
        if key not in known:
            envelope["signals"]["degradations"].append(item)
            known.add(key)


def merge_hit(hit, surface, query_index, mode, native_rank):
    metadata = hit.get("metadata") or {}
    stable_id = hit.get("stableHitId") or hit.get("id") or ""
    run_id = hit.get("runId") or metadata.get("run_id") or ""
    event_id = hit.get("eventId") or metadata.get("event_id") or ""
    if not stable_id:
        stable_id = run_id + ":" + event_id
    if not stable_id:
        return
    run = hit.get("run") or {}
    provenance = hit.get("provenance") or metadata.get("provenance") or {}
    candidate = state_data["candidates"].get(stable_id)
    if candidate is None:
        candidate = {
            "stable_hit_id": stable_id, "run_id": run_id, "event_id": event_id,
            "run_label": run.get("label") or hit.get("title") or "", "role": hit.get("role") or metadata.get("role") or "",
            "occurred_at": hit.get("occurredAt") or metadata.get("occurred_at") or "",
            "deep_link": hit.get("deepLink") or hit.get("path") or "", "weak": bool(hit.get("weak")),
            "provenance": {
                "harness": provenance.get("harness") or "",
                "source_session_id": provenance.get("sourceSessionId") or provenance.get("source_session_id") or "",
                "project_scope": provenance.get("projectScope") or provenance.get("project_scope") or "",
                "provider_origin": provenance.get("providerOrigin") or provenance.get("provider_origin") or "",
            },
            "contributing": [], "support_count": 0, "best_native_rank": native_rank,
            "snippet": "", "context": [], "context_truncated": False,
        }
        state_data["candidates"][stable_id] = candidate
    contribution = {"surface": surface, "query_ordinal": query_index + 1, "mode": mode,
                    "provider": hit.get("providerId") or "agent-manager.runs", "native_rank": native_rank}
    if contribution not in candidate["contributing"]:
        candidate["contributing"].append(contribution)
        candidate["support_count"] += 1
    candidate["best_native_rank"] = min(candidate["best_native_rank"], native_rank)
    if output_mode in ("snippets", "context") and not candidate["snippet"]:
        remaining = max(0, max_context_bytes - envelope["signals"]["content_bytes"])
        candidate["snippet"], used = trim_bytes(hit.get("snippet"), min(remaining, 800))
        envelope["signals"]["content_bytes"] += used


def collect_context(candidates):
    if output_mode != "context" or context_candidates == 0:
        return
    selected = [item for item in candidates if item.get("stable_hit_id")][:context_candidates]
    results = gather(*[lambda item=item: agent_manager.conversation.context(
        stable_hit_id=item["stable_hit_id"], before=context_before, after=context_after) for item in selected])
    for candidate, result in zip(selected, results):
        if isinstance(result, Exception):
            status, klass = classify_transport(result)
            envelope["errors"].append({"class": klass, "detail": klass, "where": "collect:context"})
            continue
        meta = result.meta() or {}
        add_degradations(meta)
        remaining = max(0, max_context_bytes - envelope["signals"]["content_bytes"])
        for event in result.head(context_before + context_after + 1):
            if remaining <= 0:
                candidate["context_truncated"] = True
                break
            content, used = trim_bytes(event.get("boundedContent"), remaining)
            candidate["context"].append({"event_id": event.get("eventId") or "", "event_sequence": event.get("eventSequence", 0),
                                         "role": event.get("role") or "", "occurred_at": event.get("occurredAt") or "",
                                         "content_class": event.get("contentClass") or "", "matched": bool(event.get("matched")),
                                         "content": content})
            remaining -= used
            envelope["signals"]["content_bytes"] += used
        candidate["context_truncated"] = bool(candidate["context_truncated"] or meta.get("truncated"))


def step_classify():
    envelope["phase"] = "classify"
    for call, result in state_data["results"]:
        label = f"{call['surface']}:q{call['query_index'] + 1}:{call['mode']}"
        if isinstance(result, Exception):
            status, klass = classify_transport(result)
            envelope["signals"]["legs_failed"] += 1
            envelope["errors"].append({"class": klass, "detail": klass, "where": label})
            continue
        envelope["signals"]["legs_succeeded"] += 1
        meta = result.meta() or {}
        add_degradations(meta)
        rows = result.head(20)
        if call["surface"] == "direct":
            for index, hit in enumerate(rows[:per_query_limit]):
                merge_hit(hit, "direct", call["query_index"], call["mode"], index + 1)
            envelope["evidence"].append(f"agent-manager/conversation/search:{label}:hits={min(len(rows), per_query_limit)}")
        else:
            owned = 0
            sibling = 0
            for group in rows:
                provider = group.get("providerId") or ""
                hits = group.get("hits") or []
                if provider == "agent-manager.runs":
                    for index, hit in enumerate(hits[:per_query_limit]):
                        merge_hit(hit, "federated", call["query_index"], call["mode"], index + 1)
                        owned += 1
                else:
                    sibling += len(hits)
            envelope["signals"]["sibling_provider_count"] += sibling
            envelope["evidence"].append(f"search-hub/query/query:{label}:agent_hits={owned}:sibling_hits={sibling}")
    candidates = sorted(state_data["candidates"].values(),
                        key=lambda item: (-item["support_count"], item["best_native_rank"], item["stable_hit_id"]))[:max_candidates]
    collect_context(candidates)
    if output_mode != "context":
        for candidate in candidates:
            candidate.pop("context", None)
            candidate.pop("context_truncated", None)
    if output_mode == "metadata":
        for candidate in candidates:
            candidate.pop("snippet", None)
    envelope["signals"]["candidates"] = candidates
    envelope["signals"]["candidate_count"] = len(candidates)
    if not candidates:
        envelope["signals"]["next_actions"].append("narrow_or_replace_one_clue")
    elif len(candidates) > 1 and candidates[0]["support_count"] == candidates[1]["support_count"]:
        envelope["signals"]["next_actions"].append("compare_provenance_or_add_filter")
    if envelope["signals"]["degradations"]:
        envelope["signals"]["next_actions"].append("treat_degraded_legs_as_unsearched")
    if envelope["signals"]["legs_failed"]:
        envelope["signals"]["next_actions"].append("retry_only_failed_surface_after_status_check")
    succeeded = envelope["signals"]["legs_succeeded"]
    failed = envelope["signals"]["legs_failed"]
    if succeeded:
        envelope["status"] = "partial" if (failed or envelope["errors"]) else "ok"
    elif failed and envelope["errors"] and all(item["class"] == "scenario_unreachable" for item in envelope["errors"]):
        envelope["status"] = "unavailable"
    else:
        envelope["status"] = "failed"
    envelope["evidence"].extend(["stable-hit:" + item["stable_hit_id"] for item in candidates])
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": "kernel_runtime", "where": envelope.get("phase") or state})
        state = "report"
