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
    wanted = set(threads)
    seen = {}
    invalid = False
    for row in rows:
        identity = row.get("threadId") if isinstance(row, dict) else None
        if identity not in wanted:
            invalid = True
        elif identity in seen:
            seen[identity] = None
            invalid = True
        else:
            seen[identity] = row
    observed = []
    for identity in threads:
        row = seen.get(identity)
        if row is None:
            invalid = True
            row = {"threadId": identity, "status": "unavailable", "goalAvailable": False}
        observed.append(row)
    envelope["signals"] = {"threads": observed}
    if invalid:
        envelope["errors"].append({"class": "invalid_response", "detail": "owner returned missing, duplicate or unexpected thread identities"})
    envelope["status"] = "ok" if not invalid and all(isinstance(row.get("status"), str) and row.get("status") not in ("", "unavailable") and row.get("goalAvailable") is True for row in observed) else "partial"
except Exception as exc:
    envelope["errors"].append({"class": "invalid_input" if isinstance(exc, ValueError) else "binding_error", "detail": str(exc)[:240]})
print(json.dumps(envelope, separators=(",", ":")))
