#!/usr/bin/env python3
"""Real-throughput history of Vrooli research, measured from files and session logs.

  history.py portfolio [--json] [--since D] [--until D]
  history.py topic <name> [--json] [--since D] [--until D]
  history.py commission <id> [--json]

Read-only over the research home and the Codex session homes. Progress here means landed
workstreams, changed files, steady agent activity and scorecard movement; folder shape,
file counts and mission_check passing are not measured. Every value carries its source;
what cannot be derived is null ("unknown"), never 0.

Sources (strongest first; each row's "sources" names the one used):
  window   start: LAUNCH.json steps, a goal whose objective names cohorts/<id>/README.md, a
           time stamp in the id, a "started"/"dispatched" line, the first dated record, the
           first linked rollout line. End: the earlier of a stop/supersede/discard control (or
           closed_at) and the later of the allowance end and the last activity. A failed
           launch or "allowance_started": false has no window.
  session  rollouts of the linked Web Console home (LAUNCH or goal), else of the thread and
           home ids the commission's files name; parsed with effort-diagnosis's
           effort_anatomy helpers. Idle = no rollout line from any thread for over 10 min.
           No-edit = no FileChange/apply_patch edit (shell writes are not seen). "repo" =
           outside the research home. Owner threads of multi-topic commissions are given a
           topic from an older cohort's study map or the agent path; the rest stay unattributed.
  records  dated entries (a heading with a date, or a list item or table row that starts with
           one) in the cohort LEDGER.md (README/STATUS/FINAL without a ledger), the topic
           LEDGER.md and run CHECKPOINT.md files. Day-only dates are events, not gap points.
  landed   MISSION.md rows with status "landed <date>" (via mission_check), credited to a
           commission when the date falls in its window.
  verdicts operator entries in reviews/*/VERDICTS.md naming the topic.
  scorecard ledger entries that claim or deny scorecard/dimension movement (self-reported).

Scan results are cached under /tmp/claude-<uid>/research-history/ keyed by file size and
mtime, so repeated runs only read rollouts that changed. Python 3.9+, standard library.
"""
from __future__ import annotations

import argparse
import datetime as dt
import glob
import importlib.util
import json
import os
import re
import sqlite3
import sys
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Tuple

HOME = os.path.expanduser("~")
RESEARCH = os.path.join(HOME, ".vrooli/research")
WEB_HOMES = os.path.join(HOME, ".vrooli/state/vrooli/web-console/sessions/codex")
TUI_SESSIONS = os.path.join(HOME, ".codex/sessions")
CACHE = os.path.join("/tmp", f"claude-{os.getuid()}", "research-history", "rollouts-v1.json")
STALL_HOURS = 2.0
UTC = dt.timezone.utc
HERE = Path(__file__).resolve().parent


def _load(name: str, relative: str) -> Any:
    """Import a sibling skill script by path (works under python3 -I)."""
    for base in (HERE.parents[3], Path(HOME) / "Vrooli/scenarios"):
        path = base / relative
        if path.exists():
            spec = importlib.util.spec_from_file_location(name, path)
            module = importlib.util.module_from_spec(spec)
            sys.modules[name] = module
            spec.loader.exec_module(module)
            return module
    raise SystemExit(f"cannot find {relative} under the scenarios folder")


ea = _load("effort_anatomy", "prompt-manager/store/skills/packs/core/effort-diagnosis/scripts/effort_anatomy.py")
mc = _load("mission_check", "research-desk/skills/research-coordination/scripts/mission_check.py")
STALE_SECONDS = ea.ACTIVE_GAP_SECONDS

# ---------------------------------------------------------------- time and text

TS_RE = re.compile(r"(20\d\d-[01]\d-[0-3]\d)(?:[T ]([0-2]\d):([0-5]\d)(?::([0-5]\d))?(?:\.\d+)?"
                   r"(?:\s*(Z|UTC|[+-]\d\d:?\d\d))?)?")
ENTRY_RE = re.compile(r"^(?:#{1,6}\s+.*|\s*(?:[-*]|\d+\.)\s+\**`?(?=20\d\d-)|\|\s*\**(?=20\d\d-))")
UPDATED_RE = re.compile(r"(?i)\b(?:last )?updated\b\W{0,6}(20\d\d-\d\d-\d\d[T ]\d\d:\d\d)")
STARTED_RE = re.compile(r"(?i)\b(?:started(?:_at)?|dispatch(?:ed)?)\W{0,8}(20\d\d-\d\d-\d\d[T ]\d\d:\d\d(?::\d\d)?Z?)")
ALLOWANCE_RE = re.compile(r"(?i)\b(\d{1,3})[- ]?(?:h|hours?)\b[^.\n]{0,24}allowance|allowance (?:is |of )?(\d{1,3})[- ]?(?:h|hours?)\b")
THREAD_RE = re.compile(r"\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b")
SCORE_RE = re.compile(r"\b(scorecard|dimensions?)\b", re.I)
SCORE_NEG = re.compile(r"\b(no|not|nor|neither|without|zero|none|unchanged|remains?)\b", re.I)
SCORE_POS = re.compile(r"\b(mov(ed|es|ing)|advanc(ed|es)|rose|raised|improved|moved up)\b", re.I)


def when(text: str) -> Tuple[Optional[float], Optional[str]]:
    """First date in text as (epoch seconds UTC, 'minute'|'day'); unzoned times are read as UTC."""
    m = TS_RE.search(text or "")
    if not m:
        return None, None
    day, hh, mm, ss, zone = m.groups()
    offset = 0
    if zone and zone[0] in "+-":
        offset = (1 if zone[0] == "+" else -1) * (int(zone[1:3]) * 3600 + int(zone[-2:]) * 60)
    try:
        if hh is None:
            return dt.datetime.fromisoformat(day).replace(tzinfo=UTC).timestamp(), "day"
        return dt.datetime.fromisoformat(f"{day}T{hh}:{mm}:{ss or '00'}").replace(tzinfo=UTC).timestamp() - offset, "minute"
    except ValueError:
        return None, None


def iso(t: Optional[float]) -> Optional[str]:
    return None if t is None else dt.datetime.fromtimestamp(t, UTC).strftime("%Y-%m-%dT%H:%MZ")


def bound(text: Optional[str], end: bool = False) -> Optional[float]:
    """--since/--until value (a date or a time) as epoch; a bare --until date includes that day."""
    if not text:
        return None
    t, precision = when(text)
    if t is None:
        raise SystemExit(f"not a date: {text}")
    return t + 86400 if end and precision == "day" else t


class Unreadable(list):
    def note(self, path: str, err: Exception) -> None:
        self.append(f"{path}: {err.__class__.__name__}")


UNREADABLE = Unreadable()


def read(path: str) -> str:
    try:
        with open(path, encoding="utf-8", errors="replace") as handle:
            return handle.read()
    except FileNotFoundError:
        return ""
    except OSError as err:
        UNREADABLE.note(path, err)
        return ""


def read_json(path: str) -> Optional[dict]:
    text = read(path)
    try:
        value = json.loads(text) if text else None
    except ValueError:
        return None
    return value if isinstance(value, dict) else None


