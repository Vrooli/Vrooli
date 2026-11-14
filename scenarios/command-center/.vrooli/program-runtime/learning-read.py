import json
inputs = program.inputs()
envelope={"program":"command-center.learning-read","version":"1","status":"failed","phase":"validate","inputs":{},
 "signals":{},"errors":[],"evidence":[]}
handles={}
def fail(status,klass,detail,where):
    envelope["status"]=status
    envelope["errors"].append({"class":klass,"detail":str(detail)[:160],"where":where})
    return "report"


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
        status,klass=program.classify(exc)
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
