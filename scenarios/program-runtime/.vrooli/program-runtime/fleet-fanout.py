"""program-runtime.fleet-fanout v2 — read several governed scenario surfaces and keep only bounded summaries.

Contract: fleet-fanout.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios").
Demonstrates: gather over three scenario bindings; count and first-row keys only, no rows.

Phases: validate -> collect -> classify -> report.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "program-runtime.fleet-fanout", "version": "2",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"surfaces": {}}, "errors": [], "evidence": [],
}
CALLS = {
    "agent_manager_runs": lambda: agent_manager.measures.run_volume(),
    "ai_gateway_calls": lambda: ai_gateway.measures.total(),
    "program_runtime_conditions": lambda: program_runtime.bindings.condition(scenario="program-runtime", window_seconds=86400, rows="conditions"),
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


def step_collect():  # COLLECT · concurrent governed reads
    envelope["phase"] = "collect"
    results = gather(*[guarded(call) for call in CALLS.values()])
    for name, result in zip(CALLS, results):
        if isinstance(result, Exception):
            if isinstance(result, (NameError, AttributeError)):
                raise result
            status, klass = program.classify(result)
            envelope["errors"].append({"class": klass, "detail": f"{name}: {str(result)[:160]}", "where": "collect"})
            if status == "unavailable":
                envelope["signals"]["surfaces"][name] = {"unavailable": True}
            continue
        handles[name] = result
    if not handles:
        envelope["status"] = "unavailable"
        return "report"
    return "classify"


def step_classify():  # CLASSIFY · bounded summaries only
    envelope["phase"] = "classify"
    for name, h in handles.items():
        count = h.count()
        keys = sorted(h.head(1)[0]) if count else []
        shown = ["".join(char if " " <= char <= "~" else "?" for char in key.encode("ascii", "replace").decode("ascii")[:40]) for key in keys[:10]]
        envelope["signals"]["surfaces"][name] = {
            "count": count,
            "first_row_keys": shown,
            "keys_truncated": len(keys) > 10 or shown != keys[:10],
        }
    envelope["evidence"].extend(["agent-manager/measures/run-volume", "ai-gateway/measures/total", "program-runtime/bindings/condition"])
    envelope["status"] = "ok" if not envelope["errors"] else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
