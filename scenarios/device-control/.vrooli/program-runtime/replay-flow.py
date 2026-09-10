inputs = program.inputs()
learn.task(operation="device-control.replay-flow", key={"device": inputs.get("device_id", ""), "context_key": inputs.get("context_key", "")})
envelope={"program":"device-control.replay-flow","version":"1","status":"failed","phase":"validate","inputs":{},
 "signals":{},"errors":[],"evidence":[]}
handles={}
def fail(status,klass,detail,where):
    envelope["status"]=status
    if klass in ("invalid_input", "identity_mismatch"):
        envelope["signals"]["outcome"]="failed"
    elif klass in ("capability_gap", "scenario_unreachable", "no_grant", "not_run_eligible"):
        envelope["signals"]["outcome"]="unavailable"
    else:
        envelope["signals"].setdefault("outcome", "unknown")
    envelope["errors"].append({"class":klass,"detail":str(detail)[:160],"where":where})
    return "report"


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
        status,klass=program.classify(exc)
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
        if result.get("disposition")=="failed" and not result.get("incomplete"):
            envelope["signals"]["outcome"]="failed"
        if result.get("disposition")!="passed" or result.get("incomplete"):
            return fail("failed","flow_failed","Saved flow did not pass; no alternate action started","act")
        envelope["status"]="ok"
        envelope["signals"]["outcome"]="verified_success"
        learn.note("preference", {"option_id": inputs["flow_id"] + "@" + str(inputs["version"])})
    except Exception as exc:
        status,klass=program.classify(exc)
        return fail(status,klass,klass,"act")
    return "report"

def step_report():
    envelope["phase"]="report"
    status = envelope["signals"].get("outcome", "unknown")
    # The replayed revision is the exact artifact a later correction should name.
    artifact = {"kind": "flow", "owner": "device-control", "id": str(inputs["flow_id"]), "revision": str(inputs["version"])} if inputs.get("flow_id") and inputs.get("version") else None
    envelope["signals"]["learning"] = {"feedback_ref": learn.result("replay", artifact=artifact)}
    learn.outcome(status if status in ("verified_success", "failed", "unavailable", "unknown") else "unknown",
                  envelope["evidence"] if status == "verified_success" else [])
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
