"""program-runtime.portfolio-audit v1 — deterministic maturity score for the declared program portfolio."""

import json


envelope = {
    "program": "program-runtime.portfolio-audit",
    "version": "1",
    "status": "failed",
    "phase": "collect",
    "inputs": dict(inputs),
    "signals": {},
    "errors": [],
    "evidence": [],
}


def fail(status, klass, detail):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240]})


def ratio(count, total):
    return {"count": count, "total": total, "ratio": (count / total) if total else 0.0}


def read_rows(handle, limit=200):
    return handle.head(limit)


try:
    requested_rung = str(inputs.get("min_rung", "S3"))
    scenario_filter = str(inputs.get("scenario", "")).strip()
    window_days = int(inputs.get("window_days", 30))
    if window_days < 0 or window_days > 365:
        fail("failed", "invalid_input", "window_days must be between 0 and 365")
    else:
        library = read_rows(program_runtime.library.list(limit=200))
        contracts = [
            row for row in library
            if row.get("kind") == "contract"
            and row.get("rung", "S3") in (["S4"] if requested_rung == "S4" else ["S3", "S4"])
            and (not scenario_filter or row.get("scenario") == scenario_filter)
        ]
        envelope["evidence"].append("program-runtime/library/list limit=200")
        portfolio = None
        portfolio_handle = None
        portfolio_error = None
        try:
            portfolio_handle = program_runtime.programs.portfolio(window_days=window_days, scenario=scenario_filter, include_ad_hoc=False)
            portfolio = read_rows(portfolio_handle, 1000)
            envelope["evidence"].append("program-runtime/programs/portfolio")
        except Exception as exc:
            portfolio_error = exc

        portfolio_by_name = {row.get("name"): row for row in (portfolio or [])}
        never_executed = set()
        if portfolio is not None:
            try:
                never_executed = set(portfolio_handle.meta().get("neverExecuted", []))
            except Exception:
                never_executed = set()

        dimensions = {
            "source_present": 0,
            "binding_resilience": 0,
            "live_evidence": 0,
            "capability_tag": 0,
            "budget_realism": 0,
            "exercised": 0,
            "learning": 0,
            "typed_output": 0,
        }
        scores = []
        worst = []
        for contract in contracts:
            name = contract.get("name", "")
            gaps = []
            values = {}
            values["source_present"] = not contract.get("sourceMissing", False) and bool(contract.get("source", ""))
            values["binding_resilience"] = contract.get("optionalBindingCount", 0) > 0
            values["live_evidence"] = contract.get("liveFixtureCount", 0) > 0
            values["capability_tag"] = bool(contract.get("verbs", []))
            values["learning"] = contract.get("rung") != "S4" or contract.get("memoryDeclared", False)
            values["typed_output"] = bool(contract.get("outputSchemaPresent", False))
            if portfolio is None:
                values["budget_realism"] = None
                values["exercised"] = None
            else:
                measured = portfolio_by_name.get(name)
                values["exercised"] = name not in never_executed
                values["budget_realism"] = measured is None or int(measured.get("declaredWallMillis", 0) or 0) == 0 or int(measured.get("p95Millis", 0) or 0) <= int(measured.get("declaredWallMillis", 0) or 0)
            available = 0
            earned = 0
            for dimension, passed in values.items():
                if passed is None:
                    continue
                available += 1
                if passed:
                    earned += 1
                else:
                    gaps.append(dimension)
                    dimensions[dimension] -= 0
            score = earned / available if available else 0.0
            scores.append(score)
            for dimension, passed in values.items():
                if passed:
                    dimensions[dimension] += 1
            worst.append({"name": name, "score": score, "gaps": gaps})

        total = len(contracts)
        signals = {dimension: ratio(count, total) for dimension, count in dimensions.items()}
        if portfolio is None:
            signals["budget_realism"] = {"count": None, "total": 0, "ratio": None, "unavailable": True}
            signals["exercised"] = {"count": None, "total": 0, "ratio": None, "unavailable": True}
        signals["score"] = sum(scores) / len(scores) if scores else 0.0
        signals["scored"] = total
        signals["below_band"] = sum(1 for score in scores if score < 0.75)
        signals["worst"] = sorted(worst, key=lambda item: (item["score"], item["name"]))[:10]
        if portfolio_error is not None:
            envelope["status"] = "partial"
            envelope["errors"].append({"class": "binding_error", "detail": str(portfolio_error)[:240], "where": "portfolio"})
        else:
            envelope["status"] = "ok"
        envelope["signals"] = signals
except Exception as exc:
    fail("failed", "kernel_runtime", exc)

envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
