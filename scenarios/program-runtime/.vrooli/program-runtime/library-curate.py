"""program-runtime.library-curate v1 — group library candidates by called-binding set and propose promotions.

Contract: library-curate.json (inputs, invariants, bindings, outputs).
Skill:    program-runtime-improve §5 (library-hygiene route).

Phases: validate -> collect -> classify -> decide -> report. Report only: no write binding is
called; `library promote` is the agent's move after reading the proposals. A candidate row is
origin == "agent-authored"; its called-binding set is the row's calledBindingIds (camelCase in-kernel).
"""

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
max_groups = int(inputs.get("max_groups", 6))

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "program-runtime.library-curate", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"max_groups": max_groups},
    "signals": {"candidates": 0, "groups": 0, "ungrouped": 0, "proposals": [], "drops": 0,
                "duplicate_binding_sets": 0, "promoted_duplicates": []},
    "errors": [], "evidence": [],
}
handles = {}


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


def binding_set(row):
    """The called-binding set of one library row, as a sorted tuple; empty when the row carries none."""
    ids = row.get("calledBindingIds") or row.get("called_binding_ids") or []
    return tuple(sorted(set(ids)))


# ---- state machine ---------------------------------------------------------------
def step_validate():  # VALIDATE · no binding call
    if max_groups < 1:
        return fail("failed", "invalid_input", f"max_groups={max_groups} below 1", "validate")
    return "collect"


def step_collect():  # COLLECT · one governed read
    envelope["phase"] = "collect"
    try:
        handles["lib"] = program_runtime.library.list()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    envelope["evidence"].append("program-runtime library list")
    return "classify"


def step_classify():  # CLASSIFY · deterministic grouping in the kernel
    envelope["phase"] = "classify"
    library = handles["lib"]
    candidates = library.filter(lambda r: r.get("origin") == "agent-authored")
    envelope["signals"]["candidates"] = candidates.count()
    groups = {}
    for r in candidates.head(500):
        groups.setdefault(binding_set(r), []).append(r)
    handles["groups"] = groups
    # Promoted entries sharing one binding set: the set-current route, reported as evidence only.
    promoted = {}
    for r in library.filter(lambda r: r.get("origin") != "agent-authored").head(500):
        promoted.setdefault(binding_set(r), set()).add(r.get("name"))
    shared = [sorted(names) for key, names in promoted.items() if key and len(names) > 1]
    envelope["signals"]["duplicate_binding_sets"] = len(shared)  # the count setpoint-read's library-hygiene row reads
    envelope["signals"]["promoted_duplicates"] = [names[:4] for names in shared][:5]
    return "decide"


def step_decide():  # DECIDE · one proposal per group: the newest candidate; the rest are drops
    envelope["phase"] = "decide"
    proposals, drops = [], 0
    ungrouped = handles["groups"].pop((), [])  # candidates with no called-binding set cannot be deduped by binding set
    envelope["signals"]["ungrouped"] = len(ungrouped)
    for key, rows in sorted(handles["groups"].items(), key=lambda kv: (-len(kv[1]), kv[0]))[:max_groups]:
        rows.sort(key=lambda r: r.get("createdAt", ""), reverse=True)
        newest = rows[0]
        proposals.append({
            "binding_set": list(key)[:6], "promote_id": newest.get("sourceProgramId"),
            "group_size": len(rows), "drop_count": len(rows) - 1,
            "reason": "dedupe" if len(rows) > 1 else "single",
        })
        drops += len(rows) - 1
    envelope["signals"]["groups"] = len(handles["groups"])
    envelope["signals"]["proposals"] = proposals
    envelope["signals"]["drops"] = drops
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "decide": step_decide, "report": step_report}
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
