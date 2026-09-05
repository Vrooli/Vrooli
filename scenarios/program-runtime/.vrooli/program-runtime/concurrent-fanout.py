"""program-runtime.concurrent-fanout v1 — run independent read bindings concurrently and report elapsed time.

Contract: concurrent-fanout.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios").
Demonstrates: gather with zero-argument callables and elapsed-time evidence; result counts only.

Phases: validate -> collect -> classify -> report.
"""

import time

try:
    inputs
except NameError:
    inputs = {}

envelope = {
    "program": "program-runtime.concurrent-fanout", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"elapsed_seconds": None, "result_counts": []}, "errors": [], "evidence": [],
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


def guarded(call):
    """Run one read on its own worker; an exception becomes the result so the other reads survive."""
    def run():
        try:
            return call()
        except Exception as exc:
            return exc
    return run


def step_validate():  # VALIDATE
    return "collect"


def step_collect():  # COLLECT · three reads on worker threads
    envelope["phase"] = "collect"
    started = time.perf_counter()
    results = gather(
        guarded(lambda: agent_manager.measures.run_volume()),
        guarded(lambda: ai_gateway.measures.total()),
        guarded(lambda: program_runtime.programs.mine(include_operator=False)),
    )
    names = ["agent-manager/measures/run-volume", "ai-gateway/measures/total", "program-runtime/programs/mine"]
    handles["results"] = []
    for name, result in zip(names, results):
        if isinstance(result, Exception):
            if isinstance(result, (NameError, AttributeError)):
                raise result
            status, klass = classify_transport(result)
            envelope["errors"].append({"class": klass, "detail": f"{name}: {str(result)[:160]}", "where": "collect"})
            continue
        handles["results"].append(result)
    if not handles["results"]:
        envelope["status"] = "unavailable"
        return "report"
    envelope["signals"]["elapsed_seconds"] = round(time.perf_counter() - started, 3)
    return "classify"


def step_classify():  # CLASSIFY · counts only
    envelope["phase"] = "classify"
    envelope["signals"]["result_counts"] = [h.count() for h in handles["results"]]
    envelope["evidence"].append(f"gather of 3 reads in {envelope['signals']['elapsed_seconds']}s")
    envelope["status"] = "ok" if not envelope["errors"] else "partial"
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
