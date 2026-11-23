#!/usr/bin/env python3
"""Validate research topic missions and render the portfolio.

Usage:
  mission_check.py <research-home> [--topic NAME] [--json] [--markdown] [--write-readme]

Checks every real topic folder (symlinked aliases are skipped) for a MISSION.md that
follows references/mission.md, and its README for the front-page shape. Prints the
portfolio table. Exit 0 when no errors, 1 when any topic has errors, 2 on bad usage.
An error is a defect that would make a window fail (no Mode, a missing section, nothing to staff) and
refuses a launch; everything else, ambition and honesty checks and table lint included, is a warning that
names what to fix (R1). --write-readme replaces the block between the portfolio markers in
<research-home>/README.md with the rendered table (movement columns, not lint status).
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

MODES = ("build", "propose-then-build", "gated-effect", "artifact", "evidence")
STATUSES = ("open", "active", "waiting-operator", "waiting-time", "landed", "proposed", "approved", "declined")
WAITING = ("waiting-operator", "waiting-time")  # open work that only the operator or elapsed time can move
LIVE = ("open", "active", "approved")  # work an owner can take now
REQUIRED_SECTIONS = (
    "Operator intent (verbatim)",
    "End state",
    "Scorecard",
    "Workstreams",
    "Approval flow",
    "Real boundaries",
    "Exemplar",
    "Not this",
)
# Self-shrinking qualifiers that turned build missions into evidence missions.
SHRINKING = (
    "decision-ready",
    "evidence for a decision",
    "narrower recommendation",
    "narrower scope",
    "foundation for later",
    "if justified",
)
MIN_WORKSTREAM_HOURS = 20
# A `## Later` line: an unbriefed workstream kept in order beneath the table, one line each.
LATER_LINE = re.compile(r"^- \*\*([A-Za-z0-9][A-Za-z0-9_.-]*)\*\* — (.+) · ([^·]+)$")
LIMITS = Path(__file__).resolve().parent.parent / "templates" / "limits.json"
RECEIPT_HEADING = re.compile(
    r"^#{2,3} .*(cohort update|hourly|delta screen|probe|datapoint|\b20\d\d-\d\d-\d\d\b)",
    re.IGNORECASE,
)


@dataclass
class Workstream:
    ident: str
    destination: str
    magnitude: str
    showcase: str
    status: str
    hours_low: int | None
    moved: str = ""   # the date in the status cell: when the status was set
    resume: str = ""  # `resume <date>` in a waiting-time cell


@dataclass
class TopicReport:
    topic: str
    mode: str = ""
    architect_pass: str = ""
    ambition_review: str = ""
    retired: str = ""
    workstreams: list[Workstream] = field(default_factory=list)
    later: list[Workstream] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)

    def count(self, status: str) -> int:
        return sum(1 for w in self.workstreams if w.status == status)

    def landed_since(self, days: int, today: dt.date) -> int:
        """Workstreams whose status cell says `landed <date>` within `days` days: the README's Landed (10 d) column."""
        since = (today - dt.timedelta(days=days)).isoformat()
        return sum(1 for w in self.workstreams if w.status == "landed" and w.moved >= since)

    def last_landed(self) -> str:
        """The newest `landed <date>` in the status cells, or ""."""
        return max((w.moved for w in self.workstreams if w.status == "landed" and w.moved), default="")

    def open_hours(self) -> int:
        if self.retired:
            return 0
        return sum(w.hours_low or 0 for w in self.workstreams + self.later if w.status in LIVE + ("later",))

    def as_dict(self, today: dt.date | None = None) -> dict:
        return {
            "topic": self.topic,
            "mode": self.mode,
            "architect_pass": self.architect_pass,
            "ambition_review": self.ambition_review,
            "retired": self.retired,
            "landed_10d": self.landed_since(10, today or dt.date.today()),
            "last_landed_at": self.last_landed(),
            "workstreams": {s: self.count(s) for s in STATUSES} | {"later": len(self.later)},
            "workstream_rows": [
                {"id": w.ident, "destination": w.destination, "magnitude": w.magnitude,
                 "showcase": w.showcase, "status": w.status}
                for w in self.workstreams
            ],
            "later": [{"id": w.ident, "outcome": w.destination, "magnitude": w.magnitude} for w in self.later],
            "open_hours_low": self.open_hours(),
            "errors": self.errors,
            "warnings": self.warnings,
        }


