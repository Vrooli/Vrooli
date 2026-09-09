"""Shared learning workflow; see README.md for caller and retry contracts."""
import json
import hashlib
import datetime

inputs = program.inputs()
envelope = {"program": "vrooli-memory.prepare-attempt", "version": "1",
            "status": "failed", "phase": "validate", "inputs": {},
            "signals": {}, "errors": [], "evidence": []}

fail = program.fail

def bounded_text(value, name, limit=1024):
    if not isinstance(value, str) or not value.strip() or len(value.encode("utf-8")) > limit or any(c in value for c in "\r\n\t\x00"):
        raise ValueError(name + " must be nonempty bounded text")
    return value

def timestamp(value):
    bounded_text(value, "timestamp", 64)
    parsed = datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    if parsed.tzinfo is None or "T" not in value:
        raise ValueError("timestamps must be RFC3339 with a timezone")
    return parsed

def stable_id(kind, values):
    body = json.dumps(values, sort_keys=True, separators=(",", ":"), ensure_ascii=True, allow_nan=False)
    return kind + "-" + hashlib.sha256(body.encode("utf-8")).hexdigest()


# validate: stable task identity and caller-observed timestamps, no clock-generated IDs.
def step_validate():
    allowed = {"scope", "task_id", "operation", "context_key", "started_at", "task_started_at",
               "attempt_number", "trigger", "approach", "provenance", "query", "advice_limit"}
    if not isinstance(inputs, dict) or set(inputs) - allowed:
        raise ValueError("unknown prepare input")
    scope = bounded_text(inputs.get("scope"), "scope", 128)
    seed = {k: bounded_text(inputs.get(k), k) for k in
            ("task_id", "operation", "context_key", "started_at", "task_started_at", "trigger", "approach", "provenance")}
    if seed["provenance"] not in ("operator", "test"):
        raise ValueError("provenance must be caller-declared operator or test")
    number = inputs.get("attempt_number")
    if type(number) is not int or not 1 <= number <= 2147483647:
        raise ValueError("attempt_number must be positive int32")
    if timestamp(seed["task_started_at"]) > timestamp(seed["started_at"]):
        raise ValueError("task_started_at must precede started_at")
    limit = inputs.get("advice_limit", 5)
    if type(limit) is not int or not 1 <= limit <= 10:
        raise ValueError("advice_limit must be 1..10")
    query = inputs.get("query", "") or seed["operation"] + " " + seed["context_key"] + " " + seed["trigger"]
    bounded_text(query, "query", 4096)
    seed["attempt_number"] = number
    seed["attempt_id"] = stable_id("attempt", [scope, seed["task_id"], seed["operation"], seed["context_key"], number])
    seed["advice"] = []
    envelope["inputs"] = {"scope": scope, "query": query, "advice_limit": limit}
    envelope["signals"] = {"attempt": seed, "recall_status": "unavailable", "advice_candidates": [],
                           "decision_required": False, "discarded_hits": 0}
    return "collect"

# collect: scoped Recall only; no model, ambient memory, or advice-use inference.
def step_collect():
    envelope["phase"] = "collect"
    signals = envelope["signals"]
    try:
        hits = vrooli_memory.recall.recall(scope=inputs["scope"], query=envelope["inputs"]["query"],
                                          limit=envelope["inputs"]["advice_limit"])
        candidates = []
        seen = set()
        for hit in hits.head(envelope["inputs"]["advice_limit"]):
            entry_id = hit.get("entryId")
            text = hit.get("text")
            if not isinstance(entry_id, str) or not entry_id or len(entry_id.encode("utf-8")) > 128 or not isinstance(text, str) or not text.strip() or hit.get("summary") or entry_id in seen:
                signals["discarded_hits"] += 1
                continue
            seen.add(entry_id)
            excerpt = text.encode("utf-8")[:512].decode("utf-8", errors="ignore")
            candidates.append({"entry_id": entry_id, "text": excerpt, "text_truncated": excerpt != text})
        signals["advice_candidates"] = candidates
        signals["recall_status"] = "matched" if candidates else "no_match"
        signals["decision_required"] = bool(candidates)
        if not candidates:
            signals["attempt"]["recall_status"] = "no_match"
        envelope["evidence"] = [c["entry_id"] for c in candidates]
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = program.classify(exc)
        signals["attempt"]["recall_status"] = "unavailable"
        fail(status, klass, exc, "collect")
    return "report"

def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, ensure_ascii=True, allow_nan=False))
    return None

STATES = {"validate": step_validate, "collect": step_collect, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except ValueError as exc:
        if state == "report":
            raise
        state = fail("failed", "invalid_input" if state == "validate" else "kernel_runtime", exc, state)
    except Exception as exc:
        if state == "report":
            raise
        state = fail("failed", "kernel_runtime", exc, state)
