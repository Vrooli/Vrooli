"""Shared learning workflow; see README.md for caller and retry contracts."""
import json
import hashlib
import datetime

inputs = program.inputs()
envelope = {"program": "vrooli-memory.compare-outcomes", "version": "1",
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


GROUPS = {
    "failure-recurrence": ["attempts", "failed", "unavailable", "unknown", "recurringFailureFingerprints", "repeatedFailures"],
    "completion-effort": ["tasks", "completedTasks", "unresolvedTasks", "leftCensoredTasks", "medianAttemptsToSuccess", "medianSecondsToSuccess"],
    "advice-outcomes": ["appliedAdvice", "rejectedAdvice", "supportedAdvice", "contradictedAdvice", "unassessedAdvice", "recallUnavailable", "noMatch", "contradictionRate"],
    "first-action-latency": ["firstActionSamples", "medianSecondsToFirstAction"],
    "agent-round-trips": ["toolRoundTripSamples", "medianToolRoundTrips"],
    "visual-reasoning": ["visualReasoningSamples", "medianVisualReasoningCalls"],
    "workflow-reuse": ["reuseSamples", "workflowReuseRate"],
}
OPTIONAL = {"contradictionRate", "medianAttemptsToSuccess", "medianSecondsToSuccess",
            "medianSecondsToFirstAction", "medianToolRoundTrips", "medianVisualReasoningCalls", "workflowReuseRate"}
measurement = {}

# validate: the same optional selectors as existing learning-read consumers.
def step_validate():
    if not isinstance(inputs, dict) or set(inputs) - {"scope", "from", "to", "operation", "context_key", "cohort_limit"}:
        raise ValueError("unknown comparison selector")
    bounded_text(inputs.get("scope"), "scope", 128)
    limit = inputs.get("cohort_limit", 5)
    if type(limit) is not int or not 1 <= limit <= 10:
        raise ValueError("cohort_limit must be 1..10")
    selectors = {"scope": inputs["scope"]}
    for key in ("from", "to", "operation", "context_key"):
        value = inputs.get(key, "")
        if not isinstance(value, str):
            raise ValueError("selectors must be strings")
        if value:
            bounded_text(value, key)
            selectors[key] = value
    now = datetime.datetime.now(datetime.timezone.utc)
    end = timestamp(selectors["to"]) if "to" in selectors else now
    start = timestamp(selectors["from"]) if "from" in selectors else end - datetime.timedelta(days=7)
    if not start < end or end > now or end - start > datetime.timedelta(days=90):
        raise ValueError("comparison window must be ordered, historical, at most 90 days")
    selectors.setdefault("from", start.isoformat().replace("+00:00", "Z"))
    selectors.setdefault("to", end.isoformat().replace("+00:00", "Z"))
    envelope["inputs"] = dict(selectors, cohort_limit=limit)
    measurement["selectors"] = selectors
    return "collect"

# collect: owner computes cohorts; preserve its validity gate and every selected cohort.
def step_collect():
    envelope["phase"] = "collect"
    try:
        source = vrooli_memory.learning.measure(rows="cohorts", **measurement["selectors"])
        meta = source.meta()
        count = source.count()
        cohorts = source.head(envelope["inputs"]["cohort_limit"])
        measurement.update({"meta": meta, "cohorts": cohorts, "count": count})
        envelope["evidence"] = [str(ref)[:512] for ref in meta.get("evidenceRefs", [])[:10]]
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = program.classify(exc)
        measurement["reason"] = "scenario_unreachable" if klass == "scenario_unreachable" else "unreliable:" + klass
        fail(status, klass, exc, "collect")
    return "report"

# report: no aggregation across identities, targets, baselines, or causal claims.
def step_report():
    envelope["phase"] = "report"
    if measurement.get("selectors"):
        meta = measurement.get("meta")
        count = measurement.get("count")
        cohorts = measurement.get("cohorts", [])
        reason = measurement.get("reason")
        if meta is not None:
            reason = meta.get("reason") or (None if meta.get("reliable") else "unreliable:missing_validity")
            if meta.get("truncated"):
                reason = reason or "unreliable:scan_limit"
            if not count:
                reason = reason or "unreliable:no_eligible_attempts"
            if count > envelope["inputs"]["cohort_limit"]:
                reason = reason or "unreliable:cohort_sample"
        reliability = {
            "reliable": bool(meta is not None and meta.get("reliable") and not reason),
            "source_reliable": bool(meta.get("reliable")) if meta is not None else None,
            "reason": reason, "cohort_count": count, "returned_cohorts": len(cohorts),
            "cohorts_truncated": count is not None and count > len(cohorts),
        }
        for key in ("truncated", "scannedEntries", "eligibleAttempts", "excludedTestAttempts",
                    "legacyTaskRecords", "invalidRecords", "duplicateAttempts"):
            reliability[key] = meta.get(key, False if key == "truncated" else 0) if meta is not None else None
        reliability["source_reason"] = str(meta.get("reason", ""))[:240] if meta is not None else None
        reliability["interpretation"] = str(meta.get("interpretation", ""))[:1024] if meta is not None else None
        envelope["signals"]["reliability"] = reliability
        rows = []
        for name, fields in GROUPS.items():
            reading = None
            if meta is not None:
                reading = {"from": meta.get("from"), "to": meta.get("to"),
                           "eligible_attempts": meta.get("eligibleAttempts", 0), "cohorts": []}
                for cohort in cohorts:
                    # Nonoptional proto scalars default to zero; optional fields retain absence.
                    projected = {"operation": cohort.get("operation", ""), "context": cohort.get("contextKey", "")}
                    projected.update({key: cohort.get(key, None if key in OPTIONAL else 0) for key in fields})
                    reading["cohorts"].append(projected)
            rows.append({"row": name, "reading": reading, "target": None, "in_band": None,
                         "unavailable": bool(reason), "reason": reason})
        envelope["signals"]["rows"] = rows
        if len(json.dumps(envelope, ensure_ascii=True, allow_nan=False).encode("utf-8")) > 60000:
            reliability.update({"reliable": False, "reason": "unreliable:output_bound", "returned_cohorts": 0, "cohorts_truncated": True})
            for row in rows:
                row.update({"reading": None, "unavailable": True, "reason": "unreliable:output_bound"})
        if envelope["status"] == "ok" and not reliability["reliable"]:
            envelope["status"] = "partial"
    print(json.dumps(envelope, ensure_ascii=True, allow_nan=False))
    return None

STATES = {"validate": step_validate, "collect": step_collect, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if state == "report":
            # Bad response data is unknown; discard it before emitting the failure envelope.
            envelope["signals"] = {}
            fail("failed", "kernel_runtime", exc, state)
            print(json.dumps(envelope, ensure_ascii=True, allow_nan=False))
            state = None
        else:
            state = fail("failed", "invalid_input" if state == "validate" and isinstance(exc, (ValueError, TypeError)) else "kernel_runtime", exc, state)
