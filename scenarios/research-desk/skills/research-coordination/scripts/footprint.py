#!/usr/bin/env python3
"""Observe what a research run has started and is holding, and report what is new.

Usage:
  footprint.py snapshot --out FILE [--force] [--watch DIR ...]
  footprint.py diff --baseline FILE [--keep NAME ...] [--keep-file FILE] [--disk] [--watch DIR ...] [--json]

snapshot records GPU memory per process key, running Vrooli resources, live capacity
claims, containers, the active Workspace Sandbox total, user systemd timers and free disk
(plus the size of each --watch directory). It refuses to overwrite an existing --out
file (exit 2) unless --force is given. diff takes a fresh observation and prints only what
is new or grown since the baseline, one line per item, ending with
`NEW SINCE BASELINE: n item(s)`. Run snapshot before work starts and diff before every
report and at the end; stop what is new unless it was meant to stay (--keep/--keep-file).
Watched directories are measured with du only when diff gets --disk.

The script only observes. Stop/release commands in the output are hints; it never runs them.
Exit code (diff): 0 nothing new and gpu and resources checked, 1 something new and not
kept, 2 usage error or unreadable baseline, 3 nothing new but gpu or resources could not be
checked (1 wins over 3). A section that could not be checked in either snapshot prints
`NOT CHECKED <section>: <error>` and is never counted as new.
"""

from __future__ import annotations

import argparse
import json
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath

DISK_FLAG_BYTES = 5 * 1024**3
GPU_GROWTH_MIB = 512
DEFAULT_TIMEOUT = 10
RESOURCE_TIMEOUT = 30  # `vrooli resource status` probes every resource and takes ~6 s idle
DU_TIMEOUT = 60
GIB = 1024**3

ALL_SECTIONS = ("gpu", "resources", "claims", "containers", "sandboxes", "timers", "disk")


def default_run(argv, timeout):
    """Run argv without a shell; return (returncode, stdout). Raises OSError or TimeoutExpired."""
    proc = subprocess.run(argv, capture_output=True, text=True, timeout=timeout, check=False)
    return proc.returncode, proc.stdout


class CollectError(Exception):
    pass


def _call(run, argv, timeout=DEFAULT_TIMEOUT):
    try:
        code, out = run(argv, timeout)
    except FileNotFoundError:
        raise CollectError(f"{argv[0]} not found")
    except subprocess.TimeoutExpired:
        raise CollectError(f"{argv[0]} timed out after {timeout}s")
    except Exception as exc:  # any runner failure must not stop the other collectors
        raise CollectError(f"{argv[0]} failed: {exc}")
    if code != 0:
        raise CollectError(f"{' '.join(argv[:3])} exited {code}")
    return out or ""


def _json(text, what):
    try:
        return json.loads(text)
    except ValueError:
        raise CollectError(f"{what} did not return JSON")


def _unavailable(exc):
    return {"available": False, "error": str(exc)}


def _int(token):
    """Integer from a possibly '[N/A]' or padded token; None when not a number."""
    token = (token or "").strip()
    try:
        return int(float(token))
    except ValueError:
        return None


# --- GPU -----------------------------------------------------------------------------

def gpu_key(name):
    """Stable identity of a GPU process: artefact directory under .vrooli/artifacts, else basename."""
    name = (name or "").strip()
    marker = ".vrooli/artifacts/"
    if marker in name:
        tail = name.split(marker, 1)[1].split("/", 1)[0]
        if tail:
            return tail
    base = PurePosixPath(name).name
    return base or name or "unknown"


def collect_gpu(run):
    mem = _call(run, ["nvidia-smi", "--query-gpu=memory.used,memory.total", "--format=csv,noheader,nounits"])
    used = total = 0
    seen = False
    for line in mem.splitlines():
        parts = [p.strip() for p in line.split(",")]
        if len(parts) < 2:
            continue
        u, t = _int(parts[0]), _int(parts[1])
        if u is None or t is None:
            continue
        used, total, seen = used + u, total + t, True
    if not seen:
        raise CollectError("nvidia-smi reported no usable GPU memory figures")
    apps = _call(run, ["nvidia-smi", "--query-compute-apps=pid,process_name,used_memory", "--format=csv,noheader,nounits"])
    processes = []
    for line in apps.splitlines():
        head, sep, tail = line.partition(",")
        if not sep or "," not in tail:
            continue
        name, _, mib = tail.rpartition(",")
        processes.append({"pid": _int(head), "name": name.strip(), "key": gpu_key(name), "mib": _int(mib)})
    keys = {}
    for proc in processes:
        keys[proc["key"]] = keys.get(proc["key"], 0) + (proc["mib"] or 0)
    return {"available": True, "used_mib": used, "total_mib": total, "processes": processes, "keys": keys}


