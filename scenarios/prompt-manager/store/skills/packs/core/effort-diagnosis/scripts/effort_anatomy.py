#!/usr/bin/env python3
"""Measure the anatomy of long agent efforts from native transcripts.

Read-only. Streams Codex rollout JSONL files, Codex goals databases and Claude Code
session transcripts, and reports what the effort-diagnosis skill needs: when the goal
was injected or dropped, compactions, sub-agent trees, worker lifetime, inherited
context, changed lines and the narration that narrows or stops work.

Subcommands:
  scan      per-thread metrics for every Codex rollout under the roots (JSONL out)
  summary   lifetime, depth and per-day tables from a scan file
  tree      one root thread and its descendants from a scan file
  timeline  chronological events of one Codex rollout
  goals     native goal corpus from goals_1.sqlite files
  claude    metrics and timeline of one Claude Code session and its subagents

Run `effort_anatomy.py <subcommand> --help` for options. Python 3.9+, stdlib only.
"""
from __future__ import annotations

import argparse
import collections
import datetime as dt
import glob
import json
import os
import re
import signal
import sqlite3
import statistics
import sys
from typing import Any, Dict, Iterable, Iterator, List, Optional, Tuple

HOME = os.path.expanduser("~")
DEFAULT_CODEX_ROOTS = [
    os.path.join(HOME, ".vrooli/state/vrooli/web-console/sessions/codex/*/sessions"),
    os.path.join(HOME, ".codex/sessions"),
]
GOAL_MARKER = 'source=\\"goal\\"'  # as it appears inside a JSON-encoded line
GOAL_MARKER_RAW = 'source="goal"'
OMITTED_MARKER = "objective omitted"
RESEARCH_PATH = os.path.join(HOME, ".vrooli/research")
REVIEW_RE = re.compile(r"review|audit|critic|challenge", re.I)
ACTIVE_GAP_SECONDS = 600  # gaps longer than this count as idle, not work

# Narration families that mark narrowing, stopping or waiting. Counts are signals for a
# human or agent to read in context, not verdicts.
NARRATION_FAMILIES: Dict[str, re.Pattern] = {
    "narrowing": re.compile(r"\b(bounded|narrow(ed|ing)?|small(est)? (probe|slice|change|fix)|first slice|minimal)\b", re.I),
    "threshold": re.compile(r"\b(evidence threshold|not enough evidence|signal is too weak|insufficient evidence|held pending)\b", re.I),
    "no_change": re.compile(r"\b(no code|no change|no-change|already (addressed|fixed)|not reproducible|historical|no defect)\b", re.I),
    "stop": re.compile(r"\b(i'?ll stop|stop here|closing|closure|close the|wrap(ping)? up|final report)\b", re.I),
    "blocked": re.compile(r"\b(blocked|needs? (operator|human|approval)|awaiting (approval|decision))\b", re.I),
    "shared_tree": re.compile(r"\b(dirty|uncommitted|other agents?'?|shared write|leave (those|these|that) (files )?alone|owner files)\b", re.I),
    "waiting": re.compile(r"\b(waiting for|wait until|next (refresh|window)|poll(ing)?)\b", re.I),
    "infra": re.compile(r"\b(test genie|admission|unavailable|degraded|not authoritative|start-failed|saturat)\b", re.I),
}


# ---------------------------------------------------------------- helpers

def parse_ts(value: Optional[str]) -> Optional[dt.datetime]:
    if not value:
        return None
    try:
        return dt.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def hours_between(start: Optional[str], end: Optional[str]) -> float:
    a, b = parse_ts(start), parse_ts(end)
    if not a or not b:
        return 0.0
    return max(0.0, (b - a).total_seconds() / 3600.0)


def count_diff_lines(diff: str) -> Tuple[int, int]:
    added = removed = 0
    for line in diff.split("\n"):
        if line.startswith("+") and not line.startswith("+++"):
            added += 1
        elif line.startswith("-") and not line.startswith("---"):
            removed += 1
    return added, removed


PATCH_FILE_RE = re.compile(r"^\*\*\* (?:Update|Add|Delete) File: (.+)$")


def count_patch_text(text: str) -> Tuple[int, int, List[str]]:
    """Count +/- lines and files in apply_patch text (older rollouts carry edits this way)."""
    added = removed = 0
    files: List[str] = []
    inside = False
    for line in text.split("\n"):
        if line.startswith("*** Begin Patch"):
            inside = True
            continue
        if line.startswith("*** End Patch"):
            inside = False
            continue
        if not inside:
            continue
        match = PATCH_FILE_RE.match(line)
        if match:
            files.append(match.group(1).strip())
        elif line.startswith("+"):
            added += 1
        elif line.startswith("-"):
            removed += 1
    return added, removed, files


def iter_lines(path: str) -> Iterator[str]:
    with open(path, "r", encoding="utf-8", errors="replace") as handle:
        for line in handle:
            yield line


def loads(line: str) -> Optional[dict]:
    try:
        value = json.loads(line)
    except (ValueError, RecursionError):
        return None
    return value if isinstance(value, dict) else None


def message_text(payload: dict) -> str:
    parts = []
    for item in payload.get("content") or []:
        if isinstance(item, dict) and isinstance(item.get("text"), str):
            parts.append(item["text"])
    return "\n".join(parts)


def is_operator_text(text: str) -> bool:
    head = text.lstrip()[:200]
    if GOAL_MARKER_RAW in text[:400]:
        return False
    return not (head.startswith("<environment_context") or "AGENTS.md instructions" in head
                or head.startswith("<user_instructions") or head.startswith("<codex_internal"))


def retained_summary(line: str, stance_marker: Optional[str] = None) -> Optional[Dict[str, int]]:
    """Count what a `compacted` record kept in `retained_context`; None for older records without it.

    Operator messages exclude goal events and Agent Manager wakes: those count toward Codex's
    eight retained user messages but carry no stance. `stance` counts operator messages that
    contain `stance_marker`, or every operator message when no marker is given.
    """
    data = loads(line) or {}
    retained = (data.get("payload") or {}).get("retained_context")
    if not isinstance(retained, dict):
        return None
    messages = [m for m in retained.get("user_messages") or [] if isinstance(m, dict)]
    operator = [t for t in (m.get("text") or "" for m in messages) if is_operator_text(t) and not t.lstrip().startswith("[wake")]
    return {
        "user": len(messages),
        "truncated": sum(1 for m in messages if m.get("complete") is False),
        "operator": len(operator),
        "stance": sum(1 for t in operator if not stance_marker or stance_marker in t),
    }


GOAL_STATUS_RE = re.compile(r'^\s*<codex_internal_context source="user_goal">\s*User set goal status: "(\w+)"')
GOAL_TOOL_STATUS_RE = re.compile(r"""update_goal\(\{\s*status:\s*["'](\w+)["']""")


