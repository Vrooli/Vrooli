#!/usr/bin/env python3
"""Check and write needs-you items (research-desk FILE-CONTRACT §5), and list every open decision.

  needs_you.py validate --file DRAFT --section "Needs you" [--target topics/<t>/DECISIONS.md]
  needs_you.py write    --file DRAFT --section "Needs you" --target topics/<t>/DECISIONS.md
  needs_you.py open [--json] [--research DIR] [--portfolio DIR]
  needs_you.py spawn <ID>     the standard message that starts an answered decision's executor

A decision is refused unless its folder, work/<ID>/, holds the brief and the page it names, and the brief
says what was checked live (a dated `Checked:` line) and what each answer settles, `other` included.

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
        if not decision:
            return
        packet, page = Path(item["packet"] or ""), Path(item["explainer"] or "")
        if not item["packet"]:
            flag(f"{item['id']}: a decision needs a Packet: line naming its brief, work/{item['id']}/{BRIEF}, "
                 "so the answer can be executed without its raiser.")
        elif packet.name != BRIEF or packet.parent.name != item["id"]:
            flag(f"{item['id']}: Packet: must name the decision's brief, work/{item['id']}/{BRIEF}, in a folder named "
                 f"for the decision, never a workstream's brief; not {item['packet']}.")
        if not item["explainer"]:
            flag(f"{item['id']}: a decision needs an Explainer: line naming its page beside its brief, "
                 f"work/{item['id']}/page.html.")
        elif item["packet"] and page.parent != packet.parent:
            flag(f"{item['id']}: Explainer: must sit beside the brief, in {packet.parent}/; not {item['explainer']}.")

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


CHECKED = re.compile(r"(?m)^Checked: (\d{4}-\d{2}-\d{2})\S*((?:\s+\S+){3,})")
# research-desk's courier (internal/courier/pages.go keyedBullet) reads these bullets as each answer's effect.
ANSWER_BULLET = re.compile(r"^[-*]\s+(?:\*\*)?`?([a-z0-9-]{1,32})`?(?:\s*\([^)]*\))?[.:]?(?:\*\*)?[.:]?\s*(?:— )?(.*)$")
FRESH_DAYS = 7


def brief_problems(text: str, keys: list[str], now: dt.datetime | None = None) -> list[str]:
    """What a fresh executor would have to re-derive from a decision's brief: the premise checked live (a dated
    `Checked:` line saying what and how, under FRESH_DAYS old when now is given) and what each answer settles,
    `other` (the operator's own words) included. needs_you_cases.json holds these as cases for the desk too."""
    problems = []
    checked = CHECKED.search(text)
    if not checked:
        problems.append("no `Checked: <YYYY-MM-DD> <the premise, verified live, and how>` line")
    elif now and (now.date() - dt.date.fromisoformat(checked[1])).days > FRESH_DAYS:
        problems.append(f"Checked: {checked[1]} is over {FRESH_DAYS} days old; check the premise live again")
    body = re.search(r"(?ms)^## On each answer[^\n]*\n(.*?)(?=^## |\Z)", text)
    said = {m[1] for line in (body[1] if body else "").splitlines() if (m := ANSWER_BULLET.match(line.strip())) and m[2].strip()}
    missing = [k for k in [*keys, "other"] if k not in said]
    if missing:
        problems.append("`## On each answer` says nothing for " + ", ".join(f"`{k}`" for k in missing)
                        + "; give each answer a bullet saying what it settles (`other` is the operator's own words)")
    return problems


def packet_problems(item: dict, base: Path, now: dt.datetime | None = None) -> list[str]:
    """A decision's brief and page on disk: both exist, and the brief passes brief_problems. Relative paths are the
    level file's, as the desk reads them."""
    if item.get("recommended") is None:
        return []
    packet = base / Path(item["packet"]).expanduser() if item.get("packet") else None
    page = base / Path(item["explainer"]).expanduser() if item.get("explainer") else None
    problems = [f"{item['id']}: {what} {path} does not exist" for what, path in (("brief", packet), ("page", page))
                if path is not None and not path.is_file()]
    if packet is not None and packet.is_file():
        keys = DEFAULT_KEYS if item.get("options") is None else item["options"]
        problems += [f"{item['id']}: {packet}: {p}" for p in brief_problems(read(packet), keys, now)]
    return problems


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


def section_spans(text: str, section: str) -> list[tuple[int, int]]:
    """Character spans of each `## <section>` block: its heading up to the next `#`/`##` heading or the end."""
    lines = text.splitlines(keepends=True)
    spans, offset, start = [], 0, None
    for line in lines:
        bare = line.rstrip()
        heading = bare.startswith("# ") or bare.startswith("## ")
        if heading and start is not None:
            spans.append((start, offset))
            start = None
        if bare == f"## {section}":
            start = offset
        offset += len(line)
    if start is not None:
        spans.append((start, offset))
    return spans


PLACEHOLDER = re.compile(r"^(No unanswered decisions|No actionable items|None open)\b", re.I)


def classify(body: list[str]) -> list[tuple[str, str]]:
    """Each line of a section body as ("item", line), ("placeholder", line), ("blank", line) or ("other", line).
    An item runs from `### ` to the next `### `, heading or top-level `- ` line (items never hold bullets), so a
    commission's status bullets after its last item are never taken for that item."""
    out, in_item = [], False
    for line in body:
        bare = line.rstrip("\n")
        if bare.startswith("### "):
            in_item = True
        elif bare.startswith("- "):
            in_item = False
        if in_item:
            out.append(("item", line))
        elif not bare.strip():
            out.append(("blank", line))
        else:
            out.append(("placeholder" if PLACEHOLDER.match(bare) else "other", line))
    return out


def line_key(line: str) -> str:
    """A status line `- Key: value` is matched by its key, so a draft may update its value."""
    bare = line.strip()
    m = re.match(r"^- ([^:]{1,60}):", bare)
    return m.group(1).strip().lower() if m else bare


def merge_section(target_text: str, draft: str, section: str) -> tuple[str, str | None]:
    """The target with only the decisions in its `## <section>` replaced; every other byte is kept.

    The section's other lines (a commission's status lines, notes) are kept in place, in order, when the draft
    holds only items. A draft that also holds other lines owns the whole section, and is refused if it would drop
    a line the target has. A target with no content takes the whole draft (a new DECISIONS.md); a target
    without the section gets it appended; a target holding the section twice is refused, never guessed at."""
    drafted = section_spans(draft, section)
    if len(drafted) != 1:
        return target_text, f"the draft must hold exactly one `## {section}` section (it holds {len(drafted)})"
    block = draft[drafted[0][0]:drafted[0][1]]
    if not target_text.strip():
        return draft, None
    spans = section_spans(target_text, section)
    if len(spans) > 1:
        return target_text, f"the target holds `## {section}` {len(spans)} times; merge them by hand first, nothing was written"
    if not block.endswith("\n"):
        block += "\n"
    if not spans:
        return target_text.rstrip("\n") + "\n\n" + block, None
    start, end = spans[0]
    old = target_text[start:end].splitlines(keepends=True)
    new = block.splitlines(keepends=True)
    old_parts, new_parts = classify(old[1:]), classify(new[1:])
    kept = [line for kind, line in old_parts if kind == "other"]
    if any(kind == "other" for kind, _ in new_parts):
        have = {line_key(line) for kind, line in new_parts if kind == "other"}
        dropped = [line.strip() for line in kept if line_key(line) not in have]
        if dropped:
            return target_text, ("the draft would drop these lines of the target's section; keep them in the draft, or "
                                 "send only the items: " + " | ".join(dropped))
        body = new[1:]
    else:
        decisions = [line for kind, line in new_parts if kind in ("item", "placeholder", "blank")]
        while decisions and not decisions[0].strip():
            decisions.pop(0)
        while decisions and not decisions[-1].strip():
            decisions.pop()
        body, placed = [], False
        for kind, line in old_parts:
            if kind in ("item", "placeholder"):
                if not placed:
                    body += decisions + ["\n"]
                    placed = True
                continue
            body.append(line)
        if not placed:
            body = ["\n"] + decisions + ["\n"] + body
    text = old[0] + "".join(body)
    text = re.sub(r"\n{3,}", "\n\n", text)
    rest = target_text[end:]
    if rest and not text.endswith("\n\n"):
        text = text.rstrip("\n") + "\n\n"
    elif not rest:
        text = text.rstrip("\n") + "\n"
    return target_text[:start] + text + rest, None


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


def waiting_answers(folder: Path, readers: list[str], now: dt.datetime) -> list[dict]:
    """Answers still in a level's INBOX.md (not yet drained): each needs its executor started. An answered item
    leaves the open list, so without this an answer waiting for nobody was invisible (two sat 11 h in
    data-backup-manager-recovery on 2026-11-15 with no commission on the topic)."""
    out = []
    for raw in re.split(r"(?m)^(?=### )", read(folder / "INBOX.md"))[1:]:
        head, _, body = raw.partition("\n")
        m = re.match(r"^### (\S+) · from \S+ · answer\s*$", head)
        if not m:
            continue
        ident = re.search(r"(?m)^([A-Z0-9][A-Z0-9_-]{0,127}): ([a-z0-9-]{1,32})\b", body) or re.search(r"item ([A-Z0-9][A-Z0-9_-]{0,127}) of", body)
        key = ident.group(2) if ident and ident.re.groups == 2 else (re.search(r"(?m)^([a-z0-9-]{1,32}) — ", body) or [None, None])[1]
        packet = (re.search(r"(?m)^Packet: (\S+)", body) or [None, None])[1]
        at = utc(m.group(1))
        item = {"id": ident.group(1) if ident else None, "answer": key, "stamp": m.group(1), "packet": packet,
                "hours": round((now - at).total_seconds() / 3600, 1) if at else None,
                "read_by": readers or None}
        if item["id"] and key and packet:
            item["spawn"], item["problems"] = spawn_message(Path(packet).expanduser(), item["id"], key, now)
        out.append(item)
    return out


RECHECK_HOURS = 24


def spawn_message(packet: Path, item_id: str, key: str, now: dt.datetime) -> tuple[str | None, list[str]]:
    """The executor's standard message (LEVELS.md): its one line, a re-check line when the brief's premise is over a
    day old or its page changed after it, then the operator's stance verbatim: the brief's `## Stance`, else its
    topic's `## Operator intent (verbatim)`. No message while the brief cannot be executed cold, so nobody hand-types
    one (O7(f): two of three hand-made spawns dropped the stance or ran on a stale packet)."""
    if packet.name != BRIEF or packet.parent.name != item_id or not packet.is_file():
        return None, [f"no brief at work/{item_id}/{BRIEF} ({packet})"]
    text = read(packet)
    problems = brief_problems(text, [key])
    stance = re.search(r"(?ms)^## Stance[^\n]*\n(.*?)(?=^## |\Z)", text)
    mission = packet.parents[2] / "MISSION.md" if packet.parents[1].name == "work" else None
    if not stance and mission and mission.is_file():
        stance = re.search(r"(?ms)^## Operator intent \(verbatim\)[^\n]*\n(.*?)(?=^## |\Z)", read(mission))
    if not stance or not stance[1].strip():
        problems.append(f"no stance: add `## Stance` with the operator's words to {packet}")
    if problems:
        return None, problems
    line = f"Execute the approved decision `{item_id}`. Read `{packet}`; the answer is `{key}`."
    checked = utc(CHECKED.search(text)[0].split()[1]) if CHECKED.search(text) else None
    page_newer = any(p.stat().st_mtime > packet.stat().st_mtime for p in packet.parent.glob("*.htm*"))
    if checked and (now - checked).total_seconds() > RECHECK_HOURS * 3600 or page_newer:
        why = "its page changed after the brief" if page_newer else f"`Checked:` is {int((now - checked).total_seconds() // 3600)} h old"
        line += f" First re-check the brief's premise live ({why}); if it no longer holds, stop and report."
    return line + "\n\n" + stance[1].strip() + "\n", []


def utc(text: str) -> dt.datetime | None:
    text = text.strip().replace(" UTC", "Z").replace(" ", "T")
    if re.fullmatch(r"\d{4}-\d{2}-\d{2}", text):
        text += "T00:00Z"
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
           "delivered": fields.get("Delivered"), "problems": item["problems"] + packet_problems(item, base)}
    out.update(extra)
    return out


