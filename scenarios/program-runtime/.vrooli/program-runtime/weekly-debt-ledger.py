"""program-runtime.weekly-debt-ledger v1: owner-backed seven-day debt snapshots."""

import datetime
import json
import re


inputs = program.inputs()
envelope = {
    "program": "program-runtime.weekly-debt-ledger",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {},
    "signals": {
        "window": {},
        "previous_window": None,
        "line_trend": {"status": "unknown", "rows": []},
        "source_trend": {"status": "unknown", "rows": [], "approach_rows": []},
        "detector_snapshot": {"status": "unknown", "rows": []},
        "lifecycle_inventory": {"status": "unknown", "reason": "no retained lifecycle artifact supplied"},
        "template_debt": {"status": "unknown", "rows": None},
        "reduction_receipts": {},
        "approach_yield": [],
        "snapshot_write": {"status": "not_requested"},
        "source_coverage": {},
    },
    "errors": [],
    "evidence": [],
}

RECEIPT_MARKER = "DR6_REDUCTION_RECEIPT:"
SNAPSHOT_MARKER = "DR6_WEEKLY_SNAPSHOT:"
SNAPSHOT_SCOPE = "debt-ledger-snapshots"
SOURCE_NAMES = ("research", "ecosystem-hunt", "delivery")
APPROACHES = ("state-machine-extraction", "owner-merge", "generator-replacement", "migration-completion", "dead-path-deletion")


def fail(kind, detail, where):
    envelope["errors"].append({"class": kind, "detail": str(detail)[:240], "where": where})


def error_status(exc, where):
    status, klass = program.classify(exc)
    fail(klass, exc, where)
    return status


def as_int(value, default):
    try:
        if isinstance(value, bool):
            return default
        result = int(value)
        return result
    except (TypeError, ValueError):
        return default


def valid_day(raw, label):
    if not isinstance(raw, str):
        raise ValueError(label + " must be YYYY-MM-DD")
    parsed = datetime.date.fromisoformat(raw)
    if parsed.isoformat() != raw:
        raise ValueError(label + " must be YYYY-MM-DD")
    return parsed


def validate_inputs():
    start = valid_day(inputs.get("window_start", ""), "window_start")
    end = valid_day(inputs.get("window_end", ""), "window_end")
    if (end - start).days != 7:
        raise ValueError("window_end must be exactly seven days after window_start")
    max_scenarios = as_int(inputs.get("max_scenarios", 160), 160)
    max_journal_rows = as_int(inputs.get("max_journal_rows", 100), 100)
    if not 1 <= max_scenarios <= 200:
        raise ValueError("max_scenarios must be within [1, 200]")
    if not 1 <= max_journal_rows <= 500:
        raise ValueError("max_journal_rows must be within [1, 500]")
    detector_snapshot = inputs.get("detector_snapshot", {})
    if not isinstance(detector_snapshot, dict):
        raise ValueError("detector_snapshot must be an object")
    lifecycle_snapshot = inputs.get("lifecycle_snapshot", {})
    if not isinstance(lifecycle_snapshot, dict):
        raise ValueError("lifecycle_snapshot must be an object")
    publish_snapshot = inputs.get("publish_snapshot", True)
    if not isinstance(publish_snapshot, bool):
        raise ValueError("publish_snapshot must be a boolean")
    envelope["inputs"] = {
        "window_start": start.isoformat(),
        "window_end": end.isoformat(),
        "max_scenarios": max_scenarios,
        "max_journal_rows": max_journal_rows,
        "publish_snapshot": publish_snapshot,
    }
    envelope["signals"]["window"] = {
        "start_inclusive": start.isoformat(),
        "end_exclusive": end.isoformat(),
        "days": 7,
        "timezone": "UTC",
    }
    return start, end, max_scenarios, max_journal_rows, detector_snapshot, lifecycle_snapshot, publish_snapshot


def validated_lifecycle_snapshot(snapshot, start, end):
    if not snapshot:
        return {"status": "unknown", "candidate_count": None, "persisted_count": None, "scenario_count": None, "reason": "no retained lifecycle artifact supplied"}
    observed_at = snapshot.get("observed_at")
    artifact = snapshot.get("artifact")
    digest = snapshot.get("sha256")
    if not isinstance(observed_at, str) or len(observed_at) < 10:
        raise ValueError("lifecycle_snapshot.observed_at is required")
    day = valid_day(observed_at[:10], "lifecycle_snapshot.observed_at")
    if day < start or day >= end:
        raise ValueError("lifecycle snapshot must be observed inside the requested week")
    if not isinstance(artifact, str) or not artifact or artifact.startswith("/") or ".." in artifact.split("/"):
        raise ValueError("lifecycle_snapshot.artifact must be a relative evidence path")
    if not isinstance(digest, str) or re.fullmatch(r"[0-9a-f]{64}", digest) is None:
        raise ValueError("lifecycle_snapshot.sha256 must be a lowercase SHA-256")
    counts = {}
    for key in ("scenario_count", "candidate_count", "persisted_count"):
        value = snapshot.get(key)
        if isinstance(value, bool) or not isinstance(value, int) or value < 0:
            raise ValueError("lifecycle_snapshot." + key + " must be a nonnegative integer")
        counts[key] = value
    if counts["persisted_count"] > counts["candidate_count"]:
        raise ValueError("persisted_count cannot exceed candidate_count")
    return {"status": "supplied_saved", "observed_at": observed_at, "artifact": artifact, "sha256": digest, **counts}


