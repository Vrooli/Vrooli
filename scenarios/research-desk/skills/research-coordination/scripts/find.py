#!/usr/bin/env python3
"""Search research files with citations. Uses Source Ledger when it answers; reads the files otherwise.

Usage:
  find.py "<words>" [--topic NAME] [--feed NAME] [--limit N] [--json] [--files] [--home DIR]

One command for research agents. It asks Source Ledger's file corpus first (fresh: every
read syncs its index against the files; lexical plus semantic ranking). If Source Ledger
is down, slow or missing, it walks the same files with the same rules (feeds.json, run
depth, dataset folders, size caps) and ranks them itself. Both paths print the same
shape: `path#Lstart-Lend — heading`, a snippet, and a score; --json gives the API's
JSON. `source` says which path answered. --files skips Source Ledger.
Exit 0 with hits, 1 with none, 2 on bad usage.
"""

from __future__ import annotations

import argparse
import fnmatch
import json
import math
import os
import re
import subprocess
import sys
import time
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

# Mirrors scenarios/source-ledger/api/internal/files (policy.go, walk.go, chunk.go).
KINDS = {".md": "markdown", ".markdown": "markdown", ".txt": "text", ".yaml": "text", ".yml": "text",
         ".json": "json", ".jsonl": "jsonl", ".ndjson": "jsonl", ".csv": "csv", ".tsv": "tsv"}
TAIL = {"jsonl", "csv", "tsv"}
PRUNED = {"node_modules", ".git", ".venv", "venv", "__pycache__", "dist", "build", "target", ".next",
          "persist", "vendor", "site-packages", ".cache", "coverage", ".pytest_cache"}
CODE_MARKERS = {".git", "go.mod", "package.json", "pyproject.toml", "Cargo.toml"}
DEFAULTS = {"run_depth": 2, "dataset_files": 200, "max_file_kb": 256, "max_json_kb": 64}
PRIVATE = ["**/OPERATOR.md", "**/OPERATOR-*.md", "**/STANCE-MESSAGE.txt", "**/OPERATOR-WORDS*",
           "**/*resume*", "**/*Resume*", "**/personal/**", "**/private/**", "**/*.private.*"]
LIMIT_KEYS = tuple(DEFAULTS)
WINDOW, SECTION, RECORDS, ROWS, MAX_CHARS, PER_FILE = 40, 60, 10, 20, 4000, 3
SL_PATH = "/vrooli.source_ledger.v1.files.FilesService/Search"


def home_dir(arg: str | None) -> Path:
    return Path(arg or os.environ.get("VROOLI_RESEARCH_HOME") or Path.home() / ".vrooli" / "research")


def load_policy(folder: Path, base: dict, problems: list[str]) -> dict | None:
    """Policy for folder over base's limits; None when its feeds.json is malformed."""
    policy = {k: base[k] for k in LIMIT_KEYS} | {"exclude": [], "include": [], "feeds": {}, "private": None}
    try:
        raw = json.loads((folder / "feeds.json").read_text())
        if not isinstance(raw, dict):
            raise ValueError("not an object")
    except FileNotFoundError:
        return policy
    except (OSError, ValueError) as err:
        problems.append(f"{folder / 'feeds.json'}: {err}")
        return None
    for key in LIMIT_KEYS:
        if isinstance(raw.get(key), int) and raw[key] > 0:
            policy[key] = raw[key]
    for key in ("exclude", "include"):
        policy[key] = list(raw.get(key) or [])
    policy["feeds"] = dict(raw.get("feeds") or {})
    if isinstance(raw.get("private"), list):
        policy["private"] = list(raw["private"])
    return policy


def match(glob: str, rel: str) -> bool:
    def seg(g: list[str], p: list[str]) -> bool:
        while g:
            if g[0] == "**":
                return any(seg(g[1:], p[i:]) for i in range(len(p) + 1))
            if not p or not fnmatch.fnmatchcase(p[0], g[0]):
                return False
            g, p = g[1:], p[1:]
        return not p
    return seg(glob.strip("/").split("/"), rel.split("/"))


