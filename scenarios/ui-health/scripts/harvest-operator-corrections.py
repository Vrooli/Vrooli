#!/usr/bin/env python3
"""Locally harvest candidate operator UX corrections; never sends transcript text out.

Default mode reports counts only. --write creates a private JSONL candidate file under
~/.vrooli/research/... and sets restrictive permissions. All rows still require review
and labels before they become evaluation instances.
"""
from __future__ import annotations
import argparse, glob, hashlib, json, os, re, sqlite3
from pathlib import Path

HOME = Path.home()
PROJECT = HOME / "Vrooli"
CORRECTION = re.compile(r"\b(looks? terrible|looks? horrible|look(s|ed)? (really )?bad|unprofessional|unintuitive|not intuitive|not clear|unclear|confusing|crowded|cluttered|poor (ux|user experience|design|layout)|too (big|small|large|tiny|crowded|busy|much|many)|not (enough|great|good|usable|readable)|should have|I (asked|expected|wanted|said)|doesn't (look|work|fit|make)|shouldn't|need(s)? to (fix|improve|change)|would be better|could look better|missing (the|a|an)|wrong (color|colour|spacing|size|layout|navigation)|hard to (use|read|understand|see)|not easily understandable)\b", re.I)
OBSERVED_DEFECT = re.compile(r"\b(terrible|horrible|atrocious|immature|unprofessional|unintuitive|not intuitive|unclear|confusing|crowded|cluttered|poor|bad|wrong|too (big|small|large|tiny|crowded|busy|much|many)|doesn't (look|work|fit|make|appear)|isn't styled|not styled|missing|removed|not appearing|hard to (use|read|understand|see)|not make sense|not understandable|not professional|not polished|not great|issues?|problems?)\b", re.I)
SENSITIVE = re.compile(r"(?:[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}|\b(?:sk|pk|ghp|github_pat|xox[baprs])-[-\w]{12,}|\b\d{3}-\d{2}-\d{4}\b|\b\d{13,19}\b|https?://|\b(?:patient|customer|client|account holder|medical record)\b)", re.I)
QUOTE = re.compile(r"(?:^|\n)\s*>|```|<attached|\[image\]", re.I)
SYNTHETIC = re.compile(r"<codex_internal_context|<pasted_content|<conversation>|<message number=|task-notification|teammate-message|another claude session sent|user-provided data|treat it as the task|continue working toward the active thread goal|your objective below|objective below is", re.I)
TARGET_UI = re.compile(r"\b(ui|ux|user interface|interface|screen|page|layout|spacing|button|tab|dropdown|navigation|menu|font|text size|color|colour|mobile|phone|desktop|touch target|viewport|header|footer|sidebar|visual|appearance|modal|dialog|toolbar|icon|typography|rcl|component library)\b", re.I)

def scenario_matcher():
    names=[]
    for path in (PROJECT/"scenarios").iterdir():
        if not path.is_dir() or not (path/"ui").exists(): continue
        names.append(re.escape(path.name).replace(r"\-", r"[\s-]+"))
    return re.compile(r"\b(?:"+"|".join(sorted(names,key=len,reverse=True))+r")\b",re.I) if names else re.compile(r"(?!)")

SCENARIO = scenario_matcher()

def text_content(value):
    if isinstance(value, str): return value
    if isinstance(value, list):
        return "\n".join(str(x.get("text", "")) for x in value if isinstance(x, dict) and x.get("type") in ("text", "input_text"))
    return ""

def candidate(text):
    text = text.strip()
    if not (text and len(text) <= 10000 and TARGET_UI.search(text) and SCENARIO.search(text) and not SENSITIVE.search(text) and not QUOTE.search(text) and not SYNTHETIC.search(text)):
        return False
    # Keep the UX cue and correction language in the same short passage so an
    # unrelated UI mention cannot pull an infrastructure request into the set.
    return any(TARGET_UI.search(text[i:i+800]) and CORRECTION.search(text[i:i+800]) and OBSERVED_DEFECT.search(text[i:i+800])
               for i in range(0, len(text), 500))

def read_claude(include_all=False):
    base = HOME / ".claude/projects/-home-matthalloran8-Vrooli"
    for path in sorted(base.glob("*.jsonl")):
        try:
            with path.open(errors="replace") as f:
                for line_no, line in enumerate(f, 1):
                    try: row = json.loads(line)
                    except Exception: continue
                    if row.get("type") != "user": continue
                    message = row.get("message") or {}
                    blocks = message.get("content") if isinstance(message, dict) else row.get("content")
                    text = text_content(blocks)
                    if include_all or candidate(text):
                        images=[]
                        if isinstance(blocks,list):
                            for index,block in enumerate(blocks):
                                if isinstance(block,dict) and block.get("type") == "image":
                                    source=block.get("source") or {}
                                    images.append({"block":index,"media_type":source.get("media_type",""),"bytes":len(source.get("data", ""))})
                        yield {"source":"claude", "file":path.name, "message":str(line_no), "message_id":str(message.get("id", "")), "session_id":row.get("sessionId", ""), "timestamp":row.get("timestamp", ""), "image_metadata":images, "text":text}
        except OSError: continue

def codex_user_text(row):
    payload = row.get("payload") or {}
    if row.get("type") == "event_msg" and payload.get("type") == "user_message":
        return text_content(payload.get("message"))
    if row.get("type") == "response_item" and payload.get("role") == "user":
        return text_content(payload.get("content"))
    return ""

