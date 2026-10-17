import json
inputs = program.inputs()

envelope = {"program": "visited-tracker.setpoint-read", "version": "2", "status": "failed", "phase": "validate", "inputs": {}, "signals": {}, "errors": [], "evidence": []}

def fail(status, kind, detail):
    envelope["status"] = status
    envelope["errors"].append({"class": kind, "detail": str(detail)[:240], "where": envelope["phase"]})
    return "report"

def meta_value(meta, snake, camel):
    return meta.get(snake, meta.get(camel))

def proto_count(meta, snake, camel, maximum):
    # ProtoJSON omits zero scalars and encodes uint64 as decimal strings.
    # A timestamped preview is required before omission can mean measured zero.
    if not isinstance(meta_value(meta, "observed_at", "observedAt"), str) or not meta_value(meta, "observed_at", "observedAt"):
        return None
    value = meta.get(snake, meta.get(camel, 0))
    if type(value) is str and len(value) <= 20 and value.isascii() and value.isdigit():
        value = int(value)
    return value if type(value) is int and 0 <= value <= maximum else None

def integrity_and_storage_rows(meta):
    rows = []
    proof = {}
    integrity = meta.get("integrity")
    violations = examined = None
    if isinstance(integrity, dict) and type(meta_value(integrity, "check_schema_version", "checkSchemaVersion")) is int and meta_value(integrity, "check_schema_version", "checkSchemaVersion") == 1:
        violations = proto_count(integrity, "violations", "violations", 4294967295)
        examined = proto_count(integrity, "examined_claims", "examinedClaims", 4294967295)
    valid = violations is not None and examined is not None and violations <= examined
    rows.append({"row":"claim-correctness", "reading":violations if valid else None, "target":0,
                 "in_band":violations == 0 if valid else None, "unavailable":not valid,
                 "reason":"" if valid else "unreliable:missing-or-invalid-integrity"})
    if valid:
        proof["integrity"] = {"scope":"current-active-claims", "examined":examined, "violations":violations}
    writes = meta_value(meta, "storage_writes", "storageWrites")
    reading, reason = None, "unreliable:missing-storage-observation"
    if isinstance(writes, dict):
        epoch = writes.get("epoch")
        total = proto_count(writes, "completed_total", "completedTotal", 18446744073709551615)
        failed = proto_count(writes, "failed_total", "failedTotal", 18446744073709551615)
        streak = proto_count(writes, "consecutive_failures", "consecutiveFailures", 18446744073709551615)
        age = proto_count(writes, "last_completed_age_seconds", "lastCompletedAgeSeconds", 18446744073709551615)
        if isinstance(epoch, str) and 0 < len(epoch) <= 64 and all(x is not None for x in (total, failed, streak, age)) and streak <= failed <= total:
            proof["storage"] = {"scope":"api-process", "epoch":epoch, "completed":total, "failed":failed, "consecutive_failures":streak, "sample_age_seconds":age}
            if total == 0:
                reason = "pending_no_write_samples"
            elif not isinstance(meta_value(writes, "last_completed_at", "lastCompletedAt"), str) or not meta_value(writes, "last_completed_at", "lastCompletedAt"):
                reason = "unreliable:missing-write-timestamp"
            elif meta_value(writes, "durable_sync_supported", "durableSyncSupported") is not True:
                reason = "pending_durability_support"
            elif age > 300:
                reason = "unreliable:stale-write-sample"
            else:
                reading, reason = streak, ""
    rows.append({"row":"durable-storage", "reading":reading, "target":0,
                 "in_band":reading == 0 if reading is not None else None, "unavailable":reading is None, "reason":reason})
    return rows, proof

def step_validate():
    campaign = inputs.get("campaign_id", "")
    if campaign is not None and (not isinstance(campaign, str) or len(campaign) > 36):
        return fail("failed", "invalid_input", "campaign_id must be a UUID when supplied")
    envelope["inputs"] = {"campaign_id": campaign}
    return "collect"

