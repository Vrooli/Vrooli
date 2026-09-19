inputs = program.inputs()
learn.task(operation="device-control.author-flow", key={"device": inputs.get("device_id", ""), "context_key": inputs.get("context_key", "")})
envelope={"program":"device-control.author-flow","version":"1","status":"failed","phase":"validate","inputs":{},
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
        status,klass=program.classify(exc)
        return fail(status,klass,klass,"collect")
    return "act"
def step_act():
    envelope["phase"]="act"
    try:
        runs=device_control.flow.run(flow=inputs["flow"],device_id=inputs["device_id"],actor=inputs["actor"]).head(1)
        run=runs[0] if runs else {}
        envelope["signals"]={"run_id":run.get("runId"),"saved":False}
        if run.get("runId"):
            envelope["evidence"]=["run:"+run["runId"]]
        if run.get("disposition")=="failed" and not run.get("incomplete"):
            envelope["signals"]["outcome"]="failed"
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
        envelope["signals"]["outcome"]="verified_success"
        learn.note("preference", {"option_id": saved[0]["id"] + "@" + str(saved[0].get("version"))})
    except Exception as exc:
        status,klass=program.classify(exc)
        return fail(status,klass,klass,"act")
    return "report"

def step_report():
    envelope["phase"]="report"
    status = envelope["signals"].get("outcome", "unknown")
    # Hand the caller a durable reference: a later learn.feedback can correct this revision
    # without repeating the authoring run.
    saved_id, saved_version = envelope["signals"].get("flow_id"), envelope["signals"].get("version")
    artifact = {"kind": "flow", "owner": "device-control", "id": str(saved_id), "revision": str(saved_version)} if saved_id and saved_version else None
    envelope["signals"]["learning"] = {"feedback_ref": learn.result("flow", artifact=artifact)}
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
