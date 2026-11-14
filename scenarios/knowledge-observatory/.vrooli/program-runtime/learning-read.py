"""Project shared Memory comparison rows into the established consumer envelope."""
import json
inputs = program.inputs()
envelope={"program":"knowledge-observatory.learning-read","version":"1","status":"failed","phase":"validate",
          "inputs":{},"signals":{},"errors":[],"evidence":[]}
try:
    if not isinstance(inputs,dict) or set(inputs)-{"from","to","operation","context_key"}:
        raise ValueError("Only explicit comparison-window selectors are accepted")
    envelope["phase"]="collect"
    result=lib.vrooli_memory.compare_outcomes(scope="knowledge-observatory-usage",cohort_limit=1,**inputs).head(1)
    if not result:
        raise RuntimeError("Shared comparison returned no envelope")
    child=result[0]
    envelope["inputs"]=child.get("inputs",{})
    envelope["errors"]=child.get("errors",[])
    envelope["evidence"]=child.get("evidence",[])
    groups={
     "failure-recurrence":["attempts","failed","unavailable","unknown","recurringFailureFingerprints"],
     "completion-effort":["tasks","completedTasks","unresolvedTasks","leftCensoredTasks","medianAttemptsToSuccess","medianSecondsToSuccess"],
     "advice-outcomes":["appliedAdvice","rejectedAdvice","supportedAdvice","contradictedAdvice","unassessedAdvice","recallUnavailable"],
     "first-action-latency":["firstActionSamples","medianSecondsToFirstAction"],
     "agent-round-trips":["toolRoundTripSamples","medianToolRoundTrips"],
     "workflow-reuse":["reuseSamples","workflowReuseRate"]}
    rows=[]
    for source in child.get("signals",{}).get("rows",[]):
        if source["row"] not in groups:
            continue
        row=dict(source)
        if row.get("reading") is not None:
            reading=dict(row["reading"])
            reading["cohorts"]=[dict(
                {"operation":c.get("operation"),"context":c.get("context")},
                **{k:c.get(k) for k in groups[row["row"]]})
                for c in reading.get("cohorts",[])[:1]]
            row["reading"]=reading
        rows.append(row)
    envelope["signals"]["rows"]=rows
    envelope["signals"]["reliability"]=child.get("signals",{}).get("reliability")
    envelope["status"]=child.get("status","failed")
except Exception as exc:
    envelope["errors"].append({"class":"invalid_input" if isinstance(exc,ValueError) else "kernel_runtime",
                               "detail":str(exc)[:160],"where":envelope["phase"]})
envelope["phase"]="report"
print(json.dumps(envelope, ensure_ascii=False))
