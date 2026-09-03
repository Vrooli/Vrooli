"""prompt-manager.meta-optimization-reads v1 — one handle over the Guide-side meta-optimization sensors.

Contract: meta-optimization-reads.json (inputs, invariants, bindings, outputs).
Skill:    prompt-manager (usage tree: "what should the meta-optimization loop look at next?").

Phases: validate -> collect -> classify -> report. Read-only. Rows: governance share (7 d),
unresolved names, discovery gaps, skill usage, and meta-optimization focus next. Each row is
read through its own binding; a binding that fails leaves its row unavailable with the reason.
"""

try:
    inputs
except NameError:
    inputs = {}
window_seconds = int(inputs.get("window_seconds", 604800))
focus_limit = int(inputs.get("focus_limit", 3))

envelope = {
    "program": "prompt-manager.meta-optimization-reads", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"window_seconds": window_seconds, "focus_limit": focus_limit},
    "signals": {"rows": [], "readable": 0, "unavailable": 0}, "errors": [], "evidence": [],
}
handles = {}
SENSORS = {
    "governance-share": ("program-runtime programs governance-share", lambda: program_runtime.programs.governance_share(window_seconds=window_seconds)),
    "unresolved-names": ("program-runtime programs mine-unresolved", lambda: program_runtime.programs.mine_unresolved(include_operator=False)),
    "discovery-gaps": ("prompt-manager discovery-gaps", lambda: prompt_manager.discovery_gaps.discovery_gaps()),
    "skill-usage": ("prompt-manager skill-usage", lambda: prompt_manager.skill_usage.skill_usage(rows="rows")),  # response has two repeated fields (rows, unread)
    "focus-next": ("meta-optimization-manager focus next", lambda: meta_optimization_manager.focus.next(limit=focus_limit)),
}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    if isinstance(exc, (NameError, AttributeError)):
        raise  # a kernel-bound name is missing: never disguise it as a binding error
    text = str(exc).lower()
    if "unreachable" in text or "bridge" in text or "scenario_not_running" in text:
        return "unavailable", "scenario_unreachable"
    if "requires an explicit grant" in text:
        return "refused", "no_grant"
    return "failed", "binding_error"


def row(name, reading, unavailable=False, reason=None, sensor=None, target=None, in_band=None):
    # canonical setpoint row shape; target/in_band stay None for pure reads that carry no band
    envelope["signals"]["rows"].append({"row": name, "reading": reading, "target": target, "in_band": in_band, "unavailable": unavailable, "reason": reason})
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1
    if sensor:
        envelope["evidence"].append(sensor)


def guarded(name):
    """Wrap one sensor so a single failure becomes an unavailable row, not a failed board."""
    sensor, call = SENSORS[name]
    def run():
        try:
            return call()
        except Exception as exc:
            return exc
    return run


def step_validate():  # VALIDATE
    if window_seconds < 60:
        return fail("failed", "invalid_input", f"window_seconds={window_seconds} below 60", "validate")
    if not (1 <= focus_limit <= 10):
        return fail("failed", "invalid_input", f"focus_limit={focus_limit} outside 1..10", "validate")
    return "collect"


def step_collect():  # COLLECT · every sensor concurrently, each individually guarded
    envelope["phase"] = "collect"
    names = list(SENSORS)
    results = gather(*[guarded(n) for n in names])
    handles.update(zip(names, results))
    return "classify"


def step_classify():  # CLASSIFY · bounded readings per row
    envelope["phase"] = "classify"
    for name, h in handles.items():
        sensor = SENSORS[name][0]
        if isinstance(h, Exception):
            status, klass = classify_transport(h)
            row(name, None, unavailable=True, reason=f"{klass}: {str(h)[:120]}", sensor=sensor)
            continue
        if name == "governance-share":
            m = h.meta()
            row(name, {"governed_share": m.get("governedShare"), "observed": m.get("observedCalls"), "governed": m.get("governedCalls"),
                       "observed_names": [r.get("attemptedName") for r in h.head(5)]}, sensor=sensor)
        elif name == "unresolved-names":
            row(name, {"names": h.count(), "top": [(r.get("attemptedName"), r.get("count")) for r in h.head(5)]}, sensor=sensor)
        elif name == "discovery-gaps":
            row(name, {"gaps": h.count(), "top": [(r.get("query"), r.get("count")) for r in sorted(h.head(50), key=lambda r: int(r.get("count", 0)), reverse=True)[:5]]}, sensor=sensor)
        elif name == "skill-usage":
            row(name, {"rows": h.count(), "sample": h.head(3)}, sensor=sensor)
        elif name == "focus-next":
            row(name, [{"gap": (r.get("gap") or {}).get("id"), "score": r.get("priorityScore"), "impact": r.get("impact")} for r in h.head(focus_limit)], sensor=sensor)
    envelope["status"] = "ok" if envelope["signals"]["unavailable"] == 0 else ("partial" if envelope["signals"]["readable"] else "unavailable")
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
