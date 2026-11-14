"""Read one immutable supervision policy and optional bounded outcome coverage."""

import json

NAME = "agent-manager.supervision-experiment-read"
KEYS = ["policy_version"]
inputs = program.inputs()
envelope = {
    "program": NAME,
    "version": "1",
    "status": "ok",
    "phase": "validate",
    "inputs": {},
    "signals": {},
    "errors": [],
    "evidence": [],
}
work = {}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:180], "where": where})
    return "report"


def safe(call):
    try:
        return call(), None
    except Exception as exc:
        return None, exc


def step_validate():
    if (
        not isinstance(inputs, dict)
        or set(inputs) - set(KEYS)
        or any(not isinstance(inputs.get(key), str) or not inputs[key].strip() or len(inputs[key]) > 128 for key in KEYS)
    ):
        return fail("failed", "invalid_input", "Supply one bounded nonempty policy version", "validate")
    envelope["inputs"] = dict(inputs)
    return "collect"


def step_collect():
    (policy, policy_error), (outcomes, outcomes_error) = gather(
        lambda: safe(lambda: agent_manager.watch.policy_get(version=inputs["policy_version"])),
        lambda: safe(lambda: agent_manager.watch.policy_outcomes(policy_version=inputs["policy_version"], limit=32)),
    )
    if policy_error is not None:
        status, klass = program.classify(policy_error)
        envelope["status"] = status
        envelope["errors"].append({"class": klass, "detail": f"policy: {str(policy_error)[:150]}", "where": "collect"})
        return "report"
    if outcomes_error is not None:
        _, klass = program.classify(outcomes_error)
        envelope["status"] = "partial"
        envelope["errors"].append({"class": klass, "detail": f"outcomes: {str(outcomes_error)[:150]}", "where": "collect"})
    work.update(policy=policy, outcomes=outcomes)
    return "classify"


def step_classify():
    record = work["policy"].meta()
    policy = record.get("policy") or {}
    if policy.get("version") != inputs["policy_version"]:
        return fail("failed", "identity_mismatch", "policy response mismatch", "classify")

    outcomes_available = work["outcomes"] is not None
    rows = work["outcomes"].head(32) if outcomes_available else []
    evaluation = record.get("evaluation")
    signals = {
        "state": record.get("state"),
        "policy_digest": record.get("digest"),
        "evaluator_digest": policy.get("evaluatorDigest"),
        "inference_identity_digest": record.get("inferenceIdentityDigest"),
        "evaluation": evaluation,
        "coverage": work["outcomes"].meta().get("coverage") if outcomes_available else "unavailable",
        "outcomes_available": outcomes_available,
        "assessed_sample": sum(bool(row.get("observedClass")) for row in rows) if outcomes_available else None,
        "unassessed_sample": sum(not row.get("observedClass") for row in rows) if outcomes_available else None,
        "sample_family_count": len({row.get("familyExecutionId") for row in rows if row.get("familyExecutionId")}) if outcomes_available else None,
        "sample_may_be_truncated": len(rows) == 32 if outcomes_available else None,
        "next_action": "retry_outcome_read" if not outcomes_available else ("collect_assessed_evidence" if not evaluation else "inspect_owner_gates"),
        "promotion_authority": "agent-manager; this read does not promote or establish causal benefit",
    }
    envelope["signals"] = signals
    envelope["evidence"] = [inputs["policy_version"]]
    return "report"


def step_report():
    envelope["phase"] = "report"
    encoded = json.dumps(envelope, sort_keys=True, separators=(",", ":"))
    if len(encoded.encode()) > 4096:
        envelope["status"] = "failed"
        envelope["signals"] = {}
        envelope["errors"] = [{"class": "output_budget_exhausted", "detail": "owner summary exceeds bound", "where": "report"}]
        encoded = json.dumps(envelope, separators=(",", ":"))
    print(encoded)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        envelope["phase"] = state
        state = STATES[state]()
    except Exception as exc:
        if state == "report":
            raise
        try:
            status, klass = program.classify(exc)
        except (NameError, AttributeError):
            status, klass = "failed", "kernel_runtime"
        state = fail(status, klass, exc, state)
