"""Capture observed outcomes separately from retrieved findings and model confidence."""
import json

inputs = program.inputs()
envelope={"program":"web-search.record-attempt","version":"1","status":"failed","phase":"validate","signals":{},"errors":[],"evidence":[]}
try:
    if not isinstance(inputs,dict) or set(inputs)-{"attempt","quality","used_finding_ids"}:
        raise ValueError("Supply attempt, quality, and selected used_finding_ids")
    if not isinstance(inputs.get("attempt"),dict):
        raise ValueError("Supply a typed attempt object")
    attempt=dict(inputs["attempt"])
    quality=inputs.get("quality",{})
    used=inputs.get("used_finding_ids",[])
    if not isinstance(quality,dict) or not isinstance(used,list) or any(not isinstance(fid,str) or not fid.strip() for fid in used):
        raise ValueError("Quality must be an object and finding IDs nonempty strings")
    if len(used)>10 or len(set(used))!=len(used):
        raise ValueError("Select at most ten unique contributing findings")
    if set(quality)-{"citation_support","freshness","coverage","contradictions_resolved"} or any(value is not None and not isinstance(value,bool) for value in quality.values()):
        raise ValueError("Quality checks must be boolean or unknown")
    if attempt.get("outcome")=="verified_success":
        if not all(quality.get(k) is True for k in ("citation_support","freshness","coverage","contradictions_resolved")) or not attempt.get("evidence_refs"):
            raise ValueError("Verified research success requires observed quality checks and evidence references")
    elif used:
        raise ValueError("Only verified successful contribution can record findings use")
    if not isinstance(attempt.get("approach"),str) or not attempt["approach"].strip():
        raise ValueError("Supply the observed approach before recording its quality assessment")
    envelope["phase"]="report"
    observations=[]
    if quality:
        dispositions=["supported" if quality.get(key) is True else "contradicted" if quality.get(key) is False else "unknown"
                      for key in ("citation_support","freshness","coverage","contradictions_resolved")]
        disposition="contradicted" if "contradicted" in dispositions else "unknown" if "unknown" in dispositions else "supported"
        observation={"observation_id":str(attempt.get("attempt_id",""))+"-quality","attempt_id":attempt.get("attempt_id",""),"disposition":disposition,"evidence_refs":list(attempt.get("evidence_refs",[]))[:20],"method_revision":attempt.get("context_key",""),"correction":json.dumps({"schema":"web-search-quality/v1","quality":quality,"contributing_finding_ids":used},sort_keys=True,separators=(",",":")),"provenance":attempt.get("provenance",""),"observed_at":attempt.get("finished_at","")}
        # Let the shared owner assign stable IDs when the caller omitted an attempt ID.
        if not attempt.get("attempt_id"):
            observation.pop("attempt_id")
            observation.pop("observation_id")
        observations.append(observation)
    rows=lib.vrooli_memory.finish_attempt(scope="web-search-usage",attempt=attempt,observations=observations).head(1)
    if not rows:
        raise RuntimeError("Shared finish returned no envelope")
    child=rows[0]
    captured=child.get("signals",{})
    signals={"entry_id":captured.get("entry_id"),"existing":captured.get("existing",False),
             "contributing_finding_ids":used,"quality":quality,
             "task_outcome":captured.get("task_outcome",attempt.get("outcome","unknown")),
             "capture_status":captured.get("capture_status","unknown"),
             "retry_inputs":captured.get("retry_inputs")}
    receipts=captured.get("observations",[])
    if receipts:
        signals["observation_id"]=receipts[0].get("entry_id") or receipts[0].get("observation_id")
        signals["observation_existing"]=receipts[0].get("existing",False)
    envelope["signals"]=signals
    envelope["evidence"]=list(attempt.get("evidence_refs",[]))[:10]
    envelope["status"]=child.get("status","failed")
    envelope["errors"]=child.get("errors",[])[:10]

except Exception as exc:
    envelope["errors"]=[{"class":"invalid_input" if isinstance(exc,ValueError) else "capture_failed","detail":str(exc)[:160],"where":envelope["phase"]}]
    if envelope["phase"]=="report":envelope["status"]="partial"
envelope["phase"]="report"
print(json.dumps(envelope, sort_keys=True))
