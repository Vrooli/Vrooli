"""Deterministic grounding guard for advisory output."""
import json

try:
    inputs
except NameError:
    inputs = {}

env = {"program": "git-control-tower.advisory", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {}, "errors": [], "evidence": []}
modes = {"summary", "commit-draft", "pr-draft", "review", "release-draft", "provenance"}

try:
    if not isinstance(inputs, dict):
        env["errors"].append({"class": "invalid_input", "detail": "inputs must be an object", "where": env["phase"]})
    else:
        mode, digest, claims = inputs.get("mode"), inputs.get("subject_digest"), inputs.get("claims")
        if mode not in modes:
            env["errors"].append({"class": "unsupported_mode", "detail": "mode is not an advisory mode", "where": env["phase"]})
        elif not isinstance(digest, str) or not digest.startswith("sha256:") or not isinstance(claims, list):
            env["errors"].append({"class": "invalid_input", "detail": "subject_digest and claims are required", "where": env["phase"]})
        else:
            for claim in claims:
                if not isinstance(claim, dict) or not claim.get("text") or not isinstance(claim.get("evidence_refs"), list) or not claim["evidence_refs"]:
                    env["errors"].append({"class": "ungrounded_claim", "detail": "every claim needs evidence_refs", "where": env["phase"]})
                    break
            unknowns = inputs.get("unknowns", [])
            if not isinstance(unknowns, list):
                env["errors"].append({"class": "invalid_input", "detail": "unknowns must be an array", "where": env["phase"]})
            elif not env["errors"]:
                env["inputs"] = {"mode": mode, "subject_digest": digest}
                env["signals"] = {"mode": mode, "result_kind": mode, "claim_count": len(claims), "unknowns": unknowns}
                env["evidence"] = sorted({ref for claim in claims for ref in claim["evidence_refs"]})
                env["status"] = "partial" if unknowns else "ok"
except Exception as exc:
    env["errors"].append({"class": "invalid_input", "detail": str(exc)[:240], "where": env["phase"]})
env["phase"] = "report"
print(json.dumps(env, sort_keys=True, separators=(",", ":")))
