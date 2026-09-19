import json

"""web-search.findings-curate v1 — read the effectiveness ledger and propose curation moves. Report only.

Contract: findings-curate.json.
Skill:    web-search-improve §5 (routes: curation moves flag / supersede / gc).

Phases: validate -> collect -> act (dry-run gc, the one write-effect binding, never retiring anything) -> classify -> report. No write is performed: `findings gc`
is called with dry_run=True only, and every proposal is returned for a human or the
usage skill's tree to execute with the cited CLI verb.
"""

inputs = program.inputs()
limit = int(inputs.get("limit", 100))
include_disputed = bool(inputs.get("include_disputed", True))
decayed_below = float(inputs.get("decayed_below", 0.5))

envelope = {
    "program": "web-search.findings-curate", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"limit": limit, "include_disputed": include_disputed, "decayed_below": decayed_below},
    "signals": {
        "findings_read": 0, "surfaced": 0, "used": 0, "never_surfaced": 0, "never_surfaced_decayed": 0, "disputed_open": 0,
        "gc_dry_run": {"superseded_decayed": 0, "cold_archive_candidates": 0, "stale_disputes": 0, "orphans": 0},
        "proposals": [], "proposal_counts": {"gc": 0, "review-unused": 0, "resolve-dispute": 0, "supersede-stale": 0},
    },
    "errors": [], "evidence": [],
}
handles = {}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:160], "where": where})
    return "report"


def propose(kind, finding_id, verb, evidence):
    sig = envelope["signals"]
    if len(sig["proposals"]) < 12:
        sig["proposals"].append({"kind": kind, "finding_id": finding_id, "verb": verb[:60], "evidence": evidence[:100]})
    sig["proposal_counts"][kind] += 1
    envelope["evidence"].append(finding_id)


def step_validate():
    if not (1 <= limit <= 500):
        return fail("failed", "invalid_input", f"limit={limit} outside 1..500", "validate")
    if not (0.0 < decayed_below <= 1.0):
        return fail("failed", "invalid_input", f"decayed_below={decayed_below} outside (0, 1]", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only, concurrent
    envelope["phase"] = "collect"
    try:
        eff, disputes = gather(
            lambda: web_search.findings.effectiveness(limit=limit, include_disputed=include_disputed),
            lambda: web_search.disputes.list(limit=50),
        )
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    handles.update(eff=eff, disputes=disputes)
    return "act"


def step_act():  # ACT · the one write-effect binding, invoked in dry-run only (invariant: dry_run=True, nothing is retired)
    envelope["phase"] = "act"
    try:
        # rows must be one of the registry's camelCase candidates; supersededDecayed is the list this program proposes on
        handles["gc"] = web_search.findings.gc(dry_run=True, rows="supersededDecayed")
        handles["gc"].count()
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "act")
    return "classify"


def step_classify():  # CLASSIFY · a deterministic table; no inference
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    gc_meta = handles["gc"].meta() or {}
    gc_lists = {k: list(gc_meta.get(c) or []) for k, c in (
        ("cold_archive_candidates", "coldArchiveCandidates"),
        ("stale_disputes", "staleDisputes"), ("orphans", "orphans"))}
    gc_lists["superseded_decayed"] = list(handles["gc"].head(500))  # the selected rows field
    sig["gc_dry_run"] = {k: len(v) for k, v in gc_lists.items()}
    disputes = handles["disputes"].head(50)
    sig["disputed_open"] = handles["disputes"].count()
    disputed_ids = {d.get("id") for d in disputes if isinstance(d, dict)}

    rows = handles["eff"].head(limit)
    sig["findings_read"] = len(rows)
    for r in rows:
        f = r.get("finding", {}) or {}
        fid = f.get("id")
        surfaced = int(r.get("surfacedCount", 0) or 0)
        used = int(r.get("usedCount", 0) or 0)
        eff_conf = float(r.get("effectiveConfidence", 0.0) or 0.0)
        status = f.get("status")
        if surfaced > 0:
            sig["surfaced"] += 1
        if used > 0:
            sig["used"] += 1
        if surfaced == 0:
            sig["never_surfaced"] += 1
            if eff_conf < decayed_below:
                sig["never_surfaced_decayed"] += 1
                if fid in gc_lists["superseded_decayed"] or fid in gc_lists["cold_archive_candidates"]:
                    propose("gc", fid, "web-search findings gc", f"never surfaced; effective_confidence {eff_conf:.2f} < {decayed_below}; listed by gc --dry-run")
        elif used == 0 and eff_conf < decayed_below:
            propose("review-unused", fid, "web-search findings flag <id> --reason ... | findings supersede <id> --replacement <id> --reason ...",
                    f"surfaced {surfaced}x, never used; effective_confidence {eff_conf:.2f}")
        if status == "FINDING_STATUS_DISPUTED" or fid in disputed_ids:
            propose("resolve-dispute", fid, "web-search disputes resolve <id> --resolution keep|supersede [--replacement <id>] --reason ...",
                    "open dispute" + ("; stale per gc" if fid in gc_lists["stale_disputes"] else ""))
    for fid in gc_lists["stale_disputes"]:
        if fid not in disputed_ids and fid not in envelope["evidence"]:
            propose("resolve-dispute", fid, "web-search disputes resolve <id> --resolution keep|supersede --reason ...", "stale dispute per gc --dry-run")
    for fid in gc_lists["superseded_decayed"]:
        if fid not in envelope["evidence"]:
            propose("supersede-stale", fid, "web-search findings gc (soft-retires it) or findings prune --dry-run", "superseded and fully decayed per gc --dry-run")
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(json.dumps(envelope, sort_keys=True))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "act": step_act, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