def selected_scope_records(rows):
    selected = []
    for row in rows:
        scope_id = str(row.get("id", row.get("scope", "")) or "").strip()
        if scope_id == "debt-reduction":
            selected.append((scope_id, "research"))
        elif scope_id == "ecosystem-hunt":
            selected.append((scope_id, "ecosystem-hunt"))
        elif scope_id.startswith("team:") and "delivery" in scope_id.lower():
            selected.append((scope_id, "delivery"))
    return selected


def extract_receipt(body):
    if not isinstance(body, str):
        return None, "body_missing"
    for line in body.splitlines():
        value = line.strip()
        if not value.startswith(RECEIPT_MARKER):
            continue
        try:
            receipt = json.loads(value[len(RECEIPT_MARKER):].strip())
        except (TypeError, ValueError):
            return None, "receipt_json_invalid"
        if not isinstance(receipt, dict):
            return None, "receipt_not_object"
        required = ("id", "source", "approach", "scenario", "before_revision", "after_revision")
        if any(not isinstance(receipt.get(key), str) or not receipt[key].strip() for key in required[:6]):
            return None, "receipt_identity_incomplete"
        if receipt.get("source") not in SOURCE_NAMES:
            return None, "receipt_source_invalid"
        if receipt.get("approach") not in APPROACHES:
            return None, "receipt_approach_invalid"
        if receipt.get("boundary_complete") is not True or not isinstance(receipt.get("change_boundary"), list):
            return None, "receipt_boundary_incomplete"
        if receipt.get("source") == "ecosystem-hunt" and not receipt.get("hunt_round"):
            return None, "receipt_hunt_round_missing"
        if receipt.get("source") == "delivery" and not receipt.get("team_scope"):
            return None, "receipt_delivery_scope_missing"
        if not isinstance(receipt.get("validation_refs"), list) or not receipt["validation_refs"]:
            return None, "receipt_validation_missing"
        boundary = {}
        language = {}
        paths = set()
        for changed in receipt["change_boundary"]:
            if not isinstance(changed, dict):
                return None, "receipt_boundary_entry_invalid"
            path = changed.get("path")
            lang = changed.get("language")
            category = changed.get("category")
            if not isinstance(path, str) or not path or "\\" in path or path.startswith("/") or ".." in path.split("/") or path in paths:
                return None, "receipt_boundary_path_invalid"
            if not isinstance(lang, str) or not lang or category not in ("production", "tests", "docs", "config", "generated", "resources", "other"):
                return None, "receipt_boundary_classification_invalid"
            paths.add(path)
            added = changed.get("added")
            deleted = changed.get("deleted")
            if isinstance(added, bool) or not isinstance(added, int) or added < 0 or isinstance(deleted, bool) or not isinstance(deleted, int) or deleted < 0:
                return None, "receipt_line_count_invalid"
            bucket = boundary.setdefault(category, {"added": 0, "deleted": 0})
            bucket["added"] += added
            bucket["deleted"] += deleted
            if category == "production":
                language_bucket = language.setdefault(lang, {"production_added": 0, "production_deleted": 0})
                language_bucket["production_added"] += added
                language_bucket["production_deleted"] += deleted
        production = boundary.get("production", {"added": 0, "deleted": 0})
        receipt["production_added"] = production["added"]
        receipt["production_deleted"] = production["deleted"]
        receipt["net_reduction_lines"] = production["deleted"] - production["added"]
        receipt["boundary_by_category"] = boundary
        receipt["production_by_language"] = language
        return receipt, "ok"
    return None, "receipt_missing"


