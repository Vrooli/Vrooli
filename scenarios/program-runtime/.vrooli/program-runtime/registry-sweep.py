"""program-runtime.registry-sweep v1 — plan a safe registry sweep without invoking write-effect bindings.

Contract: registry-sweep.json.
Skill:    program-runtime (usage tree: "which bindings are reachable right now").
Demonstrates: a dry-run plan read through a governed binding; counts by eligibility only.

Phases: validate -> collect -> classify -> report. Dry-run only: this program never passes execute.
"""

import json

inputs = program.inputs()
scenario = inputs.get("scenario")

envelope = {
    "program": "program-runtime.registry-sweep", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"scenario": scenario},
    "signals": {"result_rows": 0, "first_row_keys": [], "by_key": {}}, "errors": [], "evidence": [],
}
handles = {}


fail = program.fail


def step_validate():  # VALIDATE
    return "collect"


def step_collect():  # COLLECT · dry-run plan only
    envelope["phase"] = "collect"
    try:
        kwargs = {"dry_run": True}
        if scenario:
            kwargs["scenario"] = scenario
        handles["plan"] = program_runtime.bindings.sweep(**kwargs)
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · counts, and a key group when the rows carry one
    envelope["phase"] = "classify"
    plan = handles["plan"]
    envelope["signals"]["result_rows"] = plan.count()
    first = plan.head(1)
    envelope["signals"]["first_row_keys"] = sorted(first[0].keys()) if first else []
    key = next((k for k in ("eligible", "eligibility", "skipReason", "skipped", "status", "effect", "scenario") if first and k in first[0]), None)
    if key:
        envelope["signals"]["by_key"] = {key: dict(plan.group_by(key))}
    envelope["evidence"].append("program-runtime bindings sweep --dry-run")
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
