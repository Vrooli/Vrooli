"""Bounded change investigation over immutable prior/current cell observations."""
import hashlib
import json

inputs = program.inputs()
envelope={"program":"web-search.change-investigation","version":"1","status":"failed","phase":"validate","signals":{"results":[],"unresolved":[]},"errors":[],"evidence":[]}
try:
    if not isinstance(inputs,dict) or set(inputs)-{"prior_investigation_id","prior_cells","current_cells"}:
        raise ValueError("prior_investigation_id, prior_cells, and current_cells are required")
    prior_id=inputs.get("prior_investigation_id")
    prior=inputs.get("prior_cells")
    current=inputs.get("current_cells")
    if not isinstance(prior_id,str) or not prior_id.strip() or not isinstance(prior,list) or not isinstance(current,list):
        raise ValueError("prior investigation and cell arrays are required")
    if len(prior)>32 or len(current)>32:
        raise ValueError("at most 32 cells are allowed per side")
    def index(rows):
        out={}
        for row in rows:
            if not isinstance(row,dict) or not isinstance(row.get("cell_id"),str) or not row["cell_id"].strip():
                raise ValueError("each cell requires a bounded cell_id")
            if row["cell_id"] in out:
                raise ValueError("duplicate cell_id")
            out[row["cell_id"]]=row
        return out
    old=index(prior)
    new=index(current)
    envelope["phase"]="act"
    for cell_id in sorted(set(old)|set(new)):
        a,b=old.get(cell_id),new.get(cell_id)
        result={"cell_id":cell_id,"prior_investigation_id":prior_id}
        if a is None or b is None:
            result["status"]="unverified"
            result["reason"]="missing_prior_evidence" if a is None else "missing_current_evidence"
            envelope["signals"]["unresolved"].append(cell_id)
        elif a.get("effective_date") and b.get("effective_date") and a["effective_date"]!=b["effective_date"]:
            result["status"]="incomparable"
            result["reason"]="conflicting_effective_dates"
            envelope["signals"]["unresolved"].append(cell_id)
        else:
            def fingerprint(row):
                value=row.get("claim",row.get("brief",row.get("summary","")))
                return hashlib.sha256(json.dumps(value,sort_keys=True,separators=(",",":"),ensure_ascii=False).encode()).hexdigest()
            result["status"]="unchanged" if fingerprint(a)==fingerprint(b) else "changed"
        envelope["signals"]["results"].append(result)
    envelope["status"]="partial" if envelope["signals"]["unresolved"] else "ok"
except Exception as exc:
    envelope["errors"].append({"class":"invalid_input","detail":str(exc)[:160],"where":"validate"})
envelope["phase"]="report"
print(json.dumps(envelope,sort_keys=True))