def aggregate_receipts(entries_by_scope, start, end):
    results = {name: {"status": "unknown", "scopes": 0, "journal_rows": 0, "valid_receipts": 0, "unstructured_rows": 0, "invalid_receipts": 0, "net_reduction_lines": None, "production_added": 0, "production_deleted": 0, "by_category": {}, "by_language": {}} for name in SOURCE_NAMES}
    seen = set()
    yields = {}
    for scope_id, source, entries, saturated in entries_by_scope:
        result = results[source]
        result["scopes"] += 1
        if saturated:
            result["status"] = "partial"
        for row in entries:
            created = str(row.get("createdAt", row.get("created_at", "")) or "")
            if len(created) < 10:
                continue
            try:
                day = valid_day(created[:10], "journal created_at")
            except (TypeError, ValueError):
                result["status"] = "partial"
                continue
            if day < start or day >= end:
                continue
            result["journal_rows"] += 1
            receipt, reason = extract_receipt(row.get("body", ""))
            if receipt is None:
                if reason == "receipt_missing":
                    result["unstructured_rows"] += 1
                else:
                    result["invalid_receipts"] += 1
                continue
            expected = source
            if receipt.get("source") != expected:
                result["invalid_receipts"] += 1
                result["status"] = "partial"
                continue
            if source == "delivery" and receipt.get("team_scope") != scope_id:
                result["invalid_receipts"] += 1
                result["status"] = "partial"
                continue
            identity = receipt["id"]
            if identity in seen:
                continue
            seen.add(identity)
            result["valid_receipts"] += 1
            result["net_reduction_lines"] = (result["net_reduction_lines"] or 0) + receipt["net_reduction_lines"]
            result["production_added"] += receipt["production_added"]
            result["production_deleted"] += receipt["production_deleted"]
            for category, values in receipt["boundary_by_category"].items():
                bucket = result["by_category"].setdefault(category, {"added": 0, "deleted": 0})
                bucket["added"] += values["added"]
                bucket["deleted"] += values["deleted"]
            for language, values in receipt["production_by_language"].items():
                bucket = result["by_language"].setdefault(language, {"production_added": 0, "production_deleted": 0})
                bucket["production_added"] += values["production_added"]
                bucket["production_deleted"] += values["production_deleted"]
            key = (source, receipt["approach"])
            group = yields.setdefault(key, {"source": source, "approach": receipt["approach"], "landed_units": 0, "positive_reductions": 0, "production_deleted": 0, "production_added": 0, "net_reduction_lines": 0, "boundary_by_category": {}, "production_by_language": {}, "unit_ids": []})
            group["landed_units"] += 1
            group["positive_reductions"] += int(receipt["net_reduction_lines"] > 0)
            group["production_deleted"] += receipt["production_deleted"]
            group["production_added"] += receipt["production_added"]
            group["net_reduction_lines"] += receipt["net_reduction_lines"]
            for category, values in receipt["boundary_by_category"].items():
                bucket = group["boundary_by_category"].setdefault(category, {"added": 0, "deleted": 0})
                bucket["added"] += values["added"]
                bucket["deleted"] += values["deleted"]
            for language, values in receipt["production_by_language"].items():
                bucket = group["production_by_language"].setdefault(language, {"production_added": 0, "production_deleted": 0})
                bucket["production_added"] += values["production_added"]
                bucket["production_deleted"] += values["production_deleted"]
            group["unit_ids"].append(identity)
            if len(envelope["evidence"]) < 100:
                envelope["evidence"].append({"source": source, "scope": scope_id, "journal_id": row.get("id", ""), "receipt_id": identity})
    for source, result in results.items():
        if result["scopes"] and result["status"] != "partial":
            result["status"] = "measured" if result["valid_receipts"] and not result["unstructured_rows"] and not result["invalid_receipts"] else "partial" if result["journal_rows"] else "unknown"
    for row in yields.values():
        row["mean_net_reduction_per_landed_unit"] = row["net_reduction_lines"] / row["landed_units"] if row["landed_units"] else None
    complete_sources = {source for source, result in results.items() if result["status"] == "measured"}
    publishable_yields = [row for row in yields.values() if row["source"] in complete_sources]
    return results, sorted(publishable_yields, key=lambda row: (row["source"], row["approach"]))


def guarded(call):
    def run():
        try:
            return call()
        except Exception as exc:
            return exc
    return run


def collect_scenario_rows(max_scenarios):
    handle = flow_verifier.scenarios.list()
    rows = handle.head(max_scenarios + 1)
    # ListScenarios is a multi-list response with no primary row field in the
    # generic binding catalog, so the bridge wraps the typed response as one row.
    if len(rows) == 1 and isinstance(rows[0], dict) and isinstance(rows[0].get("scenarios"), list):
        response = rows[0]
        rows = response["scenarios"]
        count = len(rows)
        if response.get("success") is False or response.get("discoveryFailures"):
            return rows[:max_scenarios], count, True
    else:
        count = handle.count()
    truncated = count > max_scenarios or len(rows) > max_scenarios
    rows = rows[:max_scenarios]
    return rows, count, truncated


def scenario_metric_call(scenario):
    return code_facts.facts.describe(
        target={"kind": "TARGET_KIND_SCENARIO", "scenario": scenario},
        include=["FACT_FAMILY_FILE_METRICS"],
        rows="facts",
    )