def ceo_level(portfolio: Path) -> dict:
    path = portfolio / "steward" / "state" / "asks.jsonl"
    latest: dict[str, dict] = {}
    packets: dict[str, str] = {}  # an ask's packet may sit on an earlier row of its history
    for line in read(path).splitlines():
        try:
            row = json.loads(line)
        except json.JSONDecodeError:
            continue
        if isinstance(row, dict) and row.get("id"):
            latest[row["id"]] = row
            if row.get("packet"):
                packets[row["id"]] = row["packet"]
    items = []
    for row in latest.values():
        if row.get("state") not in ("sent", "file-fallback", "queued", "held") or row.get("answer"):
            continue
        packet = packets.get(row["id"])
        if packet:
            packet = str(portfolio / Path(packet).expanduser())  # the steward stores it relative to the portfolio
        keys = [str(o).partition("=")[0].strip() for o in row.get("options") or []]
        check = {"id": row["id"], "recommended": row.get("recommend"), "options": keys, "explainer": row.get("explainer"),
                 "packet": packet or f"steward/asks/{row['id']}/{BRIEF}"}
        items.append({"id": row.get("key") or row["id"], "title": row.get("question") or row["id"],
                      "recommended": row.get("recommend"), "packet": packet,
                      "page": row.get("explainer"), "source": str(path), "delivered": row.get("sent_at"),
                      "state": row.get("state"), "ask": row["id"], "problems": packet_problems(check, portfolio)})
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
                 "inbox": str(folder / "INBOX.md"), "items": [],
                 "waiting": waiting_answers(folder, [c.id for c in related if c.live(now)], now)}
        for name, section, legacy in (("DECISIONS.md", "Needs you", False), ("STATE.md", "Needs you", True)):
            items, _ = parse(read(folder / name), section, full=True)
            for item in items:
                if not answered(item["id"], folder, related):
                    extra = {"legacy": "an older topic STATE.md: not delivered; move it to DECISIONS.md"} if legacy else {}
                    add(level, entry(item, folder / name, **extra), (folder.name, item["id"]))
        topic_levels[folder.name] = level
        levels.append(level)

    supervisor = {"level": "supervisor", "folder": str(research), "decisions": str(research / "ATTENTION.md"),
                  "inbox": str(research / "INBOX.md"), "items": [], "waiting": waiting_answers(research, [], now)}
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
    levels = [l for l in levels if l["items"] or l.get("waiting") or l["level"] in ("ceo", "supervisor")]
    return {"generated_at": now.strftime("%Y-%m-%dT%H:%M:%SZ"), "open": sum(len(l["items"]) for l in levels),
            "waiting": sum(len(l.get("waiting") or []) for l in levels),
            "levels": levels, "not_counted": sorted(set(not_counted))}


