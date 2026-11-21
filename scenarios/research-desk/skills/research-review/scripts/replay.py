#!/usr/bin/env python3
"""Replay a research commission on cold agents: old prompts against new, measured and graded blind.

Usage (each with --workspace DIR; one workspace holds one snapshot and every variant compared on it):
  replay.py snapshot CID...                     freeze what every variant shares (topics, docs, skills, CLIs)
  replay.py freeze CID...                       the `old` variant: each commission's own prompts, baseline skills
  replay.py render-new CID... --stamp S         a `new-S` variant: the current skills render the same specs
  replay.py sandbox-check [--variant V]         prove the sandbox refuses writes, services and the network
  replay.py preregister --arm A=V.. --arm B=V.. --expected TEXT --claim KIND:...   before the runs
  replay.py batch --cids C.. --variants V.. [--roles R..] [--owner-topics T..] [--repeat N] [--parallel P]
  replay.py run CID --variant V --role supervisor|owner [--topic T] [--card N] [--timeout S]
  replay.py grade-all --arm A=V.. --arm B=V.. [--noise N]   blind grades, each pair in both orders
  replay.py grade RUN_A RUN_B --topic T
  replay.py metrics RUN_DIR...                  re-parse recorded runs (writes metrics.json and plan.json)
  replay.py exclude RUN... --reason TEXT        keep a run but leave it out of the report
  replay.py report [--arm NAME=V1,V2 ...] [--correction TEXT]   write REPORT.md; arms pool variants, the first is the base

Every variant reads the same snapshot: an allowlisted copy of the repository (source and text files,
never a credential-like name or content, no data/ folders, no effort folder), the commissions' topic
folders, the research home's top-level files, every non-research skill and the Vrooli CLIs as
offline stubs. A variant differs only in its research skills (a copy in the workspace) and its
rendered commission. Each run is a fresh `codex exec --json` session with the role's model and
effort, inside bubblewrap with every namespace unshared: no host process, loopback port or socket is
reachable; the root holds /usr, an allowlisted part of /etc and the Codex install read-only; the home folder is an empty
read-only tmpfs onto which the snapshot, the variant's skills and the commission are mounted
read-only, at the paths the prompts name. The only writable places are /tmp and the Codex home, both
size-capped tmpfs that vanish with the sandbox; the login copy (no refresh token) exists only there,
written by a driver that runs every turn of the run in that one sandbox. The only
network path is a host gate that terminates TLS for the provider alone (the sandbox trusts only the
gate's CA) and forwards only the model API paths, over its own TLS connection. The prompt ends with a replay protocol: act as on receipt, stop
before the first edit, sub-agent or message, and print the planned first unit as JSON. Measured per
run: bytes the model saw (Codex rollout, after truncation) and raw command output, files read,
tokens, wall time, route failures and the plan. `grade` shows a separate cold model the snapshot's
MISSION.md, replay_rubric.txt and two plans as A/B, then again as B/A. Python 3 stdlib only; run with
python3 -I. Tests: python3 -I test_replay.py.
"""

from __future__ import annotations

import argparse
import collections
import datetime as dt
import gzip
import hashlib
import ipaddress
import json
import math
import os
import random
import re
import shlex
import shutil
import signal
import socket
import ssl
import stat
import subprocess
import sys
import tempfile
import threading
import time
from pathlib import Path

sys.dont_write_bytecode = True

HOME = Path.home()
SCRIPTS = Path(__file__).resolve().parent
SKILLS = SCRIPTS.parents[1]                       # scenarios/research-desk/skills
REPO = SKILLS.parents[2]
RESEARCH = HOME / ".vrooli" / "research"
EFFORT_DIR = REPO / ".vrooli/work/research-system-maturation"
WORKSPACE = EFFORT_DIR / "owners/A/replay/v4"
BASELINE_SKILLS = EFFORT_DIR / "owners/A/baseline-skills-20261115"
RUBRIC = SCRIPTS / "replay_rubric.txt"
REPLAY_FILES = ("replay.py", "test_replay.py", "replay_rubric.txt")  # never visible to a replayed agent
PROVIDER = "chatgpt.com"  # the gate's only upstream; the gate opens the TLS, so SNI and Host are always this
PROVIDER_ROUTES = {("POST", "/backend-api/codex/responses"): "",  # the only requests it forwards, with the query
                   ("GET", "/backend-api/codex/models"): r"client_version=[0-9][0-9A-Za-z.+-]{0,30}",  # each may carry
                   ("GET", "/backend-api/wham/accounts/check"): ""}  # model turns, the model list, routing discovery
# What Codex 0.167 sends, captured through the gate with a local stand-in for the provider (no model call)
# and checked against the response items of 89 recorded runs. Anything else is refused, since the
# provider may act on it itself: hosted tools, URL fetches, stored state, other features.
TURN_STATE = "x-codex-turn-state"  # a routing value the provider issues in a reply; Codex echoes it on the turn's POSTs
REQUEST_HEADERS = {"accept", "chatgpt-account-id", "content-type", "cookie", "originator", "session-id", "thread-id",
                   "user-agent", "version", "x-client-request-id", "x-codex-beta-features", "x-codex-routing-hint", TURN_STATE,
                   "x-codex-turn-metadata", "x-codex-window-id", "x-openai-internal-codex-responses-lite",
                   "host", "connection", "content-length", "authorization"}  # the last four the gate sets itself
BETA_FEATURES = {"remote_compaction_v2"}
REQUEST_KEYS = {"model", "stream", "tool_choice", "parallel_tool_calls", "reasoning", "store", "include", "prompt_cache_key",
                "text", "client_metadata", "access_programs", "input"}
META = "internal_chat_message_metadata_passthrough"
ITEM_KEYS = {"message": {"type", "id", "role", "content", "phase", "status", META},
             "reasoning": {"type", "id", "summary", "encrypted_content", "content", META},
             "custom_tool_call": {"type", "id", "call_id", "name", "input", "status", META},
             "custom_tool_call_output": {"type", "id", "call_id", "output", META},
             "function_call": {"type", "id", "call_id", "name", "arguments", "namespace", "status", META},
             "function_call_output": {"type", "id", "call_id", "output", META},
             "additional_tools": {"type", "id", "role", "tools"}, "configuration_update": {"type", "reasoning"},
             "compaction_trigger": {"type"}, "compaction": {"type", "id", "encrypted_content"}}  # auto-compaction
PART_KEYS = {"input_text": {"type", "text"}, "output_text": {"type", "text", "annotations", "logprobs"},
             "summary_text": {"type", "text"}, "input_image": {"type", "image_url", "detail"}}  # image: data: URL only
TOOL_KEYS = {"function": {"type", "name", "description", "parameters", "strict"},  # tools Codex runs itself,
             "custom": {"type", "name", "description", "format"},  # grouped in namespaces
             "namespace": {"type", "name", "description", "tools"}}
BODY_CAP = 32 << 20  # bytes: one request body
GATE_REQUESTS, GATE_BYTES = 1500, 1 << 30  # per gate (one run): requests forwarded, bytes in both directions
TMP_CAP = 1 << 30  # bytes: the sandbox's /tmp, a tmpfs
SHM_CAP = 64 << 20  # bytes: its /dev/shm; every other mount but these and the Codex home is read-only
CODEX_HOME_CAP = 1 << 30  # bytes: the sandbox's Codex home, a tmpfs; the login copy lives only there
OUTPUT_CAP = 256 << 20  # bytes: a run's event stream, as recorded on the host
CODEX_FILES = ("config.toml", "installation_id", "version.json", "settings.json", "models_cache.json")
CODEX_SHARED = ("plugins", "rules", "skills")
ETC_ALLOW = ("alternatives", "bash.bashrc", "environment", "gai.conf", "gitconfig", "group", "host.conf", "hostname", "hosts",
             "inputrc", "ld.so.cache", "ld.so.conf", "ld.so.conf.d", "locale.alias", "localtime", "lsb-release", "magic",
             "magic.mime", "mime.types", "nsswitch.conf", "os-release", "passwd", "profile", "profile.d", "protocols",
             "python3", "python3.12", "services", "shells", "ssl", "terminfo", "timezone")  # nothing else of /etc  # mounted read-only into each run's Codex home
PROXY_PORT = 18080
SANDBOX_NOTE = ("bwrap --unshare-all: /usr, allowlisted /etc, Codex install read-only; home an empty read-only tmpfs with the "
                "snapshot (allowlisted repo copy), variant skills and commission mounted read-only; writable: capped "
                "tmpfs /tmp and Codex home (login copy without refresh token, never on the host disk); network: the gate to "
                f"https://{PROVIDER} " + ", ".join(sorted(f"{m} {r}" for m, r in PROVIDER_ROUTES)) + " only, as Codex sends them")
SKILL_IDS = ("adaptive-research", "research-coordination", "local-experimentation", "research-review", "research-desk")
ROUTE_CLIS = ("prompt-manager", "search-hub", "research-desk", "agent-manager", "notification-hub", "program-runtime",
              "source-ledger", "swarm-manager", "test-genie", "vrooli", "vrooli-memory")
DEFAULT_TIMEOUT = 1800

PROTOCOL = {
    "supervisor": """Replay protocol. Do exactly what you would do on receiving this, up to the moment you would make your first edit, start your first owner or sub-agent, or message anyone. Do not take that step. Stop there and end with one JSON object and nothing after it:
{"role": "supervisor", "owners": [{"topic": "<topic folder>", "workstream": "<workstream ID>", "destination": "<what this owner lands>", "changes": ["<each scenario or API it would change>"], "brief": "<the full brief text you would send>"}], "next_step": "<the action you stopped before>", "operator_questions": ["<anything you would ask the operator now>"]}""",
    "owner": """Replay protocol. Do exactly what you would do on receiving this, up to the moment you would make your first edit, start your first worker or sub-agent, or message anyone. Do not take that step. Stop there and end with one JSON object and nothing after it:
{"role": "owner", "workstream": "<workstream ID>", "changes": ["<each scenario or API you would change>"], "first_hour": "<what you would do in the first hour>", "showcase": "<the showcase you are driving toward>", "next_step": "<the action you stopped before>", "operator_questions": ["<anything you would ask the operator now>"]}""",
}


class Fail(Exception):
    pass


def now() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def sha(text: str | bytes) -> str:
    return hashlib.sha256(text.encode() if isinstance(text, str) else text).hexdigest()[:16]


def write(path: Path, text: str) -> Path:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")
    return path


def write_json(path: Path, obj) -> Path:
    return write(path, json.dumps(obj, indent=2, ensure_ascii=False) + "\n")


def read_json(path: Path, default=None):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return default


# ---------------------------------------------------------------- charter and mission parsing


def sections(text: str, level: int = 2) -> list[tuple[str, str]]:
    """(heading, body) for each heading of exactly `level` hashes, outside code fences."""
    out, head, body, fence = [], None, [], False
    mark = "#" * level + " "
    for line in text.splitlines():
        if line.startswith("```"):
            fence = not fence
        if not fence and line.startswith(mark):
            if head is not None:
                out.append((head, "\n".join(body).strip("\n")))
            head, body = line[len(mark):].strip(), []
        elif head is not None:
            body.append(line)
    if head is not None:
        out.append((head, "\n".join(body).strip("\n")))
    return out


def section(text: str, prefix: str) -> str:
    return next((b for h, b in sections(text) if h.lower().startswith(prefix.lower())), "")


def charter_topics(charter: str, packet: dict | None) -> list[str]:
    if packet and packet.get("spec", {}).get("topics"):
        return list(packet["spec"]["topics"])
    seen: list[str] = []
    for t in re.findall(r"topics/([A-Za-z0-9][A-Za-z0-9._-]*)/MISSION\.md", charter):
        if t not in seen:
            seen.append(t)
    return seen


def cards(charter: str, topics: list[str]) -> list[dict]:
    """Each `### Card` section with its topic: the first topics/<t>/ it names, else the only topic."""
    out = []
    cards_body = next((b for h, b in sections(charter) if h.lower().startswith("cards")), charter)
    for n, (head, body) in enumerate([s for s in sections(cards_body, 3) if s[0].startswith("Card")], 1):
        m = re.search(r"topics/([A-Za-z0-9][A-Za-z0-9._-]*)/", body)
        topic = m.group(1) if m and m.group(1) in topics else (topics[0] if len(topics) == 1 else None)
        mode = re.search(r"mode `([a-z-]+)`", head)
        out.append({"n": n, "heading": head, "topic": topic, "mode": mode.group(1) if mode else None, "body": body})
    return out


def stance_quotes(charter: str) -> list[str]:
    body = next((b for h, b in sections(charter) if h.lower().startswith("operator stance")), "")
    quotes, cur = [], []
    for line in body.splitlines() + [""]:
        if line.startswith(">"):
            cur.append(line[1:].lstrip())
        elif cur:
            quotes.append("\n".join(cur).strip())
            cur = []
    return [q for q in quotes if q]


def strip_attribution(quote: str) -> str:
    m = re.match(r'^([\"“].*[\"”])\s+—\s+[^\"”]*$', quote, re.DOTALL)
    return m.group(1) if m else quote


def numbered_list(body: str) -> list[str]:
    items, cur = [], None
    for line in body.splitlines():
        m = re.match(r"^\s*\d+\.\s+(.*)", line)
        if m:
            if cur is not None:
                items.append(cur)
            cur = m.group(1)
        elif cur is not None and line.strip() and line.startswith(" ") and not re.match(r"^\s*- ", line):
            cur += " " + line.strip()
        elif cur is not None:
            items.append(cur)
            cur = None
    if cur is not None:
        items.append(cur)
    return [" ".join(i.split()) for i in items]


def mission_mode(mission: str) -> str:
    m = re.search(r"^Mode:\s*(\S+)", mission, re.MULTILINE)
    return m.group(1).strip("`") if m else ""


def mission_title(mission: str, topic: str) -> str:
    m = re.search(r"^# (.+)$", mission, re.MULTILINE)
    return re.sub(r"\s*[—-]\s*mission\s*$", "", m.group(1).strip(), flags=re.I) if m else topic


def owner_brief_template(template_md: str) -> str:
    m = re.search(r"```text\n(.*?)\n```", template_md, re.DOTALL)
    if not m:
        raise Fail("owner-brief template has no ```text block")
    return m.group(1)


def fill_brief(template_md: str, *, charter: str, card: dict, topic: str, home: Path, cid: str, skills: Path,
               mission: str, workstreams: list[str] | None = None) -> tuple[str, list[str]]:
    """Fill the owner-brief template's [brackets] from the card, charter and mission the way the
    scheduler is told to; `workstreams` narrows an owner to its share of the card. Returns the brief
    and any bracket left unfilled."""
    body = card["body"]
    quotes = "\n\n".join(stance_quotes(charter))
    direction = "\n\n".join(b for h, b in sections(charter)
                          if re.search(r"operator steering|correction|steering\s+—", h, re.I))
    lists = re.findall(r"`([^`]+\.md)`", (re.search(r"list files ((?:`[^`]+`(?:, )?)+)", body) or [""])[0]) \
        if "list files" in body else []
    read_first = [line.strip("- ").strip() for line in body.splitlines()
                  if re.search(r"\*\*(Read first|Cards to read)", line)]
    notes = body.split("Operator's notes for this card, verbatim:", 1)[1].strip() if "Operator's notes" in body else ""
    workstreams = numbered_list(body) if workstreams is None else workstreams
    work_briefs = [f"Start from {b}" for b in sorted((home / "topics" / topic / "work").glob("*/BRIEF.md"))] \
        if "BRIEF.md" in template_md else []
    values = {
        "card topic": mission_title(mission, topic),
        "workstreams": "; ".join(workstreams) or "the card's workstreams",
        "read first": "\n".join(read_first + ([notes] if notes else []) + work_briefs),
        "stance": quotes + (f"\n\nCurrent direction:\n\n{direction}" if direction else ""),
        "mission.md path": str(home / "topics" / topic / "MISSION.md"),
        "adaptive-research path": str(skills / "adaptive-research" / "SKILL.md"),
        "mode": card.get("mode") or mission_mode(mission),
        "approval flow": section(mission, "Approval flow").split("\n\n")[0],
        "mission's sections": "\n\n".join(f"{n}:\n{section(mission, n)}" for n in ("Real boundaries", "Not this")),
        "shared machine": "\n\n".join(b for h, b in sections(charter) if re.search(r"shared machine|storage and usage", h, re.I)),
        "run folder": str(home / "topics" / topic / "runs" / cid),
        "f": str(skills / "research-coordination" / "scripts" / "footprint.py"),
        "b": str(home / "cohorts" / cid / "BASELINE.json"),
        "k": str(home / "cohorts" / cid / "FOOTPRINT-KEEP.txt"),
    }
    order = ("card topic", "workstreams", "read first", "stance", "mission.md path", "adaptive-research path",
             "approval flow", "mission's sections", "shared machine", "run folder")
    unfilled: list[str] = []

    def sub(m: re.Match) -> str:
        inner = m.group(1)
        key = inner.strip().lower()
        if key.startswith("if the card names list files"):
            rest = inner.split(":", 1)[1].strip() if ":" in inner else inner
            return f"List files: {', '.join(lists)}; {rest}" if lists else ""
        if key in ("mode", "f", "b", "k"):
            return values[key]
        if key == "topic":
            return values["card topic"]
        if key == "its path":  # "Read the adaptive-research skill first ([its path])"
            return values["adaptive-research path"]
        for k in order:
            if k in key or (k == "read first" and "operator's notes" in key):
                return values[k]
        unfilled.append(inner)
        return m.group(0)

    brief = re.sub(r"\[([^\[\]\n]{1,300})\]", sub, owner_brief_template(template_md))
    return brief, unfilled


# ---------------------------------------------------------------- workspace paths


class WS:
    def __init__(self, root: Path):
        self.root = root

    def snapshot(self) -> Path:
        return self.root / "snapshot"

    def variant(self, variant: str) -> Path:
        return self.root / "variants" / variant

    def prompts(self, cid: str, variant: str) -> Path:
        return self.root / "commissions" / cid / variant

    def runs(self) -> Path:
        return self.root / "runs"

    def grades(self) -> Path:
        return self.root / "grades"


def research_home(ws: WS) -> Path:
    """The research home path the prompts name; the sandbox mounts the snapshot there."""
    return Path((read_json(ws.snapshot() / "snapshot.json") or {}).get("research_home") or RESEARCH)


def skill_sizes(root: Path) -> dict[str, int]:
    out = {}
    for sid in SKILL_IDS:
        base = root / sid
        for p in sorted(base.rglob("*")) if base.is_dir() else []:
            if p.is_file() and not {"__pycache__", "scripts"} & set(p.parts) and p.suffix in (".md", ".txt", ".json"):
                out[str(p.relative_to(root))] = p.stat().st_size
    out["total"] = sum(out.values())
    return out


def copy_skills(src: Path, dest: Path) -> None:
    """A variant's research skills, without the replay harness: agents never see what measures them."""
    shutil.copytree(src, dest, ignore=shutil.ignore_patterns("__pycache__", *REPLAY_FILES))


# ---------------------------------------------------------------- snapshot (shared by every variant)


TEXT_SUFFIXES = {".md", ".txt", ".json", ".yaml", ".yml", ".toml", ".csv", ".tsv"}
SKIP_DIRS = {"node_modules", ".git", "persist", "__pycache__", ".venv", "venv"}
SOURCE_SUFFIXES = TEXT_SUFFIXES | {".go", ".mod", ".sum", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".py", ".sh",
                                   ".bash", ".proto", ".sql", ".css", ".scss", ".html", ".rs", ".tmpl", ".svelte", ".vue"}
SOURCE_NAMES = {"Makefile", "Dockerfile"}
REPO_SKIP = SKIP_DIRS | {"data", "dist", "build", "coverage", ".cache", "tmp", "logs", "vendor", "target", ".next",
                         ".turbo", "secrets", "test-results", "playwright-report", "dist-electron"}
SECRET_NAME = re.compile(r"^\.env|\.(key|pem|p12|pfx|jks|keystore|kdbx|db|db-wal|db-shm|sqlite3?)$|^(secrets?|credentials?)\."
                         r"|^id_(rsa|ecdsa|ed25519)|^auth\.json$|^\.(npmrc|netrc|pgpass)$", re.I)