def summarize_scenario(scenario, handle):
    rows = handle.head(10000)
    meta = handle.meta()
    total = as_int(meta.get("totalFacts", meta.get("total_facts", handle.count())), handle.count())
    production_lines = 0
    bad = 0
    for fact in rows:
        attrs = fact.get("attributes", {})
        if not isinstance(attrs, dict):
            bad += 1
            continue
        role = str(attrs.get("role", "other"))
        line_count = as_int(attrs.get("line_count"), -1)
        if line_count < 0:
            bad += 1
            continue
        if role == "implementation":
            production_lines += line_count
    cache = meta.get("cache", {}) if isinstance(meta.get("cache", {}), dict) else {}
    complete = total == len(rows) and bad == 0 and not meta.get("nextPageToken", meta.get("next_page_token", ""))
    return {
        "scenario": scenario,
        "source_hash": cache.get("source_hash", cache.get("sourceHash")),
        "config_hash": cache.get("config_hash", cache.get("configHash")),
        "metrics_revision": cache.get("analyzer_version", cache.get("analyzerVersion")),
        "metric_facts": len(rows),
        "production_lines": production_lines if complete else None,
        "complete": complete,
        "bad_facts": bad,
        "warnings": len(meta.get("warnings", [])) if isinstance(meta.get("warnings", []), list) else None,
    }


def compact_snapshot(snapshot):
    return {
        "schema": "dr6-weekly-debt-snapshot-store/1",
        "observed_at": snapshot["observed_at"],
        "window": snapshot["window"],
        "scenario_metrics": [
            [row.get("scenario"), row.get("production_lines"), row.get("source_hash"), row.get("config_hash"), row.get("metrics_revision"), row.get("complete")]
            for row in snapshot.get("scenario_metrics", [])
        ],
        "template_debt": snapshot.get("template_debt"),
        "lifecycle_inventory": snapshot.get("lifecycle_inventory"),
        "reduction_receipts": snapshot.get("reduction_receipts"),
        "approach_yield": snapshot.get("approach_yield", []),
        "source_trend": snapshot.get("source_trend"),
        "detector_snapshot": snapshot.get("detector_snapshot"),
        "source_coverage": snapshot.get("source_coverage"),
    }


def expand_snapshot(snapshot):
    if not isinstance(snapshot, dict):
        return None
    if snapshot.get("schema") == "dr6-weekly-debt-snapshot/1":
        return snapshot
    if snapshot.get("schema") != "dr6-weekly-debt-snapshot-store/1":
        return None
    rows = []
    for packed in snapshot.get("scenario_metrics", []):
        if not isinstance(packed, list) or len(packed) != 6:
            return None
        scenario, line_count, source_hash, config_hash, revision, complete = packed
        rows.append({
            "scenario": scenario,
            "production_lines": line_count,
            "source_hash": source_hash,
            "config_hash": config_hash,
            "metrics_revision": revision,
            "complete": complete,
        })
    return {
        "schema": "dr6-weekly-debt-snapshot/1",
        "observed_at": snapshot.get("observed_at"),
        "window": snapshot.get("window", {}),
        "scenario_metrics": rows,
        "template_debt": snapshot.get("template_debt"),
        "lifecycle_inventory": snapshot.get("lifecycle_inventory"),
        "reduction_receipts": snapshot.get("reduction_receipts", {}),
        "approach_yield": snapshot.get("approach_yield", []),
        "source_trend": snapshot.get("source_trend"),
        "detector_snapshot": snapshot.get("detector_snapshot"),
        "source_coverage": snapshot.get("source_coverage", {}),
    }