# --- Resources, claims, containers, sandboxes, timers ----------------------------------

def collect_resources(run):
    out = _call(run, ["vrooli", "resource", "status", "--fast", "--health-only", "--json"], RESOURCE_TIMEOUT)
    data = _json(out, "vrooli resource status")
    rows = data.get("resources") if isinstance(data, dict) else data
    if not isinstance(rows, list):
        raise CollectError("vrooli resource status returned no resources list")
    items = []
    for row in rows:
        if not isinstance(row, dict):
            continue
        name = (row.get("resource") or {}).get("name") or row.get("name")
        if name:
            items.append({"name": name, "running": bool(row.get("running"))})
    return {"available": True, "items": items}


LIVE_CLAIM_STATES = ("reserved", "granted", "degraded", "active")


def collect_claims(run):
    out = _call(run, ["vrooli", "capacity", "list", "--active", "--json"])
    data = _json(out, "vrooli capacity list")
    rows = data.get("claims") if isinstance(data, dict) else data
    if not isinstance(rows, list):
        raise CollectError("vrooli capacity list returned no claims list")
    items = []
    for row in rows:
        if not isinstance(row, dict):
            continue
        state = str(row.get("status") or row.get("state") or "")
        if state and state not in LIVE_CLAIM_STATES:
            continue
        owner = row.get("owner_id") or row.get("owner") or row.get("name") or "unknown"
        kind = row.get("resource_kind") or row.get("kind") or "unknown"
        amount = row.get("amount_bytes", row.get("amount"))
        items.append({
            "id": row.get("claim_id") or row.get("id") or "",
            "owner": owner,
            "kind": kind,
            "key": f"{owner}/{kind}",
            "amount_bytes": amount,
            "state": state,
        })
    return {"available": True, "items": items}


def collect_containers(run):
    out = _call(run, ["docker", "ps", "--format", "{{.Names}}\t{{.Image}}\t{{.Status}}"])
    items = []
    for line in out.splitlines():
        parts = line.split("\t")
        if parts and parts[0].strip():
            items.append({
                "name": parts[0].strip(),
                "image": parts[1].strip() if len(parts) > 1 else "",
                "status": parts[2].strip() if len(parts) > 2 else "",
            })
    return {"available": True, "items": items}


def collect_sandboxes(run):
    # Only the total is kept: the CLI lists just the newest 100 display rows, so the
    # "Sandboxes found: N" summary line is the one trustworthy figure.
    out = _call(run, ["workspace-sandbox", "sandbox", "list", "--status", "active", "--json"])
    data = _json(out, "workspace-sandbox sandbox list")
    if not isinstance(data, dict):
        raise CollectError("workspace-sandbox sandbox list returned an unexpected shape")
    for line in data.get("summary") or []:
        if str(line).startswith("Sandboxes found:"):
            total = _int(str(line).split(":", 1)[1])
            if total is not None:
                return {"available": True, "total_active": total}
    raise CollectError("workspace-sandbox sandbox list gave no 'Sandboxes found' total")


def collect_timers(run):
    out = _call(run, ["systemctl", "--user", "list-timers", "--all", "--no-legend", "--no-pager"])
    items = []
    for line in out.splitlines():
        unit = next((tok for tok in line.split() if tok.endswith(".timer")), None)
        if unit:
            items.append({"unit": unit})
    return {"available": True, "items": items}


def collect_disk(run, watch):
    try:
        usage = shutil.disk_usage(Path.home())
    except OSError as exc:
        return _unavailable(f"disk_usage failed: {exc}")
    watched = {}
    for directory in watch:
        try:
            out = _call(run, ["du", "-sb", directory], DU_TIMEOUT)
            size = _int(out.split()[0]) if out.split() else None
            watched[directory] = {"bytes": size} if size is not None else _unavailable("du returned no size")
        except CollectError as exc:
            watched[directory] = _unavailable(exc)
    return {"available": True, "home_free_bytes": usage.free, "home_total_bytes": usage.total, "watched": watched}


def collect(run=default_run, watch=()):
    """One observation. Every collector is isolated: a failure marks only its own section."""
    def guarded(fn, *args):
        try:
            return fn(run, *args)
        except CollectError as exc:
            return _unavailable(exc)
        except Exception as exc:  # defensive: a parser bug must not lose the other sections
            return _unavailable(f"unexpected error: {exc}")

    return {
        "taken_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "gpu": guarded(collect_gpu),
        "resources": guarded(collect_resources),
        "claims": guarded(collect_claims),
        "containers": guarded(collect_containers),
        "sandboxes": guarded(collect_sandboxes),
        "timers": guarded(collect_timers),
        "disk": guarded(collect_disk, list(watch)),
    }