def split_sections(text: str) -> dict[str, str]:
    sections: dict[str, str] = {}
    current = None
    lines: list[str] = []
    for line in text.splitlines():
        if line.startswith("## "):
            if current is not None:
                sections[current] = "\n".join(lines).strip()
            current = line[3:].strip()
            lines = []
        elif current is not None:
            lines.append(line)
    if current is not None:
        sections[current] = "\n".join(lines).strip()
    return sections


def table_rows(body: str) -> list[list[str]]:
    rows = []
    for line in body.splitlines():
        line = line.strip()
        if not line.startswith("|"):
            continue
        cells = [c.strip() for c in line.strip("|").split("|")]
        if all(re.fullmatch(r":?-{3,}:?", c) for c in cells if c):
            continue
        rows.append(cells)
    return rows[1:] if rows else []  # drop the header row


def headless_tables(body: str) -> list[str]:
    """First cell of each run of table lines that lacks a header and separator row. Markdown
    renders such a run as plain text: a blank line inside a table splits it this way."""
    found, run = [], []
    for line in body.splitlines() + [""]:
        if line.strip().startswith("|"):
            run.append(line.strip())
            continue
        if run:
            sep = [c.strip() for c in run[1].strip("|").split("|")] if len(run) > 1 else []
            if not sep or not all(re.fullmatch(r":?-{3,}:?", c) for c in sep if c):
                found.append(run[0].strip("|").split("|")[0].strip())
            run = []
    return found


DATE = re.compile(r"\b(20\d\d-\d\d-\d\d)\b")
RESUME = re.compile(r"\bresume\s+(20\d\d-\d\d-\d\d)\b")


def workstream(cells: list[str]) -> Workstream:
    """One table row: ID, Destination, Dimensions, Magnitude, Showcase, Status (`<word> [resume <date>] <date>`)."""
    ident, dest, _dims, magnitude, showcase, cell = cells[:6]
    resume = RESUME.search(cell)
    dates = DATE.findall(RESUME.sub("", cell))
    return Workstream(ident, dest, magnitude, showcase, cell.split()[0].lower() if cell else "", parse_hours(magnitude),
                      dates[0] if dates else "", resume[1] if resume else "")


def parse_hours(magnitude: str) -> int | None:
    m = re.search(r"(\d+)\s*(?:[–-]\s*(\d+))?\s*h\b", magnitude)
    return int(m.group(1)) if m else None


LIVE_WRITE_HOURS = 3  # a live supervisor overwrites STATE.md at every hourly review


def live_topics(home: Path) -> set[str]:
    """Topics a live commission names: its STATE.md was written within LIVE_WRITE_HOURS and its
    PACKET.json (or, for older packets, README.md) names the topic. Files only."""
    import time
    out: set[str] = set()
    for c in (home / "cohorts").glob("*/STATE.md") if (home / "cohorts").is_dir() else []:
        if time.time() - c.stat().st_mtime > LIVE_WRITE_HOURS * 3600:
            continue
        packet, readme = c.parent / "PACKET.json", c.parent / "README.md"
        try:
            topics = json.loads(packet.read_text(encoding="utf-8")).get("topics") if packet.is_file() else None
        except (OSError, ValueError, AttributeError):
            topics = None
        if topics is None and readme.is_file():
            topics = re.findall(r"topics/([a-z0-9][a-z0-9-]*)/", readme.read_text(encoding="utf-8", errors="replace"))
        out.update(t for t in topics or [] if isinstance(t, str))
    return out