def snapshot_trend_eligibility(snapshot, expected_start, expected_end, observed_on):
    """Reject history that lacks complete, well-formed inputs for this trend."""
    if not isinstance(snapshot, dict) or snapshot.get("schema") != "dr6-weekly-debt-snapshot/1":
        return False
    window = snapshot.get("window", {})
    try:
        start = valid_day(window.get("start_inclusive", ""), "snapshot start")
        end = valid_day(window.get("end_exclusive", ""), "snapshot end")
        observed = valid_day(str(snapshot.get("observed_at", ""))[:10], "snapshot observation")
    except (TypeError, ValueError):
        return False
    if start != expected_start or end != expected_end or (end - start).days != 7:
        return False
    # Snapshots are taken on the day after the exclusive window end. Requiring
    # that cadence keeps a midweek rerun from becoming a comparison baseline.
    if observed != end + datetime.timedelta(days=1) or (observed_on - observed).days != 7:
        return False

    coverage = snapshot.get("source_coverage")
    if not isinstance(coverage, dict):
        return False
    required_coverage = (
        "scenario_metrics_complete",
        "template_debt_complete",
        "research_receipts_complete",
        "ecosystem_hunt_receipts_complete",
        "delivery_receipts_complete",
    )
    if any(coverage.get(key) is not True for key in required_coverage):
        return False

    metrics = snapshot.get("scenario_metrics")
    if not isinstance(metrics, list) or not metrics:
        return False
    scenarios = set()
    for row in metrics:
        if not isinstance(row, dict):
            return False
        scenario = row.get("scenario")
        count = row.get("production_lines")
        if (not isinstance(scenario, str) or not scenario or scenario in scenarios
                or row.get("complete") is not True
                or isinstance(count, bool) or not isinstance(count, int) or count < 0
                or not all(isinstance(row.get(key), str) and row[key] for key in ("source_hash", "config_hash", "metrics_revision"))):
            return False
        scenarios.add(scenario)

    template = snapshot.get("template_debt")
    if not isinstance(template, dict) or template.get("status") != "measured":
        return False
    template_total = template.get("total")
    if isinstance(template_total, bool) or not isinstance(template_total, int) or template_total < 0:
        return False

    receipts = snapshot.get("reduction_receipts")
    if not isinstance(receipts, dict):
        return False
    for source in SOURCE_NAMES:
        item = receipts.get(source)
        if not isinstance(item, dict) or item.get("status") != "measured":
            return False
        for key in ("valid_receipts", "journal_rows", "scopes"):
            value = item.get(key)
            if isinstance(value, bool) or not isinstance(value, int) or value < 0:
                return False
        if item.get("unstructured_rows") != 0 or item.get("invalid_receipts") != 0:
            return False
        if isinstance(item.get("net_reduction_lines"), bool) or not isinstance(item.get("net_reduction_lines"), int):
            return False
    return True


def read_previous_snapshot(start, end, observed_on):
    try:
        handle = source_ledger.journal.list(scope=SNAPSHOT_SCOPE, limit=20, newest_first=True)
        rows = handle.head(20)
        envelope["evidence"].append({"binding": "source-ledger/journal/list", "scope": SNAPSHOT_SCOPE, "snapshot_rows": len(rows)})
    except Exception as exc:
        error_status(exc, "collect:prior_snapshots")
        return None, None
    candidates = []
    current_entries = []
    for row in rows:
        body = row.get("body", "")
        if not isinstance(body, str):
            continue
        record = None
        for line in body.splitlines():
            if line.startswith(SNAPSHOT_MARKER):
                try:
                    record = json.loads(line[len(SNAPSHOT_MARKER):].strip())
                except (TypeError, ValueError):
                    record = None
                break
        packed_snapshot = record.get("snapshot") if isinstance(record, dict) and record.get("schema") == "dr6-weekly-ledger-record/1" else None
        snapshot = expand_snapshot(packed_snapshot)
        if not isinstance(snapshot, dict):
            continue
        window = snapshot.get("window", {})
        try:
            snapshot_start = valid_day(window.get("start_inclusive", ""), "snapshot start")
            snapshot_end = valid_day(window.get("end_exclusive", ""), "snapshot end")
            snapshot_day = valid_day(str(snapshot.get("observed_at", ""))[:10], "snapshot observation")
        except (TypeError, ValueError):
            continue
        entry_id = row.get("id", "")
        created_at = str(row.get("createdAt", row.get("created_at", "")))
        if snapshot_start == start and snapshot_end == end:
            current_entries.append((created_at, snapshot, entry_id))
        if (snapshot_end == start and snapshot_trend_eligibility(snapshot, snapshot_start, snapshot_end, observed_on)):
            candidates.append((created_at, snapshot, entry_id))
    candidates.sort(key=lambda item: item[0], reverse=True)
    current_entries.sort(key=lambda item: item[0], reverse=True)
    if candidates:
        created, snapshot, entry_id = candidates[0]
        envelope["signals"]["previous_window"] = {"window": snapshot.get("window"), "source_ledger_entry_id": entry_id, "created_at": created, "trend_inputs": "complete"}
    if current_entries:
        envelope["evidence"].append({"source_ledger_current_window_snapshot": current_entries[0][2]})
    return (candidates[0] if candidates else None), (current_entries[0][2] if current_entries else None)


