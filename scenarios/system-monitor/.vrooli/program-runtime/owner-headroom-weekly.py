"""Generate and summarize the weekly per-owner host-headroom report."""

import json

try:
    inputs = program.inputs()
except NameError:
    inputs = {}

report_type = str(inputs.get("report_type", "weekly")).strip()
envelope = {
    "program": "system-monitor.owner-headroom-weekly",
    "version": "2",
    "status": "failed",
    "phase": "validate",
    "inputs": {"report_type": report_type},
    "signals": {},
    "errors": [],
    "evidence": [],
}


def step_validate():
    if report_type != "weekly":
        return program.fail("failed", "invalid_input", "report_type must be weekly", "validate")
    return "act"


def step_act():
    envelope["phase"] = "act"
    try:
        response = system_monitor.reports.generate(type="weekly")
        meta = response.meta()
        report = meta.get("report") if isinstance(meta, dict) else None
        if not isinstance(report, dict):
            return program.fail("failed", "invalid_response", "report generation returned no report object", "act")
        if report.get("reportType") != "weekly" or not report.get("reportId"):
            return program.fail("failed", "invalid_response", "report response lacks the weekly report identity", "act")

        envelope["signals"].update({
            "report_id": report.get("reportId"),
            "report_type": report.get("reportType"),
            "generated_at": report.get("generatedAt"),
        })
        recommendations = report.get("recommendations", [])
        if not isinstance(recommendations, list):
            return program.fail("failed", "invalid_response", "report recommendations is not a list", "act")
        recommendations = [item for item in recommendations if isinstance(item, str)]
        owner_followups = [
            item for item in recommendations
            if item.startswith("Swap coverage is partial") or item.startswith("Owner attribution is incomplete")
        ]
        other_recommendations = [item for item in recommendations if item not in owner_followups]
        action_items = (owner_followups + other_recommendations)[:4]
        envelope["signals"]["action_items"] = action_items
        envelope["signals"]["action_items_truncated"] = len(recommendations) > 4
        owner_resources = report.get("ownerResources")
        if not isinstance(owner_resources, dict):
            envelope["status"] = "partial"
            envelope["signals"]["owner_resources"] = {"available": False}
            envelope["errors"].append({
                "class": "missing_owner_resources",
                "detail": "the report response does not contain owner-resource accounting",
                "where": "act",
            })
            return "report"

        owner_rows = owner_resources.get("owners", [])
        if not isinstance(owner_rows, list):
            return program.fail("failed", "invalid_response", "owner_resources.owners is not a list", "act")

        envelope["signals"]["owner_resources"] = {
            "available": True,
            "window_start": owner_resources.get("windowStart"),
            "window_end": owner_resources.get("windowEnd"),
            "complete": owner_resources.get("complete") is True,
            "attribution_complete": owner_resources.get("attributionComplete") is True,
            "swap_coverage_complete": owner_resources.get("swapCoverageComplete") is True,
            "swap_measured_samples": owner_resources.get("swapMeasuredSamples", 0),
            "swap_unknown_samples": owner_resources.get("swapUnknownSamples", 0),
            "swap_coverage_start": owner_resources.get("swapCoverageStart"),
            "swap_coverage_end": owner_resources.get("swapCoverageEnd"),
            "returned_process_groups": owner_resources.get("returnedProcessGroups", 0),
            "owner_count": len(owner_rows),
            "top_owners": [row for row in owner_rows[:10] if isinstance(row, dict)],
            "truncated": len(owner_rows) > 10,
        }

        if owner_resources.get("complete") is not True:
            envelope["status"] = "partial"
            envelope["errors"].append({
                "class": "incomplete_accounting",
                "detail": "the owner-resource query had no data or reached its process-group cap",
                "where": "act",
            })
        if owner_resources.get("attributionComplete") is not True:
            envelope["status"] = "partial"
            envelope["errors"].append({
                "class": "unknown_owner",
                "detail": "one or more process groups have unknown attribution",
                "where": "act",
            })
        if owner_resources.get("swapCoverageComplete") is not True:
            envelope["status"] = "partial"
            envelope["errors"].append({
                "class": "incomplete_swap_coverage",
                "detail": "swap was not measured for every returned process sample across the report window",
                "where": "act",
            })
        if envelope["status"] == "failed":
            envelope["status"] = "ok"
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, str(exc)[:240], "act")
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


program.run({"validate": step_validate, "act": step_act, "report": step_report}, "validate")
