"""agent-manager.friction-digest v1 — recurring friction fingerprints for one owning scenario.

Contract: friction-digest.json (inputs, invariants, bindings, outputs).
Skill:    agent-manager §4 (the [S3] leaf), agent-manager-improve §3 (the fleet friction sensor),
          and every other scenario's improve skill §3.

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Episodes carry no timestamp, so the window is applied to the run's createdAt; the digest
covers the most recent `run_limit` runs that fall inside `window_days`. Zero runs in the
window is an empty digest with status ok, not an error.
"""

import datetime

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
try:
    inputs
except NameError:
    inputs = {}
scenario = str(inputs.get("scenario", "")).strip()
window_days = int(inputs.get("window_days", 7))
run_limit = int(inputs.get("run_limit", 40))
top_n = int(inputs.get("top_n", 10))

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "agent-manager.friction-digest", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"scenario": scenario, "window_days": window_days, "run_limit": run_limit, "top_n": top_n},
    "signals": {
        "runs_listed": 0, "runs_in_window": 0, "runs_unparseable_timestamp": 0,
        "window_truncated_by_run_limit": False, "earliest_run_created_at": None,
        "episodes_total": 0, "episodes_for_scenario": 0, "episode_reads_failed": 0,
        "owner_confidence": {}, "recurring_count": 0, "top_fingerprints": [],
    },
    "errors": [], "evidence": [],
}
handles = {"run_ids": [], "episodes": []}


def fail(status, klass, detail, where):
    """The one place a bad path is recorded. Sets status, appends the error, routes to report."""
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def parse_ts(value):
    """RFC3339 with nanoseconds -> aware datetime; None when unparseable."""
    if not value:
        return None
    text = str(value).replace("Z", "+00:00")
    if "." in text:
        head, tail = text.split(".", 1)
        frac = tail[: tail.index("+")] if "+" in tail else tail
        zone = tail[len(frac):]
        text = f"{head}.{frac[:6].ljust(6, '0')}{zone}"
    try:
        return datetime.datetime.fromisoformat(text)
    except ValueError:
        return None


# ---- state machine ---------------------------------------------------------------
def step_validate():
    if not scenario:
        return fail("failed", "invalid_input", "scenario is required", "validate")
    if window_days < 1 or window_days > 90:
        return fail("failed", "invalid_input", f"window_days={window_days} outside [1, 90]", "validate")
    if run_limit < 1 or run_limit > 200:
        return fail("failed", "invalid_input", f"run_limit={run_limit} outside [1, 200]", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only; episodes fan out through gather
    envelope["phase"] = "collect"
    sig = envelope["signals"]
    try:
        runs = agent_manager.run.list(limit=run_limit)
        rows = runs.head(run_limit)  # bounded by run_limit <= 200: the ids are needed for the episode reads
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    sig["runs_listed"] = len(rows)
    cutoff = datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(days=window_days)
    stamped = [(r, parse_ts(r.get("createdAt"))) for r in rows]
    sig["runs_unparseable_timestamp"] = sum(1 for _, ts in stamped if ts is None)
    in_window = [r for r, ts in stamped if ts is not None and ts >= cutoff]  # an unparseable timestamp is never in-window
    sig["runs_in_window"] = len(in_window)
    # Every listed run is inside the window and the list stopped at run_limit (or the server says more
    # exist): older in-window runs were never read, so the digest is a lower bound on the window.
    more = bool(runs.meta().get("hasMore")) or len(rows) >= run_limit
    sig["window_truncated_by_run_limit"] = bool(in_window) and len(in_window) == len(rows) and more
    envelope["evidence"].append(f"agent-manager run list --limit {run_limit}")
    if in_window:
        sig["earliest_run_created_at"] = min(r.get("createdAt") for r in in_window)
    else:
        return "classify"  # zero runs in the window: an empty digest, status ok
    handles["run_ids"] = [r["id"] for r in in_window]

    def read(run_id):
        try:
            return agent_manager.run.episodes(run_id=run_id)
        except Exception as exc:  # one run's episodes failing degrades to partial, never to a lie
            return exc

    results = gather(*[lambda i=i: read(i) for i in handles["run_ids"]])
    for run_id, result in zip(handles["run_ids"], results):
        if isinstance(result, Exception):
            sig["episode_reads_failed"] += 1
            try:
                raise result
            except Exception as exc:  # classify_transport re-raises a missing kernel name; the driver labels it
                klass = classify_transport(exc)[1]
            envelope["errors"].append({"class": klass, "detail": str(result)[:160], "where": f"collect:episodes:{run_id}"})
        else:
            handles["episodes"].append((run_id, result))
    envelope["evidence"].append(f"agent-manager run episodes <id> x{len(handles['episodes'])}")
    return "classify"


def step_classify():  # CLASSIFY · deterministic; counts and groups in the kernel, bounded materialization
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    conf = {}
    groups = {}
    for run_id, handle in handles["episodes"]:
        sig["episodes_total"] += handle.count()
        # ownerConfidence may be omitted by protojson: map with .get, then group_by; nothing materialized
        for label, n in handle.map(lambda e: {"c": e.get("ownerConfidence") or "unknown"}).group_by("c").items():
            conf[label] = conf.get(label, 0) + int(n)
        # `unknown` episodes omit suspectedOwnerScenario, so filter with .get before reading it.
        mine = handle.filter(lambda r: r.get("suspectedOwnerScenario") == scenario)
        sig["episodes_for_scenario"] += mine.count()
        projected = mine.map(lambda r: {
            "fingerprint": r.get("fingerprint"), "pattern": r.get("pattern"), "cause_scope": r.get("causeScope"),
            "owner_command": r.get("suspectedOwnerCommand"), "owner_confidence": r.get("ownerConfidence"),
            "wall_clock_ms": int(r.get("wallClockMs") or 0),
        })
        for row in projected.head(500):  # bounded: at most 500 episode projections per run (budget.materialize_limit)
            group = groups.setdefault(row["fingerprint"], {
                "fingerprint": row["fingerprint"], "occurrences": 0, "runs": set(),
                "pattern": row["pattern"], "cause_scope": row["cause_scope"],
                "owner_command": row["owner_command"], "owner_confidence": row["owner_confidence"],
                "wall_clock_ms": 0,
            })
            group["occurrences"] += 1
            group["runs"].add(run_id)
            group["wall_clock_ms"] += row["wall_clock_ms"]
    sig["owner_confidence"] = conf
    ranked = sorted(groups.values(), key=lambda g: (-len(g["runs"]), -g["occurrences"]))
    sig["recurring_count"] = sum(1 for g in ranked if len(g["runs"]) >= 3)
    sig["top_fingerprints"] = []
    for g in ranked[:top_n]:
        entry = {k: v for k, v in g.items() if k != "runs"}
        entry["distinct_runs"] = len(g["runs"])
        sig["top_fingerprints"].append(entry)
    envelope["status"] = "partial" if sig["episode_reads_failed"] else "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:  # the one catch that guarantees an envelope on every path
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
