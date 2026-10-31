"""Observe a bounded cohort without controlling native goals or threads."""
import json

inputs = program.inputs()
envelope = {"program": "web-console.native-thread-state", "version": "1", "status": "failed", "phase": "report", "inputs": {}, "signals": {}, "errors": [], "evidence": []}
try:
    session = inputs.get("session_id", "")
    threads = inputs.get("threads", [])
    if not session or not isinstance(threads, list) or not 1 <= len(threads) <= 16 or any(not isinstance(x, str) or not x for x in threads) or len(set(threads)) != len(threads):
        raise ValueError("session_id and one to sixteen unique thread IDs are required")
    envelope["inputs"] = {"threads": list(threads)}
    rows = web_console.session.native_thread_state(session_id=session, threads=threads).head(16)
    envelope["evidence"].append("web-console/session/native-thread-state")
    envelope["signals"] = {"threads": rows}
    envelope["status"] = "ok" if len(rows) == len(threads) and all(row.get("status") != "unavailable" and row.get("goalAvailable") for row in rows) else "partial"
except Exception as exc:
    envelope["errors"].append({"class": "invalid_input" if isinstance(exc, ValueError) else "binding_error", "detail": str(exc)[:240]})
print(json.dumps(envelope, separators=(",", ":")))
