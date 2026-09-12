"""Versioned comparative research method with explicit bounded cells."""
import json

inputs = program.inputs()
envelope={"program":"web-search.compare-sources","version":"2","status":"failed","phase":"validate","signals":{"answers":[],"cells":[],"unresolved":[]},"errors":[],"evidence":[]}
try:
    allowed={"questions","subjects","dimensions","temporal_scope","source_domains","minimum_sources","top_n"}
    if not isinstance(inputs,dict) or set(inputs)-allowed:
        raise ValueError("Only comparison cells and evidence policy are accepted")
    questions=inputs.get("questions")
    domains=inputs.get("source_domains") or []
    if not isinstance(questions,list) or not 2<=len(questions)<=4 or any(not isinstance(q,str) or not q.strip() or len(q)>4096 for q in questions):
        raise ValueError("Supply two to four explicit comparison questions")
    subjects=inputs.get("subjects") or [""]
    dimensions=inputs.get("dimensions") or [""]
    temporal=inputs.get("temporal_scope") if "temporal_scope" in inputs else ""
    if not isinstance(subjects,list) or not 1<=len(subjects)<=4 or any(not isinstance(x,str) or ("subjects" in inputs and not x.strip()) or len(x)>512 for x in subjects):
        raise ValueError("subjects must contain one to four bounded names")
    if not isinstance(dimensions,list) or not 1<=len(dimensions)<=8 or any(not isinstance(x,str) or ("dimensions" in inputs and not x.strip()) or len(x)>512 for x in dimensions):
        raise ValueError("dimensions must contain one to eight bounded names")
    if not isinstance(temporal,str) or ("temporal_scope" in inputs and not temporal.strip()) or len(temporal)>256:
        raise ValueError("temporal_scope must be a bounded string")
    work=[]
    for subject in subjects:
        for question in questions:
            for dimension in dimensions:
                cell_id=question if not any(k in inputs for k in ("subjects","dimensions","temporal_scope")) else f"{subject}|{question}|{dimension}|{temporal}"
                work.append({"cell_id":cell_id,"subject":subject,"question":question,"dimension":dimension,"temporal_scope":temporal,"query":"; ".join(x for x in [subject,question,dimension,temporal] if x)})
    if len(work)>32:
        raise ValueError("comparison expands to at most 32 cells")
    envelope["phase"]="act"
    for cell in work:
        question=cell["query"]
        try:
            handle=web_search.research.answer(query=question,effort="l2",max_age_seconds=0,source_domains=domains,minimum_sources=inputs.get("minimum_sources",2),top_n=inputs.get("top_n",5),capture=False,rows="results")
            meta=handle.meta() or {}
            brief=meta.get("brief") or {}
            if len(json.dumps(brief).encode("utf-8"))>14000:
                envelope["signals"]["unresolved"].append(cell["cell_id"])
                envelope["errors"].append({"class":"answer_output_limit","detail":"Answer exceeds comparison handoff; inspect this question individually","where":"act"})
                continue
            row={**cell,"brief":brief,"checked_at":meta.get("checkedAt"),"status":meta.get("status"),"abstained":meta.get("abstained",False),"gaps":meta.get("gaps",[])}
            envelope["signals"]["answers"].append(row)
            envelope["signals"]["cells"].append(row)
            if meta.get("abstained") or meta.get("status")!="ok" or meta.get("answerKind")!="cited_synthesis":
                envelope["signals"]["unresolved"].append(cell["cell_id"])
            envelope["evidence"].extend([c.get("url") for c in brief.get("citations",[]) if c.get("url")])
        except Exception as exc:
            envelope["signals"]["unresolved"].append(cell["cell_id"])
            envelope["errors"].append({"class":"source_unavailable","detail":str(exc)[:120],"where":"act"})
    envelope["status"]="partial" if envelope["signals"]["unresolved"] else "ok"
except Exception as exc:
    envelope["errors"].append({"class":"invalid_input","detail":str(exc)[:160],"where":"validate"})
envelope["phase"]="report"
print(json.dumps(envelope, sort_keys=True))