def check_mission(topic: str, path: Path) -> TopicReport:
    rep = TopicReport(topic)
    text = path.read_text(encoding="utf-8")
    m = re.search(r"^Mode:\s*(\S+)", text, re.MULTILINE)
    rep.mode = m.group(1) if m else ""
    if rep.mode not in MODES:
        rep.errors.append(f"Mode line missing or not one of {', '.join(MODES)}")
    m = re.search(r"Last architect pass:\s*([0-9-]+)", text)
    rep.architect_pass = m.group(1) if m else ""
    if not rep.architect_pass:
        rep.warnings.append("missing 'Last architect pass: <date>'")
    m = re.search(r"Last ambition review:\s*([^\n]+)", text)
    rep.ambition_review = m.group(1).strip() if m else ""
    if not rep.ambition_review:
        rep.warnings.append("missing 'Last ambition review:'")
    elif rep.ambition_review.lower().startswith("pending"):
        rep.warnings.append("ambition review pending: run the ambition brief in templates/review-briefs.md before staffing")

    m = re.search(r"^Retired:\s*(\S.*)$", text, re.MULTILINE)
    if m:  # a finished topic keeps its folder and links; it is reported, never commissioned
        rep.retired = m.group(1).strip()
        rep.warnings.append(f"retired {rep.retired}; not commissionable (remove the Retired line to reopen it)")
        rep.workstreams = [workstream(c) for c in table_rows(split_sections(text).get("Workstreams", "")) if len(c) >= 6]
        return rep

    sections = split_sections(text)
    for name in REQUIRED_SECTIONS:
        if not sections.get(name):
            rep.errors.append(f"section '## {name}' missing or empty")

    intent = sections.get("Operator intent (verbatim)", "")
    if intent and not re.search(r'^>\s*["“]', intent, re.MULTILINE):
        rep.warnings.append("Operator intent has no quoted operator words ('> \"...\"')")

    end_state = sections.get("End state", "").lower()
    if rep.mode != "evidence":
        for phrase in SHRINKING:
            if phrase in end_state:
                rep.warnings.append(f"End state contains self-shrinking qualifier '{phrase}'")

    for name in ("Scorecard", "Workstreams"):
        for first in headless_tables(sections.get(name, "")):
            rep.warnings.append(f"{name} table is split: rows from '{first}' have no header, so they render as "
                                "text outside it; remove the blank line above them")

    scorecard = table_rows(sections.get("Scorecard", ""))
    if scorecard and not 5 <= len(scorecard) <= 8:
        rep.warnings.append(f"Scorecard has {len(scorecard)} dimensions; mission.md asks for 5-8")
    if not scorecard and sections.get("Scorecard"):
        rep.warnings.append("Scorecard has no table rows")

    for cells in table_rows(sections.get("Workstreams", "")):
        if len(cells) < 6:
            rep.warnings.append(f"workstream row has {len(cells)} cells, expected 6: {cells[:1]}")
            continue
        ws = workstream(cells)
        rep.workstreams.append(ws)
        if ws.status not in STATUSES:
            rep.warnings.append(f"{ws.ident}: status '{cells[5]}' not one of {', '.join(STATUSES)}")
        if not ws.showcase:
            rep.warnings.append(f"{ws.ident}: no showcase")
        if ws.hours_low is None:
            rep.warnings.append(f"{ws.ident}: magnitude '{ws.magnitude}' names no agent-hours")
        elif ws.hours_low < MIN_WORKSTREAM_HOURS and "per " not in ws.magnitude:
            rep.warnings.append(f"{ws.ident}: {ws.hours_low} h is a task, not a workstream; fold it into a larger row")
        if ws.status == "waiting-time" and not ws.resume:
            rep.warnings.append(f"{ws.ident}: waiting-time has no `resume <date>`; an undated wait cannot expire")
    for line in sections.get("Later", "").splitlines():
        if not line.startswith("- "):
            continue
        m = LATER_LINE.match(line.strip())
        if not m:
            rep.warnings.append(f"Later line is not `- **ID** — outcome · N–M h`: {line.strip()[:60]}")
            continue
        rep.later.append(Workstream(m[1], m[2].strip(), m[3].strip(), "", "later", parse_hours(m[3])))
        if parse_hours(m[3]) is None:
            rep.warnings.append(f"{m[1]}: Later magnitude '{m[3].strip()}' names no agent-hours")
    taken = [w.ident for w in rep.later if w.ident in {x.ident for x in rep.workstreams}]
    if taken:
        rep.warnings.append(f"{', '.join(taken)}: both a table row and a Later line; keep one")
    active = [w.ident for w in rep.workstreams if w.status == "active"]
    if active and topic not in live_topics(path.parent.parent.parent):
        rep.warnings.append(f"{', '.join(active)}: `active` but no live commission names this topic; "
                            "say `open`, or `waiting-…` with what it waits on")
    live = sum(1 for w in rep.workstreams if w.status in LIVE)
    if live < 1:
        rep.errors.append("no open/active/approved workstream: a window has nothing to staff (run an architect pass)")
    elif live < 3:
        rep.warnings.append(f"only {live} staffable workstreams; keep at least two, and add or promote a third when possible")

    boundaries = [l for l in sections.get("Real boundaries", "").splitlines() if l.strip().startswith("- ")]
    if len(boundaries) > 6:
        rep.warnings.append(f"{len(boundaries)} real boundaries; six or fewer, human-owned limits only")
    traps = [l for l in sections.get("Not this", "").splitlines() if l.strip().startswith("- ")]
    if sections.get("Not this") and len(traps) < 2:
        rep.warnings.append("Not this names fewer than two traps")
    size, ceiling = len(text.encode("utf-8")), mission_ceiling()
    if ceiling and size > ceiling:
        rep.warnings.append(f"MISSION.md is {size:,} bytes, over the {ceiling:,} an owner should read at start; "
                            "the next architect pass tidies it: receipts and evidence go to the topic LEDGER.md, operator words stay")
    return rep


