"""Compose attention candidates and explicit investigation evidence for one repair decision.

This workflow gathers evidence. A caller owns selection, repair authority, and
independent validation; neither a candidate nor an agent report grants credit.
"""
import json
try:
    inputs
except NameError:
    inputs = {}

envelope = {"program":"program-runtime.improvement-evidence", "version":"1", "status":"failed", "phase":"validate", "inputs":{}, "signals":{"children":[], "reserved":False, "mutation_authorized":False}, "errors":[], "evidence":[]}

def fail(kind, detail):
    envelope["errors"].append({"class":kind, "detail":str(detail)[:240], "where":envelope["phase"]})
    return "report"

def step_validate():
    if not isinstance(inputs, dict):
        return fail("invalid_input", "inputs must be an object")
    campaign = inputs.get("campaign_id")
    runs = inputs.get("run_ids")
    limit = inputs.get("limit", 3)
    if not isinstance(campaign, str) or not 1 <= len(campaign) <= 36:
        return fail("invalid_input", "campaign_id must be an existing campaign UUID")
    if not isinstance(runs, list) or not 1 <= len(runs) <= 8 or any(not isinstance(run, str) or not run.strip() or len(run) > 128 for run in runs) or len(set(runs)) != len(runs):
        return fail("invalid_input", "run_ids must contain one to eight unique bounded subject IDs")
    if type(limit) is not int or not 1 <= limit <= 5:
        return fail("invalid_input", "limit must be 1..5")
    envelope["inputs"] = {"campaign_id":campaign, "run_ids":list(runs), "limit":limit}
    return "collect"

def read_child(name, call):
    try:
        handle = call()
        rows = handle.head(1)
        if len(rows) != 1 or not isinstance(rows[0], dict):
            raise ValueError("child did not return one envelope")
        row = rows[0]
        if row.get("status") not in ("ok", "partial", "unavailable", "refused", "failed") or not isinstance(row.get("signals"), dict) or not isinstance(row.get("errors"), list):
            raise ValueError("child returned a malformed envelope")
        if len(json.dumps(row, allow_nan=False).encode()) > 50000:
            raise ValueError("child evidence exceeds composition byte budget")
        metadata = handle.meta()
        artifact = {key:metadata[key] for key in ("name", "scenario", "version", "digest") if key in metadata}
        envelope["evidence"].append({"program":name, "artifact":artifact})
        envelope["signals"]["children"].append({"program":name, "status":row["status"], "signals":row["signals"], "errors":row["errors"]})
    except Exception as exc:
        fail("kernel_runtime", name + ": " + str(exc))
        envelope["signals"]["children"].append({"program":name, "status":"failed", "signals":{}, "errors":[]})

def step_collect():
    envelope["phase"] = "collect"
    args = envelope["inputs"]
    read_child("visited-tracker.attention-select", lambda: lib.visited_tracker.attention_select(campaign_id=args["campaign_id"], limit=args["limit"]))
    read_child("agent-manager.investigation-evidence", lambda: lib.agent_manager.investigation_evidence(run_ids=args["run_ids"]))
    children = envelope["signals"]["children"]
    successes = sum(child["status"] == "ok" for child in children)
    envelope["status"] = "ok" if successes == 2 else ("partial" if any(child["status"] in ("ok", "partial") for child in children) else "failed")
    return "report"

def step_report():
    envelope["phase"] = "report"
    encoded = json.dumps(envelope, allow_nan=False, separators=(",", ":"))
    if len(encoded.encode()) > 65536:
        envelope["status"] = "failed"
        envelope["signals"]["children"] = []
        fail("output_limit", "combined evidence exceeds 65536 bytes; inspect the child artifacts separately")
        encoded = json.dumps(envelope, allow_nan=False, separators=(",", ":"))
    print(encoded)
    return None

STATES = {"validate":step_validate, "collect":step_collect, "report":step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope["phase"] == "report":
            raise
        state = fail("kernel_runtime", exc)