def append_snapshot(snapshot):
    body = SNAPSHOT_MARKER + " " + json.dumps({"schema": "dr6-weekly-ledger-record/1", "snapshot": compact_snapshot(snapshot)}, allow_nan=False, separators=(",", ":"))
    if len(body) > 50000:
        return {"status": "failed", "reason": "compact snapshot exceeds the 50,000-character Source Ledger entry bound"}
    try:
        handle = source_ledger.journal.note(body=body, scope=SNAPSHOT_SCOPE, kind="weekly-snapshot")
        meta = handle.meta()
        entry = meta.get("entry", {}) if isinstance(meta.get("entry", {}), dict) else {}
        rows = handle.head(1)
        if not entry and rows and isinstance(rows[0].get("entry"), dict):
            entry = rows[0]["entry"]
        entry_id = entry.get("id")
        return {"status": "stored" if entry_id else "stored_identity_unknown", "entry_id": entry_id}
    except Exception as exc:
        status, klass = program.classify(exc)
        return {"status": status, "error_class": klass, "reason": str(exc)[:240]}


def previous_trend(previous, start, end, current_rows):
    if previous is None:
        return {"status": "unknown", "reason": "no adjacent prior snapshot with complete owner coverage and validated receipt populations", "rows": []}, None
    created, snapshot, run_id = previous
    previous_window = snapshot.get("window", {})
    envelope["signals"]["previous_window"] = {"window": previous_window, "source_ledger_entry_id": run_id, "created_at": created}
    try:
        prior_end = valid_day(previous_window.get("end_exclusive", ""), "previous end")
    except (TypeError, ValueError):
        return {"status": "unknown", "reason": "prior snapshot window is invalid", "rows": []}, snapshot
    if prior_end != start:
        return {"status": "unknown", "reason": "prior snapshot does not end at this window start", "rows": []}, snapshot
    prior = {row.get("scenario"): row for row in snapshot.get("scenario_metrics", []) if isinstance(row, dict)}
    deltas = []
    for row in current_rows:
        old = prior.get(row["scenario"])
        if not row.get("complete") or not isinstance(old, dict) or not old.get("complete") or not isinstance(row.get("production_lines"), int) or isinstance(row.get("production_lines"), bool) or not isinstance(old.get("production_lines"), int) or isinstance(old.get("production_lines"), bool):
            continue
        row["weekly_production_line_delta"] = row["production_lines"] - old["production_lines"]
        row["prior_source_hash"] = old.get("source_hash")
        deltas.append(row["weekly_production_line_delta"])
    total_delta = sum(deltas)
    return {"status": "measured" if deltas else "unknown", "reason": "scenario production deltas are not attributed to a team or a landed unit", "matched_scenarios": len(deltas), "fleet_net_production_line_delta": total_delta}, snapshot


def receipt_trend(previous_snapshot, current_receipts, current_yields):
    if not isinstance(previous_snapshot, dict):
        return {"status": "unknown", "reason": "no adjacent prior receipt snapshot", "rows": [], "approach_rows": []}
    prior_receipts = previous_snapshot.get("reduction_receipts", {})
    prior_yields = previous_snapshot.get("approach_yield", [])
    rows = []
    for source in SOURCE_NAMES:
        current = current_receipts.get(source, {})
        prior = prior_receipts.get(source, {}) if isinstance(prior_receipts, dict) else {}
        if (current.get("status") != "measured" or prior.get("status") != "measured"
                or not isinstance(current.get("net_reduction_lines"), int)
                or not isinstance(prior.get("net_reduction_lines"), int)):
            rows.append({"source": source, "status": "unknown", "reason": "both adjacent source populations must be complete"})
            continue
        rows.append({
            "source": source,
            "status": "measured",
            "prior_landed_units": prior.get("valid_receipts"),
            "current_landed_units": current.get("valid_receipts"),
            "landed_unit_delta": current.get("valid_receipts", 0) - prior.get("valid_receipts", 0),
            "prior_net_reduction_lines": prior["net_reduction_lines"],
            "current_net_reduction_lines": current["net_reduction_lines"],
            "net_reduction_line_delta": current["net_reduction_lines"] - prior["net_reduction_lines"],
        })
    prior_by_key = {(row.get("source"), row.get("approach")): row for row in prior_yields if isinstance(row, dict)} if isinstance(prior_yields, list) else {}
    current_by_key = {(row.get("source"), row.get("approach")): row for row in current_yields if isinstance(row, dict)}
    approach_rows = []
    for key in sorted(set(prior_by_key) | set(current_by_key)):
        prior = prior_by_key.get(key)
        current = current_by_key.get(key)
        if not isinstance(prior, dict) or not isinstance(current, dict):
            approach_rows.append({"source": key[0], "approach": key[1], "status": "unknown", "reason": "approach must have validated receipts in both weeks"})
            continue
        approach_rows.append({
            "source": key[0],
            "approach": key[1],
            "status": "measured",
            "prior_landed_units": prior.get("landed_units"),
            "current_landed_units": current.get("landed_units"),
            "prior_mean_net_reduction_per_landed_unit": prior.get("mean_net_reduction_per_landed_unit"),
            "current_mean_net_reduction_per_landed_unit": current.get("mean_net_reduction_per_landed_unit"),
        })
    measured = sum(row["status"] == "measured" for row in rows)
    status = "measured" if measured == len(SOURCE_NAMES) else "partial" if measured else "unknown"
    return {"status": status, "rows": rows, "approach_rows": approach_rows}