def entries(text: str, source: str) -> List[Dict[str, Any]]:
    """Dated entries: a heading with a date, or a list item / table row that starts with one."""
    out: List[Dict[str, Any]] = []
    for line in text.splitlines():
        if ENTRY_RE.match(line):
            t, precision = when(line)
            if t is not None:
                out.append({"t": t, "precision": precision, "title": line.strip("#*-| ").strip()[:140],
                            "body": [line], "source": source})
                continue
            if line.startswith("#"):
                out.append(None)  # an undated heading ends the previous entry
                continue
        if out and out[-1] is not None:
            out[-1]["body"].append(line)
    for line in text.splitlines()[:40]:
        m = UPDATED_RE.search(line)
        if m:
            t, precision = when(m.group(1))
            out.append({"t": t, "precision": precision, "title": "updated stamp", "body": [], "source": source})
            break
    return [e for e in out if e is not None]


def scorecard_claim(entry: Dict[str, Any]) -> Optional[str]:
    """'moved', 'none' or None for an entry, sentence by sentence (self-reported text)."""
    verdict = None
    for sentence in re.split(r"(?<=[.;])\s+", " ".join(entry["body"])):
        if not SCORE_RE.search(sentence):
            continue
        if SCORE_NEG.search(sentence):
            verdict = verdict or "none"
        elif SCORE_POS.search(sentence):
            return "moved"
    return verdict


# ---------------------------------------------------------------- intervals

def merge(spans: Iterable[Tuple[float, float]]) -> List[List[float]]:
    out: List[List[float]] = []
    for a, b in sorted(spans):
        if out and a - out[-1][1] <= STALE_SECONDS:
            out[-1][1] = max(out[-1][1], b)
        else:
            out.append([a, b])
    return out


def clip(spans: Iterable[Tuple[float, float]], windows: List[Tuple[float, float]]) -> List[List[float]]:
    return merge((max(a, s), min(b, e)) for a, b in spans for s, e in windows if min(b, e) >= max(a, s))


def gaps(points_or_spans: List[Any], windows: List[Tuple[float, float]]) -> List[Tuple[float, float]]:
    """Gaps between progress inside each window, edges included. Points are (t, t) spans."""
    out: List[Tuple[float, float]] = []
    for s, e in windows:
        cursor = s
        for a, b in sorted(x for x in points_or_spans if x[1] >= s and x[0] <= e):
            if a > cursor:
                out.append((cursor, a))
            cursor = max(cursor, b)
        if e > cursor:
            out.append((cursor, e))
    return out


def gap_stats(points_or_spans: List[Any], windows: List[Tuple[float, float]]) -> Optional[Dict[str, Any]]:
    if not windows:
        return None
    found = gaps(points_or_spans, windows)
    stalls = sorted((g for g in found if g[1] - g[0] > STALL_HOURS * 3600), key=lambda g: g[0] - g[1])
    total = sum(e - s for s, e in windows)
    return {"max_gap_h": round(max((b - a for a, b in found), default=0) / 3600, 2),
            "max_gap_at": iso(stalls[0][0]) if stalls else iso(max(found, key=lambda g: g[1] - g[0])[0]) if found else None,
            "gaps_over_2h": len(stalls),
            "stalled_share": round(sum(b - a for a, b in stalls) / total, 3) if total else None,
            "stalls": [[iso(a), round((b - a) / 3600, 2)] for a, b in stalls[:5]]}


# ---------------------------------------------------------------- sessions

def rollout_events(path: str) -> Dict[str, Any]:
    """One rollout as identity, activity spans, turn starts and timestamped edits.

    Same parsing rules as effort_anatomy.scan_rollout: inherited fork history is skipped,
    FileChange items are counted, and apply_patch text only when a file has no FileChange.
    """
    meta: Dict[str, Any] = {}
    stamps: List[float] = []
    turns: List[float] = []
    changes: List[list] = []
    patches: List[list] = []
    history_start = 0
    first: Optional[float] = None
    for index, line in enumerate(ea.iter_lines(path)):
        if index == 0 and '"session_meta"' in line[:200]:
            meta = ea.read_session_meta(line)
            history_start = meta.get("history_start") or 0
        m = re.match(r'\{"timestamp":"([^"]+)"', line)
        if not m:
            continue
        if history_start:
            ordinal = re.search(r'"ordinal":(\d+)', line[:200])
            if ordinal and int(ordinal.group(1)) < history_start:
                continue
        parsed = ea.parse_ts(m.group(1))
        if not parsed:
            continue
        t = parsed.timestamp()
        first = t if first is None else first
        if t < first:
            continue
        stamps.append(t)
        if '"type":"task_started"' in line:
            turns.append(t)
        elif '"item":{"type":"FileChange"' in line and '"item_completed"' in line:
            item = ((ea.loads(line) or {}).get("payload") or {}).get("item") or {}
            added = removed = 0
            files = []
            for changed_path, change in (item.get("changes") or {}).items():
                files.append(changed_path)
                if isinstance(change, dict):
                    diff = change.get("unified_diff") or ""
                    if diff:
                        a, r = ea.count_diff_lines(diff)
                    else:
                        content = change.get("content") or ""
                        a, r = content.count("\n") + (1 if content else 0), 0
                    added, removed = added + a, removed + r
            changes.append([t, added, removed, files])
        elif "*** Begin Patch" in line and ('"type":"function_call"' in line[:400] or '"type":"custom_tool_call"' in line[:400]):
            payload = (ea.loads(line) or {}).get("payload") or {}
            text = payload.get("input") if isinstance(payload.get("input"), str) else str(payload.get("arguments") or "")
            a, r, files = ea.count_patch_text(text.replace("\\n", "\n"))
            patches.append([t, a, r, files])
    spans = merge((t, t) for t in stamps)
    return {"id": meta.get("id"), "parent": meta.get("parent"), "depth": meta.get("depth") or 0,
            "agent": meta.get("path"), "spans": spans, "turns": turns,
            "changes": changes or patches, "lines_source": "file_change" if changes or not patches else "apply_patch"}


