#!/usr/bin/env python3
"""Append, list and drain level inboxes, and append verdicts, with the clock stamping every entry.

Usage:
  inbox.py append <folder> --from SENDER --kind KIND [--source S] [--about A] (--body TEXT | --body-file F)
  inbox.py list <folder> [--json]
  inbox.py drain <folder> --entry STAMP [--nth N] --did "<what you did>"
  inbox.py verdict <review folder> --topic T --source SOURCE (--body TEXT | --body-file F)

<folder> holds INBOX.md: a topic folder, the research home (supervisor level) or the CEO's folder.
An entry is `### <UTC> · from <sender> · <kind>`, its body, then optional `Source:` (where
operator words came from) and `About:` (whom quoted words addressed) lines; the grammar is
docs/agent-system/LEVELS.md. The time is always this machine's UTC clock; nobody types it. Drain
moves one entry verbatim to <folder>/history/INBOX.md with an `Acted:` line; a malformed heading (which
append never writes) drains by its first token, so nothing sticks. A verdict is the operator's words about one topic with their source (a Hub answer
id, an OPERATOR.md line, or the operator's chat with session and UTC); stance quoted in a brief is
never a verdict, so a verdict without a source is refused. Exit 0 ok, 1 refused, 2 bad usage.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
import time
from pathlib import Path

KINDS = ("feedback", "friction", "answer", "clarification", "priority", "decision")
SENDER = re.compile(r"^(?:(?:topic|commission):)?[a-z0-9][a-z0-9_.-]{0,127}$")  # operator, ceo, supervisor, topic:<t>, commission:<id>, an agent
HEAD = re.compile(r"^### (\S+) · from (\S+) · (\S+)\s*$")
TOPIC = re.compile(r"^[a-z0-9][a-z0-9-]{0,127}$")


def now() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def refuse(msg: str) -> int:
    print(f"refused: {msg}", file=sys.stderr)
    return 1


def body_of(a) -> str:
    text = Path(a.body_file).read_text(encoding="utf-8") if a.body_file else (a.body or "")
    text = text.strip("\n")
    if not text.strip():
        raise ValueError("the body is empty")
    if re.search(r"^### ", text, re.M):
        raise ValueError("the body may not contain a '### ' heading line")
    return text


def entries(path: Path) -> tuple[str, list[dict]]:
    """The file's preamble and its entries, each with stamp, level, kind, body and raw text."""
    text = path.read_text(encoding="utf-8") if path.exists() else ""
    parts = re.split(r"(?m)^(?=### )", text)
    pre, out = parts[0], []
    for raw in parts[1:]:
        head, _, body = raw.partition("\n")
        m = HEAD.match(head)
        loose = re.match(r"^### (\S+)", head)  # a malformed heading still drains by its first token
        out.append({"stamp": m.group(1) if m else (loose.group(1) if loose else None), "from": m.group(2) if m else None,
                    "kind": m.group(3) if m else None, "body": body.strip("\n"), "raw": raw, "valid": bool(m)})
    return pre, out


def append(path: Path, title: str, block: str) -> None:
    text = path.read_text(encoding="utf-8") if path.exists() else f"# {title}\n"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text.rstrip("\n") + "\n\n" + block.rstrip("\n") + "\n", encoding="utf-8")


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    ap = sub.add_parser("append")
    ap.add_argument("folder")
    ap.add_argument("--from", dest="level", required=True)
    ap.add_argument("--source")
    ap.add_argument("--about")
    ap.add_argument("--kind", required=True, choices=KINDS)
    lp = sub.add_parser("list")
    lp.add_argument("folder")
    lp.add_argument("--json", action="store_true")
    dp = sub.add_parser("drain")
    dp.add_argument("folder")
    dp.add_argument("--entry", required=True, help="the entry's stamp, as `list` prints it")
    dp.add_argument("--nth", type=int, default=0, help="which of several entries sharing the stamp (1 = first)")
    dp.add_argument("--did", required=True)
    vp = sub.add_parser("verdict")
    vp.add_argument("folder")
    vp.add_argument("--topic", required=True)
    vp.add_argument("--source", required=True)
    for sp in (ap, vp):
        g = sp.add_mutually_exclusive_group(required=True)
        g.add_argument("--body")
        g.add_argument("--body-file")
    a = p.parse_args(argv)
    folder = Path(a.folder).expanduser()
    if not folder.is_dir():
        return refuse(f"{folder} is not a folder")
    try:
        if a.cmd == "append":
            if not SENDER.match(a.level):
                return refuse(f"--from {a.level!r}: use operator, ceo, supervisor, topic:<name>, commission:<id> or an agent name")
            body = body_of(a)
            taken = {e["stamp"] for e in entries(folder / "INBOX.md")[1]}
            while now() in taken:  # stamps name entries for drain, so they stay unique
                time.sleep(0.2)
            extra = "".join(f"\n{k}: {' '.join(v.split())}" for k, v in (("Source", a.source), ("About", a.about)) if v)
            block = f"### {now()} · from {a.level} · {a.kind}\n{body}{extra}"
            append(folder / "INBOX.md", f"Inbox — {folder.name}", block)
            print(block.splitlines()[0])
        elif a.cmd == "list":
            _, es = entries(folder / "INBOX.md")
            if a.json:
                print(json.dumps([{k: e[k] for k in ("stamp", "from", "kind", "body", "valid")} for e in es], indent=1))
            else:
                for e in es:
                    print(f"{e['stamp']} · from {e['from']} · {e['kind']}" if e["valid"] else f"MALFORMED {e['raw'].splitlines()[0]}")
        elif a.cmd == "drain":
            path = folder / "INBOX.md"
            pre, es = entries(path)
            hit = [e for e in es if e["stamp"] == a.entry]
            if a.nth:
                hit = hit[a.nth - 1:a.nth]
            if len(hit) != 1:
                return refuse(f"{len(hit)} entries carry the stamp {a.entry!r}; name one with --nth")
            e = hit[0]
            append(folder / "history" / "INBOX.md", f"Inbox history — {folder.name}",
                   e["raw"].rstrip("\n") + f"\nActed: {' '.join(a.did.split())} ({now()})")
            path.write_text(pre.rstrip("\n") + "\n" + "".join("\n" + x["raw"].rstrip("\n") + "\n" for x in es if x is not e),
                            encoding="utf-8")
            print(f"drained {a.entry}")
        else:
            if not TOPIC.match(a.topic):
                return refuse(f"--topic {a.topic!r} is not a topic slug")
            if len(a.source.strip()) < 8:
                return refuse("--source must name the Hub answer id, the OPERATOR.md line, or the chat session and UTC")
            block = f"### {now()} · verdict {a.topic} · via hand\n{body_of(a)}\nSource: {' '.join(a.source.split())}"
            append(folder / "VERDICTS.md", f"Verdicts — {folder.name}", block)
            print(block.splitlines()[0])
    except (OSError, ValueError) as err:
        return refuse(str(err))
    return 0


if __name__ == "__main__":
    sys.exit(main())
