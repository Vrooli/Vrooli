#!/usr/bin/env python3
"""Turn screened local candidates into a private, chronological EX1 evaluation set.

Suggested defect labels are weak-label hints only. Every instance retains its provenance
and needs curator adjudication before recall/precision claims use it as gold truth.
"""
import argparse, json, os, re
from pathlib import Path

DEFAULT_IN=Path.home()/".vrooli/research/topics/experience-engineering/runs/steward-raise-ux-tooling-20261112/private/candidates.jsonl"
DEFAULT_OUT=Path.home()/".vrooli/research/topics/experience-engineering/runs/steward-raise-ux-tooling-20261112/private/corpus.jsonl"
LABELS={
 "navigation_crowding":re.compile(r"navigation|crowded|clutter|tabs|menu|sidebar|nav",re.I),
 "touch_target":re.compile(r"touch|tap|button|click target|too small|tiny",re.I),
 "typography_hierarchy":re.compile(r"font|text size|typography|heading|readable|hierarchy|oversized",re.I),
 "spacing_layout":re.compile(r"spacing|layout|padding|margin|squish|column|overlap|alignment",re.I),
 "safe_area_chrome":re.compile(r"safe area|status bar|notch|bottom nav|viewport|keyboard",re.I),
 "color_contrast":re.compile(r"contrast|color|colour|background|status bar color",re.I),
 "visual_polish":re.compile(r"horrible|terrible|unprofessional|polish|professional|looks bad|looks terrible",re.I),
 "functionality_regression":re.compile(r"missing|removed|drop(ped)?|no longer|doesn't work|wrong action|functionality",re.I),
 "design_system":re.compile(r"component library|react component|asset|dropdown|tab component|reuse",re.I),
}

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--input",type=Path,default=DEFAULT_IN)
    ap.add_argument("--output",type=Path,default=DEFAULT_OUT)
    ap.add_argument("--write",action="store_true",help="write a private corpus; default reports counts only")
    a=ap.parse_args()
    rows=[]
    with a.input.open() as f:
        for line in f:
            try: row=json.loads(line)
            except ValueError: continue
            row["suggested_labels"]=[k for k,p in LABELS.items() if p.search(row.get("text",""))]
            row["label_state"]="unreviewed"
            row["split"]=""
            rows.append(row)
    rows.sort(key=lambda r:(r.get("timestamp", ""),r.get("source", ""),r.get("file", ""),r.get("message", "")))
    cut=int(len(rows)*0.7)
    for i,row in enumerate(rows): row["split"]="train" if i<cut else "heldout_latest_30pct"
    counts={label:sum(label in r["suggested_labels"] for r in rows) for label in LABELS}
    print(json.dumps({"instances":len(rows),"training":cut,"heldout":len(rows)-cut,"suggested_labels":counts,"gold_labels":0,"written":bool(a.write)}))
    if a.write:
        a.output.parent.mkdir(parents=True,exist_ok=True,mode=0o700); os.chmod(a.output.parent,0o700)
        with a.output.open("w",encoding="utf8") as f:
            for row in rows: f.write(json.dumps(row,ensure_ascii=False)+"\n")
        os.chmod(a.output,0o600)
        print(json.dumps({"private_corpus":str(a.output),"instances":len(rows)}))

if __name__=="__main__":main()
