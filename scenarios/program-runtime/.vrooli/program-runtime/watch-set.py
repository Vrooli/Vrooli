"""program-runtime.watch-set v2 — start N delegated runs, collect each once, classify the outputs.

Contract: watch-set.json (inputs, invariants, bindings, outputs).
Skill:    program-runtime (usage tree: "several delegated runs at once").

Phases: validate -> delegate -> classify -> report. Delegation goes through the governed
agent bridge (agent.start, agent.collect once per run, wait_seconds <= 300). Classification is
one ai-gateway.classify-batch workflow over the collected outputs. No polling, no retry: a run that fails or a
bridge that is down is reported with its class, never re-tried here.
Submit with --async; exceeds the synchronous bound.
A start the owner rejects (NOT_FOUND_WORKFLOWREVISION, schema_mismatch) is the domain class
workflow_rejected, refined from binding_error at the call site; program.classify stays verbatim.
"""

import json

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
inputs = program.inputs()
requests = inputs["requests"] if "requests" in inputs else [{
    "owner": "development-toolchain-validator",
    "workflow_key": "development-toolchain-validator/skill-experiment-audit",
    "input": {
        "experiment": {"name": "watch-set-default", "objective": "Check that a bounded delegated audit returns structured evidence."},
        "assignments": [{"id": "sample", "token": "delegated runtime smoke"}],
    },
}]
labels = inputs["labels"] if "labels" in inputs else ["succeeded", "failed", "needs-review"]
wait_seconds = min(int(inputs.get("wait_seconds", 120)), 300)

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "program-runtime.watch-set", "version": "2",
    "status": "failed", "phase": "validate",
    "inputs": {"requests": len(requests), "labels": labels, "wait_seconds": wait_seconds},
    "signals": {"started": 0, "collected": 0, "by_label": {}, "runs": []},
    "errors": [], "evidence": [],
}
work = {"handles": [], "outputs": []}


def fail(status, klass, detail, where):
    """The one place a bad path is recorded. Sets status, appends the error, routes to report."""
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def refine_start(exc, status, klass):
    """Domain refinement of a start failure: an owner that rejects the workflow key or its input."""
    text = str(exc)
    if klass == "binding_error" and any(n in text for n in ("NOT_FOUND_WORKFLOW", "schema_mismatch", "unknown workflow")):
        return status, "workflow_rejected"
    return status, klass


# ---- state machine ---------------------------------------------------------------
def step_validate():  # VALIDATE · no bridge call
    if not isinstance(requests, list) or not requests:
        return fail("failed", "invalid_input", "requests must be a non-empty list", "validate")
    for i, req in enumerate(requests):
        if not all(k in req for k in ("owner", "workflow_key", "input")):
            return fail("failed", "invalid_input", f"requests[{i}] lacks owner, workflow_key, or input", "validate")
    if len(requests) > 8:
        return fail("failed", "invalid_input", "at most 8 delegated runs per submission", "validate")
    if not labels:
        return fail("failed", "invalid_input", "labels must be non-empty", "validate")
    return "delegate"


def step_delegate():  # DELEGATE · start every run, then collect each exactly once
    envelope["phase"] = "delegate"
    for req in requests:
        try:
            work["handles"].append(agent.start(**req))
            envelope["signals"]["started"] += 1
        except Exception as exc:
            status, klass = refine_start(exc, *program.classify(exc))
            return fail(status, klass, exc, "delegate:start")
    for i, h in enumerate(work["handles"]):
        try:
            rows = agent.collect(h, wait_seconds=wait_seconds).head(1)
        except Exception as exc:
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": f"delegate:collect[{i}]"})
            work["outputs"].append(None)
            continue
        envelope["signals"]["collected"] += 1
        row = rows[0] if rows else {}
        work["outputs"].append(row)
        envelope["evidence"].append(str(row.get("execution_id") or f"run[{i}]"))  # /collect payload key is execution_id
    if envelope["signals"]["collected"] == 0:
        return fail("failed", "no_output", "no delegated run returned an output", "delegate")
    return "classify"


def step_classify():  # CLASSIFY: retain original run indices through collection gaps.
    envelope["phase"] = "classify"
    collected = [(index, output) for index, output in enumerate(work["outputs"]) if output is not None]
    texts = [json.dumps(output, default=str)[:2000] for _, output in collected]
    try:
        child = lib.ai_gateway.classify_batch(
            corpus=texts, labels=labels,
            instruction="Label the outcome of this delegated run from its collected output.")
        result = child.head(1)[0]
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "classify")
    signals = result["signals"]
    for item in signals["results"]:
        envelope["signals"]["runs"].append({"index": collected[item["index"]][0],
            "label": item["label"], "validated": item["validated"], "error": item["error"]})
    envelope["signals"]["by_label"] = signals["by_label"]
    envelope["signals"]["usage"] = signals["usage"]
    for issue in result["errors"]:
        mapped = dict(issue)
        where = mapped.get("where", "")
        if where.startswith("classify:") and where[9:].isdigit():
            child_index = int(where[9:])
            if child_index < len(collected):
                mapped["where"] = "classify:" + str(collected[child_index][0])
        envelope["errors"].append(mapped)
    envelope["evidence"].append({"program": "ai-gateway.classify-batch", "artifact": child.meta()})
    envelope["status"] = result["status"]
    if envelope["status"] == "ok" and envelope["signals"]["collected"] != len(requests):
        envelope["status"] = "partial"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "delegate": step_delegate, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
