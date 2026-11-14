inputs = program.inputs()
scenario = str(inputs.get("scenario", "")).strip()
source_root = str(inputs.get("source_root", "")).strip()
envelope = {"program": "scenario-to-repository.source-export-preflight", "version": "1", "status": "failed", "phase": "validate", "evidence": [], "errors": [], "publication": "human-only"}
if not scenario or not source_root:
    envelope["errors"].append({"class": "invalid_input", "detail": "scenario and source_root are required"})
else:
    envelope["status"] = "ok"
    envelope["phase"] = "report"
    envelope["evidence"] = [scenario, source_root]
print(envelope)
