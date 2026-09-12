"""program-runtime.concurrent-fanout v1 — run independent read bindings concurrently and report elapsed time.

Contract: concurrent-fanout.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios").
Demonstrates: gather with zero-argument callables and elapsed-time evidence; result counts only.

Phases: validate -> collect -> classify -> report.
"""

import json

import time

inputs = program.inputs()

envelope = {
    "program": "program-runtime.concurrent-fanout", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"elapsed_seconds": None, "result_counts": []}, "errors": [], "evidence": [],
}
handles = {}


fail = program.fail


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
            status, klass = program.classify(result)
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
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
