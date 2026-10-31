"""program-runtime.supervision-sweep v1 — one fleet read, bounded per-scenario running/health for a delivery set.

Contract: supervision-sweep.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios, counts or keys only").
Demonstrates: discard — one vrooli/scenario/list read replaces one `vrooli scenario status <name>` call per scenario;
only the supervised slugs, their running/health labels, and an attention list cross back.

Phases: validate -> collect -> classify -> report.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "program-runtime.supervision-sweep", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {
        "requested": 0, "found": 0, "running": 0, "healthy": 0,
        "attention": [], "truncated": False, "source_truncated": False,
    },
    "errors": [], "evidence": [],
}

MAX_SCENARIOS = 32
requested = []          # ordered, de-duplicated supervised slugs
fleet = {}              # name -> {"status": str, "health": str}

fail = program.fail


def step_validate():  # VALIDATE · parse the requested slug set
    envelope["phase"] = "validate"
    raw = inputs.get("scenarios")
    if raw is None:
        raw = ("agent-manager,prompt-manager,web-console,workflow-health,"
               "browser-automation-studio,scenario-to-desktop,"
               "nutrition-planner,personal-planner")
    seen = set()
    for slug in str(raw).split(","):
        slug = slug.strip()
        if slug and slug not in seen:
            seen.add(slug)
            requested.append(slug)
    if not requested:
        return fail("failed", "invalid_input",
                    "scenarios resolved to an empty set after trimming", "validate")
    if len(requested) > MAX_SCENARIOS:
        del requested[MAX_SCENARIOS:]
        envelope["signals"]["truncated"] = True
    envelope["inputs"] = {"scenarios": ",".join(requested)}
    envelope["signals"]["requested"] = len(requested)
    return "collect"


def step_collect():  # COLLECT · one governed fleet read
    envelope["phase"] = "collect"
    try:
        rows = vrooli.scenario.list(rows="scenarios")
    except (NameError, AttributeError):
        raise
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        envelope["errors"].append({"class": klass, "detail": str(exc)[:200], "where": "collect"})
        envelope["status"] = status if status in ("unavailable", "refused", "failed") else "unavailable"
        return "report"
    # Filter the complete handle before materializing, so unrelated fleet rows
    # cannot hide a requested scenario beyond an arbitrary prefix.
    selected = rows.filter(lambda row: row.get("name") in requested)
    envelope["signals"]["source_truncated"] = selected.count() > MAX_SCENARIOS
    for row in selected.materialize(MAX_SCENARIOS):
        name = row.get("name")
        if not name:
            continue
        fleet[name] = {
            "status": str(row.get("status", "")),
            "health": str(row.get("healthStatus", "")),
        }
    envelope["evidence"].append("vrooli/scenario/list")
    return "classify"


def step_classify():  # CLASSIFY · deterministic labels, no inference
    envelope["phase"] = "classify"
    running = 0
    healthy = 0
    found = 0
    attention = []
    for slug in requested:
        entry = fleet.get(slug)
        if entry is None:
            status = "unknown" if envelope["signals"]["source_truncated"] else "absent"
            attention.append({"name": slug, "status": status, "health": "unknown"})
            continue
        found += 1
        is_running = entry["status"] == "running"
        is_healthy = entry["health"] == "healthy"
        if is_running:
            running += 1
        if is_healthy:
            healthy += 1
        if not is_running or not is_healthy:
            attention.append({
                "name": slug,
                "status": entry["status"] or "unknown",
                "health": entry["health"] or "unknown",
            })
    envelope["signals"]["found"] = found
    envelope["signals"]["running"] = running
    envelope["signals"]["healthy"] = healthy
    envelope["signals"]["attention"] = attention[:MAX_SCENARIOS]
    # ok when every supervised scenario was present; partial when a slug is absent
    # from the fleet list (a real supervision signal, not a transport failure).
    incomplete = envelope["signals"]["truncated"] or envelope["signals"]["source_truncated"]
    envelope["status"] = "ok" if found == len(requested) and not incomplete else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect,
          "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
