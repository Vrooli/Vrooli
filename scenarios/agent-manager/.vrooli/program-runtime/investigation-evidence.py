"""agent-manager.investigation-evidence v1 — bounded subject evidence collection.

Phases: validate -> collect -> classify -> report. The caller supplies every
subject run explicitly; an unavailable report is not treated as healthy.
"""

import json

inputs = program.inputs()

envelope = {
    "program": "agent-manager.investigation-evidence", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"subjects": [], "subject_count": 0, "available": 0, "unknown": 0},
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
    item = {"run_id": run_id}
    try:
        item["report"] = agent_manager.run.report(run_id=run_id, rows="tools")
    except Exception as exc:
        item["report_error"] = exc
    try:
        item["accounting"] = agent_manager.run.accounting(run_id=run_id)
    except Exception as exc:
        item["accounting_error"] = exc
    return item


def step_collect():
    envelope["phase"] = "collect"
    run_ids = envelope["inputs"]["run_ids"]
    work["subjects"] = gather(*[lambda run_id=run_id: read_subject(run_id) for run_id in run_ids])
    return "classify"


def step_classify():
    subjects = []
    available = 0
    unknown = 0
    for item in work["subjects"]:
        # Report availability says only that the read succeeded. It does not
        # establish that the underlying transcript is fully ingested or
        # current, particularly for imported live sessions.
        subject = {"run_id": item["run_id"], "availability": "complete", "source_completeness": "unknown", "source_freshness": "unknown", "report": {}, "report_availability": "available"}
        missing = 0
        error = item.get("report_error")
        if error is not None:
            subject["report_availability"] = "unknown"
            _, klass = program.classify(error)
            envelope["errors"].append({"class": klass, "detail": str(error)[:180], "where": "collect:" + item["run_id"] + ":report"})
            missing += 1
        if "report" in item:
            report_meta = item["report"].meta()
            timing = report_meta.get("timeAccounting")
            timing = timing if isinstance(timing, dict) else {}
            subject["report"] = {
                "status": report_meta.get("status"), "exit_code": report_meta.get("exitCode"),
                "events_availability": report_meta.get("eventsAvailability"), "receipts_availability": report_meta.get("receiptsAvailability"),
                "duration_ms": report_meta.get("durationMs"), "longest_event_gap_ms": report_meta.get("longestEventGapMs"),
                "repeated_tool_calls": report_meta.get("repeatedToolCalls"),
                "reported_tokens": report_meta.get("tokens"), "requested_model": report_meta.get("requestedModel"),
                "actual_model": report_meta.get("actualModel"), "goal_outcome": report_meta.get("goalOutcome"),
                "time_accounting": {key: timing.get(key) for key in ("modelGeneratingMs", "toolExecutingMs", "idleWaitingMs", "awaitingHumanMs", "unattributableMs")},
            }
            for field in ("eventsAvailability", "receiptsAvailability"):
                availability = report_meta.get(field)
                if not isinstance(availability, dict) or availability.get("state") != "available":
                    subject["report_availability"] = "partial"
                    missing += 1
                    break
        accounting = item.get("accounting")
        if accounting is not None:
            meta = accounting.meta()
            subject["accounting"] = {key: meta.get(key) for key in ("terminal", "tokensKnown", "nonCacheTokens", "cacheReadTokens", "chargeMeasured", "chargeMicroUsd", "wallSeconds", "model")}
            subject["accounting_availability"] = "available"
        else:
            subject["accounting"] = None
            subject["accounting_availability"] = "unknown"
            error = item.get("accounting_error")
            if error is not None:
                _, klass = program.classify(error)
                envelope["errors"].append({"class": klass, "detail": str(error)[:180], "where": "collect:" + item["run_id"] + ":accounting"})
            missing += 1
        any_available = "report" in item or accounting is not None
        if any_available:
            available += 1
        if missing:
            subject["availability"] = "partial" if any_available else "unknown"
            unknown += 1
        subjects.append(subject)
    envelope["signals"] = {"subjects": subjects, "subject_count": len(subjects), "available": available, "unknown": unknown, "coverage": "complete" if unknown == 0 else "partial"}
    envelope["status"] = "ok" if unknown == 0 else ("unavailable" if available == 0 else "partial")
    envelope["evidence"] = list(envelope["inputs"]["run_ids"]) + ["agent-manager/run/report", "agent-manager/run/accounting"]
    return "report"


def step_report():
    envelope["phase"] = "report"
    encoded = json.dumps(envelope, sort_keys=True, separators=(",", ":"))
    if len(encoded.encode()) > 16384:
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