# --- Diff ----------------------------------------------------------------------------

def _ok(snap, section):
    value = snap.get(section)
    return isinstance(value, dict) and value.get("available") is True


def _kept(keep, *names):
    return any(n and k and str(n).startswith(k) for k in keep for n in names)


def _gpu_by_key(gpu):
    """Total MiB per process key; processes sharing a key (same basename) add up."""
    held = {}
    for proc in gpu.get("processes") or []:
        entry = held.setdefault(proc["key"], {"mib": 0, "pid": proc.get("pid")})
        entry["mib"] += proc.get("mib") or 0
    return held


def _mark(item, keep, *names):
    item["kept"] = _kept(keep, *names)
    return item


def _error_of(base, now, section):
    for label, snap in (("", now), ("baseline: ", base)):
        if not _ok(snap, section):
            value = snap.get(section)
            return label + str((value.get("error") if isinstance(value, dict) else None) or "section missing")
    return ""


def compute_diff(base, now, keep=(), measure_disk=False):
    """Structured difference between two snapshots; the text and JSON outputs both render it."""
    result = {"baseline_taken_at": base.get("taken_at"), "now_taken_at": now.get("taken_at"), "not_checked": []}
    for section in ALL_SECTIONS:
        if not (_ok(base, section) and _ok(now, section)):
            result["not_checked"].append({"section": section, "error": _error_of(base, now, section)})
    unchecked = {n["section"] for n in result["not_checked"]}

    gpu = {"available": "gpu" not in unchecked, "new": [], "grown": []}
    if gpu["available"]:
        b, n = base["gpu"], now["gpu"]
        gpu.update(used_mib=n["used_mib"], free_mib=n["total_mib"] - n["used_mib"],
                   baseline_used_mib=b["used_mib"], baseline_free_mib=b["total_mib"] - b["used_mib"])
        before, after = _gpu_by_key(b), _gpu_by_key(n)
        running_now = {i["name"] for i in (now["resources"].get("items") or []) if i.get("running")} if _ok(now, "resources") else set()
        for key, cur in after.items():
            if key not in before:
                gpu["new"].append(_mark({"key": key, "mib": cur["mib"], "pid": cur["pid"], "is_resource": key in running_now}, keep, key))
            elif cur["mib"] - before[key]["mib"] > GPU_GROWTH_MIB:
                gpu["grown"].append(_mark({"key": key, "mib": cur["mib"], "baseline_mib": before[key]["mib"]}, keep, key))
    result["gpu"] = gpu

    if "resources" in unchecked:
        result["resources"] = {"available": False, "new": []}
    else:
        was = {i["name"] for i in base["resources"].get("items") or [] if i.get("running")}
        result["resources"] = {"available": True, "new": [
            _mark({"name": i["name"]}, keep, i["name"])
            for i in now["resources"].get("items") or [] if i.get("running") and i["name"] not in was]}

    def new_items(section, ident, extra=()):
        entry = {"available": section not in unchecked, "new": []}
        if entry["available"]:
            known = {ident(i) for i in base[section].get("items") or []}
            for item in now[section].get("items") or []:
                if ident(item) not in known:
                    entry["new"].append(_mark(dict(item), keep, ident(item), *(item.get(f) for f in extra)))
        result[section] = entry

    new_items("claims", lambda i: i["key"], extra=("id", "owner"))
    new_items("containers", lambda i: i["name"])
    new_items("timers", lambda i: i["unit"])

    # Sandboxes are only counted for information: the CLI lists the newest 100 and other
    # agents create sandboxes constantly, so neither items nor a NEW count would be honest.
    sandboxes = {"available": "sandboxes" not in unchecked}
    if sandboxes["available"]:
        now_total, base_total = now["sandboxes"].get("total_active"), base["sandboxes"].get("total_active")
        sandboxes.update(total_active=now_total, baseline_total_active=base_total,
                         change=(now_total - base_total) if isinstance(now_total, int) and isinstance(base_total, int) else None)
    result["sandboxes"] = sandboxes

    disk = {"available": "disk" not in unchecked, "measured_watched": measure_disk, "watched": []}
    if disk["available"]:
        b, n = base["disk"], now["disk"]
        disk.update(free_bytes=n["home_free_bytes"], baseline_free_bytes=b["home_free_bytes"],
                    change_bytes=n["home_free_bytes"] - b["home_free_bytes"])
        for directory, cur in (n.get("watched") or {}).items():
            prev = (b.get("watched") or {}).get(directory) or {}
            row = {"dir": directory, "bytes": cur.get("bytes"), "baseline_bytes": prev.get("bytes")}
            if row["bytes"] is None or row["baseline_bytes"] is None:
                row["growth_bytes"], row["flagged"] = None, False
            else:
                row["growth_bytes"] = row["bytes"] - row["baseline_bytes"]
                row["flagged"] = row["growth_bytes"] > DISK_FLAG_BYTES
            disk["watched"].append(row)
    result["disk"] = disk

    result["new_count"] = sum(1 for i in (*gpu["new"], *gpu["grown"], *result["resources"]["new"], *result["claims"]["new"],
                                          *result["containers"]["new"], *result["timers"]["new"]) if not i.get("kept"))
    result["exit_code"] = exit_code_for(result)
    return result


