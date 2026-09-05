# Governed program. Contract and fixtures are in the sibling JSON file.
try:
    inputs
except NameError:
    inputs = {}

envelope = {"program": "scenario-to-desktop.pipeline-inspect", "version": "1",
            "status": "failed", "phase": "validate", "inputs": {},
            "signals": {"pipeline": None, "task_count": None, "task_states": None,
                        "tasks": [], "tasks_truncated": False, "task_sample_saturated": False}, "errors": [], "evidence": []}
handles = {}
selection = {}

def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:160], "where": where})
    return "report"


def guarded(call):
    def invoke():
        try:
            return call()
        except Exception as exc:
            return exc
    return invoke


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


# VALIDATE: reject malformed selectors before any binding is called.
def step_validate():
    if not isinstance(inputs, dict) or set(inputs) - {"scenario", "pipeline_id"}:
        return fail("failed", "invalid_input", "Expected scenario and optional pipeline_id", "validate")
    for key in ("scenario", "pipeline_id"):
        value = inputs.get(key, "")
        if not isinstance(value, str) or len(value) > 128 or (key == "scenario" and not value.strip()):
            return fail("failed", "invalid_input", "Invalid " + key, "validate")
        selection[key] = value.strip()
    envelope["inputs"] = dict(selection)
    return "collect"


# COLLECT: resolve one pipeline, then read its state and investigations together.
def step_collect():
    envelope["phase"] = "collect"
    if not selection["pipeline_id"]:
        try:
            candidates = scenario_to_desktop.pipeline.list(scenario_name=selection["scenario"])
            candidates = candidates.filter(lambda item: item.get("scenarioName") == selection["scenario"])
            chosen = candidates.map(lambda item: {"id": item.get("pipelineId", ""),
                                      "created": item.get("createdAt", "")}).sort("created", reverse=True).head(1)
            envelope["evidence"].append("scenario-to-desktop/pipeline/list")
        except Exception as exc:
            status, klass = classify_transport(exc)
            return fail(status, klass, klass, "collect:list")
        if not chosen:
            return fail("failed", "pipeline_not_found", "No pipeline for the selected scenario", "collect:list")
        if not chosen[0]["id"] or not chosen[0]["created"]:
            return fail("failed", "invalid_response", "Pipeline selection lacks identity or creation time", "collect:list")
        selection["pipeline_id"] = chosen[0]["id"]
    results = gather(
        guarded(lambda: scenario_to_desktop.pipeline.status(pipeline_id=selection["pipeline_id"])),
        guarded(lambda: scenario_to_desktop.tasks.list(pipeline_id=selection["pipeline_id"], limit=100)))
    for key, result in zip(("pipeline", "tasks"), results):
        if isinstance(result, Exception):
            status, klass = classify_transport(result)
            fail(status, klass, klass, "collect:" + key)
        else:
            handles[key] = result
            envelope["evidence"].append("scenario-to-desktop/" + ("pipeline/status" if key == "pipeline" else "tasks/list"))
    if "pipeline" not in handles:
        return "report"
    return "classify"


# CLASSIFY: preserve the owner's status; do not infer readiness from completion.
def step_classify():
    envelope["phase"] = "classify"
    record = handles["pipeline"].meta()
    if record.get("scenarioName") != selection["scenario"] or record.get("pipelineId") != selection["pipeline_id"]:
        return fail("failed", "identity_mismatch", "Pipeline identity differs from requested selectors", "classify")
    owner_status = record.get("status")
    allowed = ("PENDING", "RUNNING", "COMPLETED", "FAILED", "SKIPPED", "CANCELLED")
    if owner_status not in ["STAGE_STATUS_" + value for value in allowed]:
        return fail("failed", "invalid_response", "Pipeline status missing or unknown", "classify")
    stages = record.get("stages") or {}
    if not isinstance(stages, dict) or len(stages) > 10:
        return fail("failed", "invalid_response", "Stage map exceeds declared bound or has wrong shape", "classify")
    envelope["signals"]["pipeline"] = {
        "id": selection["pipeline_id"], "scenario": selection["scenario"], "status": owner_status,
        "current_stage": record.get("currentStage"), "has_error": bool(record.get("error")),
        "stages": [{"name": name, "status": stage.get("status"), "has_error": bool(stage.get("error"))}
                   for name, stage in sorted(stages.items())]}
    if "tasks" in handles:
        tasks = handles["tasks"]
        if tasks.filter(lambda item: item.get("pipelineId") != selection["pipeline_id"]).count():
            return fail("failed", "identity_mismatch", "Task belongs to a different pipeline", "classify")
        envelope["signals"]["task_count"] = tasks.count()
        envelope["signals"]["task_sample_saturated"] = tasks.count() >= 100
        envelope["signals"]["task_states"] = dict(tasks.map(lambda item: {"status": item.get("status", "unknown")}).group_by("status"))
        envelope["signals"]["tasks"] = tasks.map(lambda item: {"id": item.get("id"), "status": item.get("status")}).head(5)
        envelope["signals"]["tasks_truncated"] = tasks.count() > 5
    envelope["status"] = "partial" if envelope["errors"] else "ok"
    return "report"


# REPORT: emit one bounded envelope, including failed and unavailable paths.
def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect,
          "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        fail("failed", "kernel_runtime", str(exc)[:160], envelope.get("phase") or state)
        state = "report"
