"""program-runtime.failure-triage v1 — find recurring program failure shapes without materializing the corpus.

Contract: failure-triage.json.
Skill:    program-runtime (usage tree: "why do my programs keep failing").
Demonstrates: one governed read, group_by in the kernel, no rows in the output.

Phases: validate -> collect -> classify -> report.
"""

import json

inputs = program.inputs()
include_operator = bool(inputs.get("include_operator", False))

envelope = {
    "program": "program-runtime.failure-triage", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"include_operator": include_operator},
    "signals": {"shapes": 0, "by_shape": {}, "top": []}, "errors": [], "evidence": [],
}
handles = {}


fail = program.fail


def step_validate():  # VALIDATE
    return "collect"


def step_collect():  # COLLECT
    envelope["phase"] = "collect"
    try:
        handles["mine"] = program_runtime.programs.mine(include_operator=include_operator)
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · group in the kernel; keep three sample ids as evidence
    envelope["phase"] = "classify"
    h = handles["mine"]
    envelope["signals"]["shapes"] = h.count()
    envelope["signals"]["by_shape"] = {r.get("shape"): int(r.get("count", 0)) for r in h.head(20)}
    ranked = sorted(h.head(20), key=lambda r: int(r.get("count", 0)), reverse=True)  # sort in Python: a row may omit count
    envelope["signals"]["top"] = [r.get("shape") for r in ranked[:3]]
    envelope["evidence"].extend([r.get("sampleProgramId") for r in h.head(3) if r.get("sampleProgramId")])
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