def print_open(report: dict) -> None:
    counts = ", ".join(f"{l['level']} {len(l['items'])}" for l in report["levels"])
    print(f"Open decisions, from files only ({report['generated_at']}): {report['open']} ({counts}); "
          f"answered and waiting for an executor: {report['waiting']}.")
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
        for w in level.get("waiting") or []:
            who = f"read by live {', '.join(w['read_by'])}" if w.get("read_by") else "no live commission reads it"
            print(f"  ANSWERED, waiting {w['hours']} h: {w['id'] or '(no item id)'} = {w['answer'] or '?'} ({who})")
            print(f"    start it: needs_you.py spawn {w['id']}" if w.get("spawn") else
                  f"    cannot start cold: {'; '.join(w.get('problems') or ['the answer names no packet'])}")
    if report["not_counted"]:
        print(f"\nNot counted: {len(report['not_counted'])} commissions hold needs-you items but their files do not say whether they "
              f"are live (no allowance end): {', '.join(report['not_counted'])}.")
    print("\nA delivered ask's Hub state is the desk's to read: `research-desk needs-you`. Its answer lands in the inbox named above.")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("action", choices=["validate", "write", "open", "spawn"])
    parser.add_argument("id", nargs="?", help="spawn: the answered decision's ID")
    parser.add_argument("--file", type=Path, help="Draft markdown file")
    parser.add_argument("--section", help='Section heading, for example "Needs you"')
    parser.add_argument("--target", type=Path, help="Destination file for the write action")
    parser.add_argument("--research", type=Path, default=Path.home() / ".vrooli" / "research")
    parser.add_argument("--portfolio", type=Path, default=Path.home() / ".vrooli" / "portfolio")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    if args.action == "spawn":
        report = open_decisions(args.research.expanduser(), args.portfolio.expanduser(), dt.datetime.now(dt.timezone.utc))
        hits = [w for level in report["levels"] for w in level.get("waiting") or [] if w["id"] == args.id]
        if not hits:
            print(f"no answer to {args.id} is waiting in any level's INBOX.md", file=sys.stderr)
            return 1
        if not hits[-1].get("spawn"):
            print(f"{args.id} cannot be executed cold:\n" + "\n".join(f"- {p}" for p in hits[-1]["problems"]), file=sys.stderr)
            return 1
        print(hits[-1]["spawn"], end="")
        return 0
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
    base = args.target.parent if args.target else Path.cwd()
    items, problems = parse(draft, args.section)
    if not problems:
        now = dt.datetime.now(dt.timezone.utc)
        problems = [p for item in items for p in packet_problems(item, base, now)]
    if problems:
        print("Needs-you draft rejected:\n" + "\n".join(f"- {problem}" for problem in problems), file=sys.stderr)
        return 1
    if args.action == "validate":
        print(f"Needs-you draft valid: {args.file} (## {args.section})")
        return 0
    try:
        current = args.target.read_text(encoding="utf-8") if args.target.exists() else ""
        merged, refusal = merge_section(current, draft, args.section)
        if refusal:
            print(f"Needs-you write refused: {args.target}: {refusal}", file=sys.stderr)
            return 1
        write_atomically(args.target, merged)
    except OSError as exc:
        print(f"cannot write validated draft to {args.target}: {exc}", file=sys.stderr)
        return 2
    print(f"Needs-you section written: {args.target} (## {args.section}; the rest of the file is unchanged)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