class Sessions:
    """Rollout index, goal links and a size/mtime-keyed scan cache."""

    def __init__(self, web_homes: str = WEB_HOMES, tui: str = TUI_SESSIONS, cache: Optional[str] = CACHE):
        self.web_homes, self.cache_path = web_homes, cache
        self.by_home: Dict[str, List[str]] = {}
        self.by_thread: Dict[str, str] = {}
        self.tui_files: List[str] = []
        for path in glob.glob(os.path.join(web_homes, "*/sessions/*/*/*/rollout-*.jsonl")):
            home = path[len(web_homes):].strip("/").split("/")[0]
            self.by_home.setdefault(home, []).append(path)
        self.tui_files = glob.glob(os.path.join(tui, "*/*/*/rollout-*.jsonl"))
        for path in [p for files in self.by_home.values() for p in files] + self.tui_files:
            m = THREAD_RE.findall(os.path.basename(path))
            if m:
                self.by_thread.setdefault(m[-1], path)
        self._goals: Optional[List[Dict[str, Any]]] = None
        self.cache: Dict[str, Any] = {}
        self.dirty = False
        if cache and os.path.exists(cache):
            try:
                with open(cache, encoding="utf-8") as handle:
                    self.cache = json.load(handle)
            except (OSError, ValueError):
                self.cache = {}

    def goals(self) -> List[Dict[str, Any]]:
        if self._goals is None:
            self._goals = []
            for db in glob.glob(os.path.join(self.web_homes, "*/goals_1.sqlite")):
                try:
                    conn = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
                    for thread, objective, created in conn.execute(
                            "select thread_id, objective, created_at_ms from thread_goals"):
                        self._goals.append({"home": os.path.basename(os.path.dirname(db)), "thread": thread,
                                            "objective": objective or "", "created": (created or 0) / 1000})
                    conn.close()
                except sqlite3.Error as err:
                    UNREADABLE.note(db, err)
        return self._goals

    def scan(self, path: str) -> Optional[Dict[str, Any]]:
        try:
            st = os.stat(path)
        except OSError as err:
            UNREADABLE.note(path, err)
            return None
        key = [st.st_size, st.st_mtime_ns]
        hit = self.cache.get(path)
        if hit and hit.get("key") == key:
            return hit["data"]
        try:
            data = rollout_events(path)
        except OSError as err:
            UNREADABLE.note(path, err)
            return None
        self.cache[path] = {"key": key, "data": data}
        self.dirty = True
        return data

    def save(self) -> None:
        if not (self.cache_path and self.dirty):
            return
        os.makedirs(os.path.dirname(self.cache_path), exist_ok=True)
        tmp = self.cache_path + ".tmp"
        with open(tmp, "w", encoding="utf-8") as handle:
            json.dump(self.cache, handle)
        os.replace(tmp, self.cache_path)

    def threads(self, homes: Iterable[str], thread_ids: Iterable[str]) -> List[Dict[str, Any]]:
        """Every rollout of the homes, plus each named thread and its descendants in its folder."""
        files = {p for h in homes for p in self.by_home.get(h, [])}
        for tid in thread_ids:
            path = self.by_thread.get(tid)
            if not path:
                continue
            pool = self.tui_files if path in self.tui_files else self.by_home.get(path[len(self.web_homes):].strip("/").split("/")[0], [])
            rows = [r for r in (self.scan(p) for p in pool) if r]
            keep, frontier = {tid}, [tid]
            while frontier:
                parent = frontier.pop()
                for r in rows:
                    if r["parent"] == parent and r["id"] not in keep:
                        keep.add(r["id"])
                        frontier.append(r["id"])
            files |= {p for p in pool if (self.scan(p) or {}).get("id") in keep}
        out = []
        for path in sorted(files):
            row = self.scan(path)
            if row:
                out.append(dict(row, file=path))
        return out


# ---------------------------------------------------------------- research home

def topic_names(home: str) -> Tuple[List[str], Dict[str, str]]:
    """Real topic folders and alias -> real name (symlinked aliases)."""
    root = os.path.join(home, "topics")
    real, alias = [], {}
    for name in sorted(os.listdir(root)) if os.path.isdir(root) else []:
        path = os.path.join(root, name)
        if os.path.islink(path):
            target = os.path.relpath(os.path.realpath(path), os.path.realpath(root)).split(os.sep)[0]
            alias[name] = target
        elif os.path.isdir(path):
            real.append(name)
    return real, alias


def mission(home: str, topic: str) -> Dict[str, Any]:
    """Workstream rows from mission_check, with the date in each status cell."""
    path = Path(home) / "topics" / topic / "MISSION.md"
    if not path.exists():
        return {"rows": [], "title": topic, "source": None, "mode": None}
    try:
        report = mc.check_mission(topic, path)
        text = path.read_text(encoding="utf-8")
    except OSError as err:
        UNREADABLE.note(str(path), err)
        return {"rows": [], "title": topic, "source": None, "mode": None}
    cells = {c[0]: c[5] for c in mc.table_rows(mc.split_sections(text).get("Workstreams", "")) if len(c) >= 6}
    rows = []
    for w in report.workstreams:
        t, precision = when(cells.get(w.ident, ""))
        rows.append({"id": w.ident, "status": w.status, "date": iso(t)[:10] if t else None, "t": t})
    title = next((l[2:].strip() for l in text.splitlines() if l.startswith("# ")), topic)
    return {"rows": rows, "title": title, "source": str(path), "mode": report.mode or None}


def topic_tokens(name: str, title: str, rows: List[Dict[str, Any]]) -> set:
    stop = {"and", "the", "for", "with", "mission", "research", "topic", "programme", "useful"}
    words = {w for w in re.findall(r"[a-z]{3,}", f"{name} {title}".lower()) if w not in stop}
    prefixes = {re.match(r"[A-Za-z]+", r["id"]).group(0).lower() for r in rows if re.match(r"[A-Za-z]+\d", r["id"])}
    return words | prefixes


def attribute(agent: Optional[str], tokens: Dict[str, set]) -> Optional[str]:
    """Topic of an owner thread from its agent path (e.g. /root/debt_reduction_owner/dr1_x)."""
    if not agent:
        return None
    parts = re.split(r"[/_\-]+", agent.lower())
    words = set(parts) | {re.sub(r"\d+$", "", p) for p in parts}
    scores = {t: len(words & toks) for t, toks in tokens.items()}
    best = max(scores.values(), default=0)
    winners = [t for t, s in scores.items() if s == best]
    return winners[0] if best and len(winners) == 1 else None


def verdicts(home: str) -> List[Dict[str, Any]]:
    out = []
    for path in sorted(glob.glob(os.path.join(home, "reviews/*/VERDICTS.md"))):
        review = os.path.basename(os.path.dirname(path))
        found = []
        for e in entries(read(path), path):
            if "provenance correction" in e["title"]:  # voids every earlier verdict in the file
                found = []
                continue
            m = re.search(r"verdict\s+([a-z0-9, -]+?)(?:\s+[A-Za-z0-9_-]*[A-Z][A-Za-z0-9_-]*)?\s*·", e["title"])
            if not m:
                continue
            names = [n.strip() for n in re.split(r"[,\s]+", m.group(1)) if n.strip()]
            text = " ".join(e["body"][1:]).strip()
            found.append({"t": e["t"], "review": review, "topics": names, "excerpt": text[:160], "source": path})
        out += found
    return out


