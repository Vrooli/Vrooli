"""unit-health.setpoint-read v1 — read every unit-health-improve setpoint row in one submission.

Contract: setpoint-read.json (inputs, invariants, bindings, outputs).
Skill:    unit-health-improve §2 (the rows), §3 (the sensors).
Canon:    program-contracts.md §"Setpoint-read rows" (row shape, `reason` vocabulary, permanent rows).

Phases: validate -> collect -> classify -> report. Read-only. No inference, no delegation.
Governed reads (unit-health/validate/scenario, two unit-health/calibrate/run partitions, and bounded child programs) feed the rows
that have a sensor. Rows whose sensor has no governed binding, no sensor at all, or lives in
another program are declared with their permanent reason and never lower the status. Only a
failed or invalid read makes the board `partial`. Executed validation is declined by construction
(an executed run for an arbitrary target can exceed the invoke budget); pass `execution=true` to
accept that budget for one read.
"""
import json
import re

inputs = program.inputs()
target = inputs.get("target", "unit-health")
execution = inputs.get("execution", False)
reviewed_cohort_id = inputs.get("reviewed_cohort_id", "")
reviewed_source_identity = inputs.get("reviewed_source_identity", "")
reviewed_observation_count = inputs.get("reviewed_observation_count", 0)
mutation_package = inputs.get("mutation_package", "./internal/testquality/...")
mutation_seed = inputs.get("mutation_seed", "pilot-1")
mutation_floor = inputs.get("mutation_floor", 0.95)
roster = inputs.get("roster", ["unit-health"])
envelope = program.envelope("unit-health.setpoint-read", "1", inputs={"target": target, "execution": execution})
envelope["signals"] = {"rows": [], "readable": 0, "unavailable": 0}

BINDING = "unit-health/validate/scenario"
PERMANENT = ("no_governed_binding", "kernel_invoke_budget", "pending_telemetry")
ADVISORY = "QUALITY_ENFORCEMENT_ADVISORY"
UNKNOWN_CEILING = 0.05
board = {"response": {}, "calibration": {}, "holdout": {}, "mutation": {}, "fleet": {}, "friction": {}, "holdout_error": "", "validation_error": "", "mutation_error": "", "fleet_error": "", "friction_error": ""}


def field(container, snake, camel, default=None):
    """ProtoJSON rows are camelCase in-kernel; fixtures may use snake_case."""
    if not isinstance(container, dict):
        return default
    if snake in container:
        return container[snake]
    return container.get(camel, default)


def child_payload(handle):
    """Library contracts return their envelope as the first bounded Handle row."""
    if hasattr(handle, "head"):
        rows = handle.head(1)
        if rows and isinstance(rows[0], dict):
            return rows[0]
    raw = handle.raw() if hasattr(handle, "raw") else None
    return raw if isinstance(raw, dict) else {}


def row(name, reading, band, in_band, unavailable=False, reason=""):
    envelope["signals"]["rows"].append({
        "row": name,
        "reading": None if unavailable else reading,
        "target": band,
        "in_band": None if (unavailable or band is None) else bool(in_band),
        "unavailable": bool(unavailable),
        "reason": reason if unavailable else "",
    })
    envelope["signals"]["unavailable" if unavailable else "readable"] += 1
    # Contract: only a transient owner failure or a failed read lowers the board; an unreliable
    # sensor keeps in_band null and is reported as-is.
    if unavailable and reason in ("scenario_unreachable", "unreliable:validate-read-failed"):
        envelope["status"] = "partial"


def declined(name, band, reason):
    row(name, None, band, None, unavailable=True, reason=reason)


