#!/usr/bin/env python3
"""Which tools the research topics depend on are down right now, and what to do instead.

  topic_health.py [--home H] [--topic T] [--friction] [--json]

Checks the agent credential path once (Agent Manager's broker), then reads each topic's `TOOLS.md` (`## <capability>` sections with `- **Monitoring scenario(s):**`
slugs and `- **Fallback:**`); a topic without one depends on the common set below. Asks
`vrooli scenario status <slug> --json` for each slug once. A tool is `up` when running and
healthy, `down` when the control plane says it is not, and `unknown` when the question itself
fails: an unreachable control plane is never read as healthy or as down. Every tool that is not
up is listed with the topics it affects and their fallbacks. `--friction` (with `--topic`) adds
Agent Manager friction through `program-runtime.research-tools-review`; when that fails the
friction is `unknown`.

A script rather than a Program Runtime program on purpose: it must answer when Program Runtime,
the control plane or Agent Manager is down. Read-only. Standard library; run with python3 -I.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

sys.dont_write_bytecode = True

COMMON = [  # the templates' common TOOLS.md
    {"capability": "Search", "scenarios": ["search-hub"], "fallback": "`rg` over the topic folder and the repository."},
    {"capability": "Skills and programs", "scenarios": ["prompt-manager", "program-runtime"],
     "fallback": "read `SKILL.md` by path; run a program's underlying commands."},
    {"capability": "Decisions", "scenarios": ["notification-hub", "research-desk"],
     "fallback": "the item stays in `STATE.md` and `ATTENTION.md`; the supervisor delivers it later."},
]
CREDENTIALS = {  # every agent-side CLI call needs a credential from Agent Manager's identity broker
    "scenario": "agent-credentials",
    "fallback": "agent CLIs refuse while it fails: read and write the research files directly, run "
                "binding-free programs with Program Runtime's standalone runner, keep decisions as "
                "needs-you items in STATE.md, use native sub-agents.",
}
MONITOR = re.compile(r"^\s*- \*\*Monitoring scenarios?:\*\*\s*(.+)$")
FIELD = re.compile(r"^\s*- \*\*([^*:]+):\*\*\s*(.*)$")
SLUG = re.compile(r"`([a-z0-9][a-z0-9-]{0,62})`")
MAX_SLUGS = 32


def parse_tools(text: str) -> list[dict]:
    caps, cur, field = [], None, None
    for line in text.splitlines():
        if line.startswith("## "):
            cur = {"capability": line[3:].strip(), "scenarios": [], "fallback": ""}
            caps.append(cur)
            field = None
            continue
        if cur is None:
            continue
        m = MONITOR.match(line)
        if m:
            cur["scenarios"] += [s for s in SLUG.findall(m.group(1)) if s not in cur["scenarios"]]
            field = None
            continue
        m = FIELD.match(line)
        if m:
            field = m.group(1).strip().lower()
            if field == "fallback":
                cur["fallback"] = m.group(2).strip()
            continue
        if field == "fallback" and line.startswith("  ") and line.strip():
            cur["fallback"] += " " + line.strip()
        elif not line.strip():
            field = None
    return [c for c in caps if c["scenarios"]]


def live(mission: Path) -> bool:
    """A topic folder (not an alias) whose mission is not retired."""
    try:
        return not re.search(r"^Retired:", mission.read_text(encoding="utf-8"), re.MULTILINE)
    except OSError:
        return False


def inventories(home: Path, topic: str | None) -> dict[str, dict]:
    root = home / "topics"
    names = [topic] if topic else sorted(p.name for p in root.iterdir() if not p.is_symlink() and live(p / "MISSION.md"))
    out = {}
    for name in names:
        path = root / name / "TOOLS.md"
        try:
            caps = parse_tools(path.read_text(encoding="utf-8"))
            out[name] = {"inventory": "own" if caps else "own-unparsed", "capabilities": caps or COMMON}
        except FileNotFoundError:
            out[name] = {"inventory": "common", "capabilities": COMMON}
        except OSError as exc:
            out[name] = {"inventory": f"unreadable: {exc}", "capabilities": COMMON}
    return out


def run_json(cmd: list[str], timeout: int) -> tuple[dict | None, str]:
    try:
        p = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    except FileNotFoundError:
        return None, f"{cmd[0]} is not installed"
    except subprocess.TimeoutExpired:
        return None, f"no answer in {timeout}s"
    text = p.stdout.strip()
    start = text.find("{")
    try:
        return json.loads(text[start:]) if start >= 0 else None, (p.stderr.strip() or text)[:200]
    except json.JSONDecodeError:
        return None, f"unreadable answer: {(text or p.stderr.strip())[:160]}"


def probe(slug: str, vrooli: str, timeout: int) -> dict:
    data, why = run_json([vrooli, "scenario", "status", slug, "--json"], timeout)
    if not isinstance(data, dict):
        return {"state": "unknown", "reason": why or "no answer"}
    if not data.get("success", True) and data.get("code") == "scenario_not_found":
        return {"state": "down", "reason": "no such scenario"}
    sc = data.get("scenario")
    if not isinstance(sc, dict):
        return {"state": "unknown", "reason": str(data.get("error") or why)[:200]}
    status, health = sc.get("status", ""), sc.get("health_status", "")
    if status == "running" and health == "healthy":
        return {"state": "up", "reason": ""}
    if status and status != "running":
        return {"state": "down", "reason": f"status {status}"}
    if health in ("unhealthy", "degraded"):
        return {"state": "down", "reason": f"health {health}: {sc.get('health_error', '')}"[:200]}
    return {"state": "unknown", "reason": f"status {status or '?'}, health {health or '?'}"}


def probe_credentials(desk: str, timeout: int) -> dict:
    """A control plane can call every service healthy while agents cannot use any of them. One
    cheap credentialed read tells: the broker refusing is `down`, any other failure `unknown`."""
    try:
        p = subprocess.run([desk, "commissions", "--json"], capture_output=True, text=True, timeout=timeout)
    except FileNotFoundError:
        return {"state": "unknown", "reason": f"{desk} is not installed"}
    except subprocess.TimeoutExpired:
        return {"state": "unknown", "reason": f"no answer in {timeout}s"}
    if p.returncode == 0:
        return {"state": "up", "reason": ""}
    err = (p.stderr or p.stdout).strip()
    line = next((l.strip() for l in err.splitlines() if re.search(r"credential|identity broker", l, re.I)), "")
    return {"state": "down", "reason": line[:200]} if line else {"state": "unknown", "reason": err.splitlines()[-1][:200] if err else f"exit {p.returncode}"}


def friction(home: Path, topic: str, runtime: str, timeout: int) -> dict:
    cmd = [runtime, "program-runtime.research-tools-review", "--json", "--provenance", "agent", "--input",
           f"workspace_root={home},inventory_relative_path=topics/{topic}/TOOLS.md,window_days=7"]
    data, why = run_json(cmd, timeout)
    try:
        env = json.loads(data["program"]["stdout"])
    except (TypeError, KeyError, json.JSONDecodeError):
        return {"status": "unknown", "reason": why or "no envelope", "rows": []}
    rows = [{"scenario": r.get("scenario"), "status": r.get("status"), "recurring": r.get("recurring_count"),
             "top": [f"{f.get('description') or f.get('kind')} [{f.get('fingerprint')}]" if isinstance(f, dict) else str(f)
                     for f in (r.get("top_fingerprints") or [])[:3]]}
            for r in env.get("signals", {}).get("friction", [])]
    return {"status": env.get("status", "unknown"), "reason": "; ".join(e.get("detail", "") for e in env.get("errors", []))[:300], "rows": rows}


def assess(home: Path, topic: str | None, vrooli: str, timeout: int, desk: str = "research-desk") -> dict:
    inv = inventories(home, topic)
    users: dict[str, list[tuple[str, str, str]]] = {}
    for name, entry in inv.items():
        for cap in entry["capabilities"]:
            for slug in cap["scenarios"]:
                users.setdefault(slug, []).append((name, cap["capability"], cap["fallback"]))
    slugs = sorted(users)[:MAX_SLUGS]
    with ThreadPoolExecutor(max_workers=8) as pool:
        states = dict(zip(slugs, pool.map(lambda s: probe(s, vrooli, timeout), slugs)))
    tools = []
    for slug in slugs:
        fallbacks: dict[str, list[str]] = {}
        for name, cap, fb in users[slug]:
            fallbacks.setdefault(f"{cap}: {fb or 'none recorded; add one to TOOLS.md'}", []).append(name)
        tools.append({"scenario": slug, **states[slug], "topics": sorted({u[0] for u in users[slug]}),
                      "fallbacks": [{"fallback": k, "topics": v} for k, v in fallbacks.items()]})
    tools.insert(0, {"scenario": CREDENTIALS["scenario"], **probe_credentials(desk, timeout), "topics": sorted(inv),
                     "fallbacks": [{"fallback": f"Agent credentials: {CREDENTIALS['fallback']}", "topics": sorted(inv)}]})
    counts = {s: sum(1 for t in tools if t["state"] == s) for s in ("up", "down", "unknown")}
    return {"home": str(home), "topics": len(inv), "own_inventory": sorted(n for n, e in inv.items() if e["inventory"] == "own"),
            "counts": counts, "omitted_scenarios": max(0, len(users) - MAX_SLUGS), "tools": tools}


def render(r: dict) -> str:
    c = r["counts"]
    lines = [f"# Topic tool health — {r['topics']} topics, {len(r['tools'])} tools: {c['up']} up, {c['down']} down, {c['unknown']} unknown", "",
             f"Topics with their own TOOLS.md: {', '.join(r['own_inventory']) or 'none'}; the rest use the common set.", ""]
    bad = [t for t in r["tools"] if t["state"] != "up"]
    if not bad:
        lines.append("Every monitored tool is up.")
    for t in bad:
        lines.append(f"## {t['scenario']} — {t['state']}" + (f" ({t['reason']})" if t["reason"] else ""))
        for fb in t["fallbacks"]:
            who = fb["topics"]
            lines.append(f"- {fb['fallback']} — {len(who)} topic(s): {', '.join(who[:6])}{' …' if len(who) > 6 else ''}")
        lines.append("")
    if r.get("omitted_scenarios"):
        lines.append(f"{r['omitted_scenarios']} more monitored scenarios were not checked (bound {MAX_SLUGS}).")
    if "friction" in r:
        f = r["friction"]
        lines += ["", f"## Friction ({r['topic']}, 7 days) — {f['status']}" + (f": {f['reason']}" if f["reason"] else "")]
        for row in f["rows"]:
            lines.append(f"- **{row['scenario']}** ({row['status']}, {row['recurring']} recurring)")
            lines += [f"  - {t}" for t in row["top"]]
    return "\n".join(lines).rstrip() + "\n"


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--home", type=Path, default=Path.home() / ".vrooli" / "research")
    ap.add_argument("--topic")
    ap.add_argument("--friction", action="store_true", help="add Agent Manager friction (needs --topic)")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--timeout", type=int, default=20)
    ap.add_argument("--vrooli", default="vrooli", help=argparse.SUPPRESS)
    ap.add_argument("--program-runtime", default="program-runtime", help=argparse.SUPPRESS)
    ap.add_argument("--research-desk", default="research-desk", help=argparse.SUPPRESS)
    a = ap.parse_args(argv)
    if a.friction and not a.topic:
        ap.error("--friction needs --topic")
    if a.topic and not (a.home / "topics" / a.topic).is_dir():
        ap.error(f"no topic folder {a.home / 'topics' / a.topic}")
    report = assess(a.home, a.topic, a.vrooli, a.timeout, a.research_desk)
    if a.friction:
        report["topic"] = a.topic
        report["friction"] = friction(a.home, a.topic, a.program_runtime, max(a.timeout, 150))
    print(json.dumps(report, indent=2) if a.json else render(report), end="" if not a.json else "\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
