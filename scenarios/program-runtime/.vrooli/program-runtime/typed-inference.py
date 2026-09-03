"""program-runtime.typed-inference v1 — classify a small corpus concurrently through the governed ai facade.

Contract: typed-inference.json.
Skill:    program-runtime (usage tree: "a label per text, closed set").
Demonstrates: gather over ai.classify calls with an enum schema; one projection row per text.

Phases: validate -> classify -> report. inference_calls budget: one per text, at most 8.
"""

import json

try:
    inputs
except NameError:
    inputs = {}
corpus = inputs["corpus"] if "corpus" in inputs else [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
]
labels = inputs["labels"] if "labels" in inputs else ["infra", "user"]

envelope = {
    "program": "program-runtime.typed-inference", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"documents": len(corpus), "labels": labels},
    "signals": {"labels": [], "by_label": {}, "model": None}, "errors": [], "evidence": [],
}
work = {}


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
    if not corpus or len(corpus) > 8:
        return fail("failed", "invalid_input", "corpus must hold 1 to 8 texts", "validate")
    if not labels:
        return fail("failed", "invalid_input", "labels must be non-empty", "validate")
    return "classify"


def step_classify():  # CLASSIFY · concurrent typed calls, enum schema
    envelope["phase"] = "classify"
    try:
        results = gather(*[
            lambda text=text: ai.classify(text, {"type": "string", "enum": labels}, "Choose the primary failure class.")
            for text in corpus
        ])
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "classify")
    for h in results:
        row = h.head(1)[0]
        value = json.loads(row.get("valueJson", "null"))
        label = value.get("label") if isinstance(value, dict) else value
        envelope["signals"]["labels"].append(label)
        envelope["signals"]["by_label"][label] = envelope["signals"]["by_label"].get(label, 0) + 1
        envelope["signals"]["model"] = row.get("model")
    envelope["evidence"].append(f"{len(results)} ai.classify calls via classify.fast")
    envelope["status"] = "ok"
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
