#!/usr/bin/env python3
"""Check and write needs-you items (research-desk FILE-CONTRACT §5), and list every open decision.

  needs_you.py validate --file DRAFT --section "Needs you"
  needs_you.py write    --file DRAFT --section "Needs you" --target topics/<t>/DECISIONS.md
  needs_you.py open [--json] [--research DIR] [--portfolio DIR]

`open` answers "what decisions are open?" across every level (docs/agent-system/LEVELS.md) from
files alone, with every service down: the CEO's open asks, the supervisor's ATTENTION.md, each
topic's DECISIONS.md, and items still written in a live commission's STATE.md. An item answered by
an `answer` entry naming it in its level's INBOX.md or history/INBOX.md, or by an OPERATOR.md
receipt, is not open. A delivered ask's Hub state is the desk's to read: `research-desk needs-you`.

needs_you_cases.json beside this script is the grammar as cases; this script's tests and
research-desk's Go parser both run it, so they agree. Works with no service running.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import sys
import tempfile
from pathlib import Path

HEADING = re.compile(r"### ([A-Z0-9][A-Z0-9_-]{0,127}) — (.+?)\s*")
OPTION = re.compile(r"([a-z0-9-]{1,32}) — (.+)")
DEFAULT_KEYS = ["accept", "hold"]
BRIEF = "BRIEF.md"


def parse(text: str, section: str, full: bool = False) -> tuple[list[dict], list[str]]:
    """The items of `## <section>` and the problems that keep the draft from being written.

    With full, each item also carries its fields, its own problems and its body."""
    items: list[dict] = []
    problems: list[str] = []
    item: dict | None = None
    active = False

    def problem(text: str) -> None:
        problems.append(text)
        if item is not None:
            item["_problems"].append(text)

    def finish(item: dict | None) -> None:
        if item is None:
            return

        def flag(text: str) -> None:
            problems.append(text)
            item["_problems"].append(text)
        decision = item.pop("_decision")
        if item["recommended"] is not None:
            key, number = item.pop("recommended"), item.pop("line")
            choices = DEFAULT_KEYS if item["options"] is None else item["options"]
            item["recommended"] = key if key in choices else None
            if key not in choices:
                flag(f"line {number} ({item['id']}): Recommended key {key!r} is not offered. Use one of: "
                     f"{', '.join(choices) or '(no valid options)'}. Fix the Recommended key or add it to Options:.")
        if decision and not item["packet"]:
            flag(f"{item['id']}: a decision needs a Packet: line naming its brief, work/<ID>/{BRIEF}, "
                 "so the answer can be executed without its raiser.")
        elif decision and Path(item["packet"]).name != BRIEF:
            flag(f"{item['id']}: Packet: must name the decision's brief, work/<ID>/{BRIEF}, not {Path(item['packet']).name}.")

    for number, raw in enumerate(text.replace("\r\n", "\n").split("\n"), start=1):
        line = raw.strip()
        if line.startswith("## "):
            finish(item)
            item, active = None, line[3:].strip() == section
            continue
        if not active:
            continue
        if line.startswith("### "):
            finish(item)
            match = HEADING.fullmatch(line)
            item = {"id": match[1], "title": match[2], "options": None, "recommended": None, "explainer": None, "packet": None,
                    "_decision": False, "_problems": [], "_fields": {}} if match else None
            if item:
                items.append(item)
            else:
                problem(f"line {number}: malformed item heading; use ### <ID> — <title>.")
            continue
        name, separator, value = line.partition(":")
        if item is None or not separator:
            continue
        value, where = value.strip(), f"line {number} ({item['id']})"
        item["_fields"].setdefault(name, value)
        if name == "Options":
            if item["options"] is not None:
                problem(f"{where}: duplicate Options: line; keep one options list.")
                continue
            item["options"] = []
            for option in value.split("; ") if value else []:
                match = OPTION.fullmatch(option.strip())
                if not match:
                    problem(f"{where}: malformed option {option!r}; use <key> — <label>.")
                    continue
                if match[1] in item["options"]:
                    problem(f"{where}: duplicate option key {match[1]!r}.")
                item["options"].append(match[1])
        elif name == "Explainer":
            item["explainer"] = value
            if not value.lower().endswith((".html", ".htm")):
                problem(f"{where}: Explainer: must name an .html page, a path inside the research home.")
        elif name == "Packet":
            item["packet"] = value
        elif name == "Recommended":
            item["_decision"] = True
            match = OPTION.fullmatch(value)
            item["recommended"], item["line"] = (match[1], number) if match else (None, None)
            if not match:
                problem(f"{where}: malformed Recommended line; use Recommended: <key> — <label>.")
    finish(item)
    for item in items:
        item.pop("line", None)
        extras = {k: item.pop(k) for k in ("_problems", "_fields")}
        if full:
            item["problems"], item["fields"] = extras["_problems"], extras["_fields"]
    return items, problems


def validate(text: str, section: str) -> list[str]:
    return parse(text, section)[1]


def write_atomically(target: Path, content: str) -> None:
    target.parent.mkdir(parents=True, exist_ok=True)
    mode = target.stat().st_mode & 0o777 if target.exists() else 0o644
    fd, temporary = tempfile.mkstemp(prefix=f".{target.name}.", suffix=".tmp", dir=target.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8", newline="") as stream:
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temporary, mode)
        os.replace(temporary, target)
    except BaseException:
        Path(temporary).unlink(missing_ok=True)
        raise


# --- open: every open decision across levels, from files ---------------------------------------

def read(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return ""


def names(body: str, item_id: str) -> bool:
    """Whether an answer entry is about the item: a line `<ID>: <key> — <label>`, or an older
    desk entry's "item <ID> of". research-desk's Go reader uses the same rule."""
    return re.search(rf"(?m)^{re.escape(item_id)}:|\bitem {re.escape(item_id)}(?![A-Za-z0-9_-])", body) is not None


def inbox_answers(folder: Path) -> list[str]:
    """The bodies of the `answer` entries in a level's inbox and its drained history."""
    bodies = []
    for path in (folder / "INBOX.md", folder / "history" / "INBOX.md"):
        for raw in re.split(r"(?m)^(?=### )", read(path))[1:]:
            head, _, body = raw.partition("\n")
            if re.match(r"^### \S+ · from \S+ · answer\s*$", head):
                bodies.append(body)
    return bodies


