NAME='agent-manager.supervision-experiment-read'
KEYS=['policy_version']
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
    work["policy"],work["outcomes"]=gather(lambda:agent_manager.watch.policy_get(version=inputs["policy_version"]),lambda:agent_manager.watch.policy_outcomes(policy_version=inputs["policy_version"],limit=32))
    return "classify"
def step_classify():
    record=work["policy"].meta()
    policy=record.get("policy") or {}
    if policy.get("version")!=inputs["policy_version"]: return fail("failed","identity_mismatch","policy response mismatch","classify")
    rows=work["outcomes"].head(32)
    evaluation=record.get("evaluation")
    envelope["signals"]={"state":record.get("state"),"policy_digest":record.get("digest"),"evaluator_digest":policy.get("evaluatorDigest"),"inference_identity_digest":record.get("inferenceIdentityDigest"),"evaluation":evaluation,"assessed_sample":sum(bool(r.get("observedClass")) for r in rows),"unassessed_sample":sum(not r.get("observedClass") for r in rows),"sample_family_count":len(set(r.get("familyExecutionId") for r in rows if r.get("familyExecutionId"))),"sample_may_be_truncated":len(rows)==32,"next_action":"collect_assessed_evidence" if not evaluation else "inspect_owner_gates","promotion_authority":"agent-manager; this read does not promote or establish causal benefit"}
    envelope["evidence"]=[inputs["policy_version"]]
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
