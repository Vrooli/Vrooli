import json
try:
    inputs
except NameError:
    inputs = {}

envelope = {"program": "visited-tracker.attention-select", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {}, "errors": [], "evidence": []}

def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")

def fail(status, kind, detail):
    envelope["status"] = status
    envelope["errors"].append({"class": kind, "detail": str(detail)[:240], "where": envelope["phase"]})
    return "report"

def step_validate():
    campaign = inputs.get("campaign_id", "")
    limit = inputs.get("limit", 10)
    if not isinstance(campaign, str) or not campaign.strip() or len(campaign) > 36 or isinstance(limit, bool) or not isinstance(limit, int) or not 1 <= limit <= 20:
        return fail("failed", "invalid_input", "require campaign_id and limit 1..20")
    envelope["inputs"] = {"campaign_id": campaign, "limit": limit}
    return "collect"

def step_collect():
    envelope["phase"] = "collect"
    try:
        handle = visited_tracker.attention.preview(campaign_id=envelope["inputs"]["campaign_id"], limit=envelope["inputs"]["limit"])
        candidates = []
        truncated = False
        used_bytes = 0
        for candidate in handle.head(envelope["inputs"]["limit"]):
            candidate_bytes = len(json.dumps(candidate).encode("utf-8"))
            if used_bytes + candidate_bytes > 48000:
                truncated = True
                break
            candidates.append(candidate)
            used_bytes += candidate_bytes
        envelope["signals"] = {"candidates": candidates, "reserved": False, "truncated": truncated}
        envelope["evidence"] = [{"binding": "visited-tracker/attention/preview", "observation": handle.meta()}]
        envelope["status"] = "partial" if truncated else "ok"
    except Exception as exc:
        status, kind = classify_transport(exc)
        return fail(status, kind, str(exc))
    return "report"

def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False))
    return None

STATES = {"validate": step_validate, "collect": step_collect, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope["phase"] == "report":
            raise
        state = fail("failed", "kernel_runtime", str(exc))
