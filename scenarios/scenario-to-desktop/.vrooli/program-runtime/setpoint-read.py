# Governed program. Contract and fixtures are in the sibling JSON file.
try:
    inputs
except NameError:
    inputs = {}

envelope = {"program": "scenario-to-desktop.setpoint-read", "version": "1",
            "status": "failed", "phase": "validate", "inputs": {},
            "signals": {"rows": []}, "errors": [], "evidence": []}
handles = {}
USAGE_BINDINGS = (
    "scenario-to-desktop/pipeline/list", "scenario-to-desktop/pipeline/status",
    "scenario-to-desktop/tasks/list", "scenario-to-desktop/evidence/list",
    "scenario-to-desktop/evidence/summary")
PERMANENT_ROWS = [
    ("external-friction", "zero recurring fingerprints in a representative window", "read_elsewhere:agent-manager.friction-digest"),
    ("desktop-behavior", "all selected contract cells evidenced; unavailable cells retained", "pending_telemetry"),
    ("runtime-performance", None, "pending_telemetry"),
    ("pipeline-performance", None, "pending_telemetry"),
    ("engineering-quality", "selected provider maturity targets met without regressions", "pending_telemetry")]

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



def row(name, reading, target, in_band, reason=None):
    envelope["signals"]["rows"].append({"row": name, "reading": reading, "target": target,
        "in_band": in_band, "unavailable": reason is not None, "reason": reason})


# VALIDATE: the board has one fixed scenario and seven-day observation window.
def step_validate():
    if not isinstance(inputs, dict) or inputs:
        return fail("failed", "invalid_input", "This board takes no inputs", "validate")
    return "collect"


# COLLECT: the condition sensor is external to the plant and owns its verdicts.
def step_collect():
    envelope["phase"] = "collect"
    results = gather(guarded(lambda: program_runtime.bindings.condition(
        scenario="scenario-to-desktop", window_seconds=604800, rows="conditions")),
        guarded(lambda: vrooli_memory.learning.measure(scope="scenario-to-desktop-usage", rows="cohorts")))
    collect_learning(results[1])
    result = results[0]
    if isinstance(result, Exception):
        status, klass = classify_transport(result)
        fail("partial", klass, klass, "collect:condition")
        handles["reason"] = "scenario_unreachable" if klass == "scenario_unreachable" else "unreliable:" + klass
    else:
        handles["condition"] = result
        envelope["evidence"].append("program-runtime/bindings/condition")
    return "classify"


# CLASSIFY: dormant exercise is not evidence that a binding serves correctly.
def step_classify():
    envelope["phase"] = "classify"
    if "condition" in handles:
        condition = handles["condition"].filter(lambda item: item.get("bindingId") in USAGE_BINDINGS)
        total = condition.count()
        healthy = condition.filter(lambda item: item.get("status") == "CONDITION_STATUS_HEALTHY").count()
        degraded = condition.filter(lambda item: item.get("status") == "CONDITION_STATUS_DEGRADED").count()
        dormant = condition.filter(lambda item: item.get("status") == "CONDITION_STATUS_DORMANT").count()
        unknown = total - healthy - degraded - dormant
        reading = {"total": total, "healthy": healthy, "degraded": degraded, "dormant": dormant, "unknown": unknown}
        reason = "unreliable:missing_bindings" if total != len(USAGE_BINDINGS) else ("unreliable:unexercised_bindings" if dormant or unknown else None)
        row("binding-condition", reading, "all five usage-program read bindings healthy over 7d", None if reason else healthy == total, reason)
    else:
        row("binding-condition", None, "all five usage-program read bindings healthy over 7d", None, handles["reason"])
    for name, target, reason in PERMANENT_ROWS:
        row(name, None, target, None, reason)
    classify_learning()
    envelope["status"] = "partial" if envelope["errors"] else "ok"
    return "report"


# REPORT: emit one bounded envelope, including failed and unavailable paths.
def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None



# Learning is a typed sensor projection. This program does not recall or capture.
def collect_learning(result):
    if isinstance(result, Exception):
        status, klass = classify_transport(result)
        envelope["errors"].append({"class": klass, "detail": klass, "where": "collect:learning"})
        handles["learning_reason"] = "scenario_unreachable" if klass == "scenario_unreachable" else "unreliable:" + klass
    else:
        handles["learning"] = result
        envelope["evidence"].append("vrooli-memory/learning/measure")


def classify_learning():
    fields = {
        "learning-failure-recurrence": ("attempts", "failed", "unavailable", "unknown", "recurringFailureFingerprints", "repeatedFailures"),
        "learning-success-effort": ("tasks", "completedTasks", "unresolvedTasks", "leftCensoredTasks", "medianAttemptsToSuccess", "medianSecondsToSuccess"),
        "learning-advice-outcomes": ("appliedAdvice", "rejectedAdvice", "supportedAdvice", "contradictedAdvice", "unassessedAdvice", "contradictionRate", "noMatch", "recallUnavailable")}
    reason = handles.get("learning_reason")
    cohorts = []
    meta = {}
    if "learning" in handles:
        source = handles["learning"]
        meta = source.meta()
        cohorts = source.head(10)
        reason = meta.get("reason") or (None if meta.get("reliable", False) else "unreliable:missing_validity")
        if source.count() > 10:
            reason = "unreliable:cohort_sample"
    for name, keys in fields.items():
        reading = None
        if "learning" in handles:
            reading = {"from": meta.get("from"), "to": meta.get("to"), "eligible_attempts": meta.get("eligibleAttempts", 0),
                       "legacy_records": meta.get("legacyTaskRecords", 0), "excluded_test": meta.get("excludedTestAttempts", 0),
                       "cohorts": []}
            for cohort in cohorts:
                item = {"operation": cohort.get("operation"), "context": cohort.get("contextKey")}
                for key in keys:
                    item[key] = cohort.get(key, None if key in ("medianAttemptsToSuccess", "medianSecondsToSuccess", "contradictionRate") else 0)
                reading["cohorts"].append(item)
        envelope["signals"]["rows"].append({"row": name, "reading": reading, "target": None,
            "in_band": None, "unavailable": bool(reason), "reason": reason})

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
