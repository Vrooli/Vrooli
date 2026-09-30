"""Bounded, read-only Web Console continuity inventory."""
import json

inputs = program.inputs()

include_generation = bool(inputs.get("include_generation", True))
envelope = {
    "program": "web-console.continuity-integrity-audit",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {"include_generation": include_generation},
    "signals": {
        "source_counts": {}, "orphan_counts": {}, "uncataloged_conversations": 0, "orphan_total": 0,
        "classification": "unknown", "generation": "", "event_content_hash": "",
        "next_actions": [],
    },
    "errors": [],
    "evidence": [],
}

try:
    envelope["phase"] = "collect"
    report = web_console.continuity.integrity().head(1)
    if not report:
        raise RuntimeError("integrity binding returned no report")
    row = report[0]
    envelope["phase"] = "classify"
    source_fields = ("sessions", "conversationSessions", "conversationEvents", "checkpoints", "workspacePanes")
    orphan_fields = ("orphanConversations", "orphanCheckpoints", "orphanWorkspacePanes")
    source_counts = {field: int(row.get(field, 0)) for field in source_fields}
    orphan_counts = {field: int(row.get(field, 0)) for field in orphan_fields}
    uncataloged = int(row.get("uncatalogedConversations", 0))
    orphan_total = sum(orphan_counts.values())
    envelope["signals"]["source_counts"] = source_counts
    envelope["signals"]["orphan_counts"] = orphan_counts
    envelope["signals"]["uncataloged_conversations"] = uncataloged
    envelope["signals"]["orphan_total"] = orphan_total
    if uncataloged:
        envelope["signals"]["classification"] = "degraded"
        envelope["signals"]["next_actions"] = [
            "review web-console continuity reconcile dry-run",
            "quarantine ambiguous aliases before any apply",
        ]
        envelope["status"] = "partial"
    elif orphan_total:
        envelope["signals"]["classification"] = "projection_drift"
        envelope["signals"]["next_actions"] = [
            "review workspace and checkpoint projections",
            "do not delete message-bearing evidence",
        ]
        envelope["status"] = "partial"
    else:
        envelope["signals"]["classification"] = "healthy"
        envelope["signals"]["next_actions"] = ["continue normal continuity monitoring"]
        envelope["status"] = "ok"
    if include_generation:
        envelope["signals"]["generation"] = str(row.get("generation", ""))
        envelope["signals"]["event_content_hash"] = str(row.get("eventContentHash", ""))
    envelope["evidence"].append("web-console/continuity/integrity")
except Exception as exc:
    text = str(exc)
    lower = text.lower()
    klass = "scenario_unreachable" if any(value in lower for value in ("unreachable", "connection refused", "not found")) else "binding_error"
    envelope["status"] = "unavailable" if klass == "scenario_unreachable" else "failed"
    envelope["errors"].append({"class": klass, "detail": text[:240], "where": envelope["phase"]})

envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