def goal_status_change(line: str) -> Optional[str]:
    """Return the native goal's new status when this line changes it, else None.

    Three sources: the operator's goal control (a user message that starts with the
    `user_goal` context, and a `thread_goal_updated` event), the agent's own `update_goal`
    call (which writes no goal event), and `create_goal` (active). Tool output that merely
    quotes these strings, such as a doc the agent printed, is not a status change.
    """
    if "goal" not in line:
        return None
    head = line[:400]
    if '"type":"thread_goal_updated"' in head:
        match = re.search(r'"status":"(\w+)"', line)
        return match.group(1) if match else None
    if '"type":"response_item"' not in head:
        return None
    if "user_goal" in line and '"role":"user"' in line:
        payload = (loads(line) or {}).get("payload") or {}
        if payload.get("type") == "message" and payload.get("role") == "user":
            match = GOAL_STATUS_RE.search(message_text(payload))
            return match.group(1) if match else None
        return None
    if '"type":"custom_tool_call"' in head or '"type":"function_call"' in head:
        payload = (loads(line) or {}).get("payload") or {}
        text = payload.get("input") if isinstance(payload.get("input"), str) else str(payload.get("arguments") or "")
        match = GOAL_TOOL_STATUS_RE.search(text)
        if match:
            return match.group(1)
        if "create_goal(" in text:
            return "active"
    return None


def parse_call(line: str) -> Tuple[Optional[str], Dict[str, Any]]:
    """Return (tool name, decoded arguments) for a response_item function_call line."""
    data = loads(line)
    payload = (data or {}).get("payload") or {}
    if payload.get("type") != "function_call":
        return None, {}
    arguments = payload.get("arguments")
    if isinstance(arguments, str):
        try:
            arguments = json.loads(arguments)
        except ValueError:
            arguments = {}
    return payload.get("name"), arguments if isinstance(arguments, dict) else {}


