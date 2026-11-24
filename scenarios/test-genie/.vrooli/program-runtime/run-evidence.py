"""Bounded Test Genie run evidence with a governed metadata-only file receipt."""
import json
import re

inputs = program.inputs()
view = inputs.get("view", "")
scenario = inputs.get("scenario", "")
run_id = inputs.get("run_id", "")
envelope = {"program": "test-genie.run-evidence", "version": "1", "status": "failed",
            "phase": "validate", "signals": {"view": view}, "errors": [], "evidence": []}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "where": where, "detail": detail[:240]})
    return "report"


def field(row, snake, camel=None):
    if not isinstance(row, dict):
        return None
    return row.get(snake, row.get(camel)) if camel else row.get(snake)


def safe_artifact(row, legacy=False):
    return {
        "id": str(field(row, "id") or "")[:256],
        "kind": str(field(row, "kind") or "")[:80],
        "media_type": str(field(row, "media_type", "mediaType") or "")[:120],
        "label": str(field(row, "label") or "")[:240],
        "producing_phase": str(field(row, "producing_phase", "producingPhase") or "")[:80],
        "size_bytes": field(row, "size_bytes", "sizeBytes"),
        "created_at": str(field(row, "created_at", "createdAt") or "")[:64],
        "access_capability": field(row, "access_capability", "accessCapability"),
        "legacy_discovered": bool(legacy),
    }


def valid_relative_path(value):
    if not isinstance(value, str) or not value or len(value) > 1024 or value.startswith("/") or "\\" in value:
        return False
    parts = value.split("/")
    return all(part not in ("", ".", "..") for part in parts)


def step_validate():
    if view not in {"show", "artifacts", "findings", "artifact", "download"}:
        return fail("failed", "invalid_input", "view must be show, artifacts, findings, artifact, or download", "validate")
    if not isinstance(scenario, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,127}", scenario):
        return fail("failed", "invalid_input", "scenario must be an exact scenario ID", "validate")
    if not isinstance(run_id, str) or not re.fullmatch(r"[A-Za-z0-9_-]{1,128}", run_id):
        return fail("failed", "invalid_input", "run_id must be an exact owner-issued run ID", "validate")
    if view in {"artifact", "download"} and (not isinstance(inputs.get("artifact_id"), str) or not inputs["artifact_id"]):
        return fail("failed", "invalid_input", "artifact view requires artifact_id copied from the typed catalog", "validate")
    if view == "download":
        if not valid_relative_path(inputs.get("path")):
            return fail("failed", "invalid_input", "path must be a bounded relative file path without traversal", "validate")
        if type(inputs.get("overwrite", False)) is not bool:
            return fail("failed", "invalid_input", "overwrite must be a boolean", "validate")
    envelope["inputs"] = {"view": view, "scenario": scenario, "run_id": run_id}
    if view in {"artifact", "download"}:
        envelope["inputs"]["artifact_id"] = inputs["artifact_id"]
    if view == "download":
        envelope["inputs"].update({"path": inputs["path"], "overwrite": inputs.get("overwrite", False)})
    return "read"


def step_read():
    envelope["phase"] = "collect"
    try:
        if view == "show":
            result = test_genie.runs.show(scenario=scenario, run_id=run_id).head(1)
            binding = "test-genie/runs/show"
            envelope["signals"]["result"] = result
        elif view == "artifacts":
            args = {"scenario": scenario, "run_id": run_id}
            if inputs.get("kinds"):
                args["kinds"] = inputs["kinds"]
            if inputs.get("phase"):
                args["phase"] = inputs["phase"]
            result = test_genie.runs.artifacts(rows="artifacts", **args)
            meta = result.meta() or {}
            rows = result.head(100)
            envelope["signals"]["result"] = [safe_artifact(row) for row in rows]
            envelope["signals"]["metadata"] = {"count": result.count(), "legacy_discovered": bool(meta.get("legacyDiscovered", False)),
                                                  "truncated": result.count() > len(rows)}
            binding = "test-genie/runs/artifacts"
        elif view == "findings":
            result = test_genie.runs.findings(scenario=scenario, run_id=run_id).head(1)
            binding = "test-genie/runs/findings"
            envelope["signals"]["result"] = result
        else:
            handle = test_genie.runs.artifact(scenario=scenario, run_id=run_id, artifact_id=inputs["artifact_id"])
            metadata = handle.meta() or {}
            rows = handle.head(1)
            artifact = rows[0] if rows else metadata.get("artifact") or {}
            if isinstance(artifact, dict) and isinstance(artifact.get("artifact"), dict):
                artifact = artifact["artifact"]
            if field(artifact, "id") != inputs["artifact_id"]:
                return fail("failed", "invalid_response", "owner artifact identity did not match the requested catalog ID", "read")
            legacy = bool(metadata.get("legacyDiscovered", metadata.get("legacy_discovered", False)))
            safe = safe_artifact(artifact, legacy)
            if view == "artifact":
                envelope["signals"]["result"] = safe
            else:
                size_bytes = safe["size_bytes"]
                if isinstance(size_bytes, str) and size_bytes.isdecimal():
                    size_bytes = int(size_bytes)
                capability = safe["access_capability"]
                stream_capability = capability in (1, "1", "ARTIFACT_ACCESS_CAPABILITY_STREAM")
                if type(size_bytes) is not int or size_bytes < 0 or size_bytes > 64 * 1024 * 1024:
                    return fail("failed", "invalid_response", "owner artifact size is missing or outside the 64 MiB transfer bound", "read")
                if not stream_capability:
                    return fail("failed", "invalid_response", "owner artifact does not declare stream access", "read")
                try:
                    receipt = program.write_artifact(
                        scenario=scenario, run_id=run_id, artifact_id=inputs["artifact_id"],
                        path=inputs["path"], overwrite=inputs.get("overwrite", False),
                    )
                except Exception:
                    return fail("partial", "effect_uncertain", "transfer returned no verifiable receipt; reconcile the destination before retrying", "transfer")
                if not isinstance(receipt, dict):
                    return fail("partial", "effect_uncertain", "transfer returned no verifiable receipt; reconcile the destination before retrying", "transfer")
                sha256 = receipt.get("sha256") if isinstance(receipt, dict) else None
                exact = (
                    receipt.get("scenario") == scenario
                    and receipt.get("run_id") == run_id
                    and receipt.get("artifact_id") == inputs["artifact_id"]
                    and receipt.get("kind") == safe["kind"]
                    and receipt.get("output_path") == inputs["path"]
                    and receipt.get("bytes") == size_bytes
                    and isinstance(sha256, str) and re.fullmatch(r"[0-9a-fA-F]{64}", sha256)
                )
                if not exact:
                    return fail("partial", "effect_uncertain", "file may have been written but receipt identity or size did not verify; reconcile before retrying", "transfer")
                envelope["signals"]["receipt"] = {
                    "scenario": scenario, "run_id": run_id, "artifact_id": inputs["artifact_id"],
                    "kind": safe["kind"], "output_path": inputs["path"], "bytes": size_bytes,
                    "sha256": sha256.lower(), "legacy_discovered": bool(receipt.get("legacy_discovered", legacy)),
                    "source_digest_verified": False,
                }
            binding = "test-genie/runs/artifact"
        envelope["evidence"].append(binding)
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, "owner evidence read failed; inspect the source status without exposing artifact paths", envelope["phase"])
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "read": step_read, "report": step_report}, "validate")