class Home:
    def __init__(self, home: str = RESEARCH, sessions: Optional[Sessions] = None, now: Optional[float] = None):
        self.home = home
        self.roots = tuple({os.path.abspath(home) + os.sep, os.path.realpath(home) + os.sep})
        self.sessions = sessions or Sessions()
        self.now = now or dt.datetime.now(UTC).timestamp()
        self.topics, self.alias = topic_names(home)
        cohorts = os.path.join(home, "cohorts")
        self.cids = sorted(d for d in os.listdir(cohorts) if os.path.isdir(os.path.join(cohorts, d))) if os.path.isdir(cohorts) else []
        self._missions: Dict[str, Dict[str, Any]] = {}
        self._verdicts: Optional[List[Dict[str, Any]]] = None
        self._runs: Optional[Dict[str, List[Tuple[str, str]]]] = None

    def in_repo(self, change: list) -> bool:
        """An edit outside the research home (research notes are counted separately)."""
        return not any(f.startswith(self.roots) or ".vrooli/research/" in f for f in change[3])

    def real(self, topic: str) -> Optional[str]:
        topic = self.alias.get(topic, topic)
        return topic if topic in self.topics else None

    def mission(self, topic: str) -> Dict[str, Any]:
        if topic not in self._missions:
            self._missions[topic] = mission(self.home, topic)
        return self._missions[topic]

    def verdicts(self) -> List[Dict[str, Any]]:
        if self._verdicts is None:
            self._verdicts = verdicts(self.home)
            for v in self._verdicts:
                v["topics"] = [self.real(n) or n for n in v["topics"]]
        return self._verdicts

    def runs(self, cid: str) -> List[Tuple[str, str]]:
        """(topic, run folder) pairs of this commission: the longest commission id that prefixes the
        folder name, else every commission whose charter link the run's CHECKPOINT/README carries."""
        if self._runs is None:
            self._runs = {}
            link = re.compile(r"cohorts/([\w.-]+?)(?:/|\)|`|\s|$)")
            for topic in self.topics:
                for run in sorted(glob.glob(os.path.join(self.home, "topics", topic, "runs", "*/"))):
                    run = run.rstrip("/")
                    name = os.path.basename(run)
                    owners = [c for c in self.cids if name == c or name.startswith(c + "-")]
                    if owners:
                        found = {max(owners, key=len)}
                    else:
                        head = read(os.path.join(run, "CHECKPOINT.md"))[:4000] + read(os.path.join(run, "README.md"))[:4000]
                        found = {m for m in link.findall(head) if m in self.cids}
                    for c in found:
                        self._runs.setdefault(c, []).append((topic, run))
        return self._runs.get(cid, [])

    # ------------------------------------------------------------ commissions

    def commission(self, cid: str, since: Optional[float] = None, until: Optional[float] = None,
                   only_topic: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """One commission's row; None when only_topic is given and the commission does not touch it."""
        folder = os.path.join(self.home, "cohorts", cid)
        if not os.path.isdir(folder):
            raise SystemExit(f"no commission {cid} under {self.home}/cohorts")
        top = {name: read(os.path.join(folder, name)) for name in sorted(os.listdir(folder))
               if os.path.isfile(os.path.join(folder, name)) and name.endswith((".md", ".json"))}
        launch = read_json(os.path.join(folder, "LAUNCH.json")) or {}
        spec = (read_json(os.path.join(folder, "PACKET.json")) or {}).get("spec") or {}
        legacy = {**(read_json(os.path.join(folder, "COHORT.json")) or {}), **(read_json(os.path.join(folder, "STATUS.json")) or {})}
        runs = self.runs(cid)
        src: Dict[str, Any] = {}

        # topics: packet, older study maps, run folders, charter links
        topics = [self.real(t) for t in spec.get("topics") or []]
        src["topics"] = "PACKET.json" if topics else None
        if not topics and isinstance(legacy.get("studies"), dict):
            topics, src["topics"] = [self.real(t) for t in legacy["studies"]], "COHORT/STATUS.json studies"
        if not topics:
            linked = re.findall(r"topics/([a-z0-9-]+)/", top.get("README.md", ""))
            topics = [self.real(t) for t in linked] + [t for t, _ in runs]
            src["topics"] = "charter links and run folders" if topics else None
        topics = sorted({t for t in topics if t})
        if only_topic and only_topic not in topics:
            return None

        # session links: LAUNCH home, goals whose objective names the charter, else ids the files name
        homes, threads, links = [], [], []
        if launch.get("web_console_session_id"):
            homes.append(launch["web_console_session_id"])
            links.append(f"launch:{launch['web_console_session_id']}")
        charter = re.compile(r"cohorts/" + re.escape(cid) + r"/README\.md")
        goal_start = None
        for goal in self.sessions.goals():
            if charter.search(goal["objective"]) and goal["home"] not in homes:
                homes.append(goal["home"])
                links.append(f"goal:{goal['home']}/{goal['thread']}")
                goal_start = min(goal_start or goal["created"], goal["created"])
        dedicated = bool(homes)
        if not homes:
            named = sorted({t for name, text in top.items() if name != "LAUNCH.json" for t in THREAD_RE.findall(text)})
            ref_homes = [t for t in named if t in self.sessions.by_home]
            threads = [t for t in named if t in self.sessions.by_thread]
            homes += ref_homes
            links += [f"home-ref:{h}" for h in ref_homes] + [f"thread-ref:{t}" for t in threads]

        # records
        record_names = ["LEDGER.md"] if "LEDGER.md" in top else [n for n in ("STATUS.md", "FINAL.md", "CHECKPOINT.md", "CLOSURE.md", "CURRENT.md", "README.md") if n in top]
        recs = [e for n in record_names for e in entries(top[n], os.path.join(folder, n))]
        checkpoints = [e for _t, run in runs for e in entries(read(os.path.join(run, "CHECKPOINT.md")), os.path.join(run, "CHECKPOINT.md"))]
        operator = [e for e in entries(top.get("OPERATOR.md", ""), os.path.join(folder, "OPERATOR.md")) if "·" in e["title"]]
        timed = sorted(e["t"] for e in recs if e["precision"] == "minute")
        allowance = spec.get("allowance_hours") or legacy.get("allowance_hours") or legacy.get("allowance_hours_each")
        if not allowance:
            m = ALLOWANCE_RE.search(top.get("KERNEL.txt", "") or read(os.path.join(folder, "KERNEL.txt")) or top.get("README.md", "")[:3000])
            allowance = int(m.group(1) or m.group(2)) if m else None

        # window start, strongest source first
        steps = [t for t in (when(s.get("at", ""))[0] for s in launch.get("steps") or []) if t]
        stamp = re.search(r"(20\d{6})T(\d{6})Z", cid)
        start, src["start"] = None, None
        if steps:
            start, src["start"] = min(steps), "LAUNCH.json steps"
        elif goal_start:
            start, src["start"] = goal_start, "goal created_at (goals_1.sqlite)"
        elif stamp:
            start = dt.datetime.strptime("".join(stamp.groups()), "%Y%m%d%H%M%S").replace(tzinfo=UTC).timestamp()
            src["start"] = "commission id time stamp"
        else:
            for name in ("STATE.md", "README.md", "CHECKPOINT.md", "FINAL.md", "STATUS.md"):
                m = STARTED_RE.search(top.get(name, ""))
                if m:
                    start, src["start"] = when(m.group(1))[0], f"{name} 'started' line"
                    break
        day = re.search(r"(20\d{6})", cid)
        floor = dt.datetime.strptime(day.group(1), "%Y%m%d").replace(tzinfo=UTC).timestamp() if day else None
        threads_rows = self.sessions.threads(homes, threads) if (homes or threads) else []
        if not dedicated and floor:  # file-referenced threads: drop predecessors that started before this commission
            threads_rows = [r for r in threads_rows if r["spans"] and r["spans"][0][0] >= (start or floor) - 3600]
        if start is None and timed:
            start, src["start"] = timed[0], "first dated record entry"
        if start is None and threads_rows:
            start, src["start"] = min(r["spans"][0][0] for r in threads_rows if r["spans"]), "first linked thread line"
        if legacy.get("allowance_started") is False and not steps:
            start, src["start"], threads_rows = None, "COHORT.json allowance_started false", []
        if launch.get("failed"):
            start, src["start"], threads_rows = None, f"LAUNCH.json failed at {launch['failed'].get('step')}", []

        # window end
        stops = [when(c.get("at", ""))[0] for c in launch.get("controls") or [] if c.get("action") in ("stop", "discard", "supersede")]
        stops = [t for t in stops if t and (start is None or t >= start)]
        last_activity = max((s[1] for r in threads_rows for s in r["spans"] if start and s[1] >= start), default=None)
        end, src["end"], allowance_end, hard = None, None, None, None
        if start is not None:
            closed = when(str(legacy.get("closed_at") or ""))[0]
            hard = min(stops) if stops else closed if closed and closed > start else None
            last = {"records": timed[-1]} if timed and timed[-1] > start else {}
            if last_activity and (dedicated or not allowance):  # file-referenced threads may run on into a successor
                last["session"] = last_activity if dedicated else min(last_activity, start + 48 * 3600)
            allowance_end = start + float(allowance) * 3600 if allowance else None
            soft, why = None, None
            if last:
                why = max(last, key=last.get)
                soft = last[why]
                why = f"last {why} activity"
            if allowance_end and (soft is None or allowance_end >= soft):
                soft, why = allowance_end, "allowance end (start + allowance)"
            if hard is not None and (soft is None or hard < soft):
                end, src["end"] = hard, "LAUNCH.json stop/supersede/discard control" if stops else "COHORT/STATUS.json closed_at"
            else:
                end, src["end"] = soft, why
            if end is not None:
                end = min(end, self.now)
        state = ("failed" if launch.get("failed") else "discarded" if launch.get("discarded") and not steps and start is None
                 else "superseded" if launch.get("superseded_by") or any(c.get("action") == "supersede" for c in launch.get("controls") or [])
                 else ("never-started" if src["start"] == "COHORT.json allowance_started false" else "no-window") if start is None
                 else "no-end" if end is None
                 else "live" if end is not None and not stops and self.now - end < 2 * 3600 and dedicated else "ended")

        row: Dict[str, Any] = {"id": cid, "kind": "steward" if cid.startswith("steward-") else "research", "state": state,
                               "topics": topics, "start": iso(start), "end": iso(end), "allowance_h": allowance,
                               "allowance_end": iso(allowance_end), "stopped_at": iso(hard),
                               "session_links": links, "sources": src}
        window = [(start, end)] if start is not None and end is not None and end > start else []
        if window and (since or until):
            window = [(max(start, since or start), min(end, until or end))]
            window = [w for w in window if w[1] > w[0]]
        row["window_h"] = round(sum(e - s for s, e in window) / 3600, 2) if window else None
        threads_rows = [r for r in threads_rows if clip((tuple(x) for x in r["spans"]), window)] if window else []
        self.attribute(threads_rows, topics, legacy)
        lines: Dict[str, int] = {}
        for th in threads_rows:
            key = th["topic"] or "unattributed"
            lines[key] = lines.get(key, 0) + sum(c[1] + c[2] for c in th["changes"] if any(a <= c[0] <= b for a, b in window))
        row["lines_by_topic"] = lines if threads_rows else None
        row.update(measure(window, threads_rows, recs, checkpoints, operator, self, topics, dedicated or bool(threads)))
        row["coverage"] = coverage(row)
        row["_window"], row["_threads"], row["_records"] = window, threads_rows, recs
        return row

    def attribute(self, threads_rows: List[Dict[str, Any]], topics: List[str], legacy: Dict[str, Any]) -> None:
        """Set each thread's topic: the only topic, an older cohort's topic -> thread map (children
        inherit), else the topic whose name, title words or workstream prefix the agent path carries."""
        by_id = {t["id"]: t for t in threads_rows if t.get("id")}
        mapped: Dict[str, str] = {}
        for key in ("threads", "studies"):
            for name, value in (legacy.get(key) or {}).items() if isinstance(legacy.get(key), dict) else []:
                for tid in THREAD_RE.findall(json.dumps(value)):
                    if self.real(name):
                        mapped.setdefault(tid, self.real(name))
        tokens = {t: topic_tokens(t, self.mission(t)["title"], self.mission(t)["rows"]) for t in topics}
        for th in threads_rows:
            th["topic"] = topics[0] if len(topics) == 1 else None
            node, hops = th, 0
            while node and not th["topic"] and hops < 10:
                th["topic"] = mapped.get(node.get("id")) or attribute(node.get("agent"), tokens)
                node, hops = by_id.get(node.get("parent")), hops + 1

    def commissions(self, since=None, until=None) -> List[Dict[str, Any]]:
        """Every commission; with --since/--until, those whose window misses the range are dropped
        (commissions with no known window stay listed)."""
        rows = [self.commission(c, since, until) for c in self.cids]
        return [r for r in rows if r["_window"] or r["start"] is None or r["end"] is None]


def measure(window, threads_rows, records, checkpoints, operator, home: Home, topics: List[str], linked: bool) -> Dict[str, Any]:
    """Activity, stalls, change, landed and scorecard values for one window; unknown stays None."""
    out: Dict[str, Any] = {}
    days = sum(e - s for s, e in window) / 86400 if window else 0
    inside = lambda t: any(s <= t <= e for s, e in window)
    within = lambda es: [e for e in es if inside(e["t"]) or (e["precision"] == "day" and any(s - 86400 < e["t"] <= e_ for s, e_ in window))]
    recs = within(records)
    spans = clip((tuple(s) for r in threads_rows for s in r["spans"]), window) if window else []
    active_days = {int(e["t"] // 86400) for e in recs} | {d for a, b in spans for d in range(int(a // 86400), int(b // 86400) + 1)}
    out["record_entries"] = len(recs) if window else None
    out["active_days"] = len(active_days) if window else None
    out["record_entries_per_active_day"] = round(len(recs) / len(active_days), 1) if window and active_days else None
    out["checkpoint_entries"] = len(within(checkpoints)) if window else None
    timed = [(e["t"], e["t"]) for e in recs if e["precision"] == "minute"]
    out["record_gaps"] = gap_stats(timed, window) if timed else None
    out["operator_entries"] = len([e for e in operator if inside(e["t"])]) if window else None
    claims = [scorecard_claim(e) for e in recs]
    out["scorecard_self_reported"] = {"moved": claims.count("moved"), "none": claims.count("none")} if recs else None
    if window and threads_rows:
        changes = [c for r in threads_rows for c in r["changes"] if inside(c[0])]
        repo = [c for c in changes if home.in_repo(c)]
        out["session"] = {
            "threads": len(threads_rows),
            "active_h": round(sum(b - a for a, b in spans) / 3600, 2),
            "active_share": round(sum(b - a for a, b in spans) / sum(e - s for s, e in window), 3),
            "turns_per_day": round(sum(1 for r in threads_rows for t in r["turns"] if inside(t)) / days, 1) if days else None,
            "idle": gap_stats(spans, window),
            "no_change": gap_stats([(c[0], c[0]) for c in changes], window),
            "lines_added": sum(c[1] for c in changes), "lines_removed": sum(c[2] for c in changes),
            "repo_lines_added": sum(c[1] for c in repo), "repo_lines_removed": sum(c[2] for c in repo),
            "paths_changed": len({f for c in changes for f in c[3]}),
            "repo_paths_changed": len({f for c in changes for f in c[3] if home.in_repo([0, 0, 0, [f]])}),
            "lines_source": sorted({r["lines_source"] for r in threads_rows}),
            # time to first edit: hours from the window's start to the first repository change
            "first_repo_edit_h": round((min(c[0] for c in repo) - window[0][0]) / 3600, 2) if repo else None,
        }
        # dated records written while no linked rollout moved: the work ran somewhere this tool cannot see
        stalls = [g for g in gaps(spans, window) if g[1] - g[0] > STALL_HOURS * 3600]
        unseen = sum(1 for t, _ in timed if any(a < t < b for a, b in stalls))
        if unseen >= 3:
            out["session"]["partial"] = f"{unseen} dated record entries fall in session idle gaps over {STALL_HOURS:g} h"
    else:
        out["session"] = None
        out["session_unknown_reason"] = ("no window" if not window else "no linked session" if not linked
                                         else "linked threads not found in session homes")
    landed = []
    for topic in topics:
        for w in home.mission(topic)["rows"]:
            if w["status"] == "landed" and w["t"] is not None and any(s - 86400 < w["t"] <= e for s, e in window):
                landed.append(f"{topic}:{w['id']}")
    out["landed"] = landed if window and topics else None
    return out


def coverage(row: Dict[str, Any]) -> str:
    """One letter per value group; '-' unknown, lower-case 's' a session that misses dated work."""
    session = row.get("session")
    groups = [("W", row.get("window_h") is not None), ("R", bool(row.get("record_entries"))),
              ("s" if session and session.get("partial") else "S", session is not None),
              ("L", row.get("landed") is not None), ("C", bool(row.get("scorecard_self_reported")))]
    return "".join(k if known else "-" for k, known in groups)


# ---------------------------------------------------------------- topics

def topic_row(home: Home, topic: str, rows: List[Dict[str, Any]], since=None, until=None) -> Dict[str, Any]:
    m = home.mission(topic)
    mine = [r for r in rows if topic in r["topics"]]
    windows = [(a, b) for a, b in merge((s, e) for r in mine for s, e in r["_window"])]
    spans, changes, shared_lines, attributed_rows = [], [], 0, 0
    measured = [r for r in mine if r.get("session") and not r["session"].get("partial")]
    seen = [(a, b) for a, b in merge((s, e) for r in measured for s, e in r["_window"])]
    for r in measured:
        for th in r["_threads"]:
            owner = th.get("topic")
            inside = [c for c in th["changes"] if any(s <= c[0] <= e for s, e in r["_window"])]
            if owner == topic:
                attributed_rows += 1
                spans.extend(clip((tuple(s) for s in th["spans"]), r["_window"]))
                changes.extend(inside)
            elif owner is None:
                shared_lines += sum(c[1] + c[2] for c in inside)
    tdir = os.path.join(home.home, "topics", topic)
    recs = entries(read(os.path.join(tdir, "LEDGER.md")), os.path.join(tdir, "LEDGER.md"))
    for path in glob.glob(os.path.join(tdir, "runs/*/CHECKPOINT.md")):
        recs.extend(entries(read(path), path))
    ids = {w["id"] for w in m["rows"]}
    named = re.compile(r"\b(" + "|".join(map(re.escape, sorted(ids | {topic}, key=len, reverse=True))) + r")\b") if ids else re.compile(re.escape(topic))
    for r in mine:
        recs.extend(e for e in r["_records"] if named.search(" ".join(e["body"])))
    lo, hi = since or float("-inf"), until or float("inf")
    recs = [e for e in recs if lo <= e["t"] < hi]
    landed = [w for w in m["rows"] if w["status"] == "landed"]
    landed_in = [w for w in landed if w["t"] is not None and lo <= w["t"] < hi]
    vs = [v for v in home.verdicts() if topic in v["topics"] and lo <= v["t"] < hi]
    names = [topic] + [a for a, real in home.alias.items() if real == topic]
    reviewed = sorted(r for r in (os.path.basename(p) for p in glob.glob(os.path.join(home.home, "reviews", "*")) if os.path.isdir(p))
                      if any(n in r for n in names) or any(re.search(rf"^#+ .*\b{re.escape(n)}\b", read(os.path.join(home.home, "reviews", r, "REVIEW.md")), re.M) for n in names))
    days = sorted({int(e["t"] // 86400) for e in recs} | {d for s, e in windows for d in range(int(s // 86400), int(e // 86400) + 1)})
    dormant = max((b - a - 1 for a, b in zip(days, days[1:])), default=None)
    last = max([e["t"] for e in recs] + [b for _a, b in spans if lo <= b < hi], default=None)
    claims = [scorecard_claim(e) for e in recs]
    statuses: Dict[str, int] = {}
    for w in m["rows"]:
        statuses[w["status"]] = statuses.get(w["status"], 0) + 1
    out = {"topic": topic, "title": m["title"], "mode": m["mode"], "commissions": [r["id"] for r in mine],
           "commissions_with_window": len([r for r in mine if r["_window"]]),
           "window_h": round(sum(b - a for a, b in windows) / 3600, 2) if windows else None,
           "mission_status": statuses, "landed_all": [f"{w['id']} {w['date'] or 'undated'}" for w in landed],
           "landed_in_range": [f"{w['id']} {w['date']}" for w in landed_in],
           "verdicts": len(vs), "verdict_reviews": sorted({v["review"] for v in vs}), "reviews": reviewed,
           "record_entries": len(recs), "last_progress": iso(last), "max_dormant_days": dormant,
           "days_since_progress": round((min(home.now, hi) - last) / 86400, 1) if last else None,
           "scorecard_self_reported": {"moved": claims.count("moved"), "none": claims.count("none")},
           "sources": {"landed": m["source"], "mission": "mission_check.check_mission",
                       "session": "owner threads attributed by agent path; single-topic commissions whole"}}
    if seen and attributed_rows:
        out["session"] = {"attributed_threads": attributed_rows, "measured_window_h": round(sum(b - a for a, b in seen) / 3600, 2),
                          "excluded_partial": [r["id"] for r in mine if r.get("session") and r["session"].get("partial")],
                          "active_share": round(sum(b - a for a, b in merge(spans)) / sum(b - a for a, b in seen), 3),
                          "idle": gap_stats(merge(spans), seen),
                          "no_change": gap_stats([(c[0], c[0]) for c in changes], seen),
                          "lines_added": sum(c[1] for c in changes), "lines_removed": sum(c[2] for c in changes),
                          "repo_lines": sum(c[1] + c[2] for c in changes if home.in_repo(c)),
                          "paths_changed": len({f for c in changes for f in c[3]}),
                          "unattributed_lines_in_shared_commissions": shared_lines}
    else:
        out["session"] = None
    out["coverage"] = "".join(k if v else "-" for k, v in {"W": bool(windows), "R": bool(recs), "S": out["session"] is not None,
                                                            "L": bool(m["rows"]), "V": bool(vs)}.items())
    return out


# ---------------------------------------------------------------- output

def public(row: Dict[str, Any]) -> Dict[str, Any]:
    return {k: v for k, v in row.items() if not k.startswith("_")}


def fmt(value: Any, digits: int = 1) -> str:
    if value is None:
        return "?"
    if isinstance(value, float):
        return f"{value:.{digits}f}"
    return str(value)


def pct(value: Optional[float]) -> str:
    return "?" if value is None else f"{value * 100:.0f}%"


COMMISSION_HEAD = ("| commission | kind | state | start | win h | active | max idle h | idle>2h | max no-edit h | +repo | -repo | "
                   "repo paths | landed | units/active day | score +/- | cov |\n|" + "---|" * 16)


def commission_line(r: Dict[str, Any]) -> str:
    s = r.get("session") or {}
    idle, edit = s.get("idle") or {}, s.get("no_change") or {}
    sc = r.get("scorecard_self_reported")
    cells = [r["id"], r["kind"], r["state"], (r["start"] or "?")[:16], fmt(r["window_h"]), pct(s.get("active_share")),
             fmt(idle.get("max_gap_h")), fmt(idle.get("gaps_over_2h")), fmt(edit.get("max_gap_h")),
             fmt(s.get("repo_lines_added")), fmt(s.get("repo_lines_removed")), fmt(s.get("repo_paths_changed")),
             fmt(len(r["landed"]) if r.get("landed") is not None else None), fmt(r.get("record_entries_per_active_day")),
             f"{sc['moved']}/{sc['none']}" if sc else "?", r["coverage"]]
    return "| " + " | ".join(cells) + " |"


TOPIC_HEAD = ("| topic | mode | commissions run/named | win h | active | max idle h | stalled | repo lines | landed (range/all) | verdicts | "
              "last progress | max dormant d | score +/- | cov |\n|" + "---|" * 14)


def topic_line(t: Dict[str, Any]) -> str:
    s = t.get("session") or {}
    return (f"| {t['topic']} | {t['mode'] or '?'} | {t['commissions_with_window']}/{len(t['commissions'])} | {fmt(t['window_h'])} | {pct(s.get('active_share'))} | "
            f"{fmt((s.get('idle') or {}).get('max_gap_h'))} | {pct((s.get('idle') or {}).get('stalled_share'))} | {fmt(s.get('repo_lines'))} | "
            f"{len(t['landed_in_range'])}/{len(t['landed_all'])} | {t['verdicts']} | {(t['last_progress'] or '?')[:10]} | "
            f"{fmt(t['max_dormant_days'])} | {t['scorecard_self_reported']['moved']}/{t['scorecard_self_reported']['none']} | {t['coverage']} |")


LEGEND = ("cov letters: W window, R dated records, S session rollouts (s = partial: dated records fall in its idle gaps), "
          "L mission landed rows, C/V scorecard claims or verdicts; '-' = unknown. ? = unknown, never 0. active = share of window with any agent line "
          "(gaps over 10 min are idle). no-edit = time between FileChange/apply_patch edits. repo = outside "
          "~/.vrooli/research. score +/- = ledger entries claiming / denying scorecard movement (self-reported).")


def portfolio(home: Home, since=None, until=None) -> Dict[str, Any]:
    rows = home.commissions(since, until)
    owners: Dict[str, List[Dict[str, Any]]] = {}
    for r in rows:
        for th in r["_threads"]:
            owners.setdefault(th["file"], []).append(r)
    overlap = lambda a, b: any(min(e1, e2) > max(s1, s2) for s1, e1 in a for s2, e2 in b)
    for r in rows:
        r["shared_session_with"] = sorted({o["id"] for th in r["_threads"] for o in owners[th["file"]]
                                           if o is not r and overlap(o["_window"], r["_window"])})
    totals = {}
    for kind in ("research", "steward"):
        mine = [r for r in rows if r["kind"] == kind and r["_window"]]
        seen: Dict[str, List[Tuple[float, float]]] = {}
        for r in mine:
            for th in r["_threads"]:
                seen.setdefault(th["file"], []).extend(r["_window"])
        files = {th["file"]: th for r in mine for th in r["_threads"]}
        changes = [c for f, th in files.items() for c in th["changes"] if any(a <= c[0] <= b for a, b in seen[f])]
        repo = [c for c in changes if home.in_repo(c)]
        spans = merge(x for f, th in files.items() for x in clip((tuple(y) for y in th["spans"]), merge(seen[f])))
        totals[kind] = {"commissions_with_window": len(mine),
                        "window_h_union": round(sum(b - a for a, b in merge(w for r in mine for w in r["_window"])) / 3600, 1),
                        "agent_active_h_union": round(sum(b - a for a, b in spans) / 3600, 1),
                        "repo_lines_added": sum(c[1] for c in repo), "repo_lines_removed": sum(c[2] for c in repo),
                        "research_home_lines": sum(c[1] + c[2] for c in changes) - sum(c[1] + c[2] for c in repo),
                        "landed": sorted({x for r in mine for x in r.get("landed") or []})}
        full = [r for r in mine if r.get("session") and not r["session"].get("partial")]
        if full:
            hours = sum(r["window_h"] for r in full)
            median = lambda xs: sorted(xs)[len(xs) // 2] if len(xs) % 2 else sum(sorted(xs)[len(xs) // 2 - 1:len(xs) // 2 + 1]) / 2
            totals[kind]["inside_windows"] = {
                "commissions": len(full), "window_h": round(hours, 1),
                "active_share": round(sum(r["session"]["active_h"] for r in full) / hours, 3),
                "median_longest_idle_h": round(median([r["session"]["idle"]["max_gap_h"] for r in full]), 2),
                "median_longest_no_edit_h": round(median([r["session"]["no_change"]["max_gap_h"] for r in full]), 2),
                "idle_stalled_h": round(sum(r["session"]["idle"]["stalled_share"] * r["window_h"] for r in full), 1),
                "no_edit_stalled_h": round(sum(r["session"]["no_change"]["stalled_share"] * r["window_h"] for r in full), 1),
                "with_no_edit_stall": sum(1 for r in full if r["session"]["no_change"]["gaps_over_2h"]),
                "repo_lines_per_window_h": round(sum(r["session"]["repo_lines_added"] + r["session"]["repo_lines_removed"] for r in full) / hours)}
    covered = [(a, b) for a, b in merge(w for r in rows for w in r["_window"])]
    span = [(since or covered[0][0], min(until or home.now, home.now))] if covered else []
    idle = sorted(gaps(covered, span), key=lambda g: g[0] - g[1])
    calendar = {"span_h": round(sum(b - a for a, b in span) / 3600, 1),
                "covered_h": round(sum(b - a for a, b in covered) / 3600, 1),
                "no_commission_gaps": [[iso(a), round((b - a) / 3600, 1)] for a, b in idle[:6]]} if span else None
    topics = [topic_row(home, t, rows, since, until) for t in home.topics]
    return {"generated_at": iso(home.now), "home": home.home, "since": iso(since), "until": iso(until),
            "totals": totals, "calendar": calendar, "commissions": [public(r) for r in rows], "topics": topics,
            "unreadable": sorted(set(UNREADABLE))}


def render_portfolio(data: Dict[str, Any]) -> str:
    out = [f"Research history · {data['home']} · generated {data['generated_at']}"
           + (f" · since {data['since']}" if data["since"] else "") + (f" · until {data['until']}" if data["until"] else ""),
           "", "## Topics", "", TOPIC_HEAD]
    out += [topic_line(t) for t in sorted(data["topics"], key=lambda t: t["last_progress"] or "", reverse=True)]
    out += ["", "## Totals (each rollout counted once)", "", "| kind | commissions | window h | agent-active h | +repo | -repo | research-home lines | landed |", "|---|---|---|---|---|---|---|---|"]
    out += [f"| {k} | {t['commissions_with_window']} | {t['window_h_union']} | {t['agent_active_h_union']} | {t['repo_lines_added']} | "
            f"{t['repo_lines_removed']} | {t['research_home_lines']} | {len(t['landed'])} |" for k, t in data["totals"].items()]
    inside = [(k, t["inside_windows"]) for k, t in data["totals"].items() if t.get("inside_windows")]
    if inside:
        out += ["", "Inside windows, commissions with full session coverage:", "",
                "| kind | commissions | window h | active | median longest idle h | median longest no-edit h | idle stalled h | no-edit stalled h | with no-edit stall | repo lines / h |",
                "|---|---|---|---|---|---|---|---|---|---|"]
        out += [f"| {k} | {i['commissions']} | {i['window_h']} | {pct(i['active_share'])} | {i['median_longest_idle_h']} | {i['median_longest_no_edit_h']} | "
                f"{i['idle_stalled_h']} | {i['no_edit_stalled_h']} | {i['with_no_edit_stall']} | {i['repo_lines_per_window_h']} |" for k, i in inside]
    cal = data.get("calendar")
    if cal:
        out += ["", f"Calendar: some commission window covered {cal['covered_h']} of {cal['span_h']} h; longest spans with none: "
                + ", ".join(f"{h} h from {at}" for at, h in cal["no_commission_gaps"])]
    out += ["", "## Commissions", "", COMMISSION_HEAD]
    out += [commission_line(r) for r in sorted(data["commissions"], key=lambda r: r["start"] or "")]
    shared = [f"{r['id']} ↔ {', '.join(r['shared_session_with'])}" for r in data["commissions"] if r.get("shared_session_with")]
    if shared:
        out += ["", "Same rollouts inside overlapping windows (counted in each row): " + "; ".join(shared)]
    out += ["", LEGEND]
    if data["unreadable"]:
        out += ["", f"Unreadable ({len(data['unreadable'])}): " + "; ".join(data["unreadable"][:5])]
    return "\n".join(out)


def render_commission(r: Dict[str, Any]) -> str:
    out = [COMMISSION_HEAD, commission_line(r), "", f"topics: {', '.join(r['topics']) or '?'} · window {r['start']} → {r['end']}",
           f"sources: {json.dumps(r['sources'])}", f"session links: {', '.join(r['session_links']) or 'none'}"]
    if r.get("session"):
        s = r["session"]
        out.append(f"session: {s['threads']} threads, {s['active_h']} active h, {s['turns_per_day']} turns/day, "
                   f"+{s['lines_added']}/-{s['lines_removed']} all lines, {s['paths_changed']} paths ({', '.join(s['lines_source'])}); "
                   f"longest idle {s['idle']['max_gap_h']} h from {s['idle']['max_gap_at']}; longest without an edit "
                   f"{s['no_change']['max_gap_h']} h from {s['no_change']['max_gap_at']}")
    else:
        out.append(f"session: unknown ({r.get('session_unknown_reason')})")
    out.append(f"landed in window: {', '.join(r['landed']) if r.get('landed') else r.get('landed')}")
    out.append(f"records: {r.get('record_entries')} dated entries, gaps {json.dumps(r.get('record_gaps'))}; operator entries {r.get('operator_entries')}")
    out += ["", LEGEND]
    return "\n".join(out)


def render_topic(t: Dict[str, Any], rows: List[Dict[str, Any]]) -> str:
    out = [f"{t['title']} ({t['topic']})", "", TOPIC_HEAD, topic_line(t), "",
           f"mission now: {json.dumps(t['mission_status'])}", f"landed: {', '.join(t['landed_all']) or 'none'}",
           f"verdict reviews: {', '.join(t['verdict_reviews']) or 'none'} · days since progress: {fmt(t['days_since_progress'])}"]
    if t.get("session"):
        out.append(f"unattributed lines in shared commissions: {t['session']['unattributed_lines_in_shared_commissions']}")
    out += ["", COMMISSION_HEAD] + [commission_line(r) for r in sorted(rows, key=lambda r: r["start"] or "")] + ["", LEGEND]
    return "\n".join(out)


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--home", default=RESEARCH, help="research home (default ~/.vrooli/research)")
    parser.add_argument("--sessions", default=WEB_HOMES, help="Web Console Codex session homes")
    parser.add_argument("--tui-sessions", default=TUI_SESSIONS, help="Codex TUI sessions folder")
    parser.add_argument("--cache", default=CACHE, help="scan cache file ('' disables)")
    parser.add_argument("--now", help="treat this time as now (for frozen baselines and tests)")
    sub = parser.add_subparsers(dest="command", required=True)
    for name in ("portfolio", "topic", "commission"):
        p = sub.add_parser(name)
        if name != "portfolio":
            p.add_argument("name")
        p.add_argument("--json", action="store_true")
        p.add_argument("--since", help="window start, YYYY-MM-DD or a UTC time")
        p.add_argument("--until", help="window end (a bare date includes that day)")
    args = parser.parse_args(argv)
    since, until = bound(args.since), bound(args.until, end=True)
    sessions = Sessions(args.sessions, args.tui_sessions, args.cache or None)
    home = Home(args.home, sessions, bound(args.now))
    try:
        if args.command == "portfolio":
            data = portfolio(home, since, until)
            print(json.dumps(data, indent=1) if args.json else render_portfolio(data))
        elif args.command == "commission":
            row = home.commission(args.name, since, until)
            print(json.dumps(public(row), indent=1) if args.json else render_commission(row))
        else:
            topic = home.real(args.name)
            if not topic:
                raise SystemExit(f"no topic {args.name} under {home.home}/topics")
            rows = [home.commission(c, since, until, only_topic=topic) for c in home.cids]
            rows = [r for r in rows if r and (r["_window"] or r["start"] is None or r["end"] is None)]
            data = topic_row(home, topic, rows, since, until)
            if args.json:
                print(json.dumps(dict(data, commission_rows=[public(r) for r in rows], unreadable=sorted(set(UNREADABLE))), indent=1))
            else:
                print(render_topic(data, rows))
    finally:
        sessions.save()
    return 0


if __name__ == "__main__":
    sys.exit(main())
