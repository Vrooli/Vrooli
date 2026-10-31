"""Read bounded native usage without claiming billing or live thread state."""
import json

inputs = program.inputs()
envelope = {"program": "web-console.native-usage-summary", "version": "1", "status": "failed", "phase": "report", "signals": {}, "errors": []}
try:
    session = inputs.get("session_id", "")
    threads = inputs.get("threads", [])
    start, end = inputs.get("start"), inputs.get("end")
    if not session or not isinstance(threads, list) or not 1 <= len(threads) <= 16 or any(not isinstance(x, str) or not x for x in threads) or not isinstance(start, str) or not start or not isinstance(end, str) or not end:
        raise ValueError("session_id, one to sixteen thread IDs, and start and end are required")
    result = web_console.session.native_usage(session_id=session, threads=threads, start=start, end=end)
    rows, meta = result.head(16), result.meta()
    envelope["signals"] = {"threads": rows, "total": meta.get("total"), "origin": meta.get("origin"), "pricing": meta.get("pricing", "unknown"), "owner_status": meta.get("status")}
    envelope["status"] = "ok" if meta.get("status") == "ok" and len(rows) == len(threads) and all(row.get("status") == "ok" for row in rows) else "partial"
except Exception as exc:
    envelope["errors"].append({"class": "invalid_input" if isinstance(exc, ValueError) else "binding_error", "detail": str(exc)[:240]})
print(json.dumps(envelope, separators=(",", ":")))
