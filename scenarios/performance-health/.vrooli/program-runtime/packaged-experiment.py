"""Run one complete packaged-experiment.v1 matrix using bounded owner bindings."""
import json

inputs = program.inputs()
envelope = {
    "program": "performance-health.packaged-experiment", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {},
    "signals": {"experiment_id": "", "expected_launches": 480, "completed_launches": 0,
                "completed_cells": 0, "built_artifacts": 0, "built_pipeline_ids": [],
                "failed_cells": [], "report_json": "",
                "report_markdown": "", "missing_data": []},
    "errors": [], "evidence": [],
}
manifest = None
operations = []


def step_validate():  # VALIDATE · local shape check only; no binding calls
    envelope["phase"] = "validate"
    try:
        global manifest
        manifest = json.loads(inputs.get("manifest_json", ""))
        if not isinstance(manifest, dict):
            raise ValueError("manifest_json must contain an object")
        envelope["signals"]["experiment_id"] = str(manifest.get("experiment_id", ""))
        if not envelope["signals"]["experiment_id"]:
            raise ValueError("experiment_id is required")
        if manifest.get("platform") != "linux-x64":
            raise ValueError("performance-health.packaged-experiment currently builds and measures linux-x64 cells only")
    except Exception as exc:
        envelope["errors"].append({"class": "invalid_input", "detail": str(exc)[:240], "where": "validate"})
        return "report"
    return "collect"


def step_collect():  # COLLECT · governed validation read
    envelope["phase"] = "collect"
    try:
        handle = performance_health.packaged_experiment.validate_experiment(manifest_json=json.dumps(manifest, separators=(",", ":")))
        row = handle.head(1)[0]
        if not row.get("valid", False):
            envelope["errors"].append({"class": "invalid_input", "detail": str(row.get("error", "owner rejected manifest"))[:240], "where": "collect"})
            return "report"
    except Exception as exc:
        if isinstance(exc, (NameError, AttributeError)):
            raise
        status, klass = program.classify(exc)
        envelope["status"] = status
        envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": "collect"})
        return "report"
    envelope["inputs"] = {"experiment_id": envelope["signals"]["experiment_id"], "subjects": len(manifest.get("subjects", []))}
    envelope["evidence"].append("performance-health/packaged-experiment/validate-experiment")
    return "decide"


def step_decide():  # DECIDE · pure, bounded sequential action plan
    envelope["phase"] = "decide"
    global operations
    operations = [
        (subject, cell)
        for subject in manifest["subjects"]
        for cell in subject["cells"]
    ]
    if len(operations) != 24:
        envelope["errors"].append({"class": "invalid_input", "detail": f"validated matrix expanded to {len(operations)} build cells; expected 24", "where": "decide"})
        return "report"
    return "act"


def step_act():  # ACT · build through the ramp, then capture both conditions serially
    envelope["phase"] = "act"
    for subject, cell in operations:
        profile = cell.get("toggles", {})
        identity = f"{subject['scenario']}:{cell['strategy']}"
        try:
            # Artifact paths and digests are always sourced from this run's
            # pipeline and capture receipt; caller-provided values are ignored.
            cell.pop("artifact_path", None)
            cell.pop("artifact_digest", None)
            build = scenario_to_desktop.pipeline.run(
                scenario=subject["scenario"], strategy=cell["strategy"],
                shared_runtime_readiness=bool(profile.get("shared_runtime_readiness", False)),
                secure_custom_scheme_ui=bool(profile.get("secure_custom_scheme_ui", False)),
                v8_snapshot_code_cache=bool(profile.get("v8_snapshot_code_cache", False)),
                electron_fuses=bool(profile.get("electron_fuses", False)), skip_smoke_test=True)
            pipeline_id = build.meta().get("pipelineId", "")
            if not pipeline_id:
                raise ValueError("scenario-to-desktop did not return a pipelineId")
            pipeline = scenario_to_desktop.pipeline.wait(pipeline_id=pipeline_id)
            artifacts = pipeline.meta().get("finalArtifacts", {})
            linux_artifacts = [str(value) for value in artifacts.values()
                               if isinstance(value, str) and value.lower().endswith(".appimage")]
            if len(linux_artifacts) != 1:
                raise ValueError(f"pipeline {pipeline_id} returned {len(linux_artifacts)} Linux AppImage artifacts; expected exactly one")
            cell["artifact_path"] = linux_artifacts[0]
            cell["artifact_digest"] = ""
            envelope["signals"]["built_artifacts"] += 1
            envelope["signals"]["built_pipeline_ids"].append(pipeline_id)
            envelope["evidence"].append("scenario-to-desktop/pipeline/run:" + pipeline_id)
            for condition in manifest["modes"]:
                request = {
                    "scenario": subject["scenario"], "ramp": "desktop",
                    "strategy": cell["strategy"], "optimization_toggles": profile,
                    "runs": manifest["sample_count"], "condition": condition,
                    "artifact_path": cell["artifact_path"], "experiment_id": manifest["experiment_id"],
                    "journey": subject["journey_id"],
                }
                handle = performance_health.packaged_experiment.capture_experiment_cell(request_json=json.dumps(request, separators=(",", ":")))
                response = json.loads(handle.head(1)[0]["resultJson"])
                count = len(response.get("samples", []))
                if count != manifest["sample_count"]:
                    raise ValueError(f"owner returned {count} samples; expected {manifest['sample_count']}")
                envelope["signals"]["completed_launches"] += count
                envelope["signals"]["completed_cells"] += 1
                envelope["evidence"].append(identity + ":" + condition)
        except Exception as exc:
            if isinstance(exc, (NameError, AttributeError)):
                raise
            status, klass = program.classify(exc)
            envelope["status"] = "partial" if envelope["signals"]["completed_cells"] else status
            envelope["errors"].append({"class": klass, "detail": f"{identity}: {str(exc)[:180]}", "where": "act"})
            envelope["signals"]["failed_cells"].append(identity)
            return "report"
    return "report"


def step_report():  # REPORT · read owner report; preserve missing evidence
    envelope["phase"] = "report"
    if envelope["signals"]["completed_cells"]:
        try:
            handle = performance_health.packaged_experiment.experiment_report(query_json=json.dumps({"experiment_id": envelope["signals"]["experiment_id"]}, separators=(",", ":")))
            row = handle.head(1)[0]
            envelope["signals"]["report_json"] = row.get("reportJson", "")
            envelope["signals"]["report_markdown"] = row.get("markdown", "")
            if envelope["signals"]["report_json"]:
                parsed = json.loads(envelope["signals"]["report_json"])
                envelope["signals"]["missing_data"] = parsed.get("missing_data", [])[:96]
            envelope["evidence"].append("performance-health/packaged-experiment/experiment-report")
        except Exception as exc:
            if isinstance(exc, (NameError, AttributeError)):
                raise
            status, klass = program.classify(exc)
            envelope["errors"].append({"class": klass, "detail": str(exc)[:180], "where": "report"})
    if envelope["signals"]["completed_launches"] == 480 and not envelope["errors"]:
        envelope["status"] = "ok"
    elif envelope["signals"]["completed_launches"] > 0 and envelope["status"] == "failed":
        envelope["status"] = "partial"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "decide": step_decide,
             "act": step_act, "report": step_report}, "validate")
