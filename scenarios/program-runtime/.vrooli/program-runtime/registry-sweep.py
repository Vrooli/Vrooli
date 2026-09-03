"""program-runtime.registry-sweep v1 — plan a safe registry sweep without invoking write-effect bindings.

Contract: registry-sweep.json.
Skill:    program-runtime (usage tree: "which bindings are reachable right now").
Demonstrates: a dry-run plan read through a governed binding; counts by eligibility only.

Phases: validate -> collect -> classify -> report. Dry-run only: this program never passes execute.
"""

try:
    inputs
except NameError:
    inputs = {}
scenario = inputs.get("scenario")

envelope = {
    "program": "program-runtime.registry-sweep", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"scenario": scenario},
    "signals": {"result_rows": 0, "first_row_keys": [], "by_key": {}}, "errors": [], "evidence": [],
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


def step_validate():  # VALIDATE
    return "collect"


def step_collect():  # COLLECT · dry-run plan only
    envelope["phase"] = "collect"
    try:
        kwargs = {"dry_run": True}
        if scenario:
            kwargs["scenario"] = scenario
        handles["plan"] = program_runtime.bindings.sweep(**kwargs)
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · counts, and a key group when the rows carry one
    envelope["phase"] = "classify"
    plan = handles["plan"]
    envelope["signals"]["result_rows"] = plan.count()
    first = plan.head(1)
    envelope["signals"]["first_row_keys"] = sorted(first[0].keys()) if first else []
    key = next((k for k in ("eligible", "eligibility", "skipReason", "skipped", "status", "effect", "scenario") if first and k in first[0]), None)
    if key:
        envelope["signals"]["by_key"] = {key: dict(plan.group_by(key))}
    envelope["evidence"].append("program-runtime bindings sweep --dry-run")
    envelope["status"] = "ok"
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
