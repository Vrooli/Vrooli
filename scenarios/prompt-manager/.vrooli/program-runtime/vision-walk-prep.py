"""prompt-manager.vision-walk-prep v1 — the bounded morning vision walk projection (adopted from library morning-vision-walk-prep@4).

Contract: vision-walk-prep.json (inputs, invariants, bindings, outputs).
Owner:    director-swarm/vision-walk-prep heartbeat (HEARTBEAT.md names this program).

Phases: validate -> collect -> classify -> report. Read-only. Two governed reads, gathered:
agent-manager run volume (fleet health) and ai-gateway total (inference activity). The
projection is the first row of each; nothing else is materialized.
"""

try:
    inputs
except NameError:
    inputs = {}
window = inputs.get("window")
WINDOW_TOKENS = {"TIME_WINDOW_TOKEN_THIS_WEEK", "TIME_WINDOW_TOKEN_LAST_7D", "TIME_WINDOW_TOKEN_LAST_30D",
                 "TIME_WINDOW_TOKEN_THIS_MONTH", "TIME_WINDOW_TOKEN_LAST_MONTH", "TIME_WINDOW_TOKEN_THIS_QUARTER"}
# measures.v1.TimeWindowToken enum names: protojson binds an enum by its name, not by the CLI's lowercase token.
# Only ai_gateway.measures.total takes a window; agent_manager.measures.run_volume takes none.

envelope = {
    "program": "prompt-manager.vision-walk-prep", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"window": window},
    "signals": {"fleet_health": None, "inference_activity": None, "readable": 0, "unavailable": 0},
    "errors": [], "evidence": [],
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


def guarded(call):
    def run():
        try:
            return call()
        except Exception as exc:
            return exc
    return run


def step_validate():  # VALIDATE
    if window is not None and window not in WINDOW_TOKENS:
        return fail("failed", "invalid_input", f"window must be one of {sorted(WINDOW_TOKENS)}", "validate")
    return "collect"


def step_collect():  # COLLECT · two governed reads, each guarded
    envelope["phase"] = "collect"
    total = (lambda: ai_gateway.measures.total(window={"token": window})) if window else (lambda: ai_gateway.measures.total())  # TimeWindow is a message, not a string
    handles["fleet"], handles["inference"] = gather(guarded(lambda: agent_manager.measures.run_volume()), guarded(total))  # run_volume takes no window
    return "classify"


def step_classify():  # CLASSIFY · first row of each, or unavailable with reason
    envelope["phase"] = "classify"
    for key, signal, sensor in (("fleet", "fleet_health", "agent-manager/measures/run-volume"), ("inference", "inference_activity", "ai-gateway/measures/total")):
        h = handles[key]
        if isinstance(h, Exception):
            status, klass = classify_transport(h)
            envelope["errors"].append({"class": klass, "detail": str(h)[:240], "where": f"collect:{key}"})
            envelope["signals"]["unavailable"] += 1
            continue
        rows = h.head(1)
        envelope["signals"][signal] = rows[0] if rows else {}
        envelope["signals"]["readable"] += 1
        envelope["evidence"].append(sensor)
    u = envelope["signals"]["unavailable"]
    envelope["status"] = "ok" if u == 0 else ("partial" if u == 1 else "unavailable")
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
