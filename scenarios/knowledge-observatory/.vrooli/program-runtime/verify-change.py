import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "knowledge-observatory.verify-change", "version": "2", "status": "failed", "phase": "validate", "inputs": {}, "signals": {"capture_required": True}, "errors": [], "evidence": []}
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
    if not isinstance(inputs,dict) or set(inputs)-{"expectations","queries","base_path","mode","absent_paths","preservation","reference_baseline"}:return fail("failed","invalid_input","Unsupported input","validate")
    expected=inputs.get("expectations");queries=inputs.get("queries")
    if not isinstance(expected,list) or not 1<=len(expected)<=8 or not isinstance(queries,list) or not 1<=len(queries)<=3 or not checked_path(inputs.get("base_path")):
        return fail("failed","invalid_input","1..8 source expectations, 1..3 retrieval cases, and base_path required","validate")
    for item in expected:
        if not isinstance(item,dict) or set(item)-{"path","sha256","knowledge_status"} or not checked_path(item.get("path")) or not isinstance(item.get("sha256"),str) or len(item["sha256"])!=64 or any(c not in "0123456789abcdef" for c in item["sha256"]):
            return fail("failed","invalid_input","Each source needs its intended post-change SHA256","validate")
    for q in queries:
        if not isinstance(q,dict) or set(q)-{"query","expected_paths","forbidden_paths"} or not isinstance(q.get("query"),str) or not 0<len(q["query"].strip())<=2000 or not isinstance(q.get("expected_paths"),list) or not 1<=len(q["expected_paths"])<=5 or not all(checked_path(p) for p in q["expected_paths"]):
            return fail("failed","invalid_input","Each retrieval case needs a query and 1..5 expected paths","validate")
        if not isinstance(q.get("forbidden_paths",[]),list) or len(q.get("forbidden_paths",[]))>8 or not all(checked_path(p) for p in q.get("forbidden_paths",[])):
            return fail("failed","invalid_input","Invalid forbidden retrieval paths","validate")
    absent=inputs.get("absent_paths",[])
    if not isinstance(absent,list) or len(absent)>8 or not all(checked_path(p) for p in absent) or any(p in [e["path"] for e in expected] for p in absent):
        return fail("failed","invalid_input","At most 8 absent paths, distinct from expected sources","validate")
    preservation=inputs.get("preservation",[])
    if not isinstance(preservation,list) or len(preservation)>12:
        return fail("failed","invalid_input","At most 12 preservation checks","validate")
    ids=[]
    for c in preservation:
        if not isinstance(c,dict) or set(c)-{"id","path","offset","excerpt","review","evidence_ref"} or c.get("path") not in [e["path"] for e in expected] or type(c.get("offset",0)) is not int or c.get("offset",0)<0:
            return fail("failed","invalid_input","Preservation must name an expected revision and character offset","validate")
        if c.get("review") not in ["preserved","unresolved"] or not all(isinstance(c.get(k),str) and 0<len(c[k])<=1000 for k in ["id","excerpt","evidence_ref"]) or c["id"] in ids:
            return fail("failed","invalid_input","Preservation needs unique ID, excerpt and editorial evidence reference","validate")
        ids.append(c["id"])
    baseline=inputs.get("reference_baseline")
    if baseline is not None and (not isinstance(baseline,dict) or set(baseline)!={"base_path","checks","skip_external_links","complete","reference_findings","content_findings"} or baseline.get("base_path")!=inputs["base_path"] or baseline.get("checks")!=["links","refs"] or baseline.get("skip_external_links") is not True or baseline.get("complete") is not True or any(not isinstance(baseline.get(k),list) or len(baseline[k])>100 or not all(isinstance(f,dict) for f in baseline[k]) for k in ["reference_findings","content_findings"])):
        return fail("failed","invalid_input","Baseline must be complete and use identical scope/checks","validate")
    envelope["inputs"]={k:v for k,v in inputs.items() if k not in ["reference_baseline","dispositions","preservation"]}
    return "collect"

def finding_keys(reference,content):
    # Retain full owner finding identity; moved line numbers are conservative new debt.
    return set(["reference:"+json.dumps(f,sort_keys=True) for f in reference]+["content:"+json.dumps(f,sort_keys=True) for f in content])

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
            envelope["signals"]["checks"].append({"kind":"retrieval","expected_paths":q["expected_paths"],"returned_paths":paths,"method":result.meta().get("method"),"forbidden_paths":q.get("forbidden_paths",[]),"passed":all(p in paths for p in q["expected_paths"]) and not any(p in paths for p in q.get("forbidden_paths",[]))})
        for p in inputs.get("absent_paths",[]):
            doc=document(knowledge_observatory.knowledge_base.inspect(path=p,allow_missing=True,limit=1))
            envelope["signals"]["checks"].append({"kind":"source_absent","path":p,"passed":doc.get("missing") is True})
        revisions={e["path"]:e["sha256"] for e in inputs["expectations"]}
        for c in inputs.get("preservation",[]):
            doc=document(knowledge_observatory.knowledge_base.inspect(path=c["path"],expected_sha256=revisions[c["path"]],offset=c.get("offset",0),limit=len(c["excerpt"])))
            envelope["signals"]["checks"].append({"kind":"preservation_excerpt","id":c["id"],"path":c["path"],"passed":doc.get("content")==c["excerpt"],"editorial_review":c["review"],"editorial_evidence_ref":c["evidence_ref"]})
        envelope["signals"]["editorial_review_complete"]=bool(inputs.get("preservation")) and all(c["review"]=="preserved" for c in inputs.get("preservation",[]))
        health=knowledge_observatory.knowledge_base.health(scope="path-exact",path=inputs["base_path"],skip_external_links=True,checks=["links","refs"],rows="referenceFindings")
        reference=health.head(100);content=health.meta().get("contentFindings",[])
        envelope["signals"]["reference_findings"]=reference[:12]
        envelope["signals"]["content_findings"]=content[:12]
        envelope["signals"]["reference_findings_count"]=health.count()+len(content)
        complete=health.count()<=100 and len(content)<=100
        current=finding_keys(reference,content[:100])
        baseline=inputs.get("reference_baseline")
        previous=finding_keys(baseline["reference_findings"],baseline["content_findings"]) if baseline else set()
        new=current-previous
        new_findings=[{"kind":kind,"finding":f} for kind,findings in [("reference",reference),("content",content[:100])] for f in findings if kind+":"+json.dumps(f,sort_keys=True) in new]
        envelope["signals"]["reference_delta"]={"baseline_supplied":baseline is not None,"complete":complete,"new_count":len(new),"retained_count":len(current & previous),"resolved_count":len(previous-current) if complete else None,"new_findings":new_findings[:12],"new_findings_truncated":len(new_findings)>12}
        envelope["signals"]["checks"].append({"kind":"reference_health","passed":complete and not new})
        if not all(c["passed"] for c in envelope["signals"]["checks"]):return fail("failed","verification_failed","At least one explicit post-change check failed","collect")
        if inputs.get("preservation") and not envelope["signals"]["editorial_review_complete"]:
            return fail("partial","editorial_review_pending","Mechanical checks passed; preservation judgment remains unresolved","collect")
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
