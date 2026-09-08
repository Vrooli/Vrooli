"""program-runtime.usage-triage v1 — one-batch triage of bounded program-run friction."""

import json


LABELS = ["helped", "fought_it", "abandoned", "wrong_program"]

envelope = {
    "program": "program-runtime.usage-triage",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {},
    "signals": {
        "labels": [],
        "pairs_classified": 0,
        "candidates_considered": 0,
        "candidates_capped": False,
        "attribution_basis": "transcript_match",
        "inference_calls": 0,
        "cohorts": [],
        "discovery_findings": [],
    },
    "errors": [],
    "evidence": [],
}


def err(kind, detail, where):
    return {"class": kind, "detail": str(detail)[:240], "where": where}


def transport_failure(exc):
    text = str(exc)
    for needle, status, kind in (
        ("requires an explicit grant", "refused", "no_grant"),
        ("not run eligible", "refused", "not_run_eligible"),
        ("inference spend", "refused", "inference_spend_exceeded"),
        ("is unreachable", "unavailable", "scenario_unreachable"),
        ("bridge unavailable", "unavailable", "scenario_unreachable"),
        ("scenario_not_running", "unavailable", "scenario_unreachable"),
        ("connection refused", "unavailable", "scenario_unreachable"),
        ("invalid_port", "unavailable", "scenario_unreachable"),
        ("no running runtime ports", "unavailable", "scenario_unreachable"),
    ):
        if needle in text:
            return status, kind
    return "failed", "kernel_runtime" if isinstance(exc, (NameError, AttributeError)) else "binding_error"


def value(row, *names, default=None):
    for name in names:
        candidate = row.get(name)
        if candidate is not None and candidate != "":
            return candidate
    return default


def as_int(raw, default=0):
    try:
        return int(raw or 0)
    except (TypeError, ValueError):
        return default


def as_float(raw, default=0.0):
    try:
        return float(raw or 0.0)
    except (TypeError, ValueError):
        return default


def program_name(row):
    # An empty program_name is an ad-hoc row. Its source is deliberately not
    # fingerprinted here: doing so would turn arbitrary Python identifiers
    # into a false declared-program identity.
    return str(value(row, "programName", "program_name", default="") or "").strip()


def caller_run_id(row):
    caller = row.get("caller") if isinstance(row.get("caller"), dict) else {}
    return str(value(row, "callerRunId", "caller_run_id", default=value(caller, "runId", "run_id", default="")) or "").strip()


def status_failed(row):
    status = str(value(row, "status", default="") or "").upper()
    return "FAILED" in status or bool(value(row, "failureDetail", "failure_detail", default=""))


def parse_label(row):
    if not isinstance(row, dict) or row.get("validated") is not True:
        return None
    try:
        parsed = json.loads(row.get("valueJson", ""))
    except (TypeError, ValueError):
        return None
    return parsed if parsed in LABELS else None


def validate_inputs():
    window_days = as_int(inputs.get("window_days", 30), 30)
    max_pairs = as_int(inputs.get("max_pairs", 32), 32)
    scenario = str(inputs.get("scenario", "") or "").strip()
    if window_days < 1 or window_days > 90:
        raise ValueError("window_days must be in [1, 90]")
    if max_pairs < 1 or max_pairs > 32:
        raise ValueError("max_pairs must be in [1, 32]")
    envelope["inputs"] = {"window_days": window_days, "max_pairs": max_pairs, "scenario": scenario}
    return window_days, max_pairs, scenario


