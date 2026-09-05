try:
    inputs
except NameError:
    inputs = {}
envelope={"program":"device-control.author-flow","version":"1","status":"failed","phase":"validate","inputs":{},
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
    for field in ("device_id","context_key","actor"):
        if not isinstance(inputs.get(field),str) or not inputs[field].strip() or len(inputs[field])>512:
            return fail("failed","invalid_input","Required string: "+field,"validate")
    if not isinstance(inputs.get("flow"),dict) or not inputs["flow"].get("steps"):
        return fail("failed","invalid_input","Nonempty typed candidate flow required","validate")
    if inputs.get("flow_id") and (type(inputs.get("expected_version"))!=int or inputs["expected_version"]<1):
        return fail("failed","invalid_input","Repair requires expected_version","validate")
    return "collect"

def step_collect():
    envelope["phase"]="collect"
    try:
        rows=device_control.flow.validate(flow=inputs["flow"],strategy_id=inputs["device_id"], require_assertion=True, baseline_id=inputs.get("flow_id",""), expected_version=inputs.get("expected_version",0)).head(1)
        if not rows or not rows[0].get("runnable"):
            return fail("failed","capability_gap","Candidate validation refused; no device action taken","collect")
    except Exception as exc:
        status,klass=classify_transport(exc)
        return fail(status,klass,klass,"collect")
    return "act"
def step_act():
    envelope["phase"]="act"
    try:
        runs=device_control.flow.run(flow=inputs["flow"],device_id=inputs["device_id"],actor=inputs["actor"]).head(1)
        run=runs[0] if runs else {}
        envelope["signals"]={"run_id":run.get("runId"),"saved":False}
        if not run.get("runId") or run.get("disposition")!="passed" or run.get("incomplete"):
            return fail("failed","flow_failed","Candidate did not pass; saved revisions unchanged","act")
        envelope["evidence"]=["run:"+run["runId"]]
        saved=device_control.flow.save(run_id=run["runId"],device_id=inputs["device_id"],
          context_key=inputs["context_key"],id=inputs.get("flow_id",""),
          expected_version=inputs.get("expected_version",0)).head(1)
        if not saved or not saved[0].get("id"):
            return fail("failed","invalid_response","Persistence reference missing","act")
        envelope["signals"].update(saved=True,flow_id=saved[0]["id"],version=saved[0].get("version"))
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
