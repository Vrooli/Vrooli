"""scenario-dependency-analyzer.advisory-lockfile-refresh v1 — refresh one surface's lockfile for named advisories, scenario by scenario.

Contract: advisory-lockfile-refresh.json.

For each scenario: preview `deps update --advisory <ids>`, and with apply=true publish it and
re-preview to prove the lockfile now holds the newest version its declared ranges allow. A
package counts as fixed unless the analyzer names it in an `exposed:` note, which it does for every
locked version still inside an advisory Security Health reports. Lockfile
only (by default node_modules is not relinked either, so nothing is downloaded): no manifest range changes, nothing is restarted, and a scenario whose range admits no
fix is reported, never forced.

Phases: validate -> act (preview, apply, re-preview; sequential, one resolution at a time)
-> classify -> report.
"""

import json
import re

inputs = program.inputs()
raw_scenarios = str(inputs.get("scenarios", "") or "")
raw_advisories = str(inputs.get("advisories", "") or "all")
raw_parents = str(inputs.get("parents", "") or "")
surface = str(inputs.get("surface", "ui") or "ui")
apply_changes = inputs.get("apply", False) is True
lock_only = inputs.get("lock_only", True) is not False

envelope = {
    "program": "scenario-dependency-analyzer.advisory-lockfile-refresh", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"scenarios": raw_scenarios[:300], "advisories": raw_advisories[:300], "parents": raw_parents, "surface": surface, "apply": apply_changes, "lock_only": lock_only},
    "signals": {"mode": "apply" if apply_changes else "preview", "counts": {}, "rows": [], "attention": []},
    "errors": [], "evidence": [],
}
REPORT_OUTPUT_BUDGET_BYTES = 3800  # leave headroom below Program Runtime's 4096-byte stdout cap
MAX_SCENARIOS = 12
MAX_ADVISORIES = 8
NAME = re.compile(r"^[a-z0-9][a-z0-9-]*$")
SURFACE = re.compile(r"^[a-z0-9][a-z0-9/_-]*$")
ADVISORY = re.compile(r"^(all|GHSA(-[a-z0-9]{4}){3}|CVE-[0-9]{4}-[0-9]+)$", re.IGNORECASE)
EXPOSED = re.compile(r"^exposed: (?P<package>\S+) to ")
LOCKED_AT = re.compile(r"^(?P<name>\S+): locked at (?P<versions>.+?) \(unchanged\)$")
SELECTS = re.compile(r"^advisory .+ selects (?P<names>.+)$")
MOVED = re.compile(r"^(?P<name>\S+): (?P<from>\S.*?) -> (?P<to>\S.*)$")

fail = program.fail
cfg = {"scenarios": [], "advisories": [], "parents": []}
results = []  # one {"scenario", "status", "detail"} per scenario, in input order


def csv(text):
    # "library run" reads commas in --input as pair separators, so lists also split on ; and spaces.
    return [part.strip() for part in re.split(r"[,;\s]+", text) if part.strip()]


def exposed(notes):
    """['name@1.2.3', ...] the analyzer says still sit inside an advisory after the change."""
    return [m.group("package") for m in (EXPOSED.match(n) for n in notes) if m]


def short(text, limit=110):
    text = str(text).replace("\n", " ")
    return text if len(text) <= limit else text[: limit - 1] + "…"


def call(apply_flag, scenario, names=None):
    """One deps update call: by advisory (or "all"), or by explicit package names once a parent has to move with them."""
    if names:
        handle = scenario_dependency_analyzer.deps.update(
            scenario=scenario, surface=surface, package=["npm/" + n for n in names], apply=apply_flag, lock_only=lock_only, rows="notes")
    else:
        handle = scenario_dependency_analyzer.deps.update(
            scenario=scenario, surface=surface, advisory=",".join(cfg["advisories"]), apply=apply_flag, lock_only=lock_only, rows="notes")
    meta = handle.meta()
    return {"verdict": str(meta.get("verdict", "")), "blocked": bool(meta.get("blocked", False)),
            "updated": bool(meta.get("updated", False)), "source_updated": bool(meta.get("sourceUpdated", False)),
            "message": str(meta.get("message", "")), "notes": [str(n) for n in meta.get("notes", [])]}


def step_validate():
    scenarios, advisories = csv(raw_scenarios), csv(raw_advisories)
    if not scenarios or len(scenarios) > MAX_SCENARIOS or not all(NAME.match(s) for s in scenarios):
        return fail("failed", "invalid_input", f"scenarios must be 1..{MAX_SCENARIOS} scenario names", "validate")
    if not advisories or len(advisories) > MAX_ADVISORIES or not all(ADVISORY.match(a) for a in advisories):
        return fail("failed", "invalid_input", f"advisories must be all, or 1..{MAX_ADVISORIES} GHSA or CVE ids", "validate")
    if not SURFACE.match(surface):
        return fail("failed", "invalid_input", f"surface={surface!r} is not a surface name such as ui", "validate")
    if "all" in [a.lower() for a in advisories] and len(advisories) > 1:
        return fail("failed", "invalid_input", "advisories is either all or a list of ids, not both", "validate")
    parents = csv(raw_parents)
    if len(parents) > 8 or not all(re.fullmatch(r"(@[a-z0-9._-]+/)?[a-z0-9._-]+", p) for p in parents):
        return fail("failed", "invalid_input", "parents must be up to 8 npm package names", "validate")
    if not isinstance(inputs.get("apply", False), bool) or not isinstance(inputs.get("lock_only", True), bool):
        return fail("failed", "invalid_input", "apply and lock_only must be true or false", "validate")
    cfg.update(scenarios=list(dict.fromkeys(scenarios)), advisories=advisories, parents=parents)
    return "act"


