import json

"""Read fixed research learning cohorts without inventing baselines or zero counts."""
try:
    inputs
except NameError:
    inputs = {}
envelope = {"program":"web-search.learning-read","version":"1","status":"failed","phase":"validate","signals":{},"errors":[],"evidence":[]}
try:
    if not isinstance(inputs,dict) or set(inputs)-{"from","to","operation","context_key"}:
        raise ValueError("Only fixed comparison selectors are accepted")
    envelope["phase"]="collect"
    handle=vrooli_memory.learning.measure(scope="web-search-usage",rows="cohorts",**inputs)
    meta=handle.meta() or {}
    count=handle.count()
    reliable=bool(meta.get("reliable",False)) and count<=10
    envelope["signals"]={"cohorts":handle.head(10),"reliable":reliable,"reason":meta.get("reason") or ("cohort_limit" if count>10 else ""),"from":meta.get("from"),"to":meta.get("to"),"eligible_attempts":meta.get("eligibleAttempts",0),"excluded_test_attempts":meta.get("excludedTestAttempts",0),"targets":None,"interpretation":meta.get("interpretation")}
    envelope["evidence"]=list(meta.get("evidenceRefs",[]))[:10]
    envelope["status"]="ok" if reliable else "partial"
except Exception as exc:
    envelope["errors"]=[{"class":"invalid_input" if isinstance(exc,ValueError) else "learning_unavailable","detail":str(exc)[:160],"where":envelope["phase"]}]
    envelope["status"]="failed" if isinstance(exc,ValueError) else "unavailable"
envelope["phase"]="report"
print(json.dumps(envelope, sort_keys=True))