SECRET_TEXT = re.compile(rb"-----BEGIN [A-Z ]*PRIVATE KEY-----|\bsk-(proj-|svcacct-)?[A-Za-z0-9_-]{24,}|\bgh[pousr]_[A-Za-z0-9]{30,}"
                         rb"|\bxox[abpr]-[A-Za-z0-9-]{10,}|\bAKIA[0-9A-Z]{16}\b|\b[a-z][a-z0-9+.-]*://[^\s/:@'\"%{}<>$\[\]]+:[^\s/@'\"%{}<>$\[\]]+@|\"refresh_token\"\s*:\s*\"[^\"]{20,}|(?m:^(//\S+:)?_authToken\s*=\s*\S)")


def copy_safe(p: Path, t: Path) -> bool:
    """Copy one file unless its name or content looks like a credential."""
    if SECRET_NAME.search(p.name):
        return False
    data = p.read_bytes()
    if SECRET_TEXT.search(data):
        return False
    t.parent.mkdir(parents=True, exist_ok=True)
    t.write_bytes(data)
    return True
OFFLINE = '#!/bin/sh\necho "$(basename "$0"): cannot reach the Vrooli service: connection refused" >&2\nexit 69\n'
SHIM = r'''#!/usr/bin/env -S python3 -I
# prompt-manager in a replay: `skill read <id>...` prints the skill body (the variant's research skills, the
# snapshot's copy of every other skill); every other verb is offline like the other Vrooli CLIs.
import os, re, sys
SKILLS, EXPORT = __SKILLS__, __EXPORT__
a = sys.argv[1:]
if len(a) >= 3 and a[0] in ("skill", "skills", "s") and a[1] == "read":
    lines, ids, i = None, [], 2
    while i < len(a):
        if a[i] == "--lines" and i + 1 < len(a):
            lines = a[i + 1]; i += 2; continue
        if not a[i].startswith("-"):
            ids.append(a[i])
        i += 1
    out = []
    for x in ids:
        p = os.path.join(SKILLS, x, "SKILL.md")
        p = p if os.path.isfile(p) else os.path.join(EXPORT, x + ".md")
        if not os.path.isfile(p):
            sys.stderr.write(f"skill not found: {x}\n"); sys.exit(1)
        out.append(re.sub(r"\A---\n.*?\n---\n", "", open(p, encoding="utf-8").read(), flags=re.S).strip() + "\n")
    text = "\n".join(out)
    if lines:
        lo, _, hi = lines.partition("-")
        text = "\n".join(text.splitlines()[int(lo) - 1:int(hi or lo)]) + "\n"
    sys.stdout.write(text); sys.exit(0)
sys.stderr.write("prompt-manager: cannot reach the Vrooli service: connection refused\n"); sys.exit(69)
'''
SKILL_EXPORT = HOME / ".vrooli/share/skills"  # where the snapshot's skill copies appear inside the sandbox


def copy_text_tree(src: Path, dest: Path, *, cap_file: int, cap_dir: int, skip_prefixes: tuple = ()) -> dict:
    """Copy a tree's text files of at most cap_file bytes. A top-level folder whose copy would pass
    cap_dir is left out whole; under runs/ only Markdown three levels deep is kept; folders whose name
    starts with a skip prefix (the replayed commissions' own runs) are left out."""
    kept, left_out = 0, {}

    def candidates(top: Path) -> list[tuple[Path, int]]:
        out = []
        for dirpath, dirs, files in os.walk(top, onerror=lambda e: None):
            rel = Path(dirpath).relative_to(src).parts
            dirs[:] = [d for d in dirs if d not in SKIP_DIRS and not (rel and rel[0] == "runs" and len(rel) == 1
                                                                    and d.startswith(skip_prefixes))]
            in_runs = bool(rel) and rel[0] == "runs"
            for f in files:
                p = Path(dirpath) / f
                try:
                    st = p.lstat()
                except OSError:
                    continue
                if not os.path.isfile(p) or os.path.islink(p) or st.st_size > cap_file:
                    continue
                if (p.suffix == ".md" and len(rel) <= 3) if in_runs else p.suffix in TEXT_SUFFIXES:
                    out.append((p, st.st_size))
        return out

    for entry in sorted(src.iterdir()) if src.is_dir() else []:
        if entry.is_dir() and not entry.is_symlink():
            files = candidates(entry) if entry.name not in SKIP_DIRS else []
            size = sum(s for _, s in files)
            if size > cap_dir:
                left_out[entry.name] = size
                continue
        elif entry.is_file() and not entry.is_symlink() and entry.suffix in TEXT_SUFFIXES and entry.stat().st_size <= cap_file:
            files = [(entry, entry.stat().st_size)]
        else:
            continue
        for p, _ in files:
            try:
                kept += copy_safe(p, dest / p.relative_to(src))
            except OSError:
                pass
    return {"files": kept, "left_out_dirs": left_out}


def copy_repo(src: Path, dest: Path, leave_out: tuple[Path, ...]) -> dict:
    """The repository as an agent may read it, an allowlist: source and text files (SOURCE_SUFFIXES,
    at most 1 MB, regular files) outside REPO_SKIP folders, Electron bundles and `leave_out`, never one whose name or
    content looks like a credential. Nothing else of the repository exists in the sandbox."""
    kept = refused = 0
    for dirpath, dirs, files in os.walk(src, onerror=lambda e: None):
        d = Path(dirpath)
        dirs[:] = [x for x in dirs if x not in REPO_SKIP and d / x not in leave_out
                   and not (x == "bundle" and "electron" in d.parts)]  # packaged copies of other scenarios' config
        for f in files:
            p = d / f
            if p.suffix not in SOURCE_SUFFIXES and f not in SOURCE_NAMES:
                continue
            try:
                st = p.lstat()
                if not stat.S_ISREG(st.st_mode) or st.st_size > 1_000_000:
                    continue
                ok = copy_safe(p, dest / p.relative_to(src))
            except OSError:
                continue
            kept, refused = kept + ok, refused + (not ok)
    return {"files": kept, "credential_like_left_out": refused}


def write_cli_stubs(bin_dir: Path, names: list[str], skills: Path, export: Path) -> None:
    """Every Vrooli CLI as an offline stub, node as on the host, prompt-manager as the skill shim."""
    bin_dir.mkdir(parents=True)
    write(bin_dir / ".offline", OFFLINE).chmod(0o755)
    for name in names:
        if name == "node" and (HOME / ".vrooli/bin/node").is_file():
            shutil.copy2(HOME / ".vrooli/bin/node", bin_dir / "node")
        elif name != "prompt-manager":
            (bin_dir / name).symlink_to(".offline")
    write(bin_dir / "prompt-manager", SHIM.replace("__SKILLS__", repr(str(skills))).replace("__EXPORT__", repr(str(export)))
          ).chmod(0o755)


def cmd_snapshot(ws: WS, cids: list[str], research: Path) -> dict:
    """One frozen copy of what every variant reads besides its own skills and commission: the topics of
    these commissions, the research home's top-level files, the allowlisted repository, every other
    skill's text, and the Vrooli CLIs as offline stubs (prompt-manager serves skill reads)."""
    snap = ws.snapshot()
    if snap.exists():
        raise Fail(f"{snap} exists; a workspace has one snapshot (use a new --workspace)")
    topics: list[str] = []
    for cid in cids:
        src = research / "cohorts" / cid
        if not (src / "README.md").is_file():
            raise Fail(f"{src}/README.md does not exist")
        for t in charter_topics((src / "README.md").read_text(encoding="utf-8"), read_json(src / "PACKET.json")):
            topics.append(t) if t not in topics else None
    home = snap / "home"
    (home / "cohorts").mkdir(parents=True)
    info = {"snapshot_at": now(), "research_home": str(research), "commissions": cids, "topics": {}}
    for t in topics:
        info["topics"][t] = copy_text_tree(research / "topics" / t, home / "topics" / t, cap_file=1_000_000,
                                           cap_dir=50_000_000, skip_prefixes=tuple(cids))
    for entry in sorted(research.iterdir()):
        if entry.is_file() and not entry.is_symlink() and entry.suffix in TEXT_SUFFIXES and entry.stat().st_size <= 2_000_000:
            copy_safe(entry, home / entry.name)
    info["repo"] = copy_repo(REPO, snap / "repo", (EFFORT_DIR, SKILLS))
    (snap / "repo" / SKILLS.relative_to(REPO)).mkdir(parents=True, exist_ok=True)  # where each variant's skills mount
    p = subprocess.run([which("prompt-manager") or str(HOME / ".vrooli/bin/prompt-manager"), "skill", "list", "--json"],
                       capture_output=True, text=True, timeout=300)
    try:
        listed = json.loads(p.stdout)
    except ValueError:
        raise Fail(f"prompt-manager skill list --json failed: {(p.stderr or p.stdout)[-300:]}")
    exported = 0
    for s in listed:
        path = Path(s.get("contentPath") or "")
        if s.get("id") and path.is_file() and SKILLS not in path.parents:
            write(snap / "skills" / f"{s['id']}.md", path.read_text(encoding="utf-8"))
            exported += 1
    info["skills_exported"] = exported
    names = sorted(e.name for e in (HOME / ".vrooli/bin").iterdir() if not e.name.startswith(".")) \
        if (HOME / ".vrooli/bin").is_dir() else list(ROUTE_CLIS)
    write_cli_stubs(snap / "bin", names, SKILLS, SKILL_EXPORT)
    info["cli_stubs"] = len(names)
    write_json(snap / "snapshot.json", info)
    return {"snapshot": str(snap), "topics": {t: v["files"] for t, v in info["topics"].items()},
            "left_out": {t: v["left_out_dirs"] for t, v in info["topics"].items() if v["left_out_dirs"]},
            "skills_exported": exported, "cli_stubs": len(names)}


def need_snapshot(ws: WS) -> Path:
    snap = ws.snapshot()
    if not (snap / "snapshot.json").is_file():
        raise Fail(f"{snap} has no snapshot: run `snapshot CID...` first")
    return snap


# ---------------------------------------------------------------- freeze (old) and render-new


def prepare_cohort(cohort: Path, skills: Path) -> dict:
    """What the scheduler has made before any owner starts: the footprint baseline (a real snapshot by
    the variant's footprint.py) and an empty keep file, so owner briefs that name them resolve."""
    out = {}
    if not (cohort / "BASELINE.json").exists():
        p = subprocess.run([sys.executable, "-I", str(skills / "research-coordination/scripts/footprint.py"), "snapshot",
                            "--out", str(cohort / "BASELINE.json")], capture_output=True, text=True, timeout=300)
        out["baseline"] = {"exit": p.returncode, "tail": (p.stdout + p.stderr)[-300:]}
    if not (cohort / "FOOTPRINT-KEEP.txt").exists():
        write(cohort / "FOOTPRINT-KEEP.txt", "")
    return out


STATE_STUB = "# State\n\nPrepared; not started. No owner, unit or review has run under this commission yet.\n"


def owner_split(charter: str) -> dict[int, int]:
    """Owners per card when the charter starts one per listed workstream ("card 1: 2; card 2: 1, one per
    listed workstream in card order"); a card not named gets one owner for all its workstreams."""
    m = re.search(r"Start \d+ owners \(([^)]*one per listed workstream[^)]*)\)", charter)
    return {int(n): int(k) for n, k in re.findall(r"card (\d+): (\d+)", m.group(1))} if m else {}


def owner_briefs(ws: WS, pdir: Path, tmpl: str, charter: str, topics: list[str], cid: str, research: Path) -> tuple[list, dict]:
    """One brief per owner the scheduler starts. A card's first owner is owner-<topic>-card<N>.txt in
    every variant, so owner runs pair across variants; a split card's i-th owner takes its i-th workstream."""
    owners, unfilled, split = [], {}, owner_split(charter)
    for card in cards(charter, topics):
        if not card["topic"]:
            continue
        mission = (ws.snapshot() / "home/topics" / card["topic"] / "MISSION.md").read_text(encoding="utf-8")
        listed = numbered_list(card["body"])
        k = split.get(card["n"], 1)
        shares = [[w] for w in listed[:k]] if k > 1 else [None]
        for i, share in enumerate(shares):
            brief, miss = fill_brief(tmpl, charter=charter, card=card, topic=card["topic"], home=research, cid=cid,
                                     skills=SKILLS, mission=mission, workstreams=share)
            name = f"owner-{card['topic']}-card{card['n']}" + (f"-owner{i + 1}" if i else "") + ".txt"
            write(pdir / name, brief)
            owners.append({"file": name, "topic": card["topic"], "card": card["n"], "heading": card["heading"],
                           "workstreams": share, "bytes": len(brief.encode())})
            if miss:
                unfilled[name] = miss
    return owners, unfilled


def cmd_freeze(ws: WS, cids: list[str], baseline: Path) -> dict:
    """The `old` variant: each commission's own charter, stance and kernel, verbatim, with the baseline
    skills mounted where the prompts name the research skills."""
    research = (need_snapshot(ws), research_home(ws))[1]
    vdir = ws.variant("old")
    if not (vdir / "skills").is_dir():
        copy_skills(baseline, vdir / "skills")
    tmpl = (baseline / "research-coordination/templates/owner-brief.md").read_text(encoding="utf-8")
    out = {}
    for cid in cids:
        src = research / "cohorts" / cid
        raw = {n: (src / n).read_text(encoding="utf-8") for n in ("README.md", "STANCE-MESSAGE.txt", "KERNEL.txt")
               if (src / n).is_file()}
        if set(raw) != {"README.md", "STANCE-MESSAGE.txt", "KERNEL.txt"}:
            raise Fail(f"{cid}: needs README.md, STANCE-MESSAGE.txt and KERNEL.txt")
        packet = read_json(src / "PACKET.json")
        topics = charter_topics(raw["README.md"], packet)
        cohort = vdir / "cohorts" / cid
        for n, v in raw.items():
            write(cohort / n, v)
        if packet:
            write_json(cohort / "PACKET.json", packet)
        write(cohort / "STATE.md", STATE_STUB)
        prepared = prepare_cohort(cohort, vdir / "skills")
        pdir = ws.prompts(cid, "old")
        write(pdir / "stance.txt", raw["STANCE-MESSAGE.txt"])
        write(pdir / "kernel.txt", raw["KERNEL.txt"])
        owners, unfilled = owner_briefs(ws, pdir, tmpl, raw["README.md"], topics, cid, research)
        info = {"commission_id": cid, "cohort": cid, "variant": "old", "frozen_at": now(), "source": str(src),
                "files": {n: {"bytes": len(v.encode()), "sha": sha(v)} for n, v in raw.items()},
                "topics": topics, "owners": owners, "unfilled_brackets": unfilled, "skills": str(baseline),
                "skill_sizes": skill_sizes(vdir / "skills"), "prepared": prepared,
                "spec": reconstruct_spec(cid, raw, packet, topics),
                "notes": "The replay is the commission's start: LEDGER.md and OPERATOR.md are omitted, STATE.md is a one-line "
                         "prepared stub (replay.STATE_STUB), BASELINE.json is a fresh footprint snapshot and FOOTPRINT-KEEP.txt "
                         "is empty, the same for every variant."}
        write_json(pdir / "freeze.json", info)
        out[cid] = {"topics": topics, "owners": [o["file"] for o in owners], "unfilled": unfilled,
                    "charter_bytes": info["files"]["README.md"]["bytes"]}
    return out


def reconstruct_spec(cid: str, texts: dict, packet: dict | None, topics: list[str]) -> dict:
    """The packet spec that would rebuild this commission today: PACKET.json when it exists, else the
    charter's topics, stance quotes, allowance and card order."""
    if packet and isinstance(packet.get("spec"), dict):
        return dict(packet["spec"])
    m = re.search(r"(\d+)\s*h allowance", texts["KERNEL.txt"])
    order: dict[str, list[str]] = {}
    for card in cards(texts["README.md"], topics):
        if not card["topic"]:
            continue
        for ident in re.findall(r"\*\*([A-Z]{1,4}\d{1,3}[a-z]?)\b", card["body"]) + \
                re.findall(r"(?<![\w-])([A-Z]{1,4}\d{1,3})(?=[:,\s])", "\n".join(numbered_list(card["body"]))):
            ident = re.sub(r"^([A-Z]+)0+(\d)", r"\1\2", ident)
            ids = order.setdefault(card["topic"], [])
            if ident not in ids:
                ids.append(ident)
    return {"topics": topics, "allowance_hours": int(m.group(1)) if m else 24,
            "stance_quotes": [strip_attribution(q) for q in stance_quotes(texts["README.md"])] or [texts["STANCE-MESSAGE.txt"]],
            "slug": re.sub(r"[^a-z0-9-]", "-", cid.lower())[:52].strip("-"), "order": order,
            "supervisor_model": "gpt-6-luna", "supervisor_effort": "medium"}


def load_module(path: Path, name: str):
    import importlib.util
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    sys.modules[name] = mod
    spec.loader.exec_module(mod)
    return mod


def live_ids(skills: Path, home: Path, topic: str) -> list[str]:
    mc = load_module(skills / "research-coordination/scripts/mission_check.py", "replay_mission_check")
    rep = mc.check_mission(topic, home / "topics" / topic / "MISSION.md")
    return [w.ident for w in rep.workstreams if w.status in mc.LIVE]


def cmd_render_new(ws: WS, cids: list[str], stamp: str) -> dict:
    """A `new-<stamp>` variant: the current research skills copied into the workspace, and each frozen
    spec rendered by that copy's packet.py inside the sandbox, where the snapshot sits at the real
    research home path, so paths, gates and packet.py's fitting are exactly what a real build makes."""
    snap = need_snapshot(ws)
    research = research_home(ws)
    variant = f"new-{stamp}"
    vdir = ws.variant(variant)
    if vdir.exists():
        raise Fail(f"{variant} is already rendered; pass a new --stamp")
    copy_skills(SKILLS, vdir / "skills")
    tmpl = (vdir / "skills/research-coordination/templates/owner-brief.md").read_text(encoding="utf-8")
    out = {}
    for cid in cids:
        freeze = read_json(ws.prompts(cid, "old") / "freeze.json")
        if not freeze:
            raise Fail(f"{cid}: run freeze first")
        spec, dropped = dict(freeze["spec"]), {}
        live = {t: live_ids(vdir / "skills", snap / "home", t) for t in spec["topics"]}
        order = {t: [i for i in ids if i in live.get(t, [])][:4] for t, ids in (spec.get("order") or {}).items()}
        dropped["order"] = {t: [i for i in ids if i not in order.get(t, [])] for t, ids in (spec.get("order") or {}).items()}
        spec["order"] = {t: ids for t, ids in order.items() if ids}
        for key in ("supersedes", "run_alongside"):
            if spec.get(key):
                dropped[key], spec[key] = spec[key], []
        if spec.get("carry_baseline_from"):
            dropped["carry_baseline_from"], spec["carry_baseline_from"] = spec["carry_baseline_from"], None
        res = None
        for _ in range(3):  # a field the current packet.py no longer knows is dropped and recorded
            inner = ["python3", "-I", str(SKILLS / "research-coordination/scripts/packet.py"), "preview",
                     "--home", str(research), "--spec", "-"]
            proc = subprocess.run(sandbox_cmd(inner, ws=ws, variant=variant), input=json.dumps(spec), capture_output=True,
                                  text=True, timeout=120, env=sandbox_env())
            res = json.loads(proc.stdout) if proc.stdout.strip().startswith("{") else {"errors": [{"message": proc.stderr}]}
            unknown = [e["message"] for e in res.get("errors", []) if e.get("message", "").startswith("unknown spec fields:")]
            if not unknown:
                break
            for f in unknown[0].split(":", 1)[1].split(","):
                dropped[f.strip()] = spec.pop(f.strip(), None)
        if not res or not res.get("texts"):
            raise Fail(f"{cid}: packet.py preview failed: {json.dumps(res.get('errors') if res else None)[:800]}")
        new_cid, texts = res["commission_id"], res["texts"]
        cohort = vdir / "cohorts" / new_cid
        write(cohort / "README.md", texts["charter"])
        write(cohort / "STANCE-MESSAGE.txt", texts["stance"])
        write(cohort / "KERNEL.txt", texts["kernel"])
        write(cohort / "STATE.md", STATE_STUB)
        write_json(cohort / "PACKET.json", {"schema": "research-desk.packet/v1", "commission_id": new_cid,
                                            "built_at": now(), "spec": spec})
        prepared = prepare_cohort(cohort, vdir / "skills")
        pdir = ws.prompts(cid, variant)
        write(pdir / "stance.txt", texts["stance"])
        write(pdir / "kernel.txt", texts["kernel"])
        owners, unfilled = owner_briefs(ws, pdir, tmpl, texts["charter"], spec["topics"], new_cid, research)
        info = {"commission_id": cid, "cohort": new_cid, "variant": variant, "rendered_at": now(),
                "new_commission_id": new_cid, "packet_py_sha": sha((vdir / "skills/research-coordination/scripts/packet.py").read_bytes()),
                "spec": spec, "dropped_from_spec": {k: v for k, v in dropped.items() if v}, "gates": res.get("gates"),
                "errors": res.get("errors"), "files": {n: {"bytes": len(v.encode()), "sha": sha(v)} for n, v in texts.items()},
                "owners": owners, "unfilled_brackets": unfilled, "skills": str(SKILLS),
                "skill_sizes": skill_sizes(vdir / "skills"), "prepared": prepared}
        write_json(pdir / "render.json", info)
        out[cid] = {"new_commission_id": new_cid, "charter_bytes": info["files"]["charter"]["bytes"],
                    "gates_failed": [g["name"] for g in res.get("gates") or [] if not g["ok"]],
                    "owners": [o["file"] for o in owners], "unfilled": unfilled}
    return out


