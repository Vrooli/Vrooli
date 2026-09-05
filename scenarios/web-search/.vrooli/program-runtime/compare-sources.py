"""Versioned comparative research method: fixed questions and explicit evidence policy."""
import json

try:
    inputs
except NameError:
    inputs={}
envelope={"program":"web-search.compare-sources","version":"1","status":"failed","phase":"validate","signals":{"answers":[],"unresolved":[]},"errors":[],"evidence":[]}
try:
    if not isinstance(inputs,dict) or set(inputs)-{"questions","source_domains","minimum_sources","top_n"}:
        raise ValueError("Only comparison questions and evidence policy are accepted")
    questions=inputs.get("questions")
    domains=inputs.get("source_domains") or []
    if not isinstance(questions,list) or not 2<=len(questions)<=4 or any(not isinstance(q,str) or not q.strip() or len(q)>4096 for q in questions):
        raise ValueError("Supply two to four explicit comparison questions")
    envelope["phase"]="act"
    for question in questions:
        try:
            handle=web_search.research.answer(query=question,effort="l2",max_age_seconds=0,source_domains=domains,minimum_sources=inputs.get("minimum_sources",2),top_n=inputs.get("top_n",5),capture=False,rows="results")
            meta=handle.meta() or {}
            brief=meta.get("brief") or {}
            if len(json.dumps(brief).encode("utf-8"))>14000:
                envelope["signals"]["unresolved"].append(question)
                envelope["errors"].append({"class":"answer_output_limit","detail":"Answer exceeds comparison handoff; inspect this question individually","where":"act"})
                continue
            envelope["signals"]["answers"].append({"query":question,"brief":brief,"checked_at":meta.get("checkedAt"),"status":meta.get("status"),"abstained":meta.get("abstained",False),"gaps":meta.get("gaps",[])})
            if meta.get("abstained") or meta.get("status")!="ok" or meta.get("answerKind")!="cited_synthesis":
                envelope["signals"]["unresolved"].append(question)
            envelope["evidence"].extend([c.get("url") for c in brief.get("citations",[]) if c.get("url")])
        except Exception as exc:
            envelope["signals"]["unresolved"].append(question)
            envelope["errors"].append({"class":"source_unavailable","detail":str(exc)[:120],"where":"act"})
    envelope["status"]="partial" if envelope["signals"]["unresolved"] else "ok"
except Exception as exc:
    envelope["errors"].append({"class":"invalid_input","detail":str(exc)[:160],"where":"validate"})
envelope["phase"]="report"
print(json.dumps(envelope, sort_keys=True))
