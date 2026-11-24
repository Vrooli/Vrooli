"""Bounded status-then-wait lifecycle read for one named scenario."""

inputs = program.inputs()
scenario = str(inputs.get("scenario") or "").strip()
timeout = inputs.get("timeout_seconds", 90)
envelope = {
    "program": "program-runtime.scenario-wait", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"scenario": scenario, "lifecycle_status": "unknown", "health": "unknown",
                 "verdict": "unknown", "exit_code": 1, "source": "", "waited_seconds": 0},
    "errors": [], "evidence": [],
}

def mark_error(status, klass, detail, where):
    envelope["status"] = status
    envelope["phase"] = where
    envelope["errors"].append({"class": klass, "detail": detail, "where": where})

if not scenario:
    mark_error("failed", "invalid_input", "scenario is required", "validate")
elif isinstance(timeout, bool) or not isinstance(timeout, int) or timeout < 1 or timeout > 90:
    mark_error("failed", "invalid_input", "timeout_seconds must be from 1 through 90", "validate")
else:
    envelope["inputs"] = {"scenario": scenario, "timeout_seconds": timeout}
    try:
        row = vrooli.scenario.status(name=scenario).meta() or {}
        if isinstance(row.get("scenario"), dict):
            row = row["scenario"]
        envelope["signals"]["lifecycle_status"] = str(row.get("status") or row.get("lifecycleStatus") or "unknown")
        envelope["signals"]["health"] = str(row.get("healthStatus") or row.get("health_status") or row.get("health") or "unknown")
        envelope["evidence"].append("vrooli/scenario/status")
    except Exception:
        mark_error("unavailable", "scenario_unreachable", "status read unavailable", "collect")
    else:
        try:
            row = vrooli.scenario.wait(name=scenario, timeout=timeout).meta() or {}
            if isinstance(row.get("scenario"), dict):
                row = row["scenario"]
            verdict = str(row.get("verdict") or "unknown")
            exit_code = row.get("exitCode", row.get("exit_code", 1))
            if isinstance(exit_code, bool) or not isinstance(exit_code, int) or exit_code not in (0, 1, 2, 124):
                exit_code = 1
            envelope["signals"].update({
                "verdict": verdict,
                "exit_code": exit_code,
                "source": str(row.get("source") or ""),
                "waited_seconds": max(0, min(90, int(row.get("waitedSeconds", row.get("waited_seconds", 0)) or 0))),
            })
            envelope["evidence"].append("vrooli/scenario/wait")
            envelope["status"] = "partial" if exit_code == 124 else ("ok" if exit_code in (0, 2) else "failed")
            envelope["phase"] = "report"
        except Exception:
            mark_error("unavailable", "binding_error", "wait read unavailable", "collect")

program.report(envelope)
