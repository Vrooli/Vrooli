"""Publish a reviewed portable baseline; the learn.act call remains in source."""
import json

inputs = program.inputs()
envelope = program.envelope("program-runtime.fragment-promote", "2")
try:
    result = tasks.fragment_promote(
        step_key=inputs["step_key"], operation=inputs["operation"],
        min_verified=inputs.get("min_verified", 5), reviewed_by=inputs["reviewed_by"],
        evidence=inputs["evidence"], fixtures=inputs["fixtures"],
        publish=inputs.get("publish", False), expected_digest=inputs.get("expected_digest", ""),
    ).head(1)[0]
    envelope["signals"] = result
    envelope["status"] = "ok"
    envelope["evidence"] = inputs["evidence"]
except Exception as exc:
    envelope["status"] = "failed"
    envelope["errors"].append({"class": "publication_failed", "detail": str(exc)[:240]})
envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True))