def any_match(globs: list[str], rel: str) -> bool:
    return any(match(g, rel) or match(g + "/**", rel) for g in globs)


def may_contain(glob: str, rel: str) -> bool:
    g = glob.strip("/").split("/")
    for i, part in enumerate(rel.split("/")):
        if i >= len(g):
            return False
        if g[i] == "**":
            return True
        if not fnmatch.fnmatchcase(part, g[i]):
            return False
    return True


def topic_of(rel: str) -> tuple[str, str]:
    parts = rel.split("/", 2)
    if len(parts) >= 2 and parts[0] == "topics":
        return parts[1], parts[2] if len(parts) == 3 else ""
    return "", rel


def run_depth(rel: str) -> int:
    parts = rel.split("/")
    for i, part in enumerate(parts):
        if part == "runs" and i + 1 < len(parts):
            return len(parts) - (i + 2)
    return -1


def summary_file(name: str) -> bool:
    return name.lower().startswith(("readme", "index", "summary"))


class Walker:
    """Mirror of Source Ledger's walk.go: same files in, same files left out."""

    def __init__(self, root: Path):
        self.root, self.topics, self.broken, self.entries, self.problems = root, {}, set(), [], []
        self.stats = {"private": 0, "excluded": 0, "summarised": 0, "oversizeJson": 0, "truncated": 0}
        base = load_policy(root, DEFAULTS, self.problems)
        if base is None:
            self.problems[-1] += " (using defaults)"
            base = load_policy(Path("/nonexistent"), DEFAULTS, [])
        self.base = base

    def policy(self, topic: str) -> dict:
        if not topic:
            return self.base
        if topic not in self.topics:
            p = load_policy(self.root / "topics" / topic, self.base, self.problems)
            if p is None:
                self.problems[-1] += " (indexing only this topic's README/INDEX/SUMMARY files until it is fixed)"
                self.broken.add(topic)
                p = {k: self.base[k] for k in LIMIT_KEYS} | {"exclude": [], "include": [], "feeds": {}, "private": None}
            self.topics[topic] = p
        return self.topics[topic]

    def _globs(self, rel: str, key: str) -> list[tuple[list[str], str]]:
        topic, inner = topic_of(rel)
        out = [(self.base[key], rel)]
        if topic and inner:
            out.append((self.policy(topic)[key], inner))
        return out

    def private(self, rel: str) -> bool:
        topic, inner = topic_of(rel)
        if topic and inner and self.policy(topic)["private"] is not None:
            return any_match(self.policy(topic)["private"], inner)
        globs = self.base["private"] if self.base["private"] is not None else PRIVATE
        return any_match(globs, rel)

    def excluded(self, rel: str) -> bool:
        return any(any_match(g, r) for g, r in self._globs(rel, "exclude"))

    def included(self, rel: str) -> bool:
        return any(any_match(g, r) for g, r in self._globs(rel, "include"))

    def may_include(self, rel: str) -> bool:
        return any(may_contain(x, r) for g, r in self._globs(rel, "include") for x in g)

    def feed(self, rel: str) -> str:
        topic, inner = topic_of(rel)
        if topic:
            feeds = self.policy(topic)["feeds"]
            for name in sorted(feeds):
                if any_match(feeds[name], inner):
                    return name
        else:
            inner = rel
        return inner.split("/", 1)[0] if "/" in inner else "(root)"

    def walk(self, rel: str = "", inherited: bool = False) -> None:
        try:
            items = list(os.scandir(self.root / rel if rel else self.root))
        except OSError:
            self.problems.append("unreadable folder: " + rel)
            return
        topic, _ = topic_of(rel)
        policy = self.policy(topic)
        files, dirs = [], []
        code = bool(rel) and any(i.name in CODE_MARKERS for i in items)
        for item in items:
            if item.is_symlink():
                continue
            if item.is_dir(follow_symlinks=False):
                if item.name not in PRUNED and not item.name.startswith("."):
                    dirs.append(item.name)
            elif item.is_file(follow_symlinks=False) and item.name != "feeds.json" and Path(item.name).suffix.lower() in KINDS:
                files.append(item)
        summarised = (inherited or code or topic in self.broken or len(files) > policy["dataset_files"]
                      or run_depth(rel) >= policy["run_depth"])
        for item in files:
            child = f"{rel}/{item.name}" if rel else item.name
            if self.private(child):
                self.stats["private"] += 1
                continue
            if self.excluded(child):
                self.stats["excluded"] += 1
                continue
            if summarised and not summary_file(item.name) and not self.included(child):
                self.stats["summarised"] += 1
                continue
            st = item.stat(follow_symlinks=False)
            kind = KINDS[Path(item.name).suffix.lower()]
            if kind == "json" and st.st_size > policy["max_json_kb"] * 1024:
                self.stats["oversizeJson"] += 1
                continue
            limit = policy["max_file_kb"] * 1024
            self.stats["truncated"] += st.st_size > limit
            self.entries.append({"path": child, "size": st.st_size, "mtime": st.st_mtime, "kind": kind, "topic": topic,
                                 "feed": self.feed(child), "max": limit})
        for name in sorted(dirs):
            child = f"{rel}/{name}" if rel else name
            if not self.excluded(child) and (not code or self.may_include(child)):
                self.walk(child, summarised)


