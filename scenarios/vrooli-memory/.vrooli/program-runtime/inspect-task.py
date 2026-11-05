"""Shared task entry point. Runtime owns durable execution and recovery."""
import json
inputs = program.inputs()
envelope = {"program": "vrooli-memory.inspect-task", "version": "1", "status": "failed", "phase": "report", "signals": {}, "errors": [], "evidence": []}
try:
    rows = tasks.get(attempt_id=inputs["attempt_id"], resume_token=inputs.get("resume_token", "")).head(1)
    if not rows:
        raise ValueError("Task receipt not found")
    record = {key: rows[0].get(key) for key in (
        "task_id", "attempt_id", "attempt_number", "operation", "scope", "context_key",
        "state", "outcome", "delivery", "started_at", "finished_at", "last_error",
        "delivery_attempts", "next_attempt_at")}
    preparation = rows[0].get("prepare", {}).get("signals", {})
    record["recall_status"] = preparation.get("recall_status", "unavailable")
    record["advice_candidates"] = [
        {"entry_id": candidate.get("entry_id"),
         "text": str(candidate.get("text", ""))[:1200],
         "text_truncated": bool(candidate.get("text_truncated")) or len(str(candidate.get("text", ""))) > 1200}
        for candidate in preparation.get("advice_candidates", [])[:5]]
    envelope.update(status="ok", signals={"task": record})
except Exception as exc:
    envelope["errors"] = [{"class": "task_unavailable", "where": "runtime", "detail": str(exc)[:240]}]
print(json.dumps(envelope))
