import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "knowledge-observatory.prepare-change", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {"capture_required": True}, "errors": [], "evidence": []}
def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"
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

def document(handle):
    value = dict(handle.meta())
    value["references"] = handle.head(12)
    value["referencesTruncated"] = bool(value.get("referencesTruncated") or handle.count() > 12)
    return value

def checked_path(value):
    return isinstance(value, str) and 0 < len(value) <= 500 and not value.startswith("/") and not any(c in value for c in ["\\", ":", "\x00"]) and ".." not in value.split("/")

def scalar(handle):
    rows = handle.head(1)
    return rows[0] if rows else {}

def step_validate():
    if not isinstance(inputs,dict) or set(inputs)-{"paths","base_path","intent","max_files"}:return fail("failed","invalid_input","Unsupported input","validate")
    paths=inputs.get("paths")
    if not isinstance(paths,list) or not 1 <= len(paths) <= 8 or not all(checked_path(p) for p in paths) or not checked_path(inputs.get("base_path")):
        return fail("failed","invalid_input","Select 1..8 repository-relative paths and a bounded scan directory","validate")
    if not isinstance(inputs.get("intent"),str) or not 0<len(inputs["intent"].strip())<=2000 or type(inputs.get("max_files",300)) is not int or not 1<=inputs.get("max_files",300)<=1000:
        return fail("failed","invalid_input","Intent and max_files 1..1000 required","validate")
    return "collect"
def step_collect():
    envelope["phase"]="collect"
    try:
        review=knowledge_observatory.knowledge_base.review(paths=inputs["paths"],base_path=inputs["base_path"],max_files=inputs.get("max_files",300),rows="documents")
        docs=review.head(8);meta=review.meta()
        for doc in docs:
            refs=doc.get("references",[]);doc["references"]=refs[:12];doc["referencesTruncated"]=bool(doc.get("referencesTruncated") or len(refs)>12)
        envelope["signals"].update({"documents":docs,"preconditions":[{"path":d["path"],"sha256":d["sha256"]} for d in docs],"observations":meta.get("observations",[])[:100],"files_checked":meta.get("filesChecked",0),"truncated":bool(meta.get("truncated")),"gaps":meta.get("gaps",[])[:12],"decisions_required":[]})
        for d in docs:
            status=(d.get("metadata") or {}).get("knowledge_status","unknown")
            decision="Inspect for unique accepted knowledge to promote; retain necessary evidence with its plan owner; retire only after verified preservation." if status=="supplemental" else "Review authority, overlap, semantic conflicts, and applicability before correcting or consolidating."
            envelope["signals"]["decisions_required"].append({"path":d["path"],"review":decision})
            envelope["evidence"].append("path:"+d["path"]+"#sha256="+d["sha256"])
        related=knowledge_observatory.knowledge_base.search(query=inputs["intent"],scope="path",target=inputs["base_path"],limit=5)
        envelope["signals"]["related_sources"]=[{"path":r.get("path"),"title":r.get("title"),"snippet":r.get("snippet","")[:280]} for r in related.head(5)]
        envelope["status"]="partial" if meta.get("truncated") else "ok"
    except Exception as exc:
        status,klass=classify_transport(exc);return fail("partial" if envelope["signals"].get("documents") else status,klass,exc,"collect")
    return "report"

def step_report():
    envelope["phase"] = "report"
    if len(json.dumps(envelope, ensure_ascii=False).encode("utf-8")) > 60000:
        envelope["status"] = "partial"
        envelope["signals"] = {"capture_required": True, "output_truncated": True, "next_step": "Narrow the source family or read one source page; this response cannot establish completeness."}
        envelope["evidence"] = [str(e)[:512] for e in envelope["evidence"][:8]]
        envelope["errors"] = [{"class": "output_bound", "detail": "Evidence exceeds the declared response budget", "where": "report"}]
    print(json.dumps(envelope, ensure_ascii=False))
    return None
STATES = {"validate": step_validate, "collect": step_collect, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