def read_capped(path: Path, kind: str, size: int, limit: int) -> tuple[str, int]:
    with open(path, "rb") as f:
        if size <= limit:
            return f.read().decode("utf-8", "replace"), 1
        if kind not in TAIL:
            data = f.read(limit)
            cut = data.rfind(b"\n")
            return (data[:cut + 1] if cut >= 0 else data).decode("utf-8", "replace"), 1
        skipped = 1 + f.read(size - limit).count(b"\n")
        data = f.read()
        nl = data.find(b"\n")
        if nl >= 0:
            data, skipped = data[nl + 1:], skipped + 1
        return data.decode("utf-8", "replace"), skipped


def chunks(kind: str, text: str, first: int) -> list[dict]:
    lines = text.replace("\r\n", "\n").rstrip("\n").split("\n")
    if len(lines) == 1 and not lines[0].strip():
        return []
    out: list[dict] = []

    def add(s: int, e: int, heading: str, body: str) -> None:
        if body.strip():
            out.append({"start": first + s, "end": first + e - 1, "heading": heading, "text": body.strip()[:MAX_CHARS]})

    if kind == "markdown":
        trail, start, fence = [], 0, False

        def flush(end: int) -> None:
            label = " › ".join(t for _, t in trail)
            for s in range(start, end, SECTION):
                e = min(s + SECTION, end)
                add(s, e, label, "\n".join(lines[s:e]))
        for i, line in enumerate(lines):
            t = line.strip()
            if t.startswith(("```", "~~~")):
                fence = not fence
            if fence or not t.startswith("#"):
                continue
            level = len(t) - len(t.lstrip("#"))
            if level > 6 or len(t) == level or t[level] != " ":
                continue
            flush(i)
            start = i
            while trail and trail[-1][0] >= level:
                trail.pop()
            trail.append((level, t[level:].strip()))
        flush(len(lines))
    elif kind in ("csv", "tsv"):
        header, body, offset = (lines[0], lines[1:], 1) if first == 1 else ("", lines, 0)
        for s in range(0, len(body), ROWS):
            e = min(s + ROWS, len(body))
            a = first + offset + s
            text_ = (header + "\n" if header else "") + "\n".join(body[s:e])
            out.append({"start": a, "end": a + e - s - 1, "heading": f"rows {a}-{a + e - s - 1}", "text": text_[:MAX_CHARS]})
    else:
        size, label = (RECORDS, "records") if kind == "jsonl" else (WINDOW, "")
        for s in range(0, len(lines), size):
            e = min(s + size, len(lines))
            add(s, e, f"{label} {first + s}-{first + e - 1}" if label else "", "\n".join(lines[s:e]))
    return out


TOKEN = re.compile(r"[^\W_]+", re.UNICODE)


def terms(text: str) -> list[str]:
    seen: dict[str, None] = {}
    for t in TOKEN.findall(text.lower()):
        if len(t) > 1:
            seen.setdefault(t)
    return list(seen)


