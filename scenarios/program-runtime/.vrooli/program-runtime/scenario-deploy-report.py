"""Privacy-bounded projection of one scenario deploy report."""

inputs = program.inputs()
scenario = str(inputs.get("scenario") or "").strip()
envelope = {
    "program": "program-runtime.scenario-deploy-report", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"scenario": scenario, "changed": False, "source_role": "", "artifacts": [],
                 "changed_paths": [], "build_key_names": [], "environment_switch_names": [], "truncated": False},
    "errors": [], "evidence": [],
}

if not scenario:
    envelope["errors"].append({"class": "invalid_input", "detail": "scenario is required", "where": "validate"})
else:
    envelope["inputs"] = {"scenario": scenario}
    try:
        handle = vrooli.scenario.deploy_report(name=scenario)
        artifact_rows = handle.head(50)
        report = handle.meta() or {}
        sig = envelope["signals"]
        sig["changed"] = bool(report.get("changed", False))
        sig["source_role"] = str(report.get("sourceRole") or report.get("source_role") or "")
        for item in artifact_rows:
            if isinstance(item, dict):
                sig["artifacts"].append({key: str(item.get(key) or "") for key in ("component", "checkType", "check_type", "baseline", "running")})
        caps = (("changedPaths", "changed_paths"), ("buildKeyNames", "build_key_names"), ("environmentSwitchNames", "environment_switch_names"))
        for source_key, output_key in caps:
            values = report.get(source_key, report.get(output_key, []))
            if not isinstance(values, list):
                continue
            for value in values[:100]:
                if isinstance(value, str) and value and not value.startswith("/") and ".." not in value.split("/"):
                    sig[output_key].append(value)
            if len(values) > 100:
                sig["truncated"] = True
        sig["truncated"] = bool(sig["truncated"] or report.get("truncated", False) or handle.count() > 50)
        envelope["evidence"].append("vrooli/scenario/deploy-report")
        envelope["status"] = "ok"
        envelope["phase"] = "report"
    except Exception:
        envelope["status"] = "unavailable"
        envelope["phase"] = "collect"
        envelope["errors"].append({"class": "scenario_unreachable", "detail": "deploy report unavailable", "where": "collect"})

program.report(envelope)
