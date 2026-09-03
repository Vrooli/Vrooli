"""program-runtime.improve-cycle v1 — route the first out-of-band setpoint row to one improve route.

Contract: improve-cycle.json (inputs, invariants, bindings, outputs).
Skill:    program-runtime-improve §2 (row order), §5 (route labels).

Phases: validate -> classify -> decide -> report. The board arrives as an input (the envelope
printed by program-runtime.setpoint-read); no scenario binding is called. `classify` picks the
first out-of-band readable row in table order and names its failure shape, calling ai.classify
only when two shapes tie. `decide` is a pure table lookup from (row, shape) to a route label.
Rows setpoint-read declines by construction (discovery-floor, authoring-floor, attribution,
external-friction, fleet-improve-coverage) are never routed here; the skill's §5 table routes
them from a shell reading or from the program named in the row's reason.
Budget: one inference call.
"""

import json

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
board = inputs.get("board")

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "program-runtime.improve-cycle", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"board": "present" if isinstance(board, dict) else "missing"},
    "signals": {"row": None, "shape": None, "route": None, "rationale": None, "declined": [], "event": None},
    "errors": [], "evidence": [],
}
work = {}

# Rows this program can route, in the improve skill's setpoint table order (§2). It is the tie-break for "first".
ROW_ORDER = [
    "agent-failure-rate", "governance-share", "act-coverage", "binding-condition",
    "delegation-live", "uncovered-recurring-shapes",
]

# Deterministic route table: (row, failure shape or "*") -> route label from the improve skill §5.
ROUTES = {
    ("agent-failure-rate", "kernel_runtime"): "failure-w3-preflight-import",
    ("agent-failure-rate", "kernel_syntax"): "failure-w3-argv-quoting",
    ("agent-failure-rate", "unclassified"): "failure-w2-closed-vocabulary",
    ("agent-failure-rate", "*"): "failure-w2-closed-vocabulary",
    ("governance-share", "*"): "governance-curation-mine-unresolved",
    ("act-coverage", "*"): "act-w1-or-blocked-note",
    ("binding-condition", "*"): "condition-report-bug",
    ("delegation-live", "*"): "delegation-w3-and-report-bug",
    ("uncovered-recurring-shapes", "*"): "shapes-nomination",
}
SHAPE_ROWS = {"agent-failure-rate"}  # rows whose sub-route is chosen by the board's failure shapes


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


# ---- state machine ---------------------------------------------------------------
def step_validate():  # VALIDATE · no binding call
    if not isinstance(board, dict):
        return fail("failed", "board_missing", "board input is required", "validate")
    rows = (board.get("signals") or {}).get("rows")
    if not isinstance(rows, list):
        return fail("failed", "invalid_input", "board.signals.rows is not a list", "validate")
    work["rows"] = rows
    work["shapes"] = (board.get("signals") or {}).get("failure_shapes") or []
    supplied_events = inputs.get("events")
    work["events"] = supplied_events if isinstance(supplied_events, list) else program_runtime.telemetry.events()
    return "classify"


def step_classify():  # CLASSIFY · first out-of-band readable row in table order, then its shape
    envelope["phase"] = "classify"
    for kind, route, rationale in [
        ("NOMINATION", "shapes-nomination", "new recurring uncovered shape nomination"),
        ("COVERAGE_MISS", "discovery-coverage-miss", "recurring shape is covered by a declared contract; discovery query failed"),
    ]:
        if isinstance(work["events"], list):
            matching = [event for event in work["events"] if isinstance(event, dict) and str(event.get("kind", "")).endswith(kind)][:1]
        else:
            matching = work["events"].filter(lambda event: str(event.get("kind", "")).endswith(kind)).head(1)
        if matching:
            event = matching[0]
            envelope["signals"]["event"] = event
            envelope["signals"]["shape"] = event.get("shapeKey") or event.get("shape_key")
            envelope["signals"]["route"] = route
            envelope["signals"]["rationale"] = rationale
            envelope["evidence"].append("program-runtime telemetry events")
            envelope["status"] = "ok"
            return "report"
    by_name = {r.get("row"): r for r in work["rows"] if isinstance(r, dict)}
    envelope["signals"]["declined"] = [
        {"row": r.get("row"), "reason": r.get("reason")} for r in work["rows"]
        if isinstance(r, dict) and r.get("unavailable")
    ][:12]
    picked = None
    for name in ROW_ORDER:
        r = by_name.get(name)
        if r and not r.get("unavailable") and r.get("in_band") is False:
            picked = r
            break
    if picked is None:
        envelope["signals"]["route"] = "in-band"
        envelope["signals"]["rationale"] = "no routable row is out of band; declined rows are listed, never routed"
        envelope["status"] = "ok"
        return "report"
    work["row"] = picked
    name = picked["row"]
    envelope["signals"]["row"] = name
    shape, rationale = "*", "table match on row"
    if name in SHAPE_ROWS and work["shapes"]:
        counted = [(int(s.get("count", 0)), s.get("shape")) for s in work["shapes"] if isinstance(s, dict)]
        counted.sort(key=lambda c: -c[0])
        top = [s for c, s in counted if c == counted[0][0]]
        if len(top) == 1:
            shape, rationale = top[0], f"top failure shape {top[0]} ({counted[0][0]})"
        elif len(top) >= 2:
            candidates = [s for s in top[:2] if (name, s) in ROUTES] or top[:2]
            try:
                verdict = ai.classify(
                    text=json.dumps({"row": name, "reading": picked.get("reading"), "tied_shapes": counted[:2]}),
                    labels=candidates,
                    instruction="Two failure shapes tie for this setpoint row. Choose the shape whose fix moves the sensor first.",
                )
                value = json.loads(verdict.head(1)[0].get("valueJson", "null"))
                shape = value.get("label") if isinstance(value, dict) else value
                rationale = f"tie between {candidates}; ai.classify chose {shape}"
            except Exception as exc:
                status, klass = classify_transport(exc)
                return fail(status, klass, exc, "classify")
    work["shape"], work["rationale"] = shape, rationale
    envelope["signals"]["shape"] = shape
    return "decide"


def step_decide():  # DECIDE · pure table lookup; no I/O
    envelope["phase"] = "decide"
    name, shape = work["row"]["row"], work["shape"]
    route = ROUTES.get((name, shape)) or ROUTES.get((name, "*"))
    if route is None:
        return fail("failed", "no_route", f"no route for row {name}", "decide")
    envelope["signals"]["route"] = route
    envelope["signals"]["rationale"] = work["rationale"]
    envelope["evidence"].append(f"setpoint row {name}: reading={work['row'].get('reading')} target={work['row'].get('target')}")
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "classify": step_classify, "decide": step_decide, "report": step_report}
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
