"""Shared learning workflow; see README.md for caller and retry contracts."""
import json
import hashlib
import datetime

inputs = program.inputs()
envelope = {"program": "vrooli-memory.finish-attempt", "version": "1",
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


def refs(values):
    if not isinstance(values, list) or len(values) > 20:
        raise ValueError("at most 20 evidence references")
    for value in values:
        bounded_text(value, "evidence reference", 512)

def validate_correction(value):
    if value and not any(value.startswith(kind + "/v1 ") for kind in ("preference", "parameter", "target-note", "avoid", "trace", "correction", "example")):
        raise ValueError("unknown learning note kind")

def choice(value, choices, name):
    if value not in choices:
        raise ValueError("invalid " + name)

def validate_attempt(a):
    fields = {"attempt_id", "task_id", "operation", "context_key", "started_at", "finished_at",
              "outcome", "failure_fingerprint", "evidence_refs", "advice", "recall_status", "provenance",
              "trigger", "approach", "task_started_at", "attempt_number", "first_action_at",
              "tool_round_trips", "visual_reasoning_calls", "reused_workflow",
              "parent_attempt_id", "step_name"}
    if set(a) - fields:
        raise ValueError("attempt uses unknown fields; use snake_case proto fields")
    for k in ("attempt_id", "task_id", "operation", "context_key", "trigger", "approach"):
        bounded_text(a.get(k), k)
    for k in ("parent_attempt_id", "step_name"):
        if k in a:
            bounded_text(a[k], k)
    start, end, task_start = [timestamp(a.get(k)) for k in ("started_at", "finished_at", "task_started_at")]
    if task_start > start or start > end or end > datetime.datetime.now(datetime.timezone.utc):
        raise ValueError("attempt timestamps must be ordered and finished")
    number = a.get("attempt_number")
    if type(number) is not int or not 1 <= number <= 2147483647:
        raise ValueError("attempt_number must be positive int32")
    if "first_action_at" in a and not start <= timestamp(a["first_action_at"]) <= end:
        raise ValueError("first_action_at must be within attempt")
    for k in ("tool_round_trips", "visual_reasoning_calls"):
        if k in a and (type(a[k]) is not int or not 0 <= a[k] <= 100000):
            raise ValueError("effort counts must be 0..100000 or omitted")
    if "reused_workflow" in a and type(a["reused_workflow"]) is not bool:
        raise ValueError("reused_workflow must be boolean or omitted")
    choice(a.get("outcome"), ("verified_success", "failed", "unavailable", "unknown"), "outcome")
    choice(a.get("provenance"), ("operator", "test", "agent"), "provenance")
    choice(a.get("recall_status"), ("matched", "no_match", "unavailable", "not_requested"), "recall_status")
    fingerprint = a.get("failure_fingerprint", "")
    if a["outcome"] == "failed":
        bounded_text(fingerprint, "failure_fingerprint", 256)
    elif fingerprint:
        raise ValueError("only failed outcomes may carry failure_fingerprint")
    refs(a.get("evidence_refs", []))
    if a["outcome"] == "verified_success" and not a.get("evidence_refs"):
        raise ValueError("verified_success requires evidence")
    advice = a.get("advice", [])
    if not isinstance(advice, list) or len(advice) > 10 or (a["recall_status"] == "matched") != bool(advice):
        raise ValueError("matched requires explicit advice uses; no_match/unavailable require none")
    seen = set()
    for use in advice:
        if not isinstance(use, dict) or set(use) - {"entry_id", "decision", "decision_change", "verdict", "evidence_refs", "derived"}:
            raise ValueError("invalid advice shape")
        entry_id = bounded_text(use.get("entry_id"), "advice entry_id", 128)
        if entry_id in seen:
            raise ValueError("duplicate advice entry_id")
        seen.add(entry_id)
        choice(use.get("decision"), ("applied", "rejected", "unassessed"), "advice decision")
        choice(use.get("verdict"), ("supported", "contradicted", "unknown"), "advice verdict")
        if use["decision"] == "unassessed":
            if use.get("decision_change") or use["verdict"] != "unknown" or use.get("evidence_refs"):
                raise ValueError("unassessed advice records exposure only")
        else:
            bounded_text(use.get("decision_change"), "decision_change")
        refs(use.get("evidence_refs", []))
        if "derived" in use and type(use["derived"]) is not bool:
            raise ValueError("derived advice marker must be boolean")
        if use["verdict"] != "unknown" and not use.get("evidence_refs"):
            raise ValueError("assessed advice requires evidence")

# validate: freeze every write body before the first side effect.
def step_validate():
    if not isinstance(inputs, dict) or set(inputs) - {"scope", "attempt", "attempts", "observations"}:
        raise ValueError("only scope, attempt, observations are accepted")
    scope = bounded_text(inputs.get("scope"), "scope", 128)
    if not isinstance(inputs.get("attempt"), dict) or not isinstance(inputs.get("observations", []), list):
        raise ValueError("attempt must be an object and observations an array")
    # Private JSON copy preserves optional-field absence and rejects non-JSON values.
    request = json.loads(json.dumps(inputs, allow_nan=False))
    a = request["attempt"]
    if "attempt_id" not in a:
        a["attempt_id"] = stable_id("attempt", [scope, a.get("task_id"), a.get("operation"), a.get("context_key"), a.get("attempt_number")])
    nodes = request.get("attempts")
    if not nodes:
        nodes = [a]
    if not isinstance(nodes, list) or not nodes or len(nodes) > 33 or any(not isinstance(node, dict) for node in nodes):
        raise ValueError("attempts must contain 1..33 attempt objects")
    if nodes[0].get("attempt_id") != a.get("attempt_id"):
        raise ValueError("attempt must be the first node in attempts")
    # Contract defaults materialize attempts=[] and JSON round-trips break
    # Python object aliasing. Rebind the root explicitly so an exact retry
    # that changes the root cannot accidentally retain the stale tree node.
    nodes[0] = a
    known = set()
    for node in nodes:
        if "attempt_id" not in node:
            node["attempt_id"] = stable_id("attempt", [scope, node.get("task_id"), node.get("operation"), node.get("context_key"), node.get("attempt_number"), node.get("step_name", "")])
        validate_attempt(node)
        if node["attempt_id"] in known:
            raise ValueError("duplicate attempt_id in tree")
        parent = node.get("parent_attempt_id")
        if parent and parent not in known:
            raise ValueError("parent attempt must precede child")
        known.add(node["attempt_id"])
    request["attempts"] = nodes
    observations = request.setdefault("observations", [])
    if len(observations) > 128:
        raise ValueError("at most 128 observations")
    seen = set()
    for o in observations:
        if not isinstance(o, dict) or set(o) - {"observation_id", "attempt_id", "disposition", "evidence_refs", "method_revision", "correction", "provenance", "observed_at"}:
            raise ValueError("invalid observation fields")
        o.setdefault("attempt_id", a["attempt_id"])
        if o["attempt_id"] not in known and o.get("method_revision") != "learn.feedback":
            raise ValueError("observation must refer to an attempt in this tree")
        choice(o.get("disposition"), ("supported", "contradicted", "insufficient", "unavailable", "unresolved", "unknown"), "disposition")
        choice(o.get("provenance"), ("operator", "test", "agent"), "observation provenance")
        if timestamp(o.get("observed_at")) > datetime.datetime.now(datetime.timezone.utc):
            raise ValueError("observation must be historical")
        refs(o.get("evidence_refs", []))
        if o["disposition"] != "unknown" and not o.get("evidence_refs"):
            raise ValueError("assessed observation requires evidence")
        for key, maximum in (("method_revision", 256), ("correction", 2048)):
            if not isinstance(o.get(key, ""), str) or len(o.get(key, "").encode("utf-8")) > maximum:
                raise ValueError("observation text too large")
        validate_correction(o.get("correction", ""))
        o.setdefault("observation_id", stable_id("observation", [scope, {k: v for k, v in o.items() if k != "observation_id"}]))
        bounded_text(o["observation_id"], "observation_id")
        if o["observation_id"] in seen:
            raise ValueError("duplicate observation_id")
        seen.add(o["observation_id"])
    if len(json.dumps(request, ensure_ascii=True, allow_nan=False).encode("utf-8")) > 24000:
        raise ValueError("resolved retry inputs exceed 24000 bytes")
    envelope["inputs"] = {"scope": scope}
    envelope["signals"] = {"task_outcome": a["outcome"], "capture_status": "pending",
                           "entry_id": None, "existing": False, "attempts": [], "observations": [], "retry_inputs": request}
    return "report"

def receipt(handle):
    rows = handle.head(1)
    result = rows[0] if rows else handle.meta()
    entry_id = result.get("entryId")
    if not isinstance(entry_id, str) or not entry_id or len(entry_id) > 1024:
        raise RuntimeError("missing or invalid capture receipt")
    return {"entry_id": entry_id, "existing": bool(result.get("existing", False))}

# report: record once, then observe once each; stop at the first failure, no retry.
def step_report():
    envelope["phase"] = "report"
    signals = envelope["signals"]
    if signals.get("capture_status") == "pending":
        request = signals["retry_inputs"]
        where = "record"
        try:
            for attempt in request["attempts"]:
                result = receipt(vrooli_memory.learning.record(scope=request["scope"], attempt=attempt))
                signals["attempts"].append({"attempt_id": attempt["attempt_id"], **result})
                if attempt["attempt_id"] == request["attempt"]["attempt_id"]:
                    signals.update(result)
                envelope["evidence"].append(result["entry_id"])
            for observation in request["observations"]:
                where = "observe"
                result = receipt(vrooli_memory.learning.observe(scope=request["scope"], observation=observation))
                result["observation_id"] = observation["observation_id"]
                signals["observations"].append(result)
                envelope["evidence"].append(result["entry_id"])
            signals["capture_status"] = "complete"
            envelope["status"] = "ok"
        except Exception as exc:
            # Mark before classifying so even a missing kernel binding cannot repeat a write.
            signals["capture_status"] = "capture_failed"
            try:
                status, klass = program.classify(exc)
            except (NameError, AttributeError):
                status, klass = "failed", "kernel_runtime"
            envelope["status"] = "partial"
            envelope["errors"].append({"class": "capture_failed", "cause": klass,
                                       "cause_status": status, "detail": str(exc)[:240], "where": where})
    print(json.dumps(envelope, ensure_ascii=True, allow_nan=False))
    return None

state = "validate"
while state:
    try:
        state = step_validate() if state == "validate" else step_report()
    except Exception as exc:
        if state == "report":
            raise
        state = fail("failed", "invalid_input" if isinstance(exc, (ValueError, TypeError)) else "kernel_runtime", exc, state)
