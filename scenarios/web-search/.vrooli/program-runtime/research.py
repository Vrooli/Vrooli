import json

"""One bounded evidence-policy call; web-search owns freshness and answer truth."""
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "web-search.research", "version": "2", "status": "failed", "phase": "validate", "signals": {}, "errors": [], "evidence": []}
try:
    allowed = {"query", "effort", "max_age_seconds", "source_domains", "minimum_sources", "top_n", "capture", "finding_id"}
    if not isinstance(inputs, dict) or set(inputs) - allowed or not str(inputs.get("query", "")).strip():
        raise ValueError("Supply query and declared evidence-policy inputs only")
    envelope["phase"] = "act"
    result = web_search.research.answer(**inputs, rows="results")
    meta = result.meta() or {}
    envelope["status"] = meta.get("status", "failed")
    envelope["signals"] = {
        "answer_kind": meta.get("answerKind", "none"),
        "brief": meta.get("brief", {}),
        "results": result.head(10),
        "finding_ids": meta.get("findingIds", []),
        "captured_finding_ids": meta.get("capturedFindingIds", []),
        "abstained": meta.get("abstained", False),
        "reason": meta.get("reason", ""),
        "checked_at": meta.get("checkedAt"),
        "live_calls": meta.get("liveCalls", 0),
        "cached": meta.get("cached", False),
        "gaps": meta.get("gaps", []),
    }
    envelope["evidence"] = list(meta.get("findingIds", [])) + list(meta.get("capturedFindingIds", []))
except Exception as exc:
    message = str(exc)
    klass = "invalid_input" if isinstance(exc, ValueError) else "binding_error"
    if "unreachable" in message or "connection refused" in message or "unavailable" in message:
        klass = "scenario_unreachable"
        envelope["status"] = "unavailable"
    elif "grant" in message or "not run eligible" in message:
        klass = "no_grant"
        envelope["status"] = "refused"
    envelope["errors"].append({"class": klass, "detail": message[:180], "where": envelope["phase"]})
envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True))
