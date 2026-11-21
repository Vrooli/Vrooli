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
    "signals": {
        "shapes": 0, "returned_shapes": 0, "complete": False, "truncated": False,
        "by_shape": {}, "shape_details": {}, "top": [], "evidence_by_shape": {},
    },
    "errors": [], "evidence": [],
}
handles = {}
MAX_SHAPES = 20


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


def step_classify():  # CLASSIFY · rank a bounded set and keep evidence linked to its shape
    envelope["phase"] = "classify"
    h = handles["mine"]
    returned_count = h.count()
    rows = h.head(MAX_SHAPES)
    meta_count = h.meta().get("count")
    if isinstance(meta_count, int) and not isinstance(meta_count, bool):
        source_count = meta_count
    elif isinstance(meta_count, str) and meta_count.isdigit():
        source_count = int(meta_count)
    else:
        source_count = None
    if source_count is not None and source_count < 0:
        source_count = None

    signals = envelope["signals"]
    signals["shapes"] = source_count if source_count is not None else returned_count
    signals["returned_shapes"] = len(rows)
    signals["truncated"] = returned_count > len(rows) or (
        source_count is not None and source_count > len(rows)
    )
    signals["by_shape"] = {r.get("shape"): int(r.get("count", 0)) for r in rows}
    signals["shape_details"] = {
        r.get("shape"): {
            "count": int(r.get("count", 0)),
            "first_seen": r.get("firstSeen", ""),
            "last_seen": r.get("lastSeen", ""),
            "sample_program_id": r.get("sampleProgramId", ""),
        }
        for r in rows
    }

    if source_count is None:
        envelope["errors"].append({"class": "source_count_unavailable", "where": "classify",
                                   "detail": "The source did not report the total number of failure shapes"})
    elif source_count != returned_count:
        envelope["errors"].append({"class": "source_count_mismatch", "where": "classify",
                                   "detail": "The source count does not match the rows returned"})
    if signals["truncated"]:
        envelope["errors"].append({"class": "shape_rows_truncated", "where": "classify",
                                   "detail": "Only the first 20 failure-shape rows were summarized"})

    ranked = sorted(rows, key=lambda r: int(r.get("count", 0)), reverse=True)
    top_rows = ranked[:3]
    signals["top"] = [r.get("shape") for r in top_rows]
    for row in top_rows:
        shape = row.get("shape")
        sample_id = row.get("sampleProgramId")
        if shape and sample_id:
            signals["evidence_by_shape"][shape] = sample_id
            envelope["evidence"].append(sample_id)
        else:
            envelope["errors"].append({"class": "shape_evidence_missing", "where": "classify",
                                       "detail": "A ranked failure shape has no sample program id"})

    signals["complete"] = not envelope["errors"] and not signals["truncated"]
    envelope["status"] = "ok" if signals["complete"] else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
