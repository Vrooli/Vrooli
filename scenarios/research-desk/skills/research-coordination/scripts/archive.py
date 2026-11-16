#!/usr/bin/env python3
"""Move heavy, rebuildable material out of closed research runs, leaving a pointer behind.

  archive.py plan  [--home H] [--archive A] [--topic T] [--json]
  archive.py apply [--home H] [--archive A] --topic T [--run R] [--json]

Candidates live under topics/<t>/runs/<run>/ and are one of two kinds:
  rehearsal   a copy of the Vrooli repository a run made to rehearse on (it holds scenarios/
              plus VISION.md, vrooli.lock or AGENTS.md); archived, never deleted.
  rebuildable dependency and build trees (node_modules, .venv, dist, target, ...); archived
              where the storage owner may reclaim them.
A run is closed when nothing in it outside its candidates changed for --quiet-hours (24) and
no live or paused commission names its topic (apply refuses when Research Desk cannot
say; --no-desk accepts the quiet-hours rule alone). A candidate that any Markdown or JSON file of
the topic links into is kept: references must keep resolving.

apply moves each candidate with one rename (same filesystem only; never copy, never delete) to
<archive>/<kind>/<topic>/<path inside the topic>, writes <name>.ARCHIVED.md beside the old
place (new path, size, date, the command to restore it) and appends to <archive>/<kind>/index.jsonl. Research Desk declares both kind folders as
storage: rehearsal copies are kept (alarm-only budget); rebuildable trees are regenerable, so
the storage owner may reclaim them once old, leaving the pointer to say what was there.
Standard library; run with python3 -I. Exit 0 done, 1 something refused, 2 usage error.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import subprocess
import sys
import time
from pathlib import Path

REBUILDABLE = {"node_modules", ".venv", "venv", "dist", "dist-electron", "build", "builds", "target",
               ".next", "__pycache__", ".pytest_cache", "coverage", ".cache", "site-packages"}
REPO_MARKERS = ("VISION.md", "vrooli.lock", "AGENTS.md")
MAX_DEPTH = 4
REF_SUFFIXES = (".md", ".json")


def kind_of(d: Path) -> str | None:
    if d.name in REBUILDABLE:
        return "rebuildable"
    if (d / "scenarios").is_dir() and any((d / m).exists() for m in REPO_MARKERS):
        return "rehearsal"
    return None


def scan_run(run: Path) -> tuple[list[tuple[Path, str]], float]:
    """Candidates in a run and the newest mtime of everything else in it."""
    found: list[tuple[Path, str]] = []
    newest = run.stat().st_mtime
    stack = [(run, 0)]
    while stack:
        d, depth = stack.pop()
        try:
            entries = list(os.scandir(d))
        except OSError:
            continue
        for e in entries:
            try:
                if e.is_symlink():
                    continue
                if e.is_dir():
                    p = Path(e.path)
                    k = kind_of(p)
                    if k:
                        found.append((p, k))
                    elif depth < MAX_DEPTH:
                        stack.append((p, depth + 1))
                    newest = max(newest, e.stat().st_mtime) if not k else newest
                else:
                    newest = max(newest, e.stat().st_mtime)
            except OSError:
                continue
    return sorted(found), newest


def references(topic: Path, run: Path, targets: list[Path]) -> dict[Path, str]:
    """Targets a Markdown or JSON file of the topic links into, with one citing file. Outside
    the run a link names `<run>/<path>/`; inside it, the path from the run folder; beside the
    target, just `<name>/`."""
    keys = {t: t.relative_to(run).as_posix() + "/" for t in targets}
    hits: dict[Path, str] = {}
    for root, dirs, files in os.walk(topic):
        here = Path(root)
        dirs[:] = [x for x in dirs if here / x not in keys and x not in REBUILDABLE]
        for f in files:
            if not f.endswith(REF_SUFFIXES) or f.endswith(".ARCHIVED.md"):
                continue
            try:
                text = (here / f).read_text(encoding="utf-8", errors="replace")
            except OSError:
                continue
            inside = here == run or run in here.parents
            for t, key in keys.items():
                cited = key if inside else run.name + "/" + key
                if t not in hits and (cited in text or (here == t.parent and t.name + "/" in text)):
                    hits[t] = str((here / f).relative_to(topic))
    return hits


def live_topics() -> tuple[set[str], str]:
    """Topics of live or paused commissions, from Research Desk when it answers."""
    out: set[str] = set()
    for state in ("live", "paused"):
        try:
            r = subprocess.run(["research-desk", "commissions", "--state", state, "--json"],
                               capture_output=True, text=True, timeout=60)
            data = json.loads(r.stdout)
        except (OSError, ValueError, subprocess.TimeoutExpired) as exc:
            return out, f"Research Desk unavailable ({exc.__class__.__name__}); only the quiet-hours rule guards live runs"
        items = data.get("commissions", data) if isinstance(data, dict) else data
        for c in items or []:
            for t in c.get("topics") or []:
                out.add(t.get("topic", "") if isinstance(t, dict) else str(t))
    return out - {""}, ""


def du(paths: list[Path]) -> int:
    if not paths:
        return 0
    try:
        r = subprocess.run(["du", "-s", "--block-size=1", "-c", *map(str, paths)], capture_output=True, text=True, timeout=600)
        return int(r.stdout.strip().splitlines()[-1].split()[0])
    except (OSError, ValueError, IndexError, subprocess.TimeoutExpired):
        return -1


def plan(home: Path, topic: str | None, run: str | None, quiet_hours: float, live: set[str]) -> dict:
    now = time.time()
    items, refused = [], []
    topics = [home / "topics" / topic] if topic else sorted((home / "topics").iterdir())
    for t in topics:
        runs = t / "runs"
        if not runs.is_dir():
            continue
        for r in sorted(runs.iterdir()):
            if not r.is_dir() or (run and r.name != run):
                continue
            cands, newest = scan_run(r)
            if not cands:
                continue
            why = ""
            if t.name in live:
                why = "a live or paused commission names this topic"
            elif now - newest < quiet_hours * 3600:
                why = f"written {int((now - newest) / 3600)} h ago"
            if why:
                refused += [{"path": str(c.relative_to(home)), "kind": k, "why": why} for c, k in cands]
                continue
            refs = references(t, r, [c for c, _ in cands])
            for c, k in cands:
                if c in refs:
                    refused.append({"path": str(c.relative_to(home)), "kind": k, "why": "linked from " + refs[c]})
                else:
                    items.append({"path": str(c.relative_to(home)), "kind": k, "topic": t.name})
    return {"items": items, "refused": refused}


def apply(home: Path, archive: Path, result: dict) -> list[dict]:
    moved = []
    today = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%MZ")
    for it in result["items"]:
        src = home / it["path"]
        dest = archive / it["kind"] / Path(it["path"]).relative_to("topics")
        if dest.exists():
            result["refused"].append({**it, "why": f"{dest} already exists"})
            continue
        size = du([src])
        dest.parent.mkdir(parents=True, exist_ok=True)
        try:
            os.rename(src, dest)
        except OSError as exc:
            result["refused"].append({**it, "why": f"not moved ({exc.strerror}); archive only by rename on one filesystem"})
            continue
        pointer = src.with_name(src.name + ".ARCHIVED.md")
        pointer.write_text(
            f"# {src.name} — archived\n\nMoved on {today} to `{dest}` ({size / 2**20:.0f} MiB, kind {it['kind']}).\n"
            f"Nothing was deleted. Restore with:\n\n    mv '{dest}' '{src}'\n"
            + ("\nA rebuildable tree may later be reclaimed by the storage owner; if the path is gone, rebuild it\n"
               "with the run's own install or build command.\n" if it["kind"] == "rebuildable" else ""), encoding="utf-8")
        record = {**it, "to": str(dest), "bytes": size, "at": today}
        with (archive / it["kind"] / "index.jsonl").open("a", encoding="utf-8") as f:
            f.write(json.dumps(record) + "\n")
        moved.append(record)
    return moved


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(prog="archive.py", description=__doc__.split("\n\n")[0])
    p.add_argument("mode", choices=["plan", "apply"])
    p.add_argument("--home", type=Path, default=Path.home() / ".vrooli" / "research")
    p.add_argument("--archive", type=Path, default=Path.home() / ".vrooli" / "archive" / "research")
    p.add_argument("--topic")
    p.add_argument("--run")
    p.add_argument("--quiet-hours", type=float, default=24)
    p.add_argument("--no-desk", action="store_true", help="do not ask Research Desk for live topics")
    p.add_argument("--json", action="store_true")
    a = p.parse_args(argv)
    if a.mode == "apply" and not a.topic:
        p.error("apply needs --topic")
    if a.topic and ("/" in a.topic or not (a.home / "topics" / a.topic).is_dir()):
        p.error(f"no topic {a.topic!r} under {a.home}")
    live, note = (set(), "live check skipped (--no-desk)") if a.no_desk else live_topics()
    if note and not a.no_desk and a.mode == "apply":
        print(f"refused: {note}. Pass --no-desk to move on the quiet-hours rule alone.", file=sys.stderr)
        return 1
    result = plan(a.home, a.topic, a.run, a.quiet_hours, live)
    result["total_bytes"] = du([a.home / i["path"] for i in result["items"]])
    if note:
        result["note"] = note
    if a.mode == "apply":
        result["moved"] = apply(a.home, a.archive, result)
    if a.json:
        print(json.dumps(result, indent=2))
    else:
        verb = "moved" if a.mode == "apply" else "would move"
        rows = result.get("moved", result["items"])
        print(f"{verb} {len(rows)} item(s), {max(result['total_bytes'], 0) / 2**30:.1f} GiB on disk"
              + (f" (archive: {a.archive})" if a.mode == "apply" else ""))
        for it in rows:
            print(f"  {it['kind']:<11} {it['path']}")
        for it in result["refused"]:
            print(f"  kept        {it['path']} — {it['why']}")
        if note:
            print("note: " + note)
    return 1 if a.mode == "apply" and len(result.get("moved", [])) < len(result["items"]) else 0


if __name__ == "__main__":
    sys.exit(main())