def exit_code_for(diff):
    if diff["new_count"]:
        return 1
    if any(n["section"] in ("gpu", "resources") for n in diff["not_checked"]):
        return 3
    return 0


def _gib(n):
    return f"{n / GIB:.1f} GiB"


def _signed_gib(n):
    return f"{'+' if n >= 0 else '-'}{abs(n) / GIB:.2f} GiB"


GREW_HINT = ("a model was loaded into a service that was already running; unload it or restart "
             "that service when no job is using it")


def render_text(diff):
    lines = []
    gpu = diff["gpu"]
    if gpu["available"]:
        lines.append(f"GPU: used {gpu['used_mib']} MiB, free {gpu['free_mib']} MiB now; "
                     f"baseline used {gpu['baseline_used_mib']} MiB, free {gpu['baseline_free_mib']} MiB")
        for item in gpu["new"]:
            hint = (f"stop: vrooli resource stop {item['key']}" if item["is_resource"]
                    else f"look: nvidia-smi --query-compute-apps=pid,process_name,used_memory --format=csv (pid {item['pid']})")
            lines.append(f"NEW gpu {item['key']} {item['mib']} MiB{_kept_note(item)}; {hint}")
        for item in gpu["grown"]:
            lines.append(f"GREW gpu {item['key']} {item['baseline_mib']} -> {item['mib']} MiB{_kept_note(item)}; {GREW_HINT}")
    hints = {
        "resources": lambda i: (i["name"], "", f"stop: vrooli resource stop {i['name']}"),
        "claims": lambda i: (i["id"] or i["owner"], f"owner={i['owner']} kind={i['kind']} {_claim_amount(i)} {i['state']}", f"release: vrooli capacity release {i['id']}"),
        "containers": lambda i: (i["name"], f"{i['image']} {i['status']}", f"stop: docker stop {i['name']}"),
        "timers": lambda i: (i["unit"], "", f"stop: systemctl --user stop {i['unit']}"),
    }
    labels = {"resources": "resource", "claims": "claim", "containers": "container", "timers": "timer"}
    for section in ("resources", "claims", "containers", "timers"):
        for item in diff[section]["new"]:
            name, detail, hint = hints[section](item)
            lines.append(f"NEW {labels[section]} {name}{' ' + detail if detail else ''}{_kept_note(item)}; {hint}")
    sb = diff["sandboxes"]
    if sb["available"]:
        change = sb["change"]
        lines.append(f"sandboxes: {sb['total_active']} active now, {sb['baseline_total_active']} at baseline"
                     + (f", {change:+d}" if change is not None else ""))
    disk = diff["disk"]
    if disk["available"]:
        lines.append(f"DISK free {_gib(disk['free_bytes'])} now, {_signed_gib(disk['change_bytes'])} since baseline ({_gib(disk['baseline_free_bytes'])})")
        for row in disk["watched"]:
            if row["growth_bytes"] is None:
                lines.append(f"DISK watch {row['dir']}: size unavailable or not in baseline")
            else:
                flag = f"; FLAG over {_gib(DISK_FLAG_BYTES)}, find what to delete: du -h --max-depth=2 {row['dir']}" if row["flagged"] else ""
                lines.append(f"DISK watch {row['dir']}: {_gib(row['bytes'])} now, {_signed_gib(row['growth_bytes'])} since baseline{flag}")
        if not disk["measured_watched"]:
            lines.append("DISK watched directories not measured (pass --disk)")
    for entry in diff["not_checked"]:
        lines.append(f"NOT CHECKED {entry['section']}: {entry['error']}")
    tail = f"NEW SINCE BASELINE: {diff['new_count']} item(s)"
    if diff["not_checked"]:
        tail += f"; {len(diff['not_checked'])} section(s) not checked"
    lines.append(tail)
    return "\n".join(lines)


