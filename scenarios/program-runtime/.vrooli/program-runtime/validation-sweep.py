"""program-runtime.validation-sweep v1 — latest recorded validation state for a scenario set.

Contract: validation-sweep.json.
Skill:    program-runtime (usage tree: "reads across two or more scenarios, counts or keys only").
Demonstrates: discard — one validate() verb read per scenario replaces a per-scenario test-genie
lookup; only pass/not-pass labels, an attention list, and an unavailable count cross back. It is the
validation-state companion to supervision-sweep (which reports running/health). It reads the LATEST
RECORDED run only; it never starts a validation run.

Phases: validate -> collect -> classify -> report.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "program-runtime.validation-sweep", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {
        "requested": 0, "found": 0, "passing": 0,
        "attention": [], "unavailable": 0, "truncated": False,
    },
    "errors": [], "evidence": [],
}

MAX_SCENARIOS = 32
# test-genie run verdicts that count as passing, normalized to lowercase.
PASSING = {"passed", "pass", "ok", "succeeded", "success", "green",
           "complete", "completed"}

requested = []          # ordered, de-duplicated supervised slugs
rows = {}               # name -> {"status": str, "run_id": str} | {"unavailable": True}

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


def _run_status(row):
    # A run row may express its verdict as a boolean or one of several status
    # fields; read defensively and return (normalized_pass_bool, raw_label).
    if not isinstance(row, dict):
        return False, ""
    passed = row.get("passed")
    if isinstance(passed, bool):
        return passed, ("passed" if passed else "failed")
    for key in ("status", "outcome", "result", "verdict", "state"):
        val = row.get(key)
        if val:
            label = str(val)
            return label.strip().lower() in PASSING, label
    return False, "unknown"


def step_collect():  # COLLECT · one latest-run read per scenario, isolated
    envelope["phase"] = "collect"
    for slug in requested:
        # Each read is isolated: an unreachable validation history for one
        # scenario is that scenario's unavailable, never the sweep's failure.
        try:
            head = validate(slug).head(1)
        except Exception as exc:  # verb may raise BindingError or a transport error
            rows[slug] = {"unavailable": True, "detail": str(exc)[:120]}
            continue
        row = head[0] if head else None
        if not isinstance(row, dict):
            rows[slug] = {"status": "", "run_id": "", "no_run": True}
            continue
        run_id = str(row.get("id") or row.get("runId") or "")
        is_pass, label = _run_status(row)
        rows[slug] = {"status": label, "run_id": run_id, "passing": is_pass}
        envelope["evidence"].append("validate:" + slug)
        if run_id:
            envelope["evidence"].append("test-genie-run:" + run_id)
    return "classify"


def step_classify():  # CLASSIFY · deterministic labels, no inference
    envelope["phase"] = "classify"
    found = 0
    passing = 0
    unavailable = 0
    attention = []
    for slug in requested:
        entry = rows.get(slug) or {}
        if entry.get("unavailable"):
            unavailable += 1
            continue
        if entry.get("no_run"):
            attention.append({"name": slug, "status": "no_run", "run_id": ""})
            continue
        found += 1
        if entry.get("passing"):
            passing += 1
        else:
            attention.append({
                "name": slug,
                "status": entry.get("status") or "unknown",
                "run_id": entry.get("run_id", ""),
            })
    envelope["signals"]["found"] = found
    envelope["signals"]["passing"] = passing
    envelope["signals"]["unavailable"] = unavailable
    envelope["signals"]["attention"] = attention[:MAX_SCENARIOS]
    # unavailable is unknown, never zero: all reads unavailable => unavailable;
    # some unavailable => partial; otherwise ok (not-passing/never-validated are
    # honest domain signals surfaced in attention, not transport failures).
    if unavailable and unavailable == len(requested):
        envelope["status"] = "unavailable"
        envelope["errors"].append({
            "class": "scenario_unreachable",
            "detail": "no scenario's validation history was readable via the validate verb",
            "where": "collect",
        })
    elif unavailable:
        envelope["status"] = "partial"
    else:
        envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect,
          "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
