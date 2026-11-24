inputs = program.inputs()
envelope = program.envelope('device-control.prepare-task', '1', phase="validate", inputs={})
handles={}


def step_validate():
    if not isinstance(inputs,dict):
        return program.fail("failed","invalid_input","Object inputs required","validate", detail_limit=160)
    for field in ("device","context_key"):
        if not isinstance(inputs.get(field),str) or not inputs[field].strip() or len(inputs[field])>512:
            return program.fail("failed","invalid_input","Required string: "+field,"validate", detail_limit=160)
    return "collect"

def step_collect():
    envelope["phase"]="collect"
    try:
        inventory=device_control.device.list()
        selected=inventory.filter(lambda d:d.get("id")==inputs["device"])
        if not selected.count():
            selected=inventory.filter(lambda d:d.get("name","").lower()==inputs["device"].lower())
        if selected.count()!=1:
            return program.fail("failed","device_selection_required","Device must match one exact ID or unique name","collect", detail_limit=160)
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
        return program.fail_exception(exc, "collect", detail_limit=160)
    return "report"
def step_act():
    return "report"

def step_report():
    envelope["phase"]="report"
    print(envelope)
    return None
STATES={"validate":step_validate,"collect":step_collect,"act":step_act,"report":step_report}
program.run(STATES, "validate", detail_limit=160)