def selected_names(notes):
    for note in notes:
        match = SELECTS.match(note)
        if match:
            return [n.strip() for n in match.group("names").split(",") if n.strip()]
    return []


def plan(scenario):
    """The preview to act on: by advisory, then, when a package would stay exposed because a parent
    may pin it, by the same names plus the declared parents (express 4.21 pins body-parser 1.20.3)."""
    preview = call(False, scenario)
    names = None
    if preview["verdict"] in ("current", "updatable") and cfg["parents"]:
        if exposed(preview["notes"]):
            selected = selected_names(preview["notes"])
            # A parent this surface does not lock is simply not part of the move: ask the owner
            # about each declared parent alone, then take the ones it can update together.
            usable = []
            for parent in cfg["parents"]:
                if parent in selected:
                    continue
                try:
                    call(False, scenario, selected + [parent])
                    usable.append(parent)
                except Exception:
                    pass
            if usable:
                try:
                    wider = call(False, scenario, selected + usable)
                except Exception:
                    wider = None  # the advisory-only result stands
                if wider is not None and wider["verdict"] == "updatable":
                    preview, names = wider, selected + usable
    return preview, names


def leftover(names):
    """What stays exposed, for a row: the range admits no fix for these."""
    return short(f"{len(names)} exposed, range admits no fix: " + ", ".join(names), 110)


def refresh_one(scenario):
    """ACT for one scenario: preview; with apply, publish then re-preview to prove convergence."""
    preview, names = plan(scenario)
    verdict = preview["verdict"]
    if preview["blocked"] or verdict in ("update_blocked", "update_failed"):
        return "blocked", short(preview["message"])
    if verdict == "not_affected":
        return "not_exposed", ""
    left = exposed(preview["notes"])
    if verdict == "current":
        if not left:
            return "already_fixed", ""
        return "no_fix_in_range", leftover(left)
    moves = "; ".join(n for n in preview["notes"] if MOVED.match(n) and "closure" not in n.split(":")[0])
    if not apply_changes:
        return "would_update", short(moves, 70) + (" | " + leftover(left) if left else "")
    applied = call(True, scenario, names)
    if applied["blocked"] or not (applied["updated"] or applied["source_updated"]):
        return "apply_failed", short(applied["message"])
    check = call(False, scenario, names)
    if check["blocked"] or check["verdict"] not in ("current", "not_affected"):
        return "not_converged", short(f"re-preview says {check['verdict'] or 'nothing'} after the apply")
    still = exposed(check["notes"])
    if still:
        return "partly_fixed", leftover(still)
    return "refreshed", short(moves, 100)


def step_act():  # ACT · sequential on purpose: each apply holds a disk admission and one staged resolution
    envelope["phase"] = "act"
    for scenario in cfg["scenarios"]:
        try:
            status, detail = refresh_one(scenario)
        except Exception as exc:
            klass_status, klass = program.classify(exc)
            status, detail = "error", f"{klass}: {short(exc, 90)}"
            envelope["errors"].append({"class": klass, "detail": short(exc, 160), "where": scenario})
            if klass_status == "unavailable":
                results.append({"scenario": scenario, "status": status, "detail": detail})
                return fail("unavailable", klass, f"scenario-dependency-analyzer unreachable at {scenario}; remaining scenarios were not attempted", "act")
        results.append({"scenario": scenario, "status": status, "detail": detail})
    return "classify"


def step_classify():
    envelope["phase"] = "classify"
    counts = {}
    for row in results:
        counts[row["status"]] = counts.get(row["status"], 0) + 1
    envelope["signals"]["counts"] = counts
    good = {"refreshed", "would_update", "already_fixed", "not_exposed"}
    envelope["signals"]["attention"] = [r["scenario"] for r in results if r["status"] not in good]
    envelope["status"] = "ok" if not envelope["signals"]["attention"] or all(r["status"] in good | {"no_fix_in_range"} for r in results) else "partial"
    return "report"


def step_report():
    envelope["phase"] = "report"
    rows = [[r["scenario"], r["status"], r["detail"]] for r in results]
    envelope["signals"]["rows"] = rows
    if len(json.dumps(envelope, allow_nan=False)) > REPORT_OUTPUT_BUDGET_BYTES:
        # Keep every non-trivial row; the quiet ones collapse to their counts.
        quiet = {"not_exposed", "already_fixed"}
        envelope["signals"]["rows"] = [r if r[1] not in quiet else [r[0], r[1], ""] for r in rows]
        envelope["signals"]["rows_compacted"] = True
    if len(json.dumps(envelope, allow_nan=False)) > REPORT_OUTPUT_BUDGET_BYTES:
        envelope["signals"]["rows"] = [r[:2] for r in rows if r[1] not in {"not_exposed", "already_fixed"}][:25]
        envelope["signals"]["rows_compacted"] = True
    program.report()
    return None


STATES = {"validate": step_validate, "act": step_act, "classify": step_classify, "report": step_report}
program.attach(envelope)
program.run(STATES, "validate")
