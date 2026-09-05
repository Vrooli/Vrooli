# One bounded task route. Caller captures the final outcome through learning.record.
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program":"browser-automation-studio.do-task","version":"2","status":"failed",
 "phase":"validate","inputs":{},"signals":{"result":None,"candidates":[],
 "capture_required":True,"reused_workflow":False},"errors":[],"evidence":[]}
def fail(status, klass, detail, where):
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
    if not isinstance(inputs,dict) or not isinstance(inputs.get("task"),str) or not inputs["task"].strip():
        return fail("failed","invalid_input","task must be a nonempty string","validate")
    for field in ("workflow_id","project_id","session","model","scenario"):
        if field in inputs and (not isinstance(inputs[field],str) or len(inputs[field])>512):
            return fail("failed","invalid_input","Invalid "+field,"validate")
    selected=sum(bool(inputs.get(k)) for k in ("workflow_id","flow","session"))
    if selected>1:
        return fail("failed","invalid_input","Select one workflow, candidate flow, or navigation session","validate")
    if inputs.get("workflow_id") and (type(inputs.get("version"))!=int or inputs["version"]<1):
        return fail("failed","invalid_input","Execution requires the exact selected version","validate")
    envelope["inputs"]={"task":inputs["task"][:160],"workflow_id":inputs.get("workflow_id")}
    return "collect"

def step_collect():
    # The usage skill owns targeted recall and advice assessment once.
    envelope["phase"]="collect"
    return "act"

def step_act():
    envelope["phase"]="act"
    try:
        if inputs.get("workflow_id"):
            result=lib.browser_automation_studio.smoke_flow(
                workflow_id=inputs["workflow_id"],version=inputs["version"],
                parameters=inputs.get("parameters",{})).head(1)
            envelope["signals"]["reused_workflow"]=True
        elif inputs.get("flow"):
            result=lib.browser_automation_studio.author_flow(
                flow=inputs["flow"],project_id=inputs.get("project_id",""),
                name=inputs.get("name","candidate"),
                workflow_id=inputs.get("repair_workflow_id",""),
                expected_version=inputs.get("expected_version",0)).head(1)
        elif inputs.get("session"):
            if not inputs.get("model"):
                return fail("failed","model_required","Select a navigator model before starting navigation","act")
            result=lib.browser_automation_studio.navigate_intent(
                session=inputs["session"],prompt=inputs["task"],model=inputs["model"],
                max_steps=inputs.get("max_steps",10)).head(1)
        else:
            result=lib.browser_automation_studio.find_flows(
                task=inputs["task"],scenario=inputs.get("scenario",""),k=5).head(1)
            child=result[0] if result else {}
            envelope["signals"]["candidates"]=(child.get("signals") or {}).get("candidates",[])[:5]
            # Search relevance is never permission to execute a guessed workflow.
            return fail("partial","selection_required","Select a matching workflow UUID and version, or supply a candidate/session","act")
        if not result:
            return fail("failed","invalid_response","Child program returned no envelope","act")
        child=result[0]
        envelope["signals"]["result"]=child.get("signals")
        envelope["evidence"]=(child.get("evidence") or [])[:10]
        envelope["errors"]+=(child.get("errors") or [])[:5]
        envelope["status"]=child.get("status","failed")
        if envelope["status"]=="ok" and envelope["errors"]:
            envelope["status"]="partial"
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