def step_collect():
    envelope["phase"] = "collect"
    rows = []
    try:
        handle = program_runtime.bindings.condition(scenario="visited-tracker", window_seconds=604800, rows="conditions")
        conditions = handle.head(20)
        bad = sum(1 for item in conditions if item.get("status") == "CONDITION_STATUS_DEGRADED")
        measured = bool(conditions) and handle.count() <= 20 and all(item.get("status") in ("CONDITION_STATUS_HEALTHY", "CONDITION_STATUS_DEGRADED", "CONDITION_STATUS_DORMANT") for item in conditions)
        rows.append({"row": "binding-condition", "reading": bad if measured else None, "target": 0, "in_band": bad == 0 if measured else None, "unavailable": not measured, "reason": "" if measured else "unreliable:unknown-empty-or-truncated"})
        envelope["evidence"].append({"binding": "program-runtime/bindings/condition", "conditions": [{"binding": str(item.get("bindingId", ""))[:200], "status": str(item.get("status", ""))[:80]} for item in conditions]})
        envelope["status"] = "ok" if measured else "partial"
    except Exception as exc:
        status, kind = program.classify(exc)
        fail("partial", kind, str(exc))
        rows.append({"row": "binding-condition", "reading": None, "target": 0, "in_band": None, "unavailable": True, "reason": "scenario_unreachable" if status == "unavailable" else "unreliable:" + kind})
    campaign = envelope["inputs"].get("campaign_id", "")
    telemetry = None
    if campaign:
        try:
            preview = visited_tracker.attention.preview(campaign_id=campaign, limit=1)
            meta = preview.meta()
            telemetry, proof = integrity_and_storage_rows(meta)
            envelope["evidence"].append({"binding":"visited-tracker/attention/preview", "observations":proof})
            eligible = proto_count(meta, "eligible_count", "eligibleCount", 4294967295)
            age = proto_count(meta, "oldest_eligible_age_seconds", "oldestEligibleAgeSeconds", 18446744073709551615)
            active = proto_count(meta, "active_claim_count", "activeClaimCount", 4294967295)
            if isinstance(eligible, int) and isinstance(age, int) and isinstance(active, int):
                rows.append({"row": "attention-fairness", "reading": age, "target": 86400, "in_band": age <= 86400, "unavailable": False, "reason": ""})
                envelope["evidence"].append({"binding": "visited-tracker/attention/preview", "fairness": {"eligible_count": eligible, "active_claim_count": active, "oldest_eligible_age_seconds": age}})
            else:
                rows.append({"row": "attention-fairness", "reading": None, "target": 86400, "in_band": None, "unavailable": True, "reason": "unreliable:missing-telemetry"})
        except Exception as exc:
            status, kind = program.classify(exc)
            fail("partial", kind, str(exc))
            rows.append({"row": "attention-fairness", "reading": None, "target": 86400, "in_band": None, "unavailable": True, "reason": "scenario_unreachable" if status == "unavailable" else "unreliable:" + kind})
    else:
        rows.append({"row": "attention-fairness", "reading": None, "target": 86400, "in_band": None, "unavailable": True, "reason": "pending_campaign_id"})
    if telemetry is not None:
        rows.extend(telemetry)
    else:
        for row in ("claim-correctness", "durable-storage"):
            rows.append({"row": row, "reading": None, "target": 0, "in_band": None, "unavailable": True, "reason": "unreliable:preview-unavailable" if campaign else "pending_campaign_id"})
    if envelope["status"] == "ok" and any(row["unavailable"] and (row["reason"].startswith("unreliable:") or row["reason"] == "scenario_unreachable") for row in rows):
        envelope["status"] = "partial"
    envelope["signals"] = {"rows": rows}
    return "report"

def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False))
    return None

STATES = {"validate": step_validate, "collect": step_collect, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:
        if envelope["phase"] == "report":
            raise
        state = fail("failed", "kernel_runtime", str(exc))
