"""Read or advance one durable learning finding; never replay the original task."""
import json

inputs = program.inputs()
envelope = program.envelope("program-runtime.learning-maintain", "1")
try:
    action = inputs.get("action", "list")
    owner = inputs.get("owner", "")
    if action == "list":
        response = tasks.learning_findings(owner=owner).head(1)[0]
        findings = response.get("findings", [])
        # Queue readers receive identities, never another caller's feedback or claim authority.
        envelope["signals"] = {"findings": [{k: v for k, v in f.items()
            if k not in ("feedback_ref", "claim_ref")} for f in findings[:100]],
            "truncated": bool(response.get("truncated")) or len(findings) > 100}
        envelope["status"] = "partial" if envelope["signals"]["truncated"] else "ok"
    elif action == "advance":
        if not owner or not inputs.get("finding_id"):
            raise ValueError("advance requires owner and finding_id")
        response = tasks.learning_finding_transition(
            finding_id=inputs["finding_id"], owner=owner,
            expected_state=inputs["expected_state"], next_state=inputs["next_state"],
            evidence=inputs.get("evidence", []), claim_ref=inputs.get("claim_ref", "")).head(1)[0]
        envelope["signals"] = response
        envelope["evidence"] = inputs.get("evidence", [])[:16]
        envelope["status"] = "partial" if response.get("status") == "unavailable" else "ok"
    else:
        raise ValueError("action must be list or advance")
except Exception as exc:
    status, kind = program.classify(exc)
    envelope["status"] = status
    envelope["errors"] = [{"class": kind, "detail": str(exc)[:240]}]
envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True))
