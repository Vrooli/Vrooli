#!/usr/bin/env python3
"""Materialize only inline, operator-provided Codex screenshots already selected for EX1.

Inputs are read from the native local Codex history. No URL/path attachments are fetched;
no image is sent to an external service. Outputs stay in the mode-0700 EX1 private corpus.
"""
import argparse, base64, hashlib, json, os, re
from pathlib import Path

HOME=Path.home()
DEFAULT_CORPUS=HOME/".vrooli/research/topics/experience-engineering/runs/steward-raise-ux-tooling-20261112/private/corpus.jsonl"
DEFAULT_DIR=HOME/".vrooli/research/topics/experience-engineering/runs/steward-raise-ux-tooling-20261112/private/screenshots"
MAX_IMAGE=10*1024*1024
MAX_TOTAL=25*1024*1024

def read_images(row):
    source=HOME/".codex/sessions/2026"
    paths=list(source.glob("**/"+row["file"]))
    if len(paths)!=1: return []
    try:
        with paths[0].open(errors="replace") as f:
            for line in f:
                try: item=json.loads(line)
                except ValueError: continue
                if str(item.get("ordinal")) != str(row.get("message")): continue
                content=(item.get("payload") or {}).get("content",[])
                out=[]
                for index,part in enumerate(content):
                    if not isinstance(part,dict) or part.get("type")!="input_image": continue
                    value=part.get("image_url","")
                    if not isinstance(value,str) or not value.startswith("data:") or "," not in value: continue
                    header,payload=value.split(",",1)
                    match=re.fullmatch(r"data:(image/(?:png|jpeg|webp));base64",header,re.I)
                    if not match: continue
                    try: data=base64.b64decode(payload,validate=True)
                    except ValueError: continue
                    if 0 < len(data) <= MAX_IMAGE: out.append((index,match.group(1).lower(),data))
                return out
    except OSError: return []
    return []

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--corpus",type=Path,default=DEFAULT_CORPUS)
    ap.add_argument("--output-dir",type=Path,default=DEFAULT_DIR)
    ap.add_argument("--write",action="store_true",help="write screenshots into the private corpus")
    args=ap.parse_args()
    rows=[json.loads(line) for line in args.corpus.open() if line.strip()]
    selected=[]
    for row in rows:
        if row.get("source")!="codex": continue
        for index,mime,data in read_images(row): selected.append((row,index,mime,data))
    total=sum(len(item[3]) for item in selected)
    if total>MAX_TOTAL: raise SystemExit(f"refusing screenshot set above {MAX_TOTAL} bytes")
    print(json.dumps({"instances_with_inline_screenshots":len({r['instance_id'] for r,_,_,_ in selected}),"images":len(selected),"bytes":total,"written":bool(args.write)}))
    if not args.write: return
    args.output_dir.mkdir(parents=True,exist_ok=True,mode=0o700); os.chmod(args.output_dir,0o700)
    ext={"image/png":"png","image/jpeg":"jpg","image/webp":"webp"}
    by_instance={}
    for row,index,mime,data in selected:
        name=f"{row['instance_id']}-image-{index}.{ext[mime]}"
        path=args.output_dir/name
        path.write_bytes(data); os.chmod(path,0o600)
        by_instance.setdefault(row["instance_id"],[]).append({"path":str(path),"media_type":mime,"bytes":len(data),"sha256":hashlib.sha256(data).hexdigest()})
    output=args.corpus.with_name("corpus-with-screenshots.jsonl")
    with output.open("w",encoding="utf8") as f:
        for row in rows:
            if row.get("instance_id") in by_instance: row["private_screenshots"]=by_instance[row["instance_id"]]
            f.write(json.dumps(row,ensure_ascii=False)+"\n")
    os.chmod(output,0o600)
    print(json.dumps({"private_corpus":str(output),"screenshot_files":len(selected)}))

if __name__=="__main__": main()
