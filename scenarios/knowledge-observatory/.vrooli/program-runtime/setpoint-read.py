import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": "knowledge-observatory.setpoint-read", "version": "1", "status": "failed", "phase": "validate", "inputs": {}, "signals": {"capture_required": True}, "errors": [], "evidence": []}
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

def row(name,reading,target=None,in_band=None,reason=None):
    envelope["signals"]["rows"].append({"row":name,"reading":reading,"target":target,"in_band":in_band,"unavailable":reading is None and not (reason or "").startswith("read_elsewhere:"),"reason":reason})
def step_validate():
    if not isinstance(inputs,dict) or set(inputs)-{"base_path","eval_run_id"} or not checked_path(inputs.get("base_path","scenarios/knowledge-observatory/docs")):
        return fail("failed","invalid_input","Use a repository-relative base_path and optional immutable eval_run_id","validate")
    if not isinstance(inputs.get("eval_run_id",""),str) or len(inputs.get("eval_run_id",""))>100:return fail("failed","invalid_input","Invalid run ID","validate")
    return "collect"
def step_collect():
    envelope["phase"]="collect";envelope["signals"]["rows"]=[]
    for name in ["index-availability","reference-health","binding-condition","retrieval-evaluation"]:
        try:
            if name=="index-availability":
                data=scalar(knowledge_observatory.knowledge_base.status())
                row(name,{"available":data.get("available",False),"indexed_count":data.get("indexedCount",0),"last_reconcile_at":data.get("lastReconcileAt"),"last_reconcile_outcome":data.get("lastReconcileOutcome")},True,bool(data.get("available")))
            elif name=="reference-health":
                data=knowledge_observatory.knowledge_base.health(scope="path-exact",path=inputs.get("base_path","scenarios/knowledge-observatory/docs"),skip_external_links=True,checks=["links","refs"],rows="referenceFindings")
                count=data.count()+len(data.meta().get("contentFindings",[]))
                row(name,{"findings":count,"base_path":inputs.get("base_path","scenarios/knowledge-observatory/docs")},0,count==0)
            elif name=="binding-condition":
                data=program_runtime.bindings.condition(scenario="knowledge-observatory",rows="conditions")
                row(name,data.head(5),None,None,"unreliable:pending_baseline")
            elif not inputs.get("eval_run_id"):
                row(name,None,None,None,"unreliable:no_selected_eval")
            else:
                data=scalar(search_hub.evals.show_run(run_id=inputs["eval_run_id"])).get("run",{})
                if data.get("suiteId")!="knowledge-observatory.docs.starter":
                    row(name,None,None,None,"unreliable:wrong_suite")
                else:
                    stats=data.get("aggregate",{});graded=stats.get("gradedCases",0)
                    reason="unreliable:degraded_eval" if data.get("degraded") or not graded or stats.get("unavailableCases",0)>0 else None
                    row(name,{"run_id":data.get("runId"),"created_at":data.get("createdAt"),"tier":data.get("tier"),"met":stats.get("met",0),"graded":graded,"pass_rate":stats.get("passRate",0),"config":data.get("config",{})},None,None,reason or "unreliable:pending_comparable_baseline")
                    envelope["evidence"].append("search-hub:eval-run/"+inputs["eval_run_id"])
        except Exception as exc:
            status,klass=classify_transport(exc)
            row(name,None,None,None,"scenario_unreachable" if klass=="scenario_unreachable" else "unreliable:"+klass)
            envelope["errors"].append({"class":klass,"detail":str(exc)[:160],"where":name})
    row("usage-learning",None,None,None,"read_elsewhere:knowledge-observatory.learning-read")
    row("external-friction",None,None,None,"read_elsewhere:agent-manager.friction-digest")
    envelope["status"]="partial" if envelope["errors"] else "ok"
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
