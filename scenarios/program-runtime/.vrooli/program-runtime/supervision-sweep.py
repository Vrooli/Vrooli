"""program-runtime.supervision-sweep v2 — one fleet read, bounded per-scenario running/health summaries.

Contract: supervision-sweep.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios, counts or keys only").
Demonstrates: discard — one vrooli/scenario/list read replaces one `vrooli scenario status <name>` call per scenario;
only the supervised slugs, their running/health labels, and an attention list cross back.

Phases: validate -> collect -> classify -> report.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "program-runtime.supervision-sweep", "version": "2",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {
        "requested": 0, "found": 0, "running": 0, "healthy": 0,
        "attention": [], "truncated": False, "source_truncated": False,
    },
    "errors": [], "evidence": [],
}

MAX_SCENARIOS = 160
MAX_REQUESTED = 32
MAX_PAGE_SIZE = 20
all_scenarios = inputs.get("all_scenarios") is True
page_offset = inputs.get("page_offset", 0)
page_size = inputs.get("page_size", MAX_PAGE_SIZE)
requested = []          # ordered, de-duplicated supervised slugs
fleet = {}              # name -> {"status": str, "health": str}

fail = program.fail


def step_validate():  # VALIDATE · parse the requested slug set
    envelope["phase"] = "validate"
    raw = inputs.get("scenarios")
    if all_scenarios:
        raw = ""
    elif raw is None:
        raw = ("agent-manager,prompt-manager,web-console,workflow-health,"
               "browser-automation-studio,scenario-to-desktop,"
               "nutrition-planner,personal-planner")
    seen = set()
    for slug in str(raw).split(","):
        slug = slug.strip()
        if slug and slug not in seen:
            seen.add(slug)
            requested.append(slug)
    if not requested and not all_scenarios:
        return fail("failed", "invalid_input",
                    "scenarios resolved to an empty set after trimming", "validate")
    if all_scenarios and (type(page_offset) is not int or page_offset < 0):
        return fail("failed", "invalid_input", "page_offset must be a non-negative integer", "validate")
    if all_scenarios and (type(page_size) is not int or page_size < 1 or page_size > MAX_PAGE_SIZE):
        return fail("failed", "invalid_input", "page_size must be an integer from 1 to 20", "validate")
    if len(requested) > MAX_REQUESTED:
        del requested[MAX_REQUESTED:]
        envelope["signals"]["truncated"] = True
    envelope["inputs"] = {"all_scenarios": all_scenarios}
    if all_scenarios:
        envelope["inputs"]["scenarios"] = ""
    else:
        envelope["inputs"]["scenario_count"] = len(requested)
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
    selected = rows if all_scenarios else rows.filter(lambda row: row.get("name") in requested)
    envelope["signals"]["source_truncated"] = selected.count() > MAX_SCENARIOS
    for row in selected.materialize(MAX_SCENARIOS):
        name = row.get("name")
        if not name:
            continue
        fleet[name] = {
            "status": str(row.get("status", "")),
            "health": str(row.get("healthStatus", "")),
        }
        if all_scenarios and name not in requested:
            requested.append(name)
    envelope["evidence"].append("vrooli/scenario/list")
    return "classify"


def step_classify():  # CLASSIFY · deterministic labels, no inference
    envelope["phase"] = "classify"
    running = 0
    healthy = 0
    found = 0
    attention = []
    details = []
    for slug in requested:
        entry = fleet.get(slug)
        if entry is None:
            status = "unknown" if envelope["signals"]["source_truncated"] else "absent"
            missing = {"name": slug, "status": status, "health": "unknown"}
            details.append(missing)
            attention.append(missing)
            continue
        details.append({"name": slug, "status": entry["status"] or "unknown",
                        "health": entry["health"] or "unknown"})
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
    envelope["signals"]["requested"] = len(requested)
    envelope["signals"]["attention"] = attention[:MAX_SCENARIOS]
    if all_scenarios:
        details.sort(key=lambda row: row["name"])
        page_end = min(page_offset + page_size, len(details))
        page = details[page_offset:page_end]
        envelope["signals"]["scenarios"] = page
        envelope["signals"]["attention"] = [row for row in page if row["status"] != "running" or row["health"] != "healthy"]
        envelope["signals"]["page_offset"] = page_offset
        envelope["signals"]["page_size"] = len(page)
        envelope["signals"]["total_scenarios"] = len(details)
        envelope["signals"]["next_offset"] = page_end if page_end < len(details) else None
    else:
        envelope["signals"]["scenarios"] = details
    # ok when every supervised scenario was present; partial when a slug is absent
    # from the fleet list (a real supervision signal, not a transport failure).
    incomplete = envelope["signals"]["truncated"] or envelope["signals"]["source_truncated"]
    envelope["status"] = "ok" if found == len(requested) and not incomplete else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    if not all_scenarios:
        signals = envelope["signals"]
        while len(json.dumps(envelope, allow_nan=False, separators=(",", ":")).encode()) > 4096 and signals["scenarios"]:
            signals["scenarios"].pop()
            included = {row["name"] for row in signals["scenarios"]}
            signals["attention"] = [row for row in signals["attention"] if row["name"] in included]
            signals["truncated"] = True
            envelope["status"] = "partial"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect,
          "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
