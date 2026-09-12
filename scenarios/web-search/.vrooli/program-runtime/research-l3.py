import json

"""Start idempotently or reattach to one declared research workflow; never poll."""
inputs = program.inputs()
envelope={"program":"web-search.research-l3","version":"1","status":"failed","phase":"validate","signals":{},"errors":[],"evidence":[]}
def fields(h):
    m=h.meta() or {}
    rows=h.head(1)
    if rows:m.update(rows[0])
    return m
try:
    if not isinstance(inputs,dict) or set(inputs)-{"query","idempotency_key","run_id","timeout_seconds"}:
        raise ValueError("Supply a query and stable idempotency_key, or an existing run_id")
    run_id=inputs.get("run_id")
    if not run_id and (not inputs.get("query") or not inputs.get("idempotency_key")):
        raise ValueError("Starting research requires query and idempotency_key")
    seconds=inputs.get("timeout_seconds",90)
    if not isinstance(seconds,int) or not 1<=seconds<=90:raise ValueError("Wait budget must be 1..90 seconds")
    if not run_id:
        envelope["phase"]="act"
        started=fields(web_search.research.l3(query=inputs["query"],idempotency_key=inputs["idempotency_key"]))
        run_id=started.get("runId")
        if not run_id:raise RuntimeError("owner returned no execution identity")
    envelope["signals"]["run_id"]=run_id
    envelope["evidence"]=[run_id]
    envelope["phase"]="collect"
    result=fields(web_search.research.wait(id=run_id,timeout_seconds=seconds))
    envelope["signals"]["execution"]=result
    if result.get("timedOut") or result.get("status") in ("running","waiting","pending"):
        envelope["status"]="partial"
        envelope["signals"]["next_action"]={"kind":"wait","run_id":run_id,"do_not_poll":True}
    elif result.get("status")=="complete":
        envelope["status"]="ok" if (result.get("result") or {}).get("status")=="answered" else "partial"
    else:
        envelope["status"]="failed"
        envelope["errors"].append({"class":"research_failed","detail":result.get("errorMsg",result.get("status","unknown")),"where":"collect"})
except Exception as exc:
    klass="invalid_input" if isinstance(exc,ValueError) else "binding_error"
    if "grant" in str(exc):envelope["status"]="refused"
    envelope["errors"].append({"class":klass,"detail":str(exc)[:160],"where":envelope["phase"]})
envelope["phase"]="report"
print(json.dumps(envelope, sort_keys=True))
