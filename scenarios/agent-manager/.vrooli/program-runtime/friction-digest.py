"""agent-manager.friction-digest v2 — the owner-level tool friction of one owning scenario.

Contract: friction-digest.json (inputs, invariants, bindings, outputs).
Skill:    agent-manager §4 (the [S3] leaf), agent-manager-improve §3 (the fleet friction sensor),
          and every other scenario's improve skill §3.

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
One tool-friction read narrowed to the scenario returns its findings (failures clustered by
error signature, degradations, shape guessing, heavy output, polling, bypasses of its CLI) and
its command profiles over the last `window_days`, with the coverage that says what was seen.
An empty digest is read beside that coverage, never as "no friction".
"""

import datetime
import json

# ---- inputs: the caller binds a dict named `inputs` before this source; contract defaults otherwise
inputs = program.inputs()
scenario = str(inputs.get("scenario", "")).strip()
window_days = int(inputs.get("window_days", 7))
top_n = int(inputs.get("top_n", 10))
policy_version = str(inputs.get("policy_version", "")).strip()
outcome_limit = int(inputs.get("outcome_limit", 100))

# ---- envelope: created first, printed once, on every path ----------------------
envelope = {
    "program": "agent-manager.friction-digest", "version": "2",
    "status": "failed", "phase": "validate",
    "inputs": {"scenario": scenario, "window_days": window_days, "top_n": top_n},
    "signals": {
        "window_start": None, "coverage": None, "profiles_for_scenario": 0, "calls_for_scenario": 0,
        "findings_for_scenario": 0, "recurring_count": 0, "top_fingerprints": [],
        "supervision_outcomes": None, "supervision_calibration_available": False, "supervision_false_positives": None,
        "supervision_false_negatives": None, "supervision_overrides": 0,
        "supervision_counterexamples": 0, "supervision_safety_violations": None,
        "supervision_completion_impact": None,
    },
    "errors": [], "evidence": [],
}
handles = {"tools": None}


def fail(status, klass, detail, where):
    """The one place a bad path is recorded. Sets status, appends the error, routes to report."""
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


# ---- state machine ---------------------------------------------------------------
def step_validate():
    if not scenario:
        return fail("failed", "invalid_input", "scenario is required", "validate")
    if window_days < 1 or window_days > 90:
        return fail("failed", "invalid_input", f"window_days={window_days} outside [1, 90]", "validate")
    if outcome_limit < 1 or outcome_limit > 500:
        return fail("failed", "invalid_input", f"outcome_limit={outcome_limit} outside [1, 500]", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads only
    envelope["phase"] = "collect"
    sig = envelope["signals"]
    try:
        outcomes = agent_manager.watch.policy_outcomes(policy_version=policy_version, limit=outcome_limit).head(outcome_limit)
        signal_classes = {"stalled", "blocked", "failed", "deadline", "quiet_time"}
        superseded = {r.get("supersedesOutcomeId") for r in outcomes if r.get("supersedesOutcomeId")}
        cutoff = datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(days=window_days)
        def in_window(row):
            try:
                return datetime.datetime.fromisoformat(str(row.get("createdAt", "")).replace("Z", "+00:00")) >= cutoff
            except (ValueError, TypeError):
                return False
        current = [r for r in outcomes if r.get("outcomeId") not in superseded and in_window(r)]
        assessed = [r for r in current if r.get("observedClass")]
        sig["supervision_outcomes"] = len(current)
        sig["supervision_assessed"] = len(assessed)
        sig["supervision_unassessed"] = len(current)-len(assessed)
        sig["supervision_sample_capped"] = len(outcomes)>=outcome_limit
        sig["supervision_calibration_available"] = False
        sig["supervision_sample_scope"] = "policy outcomes across scenarios; created-at window; bounded sample"
        sig["supervision_not_scenario_filtered"] = True
        sig["supervision_false_positives"] = sum(1 for r in assessed if r.get("predictedClass") in signal_classes and r.get("observedClass") not in signal_classes) if assessed else None
        sig["supervision_false_negatives"] = sum(1 for r in assessed if r.get("predictedClass") not in signal_classes and r.get("observedClass") in signal_classes) if assessed else None
        sig["supervision_overrides"] = sum(1 for r in current if r.get("overridden"))
        sig["supervision_counterexamples"] = sum(1 for r in current if r.get("counterexample"))
        sig["supervision_safety_violations"] = sum(1 for r in assessed if r.get("safetyViolation")) if assessed else None
        measured = [r for r in assessed if r.get("completionImpactObserved")]
        sig["supervision_completion_impact"] = sum(float(r.get("completionImpact", 0)) for r in measured)/len(measured) if measured else None
        sig["supervision_impact_samples"] = len(measured)
        sig["supervision_outcome_coverage"] = None  # decisions denominator is not supplied by this bounded read
        sig["supervision_coverage_reason"] = "pending_telemetry"
        envelope["evidence"].append(f"agent-manager watch policy-outcomes --limit {outcome_limit}")
    except Exception as exc:
        try:
            klass = program.classify(exc)[1]
        except Exception:
            klass = "kernel_runtime"
        envelope["errors"].append({"class": klass, "detail": str(exc)[:160], "where": "collect:supervision-outcomes"})
    now = datetime.datetime.now(datetime.timezone.utc)
    window = {"custom": {"from": (now - datetime.timedelta(days=window_days)).isoformat(), "to": now.isoformat()}}
    try:
        handles["tools"] = agent_manager.measures.tool_friction(window=window, owner=scenario, limit=200)
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    envelope["evidence"].append(f"agent-manager usage friction --since <{window_days} days ago>  # findings owned by {scenario}")
    return "classify"


def step_classify():  # CLASSIFY · deterministic; one record, bounded lists
    envelope["phase"] = "classify"
    sig = envelope["signals"]
    rows = handles["tools"].head(1)
    record = rows[0] if rows else {}
    summary = record.get("summary") or {}
    coverage = summary.get("coverage") or {}
    sig["window_start"] = record.get("windowStart")
    sig["coverage"] = coverage.get("sentence")
    profiles = record.get("profiles") or []
    sig["profiles_for_scenario"] = len(profiles)
    sig["calls_for_scenario"] = sum(int(p.get("calls") or 0) for p in profiles)
    findings = record.get("findings") or []  # ranked by their share of the window's harm, owner already narrowed
    sig["findings_for_scenario"] = len(findings)
    sig["recurring_count"] = sum(1 for f in findings if int(f.get("runs") or 0) >= 3)
    sig["top_fingerprints"] = [{
        "fingerprint": f.get("fingerprint"), "kind": f.get("kind"), "command_path": f.get("commandPath"),
        "description": f.get("description"), "calls": int(f.get("calls") or 0), "distinct_runs": int(f.get("runs") or 0),
        "cost": int(f.get("cost") or 0), "cost_unit": f.get("costUnit"), "cost_share": float(f.get("costShare") or 0.0),
        "status": f.get("status"),
    } for f in findings[:top_n]]
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    if envelope["errors"] and envelope["status"] == "ok":
        envelope["status"] = "partial"
    print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
