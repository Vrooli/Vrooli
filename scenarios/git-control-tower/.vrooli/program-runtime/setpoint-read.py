"""Read-only, honest improvement observations."""
import json
try:
    inputs
except NameError:
    inputs = {}
env = {"program": "git-control-tower.setpoint-read", "version": "1", "status": "failed", "phase": "validate", "signals": {"rows": []}, "errors": [], "evidence": []}
observations = inputs.get("observations") if isinstance(inputs, dict) else None
if not isinstance(observations, list):
    env["errors"].append({"class": "invalid_input", "detail": "observations must be an array", "where": "validate"})
else:
    for row in observations:
        if not isinstance(row, dict) or not row.get("row") or row.get("standing") not in {"in_band", "out_of_band", "unavailable", "unreliable"}:
            env["errors"].append({"class": "unreliable_observation", "detail": "each row needs a known standing", "where": "classify"}); continue
        env["signals"]["rows"].append(row)
        if row.get("evidence"): env["evidence"].append(row["evidence"])
    env["status"] = "ok" if env["signals"]["rows"] and not env["errors"] else "partial"
env["phase"] = "report"
print(json.dumps(env, sort_keys=True, separators=(",", ":")))
