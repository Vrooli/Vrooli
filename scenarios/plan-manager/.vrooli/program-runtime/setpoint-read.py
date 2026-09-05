"""Compare bounded usage evidence and external condition without inventing baselines."""
import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program":"plan-manager.setpoint-read","version":"1","status":"ok","phase":"validate","inputs":{},"signals":{"rows":[]},"errors":[],"evidence":[]}
handles = {}
def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class":klass,"detail":str(detail)[:160],"where":where})
    return "report"
def row(name, reading, target=None, in_band=None, reason=None):
    envelope["signals"]["rows"].append({"row":name,"reading":reading,"target":target,"in_band":in_band,"unavailable":reason is not None,"reason":reason})
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
    if not isinstance(inputs, dict) or set(inputs)-{"from","to","operation","context_key"} or any(not isinstance(v,str) or len(v)>256 for v in inputs.values()):
        return fail("failed","invalid_input","Use bounded string comparison selectors","validate")
    envelope["inputs"] = dict(inputs)
    return "collect"
def guarded(name, call):
    try:
        return call()
    except Exception as exc:
        status, klass = classify_transport(exc)
        handles[name+"_reason"] = "scenario_unreachable" if klass=="scenario_unreachable" else "unreliable:"+klass
        envelope["errors"].append({"class":klass,"detail":klass,"where":name})
        return None

def step_collect():
    envelope["phase"] = "collect"
    results = gather(lambda: guarded("condition",lambda:program_runtime.bindings.condition(scenario="plan-manager",window_seconds=604800,rows="conditions")),
                     lambda: guarded("learning",lambda:vrooli_memory.learning.measure(scope="plan-manager-usage",rows="cohorts",**inputs)))
    handles["condition"],handles["learning"] = results
    return "classify"
def step_classify():
    envelope["phase"] = "classify"
    condition = handles.get("condition")
    if condition is None:
        row("binding-health",None,"all exercised bindings healthy",reason=handles.get("condition_reason"))
    else:
        total = condition.count()
        dormant = condition.filter(lambda r:r.get("status")=="CONDITION_STATUS_DORMANT").count()
        bad = condition.filter(lambda r:r.get("status") not in ["CONDITION_STATUS_HEALTHY","CONDITION_STATUS_DORMANT"]).count()
        row("binding-health",{"total":total,"dormant":dormant,"not_healthy":bad},"all exercised bindings healthy",bad==0 if total>dormant else None,None if total>dormant else "unreliable:no_exercised_bindings")
        envelope["evidence"].append("program-runtime/bindings/condition")
    learning = handles.get("learning")
    reason = handles.get("learning_reason")
    reading = None
    if learning is not None:
        meta=learning.meta()
        reason = meta.get("reason") if not meta.get("reliable") else None
        if not meta.get("reliable") and not reason: reason="unreliable:missing_validity"
        if meta.get("truncated"): reason="unreliable:capped_scan"
        if learning.count()!=1: reason="unreliable:select_one_comparable_cohort"
        cohorts=learning.head(1)
        fields=["operation","contextKey","attempts","failed","unavailable","unknown","tasks","completedTasks","unresolvedTasks","leftCensoredTasks","medianAttemptsToSuccess","medianSecondsToSuccess","recurringFailureFingerprints","toolRoundTripSamples","medianToolRoundTrips","firstActionSamples","medianSecondsToFirstAction","appliedAdvice","supportedAdvice","contradictedAdvice","unassessedAdvice"]
        reading={"from":meta.get("from"),"to":meta.get("to"),"eligible_attempts":meta.get("eligibleAttempts"),"cohort":{k:cohorts[0].get(k) for k in fields} if cohorts else None}
        envelope["evidence"].append("vrooli-memory/learning/measure")
    row("usage-learning",reading,reason=reason)
    row("recurring-friction",None,reason="read_elsewhere:agent-manager.friction-digest")
    for name in ["validation-reuse","coordination-time","manual-recovery","family-critical-path"]:
        row(name,None,reason="pending_telemetry")
    envelope["status"]="partial" if envelope["errors"] else "ok"
    return "report"
def step_report():
    envelope["phase"]="report"
    print(json.dumps(envelope,separators=(",",":")))
    return None
STATES={"validate":step_validate,"collect":step_collect,"classify":step_classify,"report":step_report}
state="validate"
while state:
    try:
        state=STATES[state]()
    except Exception as exc:
        if state=="report": raise
        state=fail("failed","kernel_runtime",str(exc),state)