def read_codex(include_all=False):
    base = HOME / ".codex/sessions/2026"
    for path in sorted(base.glob("**/rollout-*.jsonl")):
        try:
            with path.open(errors="replace") as f:
                first = f.readline()
                try: meta_row = json.loads(first)
                except ValueError: continue
                meta = meta_row.get("payload", {}) if meta_row.get("type") == "session_meta" else {}
                if Path(str(meta.get("cwd", ""))).resolve() != PROJECT.resolve(): continue
                for line in f:
                    if not line.strip(): continue
                    try: row = json.loads(line)
                    except ValueError: continue
                    payload=row.get("payload") or {}
                    text = codex_user_text(row)
                    if include_all or candidate(text):
                        content=payload.get("content", [])
                        images=[{"block":i,"type":part.get("type","")} for i,part in enumerate(content) if isinstance(part,dict) and part.get("type") in ("input_image","image")]
                        item=payload.get("item") or {}
                        yield {"source":"codex", "file":path.name, "run_id":str(meta.get("id", "")), "message":str(row.get("ordinal", "")), "message_id":str(item.get("id", payload.get("id", ""))), "timestamp":row.get("timestamp", ""), "image_metadata":images, "text":text}
        except OSError: continue

def read_web_console(include_all=False):
    path = HOME / ".vrooli/data/vrooli/web-console/web-console.db"
    if not path.exists(): return
    try:
        db = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
        for row in db.execute("SELECT id, session_id, source, text, created_at,native_provenance_json FROM conversation_events WHERE role='user' ORDER BY created_at, sequence"):
            ident, session, source, text, timestamp, provenance_raw = row
            if include_all or candidate(text):
                try: provenance=json.loads(provenance_raw or "{}")
                except ValueError: provenance={}
                yield {"source":"web-console", "file":str(session), "message":str(ident), "message_id":str(provenance.get("messageId", "")), "source_session_id":str(provenance.get("sessionId", "")), "timestamp":timestamp or "", "image_metadata":[], "text":text}
        db.close()
    except sqlite3.Error: return

def read_agent_manager(include_all=False):
    """Read the local imported conversation archive directly, user messages only."""
    path = HOME / ".vrooli/data/vrooli/agent-manager/legacy-history/agent-manager.db"
    if not path.exists(): return
    try:
        db = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
        query = """SELECT e.id,e.run_id,e.timestamp,e.data,r.tag,r.import_source_harness,r.import_source_session_id,r.web_console_session_id
                   FROM run_events e JOIN runs r ON r.id=e.run_id
                   WHERE e.event_type='message' AND json_extract(e.data,'$.role')='user'
                     AND r.tag LIKE 'agent-manager-imported%'
                   ORDER BY e.timestamp,e.id"""
        for ident, run_id, timestamp, raw, tag, harness, source_session, web_session in db.execute(query):
            try: message=json.loads(raw)
            except ValueError: continue
            text=text_content(message.get("content"))
            if include_all or candidate(text):
                yield {"source":"agent-manager","file":str(run_id),"message":str(ident),
                       "message_id":str(message.get("messageId", "")),"conversation_id":str(message.get("conversationId", "")),
                       "source_session_id":source_session or "","web_console_session_id":web_session or "",
                       "timestamp":timestamp or "","harness":harness or tag,"image_metadata":[],"text":text}
        db.close()
    except sqlite3.Error: return

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--write", action="store_true", help="write private candidates; default prints counts only")
    ap.add_argument("--reuse-existing", action="store_true", help="re-filter the existing private candidate file and add Agent Manager archive rows without rereading native transcript trees")
    ap.add_argument("--output", type=Path, default=HOME/".vrooli/research/topics/experience-engineering/runs/steward-raise-ux-tooling-20261112/private/candidates.jsonl")
    args=ap.parse_args()
    if args.reuse_existing and args.output.exists():
        rows=[]
        with args.output.open(errors="replace") as f:
            for line in f:
                try: row=json.loads(line)
                except ValueError: continue
                if candidate(row.get("text", "")): rows.append(row)
        rows.extend(read_agent_manager())
    else:
        rows=list(read_claude())+list(read_codex())+list(read_web_console())+list(read_agent_manager())
    rows.sort(key=lambda x:(x["timestamp"],x["source"],x["file"],x["message"]))
    deduped=[]
    seen=set()
    for row in rows:
        normalized=re.sub(r"\W+", " ", row["text"].lower()).strip()
        if normalized in seen: continue
        seen.add(normalized); deduped.append(row)
    rows=deduped
    for row in rows:
        key="|".join((row.get("source",""),row.get("file",""),row.get("message_id",row.get("message","")),row.get("timestamp","")))
        row["instance_id"]=hashlib.sha256(key.encode()).hexdigest()[:20]
    sources=("claude","codex","web-console","agent-manager")
    print(json.dumps({"candidates":len(rows),"by_source":{source:sum(r["source"]==source for r in rows) for source in sources},"with_image_metadata":sum(bool(r.get("image_metadata")) for r in rows),"text_written":bool(args.write)}))
    if args.write:
        args.output.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        os.chmod(args.output.parent, 0o700)
        with args.output.open("w", encoding="utf-8") as out:
            for row in rows: out.write(json.dumps(row, ensure_ascii=False)+"\n")
        os.chmod(args.output, 0o600)
        print(json.dumps({"private_output":str(args.output),"rows":len(rows)}))

if __name__ == "__main__": main()
