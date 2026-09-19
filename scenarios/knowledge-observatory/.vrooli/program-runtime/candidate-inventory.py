import json
inputs = program.inputs()

envelope = {"program": "knowledge-observatory.candidate-inventory", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {"candidates": [], "truncated": False, "owner_gaps": []}, "errors": [], "evidence": []}

def fail(kind, detail):
    envelope["status"] = "unavailable" if kind == "scenario_unreachable" else "failed"
    envelope["errors"].append({"class": kind, "detail": str(detail)[:240], "where": envelope["phase"]})
    return "report"

def safe_path(value):
    return isinstance(value, str) and 0 < len(value) <= 500 and not value.startswith(("/", "\\")) and ".." not in value.split("/") and not any(c in value for c in ("\\", ":", "\x00"))

def validate():
    if not isinstance(inputs, dict) or set(inputs) - {"paths", "base_path", "intent", "max_files"}:
        return fail("invalid_input", "unsupported input")
    paths = inputs.get("paths")
    limit = inputs.get("max_files", 100)
    if not isinstance(paths, list) or not 1 <= len(paths) <= 8 or not all(safe_path(p) for p in paths) or not safe_path(inputs.get("base_path")) or type(limit) is not int or not 1 <= limit <= 100:
        return fail("invalid_input", "paths, base_path, and max_files are bounded and required")
    envelope["inputs"] = dict(inputs)
    envelope["phase"] = "collect"
    return "collect"

def collect():
    try:
        review = knowledge_observatory.knowledge_base.review(paths=inputs["paths"], base_path=inputs["base_path"], max_files=inputs.get("max_files", 100), rows="documents")
        rows = review.head(inputs.get("max_files", 100))
        for row in rows:
            path = row.get("path", "")
            status = (row.get("metadata") or {}).get("knowledge_status", "unknown")
            candidate = {"path": path, "sha256": row.get("sha256", ""), "artifact_class": "hypothesis", "owner_hint": path.split("/")[1] if path.startswith("scenarios/") and len(path.split("/")) > 1 else "", "owner_status": "unresolved", "action": "inspect", "inspect_handle": "inspect:" + path, "proposal_handle": "propose:" + path, "evidence_standing": "unknown", "knowledge_status": status, "portability_signals": []}
            envelope["signals"]["candidates"].append(candidate)
            envelope["evidence"].append("path:" + path + "#sha256=" + row.get("sha256", ""))
        envelope["signals"]["truncated"] = bool(review.meta().get("truncated"))
        envelope["status"] = "partial" if envelope["signals"]["truncated"] else "ok"
        envelope["phase"] = "report"
        return "report"
    except Exception as exc:
        return fail("binding_error", exc)

def report():
    if len(json.dumps(envelope).encode("utf-8")) > 60000:
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "output_bound", "detail": "candidate output exceeded bound", "where": "report"})
        envelope["signals"]["candidates"] = envelope["signals"]["candidates"][:20]
        envelope["evidence"] = envelope["evidence"][:20]
    print(json.dumps(envelope, ensure_ascii=False))

state = "validate"
while state:
    if state == "validate": state = validate()
    elif state == "collect": state = collect()
    elif state == "report": report(); state = None