def quantile(values: List[float], q: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    return ordered[min(len(ordered) - 1, int(q * (len(ordered) - 1) + 0.5))]


def discover_rollouts(roots: Iterable[str], since: Optional[str], until: Optional[str]) -> List[str]:
    files: List[str] = []
    for pattern in roots:
        for root in glob.glob(os.path.expanduser(pattern)):
            for dirpath, _dirs, names in os.walk(root):
                for name in names:
                    if name.startswith("rollout-") and name.endswith(".jsonl"):
                        files.append(os.path.join(dirpath, name))
    if since or until:
        kept = []
        for path in files:
            day = _day_from_path(path)
            if day and since and day < since:
                continue
            if day and until and day > until:
                continue
            kept.append(path)
        files = kept
    return sorted(set(files))


def _day_from_path(path: str) -> Optional[str]:
    match = re.search(r"rollout-(\d{4}-\d{2}-\d{2})T", os.path.basename(path))
    return match.group(1) if match else None


# ---------------------------------------------------------------- codex thread metrics

def read_session_meta(line: str) -> Dict[str, Any]:
    """Pull identity fields from a session_meta first line without parsing huge payloads."""
    head = line[:4_000_000]

    def grab(pattern: str) -> Optional[str]:
        match = re.search(pattern, head)
        return match.group(1) if match else None

    depth = grab(r'"depth":(\d+)')
    return {
        "id": grab(r'"payload":\{[^{}]*?"id":"([^"]+)"') or grab(r'"id":"([^"]+)"'),
        "parent": grab(r'"parent_thread_id":"([^"]+)"'),
        "depth": int(depth) if depth else 0,
        "path": grab(r'"agent_path":"([^"]*)"'),
        "role": grab(r'"agent_role":"([^"]*)"'),
        "cli_version": grab(r'"cli_version":"([^"]+)"'),
        "multi_agent_version": grab(r'"multi_agent_version":"([^"]+)"'),
        "history_start": int(grab(r'"subagent_history_start_ordinal":(\d+)') or 0),
    }


def scan_rollout(path: str) -> Dict[str, Any]:
    """Compute one thread's metrics in a single streaming pass."""
    record: Dict[str, Any] = {
        "file": path, "id": None, "parent": None, "depth": 0, "path": None, "role": None,
        "cli_version": None, "multi_agent_version": None, "model": None, "effort": None,
        "start": None, "end": None, "hours": 0.0, "active_hours": 0.0, "context_window": None,
        "first_input_tokens": None, "max_input_tokens": 0, "compactions": 0,
        "goal_injections": 0, "objective_omitted": 0, "file_changes": 0,
        "lines_added": 0, "lines_removed": 0, "research_lines": 0, "files_touched": 0,
        "spawns": 0, "spawn_fork_all": 0, "spawn_fork_none": 0, "followups": 0,
        "messages_sent": 0, "operator_messages": 0, "turns": 0, "turn_aborts": 0,
        "size_bytes": os.path.getsize(path),
    }
    files = set()
    patch_files = set()
    patch_added = patch_removed = 0
    history_start = 0
    active_seconds = 0.0
    previous_ts: Optional[dt.datetime] = None
    try:
        for index, line in enumerate(iter_lines(path)):
            if index == 0 and '"session_meta"' in line[:200]:
                meta = read_session_meta(line)
                record.update({k: v for k, v in meta.items() if k != "history_start"})
                history_start = meta["history_start"]
            ts_match = re.match(r'\{"timestamp":"([^"]+)"', line)
            if ts_match:
                record["start"] = record["start"] or ts_match.group(1)
                record["end"] = ts_match.group(1)
            inherited = False
            if history_start:
                ord_match = re.search(r'"ordinal":(\d+)', line[:200])
                inherited = bool(ord_match and int(ord_match.group(1)) < history_start)
            if ts_match and not inherited:
                current = parse_ts(ts_match.group(1))
                if current and previous_ts:
                    gap = (current - previous_ts).total_seconds()
                    if 0 < gap <= ACTIVE_GAP_SECONDS:
                        active_seconds += gap
                previous_ts = current or previous_ts
            if '"type":"token_count"' in line:
                match = re.search(r'"last_token_usage":\{"input_tokens":(\d+)', line)
                if match and not inherited:
                    value = int(match.group(1))
                    record["max_input_tokens"] = max(record["max_input_tokens"], value)
                    if record["first_input_tokens"] is None:
                        record["first_input_tokens"] = value
                window = re.search(r'"model_context_window":(\d+)', line)
                if window:
                    record["context_window"] = int(window.group(1))
                continue
            if inherited:
                continue
            if '"type":"turn_context"' in line[:300] or '"thread_settings_applied"' in line[:300]:
                model = re.search(r'"model":"([^"]+)"', line)
                effort = re.search(r'"(?:reasoning_effort|effort)":"([^"]+)"', line)
                if model and not record["model"]:
                    record["model"] = model.group(1)
                if effort and not record["effort"]:
                    record["effort"] = effort.group(1)
                continue
            if '"item":{"type":"ContextCompaction"' in line and '"item_completed"' in line:
                record["compactions"] += 1
                continue
            if '"type":"compacted"' in line[:200]:
                if OMITTED_MARKER in line:
                    record["objective_omitted"] += 1
                continue
            if '"type":"task_started"' in line:
                record["turns"] += 1
                continue
            if '"type":"turn_aborted"' in line:
                record["turn_aborts"] += 1
                continue
            if '"item":{"type":"FileChange"' in line and '"item_completed"' in line:
                data = loads(line)
                if not data:
                    continue
                changes = ((data.get("payload") or {}).get("item") or {}).get("changes") or {}
                record["file_changes"] += 1
                for changed_path, change in changes.items():
                    files.add(changed_path)
                    if not isinstance(change, dict):
                        continue
                    diff = change.get("unified_diff") or ""
                    if diff:
                        added, removed = count_diff_lines(diff)
                    else:
                        content = change.get("content") or ""
                        added, removed = (content.count("\n") + (1 if content else 0)), 0
                    record["lines_added"] += added
                    record["lines_removed"] += removed
                    if changed_path.startswith(RESEARCH_PATH):
                        record["research_lines"] += added + removed
                continue
            if "*** Begin Patch" in line and ('"type":"function_call"' in line[:400] or '"type":"custom_tool_call"' in line[:400]):
                data = loads(line)
                payload = (data or {}).get("payload") or {}
                text = payload.get("input") if isinstance(payload.get("input"), str) else ""
                if not text and isinstance(payload.get("arguments"), str):
                    try:
                        args_value = json.loads(payload["arguments"])
                    except ValueError:
                        args_value = payload["arguments"]
                    text = json.dumps(args_value) if not isinstance(args_value, str) else args_value
                    text = text.replace("\\n", "\n")
                added, removed, touched = count_patch_text(text)
                patch_added += added
                patch_removed += removed
                patch_files.update(touched)
            if '"type":"function_call"' in line[:400]:
                name, call = parse_call(line)
                if name == "spawn_agent":
                    record["spawns"] += 1
                    if call.get("fork_turns") == "none":
                        record["spawn_fork_none"] += 1
                    else:
                        record["spawn_fork_all"] += 1  # Codex defaults fork_turns to "all"
                elif name == "followup_task":
                    record["followups"] += 1
                elif name == "send_message":
                    record["messages_sent"] += 1
                continue
            if '"role":"user"' in line and '"type":"response_item"' in line[:200]:
                if GOAL_MARKER in line:
                    record["goal_injections"] += 1
                    continue
                if record["depth"] == 0:
                    data = loads(line)
                    if data and is_operator_text(message_text(data.get("payload") or {})):
                        record["operator_messages"] += 1
    except OSError as exc:
        record["error"] = str(exc)
    if not record["file_changes"] and (patch_added or patch_removed):
        # Older rollouts have no FileChange items; fall back to the patch text itself.
        record["lines_added"], record["lines_removed"] = patch_added, patch_removed
        files = patch_files
        record["lines_source"] = "apply_patch"
    else:
        record["lines_source"] = "file_change"
    record["files_touched"] = len(files)
    record["hours"] = round(hours_between(record["start"], record["end"]), 3)
    record["active_hours"] = round(active_seconds / 3600.0, 3)
    return record


def cmd_scan(args: argparse.Namespace) -> int:
    roots = args.root or DEFAULT_CODEX_ROOTS
    files = discover_rollouts(roots, args.since, args.until)
    if not files:
        print("no rollout files found under: " + ", ".join(roots), file=sys.stderr)
        return 1
    out = open(args.out, "w", encoding="utf-8") if args.out else sys.stdout
    try:
        for count, path in enumerate(files, 1):
            out.write(json.dumps(scan_rollout(path)) + "\n")
            if args.out and count % 100 == 0:
                print(f"scanned {count}/{len(files)}", file=sys.stderr)
    finally:
        if args.out:
            out.close()
    if args.out:
        print(f"wrote {len(files)} thread records to {args.out}", file=sys.stderr)
    return 0


SUMMED_FIELDS = ("active_hours", "compactions", "goal_injections", "objective_omitted", "file_changes",
                 "lines_added", "lines_removed", "research_lines", "files_touched", "spawns", "spawn_fork_all",
                 "spawn_fork_none", "followups", "messages_sent", "operator_messages", "turns", "turn_aborts", "size_bytes")


def merge_threads(rows: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
    """Merge rows that share a thread id. Codex continues a thread in a second rollout
    file (`rollout-<time>-<thread>_<id>.jsonl`) with the same session_meta id; counting
    each file as a thread double-counts roots and splits their work."""
    merged: Dict[str, Dict[str, Any]] = {}
    out: List[Dict[str, Any]] = []
    for row in sorted(rows, key=lambda r: (r.get("start") or "", r.get("file") or "")):
        key = row.get("id")
        if not key:
            out.append(row)
            continue
        if key not in merged:
            first = dict(row)
            first["files"] = [row.get("file")]
            merged[key] = first
            out.append(first)
            continue
        target = merged[key]
        target["files"].append(row.get("file"))
        for field in SUMMED_FIELDS:
            target[field] = (target.get(field) or 0) + (row.get(field) or 0)
        target["max_input_tokens"] = max(target.get("max_input_tokens") or 0, row.get("max_input_tokens") or 0)
        for field in ("parent", "path", "role", "model", "effort", "cli_version", "multi_agent_version", "context_window"):
            target[field] = target.get(field) or row.get(field)
        target["depth"] = max(target.get("depth") or 0, row.get("depth") or 0)
        ends = [e for e in (target.get("end"), row.get("end")) if e]
        target["end"] = max(ends) if ends else None
        target["hours"] = round(hours_between(target.get("start"), target.get("end")), 3)
        if row.get("lines_source") == "apply_patch" and target.get("lines_source") != "file_change":
            target["lines_source"] = "apply_patch"
    return out


def load_scan(path: str) -> List[Dict[str, Any]]:
    """Read a scan file, one row per thread (files of one thread are merged)."""
    rows = []
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return merge_threads(rows)


def thread_files(path: str) -> List[str]:
    """Every rollout file of the thread `path` belongs to, oldest first."""
    try:
        first = next(iter_lines(path), "")
    except OSError:
        return [path]
    thread = read_session_meta(first).get("id") if '"session_meta"' in first[:200] else None
    if not thread:
        return [path]
    # Rollouts live under <home>/sessions/YYYY/MM/DD/; a continuation may be on a later day.
    day_dir = os.path.dirname(os.path.abspath(path))
    sessions = os.path.dirname(os.path.dirname(os.path.dirname(day_dir)))
    if os.path.basename(sessions) == "sessions":
        pattern = os.path.join(sessions, "*", "*", "*", f"rollout-*{thread}*.jsonl")
    else:
        pattern = os.path.join(day_dir, f"rollout-*{thread}*.jsonl")
    files = []
    for candidate in sorted(glob.glob(pattern)):
        head = next(iter_lines(candidate), "")
        if '"session_meta"' in head[:200] and read_session_meta(head).get("id") == thread:
            files.append(candidate)
    return files or [path]


def changed(row: Dict[str, Any]) -> int:
    return int(row.get("lines_added", 0)) + int(row.get("lines_removed", 0))


def is_review(row: Dict[str, Any]) -> bool:
    return bool(REVIEW_RE.search(row.get("path") or ""))


LIFETIME_BUCKETS = [(0, 0.1, "<6m"), (0.1, 0.5, "6-30m"), (0.5, 2, "30m-2h"), (2, 8, "2-8h"), (8, 1e9, ">8h")]


def lifetime(row: Dict[str, Any]) -> float:
    """Active hours when the scan recorded them, else the wall-clock span."""
    value = row.get("active_hours")
    return float(value) if value is not None else float(row.get("hours", 0))


def lifetime_table(rows: List[Dict[str, Any]]) -> List[str]:
    lines = ["active lifetime  workers  median_lines  p90_lines  share>=1000  (lines = added + removed)"]
    for low, high, label in LIFETIME_BUCKETS:
        group = [changed(r) for r in rows if low <= lifetime(r) < high]
        if not group:
            continue
        share = sum(1 for v in group if v >= 1000) / len(group)
        lines.append(f"{label:<16} {len(group):>7}  {quantile(group, .5):>12.0f}  {quantile(group, .9):>9.0f}  {share:>10.0%}")
    return lines


def cmd_summary(args: argparse.Namespace) -> int:
    rows = load_scan(args.scan)
    window = None
    for row in rows:
        window = window or row.get("context_window")
    print(f"threads: {len(rows)}  context window seen: {window or 'unknown'}")
    by_depth = collections.defaultdict(list)
    for row in rows:
        by_depth[row.get("depth", 0)].append(row)
    print("\n## By depth")
    for depth in sorted(by_depth):
        group = by_depth[depth]
        first = [r["first_input_tokens"] for r in group if r.get("first_input_tokens")]
        heavy = sum(1 for v in first if v > 100_000) / len(first) if first else 0
        print(f"depth {depth}: n={len(group)} lines={sum(changed(r) for r in group)} "
              f"median_hours={quantile([r.get('hours', 0) for r in group], .5):.2f} "
              f"first_input_p50={quantile(first, .5):.0f} inherited>100k={heavy:.0%} "
              f"compacted={sum(1 for r in group if r.get('compactions'))/len(group):.0%}")
    workers = [r for r in rows if r.get("depth", 0) >= 1 and not is_review(r)]
    if workers:
        print("\n## Non-review sub-agents by lifetime")
        print("\n".join(lifetime_table(workers)))
    print("\n## Per day (sub-agents started that day)")
    print("day         workers  median_min  lines   inherited>100k  roots  root_lines")
    days = collections.defaultdict(lambda: {"w": [], "r": []})
    for row in rows:
        if row.get("start"):
            days[row["start"][:10]]["w" if row.get("depth", 0) else "r"].append(row)
    for day in sorted(days):
        w, r = days[day]["w"], days[day]["r"]
        heavy = sum(1 for x in w if (x.get("first_input_tokens") or 0) > 100_000)
        print(f"{day}  {len(w):>7}  {quantile([x.get('hours', 0) * 60 for x in w], .5):>10.1f}  "
              f"{sum(changed(x) for x in w):>6}  {heavy:>14}  {len(r):>5}  {sum(changed(x) for x in r):>10}")
    print(f"\n## Top {args.top} threads by changed lines")
    for row in sorted(rows, key=changed, reverse=True)[: args.top]:
        print(f"{changed(row):>7} lines  depth={row.get('depth')} hours={row.get('hours'):.1f} "
              f"compactions={row.get('compactions')} model={row.get('model')} "
              f"path={row.get('path') or '(root)'} file={row['file']}")
    return 0


def cmd_tree(args: argparse.Namespace) -> int:
    rows = load_scan(args.scan)
    by_id = {r["id"]: r for r in rows if r.get("id")}
    root = by_id.get(args.thread) or next((r for r in rows if args.thread in r["file"]), None)
    if not root:
        print(f"thread {args.thread} not found in {args.scan}", file=sys.stderr)
        return 1
    children = collections.defaultdict(list)
    for row in rows:
        if row.get("parent"):
            children[row["parent"]].append(row)

    def walk(node: Dict[str, Any], indent: int) -> Iterator[Tuple[int, Dict[str, Any]]]:
        yield indent, node
        for child in sorted(children.get(node["id"], []), key=lambda r: r.get("start") or ""):
            yield from walk(child, indent + 1)

    descendants = [n for _i, n in walk(root, 0)][1:]
    print(f"root {root['id']} model={root.get('model')} hours={root.get('hours'):.1f} "
          f"compactions={root.get('compactions')} goal_injections={root.get('goal_injections')} "
          f"objective_omitted={root.get('objective_omitted')} operator_messages={root.get('operator_messages')} "
          f"spawns={root.get('spawns')} (fork all {root.get('spawn_fork_all')}, none {root.get('spawn_fork_none')}) "
          f"followups={root.get('followups')} root_lines={changed(root)}")
    if descendants:
        minutes = [lifetime(d) * 60 for d in descendants]
        print(f"descendants={len(descendants)} median_active_minutes={quantile(minutes, .5):.1f} "
              f"lines={sum(changed(d) for d in descendants)} zero_change={sum(1 for d in descendants if not changed(d))} "
              f"max_depth={max(d.get('depth', 0) for d in descendants)}")
        print("\n".join(lifetime_table([d for d in descendants if not is_review(d)])))
    if args.list:
        for indent, node in walk(root, 0):
            print(f"{'  ' * indent}{node.get('path') or '(root)'}  {(node.get('start') or '')[:16]}  "
                  f"{lifetime(node) * 60:.0f}m active  lines={changed(node)}  first_in={node.get('first_input_tokens')}  "
                  f"compactions={node.get('compactions')}")
    return 0


# ---------------------------------------------------------------- codex timeline

def timeline_events(path: str, narration_limit: int) -> Tuple[List[Tuple[str, str, str]], Dict[str, Any]]:
    events: List[Tuple[str, str, str]] = []
    hourly = collections.defaultdict(lambda: [0, 0])
    families = collections.Counter()
    family_examples: Dict[str, List[str]] = collections.defaultdict(list)
    for line in iter_lines(path):
        ts_match = re.match(r'\{"timestamp":"([^"]+)"', line)
        ts = ts_match.group(1)[:19] if ts_match else "?"
        if '"type":"response_item"' in line[:200] and '"role":"user"' in line:
            data = loads(line)
            if not data:
                continue
            text = message_text(data.get("payload") or {})
            if GOAL_MARKER_RAW in text:
                objective = re.search(r"<objective>\s*(.*?)\s*</objective>", text, re.S)
                length = len(objective.group(1)) if objective else len(text)
                events.append((ts, "goal", f"objective injected ({length} chars)"))
            elif is_operator_text(text):
                events.append((ts, "operator", text.strip().replace("\n", " ")[:240]))
        elif '"type":"compacted"' in line[:200]:
            note = "objective omitted" if OMITTED_MARKER in line else "objective retained or absent"
            kept = retained_summary(line)
            if kept is not None:
                note += f"; kept {kept['user']} user messages ({kept['operator']} operator, {kept['truncated']} truncated)"
            events.append((ts, "compaction", note))
        elif '"type":"task_started"' in line:
            trigger = re.search(r'"turn_trigger":"(\w+)"', line)
            events.append((ts, "turn_start", trigger.group(1) if trigger else ""))
        elif '"type":"task_complete"' in line:
            events.append((ts, "turn_end", "complete"))
        elif '"type":"turn_aborted"' in line:
            reason = re.search(r'"reason":"(\w+)"', line)
            events.append((ts, "turn_end", "aborted " + (reason.group(1) if reason else "")))
        elif '"type":"function_call"' in line[:400]:
            name, call = parse_call(line)
            if name == "spawn_agent":
                detail = f"{call.get('task_name', '?')} fork={call.get('fork_turns') or 'all(default)'}"
                for key in ("model", "reasoning_effort"):
                    if call.get(key):
                        detail += f" {key}={call[key]}"
                events.append((ts, name, detail))
            elif name == "followup_task":
                events.append((ts, name, str(call.get("target") or call.get("task_name") or "?")))
            elif name and "goal" in name:
                events.append((ts, name, json.dumps(call)[:200]))
        elif '"item":{"type":"FileChange"' in line and '"item_completed"' in line:
            data = loads(line)
            if not data:
                continue
            changes = ((data.get("payload") or {}).get("item") or {}).get("changes") or {}
            for change in changes.values():
                if isinstance(change, dict):
                    added, removed = count_diff_lines(change.get("unified_diff") or "")
                    hourly[ts[:13]][0] += added
                    hourly[ts[:13]][1] += removed
        elif '"item":{"type":"AgentMessage"' in line and '"item_completed"' in line:
            data = loads(line)
            if not data:
                continue
            item = (data.get("payload") or {}).get("item") or {}
            text = " ".join(c.get("text", "") for c in item.get("content") or [] if isinstance(c, dict))
            for family, pattern in NARRATION_FAMILIES.items():
                if pattern.search(text):
                    families[family] += 1
                    if len(family_examples[family]) < narration_limit:
                        family_examples[family].append(f"{ts} {text.strip()[:220]}")
    stats = {"hourly": dict(hourly), "families": dict(families), "examples": dict(family_examples)}
    return events, stats


def cmd_timeline(args: argparse.Namespace) -> int:
    events, stats = timeline_events(args.rollout, args.examples)
    kinds = set(args.only.split(",")) if args.only else None
    print("## Events (UTC)")
    for ts, kind, detail in events:
        if kinds and kind not in kinds:
            continue
        print(f"{ts}  {kind:<12} {detail}")
    print("\n## Changed lines per hour (added/removed, FileChange items only)")
    for hour in sorted(stats["hourly"]):
        added, removed = stats["hourly"][hour]
        print(f"{hour}:00  +{added} -{removed}")
    print("\n## Narration families (counts of messages that match; read the examples in context)")
    for family, count in sorted(stats["families"].items(), key=lambda kv: -kv[1]):
        print(f"{family:<12} {count}")
        for example in stats["examples"].get(family, []):
            print(f"    {example}")
    return 0


# ---------------------------------------------------------------- drift tripwires

DRIFT_EXIT = 3  # exit status when any tripwire fires, so a scheduler can branch on it
OWNER_MAX_HOURS = 6.0  # healthy window for one long-lived sub-agent (LONG_EFFORT_DESIGN.md §3)
OWNER_MAX_COMPACTIONS = 8


def turn_units(path: str, stance_marker: Optional[str] = None) -> List[Dict[str, Any]]:
    """Split one rollout's own activity into turns, excluding inherited history."""
    units: List[Dict[str, Any]] = []
    current: Optional[Dict[str, Any]] = None
    history_start = 0
    is_child = False  # a child keeps no user messages; its brief is an encrypted agent_message
    previous_ts: Optional[dt.datetime] = None
    goal_paused = False  # the native goal's last known status; open_unit records it per turn

    def open_unit(ts: Optional[str], trigger: str) -> Dict[str, Any]:
        return {"start": ts, "end": ts, "trigger": trigger, "active_seconds": 0.0, "lines": 0,
                "compactions": 0, "objective_omitted": 0, "stance_lost": 0, "minutes_after_omission": 0.0,
                "goal_injected": False, "patch_lines": 0, "spawns": 0, "spawn_fork_all": 0, "families": collections.Counter(),
                "waits": 0, "wait_timeouts_ms": [], "dispatches": 0, "goal_paused": goal_paused, "_omitted_at": None}

    for index, line in enumerate(iter_lines(path)):
        if index == 0 and '"session_meta"' in line[:200]:
            meta = read_session_meta(line)
            history_start = meta["history_start"]
            is_child = bool(meta["parent"])
            continue
        if history_start:
            ord_match = re.search(r'"ordinal":(\d+)', line[:200])
            if ord_match and int(ord_match.group(1)) < history_start:
                continue
        ts_match = re.match(r'\{"timestamp":"([^"]+)"', line)
        ts = ts_match.group(1) if ts_match else None
        now = parse_ts(ts)
        status = goal_status_change(line)
        if status is not None:
            goal_paused = status == "paused"
        if '"type":"task_started"' in line:
            trigger = re.search(r'"turn_trigger":"(\w+)"', line)
            current = open_unit(ts, trigger.group(1) if trigger else "")
            units.append(current)
            previous_ts = now
            continue
        if current is None:
            current = open_unit(ts, "")
            units.append(current)
        if ts:
            current["end"] = ts
        if now and previous_ts:
            gap = (now - previous_ts).total_seconds()
            if 0 < gap <= ACTIVE_GAP_SECONDS:
                current["active_seconds"] += gap
                if current["_omitted_at"] is not None:
                    current["minutes_after_omission"] += gap / 60.0
        previous_ts = now or previous_ts
        if '"type":"task_complete"' in line or '"type":"turn_aborted"' in line:
            current = None
            continue
        if '"item":{"type":"ContextCompaction"' in line and '"item_completed"' in line:
            current["compactions"] += 1
        elif '"type":"compacted"' in line[:200]:
            if OMITTED_MARKER in line:
                current["objective_omitted"] += 1
                current["_omitted_at"] = ts
            kept = None if is_child else retained_summary(line, stance_marker)
            if kept is not None and kept["stance"] == 0:
                current["stance_lost"] += 1
        elif '"role":"user"' in line and '"type":"response_item"' in line[:200] and GOAL_MARKER in line:
            current["goal_injected"] = True
            current["_omitted_at"] = None
        elif '"item":{"type":"FileChange"' in line and '"item_completed"' in line:
            data = loads(line) or {}
            for change in (((data.get("payload") or {}).get("item") or {}).get("changes") or {}).values():
                if isinstance(change, dict):
                    added, removed = count_diff_lines(change.get("unified_diff") or "")
                    current["lines"] += added + removed
        elif "*** Begin Patch" in line and ('"type":"function_call"' in line[:400] or '"type":"custom_tool_call"' in line[:400]):
            data = loads(line) or {}
            payload = data.get("payload") or {}
            text = payload.get("input") if isinstance(payload.get("input"), str) else str(payload.get("arguments") or "")
            added, removed, _files = count_patch_text(text.replace("\\n", "\n"))
            current["patch_lines"] += added + removed
        elif '"type":"function_call"' in line[:400]:
            name, call = parse_call(line)
            if name == "spawn_agent":
                current["spawns"] += 1
                current["dispatches"] += 1
                if call.get("fork_turns") != "none":
                    current["spawn_fork_all"] += 1
            elif name in ("followup_task", "send_message"):
                current["dispatches"] += 1
            elif name == "wait_agent":
                current["waits"] += 1
                if isinstance(call.get("timeout_ms"), (int, float)):
                    current["wait_timeouts_ms"].append(call["timeout_ms"])
        elif '"item":{"type":"AgentMessage"' in line and '"item_completed"' in line:
            data = loads(line) or {}
            item = (data.get("payload") or {}).get("item") or {}
            text = " ".join(c.get("text", "") for c in item.get("content") or [] if isinstance(c, dict))
            for family, pattern in NARRATION_FAMILIES.items():
                if pattern.search(text):
                    current["families"][family] += 1
    # Older rollouts carry edits only as apply_patch text (same rule as scan_rollout).
    if not any(u["lines"] for u in units):
        for unit in units:
            unit["lines"] = unit["patch_lines"]
    for unit in units:
        unit.pop("patch_lines", None)
        unit.pop("_omitted_at", None)
        unit["families"] = dict(unit["families"])
        unit["wait_timeouts_ms"] = list(unit["wait_timeouts_ms"])
        unit["hours"] = round(hours_between(unit["start"], unit["end"]), 3)
        unit["active_minutes"] = round(unit.pop("active_seconds") / 60.0, 1)
        unit["minutes_after_omission"] = round(unit["minutes_after_omission"], 1)
    return units


def attribute_children(units: List[Dict[str, Any]], children: List[Dict[str, Any]]) -> None:
    """Credit each child's changed lines to the parent turn in which it started."""
    for unit in units:
        unit["child_lines"] = 0
        unit["children"] = 0
    for child in children:
        start = child.get("start") or ""
        for unit in units:
            if unit["start"] and unit["end"] and unit["start"] <= start <= unit["end"]:
                unit["child_lines"] += changed(child)
                unit["children"] += 1
                break


def drift_findings(units: List[Dict[str, Any]], children: List[Dict[str, Any]], max_turn_hours: float, is_child: bool = False) -> List[Dict[str, str]]:
    findings: List[Dict[str, str]] = []
    work = [u for u in units if u["active_minutes"] >= 5]

    def total(unit: Dict[str, Any]) -> int:
        return unit["lines"] + unit.get("child_lines", 0)

    lost = [u for u in units if u["minutes_after_omission"] >= 30]
    if lost:
        worst = max(lost, key=lambda u: u["minutes_after_omission"])
        findings.append({"id": "objective-lost", "mechanism": "M1",
                         "detail": f"{len(lost)} turn(s) kept working after a compaction omitted the objective; worst {worst['minutes_after_omission']:.0f} active min from {worst['start'][:16]}",
                         "fix": "one unit per turn (end the turn after each unit) and a kernel of 600 characters or fewer"})
    stance_lost = sum(u.get("stance_lost", 0) for u in units)
    if stance_lost:
        findings.append({"id": "stance-not-retained", "mechanism": "M1",
                         "detail": f"{stance_lost} compaction(s) kept no user message carrying the stance (with --stance-marker: none containing it; without: no operator message at all)",
                         "fix": "start the effort with a stance message of 3,500 characters or fewer and keep the stance in the charter, re-read after each compaction; an operator or scheduler that can message the agent resends the stance"})
    long_turns = [u for u in units if u["hours"] > max_turn_hours and u["compactions"] >= 2]
    if is_child:
        # A native sub-agent has no goal to re-inject, and an owner is meant to hold one outcome
        # for hours. What matters is its age against the healthy window, not the turn length.
        total_hours = sum(u["hours"] for u in units)
        total_compactions = sum(u["compactions"] for u in units)
        if total_hours > OWNER_MAX_HOURS or total_compactions >= OWNER_MAX_COMPACTIONS:
            findings.append({"id": "owner-age", "mechanism": "M1",
                             "detail": f"this sub-agent has worked {total_hours:.1f} h through {total_compactions} compactions (healthy window: {OWNER_MAX_HOURS:g} h, {OWNER_MAX_COMPACTIONS} compactions)",
                             "fix": "at its next return, start a fresh owner briefed from the charter and told to read this owner's checkpoint first; do not re-task this one"})
    elif long_turns:
        findings.append({"id": "unit-per-turn", "mechanism": "M1",
                         "detail": f"{len(long_turns)} turn(s) longer than {max_turn_hours:g} h with 2+ compactions; longest {max(u['hours'] for u in long_turns):.1f} h",
                         "fix": "end the turn after each round, segment return or owner return"})
    if len(work) >= 6:
        third = max(2, len(work) // 3)
        early = quantile([total(u) for u in work[:third]], .5)
        late = quantile([total(u) for u in work[-third:]], .5)
        if early >= 50 and late < 0.25 * early:
            findings.append({"id": "shrinking-units", "mechanism": "M2",
                             "detail": f"median changed lines per working turn fell from {early:.0f} (first {third}) to {late:.0f} (last {third})",
                             "fix": "re-anchor on the pinned exemplars, not the latest records; run the challenger brief"})
    streak = best = 0
    for unit in work:
        streak = streak + 1 if total(unit) == 0 else 0
        best = max(best, streak)
    if best >= 3:
        findings.append({"id": "no-change-streak", "mechanism": "M3",
                         "detail": f"{best} consecutive working turns changed no lines (own or children)",
                         "fix": "the next unit takes a large item or a new focus; a reviewer argues for change on no-change units"})
    goal_turns = [u for u in units if u["trigger"] == "goal"]
    if len(goal_turns) >= 6 and sum(u["waits"] for u in goal_turns) >= 3:
        median_minutes = quantile([u["hours"] * 60 for u in goal_turns], .5)
        if median_minutes < 10:
            findings.append({"id": "hot-loop", "mechanism": "M25",
                             "detail": f"{len(goal_turns)} goal turns with a median of {median_minutes:.1f} min in a thread that waits on sub-agents: each turn end re-injects the goal and repeats the unit-start reads",
                             "fix": "a scheduler's turn spans the wait: block on wait_agent with timeout_ms 3600000, wait again when a wake brings no segment return, and end the turn only after a return is reviewed and recorded"})
    timeouts = [t for u in units for t in u["wait_timeouts_ms"]]
    if len(timeouts) >= 10 and quantile(timeouts, .5) <= 120000:
        findings.append({"id": "short-waits", "mechanism": "M25",
                         "detail": f"{len(timeouts)} wait_agent calls with a median timeout of {quantile(timeouts, .5) / 1000:.0f} s; each timed-out wait costs a full-context request and invites a status check",
                         "fix": "wait_agent returns on the next sub-agent message whatever the timeout, so ask for 3600000 ms; polling is not work"})
    orphaned = [u for u in units if u["goal_paused"] and u["trigger"] != "goal" and u["dispatches"] >= 1]
    if len(orphaned) >= 2:
        findings.append({"id": "paused-goal", "mechanism": "M24",
                         "detail": f"{len(orphaned)} turn(s) dispatched work while the native goal was paused, from {(orphaned[0]['start'] or '?')[:16]} to {(orphaned[-1]['end'] or '?')[:16]}: nothing woke this thread between operator messages",
                         "fix": "only the operator can re-activate a paused goal (`/goal resume`): say so in the first line of the reply, and hold the wait inside the turn until the goal is active"})
    reviews = [c for c in children if is_review(c)]
    if len(children) >= 12 and len(reviews) / len(children) >= 0.5:
        findings.append({"id": "review-flood", "mechanism": "M26",
                         "detail": f"{len(reviews)} of {len(children)} descendants were reviewers, challengers or auditors; they changed {sum(changed(c) for c in reviews)} lines",
                         "fix": "one ambition review per workstream plan and one challenger per showcase claim; increments are verified by owner tests and the running showcase, not by a fresh agent"})
    workers = [c for c in children if not is_review(c)]
    if len(workers) >= 4:
        short = [c for c in workers if lifetime(c) < 0.1]
        if len(short) / len(workers) >= 0.5:
            findings.append({"id": "short-workers", "mechanism": "M8",
                             "detail": f"{len(short)} of {len(workers)} non-review children were active under 6 minutes; they changed {sum(changed(c) for c in short)} lines",
                             "fix": "delegate by outcome: one long-lived owner per coherent outcome"})
    spawns = sum(u["spawns"] for u in units)
    fork_all = sum(u["spawn_fork_all"] for u in units)
    if spawns >= 4 and fork_all / spawns >= 0.5:
        findings.append({"id": "inherited-context", "mechanism": "M9",
                         "detail": f"{fork_all} of {spawns} spawns inherited the full parent history (fork_turns default all)",
                         "fix": "brief children fresh from the charter with fork_turns none"})
    if len(work) >= 6:
        third = max(2, len(work) // 3)
        stop_early = sum(u["families"].get("stop", 0) + u["families"].get("narrowing", 0) for u in work[:third])
        stop_late = sum(u["families"].get("stop", 0) + u["families"].get("narrowing", 0) for u in work[-third:])
        if stop_late >= 3 and stop_late >= 3 * max(1, stop_early):
            findings.append({"id": "closing-narration", "mechanism": "M3/M5",
                             "detail": f"narrowing and stop narration rose from {stop_early} (first {third} turns) to {stop_late} (last {third})",
                             "fix": "read the examples with `timeline`; closure goes through the challenger brief only"})
    return findings


def cmd_drift(args: argparse.Namespace) -> int:
    files = thread_files(args.rollout)
    units = [unit for path in files for unit in turn_units(path, args.stance_marker)]
    children: List[Dict[str, Any]] = []
    if args.scan:
        rows = load_scan(args.scan)
        own = next((r for r in rows if args.rollout in (r.get("files") or [r.get("file")])), None) or scan_rollout(args.rollout)
        by_parent = collections.defaultdict(list)
        for row in rows:
            if row.get("parent"):
                by_parent[row["parent"]].append(row)
        stack = list(by_parent.get(own.get("id"), []))
        while stack:
            child = stack.pop()
            children.append(child)
            stack.extend(by_parent.get(child.get("id"), []))
        direct = [c for c in children if c.get("parent") == own.get("id")]
        attribute_children(units, direct)
    first = next(iter_lines(files[0]), "") if files else ""
    is_child = '"session_meta"' in first[:200] and bool(read_session_meta(first)["parent"])
    findings = drift_findings(units, children, args.max_turn_hours, is_child)
    if args.json:
        print(json.dumps({"rollout": args.rollout, "files": files, "turns": units, "findings": findings}, indent=2))
    else:
        print(f"files={len(files)} turns={len(units)} working_turns={sum(1 for u in units if u['active_minutes'] >= 5)} descendants={len(children)}")
        if args.turns:
            print("start             trigger  hours  active_m  lines  child_lines  compactions  omitted  min_after_omission")
            for u in units:
                print(f"{(u['start'] or '?')[:16]}  {u['trigger'][:7]:<7}  {u['hours']:5.2f}  {u['active_minutes']:8.1f}  {u['lines']:5d}  "
                      f"{u.get('child_lines', 0):11d}  {u['compactions']:11d}  {u['objective_omitted']:7d}  {u['minutes_after_omission']:18.1f}")
        if not findings:
            print("no tripwire fired")
        for finding in findings:
            print(f"TRIPWIRE {finding['id']} ({finding['mechanism']}): {finding['detail']}\n    fix: {finding['fix']}")
    return DRIFT_EXIT if findings else 0


# ---------------------------------------------------------------- goals corpus

def cmd_goals(args: argparse.Namespace) -> int:
    patterns = args.db or [os.path.join(HOME, ".vrooli/state/vrooli/web-console/sessions/codex/*/goals_1.sqlite"),
                           os.path.join(HOME, ".codex/goals_1.sqlite")]
    goals = []
    for pattern in patterns:
        for db in glob.glob(os.path.expanduser(pattern)):
            try:
                conn = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
                for row in conn.execute("select thread_id, objective, status, tokens_used, time_used_seconds, "
                                        "created_at_ms from thread_goals"):
                    goals.append({"db": db, "thread": row[0], "objective": row[1], "status": row[2],
                                  "tokens": row[3], "hours": (row[4] or 0) / 3600, "created_ms": row[5]})
                conn.close()
            except sqlite3.Error as exc:
                print(f"skip {db}: {exc}", file=sys.stderr)
    if args.contains:
        goals = [g for g in goals if args.contains.lower() in (g["objective"] or "").lower()]
    if not goals:
        print("no goals found", file=sys.stderr)
        return 1
    print(f"goals: {len(goals)}  goal-hours: {sum(g['hours'] for g in goals):.0f}")
    print(f"status: {dict(collections.Counter(g['status'] for g in goals))}")
    print("\nlength      n   " + "  ".join(f"{s:>9}" for s in ("complete", "blocked", "paused", "active")) + "  median_h")
    for low, high, label in ((0, 300, "<300"), (300, 700, "300-700"), (700, 1500, "700-1500"), (1500, 10**9, ">=1500")):
        group = [g for g in goals if low <= len(g["objective"] or "") < high]
        if not group:
            continue
        counts = collections.Counter(g["status"] for g in group)
        shares = "  ".join(f"{counts.get(s, 0) / len(group):>9.0%}" for s in ("complete", "blocked", "paused", "active"))
        print(f"{label:<9} {len(group):>4}   {shares}  {statistics.median(g['hours'] for g in group):>8.1f}")
    if args.show:
        for goal in sorted(goals, key=lambda g: -(g["created_ms"] or 0))[: args.show]:
            created = dt.datetime.fromtimestamp((goal["created_ms"] or 0) / 1000, dt.timezone.utc).isoformat()[:16]
            print(f"\n{created}Z {goal['status']} {goal['hours']:.1f}h {len(goal['objective'])} chars thread={goal['thread']}\n"
                  f"  {goal['objective'][:400]}")
    return 0


# ---------------------------------------------------------------- claude code sessions

def claude_metrics(path: str) -> Dict[str, Any]:
    record = {"file": path, "start": None, "end": None, "compactions": 0, "edit_calls": 0, "lines_added": 0,
              "lines_removed": 0, "agent_calls": 0, "operator_messages": 0, "tools": collections.Counter()}
    events: List[Tuple[str, str, str]] = []
    for line in iter_lines(path):
        data = loads(line)
        if not data:
            continue
        ts = (data.get("timestamp") or "")[:19]
        if ts:
            record["start"] = record["start"] or ts
            record["end"] = ts
        if data.get("subtype") == "compact_boundary":
            record["compactions"] += 1
            pre = (data.get("compactMetadata") or {}).get("preTokens")
            events.append((ts, "compaction", f"pre_tokens={pre}"))
            continue
        message = data.get("message") if isinstance(data.get("message"), dict) else {}
        content = message.get("content")
        if data.get("type") == "user" and not data.get("isSidechain") and not data.get("isCompactSummary") \
                and not data.get("isMeta"):
            text = content if isinstance(content, str) else " ".join(
                c.get("text", "") for c in content or [] if isinstance(c, dict) and c.get("type") == "text")
            if text.strip() and not text.lstrip().startswith("<"):
                record["operator_messages"] += 1
                events.append((ts, "operator", text.strip().replace("\n", " ")[:240]))
        if data.get("type") == "assistant" and isinstance(content, list):
            for item in content:
                if not isinstance(item, dict) or item.get("type") != "tool_use":
                    continue
                name = item.get("name") or "?"
                record["tools"][name] += 1
                tool_input = item.get("input") or {}
                if name == "Edit":
                    record["edit_calls"] += 1
                    record["lines_added"] += len((tool_input.get("new_string") or "").splitlines())
                    record["lines_removed"] += len((tool_input.get("old_string") or "").splitlines())
                elif name == "MultiEdit":
                    record["edit_calls"] += 1
                    for edit in tool_input.get("edits") or []:
                        if isinstance(edit, dict):
                            record["lines_added"] += len((edit.get("new_string") or "").splitlines())
                            record["lines_removed"] += len((edit.get("old_string") or "").splitlines())
                elif name == "Write":
                    record["edit_calls"] += 1
                    record["lines_added"] += len((tool_input.get("content") or "").splitlines())
                elif name in ("Agent", "Task"):
                    record["agent_calls"] += 1
                    events.append((ts, "spawn", f"{tool_input.get('subagent_type', '?')}: {tool_input.get('description', '')}"))
    record["hours"] = round(hours_between(record["start"], record["end"]), 3)
    record["tools"] = dict(record["tools"].most_common(10))
    return {"metrics": record, "events": events}


def cmd_claude(args: argparse.Namespace) -> int:
    result = claude_metrics(args.session)
    metrics = result["metrics"]
    print("## Session")
    print(json.dumps({k: v for k, v in metrics.items() if k != "file"}, indent=1))
    subdir = os.path.join(os.path.splitext(args.session)[0], "subagents")
    if os.path.isdir(subdir):
        print("\n## Subagents (Edit/Write lines are approximate: changed blocks, not diffs)")
        for name in sorted(os.listdir(subdir)):
            if not name.endswith(".jsonl"):
                continue
            meta_path = os.path.join(subdir, name[:-6] + ".meta.json")
            meta = {}
            if os.path.exists(meta_path):
                try:
                    with open(meta_path, encoding="utf-8") as meta_file:
                        meta = json.load(meta_file)
                except ValueError:
                    meta = {}
            sub = claude_metrics(os.path.join(subdir, name))["metrics"]
            print(f"{meta.get('agentType', '?'):<14} depth={meta.get('spawnDepth', '?')} {sub['hours'] * 60:.0f}m "
                  f"edits={sub['edit_calls']} +{sub['lines_added']} -{sub['lines_removed']} "
                  f"compactions={sub['compactions']}  {meta.get('description', '')}")
    if args.events:
        print("\n## Events")
        for ts, kind, detail in result["events"]:
            print(f"{ts}  {kind:<10} {detail}")
    return 0


# ---------------------------------------------------------------- cli

def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)

    scan = sub.add_parser("scan", help="per-thread metrics for Codex rollouts (JSONL)")
    scan.add_argument("--root", action="append", help="glob of a sessions directory; repeatable (default: web-console homes and ~/.codex/sessions)")
    scan.add_argument("--since", help="first day to include, YYYY-MM-DD (from the rollout filename, local time)")
    scan.add_argument("--until", help="last day to include, YYYY-MM-DD")
    scan.add_argument("--out", help="write JSONL here instead of stdout")
    scan.set_defaults(func=cmd_scan)

    summary = sub.add_parser("summary", help="tables from a scan file")
    summary.add_argument("scan")
    summary.add_argument("--top", type=int, default=10)
    summary.set_defaults(func=cmd_summary)

    tree = sub.add_parser("tree", help="one root thread and its descendants")
    tree.add_argument("scan")
    tree.add_argument("thread", help="root thread id (or a fragment of its rollout path)")
    tree.add_argument("--list", action="store_true", help="print every descendant")
    tree.set_defaults(func=cmd_tree)

    timeline = sub.add_parser("timeline", help="events of one Codex rollout")
    timeline.add_argument("rollout")
    timeline.add_argument("--only", help="comma list of event kinds: goal,operator,compaction,turn_start,turn_end,spawn_agent,followup_task")
    timeline.add_argument("--examples", type=int, default=3, help="narration examples per family")
    timeline.set_defaults(func=cmd_timeline)

    drift = sub.add_parser("drift", help="tripwires for a live or finished effort: objective or stance loss, long turns, a paused goal, a hot scheduler loop, short waits, a review flood, shrinking units, no-change streaks, short workers (exit 3 when any fires)")
    drift.add_argument("rollout", help="the scheduler's or owner's rollout")
    drift.add_argument("--scan", help="a scan file covering its descendants, to credit child lines and check worker lifetimes")
    drift.add_argument("--max-turn-hours", type=float, default=3.0)
    drift.add_argument("--stance-marker", help="a distinctive substring of the stance message; a compaction that keeps no operator message containing it counts as stance loss (without it, only compactions that keep no operator message count)")
    drift.add_argument("--turns", action="store_true", help="print the per-turn table")
    drift.add_argument("--json", action="store_true")
    drift.set_defaults(func=cmd_drift)

    goals = sub.add_parser("goals", help="native goal corpus")
    goals.add_argument("--db", action="append", help="glob of goals_1.sqlite files; repeatable")
    goals.add_argument("--contains", help="only goals whose objective contains this text")
    goals.add_argument("--show", type=int, default=0, help="print the N most recent goals")
    goals.set_defaults(func=cmd_goals)

    claude = sub.add_parser("claude", help="one Claude Code session transcript and its subagents")
    claude.add_argument("session", help="path to ~/.claude/projects/<project>/<session>.jsonl")
    claude.add_argument("--events", action="store_true")
    claude.set_defaults(func=cmd_claude)
    return parser


def main(argv: Optional[List[str]] = None) -> int:
    if hasattr(signal, "SIGPIPE") and argv is None:
        signal.signal(signal.SIGPIPE, signal.SIG_DFL)  # exit quietly when piped into head
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
