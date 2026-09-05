try:
    inputs
except NameError:
    inputs = {}
envelope={"program":"device-control.replay-flow","version":"1","status":"failed","phase":"validate","inputs":{},
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
    if not isinstance(inputs,dict):
        return fail("failed","invalid_input","Object inputs required","validate")
    for field in ("device_id","context_key","flow_id","actor"):
        if not isinstance(inputs.get(field),str) or not inputs[field].strip() or len(inputs[field])>512:
            return fail("failed","invalid_input","Required string: "+field,"validate")
    if type(inputs.get("version"))!=int or inputs["version"]<1:
        return fail("failed","invalid_input","Exact positive revision required","validate")
    return "collect"

def step_collect():
    envelope["phase"]="collect"
    try:
        rows=device_control.flow.get(id=inputs["flow_id"],version=inputs["version"]).head(1)
        saved=rows[0] if rows else {}
        if saved.get("id")!=inputs["flow_id"] or saved.get("version")!=inputs["version"] or saved.get("deviceId")!=inputs["device_id"] or saved.get("contextKey")!=inputs["context_key"]:
            return fail("failed","identity_mismatch","Saved flow differs from requested identity","collect")
    except Exception as exc:
        status,klass=classify_transport(exc)
        return fail(status,klass,klass,"collect")
    return "act"
def step_act():
    envelope["phase"]="act"
    try:
        rows=device_control.flow.replay(id=inputs["flow_id"],version=inputs["version"],
           device_id=inputs["device_id"],context_key=inputs["context_key"],actor=inputs["actor"]).head(1)
        result=rows[0] if rows else {}
        envelope["signals"]={"run_id":result.get("runId"),"disposition":result.get("disposition"),"reused_workflow":True}
        if not result.get("runId"):
            return fail("failed","invalid_response","Run reference missing","act")
        envelope["evidence"]=["run:"+result["runId"]]
        if result.get("disposition")!="passed" or result.get("incomplete"):
            return fail("failed","flow_failed","Saved flow did not pass; no alternate action started","act")
        envelope["status"]="ok"
    except Exception as exc:
        status,klass=classify_transport(exc)
        return fail(status,klass,klass,"act")
    return "report"

def step_report():
    envelope["phase"]="report"
    print(envelope)
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