# ---------------------------------------------------------------- sandbox and harness


BRIDGE = r'''import signal, socket, subprocess, sys, threading
# Inside the sandbox: 127.0.0.1:PORT is the only way out; each connection is passed to the host gate.
port, gate, cmd = int(sys.argv[1]), sys.argv[2], sys.argv[4:]
def pipe(a, b):
    try:
        while (d := a.recv(65536)):
            b.sendall(d)
    except OSError:
        pass
    for s in (a, b):
        try:
            s.shutdown(socket.SHUT_RDWR)
        except OSError:
            pass
def serve(srv):
    while True:
        c, _ = srv.accept()
        g = socket.socket(socket.AF_UNIX)
        try:
            g.connect(gate)
        except OSError:
            c.close(); continue
        threading.Thread(target=pipe, args=(c, g), daemon=True).start()
        threading.Thread(target=pipe, args=(g, c), daemon=True).start()
srv = socket.socket()
srv.bind(("127.0.0.1", port)); srv.listen(64)
threading.Thread(target=serve, args=(srv,), daemon=True).start()
child = subprocess.Popen(cmd)
for sig in (signal.SIGINT, signal.SIGTERM):
    signal.signal(sig, lambda s, f: child.send_signal(s))
sys.exit(child.wait())
'''


DRIVER = r'''import glob, json, os, re, signal, subprocess, sys, threading
# Inside the sandbox: write the Codex home (a tmpfs) from the header line on stdin, run each turn, obey
# "stop N" (end turn N, go on) and "quit N" (end turn N and the run), then print the session rollouts.
home = os.environ["CODEX_HOME"]
cfg = json.loads(sys.stdin.readline())
for d in cfg["dirs"]:
    os.makedirs(os.path.join(home, d), exist_ok=True)
for name, text in cfg["files"].items():
    with os.fdopen(os.open(os.path.join(home, name), os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600), "w") as f:
        f.write(text)
state = {"p": None, "n": 0, "quit": False}
def emit(o):
    sys.stdout.write(json.dumps(o) + "\n")
    sys.stdout.flush()
def control():
    for line in sys.stdin:
        verb, _, n = line.strip().partition(" ")
        if n != str(state["n"]):
            continue
        state["quit"] = state["quit"] or verb == "quit"
        p = state["p"]
        try:
            os.killpg(p.pid, signal.SIGINT)
            p.wait(5)
        except subprocess.TimeoutExpired:
            os.killpg(p.pid, signal.SIGKILL)
        except (OSError, AttributeError):
            pass
threading.Thread(target=control, daemon=True).start()
thread = None
for n, argv in enumerate(cfg["turns"], 1):
    if state["quit"] or (not thread and any("{thread}" in a for a in argv)):
        break
    emit({"replay": "turn", "n": n})
    p = subprocess.Popen([a.replace("{thread}", thread or "") for a in argv], stdout=subprocess.PIPE,
                         stdin=subprocess.DEVNULL, start_new_session=True)
    state["p"], state["n"] = p, n
    for line in p.stdout:
        sys.stdout.buffer.write(line)
        sys.stdout.flush()
        if not thread and b'"thread.started"' in line:
            thread = (json.loads(line) or {}).get("thread_id")
    emit({"replay": "exit", "n": n, "code": p.wait()})
for path in sorted(glob.glob(os.path.join(home, "sessions", "**", "*.jsonl"), recursive=True))[:8]:
    with open(path, errors="replace") as f:
        for line in f:
            emit({"replay": "rollout", "name": os.path.basename(path), "line": line.rstrip("\n")})
emit({"replay": "end"})
'''