def mission_ceiling() -> int | None:
    """templates/limits.json ceilings.mission_bytes; None when the file is absent (the script runs alone)."""
    try:
        return json.loads(LIMITS.read_text(encoding="utf-8"))["ceilings"]["mission_bytes"]
    except (OSError, ValueError, KeyError):
        return None


def check_readme(rep: TopicReport, readme: Path) -> None:
    if not readme.exists():
        rep.warnings.append("README.md missing")
        return
    text = readme.read_text(encoding="utf-8")
    if "MISSION.md" not in text[:1500]:
        rep.warnings.append("README does not link MISSION.md near the top")
    receipts = [l for l in text.splitlines() if RECEIPT_HEADING.match(l)]
    if len(receipts) > 2:
        rep.warnings.append(f"README carries {len(receipts)} per-run receipt headings; move them to LEDGER.md")


def topics(home: Path, only: str | None) -> list[Path]:
    root = home / "topics"
    if not root.is_dir():
        raise SystemExit(f"no topics/ under {home}")
    out = []
    for p in sorted(root.iterdir()):
        if p.is_symlink() or not p.is_dir():
            continue
        if only and p.name != only:
            continue
        out.append(p)
    if only and not out:
        raise SystemExit(f"topic {only!r} not found (aliases are skipped)")
    return out


def waits_expired(r: TopicReport, today: dt.date) -> int:
    """waiting-time rows whose resume date has passed, or that name none (an undated wait never signals)."""
    return sum(1 for w in r.workstreams if w.status == "waiting-time" and (not w.resume or w.resume <= today.isoformat()))


def render_markdown(reports: list[TopicReport], today: dt.date | None = None) -> str:
    """The portfolio table: what moved, not whether the file is well formed (check output carries that)."""
    today = today or dt.date.today()
    lines = [
        "| Topic | Mode | Open | Active | Waiting | Landed (10 d) | Days since a row moved | Waits expired |",
        "|---|---|---|---|---|---|---|---|",
    ]
    for r in reports:
        landed = r.landed_since(10, today)
        moved = [w.moved for w in r.workstreams if w.moved]
        age = str((today - dt.date.fromisoformat(max(moved))).days) if moved else "—"
        label = f"{r.topic} (retired)" if r.retired else r.topic
        open_, active, waiting = (0, 0, 0) if r.retired else (r.count("open"), r.count("active"), sum(r.count(w) for w in WAITING))
        lines.append(
            f"| [{label}](topics/{r.topic}/MISSION.md) | {r.mode or '—'} | {open_} | {active} | {waiting} "
            f"| {landed} | {age} | {0 if r.retired else waits_expired(r, today)} |"
        )
    return "\n".join(lines)


PORTFOLIO_START = "<!-- portfolio:start (generated by mission_check.py --write-readme) -->"
PORTFOLIO_END = "<!-- portfolio:end -->"