def main():
    window_days, max_pairs, scenario = validate_inputs()
    envelope["phase"] = "collect"

    portfolio_rows = []
    portfolio_by_name = {}
    portfolio_error = None
    try:
        portfolio_handle = program_runtime.programs.portfolio(
            window_days=window_days, scenario=scenario, include_ad_hoc=False
        )
        portfolio_rows = portfolio_handle.head(512)
        portfolio_by_name = {str(row.get("name")): row for row in portfolio_rows if row.get("name")}
        envelope["evidence"].append("program-runtime/programs/portfolio")
    except Exception as exc:
        portfolio_error = exc
        status, kind = transport_failure(exc)
        envelope["errors"].append(err(kind, exc, "collect:portfolio"))
        if status == "refused":
            envelope["status"] = status

    try:
        program_handle = program_runtime.programs.list(
            provenance="agent", since_seconds=window_days * 86400, limit=512,
        )
        program_rows = program_handle.head(512)
        envelope["evidence"].append("program-runtime/programs/list provenance=agent")
    except Exception as exc:
        status, kind = transport_failure(exc)
        # A missing classifier is a usable triage degradation: the
        # deterministic candidate read completed, but no label is claimed.
        envelope["status"] = "partial" if status == "unavailable" else status
        envelope["errors"].append(err(kind, exc, "collect:programs"))
        program_rows = []

    runs = []
    runs_error = None
    try:
        run_handle = agent_manager.run.list(limit=200)
        runs = run_handle.head(200)
        envelope["evidence"].append("agent-manager/run/list limit=200")
    except Exception as exc:
        runs_error = exc
        status, kind = transport_failure(exc)
        envelope["errors"].append(err(kind, exc, "collect:runs"))

    run_previews = {}
    for run in runs:
        run_id = str(value(run, "id", default="") or "").strip()
        if run_id:
            run_previews[run_id] = str(value(run, "promptPreview", "prompt_preview", default="") or "").lower()

    # Candidate construction is deliberately plain Python and happens before the batch call.
    # First aggregate the durable program rows. A row without caller identity
    # is not attributed by timing; it must earn a transcript-match run id from
    # the governed conversation search below.
    aggregates = {}
    grouped = {}
    for row in portfolio_rows:
        name = str(row.get("name", "") or "").strip()
        if not name or (scenario and not name.startswith(scenario + ".")):
            continue
        aggregates[name] = {
            "program": name,
            "invocations": as_int(row.get("runs")),
            "failed": as_float(row.get("successRate", row.get("success_rate", 1))) < 1.0,
            "budget_pressure": as_float(row.get("budgetPressure", row.get("budget_pressure", 0))),
            "caller_rows": 0,
        }
    for row in program_rows:
        name = program_name(row)
        if not name or (scenario and not name.startswith(scenario + ".")):
            continue
        portfolio = portfolio_by_name.get(name, {})
        declared = as_float(value(portfolio, "declaredWallMillis", "declared_wall_millis", default=0))
        wall = as_float(value(row, "wallTimeMillis", "wall_time_millis", default=0))
        pressure = wall / declared if declared > 0 else 0.0
        aggregate = aggregates.setdefault(name, {
            "program": name, "invocations": 0, "failed": False,
            "budget_pressure": 0.0, "caller_rows": 0,
        })
        if name not in portfolio_by_name:
            aggregate["invocations"] += 1
        aggregate["failed"] = aggregate["failed"] or status_failed(row)
        aggregate["budget_pressure"] = max(aggregate["budget_pressure"], pressure)
        run_id = caller_run_id(row)
        if run_id:
            aggregate["caller_rows"] += 1
            key = (run_id, name)
            item = grouped.setdefault(key, {
                "program": name, "run_id": run_id, "invocations": 0,
                "failed": False, "budget_pressure": 0.0, "basis": "caller_identity",
            })
            item["invocations"] += 1
            item["failed"] = item["failed"] or status_failed(row)
            item["budget_pressure"] = max(item["budget_pressure"], pressure)

    priority = [item for item in aggregates.values()
                if item["invocations"] >= 3 or item["failed"] or item["budget_pressure"] > 1.0]
    priority.sort(key=lambda item: (-item["invocations"], -int(item["failed"]),
                                   -item["budget_pressure"], item["program"]))

    # Conversation search is the only acceptable fallback for rows without a
    # caller run id. It returns run ids, not transcript bodies, and is treated
    # as a lower bound. Prompt previews remain a cheap secondary match for
    # hosts where the search projection has not caught up yet.
    search_items = [item for item in priority if item["caller_rows"] == 0][:32]
    search_results = []
    search_error = None
    if search_items:
        try:
            def search(item):
                return agent_manager.conversation.search(
                    query=item["program"],
                    mode="CONVERSATION_SEARCH_MODE_TEXT",
                    sort="CONVERSATION_SEARCH_SORT_RELEVANCE",
                    page_size=5,
                )
            search_results = list(zip(search_items, gather(*[lambda item=item: search(item) for item in search_items])))
            envelope["evidence"].append("agent-manager/conversation/search text by program name")
        except Exception as exc:
            search_error = exc
            status, kind = transport_failure(exc)
            envelope["errors"].append(err(kind, exc, "collect:conversation-search"))

    for item, result in search_results:
        if isinstance(result, Exception):
            status, kind = transport_failure(result)
            envelope["errors"].append(err(kind, result, "collect:conversation-search"))
            continue
        hits = result.head(5)
        hit_counts = {}
        for hit in hits:
            metadata = hit.get("metadata") if isinstance(hit.get("metadata"), dict) else {}
            run_id = str(value(hit, "runId", "run_id", default=value(metadata, "run_id", default="")) or "").strip()
            if not run_id:
                continue
            hit_counts[run_id] = hit_counts.get(run_id, 0) + 1
        for run_id, invocation_count in sorted(hit_counts.items()):
            grouped[(run_id, item["program"])] = {
                "program": item["program"], "run_id": run_id,
                "invocations": invocation_count, "failed": item["failed"],
                "budget_pressure": item["budget_pressure"],
                "basis": "transcript_match",
            }

    candidates = [item for item in grouped.values()
                  if item["invocations"] >= 3 or item["failed"] or item["budget_pressure"] > 1.0]
    candidates.sort(key=lambda item: (-item["invocations"], -int(item["failed"]),
                                      -item["budget_pressure"], item["program"], item["run_id"]))
    envelope["signals"]["candidates_considered"] = len(candidates)
    if len(candidates) > max_pairs:
        envelope["signals"]["candidates_capped"] = True
    candidates = candidates[:max_pairs]
    envelope["signals"]["attribution_basis"] = (
        "caller_identity" if any(item["basis"] == "caller_identity" for item in candidates)
        else "transcript_match"
    )

    # One bounded corpus, with an abstention sentinel when there is no candidate.
    corpus = [
        f"program={item['program']}; purpose=declared program; invocations={item['invocations']}; "
        f"outcome={'failed' if item['failed'] else 'slow' if item['budget_pressure'] > 1.0 else 'repeated'}"
        for item in candidates
    ]
    if not corpus:
        corpus = ["No attributed program-run candidate was found; abstain from reporting a cohort."]

    envelope["phase"] = "classify"
    envelope["signals"]["inference_calls"] = 1
    try:
        batch_handle = ai.batch(
            corpus,
            {"type": "string", "enum": LABELS},
            "Classify each candidate as helped, fought_it, abandoned, or wrong_program. "
            "Use abandoned when the caller stopped after friction, fought_it when the program resisted the work, "
            "wrong_program when a different capability was needed, and helped only when the program clearly advanced the work. "
            "Abstain for the no-candidate sentinel.",
            role="classify.fast",
        )
        batch_rows = batch_handle.head(1)
        if len(batch_rows) != 1 or not isinstance(batch_rows[0], dict):
            raise ValueError("batch response must contain one object")
        batch = batch_rows[0]
        envelope["signals"]["usage"] = batch.get("usage")
        result_rows = batch.get("results") if isinstance(batch.get("results"), list) else []
        for field in ("provider", "model"):
            served = batch.get(field)
            if not served and result_rows and isinstance(result_rows[0], dict):
                served = result_rows[0].get(field)
            if served:
                envelope["signals"][field] = str(served)[:120]
        results = result_rows
        if not isinstance(results, list):
            raise ValueError("batch results must be an array")
        labels = []
        for index, item in enumerate(candidates):
            label = parse_label(results[index]) if index < len(results) else None
            if label is None:
                continue
            labels.append(label)
            entry = {
                "label": label,
                "program": item["program"],
                "exemplar_run_ids": [item["run_id"]],
                "invocations": item["invocations"],
            }
            if label == "wrong_program":
                envelope["signals"]["discovery_findings"].append(entry)
            elif label != "helped":
                envelope["signals"]["cohorts"].append(entry)
        envelope["signals"]["labels"] = labels
        envelope["signals"]["pairs_classified"] = len(labels) if candidates else 0
        envelope["evidence"].append({
            "binding": "ai-gateway/inference/run-batch", "requests": 1,
            "items": len(corpus), "role": "classify.fast",
            "provider": envelope["signals"].get("provider"),
            "model": envelope["signals"].get("model"),
            "usage": envelope["signals"].get("usage"),
        })
        envelope["status"] = "ok" if not envelope["errors"] else "partial"
    except Exception as exc:
        status, kind = transport_failure(exc)
        envelope["status"] = status
        envelope["errors"].append(err(kind, exc, "classify"))
        envelope["signals"]["cohorts"] = []
        envelope["signals"]["discovery_findings"] = []

    if portfolio_error is not None and envelope["status"] == "ok":
        envelope["status"] = "partial"
    if runs_error is not None and envelope["status"] == "ok":
        envelope["status"] = "partial"
    if search_error is not None and envelope["status"] == "ok":
        envelope["status"] = "partial"
    envelope["phase"] = "report"


try:
    main()
except Exception as exc:
    envelope["status"] = "failed"
    envelope["errors"].append(err("invalid_input" if envelope["phase"] == "validate" else "kernel_runtime", exc, envelope["phase"]))
finally:
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