def main():
    try:
        start, end, max_scenarios, max_journal_rows, detector_input, lifecycle_input, publish_snapshot = validate_inputs()
        lifecycle_inventory = validated_lifecycle_snapshot(lifecycle_input, start, end)
    except (TypeError, ValueError) as exc:
        fail("invalid_input", exc, "validate")
        envelope["status"] = "failed"
        print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
        return
    envelope["signals"]["lifecycle_inventory"] = lifecycle_inventory

    envelope["phase"] = "collect"
    roster = []
    roster_count = 0
    roster_truncated = False
    try:
        roster, roster_count, roster_truncated = collect_scenario_rows(max_scenarios)
        envelope["evidence"].append({"binding": "flow-verifier/scenarios/list", "scenarios": len(roster), "reported": roster_count})
    except Exception as exc:
        error_status(exc, "collect:scenario_roster")

    metric_rows = []
    metric_failures = []
    scenarios = [str(row.get("id", "")).strip() for row in roster if isinstance(row, dict) and str(row.get("id", "")).strip()]
    for offset in range(0, len(scenarios), 8):
        batch = scenarios[offset:offset + 8]
        results = gather(*[guarded(lambda scenario=scenario: scenario_metric_call(scenario)) for scenario in batch])
        for scenario, result in zip(batch, results):
            if isinstance(result, Exception):
                metric_failures.append(scenario)
                error_status(result, "collect:code_facts:" + scenario)
                continue
            try:
                metric_rows.append(summarize_scenario(scenario, result))
            except Exception as exc:
                metric_failures.append(scenario)
                fail("invalid_response", exc, "aggregate:code_facts:" + scenario)
    envelope["signals"]["scenario_coverage"] = {
        "roster_count": roster_count,
        "selected_count": len(scenarios),
        "measured_count": len(metric_rows),
        "complete_count": sum(1 for row in metric_rows if row["complete"]),
        "failed_scenarios": metric_failures,
        "truncated": roster_truncated,
    }

    envelope["phase"] = "classify"
    observed_at = datetime.datetime.now(datetime.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
    observed_on = datetime.date.fromisoformat(observed_at[:10])
    previous, current_snapshot_id = read_previous_snapshot(start, end, observed_on)
    line_trend, previous_snapshot = previous_trend(previous, start, end, metric_rows)
    envelope["signals"]["line_trend"] = line_trend
    detector_snapshot = {"status": "unknown", "rows": []}
    detector_targets = detector_input.get("scenarios", [])
    if isinstance(detector_targets, list):
        metric_sources = {row["scenario"]: row.get("source_hash") for row in metric_rows}
        detector_counts = {}
        seen_targets = set()
        partial = False
        for target in detector_targets[:200]:
            if not isinstance(target, dict):
                partial = True
                continue
            scenario = str(target.get("scenario", ""))
            source_hash = str(target.get("source_hash", ""))
            recommendations = target.get("recommendations", [])
            if not scenario or scenario not in metric_sources or not source_hash or source_hash != metric_sources[scenario] or not isinstance(recommendations, list) or scenario in seen_targets:
                partial = True
                continue
            seen_targets.add(scenario)
            for rec in recommendations:
                detector_id = rec.get("detector_id") if isinstance(rec, dict) else None
                if not isinstance(detector_id, str) or not detector_id.strip():
                    partial = True
                    continue
                row = detector_counts.setdefault(detector_id, {"detector_id": detector_id, "candidate_groups_returned": 0, "scenarios": 0})
                row["candidate_groups_returned"] += 1
                row["scenarios"] += 1
        detector_snapshot["rows"] = sorted(detector_counts.values(), key=lambda row: row["detector_id"])
        detector_snapshot["scenarios_measured"] = len(seen_targets)
        detector_snapshot["scenarios_in_roster"] = len(scenarios)
        detector_snapshot["status"] = "measured" if len(seen_targets) == len(scenarios) and not partial else "partial" if seen_targets or partial else "unknown"
    if not detector_snapshot["rows"]:
        detector_snapshot["reason"] = "no source-hash-bound recommendation snapshots supplied"
    envelope["signals"]["detector_snapshot"] = detector_snapshot

    template_debt = {"status": "unknown", "total": None, "by_status": {}}
    try:
        handle = template_manager.debt.list()
        rows = handle.head(10000)
        if handle.count() <= 10000:
            status_counts = {}
            for row in rows:
                status = str(row.get("status", "unknown"))
                status_counts[status] = status_counts.get(status, 0) + 1
            template_debt = {"status": "measured", "total": len(rows), "by_status": status_counts}
            envelope["evidence"].append({"binding": "template-manager/debt/list", "rows": len(rows)})
        else:
            template_debt = {"status": "partial", "total": handle.count(), "returned": len(rows), "by_status": {}}
    except Exception as exc:
        error_status(exc, "collect:template_debt")
    envelope["signals"]["template_debt"] = template_debt

    scope_rows = []
    try:
        scope_handle = source_ledger.scopes.list()
        scope_rows = scope_handle.head(200)
        if scope_handle.count() > 200:
            fail("incomplete_source", "scope list exceeds the 200-row materialization limit", "collect:scopes")
        envelope["evidence"].append({"binding": "source-ledger/scopes/list", "rows": len(scope_rows)})
    except Exception as exc:
        error_status(exc, "collect:scopes")

    entries_by_scope = []
    scope_defs = selected_scope_records(scope_rows)
    journal_calls = [
        (scope_id, source, lambda scope_id=scope_id: source_ledger.journal.list(scope=scope_id, limit=max_journal_rows, newest_first=True))
        for scope_id, source in scope_defs
    ]
    for offset in range(0, len(journal_calls), 8):
        batch = journal_calls[offset:offset + 8]
        results = gather(*[guarded(call) for _, _, call in batch])
        for (scope_id, source, _), result in zip(batch, results):
            if isinstance(result, Exception):
                error_status(result, "collect:journal:" + scope_id)
                continue
            rows = result.head(max_journal_rows)
            entries_by_scope.append((scope_id, source, rows, len(rows) >= max_journal_rows))
    receipts, yields = aggregate_receipts(entries_by_scope, start, end)
    source_trend = receipt_trend(previous_snapshot, receipts, yields)
    envelope["signals"]["reduction_receipts"] = receipts
    envelope["signals"]["approach_yield"] = yields
    envelope["signals"]["source_trend"] = source_trend
    for source in SOURCE_NAMES:
        if source == "research" and receipts[source]["scopes"] == 0:
            receipts[source]["status"] = "unknown"
            receipts[source]["reason"] = "no registered research ledger scope; add an explicit source-owned scope before claiming coverage"
        if source == "delivery" and receipts[source]["scopes"] == 0:
            receipts[source]["status"] = "unknown"

    lifecycle = envelope["signals"]["lifecycle_inventory"]
    if lifecycle.get("status") == "unknown":
        lifecycle["reason"] = "live Flow Verifier route was stopped after 45 seconds; no retry or estimate used"
    envelope["signals"]["source_coverage"] = {
        "scenario_metrics_complete": bool(metric_rows) and not metric_failures and not roster_truncated and all(row["complete"] for row in metric_rows),
        "template_debt_complete": template_debt.get("status") == "measured",
        "lifecycle_complete": False,
        "lifecycle_saved_evidence_present": lifecycle.get("status") == "supplied_saved",
        "research_receipts_complete": receipts["research"]["status"] == "measured",
        "ecosystem_hunt_receipts_complete": receipts["ecosystem-hunt"]["status"] == "measured",
        "delivery_receipts_complete": receipts["delivery"]["status"] == "measured",
        "detector_snapshot_complete": detector_snapshot["status"] == "measured",
    }
    weekly_snapshot = {
        "schema": "dr6-weekly-debt-snapshot/1",
        "observed_at": observed_at,
        "window": envelope["signals"]["window"],
        "scenario_metrics": metric_rows,
        "template_debt": {"status": template_debt.get("status"), "total": template_debt.get("total"), "by_status": template_debt.get("by_status")},
        "lifecycle_inventory": lifecycle,
        "reduction_receipts": receipts,
        "approach_yield": yields,
        "source_trend": source_trend,
        "source_coverage": envelope["signals"]["source_coverage"],
    }
    envelope["signals"]["weekly_snapshot"] = weekly_snapshot
    if publish_snapshot:
        envelope["signals"]["snapshot_write"] = append_snapshot(weekly_snapshot)
        envelope["signals"]["source_coverage"]["weekly_snapshot_persisted"] = envelope["signals"]["snapshot_write"].get("status") == "stored"
        if current_snapshot_id:
            envelope["signals"]["snapshot_write"]["previous_current_window_entry_id"] = current_snapshot_id
        if envelope["signals"]["snapshot_write"].get("status") != "stored":
            fail("incomplete_source", "weekly snapshot was not stored in Source Ledger", "report:snapshot_write")
    else:
        envelope["signals"]["snapshot_write"] = {"status": "not_requested"}
        envelope["signals"]["source_coverage"]["weekly_snapshot_persisted"] = False
    evidence_complete = all(envelope["signals"]["source_coverage"].values())
    envelope["status"] = "ok" if evidence_complete and not envelope["errors"] else "partial"
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))


main()
