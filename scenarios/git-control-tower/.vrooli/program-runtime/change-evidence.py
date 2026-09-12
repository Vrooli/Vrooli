"""Deterministic, read-only ChangeSubject/evidence envelope."""
import json

inputs = program.inputs()

env = {"program": "git-control-tower.change-evidence", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {}, "errors": [], "evidence": []}
try:
    if not isinstance(inputs, dict) or not str(inputs.get("operation_id", "")).strip():
        env["errors"].append({"class": "invalid_input", "detail": "operation_id is required", "where": env["phase"]})
    else:
        subject, coverage = inputs.get("subject"), inputs.get("coverage")
        if not isinstance(subject, dict) or not isinstance(coverage, dict):
            env["errors"].append({"class": "invalid_input", "detail": "subject and coverage must be objects", "where": env["phase"]})
        elif any(not str(subject.get(key, "")).strip() for key in ("repository_id", "kind", "snapshot_digest")):
            env["errors"].append({"class": "ambiguous_subject", "detail": "subject identity is incomplete", "where": env["phase"]})
        else:
            scope = subject.get("scope")
            if not isinstance(scope, dict) or not isinstance(scope.get("paths"), list) or not scope["paths"] or not scope.get("selection_digest"):
                env["errors"].append({"class": "ambiguous_subject", "detail": "subject scope is incomplete", "where": env["phase"]})
            else:
                included, omitted, omissions = coverage.get("included_files"), coverage.get("omitted_files"), coverage.get("omissions", [])
                if not isinstance(included, int) or not isinstance(omitted, int) or included < 0 or omitted < 0 or not isinstance(omissions, list) or omitted != len(omissions):
                    env["errors"].append({"class": "invalid_input", "detail": "coverage does not account for omissions", "where": env["phase"]})
                else:
                    env["inputs"] = {"operation_id": inputs["operation_id"], "kind": subject["kind"]}
                    env["signals"] = {"subject_digest": subject["snapshot_digest"], "coverage": {"included_files": included, "omitted_files": omitted, "omissions": omissions}}
                    env["evidence"].append("caller-supplied immutable subject and coverage")
                    env["status"] = "partial" if omitted else "ok"
except Exception as exc:
    env["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": env["phase"]})
env["phase"] = "report"
print(json.dumps(env, sort_keys=True, separators=(",", ":")))
