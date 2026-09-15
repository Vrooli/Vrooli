"""agent-manager.efficiency-report v1 — deterministic read-only efficiency sensor.

The program composes the existing governed MeasuresService reads. The REST
endpoint remains the richer operator surface; this path is intentionally
bounded and portable through Program Runtime's Connect-only binding model.
"""

import datetime
import json

inputs = program.inputs()
preset = str(inputs.get("preset", "24h")).strip()
start = str(inputs.get("start", "")).strip()
end = str(inputs.get("end", "")).strip()
tag_prefix = str(inputs.get("tag_prefix", "")).strip()
group_by = str(inputs.get("group_by", "profile")).strip()
limit = int(inputs.get("limit", 20))
compare = str(inputs.get("compare", "")).strip()

envelope = {
    "program": "agent-manager.efficiency-report", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"preset": preset, "start": start, "end": end, "tag_prefix": tag_prefix, "group_by": group_by, "limit": limit, "compare": compare},
    "report": None,
    "evidence": {"source_projection": "invocation_read_model_runs", "inference_calls": 0, "read_only": True},
    "errors": []
}


def fail(detail):
    envelope["errors"].append({"class": "invalid_input", "detail": str(detail)[:240]})
    return "report"


def step_validate():
    if preset not in {"6h", "12h", "24h", "7d", "30d"}:
        return fail("preset must be one of 6h, 12h, 24h, 7d, 30d")
    if group_by not in {"runner", "model", "profile"}:
        return fail("group_by must be one of runner, model, or profile")
    if limit < 1 or limit > 100:
        return fail("limit must be between 1 and 100")
    if compare not in {"", "previous"}:
        return fail("compare must be empty or previous")
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    now = datetime.datetime.now(datetime.timezone.utc)
    spans = {"6h": datetime.timedelta(hours=6), "12h": datetime.timedelta(hours=12),
             "24h": datetime.timedelta(hours=24), "7d": datetime.timedelta(days=7),
             "30d": datetime.timedelta(days=30)}

    def window(offset=0):
        if start or end:
            from_value = start or (now - spans[preset]).isoformat().replace("+00:00", "Z")
            to_value = end or now.isoformat().replace("+00:00", "Z")
        else:
            to = now - offset * spans[preset]
            from_value = (to - spans[preset]).isoformat().replace("+00:00", "Z")
            to_value = to.isoformat().replace("+00:00", "Z")
        return {"custom": {"from": from_value, "to": to_value}}

    def binding(name):
        return getattr(agent_manager.measures, name)

    dimension = {"profile": "profile_breakdown", "runner": "runner_breakdown",
                 "model": "model_breakdown"}.get(group_by)
    if dimension is None:
        envelope["status"] = "unavailable"
        envelope["errors"].append({"class": "invalid_input", "detail": "program bindings support runner, model, and profile group_by", "where": "collect"})
        return "report"

    def collect_one(offset=0):
        w = window(offset)
        f = {"tag_prefix": tag_prefix} if tag_prefix else None
        kwargs = {"window": w}
        if f:
            kwargs["filter"] = f
        calls = [
            ("volume", lambda: binding("run_volume")(**kwargs)),
            ("success", lambda: binding("run_success_rate")(**kwargs)),
            ("durations", lambda: binding("run_duration_statistics")(**kwargs)),
            ("cost", lambda: binding("run_cost")(**kwargs)),
            ("statuses", lambda: binding("run_status_distribution")(**kwargs)),
            ("breakdown", lambda: binding(dimension)(**kwargs)),
            ("tools", lambda: binding("tool_usage")(**kwargs)),
        ]
        values = {}
        errors = []
        results = gather(*[call for _, call in calls])
        for (name, _), result in zip(calls, results):
            if isinstance(result, Exception):
                values[name] = None
                errors.append(name + ":" + str(result)[:120])
                continue
            try:
                rows = result.head(limit)
                values[name] = rows
                if not rows:
                    errors.append(name + ":empty")
            except Exception as exc:
                values[name] = None
                errors.append(name + ":" + str(exc)[:120])
        values["window"] = {"from": w["custom"]["from"], "to": w["custom"]["to"]}
        values["errors"] = errors
        return values

    def compact(record, fields):
        if not record:
            return None
        return {field: record[field] for field in fields if field in record}

    def compact_report(raw, row_limit=limit):
        if raw is None:
            return None
        scalar_fields = {
            "volume": ["totalRuns", "terminalRuns", "validity", "provenance"],
            "success": ["rate", "successfulRuns", "terminalRuns", "validity", "provenance"],
            "durations": ["averageDurationMs", "p50DurationMs", "p95DurationMs", "p99DurationMs", "minDurationMs", "maxDurationMs", "count", "validity", "provenance"],
            "cost": ["totalCostUsd", "averageCostUsd", "totalTokens", "totalRuns", "inputTokens", "outputTokens", "cacheReadTokens", "cacheCreationTokens", "validity", "provenance"],
        }
        out = {"window": raw["window"], "errors": raw["errors"]}
        for name, fields in scalar_fields.items():
            record = compact(raw[name][0] if raw.get(name) else None, fields)
            if record and isinstance(record.get("validity"), dict):
                validity = record["validity"]
                record["validity"] = {key: validity[key] for key in ("state", "sampleSize", "classifiedBase") if key in validity}
            if record and isinstance(record.get("provenance"), dict):
                provenance = record["provenance"]
                record["provenance"] = {key: provenance[key] for key in ("projectionAt", "projectionAgeMs", "projectionStale", "projectionStaleReason") if key in provenance}
            out[name] = record
        out["statuses"] = [compact(row, ["status", "count"]) for row in raw.get("statuses", [])[:row_limit]]
        out["breakdown"] = [compact(row, ["value", "runCount", "successCount", "failedCount", "totalTokens", "averageDurationMs", "completionRate"]) for row in raw.get("breakdown", [])[:row_limit]]
        out["tools"] = [compact(row, ["toolName", "callCount", "failedCount", "totalTokens"]) for row in raw.get("tools", [])[:row_limit]]
        freshness = next((item.get("provenance") for name in ("volume", "success", "durations", "cost")
                          for item in (raw.get(name) or []) if item.get("provenance")), None)
        out["freshness"] = freshness or {"available": False}
        out["_stale"] = any(bool(item.get("provenance", {}).get("projectionStale"))
                           for name in ("volume", "success", "durations", "cost")
                           for item in (raw.get(name) or []))
        return out

    try:
        current = collect_one()
        baseline = collect_one(1) if compare else None
        current_report = compact_report(current)
        baseline_report = compact_report(baseline, 1)
        status = "partial" if current["errors"] or current_report.pop("_stale", False) else "ok"
        if status == "partial" and not current["errors"]:
            envelope["errors"].append({"class": "stale_evidence", "detail": "projection is older than the requested window", "where": "collect"})
        envelope["report"] = {"status": status,
                               "window": current["window"], "measures": current_report,
                               "baseline": baseline_report,
                               "evidence": envelope["evidence"]}
        envelope["status"] = envelope["report"]["status"]
        if current["errors"]:
            envelope["errors"].extend({"class": "binding_error", "detail": e, "where": "collect"} for e in current["errors"])
    except Exception as exc:
        envelope["status"] = "unavailable"
        envelope["errors"].append({"class": "binding_error", "detail": str(exc)[:240]})
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
