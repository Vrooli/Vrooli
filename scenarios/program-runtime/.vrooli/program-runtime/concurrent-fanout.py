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
    "signals": {"elapsed_seconds": None, "result_counts": [], "result_outcomes": []}, "errors": [], "evidence": [],
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
    if len(results) < len(names):
        results = list(results) + [RuntimeError("gather returned no result for requested binding") for _ in names[len(results):]]
    elif len(results) > len(names):
        envelope["errors"].append({"class": "ambiguous_response", "detail": "gather returned more results than requested", "where": "collect"})
        results = results[:len(names)]
    handles["results"] = []
    handles["error_classes"] = {}
    for name, result in zip(names, results):
        if isinstance(result, Exception):
            if isinstance(result, (NameError, AttributeError)):
                raise result
            status, klass = program.classify(result)
            envelope["errors"].append({"class": klass, "detail": f"{name}: {str(result)[:160]}", "where": "collect"})
            handles["error_classes"][name] = klass
        handles["results"].append({"binding": name, "value": result})
    envelope["signals"]["elapsed_seconds"] = round(time.perf_counter() - started, 3)
    return "classify"


def step_classify():  # CLASSIFY · counts only
    envelope["phase"] = "classify"
    counts = []
    outcomes = []
    for item in handles["results"]:
        name, result = item["binding"], item["value"]
        if isinstance(result, Exception):
            counts.append(None)
            outcomes.append({"binding": name, "status": "error", "count": None,
                             "error_class": handles["error_classes"].get(name, "unclassified")})
            continue
        try:
            count = result.count()
        except Exception as exc:
            if isinstance(exc, (NameError, AttributeError)):
                raise
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": f"{name}: {str(exc)[:160]}", "where": "classify"})
            counts.append(None)
            outcomes.append({"binding": name, "status": "error", "count": None, "error_class": klass})
            continue
        counts.append(count)
        outcomes.append({"binding": name, "status": "ok", "count": count})
    envelope["signals"]["result_counts"] = counts
    envelope["signals"]["result_outcomes"] = outcomes
    envelope["evidence"].append(f"gather of 3 reads in {envelope['signals']['elapsed_seconds']}s")
    successful = sum(1 for outcome in outcomes if outcome["status"] == "ok")
    envelope["status"] = "ok" if successful == len(outcomes) and not envelope["errors"] else "partial" if successful else "unavailable"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
