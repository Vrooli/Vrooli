"""program-runtime.failure-triage v1 — find recurring program failure shapes without materializing the corpus.

Contract: failure-triage.json.
Skill:    program-runtime (usage tree: "why do my programs keep failing").
Demonstrates: one governed read, group_by in the kernel, no rows in the output.

Phases: validate -> collect -> classify -> report.
"""

try:
    inputs
except NameError:
    inputs = {}
include_operator = bool(inputs.get("include_operator", False))

envelope = {
    "program": "program-runtime.failure-triage", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"include_operator": include_operator},
    "signals": {"shapes": 0, "by_shape": {}, "top": []}, "errors": [], "evidence": [],
}
handles = {}


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
    return "collect"


def step_collect():  # COLLECT
    envelope["phase"] = "collect"
    try:
        handles["mine"] = program_runtime.programs.mine(include_operator=include_operator)
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · group in the kernel; keep three sample ids as evidence
    envelope["phase"] = "classify"
    h = handles["mine"]
    envelope["signals"]["shapes"] = h.count()
    envelope["signals"]["by_shape"] = {r.get("shape"): int(r.get("count", 0)) for r in h.head(20)}
    ranked = sorted(h.head(20), key=lambda r: int(r.get("count", 0)), reverse=True)  # sort in Python: a row may omit count
    envelope["signals"]["top"] = [r.get("shape") for r in ranked[:3]]
    envelope["evidence"].extend([r.get("sampleProgramId") for r in h.head(3) if r.get("sampleProgramId")])
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
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
