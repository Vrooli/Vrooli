# Governed program. Contract and fixtures are in the sibling JSON file.
try:
    inputs
except NameError:
    inputs = {}

envelope = {"program": "scenario-to-desktop.evidence-inventory", "version": "1",
            "status": "failed", "phase": "validate", "inputs": {},
            "signals": {"capture_count": None, "summary_count": None, "total_bytes": None,
                        "by_kind": None, "incomplete_metadata": None, "captures": [], "truncated": False},
            "errors": [], "evidence": []}
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


# VALIDATE: named scenario required; this inventory never accepts artifact claims.
def step_validate():
    if not isinstance(inputs, dict) or set(inputs) - {"scenario"}:
        return fail("failed", "invalid_input", "Expected only scenario", "validate")
    value = inputs.get("scenario")
    if not isinstance(value, str) or not value.strip() or len(value) > 128:
        return fail("failed", "invalid_input", "scenario must be a nonempty string of at most 128 characters", "validate")
    selection["scenario"] = value.strip()
    envelope["inputs"] = dict(selection)
    return "collect"


# COLLECT: list metadata and its producer summary; fetch no evidence bytes.
def step_collect():
    envelope["phase"] = "collect"
    results = gather(
        guarded(lambda: scenario_to_desktop.evidence.list(scenario_name=selection["scenario"])),
        guarded(lambda: scenario_to_desktop.evidence.summary(scenario_name=selection["scenario"])))
    for key, result in zip(("list", "summary"), results):
        if isinstance(result, Exception):
            status, klass = classify_transport(result)
            fail(status, klass, klass, "collect:" + key)
        else:
            handles[key] = result
            envelope["evidence"].append("scenario-to-desktop/evidence/" + key)
    return "classify" if handles else "report"


# CLASSIFY: counts describe retained captures, never validated capabilities.
def step_classify():
    envelope["phase"] = "classify"
    signals = envelope["signals"]
    if "list" in handles:
        captures = handles["list"]
        if captures.filter(lambda item: item.get("scenarioName") != selection["scenario"]).count():
            return fail("failed", "identity_mismatch", "Capture scenario differs from requested scenario", "classify")
        signals["capture_count"] = captures.count()
        kinds = captures.map(lambda item: {"kind": item.get("kind", "unknown")}).group_by("kind")
        if len(kinds) > 10:
            return fail("failed", "invalid_response", "Capture kinds exceed output bound", "classify")
        signals["by_kind"] = dict(kinds)
        signals["incomplete_metadata"] = captures.filter(lambda item: not all(item.get(key) for key in
                                                       ("captureId", "checksum", "sourceSessionId", "createdAt"))).count()
        signals["captures"] = captures.map(lambda item: {
            "id": item.get("captureId"), "kind": item.get("kind"), "checksum": item.get("checksum"),
            "session": item.get("sourceSessionId"), "created": item.get("createdAt", "")
        }).sort("created", reverse=True).head(5)
        signals["truncated"] = captures.count() > 5
    if "summary" in handles:
        summary = handles["summary"].meta()
        signals["summary_count"] = int(summary.get("count", 0))
        signals["total_bytes"] = int(summary.get("totalBytes", 0))
        if signals["summary_count"] < 0 or signals["total_bytes"] < 0:
            return fail("failed", "invalid_response", "Negative capture summary", "classify")
    envelope["status"] = "partial" if envelope["errors"] else "ok"
    if "list" in handles and "summary" in handles and signals["capture_count"] != signals["summary_count"]:
        return fail("partial", "snapshot_changed", "List and summary disagree; neither is a release verdict", "classify")
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
