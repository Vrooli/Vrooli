"""Prepare one immutable readiness review through its governed binding."""
import json
inputs = program.inputs()
keys = ("scenario", "profile_id", "candidate_commit", "artifact_digest", "channel")
resolved = {key: str(inputs.get(key, "")).strip() for key in keys}
resolved["target"] = str(inputs.get("target", "")).strip()
resolved["candidate_id"] = str(inputs.get("candidate_id", "")).strip()
resolved["destination_revision_id"] = str(inputs.get("destination_revision_id", "")).strip()
raw_authorization_epoch = inputs.get("authorization_epoch", 0)
authorization_epoch_valid = isinstance(raw_authorization_epoch, int) and not isinstance(raw_authorization_epoch, bool) and raw_authorization_epoch >= 0
resolved["authorization_epoch"] = raw_authorization_epoch if authorization_epoch_valid else 0
raw_targets = inputs.get("targets", [])
if raw_targets is None:
    raw_targets = []
targets_valid = isinstance(raw_targets, list)
resolved["targets"] = [str(value).strip() for value in raw_targets if str(value).strip()] if targets_valid else []
if not resolved["targets"] and resolved["target"]:
    resolved["targets"] = [resolved["target"]]
resolved["deliverable"] = str(inputs.get("deliverable", "release readiness"))
resolved["trigger"] = str(inputs.get("trigger", "operator requested readiness review"))
raw_facts = inputs.get("facts", {})
if raw_facts is None:
    raw_facts = {}
facts_valid = isinstance(raw_facts, dict)
resolved["facts"] = {str(key).strip(): str(value).strip() for key, value in raw_facts.items() if str(key).strip() and str(value).strip()} if facts_valid else {}
envelope = {"program": "deployment-manager.readiness-review", "version": "1", "status": "failed", "phase": "validate", "inputs": resolved, "signals": {}, "errors": [], "evidence": []}

fail = program.fail

def step_validate():
    missing = [key for key in keys if not resolved[key]]
    if missing: return fail("failed", "invalid_input", "missing: " + ",".join(missing), "validate")
    if not targets_valid: return fail("failed", "invalid_input", "targets must be an array", "validate")
    if not resolved["targets"]: return fail("failed", "invalid_input", "at least one target is required", "validate")
    if resolved["target"] and resolved["target"] not in resolved["targets"]: return fail("failed", "invalid_input", "target and targets disagree", "validate")
    if not authorization_epoch_valid: return fail("failed", "invalid_input", "authorization_epoch must be a non-negative integer", "validate")
    if not facts_valid: return fail("failed", "invalid_input", "facts must be an object", "validate")
    return "act"

def step_act():
    envelope["phase"] = "act"
    try:
        handle = deployment_manager.readiness_reviews.prepare(scenario=resolved["scenario"], profile_id=resolved["profile_id"], candidate_commit=resolved["candidate_commit"], artifact_digest=resolved["artifact_digest"], targets=resolved["targets"], channel=resolved["channel"], policy_version=2, facts=resolved["facts"], candidate_id=resolved["candidate_id"], destination_revision_id=resolved["destination_revision_id"], authorization_epoch=resolved["authorization_epoch"], deliverable=resolved["deliverable"], trigger=resolved["trigger"], rows="findings")
        meta = handle.meta() or {}
        findings = handle.head(20)
        envelope["signals"] = {"review_key": meta.get("reviewKey"), "review_status": meta.get("status"), "comparison_mode": meta.get("comparisonMode"), "goal_ref": meta.get("goalRef"), "unresolved": handle.count(), "findings": findings, "next_actions": (meta.get("nextActions") or [])[:20]}
        envelope["evidence"] = [value for value in (meta.get("reviewKey"), meta.get("goalRef")) if value]
        envelope["status"] = "ok"
        return "report"
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "act")

def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False))
    return None

STATES = {"validate": step_validate, "act": step_act, "report": step_report}
state = "validate"
program.run(STATES, state)
