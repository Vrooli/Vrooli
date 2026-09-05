"""Capture observed outcomes separately from retrieved findings and model confidence."""
import json

try:
    inputs
except NameError:
    inputs = {}
envelope={"program":"web-search.record-attempt","version":"1","status":"failed","phase":"validate","signals":{},"errors":[],"evidence":[]}
try:
    if not isinstance(inputs,dict) or set(inputs)-{"attempt","quality","used_finding_ids"}:
        raise ValueError("Supply attempt, quality, and selected used_finding_ids")
    attempt=dict(inputs.get("attempt") or {})
    quality=inputs.get("quality") or {}
    used=inputs.get("used_finding_ids") or []
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
    assessment={"schema":"web-search-quality/v1","quality":quality,"contributing_finding_ids":used}
    # One durable owner write preserves assessment and contributions atomically.
    # Legacy finding counters are not an outcome ledger and cannot be retried safely.
    attempt["approach"]=(attempt.get("approach","")+"; Research assessment: "+json.dumps(assessment,sort_keys=True,separators=(",",":"))).strip()
    if len(attempt["approach"].encode("utf-8"))>1024:
        raise ValueError("Approach plus assessment exceeds 1024 bytes; shorten the approach or select fewer findings")
    envelope["phase"]="report"
    receipt=vrooli_memory.learning.record(scope="web-search-usage",attempt=attempt)
    meta=receipt.meta() or {}
    rows=receipt.head(1)
    result=rows[0] if rows else meta
    envelope["signals"]={"entry_id":result.get("entryId"),"existing":result.get("existing",False),"contributing_finding_ids":used,"quality":quality}
    envelope["evidence"]=list(attempt.get("evidence_refs",[]))[:10]
    envelope["status"]="ok"
except Exception as exc:
    envelope["errors"]=[{"class":"invalid_input" if isinstance(exc,ValueError) else "capture_failed","detail":str(exc)[:160],"where":envelope["phase"]}]
    if envelope["phase"]=="report":envelope["status"]="partial"
envelope["phase"]="report"
print(json.dumps(envelope, sort_keys=True))
