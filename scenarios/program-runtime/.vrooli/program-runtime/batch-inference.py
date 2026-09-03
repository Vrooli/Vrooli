"""program-runtime.batch-inference v1 — classify a corpus through one ordered governed batch call.

Contract: batch-inference.json.
Skill:    program-runtime (usage tree: "a label per text, closed set", many texts).
Demonstrates: ai.batch with a schema, an instruction, and an explicit role; the one-row response whose
`results` list carries the per-item records in input order.

Phases: validate -> classify -> report. inference_calls budget: 1.
"""

import json

try:
    inputs
except NameError:
    inputs = {}
corpus = inputs["corpus"] if "corpus" in inputs else [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
    "The deployment lost its database connection.",
]
labels = inputs["labels"] if "labels" in inputs else ["infra", "user"]

envelope = {
    "program": "program-runtime.batch-inference", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"documents": len(corpus), "labels": labels},
    "signals": {"labels": [], "by_label": {}, "validated": 0}, "errors": [], "evidence": [],
}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def step_validate():  # VALIDATE
    if not corpus or len(corpus) > 32:
        return fail("failed", "invalid_input", "corpus must hold 1 to 32 texts", "validate")
    return "classify"


def step_classify():  # CLASSIFY · one ordered batch call
    envelope["phase"] = "classify"
    try:
        result = ai.batch(corpus, {"type": "string", "enum": labels}, "Choose the primary failure class.", role="classify.fast")
        # ai.batch answers with ONE row carrying `results` and `usage`; the per-item RunResponse
        # records are inside `results`, in input order. head(len(corpus)) would read one row.
        batch = result.head(1)
        rows = (batch[0].get("results") or []) if batch else []
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "classify")
    for row in rows:
        value = json.loads(row.get("valueJson", "null"))
        label = value.get("label") if isinstance(value, dict) else value
        envelope["signals"]["labels"].append(label)
        envelope["signals"]["by_label"][label] = envelope["signals"]["by_label"].get(label, 0) + 1
        envelope["signals"]["validated"] += 1 if row.get("validated") else 0
    envelope["evidence"].append(f"one ai.batch call over {len(corpus)} documents")
    envelope["status"] = "ok" if len(rows) == len(corpus) else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:  # the one catch that guarantees an envelope on every path
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
