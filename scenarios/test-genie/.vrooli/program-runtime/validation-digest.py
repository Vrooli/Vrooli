"""Bounded, read-only digest of Test Genie validation receipts."""
import json

try:
    inputs
except NameError:
    inputs = {}

limit = int(inputs.get("limit", 100))
caller_scenario = str(inputs.get("caller_scenario", "")).strip()
envelope = {
    "program": "test-genie.validation-digest", "version": "1", "status": "failed",
    "phase": "validate", "inputs": {"limit": limit, "caller_scenario": caller_scenario},
    "signals": {"receipts": 0, "states": {}, "terminal": 0, "active": 0,
                "with_evidence": 0, "retry_pending": 0, "degraded": 0},
    "errors": [], "evidence": [],
}

if limit < 1 or limit > 500:
    envelope["errors"].append({"class": "invalid_input", "detail": "limit must be within [1, 500]", "where": "validate"})
else:
    try:
        envelope["phase"] = "collect"
        receipts = test_genie.validation.list(caller_scenario=caller_scenario, page_size=limit).head(limit)
        envelope["phase"] = "classify"
        terminal_states = {"RECEIPT_STATE_SUCCEEDED", "RECEIPT_STATE_FAILED", "RECEIPT_STATE_DEGRADED", "RECEIPT_STATE_CANCELLED", "RECEIPT_STATE_SUPERSEDED"}
        states = {}
        for receipt in receipts:
            state = str(receipt.get("state") or "RECEIPT_STATE_UNSPECIFIED")
            states[state] = states.get(state, 0) + 1
        signals = envelope["signals"]
        signals["receipts"] = len(receipts)
        signals["states"] = dict(sorted(states.items()))
        signals["terminal"] = sum(count for state, count in states.items() if state in terminal_states)
        signals["active"] = len(receipts) - signals["terminal"]
        signals["with_evidence"] = sum(1 for receipt in receipts if receipt.get("evidence"))
        signals["retry_pending"] = states.get("RECEIPT_STATE_RETRY_PENDING", 0)
        signals["degraded"] = states.get("RECEIPT_STATE_DEGRADED", 0)
        envelope["evidence"].append(f"test-genie validation list --page-size {limit}")
        envelope["status"] = "ok"
    except Exception as exc:
        text = str(exc)
        klass = "scenario_unreachable" if any(value in text.lower() for value in ("unreachable", "connection refused", "404 not found")) else "binding_error"
        envelope["status"] = "unavailable" if klass == "scenario_unreachable" else "failed"
        envelope["errors"].append({"class": klass, "detail": text[:240], "where": envelope["phase"]})

envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
