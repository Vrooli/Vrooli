NAME='agent-manager.supervision-case-read'
KEYS=['watch_id']
import json
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program": NAME, "version":"1", "status":"ok", "phase":"validate", "inputs":{}, "signals":{}, "errors":[], "evidence":[]}
work = {}
def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class":klass,"detail":str(detail)[:180],"where":where})
    return "report"
def short(value):
    return str(value or "")[:160]
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
    if not isinstance(inputs,dict) or set(inputs)-set(KEYS) or any(not isinstance(inputs.get(k),str) or not inputs[k].strip() or len(inputs[k])>128 for k in KEYS):
        return fail("failed","invalid_input","Supply the declared bounded nonempty owner IDs","validate")
    envelope["inputs"] = dict(inputs)
    return "collect"
def step_collect():
    work["inspection"],work["actions"],work["outcomes"]=gather(
        lambda:agent_manager.watch.inspect(watch_id=inputs["watch_id"],event_limit=16),
        lambda:agent_manager.watch.actions(watch_id=inputs["watch_id"],limit=8),
        lambda:agent_manager.watch.policy_outcomes(watch_id=inputs["watch_id"],limit=16))
    return "classify"
def step_classify():
    meta=work["inspection"].meta()
    watch=meta.get("watch") or {}
    if watch.get("watchId")!=inputs["watch_id"]: return fail("failed","identity_mismatch","watch response mismatch","classify")
    decision=watch.get("lastDecision") or {}
    actions=work["actions"].head(8)
    outcomes=work["outcomes"].head(16)
    signals={"watch_status":watch.get("status"),"revision":watch.get("revision"),"policy_version":(watch.get("spec") or {}).get("policyVersion"),"decision_id":decision.get("decisionId"),"disposition":decision.get("disposition"),"classification":decision.get("classification"),"cursor_reset":meta.get("cursorResetRequired",False),"pending_event_count":work["inspection"].count(),"actions":[{"id":r.get("actionId"),"state":r.get("state"),"decision_id":r.get("decisionId"),"target":r.get("targetRunId")} for r in actions],"assessed_sample":sum(bool(r.get("observedClass")) for r in outcomes),"unassessed_sample":sum(not r.get("observedClass") for r in outcomes),"sample_may_be_truncated":len(outcomes)==16 or len(actions)==8,"history_scope":"latest decision and bounded owner records"}
    signals["next_action"]="reconcile_cursor" if signals["cursor_reset"] else "inspect_outcome_evidence"
    envelope["signals"]=signals
    envelope["evidence"]=[inputs["watch_id"]]+([decision["decisionId"]] if decision.get("decisionId") else [])
    return "report"

def step_report():
    envelope["phase"]="report"
    encoded=json.dumps(envelope,sort_keys=True,separators=(",",":"))
    if len(encoded.encode())>4096:
        envelope["status"]="failed"
        envelope["signals"]={}
        envelope["errors"]=[{"class":"output_budget_exhausted","detail":"owner summary exceeds bound","where":"report"}]
        encoded=json.dumps(envelope,separators=(",",":"))
    print(encoded)
    return None
states={"validate":step_validate,"collect":step_collect,"classify":step_classify,"report":step_report}
state="validate"
while state:
    try:
        envelope["phase"]=state
        state=states[state]()
    except Exception as exc:
        if state=="report": raise
        try:
            status,klass=classify_transport(exc)
        except (NameError,AttributeError):
            status,klass="failed","kernel_runtime"
        state=fail(status,klass,exc,state)