def _kept_note(item):
    return " (kept)" if item.get("kept") else ""


def _claim_amount(item):
    amount = item.get("amount_bytes")
    return _gib(amount) if isinstance(amount, (int, float)) else "?"


def read_keep_file(path):
    """Names from a keep file: one per line, blanks and `#` comments ignored; missing file is empty."""
    try:
        text = Path(path).read_text()
    except OSError:
        return []
    names = (line.strip() for line in text.splitlines())
    return [n for n in names if n and not n.startswith("#")]


# --- CLI -----------------------------------------------------------------------------

DIFF_EPILOG = """\
exit codes:
  0  nothing new, and gpu and resources were both checked
  1  something new and not kept (wins over 3)
  2  usage error or unreadable baseline
  3  nothing new, but the gpu or resources section could not be checked
item lines start with a fixed shape whose name can be copied into a keep file:
  NEW gpu <key> | GREW gpu <key> | NEW resource <name> | NEW claim <id> | NEW container <name> | NEW timer <unit>
GREW means a key's total GPU memory grew by more than 512 MiB. Sandboxes are only counted, never listed or flagged.
"""


def build_parser():
    parser = argparse.ArgumentParser(prog="footprint.py", description=__doc__.split("\n\n")[0])
    sub = parser.add_subparsers(dest="command", required=True)
    snap = sub.add_parser("snapshot", help="record what is running and using space now")
    snap.add_argument("--out", required=True, help="snapshot file to write; refused (exit 2) if it already exists")
    snap.add_argument("--force", action="store_true", help="replace an existing --out file")
    snap.add_argument("--watch", action="append", default=[], metavar="DIR", help="directory whose size to record (repeatable)")
    diff = sub.add_parser("diff", help="print what is new since a snapshot", epilog=DIFF_EPILOG,
                          formatter_class=argparse.RawDescriptionHelpFormatter)
    diff.add_argument("--baseline", required=True, help="snapshot file written by `snapshot`")
    diff.add_argument("--keep", action="append", default=[], metavar="NAME", help="item key or name (exact or prefix) meant to stay (repeatable)")
    diff.add_argument("--keep-file", metavar="FILE", help="file of names to keep, one per line; blank lines and # comments ignored; a missing file is empty")
    diff.add_argument("--disk", action="store_true", help="also measure watched directories with du (slow, can take a minute); without it only free disk is shown")
    diff.add_argument("--watch", action="append", default=[], metavar="DIR", help="extra directory to size with --disk (repeatable); the baseline's watched directories are always included")
    diff.add_argument("--json", action="store_true", help="print one JSON object instead of text")
    return parser


def main(argv=None, run=default_run, out=None, err=None):
    out = out or sys.stdout
    err = err or sys.stderr
    try:
        args = build_parser().parse_args(argv)
    except SystemExit as exc:
        return exc.code if isinstance(exc.code, int) else 2

    if args.command == "snapshot":
        path = Path(args.out)
        if path.exists() and not args.force:
            print(f"baseline already exists: {args.out} (not overwritten; pass --force to replace it)", file=err)
            return 2
        snap = collect(run, args.watch)
        try:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(json.dumps(snap, indent=2) + "\n")
        except OSError as exc:
            print(f"cannot write snapshot: {exc}", file=err)
            return 2
        gone = [s for s in ALL_SECTIONS if not _ok(snap, s)]
        note = f"; not available: {', '.join(gone)}" if gone else ""
        print(f"snapshot written to {args.out} at {snap['taken_at']}{note}", file=out)
        return 0

    try:
        base = json.loads(Path(args.baseline).read_text())
        if not isinstance(base, dict) or "taken_at" not in base:
            raise ValueError("not a footprint snapshot")
    except (OSError, ValueError) as exc:
        print(f"unreadable baseline {args.baseline}: {exc}", file=err)
        return 2
    watch = list(dict.fromkeys([*((base.get("disk") or {}).get("watched") or {}), *args.watch])) if args.disk else []
    keep = [*args.keep, *(read_keep_file(args.keep_file) if args.keep_file else [])]
    diff = compute_diff(base, collect(run, watch), keep, measure_disk=args.disk)
    print(json.dumps(diff, indent=2) if args.json else render_text(diff), file=out)
    return diff["exit_code"]


if __name__ == "__main__":
    sys.exit(main())
