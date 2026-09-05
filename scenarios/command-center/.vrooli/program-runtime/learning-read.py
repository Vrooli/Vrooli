import json
try:
    inputs
except NameError:
    inputs = {}
envelope={"program":"command-center.learning-read","version":"1","status":"failed","phase":"validate","inputs":{},
 "signals":{},"errors":[],"evidence":[]}
handles={}
def fail(status,klass,detail,where):
    envelope["status"]=status
    envelope["errors"].append({"class":klass,"detail":str(detail)[:160],"where":where})
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




def step_validate():
    if not isinstance(inputs,dict) or set(inputs)-{"from","to","operation","context_key"}:
        return fail("failed","invalid_input","Only explicit comparison-window selectors are accepted","validate")
    envelope["inputs"] = dict(inputs)
    return "collect"
def step_collect():
    envelope["phase"]="collect"
    try:
        handles["learning"]=vrooli_memory.learning.measure(scope="command-center-usage",rows="cohorts",**inputs)
        envelope["evidence"].append("vrooli-memory/learning/measure")
    except Exception as exc:
        status,klass=classify_transport(exc)
        handles["reason"]="scenario_unreachable" if klass=="scenario_unreachable" else "unreliable:"+klass
        envelope["errors"].append({"class":klass,"detail":klass,"where":"collect"})
    return "act"
def step_act():
    envelope["phase"]="act"
    source=handles.get("learning")
    meta=source.meta() if source is not None else {}
    cohorts=source.head(1) if source is not None else []
    reason=handles.get("reason") or meta.get("reason") or (None if meta.get("reliable") else "unreliable:missing_validity")
    if source is not None and source.count()>1:
        reason="unreliable:cohort_sample"
    groups={
     "failure-recurrence":["attempts","failed","unavailable","unknown","recurringFailureFingerprints"],
     "completion-effort":["tasks","completedTasks","unresolvedTasks","leftCensoredTasks","medianAttemptsToSuccess","medianSecondsToSuccess"],
     "advice-outcomes":["appliedAdvice","rejectedAdvice","supportedAdvice","contradictedAdvice","unassessedAdvice","recallUnavailable"],
     "first-action-latency":["firstActionSamples","medianSecondsToFirstAction"],
     "agent-round-trips":["toolRoundTripSamples","medianToolRoundTrips"],
     "visual-reasoning":["visualReasoningSamples","medianVisualReasoningCalls"],
     "workflow-reuse":["reuseSamples","workflowReuseRate"]}
    envelope["signals"]["rows"]=[]
    for row,fields in groups.items():
        reading={"from":meta.get("from"),"to":meta.get("to"),"eligible_attempts":meta.get("eligibleAttempts",0),
          "cohorts":[dict({"operation":str(c.get("operation") or "")[:48],"context":str(c.get("contextKey") or "")[:64]},**{k:c.get(k) for k in fields}) for c in cohorts]}
        envelope["signals"]["rows"].append({"row":row,"reading":reading if source is not None else None,
         "target":None,"in_band":None,"unavailable":bool(reason),"reason":reason})
    envelope["status"]="partial" if envelope["errors"] else "ok"
    return "report"
def step_report():
    envelope["phase"]="report"
    print(json.dumps(envelope,separators=(",",":")))
    return None
STATES={"validate":step_validate,"collect":step_collect,"act":step_act,"report":step_report}
state="validate"
while state:
    try:
        state=STATES[state]()
    except Exception as exc:
        if state=="report":
            raise
        state=fail("failed","kernel_runtime",str(exc)[:160],state)