def utc(text: str) -> dt.datetime | None:
    text = text.strip().replace(" UTC", "Z").replace(" ", "T")
    for layout in ("%Y-%m-%dT%H:%M:%SZ", "%Y-%m-%dT%H:%MZ", "%Y-%m-%dT%H:%M:%S.%fZ"):
        try:
            return dt.datetime.strptime(text[:26 if "." in text else 20], layout).replace(tzinfo=dt.timezone.utc)
        except ValueError:
            continue
    try:
        return dt.datetime.fromisoformat(text.replace("Z", "+00:00")).astimezone(dt.timezone.utc)
    except ValueError:
        return None


def jsonfile(path: Path) -> dict:
    try:
        value = json.loads(read(path) or "{}")
    except json.JSONDecodeError:
        return {}
    return value if isinstance(value, dict) else {}


class Commission:
    def __init__(self, folder: Path, research: Path):
        self.id, self.folder = folder.name, folder
        self.state = read(folder / "STATE.md")
        launch, packet = jsonfile(folder / "LAUNCH.json"), jsonfile(folder / "PACKET.json")
        spec = packet.get("spec") if isinstance(packet.get("spec"), dict) else {}
        self.topics = set(spec.get("topics") or [])
        self.topics |= {p.parts[-3] for p in research.glob(f"topics/*/runs/{self.id}*")}
        self.answers = set(re.findall(r"(?m)^### \S+ · answer ([A-Z0-9][A-Z0-9_-]*) · via ", read(folder / "OPERATOR.md")))
        self.end, self.ended = None, bool(launch.get("discarded"))
        match = re.search(r"Allowance ends:\s*([0-9][0-9T:. -]+(?:Z|UTC)?)", self.state)
        if match:
            self.end = utc(match[1])
        if self.end is None and spec.get("allowance_hours"):
            sent = [s.get("at") for s in launch.get("steps") or [] if isinstance(s, dict) and s.get("step") == "goal_sent"]
            start = utc(sent[-1]) if sent and sent[-1] else None
            self.end = start + dt.timedelta(hours=spec["allowance_hours"]) if start else None

    def live(self, now: dt.datetime) -> bool | None:
        """True or False by the files; None when they do not say."""
        if self.ended:
            return False
        return None if self.end is None else self.end > now


