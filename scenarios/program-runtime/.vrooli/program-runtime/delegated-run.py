"""program-runtime.delegated-run v1 — start two governed workflows, then collect their bounded evidence.

Contract: delegated-run.json.
Skill:    program-runtime (usage tree: "genuinely agentic work").
Demonstrates: agent.start is non-blocking (both ids persist against the session before either
collect); agent.collect waits once; a collect from another session is refused.

Phases: validate -> delegate -> report. delegated_runs budget: 2.
Submit with --async; exceeds the synchronous bound.
A start the owner rejects (NOT_FOUND_WORKFLOWREVISION, schema_mismatch) is the domain class
workflow_rejected, refined from binding_error at the call site; program.classify stays verbatim.
"""

import json

inputs = program.inputs()
request = inputs["request"] if "request" in inputs else {
    "owner": "development-toolchain-validator",
    "workflow_key": "development-toolchain-validator/skill-experiment-audit",
    "input": {
        "experiment": {"name": "program-runtime-example", "objective": "Check that a bounded delegated audit returns structured evidence."},
        "assignments": [{"id": "sample", "token": "delegated runtime smoke"}],
    },
}
wait_seconds = min(int(inputs.get("wait_seconds", 30)), 300)

envelope = {
    "program": "program-runtime.delegated-run", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"workflow_key": request.get("workflow_key"), "wait_seconds": wait_seconds},
    "signals": {"started": 0, "collected": 0, "first_row_keys": []}, "errors": [], "evidence": [],
}
work = {}


fail = program.fail


def refine_start(exc, status, klass):
    """Domain refinement of a start failure: an owner that rejects the workflow key or its input."""
    text = str(exc)
    if klass == "binding_error" and any(n in text for n in ("NOT_FOUND_WORKFLOW", "schema_mismatch", "unknown workflow")):
        return status, "workflow_rejected"
    return status, klass


def step_validate():  # VALIDATE
    if not all(k in request for k in ("owner", "workflow_key", "input")):
        return fail("failed", "invalid_input", "request needs owner, workflow_key, and input", "validate")
    return "delegate"


def step_delegate():  # DELEGATE · two starts, then two single collects
    envelope["phase"] = "delegate"
    try:
        first = agent.start(**request)
        second = agent.start(**request)
    except Exception as exc:
        status, klass = refine_start(exc, *program.classify(exc))
        return fail(status, klass, exc, "delegate:start")
    envelope["signals"]["started"] = 2
    for i, h in enumerate((first, second)):
        try:
            rows = agent.collect(h, wait_seconds=wait_seconds).head(1)
        except Exception as exc:
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": f"delegate:collect[{i}]"})
            continue
        envelope["signals"]["collected"] += 1
        if rows:
            envelope["signals"]["first_row_keys"] = sorted(rows[0].keys())
            envelope["evidence"].append(str(rows[0].get("execution_id") or f"run[{i}]"))  # /collect payload key is execution_id
    if envelope["signals"]["collected"] == 0:
        return fail("failed", "no_output", "neither delegated run returned an output", "delegate")
    envelope["status"] = "ok" if envelope["signals"]["collected"] == 2 else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "delegate": step_delegate, "report": step_report}
state = "validate"
program.run(STATES, state)