def write_readme(home: Path, table: str) -> None:
    readme = home / "README.md"
    text = readme.read_text(encoding="utf-8")
    if PORTFOLIO_START not in text or PORTFOLIO_END not in text:
        raise SystemExit(f"{readme} has no portfolio markers; add {PORTFOLIO_START} and {PORTFOLIO_END}")
    head, rest = text.split(PORTFOLIO_START, 1)
    _, tail = rest.split(PORTFOLIO_END, 1)
    readme.write_text(f"{head}{PORTFOLIO_START}\n{table}\n{PORTFOLIO_END}{tail}", encoding="utf-8")


SOURCED_SINCE = "2026-11-15T12"  # hand verdicts written from here on carry a Source: line (inbox.py verdict)


def check_verdicts(home: Path, reports: list["TopicReport"]) -> None:
    """Warn on hand verdicts whose words may not be the operator's about that topic.

    A later `provenance correction` entry voids the verdicts above it. Of the rest, warn when the
    same words were filed as verdicts on different topics (a stance pasted per topic), or when a
    verdict written since SOURCED_SINCE has no `Source:` line.
    """
    found = []
    for path in sorted(home.glob("reviews/*/VERDICTS.md")) + sorted(home.glob("topics/*/history/VERDICTS*.md")):
        live = []
        for block in re.split(r"(?m)^(?=### )", path.read_text(encoding="utf-8", errors="replace"))[1:]:
            head, _, body = block.partition("\n")
            if "provenance correction" in head:
                live = []
                continue
            m = re.match(r"### (\S+) · verdict ([a-z0-9, -]+?)\s*(?:[A-Za-z0-9_-]*[A-Z][A-Za-z0-9_-]*)?\s*· via hand", head)
            if m:
                words = " ".join(l for l in body.split("\n") if not l.startswith("Source:")).split()
                live.append({"at": m.group(1), "topics": [t for t in re.split(r"[,\s]+", m.group(2)) if t],
                             "key": " ".join(words[:12]).strip("\"“”").lower(), "sourced": "\nSource:" in "\n" + body,
                             "file": path.relative_to(home)})
        found += live
    spread: dict[str, set] = {}  # words -> the topic sets of the separate entries carrying them
    for v in found:
        spread.setdefault(v["key"], set()).add(tuple(v["topics"]))
    by_topic = {r.topic: r for r in reports}
    for v in found:
        for t in v["topics"]:
            r = by_topic.get(t)
            if not r:
                continue
            if v["key"] and len(spread[v["key"]]) > 1:
                r.warnings.append(f"verdict {v['at']} in {v['file']}: the same words were filed as {len(spread[v['key']])} "
                                  "separate topic verdicts; check that the operator said them about this one (research-review)")
            elif not v["sourced"] and v["at"] >= SOURCED_SINCE:
                r.warnings.append(f"verdict {v['at']} in {v['file']} has no Source: line; it may not be the operator's words")


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("home", type=Path)
    ap.add_argument("--topic")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--markdown", action="store_true", help="print only the portfolio table")
    ap.add_argument("--write-readme", action="store_true", help="regenerate the portfolio block in README.md")
    args = ap.parse_args(argv)

    reports = []
    for tdir in topics(args.home, args.topic):
        mission = tdir / "MISSION.md"
        if mission.exists():
            rep = check_mission(tdir.name, mission)
        else:
            rep = TopicReport(tdir.name, errors=["MISSION.md missing (references/mission.md)"])
        check_readme(rep, tdir / "README.md")
        reports.append(rep)
    check_verdicts(args.home, reports)

    if args.write_readme:
        if args.topic:
            raise SystemExit("--write-readme renders the whole portfolio; omit --topic")
        write_readme(args.home, render_markdown(reports))
    if args.json:
        print(json.dumps([r.as_dict() for r in reports], indent=2))
    elif args.markdown:
        print(render_markdown(reports))
    else:
        print(render_markdown(reports))
        for r in reports:
            for e in r.errors:
                print(f"ERROR {r.topic}: {e}")
            for w in r.warnings:
                print(f"warn  {r.topic}: {w}")
    return 1 if any(r.errors for r in reports) else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
