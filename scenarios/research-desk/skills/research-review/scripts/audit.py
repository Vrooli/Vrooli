#!/usr/bin/env python3
"""Measure a research topic folder before and after an audit.

  audit.py measure <topic> [--home H] [--out FILE]   one JSON snapshot
  audit.py compare <before.json> <after.json>        a before/after table

Read cost is what an owner reads before working (README.md, MISSION.md, CURRENT.md when
present) plus what the folder asks a reader to wade through: top-level files, Markdown outside
runs, oversize pages, broken links, runs without a guide. Throughput comes from history.py and
is reported beside it, because a smaller folder that lands nothing is not an improvement.
Read-only; standard library; run with python3 -I.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
MISSION_CHECK = HERE.parent.parent / "research-coordination" / "scripts" / "mission_check.py"
ENTRY = ("README.md", "MISSION.md", "CURRENT.md")
OVERSIZE = 40_000
LINK = re.compile(r"\]\(([^)\s#]+)(?:#[^)]*)?\)")


def md_files(topic: Path) -> list[Path]:
    return sorted(p for p in topic.rglob("*.md") if "runs" not in p.relative_to(topic).parts[:1]
                  and "history" not in p.relative_to(topic).parts[:1])


def broken_links(topic: Path, files: list[Path]) -> list[str]:
    out = []
    for f in files:
        for target in LINK.findall(f.read_text(encoding="utf-8", errors="replace")):
            if "://" in target or target.startswith("mailto:"):
                continue
            target = re.sub(r"(#.*|:\d+(?:-\d+)?)$", "", target)  # anchors and file.go:104 line refs
            if not target:
                continue
            path = Path(target) if target.startswith("/") else f.parent / target
            if not path.exists():
                out.append(f"{f.relative_to(topic)} -> {target}")
    return out


def run_json(args: list[str]) -> dict | None:
    try:
        res = subprocess.run([sys.executable, "-I", *args], capture_output=True, text=True, timeout=120)
        return json.loads(res.stdout) if res.stdout.strip() else None
    except (OSError, subprocess.TimeoutExpired, json.JSONDecodeError):
        return None


def owner_start(topic: Path, runs: list[Path]) -> dict:
    """What a fresh owner reads before working: MISSION plus the next brief, or, with no plan,
    the newest run's checkpoint it would resume from."""
    read = {"MISSION.md": (topic / "MISSION.md").stat().st_size}
    plan = topic / "work" / "README.md"
    briefs = re.findall(r"^\|\s*\[?`?([A-Za-z0-9][A-Za-z0-9_.-]*)", plan.read_text(encoding="utf-8"), re.M) if plan.is_file() else []
    nxt = next((topic / "work" / b / "BRIEF.md" for b in briefs if (topic / "work" / b / "BRIEF.md").is_file()), None)
    if nxt:
        read[str(nxt.relative_to(topic))] = nxt.stat().st_size
    else:
        cps = [r / "CHECKPOINT.md" for r in runs if (r / "CHECKPOINT.md").is_file()]
        if cps:
            cp = max(cps, key=lambda c: c.stat().st_mtime)
            read[str(cp.relative_to(topic))] = cp.stat().st_size
    return {"bytes": sum(read.values()), "files": read}


def measure(home: Path, name: str) -> dict:
    topic = home / "topics" / name
    if not (topic / "MISSION.md").is_file():
        raise SystemExit(f"{topic} has no MISSION.md")
    entry = {f: (topic / f).stat().st_size for f in ENTRY if (topic / f).is_file()}
    top = [p for p in topic.iterdir() if not p.name.startswith(".")]
    files = md_files(topic)
    runs = [p for p in (topic / "runs").iterdir() if p.is_dir()] if (topic / "runs").is_dir() else []
    mc = run_json([str(MISSION_CHECK), str(home), "--topic", name, "--json"])
    rep = mc[0] if isinstance(mc, list) and mc else None
    hist = run_json([str(HERE / "history.py"), "--home", str(home), "topic", name, "--json"])
    return {
        "topic": name,
        "entry_bytes": sum(entry.values()),
        "owner_start": owner_start(topic, runs),
        "entry_files": entry,
        "top_level": sorted(p.name + ("/" if p.is_dir() else "") for p in top),
        "markdown_outside_runs": {"files": len(files), "bytes": sum(p.stat().st_size for p in files)},
        "work_markdown": {"files": len(work := [p for p in files if p.relative_to(topic).parts[0] == "work"]),
                          "bytes": sum(p.stat().st_size for p in work)},
        "oversize_pages": [f"{p.relative_to(topic)} {p.stat().st_size}" for p in files if p.stat().st_size > OVERSIZE],
        "broken_links": broken_links(topic, files),
        "runs": len(runs),
        "runs_without_guide": sorted(r.name for r in runs
                                     if not any((r / g).is_file() for g in ("README.md", "CHECKPOINT.md"))),
        "optional_parts": sorted(p for p in ("INBOX.md", "TOOLS.md", "work", "knowledge", "feedback", "skills", "programs")
                                 if (topic / p).exists()),
        "mission": ({"mode": rep.get("mode"), "workstreams": rep.get("workstreams"), "errors": rep.get("errors"),
                     "warnings": rep.get("warnings"),
                     "status": "errors" if rep.get("errors") else "warnings" if rep.get("warnings") else "ok"}
                    if isinstance(rep, dict) else "unknown"),
        "throughput": ({k: hist.get(k) for k in ("window_h", "mission_status", "landed_all", "last_progress",
                                                   "days_since_progress", "scorecard_self_reported")}
                       if isinstance(hist, dict) else "unknown"),
    }