class Gate:
    """The sandbox's only network path, on the host: an HTTPS proxy on a Unix socket that terminates
    TLS. It accepts CONNECT to PROVIDER:443 only, then completes the handshake itself with a
    certificate from a CA made for this gate (the sandbox trusts that CA and nothing else; its key never
    enters the sandbox), refusing any other SNI. Each decrypted request must be one of PROVIDER_ROUTES,
    with well-formed, allowlisted header lines, no upgrade and no chunked or encoded body; a POST body
    is read whole and must be a model turn shaped as Codex sends it (model_request), forwarded as the
    gate's own serialization. The gate then sends that request, with Host
    PROVIDER and the host login's Authorization (the sandbox only has tokens with dummy signatures),
    over its own TLS connection to a public address of PROVIDER, one request per tunnel, within a
    request and byte budget per gate. A turn state is forwarded only on a POST and only as a value the
    provider itself issued to this gate (one gate per run). Everything else is refused and logged."""

    HOP = {"host", "connection", "content-length", "content-type", "authorization"}  # the headers the gate sets itself
    BODY_WAIT = 10  # seconds a body may pause

    def __init__(self, token: str | None = None, model_turns: bool = True):
        self.token = host_access_token() if token is None else token  # sent upstream; the sandbox holds a dummy
        self.model_turns = model_turns  # False: every POST is refused, so no model call is made
        self.requests = self.bytes = 0
        self.turn_states: set[str] = set()  # issued by the provider in this gate's replies
        base = HOME / ".vrooli/tmp"
        base.mkdir(parents=True, exist_ok=True)
        self.dir = Path(tempfile.mkdtemp(prefix="rg-", dir=base))  # short: a Unix socket path has 108 bytes
        write(self.dir / "bridge.py", BRIDGE)
        write(self.dir / "driver.py", DRIVER)
        self.ca = self.dir / "ca.pem"
        self.tls = self._issue()
        self.sock = self.dir / "gate.sock"
        self.log = []
        self.srv = socket.socket(socket.AF_UNIX)
        self.srv.bind(str(self.sock))
        self.srv.listen(64)
        threading.Thread(target=self._accept, daemon=True).start()

    def _issue(self) -> ssl.SSLContext:
        """A CA for this gate only and a PROVIDER certificate from it; the server context refuses other SNI."""
        d, ossl = self.dir, which("openssl") or "openssl"
        ec = ["-newkey", "ec", "-pkeyopt", "ec_paramgen_curve:P-256", "-nodes"]
        write(d / "leaf.ext", f"subjectAltName=DNS:{PROVIDER}\nextendedKeyUsage=serverAuth\nbasicConstraints=CA:FALSE\n")
        for cmd in ([ossl, "req", "-x509", *ec, "-keyout", "ca.key", "-out", "ca.pem", "-days", "3", "-subj",
                     "/CN=replay gate CA", "-addext", "basicConstraints=critical,CA:TRUE",
                     "-addext", "keyUsage=critical,keyCertSign"],
                    [ossl, "req", *ec, "-keyout", "leaf.key", "-out", "leaf.csr", "-subj", f"/CN={PROVIDER}"],
                    [ossl, "x509", "-req", "-in", "leaf.csr", "-CA", "ca.pem", "-CAkey", "ca.key", "-CAcreateserial",
                     "-out", "leaf.pem", "-days", "3", "-extfile", "leaf.ext"]):
            subprocess.run(cmd, cwd=d, check=True, capture_output=True)
        ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        ctx.load_cert_chain(d / "leaf.pem", d / "leaf.key")
        ctx.set_alpn_protocols(["http/1.1"])
        ctx.sni_callback = lambda sock, name, _: None if name == PROVIDER else ssl.ALERT_DESCRIPTION_UNRECOGNIZED_NAME
        return ctx

    @staticmethod
    def allowed(method: str, target: str) -> bool:
        path, sep, query = target.partition("?")
        pattern = PROVIDER_ROUTES.get((method, path))
        return (pattern is not None and (not sep or bool(pattern) and re.fullmatch(pattern, query) is not None)
                and not re.search(r"/\.|//|\\|%2e|%2f|%5c|[^\x21-\x7e]", target, re.I))

    @staticmethod
    def model_request(body: bytes) -> tuple[str | None, bytes]:
        """A /responses body as the gate forwards it: (None, the re-serialized request) when it is strict
        UTF-8 JSON without duplicate keys or NaN, shaped as Codex sends it (request_refusal), else the
        reason and nothing. Forwarding the gate's own serialization leaves no parser differential."""
        def pairs(kv):
            if len({k for k, _ in kv}) != len(kv):
                raise ValueError("duplicate key")
            return dict(kv)

        def number(x):
            if not math.isfinite(f := float(x)):  # NaN, Infinity, or 1e999
                raise ValueError(x)
            return f
        try:
            req = json.loads(body.decode("utf-8"), object_pairs_hook=pairs, parse_constant=number, parse_float=number)
        except (ValueError, RecursionError):
            return "body is not strict UTF-8 JSON without duplicate keys", b""
        why = request_refusal(req)
        return why, b"" if why else json.dumps(req, ensure_ascii=False).encode()

    @staticmethod
    def upstream() -> socket.socket:
        addrs = {a[4][0] for a in socket.getaddrinfo(PROVIDER, 443, type=socket.SOCK_STREAM)}
        if not addrs or not all(ipaddress.ip_address(a).is_global for a in addrs):
            raise OSError(f"{PROVIDER} resolves to a non-public address: {sorted(addrs)}")
        raw = socket.create_connection((sorted(addrs)[0], 443), timeout=30)
        ctx = ssl.create_default_context()
        ctx.set_alpn_protocols(["http/1.1"])
        tls = ctx.wrap_socket(raw, server_hostname=PROVIDER)
        tls.settimeout(900)
        return tls

    def _accept(self):
        while True:
            try:
                c, _ = self.srv.accept()
            except OSError:
                return
            threading.Thread(target=self._handle, args=(c,), daemon=True).start()

    @staticmethod
    def _head(c: socket.socket) -> tuple[str, str, list, bytes, bool]:
        """Read one request head; anything that does not start like HTTP is refused at once. The flag says
        whether every header line is `name: value` with a token name and no CR, LF or NUL (no obs-fold)."""
        buf = b""
        c.settimeout(60)
        while b"\r\n\r\n" not in buf and len(buf) < 65536 and re.match(rb"[A-Z]{3,7} |[A-Z]{0,6}$", buf):
            d = c.recv(65536)
            if not d:
                raise OSError("closed")
            buf += d
        c.settimeout(None)
        head, sep, rest = buf.partition(b"\r\n\r\n")
        lines = head.decode("latin1", "replace").split("\r\n")
        parts = lines[0].split(" ")
        if not sep or len(parts) != 3 or parts[2] != "HTTP/1.1":
            return "?", re.sub(r"[^\x20-\x7e]", "?", lines[0])[:40], [], b"", False
        valid = all(re.fullmatch(r"[!#$%&'*+.^_`|~0-9A-Za-z-]+:[^\r\n\x00]*", x) for x in lines[1:])
        return parts[0], parts[1], [(k.strip(), v.strip()) for k, _, v in (x.partition(":") for x in lines[1:])], rest, valid

    def _refuse(self, c: socket.socket, request: str):
        self.log.append({"t": now(), "request": request, "allowed": False})
        try:
            c.sendall(b"HTTP/1.1 403 Forbidden\r\nX-Replay-Gate: refused\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
        finally:
            c.close()

    def _handle(self, c: socket.socket):
        u = None
        try:
            method, target, _, rest, valid = self._head(c)
            if not valid or method != "CONNECT" or target != f"{PROVIDER}:443" or rest:
                return self._refuse(c, f"{method} {target[:160]}")
            c.sendall(b"HTTP/1.1 200 Connection established\r\n\r\n")
            try:
                c = self.tls.wrap_socket(c, server_side=True)  # the SNI callback refuses any other name
            except (OSError, ssl.SSLError) as e:
                self.log.append({"t": now(), "request": f"TLS refused: {type(e).__name__} {str(e)[:80]}", "allowed": False})
                raise
            method, target, headers, rest, valid = self._head(c)
            h = {k.lower(): v for k, v in headers}
            length = h.get("content-length", "0")
            request = f"{method} {target.split('?')[0][:160]}"
            names = [k.lower() for k, _ in headers]
            why = ("malformed head" if not valid else "route" if not self.allowed(method, target)
                   else "header" if set(names) - REQUEST_HEADERS or len(set(names)) != len(names)
                   or set(filter(None, re.split(r"\s*,\s*", h.get("x-codex-beta-features", "")))) - BETA_FEATURES
                   or TURN_STATE in h and (method != "POST" or h[TURN_STATE] not in self.turn_states)
                   else "framing" if {"transfer-encoding", "content-encoding", "upgrade"} & set(h) or not length.isdigit()
                   or int(length) > BODY_CAP or (method == "GET") == bool(int(length))
                   else "budget" if self.requests >= GATE_REQUESTS or self.bytes >= GATE_BYTES else None)
            body = rest[:int(length)] if not why else b""
            c.settimeout(self.BODY_WAIT)  # a body shorter than its Content-Length ends here
            while not why and len(body) < int(length):
                try:
                    d = c.recv(min(65536, int(length) - len(body)))
                except TimeoutError:
                    d = b""
                why = None if d else "body shorter than its Content-Length"
                body += d
            c.settimeout(None)
            if method == "POST" and not why:
                why, body = self.model_request(body) if self.model_turns else ("model turns off", b"")
            if why:
                return self._refuse(c, f"{request} ({why})")
            self.requests += 1
            self.bytes += len(body)
            self.log.append({"t": now(), "request": request, "allowed": True})
            u = self.upstream()
            out = [f"{method} {target} HTTP/1.1", f"Host: {PROVIDER}",
                   *(f"{k}: {v}" for k, v in headers if k.lower() not in self.HOP),
                   *([f"Authorization: Bearer {self.token}"] if self.token else []), "Connection: close",
                   *(["Content-Type: application/json", f"Content-Length: {len(body)}"] if body else [])]
            u.sendall(("\r\n".join(out) + "\r\n\r\n").encode("latin1") + body)  # bytes past the body are dropped
        except (OSError, ssl.SSLError, ValueError, IndexError):
            for x in (c, u):
                if x is not None:
                    x.close()
            return
        try:  # the answer, within the byte budget; the turn states its head issues are kept
            head = b""
            while (d := u.recv(65536)) and self.bytes + len(d) <= GATE_BYTES:
                if head is not None and len(head) < 1 << 16:
                    head += d
                    if b"\r\n\r\n" in head:  # the whole head: 1 to 2048 visible ASCII bytes per value
                        self.turn_states.update(v.decode() for v in re.findall(
                            rb"(?im)^x-codex-turn-state:[ \t]*([\x21-\x7e]{1,2048})[ \t]*\r\n", head.partition(b"\r\n\r\n")[0] + b"\r\n"))
                        head = None
                self.bytes += len(d)
                c.sendall(d)
        except OSError:
            pass
        for x in (u, c):
            try:
                x.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            x.close()

    def summary(self) -> dict:
        allowed: dict[str, int] = {}
        for e in self.log:
            if e["allowed"]:
                allowed[e["request"]] = allowed.get(e["request"], 0) + 1
        return {"allowed": allowed, "denied": [e["request"] for e in self.log if not e["allowed"]],
                "requests": self.requests, "bytes": self.bytes}

    def close(self):
        try:
            self.srv.close()
        finally:
            shutil.rmtree(self.dir, ignore_errors=True)

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.close()


def which(name: str) -> str | None:
    return shutil.which(name)


def codex_install() -> tuple[Path, Path]:
    """The newest installed Codex binary and the version folder to mount (the ~/.vrooli/shims/codex
    launcher needs host state the sandbox does not have)."""
    root = HOME / ".local/share/vrooli/resources/codex/cli"
    found = sorted(root.glob("*/node_modules/@openai/codex-linux-x64/vendor/*/bin/codex"),
                   key=lambda p: [int(x) for x in re.findall(r"\d+", p.relative_to(root).parts[0])[:3]])
    if found:
        return found[-1], root / found[-1].relative_to(root).parts[0]
    real = Path(os.path.realpath(which("codex") or "/nonexistent"))
    if not real.is_file():
        raise Fail("no Codex binary found")
    return real, real.parent


def sandbox_cmd(inner: list[str], *, ws: WS, variant: str | None = None, cohort: str | None = None,
                gate: Gate | None = None, cwd: Path | None = None, extra: tuple[str, ...] = ()) -> list[str]:
    """bubblewrap with every namespace unshared (no network but loopback, own pids). The root holds only
    /usr, the ETC_ALLOW part of /etc and the Codex install, read-only; /tmp is a private capped tmpfs and the home folder an
    empty read-only tmpfs. An agent run adds, read-only: the snapshot (allowlisted repository copy,
    research home, skill copies, offline CLIs), the variant's research skills and its own commission
    folder. `gate` adds the provider path and a capped tmpfs Codex home that DRIVER fills: inner is
    then run under the bridge, so it should be DRIVER (`python3 -I /tmp/.gate/driver.py`)."""
    bw = which("bwrap")
    if not bw:
        raise Fail("bwrap (bubblewrap) is not installed; the replay refuses to run unsandboxed")
    a = [bw, "--unshare-all", "--die-with-parent", "--new-session", "--ro-bind", "/usr", "/usr"]
    for name in ("bin", "lib", "lib64", "sbin"):
        p = Path("/") / name
        if p.is_symlink():
            a += ["--symlink", os.readlink(p), str(p)]
        elif p.is_dir():
            a += ["--ro-bind", str(p), str(p)]
    a += [x for n in ETC_ALLOW if os.path.lexists(f"/etc/{n}") for x in ("--ro-bind", f"/etc/{n}", f"/etc/{n}")]
    a += ["--proc", "/proc", "--dev", "/dev", "--size", str(SHM_CAP), "--tmpfs", "/dev/shm",
          "--size", str(TMP_CAP), "--tmpfs", "/tmp",
          "--tmpfs", "/home", "--dir", str(HOME)]
    _, install = codex_install()
    for p in (install, HOME / ".vrooli/opt/node"):
        if p.is_dir():
            a += ["--ro-bind", str(p), str(p)]
    if variant:
        snap, research = ws.snapshot(), research_home(ws)
        if not (snap / "repo/AGENTS.md").is_file():
            raise Fail(f"{snap}/repo has no repository copy; take a new snapshot")
        a += ["--ro-bind", str(snap / "repo"), str(REPO),
              "--ro-bind", str(ws.variant(variant) / "skills"), str(SKILLS),
              "--ro-bind", str(snap / "home"), str(research), "--tmpfs", str(research / "cohorts"),
              "--ro-bind", str(snap / "bin"), str(HOME / ".vrooli/bin"),
              "--ro-bind", str(snap / "skills"), str(SKILL_EXPORT)]
        if cohort:
            a += ["--ro-bind", str(ws.variant(variant) / "cohorts" / cohort), str(research / "cohorts" / cohort)]
        a += ["--remount-ro", str(research / "cohorts")]
    if gate:
        a += ["--size", str(CODEX_HOME_CAP), "--tmpfs", str(HOME / ".codex")]
        for name in CODEX_SHARED:
            if (HOME / ".codex" / name).is_dir():
                a += ["--ro-bind", str(HOME / ".codex" / name), str(HOME / ".codex" / name)]
        a += [x for f in ("bridge.py", "driver.py", "ca.pem") for x in ("--ro-bind", str(gate.dir / f), f"/tmp/.gate/{f}")]
        a += ["--ro-bind", str(gate.sock), "/tmp/.gate/gate.sock"]  # connecting needs no write access to the mount
        inner = ["python3", "-I", "/tmp/.gate/bridge.py", str(PROXY_PORT), "/tmp/.gate/gate.sock", "--", *inner]
    a += [*extra, "--remount-ro", "/home", "--remount-ro", "/dev", "--remount-ro", "/",
          "--chdir", str(cwd or (REPO if variant else HOME)), "--"]
    return a + inner


def sandbox_env() -> dict:
    """The whole environment inside the sandbox; nothing of the caller's leaks in. Every HTTPS client
    goes through the bridge to the gate and trusts the gate's CA alone."""
    url, ca = f"http://127.0.0.1:{PROXY_PORT}", "/tmp/.gate/ca.pem"
    return {"HOME": str(HOME), "USER": os.environ.get("USER", "user"), "LOGNAME": os.environ.get("USER", "user"),
            "PATH": f"{HOME}/.vrooli/bin:/usr/local/bin:/usr/bin:/bin", "LANG": "C.UTF-8", "TERM": "dumb",
            "CODEX_HOME": str(HOME / ".codex"), "HTTPS_PROXY": url, "https_proxy": url, "HTTP_PROXY": url,
            "http_proxy": url, "ALL_PROXY": url, "NO_PROXY": "localhost,127.0.0.1", "no_proxy": "localhost,127.0.0.1",
            "CODEX_CA_CERTIFICATE": ca, "SSL_CERT_FILE": ca}


def auth_check() -> None:
    """The sandbox cannot refresh the Codex login (the refresh host is not reachable from it, by design):
    refuse to start unless the host token stays valid well past a run."""
    import base64
    try:
        auth = json.loads((HOME / ".codex/auth.json").read_text(encoding="utf-8"))
        part = auth["tokens"]["access_token"].split(".")[1]
        exp = json.loads(base64.urlsafe_b64decode(part + "=" * (-len(part) % 4)))["exp"]
        last = dt.datetime.strptime(auth["last_refresh"][:19], "%Y-%m-%dT%H:%M:%S").replace(tzinfo=dt.timezone.utc)
    except (OSError, KeyError, ValueError, IndexError) as e:
        raise Fail(f"cannot read the Codex login in ~/.codex/auth.json: {e}")
    if exp - time.time() < 86400 or (dt.datetime.now(dt.timezone.utc) - last).days >= 7:
        raise Fail("the Codex login is due for refresh; run any codex command on the host first (the sandbox "
                   "cannot refresh it)")


def codex_header(turns: list[list[str]]) -> dict:
    """DRIVER's header: the turns, and the Codex home it writes into the sandbox's own tmpfs: the config
    and install files, and the login with its refresh token blanked and its tokens' signatures replaced
    (Codex reads their claims; the gate sends the real access token). No usable credential enters the
    sandbox, and no copy of the login is written to the host disk."""
    src, files = HOME / ".codex", {}
    for name in CODEX_FILES:
        if (src / name).is_file():
            files[name] = (src / name).read_text(encoding="utf-8", errors="replace")
    auth = read_json(src / "auth.json")
    if auth:
        tokens = auth.get("tokens") or {}
        auth["tokens"] = {**tokens, "refresh_token": "", **{k: dummy_jwt(tokens[k]) for k in ("access_token", "id_token")
                                                           if isinstance(tokens.get(k), str)}}
        files["auth.json"] = json.dumps(auth)
    return {"turns": turns, "files": files, "dirs": ["log", "logs", "tmp", "outputs", "shell_snapshots", "cache", "sessions"]}


DRIVE = ["python3", "-I", "/tmp/.gate/driver.py"]
DUMMY_SIGNATURE = "replay-dummy-signature"


def dummy_jwt(token: str) -> str:
    return ".".join(token.split(".")[:2] + [DUMMY_SIGNATURE])


def request_refusal(req) -> str | None:
    """Why a parsed /responses body is not a model turn as Codex sends it, or None: only REQUEST_KEYS,
    no tool choice but auto, no stored response, and an input of ITEM_KEYS items whose parts are
    PART_KEYS (an image only as a data: URL) and whose tools are TOOL_KEYS, none run by the provider."""
    def flat(d, keys=None):
        return isinstance(d, dict) and (keys is None or set(d) <= keys) and \
            all(isinstance(v, (str, bool, int, float)) for v in d.values())

    def text(v, none=True):
        return isinstance(v, str) or (none and v is None)

    def tools_refusal(tools, nested=False):
        for t in tools if isinstance(tools, list) else [None]:
            kind = t.get("type") if isinstance(t, dict) else None
            if why := kind == "namespace" and not nested and tools_refusal(t.get("tools"), True):
                return why
            if kind not in TOOL_KEYS or set(t) - TOOL_KEYS[kind] or (kind == "namespace" and nested) \
                    or not text(t.get("name"), False) or not text(t.get("description")) \
                    or (kind == "custom" and not flat(t.get("format"), {"type", "syntax", "definition"})) \
                    or (kind == "function" and (not isinstance(t.get("strict", False), bool) or remote_ref(t.get("parameters")))):
                return f"tool {str(kind)[:30]}"
        return None

    def remote_ref(v) -> bool:  # a JSON Schema reference the provider might resolve
        if isinstance(v, dict):
            return any((k in ("$ref", "$id") and "://" in str(x)) or remote_ref(x) for k, x in v.items())
        return isinstance(v, list) and any(remote_ref(x) for x in v)

    def part_refusal(part):
        kind = part.get("type") if isinstance(part, dict) else None
        if kind not in PART_KEYS or set(part) - PART_KEYS[kind] or not text(part.get("text")):
            return f"part {str(kind)[:30]}"
        if kind == "input_image" and not re.match(r"data:image/(png|jpeg|webp|gif);base64,", str(part.get("image_url"))):
            return "image not a data: URL of a raster image"
        if kind == "output_text" and (part.get("annotations") or []) != []:
            return "annotations"
        return None

    if not isinstance(req, dict) or set(req) - REQUEST_KEYS:
        return "keys " + ",".join(sorted(set(req) - REQUEST_KEYS) if isinstance(req, dict) else ["?"])[:80]
    for key, ok in (("model", text(req.get("model"), False)), ("tool_choice", req.get("tool_choice", "auto") == "auto"),
                    ("store", req.get("store", False) is False), ("stream", isinstance(req.get("stream", True), bool)),
                    ("parallel_tool_calls", isinstance(req.get("parallel_tool_calls", False), bool)),
                    ("include", req.get("include", []) in ([], ["reasoning.encrypted_content"])),
                    ("text", flat(req.get("text", {}), {"verbosity"})), ("prompt_cache_key", text(req.get("prompt_cache_key"))),
                    ("reasoning", flat(req.get("reasoning", {}), {"effort", "context", "summary"})),
                    ("client_metadata", flat(req.get("client_metadata", {}))), ("access_programs", req.get("access_programs", {}) in ({}, {"cyber": "standard"})),
                    ("input", isinstance(req.get("input"), list))):
        if not ok:
            return key
    for item in req["input"]:
        kind = item.get("type") if isinstance(item, dict) else None
        if kind not in ITEM_KEYS or set(item) - ITEM_KEYS[kind]:
            return f"item {str(kind)[:30]}"
        meta = json.dumps(item.get(META))
        if "://" in meta or not all(text(item.get(k)) for k in ("id", "call_id", "name", "arguments", "input", "status",
                                                                  "phase", "namespace", "encrypted_content", "role")):
            return f"{kind} fields"
        if kind == "message" and item.get("role") not in ("developer", "user", "assistant"):
            return "role"
        if kind == "additional_tools" and (item.get("role") != "developer" or tools_refusal(item.get("tools"))):
            return tools_refusal(item.get("tools")) or "role"
        if kind == "configuration_update" and not flat(item.get("reasoning"), {"effort", "context", "summary"}):
            return "configuration_update"
        if kind == "reasoning" and item.get("content") is not None:
            return "reasoning content"
        parts = {"message": item.get("content"), "reasoning": item.get("summary", [])}.get(kind, item.get("output", []))
        if kind.endswith("_output") and isinstance(parts, str):
            continue
        for part in parts if isinstance(parts, list) else [None]:
            if why := part_refusal(part):
                return why
    return None


def host_access_token() -> str:
    return ((read_json(HOME / ".codex/auth.json") or {}).get("tokens") or {}).get("access_token") or ""


def subagent_instructions() -> str | None:
    import tomllib
    try:
        cfg = tomllib.loads((HOME / ".codex/config.toml").read_text(encoding="utf-8"))
        return cfg["features"]["multi_agent_v2"]["subagent_developer_instructions"]
    except (OSError, KeyError, ValueError):
        return None


# The model API over HTTPS only. Codex's built-in provider opens a WebSocket first; the gate refuses the
# upgrade and Codex falls back to HTTPS after five failures, and turns after that fallback came back with
# no reasoning and few tool calls (diagnostic-transport2: host through the gate 0/3 engaged, the sandbox
# with this provider 3/3). This provider is the built-in one with WebSockets switched off, so no turn ever
# tries the upgrade; a run that does is refused (websocket_attempts).
HTTPS_PROVIDER = ("-c", 'model_providers.openai_https={name="OpenAI (HTTPS only)", '
                        'base_url="https://chatgpt.com/backend-api/codex", wire_api="responses", '
                        'requires_openai_auth=true, supports_websockets=false}',
                  "-c", 'model_provider="openai_https"')
WS_ROUTE = "GET /backend-api/codex/responses"  # a WebSocket upgrade, as the gate logs it


def websocket_attempts(network: dict, stderr: str = "", events: list | None = None) -> list[str]:
    """Every sign that a run tried the model API over a WebSocket: an upgrade the gate refused, Codex's
    connect failure on stderr, or its fallback notice in the event stream. Empty when it used HTTPS only."""
    seen = [r for r in network.get("denied", []) if r.startswith(WS_ROUTE)]
    seen += [f"stderr: {m}" for m in sorted(set(re.findall(r"failed to connect to Responses WebSocket", stderr)))]
    if any("Falling back from WebSockets" in json.dumps(e) for e in events or []):
        seen.append("events: Falling back from WebSockets")
    return seen


def codex_turn(prompt: str, *, model: str, effort: str, cwd: Path, resume: str | None, dev: str | None,
               extra: tuple[str, ...] = ()) -> list[str]:
    codex, _ = codex_install()
    cfg = [*extra, *HTTPS_PROVIDER, "--skip-git-repo-check", "-c", 'approval_policy="never"', "-c", 'sandbox_mode="danger-full-access"',
           "-c", 'web_search="disabled"', "-c", "features.image_generation=false",  # no tool that runs on the provider's side
           "-c", "features.enable_request_compression=false",  # bodies the gate can read
           "-c", f"model_reasoning_effort={json.dumps(effort)}", "-m", model]
    if dev:
        cfg += ["-c", f"developer_instructions={json.dumps(dev)}"]
    if resume:
        return [str(codex), "exec", "resume", resume, "--json", *cfg, prompt]
    return [str(codex), "exec", "--json", "-C", str(cwd), *cfg, prompt]


SPAWN_TYPES = re.compile(r"collab|spawn|agent(?!_message)", re.I)


def stream(cmd: list[str], header: dict, env: dict, deadline: float, out, t0: float,
           rollouts: Path | None = None) -> dict:
    """Run DRIVER's turns in one sandbox, appending {"t", "turn", "e"} lines to `out` and the session
    rollouts to `rollouts`. A turn ends when the agent has answered: its first message before the last
    turn (the stance turn, where an agent told to wait for its goal otherwise sleeps for an hour), its
    replay JSON in the last. A turn also ends at the deadline and three minutes after the agent starts a
    sub-agent (the protocol forbids it), and either ends the run; so does an event stream past OUTPUT_CAP."""
    proc = subprocess.Popen(cmd, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env,
                            text=True, start_new_session=True)
    last = len(header["turns"])
    st: dict = {"turns": [], "spawn_at": None, "answered_at": None}
    err: collections.deque = collections.deque(maxlen=16)
    threading.Thread(target=lambda: err.extend(iter(lambda: proc.stderr.read(4096), "")), daemon=True).start()

    def send(line: str):
        try:
            proc.stdin.write(line + "\n")
            proc.stdin.flush()
        except (OSError, ValueError):
            pass

    def watchdog():
        while proc.poll() is None:
            t = st["turns"][-1] if st["turns"] else None
            limit = min(deadline, st["spawn_at"] + 180) if st["spawn_at"] else deadline
            if st["answered_at"]:
                limit = min(limit, st["answered_at"] + 8)
            if t and t["killed"] is None and time.time() > limit:
                t["killed"] = ("answered" if st["answered_at"] and time.time() < deadline else
                               "timeout" if time.time() > deadline else "spawned")
                send(f"{'stop' if t['killed'] == 'answered' and t['turn'] < last else 'quit'} {t['turn']}")
            if time.time() > deadline + 60:
                os.killpg(proc.pid, signal.SIGKILL)
                return
            time.sleep(1)

    send(json.dumps(header))
    threading.Thread(target=watchdog, daemon=True).start()
    written, files = 0, {}
    try:
        for line in iter(lambda: proc.stdout.readline(16 << 20), ""):
            written += len(line)
            if written > OUTPUT_CAP:
                if st["turns"]:
                    st["turns"][-1]["killed"] = "output"
                os.killpg(proc.pid, signal.SIGKILL)
                break
            try:
                ev = json.loads(line)
            except (ValueError, RecursionError):
                ev = {"raw": line.rstrip("\n")} if line.strip() else None
            if not isinstance(ev, dict):
                continue
            kind = ev.get("replay")
            if kind == "turn":
                st["turns"].append({"turn": ev.get("n"), "exit": None, "killed": None, "thread": None})
                st["spawn_at"] = st["answered_at"] = None
                continue
            if kind == "exit" and st["turns"]:
                st["turns"][-1]["exit"] = ev.get("code")
            if kind == "rollout" and rollouts is not None:
                name = re.sub(r"[^A-Za-z0-9-]", "", Path(str(ev.get("name"))).stem)[-12:] or "x"
                if name not in files and len(files) < 8:
                    files[name] = gzip.open(rollouts / f"rollout-{name}.jsonl.gz", "wt", encoding="utf-8")
                if name in files:
                    files[name].write(str(ev.get("line")) + "\n")
            if kind:
                continue
            t = st["turns"][-1] if st["turns"] else {"turn": 0}
            out.write(json.dumps({"t": round(time.time() - t0, 3), "turn": t["turn"], "e": ev}, ensure_ascii=False) + "\n")
            out.flush()
            if ev.get("type") == "thread.started" and "thread" in t:
                t["thread"] = ev.get("thread_id")
            item = ev.get("item") or {}
            if item.get("type") and SPAWN_TYPES.search(item["type"]) and not st["spawn_at"]:
                st["spawn_at"] = time.time()
            if ev.get("type") == "turn.completed":
                st["answered_at"] = None  # the turn ended by itself
            elif item.get("type") == "agent_message" and not st["answered_at"] and \
                    (t["turn"] < last or (extract_json(item.get("text", "")) or {}).get("role")):
                st["answered_at"] = time.time()
    finally:
        for f in files.values():
            f.close()
        try:
            proc.stdin.close()
        except OSError:
            pass
        proc.wait()
        proc.stdout.close()
        proc.stderr.close()
    return {"exit": proc.returncode, "turns": st["turns"], "rollouts": len(files), "stderr": "".join(err)[-20000:]}


def next_run_dir(ws: WS, key: str) -> Path:
    ws.runs().mkdir(parents=True, exist_ok=True)
    n = 1
    while True:
        d = ws.runs() / f"{key}.{n}"
        try:
            d.mkdir()
            return d
        except FileExistsError:
            n += 1


def prompt_info(ws: WS, cid: str, variant: str) -> dict:
    return read_json(ws.prompts(cid, variant) / ("freeze.json" if variant == "old" else "render.json")) or {}


def role_model(ws: WS, cid: str, variant: str) -> tuple[str, str]:
    spec = prompt_info(ws, cid, variant).get("spec") or {}
    limits = read_json(ws.variant(variant) / "skills/research-coordination/templates/limits.json") or {}
    d = limits.get("defaults", {})
    return (spec.get("supervisor_model") or d.get("supervisor_model") or "gpt-6-luna",
            spec.get("supervisor_effort") or d.get("supervisor_effort") or "medium")


def owner_brief_file(pdir: Path, topic: str, card: int | None) -> Path:
    """The topic's owner brief: the given card, else its first card by number (one owner per ordered
    workstream gives several)."""
    files = sorted(pdir.glob(f"owner-{topic}-card*.txt"), key=lambda f: (int(re.findall(r"card(\d+)", f.name)[-1]), "-owner" in f.name, f.name))
    files = [f for f in files if card is None or f.name == f"owner-{topic}-card{card}.txt"]
    if not files:
        raise Fail(f"no owner brief for topic {topic!r} in {pdir}")
    return files[0]


def cmd_run(ws: WS, cid: str, variant: str, role: str, topic: str | None, card: int | None, timeout: int,
            model_override: str | None) -> dict:
    pdir = ws.prompts(cid, variant)
    info = prompt_info(ws, cid, variant)
    if not info:
        raise Fail(f"{pdir} has no freeze.json or render.json: run freeze or render-new first")
    auth_check()
    model, effort = role_model(ws, cid, variant)  # owners inherit the supervisor's model and effort
    if role == "supervisor":
        turns = [pdir.joinpath("stance.txt").read_text(encoding="utf-8"),
                 pdir.joinpath("kernel.txt").read_text(encoding="utf-8") + "\n\n" + PROTOCOL["supervisor"]]
        key, dev = f"{cid}.{variant}.supervisor", None
    else:
        brief = owner_brief_file(pdir, topic, card)
        turns = [brief.read_text(encoding="utf-8") + "\n\n" + PROTOCOL["owner"]]
        key, dev = f"{cid}.{variant}.owner.{brief.stem.removeprefix('owner-')}", subagent_instructions()
    model = model_override or model
    run_dir = next_run_dir(ws, key)
    codex, _ = codex_install()
    agents_md = (ws.snapshot() / "repo/AGENTS.md").read_bytes() if (ws.snapshot() / "repo/AGENTS.md").exists() else b""
    meta = {"run_id": run_dir.name, "commission_id": cid, "cohort": info.get("cohort") or cid, "variant": variant,
            "role": role, "topic": topic, "harness": "codex", "codex": str(codex), "model": model, "effort": effort,
            "timeout_s": timeout, "started_at": now(), "cwd": str(REPO), "sandbox": SANDBOX_NOTE,
            "agents_md": {"bytes": len(agents_md), "sha": sha(agents_md)},
            "skill_sizes_total": skill_sizes(ws.variant(variant) / "skills")["total"],
            "prompt_bytes": [len(t.encode()) for t in turns], "turns": []}
    for i, t in enumerate(turns, 1):
        write(run_dir / f"prompt-{i}.txt", t)
    t0 = time.time()
    argvs = [codex_turn(t, model=model, effort=effort, cwd=REPO, resume="{thread}" if i else None, dev=dev)
             for i, t in enumerate(turns)]
    gate = Gate()
    try:
        with open(run_dir / "events.jsonl", "w", encoding="utf-8") as out:
            cmd = sandbox_cmd(DRIVE, ws=ws, variant=variant, cohort=meta["cohort"], gate=gate)
            res = stream(cmd, codex_header(argvs), sandbox_env(), t0 + timeout, out, t0, rollouts=run_dir)
        meta["turns"], meta["rollouts"], meta["stderr_tail"] = res["turns"], res["rollouts"], res["stderr"][-2000:]
        write(run_dir / "stderr.txt", res["stderr"])
    finally:
        gate.close()
        meta["network"] = gate.summary()
        meta["wall_s"] = round(time.time() - t0, 1)
        meta["ended_at"] = now()
        events = [json.loads(x) for x in (run_dir / "events.jsonl").read_text(encoding="utf-8").splitlines()
                  if x.strip()] if (run_dir / "events.jsonl").exists() else []
        err = (run_dir / "stderr.txt").read_text(encoding="utf-8", errors="replace") if (run_dir / "stderr.txt").exists() else ""
        meta["websocket_attempts"] = websocket_attempts(meta["network"], err, events)
        if meta["websocket_attempts"]:
            meta["excluded"] = "the run tried a WebSocket; replay turns run over HTTPS only"
        write_json(run_dir / "meta.json", meta)
    if meta["websocket_attempts"]:
        raise Fail(f"{run_dir.name} tried a WebSocket ({meta['websocket_attempts'][0]}); it is excluded")
    metrics = analyse_run(run_dir)
    return {"run": str(run_dir), "wall_s": meta["wall_s"], "turns": [{k: t[k] for k in ("turn", "exit", "killed")}
                                                                     for t in meta["turns"]],
            "bytes_seen": metrics.get("bytes_seen_before_first_action"), "bytes_raw": metrics["bytes_before_first_action"],
            "tokens": metrics["tokens"], "plan_ok": metrics["plan_ok"], "first_action": metrics["first_action"],
            "network_denied": meta["network"]["denied"]}


def run_jobs(jobs: list, worker, parallel: int) -> list:
    """Run jobs `parallel` at a time, in the given order; each result or error is kept."""
    from concurrent.futures import ThreadPoolExecutor
    def safe(job):
        try:
            return {"job": job, "result": worker(job)}
        except Exception as e:  # one failed job must not stop the batch; the error is recorded
            return {"job": job, "error": f"{type(e).__name__}: {e}"}
    with ThreadPoolExecutor(max_workers=max(1, parallel)) as pool:
        return list(pool.map(safe, jobs))


def batch_jobs(ws: WS, cids: list[str], variants: list[str], roles: list[str], owner_topics: list[str],
               repeat: int) -> list[dict]:
    """Every (commission, role, topic) prompt for every variant, `repeat` times, interleaved so each
    repetition runs every variant before the next starts: time-of-day and service drift fall on both."""
    prompts = []
    for cid in cids:
        topics = (prompt_info(ws, cid, "old").get("topics") or [])
        if "supervisor" in roles:
            prompts.append((cid, "supervisor", None))
        if "owner" in roles:
            prompts += [(cid, "owner", t) for t in topics if t in owner_topics]
    jobs = []
    for i in range(repeat):
        for cid, role, topic in prompts:
            order = variants if i % 2 == 0 else list(reversed(variants))
            jobs += [{"cid": cid, "variant": v, "role": role, "topic": topic} for v in order]
    return jobs


# ---------------------------------------------------------------- parsing and metrics


WRITE_CMD = re.compile(
    r"(^|[;&|(]\s*|\bsudo\s+)(mkdir|touch|tee|mv|cp|rm|ln|install|truncate|chmod)\s"
    r"|(?<![<>&2])>{1,2}\s*(?!/dev/null|&)[^\s&|;]"
    r"|\bsed\s+-i|\bapply_patch\b|\bsnapshot\s+--out\b|needs_you\.py\s+write|\bgit\s+(commit|add|checkout|reset)\b")
MUTATING_ROUTE = re.compile(r"\b(research-desk\s+(say|launch|build|control|answer|verdict|courier|mark-seen|enable-steering)"
                            r"|notification-hub\s+\S*\s*(send|notify|create)|agent-manager\s+\S*\s*(run|start|stop|send)"
                            r"|prompt-manager\s+\S*\s*(add|create|update|delete|sync|rate|run|heartbeat-trigger)"
                            r"|vrooli\s+scenario\s+(start|stop|restart))\b")
ROUTE_FAIL = re.compile(r"Unable to reach|unavailable|connection refused|not admitted|Runtime error|"
                        r"no such host|timed out|operation not permitted|i/o timeout", re.I)
VALUED_FLAGS = ("--lines", "--limit", "--type", "--group", "--format", "-n")
QUOTED = re.compile(r"'[^']*'|\"(?:[^\"\\\\]|\\\\.)*\"")


# Python that opens a file for writing (heredocs and -c scripts keep their quotes, so match the raw text).
PY_WRITE = re.compile(r"\bpython3?\b[\s\S]*(\.write_text\(|\.write_bytes\(|\bopen\([^,)]*,\s*(mode\s*=\s*)?['\"][wax]b?\+?['\"]"
                      r"|\.open\(\s*(mode\s*=\s*)?['\"][wax]|shutil\.(copy|move|rmtree)|os\.(remove|unlink|rename|replace|makedirs|mkdir)\()")


def is_write(command: str, output: str = "") -> bool:
    bare = QUOTED.sub("''", command)
    return bool(WRITE_CMD.search(bare) or MUTATING_ROUTE.search(bare) or PY_WRITE.search(command)
                or "Read-only file system" in output)


WRAPPER = re.compile(r"^(?:/usr)?/bin/(?:ba|z)?sh\s+-l?c\s+")


def unwrap(command: str) -> str:
    if WRAPPER.match(command):
        try:
            parts = shlex.split(command)
            return parts[-1] if len(parts) >= 3 else command
        except ValueError:
            return WRAPPER.sub("", command)
    return command


def sandbox_view(ws: WS, variant: str, cohort: str | None) -> list[tuple[str, Path]]:
    """Where a path inside a run's sandbox lives on the host (sandbox_cmd's mounts), longest prefix first:
    a run's files exist only there, so the host's own files say nothing about what it read."""
    snap, research = ws.snapshot(), research_home(ws)
    view = {str(REPO): snap / "repo", str(SKILLS): ws.variant(variant) / "skills", str(research): snap / "home",
            str(research / "cohorts"): Path("/nonexistent"), str(HOME / ".vrooli/bin"): snap / "bin",
            str(SKILL_EXPORT): snap / "skills"}
    if cohort:
        view[str(research / "cohorts" / cohort)] = ws.variant(variant) / "cohorts" / cohort
    return sorted(view.items(), key=lambda kv: -len(kv[0]))


def on_host(path: Path, view: list[tuple[str, Path]] | None) -> Path:
    for prefix, host in view or []:
        if str(path) == prefix or str(path).startswith(prefix + "/"):
            return host / os.path.relpath(path, prefix)
    return path


def command_paths(command: str, cwd: Path, view: list[tuple[str, Path]] | None = None) -> list[str]:
    """Files a shell command reads: existing file paths among its arguments (checked where the run's
    sandbox view puts them), skill reads and route CLIs."""
    inner = unwrap(command)
    out: list[str] = []
    try:
        lex = shlex.shlex(inner, posix=True, punctuation_chars=";|&<>()")
        lex.whitespace_split = True
        tokens = list(lex)
    except ValueError:
        tokens = inner.split()
    prog_next = True
    for i, tok in enumerate(tokens):
        if tok in (";", "|", "||", "&&", "&", "(", ")"):
            prog_next = True
            continue
        if prog_next:
            prog_next = False
            name = os.path.basename(tok)
            if name in ROUTE_CLIS:
                rest = [t for t in tokens[i + 1:i + 6] if t not in (";", "|", "&&", "||")]
                if name == "prompt-manager" and rest[:2] == ["skill", "read"]:
                    ids = []
                    for t in tokens[i + 3:]:
                        if t in (";", "|", "||", "&&", "&", ">", ">>", "<", "(", ")"):
                            break
                        ids.append(t)
                    out += [f"skill:{t}" for j, t in enumerate(ids)
                            if not t.startswith("-") and not (j and ids[j - 1] in VALUED_FLAGS)]
                else:
                    verb = next((t for t in rest if not t.startswith("-")), "")
                    out.append(f"cli:{name} {verb}".strip())
            elif name.startswith("python") or name in ("bash", "sh", "node"):
                prog_next = False
            continue
        if tok.startswith("-") or tok in ("<", ">", ">>"):
            continue
        p = Path(os.path.expanduser(tok))
        p = Path(os.path.normpath(p if p.is_absolute() else cwd / p))
        try:
            if ("/" in tok or "." in tok) and on_host(p, view).is_file():
                out.append(str(p))
        except OSError:
            pass
    return list(dict.fromkeys(out))


def normalize(events: list[dict], cwd: Path, workdirs: dict[str, str] | None = None,
              view: list[tuple[str, Path]] | None = None) -> tuple[list[dict], dict, list[str]]:
    """Codex `exec --json` events into steps:
    {kind: read|write|spawn|message|other, t, command, paths, bytes, exit, text}. Returns steps,
    token usage and the agent's messages. A command's relative paths resolve against the working
    folder the agent gave it (workdirs, from the rollout: the event stream does not carry it), else cwd."""
    steps: list[dict] = []
    usage = {"input": 0, "cached_input": 0, "output": 0, "reasoning": 0}
    messages: list[str] = []
    for rec in events:
        e, t = rec.get("e", rec), rec.get("t")
        typ = e.get("type")
        if typ == "turn.completed":
            u = e.get("usage") or {}
            usage["input"] += u.get("input_tokens", 0)
            usage["cached_input"] += u.get("cached_input_tokens", 0)
            usage["output"] += u.get("output_tokens", 0)
            usage["reasoning"] += u.get("reasoning_output_tokens", 0)
        elif typ == "item.completed":
            it = e.get("item") or {}
            ityp = it.get("type", "")
            if ityp == "agent_message":
                messages.append(it.get("text", ""))
            elif ityp == "reasoning":
                continue
            elif ityp == "command_execution":
                cmd = it.get("command", "")
                output = it.get("aggregated_output") or ""
                inner = unwrap(cmd)
                kind = "write" if is_write(inner, output) else "read"
                where = cwd / (workdirs or {}).get(inner, "")
                steps.append({"kind": kind, "t": t, "command": inner, "paths": command_paths(cmd, where, view),
                              "bytes": len(output.encode()), "exit": it.get("exit_code"), "output_head": output[:300]})
            elif ityp in ("file_change", "patch_apply"):
                steps.append({"kind": "write", "t": t, "command": ityp,
                              "paths": [c.get("path", "") for c in it.get("changes", []) if isinstance(c, dict)],
                              "bytes": 0, "exit": None})
            elif ityp in ("todo_list", "web_search", "error"):  # an error item (a transport fallback) is no action
                steps.append({"kind": "other", "t": t, "command": ityp, "paths": [], "bytes": 0, "exit": None})
            else:
                kind = "spawn" if SPAWN_TYPES.search(ityp) else ("message" if "message" in ityp else "write")
                steps.append({"kind": kind, "t": t, "command": ityp + " " + json.dumps(it)[:300], "paths": [],
                              "bytes": 0, "exit": None})
    return steps, usage, messages


def extract_json(text: str) -> dict | None:
    """The last top-level JSON object in a message (fenced or bare)."""
    best, i = None, 0
    while (i := text.find("{", i)) != -1:
        depth, instr, esc, end = 0, False, False, None
        for j in range(i, len(text)):
            ch = text[j]
            if instr:
                if esc:
                    esc = False
                elif ch == "\\":
                    esc = True
                elif ch == '"':
                    instr = False
            elif ch == '"':
                instr = True
            elif ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if depth == 0:
                    end = j
                    break
        if end is None:
            break
        try:
            obj = json.loads(text[i:end + 1])
        except ValueError:
            i += 1
            continue
        if isinstance(obj, dict):
            best = obj
        i = end + 1
    return best


def scenario_names() -> set[str]:
    d = REPO / "scenarios"
    return {p.name for p in d.iterdir() if p.is_dir()} if d.is_dir() else set()


def plan_units(plan: dict | None) -> list[dict]:
    if not plan:
        return []
    if isinstance(plan.get("owners"), list):
        return [u for u in plan["owners"] if isinstance(u, dict)]
    return [plan]


def metrics(events: list[dict], cwd: Path, scenarios: set[str] | None = None,
            workdirs: dict[str, str] | None = None, view: list[tuple[str, Path]] | None = None) -> dict:
    steps, usage, messages = normalize(events, cwd, workdirs, view)
    first = next((i for i, s in enumerate(steps) if s["kind"] in ("write", "spawn", "message")), None)
    before = steps if first is None else steps[:first]
    paths: list[str] = []
    for s in before:
        paths += [p for p in s["paths"] if p and p not in paths]
    routes = []
    for s in steps:
        names = [n for n in ROUTE_CLIS if re.search(rf"(^|[\s;|&(/]){re.escape(n)}\s", s["command"] + " ")]
        head = (s.get("output_head") or "").strip().splitlines()
        if names and s["kind"] == "read" and (s["exit"] not in (0, None) or (head and ROUTE_FAIL.search(head[0]))):
            routes.append({"route": names[0], "command": s["command"][:200], "exit": s["exit"],
                           "first_line": (s.get("output_head") or "").strip().splitlines()[0][:200] if s.get("output_head") else ""})
    plan = None
    for msg in reversed(messages):
        plan = extract_json(msg)
        if plan:
            break
    units = plan_units(plan)
    scen = scenarios if scenarios is not None else scenario_names()
    named = sorted({s for u in units for c in (u.get("changes") or []) for s in scen
                    if re.search(rf"(?<![\w-]){re.escape(s)}(?![\w-])", str(c))})
    t_first = before[-1]["t"] if before else None
    last_turn = max((r.get("turn", 1) for r in events if isinstance(r, dict) and "e" in r), default=None)
    ts = [r["t"] for r in events if isinstance(r, dict) and r.get("turn") == last_turn and r.get("t") is not None]
    return {
        "wall_task_s": round(max(ts) - min(ts), 1) if ts else None,
        "steps": len(steps),
        "first_action": None if first is None else {"index": first, "kind": steps[first]["kind"], "t": steps[first]["t"],
                                                    "command": steps[first]["command"][:300]},
        "bytes_before_first_action": sum(s["bytes"] for s in before),
        "commands_before_first_action": len(before),
        "paths_before_first_action": paths,
        "files_before_first_action": [p for p in paths if p.startswith("/")],
        "t_last_read_before_action": t_first,
        "bytes_total": sum(s["bytes"] for s in steps),
        "tokens": usage,
        "route_failures": routes,
        "writes_attempted": [s["command"][:200] for s in steps if s["kind"] == "write"],
        "spawned": any(s["kind"] == "spawn" for s in steps),
        "plan_ok": plan is not None,
        "units": len(units) if plan is not None else None,
        "scenarios_named": named,
        "operator_questions": len((plan or {}).get("operator_questions") or []),
    }


ACTION_CALLS = {"spawn_agent", "send_message", "followup_task", "apply_patch", "update_goal", "create_goal", "write_stdin"}
JS_CMD = re.compile(r"""\bcmd\s*:\s*("(?:[^"\\\\]|\\\\.)*")""")
JS_STR = re.compile(r'"(?:[^"\\]|\\.)*"')


def js_string(literal: str) -> str | None:
    try:
        return json.loads(literal)
    except ValueError:
        return None


JS_EXEC = re.compile(r"exec_command\(\s*\{(.*?)\}\s*\)", re.S)
JS_FIELD = r"""\b%s\s*:\s*("(?:[^"\\]|\\.)*")"""


def command_workdirs(records: list[dict]) -> dict[str, str]:
    """Each shell command in a Codex rollout and the working folder the agent ran it in, from the
    exec_command calls (a code-mode `tools.exec_command({cmd, workdir})` or a function call)."""
    out: dict[str, str] = {}
    for o in records:
        p = o.get("payload") or {}
        if p.get("type") == "function_call" and p.get("name") == "exec_command":
            try:
                args = json.loads(p.get("arguments") or "{}")
            except ValueError:
                continue
            pairs = [(args.get("cmd"), args.get("workdir"))]
        elif p.get("type") == "custom_tool_call":
            pairs = []
            for m in JS_EXEC.finditer(p.get("input") or ""):
                cmd, wd = (re.search(JS_FIELD % k, m.group(1)) for k in ("cmd", "workdir"))
                pairs.append((cmd and js_string(cmd.group(1)), wd and js_string(wd.group(1))))
        else:
            continue
        out.update({c: w for c, w in pairs if isinstance(c, str) and isinstance(w, str) and w})
    return out


def rollout_calls(records: list[dict]) -> list[dict]:
    """Tool calls in a Codex rollout with the text the model got back: what actually entered context
    (Codex truncates long outputs; the exec stream reports the full command output)."""
    calls: dict[str, dict] = {}
    order: list[str] = []
    for o in records:
        p = o.get("payload") or {}
        typ = p.get("type")
        if typ in ("custom_tool_call", "function_call"):
            cid = p.get("call_id") or str(len(order))
            text = p.get("input") if typ == "custom_tool_call" else p.get("arguments")
            cmds = []
            for m in JS_CMD.finditer(text or ""):
                try:
                    cmds.append(json.loads(m.group(1)))
                except ValueError:
                    pass
            if not cmds and "exec_command" in (text or ""):  # commands built in arrays: every string literal
                cmds = [x for x in (js_string(m.group(0)) for m in JS_STR.finditer(text)) if x and " " in x and len(x) > 3]
            calls[cid] = {"name": p.get("name"), "input": text or "", "cmds": cmds, "bytes": 0}
            order.append(cid)
        elif typ in ("custom_tool_call_output", "function_call_output") and p.get("call_id") in calls:
            out = p.get("output")
            if isinstance(out, list):
                text = "".join(x.get("text", "") for x in out if isinstance(x, dict))
            elif isinstance(out, dict):
                text = json.dumps(out)
            else:
                text = str(out or "")
            calls[p["call_id"]]["bytes"] += len(text.encode())
            calls[p["call_id"]]["output_head"] = text[:200]
    return [calls[c] for c in order]


def seen_before_action(calls: list[dict], first_action_cmd: str | None) -> dict:
    """Bytes the model received before its first action: a collab or patch call, a shell write, or
    the call that ran the exec stream's first action."""
    probe = (first_action_cmd or "")[:80]
    for i, c in enumerate(calls):
        if c["name"] in ACTION_CALLS or "tools.apply_patch" in c["input"] or any(is_write(cmd) for cmd in c["cmds"]) \
                or (probe and any(probe in cmd for cmd in c["cmds"])):
            return {"bytes_seen_before_first_action": sum(x["bytes"] for x in calls[:i]), "calls_before_first_action": i,
                    "first_action_call": c["name"] if c["name"] != "exec" else "exec: " + (c["cmds"][0][:120] if c["cmds"] else "")}
    return {"bytes_seen_before_first_action": sum(x["bytes"] for x in calls), "calls_before_first_action": len(calls),
            "first_action_call": None}


def load_rollouts(run_dir: Path) -> list[dict]:
    out = []
    for f in sorted(run_dir.glob("rollout-*.jsonl.gz")):
        with gzip.open(f, "rt", encoding="utf-8") as fh:
            for line in fh:
                try:
                    out.append(json.loads(line))
                except ValueError:
                    pass
    out.sort(key=lambda o: o.get("timestamp", ""))
    return out


def load_events(run_dir: Path) -> list[dict]:
    out = []
    p = run_dir / "events.jsonl"
    for line in p.read_text(encoding="utf-8").splitlines() if p.exists() else []:
        try:
            out.append(json.loads(line))
        except ValueError:
            pass
    return out


def analyse_run(run_dir: Path) -> dict:
    meta = read_json(run_dir / "meta.json", {})
    events, records = load_events(run_dir), load_rollouts(run_dir)
    view = sandbox_view(WS(run_dir.parent.parent), meta["variant"], meta.get("cohort")) if meta.get("variant") else None
    m = metrics(events, Path(meta.get("cwd") or REPO), workdirs=command_workdirs(records), view=view)
    m["wall_s"] = meta.get("wall_s")
    calls = rollout_calls(records)
    m.update(seen_before_action(calls, (m["first_action"] or {}).get("command") if m["first_action"] and
                                m["first_action"]["kind"] == "write" else None))
    m["collab_calls"] = [c["name"] for c in calls if c["name"] != "exec"]
    m["tool_calls"] = len(calls) if records else m["commands_before_first_action"]  # no rollout: the event stream
    _, _, messages = normalize(events, REPO)
    plan = None
    for msg in reversed(messages):
        plan = extract_json(msg)
        if plan:
            break
    write_json(run_dir / "metrics.json", m)
    write_json(run_dir / "plan.json", {"plan": plan, "final_message": messages[-1] if messages else None})
    return m


# ---------------------------------------------------------------- grade


def sanitize(text: str, research: Path) -> str:
    """Both variants name the same real paths; the grader sees neither paths nor commission ids."""
    text = text.replace(str(SKILLS), "<skills>").replace(str(research), "<research-home>")
    text = re.sub(r"\b(replay-)?[a-z0-9-]+-20\d{6}(-20\d{6})?(-\d+)?\b", "<commission>", text)
    return text


NO_PLAN = {"plan_failed": "the agent's reply held no parseable plan"}


def topic_units(run_dir: Path, topic: str) -> list[dict]:
    plan = (read_json(run_dir / "plan.json") or {}).get("plan")
    if plan is None:
        return [NO_PLAN]
    units = plan_units(plan)
    if (read_json(run_dir / "meta.json", {})).get("role") == "owner":
        return units
    return [u for u in units if topic in str(u.get("topic", ""))]


def grade_prompt(mission: str, rubric: str, a: list[dict], b: list[dict]) -> str:
    return (f"{rubric.strip()}\n\nMISSION.md of the topic, verbatim:\n\n<mission>\n{mission}\n</mission>\n\n"
            f"Plan A (the units it proposes for this topic):\n\n<plan-a>\n{json.dumps(a, indent=1, ensure_ascii=False)}\n</plan-a>\n\n"
            f"Plan B:\n\n<plan-b>\n{json.dumps(b, indent=1, ensure_ascii=False)}\n</plan-b>\n")


def grade_once(ws: WS, gdir: Path, label: str, prompt: str, model: str, effort: str, timeout: int) -> dict:
    write(gdir / f"prompt-{label}.txt", prompt)
    with Gate() as gate:
        inner = codex_turn(prompt, model=model, effort=effort, cwd=HOME / ".codex", resume=None, dev=None)
        t0 = time.time()
        with open(gdir / f"events-{label}.jsonl", "w", encoding="utf-8") as out:
            res = stream(sandbox_cmd(DRIVE, ws=ws, gate=gate, cwd=HOME / ".codex"), codex_header([inner]), sandbox_env(),
                         t0 + timeout, out, t0)
        net = gate.summary()
    events = [json.loads(x) for x in (gdir / f"events-{label}.jsonl").read_text(encoding="utf-8").splitlines() if x.strip()]
    if (seen := websocket_attempts(net, res["stderr"], events)):
        raise Fail(f"grade {label} tried a WebSocket ({seen[0]}); grades run over HTTPS only")
    _, usage, messages = normalize(events, HOME)
    verdict = next((v for v in (extract_json(m) for m in reversed(messages)) if v), None)
    return {"verdict": verdict, "exit": res["exit"], "killed": (res["turns"] or [{}])[0].get("killed"), "tokens": usage,
            "stderr_tail": res["stderr"][-1500:]}


def cmd_grade(ws: WS, run_a: Path, run_b: Path, topic: str, model: str, effort: str, timeout: int) -> dict:
    """Blind rubric grade of two plans for one topic against the snapshot's MISSION.md (the topic text
    both variants read), asked twice with the sides swapped. A run's score is its mean over both
    orders; the better run is the one both orders prefer, otherwise the pair is a tie."""
    research = (need_snapshot(ws), research_home(ws))[1]
    units = {r.name: json.loads(sanitize(json.dumps(topic_units(r, topic), ensure_ascii=False), research))
             for r in (run_a, run_b)}
    mission_path = ws.snapshot() / "home/topics" / topic / "MISSION.md"
    mission, rubric = mission_path.read_text(encoding="utf-8"), RUBRIC.read_text(encoding="utf-8")
    gdir = ws.grades() / f"{topic}.{run_a.name}.vs.{run_b.name}"
    if (gdir / "grade.json").exists():
        raise Fail(f"{gdir.name} is already graded")
    gdir.mkdir(parents=True, exist_ok=True)
    orders = [{"A": run_a.name, "B": run_b.name}, {"A": run_b.name, "B": run_a.name}]
    write_json(gdir / "mapping.json", {"orders": orders, "topic": topic, "runs": [run_a.name, run_b.name],
                                       "model": model, "effort": effort, "mission": str(mission_path), "graded_at": now()})
    results = run_jobs([0, 1], lambda i: grade_once(ws, gdir, "AB"[i] + "first", grade_prompt(
        mission, rubric, units[orders[i]["A"]], units[orders[i]["B"]]), model, effort, timeout), 2)
    out = []
    for o, r in zip(orders, results):
        res = r.get("result") or {"verdict": None, "error": r.get("error")}
        v = res.get("verdict") or {}
        better = v.get("better")
        out.append({**o, **res, "better_run": o.get(better) if better in ("A", "B") else ("tie" if better == "tie" else None),
                    "scores": {o["A"]: (v.get("A") or {}).get("score"), o["B"]: (v.get("B") or {}).get("score")}})
    picks = [x["better_run"] for x in out]
    scores = {r: [x["scores"][r] for x in out if isinstance(x["scores"].get(r), (int, float))] for r in (run_a.name, run_b.name)}
    result = {"orders": out, "better_run": picks[0] if picks[0] == picks[1] and picks[0] not in (None, "tie") else
              ("tie" if None not in picks else None),
              "agree": picks[0] == picks[1], "score": {r: (sum(s) / len(s) if s else None) for r, s in scores.items()}}
    write_json(gdir / "grade.json", result)
    return {"grade": str(gdir), "better_run": result["better_run"], "agree": result["agree"], "score": result["score"]}


def grade_pairs(ws: WS, arms: dict[str, list[str]], noise: int) -> list[dict]:
    """Pairs to grade between the first two arms: run i of one arm against run i of the other for each
    prompt and topic, then `noise` same-arm pairs (run 1 against run 2) spread over the prompts."""
    names = list(arms)
    if len(names) < 2:
        raise Fail("grade-all needs two arms (--arm NAME=VARIANT[,VARIANT...] twice)")
    by_prompt: dict[tuple, dict[str, list[str]]] = {}
    for d in sorted(ws.runs().iterdir()) if ws.runs().is_dir() else []:
        m = read_json(d / "meta.json")
        if not m or m.get("excluded") or not (d / "plan.json").exists():
            continue
        arm = next((a for a, vs in arms.items() if m["variant"] in vs), None)
        if arm:
            key = (m["commission_id"], m["role"], m.get("topic") or "")
            by_prompt.setdefault(key, {}).setdefault(arm, []).append(d.name)
    num = lambda n: int(n.rsplit(".", 1)[1])
    pairs, same = [], []
    for (cid, role, topic), by in sorted(by_prompt.items()):
        topics = [topic] if role == "owner" else prompt_info(ws, cid, "old").get("topics") or []
        a, b = sorted(by.get(names[0], []), key=num), sorted(by.get(names[1], []), key=num)
        for t in topics:
            pairs += [{"a": x, "b": y, "topic": t} for x, y in zip(a, b)]
            for side in (a, b):
                if len(side) >= 2:
                    same.append({"a": side[0], "b": side[1], "topic": t})
    step = max(1, len(same) // max(1, noise))
    return pairs + same[::step][:noise]


# ---------------------------------------------------------------- report


MEASURES = {"bytes_seen": lambda m: m.get("bytes_seen_before_first_action", m["bytes_before_first_action"]),
            "bytes_raw": lambda m: m["bytes_before_first_action"],
            "tokens_in": lambda m: m["tokens"]["input"],
            "files": lambda m: len(m["files_before_first_action"]),
            "units": lambda m: m["units"] if m.get("plan_ok") else None}  # a failed plan is not 0 units


def mission_check_status(ws: WS, variant: str, topic: str) -> str:
    script, home = ws.variant(variant) / "skills/research-coordination/scripts/mission_check.py", ws.snapshot() / "home"
    if not script.is_file() or not (home / "topics" / topic).is_dir():
        return "n/a"
    proc = subprocess.run([sys.executable, "-I", str(script), str(home), "--topic", topic, "--json"],
                          capture_output=True, text=True, timeout=60)
    try:
        r = json.loads(proc.stdout)[0]
    except (ValueError, IndexError):
        return f"failed: {(proc.stderr or proc.stdout).strip()[:80]}"
    return "errors " + str(len(r["errors"])) if r["errors"] else (f"warnings {len(r['warnings'])}" if r["warnings"] else "ok")


def grade_records(ws: WS) -> list[dict]:
    """Every grade as {topic, runs, score per run, better_run, agree, levels per run, order scores}.
    Grades made before swapped orders existed (one order) read the same way with agree None."""
    out = []
    for g in sorted(ws.grades().glob("*/grade.json")) if ws.grades().is_dir() else []:
        mp, res = read_json(g.parent / "mapping.json", {}), read_json(g, {})
        if "orders" in res:
            orders = res["orders"]
        else:
            v = res.get("verdict") or {}
            b = v.get("better")
            orders = [{"A": mp.get("A"), "B": mp.get("B"), "verdict": v,
                       "better_run": mp.get(b) if b in ("A", "B") else ("tie" if b == "tie" else None),
                       "scores": {mp.get("A"): (v.get("A") or {}).get("score"), mp.get("B"): (v.get("B") or {}).get("score")}}]
        runs = [r for r in (orders[0].get("A"), orders[0].get("B")) if r]
        if len(runs) != 2:
            continue
        levels = {r: [] for r in runs}
        order_scores = {r: [] for r in runs}
        for o in orders:
            v = o.get("verdict") or {}
            for label in ("A", "B"):
                r = o.get(label)
                levels[r] += [u.get("scope_level", "?") for u in (v.get(label) or {}).get("units", []) if isinstance(u, dict)]
                s = (o.get("scores") or {}).get(r)
                if isinstance(s, (int, float)):
                    order_scores[r].append(s)
        picks = [o.get("better_run") for o in orders]
        better = res.get("better_run") if "orders" in res else picks[0]
        out.append({"grade": g.parent.name, "topic": mp.get("topic"), "runs": runs, "better_run": better,
                    "agree": (picks[0] == picks[1]) if len(picks) == 2 else None, "levels": levels,
                    "order_scores": order_scores,
                    "score": {r: (sum(s) / len(s) if s else None) for r, s in order_scores.items()}})
    return out


def unit_topic(u: dict, topics: list[str]) -> str | None:
    t = str(u.get("topic", ""))
    return next((x for x in topics if x == t), None) or next((x for x in topics if x in t), None)


def median(xs: list[float]) -> float | None:
    xs = sorted(xs)
    return None if not xs else (xs[len(xs) // 2] if len(xs) % 2 else (xs[len(xs) // 2 - 1] + xs[len(xs) // 2]) / 2)


def sign_test(wins: int, losses: int) -> float | None:
    """Two-sided exact binomial p-value for wins against losses (ties left out)."""
    n = wins + losses
    if not n:
        return None
    k = min(wins, losses)
    return min(1.0, 2 * sum(math.comb(n, i) for i in range(k + 1)) / 2 ** n)


def log_ratio_ci(prompts: list[tuple[list, list]], n: int = 2000, seed: int = 0) -> dict | None:
    """Pooled ratio B/A: the mean over prompts of (mean log(B+1) − mean log(A+1)), with a 95% interval
    from a bootstrap that resamples runs within each prompt and arm. Prompts need two runs per arm."""
    ps = [([math.log(x + 1) for x in a], [math.log(x + 1) for x in b]) for a, b in prompts
          if len([x for x in a if x is not None]) >= 2 and len([x for x in b if x is not None]) >= 2
          and None not in a and None not in b]
    if not ps:
        return None
    est = lambda sample: sum(sum(b) / len(b) - sum(a) / len(a) for a, b in sample) / len(sample)
    rng = random.Random(seed)
    boots = sorted(est([([rng.choice(a) for _ in a], [rng.choice(b) for _ in b]) for a, b in ps]) for _ in range(n))
    return {"ratio": math.exp(est(ps)), "lo": math.exp(boots[int(0.025 * n)]), "hi": math.exp(boots[int(0.975 * n) - 1]),
            "prompts": len(ps), "runs_a": sum(len(a) for a, _ in ps), "runs_b": sum(len(b) for _, b in ps)}


def parse_arms(specs: list[str] | None, ws: WS, variants: list[str]) -> dict[str, list[str]]:
    """NAME=VARIANT[,VARIANT...]; the first arm is the baseline. Default: the pre-registered arms, else
    `old` against each other variant on its own."""
    if specs:
        arms = {}
        for s in specs:
            name, _, vs = s.partition("=")
            if not name or not vs:
                raise Fail(f"--arm {s!r}: use NAME=VARIANT[,VARIANT...]")
            arms[name] = vs.split(",")
        return arms
    pre = read_json(ws.root / "PREREGISTRATION.json")
    if pre and pre.get("arms"):
        return pre["arms"]
    return {"old": ["old"], **{v: [v] for v in variants if v != "old"}}


def arm_values(runs: list[dict], arms: dict) -> dict:
    """{(cid, role, topic): {arm: {measure: [values]}}}"""
    out: dict = {}
    for r in runs:
        arm = next((a for a, vs in arms.items() if r["meta"]["variant"] in vs), None)
        if arm is None:
            continue
        key = (r["meta"]["commission_id"], r["meta"]["role"], r["meta"].get("topic") or "")
        slot = out.setdefault(key, {}).setdefault(arm, {k: [] for k in MEASURES})
        for k, f in MEASURES.items():
            slot[k].append(f(r["m"]))
    return out


def fmt_ci(c: dict | None) -> str:
    return "—" if not c else f"×{c['ratio']:.2f} [{c['lo']:.2f}, {c['hi']:.2f}]"


def summary_lines(vals: dict, arms: dict, records: list[dict], run_arm: dict, run_role: dict) -> list[str]:
    names = list(arms)
    base = names[0]
    mean = lambda xs: sum(xs) / len(xs)
    out = ["## Summary", "",
           f"Arms: " + "; ".join(f"`{a}` = {', '.join(f'`{v}`' for v in vs)}" for a, vs in arms.items())
           + f". Ratios are arm / `{base}`: the mean over prompts of the per-prompt mean log(value+1) difference, with a 95% "
             "bootstrap interval (runs resampled within each prompt and arm; prompts with two or more runs per arm). Units "
             "leave out runs whose plan failed to parse; failed plans are counted on their own.", "",
           "| Role | Arm | Prompts | Runs base / arm | Plan failed base / arm | Bytes seen | Bytes raw | Tokens in | Files "
           "| Units arm − base (median of prompt means) | Median bytes-seen spread between repeats, base / arm |",
           "|---|---|---|---|---|---|---|---|---|---|---|"]
    verdicts = {}
    for role in ("supervisor", "owner"):
        for arm in names[1:]:
            ps = [by for (cid, r, t), by in sorted(vals.items()) if r == role and by.get(base) and by.get(arm)]
            if not ps:
                continue
            ci = {k: log_ratio_ci([(b[base][k], b[arm][k]) for b in ps]) for k in ("bytes_seen", "bytes_raw", "tokens_in", "files")}
            du = median([mean([u for u in b[arm]["units"] if u is not None]) - mean([u for u in b[base]["units"] if u is not None])
                         for b in ps if any(u is not None for u in b[arm]["units"]) and any(u is not None for u in b[base]["units"])])
            failed = lambda side: sum(1 for b in ps for u in b[side]["units"] if u is None)
            spread_pct = lambda side: median([100 * (max(b[side]["bytes_seen"]) - min(b[side]["bytes_seen"])) / max(b[side]["bytes_seen"])
                                          for b in ps if len(b[side]["bytes_seen"]) >= 2 and max(b[side]["bytes_seen"])])
            sb, sa = spread_pct(base), spread_pct(arm)
            out.append(f"| {role} | `{arm}` | {len(ps)} | {sum(len(b[base]['bytes_seen']) for b in ps)} / "
                       f"{sum(len(b[arm]['bytes_seen']) for b in ps)} | {failed(base)} / {failed(arm)} | {fmt_ci(ci['bytes_seen'])} "
                       f"| {fmt_ci(ci['bytes_raw'])} | {fmt_ci(ci['tokens_in'])} | {fmt_ci(ci['files'])} "
                       f"| {'—' if du is None else f'{du:+.2f}'} | {'—' if sb is None else f'{sb:.0f}%'} / {'—' if sa is None else f'{sa:.0f}%'} |")
            verdicts[(role, arm)] = ci
    # the blind grader
    tally: dict = {}
    diffs: dict = {}
    same_pref = same_n = 0
    same_abs, order_agree, order_n, order_abs = [], 0, 0, []
    for g in records:
        a, b = g["runs"]
        if a not in run_arm or b not in run_arm:
            continue
        if g["agree"] is not None:
            order_n += 1
            order_agree += bool(g["agree"])
            order_abs += [abs(s[0] - s[1]) for s in g["order_scores"].values() if len(s) == 2]
        if run_arm[a] == run_arm[b]:
            same_n += 1
            same_pref += g["better_run"] in (a, b)
            if None not in g["score"].values():
                same_abs.append(abs(g["score"][a] - g["score"][b]))
            continue
        if base not in (run_arm[a], run_arm[b]):
            continue
        new = a if run_arm[a] != base else b
        old = b if new == a else a
        key = (run_role[new], run_arm[new])
        t = tally.setdefault(key, {"better": 0, "worse": 0, "tie": 0})
        t["better" if g["better_run"] == new else "worse" if g["better_run"] == old else "tie"] += 1
        if g["score"][new] is not None and g["score"][old] is not None:
            diffs.setdefault(key, []).append(g["score"][new] - g["score"][old])
    stats = lambda xs: "—" if not xs else f"median {median(xs):+.2f}, mean {sum(xs) / len(xs):+.2f} over {len(xs)}"
    fmtp = lambda p: "—" if p is None else f"{p:.2f}"
    out += ["", "| Role | Arm | Grades against base | Arm better | Arm worse | Tie or orders disagree | Sign test p | Rubric score arm − base |",
            "|---|---|---|---|---|---|---|---|"]
    for key in sorted(tally):
        t = tally[key]
        out.append(f"| {key[0]} | `{key[1]}` | {sum(t.values())} | {t['better']} | {t['worse']} | {t['tie']} "
                   f"| {fmtp(sign_test(t['better'], t['worse']))} | {stats(diffs.get(key, []))} |")
    out += ["", f"Noise. Same arm graded against itself: a preference both orders agree on in {same_pref} of {same_n} grades; "
                f"|score difference| {stats(same_abs)}. Each pair is graded twice with the sides swapped: the orders agree "
                f"on the better plan in {order_agree} of {order_n} grades; |score change between orders| {stats(order_abs)}.",
            "", "Verdict per role and arm:", ""]
    for (role, arm), ci in sorted(verdicts.items()):
        words = []
        for k, label in (("bytes_seen", "bytes seen"), ("tokens_in", "tokens")):
            c = ci[k]
            words.append(f"{label} " + ("not measurable (needs two runs per arm per prompt)" if not c else
                                        ("lower " if c["hi"] < 1 else "higher " if c["lo"] > 1 else "no detectable change ")
                                        + fmt_ci(c)))
        t = tally.get((role, arm))
        if t:
            p = sign_test(t["better"], t["worse"])
            words.append(f"grader prefers the arm {t['better']}–{t['worse']} (p {fmtp(p)})" if p is not None and p < 0.05 and t["better"] > t["worse"]
                         else f"grader prefers the base {t['worse']}–{t['better']} (p {fmtp(p)})" if p is not None and p < 0.05
                         else f"grader shows no detectable preference ({t['better']} better, {t['worse']} worse, {t['tie']} tie, p {fmtp(p)})")
        out.append(f"- {role} · `{arm}`: " + "; ".join(words) + ".")
    out.append("")
    return out


def engaged(m: dict) -> bool:
    """A run engaged unless it both reasoned no tokens and made no tool call: it answered from the prompt alone."""
    return bool(m["tokens"]["reasoning"] or m.get("tool_calls", m.get("calls_before_first_action")))


def engagement_lines(runs: list[dict], run_arm: dict, arms: dict) -> list[str]:
    """Post-hoc, exploratory: each arm's share of runs that did not engage, and bytes seen among those that did."""
    out = ["## Did not engage (post-hoc exploratory analysis, NOT pre-registered)", "",
           "Added after the runs. A run \"did not engage\" when it reasoned 0 tokens AND made 0 tool calls. The "
           "pre-registered verdicts use every run and are unchanged; nothing here is a claim.", "",
           "| Role · arm | Runs | Did not engage | Bytes seen median, engaged runs (n) |", "|---|---|---|---|"]
    for role in ("supervisor", "owner"):
        for arm in arms:
            rs = [x["m"] for x in runs if x["meta"]["role"] == role and run_arm.get(x["dir"].name) == arm]
            if not rs:
                continue
            on = [MEASURES["bytes_seen"](m) for m in rs if engaged(m)]
            med = median([x for x in on if x is not None])
            out.append(f"| {role} · `{arm}` | {len(rs)} | {len(rs) - len(on)} ({(len(rs) - len(on)) / len(rs):.0%}) | "
                       f"{'—' if med is None else f'{med:,.0f}'} ({len(on)}) |")
    return out + [""]


def prereg_lines(ws: WS, runs: list[dict], vals: dict) -> list[str]:
    pre = read_json(ws.root / "PREREGISTRATION.json")
    if not pre:
        return []
    arms = pre["arms"]
    names = list(arms)
    out = ["## Pre-registration", "", f"Written {pre['written_at']}, before any run of these arms "
           + ("(none existed)." if not any(pre.get("runs_existing", {}).values()) else
              f"(runs already on disk then: {pre['runs_existing']})."), "", f"Expected effect: {pre.get('expected') or '—'}", ""]
    for c in pre["claims"]:
        role, metric = c["role"], c["metric"]
        if c["kind"] == "median_below":
            xs = [MEASURES[metric](r["m"]) for r in runs if r["meta"]["role"] == role and r["meta"]["variant"] in arms[c["arm"]]]
            xs = [x for x in xs if x is not None]
            med = median(xs)
            res = "NO DATA" if med is None else ("PASS" if med < c["value"] else "FAIL")
            out.append(f"- {role} {metric} median in `{c['arm']}` below {c['value']:,}: {res} "
                       f"(median {'—' if med is None else f'{med:,.0f}'} over {len(xs)} runs)")
        else:
            ci = log_ratio_ci([(by[names[0]][metric], by[names[1]][metric]) for (cid, r, t), by in vals.items()
                               if r == role and by.get(names[0]) and by.get(names[1])])
            res = "NO DATA" if not ci else ("PASS" if ci["hi"] < c["value"] else "FAIL" if ci["lo"] >= c["value"] else "INCONCLUSIVE")
            out.append(f"- {role} {metric} `{names[1]}` / `{names[0]}` no worse than ×{c['value']:.2f}: {res} ({fmt_ci(ci)})")
    return out + [""]


def cmd_preregister(ws: WS, arms: list[str], expected: str, claims: list[str]) -> dict:
    """Write PREREGISTRATION.json before the runs: the arms, the expected effect and the claims the
    report will check (median_below ROLE:METRIC:ARM:VALUE, ratio_below ROLE:METRIC:VALUE)."""
    path = ws.root / "PREREGISTRATION.json"
    if path.exists():
        raise Fail(f"{path} exists; a pre-registration is written once")
    parsed_arms = parse_arms(arms, ws, [])
    if len(parsed_arms) != 2:
        raise Fail("pre-register exactly two arms: --arm BASE=... --arm NEW=...")
    parsed = []
    for c in claims:
        kind, _, rest = c.partition(":")
        parts = rest.split(":")
        if kind == "median_below" and len(parts) == 4 and parts[2] in parsed_arms:
            parsed.append({"kind": kind, "role": parts[0], "metric": parts[1], "arm": parts[2], "value": float(parts[3])})
        elif kind == "ratio_below" and len(parts) == 3:
            parsed.append({"kind": kind, "role": parts[0], "metric": parts[1], "value": float(parts[2])})
        else:
            raise Fail(f"claim {c!r}: use median_below:ROLE:METRIC:ARM:VALUE or ratio_below:ROLE:METRIC:VALUE")
        if parsed[-1]["metric"] not in MEASURES or parsed[-1]["role"] not in ("supervisor", "owner"):
            raise Fail(f"claim {c!r}: metric one of {sorted(MEASURES)}, role supervisor or owner")
    existing = {}
    for d in ws.runs().iterdir() if ws.runs().is_dir() else []:
        v = (read_json(d / "meta.json") or {}).get("variant")
        if any(v in vs for vs in parsed_arms.values()):
            existing[v] = existing.get(v, 0) + 1
    doc = {"written_at": now(), "arms": parsed_arms, "expected": expected, "claims": parsed, "runs_existing": existing}
    write_json(path, doc)
    return doc


def guidance_key(path: str, ws: WS) -> str:
    """One name per file, whichever variant's copy was read: research skills relative to the skills
    root (a prompt-manager skill read counts as its SKILL.md), research-home files by their role."""
    if path.startswith("skill:"):
        return f"skills/{path[6:]}/SKILL.md"
    if not path.startswith(("/", "cli:")):  # relative to the run's cwd, the repo
        path = f"{REPO}/{path}"
    if path.startswith(str(SKILLS) + "/"):
        return "skills/" + path[len(str(SKILLS)) + 1:]
    research = research_home(ws)
    m = re.match(rf"{re.escape(str(research))}/(cohorts|topics)/[^/]+/(.*)", path)
    if m:
        return f"<{m.group(1)[:-1]}>/" + m.group(2)
    if path.startswith(f"{research}/"):
        return "<home>/" + path[len(str(research)) + 1:]
    return path.replace(str(REPO) + "/", "")


def guidance_lines(runs: list, ws: WS, run_arm: dict) -> list[str]:
    """Which files each role and arm opened before its first action: k of n runs."""
    cols = sorted({(r["meta"]["role"], run_arm[r["dir"].name]) for r in runs if r["dir"].name in run_arm},
                  key=lambda c: (c[0] != "supervisor", c[1]))
    counts: dict[str, dict] = {}
    n = {c: 0 for c in cols}
    for r in runs:
        if r["dir"].name not in run_arm:
            continue
        c = (r["meta"]["role"], run_arm[r["dir"].name])
        n[c] += 1
        for key in {guidance_key(p, ws) for p in r["m"]["paths_before_first_action"]}:
            counts.setdefault(key, {}).setdefault(c, 0)
            counts[key][c] += 1
    out = ["## Files opened before the first action", "",
           "Runs that opened each file (k of n) before their first write, sub-agent or message. `<cohort>` and `<topic>` "
           "are the run's own commission and topic folders; a `prompt-manager skill read` counts as that skill's SKILL.md.", "",
           "| File | " + " | ".join(f"{role} `{a}`" for role, a in cols) + " |", "|---|" + "---|" * len(cols)]
    for key in sorted(counts, key=lambda k: (-sum(counts[k].values()), k)):
        out.append(f"| `{key}` | " + " | ".join(f"{counts[key].get(c, 0)}/{n[c]}" for c in cols) + " |")
    return out + [""]


def cmd_report(ws: WS, arm_specs: list[str] | None, correction: str | None = None) -> Path:
    """REPORT.md over every run not excluded. A correction (a parser or harness fix after the runs that
    leaves the pre-registered measure unchanged) is kept in corrections.json and shown in every report."""
    corrections = read_json(ws.root / "corrections.json") or []
    if correction:
        corrections.append({"at": now(), "text": correction})
        write_json(ws.root / "corrections.json", corrections)
    runs, excluded = [], []
    for d in sorted(ws.runs().iterdir()) if ws.runs().is_dir() else []:
        meta = read_json(d / "meta.json")
        if not meta:
            continue
        if meta.get("excluded"):
            excluded.append((d.name, meta["excluded"]))
            continue
        m = analyse_run(d)  # always re-parse: the parser may have improved since the run
        plan = (read_json(d / "plan.json") or {}).get("plan")
        runs.append({"dir": d, "meta": meta, "m": m, "plan": plan,
                     "topics": [meta["topic"]] if meta["role"] == "owner" else
                     prompt_info(ws, meta["commission_id"], "old").get("topics") or []})
    variants = sorted({r["meta"]["variant"] for r in runs})
    arms = parse_arms(arm_specs, ws, variants)
    run_arm = {r["dir"].name: a for r in runs for a, vs in arms.items() if r["meta"]["variant"] in vs}
    run_role = {r["dir"].name: r["meta"]["role"] for r in runs}
    records = grade_records(ws)
    vals = arm_values(runs, arms)
    lines = ["# Commission replay report", "",
             f"Generated {now()} by `python3 -I {Path(__file__)} --workspace {ws.root} report"
             + "".join(f" --arm {a}={','.join(vs)}" for a, vs in arms.items()) + "`. Each row is one cold session in "
             "the sandbox described by `replay.py sandbox-check`. \"Bytes seen\" is the tool output the model received "
             "(Codex rollout, after Codex's own truncation) up to its first write, sub-agent or message, or over the whole "
             "run when it stopped as the replay protocol asks; \"bytes raw\" is the full command output. Re-parse a run "
             "with `replay.py metrics <run dir>`.", ""]
    if corrections:
        lines += ["## Corrections after the runs", "", "Not new analyses: each changes no pre-registered measure.", ""]
        lines += [f"- {c['at']}: {c['text']}" for c in corrections] + [""]
    lines += prereg_lines(ws, runs, vals)
    lines += ["## Variants", "", "| Variant | Arm | Research skill text bytes | Prompts frozen or rendered | Packet gates failed |",
              "|---|---|---|---|---|"]
    for v in variants:
        infos = [prompt_info(ws, p.parent.name, v) for p in sorted(ws.root.glob("commissions/*")) if ws.prompts(p.name, v).is_dir()]
        when = sorted({(i.get("frozen_at") or i.get("rendered_at") or "?") for i in infos})
        failed = sorted({f"{i['commission_id']}: {g['name']} {g['value']}" for i in infos for g in i.get("gates") or [] if not g["ok"]})
        size = skill_sizes(ws.variant(v) / "skills")["total"] if (ws.variant(v) / "skills").is_dir() else "?"
        arm = next((a for a, vs in arms.items() if v in vs), "—")
        lines.append(f"| `{v}` | `{arm}` | {size if isinstance(size, str) else f'{size:,}'} | {when[0] if when else '?'}"
                     f"{' … ' + when[-1] if len(when) > 1 else ''} | {'; '.join(failed) or '—'} |")
    lines.append("")
    lines += summary_lines(vals, arms, records, run_arm, run_role)
    lines += engagement_lines(runs, run_arm, arms)
    lines += guidance_lines(runs, ws, run_arm)
    by_run: dict[str, list] = {}
    for g in records:
        for r in g["runs"]:
            other = [x for x in g["runs"] if x != r][0]
            verdict = ("better" if g["better_run"] == r else "worse" if g["better_run"] == other else
                       "tie" if g["better_run"] == "tie" else "orders disagree" if g["agree"] is False else "no verdict")
            by_run.setdefault(r, []).append({"topic": g["topic"], "levels": g["levels"][r], "verdict": verdict, "vs": other,
                                             "score": g["score"][r]})
    mc_cache: dict = {}
    scen = scenario_names()
    for topic in sorted({t for r in runs for t in r["topics"] if t}):
        rows = [r for r in runs if topic in r["topics"]]
        lines += [f"## {topic}", "",
                  "| Commission | Role | Variant | Run | Model/effort | Bytes seen (raw) | Files read | Tokens in (cached) / out | Task wall s | First action | Units for topic | Scenarios named | Grader scope | Grader verdict | mission_check |",
                  "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
        for r in sorted(rows, key=lambda r: (r["meta"]["commission_id"], r["meta"]["role"], r["meta"]["variant"], r["dir"].name)):
            meta, m, v = r["meta"], r["m"], r["meta"]["variant"]
            units = plan_units(r["plan"]) if meta["role"] == "owner" else \
                [u for u in plan_units(r["plan"]) if unit_topic(u, r["topics"]) == topic]
            named = sorted({s for u in units for c in (u.get("changes") or []) for s in scen
                            if re.search(rf"(?<![\w-]){re.escape(s)}(?![\w-])", str(c))})
            if (v, topic) not in mc_cache:
                mc_cache[(v, topic)] = mission_check_status(ws, v, topic)
            g = [x for x in by_run.get(r["dir"].name, []) if x["topic"] == topic]
            scope = "; ".join(", ".join(x["levels"]) or "—" for x in g) or "—"
            verdict = "; ".join(f"{x['verdict']} vs `{x['vs']}` ({x['score']})" for x in g) or "—"
            tok, fa = m["tokens"], m["first_action"]
            fa_txt = "stopped as asked" if not fa else f"{fa['kind']}: `{fa['command'][:50].replace('|', '/')}`"
            lines.append(f"| {meta['commission_id']} | {meta['role']} | `{v}` | `{r['dir'].name}` | {meta['model']}/{meta['effort']} "
                         f"| {MEASURES['bytes_seen'](m):,} ({m['bytes_before_first_action']:,}) | {len(m['files_before_first_action'])} "
                         f"| {tok['input']:,} ({tok['cached_input']:,}) / {tok['output']:,} | {m.get('wall_task_s') or meta.get('wall_s')} | {fa_txt} "
                         f"| {len(units) if m['plan_ok'] else 'plan failed'} | {len(named)} | {scope} | {verdict} | {mc_cache[(v, topic)]} |")
        lines.append("")
    lines += ["## Paths read before the first action, network and route failures", ""]
    for r in runs:
        m = r["m"]
        lines.append(f"- `{r['dir'].name}` ({MEASURES['bytes_seen'](m):,} bytes seen): " + ", ".join(
            f"`{guidance_key(p, ws)}`" for p in m["paths_before_first_action"]))
        denied = (r["meta"].get("network") or {}).get("denied")
        if denied:
            lines.append(f"  - network requests refused: {denied[:5]}")
        for rf in m["route_failures"]:
            lines.append(f"  - route failure · {rf['route']} · `{rf['command'][:100]}` · {rf['first_line']}")
        if r["meta"].get("turns") and any(t.get("killed") not in (None, "answered") for t in r["meta"]["turns"]):
            lines.append(f"  - stopped by the harness: {[t.get('killed') for t in r['meta']['turns']]}")
    if excluded:
        lines += ["", "## Excluded runs", ""] + [f"- `{n}`: {why}" for n, why in excluded]
    path = ws.root / "REPORT.md"
    write(path, "\n".join(lines) + "\n")
    return path


# ---------------------------------------------------------------- sandbox check


PROBE = r'''
import json, os, socket, subprocess, sys
cfg, out = json.loads(sys.argv[1]), {}
def try_write(path):
    try:
        with open(path, "w") as f:
            f.write("x")
        os.unlink(path)
        return "WRITTEN"
    except OSError as e:
        return "refused: " + (e.strerror or str(e))
out["writes"] = {name: try_write(p) for name, p in cfg["writes"].items()}
reach = []
for p in cfg["ports"]:
    s = socket.socket(); s.settimeout(0.5)
    try:
        s.connect(("127.0.0.1", p)); reach.append(p)
    except OSError:
        pass
    finally:
        s.close()
out["host_ports_reachable"] = reach
out["host_sockets_visible"] = [p for p in cfg["sockets"] if os.path.exists(p)]
try:
    socket.create_connection(("1.1.1.1", 443), timeout=3).close(); out["direct_internet"] = "CONNECTED"
except OSError as e:
    out["direct_internet"] = "refused: " + str(e)
import re, ssl
def gate(raw, sni=None):
    """Send raw bytes to the gate, or with sni through a CONNECT tunnel and a TLS session that names sni."""
    s = socket.create_connection(("127.0.0.1", cfg["proxy_port"]), timeout=30)
    ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
    ctx.check_hostname, ctx.verify_mode = False, ssl.CERT_NONE  # the gate, not this client, must refuse
    try:
        if sni == "direct-tls":
            s = ctx.wrap_socket(s, server_hostname="example.com")
        elif sni:
            s.sendall(("CONNECT %s:443 HTTP/1.1\r\n\r\n" % cfg["provider"]).encode())
            first = s.recv(4096).decode("latin1")
            if " 200 " not in first.split("\r\n")[0] + " ":
                return {"status": first.split("\r\n")[0], "gate_refused": True, "responses": 0}
            s = ctx.wrap_socket(s, server_hostname=sni)
        s.sendall(raw.encode("latin1"))
        got = b""
        while (d := s.recv(65536)) and len(got) < 1 << 20:
            got += d
    except (OSError, ssl.SSLError) as e:
        return "error: " + type(e).__name__
    finally:
        s.close()
    text = got.decode("latin1")
    return {"status": text.split("\r\n")[0], "gate_refused": "X-Replay-Gate: refused" in text,
            "responses": len(re.findall(r"(?m)^HTTP/1\.[01] \d{3}", text))}
out["gate"] = {name: gate(raw, sni) for name, (raw, sni) in cfg["gate_requests"].items()}
name_re, text_re = re.compile(cfg["secret_name"], re.I), re.compile(cfg["secret_text"].encode("latin1"))
found = []
for top in cfg["scan"]:
    for d, dirs, files in os.walk(top):
        for f in files:
            p = os.path.join(d, f)
            if p == cfg["auth"]:
                continue
            ours = p.startswith(tuple(cfg["name_scan"]))  # copied by the snapshot; elsewhere an install or /etc
            bad = (name_re.search(f) and ours) or ("/data/" in p and p.startswith(cfg["repo"]))
            if not bad and (p.startswith(tuple(cfg["text_scan"])) or name_re.search(f)) and os.path.isfile(p) and not os.path.islink(p) and os.path.getsize(p) <= 1 << 20:
                try:
                    bad = text_re.search(open(p, "rb").read())
                except OSError:
                    pass
            if bad:
                found.append(p)
out["credentials_readable"] = found[:40]
out["named_secrets_readable"] = [p for p in cfg["named_secrets"] if os.path.exists(p)]
try:
    tokens = json.load(open(cfg["auth"]))["tokens"]
    out["login_refresh_token"] = tokens.get("refresh_token")
    out["login_signatures"] = [str(tokens.get(k)).split(".")[-1] for k in ("access_token", "id_token")]
except (OSError, ValueError, KeyError) as e:
    out["login_refresh_token"] = out["login_signatures"] = "unreadable: " + str(e)
out["tmpfs_bytes"] = {p: os.statvfs(p).f_blocks * os.statvfs(p).f_frsize for p in ("/tmp", os.environ["CODEX_HOME"])}
out["routes"] = {}
for name, argv in cfg["routes"].items():
    p = subprocess.run(argv, capture_output=True, text=True)
    out["routes"][name] = {"exit": p.returncode, "out": (p.stdout + p.stderr).strip()[-160:]}
out["effort_folder_exists"] = os.path.exists(cfg["effort"])
out["harness_visible"] = [p for p in cfg["hidden"] if os.path.exists(p)]
out["pids"] = len([d for d in os.listdir("/proc") if d.isdigit()])
out["pid1"] = open("/proc/1/cmdline", "rb").read().split(b"\0")[0].decode()
print(json.dumps(out))
'''


def host_listeners() -> tuple[list[int], list[str]]:
    """Loopback or wildcard TCP ports listening on the host, and Unix socket paths, from /proc/net."""
    ports = set()
    for f, loop in (("/proc/net/tcp", ("0100007F", "00000000")), ("/proc/net/tcp6", ("00000000000000000000000001000000",
                                                                                    "00000000000000000000000000000000"))):
        try:
            rows = Path(f).read_text().splitlines()[1:]
        except OSError:
            continue
        for row in rows:
            cols = row.split()
            addr, port = cols[1].split(":")
            if cols[3] == "0A" and addr in loop:
                ports.add(int(port, 16))
    socks = set()
    try:
        for row in Path("/proc/net/unix").read_text().splitlines()[1:]:
            cols = row.split()
            if len(cols) >= 8 and cols[7].startswith("/"):
                socks.add(cols[7])
    except OSError:
        pass
    return sorted(ports), sorted(socks)


def cmd_sandbox_check(ws: WS, variant: str) -> dict:
    """Prove the agent sandbox, by trying what it must refuse: writes to the repository, research home,
    skills, CLI state, CLI binaries and home folder; every host loopback port and Unix socket; the
    internet; every gate request but the model API (CONNECT, other paths, absolute and dot-dot targets,
    chunked, compressed or non-JSON bodies, provider-side tools, WebSocket upgrades, malformed header
    lines, raw TLS, a second request pipelined after the first, a foreign Host); any readable
    credential-like file; the login's refresh token; the mutating routes; the effort folder and the
    harness; and the /tmp and file size caps."""
    need_snapshot(ws)
    research = research_home(ws)
    info = prompt_info(ws, next((p.name for p in sorted((ws.root / "commissions").iterdir())
                                 if ws.prompts(p.name, variant).is_dir()), ""), variant) if (ws.root / "commissions").is_dir() else {}
    cohort = info.get("cohort")
    topic = next(iter(info.get("topics") or []), None)
    probe = f".replay-probe-{os.getpid()}"
    ports, socks = host_listeners()
    auth = HOME / ".codex/auth.json"
    auth_before = sha(auth.read_bytes()) if auth.exists() else None
    writes = {"repo": REPO / probe, "research_home": research / probe, "skills": SKILLS / probe,
              "cli_state": HOME / ".vrooli/state" / probe, "cli_bin": HOME / ".vrooli/bin" / probe, "home": HOME / probe,
              "effort_folder": EFFORT_DIR / probe}
    if topic:
        writes["topic"] = research / "topics" / topic / probe
    if cohort:
        writes["cohort"] = research / "cohorts" / cohort / probe
    api, resp = "/backend-api/codex/models?client_version=0.0.0", "/backend-api/codex/responses"

    def post(body, length=0) -> str:  # a model turn with one thing Codex never sends
        body = body if isinstance(body, str) else json.dumps({"model": "m", "input": [], **body})
        return f"POST {resp} HTTP/1.1\r\nContent-Type: application/json\r\nContent-Length: {len(body) + length}{end}{body}"

    def tools(*t) -> dict:
        return {"input": [{"type": "additional_tools", "role": "developer", "tools": list(t)}]}

    def user(*parts) -> dict:
        return {"input": [{"type": "message", "role": "user", "content": list(parts)}]}

    end = "\r\n\r\n"
    cfg = {"writes": {k: str(v) for k, v in writes.items()}, "ports": ports, "sockets": socks, "proxy_port": PROXY_PORT,
           "provider": PROVIDER, "gate_requests": {
               "CONNECT example.com:443": f"CONNECT example.com:443 HTTP/1.1\r\nHost: example.com{end}",
               "CONNECT 127.0.0.1:22": f"CONNECT 127.0.0.1:22 HTTP/1.1{end}",
               f"CONNECT {PROVIDER}:80": f"CONNECT {PROVIDER}:80 HTTP/1.1{end}",
               "plain GET absolute http://example.com/": f"GET http://example.com/ HTTP/1.1\r\nHost: example.com{end}",
               "plain GET of the model API": f"GET {api} HTTP/1.1\r\nHost: {PROVIDER}{end}",
               "raw TLS to the gate": ("", "direct-tls"),
               "tunnel, SNI example.com (domain fronting)": (f"GET / HTTP/1.1\r\nHost: example.com{end}", "example.com"),
               "tunnel, SNI cloudflare.com": (f"GET / HTTP/1.1\r\nHost: www.cloudflare.com{end}", "cloudflare.com"),
               "tunnel, GET / (not the model API)": (f"GET / HTTP/1.1\r\nHost: {PROVIDER}{end}", PROVIDER),
               "tunnel, Host example.com, GET /": (f"GET / HTTP/1.1\r\nHost: example.com{end}", PROVIDER),
               "tunnel, dot-dot out of the API": (f"GET /backend-api/codex/../../ HTTP/1.1\r\nHost: {PROVIDER}{end}", PROVIDER),
               "tunnel, wham/accounts/check-x": (f"GET /backend-api/wham/accounts/check-x HTTP/1.1{end}", PROVIDER),
               "tunnel, WebSocket upgrade": (f"GET {resp} HTTP/1.1\r\nUpgrade: websocket\r\nConnection: Upgrade{end}", PROVIDER),
               "tunnel, colonless header line": (f"GET {api} HTTP/1.1\r\njunk-line{end}", PROVIDER),
               "tunnel, obs-fold header line": (f"GET {api} HTTP/1.1\r\nX-T: a\r\n b{end}", PROVIDER),
               "tunnel, top-level tools web_search": (post({"tools": [{"type": "web_search"}]}), PROVIDER),
               "tunnel, tool web_search": (post(tools({"type": "web_search"})), PROVIDER),
               "tunnel, tool mcp at an outside URL": (post(tools({"type": "mcp", "server_label": "x",
                                                                  "server_url": "https://example.com/mcp"})), PROVIDER),
               "tunnel, tool image_generation beside a function": (post(tools({"type": "function", "name": "f"},
                                                                               {"type": "image_generation"})), PROVIDER),
               "tunnel, function tool with a server_url": (post(tools({"type": "function", "name": "f",
                                                                       "server_url": "https://example.com"})), PROVIDER),
               "tunnel, duplicate key hiding a web_search tool": (post(
                   '{"model":"m","input":[{"type":"additional_tools","role":"developer","tools":[{"type":"web_search"}],'
                   '"tools":[]}]}'), PROVIDER),
               "tunnel, input_image at an outside URL": (post(user({"type": "input_image",
                                                                    "image_url": "https://example.com/x.png"})), PROVIDER),
               "tunnel, input_file at an outside URL": (post(user({"type": "input_file",
                                                                   "file_url": "https://example.com/x.pdf"})), PROVIDER),
               "tunnel, mcp_call item": (post({"input": [{"type": "mcp_call", "server_label": "s", "name": "n"}]}), PROVIDER),
               "tunnel, previous_response_id": (post({"previous_response_id": "resp_1"}), PROVIDER),
               "tunnel, Content-Length beyond the body": (post({}, length=50), PROVIDER),
               "tunnel, header Codex never sends": (f"GET {api} HTTP/1.1\r\nX-Forwarded-Host: example.com{end}", PROVIDER),
               "tunnel, models with another query": (f"GET {api}&q=1 HTTP/1.1{end}", PROVIDER),
               "tunnel, accounts check with a query": (f"GET /backend-api/wham/accounts/check?x=1 HTTP/1.1{end}", PROVIDER),
               "tunnel, body not JSON": (f"POST {resp} HTTP/1.1\r\nContent-Length: 3{end}abc", PROVIDER),
               "tunnel, compressed body": (f"POST {resp} HTTP/1.1\r\nContent-Encoding: zstd\r\nContent-Length: 2{end}{{}}", PROVIDER),
               "tunnel, chunked POST": (f"POST {resp} HTTP/1.1\r\nTransfer-Encoding: chunked{end}0{end}",
                                        PROVIDER),
               "tunnel, bare LF smuggling a Host header": (f"GET {api} HTTP/1.1\r\nX-A: a\nHost: example.com{end}", PROVIDER),
               "tunnel, NUL in a header": (f"GET {api} HTTP/1.1\r\nX-A: a\x00b{end}", PROVIDER),
               "model API, Host example.com": (f"GET {api} HTTP/1.1\r\nHost: example.com{end}", PROVIDER),
               "model API, then a pipelined Host example.com": (
                   f"GET {api} HTTP/1.1\r\nHost: {PROVIDER}\r\nConnection: keep-alive{end}"
                   f"GET / HTTP/1.1\r\nHost: example.com{end}", PROVIDER),
           },
           "routes": {"research-desk say": ["research-desk", "say", "replay probe"],
                      "agent-manager run": ["agent-manager", "run", "--help"],
                      "notification-hub send": ["notification-hub", "send", "probe"],
                      "vrooli scenario stop": ["vrooli", "scenario", "stop", "research-desk"],
                      "real research-desk binary: say": ["/tmp/.probe/research-desk", "say", "replay probe"],
                      "prompt-manager skill read": ["prompt-manager", "skill", "read", "research-coordination", "--lines", "1-1"]},
           "effort": str(EFFORT_DIR), "hidden": [str(SKILLS / "research-review/scripts" / f) for f in REPLAY_FILES],
           "secret_name": SECRET_NAME.pattern, "secret_text": SECRET_TEXT.pattern.decode("latin1"),
           "scan": [str(HOME), "/etc"], "name_scan": [str(REPO), str(research), str(SKILL_EXPORT)],
           "text_scan": [str(REPO), str(research), str(SKILL_EXPORT), "/etc"], "repo": str(REPO),
           "auth": str(HOME / ".codex/auth.json"),
           "named_secrets": [str(REPO / r) for r in ("scenarios/agent-manager/data/identity-secret.key",
                                                     "scenarios/agent-manager/data/agent-manager.db",
                                                     "scenarios/system-monitor/.env", "scenarios/app-monitor/ui/.env",
                                                     "data/resources/home-assistant/config/secrets.yaml")]}
    cfg["gate_requests"] = {k: v if isinstance(v, tuple) else (v, None) for k, v in cfg["gate_requests"].items()}
    replay_tmp = HOME / ".vrooli/tmp/research-replay"  # where earlier harness versions kept Codex homes
    before = {str(p) for p in replay_tmp.rglob("*")} if replay_tmp.is_dir() else set()
    with Gate() as gate:
        real = HOME / ".vrooli/bin/research-desk"  # the real CLI, to show a mutating call cannot reach its service
        cmd = sandbox_cmd(DRIVE, ws=ws, variant=variant, cohort=cohort, gate=gate,
                          extra=("--ro-bind", str(real), "/tmp/.probe/research-desk") if real.is_file() else ())
        p = subprocess.run(cmd, input=json.dumps(codex_header([["python3", "-I", "-c", PROBE, json.dumps(cfg)]])) + "\n",
                           capture_output=True, text=True, timeout=600, env=sandbox_env())
        net = gate.summary()
    transport = transport_probe(ws, variant, cohort)
    after = {str(p) for p in replay_tmp.rglob("*")} if replay_tmp.is_dir() else set()
    new_on_host = sorted(after - before) + ([str(gate.dir)] if gate.dir.exists() else [])
    logins_on_host = sorted(p for p in after if p.endswith("/auth.json"))
    got = next((json.loads(x) for x in reversed(p.stdout.splitlines()) if x.startswith('{"writes"')), None)
    if got is None:
        return {"ok": False, "error": "the probe did not report", "stdout": p.stdout[-2000:], "stderr": p.stderr[-2000:]}
    leaked = [str(v) for v in writes.values() if v.exists()]
    checks = {
        "every write refused": all(x.startswith("refused") for x in got["writes"].values()),
        "no host loopback port reachable": not got["host_ports_reachable"],
        "no host Unix socket visible": not got["host_sockets_visible"],
        "no direct internet": got["direct_internet"].startswith("refused"),
        "gate refuses all but the model API": all(
            isinstance(r, str) or r["gate_refused"] for k, r in got["gate"].items() if not k.startswith("model API")),
        "model API forwarded, one request per connection": all(
            isinstance(got["gate"][k], dict) and not got["gate"][k]["gate_refused"] and got["gate"][k]["responses"] == 1
            for k in got["gate"] if k.startswith("model API")),
        "no credential-like file readable": not got["credentials_readable"] and not got["named_secrets_readable"],
        "login copy has no refresh token": got["login_refresh_token"] == "",
        "/tmp and Codex home are capped tmpfs": 0 < got["tmpfs_bytes"]["/tmp"] <= TMP_CAP
                                                 and 0 < got["tmpfs_bytes"][str(HOME / ".codex")] <= CODEX_HOME_CAP,
        "nothing left on the host by this check": not new_on_host,
        "no login copy anywhere under ~/.vrooli/tmp/research-replay": not logins_on_host,
        "sandbox tokens carry a dummy signature": got["login_signatures"] == [DUMMY_SIGNATURE] * 2,
        "mutating routes refused": all(got["routes"][k]["exit"] == 69 for k in cfg["routes"] if k.split()[0] in ROUTE_CLIS
                                       and k != "prompt-manager skill read")
                                   and got["routes"]["real research-desk binary: say"]["exit"] not in (0, None),
        "skill reads served": got["routes"]["prompt-manager skill read"]["exit"] == 0,
        "effort folder absent": not got["effort_folder_exists"],
        "harness invisible": not got["harness_visible"],
        "own pid namespace": got["pid1"] != "" and "systemd" not in got["pid1"] and got["pids"] < 20,
        "nothing leaked to the host": not leaked,
        "host login unchanged": (sha(auth.read_bytes()) if auth.exists() else None) == auth_before,
        "replay turns use HTTPS only, no WebSocket attempt": transport["https_turns"] > 0 and not transport["websocket"],
    }
    return {"ok": all(checks.values()), "variant": variant, "checks": checks, "probe": got, "new_on_host": new_on_host,
            "logins_on_host": logins_on_host,
            "host_ports_tried": len(ports), "host_sockets_tried": len(socks), "gate": net, "leaked": leaked,
            "transport": transport}


def transport_probe(ws: WS, variant: str, cohort: str | None) -> dict:
    """A replay turn exactly as cmd_run starts it (codex_turn in the sandbox, through a gate), except
    that this gate refuses every model turn, so no model call is made: the turn must reach the model
    API as an HTTPS POST and never try a WebSocket upgrade."""
    with Gate(model_turns=False) as gate:
        inner = codex_turn("Reply OK.", model="gpt-6-luna", effort="low", cwd=REPO, resume=None, dev=None)
        p = subprocess.run(sandbox_cmd(DRIVE, ws=ws, variant=variant, cohort=cohort, gate=gate),
                           input=json.dumps(codex_header([inner])) + "\n", capture_output=True, text=True,
                           timeout=300, env=sandbox_env())
        net = gate.summary()
    def load(line):
        try:
            return json.loads(line)
        except ValueError:
            return None
    events = [e for e in map(load, p.stdout.splitlines()) if isinstance(e, dict)]
    return {"https_turns": sum(r.startswith("POST /backend-api/codex/responses") for r in net["denied"]),
            "websocket": websocket_attempts(net, p.stderr, events), "denied": net["denied"]}


# ---------------------------------------------------------------- main


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--workspace", type=Path, default=WORKSPACE)
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("snapshot")
    p.add_argument("cids", nargs="+")
    p.add_argument("--research-home", type=Path, default=RESEARCH)
    p = sub.add_parser("freeze")
    p.add_argument("cids", nargs="+")
    p.add_argument("--baseline-skills", type=Path, default=BASELINE_SKILLS)
    p = sub.add_parser("render-new")
    p.add_argument("cids", nargs="+")
    p.add_argument("--stamp", default=dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%MZ"))
    p = sub.add_parser("run")
    p.add_argument("cid")
    p.add_argument("--variant", required=True)
    p.add_argument("--role", choices=("supervisor", "owner"), required=True)
    p.add_argument("--topic")
    p.add_argument("--card", type=int)
    p.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT)
    p.add_argument("--model", help="override the role's model (recorded); default: the packet's or limits.json's")
    p = sub.add_parser("batch", help="run every prompt for every variant, interleaved, several at a time")
    p.add_argument("--cids", nargs="+", required=True)
    p.add_argument("--variants", nargs="+", required=True)
    p.add_argument("--roles", nargs="+", default=["supervisor", "owner"])
    p.add_argument("--owner-topics", nargs="*", default=[])
    p.add_argument("--repeat", type=int, default=6)
    p.add_argument("--parallel", type=int, default=5)
    p.add_argument("--timeout", type=int, default=DEFAULT_TIMEOUT)
    p = sub.add_parser("metrics")
    p.add_argument("runs", nargs="+", type=Path)
    p = sub.add_parser("grade")
    p.add_argument("run_a", type=Path)
    p.add_argument("run_b", type=Path)
    p.add_argument("--topic", required=True)
    p.add_argument("--model", default="gpt-6-sol")
    p.add_argument("--effort", default="high")
    p.add_argument("--timeout", type=int, default=900)
    p = sub.add_parser("grade-all", help="grade arm against arm per prompt and topic, plus same-arm noise pairs")
    p.add_argument("--arm", action="append", required=True)
    p.add_argument("--noise", type=int, default=6)
    p.add_argument("--parallel", type=int, default=3)
    p.add_argument("--model", default="gpt-6-sol")
    p.add_argument("--effort", default="high")
    p.add_argument("--timeout", type=int, default=900)
    p = sub.add_parser("preregister", help="write the arms, expected effect and claims before the runs")
    p.add_argument("--arm", action="append", required=True)
    p.add_argument("--expected", required=True)
    p.add_argument("--claim", action="append", default=[])
    p = sub.add_parser("exclude", help="keep a run on disk but out of the report, with the reason")
    p.add_argument("runs", nargs="+", type=Path)
    p.add_argument("--reason", required=True)
    p = sub.add_parser("report")
    p.add_argument("--arm", action="append")
    p.add_argument("--correction", help="record a post-run correction that changes no pre-registered measure")
    p = sub.add_parser("sandbox-check")
    p.add_argument("--variant", default="old")
    args = ap.parse_args(argv)
    ws = WS(args.workspace.expanduser().absolute())
    resolve = lambda r: r if r.is_absolute() and r.exists() else (ws.runs() / r.name)
    try:
        if args.cmd == "snapshot":
            res = cmd_snapshot(ws, args.cids, args.research_home.expanduser().absolute())
        elif args.cmd == "freeze":
            res = cmd_freeze(ws, args.cids, args.baseline_skills)
        elif args.cmd == "render-new":
            res = cmd_render_new(ws, args.cids, args.stamp)
        elif args.cmd == "run":
            if args.role == "owner" and not args.topic:
                raise Fail("--role owner needs --topic")
            res = cmd_run(ws, args.cid, args.variant, args.role, args.topic, args.card, args.timeout, args.model)
        elif args.cmd == "batch":
            jobs = batch_jobs(ws, args.cids, args.variants, args.roles, args.owner_topics, args.repeat)
            done = run_jobs(jobs, lambda j: cmd_run(ws, j["cid"], j["variant"], j["role"], j["topic"], None,
                                                    args.timeout, None), args.parallel)
            res = {"jobs": len(jobs), "failed": [d for d in done if "error" in d],
                   "runs": [d["result"]["run"].rsplit("/", 1)[-1] for d in done if "result" in d]}
        elif args.cmd == "metrics":
            res = {str(r): {k: v for k, v in analyse_run(r.absolute()).items() if k != "paths_before_first_action"}
                   for r in args.runs}
        elif args.cmd == "grade":
            res = cmd_grade(ws, resolve(args.run_a), resolve(args.run_b), args.topic, args.model, args.effort, args.timeout)
        elif args.cmd == "grade-all":
            pairs = grade_pairs(ws, parse_arms(args.arm, ws, []), args.noise)
            pairs = [x for x in pairs if not (ws.grades() / f"{x['topic']}.{x['a']}.vs.{x['b']}" / "grade.json").exists()]
            done = run_jobs(pairs, lambda x: cmd_grade(ws, ws.runs() / x["a"], ws.runs() / x["b"], x["topic"], args.model,
                                                       args.effort, args.timeout), args.parallel)
            res = {"pairs": len(pairs), "failed": [d for d in done if "error" in d]}
        elif args.cmd == "preregister":
            res = cmd_preregister(ws, args.arm, args.expected, args.claim)
        elif args.cmd == "exclude":
            res = {}
            for r in args.runs:
                d = resolve(r)
                meta = read_json(d / "meta.json")
                if meta is None:
                    raise Fail(f"{d} has no meta.json")
                meta["excluded"] = args.reason
                write_json(d / "meta.json", meta)
                res[d.name] = args.reason
        elif args.cmd == "report":
            res = {"report": str(cmd_report(ws, args.arm, args.correction))}
        else:
            res = cmd_sandbox_check(ws, args.variant)
    except Fail as e:
        print(json.dumps({"ok": False, "error": str(e)}))
        return 2
    print(json.dumps(res, indent=2, ensure_ascii=False))
    return 0 if not (isinstance(res, dict) and res.get("ok") is False) else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
