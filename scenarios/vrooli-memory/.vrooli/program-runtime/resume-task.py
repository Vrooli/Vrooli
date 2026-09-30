"""Retry frozen learning delivery, never the domain action."""
import json
inputs = program.inputs()
envelope = {"program": "vrooli-memory.resume-task", "version": "1", "status": "failed", "phase": "report", "signals": {}, "errors": [], "evidence": []}
try:
    rows = tasks.resume(attempt_id=inputs["attempt_id"], resume_token=inputs["resume_token"]).head(1)
    if not rows:
        raise ValueError("Task receipt not found")
    envelope.update(status="ok", signals={"task": {key: rows[0].get(key) for key in (
        "task_id", "attempt_id", "state", "outcome", "delivery", "last_error")}})
except Exception as exc:
    envelope["errors"] = [{"class": "task_unavailable", "where": "runtime", "detail": str(exc)[:240]}]
print(json.dumps(envelope))
