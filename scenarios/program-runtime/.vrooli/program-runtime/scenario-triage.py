"""program-runtime.scenario-triage v1 — one-read triage of a flagged scenario.

Contract: scenario-triage.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios, counts or keys only").
Demonstrates: join — status + freshness + a bounded log tail for one scenario join into advisory
hypotheses (schema drift / port conflict / dependency / stale artifact) so a supervisor who saw a
scenario flagged by supervision-sweep triages it in one program instead of a multi-command log hunt.
status is the one required read; freshness and logs are optional and degrade to partial.

Phases: validate -> collect -> classify -> report.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "program-runtime.scenario-triage", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {
        "scenario": "", "lifecycle_status": "", "health": "",
        "freshness": "", "attention": False, "log_lines_read": 0,
        "hypotheses": [],
    },
    "errors": [], "evidence": [],
}

MAX_TAIL = 200
# Deterministic advisory classification: cause -> substrings matched in a log line (lowercased).
PATTERNS = [
    ("schema_drift", ("no such column", "no such table", "has no column",
                      "schema", "migrat", "alter table")),
    ("port_conflict", ("address already in use", "eaddrinuse", "bind:",
                       "port is already", "listen tcp")),
    ("dependency", ("connection refused", "dial tcp", "failed dependency",
                    "dependency", "unreachable", "no such host")),
    ("permission", ("permission denied", "unauthorized", "forbidden",
                    " 401", " 403")),
    ("disk", ("no space left", "disk full")),
]

collected = {"status_read": False}
log_lines = []

fail = program.fail


def _unwrap(row):
    # The singular status/freshness bindings return a wrapper whose payload is
    # nested under "scenario"; the list binding returns flat rows. Handle both.
    if isinstance(row, dict) and isinstance(row.get("scenario"), dict):
        return row["scenario"]
    return row if isinstance(row, dict) else {}


def _first(row, keys):
    if not isinstance(row, dict):
        return ""
    for k in keys:
        v = row.get(k)
        if v not in (None, ""):
            return str(v)
    return ""


def _line_text(row):
    if isinstance(row, str):
        return row
    return _first(row, ("line", "message", "text", "log", "content"))


def step_validate():  # VALIDATE
    envelope["phase"] = "validate"
    slug = str(inputs.get("scenario") or "").strip()
    if not slug:
        return fail("failed", "invalid_input",
                    "scenario resolved to an empty slug", "validate")
    tail = inputs.get("tail", 80)
    try:
        tail = int(tail)
    except (TypeError, ValueError):
        tail = 80
    tail = max(1, min(MAX_TAIL, tail))
    collected["scenario"] = slug
    collected["tail"] = tail
    envelope["inputs"] = {"scenario": slug, "tail": tail}
    envelope["signals"]["scenario"] = slug
    return "collect"


def step_collect():  # COLLECT · status (required) + freshness + logs (optional)
    envelope["phase"] = "collect"
    slug = collected["scenario"]

    # Required: status. An unreadable status is unavailable (unknown), not a failure.
    try:
        rows = vrooli.scenario.status(name=slug).materialize(4)
    except (NameError, AttributeError):
        raise
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        envelope["errors"].append({"class": klass, "detail": str(exc)[:200], "where": "collect"})
        envelope["status"] = status if status in ("unavailable", "refused", "failed") else "unavailable"
        return "report"
    row = _unwrap(rows[0] if rows else {})
    collected["status_read"] = True
    collected["lifecycle_status"] = _first(row, ("status", "lifecycleStatus", "state"))
    collected["health"] = _first(row, ("healthStatus", "health_status", "health"))
    envelope["evidence"].append("vrooli/scenario/status")

    # Optional: freshness. Degrade to partial if unavailable.
    try:
        frows = vrooli.scenario.freshness(name=slug).materialize(4)
        frow = _unwrap(frows[0] if frows else {})
        stale = frow.get("stale")
        if isinstance(stale, bool):
            collected["freshness"] = "stale" if stale else "fresh"
        else:
            collected["freshness"] = _first(frow, ("verdict", "freshness", "status", "state")) or "unknown"
        envelope["evidence"].append("vrooli/scenario/freshness")
    except Exception as exc:  # optional read: never aborts the triage
        collected["freshness"] = "unavailable"
        envelope["errors"].append({"class": "binding_error",
                                   "detail": "freshness: " + str(exc)[:160], "where": "collect"})

    # Optional: bounded log tail for keyword classification.
    try:
        lrows = vrooli.scenario.logs(name=slug, tail=str(collected["tail"])).materialize(collected["tail"])
        for lr in lrows:
            txt = _line_text(lr)
            if txt:
                log_lines.append(txt)
        envelope["evidence"].append("vrooli/scenario/logs")
    except Exception as exc:  # optional read: never aborts the triage
        envelope["errors"].append({"class": "binding_error",
                                   "detail": "logs: " + str(exc)[:160], "where": "collect"})
    return "classify"


def step_classify():  # CLASSIFY · deterministic keyword heuristics, no inference
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    sig["lifecycle_status"] = collected.get("lifecycle_status") or "unknown"
    sig["health"] = collected.get("health") or "unknown"
    sig["freshness"] = collected.get("freshness") or "unknown"
    sig["log_lines_read"] = len(log_lines)
    is_running = sig["lifecycle_status"] == "running"
    is_healthy = sig["health"] == "healthy"
    sig["attention"] = not (is_running and is_healthy)

    seen = set()
    hypotheses = []
    for line in log_lines:
        low = line.lower()
        for cause, needles in PATTERNS:
            if cause in seen:
                continue
            if any(n in low for n in needles):
                seen.add(cause)
                hypotheses.append({"cause": cause, "evidence": line.strip()[:200]})
    sig["hypotheses"] = hypotheses

    # ok when status was read (an unhealthy scenario is a domain result); partial
    # when an optional read failed (some evidence is missing / unknown).
    optional_gap = any(e.get("where") == "collect" for e in envelope["errors"])
    envelope["status"] = "partial" if optional_gap else "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect,
          "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