def entry(item: dict, source: Path, **extra) -> dict:
    fields = item["fields"]
    base = source.parent

    def path(value: str | None) -> str | None:
        return None if not value else str(Path(value) if Path(value).is_absolute() else base / value)

    out = {"id": item["id"], "title": item["title"], "recommended": fields.get("Recommended"),
           "packet": path(item["packet"]), "page": path(item["explainer"]), "source": str(source),
           "delivered": fields.get("Delivered"), "problems": item["problems"]}
    out.update(extra)
    return out


def ceo_level(portfolio: Path) -> dict:
    path = portfolio / "steward" / "state" / "asks.jsonl"
    latest: dict[str, dict] = {}
    for line in read(path).splitlines():
        try:
            row = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(row, dict) and row.get("id"):
            latest[row["id"]] = row
    items = []
    for row in latest.values():
        if row.get("state") not in ("sent", "file-fallback", "queued") or row.get("answer"):
            continue
        delegation = portfolio / "steward" / "delegations" / row["id"]
        items.append({"id": row.get("key") or row["id"], "title": row.get("question") or row["id"],
                      "recommended": row.get("recommend"), "packet": str(delegation) if delegation.is_dir() else None,
                      "page": row.get("explainer"), "source": str(path), "delivered": row.get("sent_at"),
                      "state": row.get("state"), "ask": row["id"], "problems": []})
    return {"level": "ceo", "folder": str(portfolio), "decisions": str(path), "inbox": str(portfolio / "INBOX.md"), "items": items}


def open_decisions(research: Path, portfolio: Path, now: dt.datetime) -> dict:
    commissions = [Commission(f, research) for f in sorted((research / "cohorts").glob("*")) if f.is_dir()]
    by_id = {c.id: c for c in commissions}
    levels, seen, not_counted = [], {}, []

    def answered(item_id: str, folder: Path, related: list[Commission]) -> bool:
        return any(names(b, item_id) for b in inbox_answers(folder)) or any(item_id in c.answers for c in related)

    def add(level: dict, item: dict, key: tuple) -> None:
        if key in seen:
            seen[key].setdefault("also_in", []).append(item["source"])
            return
        seen[key] = item
        level["items"].append(item)

    levels.append(ceo_level(portfolio))
    topic_levels = {}
    for folder in sorted((research / "topics").glob("*")):
        if not folder.is_dir() or folder.is_symlink():
            continue
        related = [c for c in commissions if folder.name in c.topics]
        level = {"level": f"topic:{folder.name}", "folder": str(folder), "decisions": str(folder / "DECISIONS.md"),
                 "inbox": str(folder / "INBOX.md"), "items": []}
        for name, section, legacy in (("DECISIONS.md", "Needs you", False), ("STATE.md", "Needs you", True)):
            items, _ = parse(read(folder / name), section, full=True)
            for item in items:
                if not answered(item["id"], folder, related):
                    extra = {"legacy": "an older topic STATE.md: not delivered; move it to DECISIONS.md"} if legacy else {}
                    add(level, entry(item, folder / name, **extra), (folder.name, item["id"]))
        topic_levels[folder.name] = level
        levels.append(level)

    supervisor = {"level": "supervisor", "folder": str(research), "decisions": str(research / "ATTENTION.md"),
                  "inbox": str(research / "INBOX.md"), "items": []}
    commission_levels: dict[str, dict] = {}

    def commission_level(c: Commission) -> dict:
        return commission_levels.setdefault(c.id, {
            "level": f"commission:{c.id}", "folder": str(c.folder), "decisions": str(c.folder / "STATE.md"),
            "inbox": str(c.folder / "OPERATOR.md"), "live_until": c.end.strftime("%Y-%m-%dT%H:%MZ") if c.end else None, "items": []})

    def commission_key(c: Commission, item_id: str) -> tuple:
        """A commission's item and its topic's DECISIONS.md copy are one decision."""
        topic = next((t for t in sorted(c.topics) if (t, item_id) in seen), None)
        return (topic, item_id) if topic else (c.id, item_id)

    for c in commissions:
        items, _ = parse(c.state, "Needs you", full=True)
        if not items:
            continue
        live = c.live(now)
        if live is None:
            not_counted.append(c.id)
        if not live:
            continue
        for item in items:
            if item["id"] in c.answers or any(names(b, item["id"]) for t in c.topics for b in inbox_answers(research / "topics" / t)):
                continue
            add(commission_level(c), entry(item, c.folder / "STATE.md"), commission_key(c, item["id"]))
    for item in parse(read(research / "ATTENTION.md"), "Actionable now", full=True)[0]:
        owner = item["fields"].get("Commission")
        if owner:
            c = by_id.get(owner)
            if c is None or not c.live(now) or item["id"] in c.answers:
                continue
            add(commission_level(c), entry(item, research / "ATTENTION.md"), commission_key(c, item["id"]))
            continue
        if answered(item["id"], research, []):
            continue
        topic = next((t for (t, i) in seen if i == item["id"] and t in topic_levels), None)
        add(supervisor, entry(item, research / "ATTENTION.md"), (topic, item["id"]) if topic else ("supervisor", item["id"]))
    levels.insert(1, supervisor)
    levels += list(commission_levels.values())
    levels = [l for l in levels if l["items"] or l["level"] in ("ceo", "supervisor")]
    return {"generated_at": now.strftime("%Y-%m-%dT%H:%M:%SZ"), "open": sum(len(l["items"]) for l in levels),
            "levels": levels, "not_counted": sorted(set(not_counted))}


