import json

inputs = program.inputs()

envelope = {
    "program": "knowledge-observatory.candidate-proposals",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {},
    "signals": {"proposals": [], "count": 0},
    "errors": [],
    "evidence": [],
}




def validate():
    if not isinstance(inputs, dict) or set(inputs) - {"candidates", "reader_task"}:
        return program.fail("failed", 'invalid_input', 'candidates and reader_task are the only inputs', envelope["phase"])
    candidates = inputs.get("candidates")
    if not isinstance(candidates, list) or not 1 <= len(candidates) <= 100:
        return program.fail("failed", 'invalid_input', 'candidates must contain 1..100 records', envelope["phase"])
    if not all(isinstance(candidate, dict) for candidate in candidates):
        return program.fail("failed", 'invalid_input', 'each candidate must be an object', envelope["phase"])
    envelope["inputs"] = dict(inputs)
    envelope["phase"] = "collect"
    return "collect"


def collect():
    for candidate in inputs["candidates"]:
        proposal = {
            "path": str(candidate.get("path", "")),
            "owner_hint": str(candidate.get("owner_hint", "")),
            "action": "review",
            "status": "proposed",
            "evidence_standing": str(candidate.get("evidence_standing", "unknown")),
            "requires_owner_authorization": True,
        }
        envelope["signals"]["proposals"].append(proposal)
        if proposal["path"]:
            envelope["evidence"].append("candidate:" + proposal["path"])
    envelope["signals"]["count"] = len(envelope["signals"]["proposals"])
    envelope["status"] = "ok"
    envelope["phase"] = "report"
    return "report"


def report():
    print(json.dumps(envelope, ensure_ascii=False))


state = "validate"
while state:
    if state == "validate":
        state = validate()
    elif state == "collect":
        state = collect()
    elif state == "report":
        report()
        state = None
