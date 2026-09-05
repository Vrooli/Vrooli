import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "knowledge-observatory.verify-change", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {"capture_required": True}, "errors": [], "evidence": []}
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
    if not isinstance(inputs,dict) or set(inputs)-{"expectations","queries","base_path","mode"}:return fail("failed","invalid_input","Unsupported input","validate")
    expected=inputs.get("expectations");queries=inputs.get("queries")
    if not isinstance(expected,list) or not 1<=len(expected)<=8 or not isinstance(queries,list) or not 1<=len(queries)<=3 or not checked_path(inputs.get("base_path")):
        return fail("failed","invalid_input","1..8 source expectations, 1..3 retrieval cases, and base_path required","validate")
    for item in expected:
        if not isinstance(item,dict) or set(item)-{"path","sha256","knowledge_status"} or not checked_path(item.get("path")) or not isinstance(item.get("sha256"),str) or len(item["sha256"])!=64 or any(c not in "0123456789abcdef" for c in item["sha256"]):
            return fail("failed","invalid_input","Each source needs its intended post-change SHA256","validate")
    for q in queries:
        if not isinstance(q,dict) or set(q)-{"query","expected_paths"} or not isinstance(q.get("query"),str) or not 0<len(q["query"].strip())<=2000 or not isinstance(q.get("expected_paths"),list) or not 1<=len(q["expected_paths"])<=5 or not all(checked_path(p) for p in q["expected_paths"]):
            return fail("failed","invalid_input","Each retrieval case needs a query and 1..5 expected paths","validate")
    return "collect"
def step_collect():
    envelope["phase"]="collect"
    envelope["signals"].update({"checks":[],"task_success_verified":False})
    try:
        for item in inputs["expectations"]:
            doc=document(knowledge_observatory.knowledge_base.inspect(path=item["path"],expected_sha256=item["sha256"],limit=1))
            valid=doc.get("sha256")==item["sha256"] and ("knowledge_status" not in item or (doc.get("metadata") or {}).get("knowledge_status")==item["knowledge_status"])
            envelope["signals"]["checks"].append({"kind":"source_revision","path":item["path"],"passed":valid})
            envelope["evidence"].append("path:"+item["path"]+"#sha256="+item["sha256"])
        for q in inputs["queries"]:
            result=knowledge_observatory.knowledge_base.search(query=q["query"],scope="path",target=inputs["base_path"],mode=inputs.get("mode","auto"),limit=5)
            paths=[r.get("path") for r in result.head(5)]
            envelope["signals"]["checks"].append({"kind":"retrieval","expected_paths":q["expected_paths"],"returned_paths":paths,"method":result.meta().get("method"),"passed":all(p in paths for p in q["expected_paths"])})
        health=knowledge_observatory.knowledge_base.health(scope="path-exact",path=inputs["base_path"],skip_external_links=True,checks=["links","refs"],rows="referenceFindings")
        envelope["signals"]["reference_findings"]=health.head(12)
        envelope["signals"]["content_findings"]=health.meta().get("contentFindings",[])[:12]
        finding_count=health.count()+len(health.meta().get("contentFindings",[]))
        envelope["signals"]["reference_findings_count"]=finding_count
        envelope["signals"]["checks"].append({"kind":"reference_health","passed":finding_count==0})
        if not all(c["passed"] for c in envelope["signals"]["checks"]):return fail("failed","verification_failed","At least one explicit post-change check failed","collect")
        envelope["status"]="ok"
    except Exception as exc:
        if "source revision changed" in str(exc):return fail("failed","revision_changed","Source changed; inspect again before any edit","collect")
        status,klass=classify_transport(exc);return fail(status,klass,exc,"collect")
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