def print_open(report: dict) -> None:
    counts = ", ".join(f"{l['level']} {len(l['items'])}" for l in report["levels"])
    print(f"Open decisions, from files only ({report['generated_at']}): {report['open']} ({counts}).")
    for level in report["levels"]:
        print(f"\n{level['level']} · {level['decisions']}" + (f" · live until {level['live_until']}" if level.get("live_until") else ""))
        print(f"  answers land in: {level['inbox']}")
        if not level["items"]:
            print("  none open")
        for item in level["items"]:
            print(f"  {item['id']} — {item['title']}" + (f"  [in {item['source']}]" if item["source"] != level["decisions"] else ""))
            details = [f"recommended: {item['recommended']}" if item.get("recommended") else "notice (no recommendation)"]
            details += [f"{k}: {item[k]}" for k in ("state", "delivered") if item.get(k)]
            print("    " + "; ".join(details))
            for key in ("packet", "page"):
                print(f"    {key}: {item.get(key) or 'none'}")
            for note in ([item["legacy"]] if item.get("legacy") else []) + item.get("problems", []):
                print(f"    note: {note}")
            if item.get("also_in"):
                print(f"    also in: {', '.join(item['also_in'])}")
    if report["not_counted"]:
        print(f"\nNot counted: {len(report['not_counted'])} commissions hold needs-you items but their files do not say whether they "
              f"are live (no allowance end): {', '.join(report['not_counted'])}.")
    print("\nA delivered ask's Hub state is the desk's to read: `research-desk needs-you`. Its answer lands in the inbox named above.")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("action", choices=["validate", "write", "open"])
    parser.add_argument("--file", type=Path, help="Draft markdown file")
    parser.add_argument("--section", help='Section heading, for example "Needs you"')
    parser.add_argument("--target", type=Path, help="Destination file for the write action")
    parser.add_argument("--research", type=Path, default=Path.home() / ".vrooli" / "research")
    parser.add_argument("--portfolio", type=Path, default=Path.home() / ".vrooli" / "portfolio")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    if args.action == "open":
        report = open_decisions(args.research.expanduser(), args.portfolio.expanduser(), dt.datetime.now(dt.timezone.utc))
        print(json.dumps(report, indent=1)) if args.json else print_open(report)
        return 0
    if args.file is None or args.section is None:
        parser.error(f"{args.action} requires --file and --section")
    if args.action == "write" and args.target is None:
        parser.error("write requires --target")
    try:
        draft = args.file.read_text(encoding="utf-8")
    except OSError as exc:
        print(f"cannot read draft {args.file}: {exc}", file=sys.stderr)
        return 2
    problems = validate(draft, args.section)
    if problems:
        print("Needs-you draft rejected:\n" + "\n".join(f"- {problem}" for problem in problems), file=sys.stderr)
        return 1
    if args.action == "validate":
        print(f"Needs-you draft valid: {args.file} (## {args.section})")
        return 0
    try:
        write_atomically(args.target, draft)
    except OSError as exc:
        print(f"cannot write validated draft to {args.target}: {exc}", file=sys.stderr)
        return 2
    print(f"Needs-you draft written: {args.target} (## {args.section})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