def step_validate():
    if not isinstance(target, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,127}", target):
        return program.fail("failed", "invalid_input", "target must be an exact scenario id", "validate")
    if type(execution) is not bool:
        return program.fail("failed", "invalid_input", "execution must be a boolean", "validate")
    if not isinstance(reviewed_cohort_id, str) or not isinstance(reviewed_source_identity, str) or type(reviewed_observation_count) is not int or reviewed_observation_count < 0 or not isinstance(mutation_package, str) or not isinstance(mutation_seed, str) or type(mutation_floor) not in (int, float) or mutation_floor < 0 or mutation_floor > 1 or not isinstance(roster, list) or not roster or len(roster) > 40 or any(not isinstance(item, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,127}", item) for item in roster):
        return program.fail("failed", "invalid_input", "reviewed cohort, mutation, and roster fields must be bounded", "validate")
    envelope["status"] = "ok"
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    try:
        # One governed read on its own worker so a hung owner ends as a classified result, not a stuck program.
        validation_args = {"scenario": target, "include_execution": execution, "rows": "findings"}
        if reviewed_cohort_id and reviewed_source_identity and reviewed_observation_count > 0:
            validation_args.update({"reviewed_cohort_id": reviewed_cohort_id, "reviewed_source_identity": reviewed_source_identity, "reviewed_observation_count": reviewed_observation_count})
        validation_read = program.guarded(lambda: unit_health.validate.scenario(**validation_args))
        validation_handle = validation_read()
        handle = validation_handle
        if isinstance(handle, Exception):
            raise handle
        raw = handle.raw() if hasattr(handle, "raw") else None
        board["response"] = raw if isinstance(raw, dict) else (handle.meta() if hasattr(handle, "meta") else {})
        response = board["response"]
        envelope["evidence"].append({
            "binding": BINDING,
            "target": target,
            "executed": execution,
            "run_id": str(field(response, "run_id", "runId", ""))[:96],
            "status": str(field(response, "status", "status", ""))[:32],
            "evidence_stages": field(response, "evidence_stages", "evidenceStages", {}),
        })
        calibration_read = program.guarded(lambda: unit_health.calibrate.run(partition="inventory", rows="cases"))
        calibration_handle = calibration_read()
        if isinstance(calibration_handle, Exception):
            status, klass = program.classify(calibration_handle)
            envelope["errors"].append({"class": klass, "detail": str(calibration_handle)[:160], "where": "collect:calibration"})
        else:
            raw_calibration = calibration_handle.raw() if hasattr(calibration_handle, "raw") else None
            board["calibration"] = raw_calibration if isinstance(raw_calibration, dict) else (calibration_handle.meta() if hasattr(calibration_handle, "meta") else {})
        holdout_read = program.guarded(lambda: unit_health.calibrate.run(partition="reviewed-holdout", holdout="assertion-observation-go-v1", rule="assertion-observation", rows="holdout"))
        holdout_handle = holdout_read()
        if isinstance(holdout_handle, Exception):
            status, klass = program.classify(holdout_handle)
            board["holdout_error"] = klass
            envelope["errors"].append({"class": klass, "detail": str(holdout_handle)[:160], "where": "collect:holdout"})
        else:
            raw_holdout = holdout_handle.raw() if hasattr(holdout_handle, "raw") else None
            board["holdout"] = raw_holdout if isinstance(raw_holdout, dict) else (holdout_handle.meta() if hasattr(holdout_handle, "meta") else {})
        if execution:
            mutation_read = program.guarded(lambda: unit_health.mutation.pilot(scenario=target, workspace="api", package=mutation_package, seed=mutation_seed, rows="receipts"))
            mutation_handle = mutation_read()
            if isinstance(mutation_handle, Exception):
                status, klass = program.classify(mutation_handle)
                board["mutation_error"] = klass
                envelope["errors"].append({"class": klass, "detail": str(mutation_handle)[:160], "where": "collect:mutation"})
            else:
                raw_mutation = mutation_handle.raw() if hasattr(mutation_handle, "raw") else None
                board["mutation"] = raw_mutation if isinstance(raw_mutation, dict) else (mutation_handle.meta() if hasattr(mutation_handle, "meta") else {})
        fleet = {"applicable": 0, "covered": 0, "uncovered": [], "child_failures": []}
        for scenario in roster:
            validation = program.guarded(lambda scenario=scenario: unit_health.validate.scenario(scenario=scenario, include_execution=False, rows="findings"))()
            if isinstance(validation, Exception):
                status, klass = program.classify(validation)
                fleet["child_failures"].append({"scenario": scenario, "class": klass, "where": "unit-health/validate/scenario"})
                envelope["status"] = "partial"
                continue
            validation_payload = validation.raw() if hasattr(validation, "raw") else {}
            workspaces = field(validation_payload, "workspaces", "workspaces", []) if isinstance(validation_payload, dict) else []
            if not isinstance(workspaces, list) or not workspaces:
                continue
            fleet["applicable"] += 1
            digest = program.guarded(lambda scenario=scenario: lib.test_genie.validation_digest(caller_scenario=scenario, limit=20))()
            if isinstance(digest, Exception):
                status, klass = program.classify(digest)
                fleet["child_failures"].append({"scenario": scenario, "class": klass, "where": "test-genie/validation-digest"})
                envelope["status"] = "partial"
                continue
            digest_payload = child_payload(digest)
            digest_status = str(field(digest_payload, "status", "status", "")) if isinstance(digest_payload, dict) else ""
            digest_signals = field(digest_payload, "signals", "signals", {})
            terminal = int(field(digest_signals, "terminal", "terminal", 0) or 0)
            with_evidence = int(field(digest_signals, "with_evidence", "withEvidence", 0) or 0)
            if digest_status != "ok":
                fleet["child_failures"].append({"scenario": scenario, "class": "child_status_" + (digest_status or "unknown"), "where": "test-genie/validation-digest"})
                envelope["status"] = "partial"
            elif terminal > 0 and with_evidence > 0:
                fleet["covered"] += 1
            else:
                fleet["uncovered"].append(scenario)
        board["fleet"] = fleet
        friction = program.guarded(lambda: lib.agent_manager.friction_digest(scenario="unit-health", window_days=7))()
        if isinstance(friction, Exception):
            status, klass = program.classify(friction)
            board["friction_error"] = klass
            envelope["status"] = "partial"
            envelope["errors"].append({"class": klass, "detail": str(friction)[:160], "where": "collect:agent-manager"})
        else:
            board["friction"] = child_payload(friction)
    except Exception as exc:
        status, klass = program.classify(exc)
        board["validation_error"] = klass
        program.fail("partial", klass, str(exc), "collect")
        board["response"] = {}
        board["calibration"] = {}
        board["holdout"] = {}
        board["holdout_error"] = ""
        envelope["evidence"].append({"binding": BINDING, "target": target, "failed": klass})
    return "classify"


def sensor_reason():
    if board["response"]:
        return ""
    if board.get("validation_error") == "scenario_unreachable":
        return "scenario_unreachable"
    return "unreliable:validate-read-failed"


def maturity_rows():
    response = board["response"]
    dead = sensor_reason()
    assessment = field(response, "assessment", "assessment", {})
    local = field(assessment, "local", "local", {})
    level = field(local, "current_level", "currentLevel")
    if dead or not isinstance(level, str) or not level:
        declined("self-maturity", "top rung, 0 blocking codes", dead or "unreliable:assessment-missing")
    else:
        blocking = field(local, "blocking_finding_codes", "blockingFindingCodes", []) or []
        next_level = field(local, "next_level", "nextLevel", "") or ""
        capabilities = {}
        for cap in field(assessment, "capabilities", "capabilities", []) or []:
            cap_id = field(cap, "id", "id")
            if isinstance(cap_id, str):
                capabilities[cap_id] = field(cap, "current_level", "currentLevel")
        reading = {"level": level, "next": next_level or None, "blocking": [str(c)[:64] for c in blocking][:16], "capabilities": capabilities}
        row("self-maturity", reading, "top rung, 0 blocking codes", not blocking and not next_level)
    if not execution:
        declined("self-execution", "executed=passed, 0 LOW_COVERAGE", "kernel_invoke_budget")
        return
    if dead:
        declined("self-execution", "executed=passed, 0 LOW_COVERAGE", dead)
        return
    stages = field(response, "evidence_stages", "evidenceStages", {})
    executed = str(field(stages, "executed", "executed", ""))
    findings = field(response, "findings", "findings", []) or []
    low = sum(1 for f in findings if isinstance(f, dict) and f.get("code") == "LOW_COVERAGE")
    if executed in ("", "not_requested"):
        declined("self-execution", "executed=passed, 0 LOW_COVERAGE", "unreliable:execution-not-observed")
    else:
        row("self-execution", {"executed": executed, "low_coverage": low}, "executed=passed, 0 LOW_COVERAGE", executed == "passed" and low == 0)


def quality_rows():
    response = board["response"]
    dead = sensor_reason()
    quality = field(response, "test_quality", "testQuality", {})
    results = field(quality, "results", "results", []) if isinstance(quality, dict) else []
    if dead or not isinstance(results, list) or not results:
        reason = dead or "unreliable:no-test-quality-results"
        declined("unknown-share", "<= 0.05 per rule", reason)
        declined("rules-promoted", None, reason)
        return
    per_rule = {}
    not_executed = 0
    for item in results:
        if not isinstance(item, dict):
            continue
        rule = str(field(item, "rule_id", "ruleId", "unknown"))[:64]
        status = str(item.get("status", ""))
        entry = per_rule.setdefault(rule, {"assessed": 0, "unknown": 0, "promoted": False})
        if status.endswith("NOT_APPLICABLE"):
            continue
        if str(item.get("reason", "")).endswith("_NOT_EXECUTED"):
            # A rule whose profile needs a native run was not assessed in a plan-only read.
            not_executed += 1
            continue
        entry["assessed"] += 1
        if status.endswith("_UNKNOWN"):
            entry["unknown"] += 1
        if str(item.get("enforcement", ADVISORY)) != ADVISORY:
            entry["promoted"] = True
    shares = {rule: (round(v["unknown"] / v["assessed"], 4) if v["assessed"] else None) for rule, v in per_rule.items()}
    worst = max((s for s in shares.values() if s is not None), default=None)
    if worst is None:
        declined("unknown-share", "<= 0.05 per rule", "unreliable:no-assessed-results")
    else:
        row("unknown-share", {"worst": worst, "per_rule": shares, "not_executed": not_executed}, "<= 0.05 per rule", worst <= UNKNOWN_CEILING)
    promoted = sorted(rule for rule, v in per_rule.items() if v["promoted"])
    row("rules-promoted", {"promoted": len(promoted), "rules_observed": len(per_rule), "promoted_rules": promoted}, None, None)


def evidence_rows():
    response = board["response"]
    dead = sensor_reason()
    stages = field(response, "evidence_stages", "evidenceStages", {})
    reviewed = str(field(stages, "reviewed", "reviewed", "")) if isinstance(stages, dict) else ""
    if dead or not reviewed:
        declined("reviewed-evidence", "supplied", dead or "unreliable:evidence-stages-missing")
    else:
        row("reviewed-evidence", reviewed, "supplied", reviewed == "supplied")
    trace = field(response, "traceability", "traceability", {})
    reason = str(field(trace, "unavailable_reason", "unavailableReason", "")) if isinstance(trace, dict) else ""
    if dead or not reason:
        declined("requirement-traceability", "owner reachable, 0 untagged", dead or "unreliable:traceability-missing")
    elif reason != "QUALITY_REASON_NONE":
        declined("requirement-traceability", "owner reachable, 0 untagged", "unreliable:" + reason.replace("QUALITY_REASON_", "").lower())
    else:
        quality = field(response, "test_quality", "testQuality", {})
        results = field(quality, "results", "results", []) if isinstance(quality, dict) else []
        untagged = sum(1 for r in results if isinstance(r, dict) and field(r, "rule_id", "ruleId") == "requirement-link" and str(r.get("status", "")).endswith("_VIOLATION"))
        row("requirement-traceability", {"owner": "reachable", "untagged": untagged}, "owner reachable, 0 untagged", untagged == 0)


def declared_rows():
    calibration = board["calibration"]
    missing = None
    if not calibration:
        declined("calibration-floor", "matched = implemented", "unreliable:calibration-read-failed")
        declined("corpus-implemented-share", "implemented + retired = specified", "unreliable:calibration-read-failed")
        missing = None
    else:
        corpus = field(calibration, "corpus", "corpus", {})
        cases = field(calibration, "cases", "cases", []) or []
        matched = sum(1 for case in cases if isinstance(case, dict) and bool(field(case, "matched", "matched", False)))
        implemented = int(field(corpus, "implemented", "implemented", 0) or 0)
        retired = int(field(corpus, "retired", "retired", 0) or 0)
        specified = int(field(corpus, "specified", "specified", 0) or 0)
        floor = str(field(corpus, "development_floor", "developmentFloor", ""))
        row("calibration-floor", {"matched": matched, "implemented": implemented, "floor": floor}, "matched = implemented", matched == implemented)
        row("corpus-implemented-share", {"implemented": implemented, "retired": retired, "specified": specified}, "implemented + retired = specified", implemented + retired == specified)
        missing = int(field(corpus, "spec_codes_without_emitter", "specCodesWithoutEmitter", 0) or 0)
    holdout = board["holdout"]
    holdout_rows = field(holdout, "holdout", "holdout", []) if isinstance(holdout, dict) else []
    if not holdout_rows or not isinstance(holdout_rows, list) or not isinstance(holdout_rows[0], dict):
        declined("holdout-agreement", "FP within the rule's declared budget on a reviewed holdout", "unreliable:holdout-read-failed" if board["holdout_error"] else "pending_telemetry")
    else:
        comparison = holdout_rows[0]
        labelled = int(field(comparison, "labelled", "labelled", 0) or 0)
        observed = int(field(comparison, "observed", "observed", 0) or 0)
        fp_rate = float(field(comparison, "fp_rate", "fpRate", 0) or 0)
        fn_rate = float(field(comparison, "fn_rate", "fnRate", 0) or 0)
        unknown = int(field(comparison, "unknown", "unknown", 0) or 0)
        budget = float(field(comparison, "budget", "budget", 0) or 0)
        within_budget = bool(field(comparison, "within_budget", "withinBudget", False))
        promotion_allowed = bool(field(comparison, "promotion_allowed", "promotionAllowed", False))
        reading = {"labelled": labelled, "observed": observed, "fp_rate": fp_rate, "fn_rate": fn_rate, "unknown": unknown, "budget": budget, "promotion_allowed": promotion_allowed}
        row("holdout-agreement", reading, "FP within the rule's declared budget on a reviewed holdout", labelled >= 30 and within_budget and not promotion_allowed)
    if missing is None:
        declined("declared-vs-emitted", "0 maturity codes without an emitter", "unreliable:calibration-read-failed")
    else:
        row("declared-vs-emitted", {"spec_codes_without_emitter": missing}, "0 maturity codes without an emitter", missing == 0)
    if not execution:
        declined("mutation-signal", "kill_rate >= 0.95 on valid mutants", "kernel_invoke_budget")
    elif not board["mutation"]:
        declined("mutation-signal", "kill_rate >= 0.95 on valid mutants", board["mutation_error"] or "unreliable:mutation-read-failed")
    else:
        summary = field(board["mutation"], "summary", "summary", {})
        killed = int(field(summary, "killed", "killed", 0) or 0)
        survived = int(field(summary, "survived", "survived", 0) or 0)
        invalid = int(field(summary, "invalid", "invalid", 0) or 0)
        kill_rate = float(field(summary, "kill_rate", "killRate", 0.0) or 0.0)
        reading = {"killed": killed, "survived": survived, "invalid": invalid, "kill_rate": kill_rate}
        row("mutation-signal", reading, "kill_rate >= %.2f on valid mutants" % float(mutation_floor), kill_rate >= float(mutation_floor))
    fleet = board["fleet"]
    if not fleet:
        declined("fleet-adoption", "uncovered = 0", board["fleet_error"] or "unreliable:roster-read-failed")
    else:
        fleet_reading = {"applicable": int(fleet.get("applicable", 0)), "covered": int(fleet.get("covered", 0)), "uncovered": list(fleet.get("uncovered", []))[:40], "child_failures": list(fleet.get("child_failures", []))[:40]}
        row("fleet-adoption", fleet_reading, "uncovered = 0", fleet_reading["applicable"] > 0 and not fleet_reading["uncovered"] and not fleet_reading["child_failures"])
    if not board["friction"]:
        declined("external-friction", "0 recurring fingerprints", "unreliable:" + (board["friction_error"] or "friction-read-failed"))
    else:
        friction_signals = field(board["friction"], "signals", "signals", {})
        recurring = int(field(friction_signals, "recurring_count", "recurringCount", 0) or 0)
        top = field(friction_signals, "top_fingerprints", "topFingerprints", []) or []
        row("external-friction", {"recurring_count": recurring, "top_fingerprints": top[:10]}, "0 recurring fingerprints", recurring == 0)


def step_classify():
    envelope["phase"] = "classify"
    maturity_rows()
    quality_rows()
    evidence_rows()
    declared_rows()
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
