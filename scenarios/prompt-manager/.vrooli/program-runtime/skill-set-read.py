"""prompt-manager.skill-set-read v1 — read one scenario's skill set as the registry sees it (the Guide-side sensor).

Contract: skill-set-read.json (inputs, invariants, bindings, outputs).
Skill:    prompt-manager (usage tree: "does scenario X ship a conformant skill set?").

Phases: validate -> collect -> classify -> report. Read-only. Rows: registered skills under the
scenario pack for the scenario, presence of the usage id (<scenario>) and improve id
(<scenario>-improve), token size from skill read, and read counts from skill-usage when that
binding answers; a row whose binding fails is reported unavailable with the reason.
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
        row("set-token-size", None, unavailable=True, reason=f"{klass}: {str(exc)[:120]}")
    # Read counts: the skill-usage binding answers or it does not.
    if "usage" in handles:
        used = handles["usage"].filter(lambda r: belongs({"id": r.get("skillId") or r.get("id") or r.get("skill") or ""}))
        row("read-counts", {"rows": used.count(), "sample": used.head(3)}, sensor="prompt-manager skill-usage")
    else:
        row("read-counts", None, unavailable=True, reason=f"binding_error: {handles.get('usage_error')}")
    envelope["status"] = "ok" if envelope["signals"]["unavailable"] == 0 else "partial"
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
