"""Shared task entry point. Runtime owns durable execution and recovery."""
import json
inputs = program.inputs()
envelope = {"program": "vrooli-memory.run-task", "version": "1", "status": "failed", "phase": "report", "signals": {}, "errors": [], "evidence": []}
try:
    child = tasks.run(**inputs)
    rows = child.head(1)
    if not rows:
        raise ValueError("Task execution returned no result")
    result = rows[0]
    envelope.update(status=result.get("status", "failed"), signals={"result": result.get("signals", {}), "learning": result.get("learning", {})}, errors=result.get("errors", []), evidence=result.get("evidence", []))
except Exception as exc:
    envelope["errors"] = [{"class": "task_unavailable", "where": "runtime", "detail": str(exc)[:240]}]
print(json.dumps(envelope))
