"""agent-manager.investigation-evidence v1 — bounded subject evidence collection.

Phases: validate -> collect -> classify -> report. The caller supplies every
subject run explicitly; unavailable projections are not treated as healthy.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "agent-manager.investigation-evidence", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"subjects": [], "subject_count": 0, "available": 0, "unknown": 0, "projection_rows": 0},
    "errors": [], "evidence": [],
}
work = {}


fail = program.fail


def step_validate():
    run_ids = inputs.get("run_ids") if isinstance(inputs, dict) else None
    if not isinstance(run_ids, list) or not 1 <= len(run_ids) <= 8:
        return fail("failed", "invalid_input", "run_ids must contain one to eight explicit subjects", "validate")
    if any(not isinstance(run_id, str) or not run_id.strip() or len(run_id) > 128 for run_id in run_ids):
        return fail("failed", "invalid_input", "each run_id must be a bounded nonempty string", "validate")
    if len(set(run_ids)) != len(run_ids):
        return fail("failed", "invalid_input", "run_ids must be unique", "validate")
    envelope["inputs"] = {"run_ids": list(run_ids)}
    return "collect"


def read_subject(run_id):
    try:
        report = agent_manager.run.report(run_id=run_id)
        episodes = agent_manager.run.episodes(run_id=run_id)
        return {"run_id": run_id, "report": report, "episodes": episodes}
    except Exception as exc:
        return {"run_id": run_id, "error": exc}


def step_collect():
    envelope["phase"] = "collect"
    run_ids = envelope["inputs"]["run_ids"]
    work["subjects"] = gather(*[lambda run_id=run_id: read_subject(run_id) for run_id in run_ids])
    return "classify"


def step_classify():
    subjects = []
    available = 0
    unknown = 0
    rows_total = 0
    for item in work["subjects"]:
        subject = {"run_id": item["run_id"], "availability": "complete", "report": {}, "episodes": []}
        if item.get("error") is not None:
            status, klass = program.classify(item["error"])
            subject["availability"] = "unknown"
            subject["reason"] = klass
            envelope["errors"].append({"class": klass, "detail": str(item["error"])[:180], "where": "collect:" + item["run_id"]})
            unknown += 1
        else:
            report_meta = item["report"].meta()
            episode_rows = item["episodes"].head(4)
            subject["report"] = {"status": report_meta.get("status"), "exit_code": report_meta.get("exitCode"), "events_availability": report_meta.get("eventsAvailability"), "receipts_availability": report_meta.get("receiptsAvailability")}
            subject["episodes"] = [{"episode_id": row.get("episodeId"), "pattern": row.get("pattern"), "owner_confidence": row.get("ownerConfidence")} for row in episode_rows]
            subject["sample_truncated"] = len(episode_rows) == 4
            rows_total += len(episode_rows)
            available += 1
        subjects.append(subject)
    envelope["signals"] = {"subjects": subjects, "subject_count": len(subjects), "available": available, "unknown": unknown, "projection_rows": rows_total, "coverage": "complete" if unknown == 0 else "partial"}
    envelope["status"] = "ok" if unknown == 0 else ("unavailable" if available == 0 else "partial")
    envelope["evidence"] = list(envelope["inputs"]["run_ids"]) + ["agent-manager/run/report", "agent-manager/run/episodes"]
    return "report"


def step_report():
    envelope["phase"] = "report"
    encoded = json.dumps(envelope, sort_keys=True, separators=(",", ":"))
    if len(encoded.encode()) > 4096:
        envelope["status"] = "failed"
        envelope["signals"] = {"subject_count": len(envelope["inputs"].get("run_ids", [])), "output_truncated": True}
        envelope["errors"].append({"class": "binding_error", "detail": "bounded evidence envelope exceeded output limit", "where": "report"})
        encoded = json.dumps(envelope, sort_keys=True, separators=(",", ":"))
    print(encoded)
    return None


states = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        envelope["phase"] = state
        state = states[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        try:
            status, klass = program.classify(exc)
        except (NameError, AttributeError):
            status, klass = "failed", "kernel_runtime"
        state = fail(status, klass, exc, envelope.get("phase") or state)
