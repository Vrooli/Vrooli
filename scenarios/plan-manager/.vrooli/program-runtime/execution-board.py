NAME='plan-manager.execution-board'
KEYS=['execution_id', 'family_id']
import json
inputs = program.inputs()
envelope = {"program": NAME, "version":"1", "status":"ok", "phase":"validate", "inputs":{}, "signals":{}, "errors":[], "evidence":[]}
work = {}
def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class":klass,"detail":str(detail)[:180],"where":where})
    return "report"
def short(value):
    return str(value or "")[:160]


def step_validate():
    if not isinstance(inputs,dict) or set(inputs)-{"execution_id","family_id","supervisor_execution_id"}:
        return fail("failed","invalid_input","Supply bounded owner IDs","validate")
    for key in ["execution_id","family_id","supervisor_execution_id"]:
        value=inputs.get(key,"")
        if not isinstance(value,str) or len(value)>128 or (key=="execution_id" and not value.strip()):
            return fail("failed","invalid_input","Supply bounded owner IDs","validate")
    envelope["inputs"] = dict(inputs)
    return "collect"
def step_collect():
    work["execution"]=plan_manager.exec.status(execution_id=inputs["execution_id"])
    execution=work["execution"].meta().get("execution") or {}
    if execution.get("id")!=inputs["execution_id"]: return fail("failed","identity_mismatch","execution response mismatch","collect")
    if inputs.get("family_id"):
        work["family"],work["frontier"]=gather(lambda:plan_manager.families.get(family_id=inputs["family_id"]),lambda:plan_manager.families.frontier(family_id=inputs["family_id"]))
    receipt_id=(execution.get("baselineSet") or {}).get("receiptId")
    if receipt_id:
        work["receipt_id"]=receipt_id
        work["receipt"]=test_genie.validation.get(receipt_id=receipt_id)
    if inputs.get("supervisor_execution_id"):
        work["watches"]=agent_manager.watch.list(family_execution_id=inputs["supervisor_execution_id"],page_size=8)
    return "classify"
def step_classify():
    state=work["execution"].meta()
    execution=state.get("execution") or {}
    step=state.get("step") or {}
    actions=step.get("nextActions") or []
    signals={"execution_complete":execution.get("complete",False),"lifecycle_state":execution.get("lifecycleState"),"phase_id":execution.get("currentPhaseId"),"owner_step":{k:step.get(k) for k in ["stepKind","title","summary"]},"next_action":actions[0] if actions else None}
    if work.get("family") is not None:
        family=work["family"].meta().get("family") or {}
        frontier=work["frontier"].meta()
        if family.get("familyId")!=inputs["family_id"]: return fail("failed","identity_mismatch","family response mismatch","classify")
        members=family.get("members") or []
        if not any(m.get("executionId")==inputs["execution_id"] and m.get("planId")==execution.get("planId") for m in members): return fail("failed","identity_mismatch","execution is not a declared family member","classify")
        if str((family.get("graph") or {}).get("revision"))!=str(frontier.get("graphRevision")): return fail("unavailable","revision_changed","family graph changed between reads","classify")
        batches=frontier.get("batches") or []
        signals.update({"family_state":family.get("executionState"),"family_revision":family.get("revision"),"graph_revision":frontier.get("graphRevision"),"launchable":frontier.get("launchable",False),"runnable_now":(batches[0].get("planIds") or [])[:8] if batches else [],"running":[m.get("planId") for m in members if m.get("state")=="MEMBER_STATE_RUNNING"][:8],"diagnostics":[short(d) for d in (frontier.get("diagnostics") or [])[:3]],"launch_authority":"revision-checked Plan Manager member admission"})
    if work.get("receipt") is not None:
        receipt=work["receipt"].meta().get("receipt") or {}
        if receipt.get("receiptId")!=work["receipt_id"]:return fail("failed","identity_mismatch","receipt response mismatch","classify")
        signals["validation"]={k:receipt.get(k) for k in ["receiptId","state","reason"]}
        terminal=receipt.get("state") in ["RECEIPT_STATE_SUCCEEDED","RECEIPT_STATE_FAILED","RECEIPT_STATE_DEGRADED","RECEIPT_STATE_CANCELLED","RECEIPT_STATE_SUPERSEDED"]
        sync=(execution.get("baselineSet") or {}).get("syncArgv")
        if terminal and step.get("stepKind")=="baseline_receipt_pending" and sync:
            signals["coordination_state"]="receipt_terminal_plan_sync_pending"
            signals["next_action"]={"id":"baseline-sync","argv":sync,"label":"Consume the terminal owner receipt"}
    if work.get("watches") is not None:
        watches=work["watches"].head(8)
        if any((w.get("spec") or {}).get("familyExecutionId")!=inputs["supervisor_execution_id"] for w in watches):return fail("failed","identity_mismatch","watch execution mismatch","classify")
        signals["supervision"]=[{"watch_id":w.get("watchId"),"status":w.get("status"),"classification":(w.get("lastDecision") or {}).get("classification")} for w in watches]
        signals["watches_truncated"]=bool(work["watches"].meta().get("nextPageToken"))
    signals["read_consistency"]="owner reads at distinct instants; admission must recheck current revision"
    envelope["signals"]=signals
    envelope["evidence"]=[inputs["execution_id"]]+([inputs["family_id"]] if inputs.get("family_id") else [])
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
            status,klass=program.classify(exc)
        except (NameError,AttributeError):
            status,klass="failed","kernel_runtime"
        state=fail(status,klass,exc,state)
