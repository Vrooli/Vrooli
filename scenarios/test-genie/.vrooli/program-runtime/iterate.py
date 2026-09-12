"""Submit one scoped validation intent; Test Genie owns planning and execution.

Required checks never disappear to meet a budget. With no explicit check set,
the owner's targeted/quick profile uses its historical cost planner. This is
iteration, not certification. A returned receipt is not a passing test result.
"""
import json
import re
inputs = program.inputs()
envelope = {"program": "test-genie.iterate", "version": "1", "status": "failed", "phase": "validate",
            "signals": {"outcome": "unknown"}, "errors": [], "evidence": []}
try:
    scenario = inputs["scenario"]
    key = inputs["request_id"]
    phases = inputs.get("phases", [])
    budget = inputs.get("budget_seconds", 600)
    if not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,127}", scenario):
        raise ValueError("scenario must be an exact scenario ID")
    if not isinstance(key, str) or not key.strip() or len(key) > 128:
        raise ValueError("request_id must identify this validation request; reuse it only for reattachment")
    if not isinstance(phases, list) or len(phases) > 32 or any(
            not isinstance(p, str) or not re.fullmatch(r"[a-z][a-z0-9-]{0,63}", p) for p in phases):
        raise ValueError("phases must contain at most 32 exact phase names")
    if type(budget) is not int or not 30 <= budget <= 3600:
        raise ValueError("budget_seconds must be 30..3600")
    phases = sorted(set(phases))
    intent = {"schema_version": 1, "idempotency_key": key, "caller_scenario": "test-genie",
              "caller_execution_id": key,
              "targets": [{"kind": "VALIDATION_TARGET_KIND_SCENARIO", "id": scenario, "root": "scenarios/" + scenario}],
              "purpose": "VALIDATION_PURPOSE_INVESTIGATION", "required_strength": "VALIDATION_STRENGTH_TARGETED",
              "phases": phases, "reuse_policy": {"mode": "REUSE_MODE_ATTACH_OR_TERMINAL", "maximum_age": "3600s"},
              "concurrency_policy": {"mode": "CONCURRENCY_MODE_SHARED_COMPATIBLE", "maximum_parallelism": 1},
              "evidence_policy": {"required_evidence_kinds": ["test-genie-run"]},
              "deadline_policy": {"queue_budget": "300s", "execution_budget": str(budget) + "s", "maximum_attempts": 1},
              "content_inputs": [{"name": "scenario", "root": "scenarios/" + scenario,
                                  "selections": [{"glob": "**", "required": True}]}]}
    extra = inputs.get("dependency_inputs", [])
    if not isinstance(extra, list) or len(extra) > 16:
        raise ValueError("at most 16 explicit dependency input roots")
    for dependency in extra:
        if not isinstance(dependency, dict) or dependency.get("dependency") is not True:
            raise ValueError("additional content roots must be marked dependency=true")
    intent["content_inputs"] += extra
    envelope["signals"].update(intent=intent, selection={
        "mode": "explicit_required" if phases else "owner_budget_profile",
        "phases": phases, "explanation": "Explicit phases are mandatory; otherwise Test Genie selects its history-informed quick profile. Budget exhaustion is incomplete validation, never permission to omit required checks."})
    envelope["status"] = "ok"
    if not inputs.get("plan_only", False):
        envelope["phase"] = "act"
        handle = test_genie.validation.create(intent=intent)
        rows = handle.head(1)
        receipt = rows[0] if rows else handle.meta().get("receipt", {})
        receipt = receipt.get("receipt", receipt)
        receipt_id = receipt.get("receiptId", "")
        if not re.fullmatch(r"[A-Za-z0-9_-]{1,128}", receipt_id):
            raise RuntimeError("owner returned no valid receipt ID")
        # The owner retains full file identities and evidence. Returning those
        # inventories can exhaust the kernel output bound and hide the receipt.
        envelope["signals"]["receipt"] = {k: receipt[k] for k in (
            "receiptId", "state", "revision", "reasonCode", "createdAt", "updatedAt", "terminalAt") if k in receipt}
        envelope["signals"]["receipt"]["identity"] = receipt.get("admittedIdentity", {}).get("identity")
        envelope["signals"]["get_command"] = "test-genie validation get " + receipt_id + " --json"
        envelope["signals"]["wait_command"] = "test-genie validation wait " + receipt_id + " --wait-id " + receipt_id + "-iteration --timeout 30m --json"
        envelope["evidence"] = ["test-genie:validation:" + receipt_id]
except Exception as exc:
    envelope["status"] = "failed" if isinstance(exc, (ValueError, KeyError, TypeError)) else "unavailable"
    envelope["errors"] = [{"class": "invalid_input" if envelope["status"] == "failed" else "admission_unavailable",
                            "where": envelope["phase"], "detail": str(exc)[:240]}]
envelope["phase"] = "report"
print(json.dumps(envelope))