def work_bytes(d: dict) -> int | None:
    work = (d.get("work_markdown") or {}).get("bytes")
    return 0 if work is None and "work" not in d.get("optional_parts", ["work"]) else work


def outside_work(d: dict) -> int | None:
    """Markdown outside runs, history and work/: what an audit must leave no larger. A snapshot
    taken before `work_markdown` was measured counts work/ as empty only when it had no work/."""
    md, work = (d.get("markdown_outside_runs") or {}).get("bytes"), work_bytes(d)
    return None if md is None or work is None else md - work


def smaller_overall(before: dict, after: dict) -> str:
    """The audit's bar: the owner start read and the Markdown outside runs, history and work/ end
    no larger, and work/ (the plan and its briefs, which replace reading) grows by no more than the
    owner start read shrank, so growth cannot just move into work/."""
    grew, unknown = [], []
    start = [(d.get("owner_start") or {}).get("bytes") for d in (before, after)]
    for label, b, a in (("owner start read", *start),
                        ("Markdown outside runs/, history/ and work/", outside_work(before), outside_work(after))):
        if a is None or b is None:
            unknown.append(label)
        elif a > b:
            grew.append(f"{label} +{a - b} B")
    wb, wa = work_bytes(before), work_bytes(after)
    if None in (wb, wa, *start):
        unknown.append("work/")
    elif wa - wb > max(0, start[0] - start[1]):
        grew.append(f"work/ +{wa - wb} B, more than the start-read cut of {max(0, start[0] - start[1])} B")
    return f"no: {'; '.join(grew)}" if grew else f"not measured: {', '.join(unknown)}" if unknown else "yes"


def compare(before: dict, after: dict) -> str:
    def n(d: dict, *path):
        for k in path:
            d = d.get(k) if isinstance(d, dict) else None
        return d

    rows = [
        ("Entry bytes (README+MISSION+CURRENT)", n(before, "entry_bytes"), n(after, "entry_bytes")),
        ("Owner start read (MISSION + next brief, else newest checkpoint)", n(before, "owner_start", "bytes"), n(after, "owner_start", "bytes")),
        ("Top-level entries", len(before["top_level"]), len(after["top_level"])),
        ("Markdown outside runs, files", n(before, "markdown_outside_runs", "files"), n(after, "markdown_outside_runs", "files")),
        ("Markdown outside runs, bytes", n(before, "markdown_outside_runs", "bytes"), n(after, "markdown_outside_runs", "bytes")),
        ("Markdown outside runs/, history/ and work/, bytes", outside_work(before), outside_work(after)),
        ("work/ Markdown (plan and briefs), bytes", work_bytes(before), work_bytes(after)),
        ("Pages over 40 KB", len(before["oversize_pages"]), len(after["oversize_pages"])),
        ("Broken links", len(before["broken_links"]), len(after["broken_links"])),
        ("Runs without a guide", len(before["runs_without_guide"]), len(after["runs_without_guide"])),
        ("Landed workstreams", len(n(before, "throughput", "landed_all") or []), len(n(after, "throughput", "landed_all") or [])),
    ]
    lines = [f"# {after['topic']}: before → after", "", "| Measure | Before | After | Change |", "|---|---|---|---|"]
    for label, b, a in rows:
        delta = f"{(a - b) / b:+.0%}" if isinstance(a, int) and isinstance(b, int) and b else "—"
        lines.append(f"| {label} | {'not measured' if b is None else b} | {'not measured' if a is None else a} | {delta} |")
    lines += ["", f"Mission check: {n(before, 'mission', 'status')} → {n(after, 'mission', 'status')}",
              f"Smaller overall: {smaller_overall(before, after)}"]
    return "\n".join(lines) + "\n"


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    m = sub.add_parser("measure")
    m.add_argument("topic")
    m.add_argument("--home", type=Path, default=Path.home() / ".vrooli" / "research")
    m.add_argument("--out", type=Path)
    c = sub.add_parser("compare")
    c.add_argument("before", type=Path)
    c.add_argument("after", type=Path)
    a = ap.parse_args(argv)
    if a.cmd == "measure":
        text = json.dumps(measure(a.home, a.topic), indent=1) + "\n"
        if a.out:
            a.out.write_text(text, encoding="utf-8")
        sys.stdout.write(text)
    else:
        sys.stdout.write(compare(json.loads(a.before.read_text()), json.loads(a.after.read_text())))
    return 0


if __name__ == "__main__":
    sys.exit(main())
