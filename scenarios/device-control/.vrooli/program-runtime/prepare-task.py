try:
    inputs
except NameError:
    inputs = {}
envelope={"program":"device-control.prepare-task","version":"1","status":"failed","phase":"validate","inputs":{},
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
    for field in ("device","context_key"):
        if not isinstance(inputs.get(field),str) or not inputs[field].strip() or len(inputs[field])>512:
            return fail("failed","invalid_input","Required string: "+field,"validate")
    return "collect"

def step_collect():
    envelope["phase"]="collect"
    try:
        inventory=device_control.device.list()
        selected=inventory.filter(lambda d:d.get("id")==inputs["device"])
        if not selected.count():
            selected=inventory.filter(lambda d:d.get("name","").lower()==inputs["device"].lower())
        if selected.count()!=1:
            return fail("failed","device_selection_required","Device must match one exact ID or unique name","collect")
        device=selected.head(1)[0]
        handles["device"]=device
        flows=device_control.flow.list(device_id=device["id"],context_key=inputs["context_key"])
        envelope["signals"]={"device_id":device["id"],"strategy_id":device.get("strategyId"),
          "health":device.get("health"),"transport":device.get("transport"),
          "capabilities":(device.get("capabilities") or [])[:20],
          "flows":flows.map(lambda f:{"id":f.get("id"),"version":f.get("version"),
             "name":(f.get("flow") or {}).get("name"),"source_run_id":f.get("sourceRunId")}).head(5),
          "flow_count":flows.count(),"truncated":flows.count()>5}
        envelope["status"]="ok"
    except Exception as exc:
        status,klass=classify_transport(exc)
        return fail(status,klass,klass,"collect")
    return "report"
def step_act():
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
