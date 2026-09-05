"""program-runtime.watch-set v1 — start N delegated runs, collect each once, classify the outputs.

Contract: watch-set.json (inputs, invariants, bindings, outputs).
Skill:    program-runtime (usage tree: "several delegated runs at once").

Phases: validate -> delegate -> classify -> report. Delegation goes through the governed
agent bridge (agent.start, agent.collect once per run, wait_seconds <= 300). Classification is
one ai.classify batch over the collected outputs. No polling, no retry: a run that fails or a
bridge that is down is reported with its class, never re-tried here.
Submit with --async; exceeds the synchronous bound.
A start the owner rejects (NOT_FOUND_WORKFLOWREVISION, schema_mismatch) is the domain class
workflow_rejected, refined from binding_error at the call site; classify_transport stays verbatim.
"""

import json

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
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
    "program": "program-runtime.watch-set", "version": "1",
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
            status, klass = refine_start(exc, *classify_transport(exc))
            return fail(status, klass, exc, "delegate:start")
    for i, h in enumerate(work["handles"]):
        try:
            rows = agent.collect(h, wait_seconds=wait_seconds).head(1)
        except Exception as exc:
            status, klass = classify_transport(exc)
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


def step_classify():  # CLASSIFY · one governed batch classification over collected outputs
    envelope["phase"] = "classify"
    texts = [json.dumps(o, default=str)[:2000] for o in work["outputs"] if o is not None]
    try:
        verdicts = ai.classify(texts=texts, labels=labels,
                               instruction="Label the outcome of this delegated run from its collected output.").head(len(texts))
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "classify")
    by_label = {}
    for i, v in enumerate(verdicts):
        # labels= builds an object schema {"label": enum} (kernel/host/engine.py _resolve_labels_schema), and
        # _classify_batch spreads a dict value into the row and adds "text": the row is {label, text}.
        label = v.get("label")
        by_label[label] = by_label.get(label, 0) + 1
        envelope["signals"]["runs"].append({"index": i, "label": label, "validated": label is not None})
    envelope["signals"]["by_label"] = by_label
    envelope["status"] = "ok" if envelope["signals"]["collected"] == len(requests) else "partial"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "delegate": step_delegate, "classify": step_classify, "report": step_report}
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
