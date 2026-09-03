"""prompt-manager.skill-set-read v1 — read one scenario's skill set as the registry sees it (the Guide-side sensor).

Contract: skill-set-read.json (inputs, invariants, bindings, outputs).
Skill:    prompt-manager (usage tree: "does scenario X ship a conformant skill set?").

Phases: validate -> collect -> classify -> report. Read-only. Rows: registered skills under the
scenario pack for the scenario, presence of the usage id (<scenario>) and improve id
(<scenario>-improve), token size from skill read, and read counts from skill-usage when that
binding answers; a row whose binding fails is reported unavailable with the reason.

Status rule (program-contracts.md §"The envelope"): a row with a permanent reason never lowers
the status; only a transient scenario_unreachable row or a failed read makes the board partial.
The read-counts row is unreliable:proto_drift_skill_usage while the skill-usage binding 500s on
the unknown proto field `projected` (2026-09-02); that is a failed read, so the board is partial
until the binding is repaired.
"""

try:
    inputs
except NameError:
    inputs = {}
scenario = inputs["scenario"] if "scenario" in inputs else "program-runtime"

envelope = {
    "program": "prompt-manager.skill-set-read", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"scenario": scenario},
    "signals": {"rows": [], "readable": 0, "unavailable": 0, "registered": [], "usage_present": None, "improve_present": None},
    "errors": [], "evidence": [],
}
handles = {}
PERMANENT = ("no_governed_binding", "kernel_invoke_budget", "read_elsewhere:", "pending_telemetry")
counters = {"transient_unavailable": 0}


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


def row(name, reading, unavailable=False, reason=None, sensor=None, target=None, in_band=None):
    # canonical setpoint row shape; target/in_band stay None for pure reads that carry no band
    envelope["signals"]["rows"].append({"row": name, "reading": reading, "target": target, "in_band": in_band, "unavailable": unavailable, "reason": reason})
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1
    if unavailable and not (reason or "").startswith(PERMANENT):
        counters["transient_unavailable"] += 1
    if sensor:
        envelope["evidence"].append(sensor)


def belongs(r):
    sid = r.get("id") or ""
    return sid == scenario or sid.startswith(scenario + "-") or f"/scenarios/{scenario}/skills/" in (r.get("contentPath") or "")


def step_validate():  # VALIDATE
    if not isinstance(scenario, str) or not scenario:
        return fail("failed", "invalid_input", "scenario must be a non-empty string", "validate")
    return "collect"


def step_collect():  # COLLECT · registry list first; usage counts are optional
    envelope["phase"] = "collect"
    try:
        handles["list"] = prompt_manager.skill.list()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    try:
        handles["usage"] = prompt_manager.skill_usage.skill_usage(rows="rows")  # two repeated fields: rows, unread
    except Exception as exc:
        classify_transport(exc)  # re-raises kernel_runtime; a transport/binding error is recorded per row
        handles["usage_error"] = str(exc)[:200]
    return "classify"


def step_classify():  # CLASSIFY · deterministic; in-kernel filters only
    envelope["phase"] = "classify"
    mine = handles["list"].filter(belongs)
    ids = sorted(r.get("id") for r in mine.head(50))
    pack = sorted(r.get("id") for r in mine.filter(lambda r: r.get("folder") == "scenario").head(50))
    envelope["signals"]["registered"] = ids
    envelope["signals"]["usage_present"] = scenario in ids
    envelope["signals"]["improve_present"] = f"{scenario}-improve" in ids
    row("registered-under-scenario-pack", {"ids": pack, "count": len(pack)}, sensor="prompt-manager skill list (folder == scenario)")
    row("usage-id-present", scenario in ids, sensor="prompt-manager skill list")
    row("improve-id-present", f"{scenario}-improve" in ids, sensor="prompt-manager skill list")
    # Token size of the set, read through skill read (combined output; missing ids listed in meta).
    try:
        meta = prompt_manager.skill.read(identifiers=[scenario, f"{scenario}-improve"], allow_missing=True, rows="missing").meta()
        row("set-token-size", {"total_tokens": meta.get("totalTokens"), "skill_count": meta.get("skillCount"), "missing": meta.get("missing")},
            sensor="prompt-manager skill read <usage> <improve>")
    except Exception as exc:
        status, klass = classify_transport(exc)
        envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": "classify:set-token-size"})
        row("set-token-size", None, unavailable=True, reason="scenario_unreachable" if klass == "scenario_unreachable" else f"unreliable:{klass}",
            sensor="prompt-manager skill read <usage> <improve>")
    # Read counts: the skill-usage binding answers or it does not.
    if "usage" in handles:
        used = handles["usage"].filter(lambda r: belongs({"id": r.get("skillId") or r.get("id") or r.get("skill") or ""}))
        row("read-counts", {"rows": used.count(), "sample": used.head(3)}, sensor="prompt-manager skill-usage")
    else:
        # The binding 500s on the unknown proto field `projected`: a binding_error, reason proto drift.
        envelope["errors"].append({"class": "binding_error", "detail": str(handles.get("usage_error"))[:240], "where": "collect:read-counts"})
        row("read-counts", None, unavailable=True, reason="unreliable:proto_drift_skill_usage", sensor="prompt-manager skill-usage")
    envelope["status"] = "ok" if counters["transient_unavailable"] == 0 else "partial"
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