def snippet(text: str, words: list[str], width: int = 280) -> str:
    low = text.lower()
    pos = min((p for p in (low.find(w) for w in words) if p >= 0), default=0)
    start = max(0, pos - width // 3)
    body = " ".join(text[start:start + width].split())
    return ("… " if start else "") + body + (" …" if start + width < len(text) else "")


RAW_DIRS = {"runs", "support", "preimages", "history", "replay"}
RAW_WEIGHT = 0.5


def authority(rel: str, include_raw: bool = False) -> float:
    """Same prior as Source Ledger: entry files over evidence; raw material (run output, support
    copies, preimages, history, compiled files) lower unless asked for."""
    name = rel.rsplit("/", 1)[-1].upper()
    w = 1.25 if name in ("SKILL.MD", "MISSION.MD", "README.MD") or name.endswith("MAP.MD") else 1.0
    if not include_raw and (rel.endswith(".pyc") or RAW_DIRS & set(rel.split("/")[:-1])):
        w *= RAW_WEIGHT
    return w


def distinct_files_first(scored: list, limit: int) -> list:
    """Each file's best chunk first, then further chunks (up to PER_FILE per file) in rank order."""
    out, per, taken = [], {}, set()
    for rnd in range(1, PER_FILE + 1):
        for i, item in enumerate(scored):
            if len(out) == limit:
                return out
            if i not in taken and per.get(item[1]["path"], 0) == rnd - 1:
                per[item[1]["path"]] = rnd
                taken.add(i)
                out.append(item)
    return out


def search_files(home: Path, query: str, topic: str, feed: str, limit: int, include_raw: bool = False) -> dict:
    started = time.time()
    if topic and ("/" in topic or topic in (".", "..")):
        raise SystemExit(f"find.py: invalid topic {topic!r}")
    words = terms(query)
    walker = Walker(home)
    walker.walk(f"topics/{topic}" if topic else "")
    pool, df = [], {}
    for e in walker.entries:
        if feed and e["feed"] != feed:
            continue
        try:
            text, first = read_capped(home / e["path"], e["kind"], e["size"], e["max"])
        except OSError:
            continue
        low = text.lower()
        if not any(w in low for w in words):
            continue
        for c in chunks(e["kind"], text, first):
            toks = TOKEN.findall((e["path"] + " " + c["heading"] + " " + c["text"]).lower())
            tf = {w: toks.count(w) for w in words}
            if not any(tf.values()):
                continue
            for w, n in tf.items():
                df[w] = df.get(w, 0) + (1 if n else 0)
            head = (e["path"] + " " + c["heading"]).lower()
            pool.append((e, c, tf, len(toks), sum(1 for w in words if w in head)))
    n = max(len(pool), 1)
    avg = sum(p[3] for p in pool) / n if pool else 1
    scored = []
    for e, c, tf, length, head_hits in pool:
        s = sum(math.log(1 + (n - df[w] + 0.5) / (df[w] + 0.5)) * tf[w] * 2.2 / (tf[w] + 1.2 * (0.25 + 0.75 * length / avg))
                for w in words if tf[w])
        scored.append(((s + head_hits) * authority(e["path"], include_raw), e, c))
    scored.sort(key=lambda x: -x[0])
    best = scored[0][0] if scored else 1
    hits = []
    for s, e, c in distinct_files_first(scored, limit):
        hits.append({"citation": f"{e['path']}#L{c['start']}-L{c['end']}", "path": e["path"], "absPath": str(home / e["path"]),
                     "topic": e["topic"], "feed": e["feed"], "lineStart": c["start"], "lineEnd": c["end"], "heading": c["heading"],
                     "snippet": snippet(c["text"], words), "score": round(s / best, 3), "truncated": e["size"] > e["max"],
                     "modifiedAt": datetime.fromtimestamp(e["mtime"], timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")})
    st = walker.stats
    problems = sorted(set(walker.problems))
    degradations = []
    if problems:
        detail = "; ".join(problems[:3]) + (f"; and {len(problems) - 3} more" if len(problems) > 3 else "")
        degradations.append({"reason": "CORPUS_PROBLEMS", "leg": "walk", "detail": detail})
    return {"hits": hits, "mode": "lexical", "source": "files",
            "sync": {"filesSeen": len(walker.entries), "durationMs": int((time.time() - started) * 1000),
                     "syncedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
                     "skippedPrivate": st["private"], "skippedExcluded": st["excluded"], "summarised": st["summarised"],
                     "skippedOversizeJson": st["oversizeJson"], "truncated": st["truncated"], "problems": problems},
            "degradations": degradations}


def source_ledger_base() -> str | None:
    if os.environ.get("SOURCE_LEDGER_API_BASE"):
        return os.environ["SOURCE_LEDGER_API_BASE"].rstrip("/")
    try:
        port = subprocess.run(["vrooli", "scenario", "port", "source-ledger", "API_PORT"], capture_output=True, text=True, timeout=4).stdout.strip()
    except (OSError, subprocess.SubprocessError):
        return None
    return f"http://127.0.0.1:{port}" if port.isdigit() else None


def search_ledger(base: str, query: str, topic: str, feed: str, limit: int, include_raw: bool = False) -> dict:
    body = json.dumps({"query": query, "topic": topic, "feed": feed, "limit": limit, "includeRaw": include_raw}).encode()
    req = urllib.request.Request(base + SL_PATH, data=body, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=12) as resp:
        return json.loads(resp.read())


def render(res: dict) -> str:
    sync = res.get("sync", {})
    lines = [f"Found {len(res.get('hits', []))} cited hit(s); ranking {res.get('mode', 'lexical')}; answered by {res.get('source')}.",
             f"Checked {sync.get('filesSeen', 0)} files in {sync.get('durationMs', 0)}ms.",
             f"Not indexed: {sync.get('skippedPrivate', 0)} private, {sync.get('skippedExcluded', 0)} excluded, "
             f"{sync.get('summarised', 0)} inside summarised folders, {sync.get('skippedOversizeJson', 0)} oversize JSON; "
             f"{sync.get('truncated', 0)} indexed files are cut short."]
    lines += [f"Degraded {d.get('leg')}: {d.get('detail')}" for d in res.get("degradations", [])]
    for h in res.get("hits", []):
        heading = f" — {h['heading']}" if h.get("heading") else ""
        cut = " [file cut short in the index]" if h.get("truncated") else ""
        lines.append(f"  {h['citation']}{heading} (score {h.get('score', 0):.3f}){cut}\n      {' '.join(h.get('snippet', '').split())}")
    if res.get("hits"):
        lines.append(f"Paths are under {Path(res['hits'][0]['absPath'][:-len(res['hits'][0]['path'])])}; read a hit with sed -n '<start>,<end>p'.")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("query")
    ap.add_argument("--topic", default="")
    ap.add_argument("--feed", default="")
    ap.add_argument("--limit", type=int, default=10)
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--include-raw", action="store_true", help="rank run output, support copies, preimages and history like curated pages")
    ap.add_argument("--files", action="store_true", help="skip Source Ledger and read the files")
    ap.add_argument("--home", help="research home (default $VROOLI_RESEARCH_HOME or ~/.vrooli/research)")
    args = ap.parse_args(argv)
    if not terms(args.query):
        print("find.py: the query has no searchable words", file=sys.stderr)
        return 2
    home = home_dir(args.home)
    res, why = None, "skipped (--files)"
    base = None if args.files or args.home else source_ledger_base()
    if base:
        try:
            res = search_ledger(base, args.query, args.topic, args.feed, args.limit, args.include_raw)
            res.setdefault("hits", [])
        except Exception as err:  # any failure falls back to files
            why = f"unavailable ({err})"
    elif not args.files:
        why = "not reachable" if not args.home else "skipped (--home)"
    if res is None:
        res = search_files(home, args.query, args.topic, args.feed, args.limit, args.include_raw)
        res["degradations"].insert(0, {"reason": "SOURCE_LEDGER_UNAVAILABLE", "leg": "source-ledger", "detail": f"Source Ledger {why}; searched the files directly"})
    print(json.dumps(res, indent=2) if args.json else render(res))
    return 0 if res["hits"] else 1


if __name__ == "__main__":
    sys.exit(main())
